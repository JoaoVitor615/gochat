package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const maxResponseBodySize = 64 * 1024

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) (*Client, error) {
	if baseURL == "" {
		return nil, errors.New("baseURL cannot be empty")
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

// APIError represents an error response returned by the Discovery API.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	if e == nil {
		return "discovery API error"
	}
	if e.Message == "" {
		return fmt.Sprintf("discovery API returned HTTP %d", e.StatusCode)
	}
	return fmt.Sprintf("discovery API returned HTTP %d: %s", e.StatusCode, e.Message)
}

type apiErrorResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

func (c *Client) CreateInvite(ctx context.Context, peerID string) (*CreateInviteResponse, error) {
	reqBody := &CreateInviteReq{
		PeerID: peerID,
	}

	var createInviteRes CreateInviteResponse
	if err := c.doJSON(ctx, http.MethodPost, CREATE_INVITE_ENDPOINT, reqBody, &createInviteRes); err != nil {
		return nil, fmt.Errorf("create invite: %w", err)
	}
	return &createInviteRes, nil
}

func (c *Client) AddPeer(ctx context.Context, inviteCode string) (*AddPeerResponse, error) {
	reqBody := &ResolveInviteReq{Code: inviteCode}

	var addPeerRes AddPeerResponse
	if err := c.doJSON(ctx, http.MethodPost, ADD_PEER_ENDPOINT, reqBody, &addPeerRes); err != nil {
		return nil, fmt.Errorf("resolve invite: %w", err)
	}
	return &addPeerRes, nil
}

func (c *Client) Heartbeat(ctx context.Context, peerID string, addresses []string) error {
	reqBody := &HeartbeatReq{PeerID: peerID, Addresses: addresses}
	if err := c.doJSON(ctx, http.MethodPost, HEARTBEAT_ENDPOINT, reqBody, nil); err != nil {
		return fmt.Errorf("heartbeat: %w", err)
	}
	return nil
}

func (c *Client) GetObserver(ctx context.Context) (*ObserverInfo, error) {
	var observerInfo ObserverInfo
	if err := c.doJSON(ctx, http.MethodGet, OBSERVER_ENDPOINT, nil, &observerInfo); err != nil {
		return nil, fmt.Errorf("get public address observer: %w", err)
	}
	return &observerInfo, nil
}

func (c *Client) doJSON(ctx context.Context, method, endpoint string, requestBody, responseBody any) error {
	if c == nil || c.httpClient == nil {
		return errors.New("discovery client is not initialized")
	}

	var requestReader io.Reader
	if requestBody != nil {
		requestBytes, err := json.Marshal(requestBody)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		requestReader = bytes.NewReader(requestBytes)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		method,
		c.baseURL+endpoint,
		requestReader,
	)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	if requestBody != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer response.Body.Close()

	responseBytes, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBodySize+1))
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if len(responseBytes) > maxResponseBodySize {
		return errors.New("discovery API response is too large")
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return newAPIError(response.StatusCode, responseBytes)
	}
	if responseBody == nil {
		return nil
	}

	decoder := json.NewDecoder(bytes.NewReader(responseBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(responseBody); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("discovery API returned an invalid response")
	}

	return nil
}

func newAPIError(httpStatus int, body []byte) error {
	apiErr := &APIError{StatusCode: httpStatus}

	var response apiErrorResponse
	if err := json.Unmarshal(body, &response); err == nil && response.Status == httpStatus && response.Message != "" {
		apiErr.Message = response.Message
	}

	return apiErr
}
