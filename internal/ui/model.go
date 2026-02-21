package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jay-418/gh-ci/internal/config"
	"github.com/jay-418/gh-ci/internal/gh"
	"github.com/jay-418/gh-ci/internal/types"
	"github.com/jay-418/gh-ci/internal/ui/components"
	"github.com/jay-418/gh-ci/internal/ui/keys"
	"github.com/jay-418/gh-ci/internal/ui/styles"
	"github.com/jay-418/gh-ci/internal/ui/views"
)

// View represents the current view
type View int

const (
	ViewMain View = iota
	ViewDetail
	ViewLogs
	ViewHelp
)

// Model is the main Bubble Tea model
type Model struct {
	config      *config.Config
	client      *gh.Client
	styles      styles.Styles
	keys        keys.KeyMap
	mainView    views.MainView
	detailView  views.DetailView
	textInput   textinput.Model
	currentView View
	allRuns     []types.WorkflowRun
	width       int
	height      int
	loading     bool
	err         error
	showFilter  bool
	message     string
}

// NewModel creates a new model
func NewModel(cfg *config.Config) Model {
	s := styles.DefaultStyles()
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.CharLimit = 50

	return Model{
		config:      cfg,
		client:      gh.NewClient(),
		styles:      s,
		keys:        keys.DefaultKeyMap(),
		mainView:    views.NewMainView(s),
		detailView:  views.NewDetailView(s),
		textInput:   ti,
		currentView: ViewMain,
		allRuns:     []types.WorkflowRun{},
		loading:     true,
	}
}

// Messages
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

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadRuns(),
		m.tick(),
	)
}

// tick returns a command that sends a tick message
func (m Model) tick() tea.Cmd {
	return tea.Tick(time.Duration(m.config.RefreshInterval)*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// loadRuns loads workflow runs from GitHub
func (m Model) loadRuns() tea.Cmd {
	return func() tea.Msg {
		var allRuns []types.WorkflowRun
		for _, repo := range m.config.Repos {
			runs, err := m.client.ListWorkflowRuns(repo, 20)
			if err != nil {
				return runsLoadedMsg{err: err}
			}
			allRuns = append(allRuns, runs...)
		}
		return runsLoadedMsg{runs: allRuns}
	}
}

// loadJobs loads jobs for a run
func (m Model) loadJobs(repo string, runID int64) tea.Cmd {
	return func() tea.Msg {
		jobs, err := m.client.GetJobs(repo, runID)
		if err != nil {
			return jobsLoadedMsg{err: err}
		}
		return jobsLoadedMsg{jobs: jobs}
	}
}

// loadLogs loads logs for a job
func (m Model) loadLogs(repo string, jobID int64, jobName string) tea.Cmd {
	return func() tea.Msg {
		logs, err := m.client.GetJobLogs(repo, jobID)
		if err != nil {
			return logsLoadedMsg{err: err, jobName: jobName}
		}
		return logsLoadedMsg{logs: logs, jobName: jobName}
	}
}

// rerunWorkflow re-runs a workflow
func (m Model) rerunWorkflow(repo string, runID int64) tea.Cmd {
	return func() tea.Msg {
		err := m.client.RerunWorkflow(repo, runID)
		if err != nil {
			return actionResultMsg{err: err}
		}
		return actionResultMsg{message: "Workflow re-run triggered"}
	}
}

// cancelWorkflow cancels a workflow
func (m Model) cancelWorkflow(repo string, runID int64) tea.Cmd {
	return func() tea.Msg {
		err := m.client.CancelWorkflow(repo, runID)
		if err != nil {
			return actionResultMsg{err: err}
		}
		return actionResultMsg{message: "Workflow cancelled"}
	}
}

// clearMessage returns a command that clears the message after a delay
func clearMessage() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return clearMsgMsg{}
	})
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.mainView.SetSize(msg.Width, msg.Height)
		m.detailView.SetSize(msg.Width, msg.Height)

	case tea.KeyMsg:
		// Handle filter input mode
		if m.showFilter {
			return m.handleFilterInput(msg)
		}
		return m.handleKeyPress(msg)

	case runsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.mainView.SetMessage("Error: " + msg.err.Error())
		} else {
			m.allRuns = msg.runs
			m.updateFilteredRuns()
		}

	case jobsLoadedMsg:
		if msg.err != nil {
			m.detailView.SetError(msg.err.Error())
		} else {
			m.detailView.SetJobs(msg.jobs)
		}

	case logsLoadedMsg:
		if msg.err != nil {
			m.detailView.LogViewer.SetError(msg.err.Error())
		} else {
			m.detailView.LogViewer.SetLogs(msg.logs, msg.jobName)
		}
		m.detailView.ShowLogs = true
		m.currentView = ViewLogs

	case actionResultMsg:
		if msg.err != nil {
			m.mainView.SetMessage("Error: " + msg.err.Error())
		} else {
			m.mainView.SetMessage(msg.message)
		}
		cmds = append(cmds, clearMessage(), m.loadRuns())

	case tickMsg:
		cmds = append(cmds, m.loadRuns(), m.tick())

	case clearMsgMsg:
		m.mainView.ClearMessage()
	}

	return m, tea.Batch(cmds...)
}

