package p4

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var p4ChangeCreatedRE = regexp.MustCompile(`Change\s+(\d+)\s+created`)

// Client is the minimal p4 command surface used by pcraft.
type Client interface {
	ListClients(ctx context.Context, p4user string) ([]string, error)
	CreateChangelist(ctx context.Context, description string) (string, error)
	CheckoutFiles(ctx context.Context, changelist string, files []string) error
	RevertChangelist(ctx context.Context, changelist string) error
	OpenedFiles(ctx context.Context, changelist string) ([]string, error)
	IsSubmitted(ctx context.Context, changelist string) (bool, error)
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

func (c *CLIClient) CreateChangelist(ctx context.Context, description string) (string, error) {
	desc := strings.TrimSpace(description)
	if desc == "" {
		desc = "pcraft task"
	}
	spec := fmt.Sprintf("Change: new\n\nDescription:\n\t%s\n\n", desc)
	cmd := exec.CommandContext(ctx, "p4", "change", "-i")
	cmd.Stdin = strings.NewReader(spec)
	raw, err := cmd.CombinedOutput()
	out := string(raw)
	if err != nil {
		return "", fmt.Errorf("p4 change -i: %w: %s", err, strings.TrimSpace(out))
	}
	if match := p4ChangeCreatedRE.FindStringSubmatch(out); len(match) == 2 {
		return match[1], nil
	}
	return "", fmt.Errorf("p4 change -i: unexpected output: %s", strings.TrimSpace(out))
}

func (c *CLIClient) CheckoutFiles(ctx context.Context, changelist string, files []string) error {
	args := []string{"edit"}
	if strings.TrimSpace(changelist) != "" {
		args = append(args, "-c", changelist)
	}
	args = append(args, files...)
	_, err := runP4(ctx, args...)
	return err
}

func (c *CLIClient) RevertChangelist(ctx context.Context, changelist string) error {
	_, err := runP4(ctx, "revert", "-c", changelist, "//...")
	return err
}

func (c *CLIClient) OpenedFiles(ctx context.Context, changelist string) ([]string, error) {
	args := []string{"opened"}
	if strings.TrimSpace(changelist) != "" {
		args = append(args, "-c", changelist)
	}
	out, err := runP4(ctx, args...)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(out, "\n")
	files := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) > 0 {
			files = append(files, fields[0])
		}
	}
	return files, nil
}

func (c *CLIClient) IsSubmitted(ctx context.Context, changelist string) (bool, error) {
	out, err := runP4(ctx, "describe", "-s", changelist)
	if err != nil {
		return false, err
	}
	return strings.Contains(out, "submitted"), nil
}

func runP4(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "p4", args...)
	raw, err := cmd.CombinedOutput()
	return string(raw), err
}
