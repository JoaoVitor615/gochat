package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/JoaoVitor615/gochat/internal/discovery/service"
)

type Handler struct {
	service *service.Service
}

type announceRequest struct {
	PeerID    string   `json:"peer_id"`
	Addresses []string `json:"addresses"`
}

type inviteRequest struct {
	PeerID string `json:"peer_id"`
}

type inviteResponse struct {
	Code string `json:"code"`
}

type resolveResponse struct {
	PeerID    string   `json:"peer_id"`
	Addresses []string `json:"addresses"`
}

func New(discoveryService *service.Service) *Handler {
	return &Handler{service: discoveryService}
}

func (h *Handler) Announce(w http.ResponseWriter, r *http.Request) {
	req, ok := h.decodePeerRequest(w, r)
	if !ok {
		return
	}

	if err := h.service.Announce(r.Context(), req.PeerID, req.Addresses); err != nil {
		http.Error(w, "failed to store peer", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	req, ok := h.decodePeerRequest(w, r)
	if !ok {
		return
	}

	if err := h.service.Heartbeat(r.Context(), req.PeerID, req.Addresses); err != nil {
		http.Error(w, "failed to update peer", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Invite(w http.ResponseWriter, r *http.Request) {
	var req inviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.PeerID == "" {
		http.Error(w, "peer_id is required", http.StatusBadRequest)
		return
	}

	code, err := h.service.CreateInvite(r.Context(), req.PeerID)
	if err != nil {
		http.Error(w, "failed to store invite", http.StatusInternalServerError)
		return
	}

	writeJSON(w, inviteResponse{Code: code})
}

func (h *Handler) Resolve(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimPrefix(r.URL.Path, "/resolve/")
	if code == "" {
		http.Error(w, "invite code is required", http.StatusBadRequest)
		return
	}

	peer, err := h.service.ResolveInvite(r.Context(), code)
	if err != nil {
		if errors.Is(err, service.ErrPeerOffline) {
			http.Error(w, "peer is offline", http.StatusNotFound)
			return
		}

		http.Error(w, "invalid or expired invite", http.StatusNotFound)
		return
	}

	writeJSON(w, resolveResponse{PeerID: peer.PeerID, Addresses: peer.Addresses})
}

func (h *Handler) decodePeerRequest(w http.ResponseWriter, r *http.Request) (announceRequest, bool) {
	var req announceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return announceRequest{}, false
	}
	if req.PeerID == "" || len(req.Addresses) == 0 {
		http.Error(w, "peer_id and addresses are required", http.StatusBadRequest)
		return announceRequest{}, false
	}

	return req, true
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
