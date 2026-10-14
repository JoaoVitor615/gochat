package client

import (
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
	"github.com/JoaoVitor615/gochat/internal/chat/identity"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

func generateTestIdentity(t *testing.T) *identity.Identity {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
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
	return &identity.Identity{
		PrivateKey: priv,
		PublicKey:  pub,
		PeerID:     peerID.String(),
	}
}

func TestClient_Heartbeat_SignsRequest(t *testing.T) {
	id := generateTestIdentity(t)
	var receivedReq HeartbeatReq

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != HEARTBEAT_ENDPOINT {
			http.NotFound(w, r)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&receivedReq); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// Verify signature received by the server
		payload := auth.HeartbeatPayload(receivedReq.PeerID, receivedReq.Timestamp, receivedReq.Addresses)
		if err := auth.Verify(receivedReq.PeerID, payload, receivedReq.Signature, receivedReq.Timestamp); err != nil {
			http.Error(w, "invalid signature: "+err.Error(), http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	c, err := NewClient(server.URL, id)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	addresses := []string{"/ip4/127.0.0.1/udp/4001/quic-v1"}
	if err := c.Heartbeat(context.Background(), id.PeerID, addresses); err != nil {
		t.Fatalf("heartbeat failed: %v", err)
	}

	if receivedReq.PeerID != id.PeerID {
		t.Errorf("expected peer ID %s, got %s", id.PeerID, receivedReq.PeerID)
	}
	if receivedReq.Signature == "" {
		t.Errorf("expected signature to be non-empty")
	}
}

func TestClient_CreateInvite_SignsRequest(t *testing.T) {
	id := generateTestIdentity(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != CREATE_INVITE_ENDPOINT {
			http.NotFound(w, r)
			return
		}
		var req CreateInviteReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		payload := auth.InvitePayload(req.PeerID, req.Timestamp)
		if err := auth.Verify(req.PeerID, payload, req.Signature, req.Timestamp); err != nil {
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(CreateInviteResponse{Code: "ABC-DEF-GHI"})
	}))
	defer server.Close()

	c, err := NewClient(server.URL, id)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	res, err := c.CreateInvite(context.Background(), id.PeerID)
	if err != nil {
		t.Fatalf("create invite failed: %v", err)
	}
	if res.Code != "ABC-DEF-GHI" {
		t.Errorf("expected code ABC-DEF-GHI, got %s", res.Code)
	}
}

func TestClient_ListAcceptances_SendsAuthHeaders(t *testing.T) {
	id := generateTestIdentity(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != ACCEPTANCES_ENDPOINT {
			http.NotFound(w, r)
			return
		}
		peerID := r.URL.Query().Get("peer_id")
		timestampStr := r.Header.Get(auth.HeaderTimestamp)
		sig := r.Header.Get(auth.HeaderSignature)

		if timestampStr == "" || sig == "" {
			http.Error(w, "missing auth headers", http.StatusUnauthorized)
			return
		}
		ts, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid timestamp", http.StatusBadRequest)
			return
		}

		payload := auth.AcceptancesPayload(peerID, ts)
		if err := auth.Verify(peerID, payload, sig, ts); err != nil {
			http.Error(w, "invalid signature: "+err.Error(), http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ListAcceptancesResponse{
			Acceptances: []InviteAcceptance{
				{
					InviteID:   "inv-1",
					PeerID:     "12D3KooWPeerB",
					Addresses:  []string{"/ip4/1.2.3.4/udp/4001/quic-v1"},
					AcceptedAt: time.Now(),
				},
			},
		})
	}))
	defer server.Close()

	c, err := NewClient(server.URL, id)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	res, err := c.ListAcceptances(context.Background(), id.PeerID)
	if err != nil {
		t.Fatalf("list acceptances failed: %v", err)
	}
	if len(res.Acceptances) != 1 || res.Acceptances[0].InviteID != "inv-1" {
		t.Errorf("unexpected acceptances: %+v", res)
	}
}
