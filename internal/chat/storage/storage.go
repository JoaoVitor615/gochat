// Package storage provides the local persistence boundary for chat data.
package storage

import (
	"context"
	"errors"
	"time"

	"github.com/JoaoVitor615/gochat/internal/chat/message"
)

var (
	ErrNotFound        = errors.New("storage: record not found")
	ErrMessageIDExists = errors.New("storage: message ID conflicts with an existing message")
	ErrInvalidRecord   = errors.New("storage: invalid record")
)

// Peer is a locally saved contact. Addresses are a cache and may become stale.
type Peer struct {
	PeerID      string    `json:"peer_id"`
	DisplayName string    `json:"display_name,omitempty"`
	AddedAt     time.Time `json:"added_at"`
	LastSeenAt  time.Time `json:"last_seen_at,omitempty"`
	Addresses   []string  `json:"addresses,omitempty"`
}

// StoredMessage combines the wire envelope with metadata that stays local.
type StoredMessage struct {
	ConversationPeerID string                 `json:"conversation_peer_id"`
	Envelope           message.Message        `json:"envelope"`
	Status             message.DeliveryStatus `json:"status"`
}

// OutboxEntry is a message retained for delivery or delivery confirmation.
type OutboxEntry struct {
	ConversationPeerID string          `json:"conversation_peer_id"`
	Message            message.Message `json:"message"`
}

// Repository defines persistence operations used by chat services.
type Repository interface {
	Close() error
	SavePeer(context.Context, Peer) error
	GetPeer(context.Context, string) (Peer, error)
	ListPeers(context.Context) ([]Peer, error)
	DeletePeer(context.Context, string) error
	SaveMessage(context.Context, StoredMessage) error
	SaveMessageOnce(context.Context, StoredMessage) (bool, error)
	GetMessage(context.Context, string) (StoredMessage, error)
	ListMessages(context.Context, string, time.Time, int) ([]StoredMessage, error)
	UpdateMessageStatus(context.Context, string, message.DeliveryStatus) error
	QueueMessage(context.Context, string, message.Message) error
	ListOutbox(context.Context, int) ([]OutboxEntry, error)
}
