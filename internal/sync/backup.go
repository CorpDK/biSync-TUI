package sync

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/CorpDK/bisync-tui/internal/config"
)

// BackupManager handles backup directory creation and rotation.
type BackupManager struct {
	engine *Engine
}

// NewBackupManager creates a backup manager.
func NewBackupManager(engine *Engine) *BackupManager {
	return &BackupManager{engine: engine}
}

// BackupDir returns the dated backup directory path for a mapping.
func (b *BackupManager) BackupDir(mapping config.Mapping) string {
	date := time.Now().Format("2006-01-02")
	return mapping.Remote + "/.backups/" + date
}

// BuildBackupFlags returns rclone flags for backup if enabled.
func (b *BackupManager) BuildBackupFlags(mapping config.Mapping) []string {
	if !mapping.BackupEnabled {
		return nil
	}
	return []string{"--backup-dir", b.BackupDir(mapping)}
}

// CleanupOldBackups removes backup directories older than the retention period.
func (b *BackupManager) CleanupOldBackups(ctx context.Context, mapping config.Mapping) error {
	if !mapping.BackupEnabled || mapping.BackupRetention <= 0 {
		return nil
	}

	backupRoot := mapping.Remote + "/.backups"
	dirs, err := b.ListBackups(ctx, mapping)
	if err != nil {
		return err
	}

	cutoff := time.Now().AddDate(0, 0, -mapping.BackupRetention)

	for _, dir := range dirs {
		t, err := time.Parse("2006-01-02", dir)
		if err != nil {
			continue
		}
		if t.Before(cutoff) {
			purgeCmd := exec.CommandContext(ctx, b.engine.rclonePath,
				"purge", backupRoot+"/"+dir)
			purgeCmd.Run()
		}
	}
	return nil
}

// ListBackups returns the list of backup date directories.
func (b *BackupManager) ListBackups(ctx context.Context, mapping config.Mapping) ([]string, error) {
	backupRoot := mapping.Remote + "/.backups"
	cmd := exec.CommandContext(ctx, b.engine.rclonePath, "lsd", backupRoot)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("list backups: %w", err)
	}

	var dirs []string
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 {
			dir := fields[len(fields)-1]
			if _, err := time.Parse("2006-01-02", dir); err == nil {
				dirs = append(dirs, dir)
			}
		}
	}
	return dirs, nil
}
