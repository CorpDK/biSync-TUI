package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// RemoteInfo holds parsed config details for a single rclone remote.
type RemoteInfo struct {
	Name    string
	Type    string
	Details map[string]string
}

// RemoteAbout holds storage space info for a remote.
type RemoteAbout struct {
	Remote  string `json:"-"`
	Total   int64  `json:"total"`
	Used    int64  `json:"used"`
	Free    int64  `json:"free"`
	Trashed int64  `json:"trashed"`
	Other   int64  `json:"other"`
}

// ConfigDump returns all configured remotes with their settings.
func (e *Engine) ConfigDump(ctx context.Context) ([]RemoteInfo, error) {
	cmd := exec.CommandContext(ctx, e.rclonePath, "config", "dump")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("rclone config dump: %w", err)
	}
	var raw map[string]map[string]string
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("parse config dump: %w", err)
	}
	remotes := make([]RemoteInfo, 0, len(raw))
	for name, cfg := range raw {
		remotes = append(remotes, RemoteInfo{Name: name, Type: cfg["type"], Details: cfg})
	}
	return remotes, nil
}

// CreateRemote creates a new rclone remote non-interactively.
func (e *Engine) CreateRemote(ctx context.Context, name, remoteType string, params map[string]string) error {
	args := []string{"config", "create", name, remoteType}
	for k, v := range params {
		args = append(args, k, v)
	}
	cmd := exec.CommandContext(ctx, e.rclonePath, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("rclone config create: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// DeleteRemote removes a configured rclone remote.
func (e *Engine) DeleteRemote(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, e.rclonePath, "config", "delete", name)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("rclone config delete: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// CheckConnectivity verifies that a remote is reachable using a quick about query.
func (e *Engine) CheckConnectivity(ctx context.Context, remotePath string) error {
	parts := strings.SplitN(remotePath, ":", 2)
	if len(parts) < 2 {
		return fmt.Errorf("invalid remote path: %s", remotePath)
	}
	remote := parts[0] + ":"

	// Use a 10s timeout so the TUI doesn't hang
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// rclone about is faster than lsd — just queries storage quota, no directory listing
	cmd := exec.CommandContext(timeoutCtx, e.rclonePath, "about", remote, "--json")
	if err := cmd.Run(); err != nil {
		if timeoutCtx.Err() != nil {
			return fmt.Errorf("remote %s timed out after 10s", remote)
		}
		return fmt.Errorf("remote %s unreachable: %w", remote, err)
	}
	return nil
}

// GetRemoteSize returns the size of a remote path as JSON.
func (e *Engine) GetRemoteSize(ctx context.Context, remotePath string) (string, error) {
	cmd := exec.CommandContext(ctx, e.rclonePath, "size", remotePath, "--json")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("rclone size: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// GetRemoteAbout returns storage space info for a remote.
func (e *Engine) GetRemoteAbout(ctx context.Context, remote string) (*RemoteAbout, error) {
	remoteName := remote
	if !strings.HasSuffix(remoteName, ":") {
		parts := strings.SplitN(remoteName, ":", 2)
		if len(parts) >= 1 {
			remoteName = parts[0] + ":"
		}
	}
	cmd := exec.CommandContext(ctx, e.rclonePath, "about", remoteName, "--json")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("rclone about: %w", err)
	}
	var about RemoteAbout
	if err := json.Unmarshal(out, &about); err != nil {
		return nil, fmt.Errorf("parse about: %w", err)
	}
	about.Remote = remote
	return &about, nil
}
