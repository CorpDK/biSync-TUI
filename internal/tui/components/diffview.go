package components

import (
	"fmt"
	"strings"

	bisync "github.com/CorpDK/bisync-tui/internal/sync"
	"github.com/CorpDK/bisync-tui/internal/tui/theme"
)

// DiffViewModel renders color-coded diff entries.
type DiffViewModel struct {
	entries []bisync.DiffEntry
	width   int
	height  int
}

// NewDiffView creates a new diff view component.
func NewDiffView(w, h int) DiffViewModel {
	return DiffViewModel{width: w, height: h}
}

// SetEntries updates the displayed diff entries.
func (m *DiffViewModel) SetEntries(entries []bisync.DiffEntry) {
	m.entries = entries
}

// SetSize updates dimensions.
func (m *DiffViewModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// View renders the diff entries with color coding.
func (m DiffViewModel) View() string {
	if len(m.entries) == 0 {
		return "  No changes detected. Press D to run diff."
	}

	var b strings.Builder
	header := theme.DetailHeaderStyle.Render("Diff Preview (dry-run)")
	b.WriteString(header + "\n\n")

	// Summary counts
	added, deleted, modified := m.counts()
	summary := fmt.Sprintf("  %s added, %s deleted, %s modified",
		theme.DiffAddedStyle.Render(fmt.Sprintf("%d", added)),
		theme.DiffDeletedStyle.Render(fmt.Sprintf("%d", deleted)),
		theme.DiffModifiedStyle.Render(fmt.Sprintf("%d", modified)),
	)
	b.WriteString(summary + "\n\n")

	// Entries
	for _, e := range m.entries {
		b.WriteString(m.renderEntry(e) + "\n")
	}

	return b.String()
}

func (m DiffViewModel) renderEntry(e bisync.DiffEntry) string {
	switch e.Type {
	case bisync.DiffAdded:
		return theme.DiffAddedStyle.Render(fmt.Sprintf("  + [%s] %s", e.Side, e.Path))
	case bisync.DiffDeleted:
		return theme.DiffDeletedStyle.Render(fmt.Sprintf("  - [%s] %s", e.Side, e.Path))
	case bisync.DiffModified:
		return theme.DiffModifiedStyle.Render(fmt.Sprintf("  ~ [%s] %s", e.Side, e.Path))
	default:
		return fmt.Sprintf("  ? [%s] %s", e.Side, e.Path)
	}
}

func (m DiffViewModel) counts() (added, deleted, modified int) {
	for _, e := range m.entries {
		switch e.Type {
		case bisync.DiffAdded:
			added++
		case bisync.DiffDeleted:
			deleted++
		case bisync.DiffModified:
			modified++
		}
	}
	return
}
