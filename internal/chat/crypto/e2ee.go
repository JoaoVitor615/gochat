package crypto

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math/big"

	"github.com/JoaoVitor615/gochat/internal/chat/message"
	"github.com/libp2p/go-libp2p/core/peer"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

var (
	// pCurve25519 is the prime 2^255 - 19 for Curve25519/Ed25519 field arithmetic.
	pCurve25519, _ = new(big.Int).SetString("7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffed", 16)
	hkdfInfo       = []byte("gochat-e2ee-v1")
)

const (
	x25519KeySize = 32
	nonceSize     = chacha20poly1305.NonceSizeX // 24 bytes
	overheadSize  = x25519KeySize + nonceSize   // 56 bytes
)

// PublicKeyFromPeerID extracts the Ed25519 public key embedded in a libp2p PeerID.
func PublicKeyFromPeerID(peerIDString string) (ed25519.PublicKey, error) {
	peerID, err := peer.Decode(peerIDString)
	if err != nil {
		return nil, fmt.Errorf("decode peer ID: %w", err)
	}

	pubKey, err := peerID.ExtractPublicKey()
	if err != nil {
		return nil, fmt.Errorf("extract public key from peer ID: %w", err)
	}

	raw, err := pubKey.Raw()
	if err != nil {
		return nil, fmt.Errorf("read raw public key bytes: %w", err)
	}

	if len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key length: expected %d, got %d", ed25519.PublicKeySize, len(raw))
	}

	return ed25519.PublicKey(raw), nil
}

// Ed25519ToX25519PublicKey converts an Ed25519 public key (Edwards y-coordinate)
// to an X25519 public key (Montgomery u-coordinate) using u = (1 + y) / (1 - y) mod p.
func Ed25519ToX25519PublicKey(edPub ed25519.PublicKey) ([]byte, error) {
	if len(edPub) != ed25519.PublicKeySize {
		return nil, errors.New("invalid Ed25519 public key size")
	}

	yBytes := make([]byte, 32)
	copy(yBytes, edPub)
	yBytes[31] &= 0x7F // Clear sign bit

	// Reverse to big-endian for big.Int
	for i := 0; i < 16; i++ {
		yBytes[i], yBytes[31-i] = yBytes[31-i], yBytes[i]
	}
	y := new(big.Int).SetBytes(yBytes)

	one := big.NewInt(1)
	num := new(big.Int).Mod(new(big.Int).Add(one, y), pCurve25519)
	den := new(big.Int).Mod(new(big.Int).Sub(one, y), pCurve25519)
	invDen := new(big.Int).ModInverse(den, pCurve25519)
	if invDen == nil {
		return nil, errors.New("failed to calculate modular inverse for Montgomery conversion")
	}
	u := new(big.Int).Mod(new(big.Int).Mul(num, invDen), pCurve25519)

	uBigEndian := u.Bytes()
	uLittleEndian := make([]byte, 32)
	copy(uLittleEndian[32-len(uBigEndian):], uBigEndian)
	for i := 0; i < 16; i++ {
		uLittleEndian[i], uLittleEndian[31-i] = uLittleEndian[31-i], uLittleEndian[i]
	}

	return uLittleEndian, nil
}

// Ed25519ToX25519PrivateKey derives an X25519 private key from an Ed25519 private key.
func Ed25519ToX25519PrivateKey(edPriv ed25519.PrivateKey) []byte {
	seed := edPriv.Seed()
	h := sha512.Sum512(seed)
	h[0] &= 248
	h[31] &= 127
	h[31] |= 64
	return h[:32]
}

// Encrypt encrypts a plaintext message for the specified recipient peer ID using
// ephemeral X25519 ECDH + HKDF-SHA256 + XChaCha20-Poly1305.
func Encrypt(plaintext []byte, recipientPeerID string) (string, error) {
	edPub, err := PublicKeyFromPeerID(recipientPeerID)
	if err != nil {
		return "", fmt.Errorf("resolve recipient public key: %w", err)
	}

	x25519PubBytes, err := Ed25519ToX25519PublicKey(edPub)
	if err != nil {
		return "", fmt.Errorf("convert recipient key to X25519: %w", err)
	}

	recipientX25519Pub, err := ecdh.X25519().NewPublicKey(x25519PubBytes)
	if err != nil {
		return "", fmt.Errorf("create X25519 public key: %w", err)
	}

	// Generate ephemeral X25519 key pair for this message
	ephemeralPriv, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return "", fmt.Errorf("generate ephemeral key: %w", err)
	}
	ephemeralPub := ephemeralPriv.PublicKey().Bytes()

	// Compute shared secret
	sharedSecret, err := ephemeralPriv.ECDH(recipientX25519Pub)
	if err != nil {
		return "", fmt.Errorf("compute shared secret: %w", err)
	}

	// Derive symmetric key
	symmetricKey, err := deriveSymmetricKey(sharedSecret)
	if err != nil {
		return "", fmt.Errorf("derive symmetric key: %w", err)
	}

	aead, err := chacha20poly1305.NewX(symmetricKey)
	if err != nil {
		return "", fmt.Errorf("initialize cipher: %w", err)
	}

	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := aead.Seal(nil, nonce, plaintext, nil)

	// Combine: [ephemeralPub (32B)] + [nonce (24B)] + [ciphertext + tag]
	payload := make([]byte, 0, len(ephemeralPub)+len(nonce)+len(ciphertext))
	payload = append(payload, ephemeralPub...)
	payload = append(payload, nonce...)
	payload = append(payload, ciphertext...)

	return base64.StdEncoding.EncodeToString(payload), nil
}

