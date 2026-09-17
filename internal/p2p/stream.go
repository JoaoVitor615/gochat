package p2p

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

const ProtocolID = "/gochat/1.0.0"

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

func HandleStream(stream network.Stream) {
	connection := newConnection(stream)
	defer connection.Close()

	msg, err := connection.Receive()
	if err != nil {
		fmt.Println("Error receiving message")
		return
	}

	fmt.Println("Received:", msg.Content)
}
