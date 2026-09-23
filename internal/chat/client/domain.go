package client

type CreateInviteReq struct {
	PeerID string `json:"peer_id"`
}

type CreateInviteResponse struct {
	Code string `json:"code"`
}

type AddPeerResponse struct {
	PeerID  string `json:"peer_id"`
	Address string `json:"address"`
}
