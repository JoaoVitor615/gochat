package p2p

import (
	"errors"

	"github.com/libp2p/go-libp2p/core/host"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multiaddr/net"
)

var ErrHostRequired = errors.New(PackageName + " host is required")

// AddressSnapshot contains addresses known by libp2p for this host. Public
// candidates are globally routable QUIC/UDP addresses, while reachable public
// addresses have additionally been confirmed by AutoNATv2 when available.
type AddressSnapshot struct {
	Candidates       []ma.Multiaddr
	PublicCandidates []ma.Multiaddr
	ReachablePublic  []ma.Multiaddr
}

type reachabilityReporter interface {
	ConfirmedAddrs() (reachable []ma.Multiaddr, unreachable []ma.Multiaddr, unknown []ma.Multiaddr)
}

// DiscoverHostAddresses reads the host's current libp2p address view. Public
// addresses can include observations learned from connected libp2p peers; they
// are not inferred by combining an external IP with a local port.
func DiscoverHostAddresses(h host.Host) (AddressSnapshot, error) {
	if h == nil {
		return AddressSnapshot{}, ErrHostRequired
	}

	snapshot := AddressSnapshot{Candidates: uniqueAddresses(h.Addrs())}
	snapshot.PublicCandidates = publicQUICAddresses(snapshot.Candidates)

	if reporter, ok := h.(reachabilityReporter); ok {
		reachable, _, _ := reporter.ConfirmedAddrs()
		snapshot.ReachablePublic = publicQUICAddresses(reachable)
	}

	return snapshot, nil
}

func publicQUICAddresses(addresses []ma.Multiaddr) []ma.Multiaddr {
	var public []ma.Multiaddr
	for _, address := range addresses {
		if _, err := address.ValueForProtocol(ma.P_UDP); err != nil {
			continue
		}
		if _, err := address.ValueForProtocol(ma.P_QUIC_V1); err != nil {
			continue
		}
		if !manet.IsPublicAddr(address) {
			continue
		}
		public = append(public, address)
	}
	return uniqueAddresses(public)
}

func uniqueAddresses(addresses []ma.Multiaddr) []ma.Multiaddr {
	unique := make([]ma.Multiaddr, 0, len(addresses))
	seen := make(map[string]struct{}, len(addresses))
	for _, address := range addresses {
		if address == nil {
			continue
		}
		key := address.String()
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, address)
	}
	return unique
}
