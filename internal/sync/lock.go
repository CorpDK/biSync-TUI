package sync

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
)

// LockManager handles per-mapping file locks.
type LockManager struct {
	lockDir string
}

// NewLockManager creates a LockManager backed by the given directory.
func NewLockManager(lockDir string) *LockManager {
	return &LockManager{lockDir: lockDir}
}

// Acquire attempts a non-blocking lock for the named mapping.
// Returns the lock handle on success, or an error if already locked.
func (lm *LockManager) Acquire(name string) (*flock.Flock, error) {
	if err := os.MkdirAll(lm.lockDir, 0o755); err != nil {
		return nil, fmt.Errorf("create lock dir: %w", err)
	}

	lockPath := filepath.Join(lm.lockDir, name+".lock")
	fl := flock.New(lockPath)

	locked, err := fl.TryLock()
	if err != nil {
		return nil, fmt.Errorf("acquire lock for %q: %w", name, err)
	}
	if !locked {
		return nil, fmt.Errorf("mapping %q is already locked by another process", name)
	}

	return fl, nil
}

// Release unlocks and cleans up the lock file.
func (lm *LockManager) Release(fl *flock.Flock) error {
	if fl == nil {
		return nil
	}
	if err := fl.Unlock(); err != nil {
		return fmt.Errorf("release lock: %w", err)
	}
	// Clean up the lock file
	os.Remove(fl.Path())
	return nil
}

// IsLocked checks if a mapping is currently locked.
func (lm *LockManager) IsLocked(name string) bool {
	lockPath := filepath.Join(lm.lockDir, name+".lock")
	fl := flock.New(lockPath)

	locked, err := fl.TryLock()
	if err != nil || !locked {
		return true
	}
	// We got the lock, so it wasn't locked — release immediately
	fl.Unlock()
	os.Remove(lockPath)
	return false
}
