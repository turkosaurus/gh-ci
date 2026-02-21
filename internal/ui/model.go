package ui

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"gopkg.in/yaml.v3"

	"github.com/turkosaurus/gh-ci/internal/config"
	"github.com/turkosaurus/gh-ci/internal/gh"
	"github.com/turkosaurus/gh-ci/internal/types"
	"github.com/turkosaurus/gh-ci/internal/ui/keys"
	"github.com/turkosaurus/gh-ci/internal/ui/styles"
)

type Screen int

const (
	ScreenMain Screen = iota
	ScreenLogs
)

type Model struct {
	config *config.Config
	client *gh.Client
	styles styles.Styles
	keys   keys.KeyMap

	screen      Screen
	activePanel int // 0=workflows, 1=runs, 2=detail

	// data
	allRuns          []types.WorkflowRun
	filteredRuns     []types.WorkflowRun
	workflows        []string // flat: ["all", "fast", "slow"]
	availableBranches []string // ["", "main", "feat/x", ...] — "" = all branches
	branchIdx        int      // index into availableBranches; 0 = all branches
	jobs             []types.Job
	logs         string
	logJobName   string

	// local workflow definitions discovered from .github/workflows/
	localDefs []types.WorkflowDef

	// workflow filename cache: workflow name → filename (e.g. "fast" → "fast.yaml")
	workflowFiles map[string]string

	// dispatch confirmation state
	dispatchConfirming bool
	dispatchRepo       string
	dispatchFile       string
	dispatchRef        string

	// navigation
	workflowCursor int
	cursor         int
	jobCursor      int
	logOffset      int

	// search
	searchQuery string
	searching   bool
	textInput   textinput.Model

	// branch selection
	branchSelecting        bool
	branchInput            textinput.Model
	branchSuggestionCursor int

	// layout
	width  int
	height int

	// state
	loading       bool
	message       string
	defaultBranch string

	// rerun confirmation
	confirming  bool
	confirmRepo string
	confirmID   int64
}

type (
	runsLoadedMsg struct {
		runs []types.WorkflowRun
		err  error
	}
	jobsLoadedMsg struct {
		jobs []types.Job
		err  error
	}
	logsLoadedMsg struct {
		logs    string
		jobName string
		err     error
	}
	actionResultMsg struct {
		message string
		err     error
	}
	dispatchResultMsg struct {
		message string
		err     error
	}
	tickMsg     time.Time
	clearMsgMsg struct{}

)

func currentGitBranch() string {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func scanLocalWorkflows() []types.WorkflowDef {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return nil
	}
	root := strings.TrimSpace(string(out))

	var defs []types.WorkflowDef
	for _, pattern := range []string{"*.yaml", "*.yml"} {
		matches, _ := filepath.Glob(filepath.Join(root, ".github", "workflows", pattern))
		for _, path := range matches {
			data, err := os.ReadFile(path)
			if err != nil {
				continue
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
		}
	}
	return defs
}

func NewModel(cfg *config.Config) Model {
	ti := textinput.New()
	ti.Placeholder = "search..."
	ti.CharLimit = 50
	bi := textinput.New()
	bi.Placeholder = "filter branches..."
	bi.CharLimit = 100
	return Model{
		config:        cfg,
		client:        gh.NewClient(),
		styles:        styles.DefaultStyles(),
		keys:          keys.DefaultKeyMap(),
		textInput:     ti,
		branchInput:   bi,
		loading:       true,
		localDefs:     scanLocalWorkflows(),
		workflowFiles: make(map[string]string),
		defaultBranch: currentGitBranch(),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.loadRuns(), m.tick())
}

func (m Model) tick() tea.Cmd {
	return tea.Tick(time.Duration(m.config.RefreshInterval)*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) loadRuns() tea.Cmd {
	return func() tea.Msg {
		var all []types.WorkflowRun
		for _, repo := range m.config.Repos {
			runs, err := m.client.ListWorkflowRuns(repo, 100)
			if err != nil {
				return runsLoadedMsg{err: err}
			}
			all = append(all, runs...)
		}
		return runsLoadedMsg{runs: all}
	}
}

func (m Model) loadJobs(repo string, runID int64) tea.Cmd {
	return func() tea.Msg {
		jobs, err := m.client.GetJobs(repo, runID)
		if err != nil {
			return jobsLoadedMsg{err: err}
		}
		return jobsLoadedMsg{jobs: jobs}
	}
}

func (m Model) loadLogs(repo string, jobID int64, jobName string) tea.Cmd {
	return func() tea.Msg {
		logs, err := m.client.GetJobLogs(repo, jobID)
		if err != nil {
			return logsLoadedMsg{err: err, jobName: jobName}
		}
		return logsLoadedMsg{logs: logs, jobName: jobName}
	}
}

func (m Model) rerunWorkflow(repo string, runID int64, debug bool) tea.Cmd {
	return func() tea.Msg {
		err := m.client.RerunWorkflow(repo, runID, debug)
		if err != nil {
			return actionResultMsg{err: err}
		}
		if debug {
			return actionResultMsg{message: "re-run triggered (debug logging enabled)"}
		}
		return actionResultMsg{message: "re-run triggered"}
	}
}

func (m Model) cancelWorkflow(repo string, runID int64) tea.Cmd {
	return func() tea.Msg {
		err := m.client.CancelWorkflow(repo, runID)
		if err != nil {
			return actionResultMsg{err: err}
		}
		return actionResultMsg{message: "workflow cancelled"}
	}
}

func (m Model) runDispatch(repo, file, ref string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.DispatchWorkflow(repo, file, ref)
		if err != nil {
			return dispatchResultMsg{err: err}
		}
		return dispatchResultMsg{message: "dispatched " + file + " on " + ref}
	}
}

func (m Model) repoForWorkflow(name string) string {
	for _, r := range m.allRuns {
		if r.Name == name {
			return r.Repository.FullName
		}
	}
	if len(m.config.Repos) > 0 {
		return m.config.Repos[0]
	}
	return ""
}

func clearMsg() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return clearMsgMsg{}
	})
}

