package p2p

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"strings"
	"syscall"

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
var ErrInvalidPortRange = errors.New(PackageName + " invalid listen port range")
var ErrNoAvailablePort = errors.New(PackageName + " no available listen port in range")

const (
	DefaultPortRangeStart = 4001
	DefaultPortRangeEnd   = 4010
)

// NewHostOnFirstAvailablePort tries each UDP port in the configured range and
// returns a host on the first port that libp2p can bind.
func NewHostOnFirstAvailablePort(id *identity.Identity) (host.Host, error) {
	if DefaultPortRangeStart < 1 || DefaultPortRangeEnd > 65535 || DefaultPortRangeStart > DefaultPortRangeEnd {
		return nil, ErrInvalidPortRange
	}

	for port := DefaultPortRangeStart; port <= DefaultPortRangeEnd; port++ {
		h, err := NewHost(id, port)
		if err == nil {
			return h, nil
		}
		if !isPortAlreadyInUse(err) {
			return nil, err
		}
	}

	return nil, fmt.Errorf("%w: UDP ports %d-%d", ErrNoAvailablePort, DefaultPortRangeStart, DefaultPortRangeEnd)
}

func isPortAlreadyInUse(err error) bool {
	return errors.Is(err, syscall.EADDRINUSE) || strings.Contains(err.Error(), syscall.EADDRINUSE.Error())
}

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
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", port)),
		libp2p.EnableAutoNATv2(),
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
