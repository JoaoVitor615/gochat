package client

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

type Client struct {
	BaseURL string
}

type CreateInviteReq struct {
	PeerID string `json:"peer_id"`
}

func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
	}
}

func (c *Client) CreateInvite(peerID string) (string, error) {
	reqBody := &CreateInviteReq{
		PeerID: peerID,
	}

	reqBodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(c.BaseURL+CREATE_INVITE_ENDPOINT, "application/json", bytes.NewReader(reqBodyBytes))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
