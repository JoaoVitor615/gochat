package p2p

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"

	"github.com/JoaoVitor615/gochat/internal/chat/crypto"
	"github.com/JoaoVitor615/gochat/internal/chat/message"
	"github.com/JoaoVitor615/gochat/internal/chat/storage"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

const ProtocolID = "/gochat/2.0.0"

func OpenStream(ctx context.Context, h host.Host, peerID peer.ID) (network.Stream, error) {
	stream, err := h.NewStream(ctx, peerID, ProtocolID)
	if err != nil {
		return nil, fmt.Errorf("failed to open stream: %w", err)
	}
	return stream, nil
}

func SetStreamHandler(h host.Host, handler network.StreamHandler) {
	h.SetStreamHandler(ProtocolID, handler)
}

func HandleStream(ctx context.Context, stream network.Stream, store storage.Repository, localPeerID string, notifier func(storage.StoredMessage), privKey ...ed25519.PrivateKey) error {
	if ctx == nil || stream == nil || store == nil || localPeerID == "" {
		return errors.New("handle P2P stream: context, stream, store, and local peer ID are required")
	}
	connection := newConnection(stream)
	defer func() { _ = connection.Close() }()

	frame, err := connection.ReceiveFrame()
	if err != nil {
		return fmt.Errorf("receive P2P frame: %w", err)
	}
	if frame.Type != message.FrameTypeMessage || frame.Message == nil {
		return fmt.Errorf("handle P2P stream: %w: expected message, got %s", ErrUnexpectedFrame, frame.Type)
	}

	msg := *frame.Message
	remotePeerID := stream.Conn().RemotePeer().String()
	if msg.ProtocolVersion != message.CurrentProtocolVersion {
		return fmt.Errorf("unsupported message protocol version %d", msg.ProtocolVersion)
	}
	if msg.ID == "" || msg.CreatedAt.IsZero() || msg.SenderPeerID != remotePeerID || msg.RecipientPeerID != localPeerID {
		return errors.New("invalid message envelope or peer identity mismatch")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	// Verify digital signature if present
	if msg.Signature != "" {
		if err := crypto.VerifyMessage(msg); err != nil {
			return fmt.Errorf("verify message signature: %w", err)
		}
	}

	// Decrypt E2EE content if encrypted
	if msg.Encrypted {
		var localPrivKey ed25519.PrivateKey
		if len(privKey) > 0 && privKey[0] != nil {
			localPrivKey = privKey[0]
		}
		if localPrivKey == nil {
			return errors.New("cannot decrypt E2EE message: local private key not configured")
		}
		decryptedBytes, err := crypto.Decrypt(msg.Content, localPrivKey)
		if err != nil {
			return fmt.Errorf("decrypt E2EE message: %w", err)
		}
		msg.Content = string(decryptedBytes)
		msg.Encrypted = false
	}
	msg.CreatedAt = msg.CreatedAt.UTC()
	storedMsg := storage.StoredMessage{
		ConversationPeerID: remotePeerID,
		Envelope:           msg,
		Status:             message.DeliveryDelivered,
	}
	created, err := store.SaveMessageOnce(ctx, storedMsg)
	if err != nil {
		return fmt.Errorf("persist received message: %w", err)
	}
	if created && notifier != nil {
		notifier(storedMsg)
	}
	if err := connection.SendAck(msg.ID); err != nil {
		return fmt.Errorf("acknowledge persisted message: %w", err)
	}
	return nil
}
