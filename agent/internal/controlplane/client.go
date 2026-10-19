package controlplane

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

	"github.com/ekkywi/sailguard/pkgs/agentcontract"
)

// ErrDeviceDisabled is returned when the control plane rejects the device
// because an admin set status=disabled (HTTP 403, code device_disabled).
var ErrDeviceDisabled = errors.New("device disabled")

// ErrUnauthorized is returned when a device Bearer token is rejected
// (HTTP 401, code unauthorized — invalid/revoked device credential).
var ErrUnauthorized = errors.New("device unauthorized")

// ErrEnrollmentInvalid is returned when an enrollment token is rejected
// (HTTP 401, code enrollment_invalid — not a device-token failure).
var ErrEnrollmentInvalid = errors.New("enrollment token invalid")

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	UserAgent  string
}

func apiError(status int, err *agentcontract.Error, raw []byte, op string) error {
	if err != nil {
		switch err.Code {
		case "device_disabled":
			return fmt.Errorf("%s: %w", op, ErrDeviceDisabled)
		case "enrollment_invalid":
			return fmt.Errorf("%s: %w", op, ErrEnrollmentInvalid)
		}
	}
	// Device auth only — do not treat enrollment_invalid (also HTTP 401) as this.
	if status == http.StatusUnauthorized && (err == nil || err.Code == "unauthorized" || err.Code == "") {
		return fmt.Errorf("%s: %w", op, ErrUnauthorized)
	}
	if err != nil {
		return fmt.Errorf("%s rejected (%d): %s: %s", op, status, err.Code, err.Message)
	}
	return fmt.Errorf("%s rejected (%d): %s", op, status, strings.TrimSpace(string(raw)))
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
		return nil, apiError(res.StatusCode, env.Error, raw, "enroll")
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
		return nil, apiError(res.StatusCode, env.Error, raw, "events")
	}
	return env.Data, nil
}

func (c *Client) GetPolicy(
	ctx context.Context,
	deviceToken string,
	currentHash string,
) (*agentcontract.PolicySyncData, error) {
	if c.BaseURL == "" {
		return nil, fmt.Errorf("server URL is empty")
	}
	if strings.TrimSpace(deviceToken) == "" {
		return nil, fmt.Errorf("device token is empty")
	}

	url := c.BaseURL + "/v1/agent/policy"
	if h := strings.TrimSpace(currentHash); h != "" {
		url = url + "?hash=" + h
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build policy request: %w", err)
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+deviceToken)
	if c.UserAgent != "" {
		httpReq.Header.Set("User-Agent", c.UserAgent)
	}

	res, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("policy request failed: %w", err)
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read policy response: %w", err)
	}

	var env agentcontract.Envelope[agentcontract.PolicySyncData]
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("decode policy response (status %d): %w", res.StatusCode, err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 || !env.OK || env.Data == nil {
		return nil, apiError(res.StatusCode, env.Error, raw, "policy")
	}
	return env.Data, nil
}