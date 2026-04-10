package sync

import "time"

// DiffEntryType classifies a change detected during a dry-run diff.
type DiffEntryType string

const (
	DiffAdded    DiffEntryType = "added"
	DiffDeleted  DiffEntryType = "deleted"
	DiffModified DiffEntryType = "modified"
)

// DiffSide indicates which side of the sync pair a change originates from.
type DiffSide string

const (
	DiffSideLocal  DiffSide = "local"
	DiffSideRemote DiffSide = "remote"
)

// DiffEntry represents a single file change detected by a dry-run bisync.
type DiffEntry struct {
	Path string
	Type DiffEntryType
	Side DiffSide
}

// DiffResult holds the outcome of a diff operation for a single mapping.
type DiffResult struct {
	MappingName string
	Entries     []DiffEntry
	Error       string
	Duration    time.Duration
}
