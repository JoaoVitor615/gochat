package repository

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("record not found")

// Repository defines the persistence operations required by discovery.
type Repository interface {
	SetPeer(ctx context.Context, peerID string, addresses []string) error
	GetPeer(ctx context.Context, peerID string) ([]string, error)
	SetInvite(ctx context.Context, code, peerID string) error
	GetAndDeleteInvite(ctx context.Context, code string) (string, error)
}
