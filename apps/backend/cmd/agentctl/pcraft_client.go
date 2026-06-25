package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// pcraftClient is a thin HTTP client for the pcraft office API.
// It reads configuration from environment variables set by the pcraft backend
// when launching agent containers or processes.
type pcraftClient struct {
	apiURL      string // PCRAFT_API_URL
	apiKey      string // PCRAFT_API_KEY
	runID       string // PCRAFT_RUN_ID
	agentID     string // PCRAFT_AGENT_ID
	taskID      string // PCRAFT_TASK_ID (default for --id/--task flags)
	workspaceID string // PCRAFT_WORKSPACE_ID
	http        *http.Client
}

// newPcraftClient creates a client from environment variables.
// Returns an error if required variables (PCRAFT_API_URL, PCRAFT_API_KEY) are missing.
func newPcraftClient() (*pcraftClient, error) {
	apiURL := os.Getenv("PCRAFT_API_URL")
	apiKey := os.Getenv("PCRAFT_API_KEY")
	if apiURL == "" || apiKey == "" {
		return nil, fmt.Errorf("PCRAFT_API_URL and PCRAFT_API_KEY must be set")
	}
	return &pcraftClient{
		apiURL:      apiURL,
		apiKey:      apiKey,
		runID:       os.Getenv("PCRAFT_RUN_ID"),
		agentID:     os.Getenv("PCRAFT_AGENT_ID"),
		taskID:      os.Getenv("PCRAFT_TASK_ID"),
		workspaceID: os.Getenv("PCRAFT_WORKSPACE_ID"),
		http:        &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// isMutating returns true for methods that modify server state.
func isMutating(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete:
		return true
	}
	return false
}

// do sends an HTTP request to the pcraft API and returns the response body,
// status code, and any error. It sets Authorization and, for mutating requests,
// X-Pcraft-Run-Id headers automatically.
func (c *pcraftClient) do(method, path string, body any) ([]byte, int, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	url := c.apiURL + path
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	if isMutating(method) && c.runID != "" {
		req.Header.Set("X-Pcraft-Run-Id", c.runID)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("http request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response: %w", err)
	}
	return respBody, resp.StatusCode, nil
}
