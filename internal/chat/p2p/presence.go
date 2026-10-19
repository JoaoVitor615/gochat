package p2p

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

// PresenceProtocolID is the libp2p protocol ID for GoChat presence pings.
const PresenceProtocolID protocol.ID = "/gochat/presence/1.0.0"

// RegisterPresenceHandler listens for incoming peer presence ping signals.
func RegisterPresenceHandler(h host.Host, onPingReceived func(remotePeerID string)) {
	if h == nil {
		return
	}
	h.SetStreamHandler(PresenceProtocolID, func(stream network.Stream) {
		defer stream.Close()
		remotePeerID := stream.Conn().RemotePeer().String()
		_ = stream.SetDeadline(time.Now().Add(3 * time.Second))

		buf, err := io.ReadAll(io.LimitReader(stream, 64))
		if err == nil && strings.HasPrefix(string(buf), "ping") {
			_, _ = io.WriteString(stream, "pong\n")
			_ = stream.CloseWrite()
			if onPingReceived != nil {
				onPingReceived(remotePeerID)
			}
		}
	})
}

// PingPeer sends a presence ping to a remote peer and waits for pong confirmation.
func PingPeer(ctx context.Context, h host.Host, peerID peer.ID) error {
	if ctx == nil || h == nil {
		return errors.New("context and host are required")
	}
	dialCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	stream, err := h.NewStream(dialCtx, peerID, PresenceProtocolID)
	if err != nil {
		return err
	}
	defer stream.Close()

	deadline := time.Now().Add(3 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	_ = stream.SetDeadline(deadline)

	if _, err := io.WriteString(stream, "ping\n"); err != nil {
		return fmt.Errorf("write ping: %w", err)
	}
	if err := stream.CloseWrite(); err != nil {
		return fmt.Errorf("close write ping: %w", err)
	}

	response, err := io.ReadAll(io.LimitReader(stream, 64))
	if err != nil {
		return fmt.Errorf("read pong: %w", err)
	}
	if !strings.HasPrefix(string(response), "pong") {
		return errors.New("unexpected response to ping")
	}
	return nil
}
