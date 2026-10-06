package repository

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound    = errors.New("record not found")
	ErrPeerOffline = errors.New("peer is offline")
	ErrSelfInvite  = errors.New("cannot accept own invite")
)

// Repository defines the persistence operations required by discovery.
type Repository interface {
	SetPeer(ctx context.Context, peerID string, addresses []string) error
	GetPeer(ctx context.Context, peerID string) ([]string, error)
	SetInvite(ctx context.Context, code, peerID string) error
	AcceptInvite(ctx context.Context, code, acceptingPeerID string, acceptedAt time.Time) (string, []string, error)
	ListAcceptances(ctx context.Context, peerID string) ([]Acceptance, error)
	AcknowledgeAcceptances(ctx context.Context, peerID string, inviteIDs []string) error
}

type Acceptance struct {
	InviteID   string    `json:"invite_id"`
	PeerID     string    `json:"peer_id"`
	Addresses  []string  `json:"addresses"`
	AcceptedAt time.Time `json:"accepted_at"`
}
