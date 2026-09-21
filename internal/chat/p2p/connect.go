package p2p

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

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
