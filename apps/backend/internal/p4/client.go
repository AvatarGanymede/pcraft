package p4

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Client is the minimal p4 command surface used by pcraft: listing the
// developer's clients and resolving a bound client's root. Actual p4
// operations (edit/revert/submit/changelists) are performed by the agent
// itself via workflow-step prompts, not by pcraft.
type Client interface {
	ListClients(ctx context.Context, p4user string) ([]string, error)
	CurrentUser(ctx context.Context) (string, error)
	// GetClientRoot resolves a client's local Root and (optional) Stream via
	// `p4 client -o <name>`. Root is the on-disk workspace directory pcraft
	// uses as a task's working directory.
	GetClientRoot(ctx context.Context, clientName string) (root, stream string, err error)
}

type CLIClient struct{}

func NewCLIClient() *CLIClient {
	return &CLIClient{}
}

func (c *CLIClient) ListClients(ctx context.Context, p4user string) ([]string, error) {
	args := []string{"clients"}
	if strings.TrimSpace(p4user) != "" {
		args = append(args, "-u", p4user)
	}
	out, err := runP4(ctx, args...)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(out, "\n")
	clients := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Client ") {
			parts := strings.Split(trimmed, " ")
			if len(parts) >= 2 {
				clients = append(clients, strings.TrimSpace(parts[1]))
			}
		}
	}
	return clients, nil
}

// CurrentUser resolves the configured P4USER from the p4 environment
// (P4CONFIG/P4ENVIRO/env vars). `p4 set P4USER` prints a line like
// "P4USER=beilin (enviro)"; we strip the key and the trailing source tag.
func (c *CLIClient) CurrentUser(ctx context.Context) (string, error) {
	out, err := runP4(ctx, "set", "P4USER")
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "P4USER=") {
			continue
		}
		value := strings.TrimPrefix(trimmed, "P4USER=")
		if idx := strings.Index(value, " ("); idx >= 0 {
			value = value[:idx]
		}
		return strings.TrimSpace(value), nil
	}
	return "", nil
}

// GetClientRoot runs `p4 client -o <name>` and parses the client spec's
// `Root:` and `Stream:` fields. The spec prints one field per line, e.g.
// "Root:\t/home/user/ws" and "Stream:\t//depot/main"; comment lines (starting
// with '#') and other fields are ignored. Stream is empty for classic
// (non-stream) clients.
func (c *CLIClient) GetClientRoot(ctx context.Context, clientName string) (string, string, error) {
	name := strings.TrimSpace(clientName)
	if name == "" {
		return "", "", fmt.Errorf("client name is required")
	}
	out, err := runP4(ctx, "client", "-o", name)
	if err != nil {
		return "", "", fmt.Errorf("p4 client -o %s: %w: %s", name, err, strings.TrimSpace(out))
	}
	var root, stream string
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "Root:") {
			root = strings.TrimSpace(strings.TrimPrefix(trimmed, "Root:"))
		} else if strings.HasPrefix(trimmed, "Stream:") {
			stream = strings.TrimSpace(strings.TrimPrefix(trimmed, "Stream:"))
		}
	}
	return root, stream, nil
}

func runP4(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "p4", args...)
	raw, err := cmd.CombinedOutput()
	return string(raw), err
}
