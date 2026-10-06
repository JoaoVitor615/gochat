package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/JoaoVitor615/gochat/internal/discovery/repository"
)

var (
	ErrInvalidInvite = errors.New("invalid or expired invite")
	ErrPeerOffline   = errors.New("peer is offline")
	ErrSelfInvite    = errors.New("cannot accept own invite")
)

type Service struct {
	repository repository.Repository
}

type ResolvedPeer struct {
	PeerID    string
	Addresses []string
}

type Acceptance = repository.Acceptance

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

func (s *Service) ResolveInvite(ctx context.Context, code, acceptingPeerID string) (ResolvedPeer, error) {
	code, ok := normalizeInviteCode(code)
	if !ok {
		return ResolvedPeer{}, ErrInvalidInvite
	}

	peerID, addresses, err := s.repository.AcceptInvite(ctx, code, acceptingPeerID, time.Now().UTC())
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			return ResolvedPeer{}, ErrInvalidInvite
		case errors.Is(err, repository.ErrPeerOffline):
			return ResolvedPeer{}, ErrPeerOffline
		case errors.Is(err, repository.ErrSelfInvite):
			return ResolvedPeer{}, ErrSelfInvite
		default:
			return ResolvedPeer{}, fmt.Errorf("accept invite: %w", err)
		}
	}
	return ResolvedPeer{PeerID: peerID, Addresses: addresses}, nil
}

func (s *Service) ListAcceptances(ctx context.Context, peerID string) ([]Acceptance, error) {
	return s.repository.ListAcceptances(ctx, peerID)
}

func (s *Service) AcknowledgeAcceptances(ctx context.Context, peerID string, inviteIDs []string) error {
	if len(inviteIDs) > 100 {
		return errors.New("at most 100 invite IDs can be acknowledged at once")
	}
	return s.repository.AcknowledgeAcceptances(ctx, peerID, inviteIDs)
}

func normalizeInviteCode(code string) (string, bool) {
	var token strings.Builder
	token.Grow(9)
	for _, char := range strings.TrimSpace(code) {
		if char == '-' || unicode.IsSpace(char) {
			continue
		}
		char = unicode.ToUpper(char)
		if (char < 'A' || char > 'Z') && (char < '0' || char > '9') {
			return "", false
		}
		token.WriteRune(char)
	}
	if token.Len() != 9 {
		return "", false
	}

	value := token.String()
	return fmt.Sprintf("%s-%s-%s", value[:3], value[3:6], value[6:]), true
}
