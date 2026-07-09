package jnpm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const userAgent = "pcraft/1.0"

// Client is the JNPM API surface pcraft depends on. Kept intentionally tiny —
// only issue lookup is needed for assignee resolution.
type Client interface {
	GetIssue(ctx context.Context, issueID int) (*IssueDetail, error)
}

// HTTPClient is the real JNPM REST client. It authenticates with a
// `PRIVATE-TOKEN` header and holds no state beyond credentials.
type HTTPClient struct {
	http        *http.Client
	baseURL     string
	token       string
	maxBodySize int64
}

// NewHTTPClient builds a client from a base URL + private token.
func NewHTTPClient(baseURL, token string) *HTTPClient {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		base = strings.TrimRight(DefaultBaseURL, "/")
	}
	return &HTTPClient{
		http:        &http.Client{Timeout: 30 * time.Second},
		baseURL:     base,
		token:       token,
		maxBodySize: 4 << 20, // 4 MB — issue payloads are small.
	}
}

// jnpmUser mirrors the user shape JNPM embeds under assignee/assignedTo/owner.
type jnpmUser struct {
	Email    string `json:"email"`
	FullName string `json:"fullName"`
	Name     string `json:"name"`
	Username string `json:"username"`
}

// issueEnvelope is the outer JNPM response; the useful data lives in `payload`.
type issueEnvelope struct {
	Payload issuePayload `json:"payload"`
}

type issuePayload struct {
	Title      string    `json:"title"`
	Assignee   *jnpmUser `json:"assignee"`
	AssignedTo *jnpmUser `json:"assignedTo"`
	Owner      *jnpmUser `json:"owner"`
}

// GetIssue fetches a single issue and projects it to IssueDetail.
func (c *HTTPClient) GetIssue(ctx context.Context, issueID int) (*IssueDetail, error) {
	path := "/v1/open-api/projects/issues/" + strconv.Itoa(issueID)
	var env issueEnvelope
	if err := c.do(ctx, http.MethodGet, path, &env); err != nil {
		return nil, err
	}
	return &IssueDetail{
		IssueID:  issueID,
		Title:    env.Payload.Title,
		Assignee: extractAssignee(env.Payload),
	}, nil
}

// extractAssignee mirrors ai_summarize's precedence: assignee → assignedTo →
// owner. The first non-nil candidate wins.
func extractAssignee(p issuePayload) Assignee {
	for _, u := range []*jnpmUser{p.Assignee, p.AssignedTo, p.Owner} {
		if u == nil {
			continue
		}
		name := firstNonEmpty(u.FullName, u.Name, u.Username)
		return Assignee{
			Name:     name,
			Username: firstNonEmpty(u.Username, u.Name),
			Email:    u.Email,
		}
	}
	return Assignee{}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// do executes a request and decodes a 2xx JSON body into out (may be nil).
func (c *HTTPClient) do(ctx context.Context, method, path string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("PRIVATE-TOKEN", c.token)
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
		return fmt.Errorf("jnpm: decode response: %w", err)
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
