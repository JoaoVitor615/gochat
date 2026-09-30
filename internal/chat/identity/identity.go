package identity

import (
	"crypto/ed25519"
	"errors"
	"fmt"

	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
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
	if err := id.initializePeerID(); err != nil {
		return nil, err
	}

	return id, nil
}

func (id *Identity) initializePeerID() error {
	privateKey, _, err := libp2pcrypto.KeyPairFromStdKey(&id.PrivateKey)
	if err != nil {
		return fmt.Errorf("convert identity for libp2p: %w", err)
	}

	peerID, err := peer.IDFromPrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("derive peer ID: %w", err)
	}

	id.PeerID = peerID.String()
	return nil
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