// deriveWorkflows collects unique workflow names (sorted, prefixed with "all")
// and unique branch names (sorted, prefixed with "" meaning all branches).
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
	workflows = append([]string{"all"}, workflows...)

	for b := range brSeen {
		branches = append(branches, b)
	}
	sort.Strings(branches)
	branches = append([]string{""}, branches...)
	return
}

func (m Model) filteredBranches() []string {
	q := strings.ToLower(m.branchInput.Value())
	var out []string
	for _, b := range m.availableBranches {
		display := b
		if b == "" {
			display = "all branches"
		}
		if q == "" || strings.Contains(strings.ToLower(display), q) {
			out = append(out, b)
		}
	}
	return out
}

func (m Model) selectedBranch() string {
	if m.branchIdx > 0 && m.branchIdx < len(m.availableBranches) {
		return m.availableBranches[m.branchIdx]
	}
	return m.defaultBranch
}

func (m *Model) applyFilter() {
	runs := m.allRuns

	// Apply branch filter
	if m.branchIdx > 0 && m.branchIdx < len(m.availableBranches) {
		branch := m.availableBranches[m.branchIdx]
		var br []types.WorkflowRun
		for _, r := range runs {
			if r.HeadBranch == branch {
				br = append(br, r)
			}
		}
		runs = br
	}

	// Apply workflow filter (workflowCursor 0 = branch cell; 1..N = workflows[0..N-1])
	if m.workflowCursor > 0 && m.workflowCursor <= len(m.workflows) {
		wfName := m.workflows[m.workflowCursor-1]
		if wfName != "all" {
			var wf []types.WorkflowRun
			for _, r := range runs {
				if r.Name == wfName {
					wf = append(wf, r)
				}
			}
			runs = wf
		}
	}

	m.filteredRuns = filterRuns(runs, m.searchQuery)
	if m.cursor >= len(m.filteredRuns) {
		m.cursor = max(0, len(m.filteredRuns)-1)
	}
}

func (m Model) selectedRun() *types.WorkflowRun {
	if m.cursor >= 0 && m.cursor < len(m.filteredRuns) {
		return &m.filteredRuns[m.cursor]
	}
	return nil
}

