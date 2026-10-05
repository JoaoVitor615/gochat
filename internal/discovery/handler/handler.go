package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/JoaoVitor615/gochat/internal/discovery/observation"
	"github.com/JoaoVitor615/gochat/internal/discovery/service"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

const (
	maxRequestBodySize = 64 * 1024
	maxPeerAddresses   = 16
)

type Handler struct {
	service  *service.Service
	observer *observation.Observer
}

type peerRequest struct {
	PeerID    string   `json:"peer_id"`
	Addresses []string `json:"addresses"`
}

type inviteRequest struct {
	PeerID string `json:"peer_id"`
}

type resolveRequest struct {
	Code string `json:"code"`
}

type inviteResponse struct {
	Code string `json:"code"`
}

type resolveResponse struct {
	PeerID    string   `json:"peer_id"`
	Addresses []string `json:"addresses"`
}

type observerResponse struct {
	Address string `json:"address"`
}

type errorResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

func New(discoveryService *service.Service, addressObserver *observation.Observer) *Handler {
	return &Handler{service: discoveryService, observer: addressObserver}
}

func (h *Handler) NotFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotFound, "route not found")
}

func (h *Handler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	req, ok := decodePeerRequest(w, r)
	if !ok {
		return
	}

	if err := h.service.Heartbeat(r.Context(), req.PeerID, req.Addresses); err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Invite(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	var req inviteRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if !validPeerID(req.PeerID) {
		writeError(w, http.StatusBadRequest, "invalid peer_id")
		return
	}

	code, err := h.service.CreateInvite(r.Context(), req.PeerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, inviteResponse{Code: code})
}

func (h *Handler) Resolve(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	var req resolveRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Code == "" {
		writeError(w, http.StatusBadRequest, "code is required")
		return
	}

	resolvedPeer, err := h.service.ResolveInvite(r.Context(), req.Code)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInvite):
			writeError(w, http.StatusNotFound, "invalid or expired invite")
		case errors.Is(err, service.ErrPeerOffline):
			writeError(w, http.StatusNotFound, "peer is offline")
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	writeJSON(w, http.StatusOK, resolveResponse{
		PeerID:    resolvedPeer.PeerID,
		Addresses: resolvedPeer.Addresses,
	})
}

// Observer returns the public libp2p multiaddress of the address observer.
// It contains connectivity metadata only and never carries chat messages.
func (h *Handler) Observer(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	if h.observer == nil || !h.observer.Enabled() {
		writeError(w, http.StatusServiceUnavailable, "public address observer is not configured")
		return
	}
	address, err := h.observer.Address()
	if err != nil {
		if errors.Is(err, observation.ErrUnavailable) {
			writeError(w, http.StatusServiceUnavailable, "public address observer is not configured")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not get observer address")
		return
	}
	writeJSON(w, http.StatusOK, observerResponse{Address: address})
}

func decodePeerRequest(w http.ResponseWriter, r *http.Request) (peerRequest, bool) {
	var req peerRequest
	if !decodeJSON(w, r, &req) {
		return peerRequest{}, false
	}
	if !validPeerID(req.PeerID) {
		writeError(w, http.StatusBadRequest, "invalid peer_id")
		return peerRequest{}, false
	}
	if len(req.Addresses) == 0 || len(req.Addresses) > maxPeerAddresses {
		writeError(w, http.StatusBadRequest, "addresses must contain between 1 and 16 entries")
		return peerRequest{}, false
	}
	for _, address := range req.Addresses {
		if _, err := multiaddr.NewMultiaddr(address); err != nil {
			writeError(w, http.StatusBadRequest, "invalid address")
			return peerRequest{}, false
		}
	}

	return req, true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, value any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
		} else {
			writeError(w, http.StatusBadRequest, "invalid request body")
		}
		return false
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return false
	}

	return true
}

func validPeerID(value string) bool {
	_, err := peer.Decode(value)
	return err == nil
}

func requireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method == method {
		return true
	}

	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	return false
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Status: status, Message: message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	payload, err := json.Marshal(value)
	if err != nil {
		payload = []byte(`{"status":500,"message":"internal server error"}`)
		status = http.StatusInternalServerError
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(append(payload, '\n')); err != nil {
		return
	}
}
