package cli

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/JoaoVitor615/gochat/internal/chat/p2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

const heartbeatInterval = 2 * time.Minute

func (a *App) startPresence(ctx context.Context) (func(), error) {
	if a.Host != nil {
		return nil, fmt.Errorf("peer host is already running")
	}

	h, err := p2p.NewHostOnFirstAvailablePort(a.Identity)
	if err != nil {
		return nil, fmt.Errorf("start peer host: %w", err)
	}
	a.Host = h

	addresses, err := peerAddressStrings(h)
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
				currentAddresses, err := peerAddressStrings(h)
				if err == nil {
					err = a.DiscoveryClient.Heartbeat(heartbeatCtx, a.Identity.PeerID, currentAddresses)
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

func peerAddressStrings(h host.Host) ([]string, error) {
	snapshot, err := p2p.DiscoverHostAddresses(h)
	if err != nil {
		return nil, err
	}
	addresses, err := peer.AddrInfoToP2pAddrs(&peer.AddrInfo{
		ID:    h.ID(),
		Addrs: snapshot.Candidates,
	})
	if err != nil {
		return nil, fmt.Errorf("get peer addresses: %w", err)
	}
	if len(snapshot.PublicCandidates) == 0 {
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