// handleFilterInput handles input when filter is active
func (m Model) handleFilterInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		m.showFilter = false
		m.textInput.Blur()
		return m, nil
	case tea.KeyEnter:
		m.mainView.SearchQuery = m.textInput.Value()
		m.showFilter = false
		m.textInput.Blur()
		m.updateFilteredRuns()
		return m, nil
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// handleKeyPress handles key presses
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.currentView {
	case ViewMain:
		return m.handleMainKeys(msg)
	case ViewDetail:
		return m.handleDetailKeys(msg)
	case ViewLogs:
		return m.handleLogsKeys(msg)
	case ViewHelp:
		return m.handleHelpKeys(msg)
	}
	return m, nil
}

// handleMainKeys handles key presses in main view
func (m Model) handleMainKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Up):
		m.mainView.RunList.MoveUp()

	case key.Matches(msg, m.keys.Down):
		m.mainView.RunList.MoveDown()

	case key.Matches(msg, m.keys.PageUp):
		m.mainView.RunList.PageUp(10)

	case key.Matches(msg, m.keys.PageDown):
		m.mainView.RunList.PageDown(10)

	case key.Matches(msg, m.keys.Top):
		m.mainView.RunList.GoToTop()

	case key.Matches(msg, m.keys.Bottom):
		m.mainView.RunList.GoToBottom()

	case key.Matches(msg, m.keys.Enter):
		run := m.mainView.RunList.SelectedRun()
		if run != nil {
			m.detailView.SetRun(run)
			m.currentView = ViewDetail
			return m, m.loadJobs(run.Repository.FullName, run.ID)
		}

	case key.Matches(msg, m.keys.Logs):
		run := m.mainView.RunList.SelectedRun()
		if run != nil {
			m.detailView.SetRun(run)
			m.detailView.LogViewer.SetLoading(true)
			m.currentView = ViewLogs
			// Load jobs first to get job ID
			return m, m.loadJobs(run.Repository.FullName, run.ID)
		}

	case key.Matches(msg, m.keys.Open):
		run := m.mainView.RunList.SelectedRun()
		if run != nil {
			m.client.OpenInBrowser(run.HTMLURL)
		}

	case key.Matches(msg, m.keys.Rerun):
		run := m.mainView.RunList.SelectedRun()
		if run != nil {
			m.mainView.SetMessage("Re-running workflow...")
			return m, m.rerunWorkflow(run.Repository.FullName, run.ID)
		}

	case key.Matches(msg, m.keys.Cancel):
		run := m.mainView.RunList.SelectedRun()
		if run != nil && run.Status == "in_progress" {
			m.mainView.SetMessage("Cancelling workflow...")
			return m, m.cancelWorkflow(run.Repository.FullName, run.ID)
		}

	case key.Matches(msg, m.keys.Filter):
		m.showFilter = true
		m.textInput.Focus()
		return m, textinput.Blink

	case key.Matches(msg, m.keys.Tab):
		m.mainView.CycleFilter()
		m.updateFilteredRuns()

	case key.Matches(msg, m.keys.Refresh):
		m.mainView.SetMessage("Refreshing...")
		return m, m.loadRuns()

	case key.Matches(msg, m.keys.Help):
		m.mainView.Help.Toggle()

	case key.Matches(msg, m.keys.Back):
		m.mainView.ClearFilter()
		m.updateFilteredRuns()
	}

	return m, nil
}

