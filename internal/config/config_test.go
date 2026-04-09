package config

import (
	"os"
	"path/filepath"
	"testing"
)

const testTOML = `
[global]
max_workers = 5
default_conflict_resolve = "newer"
log_level = "info"

[global.notifications]
enabled = true
on_success = false
on_failure = true

[mapping.exploratory]
local = "/tmp/test-local"
remote = "gdrive:test-remote"
filters_file = "filters/exploratory.txt"

[mapping.notes]
local = "/tmp/notes"
remote = "gdrive:notes"
bandwidth_limit = "10M"
backup_enabled = true
backup_retention_days = 7
`

func TestLoadFrom(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.toml")
	if err := os.WriteFile(path, []byte(testTOML), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}

	if cfg.Global.MaxWorkers != 5 {
		t.Errorf("expected MaxWorkers=5, got %d", cfg.Global.MaxWorkers)
	}
	if !cfg.Global.Notifications.Enabled {
		t.Error("expected Notifications.Enabled=true")
	}
	if !cfg.Global.Notifications.OnFailure {
		t.Error("expected Notifications.OnFailure=true")
	}

	if len(cfg.Mappings) != 2 {
		t.Fatalf("expected 2 mappings, got %d", len(cfg.Mappings))
	}

	// Find by name since TOML map order is not guaranteed
	byName := make(map[string]Mapping)
	for _, m := range cfg.Mappings {
		byName[m.Name] = m
	}

	exp := byName["exploratory"]
	if exp.Local != "/tmp/test-local" || exp.Remote != "gdrive:test-remote" {
		t.Errorf("unexpected exploratory mapping: %+v", exp)
	}
	if exp.FiltersFile != "filters/exploratory.txt" {
		t.Errorf("expected FiltersFile, got %q", exp.FiltersFile)
	}

	notes := byName["notes"]
	if notes.BandwidthLimit != "10M" {
		t.Errorf("expected BandwidthLimit=10M, got %q", notes.BandwidthLimit)
	}
	if !notes.BackupEnabled || notes.BackupRetention != 7 {
		t.Errorf("unexpected backup config: enabled=%v retention=%d", notes.BackupEnabled, notes.BackupRetention)
	}
}

func TestLoadFromDefaults(t *testing.T) {
	content := `
[mapping.test]
local = "/tmp/test"
remote = "gdrive:test"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.toml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}

	if cfg.Global.MaxWorkers != 3 {
		t.Errorf("expected default MaxWorkers=3, got %d", cfg.Global.MaxWorkers)
	}
	if cfg.Global.DefaultConflictResolve != "newer" {
		t.Errorf("expected default conflict=newer, got %q", cfg.Global.DefaultConflictResolve)
	}
}

func TestValidate(t *testing.T) {
	dir := t.TempDir()

	cfg := &Config{
		Mappings: []Mapping{
			{Name: "ok", Local: dir, Remote: "gdrive:test"},
			{Name: "bad-local", Local: "/nonexistent/path", Remote: "gdrive:test"},
			{Name: "bad-remote", Local: dir, Remote: "no-colon"},
			{Name: "ok", Local: dir, Remote: "gdrive:dup"},                          // duplicate
			{Name: "remote-remote", Local: "s3:bucket/path", Remote: "gdrive:path"}, // valid multi-remote
		},
	}

	errs := cfg.Validate()
	if len(errs) != 3 {
		t.Errorf("expected 3 validation errors, got %d: %v", len(errs), errs)
	}
}

func TestIsRemotePath(t *testing.T) {
	if !IsRemotePath("gdrive:test") {
		t.Error("expected gdrive:test to be remote")
	}
	if !IsRemotePath("s3:bucket/path") {
		t.Error("expected s3:bucket/path to be remote")
	}
	if IsRemotePath("/home/user/local") {
		t.Error("expected /home/user/local to not be remote")
	}
}
