package sync

import (
	"regexp"
	"strings"
)

// Conflict represents a file that differs between the two sync paths.
type Conflict struct {
	Path       string
	Path1Info  string
	Path2Info  string
	Resolved   bool
	Resolution string // "newer", "older", "path1", "path2", "skip"
}

var conflictRegex = regexp.MustCompile(`(?i)NOTICE:.*?:\s*(.*?)\s*(?:files differ|is new|not found)`)

// ParseConflicts extracts conflict entries from rclone bisync output.
func ParseConflicts(output string) []Conflict {
	seen := make(map[string]bool)
	var conflicts []Conflict

	for _, line := range strings.Split(output, "\n") {
		clean := StripANSI(line)

		if m := conflictRegex.FindStringSubmatch(clean); len(m) > 1 {
			path := strings.TrimSpace(m[1])
			if path != "" && !seen[path] {
				seen[path] = true
				conflicts = append(conflicts, Conflict{
					Path:      path,
					Path1Info: extractInfo(clean, "Path1"),
					Path2Info: extractInfo(clean, "Path2"),
				})
			}
		}
	}
	return conflicts
}

func extractInfo(line, side string) string {
	if strings.Contains(line, side) {
		return side + " version"
	}
	if strings.Contains(strings.ToLower(line), "not found") {
		return "missing"
	}
	if strings.Contains(strings.ToLower(line), "is new") {
		return "new"
	}
	return "differs"
}
