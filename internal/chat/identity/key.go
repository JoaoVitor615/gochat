package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
)

// GenerateKey creates an Ed25519 key pair.
func GenerateKey() (ed25519.PrivateKey, ed25519.PublicKey, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generate identity key: %w", err)
	}

	return privateKey, publicKey, nil
}

// SerializePrivateKey encodes a private key as base64(PKCS#8).
func SerializePrivateKey(key ed25519.PrivateKey) (string, error) {
	if len(key) != ed25519.PrivateKeySize {
		return "", errors.New("invalid Ed25519 private key")
	}

	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return "", fmt.Errorf("serialize private key: %w", err)
	}

	return base64.RawStdEncoding.EncodeToString(der), nil
}

// DeserializePrivateKey decodes a base64(PKCS#8) private key.
func DeserializePrivateKey(encoded string) (ed25519.PrivateKey, error) {
	der, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode private key: %w", err)
	}

	key, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	privateKey, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, errors.New("key is not an Ed25519 private key")
	}

	return privateKey, nil
}

// SerializePublicKey encodes a public key as base64(PKIX).
func SerializePublicKey(key ed25519.PublicKey) (string, error) {
	if len(key) != ed25519.PublicKeySize {
		return "", errors.New("invalid Ed25519 public key")
	}

	der, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return "", fmt.Errorf("serialize public key: %w", err)
	}

	return base64.RawStdEncoding.EncodeToString(der), nil
}

// DeserializePublicKey decodes a base64(PKIX) public key.
func DeserializePublicKey(encoded string) (ed25519.PublicKey, error) {
	der, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode public key: %w", err)
	}

	key, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}

	publicKey, ok := key.(ed25519.PublicKey)
	if !ok {
		return nil, errors.New("key is not an Ed25519 public key")
	}

	return publicKey, nil
}

// Sign signs message with the identity's private key.
func Sign(key ed25519.PrivateKey, message []byte) []byte {
	return ed25519.Sign(key, message)
}

// Verify verifies a signature made by the corresponding public key.
func Verify(key ed25519.PublicKey, message, signature []byte) bool {
	return ed25519.Verify(key, message, signature)
}
