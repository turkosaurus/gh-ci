package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/turkosaurus/gh-ci/internal/config"
	"github.com/turkosaurus/gh-ci/internal/gh"
	"github.com/turkosaurus/gh-ci/internal/types"
	"github.com/turkosaurus/gh-ci/internal/ui/keys"
	"github.com/turkosaurus/gh-ci/internal/ui/styles"
)

const (
	panelWorkflows = iota
	panelRuns
	panelDetail
)

const timestampFormat = "2006-01-02 15:04"

// colSep is the number of spaces between columns in the runs list.
const colSep = 2

// Dashboard manages the main three-panel view (workflows, runs, detail).
type Dashboard struct {
	activePanel    int
	workflowCursor int
	cursor         int
	jobCursor      int

	filteredRuns      []types.WorkflowRun
	workflows         []string
	availableBranches []string
	branchIdx         int
	jobs              []types.Job

	branchPicker   BranchPicker
	confirmDialog  ConfirmDialog
	dispatchDialog DispatchDialog

	PendingMessage string // App reads and clears after each Update

	config        *config.Config
	client        gh.Client
	styles        styles.Styles
	keys          keys.KeyMap
	allRuns       []types.WorkflowRun
	localDefs     []types.WorkflowDef
	workflowFiles map[string]string
	defaultBranch string
	localBranch   string
}

// NewDashboard creates a Dashboard with the given dependencies.
func NewDashboard(cfg *config.Config, client gh.Client, s styles.Styles, k keys.KeyMap, defaultBranch, localBranch string) Dashboard {
	return Dashboard{
		config:         cfg,
		client:         client,
		styles:         s,
		keys:           k,
		branchPicker:   NewBranchPicker(),
		workflowCursor: 1, // start on workflowAll (0=branch, 1=workflows[0])
		workflowFiles:  make(map[string]string),
		defaultBranch:  defaultBranch,
		localBranch:    localBranch,
	}
}

// SetRuns updates the dashboard with fresh run data and re-derives filters.
// workflowFiles is passed from App (which populates it from run paths + localDefs).
func (d *Dashboard) SetRuns(allRuns []types.WorkflowRun, localDefs []types.WorkflowDef, workflowFiles map[string]string) tea.Cmd {
	d.allRuns = allRuns
	d.localDefs = localDefs
	d.workflowFiles = workflowFiles

	// Preserve selected branch by name; on first load start on local checkout
	prevBranch := ""
	if d.availableBranches == nil {
		prevBranch = d.localBranch
	} else if d.branchIdx < len(d.availableBranches) {
		prevBranch = d.availableBranches[d.branchIdx]
	}
	_, d.availableBranches = deriveWorkflows(d.allRuns, d.localDefs)
	// ensure both the configured primary branch and the local checkout are
	// always present, even when they have no runs yet
	for _, branch := range []string{d.defaultBranch, d.localBranch} {
		if branch == "" {
			continue
		}
		found := false
		for _, b := range d.availableBranches {
			if b == branch {
				found = true
				break
			}
		}
		if !found {
			d.availableBranches = append(d.availableBranches, branch)
			sort.Strings(d.availableBranches)
		}
	}
	d.branchIdx = 0
	for i, b := range d.availableBranches {
		if b == prevBranch {
			d.branchIdx = i
			break
		}
	}
	d.applyFilter()
	if run := d.selectedRun(); run != nil {
		return loadJobs(d.client, run.Repository.FullName, run.ID)
	}
	return nil
}

// SetJobs updates the dashboard with fresh job data.
func (d *Dashboard) SetJobs(jobs []types.Job) {
	d.jobs = jobs
	if d.jobCursor >= len(d.jobs) {
		d.jobCursor = 0
	}
}

// Update handles a key event and returns the updated dashboard and command.
func (d Dashboard) Update(msg tea.KeyMsg) (Dashboard, tea.Cmd) {
	if d.branchPicker.Active() {
		return d.handleBranchSelect(msg)
	}
	if d.dispatchDialog.Active() {
		return d.handleDispatchConfirm(msg)
	}
	if d.confirmDialog.Active() {
		return d.handleConfirm(msg)
	}
	return d.handleMainKeys(msg)
}

