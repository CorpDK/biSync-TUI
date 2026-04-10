package sync

import (
	"regexp"
	"strings"
)

// Regexes for parsing rclone bisync --dry-run verbose output.
var (
	// Matches lines like: NOTICE: file.txt - Path1 File is new
	newFileRegex = regexp.MustCompile(`(?i)NOTICE:\s*(.*?)\s*-\s*(Path[12])\s+File is new`)

	// Matches lines like: NOTICE: file.txt - Path1 File not found
	notFoundRegex = regexp.MustCompile(`(?i)NOTICE:\s*(.*?)\s*-\s*(Path[12])\s+File not found`)

	// Matches lines like: NOTICE: file.txt - files differ
	differRegex = regexp.MustCompile(`(?i)NOTICE:\s*(.*?)\s*-\s*(?:files differ|sizes differ|timestamps differ)`)

	// Matches copy/delete action lines from dry-run:
	// NOTICE: - Path1    copy  file.txt
	// NOTICE: - Path2    delete  file.txt
	actionCopyRegex   = regexp.MustCompile(`(?i)NOTICE:\s*-\s*(Path[12])\s+copy\s+(.+)`)
	actionDeleteRegex = regexp.MustCompile(`(?i)NOTICE:\s*-\s*(Path[12])\s+delete\s+(.+)`)
)

// ParseDiffOutput parses rclone bisync --dry-run verbose output into DiffEntries.
func ParseDiffOutput(output string) []DiffEntry {
	seen := make(map[string]bool)
	var entries []DiffEntry

	addEntry := func(path string, typ DiffEntryType, side DiffSide) {
		key := path + "|" + string(typ) + "|" + string(side)
		if seen[key] {
			return
		}
		seen[key] = true
		entries = append(entries, DiffEntry{
			Path: path,
			Type: typ,
			Side: side,
		})
	}

	for _, line := range strings.Split(output, "\n") {
		clean := StripANSI(line)

		// New file on a side
		if m := newFileRegex.FindStringSubmatch(clean); len(m) > 2 {
			path := strings.TrimSpace(m[1])
			side := parseSide(m[2])
			if path != "" {
				addEntry(path, DiffAdded, side)
			}
			continue
		}

		// File not found on a side (deleted)
		if m := notFoundRegex.FindStringSubmatch(clean); len(m) > 2 {
			path := strings.TrimSpace(m[1])
			side := parseSide(m[2])
			if path != "" {
				// "not found" on Path1 means it was deleted locally
				addEntry(path, DiffDeleted, side)
			}
			continue
		}

		// Files differ
		if m := differRegex.FindStringSubmatch(clean); len(m) > 1 {
			path := strings.TrimSpace(m[1])
			if path != "" {
				addEntry(path, DiffModified, DiffSideLocal)
			}
			continue
		}

		// Action: copy
		if m := actionCopyRegex.FindStringSubmatch(clean); len(m) > 2 {
			path := strings.TrimSpace(m[2])
			side := parseSide(m[1])
			if path != "" {
				addEntry(path, DiffAdded, side)
			}
			continue
		}

		// Action: delete
		if m := actionDeleteRegex.FindStringSubmatch(clean); len(m) > 2 {
			path := strings.TrimSpace(m[2])
			side := parseSide(m[1])
			if path != "" {
				addEntry(path, DiffDeleted, side)
			}
			continue
		}
	}

	return entries
}

func parseSide(pathLabel string) DiffSide {
	if strings.Contains(strings.ToLower(pathLabel), "path1") {
		return DiffSideLocal
	}
	return DiffSideRemote
}
