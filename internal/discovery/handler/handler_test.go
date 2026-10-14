package handler

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/JoaoVitor615/gochat/internal/auth"
	"github.com/JoaoVitor615/gochat/internal/discovery/repository"
	"github.com/JoaoVitor615/gochat/internal/discovery/service"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

type mockRepo struct {
	peers       map[string][]string
	invites     map[string]string
	acceptances map[string][]repository.Acceptance
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		peers:       make(map[string][]string),
		invites:     make(map[string]string),
		acceptances: make(map[string][]repository.Acceptance),
	}
}

func (m *mockRepo) SetPeer(_ context.Context, peerID string, addresses []string) error {
	m.peers[peerID] = addresses
	return nil
}

func (m *mockRepo) GetPeer(_ context.Context, peerID string) ([]string, error) {
	addrs, ok := m.peers[peerID]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return addrs, nil
}

func (m *mockRepo) SetInvite(_ context.Context, code, peerID string) error {
	m.invites[code] = peerID
	return nil
}

func (m *mockRepo) AcceptInvite(_ context.Context, code, acceptingPeerID string, acceptedAt time.Time) (string, []string, error) {
	owner, ok := m.invites[code]
	if !ok {
		return "", nil, repository.ErrNotFound
	}
	if owner == acceptingPeerID {
		return "", nil, repository.ErrSelfInvite
	}
	ownerAddrs, ok := m.peers[owner]
	if !ok {
		return "", nil, repository.ErrPeerOffline
	}
	return owner, ownerAddrs, nil
}

func (m *mockRepo) ListAcceptances(_ context.Context, peerID string) ([]repository.Acceptance, error) {
	return m.acceptances[peerID], nil
}

func (m *mockRepo) AcknowledgeAcceptances(_ context.Context, peerID string, inviteIDs []string) error {
	return nil
}

func generatePeer(t *testing.T) (ed25519.PrivateKey, string) {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate ed25519 key: %v", err)
	}
	libp2pPriv, _, err := libp2pcrypto.KeyPairFromStdKey(&priv)
	if err != nil {
		t.Fatalf("convert to libp2p key: %v", err)
	}
	peerID, err := peer.IDFromPrivateKey(libp2pPriv)
	if err != nil {
		t.Fatalf("derive peer ID: %v", err)
	}
	return priv, peerID.String()
}

func setupTestHandler() (*Handler, *mockRepo) {
	repo := newMockRepo()
	svc := service.New(repo)
	h := New(svc)
	return h, repo
}