// handleDetailKeys handles key presses in detail view
func (m Model) handleDetailKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Back):
		m.currentView = ViewMain

	case key.Matches(msg, m.keys.Logs):
		// Load logs for first job
		if len(m.detailView.Jobs) > 0 {
			job := m.detailView.Jobs[0]
			m.detailView.LogViewer.SetLoading(true)
			m.detailView.ShowLogs = true
			m.currentView = ViewLogs
			return m, m.loadLogs(m.detailView.Run.Repository.FullName, job.ID, job.Name)
		}

	case key.Matches(msg, m.keys.Open):
		if m.detailView.Run != nil {
			m.client.OpenInBrowser(m.detailView.Run.HTMLURL)
		}

	case key.Matches(msg, m.keys.Rerun):
		if m.detailView.Run != nil {
			m.mainView.SetMessage("Re-running workflow...")
			return m, m.rerunWorkflow(m.detailView.Run.Repository.FullName, m.detailView.Run.ID)
		}

	case key.Matches(msg, m.keys.Cancel):
		if m.detailView.Run != nil && m.detailView.Run.Status == "in_progress" {
			m.mainView.SetMessage("Cancelling workflow...")
			return m, m.cancelWorkflow(m.detailView.Run.Repository.FullName, m.detailView.Run.ID)
		}

	case key.Matches(msg, m.keys.Help):
		m.detailView.Help.Toggle()
	}

	return m, nil
}

// handleLogsKeys handles key presses in logs view
func (m Model) handleLogsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Back):
		m.detailView.ShowLogs = false
		m.currentView = ViewDetail

	case key.Matches(msg, m.keys.Up):
		m.detailView.LogViewer.ScrollUp()

	case key.Matches(msg, m.keys.Down):
		m.detailView.LogViewer.ScrollDown()

	case key.Matches(msg, m.keys.PageUp):
		m.detailView.LogViewer.PageUp()

	case key.Matches(msg, m.keys.PageDown):
		m.detailView.LogViewer.PageDown()

	case key.Matches(msg, m.keys.HalfPageUp):
		m.detailView.LogViewer.HalfPageUp()

	case key.Matches(msg, m.keys.HalfPageDown):
		m.detailView.LogViewer.HalfPageDown()

	case key.Matches(msg, m.keys.Top):
		m.detailView.LogViewer.GoToTop()

	case key.Matches(msg, m.keys.Bottom):
		m.detailView.LogViewer.GoToBottom()

	case key.Matches(msg, m.keys.Help):
		m.detailView.Help.Toggle()
	}

	return m, nil
}

// handleHelpKeys handles key presses in help view
func (m Model) handleHelpKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Back), key.Matches(msg, m.keys.Help):
		m.currentView = ViewMain
	}
	return m, nil
}

// updateFilteredRuns updates the filtered runs in the view
func (m *Model) updateFilteredRuns() {
	filtered := components.FilterRuns(m.allRuns, m.mainView.StatusFilter, m.mainView.SearchQuery)
	m.mainView.RunList.SetRuns(filtered)
}

// View renders the model
func (m Model) View() string {
	if m.loading && len(m.allRuns) == 0 {
		return m.styles.App.Render(m.styles.Dimmed.Render("Loading workflow runs..."))
	}

	if m.showFilter {
		return m.renderWithFilter()
	}

	switch m.currentView {
	case ViewMain:
		return m.mainView.View()
	case ViewDetail, ViewLogs:
		return m.detailView.View()
	case ViewHelp:
		return m.mainView.Help.Render()
	}

	return m.mainView.View()
}

// renderWithFilter renders the view with filter input
func (m Model) renderWithFilter() string {
	view := m.mainView.View()
	filterInput := "\n" + m.styles.Dimmed.Render("Filter: ") + m.textInput.View()
	return view + filterInput
}
