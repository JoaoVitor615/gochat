// Package messaging coordinates durable local queuing and P2P delivery.
package messaging

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/JoaoVitor615/gochat/internal/chat/message"
	"github.com/JoaoVitor615/gochat/internal/chat/p2p"
	"github.com/JoaoVitor615/gochat/internal/chat/storage"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	ma "github.com/multiformats/go-multiaddr"
)

const maxMessageContentBytes = 60 * 1024

// SendResult reports the durable message and its latest known local status.
// DeliveryError is non-nil when the message is queued but immediate delivery
// or its status update could not be completed.
type SendResult struct {
	Message       message.Message
	Status        message.DeliveryStatus
	DeliveryError error
}

// Service queues messages before attempting P2P delivery and retains them
// until a matching receiver ACK is persisted locally.
type Service struct {
	host        host.Host
	localPeerID string
	store       storage.Repository

	locksMu   sync.Mutex
	peerLocks map[string]*sync.Mutex
}

func New(h host.Host, localPeerID string, store storage.Repository) *Service {
	return &Service{
		host: h, localPeerID: localPeerID, store: store,
		peerLocks: make(map[string]*sync.Mutex),
	}
}

// AddPeerAddresses registers Discovery-provided addresses in libp2p's
// peerstore so NewStream can dial the peer by its PeerID.
func (s *Service) AddPeerAddresses(peerIDString string, addressStrings []string) error {
	if s == nil || s.host == nil {
		return errors.New("register peer addresses: P2P host is not running")
	}
	peerID, err := peer.Decode(peerIDString)
	if err != nil {
		return fmt.Errorf("register peer addresses: invalid peer ID: %w", err)
	}
	if len(addressStrings) == 0 {
		return errors.New("register peer addresses: at least one address is required")
	}

	addresses := make([]ma.Multiaddr, 0, len(addressStrings))
	for _, addressString := range addressStrings {
		address, err := ma.NewMultiaddr(addressString)
		if err != nil {
			return fmt.Errorf("register peer addresses: invalid multiaddr %q: %w", addressString, err)
		}
		info, err := peer.AddrInfoFromP2pAddr(address)
		if err != nil {
			return fmt.Errorf("register peer addresses: address %q must include a peer ID: %w", addressString, err)
		}
		if info.ID != peerID {
			return fmt.Errorf("register peer addresses: address peer ID %s does not match contact %s", info.ID, peerID)
		}
		addresses = append(addresses, info.Addrs...)
	}
	if len(addresses) == 0 {
		return errors.New("register peer addresses: no dialable addresses were provided")
	}

	s.host.Peerstore().ClearAddrs(peerID)
	s.host.Peerstore().AddAddrs(peerID, addresses, peerstore.PermanentAddrTTL)
	return nil
}

// Send persists the message and outbox entry atomically before attempting
// delivery. A network failure is reported in SendResult and does not discard
// the queued message.
func (s *Service) Send(ctx context.Context, recipientPeerID, content string) (SendResult, error) {
	if ctx == nil || s == nil || s.store == nil {
		return SendResult{}, errors.New("send message: context and message store are required")
	}
	if content == "" {
		return SendResult{}, errors.New("send message: content cannot be empty")
	}
	if len(content) > maxMessageContentBytes {
		return SendResult{}, fmt.Errorf("send message: content exceeds %d bytes", maxMessageContentBytes)
	}
	peerID, err := peer.Decode(recipientPeerID)
	if err != nil {
		return SendResult{}, fmt.Errorf("send message: invalid recipient peer ID: %w", err)
	}
	if s.localPeerID == "" || peerID.String() == s.localPeerID {
		return SendResult{}, errors.New("send message: a different local and recipient peer ID are required")
	}
	messageID, err := newMessageID()
	if err != nil {
		return SendResult{}, fmt.Errorf("generate message ID: %w", err)
	}
	envelope := message.Message{
		ID:              messageID,
		SenderPeerID:    s.localPeerID,
		RecipientPeerID: peerID.String(),
		CreatedAt:       time.Now().UTC(),
		Content:         content,
		ProtocolVersion: message.CurrentProtocolVersion,
	}
	if err := s.store.QueueMessage(ctx, peerID.String(), envelope); err != nil {
		return SendResult{}, fmt.Errorf("queue message locally: %w", err)
	}

	result := SendResult{Message: envelope, Status: message.DeliveryPending}
	result.DeliveryError = s.RetryPeer(ctx, peerID.String())
	stored, err := s.store.GetMessage(context.Background(), envelope.ID)
	if err != nil {
		return result, errors.Join(result.DeliveryError, fmt.Errorf("read queued message status: %w", err))
	}
	result.Status = stored.Status
	return result, nil
}

