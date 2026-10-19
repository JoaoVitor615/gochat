package identity

import (
	"testing"

	"github.com/JoaoVitor615/gochat/internal/chat/crypto"
)

func TestIdentityCryptoIntegration(t *testing.T) {
	id, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}

	xPub, err := crypto.Ed25519ToX25519PublicKey(id.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	if len(xPub) != 32 {
		t.Fatalf("expected 32 bytes X25519 pubkey, got %d", len(xPub))
	}

	xPriv := crypto.Ed25519ToX25519PrivateKey(id.PrivateKey)
	if len(xPriv) != 32 {
		t.Fatalf("expected 32 bytes X25519 privkey, got %d", len(xPriv))
	}
}
