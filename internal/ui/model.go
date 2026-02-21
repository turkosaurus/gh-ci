package ui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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

	screen Screen

	// data
	allRuns      []types.WorkflowRun
	filteredRuns []types.WorkflowRun
	jobs         []types.Job
	logs         string
	logJobName   string

	// navigation
	cursor    int
	logOffset int

	// filter/search
	filter      types.StatusFilter
	searchQuery string
	searching   bool
	textInput   textinput.Model

	// layout
	width  int
	height int

	// state
	loading    bool
	message    string
	panelOpen  bool

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
	tickMsg     time.Time
	clearMsgMsg struct{}
)

func NewModel(cfg *config.Config) Model {
	ti := textinput.New()
	ti.Placeholder = "search..."
	ti.CharLimit = 50
	return Model{
		config:    cfg,
		client:    gh.NewClient(),
		styles:    styles.DefaultStyles(),
		keys:      keys.DefaultKeyMap(),
		filter:    types.StatusAll,
		textInput: ti,
		loading:   true,
		panelOpen: true,
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
			runs, err := m.client.ListWorkflowRuns(repo, 20)
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

func clearMsg() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return clearMsgMsg{}
	})
}

func (m *Model) applyFilter() {
	m.filteredRuns = filterRuns(m.allRuns, m.filter, m.searchQuery)
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

func filterRuns(runs []types.WorkflowRun, filter types.StatusFilter, query string) []types.WorkflowRun {
	if filter == types.StatusAll && query == "" {
		return runs
	}
	var out []types.WorkflowRun
	for _, r := range runs {
		if filter != types.StatusAll {
			status := r.GetStatus()
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
				if r.Status != "in_progress" && r.Status != "queued" {
					continue
				}
			}
		}
		if query != "" {
			q := strings.ToLower(query)
			if !strings.Contains(strings.ToLower(r.Name), q) &&
				!strings.Contains(strings.ToLower(r.HeadBranch), q) &&
				!strings.Contains(strings.ToLower(r.Repository.FullName), q) {
				continue
			}
		}
		out = append(out, r)
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
		if m.searching {
			return m.handleSearch(msg)
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
			m.applyFilter()
			if run := m.selectedRun(); run != nil {
				cmds = append(cmds, m.loadJobs(run.Repository.FullName, run.ID))
			}
		}

	case jobsLoadedMsg:
		if msg.err == nil {
			m.jobs = msg.jobs
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

func (m Model) handleMainKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
			m.jobs = nil
			if run := m.selectedRun(); run != nil {
				return m, m.loadJobs(run.Repository.FullName, run.ID)
			}
		}

	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(m.filteredRuns)-1 {
			m.cursor++
			m.jobs = nil
			if run := m.selectedRun(); run != nil {
				return m, m.loadJobs(run.Repository.FullName, run.ID)
			}
		}

	case key.Matches(msg, m.keys.PageUp):
		m.cursor = max(0, m.cursor-10)
		m.jobs = nil
		if run := m.selectedRun(); run != nil {
			return m, m.loadJobs(run.Repository.FullName, run.ID)
		}

	case key.Matches(msg, m.keys.PageDown):
		if len(m.filteredRuns) > 0 {
			m.cursor = min(len(m.filteredRuns)-1, m.cursor+10)
		}
		m.jobs = nil
		if run := m.selectedRun(); run != nil {
			return m, m.loadJobs(run.Repository.FullName, run.ID)
		}

	case key.Matches(msg, m.keys.Top):
		m.cursor = 0
		m.jobs = nil
		if run := m.selectedRun(); run != nil {
			return m, m.loadJobs(run.Repository.FullName, run.ID)
		}

	case key.Matches(msg, m.keys.Bottom):
		if len(m.filteredRuns) > 0 {
			m.cursor = len(m.filteredRuns) - 1
		}
		m.jobs = nil
		if run := m.selectedRun(); run != nil {
			return m, m.loadJobs(run.Repository.FullName, run.ID)
		}

	case key.Matches(msg, m.keys.Logs):
		if len(m.jobs) > 0 {
			if run := m.selectedRun(); run != nil {
				m.message = "loading logs..."
				return m, m.loadLogs(run.Repository.FullName, m.jobs[0].ID, m.jobs[0].Name)
			}
		}

	case key.Matches(msg, m.keys.Open):
		if run := m.selectedRun(); run != nil {
			m.client.OpenInBrowser(run.HTMLURL)
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

	case key.Matches(msg, m.keys.Filter):
		m.searching = true
		m.textInput.SetValue(m.searchQuery)
		m.textInput.Focus()
		return m, textinput.Blink

	case key.Matches(msg, m.keys.Tab):
		switch m.filter {
		case types.StatusAll:
			m.filter = types.StatusFailed
		case types.StatusFailed:
			m.filter = types.StatusInProgress
		case types.StatusInProgress:
			m.filter = types.StatusSuccess
		case types.StatusSuccess:
			m.filter = types.StatusAll
		}
		m.applyFilter()

	case key.Matches(msg, m.keys.Back):
		m.filter = types.StatusAll
		m.searchQuery = ""
		m.textInput.SetValue("")
		m.applyFilter()

	case key.Matches(msg, m.keys.Refresh):
		m.message = "refreshing..."
		return m, m.loadRuns()

	case key.Matches(msg, m.keys.Panel):
		m.panelOpen = !m.panelOpen
	}

	return m, nil
}

func (m Model) handleLogsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	logLines := strings.Split(m.logs, "\n")
	visibleLines := m.height - 4

	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Back), msg.String() == "h":
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
