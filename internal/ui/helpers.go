package ui

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/turkosaurus/gh-ci/internal/types"
)

const (
	workflowAll     = "*" // "show all workflows"
	logViewOverhead = 4   // number of rows consumed by header, spacing, and help bar in the log view.
)

// logContextLine is one display row in the context-window log view.
type logContextLine struct {
	lineNo  int // 1-based original line number; 0 = blank separator
	text    string
	isMatch bool
}

// fuzzyMatch returns true if every character of query appears in order in line (case-insensitive).
func fuzzyMatch(line, query string) bool {
	line = strings.ToLower(line)
	query = strings.ToLower(query)
	queryRunes := []rune(query)
	qi := 0
	for _, ch := range line {
		if qi < len(queryRunes) && ch == queryRunes[qi] {
			qi++
		}
	}
	return qi == len(queryRunes)
}

// buildLogContext produces a grep -C ctx style context-window view.
// Returns the flat row list and, for each match group, the row offset where it starts.
func buildLogContext(lines []string, query string, ctx int) (rows []logContextLine, groupOffsets []int) {
	// collect matching line indices
	var matches []int
	for i, l := range lines {
		if fuzzyMatch(l, query) {
			matches = append(matches, i)
		}
	}
	if len(matches) == 0 {
		return
	}

	// merge overlapping windows and emit rows
	prevEnd := -1
	for _, mIdx := range matches {
		start := max(0, mIdx-ctx)
		end := min(len(lines)-1, mIdx+ctx)

		if prevEnd < 0 || start > prevEnd+1 {
			// new non-adjacent group
			if prevEnd >= 0 {
				rows = append(rows, logContextLine{}) // blank separator
			}
			groupOffsets = append(groupOffsets, len(rows))
			for i := start; i <= end; i++ {
				rows = append(rows, logContextLine{lineNo: i + 1, text: lines[i], isMatch: i == mIdx})
			}
		} else {
			// overlapping with previous group: extend (this match is another hit inside same window)
			// mark the match line itself if it wasn't already included
			for i := prevEnd + 1; i <= end; i++ {
				rows = append(rows, logContextLine{lineNo: i + 1, text: lines[i], isMatch: i == mIdx})
			}
			// also mark already-emitted rows that happen to be this match
			for j := range rows {
				if rows[j].lineNo == mIdx+1 {
					rows[j].isMatch = true
					break
				}
			}
		}
		prevEnd = end
	}
	return
}

// deriveWorkflows collects unique workflow names (sorted, prefixed with workflowAll)
// and unique branch names (sorted, no sentinel — all entries are real branches).
// localDefs are merged in so workflows with no runs still appear.
func deriveWorkflows(runs []types.WorkflowRun, localDefs []types.WorkflowDef) (workflows []string, branches []string) {
	wfSeen := map[string]bool{}
	brSeen := map[string]bool{}
	for _, r := range runs {
		wfSeen[r.Name] = true
		brSeen[r.HeadBranch] = true
	}
	for _, def := range localDefs {
		wfSeen[def.Name] = true
	}
	for w := range wfSeen {
		workflows = append(workflows, w)
	}
	sort.Strings(workflows)
	workflows = append([]string{workflowAll}, workflows...)

	for b := range brSeen {
		branches = append(branches, b)
	}
	sort.Strings(branches)
	return
}

func currentGitBranch() string {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

var ErrNoLocalWorkflows = errors.New("no local workflows found")

// scanLocalWorkflows looks for workflow definition files
// in .github/workflows/ and returns their names and filenames.
func scanLocalWorkflows() ([]types.WorkflowDef, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return nil, fmt.Errorf("no parseable git root: %w", err)
	}
	root := strings.TrimSpace(string(out))

	var defs []types.WorkflowDef
	for _, pattern := range []string{"*.yaml", "*.yml"} {
		matches, _ := filepath.Glob(filepath.Join(root, ".github", "workflows", pattern))
		if len(matches) == 0 {
			return nil, ErrNoLocalWorkflows
		}
		for _, path := range matches {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("read workflow file %q: %w", path, err)
			}
			var wf struct {
				Name string `yaml:"name"`
			}
			_ = yaml.Unmarshal(data, &wf)
			name := wf.Name
			if name == "" {
				ext := filepath.Ext(path)
				name = strings.TrimSuffix(filepath.Base(path), ext)
			}
			file := filepath.Base(path)
			defs = append(defs, types.WorkflowDef{Name: name, File: file})
			slog.Debug("discovered local workflow definition", "name", name, "file", file)
		}
	}
	return defs, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