func filterRuns(runs []types.WorkflowRun, query string) []types.WorkflowRun {
	if query == "" {
		return runs
	}
	q := strings.ToLower(query)
	var out []types.WorkflowRun
	for _, r := range runs {
		if strings.Contains(strings.ToLower(r.Name), q) ||
			strings.Contains(strings.ToLower(r.HeadBranch), q) ||
			strings.Contains(strings.ToLower(r.Repository.FullName), q) {
			out = append(out, r)
		}
	}
	return out
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		if m.branchSelecting {
			return m.handleBranchSelect(msg)
		}
		if m.searching {
			return m.handleSearch(msg)
		}
		if m.dispatchConfirming {
			return m.handleDispatchConfirm(msg)
		}
		if m.confirming {
			return m.handleConfirm(msg)
		}
		switch m.screen {
		case ScreenMain:
			return m.handleMainKeys(msg)
		case ScreenLogs:
			return m.handleLogsKeys(msg)
		}

	case runsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.message = "error: " + msg.err.Error()
		} else {
			m.allRuns = msg.runs
			// Preserve selected branch by name; on first load default to current git branch
			prevBranch := ""
			if m.availableBranches == nil {
				prevBranch = m.defaultBranch
			} else if m.branchIdx < len(m.availableBranches) {
				prevBranch = m.availableBranches[m.branchIdx]
			}
			m.workflows, m.availableBranches = deriveWorkflows(m.allRuns, m.localDefs)
			// ensure defaultBranch is always present in availableBranches
			if m.defaultBranch != "" {
				found := false
				for _, b := range m.availableBranches {
					if b == m.defaultBranch {
						found = true
						break
					}
				}
				if !found {
					rest := append(m.availableBranches[1:], m.defaultBranch)
					sort.Strings(rest)
					m.availableBranches = append([]string{""}, rest...)
				}
			}
			m.branchIdx = 0
			for i, b := range m.availableBranches {
				if b == prevBranch {
					m.branchIdx = i
					break
				}
			}
			// workflowCursor 0 = branch cell; max = len(workflows)
			if m.workflowCursor > len(m.workflows) {
				m.workflowCursor = 0
			}
			m.applyFilter()
			if run := m.selectedRun(); run != nil {
				cmds = append(cmds, m.loadJobs(run.Repository.FullName, run.ID))
			}
			// populate workflowFiles (name → filename) from path field in run response
			for _, r := range m.allRuns {
				if r.Path == "" {
					continue
				}
				if _, ok := m.workflowFiles[r.Name]; ok {
					continue
				}
				parts := strings.Split(r.Path, "/")
				m.workflowFiles[r.Name] = parts[len(parts)-1]
			}
			// also populate from local defs (covers workflows with no runs)
			for _, def := range m.localDefs {
				if _, ok := m.workflowFiles[def.Name]; !ok {
					m.workflowFiles[def.Name] = def.File
				}
			}
		}

	case jobsLoadedMsg:
		if msg.err == nil {
			m.jobs = msg.jobs
			if m.jobCursor >= len(m.jobs) {
				m.jobCursor = 0
			}
		}

	case logsLoadedMsg:
		m.message = ""
		if msg.err != nil {
			m.message = "error loading logs: " + msg.err.Error()
		} else {
			m.logs = msg.logs
			m.logJobName = msg.jobName
			m.logOffset = 0
			m.screen = ScreenLogs
		}

	case actionResultMsg:
		if msg.err != nil {
			m.message = "error: " + msg.err.Error()
		} else {
			m.message = msg.message
		}
		cmds = append(cmds, clearMsg(), m.loadRuns())

	case dispatchResultMsg:
		if msg.err != nil {
			m.message = "error: " + msg.err.Error()
		} else {
			m.message = msg.message
		}
		cmds = append(cmds, clearMsg(), m.loadRuns())

	case tickMsg:
		cmds = append(cmds, m.loadRuns(), m.tick())

	case clearMsgMsg:
		m.message = ""

	}

	return m, tea.Batch(cmds...)
}

func (m Model) handleSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		m.searching = false
		m.textInput.Blur()
		return m, nil
	case tea.KeyEnter:
		m.searchQuery = m.textInput.Value()
		m.searching = false
		m.textInput.Blur()
		m.applyFilter()
		return m, nil
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m Model) handleBranchSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		m.branchSelecting = false
		m.branchInput.Blur()
		return m, nil
	case tea.KeyEnter:
		suggestions := m.filteredBranches()
		if len(suggestions) > 0 {
			idx := m.branchSuggestionCursor
			if idx >= len(suggestions) {
				idx = len(suggestions) - 1
			}
			chosen := suggestions[idx]
			for i, b := range m.availableBranches {
				if b == chosen {
					m.branchIdx = i
					break
				}
			}
		}
		m.branchSelecting = false
		m.branchInput.Blur()
		m.applyFilter()
		m.cursor = 0
		return m, nil
	case tea.KeyUp:
		if m.branchSuggestionCursor > 0 {
			m.branchSuggestionCursor--
		}
		return m, nil
	case tea.KeyDown:
		suggestions := m.filteredBranches()
		if m.branchSuggestionCursor < len(suggestions)-1 {
			m.branchSuggestionCursor++
		}
		return m, nil
	default:
		var cmd tea.Cmd
		m.branchInput, cmd = m.branchInput.Update(msg)
		m.branchSuggestionCursor = 0
		return m, cmd
	}
}

