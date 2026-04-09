package state

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStoreLoadSave(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "state"))

	// Load non-existent returns needs-init
	ms, err := store.Load("test")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if ms.LastStatus != StatusNeedsInit {
		t.Errorf("expected needs-init, got %s", ms.LastStatus)
	}
	if ms.Initialized {
		t.Error("expected Initialized=false")
	}

	// Save and reload
	now := time.Now().Truncate(time.Second)
	ms.Initialized = true
	ms.LastSync = &now
	ms.LastStatus = StatusIdle
	ms.SyncCount = 5
	ms.LastDuration = "12s"

	if err := store.Save("test", ms); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load("test")
	if err != nil {
		t.Fatalf("Load after save: %v", err)
	}

	if !loaded.Initialized {
		t.Error("expected Initialized=true")
	}
	if loaded.LastStatus != StatusIdle {
		t.Errorf("expected idle, got %s", loaded.LastStatus)
	}
	if loaded.SyncCount != 5 {
		t.Errorf("expected SyncCount=5, got %d", loaded.SyncCount)
	}
	if loaded.LastSync == nil || loaded.LastSync.Unix() != now.Unix() {
		t.Errorf("LastSync mismatch: %v vs %v", loaded.LastSync, now)
	}
}
