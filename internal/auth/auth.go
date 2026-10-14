package auth

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

const (
	HeaderTimestamp = "X-GoChat-Timestamp"
	HeaderSignature = "X-GoChat-Signature"

	MaxTimestampAge = 2 * time.Minute
	MaxClockSkew   = 1 * time.Minute
)

var (
	ErrInvalidPeerID    = errors.New("invalid peer ID")
	ErrNoPublicKey      = errors.New("cannot extract public key from peer ID")
	ErrInvalidSignature = errors.New("invalid signature")
	ErrTimestampExpired = errors.New("timestamp expired")
	ErrTimestampFuture  = errors.New("timestamp is in the future")
	ErrMissingAuth      = errors.New("missing authentication parameters")
)

// NowFunc allows overriding current time in unit tests.
var NowFunc = time.Now

// VerifyTimestamp ensures the timestamp is within acceptable skew and age limits.
func VerifyTimestamp(timestamp int64) error {
	now := NowFunc().UTC()
	reqTime := time.Unix(timestamp, 0).UTC()

	if now.Sub(reqTime) > MaxTimestampAge {
		return ErrTimestampExpired
	}
	if reqTime.Sub(now) > MaxClockSkew {
		return ErrTimestampFuture
	}
	return nil
}

// HeartbeatPayload creates the canonical byte sequence for a heartbeat request.
func HeartbeatPayload(peerID string, timestamp int64, addresses []string) []byte {
	sortedAddrs := make([]string, len(addresses))
	copy(sortedAddrs, addresses)
	sort.Strings(sortedAddrs)
	return []byte(fmt.Sprintf("gochat:heartbeat:%s:%d:%s", peerID, timestamp, strings.Join(sortedAddrs, ",")))
}

// InvitePayload creates the canonical byte sequence for an invite creation request.
func InvitePayload(peerID string, timestamp int64) []byte {
	return []byte(fmt.Sprintf("gochat:invite:%s:%d", peerID, timestamp))
}

// ResolvePayload creates the canonical byte sequence for an invite resolution request.
func ResolvePayload(peerID string, code string, timestamp int64) []byte {
	return []byte(fmt.Sprintf("gochat:resolve:%s:%s:%d", peerID, code, timestamp))
}

// AcceptancesPayload creates the canonical byte sequence for polling acceptances.
func AcceptancesPayload(peerID string, timestamp int64) []byte {
	return []byte(fmt.Sprintf("gochat:acceptances:%s:%d", peerID, timestamp))
}

// AcknowledgeAcceptancesPayload creates the canonical byte sequence for acknowledging acceptances.
func AcknowledgeAcceptancesPayload(peerID string, timestamp int64, inviteIDs []string) []byte {
	sortedIDs := make([]string, len(inviteIDs))
	copy(sortedIDs, inviteIDs)
	sort.Strings(sortedIDs)
	return []byte(fmt.Sprintf("gochat:acceptances-ack:%s:%d:%s", peerID, timestamp, strings.Join(sortedIDs, ",")))
}

// Sign signs the payload with the Ed25519 private key and returns Base64.
func Sign(privateKey ed25519.PrivateKey, payload []byte) string {
	sig := ed25519.Sign(privateKey, payload)
	return base64.RawStdEncoding.EncodeToString(sig)
}

// Verify verifies that the given payload was signed by the private key corresponding to peerIDString.
func Verify(peerIDString string, payload []byte, signatureBase64 string, timestamp int64) error {
	if err := VerifyTimestamp(timestamp); err != nil {
		return err
	}

	peerID, err := peer.Decode(peerIDString)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidPeerID, err)
	}

	pubKey, err := peerID.ExtractPublicKey()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrNoPublicKey, err)
	}

	sigBytes, err := base64.RawStdEncoding.DecodeString(signatureBase64)
	if err != nil {
		sigBytes, err = base64.StdEncoding.DecodeString(signatureBase64)
		if err != nil {
			return fmt.Errorf("%w: cannot decode base64: %v", ErrInvalidSignature, err)
		}
	}

	valid, err := pubKey.Verify(payload, sigBytes)
	if err != nil || !valid {
		return ErrInvalidSignature
	}

	return nil
}
