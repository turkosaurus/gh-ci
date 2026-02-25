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

func TestScanLocalWorkflows(t *testing.T) {
	wfs, err := scanLocalWorkflows()
	require.NoError(t, err, "Expected to find local workflows")
	require.NotEmpty(t, wfs, "Expected to find at least one local workflow file in .github/workflows")
	for _, wf := range wfs {
		t.Logf("Found local workflow: %s", wf.File)
	}
}

func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		line  string
		query string
		want  bool
	}{
		{"hello world", "hw", true},
		{"hello world", "HW", true},
		{"hello", "hello", true},
		{"hello", "", true},
		{"hi", "hello", false},
		{"abc", "ac", true},
		{"abc", "ca", false},
	}
	for _, tt := range tests {
		got := fuzzyMatch(tt.line, tt.query)
		if got != tt.want {
			t.Errorf("fuzzyMatch(%q, %q) = %v, want %v", tt.line, tt.query, got, tt.want)
		}
	}
}

func TestBuildLogContext(t *testing.T) {
	t.Run("no matches", func(t *testing.T) {
		lines := []string{"alpha", "beta", "gamma"}
		rows, offsets := buildLogContext(lines, "nomatch", 2)
		if len(rows) != 0 {
			t.Errorf("expected 0 rows, got %d", len(rows))
		}
		if len(offsets) != 0 {
			t.Errorf("expected 0 offsets, got %d", len(offsets))
		}
	})

	t.Run("single match in middle ctx=1", func(t *testing.T) {
		lines := []string{"a", "b", "c", "d", "e"}
		rows, offsets := buildLogContext(lines, "c", 1)
		// expect lines b(2), c(3), d(4) => 3 rows
		if len(rows) != 3 {
			t.Fatalf("expected 3 rows, got %d", len(rows))
		}
		if rows[0].text != "b" || rows[1].text != "c" || rows[2].text != "d" {
			t.Errorf("unexpected rows: %+v", rows)
		}
		if !rows[1].isMatch {
			t.Errorf("expected row[1] (c) to be a match")
		}
		if rows[0].isMatch || rows[2].isMatch {
			t.Errorf("expected only the match line to have isMatch=true")
		}
		if len(offsets) != 1 || offsets[0] != 0 {
			t.Errorf("expected groupOffsets=[0], got %v", offsets)
		}
	})

	t.Run("two adjacent matches merge into one window", func(t *testing.T) {
		// "b" and "d" both match with ctx=1: windows [a,b,c] and [c,d,e] overlap → merged
		lines := []string{"a", "b-match", "c", "d-match", "e"}
		rows, offsets := buildLogContext(lines, "match", 1)
		// merged window: a, b-match, c, d-match, e => 5 rows, 1 group
		if len(rows) != 5 {
			t.Fatalf("expected 5 rows, got %d: %+v", len(rows), rows)
		}
		if len(offsets) != 1 {
			t.Errorf("expected 1 group offset, got %d: %v", len(offsets), offsets)
		}
		matchCount := 0
		for _, r := range rows {
			if r.isMatch {
				matchCount++
			}
		}
		if matchCount != 2 {
			t.Errorf("expected 2 match rows, got %d", matchCount)
		}
	})

	t.Run("two non-adjacent matches with blank separator", func(t *testing.T) {
		lines := []string{"a", "match1", "c", "d", "e", "f", "match2", "h"}
		rows, offsets := buildLogContext(lines, "match", 1)
		// group1: a,match1,c  group2: f,match2,h  + blank separator between them
		if len(offsets) != 2 {
			t.Errorf("expected 2 group offsets, got %d: %v", len(offsets), offsets)
		}
		// find blank separator
		hasSeparator := false
		for _, r := range rows {
			if r.lineNo == 0 {
				hasSeparator = true
				break
			}
		}
		if !hasSeparator {
			t.Errorf("expected a blank separator row (lineNo==0) between groups")
		}
	})

	t.Run("match at first line boundary", func(t *testing.T) {
		lines := []string{"match", "b", "c", "d"}
		rows, offsets := buildLogContext(lines, "match", 1)
		if len(offsets) != 1 {
			t.Errorf("expected 1 group, got %d", len(offsets))
		}
		if len(rows) < 1 {
			t.Fatal("expected at least 1 row")
		}
		if !rows[0].isMatch {
			t.Errorf("expected first row to be the match")
		}
	})

	t.Run("match at last line boundary", func(t *testing.T) {
		lines := []string{"a", "b", "c", "match"}
		rows, offsets := buildLogContext(lines, "match", 1)
		if len(offsets) != 1 {
			t.Errorf("expected 1 group, got %d", len(offsets))
		}
		last := rows[len(rows)-1]
		if !last.isMatch {
			t.Errorf("expected last row to be the match, got %+v", last)
		}
	})
}

