package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ekkywi/sailguard/pkgs/agentcontract"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	UserAgent  string
}

func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		UserAgent: "sailguard-agent",
	}
}

func (c *Client) Enroll(ctx context.Context, req agentcontract.EnrollRequest) (*agentcontract.EnrollResponse, error) {
	if c.BaseURL == "" {
		return nil, fmt.Errorf("server URL is empty")
	}
	if req.SchemaVersion == 0 {
		req.SchemaVersion = 1
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal enroll request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.BaseURL+"/v1/agent/enroll",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("build enroll request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	if c.UserAgent != "" {
		httpReq.Header.Set("User-Agent", c.UserAgent)
	}

	res, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("enroll request failed: %w", err)
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read enroll response: %w", err)
	}

	var env agentcontract.Envelope[agentcontract.EnrollResponse]
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("decode enroll response (status %d): %w", res.StatusCode, err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 || !env.OK || env.Data == nil {
		if env.Error != nil {
			return nil, fmt.Errorf("enroll rejected: (%d): %s: %s",
				res.StatusCode, env.Error.Code, env.Error.Message,
			)
		}
		return nil, fmt.Errorf("enroll rejected (%d): %s", res.StatusCode, strings.TrimSpace(string(raw)))
	}

	if env.Data.DeviceID == "" || env.Data.DeviceToken == "" {
		return nil, fmt.Errorf("enroll response missing device_id or device_token")
	}
	return env.Data, nil
}

func (c *Client) PostEvents(
	ctx context.Context,
	deviceToken string,
	req agentcontract.EventBatchRequest,
) (*agentcontract.EventBatchResponse, error) {
	if c.BaseURL == "" {
		return nil, fmt.Errorf("server URL is empty")
	}
	if strings.TrimSpace(deviceToken) == "" {
		return nil, fmt.Errorf("device token is empty")
	}
	if req.SchemaVersion == 0 {
		req.SchemaVersion = 1
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal event request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.BaseURL+"/v1/agent/events",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("build events request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+deviceToken)
	if c.UserAgent != "" {
		httpReq.Header.Set("User-Agent", c.UserAgent)
	}

	res, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("events request failed: %w", err)
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read events response: %w", err)
	}

	var env agentcontract.Envelope[agentcontract.EventBatchResponse]
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("decode events response (status %d): %w", res.StatusCode, err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 || !env.OK || env.Data == nil {
		if env.Error != nil {
			return nil, fmt.Errorf("events rejected (%d): %s: %s", res.StatusCode, env.Error.Code, env.Error.Message)
		}
		return nil, fmt.Errorf("events rejected (%d): %s", res.StatusCode, strings.TrimSpace(string(raw)))
	}
	return env.Data, nil
}
