package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jay-418/gh-ci/internal/gh"
	"github.com/jay-418/gh-ci/internal/types"
	"github.com/jay-418/gh-ci/internal/ui/styles"
)

// RunList is a component that displays a list of workflow runs
type RunList struct {
	Runs     []types.WorkflowRun
	Selected int
	Styles   styles.Styles
	Width    int
	Height   int
}

// NewRunList creates a new run list component
func NewRunList(s styles.Styles) RunList {
	return RunList{
		Runs:     []types.WorkflowRun{},
		Selected: 0,
		Styles:   s,
	}
}

// SetRuns sets the workflow runs
func (r *RunList) SetRuns(runs []types.WorkflowRun) {
	r.Runs = runs
	if r.Selected >= len(runs) {
		r.Selected = max(0, len(runs)-1)
	}
}

// SetSize sets the dimensions of the list
func (r *RunList) SetSize(width, height int) {
	r.Width = width
	r.Height = height
}

// MoveUp moves the selection up
func (r *RunList) MoveUp() {
	if r.Selected > 0 {
		r.Selected--
	}
}

// MoveDown moves the selection down
func (r *RunList) MoveDown() {
	if r.Selected < len(r.Runs)-1 {
		r.Selected++
	}
}

// PageUp moves the selection up by a page
func (r *RunList) PageUp(pageSize int) {
	r.Selected = max(0, r.Selected-pageSize)
}

// PageDown moves the selection down by a page
func (r *RunList) PageDown(pageSize int) {
	r.Selected = min(len(r.Runs)-1, r.Selected+pageSize)
}

// GoToTop moves the selection to the top
func (r *RunList) GoToTop() {
	r.Selected = 0
}

// GoToBottom moves the selection to the bottom
func (r *RunList) GoToBottom() {
	if len(r.Runs) > 0 {
		r.Selected = len(r.Runs) - 1
	}
}

// SelectedRun returns the currently selected run
func (r *RunList) SelectedRun() *types.WorkflowRun {
	if r.Selected >= 0 && r.Selected < len(r.Runs) {
		return &r.Runs[r.Selected]
	}
	return nil
}

// View renders the run list
func (r *RunList) View() string {
	if len(r.Runs) == 0 {
		return r.Styles.Dimmed.Render("No workflow runs found")
	}

	var rows []string

	// Header
	header := r.renderHeader()
	rows = append(rows, header)

	// Calculate visible rows (leave room for header)
	visibleRows := r.Height - 2
	if visibleRows < 1 {
		visibleRows = 10
	}

	// Calculate scroll offset
	startIdx := 0
	if r.Selected >= visibleRows {
		startIdx = r.Selected - visibleRows + 1
	}

	endIdx := min(startIdx+visibleRows, len(r.Runs))

	for i := startIdx; i < endIdx; i++ {
		run := r.Runs[i]
		row := r.renderRow(run, i == r.Selected)
		rows = append(rows, row)
	}

	// Scroll indicator
	if len(r.Runs) > visibleRows {
		scrollInfo := fmt.Sprintf(" %d/%d ", r.Selected+1, len(r.Runs))
		rows = append(rows, r.Styles.Dimmed.Render(scrollInfo))
	}

	return strings.Join(rows, "\n")
}

// renderHeader renders the header row
func (r *RunList) renderHeader() string {
	// Column widths
	colStatus := 3
	colRepo := 25
	colWorkflow := 20
	colBranch := 20
	colDuration := 10
	colNumber := 8

	header := fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s %-*s",
		colStatus, " ",
		colRepo, "REPO",
		colWorkflow, "WORKFLOW",
		colBranch, "BRANCH",
		colNumber, "RUN",
		colDuration, "DURATION",
	)

	return r.Styles.Header.Render(header)
}

// renderRow renders a single row
func (r *RunList) renderRow(run types.WorkflowRun, selected bool) string {
	// Column widths
	colStatus := 3
	colRepo := 25
	colWorkflow := 20
	colBranch := 20
	colDuration := 10
	colNumber := 8

	// Status icon
	icon := styles.StatusIcon(run.Status, run.Conclusion)
	statusStyle := r.Styles.StatusStyle(run.Status, run.Conclusion)
	statusStr := statusStyle.Render(fmt.Sprintf("%-*s", colStatus, icon))

	// Repo name (truncate if needed)
	repoName := gh.TruncateString(run.Repository.FullName, colRepo)
	repoStr := r.Styles.Repo.Render(fmt.Sprintf("%-*s", colRepo, repoName))

	// Workflow name
	workflowName := gh.TruncateString(run.Name, colWorkflow)
	workflowStr := fmt.Sprintf("%-*s", colWorkflow, workflowName)

	// Branch
	branch := gh.TruncateString(run.HeadBranch, colBranch)
	branchStr := r.Styles.Branch.Render(fmt.Sprintf("%-*s", colBranch, branch))

	// Run number
	runNum := fmt.Sprintf("#%d", run.RunNumber)
	runNumStr := fmt.Sprintf("%-*s", colNumber, runNum)

	// Duration
	duration := run.Duration()
	durationSec := int64(duration.Seconds())
	durationStr := r.Styles.Duration.Render(fmt.Sprintf("%-*s", colDuration, gh.FormatDuration(durationSec)))

	row := fmt.Sprintf("%s %s %s %s %s %s",
		statusStr,
		repoStr,
		workflowStr,
		branchStr,
		runNumStr,
		durationStr,
	)

	if selected {
		// Apply selected style - we need to strip existing styles first
		row = lipgloss.NewStyle().
			Bold(true).
			Background(styles.ColorBgLight).
			Foreground(styles.ColorWhite).
			Render(row)
	}

	return row
}

// FilterRuns filters the runs based on status
func FilterRuns(runs []types.WorkflowRun, filter types.StatusFilter, searchQuery string) []types.WorkflowRun {
	if filter == types.StatusAll && searchQuery == "" {
		return runs
	}

	var filtered []types.WorkflowRun
	for _, run := range runs {
		// Filter by status
		if filter != types.StatusAll {
			status := run.GetStatus()
			switch filter {
			case types.StatusFailed:
				if status != "failure" {
					continue
				}
			case types.StatusSuccess:
				if status != "success" {
					continue
				}
			case types.StatusInProgress:
				if run.Status != "in_progress" && run.Status != "queued" {
					continue
				}
			}
		}

		// Filter by search query
		if searchQuery != "" {
			query := strings.ToLower(searchQuery)
			if !strings.Contains(strings.ToLower(run.Name), query) &&
				!strings.Contains(strings.ToLower(run.HeadBranch), query) &&
				!strings.Contains(strings.ToLower(run.Repository.FullName), query) {
				continue
			}
		}

		filtered = append(filtered, run)
	}

	return filtered
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
