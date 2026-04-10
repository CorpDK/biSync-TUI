package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// SaveConfig writes a full Config to the given TOML file path.
func SaveConfig(path string, cfg *Config) error {
	tf := tomlFile{
		Global:  cfg.Global,
		Mapping: make(map[string]Mapping, len(cfg.Mappings)),
	}
	for _, m := range cfg.Mappings {
		tf.Mapping[m.Name] = m
	}
	return writeToml(path, tf)
}

// AddMapping appends a mapping to an existing config file.
// If the file does not exist it is created with defaults.
func AddMapping(path string, m Mapping) error {
	cfg, err := loadOrDefault(path)
	if err != nil {
		return err
	}

	// Check for duplicates
	for _, existing := range cfg.Mappings {
		if existing.Name == m.Name {
			return fmt.Errorf("mapping %q already exists", m.Name)
		}
	}

	cfg.Mappings = append(cfg.Mappings, m)
	return SaveConfig(path, cfg)
}

// UpdateMapping replaces a mapping in the config file by name.
func UpdateMapping(path string, m Mapping) error {
	cfg, err := loadOrDefault(path)
	if err != nil {
		return err
	}

	found := false
	for i, existing := range cfg.Mappings {
		if existing.Name == m.Name {
			cfg.Mappings[i] = m
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("mapping %q not found", m.Name)
	}

	return SaveConfig(path, cfg)
}

// RemoveMapping removes a mapping from a config file by name.
func RemoveMapping(path string, name string) error {
	cfg, err := loadOrDefault(path)
	if err != nil {
		return err
	}

	filtered := cfg.Mappings[:0]
	for _, m := range cfg.Mappings {
		if m.Name != name {
			filtered = append(filtered, m)
		}
	}

	if len(filtered) == len(cfg.Mappings) {
		return fmt.Errorf("mapping %q not found", name)
	}

	cfg.Mappings = filtered
	return SaveConfig(path, cfg)
}

// CreateDefaultConfig writes a new config file with sensible defaults.
func CreateDefaultConfig(path string) error {
	cfg := &Config{
		Global: GlobalSettings{
			MaxWorkers:             3,
			DefaultConflictResolve: "newer",
			LogLevel:               "info",
			Notifications: NotificationSettings{
				Enabled:   true,
				OnSuccess: false,
				OnFailure: true,
			},
		},
	}
	return SaveConfig(path, cfg)
}

func loadOrDefault(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &Config{
			Global: GlobalSettings{
				MaxWorkers:             3,
				DefaultConflictResolve: "newer",
				LogLevel:               "info",
			},
		}, nil
	}
	return LoadFrom(path)
}

func writeToml(path string, v interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("encode config: %w", err)
	}

	return os.WriteFile(path, buf.Bytes(), 0o644)
}
