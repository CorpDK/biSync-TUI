package logs

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// LogEntry represents a single log line.
type LogEntry struct {
	Timestamp   time.Time
	MappingName string
	Level       string
	Message     string
}

// LogManager handles per-mapping log files.
type LogManager struct {
	logDir string
}

// NewLogManager creates a log manager.
func NewLogManager(logDir string) *LogManager {
	return &LogManager{logDir: logDir}
}

// Write appends a timestamped line to the mapping's log file.
func (l *LogManager) Write(mappingName, line string) error {
	if err := os.MkdirAll(l.logDir, 0o755); err != nil {
		return err
	}

	f, err := os.OpenFile(l.path(mappingName), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	level := classifyLevel(line)
	entry := fmt.Sprintf("%s [%s] %s\n", time.Now().Format(time.RFC3339), level, line)
	_, err = f.WriteString(entry)
	return err
}

// Read returns the most recent `limit` entries for a mapping.
func (l *LogManager) Read(mappingName string, limit int) ([]LogEntry, error) {
	return l.readFile(mappingName, l.path(mappingName), limit)
}

// ReadAll reads from all mapping logs, merges by timestamp, returns most recent `limit`.
func (l *LogManager) ReadAll(limit int) ([]LogEntry, error) {
	entries, err := os.ReadDir(l.logDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var all []LogEntry
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".log") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".log")
		records, err := l.readFile(name, filepath.Join(l.logDir, e.Name()), 0)
		if err != nil {
			continue
		}
		all = append(all, records...)
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].Timestamp.Before(all[j].Timestamp)
	})

	if limit > 0 && len(all) > limit {
		all = all[len(all)-limit:]
	}
	return all, nil
}

// Search returns entries matching a query string across all logs.
func (l *LogManager) Search(query string, limit int) ([]LogEntry, error) {
	all, err := l.ReadAll(0)
	if err != nil {
		return nil, err
	}

	var matches []LogEntry
	q := strings.ToLower(query)
	for _, e := range all {
		if strings.Contains(strings.ToLower(e.Message), q) ||
			strings.Contains(strings.ToLower(e.MappingName), q) {
			matches = append(matches, e)
		}
	}

	if limit > 0 && len(matches) > limit {
		matches = matches[len(matches)-limit:]
	}
	return matches, nil
}

// Export writes filtered entries to a writer.
func (l *LogManager) Export(w io.Writer, filter func(LogEntry) bool) error {
	all, err := l.ReadAll(0)
	if err != nil {
		return err
	}
	for _, e := range all {
		if filter == nil || filter(e) {
			fmt.Fprintf(w, "%s [%s] [%s] %s\n",
				e.Timestamp.Format(time.RFC3339), e.MappingName, e.Level, e.Message)
		}
	}
	return nil
}

func (l *LogManager) readFile(mappingName, path string, limit int) ([]LogEntry, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []LogEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		entry := parseLine(mappingName, line)
		entries = append(entries, entry)
	}

	if limit > 0 && len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}
	return entries, scanner.Err()
}

func parseLine(mappingName, line string) LogEntry {
	entry := LogEntry{MappingName: mappingName, Message: line, Timestamp: time.Now()}

	// Parse format: "2006-01-02T15:04:05Z07:00 [LEVEL] message"
	if len(line) > 25 && line[25] == '[' {
		if t, err := time.Parse(time.RFC3339, line[:25]); err == nil {
			entry.Timestamp = t
			rest := line[26:]
			if idx := strings.Index(rest, "] "); idx >= 0 {
				entry.Level = rest[:idx]
				entry.Message = rest[idx+2:]
			}
		}
	}
	return entry
}

func classifyLevel(line string) string {
	lower := strings.ToLower(line)
	switch {
	case strings.Contains(lower, "error"):
		return "ERROR"
	case strings.Contains(lower, "notice"):
		return "NOTICE"
	case strings.Contains(lower, "warning"):
		return "WARN"
	default:
		return "INFO"
	}
}

func (l *LogManager) path(mappingName string) string {
	return filepath.Join(l.logDir, mappingName+".log")
}