func (d Dashboard) handleBranchSelect(msg tea.KeyMsg) (Dashboard, tea.Cmd) {
	var cmd tea.Cmd
	var result *BranchPickResult
	d.branchPicker, cmd, result = d.branchPicker.Update(msg)
	if result != nil {
		for i, b := range d.availableBranches {
			if b == result.Chosen {
				d.branchIdx = i
				break
			}
		}
		d.workflowCursor = 1 // land on workflowAll so next Enter goes right, not re-opens selector
		d.applyFilter()
		d.cursor = 0
	}
	return d, cmd
}

func (d Dashboard) handleConfirm(msg tea.KeyMsg) (Dashboard, tea.Cmd) {
	var cmd tea.Cmd
	var statusMsg string
	d.confirmDialog, cmd, statusMsg = d.confirmDialog.Update(msg, d.client)
	if statusMsg != "" {
		d.PendingMessage = statusMsg
	}
	return d, cmd
}

func (d Dashboard) handleDispatchConfirm(msg tea.KeyMsg) (Dashboard, tea.Cmd) {
	var cmd tea.Cmd
	var statusMsg string
	d.dispatchDialog, cmd, statusMsg = d.dispatchDialog.Update(msg, d.client)
	if statusMsg != "" {
		d.PendingMessage = statusMsg
	}
	return d, cmd
}

func (d Dashboard) filteredBranches() []string {
	q := strings.ToLower(d.branchPicker.input.Value())
	var out []string
	for _, b := range d.availableBranches {
		if q == "" || strings.Contains(strings.ToLower(b), q) {
			out = append(out, b)
		}
	}
	return out
}

func (d Dashboard) selectedBranch() string {
	if d.branchIdx < len(d.availableBranches) {
		return d.availableBranches[d.branchIdx]
	}
	return d.defaultBranch
}

func (d *Dashboard) applyFilter() {
	runs := d.allRuns

	// Apply branch filter — always active since there is no "all branches" option.
	branchRuns := runs
	if d.branchIdx < len(d.availableBranches) {
		branch := d.availableBranches[d.branchIdx]
		var br []types.WorkflowRun
		for _, r := range runs {
			if r.HeadBranch == branch {
				br = append(br, r)
			}
		}
		branchRuns = br
	}

	// Re-derive workflow list from branch-filtered runs, plus any local def that
	// has never run anywhere (so it can be dispatched from any branch).
	wfSeen := map[string]bool{}
	for _, r := range branchRuns {
		wfSeen[r.Name] = true
	}
	selectedBranch := ""
	if d.branchIdx < len(d.availableBranches) {
		selectedBranch = d.availableBranches[d.branchIdx]
	}
	if selectedBranch == d.localBranch && d.localBranch != "" {
		hasRunsAnywhere := map[string]bool{}
		for _, r := range d.allRuns {
			hasRunsAnywhere[r.Name] = true
		}
		for _, def := range d.localDefs {
			if !hasRunsAnywhere[def.Name] {
				wfSeen[def.Name] = true
			}
		}
	}
	var workflows []string
	for w := range wfSeen {
		workflows = append(workflows, w)
	}
	sort.Strings(workflows)
	workflows = append([]string{workflowAll}, workflows...)
	// Preserve workflowCursor by name across re-derives.
	if prevWf := d.selectedWorkflow(); prevWf != "" {
		d.workflowCursor = 1 // default to workflowAll if not found
		for i, w := range workflows {
			if w == prevWf {
				d.workflowCursor = i + 1
				break
			}
		}
	} else if d.workflowCursor > len(workflows) {
		d.workflowCursor = 0
	}
	d.workflows = workflows

	// Apply workflow filter.
	runs = branchRuns
	if wfName := d.selectedWorkflow(); wfName != "" && wfName != workflowAll {
		var wf []types.WorkflowRun
		for _, r := range runs {
			if r.Name == wfName {
				wf = append(wf, r)
			}
		}
		runs = wf
	}

	d.filteredRuns = runs
	if d.cursor >= len(d.filteredRuns) {
		d.cursor = max(0, len(d.filteredRuns)-1)
	}
}

func (d Dashboard) selectedRun() *types.WorkflowRun {
	if d.cursor >= 0 && d.cursor < len(d.filteredRuns) {
		return &d.filteredRuns[d.cursor]
	}
	return nil
}

// selectedWorkflow returns the workflow name at the current workflow cursor,
// or "" when the branch row (cursor 0) or an out-of-range position is selected.
func (d Dashboard) selectedWorkflow() string {
	if d.workflowCursor > 0 && d.workflowCursor <= len(d.workflows) {
		return d.workflows[d.workflowCursor-1]
	}
	return ""
}

