package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	PeerTTL   = 5 * time.Minute
	InviteTTL = 10 * time.Minute
)

type Store struct {
	client *redis.Client
}

func NewStore(addr string) *Store {
	return &Store{
		client: redis.NewClient(&redis.Options{
			Addr: addr,
		}),
	}
}

func (s *Store) Close() error {
	return s.client.Close()
}

func (s *Store) SetPeer(
	ctx context.Context,
	peerID string,
	addresses []string,
) error {
	data, err := json.Marshal(addresses)
	if err != nil {
		return fmt.Errorf("marshal peer addresses: %w", err)
	}

	key := "peer:" + peerID

	if err := s.client.Set(ctx, key, data, PeerTTL).Err(); err != nil {
		return fmt.Errorf("store peer: %w", err)
	}

	return nil
}

func (s *Store) GetPeer(
	ctx context.Context,
	peerID string,
) ([]string, error) {
	key := "peer:" + peerID

	data, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, fmt.Errorf("get peer: %w", err)
	}

	var addresses []string

	if err := json.Unmarshal(data, &addresses); err != nil {
		return nil, fmt.Errorf("unmarshal peer addresses: %w", err)
	}

	return addresses, nil
}

func (s *Store) SetInvite(
	ctx context.Context,
	code string,
	peerID string,
) error {
	key := "invite:" + code

	if err := s.client.Set(ctx, key, peerID, InviteTTL).Err(); err != nil {
		return fmt.Errorf("store invite: %w", err)
	}

	return nil
}

func (s *Store) GetAndDeleteInvite(
	ctx context.Context,
	code string,
) (string, error) {
	key := "invite:" + code

	peerID, err := s.client.GetDel(ctx, key).Result()
	if err != nil {
		return "", fmt.Errorf("get invite: %w", err)
	}

	return peerID, nil
}
