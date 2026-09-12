package identity

import (
	"crypto/ed25519"
	"errors"
)

type Identity struct {
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
	PeerID     string
}

func NewIdentity() (*Identity, error) {
	id := &Identity{}
	err := id.initialize()

	if err != nil {
		return nil, err
	}

	return id, nil
}

// generates a new identity and persists its private key.
func (id *Identity) initialize() error {
	exists, err := IdentityExists()
	if err != nil {
		return err
	}

	if exists {
		return id.Load()
	}

	return id.create()
}

func (id *Identity) create() error {
	privateKey, publicKey, err := GenerateKey()
	if err != nil {
		return err
	}

	serializedPrivateKey, err := SerializePrivateKey(privateKey)
	if err != nil {
		return err
	}

	if err := StoreIdentity([]byte(serializedPrivateKey)); err != nil {
		return err
	}

	id.PrivateKey = privateKey
	id.PublicKey = publicKey

	return nil
}

// Load loads the persisted identity.
func (id *Identity) Load() error {
	serializedPrivateKey, err := LoadIdentity()
	if err != nil {
		return err
	}

	privateKey, err := DeserializePrivateKey(string(serializedPrivateKey))
	if err != nil {
		return err
	}

	id.PrivateKey = privateKey
	publicKey, ok := privateKey.Public().(ed25519.PublicKey)
	if !ok {
		return errors.New("failed to derive Ed25519 public key")
	}

	id.PublicKey = publicKey

	return nil
}

// GetPeerID returns the libp2p peer ID associated with the identity.
// It will be populated when libp2p identity integration is added.
func (i *Identity) GetPeerID() string {
	return i.PeerID
}
