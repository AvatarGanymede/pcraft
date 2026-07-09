package lark

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const userAgent = "pcraft/1.0"

// tokenRefreshSkew refreshes the tenant_access_token slightly before it
// actually expires so an in-flight request never races the boundary.
const tokenRefreshSkew = 60 * time.Second

// Client is the Feishu API surface pcraft depends on.
type Client interface {
	// SendTextByEmail sends a plain-text message to the user with the given
	// email and returns the created message id.
	SendTextByEmail(ctx context.Context, email, text string) (string, error)
}

// HTTPClient is the real Feishu bot client. It caches the tenant_access_token
// and refreshes it on demand.
type HTTPClient struct {
	http        *http.Client
	baseURL     string
	appID       string
	appSecret   string
	maxBodySize int64

	mu          sync.Mutex
	token       string
	tokenExpiry time.Time
}

// NewHTTPClient builds a client from app credentials + base domain.
func NewHTTPClient(baseDomain, appID, appSecret string) *HTTPClient {
	base := strings.TrimRight(strings.TrimSpace(baseDomain), "/")
	if base == "" {
		base = DefaultBaseDomain
	}
	return &HTTPClient{
		http:        &http.Client{Timeout: 30 * time.Second},
		baseURL:     base,
		appID:       appID,
		appSecret:   appSecret,
		maxBodySize: 4 << 20,
	}
}

// SendTextByEmail delivers a plain-text direct message addressed by email.
func (c *HTTPClient) SendTextByEmail(ctx context.Context, email, text string) (string, error) {
	token, err := c.ensureToken(ctx)
	if err != nil {
		return "", err
	}
	// Feishu text content is a JSON object {"text": "..."} serialized to a
	// string and placed in the request's `content` field.
	contentBytes, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		return "", err
	}
	body := map[string]string{
		"receive_id": email,
		"msg_type":   "text",
		"content":    string(contentBytes),
	}
	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			MessageID string `json:"message_id"`
		} `json:"data"`
	}
	path := "/open-apis/im/v1/messages?receive_id_type=email"
	if err := c.do(ctx, http.MethodPost, path, token, body, &resp); err != nil {
		return "", err
	}
	if resp.Code != 0 {
		return "", &APIError{Code: resp.Code, Message: resp.Msg}
	}
	return resp.Data.MessageID, nil
}

// ensureToken returns a cached tenant_access_token, refreshing when missing or
// close to expiry.
func (c *HTTPClient) ensureToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.tokenExpiry.Add(-tokenRefreshSkew)) {
		return c.token, nil
	}
	body := map[string]string{"app_id": c.appID, "app_secret": c.appSecret}
	var resp struct {
		Code              int    `json:"code"`
		Msg               string `json:"msg"`
		TenantAccessToken string `json:"tenant_access_token"`
		Expire            int    `json:"expire"`
	}
	path := "/open-apis/auth/v3/tenant_access_token/internal"
	if err := c.do(ctx, http.MethodPost, path, "", body, &resp); err != nil {
		return "", err
	}
	if resp.Code != 0 || resp.TenantAccessToken == "" {
		return "", &APIError{Code: resp.Code, Message: resp.Msg}
	}
	c.token = resp.TenantAccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(resp.Expire) * time.Second)
	return c.token, nil
}

// do executes a JSON request, optionally bearer-authorized, decoding the 2xx
// body into out.
func (c *HTTPClient) do(ctx context.Context, method, path, token string, body, out interface{}) error {
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("lark: marshal body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, c.maxBodySize))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{StatusCode: resp.StatusCode, Message: summarize(raw)}
	}
	if out == nil || len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("lark: decode response: %w", err)
	}
	return nil
}

func summarize(raw []byte) string {
	const maxMsg = 500
	if len(raw) > maxMsg {
		return string(raw[:maxMsg]) + "…"
	}
	return string(raw)
}
