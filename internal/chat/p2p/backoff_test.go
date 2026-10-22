package p2p_test

import (
	"context"
	"testing"
	"time"

	"github.com/JoaoVitor615/gochat/internal/chat/p2p"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/net/swarm"
	ma "github.com/multiformats/go-multiaddr"
)

func TestClearBackoff(t *testing.T) {
	// Should not panic on nil host
	p2p.ClearBackoff(nil, "")

	id := createTestIdentity(t)

	h, err := p2p.NewHost(id, 0)
	if err != nil {
		t.Fatalf("start host: %v", err)
	}
	defer h.Close()

	dummyPeerID, _ := peer.Decode("12D3KooWTestPeerDummy12345678901234567890123456789012")
	dummyAddr, _ := ma.NewMultiaddr("/ip4/192.0.2.1/udp/4001/quic-v1")

	sw, ok := h.Network().(*swarm.Swarm)
	if !ok {
		t.Skip("network is not swarm.Swarm")
	}

	// Add dummy peer to backoff
	sw.Backoff().AddBackoff(dummyPeerID, dummyAddr)
	if !sw.Backoff().Backoff(dummyPeerID, dummyAddr) {
		t.Fatalf("expected peer to be in backoff")
	}

	// Clear backoff using helper
	p2p.ClearBackoff(h, dummyPeerID)
	if sw.Backoff().Backoff(dummyPeerID, dummyAddr) {
		t.Fatalf("expected peer backoff to be cleared")
	}
}

func TestPingPeer_ClearsBackoff(t *testing.T) {
	aliceID := createTestIdentity(t)
	bobID := createTestIdentity(t)

	aliceHost, err := p2p.NewHost(aliceID, 0)
	if err != nil {
		t.Fatalf("start alice: %v", err)
	}
	defer aliceHost.Close()

	bobHost, err := p2p.NewHost(bobID, 0)
	if err != nil {
		t.Fatalf("start bob: %v", err)
	}
	defer bobHost.Close()

	p2p.RegisterPresenceHandler(bobHost, nil)

	bobPeerID, _ := peer.Decode(bobID.PeerID)
	bobAddrInfo := peer.AddrInfo{ID: bobPeerID, Addrs: bobHost.Addrs()}
	aliceHost.Peerstore().AddAddrs(bobPeerID, bobAddrInfo.Addrs, time.Hour)

	// Artificially put Bob into Alice's dial backoff
	sw, ok := aliceHost.Network().(*swarm.Swarm)
	if ok {
		for _, a := range bobAddrInfo.Addrs {
			sw.Backoff().AddBackoff(bobPeerID, a)
		}
	}

	// PingPeer should clear backoff and succeed
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := p2p.PingPeer(ctx, aliceHost, bobPeerID); err != nil {
		t.Fatalf("ping failed: %v", err)
	}
}
