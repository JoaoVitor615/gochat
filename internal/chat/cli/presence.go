package cli

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/JoaoVitor615/gochat/internal/chat/p2p"
	"github.com/libp2p/go-libp2p/core/peer"
)

const heartbeatInterval = 2 * time.Minute

func (a *App) startPresence(ctx context.Context) (func(), error) {
	if a.Host != nil {
		return nil, fmt.Errorf("peer host is already running")
	}

	h, err := p2p.NewHost(a.Identity, defaultPort)
	if err != nil {
		return nil, fmt.Errorf("start peer host: %w", err)
	}
	a.Host = h

	addresses, err := peer.AddrInfoToP2pAddrs(&peer.AddrInfo{
		ID:    h.ID(),
		Addrs: h.Addrs(),
	})
	if err != nil {
		_ = h.Close()
		a.Host = nil
		return nil, fmt.Errorf("get peer addresses: %w", err)
	}

	addressStrings := make([]string, len(addresses))
	for i, address := range addresses {
		addressStrings[i] = address.String()
	}

	if err := a.DiscoveryClient.Heartbeat(ctx, a.Identity.PeerID, addressStrings); err != nil {
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
				if err := a.DiscoveryClient.Heartbeat(heartbeatCtx, a.Identity.PeerID, addressStrings); err != nil && heartbeatCtx.Err() == nil {
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
