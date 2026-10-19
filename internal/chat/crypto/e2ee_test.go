package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	"github.com/JoaoVitor615/gochat/internal/chat/message"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

func TestE2EEEncryptDecrypt(t *testing.T) {
	// Generate Bob's key pair and PeerID
	_, bobPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	bobLibp2pPriv, _, err := libp2pcrypto.KeyPairFromStdKey(&bobPriv)
	if err != nil {
		t.Fatal(err)
	}
	bobPeerID, err := peer.IDFromPrivateKey(bobLibp2pPriv)
	if err != nil {
		t.Fatal(err)
	}

	// Alice encrypts a message for Bob using only Bob's PeerID string
	originalMessage := "Hello Bob! This is an end-to-end encrypted message."
	ciphertextBase64, err := Encrypt([]byte(originalMessage), bobPeerID.String())
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	if ciphertextBase64 == originalMessage {
		t.Fatal("ciphertext matches plaintext")
	}

	// Bob decrypts the message using his private key
	decryptedBytes, err := Decrypt(ciphertextBase64, bobPriv)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if string(decryptedBytes) != originalMessage {
		t.Fatalf("decrypted text mismatch! Got %q, want %q", string(decryptedBytes), originalMessage)
	}
}

func TestE2EETamperDetection(t *testing.T) {
	_, bobPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	bobLibp2pPriv, _, err := libp2pcrypto.KeyPairFromStdKey(&bobPriv)
	if err != nil {
		t.Fatal(err)
	}
	bobPeerID, err := peer.IDFromPrivateKey(bobLibp2pPriv)
	if err != nil {
		t.Fatal(err)
	}

	ciphertextBase64, err := Encrypt([]byte("tamper test"), bobPeerID.String())
	if err != nil {
		t.Fatal(err)
	}

	// Tamper with the base64 string
	tampered := []byte(ciphertextBase64)
	if tampered[len(tampered)-5] == 'A' {
		tampered[len(tampered)-5] = 'B'
	} else {
		tampered[len(tampered)-5] = 'A'
	}

	_, err = Decrypt(string(tampered), bobPriv)
	if err == nil {
		t.Fatal("expected decryption error on tampered ciphertext, got nil")
	}
}

func TestSignAndVerifyMessage(t *testing.T) {
	_, alicePriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	aliceLibp2pPriv, _, err := libp2pcrypto.KeyPairFromStdKey(&alicePriv)
	if err != nil {
		t.Fatal(err)
	}
	alicePeerID, err := peer.IDFromPrivateKey(aliceLibp2pPriv)
	if err != nil {
		t.Fatal(err)
	}

	msg := message.Message{
		ID:              "msg-12345",
		SenderPeerID:    alicePeerID.String(),
		RecipientPeerID: "12D3KooWBobFakePeerID",
		LamportClock:    42,
		Content:         "Authentic message from Alice",
	}

	// Sign
	sig := SignMessage(alicePriv, msg)
	msg.Signature = sig

	// Verify - should succeed
	if err := VerifyMessage(msg); err != nil {
		t.Fatalf("signature verification failed: %v", err)
	}

	// Tamper with content - should fail
	tamperedMsg := msg
	tamperedMsg.Content = "Forged message"
	if err := VerifyMessage(tamperedMsg); err == nil {
		t.Fatal("expected signature verification failure on tampered message, got nil")
	}

	// Tamper with sender - should fail
	tamperedSenderMsg := msg
	tamperedSenderMsg.SenderPeerID = "12D3KooWEvilFakePeerID"
	if err := VerifyMessage(tamperedSenderMsg); err == nil {
		t.Fatal("expected signature verification failure on tampered sender, got nil")
	}
}

