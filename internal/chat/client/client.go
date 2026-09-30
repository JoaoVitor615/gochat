package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	BaseURL string
}

func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
	}
}

func (c *Client) CreateInvite(peerID string) (*CreateInviteResponse, error) {
	reqBody := &CreateInviteReq{
		PeerID: peerID,
	}

	reqBodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	res, err := http.Post(c.BaseURL+CREATE_INVITE_ENDPOINT, "application/json", bytes.NewReader(reqBodyBytes))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to create invite: %s", body)
	}

	var createInviteRes CreateInviteResponse
	if err := json.Unmarshal(body, &createInviteRes); err != nil {
		return nil, err
	}
	return &createInviteRes, nil
}

func (c *Client) AddPeer(inviteCode string) (*AddPeerResponse, error) {
	res, err := http.Get(c.BaseURL + ADD_PEER_ENDPOINT + "/" + inviteCode)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var addPeerRes AddPeerResponse
	if err := json.Unmarshal(body, &addPeerRes); err != nil {
		return nil, err
	}
	return &addPeerRes, nil
}
