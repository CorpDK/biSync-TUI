package sync

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/CorpDK/bisync-tui/internal/config"
)

// SyncOptions configures a sync operation.
type SyncOptions struct {
	DryRun          bool
	Resync          bool
	Verbose         bool
	ConflictResolve string // "newer", "older", "path1", "path2"
	BandwidthLimit  string // e.g. "10M"
	FiltersFile     string
	ExtraFlags      []string
}

// SyncResult holds the outcome of a sync operation.
type SyncResult struct {
	Success  bool
	Duration time.Duration
	Output   string
	ErrorMsg string
}

// Engine wraps rclone command execution.
type Engine struct {
	rclonePath string
}

// NewEngine creates a new Engine, locating the rclone binary.
func NewEngine() (*Engine, error) {
	path, err := exec.LookPath("rclone")
	if err != nil {
		return nil, fmt.Errorf("rclone not found in PATH: %w", err)
	}
	return &Engine{rclonePath: path}, nil
}

// BuildBisyncArgs constructs the argument list for a bisync command.
func (e *Engine) BuildBisyncArgs(m config.Mapping, opts SyncOptions) []string {
	args := []string{"bisync", m.Local, m.Remote}

	if opts.Resync {
		args = append(args, "--resync")
	}
	if opts.DryRun {
		args = append(args, "--dry-run")
	}

	args = append(args,
		"--create-empty-src-dirs",
		"--resilient",
		"--recover",
		"--max-lock", "5m",
		"--verbose",
	)

	resolve := opts.ConflictResolve
	if resolve == "" {
		resolve = "newer"
	}
	args = append(args, "--conflict-resolve", resolve)

	if opts.BandwidthLimit != "" {
		args = append(args, "--bwlimit", opts.BandwidthLimit)
	}
	if opts.FiltersFile != "" {
		args = append(args, "--filters-file", opts.FiltersFile)
	}
	args = append(args, opts.ExtraFlags...)

	return args
}

// RunSync executes a bisync operation, streaming output to outputCh.
// Returns when the sync completes or the context is cancelled.
func (e *Engine) RunSync(ctx context.Context, m config.Mapping, opts SyncOptions, outputCh chan<- string) SyncResult {
	start := time.Now()
	args := e.BuildBisyncArgs(m, opts)

	cmd := exec.CommandContext(ctx, e.rclonePath, args...)
	var output strings.Builder

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return SyncResult{ErrorMsg: fmt.Sprintf("stdout pipe: %v", err)}
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return SyncResult{ErrorMsg: fmt.Sprintf("stderr pipe: %v", err)}
	}

	if err := cmd.Start(); err != nil {
		return SyncResult{ErrorMsg: fmt.Sprintf("start: %v", err)}
	}

	// Stream both stdout and stderr
	done := make(chan struct{}, 2)
	streamPipe := func(r io.Reader) {
		defer func() { done <- struct{}{} }()
		scanner := bufio.NewScanner(r)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			output.WriteString(line + "\n")
			if outputCh != nil {
				select {
				case outputCh <- line:
				case <-ctx.Done():
					return
				}
			}
		}
	}

	go streamPipe(stdout)
	go streamPipe(stderr)

	// Wait for both streams to close
	<-done
	<-done

	err = cmd.Wait()
	duration := time.Since(start)

	result := SyncResult{
		Duration: duration,
		Output:   output.String(),
		Success:  err == nil,
	}
	if err != nil {
		result.ErrorMsg = err.Error()
	}
	return result
}

// RunDiff performs a dry-run bisync and parses the output into structured diff entries.
func (e *Engine) RunDiff(ctx context.Context, m config.Mapping, opts SyncOptions) DiffResult {
	opts.DryRun = true
	start := time.Now()
	result := e.RunSync(ctx, m, opts, nil)
	dr := DiffResult{
		MappingName: m.Name,
		Duration:    time.Since(start),
	}
	if !result.Success {
		dr.Error = result.ErrorMsg
	}
	dr.Entries = ParseDiffOutput(result.Output)
	return dr
}

// RunCopy executes rclone copy (used for first-run bootstrap: remote → local).
func (e *Engine) RunCopy(ctx context.Context, src, dst string, outputCh chan<- string) SyncResult {
	start := time.Now()
	args := []string{"copy", src, dst, "--verbose"}

	cmd := exec.CommandContext(ctx, e.rclonePath, args...)
	var output strings.Builder

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return SyncResult{ErrorMsg: fmt.Sprintf("stdout pipe: %v", err)}
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return SyncResult{ErrorMsg: fmt.Sprintf("stderr pipe: %v", err)}
	}

	if err := cmd.Start(); err != nil {
		return SyncResult{ErrorMsg: fmt.Sprintf("start: %v", err)}
	}

	done := make(chan struct{}, 2)
	streamPipe := func(r io.Reader) {
		defer func() { done <- struct{}{} }()
		scanner := bufio.NewScanner(r)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			output.WriteString(line + "\n")
			if outputCh != nil {
				select {
				case outputCh <- line:
				case <-ctx.Done():
					return
				}
			}
		}
	}

	go streamPipe(stdout)
	go streamPipe(stderr)
	<-done
	<-done

	err = cmd.Wait()
	duration := time.Since(start)

	result := SyncResult{
		Duration: duration,
		Output:   output.String(),
		Success:  err == nil,
	}
	if err != nil {
		result.ErrorMsg = err.Error()
	}
	return result
}

// GetRemoteSize returns the size of a remote path as a formatted string.
func (e *Engine) GetRemoteSize(ctx context.Context, remotePath string) (string, error) {
	cmd := exec.CommandContext(ctx, e.rclonePath, "size", remotePath, "--json")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("rclone size: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
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

// CheckConnectivity verifies that a remote is reachable.
func (e *Engine) CheckConnectivity(ctx context.Context, remotePath string) error {
	// Extract remote name (everything before the first colon + path)
	parts := strings.SplitN(remotePath, ":", 2)
	if len(parts) < 2 {
		return fmt.Errorf("invalid remote path: %s", remotePath)
	}
	remote := parts[0] + ":"

	cmd := exec.CommandContext(ctx, e.rclonePath, "lsd", remote, "--max-depth", "0")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("remote %s unreachable: %w", remote, err)
	}
	return nil
}
