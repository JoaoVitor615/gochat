package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/JoaoVitor615/gochat/internal/discovery/repository"
)

var (
	ErrInvalidInvite = errors.New("invalid or expired invite")
	ErrPeerOffline   = errors.New("peer is offline")
)

type Service struct {
	repository repository.Repository
}

type ResolvedPeer struct {
	PeerID    string
	Addresses []string
}

func New(repository repository.Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Heartbeat(ctx context.Context, peerID string, addresses []string) error {
	if err := s.repository.SetPeer(ctx, peerID, addresses); err != nil {
		return fmt.Errorf("heartbeat peer: %w", err)
	}

	return nil
}

func (s *Service) CreateInvite(ctx context.Context, peerID string) (string, error) {
	code, err := generateInviteCode()
	if err != nil {
		return "", err
	}

	if err := s.repository.SetInvite(ctx, code, peerID); err != nil {
		return "", fmt.Errorf("store invite: %w", err)
	}

	return code, nil
}

func (s *Service) ResolveInvite(ctx context.Context, code string) (ResolvedPeer, error) {
	peerID, err := s.repository.GetAndDeleteInvite(ctx, code)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ResolvedPeer{}, ErrInvalidInvite
		}
		return ResolvedPeer{}, fmt.Errorf("get invite: %w", err)
	}

	addresses, err := s.repository.GetPeer(ctx, peerID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ResolvedPeer{}, ErrPeerOffline
		}
		return ResolvedPeer{}, fmt.Errorf("get peer: %w", err)
	}

	return ResolvedPeer{PeerID: peerID, Addresses: addresses}, nil
}