// RetryPeer attempts queued messages for one peer in chronological order.
// It stops at the first failure to avoid delivering later messages first.
func (s *Service) RetryPeer(ctx context.Context, peerID string) error {
	if ctx == nil || s == nil || s.store == nil {
		return errors.New("retry messages: context and message store are required")
	}
	decodedPeerID, err := peer.Decode(peerID)
	if err != nil {
		return fmt.Errorf("retry messages: invalid peer ID: %w", err)
	}
	if s.host == nil {
		return errors.New("retry messages: P2P host is not running")
	}
	if s.localPeerID == "" {
		return errors.New("retry messages: local peer ID is required")
	}

	unlock := s.lockPeer(decodedPeerID.String())
	defer unlock()

	entries, err := s.store.ListOutbox(ctx, 0)
	if err != nil {
		return fmt.Errorf("list queued messages: %w", err)
	}
	for _, entry := range entries {
		if entry.ConversationPeerID != decodedPeerID.String() {
			continue
		}
		if err := s.deliver(ctx, decodedPeerID, entry.Message); err != nil {
			return fmt.Errorf("deliver message %s: %w", entry.Message.ID, err)
		}
	}
	return nil
}

// RetryAll attempts queued messages for every peer, keeping each conversation
// ordered while allowing failures for one peer not to block other peers.
func (s *Service) RetryAll(ctx context.Context) error {
	if ctx == nil || s == nil || s.store == nil {
		return errors.New("retry messages: context and message store are required")
	}
	entries, err := s.store.ListOutbox(ctx, 0)
	if err != nil {
		return fmt.Errorf("list queued messages: %w", err)
	}
	peerIDs := make([]string, 0)
	seen := make(map[string]struct{})
	for _, entry := range entries {
		if _, ok := seen[entry.ConversationPeerID]; ok {
			continue
		}
		seen[entry.ConversationPeerID] = struct{}{}
		peerIDs = append(peerIDs, entry.ConversationPeerID)
	}
	sort.Strings(peerIDs)
	var retryErrors []error
	for _, peerID := range peerIDs {
		if err := s.RetryPeer(ctx, peerID); err != nil {
			retryErrors = append(retryErrors, fmt.Errorf("peer %s: %w", peerID, err))
		}
	}
	return errors.Join(retryErrors...)
}

func (s *Service) deliver(ctx context.Context, peerID peer.ID, envelope message.Message) error {
	written, sendErr := p2p.SendMessageAndWaitForAck(ctx, s.host, peerID, envelope)
	if sendErr != nil {
		if written {
			if err := s.store.UpdateMessageStatus(context.WithoutCancel(ctx), envelope.ID, message.DeliverySent); err != nil {
				return errors.Join(sendErr, fmt.Errorf("save sent status: %w", err))
			}
		}
		return sendErr
	}
	if err := s.store.UpdateMessageStatus(context.WithoutCancel(ctx), envelope.ID, message.DeliveryDelivered); err != nil {
		return fmt.Errorf("save delivered status after ACK: %w", err)
	}
	return nil
}

func (s *Service) lockPeer(peerID string) func() {
	s.locksMu.Lock()
	lock := s.peerLocks[peerID]
	if lock == nil {
		lock = &sync.Mutex{}
		s.peerLocks[peerID] = lock
	}
	s.locksMu.Unlock()
	lock.Lock()
	return lock.Unlock
}

func newMessageID() (string, error) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(id[:]), nil
}
