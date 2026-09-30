package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	peerTTL   = 5 * time.Minute
	inviteTTL = 10 * time.Minute
)

// Redis persists discovery peers and one-time invites in Redis.
type Redis struct {
	client *redis.Client
}

func NewRedis(addr string) *Redis {
	return &Redis{client: redis.NewClient(&redis.Options{Addr: addr})}
}

func (r *Redis) Close() error {
	return r.client.Close()
}

func (r *Redis) SetPeer(ctx context.Context, peerID string, addresses []string) error {
	data, err := json.Marshal(addresses)
	if err != nil {
		return fmt.Errorf("marshal peer addresses: %w", err)
	}

	if err := r.client.Set(ctx, "peer:"+peerID, data, peerTTL).Err(); err != nil {
		return fmt.Errorf("store peer: %w", err)
	}

	return nil
}

func (r *Redis) GetPeer(ctx context.Context, peerID string) ([]string, error) {
	data, err := r.client.Get(ctx, "peer:"+peerID).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get peer: %w", err)
	}

	var addresses []string
	if err := json.Unmarshal(data, &addresses); err != nil {
		return nil, fmt.Errorf("unmarshal peer addresses: %w", err)
	}

	return addresses, nil
}

func (r *Redis) SetInvite(ctx context.Context, code, peerID string) error {
	if err := r.client.Set(ctx, "invite:"+code, peerID, inviteTTL).Err(); err != nil {
		return fmt.Errorf("store invite: %w", err)
	}

	return nil
}

func (r *Redis) GetAndDeleteInvite(ctx context.Context, code string) (string, error) {
	peerID, err := r.client.GetDel(ctx, "invite:"+code).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("get invite: %w", err)
	}

	return peerID, nil
}

var _ Repository = (*Redis)(nil)
