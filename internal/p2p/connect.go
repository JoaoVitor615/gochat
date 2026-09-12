package p2p

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

func Connect(ctx context.Context, h host.Host, addr string) error {
	id, err := peer.Decode(h.ID().String())
	if err != nil {
		return fmt.Errorf("invalid peer ID: %w", err)
	}

	maddr, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return fmt.Errorf("invalid multiaddr: %w", err)
	}

	info := peer.AddrInfo{
		ID:    id,
		Addrs: []multiaddr.Multiaddr{maddr},
	}

	if err := h.Connect(ctx, info); err != nil {
		return fmt.Errorf("connect to peer: %w", err)
	}

	return nil
}