func (d Dashboard) moveCursor(delta int) (Dashboard, tea.Cmd) {
	switch d.activePanel {
	case panelWorkflows:
		n := d.workflowCursor + delta
		if n >= 0 && n <= len(d.workflows) {
			d.workflowCursor = n
			d.applyFilter()
			d.cursor = 0
			d.jobs = nil
			d.jobCursor = 0
			if run := d.selectedRun(); run != nil {
				return d, loadJobs(d.client, run.Repository.FullName, run.ID)
			}
		}
	case panelRuns:
		n := d.cursor + delta
		if n >= 0 && n < len(d.filteredRuns) {
			d.cursor = n
			d.jobs = nil
			d.jobCursor = 0
			if run := d.selectedRun(); run != nil {
				return d, loadJobs(d.client, run.Repository.FullName, run.ID)
			}
		}
	case panelDetail:
		n := d.jobCursor + delta
		if n >= 0 && n < len(d.jobs) {
			d.jobCursor = n
		}
	}
	return d, nil
}

func (d Dashboard) moveCursorPage(dir int) (Dashboard, tea.Cmd) {
	const pageSize = 10
	switch d.activePanel {
	case panelWorkflows:
		n := max(0, min(len(d.workflows), d.workflowCursor+dir*pageSize))
		if n != d.workflowCursor {
			d.workflowCursor = n
			d.applyFilter()
			d.cursor = 0
			d.jobs = nil
			d.jobCursor = 0
			if run := d.selectedRun(); run != nil {
				return d, loadJobs(d.client, run.Repository.FullName, run.ID)
			}
		}
	case panelRuns:
		n := max(0, min(len(d.filteredRuns)-1, d.cursor+dir*pageSize))
		if n != d.cursor {
			d.cursor = n
			d.jobs = nil
			d.jobCursor = 0
			if run := d.selectedRun(); run != nil {
				return d, loadJobs(d.client, run.Repository.FullName, run.ID)
			}
		}
	}
	return d, nil
}

func (d Dashboard) moveCursorEdge(top bool) (Dashboard, tea.Cmd) {
	switch d.activePanel {
	case panelWorkflows:
		if top {
			d.workflowCursor = 0
		} else {
			d.workflowCursor = len(d.workflows)
		}
		d.applyFilter()
		d.cursor = 0
		d.jobs = nil
		d.jobCursor = 0
		if run := d.selectedRun(); run != nil {
			return d, loadJobs(d.client, run.Repository.FullName, run.ID)
		}
	case panelRuns:
		if top {
			d.cursor = 0
		} else {
			d.cursor = max(0, len(d.filteredRuns)-1)
		}
		d.jobs = nil
		d.jobCursor = 0
		if run := d.selectedRun(); run != nil {
			return d, loadJobs(d.client, run.Repository.FullName, run.ID)
		}
	case panelDetail:
		if top {
			d.jobCursor = 0
		} else {
			d.jobCursor = max(0, len(d.jobs)-1)
		}
	}
	return d, nil
}

func (d Dashboard) openURL() string {
	switch d.activePanel {
	case panelWorkflows:
		wfName := d.selectedWorkflow()
		if wfName == "" || wfName == workflowAll {
			// branch cell or "all workflows" — open repo actions page
			if run := d.selectedRun(); run != nil {
				return run.Repository.HTMLURL + "/actions"
			}
			if len(d.config.Repos) > 0 {
				return "https://github.com/" + d.config.Repos[0] + "/actions"
			}
		} else {
			// specific workflow — open its actions/workflows page
			for _, r := range d.allRuns {
				if r.Name == wfName {
					if filename, ok := d.workflowFiles[wfName]; ok {
						return r.Repository.HTMLURL + "/actions/workflows/" + filename
					}
					return r.HTMLURL
				}
			}
		}
	case panelRuns:
		if run := d.selectedRun(); run != nil {
			return run.HTMLURL
		}
	case panelDetail:
		if d.jobCursor < len(d.jobs) {
			return d.jobs[d.jobCursor].HTMLURL
		}
	}
	return ""
}

