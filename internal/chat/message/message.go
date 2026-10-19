package message

import "time"

const CurrentProtocolVersion = 2

// Message is the versioned envelope exchanged between GoChat peers.
type Message struct {
	ID              string    `json:"id,omitempty"`
	SenderPeerID    string    `json:"sender_peer_id,omitempty"`
	RecipientPeerID string    `json:"recipient_peer_id,omitempty"`
	CreatedAt       time.Time `json:"created_at,omitempty"`
	LamportClock    uint64    `json:"lamport_clock,omitempty"`
	Content         string    `json:"content"`
	Signature       string    `json:"signature,omitempty"`
	Encrypted       bool      `json:"encrypted,omitempty"`
	ProtocolVersion int       `json:"protocol_version,omitempty"`
}

// Compare returns -1 if left < right, 1 if left > right, and 0 if left == right.
// The primary ordering is causal via LamportClock. Tied clocks are ordered by
// CreatedAt (UTC), and finally by message ID as a deterministic tie-breaker.
func Compare(left, right Message) int {
	if left.LamportClock != right.LamportClock {
		if left.LamportClock < right.LamportClock {
			return -1
		}
		return 1
	}
	if !left.CreatedAt.Equal(right.CreatedAt) {
		if left.CreatedAt.Before(right.CreatedAt) {
			return -1
		}
		return 1
	}
	if left.ID < right.ID {
		return -1
	}
	if left.ID > right.ID {
		return 1
	}
	return 0
}

// FrameType identifies the kind of payload carried by a P2P stream.
type FrameType string

const (
	FrameTypeMessage FrameType = "message"
	FrameTypeAck     FrameType = "ack"
)

// Frame is the protocol-level wrapper for a message or its receipt ACK.
type Frame struct {
	Type    FrameType `json:"type"`
	Message *Message  `json:"message,omitempty"`
	Ack     *Ack      `json:"ack,omitempty"`
}

// Ack confirms that the receiver persisted the referenced message.
type Ack struct {
	MessageID string `json:"message_id"`
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