func (m Model) handleConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		m.confirming = false
		m.message = "re-running..."
		return m, m.rerunWorkflow(m.confirmRepo, m.confirmID, false)
	case "d":
		m.confirming = false
		m.message = "re-running with debug..."
		return m, m.rerunWorkflow(m.confirmRepo, m.confirmID, true)
	case "esc", "q":
		m.confirming = false
	}
	return m, nil
}

func (m Model) handleDispatchConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		m.dispatchConfirming = false
		m.message = "dispatching..."
		return m, m.runDispatch(m.dispatchRepo, m.dispatchFile, m.dispatchRef)
	case "esc", "q":
		m.dispatchConfirming = false
	}
	return m, nil
}

func (m Model) moveCursor(delta int) (tea.Model, tea.Cmd) {
	switch m.activePanel {
	case 0:
		n := m.workflowCursor + delta
		if n >= 0 && n <= len(m.workflows) {
			m.workflowCursor = n
			m.applyFilter()
			m.cursor = 0
			m.jobs = nil
			m.jobCursor = 0
			if run := m.selectedRun(); run != nil {
				return m, m.loadJobs(run.Repository.FullName, run.ID)
			}
		}
	case 1:
		n := m.cursor + delta
		if n >= 0 && n < len(m.filteredRuns) {
			m.cursor = n
			m.jobs = nil
			m.jobCursor = 0
			if run := m.selectedRun(); run != nil {
				return m, m.loadJobs(run.Repository.FullName, run.ID)
			}
		}
	case 2:
		n := m.jobCursor + delta
		if n >= 0 && n < len(m.jobs) {
			m.jobCursor = n
		}
	}
	return m, nil
}

func (m Model) moveCursorPage(dir int) (tea.Model, tea.Cmd) {
	const pageSize = 10
	switch m.activePanel {
	case 0:
		n := max(0, min(len(m.workflows), m.workflowCursor+dir*pageSize))
		if n != m.workflowCursor {
			m.workflowCursor = n
			m.applyFilter()
			m.cursor = 0
			m.jobs = nil
			m.jobCursor = 0
			if run := m.selectedRun(); run != nil {
				return m, m.loadJobs(run.Repository.FullName, run.ID)
			}
		}
	case 1:
		n := max(0, min(len(m.filteredRuns)-1, m.cursor+dir*pageSize))
		if n != m.cursor {
			m.cursor = n
			m.jobs = nil
			m.jobCursor = 0
			if run := m.selectedRun(); run != nil {
				return m, m.loadJobs(run.Repository.FullName, run.ID)
			}
		}
	}
	return m, nil
}

func (m Model) moveCursorEdge(top bool) (tea.Model, tea.Cmd) {
	switch m.activePanel {
	case 0:
		if top {
			m.workflowCursor = 0
		} else {
			m.workflowCursor = len(m.workflows)
		}
		m.applyFilter()
		m.cursor = 0
		m.jobs = nil
		m.jobCursor = 0
		if run := m.selectedRun(); run != nil {
			return m, m.loadJobs(run.Repository.FullName, run.ID)
		}
	case 1:
		if top {
			m.cursor = 0
		} else {
			m.cursor = max(0, len(m.filteredRuns)-1)
		}
		m.jobs = nil
		m.jobCursor = 0
		if run := m.selectedRun(); run != nil {
			return m, m.loadJobs(run.Repository.FullName, run.ID)
		}
	case 2:
		if top {
			m.jobCursor = 0
		} else {
			m.jobCursor = max(0, len(m.jobs)-1)
		}
	}
	return m, nil
}

func (m Model) openURL() string {
	switch m.activePanel {
	case 0:
		wfName := ""
		if m.workflowCursor > 0 && m.workflowCursor <= len(m.workflows) {
			wfName = m.workflows[m.workflowCursor-1]
		}
		if wfName == "" || wfName == "all" {
			// branch cell or "all" — open repo actions page
			if run := m.selectedRun(); run != nil {
				return run.Repository.HTMLURL + "/actions"
			}
			if len(m.config.Repos) > 0 {
				return "https://github.com/" + m.config.Repos[0] + "/actions"
			}
		} else {
			// specific workflow — open its actions/workflows page
			for _, r := range m.allRuns {
				if r.Name == wfName {
					if filename, ok := m.workflowFiles[wfName]; ok {
						return r.Repository.HTMLURL + "/actions/workflows/" + filename
					}
					return r.HTMLURL
				}
			}
		}
	case 1:
		if run := m.selectedRun(); run != nil {
			return run.HTMLURL
		}
	case 2:
		if m.jobCursor < len(m.jobs) {
			return m.jobs[m.jobCursor].HTMLURL
		}
	}
	return ""
}

