package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/turkosaurus/gh-ci/internal/ui/keys"
	"github.com/turkosaurus/gh-ci/internal/ui/styles"
)

func TestLogViewerSearch(t *testing.T) {
	s := styles.DefaultStyles()
	k := keys.DefaultKeyMap()

	t.Run("enter search mode and submit query", func(t *testing.T) {
		lv := NewLogViewer(s, k)
		lv.SetLogs("line one\nline two error\nline three\nline four error\nline five", "test-job")

		// press / to enter search
		lv, _ = lv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}, 40)
		assert.True(t, lv.searching)

		// type "error"
		for _, r := range "error" {
			lv, _ = lv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}, 40)
		}
		// press enter to submit
		lv, _ = lv.Update(tea.KeyMsg{Type: tea.KeyEnter}, 40)
		assert.False(t, lv.searching)
		assert.Equal(t, "error", lv.logQuery)
		assert.NotEmpty(t, lv.contextLines)
		assert.NotEmpty(t, lv.matchGroups)
	})

	t.Run("search next and prev", func(t *testing.T) {
		lv := NewLogViewer(s, k)
		lv.SetLogs(strings.Repeat("filler\n", 20)+"ERROR here\n"+strings.Repeat("filler\n", 20)+"ERROR again\n", "test-job")

		// manually set up search state
		lines := strings.Split(lv.logs, "\n")
		lv.logQuery = "ERROR"
		lv.contextLines, lv.matchGroups = buildLogContext(lines, "ERROR", 3)
		require.True(t, len(lv.matchGroups) >= 2, "expected at least 2 match groups")

		assert.Equal(t, 0, lv.matchIdx)

		// press n for next
		lv, _ = lv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}, 40)
		assert.Equal(t, 1, lv.matchIdx)

		// press p for prev
		lv, _ = lv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}, 40)
		assert.Equal(t, 0, lv.matchIdx)

		// press p again, should not go below 0
		lv, _ = lv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}, 40)
		assert.Equal(t, 0, lv.matchIdx)
	})

	t.Run("escape cancels search mode", func(t *testing.T) {
		lv := NewLogViewer(s, k)
		lv.SetLogs("test", "job")

		// enter search mode
		lv, _ = lv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}, 40)
		assert.True(t, lv.searching)

		// escape
		lv, _ = lv.Update(tea.KeyMsg{Type: tea.KeyEscape}, 40)
		assert.False(t, lv.searching)
	})

	t.Run("back returns to main", func(t *testing.T) {
		lv := NewLogViewer(s, k)
		lv.SetLogs("test", "job")

		lv, cmd := lv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}, 40)
		require.NotNil(t, cmd)
		msg := cmd()
		_, ok := msg.(backToMainMsg)
		assert.True(t, ok, "expected backToMainMsg, got %T", msg)
	})
}

func TestLogViewerScrolling(t *testing.T) {
	s := styles.DefaultStyles()
	k := keys.DefaultKeyMap()

	// create a viewer with many lines
	lines := make([]string, 100)
	for i := range lines {
		lines[i] = "line"
	}
	lv := NewLogViewer(s, k)
	lv.SetLogs(strings.Join(lines, "\n"), "job")

	// height 30 with logViewOverhead gives visible area
	height := 30
	assert.Equal(t, 0, lv.logOffset)

	// scroll down
	lv, _ = lv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}, height)
	assert.Equal(t, 1, lv.logOffset)

	// scroll up
	lv, _ = lv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}, height)
	assert.Equal(t, 0, lv.logOffset)

	// up at 0 stays at 0
	lv, _ = lv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}, height)
	assert.Equal(t, 0, lv.logOffset)

	// jump to bottom
	lv, _ = lv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}, height)
	assert.Greater(t, lv.logOffset, 0)

	// jump to top
	lv, _ = lv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}, height)
	assert.Equal(t, 0, lv.logOffset)
}
