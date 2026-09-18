package discovery

type AnnounceRequest struct {
	PeerID    string   `json:"peer_id"`
	Addresses []string `json:"addresses"`
}

type InviteRequest struct {
	PeerID string `json:"peer_id"`
}

type InviteResponse struct {
	Code string `json:"code"`
}

type ResolveResponse struct {
	PeerID    string   `json:"peer_id"`
	Addresses []string `json:"addresses"`
}
