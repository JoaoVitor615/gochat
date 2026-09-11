package identity

import (
	"crypto/ed25519"
)

type Identity struct {
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
	PeerID     string
}

// Create generates a new identity and persists its private key.
func Create() (*Identity, error) {
	exists, err := IdentityExists()
	if err != nil {
		return nil, err
	}
	if exists {
		return Load()
	}

	privateKey, publicKey, err := GenerateKey()
	if err != nil {
		return nil, err
	}

	serializedPrivateKey, err := SerializePrivateKey(privateKey)
	if err != nil {
		return nil, err
	}

	if err := StoreIdentity([]byte(serializedPrivateKey)); err != nil {
		return nil, err
	}

	return &Identity{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}, nil
}

// Load loads the persisted identity.
func Load() (*Identity, error) {
	serializedPrivateKey, err := LoadIdentity()
	if err != nil {
		return nil, err
	}

	privateKey, err := DeserializePrivateKey(string(serializedPrivateKey))
	if err != nil {
		return nil, err
	}

	return &Identity{
		PrivateKey: privateKey,
		PublicKey:  privateKey.Public().(ed25519.PublicKey),
	}, nil
}

// GetPeerID returns the libp2p peer ID associated with the identity.
// It will be populated when libp2p identity integration is added.
func (i *Identity) GetPeerID() string {
	return i.PeerID
}