func (m Model) handleMainKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Up):
		return m.moveCursor(-1)

	case key.Matches(msg, m.keys.Down):
		return m.moveCursor(1)

	case key.Matches(msg, m.keys.PageUp):
		return m.moveCursorPage(-1)

	case key.Matches(msg, m.keys.PageDown):
		return m.moveCursorPage(1)

	case key.Matches(msg, m.keys.Top):
		return m.moveCursorEdge(true)

	case key.Matches(msg, m.keys.Bottom):
		return m.moveCursorEdge(false)

	case key.Matches(msg, m.keys.Logs): // l — move right between panels
		if m.activePanel < 2 {
			m.activePanel++
		}

	case key.Matches(msg, m.keys.Enter):
		if m.activePanel == 0 && m.workflowCursor == 0 {
			cur := ""
			if m.branchIdx > 0 && m.branchIdx < len(m.availableBranches) {
				cur = m.availableBranches[m.branchIdx]
			}
			m.branchInput.SetValue(cur)
			m.branchInput.Focus()
			m.branchSuggestionCursor = 0
			m.branchSelecting = true
			return m, textinput.Blink
		} else if m.activePanel == 2 && m.jobCursor < len(m.jobs) {
			// open logs from detail panel
			if run := m.selectedRun(); run != nil {
				job := m.jobs[m.jobCursor]
				m.message = "loading logs..."
				return m, m.loadLogs(run.Repository.FullName, job.ID, job.Name)
			}
		}

	case msg.String() == "h": // move left between panels
		if m.activePanel > 0 {
			m.activePanel--
		}

	case key.Matches(msg, m.keys.Open):
		if url := m.openURL(); url != "" {
			m.client.OpenInBrowser(url)
		}

	case key.Matches(msg, m.keys.Rerun):
		if run := m.selectedRun(); run != nil {
			m.confirming = true
			m.confirmRepo = run.Repository.FullName
			m.confirmID = run.ID
		}

	case key.Matches(msg, m.keys.Cancel):
		if run := m.selectedRun(); run != nil && run.Status == "in_progress" {
			m.message = "cancelling..."
			return m, m.cancelWorkflow(run.Repository.FullName, run.ID)
		}

	case key.Matches(msg, m.keys.Dispatch):
		if m.activePanel == 0 && m.workflowCursor > 0 && m.workflowCursor <= len(m.workflows) {
			wfName := m.workflows[m.workflowCursor-1]
			if wfName != "all" {
				if file, ok := m.workflowFiles[wfName]; ok {
					m.dispatchConfirming = true
					m.dispatchFile = file
					m.dispatchRef = m.selectedBranch()
					m.dispatchRepo = m.repoForWorkflow(wfName)
				}
			}
		}

	case key.Matches(msg, m.keys.Filter):
		m.searching = true
		m.textInput.SetValue(m.searchQuery)
		m.textInput.Focus()
		return m, textinput.Blink

	case key.Matches(msg, m.keys.Back):
		m.searchQuery = ""
		m.textInput.SetValue("")
		m.applyFilter()

	case key.Matches(msg, m.keys.Refresh):
		m.message = "refreshing..."
		return m, m.loadRuns()
	}

	return m, nil
}

func (m Model) handleLogsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	logLines := strings.Split(m.logs, "\n")
	visibleLines := m.height - 4

	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Back), msg.String() == "h", msg.Type == tea.KeyBackspace:
		m.screen = ScreenMain

	case key.Matches(msg, m.keys.Up):
		if m.logOffset > 0 {
			m.logOffset--
		}

	case key.Matches(msg, m.keys.Down):
		if maxOffset := len(logLines) - visibleLines; m.logOffset < maxOffset {
			m.logOffset++
		}

	case key.Matches(msg, m.keys.PageUp):
		m.logOffset = max(0, m.logOffset-visibleLines)

	case key.Matches(msg, m.keys.PageDown):
		m.logOffset = min(max(0, len(logLines)-visibleLines), m.logOffset+visibleLines)

	case key.Matches(msg, m.keys.HalfPageUp):
		m.logOffset = max(0, m.logOffset-visibleLines/2)

	case key.Matches(msg, m.keys.HalfPageDown):
		m.logOffset = min(max(0, len(logLines)-visibleLines), m.logOffset+visibleLines/2)

	case key.Matches(msg, m.keys.Top):
		m.logOffset = 0

	case key.Matches(msg, m.keys.Bottom):
		m.logOffset = max(0, len(logLines)-visibleLines)
	}

	return m, nil
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