func (d Dashboard) handleMainKeys(msg tea.KeyMsg) (Dashboard, tea.Cmd) {
	switch {
	case key.Matches(msg, d.keys.Quit):
		return d, tea.Quit

	case key.Matches(msg, d.keys.Up):
		return d.moveCursor(-1)

	case key.Matches(msg, d.keys.Down):
		return d.moveCursor(1)

	case key.Matches(msg, d.keys.PageUp):
		return d.moveCursorPage(-1)

	case key.Matches(msg, d.keys.PageDown):
		return d.moveCursorPage(1)

	case key.Matches(msg, d.keys.Top):
		return d.moveCursorEdge(true)

	case key.Matches(msg, d.keys.Bottom):
		return d.moveCursorEdge(false)

	case key.Matches(msg, d.keys.Logs): // l — move right between panels
		if d.activePanel < panelDetail {
			d.activePanel++
		}

	case key.Matches(msg, d.keys.Enter):
		if d.activePanel == panelWorkflows && d.workflowCursor == 0 {
			cmd := d.branchPicker.Open(d.availableBranches)
			return d, cmd
		} else if d.activePanel < panelDetail {
			d.activePanel++
		} else if d.jobCursor < len(d.jobs) {
			// detail panel: enter opens logs for the selected job
			if run := d.selectedRun(); run != nil {
				job := d.jobs[d.jobCursor]
				d.PendingMessage = "loading logs..."
				return d, loadLogs(d.client, run.Repository.FullName, job.ID, job.Name)
			}
		}

	case key.Matches(msg, d.keys.Left): // move left between panels
		if d.activePanel > panelWorkflows {
			d.activePanel--
		}

	case key.Matches(msg, d.keys.Open):
		if url := d.openURL(); url != "" {
			d.client.OpenInBrowser(url)
		}

	case key.Matches(msg, d.keys.Rerun):
		if run := d.selectedRun(); run != nil {
			d.confirmDialog.Open(run.Repository.FullName, run.ID)
		}

	case key.Matches(msg, d.keys.Cancel):
		if run := d.selectedRun(); run != nil && run.Status == types.RunStatusInProgress {
			d.PendingMessage = "cancelling..."
			return d, cancelWorkflow(d.client, run.Repository.FullName, run.ID)
		}

	case key.Matches(msg, d.keys.Dispatch):
		if d.activePanel == panelWorkflows {
			if wfName := d.selectedWorkflow(); wfName != "" && wfName != workflowAll {
				if file, ok := d.workflowFiles[wfName]; ok {
					var repo string
					if run := d.selectedRun(); run != nil {
						// Derive repo from the currently visible run (already filtered by
						// branch + workflow), so dispatch always targets the right repo.
						repo = run.Repository.FullName
					} else if len(d.config.Repos) == 1 {
						// Local-only workflow (no runs yet) with a single configured repo.
						repo = d.config.Repos[0]
					} else {
						d.PendingMessage = "cannot dispatch: no runs for this workflow on this branch"
						return d, clearMsg()
					}
					d.dispatchDialog.Open(repo, file, d.selectedBranch())
				}
			}
		}

	case key.Matches(msg, d.keys.Back):
		if d.activePanel > 0 {
			d.activePanel--
		}

	case key.Matches(msg, d.keys.Refresh):
		d.PendingMessage = "refreshing..."
		return d, loadRuns(d.client, d.config.Repos)
	}

	return d, nil
}

// ── Render methods ──────────────────────────────────────────────────────────

// View renders the complete dashboard (title + panels + help bar).
func (d Dashboard) View(width, height int, message string, loading bool) string {
	w, h := width, height
	if w == 0 {
		w = 80
	}
	if h == 0 {
		h = 24
	}

	bodyH := h - 3 // title + panel-headers + help
	// set minimum to avoid rendering issues when tiny
	if bodyH < 5 {
		bodyH = 5
	}

	const workflowW = 22
	const maxDetailW = 40
	detailW := min(maxDetailW, w*30/100)
	runsW := w - workflowW - detailW - 2 // 2 separators

	sep := lipgloss.NewStyle().
		Foreground(styles.ColorSubtle).
		Render(strings.Repeat("│\n", bodyH-1) + "│")

	body := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(workflowW).Height(bodyH).Render(d.renderWorkflows(workflowW, bodyH)),
		sep,
		lipgloss.NewStyle().Width(runsW).Height(bodyH).Render(d.renderList(runsW, bodyH)),
		sep,
		lipgloss.NewStyle().Width(detailW).Height(bodyH).Render(d.renderDetail(detailW)),
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		renderTitle(w),
		d.renderPanelHeaders(workflowW, runsW, detailW),
		body,
		d.renderHelpBar(w, message),
	)
}

