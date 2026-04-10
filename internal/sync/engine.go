package sync

import (
	"bufio"
	"context"
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
	ConflictResolve string
	BandwidthLimit  string
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

// RclonePath returns the path to the rclone binary.
func (e *Engine) RclonePath() string { return e.rclonePath }

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
		"--create-empty-src-dirs", "--resilient", "--recover",
		"--max-lock", "5m", "--verbose",
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
func (e *Engine) RunSync(ctx context.Context, m config.Mapping, opts SyncOptions, outputCh chan<- string) SyncResult {
	start := time.Now()
	args := e.BuildBisyncArgs(m, opts)
	return e.runStreaming(ctx, args, outputCh, start)
}

// RunDiff performs a dry-run bisync and parses the output.
func (e *Engine) RunDiff(ctx context.Context, m config.Mapping, opts SyncOptions) DiffResult {
	opts.DryRun = true
	start := time.Now()
	result := e.RunSync(ctx, m, opts, nil)
	dr := DiffResult{MappingName: m.Name, Duration: time.Since(start)}
	if !result.Success {
		dr.Error = result.ErrorMsg
	}
	dr.Entries = ParseDiffOutput(result.Output)
	return dr
}

// RunCopy executes rclone copy (used for first-run bootstrap).
func (e *Engine) RunCopy(ctx context.Context, src, dst string, outputCh chan<- string) SyncResult {
	start := time.Now()
	args := []string{"copy", src, dst, "--verbose"}
	return e.runStreaming(ctx, args, outputCh, start)
}

// runStreaming runs an rclone command, streaming stdout/stderr to outputCh.
func (e *Engine) runStreaming(ctx context.Context, args []string, outputCh chan<- string, start time.Time) SyncResult {
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
	result := SyncResult{
		Duration: time.Since(start),
		Output:   output.String(),
		Success:  err == nil,
	}
	if err != nil {
		result.ErrorMsg = err.Error()
	}
	return result
}
