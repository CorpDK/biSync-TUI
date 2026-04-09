package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/CorpDK/bisync-tui/internal/config"
)

// SyncStatus represents the current state of a mapping.
type SyncStatus string

const (
	StatusIdle      SyncStatus = "idle"
	StatusSyncing   SyncStatus = "syncing"
	StatusError     SyncStatus = "error"
	StatusNeedsInit SyncStatus = "needs-init"
)

// MappingState tracks the persistent state of a single mapping.
type MappingState struct {
	Name         string     `json:"name"`
	Initialized  bool       `json:"initialized"`
	LastSync     *time.Time `json:"last_sync,omitempty"`
	LastStatus   SyncStatus `json:"last_status"`
	LastError    string     `json:"last_error,omitempty"`
	SyncCount    int        `json:"sync_count"`
	LastDuration string     `json:"last_duration,omitempty"`
}

// Store manages persisted state for all mappings.
type Store struct {
	stateDir string
}

// NewStore creates a Store backed by the given directory.
func NewStore(stateDir string) *Store {
	return &Store{stateDir: stateDir}
}

// NewDefaultStore creates a Store using XDG state directory.
func NewDefaultStore() *Store {
	return NewStore(filepath.Join(config.StateDir(), "state"))
}

// Load reads the state for a single mapping.
func (s *Store) Load(name string) (*MappingState, error) {
	path := s.path(name)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &MappingState{
			Name:       name,
			LastStatus: StatusNeedsInit,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read state %q: %w", name, err)
	}

	var ms MappingState
	if err := json.Unmarshal(data, &ms); err != nil {
		return nil, fmt.Errorf("parse state %q: %w", name, err)
	}
	return &ms, nil
}

// Save persists the state for a single mapping.
func (s *Store) Save(name string, ms *MappingState) error {
	if err := os.MkdirAll(s.stateDir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(ms, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state %q: %w", name, err)
	}
	return os.WriteFile(s.path(name), data, 0o644)
}

// LoadAll loads state for all given mappings.
func (s *Store) LoadAll(mappings []config.Mapping) map[string]*MappingState {
	states := make(map[string]*MappingState, len(mappings))
	for _, m := range mappings {
		ms, err := s.Load(m.Name)
		if err != nil {
			ms = &MappingState{
				Name:       m.Name,
				LastStatus: StatusError,
				LastError:  err.Error(),
			}
		}
		states[m.Name] = ms
	}
	return states
}

func (s *Store) path(name string) string {
	return filepath.Join(s.stateDir, name+".json")
}