func TestHeartbeat_Success(t *testing.T) {
	h, _ := setupTestHandler()
	priv, peerID := generatePeer(t)
	now := time.Now().Unix()
	addresses := []string{"/ip4/127.0.0.1/udp/4001/quic-v1"}

	payload := auth.HeartbeatPayload(peerID, now, addresses)
	sig := auth.Sign(priv, payload)

	body, _ := json.Marshal(peerRequest{
		PeerID:    peerID,
		Addresses: addresses,
		Timestamp: now,
		Signature: sig,
	})

	req := httptest.NewRequest(http.MethodPost, "/heartbeat", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Heartbeat(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHeartbeat_SpoofedPeerID(t *testing.T) {
	h, _ := setupTestHandler()
	attackerPriv, _ := generatePeer(t)
	_, victimPeerID := generatePeer(t)
	now := time.Now().Unix()
	addresses := []string{"/ip4/1.2.3.4/udp/4001/quic-v1"}

	// Attacker attempts to forge a heartbeat for victimPeerID signed with attacker's key
	payload := auth.HeartbeatPayload(victimPeerID, now, addresses)
	sig := auth.Sign(attackerPriv, payload)

	body, _ := json.Marshal(peerRequest{
		PeerID:    victimPeerID,
		Addresses: addresses,
		Timestamp: now,
		Signature: sig,
	})

	req := httptest.NewRequest(http.MethodPost, "/heartbeat", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Heartbeat(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for spoofed heartbeat, got %d", w.Code)
	}
}

func TestHeartbeat_ExpiredTimestamp(t *testing.T) {
	h, _ := setupTestHandler()
	priv, peerID := generatePeer(t)
	expired := time.Now().Add(-5 * time.Minute).Unix()
	addresses := []string{"/ip4/127.0.0.1/udp/4001/quic-v1"}

	payload := auth.HeartbeatPayload(peerID, expired, addresses)
	sig := auth.Sign(priv, payload)

	body, _ := json.Marshal(peerRequest{
		PeerID:    peerID,
		Addresses: addresses,
		Timestamp: expired,
		Signature: sig,
	})

	req := httptest.NewRequest(http.MethodPost, "/heartbeat", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Heartbeat(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for expired timestamp, got %d", w.Code)
	}
}

func TestInvite_Success(t *testing.T) {
	h, _ := setupTestHandler()
	priv, peerID := generatePeer(t)
	now := time.Now().Unix()

	payload := auth.InvitePayload(peerID, now)
	sig := auth.Sign(priv, payload)

	body, _ := json.Marshal(inviteRequest{
		PeerID:    peerID,
		Timestamp: now,
		Signature: sig,
	})

	req := httptest.NewRequest(http.MethodPost, "/invite", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Invite(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var res inviteResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil || res.Code == "" {
		t.Fatalf("invalid invite response: %v, body: %s", err, w.Body.String())
	}
}

func TestInvite_SpoofedPeerID(t *testing.T) {
	h, _ := setupTestHandler()
	attackerPriv, _ := generatePeer(t)
	_, victimPeerID := generatePeer(t)
	now := time.Now().Unix()

	payload := auth.InvitePayload(victimPeerID, now)
	sig := auth.Sign(attackerPriv, payload)

	body, _ := json.Marshal(inviteRequest{
		PeerID:    victimPeerID,
		Timestamp: now,
		Signature: sig,
	})

	req := httptest.NewRequest(http.MethodPost, "/invite", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Invite(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for spoofed invite, got %d", w.Code)
	}
}

func TestResolve_Success(t *testing.T) {
	h, repo := setupTestHandler()
	_, ownerPeerID := generatePeer(t)
	repo.peers[ownerPeerID] = []string{"/ip4/127.0.0.1/udp/4001/quic-v1"}

	code, err := h.service.CreateInvite(context.Background(), ownerPeerID)
	if err != nil {
		t.Fatalf("create invite: %v", err)
	}

	priv, acceptorPeerID := generatePeer(t)
	now := time.Now().Unix()
	repo.peers[acceptorPeerID] = []string{"/ip4/127.0.0.1/udp/4002/quic-v1"}

	payload := auth.ResolvePayload(acceptorPeerID, code, now)
	sig := auth.Sign(priv, payload)

	body, _ := json.Marshal(resolveRequest{
		Code:      code,
		PeerID:    acceptorPeerID,
		Timestamp: now,
		Signature: sig,
	})

	req := httptest.NewRequest(http.MethodPost, "/resolve", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Resolve(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}
}

func TestResolve_SpoofedPeerID(t *testing.T) {
	h, _ := setupTestHandler()
	attackerPriv, _ := generatePeer(t)
	_, victimPeerID := generatePeer(t)
	now := time.Now().Unix()

	payload := auth.ResolvePayload(victimPeerID, "TEST-CODE-1", now)
	sig := auth.Sign(attackerPriv, payload)

	body, _ := json.Marshal(resolveRequest{
		Code:      "TEST-CODE-1",
		PeerID:    victimPeerID,
		Timestamp: now,
		Signature: sig,
	})

	req := httptest.NewRequest(http.MethodPost, "/resolve", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Resolve(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for spoofed resolve, got %d", w.Code)
	}
}

func TestAcceptances_Success(t *testing.T) {
	h, repo := setupTestHandler()
	priv, peerID := generatePeer(t)
	now := time.Now().Unix()

	repo.acceptances[peerID] = []repository.Acceptance{
		{
			InviteID:   "inv-1",
			PeerID:     "12D3KooWOtherPeer",
			Addresses:  []string{"/ip4/127.0.0.1/udp/4002/quic-v1"},
			AcceptedAt: time.Now(),
		},
	}

	payload := auth.AcceptancesPayload(peerID, now)
	sig := auth.Sign(priv, payload)

	req := httptest.NewRequest(http.MethodGet, "/acceptances?peer_id="+peerID, nil)
	req.Header.Set(auth.HeaderTimestamp, strconv.FormatInt(now, 10))
	req.Header.Set(auth.HeaderSignature, sig)
	w := httptest.NewRecorder()

	h.Acceptances(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAcceptances_MissingHeaders(t *testing.T) {
	h, _ := setupTestHandler()
	_, peerID := generatePeer(t)

	req := httptest.NewRequest(http.MethodGet, "/acceptances?peer_id="+peerID, nil)
	w := httptest.NewRecorder()

	h.Acceptances(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for missing headers, got %d", w.Code)
	}
}

func TestAcceptances_SpoofedSignature(t *testing.T) {
	h, _ := setupTestHandler()
	attackerPriv, _ := generatePeer(t)
	_, victimPeerID := generatePeer(t)
	now := time.Now().Unix()

	payload := auth.AcceptancesPayload(victimPeerID, now)
	sig := auth.Sign(attackerPriv, payload)

	req := httptest.NewRequest(http.MethodGet, "/acceptances?peer_id="+victimPeerID, nil)
	req.Header.Set(auth.HeaderTimestamp, strconv.FormatInt(now, 10))
	req.Header.Set(auth.HeaderSignature, sig)
	w := httptest.NewRecorder()

	h.Acceptances(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for spoofed acceptances query, got %d", w.Code)
	}
}
