package p2p

import (
	"crypto/ed25519"
	"errors"
	"fmt"

	"github.com/JoaoVitor615/gochat/internal/chat/identity"
	libp2p "github.com/libp2p/go-libp2p"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
)

var PackageName = "[p2p]"
var ErrInvalidPort = errors.New(PackageName + " invalid listen port")
var ErrIdentityRequired = errors.New(PackageName + " identity is required")
var ErrInvalidPrivateKey = errors.New(PackageName + " invalid Ed25519 private key")
var ErrConvertIdentity = errors.New(PackageName + " failed to convert identity for libp2p")
var ErrPeerIDMismatch = errors.New(PackageName + " peer ID does not match identity")

// NewHost creates a libp2p host using the application's persisted identity.
// A port of 0 asks the operating system to choose an available UDP port.
func NewHost(id *identity.Identity, port int) (host.Host, error) {
	if id == nil {
		return nil, ErrIdentityRequired
	}
	if len(id.PrivateKey) != ed25519.PrivateKeySize {
		return nil, ErrInvalidPrivateKey
	}
	if port < 0 || port > 65535 {
		return nil, ErrInvalidPort
	}

	privateKey, _, err := libp2pcrypto.KeyPairFromStdKey(&id.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConvertIdentity, err)
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

	if h.ID().String() != id.PeerID {
		_ = h.Close()
		return nil, fmt.Errorf("%w: identity has %q but host derived %q", ErrPeerIDMismatch, id.PeerID, h.ID())
	}

	return h, nil
}
