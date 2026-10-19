package p2p_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/JoaoVitor615/gochat/internal/chat/identity"
	"github.com/JoaoVitor615/gochat/internal/chat/p2p"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

func createTestIdentity(t *testing.T) *identity.Identity {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	libPriv, _, err := libp2pcrypto.KeyPairFromStdKey(&priv)
	if err != nil {
		t.Fatalf("convert key: %v", err)
	}
	pID, err := peer.IDFromPrivateKey(libPriv)
	if err != nil {
		t.Fatalf("derive peer ID: %v", err)
	}
	return &identity.Identity{
		PrivateKey: priv,
		PublicKey:  pub,
		PeerID:     pID.String(),
	}
}

func TestPresence_PingPong(t *testing.T) {
	aliceID := createTestIdentity(t)
	bobID := createTestIdentity(t)

	aliceHost, err := p2p.NewHost(aliceID, 0)
	if err != nil {
		t.Fatalf("start alice host: %v", err)
	}
	defer aliceHost.Close()

	bobHost, err := p2p.NewHost(bobID, 0)
	if err != nil {
		t.Fatalf("start bob host: %v", err)
	}
	defer bobHost.Close()

	bobPingReceived := make(chan string, 1)

	// Bob listens for presence pings
	p2p.RegisterPresenceHandler(bobHost, func(remotePeerID string) {
		bobPingReceived <- remotePeerID
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Connect Alice to Bob
	bobAddrsInfo, err := peer.AddrInfoToP2pAddrs(&peer.AddrInfo{
		ID:    bobHost.ID(),
		Addrs: bobHost.Addrs(),
	})
	if err != nil {
		t.Fatalf("bob addr info to p2p addrs: %v", err)
	}
	addrInfo, _ := peer.AddrInfoFromP2pAddr(bobAddrsInfo[0])
	if err := aliceHost.Connect(ctx, *addrInfo); err != nil {
		t.Fatalf("connect alice to bob: %v", err)
	}

	// Alice pings Bob
	bobPeerID, _ := peer.Decode(bobID.PeerID)
	if err := p2p.PingPeer(ctx, aliceHost, bobPeerID); err != nil {
		t.Fatalf("alice ping bob failed: %v", err)
	}

	select {
	case remote := <-bobPingReceived:
		if remote != aliceID.PeerID {
			t.Errorf("bob received ping from %s, want %s", remote, aliceID.PeerID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for bob to receive presence ping")
	}
}
