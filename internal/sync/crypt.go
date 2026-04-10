package sync

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// ListRemotes returns all configured rclone remotes (e.g., ["gdrive:", "mycrypt:"]).
func (e *Engine) ListRemotes(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, e.rclonePath, "listremotes")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("rclone listremotes: %w", err)
	}

	var remotes []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			remotes = append(remotes, line)
		}
	}
	return remotes, nil
}

// ValidateCryptRemote verifies that a remote exists and is of type "crypt".
func (e *Engine) ValidateCryptRemote(ctx context.Context, cryptRemote string) error {
	// Strip trailing colon and path for config lookup
	remoteName := strings.TrimRight(strings.SplitN(cryptRemote, ":", 2)[0], " ")

	isCrypt, err := e.IsCryptRemote(ctx, remoteName)
	if err != nil {
		return err
	}
	if !isCrypt {
		return fmt.Errorf("remote %q is not a crypt remote", remoteName)
	}
	return nil
}

// IsCryptRemote checks if a specific remote is configured as type=crypt.
func (e *Engine) IsCryptRemote(ctx context.Context, remoteName string) (bool, error) {
	cmd := exec.CommandContext(ctx, e.rclonePath, "config", "show", remoteName)
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("rclone config show %s: %w", remoteName, err)
	}

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "type") && strings.Contains(line, "crypt") {
			return true, nil
		}
	}
	return false, nil
}