func (d Dashboard) renderPanelHeaders(workflowW, runsW, detailW int) string {
	sep := lipgloss.NewStyle().Background(styles.ColorBgLight).Foreground(styles.ColorSubtle).Render("│")
	label := func(panel int, text string, w int) string {
		style := lipgloss.NewStyle().
			Width(w).
			Align(lipgloss.Center) // Center the label text
		if d.activePanel == panel {
			style = style.Bold(true).
				Background(styles.ColorPurple).Foreground(styles.ColorBg)
		} else {
			style = style.Background(styles.ColorBgLight).Foreground(styles.ColorWhite)
		}
		return style.Render(text)
	}
	return lipgloss.NewStyle().
		Width(workflowW + runsW + detailW + 2). // 2 for separators
		Align(lipgloss.Center).
		Render(
			lipgloss.JoinHorizontal(lipgloss.Top,
				label(panelWorkflows, "WORKFLOWS", workflowW),
				sep,
				label(panelRuns, "RUNS", runsW),
				sep,
				label(panelDetail, "DETAIL", detailW),
			),
		)
}

func (d Dashboard) renderWorkflows(width, height int) string {
	active := d.activePanel == panelWorkflows

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorGray)
	if active {
		headerStyle = headerStyle.Foreground(styles.ColorPurple)
	}
	selectedStyle := lipgloss.NewStyle().Bold(true).Background(styles.ColorBgLight).Foreground(styles.ColorWhite)

	var rows []string

	// ── REPO section (display only) ──────────────────────────────────────────
	rows = append(rows, headerStyle.Render("REPO"))
	repoDisplay := strings.Join(d.config.Repos, ", ")
	rows = append(rows, d.styles.Repo.Render(fmt.Sprintf("%-*s", width-2, gh.TruncateString(repoDisplay, width-2))))

	// Separator
	rows = append(rows, d.styles.Dimmed.Render(strings.Repeat("─", width-1)))

	// ── BRANCH section ──────────────────────────────────────────────────────
	rows = append(rows, headerStyle.Render("BRANCH"))

	branchDisplay := d.defaultBranch
	if d.branchIdx < len(d.availableBranches) {
		branchDisplay = d.availableBranches[d.branchIdx]
	}

	if d.branchPicker.Active() {
		for _, row := range d.branchPicker.View(d.styles, width) {
			rows = append(rows, row)
		}
	} else {
		text := fmt.Sprintf("%-*s", width-2, gh.TruncateString(branchDisplay, width-2))
		if d.workflowCursor == 0 && active {
			rows = append(rows, selectedStyle.Render(text))
		} else {
			rows = append(rows, d.styles.Branch.Render(text))
		}
	}

	// Separator
	rows = append(rows, d.styles.Dimmed.Render(strings.Repeat("─", width-1)))

	// ── NAME section ─────────────────────────────────────────────────────────
	rows = append(rows, headerStyle.Render("NAME"))

	if len(d.workflows) == 0 {
		rows = append(rows, d.styles.Dimmed.Render("loading..."))
		return strings.Join(rows, "\n")
	}

	// Check if we have a filename to pin at the bottom
	var filenameStr string
	if wfName := d.selectedWorkflow(); wfName != "" && wfName != workflowAll {
		filenameStr = d.workflowFiles[wfName]
	}

	branchSectionH := len(rows)
	workflowListH := height - branchSectionH
	if filenameStr != "" {
		workflowListH--
	}
	if workflowListH < 1 {
		workflowListH = 1
	}

	// wfCursor: index within d.workflows for scroll calculation
	// cursor scheme: 0=branch, 1..N=workflows[0..N-1]
	wfCursor := 0
	if d.workflowCursor > 0 {
		wfCursor = d.workflowCursor - 1
	}
	startIdx := 0
	if wfCursor >= workflowListH {
		startIdx = wfCursor - workflowListH + 1
	}
	endIdx := min(startIdx+workflowListH, len(d.workflows))

	for i := startIdx; i < endIdx; i++ {
		wfName := d.workflows[i]
		selected := (i + 1) == d.workflowCursor
		text := fmt.Sprintf("%-*s", width-2, gh.TruncateString(wfName, width-2))
		var row string
		switch {
		case selected && active:
			row = selectedStyle.Render(text)
		case selected:
			row = lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPurple).Render(text)
		default:
			row = d.styles.Normal.Render(text)
		}
		rows = append(rows, row)
	}

	if filenameStr != "" {
		for len(rows) < height-1 {
			rows = append(rows, "")
		}
		rows = append(rows, d.styles.Dimmed.Render(gh.TruncateString(filenameStr, width-2)))
	}

	return strings.Join(rows, "\n")
}

