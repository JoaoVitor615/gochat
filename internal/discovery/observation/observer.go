package observation

import (
	"bufio"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	ma "github.com/multiformats/go-multiaddr"
)

const (
	DefaultListenPort = 4001
	ObservationTTL    = 2 * time.Minute
	maxObservedPeers  = 10000
	identityFileName  = "identity.key"
)

const observationProtocol protocol.ID = "/gochat/address-observation/1.0.0"
const observationRequest = "observe\n"

var (
	ErrUnavailable       = errors.New("public address observer is not configured")
	ErrInvalidPublicAddr = errors.New("invalid observer public multiaddr")
)

type observedAddress struct {
	address      ma.Multiaddr
	connectionID string
	updatedAt    time.Time
	connected    bool
}

// Observer is a public libp2p host used only to observe the UDP/QUIC source
// address of connecting peers. It does not register a GoChat protocol handler.
type Observer struct {
	host       host.Host
	publicAddr ma.Multiaddr
	notifier   *network.NotifyBundle
	mu         sync.RWMutex
	observed   map[peer.ID]observedAddress
	closed     bool
}

// New starts the observer. The identity is persisted at identityDir so clients
// can pin its Peer ID in their configuration across container restarts.
func New(publicAddress, identityDir string) (*Observer, error) {
	if publicAddress == "" {
		return &Observer{}, nil
	}

	publicAddr, err := ma.NewMultiaddr(publicAddress)
	if err != nil || !validObserverPublicAddr(publicAddr) || containsPeerComponent(publicAddr) {
		return nil, fmt.Errorf("%w: expected an IPv4 or dns4 address such as /ip4/203.0.113.10/udp/4001/quic-v1 without /p2p", ErrInvalidPublicAddr)
	}
	port, _ := publicAddr.ValueForProtocol(ma.P_UDP)
	if port != strconv.Itoa(DefaultListenPort) {
		return nil, fmt.Errorf("%w: observer must use UDP port %d", ErrInvalidPublicAddr, DefaultListenPort)
	}
	privateKey, err := loadOrCreateIdentity(identityDir)
	if err != nil {
		return nil, err
	}
	return newWithListenAddress(publicAddr, privateKey, fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", DefaultListenPort))
}

func newWithListenAddress(publicAddr ma.Multiaddr, privateKey libp2pcrypto.PrivKey, listenAddress string) (*Observer, error) {
	h, err := libp2p.New(
		libp2p.Identity(privateKey),
		libp2p.ListenAddrStrings(listenAddress),
		libp2p.AddrsFactory(func(addresses []ma.Multiaddr) []ma.Multiaddr {
			return ma.Unique(append(addresses, publicAddr))
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("start public address observer: %w", err)
	}

	o := &Observer{
		host:       h,
		publicAddr: publicAddr,
		observed:   make(map[peer.ID]observedAddress),
	}
	o.notifier = &network.NotifyBundle{
		ConnectedF:    o.onConnected,
		DisconnectedF: o.onDisconnected,
	}
	h.Network().Notify(o.notifier)
	h.SetStreamHandler(observationProtocol, o.handleObservation)
	return o, nil
}

func (o *Observer) Enabled() bool {
	return o != nil && o.host != nil
}

func (o *Observer) Address() (string, error) {
	if !o.Enabled() {
		return "", ErrUnavailable
	}
	addresses, err := peer.AddrInfoToP2pAddrs(&peer.AddrInfo{ID: o.host.ID(), Addrs: []ma.Multiaddr{o.publicAddr}})
	if err != nil {
		return "", fmt.Errorf("format observer address: %w", err)
	}
	return addresses[0].String(), nil
}

func (o *Observer) handleObservation(stream network.Stream) {
	defer stream.Close()
	_ = stream.SetDeadline(time.Now().Add(5 * time.Second))
	request, err := bufio.NewReader(io.LimitReader(stream, int64(len(observationRequest)))).ReadString('\n')
	if err != nil || request != observationRequest {
		_ = stream.Reset()
		return
	}
	o.mu.RLock()
	observation, ok := o.observed[stream.Conn().RemotePeer()]
	if ok && !observation.connected && time.Since(observation.updatedAt) > ObservationTTL {
		ok = false
	}
	o.mu.RUnlock()
	if !ok {
		_, _ = io.WriteString(stream, "\n")
		return
	}
	_, _ = io.WriteString(stream, observation.address.String()+"\n")
}

func loadOrCreateIdentity(identityDir string) (libp2pcrypto.PrivKey, error) {
	if identityDir == "" {
		return nil, errors.New("observer identity directory is required")
	}
	if err := os.MkdirAll(identityDir, 0o700); err != nil {
		return nil, fmt.Errorf("create observer identity directory: %w", err)
	}
	path := filepath.Join(identityDir, identityFileName)
	serialized, err := os.ReadFile(path)
	if err == nil {
		key, err := libp2pcrypto.UnmarshalPrivateKey(serialized)
		if err != nil {
			return nil, fmt.Errorf("read observer identity: %w", err)
		}
		return key, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read observer identity: %w", err)
	}
	key, _, err := libp2pcrypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate observer identity: %w", err)
	}
	serialized, err = libp2pcrypto.MarshalPrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("serialize observer identity: %w", err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, fmt.Errorf("create observer identity: %w", err)
	}
	if _, err := file.Write(serialized); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("persist observer identity: %w", err)
	}
	if err := file.Close(); err != nil {
		return nil, fmt.Errorf("close observer identity file: %w", err)
	}
	return key, nil
}

func (o *Observer) Close() error {
	if !o.Enabled() {
		return nil
	}
	o.host.Network().StopNotify(o.notifier)
	o.mu.Lock()
	o.closed = true
	o.mu.Unlock()
	return o.host.Close()
}

func (o *Observer) onConnected(_ network.Network, conn network.Conn) {
	if conn == nil || conn.RemotePeer() == "" || !validQUICAddr(conn.RemoteMultiaddr()) {
		return
	}

	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return
	}
	now := time.Now()
	for peerID, observation := range o.observed {
		if !observation.connected && now.Sub(observation.updatedAt) > ObservationTTL {
			delete(o.observed, peerID)
		}
	}
	if _, exists := o.observed[conn.RemotePeer()]; !exists && len(o.observed) >= maxObservedPeers {
		return
	}
	o.observed[conn.RemotePeer()] = observedAddress{
		address:      conn.RemoteMultiaddr(),
		connectionID: conn.ID(),
		updatedAt:    now,
		connected:    true,
	}
}

