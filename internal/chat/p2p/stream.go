package p2p

import (
	"context"
	"errors"
	"fmt"

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

func HandleStream(ctx context.Context, stream network.Stream, store storage.Repository, localPeerID string) error {
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
	_, err = store.SaveMessageOnce(ctx, storage.StoredMessage{
		ConversationPeerID: remotePeerID,
		Envelope:           msg,
		Status:             message.DeliveryDelivered,
	})
	if err != nil {
		return fmt.Errorf("persist received message: %w", err)
	}
	if err := connection.SendAck(msg.ID); err != nil {
		return fmt.Errorf("acknowledge persisted message: %w", err)
	}
	return nil
}