func (d Dashboard) renderList(width, height int) string {
	active := d.activePanel == panelRuns

	if len(d.filteredRuns) == 0 {
		return d.styles.Dimmed.Render("no workflow runs")
	}

	const colOk, colNum, colDur, colFile, colDispatched = 2, 6, 7, 14, 16
	colWorkflow := width - colOk - colNum - colDur - colFile - colDispatched - 5*colSep
	if colWorkflow < 10 {
		colWorkflow = 10
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorGray)
	if active {
		headerStyle = headerStyle.Foreground(styles.ColorPurple)
	}
	header := fmt.Sprintf("%-*s  %-*s  %-*s  %*s  %-*s  %-*s",
		colDispatched, "DISPATCHED (UTC)",
		colFile, "FILE",
		colWorkflow, "NAME",
		colNum, "RUN",
		colDur, "TIME",
		colOk, "OK",
	)
	rows := []string{headerStyle.Render(header)}

	listH := height - 2
	startIdx := 0
	if d.cursor >= listH {
		startIdx = d.cursor - listH + 1
	}
	endIdx := min(startIdx+listH, len(d.filteredRuns))

	for i := startIdx; i < endIdx; i++ {
		rows = append(rows, d.renderRunRow(d.filteredRuns[i], i == d.cursor, active,
			width, colWorkflow, colFile, colNum, colDur, colDispatched, colOk))
	}

	if len(d.filteredRuns) > listH {
		rows = append(rows, d.styles.Dimmed.Render(
			fmt.Sprintf(" %d/%d", d.cursor+1, len(d.filteredRuns))))
	}

	return strings.Join(rows, "\n")
}

func (d Dashboard) renderRunRow(run types.WorkflowRun, selected, active bool, width, colWorkflow, colFile, colNum, colDur, colDispatched, colOk int) string {
	icon := styles.StatusIcon(run.Status, run.Conclusion)
	iconS := fmt.Sprintf("%-*s", colOk, icon)
	wfS := fmt.Sprintf("%-*s", colWorkflow, gh.TruncateString(run.Name, colWorkflow))
	fileS := fmt.Sprintf("%-*s", colFile, gh.TruncateString(d.workflowFiles[run.Name], colFile))
	numS := fmt.Sprintf("%*s", colNum, fmt.Sprintf("#%d", run.RunNumber))
	durS := fmt.Sprintf("%-*s", colDur, gh.FormatDuration(int64(run.Duration().Seconds())))
	dispS := fmt.Sprintf("%-*s", colDispatched, run.CreatedAt.Format(timestampFormat))

	if selected && active {
		// Per-element styles with shared background so status/duration colors are preserved.
		bg := lipgloss.NewStyle().Background(styles.ColorBgLight)
		sep := bg.Render("  ")
		disp := d.styles.Dimmed.Background(styles.ColorBgLight).Render(dispS)
		wf := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorWhite).Background(styles.ColorBgLight).Render(wfS)
		file := d.styles.Dimmed.Background(styles.ColorBgLight).Render(fileS)
		num := lipgloss.NewStyle().Foreground(styles.ColorWhite).Background(styles.ColorBgLight).Render(numS)
		dur := d.styles.Duration.Background(styles.ColorBgLight).Render(durS)
		st := d.styles.StatusStyle(run.Status, run.Conclusion).Background(styles.ColorBgLight).Render(iconS)
		row := disp + sep + file + sep + wf + sep + num + sep + dur + sep + st
		// pad remaining width with background so the bar extends to the edge
		used := colDispatched + colWorkflow + colFile + colNum + colDur + colOk + 5*colSep
		if pad := width - used; pad > 0 {
			row += bg.Render(strings.Repeat(" ", pad))
		}
		return row
	}

	if selected {
		plainRow := dispS + "  " + fileS + "  " + wfS + "  " + numS + "  " + durS + "  " + iconS
		return lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPurple).Render(plainRow)
	}

	// normal: per-element styles
	st := d.styles.StatusStyle(run.Status, run.Conclusion).Render(iconS)
	file := d.styles.Dimmed.Render(fileS)
	dur := d.styles.Duration.Render(durS)
	disp := d.styles.Dimmed.Render(dispS)
	return disp + "  " + file + "  " + wfS + "  " + numS + "  " + dur + "  " + st
}

