package p2p

import (
	"crypto/ed25519"
	"errors"
	"fmt"

	"github.com/JoaoVitor615/gochat/internal/identity"
	libp2p "github.com/libp2p/go-libp2p"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
)

// NewHost creates a libp2p host using the application's persisted identity.
// A port of 0 asks the operating system to choose an available UDP port.
func NewHost(id *identity.Identity, port int) (host.Host, error) {
	if id == nil {
		return nil, errors.New("identity is required")
	}
	if len(id.PrivateKey) != ed25519.PrivateKeySize {
		return nil, errors.New("identity has an invalid Ed25519 private key")
	}
	if port < 0 || port > 65535 {
		return nil, fmt.Errorf("invalid listen port: %d", port)
	}

	privateKey, _, err := libp2pcrypto.KeyPairFromStdKey(id.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("convert identity for libp2p: %w", err)
	}

	h, err := libp2p.New(
		libp2p.Identity(privateKey),
		libp2p.ListenAddrStrings(
			fmt.Sprintf("/ip4/127.0.0.1/udp/%d/quic-v1", port),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create libp2p host: %w", err)
	}

	return h, nil
}
