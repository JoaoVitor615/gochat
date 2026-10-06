package client

import "time"

type CreateInviteReq struct {
	PeerID string `json:"peer_id"`
}

type CreateInviteResponse struct {
	Code string `json:"code"`
}

type AddPeerResponse struct {
	PeerID    string   `json:"peer_id"`
	Addresses []string `json:"addresses"`
}

type InviteAcceptance struct {
	InviteID   string    `json:"invite_id"`
	PeerID     string    `json:"peer_id"`
	Addresses  []string  `json:"addresses"`
	AcceptedAt time.Time `json:"accepted_at"`
}

type ListAcceptancesResponse struct {
	Acceptances []InviteAcceptance `json:"acceptances"`
}

type ResolveInviteReq struct {
	Code   string `json:"code"`
	PeerID string `json:"peer_id"`
}

type AcknowledgeAcceptancesReq struct {
	PeerID    string   `json:"peer_id"`
	InviteIDs []string `json:"invite_ids"`
}

type HeartbeatReq struct {
	PeerID    string   `json:"peer_id"`
	Addresses []string `json:"addresses"`
}

type ObserverInfo struct {
	Address string `json:"address"`
}
