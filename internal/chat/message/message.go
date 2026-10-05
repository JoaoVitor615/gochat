package message

import "time"

// Message is the versioned envelope exchanged between GoChat peers.
type Message struct {
	ID              string    `json:"id,omitempty"`
	SenderPeerID    string    `json:"sender_peer_id,omitempty"`
	RecipientPeerID string    `json:"recipient_peer_id,omitempty"`
	CreatedAt       time.Time `json:"created_at,omitempty"`
	Content         string
	ProtocolVersion int `json:"protocol_version,omitempty"`
}

// DeliveryStatus is local metadata; it is not part of the peer message
// envelope and must not be sent over the network.
type DeliveryStatus string

const (
	// DeliveryPending means the message is persisted locally and awaits sending.
	DeliveryPending DeliveryStatus = "pending"
	// DeliverySent means the transport accepted the write, but no peer ACK arrived.
	DeliverySent DeliveryStatus = "sent"
	// DeliveryDelivered means the recipient acknowledged persisting the message.
	DeliveryDelivered DeliveryStatus = "delivered"
)
