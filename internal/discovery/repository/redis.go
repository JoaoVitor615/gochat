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

var acceptInviteScript = redis.NewScript(`
local owner = redis.call('GET', KEYS[1])
if not owner then
	return cjson.encode({status = 'invite_missing'})
end
if owner == ARGV[2] then
	return cjson.encode({status = 'self_invite'})
end
local ownerAddresses = redis.call('GET', 'peer:' .. owner)
if not ownerAddresses then
	return cjson.encode({status = 'owner_offline'})
end
local acceptingAddresses = redis.call('GET', KEYS[2])
if not acceptingAddresses then
	return cjson.encode({status = 'acceptor_offline'})
end
local acceptanceKey = 'acceptances:' .. owner
local acceptance = {
	invite_id = ARGV[1],
	peer_id = ARGV[2],
	addresses = cjson.decode(acceptingAddresses),
	accepted_at = ARGV[3]
}
redis.call('HSET', acceptanceKey, ARGV[1], cjson.encode(acceptance))
redis.call('EXPIRE', acceptanceKey, tonumber(ARGV[4]))
redis.call('DEL', KEYS[1])
return cjson.encode({status = 'ok', peer_id = owner, addresses = cjson.decode(ownerAddresses)})
`)

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

func (r *Redis) AcceptInvite(ctx context.Context, code, acceptingPeerID string, acceptedAt time.Time) (string, []string, error) {
	result, err := acceptInviteScript.Run(ctx, r.client, []string{
		"invite:" + code,
		"peer:" + acceptingPeerID,
	}, code, acceptingPeerID, acceptedAt.UTC().Format(time.RFC3339Nano), int(inviteTTL.Seconds())).Text()
	if err != nil {
		return "", nil, fmt.Errorf("accept invite: %w", err)
	}

	var response struct {
		Status    string   `json:"status"`
		PeerID    string   `json:"peer_id"`
		Addresses []string `json:"addresses"`
	}
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		return "", nil, fmt.Errorf("decode invite acceptance result: %w", err)
	}
	switch response.Status {
	case "ok":
		return response.PeerID, response.Addresses, nil
	case "invite_missing":
		return "", nil, ErrNotFound
	case "owner_offline", "acceptor_offline":
		return "", nil, ErrPeerOffline
	case "self_invite":
		return "", nil, ErrSelfInvite
	default:
		return "", nil, fmt.Errorf("accept invite: unexpected Redis result %q", response.Status)
	}
}

func (r *Redis) ListAcceptances(ctx context.Context, peerID string) ([]Acceptance, error) {
	values, err := r.client.HVals(ctx, "acceptances:"+peerID).Result()
	if err != nil {
		return nil, fmt.Errorf("list invite acceptances: %w", err)
	}
	acceptances := make([]Acceptance, 0, len(values))
	for _, value := range values {
		var acceptance Acceptance
		if err := json.Unmarshal([]byte(value), &acceptance); err != nil {
			return nil, fmt.Errorf("decode invite acceptance: %w", err)
		}
		acceptances = append(acceptances, acceptance)
	}
	return acceptances, nil
}

func (r *Redis) AcknowledgeAcceptances(ctx context.Context, peerID string, inviteIDs []string) error {
	if len(inviteIDs) == 0 {
		return nil
	}
	if err := r.client.HDel(ctx, "acceptances:"+peerID, inviteIDs...).Err(); err != nil {
		return fmt.Errorf("acknowledge invite acceptances: %w", err)
	}
	return nil
}

var _ Repository = (*Redis)(nil)
