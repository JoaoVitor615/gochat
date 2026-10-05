package p2p

import (
	"context"
	"fmt"
	"time"

	"github.com/JoaoVitor615/gochat/internal/chat/message"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

const messageAckTimeout = 15 * time.Second

// Connect opens a GoChat protocol stream to a known peer.
func Connect(ctx context.Context, h host.Host, peerID peer.ID) (*Connection, error) {
	if ctx == nil {
		return nil, fmt.Errorf("connect: context is required")
	}
	if h == nil {
		return nil, fmt.Errorf("connect: host is required")
	}
	if err := peerID.Validate(); err != nil {
		return nil, fmt.Errorf("connect: invalid peer ID: %w", err)
	}

	stream, err := OpenStream(ctx, h, peerID)
	if err != nil {
		return nil, err
	}

	return newConnection(stream), nil
}

// SendMessageAndWaitForAck sends one message frame and waits for an ACK with
// the same message ID. written reports whether the complete frame was written.
func SendMessageAndWaitForAck(ctx context.Context, h host.Host, peerID peer.ID, msg message.Message) (written bool, err error) {
	if ctx == nil {
		return false, fmt.Errorf("send message: context is required")
	}
	if h == nil {
		return false, fmt.Errorf("send message: host is required")
	}
	if err := peerID.Validate(); err != nil {
		return false, fmt.Errorf("send message: invalid peer ID: %w", err)
	}
	if msg.ID == "" || msg.SenderPeerID != h.ID().String() || msg.RecipientPeerID != peerID.String() || msg.ProtocolVersion != message.CurrentProtocolVersion || msg.CreatedAt.IsZero() {
		return false, fmt.Errorf("send message: invalid message envelope")
	}
	stream, err := OpenStream(ctx, h, peerID)
	if err != nil {
		return false, err
	}
	defer func() { _ = stream.Close() }()

	deadline := time.Now().Add(messageAckTimeout)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}
	if err := stream.SetDeadline(deadline); err != nil {
		return false, fmt.Errorf("set message stream deadline: %w", err)
	}
	stopCancel := context.AfterFunc(ctx, func() { _ = stream.Reset() })
	defer stopCancel()

	connection := newConnection(stream)
	if err := connection.Send(msg); err != nil {
		return false, fmt.Errorf("send message frame: %w", err)
	}
	frame, err := connection.ReceiveFrame()
	if err != nil {
		return true, fmt.Errorf("wait for message ACK: %w", err)
	}
	if frame.Type != message.FrameTypeAck || frame.Ack == nil || frame.Ack.MessageID != msg.ID {
		return true, fmt.Errorf("wait for message ACK: %w", ErrUnexpectedFrame)
	}
	return true, nil
}