func (d Dashboard) renderDetail(width int) string {
	active := d.activePanel == panelDetail

	run := d.selectedRun()
	if run == nil {
		return d.styles.Dimmed.Render("no run selected")
	}

	var sb strings.Builder

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorGray)
	if active {
		headerStyle = headerStyle.Foreground(styles.ColorPurple)
	}
	sb.WriteString(headerStyle.Render(fmt.Sprintf("[#%d] %s", run.RunNumber, gh.TruncateString(run.Name, width-10))))
	sb.WriteString("\n\n")

	field := func(label, value string) {
		sb.WriteString(d.styles.Dimmed.Render(fmt.Sprintf("%-8s", label)))
		sb.WriteString(value + "\n")
	}

	sha := run.HeadSHA
	if len(sha) > 8 {
		sha = sha[:8]
	}
	statusStyle := d.styles.StatusStyle(run.Status, run.Conclusion)
	icon := styles.StatusIcon(run.Status, run.Conclusion)
	dur := gh.FormatDuration(int64(run.Duration().Seconds()))

	field("repo", d.styles.Repo.Render(gh.TruncateString(run.Repository.FullName, width-10)))
	field("branch", d.styles.Branch.Render(run.HeadBranch))
	field("commit", d.styles.Normal.Render(sha))
	field("status", statusStyle.Render(icon+" "+run.GetStatus())+"  "+d.styles.Duration.Render(dur))

	sb.WriteString("\n")

	jobsHeaderStyle := d.styles.Dimmed
	if active {
		jobsHeaderStyle = lipgloss.NewStyle().Foreground(styles.ColorPurple)
	}
	sb.WriteString(jobsHeaderStyle.Render("jobs") + "\n")

	if len(d.jobs) == 0 {
		sb.WriteString("  " + d.styles.Dimmed.Render("loading..."))
	} else {
		for i, job := range d.jobs {
			jIcon := styles.StatusIcon(job.Status, job.Conclusion)
			name := gh.TruncateString(job.Name, width-5)
			var line string
			switch {
			case i == d.jobCursor && active:
				line = lipgloss.NewStyle().Bold(true).Background(styles.ColorBgLight).Foreground(styles.ColorWhite).
					Render(fmt.Sprintf("  %s %s", jIcon, name))
			case i == d.jobCursor:
				line = lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPurple).
					Render(fmt.Sprintf("  %s %s", jIcon, name))
			default:
				jStyle := d.styles.StatusStyle(job.Status, job.Conclusion)
				line = "  " + jStyle.Render(jIcon) + " " + name
			}
			sb.WriteString(line + "\n")
		}
	}

	return sb.String()
}

func (d Dashboard) renderHelpBar(width int, message string) string {
	if d.confirmDialog.Active() {
		return d.confirmDialog.HelpView(d.styles)
	}

	if d.dispatchDialog.Active() {
		return d.dispatchDialog.HelpView(d.styles)
	}

	if d.branchPicker.Active() {
		return d.branchPicker.HelpView(d.styles)
	}

	if message != "" {
		return d.styles.Dimmed.Render(message)
	}

	var items []string
	if run := d.selectedRun(); run != nil {
		items = append(items, bindingHelp(d.styles, d.keys.Rerun))
		if run.Status == types.RunStatusInProgress {
			items = append(items, bindingHelp(d.styles, d.keys.Cancel))
		}
	}
	if d.activePanel == panelWorkflows {
		if wfName := d.selectedWorkflow(); wfName != "" && wfName != workflowAll {
			if _, ok := d.workflowFiles[wfName]; ok {
				items = append(items, bindingHelp(d.styles, d.keys.Dispatch))
			}
		}
	}
	items = append(items, bindingHelp(d.styles, d.keys.Open))

	left := strings.Join(items, "  ")
	right := bindingHelp(d.styles, d.keys.Quit)
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 2 {
		gap = 2
	}
	return left + strings.Repeat(" ", gap) + right
}
