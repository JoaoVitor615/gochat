package p2p

import (
	"testing"

	ma "github.com/multiformats/go-multiaddr"
)

func TestPublicQUICAddresses(t *testing.T) {
	tests := []struct {
		name      string
		addresses []string
		want      []string
	}{
		{
			name: "keeps globally routable quic addresses only",
			addresses: []string{
				"/ip4/8.8.8.8/udp/4001/quic-v1",
				"/ip4/192.168.1.10/udp/4001/quic-v1",
				"/ip4/8.8.4.4/tcp/4001",
				"/ip4/1.1.1.1/udp/4001/quic-v1",
			},
			want: []string{
				"/ip4/8.8.8.8/udp/4001/quic-v1",
				"/ip4/1.1.1.1/udp/4001/quic-v1",
			},
		},
		{
			name: "removes duplicates",
			addresses: []string{
				"/ip4/8.8.8.8/udp/4001/quic-v1",
				"/ip4/8.8.8.8/udp/4001/quic-v1",
			},
			want: []string{"/ip4/8.8.8.8/udp/4001/quic-v1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := make([]ma.Multiaddr, 0, len(tt.addresses))
			for _, raw := range tt.addresses {
				address, err := ma.NewMultiaddr(raw)
				if err != nil {
					t.Fatalf("parse input address %q: %v", raw, err)
				}
				input = append(input, address)
			}

			got := publicQUICAddresses(input)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d addresses, want %d: %v", len(got), len(tt.want), got)
			}
			for i, address := range got {
				if address.String() != tt.want[i] {
					t.Errorf("address[%d] = %q, want %q", i, address, tt.want[i])
				}
			}
		})
	}
}
