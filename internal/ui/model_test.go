package ui

import (
	"testing"
)

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
