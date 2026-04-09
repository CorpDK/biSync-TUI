package state

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// HistoryRecord stores one sync event.
type HistoryRecord struct {
	Timestamp        time.Time     `json:"timestamp"`
	Duration         time.Duration `json:"duration"`
	Status           string        `json:"status"` // "success" or "error"
	FilesTransferred int           `json:"files_transferred"`
	BytesTransferred int64         `json:"bytes_transferred"`
	Error            string        `json:"error,omitempty"`
}

// HistoryStore manages JSONL-based sync history per mapping.
type HistoryStore struct {
	historyDir string
	maxEntries int
}

// NewHistoryStore creates a history store.
func NewHistoryStore(historyDir string, maxEntries int) *HistoryStore {
	return &HistoryStore{historyDir: historyDir, maxEntries: maxEntries}
}

// Append adds a record and trims old entries if needed.
func (h *HistoryStore) Append(name string, record HistoryRecord) error {
	if err := os.MkdirAll(h.historyDir, 0o755); err != nil {
		return err
	}

	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal history: %w", err)
	}

	f, err := os.OpenFile(h.path(name), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open history: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write history: %w", err)
	}

	return h.trim(name)
}

// Load reads the most recent `limit` records for a mapping.
func (h *HistoryStore) Load(name string, limit int) ([]HistoryRecord, error) {
	f, err := os.Open(h.path(name))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var all []HistoryRecord
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var r HistoryRecord
		if err := json.Unmarshal(scanner.Bytes(), &r); err != nil {
			continue
		}
		all = append(all, r)
	}

	if limit > 0 && len(all) > limit {
		all = all[len(all)-limit:]
	}
	return all, scanner.Err()
}

func (h *HistoryStore) trim(name string) error {
	records, err := h.Load(name, 0)
	if err != nil || len(records) <= h.maxEntries {
		return err
	}

	records = records[len(records)-h.maxEntries:]
	f, err := os.Create(h.path(name))
	if err != nil {
		return err
	}
	defer f.Close()

	for _, r := range records {
		data, _ := json.Marshal(r)
		f.Write(append(data, '\n'))
	}
	return nil
}

func (h *HistoryStore) path(name string) string {
	return filepath.Join(h.historyDir, name+".jsonl")
}
