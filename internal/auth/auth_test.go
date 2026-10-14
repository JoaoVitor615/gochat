package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

func generateTestPeer(t *testing.T) (ed25519.PrivateKey, string) {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate ed25519 key: %v", err)
	}
	libp2pPriv, _, err := libp2pcrypto.KeyPairFromStdKey(&priv)
	if err != nil {
		t.Fatalf("convert to libp2p key: %v", err)
	}
	peerID, err := peer.IDFromPrivateKey(libp2pPriv)
	if err != nil {
		t.Fatalf("derive peer ID: %v", err)
	}
	return priv, peerID.String()
}

func TestVerify_ValidSignature(t *testing.T) {
	priv, peerID := generateTestPeer(t)
	now := time.Now().Unix()

	payload := HeartbeatPayload(peerID, now, []string{"/ip4/127.0.0.1/udp/4001/quic-v1"})
	sig := Sign(priv, payload)

	if err := Verify(peerID, payload, sig, now); err != nil {
		t.Fatalf("expected valid signature, got: %v", err)
	}
}

func TestVerify_SpoofingRejected(t *testing.T) {
	attackerPriv, _ := generateTestPeer(t)
	_, victimPeerID := generateTestPeer(t)
	now := time.Now().Unix()

	// Attacker attempts to sign a payload claiming to be the victim
	payload := HeartbeatPayload(victimPeerID, now, []string{"/ip4/1.2.3.4/udp/4001/quic-v1"})
	sig := Sign(attackerPriv, payload)

	err := Verify(victimPeerID, payload, sig, now)
	if err == nil {
		t.Fatalf("expected signature verification failure for spoofed peer ID, but it passed")
	}
}

func TestVerify_TamperedPayloadRejected(t *testing.T) {
	priv, peerID := generateTestPeer(t)
	now := time.Now().Unix()

	payload := HeartbeatPayload(peerID, now, []string{"/ip4/127.0.0.1/udp/4001/quic-v1"})
	sig := Sign(priv, payload)

	tamperedPayload := HeartbeatPayload(peerID, now, []string{"/ip4/6.6.6.6/udp/4001/quic-v1"})
	err := Verify(peerID, tamperedPayload, sig, now)
	if err == nil {
		t.Fatalf("expected signature verification failure for tampered payload, but it passed")
	}
}

func TestVerify_TimestampExpired(t *testing.T) {
	priv, peerID := generateTestPeer(t)
	expiredTime := time.Now().Add(-3 * time.Minute).Unix()

	payload := InvitePayload(peerID, expiredTime)
	sig := Sign(priv, payload)

	err := Verify(peerID, payload, sig, expiredTime)
	if err == nil || err != ErrTimestampExpired {
		t.Fatalf("expected ErrTimestampExpired, got: %v", err)
	}
}

func TestVerify_TimestampInFuture(t *testing.T) {
	priv, peerID := generateTestPeer(t)
	futureTime := time.Now().Add(2 * time.Minute).Unix()

	payload := InvitePayload(peerID, futureTime)
	sig := Sign(priv, payload)

	err := Verify(peerID, payload, sig, futureTime)
	if err == nil || err != ErrTimestampFuture {
		t.Fatalf("expected ErrTimestampFuture, got: %v", err)
	}
}

func TestCanonicalPayloads(t *testing.T) {
	now := int64(1700000000)
	peerID := "12D3KooWTestPeer"

	hb1 := HeartbeatPayload(peerID, now, []string{"/ip4/2.2.2.2/udp/4001/quic-v1", "/ip4/1.1.1.1/udp/4001/quic-v1"})
	hb2 := HeartbeatPayload(peerID, now, []string{"/ip4/1.1.1.1/udp/4001/quic-v1", "/ip4/2.2.2.2/udp/4001/quic-v1"})

	if string(hb1) != string(hb2) {
		t.Fatalf("expected address sorting in heartbeat payload to produce identical bytes, got %q vs %q", hb1, hb2)
	}

	ack1 := AcknowledgeAcceptancesPayload(peerID, now, []string{"b", "a"})
	ack2 := AcknowledgeAcceptancesPayload(peerID, now, []string{"a", "b"})
	if string(ack1) != string(ack2) {
		t.Fatalf("expected invite ID sorting to produce identical bytes, got %q vs %q", ack1, ack2)
	}
}
