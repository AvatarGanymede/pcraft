package process

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AvatarGanymede/pcraft/internal/agentctl/server/adapter"
	"github.com/AvatarGanymede/pcraft/internal/common/ports"
	"go.uber.org/zap"
)

type p4CheckoutResponse struct {
	Allowed        bool   `json:"allowed"`
	ConflictTaskID string `json:"conflict_task_id,omitempty"`
}

type p4ConflictRequest struct {
	ConflictTaskID string `json:"conflict_task_id"`
	SessionID      string `json:"session_id"`
}

func p4APIBaseURL() string {
	if v := strings.TrimSpace(os.Getenv("PCRAFT_API_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}
	return fmt.Sprintf("http://127.0.0.1:%d/api/v1", ports.Backend)
}

func isP4WritePermission(req *adapter.PermissionRequest) bool {
	action := strings.ToLower(strings.TrimSpace(req.ActionType))
	switch action {
	case "edit", "write", "file_write":
		return true
	}
	title := strings.ToLower(req.Title)
	return strings.HasPrefix(title, "write ") ||
		strings.HasPrefix(title, "edit ") ||
		strings.Contains(title, "write file") ||
		strings.Contains(title, "edit file")
}

func extractFilePathsFromPermission(req *adapter.PermissionRequest, workDir string) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return
		}
		resolved := resolveP4FilePath(raw, workDir)
		if resolved == "" {
			return
		}
		if _, ok := seen[resolved]; ok {
			return
		}
		seen[resolved] = struct{}{}
		out = append(out, resolved)
	}

	if req.ActionDetails != nil {
		if rawInput, ok := req.ActionDetails["raw_input"].(map[string]interface{}); ok {
			collectPathsFromMap(rawInput, add)
		}
	}
	return out
}

func collectPathsFromMap(raw map[string]interface{}, add func(string)) {
	if raw == nil {
		return
	}
	for _, key := range []string{"path", "file_path", "filePath"} {
		if v, ok := raw[key].(string); ok {
			add(v)
		}
	}
	if paths, ok := raw["paths"].([]interface{}); ok {
		for _, item := range paths {
			if s, ok := item.(string); ok {
				add(s)
			}
		}
	}
	if locations, ok := raw["locations"].([]interface{}); ok {
		for _, item := range locations {
			switch loc := item.(type) {
			case string:
				add(loc)
			case map[string]interface{}:
				if p, ok := loc["path"].(string); ok {
					add(p)
				}
			}
		}
	}
}

func resolveP4FilePath(raw, workDir string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if filepath.IsAbs(raw) {
		return filepath.Clean(raw)
	}
	if workDir == "" {
		return filepath.Clean(raw)
	}
	return filepath.Clean(filepath.Join(workDir, raw))
}

func requestP4Checkout(ctx context.Context, baseURL, taskID string, files []string) (allowed bool, conflictTaskID string, err error) {
	payload, err := json.Marshal(map[string]any{"files": files})
	if err != nil {
		return false, "", err
	}
	url := baseURL + "/tasks/" + taskID + "/p4/checkout"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return false, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, "", err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, "", err
	}
	if resp.StatusCode == http.StatusConflict {
		var result p4CheckoutResponse
		if err := json.Unmarshal(body, &result); err != nil {
			return false, "", fmt.Errorf("decode conflict response: %w", err)
		}
		return false, result.ConflictTaskID, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, "", fmt.Errorf("p4 checkout HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var result p4CheckoutResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return true, "", nil
	}
	return result.Allowed, result.ConflictTaskID, nil
}

func notifyP4Conflict(baseURL, taskID, sessionID, conflictTaskID string) {
	payload, err := json.Marshal(p4ConflictRequest{
		ConflictTaskID: conflictTaskID,
		SessionID:      sessionID,
	})
	if err != nil {
		return
	}
	url := baseURL + "/tasks/" + taskID + "/p4/conflict"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	_ = resp.Body.Close()
}

// maybeCheckoutP4Files runs scheme-A checkout before allowing Write/Edit tools.
func (m *Manager) maybeCheckoutP4Files(ctx context.Context, req *adapter.PermissionRequest) error {
	if !isP4WritePermission(req) {
		return nil
	}
	taskID := strings.TrimSpace(m.cfg.TaskID)
	if taskID == "" {
		taskID = strings.TrimSpace(os.Getenv("PCRAFT_TASK_ID"))
	}
	if taskID == "" {
		return nil
	}
	files := extractFilePathsFromPermission(req, m.cfg.WorkDir)
	if len(files) == 0 {
		return nil
	}

	checkCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	baseURL := p4APIBaseURL()
	allowed, conflictID, err := requestP4Checkout(checkCtx, baseURL, taskID, files)
	if err != nil {
		if m.logger != nil {
			m.logger.Warn("p4 checkout failed; denying write",
				zap.String("task_id", taskID),
				zap.Error(err))
		}
		return fmt.Errorf("p4 checkout failed: %w", err)
	}
	if !allowed {
		sessionID := strings.TrimSpace(req.SessionID)
		if sessionID == "" {
			sessionID = strings.TrimSpace(m.cfg.SessionID)
		}
		go notifyP4Conflict(baseURL, taskID, sessionID, conflictID)
		if m.logger != nil {
			m.logger.Info("p4 checkout denied due to file conflict",
				zap.String("task_id", taskID),
				zap.String("conflict_task_id", conflictID),
				zap.Strings("files", files))
		}
		return fmt.Errorf("p4 file locked by task %s", conflictID)
	}
	return nil
}
