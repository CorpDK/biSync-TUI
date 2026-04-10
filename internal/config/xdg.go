package config

import (
	"os"
	"path/filepath"
)

const appName = "rclone-bisync"

// ConfigDir returns the XDG config directory for the app.
func ConfigDir() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, appName)
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", appName)
}

// StateDir returns the XDG state directory for the app.
func StateDir() string {
	if dir := os.Getenv("XDG_STATE_HOME"); dir != "" {
		return filepath.Join(dir, appName)
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "state", appName)
}

// CacheDir returns the XDG cache directory for the app.
func CacheDir() string {
	if dir := os.Getenv("XDG_CACHE_HOME"); dir != "" {
		return filepath.Join(dir, appName)
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", appName)
}

// EnsureDirs creates all required directories.
func EnsureDirs() error {
	dirs := []string{
		ConfigDir(),
		filepath.Join(ConfigDir(), "filters"),
		filepath.Join(ConfigDir(), "profiles"),
		filepath.Join(StateDir(), "state"),
		filepath.Join(StateDir(), "logs"),
		filepath.Join(StateDir(), "history"),
		filepath.Join(CacheDir(), "locks"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	return nil
}