// --- Sub-model tests ---

func TestBranchPicker(t *testing.T) {
	branches := []string{"feature/auth", "main", "staging"}

	t.Run("open and close with esc", func(t *testing.T) {
		bp := NewBranchPicker()
		assert.False(t, bp.Active())

		bp.Open(branches)
		assert.True(t, bp.Active())

		bp, _, result := bp.Update(tea.KeyMsg{Type: tea.KeyEscape})
		assert.False(t, bp.Active())
		assert.Nil(t, result)
	})

	t.Run("select with enter", func(t *testing.T) {
		bp := NewBranchPicker()
		bp.Open(branches)

		// press enter to select first suggestion (feature/auth)
		bp, _, result := bp.Update(tea.KeyMsg{Type: tea.KeyEnter})
		assert.False(t, bp.Active())
		require.NotNil(t, result)
		assert.Equal(t, "feature/auth", result.Chosen)
	})

	t.Run("navigate down and select", func(t *testing.T) {
		bp := NewBranchPicker()
		bp.Open(branches)

		// move down once then select
		bp, _, _ = bp.Update(tea.KeyMsg{Type: tea.KeyDown})
		bp, _, result := bp.Update(tea.KeyMsg{Type: tea.KeyEnter})
		require.NotNil(t, result)
		assert.Equal(t, "main", result.Chosen)
	})

	t.Run("up does not go below 0", func(t *testing.T) {
		bp := NewBranchPicker()
		bp.Open(branches)

		bp, _, _ = bp.Update(tea.KeyMsg{Type: tea.KeyUp})
		bp, _, result := bp.Update(tea.KeyMsg{Type: tea.KeyEnter})
		require.NotNil(t, result)
		assert.Equal(t, "feature/auth", result.Chosen)
	})
}

func TestConfirmDialog(t *testing.T) {
	t.Run("open and cancel", func(t *testing.T) {
		var cd ConfirmDialog
		assert.False(t, cd.Active())

		cd.Open("owner/repo", 123)
		assert.True(t, cd.Active())

		cd, cmd, msg := cd.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}, nil)
		assert.False(t, cd.Active())
		assert.Nil(t, cmd)
		assert.Empty(t, msg)
	})

	t.Run("confirm with y", func(t *testing.T) {
		var cd ConfirmDialog
		cd.Open("owner/repo", 123)

		cd, cmd, msg := cd.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}, nil)
		assert.False(t, cd.Active())
		assert.NotNil(t, cmd, "expected a rerun command")
		assert.Equal(t, "re-running...", msg)
	})

	t.Run("confirm with d for debug", func(t *testing.T) {
		var cd ConfirmDialog
		cd.Open("owner/repo", 456)

		cd, cmd, msg := cd.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}, nil)
		assert.False(t, cd.Active())
		assert.NotNil(t, cmd, "expected a debug rerun command")
		assert.Equal(t, "re-running with debug...", msg)
	})
}

func TestDispatchDialog(t *testing.T) {
	t.Run("open and cancel with esc", func(t *testing.T) {
		var dd DispatchDialog
		assert.False(t, dd.Active())

		dd.Open("owner/repo", "ci.yml", "main")
		assert.True(t, dd.Active())

		dd, cmd, msg := dd.Update(tea.KeyMsg{Type: tea.KeyEscape}, nil)
		assert.False(t, dd.Active())
		assert.Nil(t, cmd)
		assert.Empty(t, msg)
	})

	t.Run("confirm dispatch with y", func(t *testing.T) {
		var dd DispatchDialog
		dd.Open("owner/repo", "ci.yml", "main")

		dd, cmd, msg := dd.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}, nil)
		assert.False(t, dd.Active())
		assert.NotNil(t, cmd, "expected a dispatch command")
		assert.Equal(t, "dispatching...", msg)
	})

	t.Run("help view includes file and ref", func(t *testing.T) {
		var dd DispatchDialog
		dd.Open("owner/repo", "deploy.yml", "staging")

		s := styles.DefaultStyles()
		help := dd.HelpView(s)
		assert.Contains(t, help, "deploy.yml")
		assert.Contains(t, help, "staging")
	})
}

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
