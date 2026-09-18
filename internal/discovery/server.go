package discovery

import (
	"encoding/json"
	"net/http"
	"strings"
)

type Server struct {
	store *Store
}

func NewServer(store *Store) *Server {
	return &Server{
		store: store,
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/announce", s.handleAnnounce)
	mux.HandleFunc("/heartbeat", s.handleHeartbeat)
	mux.HandleFunc("/invite", s.handleInvite)
	mux.HandleFunc("/resolve/", s.handleResolve)

	return mux
}

func (s *Server) handleAnnounce(
	w http.ResponseWriter,
	r *http.Request,
) {
	var req AnnounceRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.PeerID == "" || len(req.Addresses) == 0 {
		http.Error(w, "peer_id and addresses are required", http.StatusBadRequest)
		return
	}

	if err := s.store.SetPeer(
		r.Context(),
		req.PeerID,
		req.Addresses,
	); err != nil {
		http.Error(w, "failed to store peer", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleHeartbeat(
	w http.ResponseWriter,
	r *http.Request,
) {
	var req AnnounceRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.PeerID == "" || len(req.Addresses) == 0 {
		http.Error(w, "peer_id and addresses are required", http.StatusBadRequest)
		return
	}

	if err := s.store.SetPeer(
		r.Context(),
		req.PeerID,
		req.Addresses,
	); err != nil {
		http.Error(w, "failed to update peer", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleInvite(
	w http.ResponseWriter,
	r *http.Request,
) {
	var req InviteRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.PeerID == "" {
		http.Error(w, "peer_id is required", http.StatusBadRequest)
		return
	}

	code, err := GenerateInviteCode()
	if err != nil {
		http.Error(w, "failed to generate invite", http.StatusInternalServerError)
		return
	}

	if err := s.store.SetInvite(
		r.Context(),
		code,
		req.PeerID,
	); err != nil {
		http.Error(w, "failed to store invite", http.StatusInternalServerError)
		return
	}

	response := InviteResponse{
		Code: code,
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		return
	}
}

func (s *Server) handleResolve(
	w http.ResponseWriter,
	r *http.Request,
) {
	code := strings.TrimPrefix(r.URL.Path, "/resolve/")

	if code == "" {
		http.Error(w, "invite code is required", http.StatusBadRequest)
		return
	}

	peerID, err := s.store.GetAndDeleteInvite(
		r.Context(),
		code,
	)
	if err != nil {
		http.Error(w, "invalid or expired invite", http.StatusNotFound)
		return
	}

	addresses, err := s.store.GetPeer(
		r.Context(),
		peerID,
	)
	if err != nil {
		http.Error(w, "peer is offline", http.StatusNotFound)
		return
	}

	response := ResolveResponse{
		PeerID:    peerID,
		Addresses: addresses,
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		return
	}
}