func (o *Observer) onDisconnected(_ network.Network, conn network.Conn) {
	if conn == nil {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return
	}
	observation, ok := o.observed[conn.RemotePeer()]
	if !ok || observation.connectionID != conn.ID() {
		return
	}
	observation.connected = false
	observation.updatedAt = time.Now()
	o.observed[conn.RemotePeer()] = observation
}

func validQUICAddr(address ma.Multiaddr) bool {
	if address == nil {
		return false
	}
	_, hasUDP := address.ValueForProtocol(ma.P_UDP)
	_, hasQUIC := address.ValueForProtocol(ma.P_QUIC_V1)
	_, hasIP4 := address.ValueForProtocol(ma.P_IP4)
	_, hasIP6 := address.ValueForProtocol(ma.P_IP6)
	_, hasDNS4 := address.ValueForProtocol(ma.P_DNS4)
	_, hasDNS6 := address.ValueForProtocol(ma.P_DNS6)
	return hasUDP == nil && hasQUIC == nil && (hasIP4 == nil || hasIP6 == nil || hasDNS4 == nil || hasDNS6 == nil)
}

func validObserverPublicAddr(address ma.Multiaddr) bool {
	if !validQUICAddr(address) {
		return false
	}
	_, hasIP4 := address.ValueForProtocol(ma.P_IP4)
	_, hasDNS4 := address.ValueForProtocol(ma.P_DNS4)
	return hasIP4 == nil || hasDNS4 == nil
}

func containsPeerComponent(address ma.Multiaddr) bool {
	_, err := address.ValueForProtocol(ma.P_P2P)
	return err == nil
}
