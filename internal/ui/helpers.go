package ui

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
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

// deriveLists collects unique workflow names (sorted, prefixed with workflowAll)
// and unique branch names (sorted, no sentinel — all entries are real branches).
// localDefs are merged in so workflows with no runs still appear.
func deriveLists(localDefs []types.WorkflowDef, runs []types.WorkflowRun) ([]string, []string) {
	workflowList := []string{workflowAll} // initiate list with "all" selector
	// append all local names first
	for _, def := range localDefs {
		workflowList = append(workflowList, def.Name)
	}
	var branchList []string
	for _, wf := range runs {
		if !slices.Contains(branchList, wf.HeadBranch) {
			branchList = append(branchList, wf.HeadBranch)
		}
		if !slices.Contains(workflowList, wf.Name) {
			// TODO: merge on name or file path
			if !slices.Contains(workflowList, filepath.Base(wf.Path)) {
				workflowList = append(workflowList, wf.Name)
			}
		}
	}
	slog.Debug("derrived lists",
		"workflowList", workflowList,
		"branchList", branchList,
	)
	return workflowList, branchList
}

func gitRoot() string {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		slog.Error("determine git root", "error", err)
		return ""
	}
	return strings.TrimSpace(string(out))
}

func gitBranch() string {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		slog.Error("determine git branch", "error", err)
		return ""
	}
	return strings.TrimSpace(string(out))
}

var workflowDirGitHub = filepath.Join(".github", "workflows")

// scanLocalWorkflows looks for workflow definition files
// in .github/workflows/ and returns their names and filenames.
func scanLocalWorkflows() ([]types.WorkflowDef, error) {
	var defs []types.WorkflowDef
	for _, pattern := range []string{"*.yaml", "*.yml"} {
		matches, err := filepath.Glob(filepath.Join(gitRoot(), workflowDirGitHub, pattern))
		if err != nil {
			return nil, fmt.Errorf("glob workflow files %q: %w", pattern, err)
		}
		if len(matches) == 0 {
			continue
		}
		for _, path := range matches {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("read workflow file %q: %w", path, err)
			}
			var wf struct {
				Name string `yaml:"name"`
			}
			err = yaml.Unmarshal(data, &wf)
			if err != nil {
				slog.Error(fmt.Errorf("unmarshal workflow: %w", err).Error(),
					"file", path,
				)
			}
			file := filepath.Base(path)
			var name string
			if wf.Name != "" {
				name = wf.Name
			} else {
				name = file
			}
			defs = append(defs, types.WorkflowDef{
				Name: name,
				File: file,
			})
			slog.Debug("discovered local workflow definition", "name", name, "file", file)
		}
	}
	slog.Debug("returning local workflows", "count", len(defs))
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
