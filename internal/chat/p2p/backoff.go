package p2p

import (
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/net/swarm"
)

// ClearBackoff clears any libp2p dial backoff tracking for the given peer ID.
// This allows immediate reconnection attempts when peer addresses are refreshed
// or when retrying delivery to a peer that was previously offline.
func ClearBackoff(h host.Host, peerID peer.ID) {
	if h == nil {
		return
	}
	if s, ok := h.Network().(*swarm.Swarm); ok && s != nil {
		if b := s.Backoff(); b != nil {
			b.Clear(peerID)
		}
	}
}
