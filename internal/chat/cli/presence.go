package cli

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/JoaoVitor615/gochat/internal/chat/p2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
)

const heartbeatInterval = 2 * time.Minute
const observerProtocol protocol.ID = "/gochat/address-observation/1.0.0"
const observerRequest = "observe\n"

func (a *App) startPresence(ctx context.Context) (func(), error) {
	if a.Host != nil {
		return nil, fmt.Errorf("peer host is already running")
	}

	h, err := p2p.NewHostOnFirstAvailablePort(a.Identity)
	if err != nil {
		return nil, fmt.Errorf("start peer host: %w", err)
	}
	a.Host = h
	p2p.SetStreamHandler(h, func(stream network.Stream) {
		if err := p2p.HandleStream(ctx, stream, a.Store, a.Identity.PeerID); err != nil && ctx.Err() == nil {
			log.Printf("handle peer message: %v", err)
		}
	})

	observedAddress, err := connectToAddressObserver(ctx, h, a.ObserverAddress)
	if err != nil && a.ObserverAddress != "" {
		log.Printf("public UDP address observation unavailable; continuing with local libp2p addresses: %v", err)
	}
	addresses, err := peerAddressStrings(h, observedAddress)
	if err != nil {
		_ = h.Close()
		a.Host = nil
		return nil, fmt.Errorf("get peer addresses: %w", err)
	}

	if err := a.DiscoveryClient.Heartbeat(ctx, a.Identity.PeerID, addresses); err != nil {
		_ = h.Close()
		a.Host = nil
		return nil, err
	}

	heartbeatCtx, cancelHeartbeat := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(heartbeatInterval)
		defer ticker.Stop()

		for {
			select {
			case <-heartbeatCtx.Done():
				return
			case <-ticker.C:
				currentObservedAddress, err := observePublicAddress(heartbeatCtx, h, a.ObserverAddress)
				if err != nil && a.ObserverAddress != "" {
					log.Printf("refresh public UDP address observation: %v", err)
				}
				currentAddresses, addressErr := peerAddressStrings(h, currentObservedAddress)
				if addressErr == nil {
					err = a.DiscoveryClient.Heartbeat(heartbeatCtx, a.Identity.PeerID, currentAddresses)
				} else {
					err = addressErr
				}
				if err != nil && heartbeatCtx.Err() == nil {
					log.Printf("refresh peer heartbeat: %v", err)
				}
			}
		}
	}()

	return func() {
		cancelHeartbeat()
		<-done
		if err := h.Close(); err != nil {
			log.Printf("close peer host: %v", err)
		}
		a.Host = nil
	}, nil
}

func peerAddressStrings(h host.Host, observed []ma.Multiaddr) ([]string, error) {
	snapshot, err := p2p.DiscoverHostAddresses(h)
	if err != nil {
		return nil, err
	}
	addressesToPublish := publishableAddresses(observed, snapshot.ReachablePublic, snapshot.PublicCandidates, snapshot.Candidates)
	addresses, err := peer.AddrInfoToP2pAddrs(&peer.AddrInfo{
		ID:    h.ID(),
		Addrs: addressesToPublish,
	})
	if err != nil {
		return nil, fmt.Errorf("get peer addresses: %w", err)
	}
	if len(snapshot.PublicCandidates) == 0 && len(observed) == 0 {
		log.Printf("no public QUIC address observed yet; libp2p candidates will be refreshed on the next heartbeat")
	}
	if len(snapshot.ReachablePublic) > 0 {
		log.Printf("libp2p confirmed %d publicly reachable QUIC address(es)", len(snapshot.ReachablePublic))
	}

	addressStrings := make([]string, len(addresses))
	for i, address := range addresses {
		addressStrings[i] = address.String()
	}
	return addressStrings, nil
}

func publishableAddresses(groups ...[]ma.Multiaddr) []ma.Multiaddr {
	const maxAddresses = 16
	addresses := make([]ma.Multiaddr, 0, maxAddresses)
	seen := make(map[string]struct{}, maxAddresses)
	for _, group := range groups {
		for _, address := range group {
			if address == nil || manet.IsIPLoopback(address) || manet.IsIPUnspecified(address) {
				continue
			}
			key := address.String()
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			addresses = append(addresses, address)
			if len(addresses) == maxAddresses {
				return addresses
			}
		}
	}
	return addresses
}

func connectToAddressObserver(ctx context.Context, h host.Host, observerAddress string) ([]ma.Multiaddr, error) {
	observerAddr, err := ma.NewMultiaddr(observerAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid observer address: %w", err)
	}
	observer, err := peer.AddrInfoFromP2pAddr(observerAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid observer peer info: %w", err)
	}
	return observeAtPeer(ctx, h, observer)
}

func observePublicAddress(ctx context.Context, h host.Host, observerAddress string) ([]ma.Multiaddr, error) {
	if observerAddress == "" {
		return nil, nil
	}
	observerAddr, err := ma.NewMultiaddr(observerAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid observer address: %w", err)
	}
	observer, err := peer.AddrInfoFromP2pAddr(observerAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid observer peer info: %w", err)
	}
	return observeAtPeer(ctx, h, observer)
}

func observeAtPeer(ctx context.Context, h host.Host, observer *peer.AddrInfo) ([]ma.Multiaddr, error) {
	dialCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	if err := h.Connect(dialCtx, *observer); err != nil {
		return nil, fmt.Errorf("connect to libp2p address observer: %w", err)
	}

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		address, err := requestObservedAddress(dialCtx, h, observer.ID)
		if err != nil {
			return nil, err
		}
		if address != nil {
			return []ma.Multiaddr{address}, nil
		}

		select {
		case <-dialCtx.Done():
			return nil, fmt.Errorf("timed out waiting for observer to see this peer: %w", dialCtx.Err())
		case <-ticker.C:
		}
	}
}

func requestObservedAddress(ctx context.Context, h host.Host, observerID peer.ID) (ma.Multiaddr, error) {
	stream, err := h.NewStream(ctx, observerID, observerProtocol)
	if err != nil {
		return nil, fmt.Errorf("request public UDP address from observer: %w", err)
	}
	defer stream.Close()
	_ = stream.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err := io.WriteString(stream, observerRequest); err != nil {
		return nil, fmt.Errorf("send observer request: %w", err)
	}
	if err := stream.CloseWrite(); err != nil {
		return nil, fmt.Errorf("finish observer request: %w", err)
	}
	response, err := io.ReadAll(io.LimitReader(stream, 512))
	if err != nil {
		return nil, fmt.Errorf("read observer response: %w", err)
	}
	if len(response) == 0 || string(response) == "\n" {
		return nil, nil
	}
	address, err := ma.NewMultiaddr(string(response[:len(response)-1]))
	if err != nil {
		return nil, fmt.Errorf("invalid observed UDP address: %w", err)
	}
	return address, nil
}
