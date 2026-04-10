package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// GlobalSettings holds application-wide configuration.
type GlobalSettings struct {
	MaxWorkers             int                  `toml:"max_workers"`
	DefaultConflictResolve string               `toml:"default_conflict_resolve"`
	LogLevel               string               `toml:"log_level"`
	Notifications          NotificationSettings `toml:"notifications"`
}

// NotificationSettings configures desktop notifications.
type NotificationSettings struct {
	Enabled   bool `toml:"enabled"`
	OnSuccess bool `toml:"on_success"`
	OnFailure bool `toml:"on_failure"`
}

// Mapping represents a sync pair with per-mapping options.
type Mapping struct {
	Name            string   `toml:"-"` // Populated from TOML map key
	Local           string   `toml:"local"`
	Remote          string   `toml:"remote"`
	FiltersFile     string   `toml:"filters_file"`
	BandwidthLimit  string   `toml:"bandwidth_limit"`
	ExtraFlags      []string `toml:"extra_flags"`
	ConflictResolve string   `toml:"conflict_resolve"`
	BackupEnabled   bool             `toml:"backup_enabled"`
	BackupRetention int              `toml:"backup_retention_days"`
	Encryption      EncryptionConfig `toml:"encryption"`
}

// Config holds the full application configuration.
type Config struct {
	ConfigDir string
	Global    GlobalSettings
	Mappings  []Mapping
}

// tomlFile is the on-disk TOML structure.
type tomlFile struct {
	Global  GlobalSettings     `toml:"global"`
	Mapping map[string]Mapping `toml:"mapping"`
}

// Load reads config from the default XDG config directory.
func Load() (*Config, error) {
	return LoadProfile("")
}

// LoadProfile reads config for the given profile name.
// An empty name loads the default settings.toml.
func LoadProfile(profile string) (*Config, error) {
	return LoadFrom(ProfilePath(profile))
}

// ProfilePath returns the config file path for a given profile.
// An empty name returns the default settings.toml path.
func ProfilePath(profile string) string {
	if profile == "" {
		return filepath.Join(ConfigDir(), "settings.toml")
	}
	return filepath.Join(ConfigDir(), "profiles", profile+".toml")
}

// LoadFrom reads config from a specific TOML file.
func LoadFrom(path string) (*Config, error) {
	var tf tomlFile
	if _, err := toml.DecodeFile(path, &tf); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Apply defaults
	if tf.Global.MaxWorkers <= 0 {
		tf.Global.MaxWorkers = 3
	}
	if tf.Global.DefaultConflictResolve == "" {
		tf.Global.DefaultConflictResolve = "newer"
	}
	if tf.Global.LogLevel == "" {
		tf.Global.LogLevel = "info"
	}

	mappings := make([]Mapping, 0, len(tf.Mapping))
	for name, m := range tf.Mapping {
		m.Name = name
		mappings = append(mappings, m)
	}

	return &Config{
		ConfigDir: ConfigDir(),
		Global:    tf.Global,
		Mappings:  mappings,
	}, nil
}

// IsRemotePath returns true if the path is a remote (contains ':').
func IsRemotePath(p string) bool {
	return strings.Contains(p, ":")
}

// Validate checks config for issues.
func (c *Config) Validate() []error {
	var errs []error
	seen := make(map[string]bool)

	for _, m := range c.Mappings {
		if seen[m.Name] {
			errs = append(errs, fmt.Errorf("duplicate mapping name: %q", m.Name))
		}
		seen[m.Name] = true

		if m.Local == "" {
			errs = append(errs, fmt.Errorf("mapping %q: local path is empty", m.Name))
		} else if !IsRemotePath(m.Local) {
			if _, err := os.Stat(m.Local); os.IsNotExist(err) {
				errs = append(errs, fmt.Errorf("mapping %q: local path does not exist: %s", m.Name, m.Local))
			}
		}

		if m.Remote == "" {
			errs = append(errs, fmt.Errorf("mapping %q: remote path is empty", m.Name))
		} else if !IsRemotePath(m.Remote) {
			errs = append(errs, fmt.Errorf("mapping %q: remote path missing ':' separator: %s", m.Name, m.Remote))
		}

		if m.Encryption.Enabled && m.Encryption.CryptRemote == "" {
			errs = append(errs, fmt.Errorf("mapping %q: encryption enabled but crypt_remote is empty", m.Name))
		}
		if m.Encryption.Enabled && m.Encryption.CryptRemote != "" && !IsRemotePath(m.Encryption.CryptRemote) {
			errs = append(errs, fmt.Errorf("mapping %q: crypt_remote missing ':' separator: %s", m.Name, m.Encryption.CryptRemote))
		}
	}

	return errs
}