// Decrypt decrypts an E2EE base64-encoded ciphertext using the local Ed25519 private key.
func Decrypt(encryptedBase64 string, myPrivKey ed25519.PrivateKey) ([]byte, error) {
	payload, err := base64.StdEncoding.DecodeString(encryptedBase64)
	if err != nil {
		return nil, fmt.Errorf("decode base64 payload: %w", err)
	}

	if len(payload) < overheadSize+chacha20poly1305.Overhead {
		return nil, errors.New("encrypted payload too short")
	}

	ephemeralPubBytes := payload[:x25519KeySize]
	nonce := payload[x25519KeySize:overheadSize]
	ciphertext := payload[overheadSize:]

	// Derive local X25519 private key
	x25519PrivBytes := Ed25519ToX25519PrivateKey(myPrivKey)
	x25519Priv, err := ecdh.X25519().NewPrivateKey(x25519PrivBytes)
	if err != nil {
		return nil, fmt.Errorf("create local X25519 private key: %w", err)
	}

	ephemeralPub, err := ecdh.X25519().NewPublicKey(ephemeralPubBytes)
	if err != nil {
		return nil, fmt.Errorf("parse sender ephemeral public key: %w", err)
	}

	sharedSecret, err := x25519Priv.ECDH(ephemeralPub)
	if err != nil {
		return nil, fmt.Errorf("compute shared secret: %w", err)
	}

	symmetricKey, err := deriveSymmetricKey(sharedSecret)
	if err != nil {
		return nil, fmt.Errorf("derive symmetric key: %w", err)
	}

	aead, err := chacha20poly1305.NewX(symmetricKey)
	if err != nil {
		return nil, fmt.Errorf("initialize cipher: %w", err)
	}

	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt payload: %w", err)
	}

	return plaintext, nil
}

func deriveSymmetricKey(sharedSecret []byte) ([]byte, error) {
	kdf := hkdf.New(sha256.New, sharedSecret, nil, hkdfInfo)
	key := make([]byte, chacha20poly1305.KeySize)
	if _, err := io.ReadFull(kdf, key); err != nil {
		return nil, err
	}
	return key, nil
}

// CanonicalMessagePayload creates a deterministic byte sequence representing the message envelope for signing.
func CanonicalMessagePayload(msg message.Message) []byte {
	return []byte(fmt.Sprintf("%s|%s|%s|%d|%d|%s",
		msg.ID,
		msg.SenderPeerID,
		msg.RecipientPeerID,
		msg.CreatedAt.UnixNano(),
		msg.LamportClock,
		msg.Content,
	))
}

// SignMessage signs the message payload using the sender's Ed25519 private key.
func SignMessage(priv ed25519.PrivateKey, msg message.Message) string {
	payload := CanonicalMessagePayload(msg)
	sig := ed25519.Sign(priv, payload)
	return base64.StdEncoding.EncodeToString(sig)
}

// VerifyMessage verifies the Ed25519 signature of a message using the sender's PeerID.
func VerifyMessage(msg message.Message) error {
	if msg.Signature == "" {
		return errors.New("message is missing digital signature")
	}
	sig, err := base64.StdEncoding.DecodeString(msg.Signature)
	if err != nil {
		return fmt.Errorf("decode message signature: %w", err)
	}
	if len(sig) != ed25519.SignatureSize {
		return errors.New("invalid signature length")
	}
	pubKey, err := PublicKeyFromPeerID(msg.SenderPeerID)
	if err != nil {
		return fmt.Errorf("resolve sender public key: %w", err)
	}
	payload := CanonicalMessagePayload(msg)
	if !ed25519.Verify(pubKey, payload, sig) {
		return errors.New("invalid message signature: message altered or sender forged")
	}
	return nil
}

