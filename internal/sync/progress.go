package sync

import (
	"fmt"
	"regexp"
	"strings"
)

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// StripANSI removes ANSI escape sequences from a string.
func StripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// ProgressInfo holds parsed progress data from rclone output.
type ProgressInfo struct {
	Transferred string // e.g. "1.234 GiB / 5.678 GiB"
	Speed       string // e.g. "10.5 MiB/s"
	ETA         string // e.g. "1m30s"
	Percent     int    // 0-100
	Checks      string
	Errors      int
}

var (
	transferredRegex = regexp.MustCompile(`Transferred:\s+(.+)`)
	speedRegex       = regexp.MustCompile(`(\d+[\.\d]*\s*[KMGT]?i?B/s)`)
	etaRegex         = regexp.MustCompile(`ETA\s+(\S+)`)
	percentRegex     = regexp.MustCompile(`(\d+)%`)
	errorsRegex      = regexp.MustCompile(`Errors:\s+(\d+)`)
)

// ParseProgress extracts progress info from a line of rclone output.
func ParseProgress(line string) *ProgressInfo {
	clean := StripANSI(line)
	if !strings.Contains(clean, "Transferred") && !strings.Contains(clean, "Errors") {
		return nil
	}

	info := &ProgressInfo{}
	found := false

	if m := transferredRegex.FindStringSubmatch(clean); len(m) > 1 {
		info.Transferred = strings.TrimSpace(m[1])
		found = true
	}
	if m := speedRegex.FindStringSubmatch(clean); len(m) > 1 {
		info.Speed = m[1]
		found = true
	}
	if m := etaRegex.FindStringSubmatch(clean); len(m) > 1 {
		info.ETA = m[1]
		found = true
	}
	if m := percentRegex.FindStringSubmatch(clean); len(m) > 1 {
		fmt.Sscanf(m[1], "%d", &info.Percent) //nolint:errcheck
		found = true
	}
	if m := errorsRegex.FindStringSubmatch(clean); len(m) > 1 {
		fmt.Sscanf(m[1], "%d", &info.Errors) //nolint:errcheck
		found = true
	}

	if !found {
		return nil
	}
	return info
}

var (
	filesTransRegex = regexp.MustCompile(`Transferred:\s+(\d+)\s*/`)
	bytesTransRegex = regexp.MustCompile(`Transferred:\s+[\d.]+\s*\w+\s*/\s*([\d.]+)\s*(\w+)`)
)

// ParseTransferSummary extracts files and bytes transferred from rclone output.
func ParseTransferSummary(output string) (files int, bytes int64) {
	for _, line := range strings.Split(output, "\n") {
		clean := StripANSI(line)
		if m := filesTransRegex.FindStringSubmatch(clean); len(m) > 1 {
			fmt.Sscanf(m[1], "%d", &files) //nolint:errcheck
		}
		if m := bytesTransRegex.FindStringSubmatch(clean); len(m) > 2 {
			var val float64
			fmt.Sscanf(m[1], "%f", &val) //nolint:errcheck
			unit := strings.ToUpper(m[2])
			switch {
			case strings.HasPrefix(unit, "G"):
				bytes = int64(val * 1024 * 1024 * 1024)
			case strings.HasPrefix(unit, "M"):
				bytes = int64(val * 1024 * 1024)
			case strings.HasPrefix(unit, "K"):
				bytes = int64(val * 1024)
			default:
				bytes = int64(val)
			}
		}
	}
	return
}
