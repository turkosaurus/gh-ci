package ui

import (
	"fmt"
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

type screen int

const (
	screenDashboard screen = iota
	screenLogs
)

// override at build time
//
//	go build -ldflags "-X 'github.com/turkosaurus/gh-ci/internal/ui.Version=1.2.3'"
var Version string = "dev"

// App is the top-level tea.Model.
type App struct {
	config *config.Config
	client gh.Client
	styles styles.Styles
	keys   keys.KeyMap

	screen screen

	runs      Fetchable[[]types.WorkflowRun]
	localDefs Fetchable[[]types.WorkflowDef]

	workflowFiles map[string]string
	defaultBranch string
	localBranch   string

	width, height int
	message       string

	dashboard Dashboard
	logViewer LogViewer
}

func NewApp(cfg *config.Config) App {
	s := styles.DefaultStyles()
	k := keys.DefaultKeyMap()
	client := gh.NewClient()
	defaultBranch := cfg.DefaultPrimaryBranch
	localBranch := currentGitBranch()
	return App{
		config:        cfg,
		client:        client,
		styles:        s,
		keys:          k,
		runs:          Fetchable[[]types.WorkflowRun]{Fetching: true},
		localDefs:     Fetchable[[]types.WorkflowDef]{Fetching: true},
		workflowFiles: make(map[string]string),
		defaultBranch: defaultBranch,
		localBranch:   localBranch,
		dashboard:     NewDashboard(cfg, client, s, k, defaultBranch, localBranch),
		logViewer:     NewLogViewer(s, k),
	}
}

func (a App) Init() tea.Cmd {
	return tea.Batch(
		loadLocalDefs(),
		loadRunsPartial(a.client, a.config.Repos),
		tick(a.config.RefreshInterval),
	)
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height

	case tea.KeyMsg:
		switch a.screen {
		case screenDashboard:
			var cmd tea.Cmd
			a.dashboard, cmd = a.dashboard.Update(msg)
			if a.dashboard.PendingMessage != "" {
				a.message = a.dashboard.PendingMessage
				a.dashboard.PendingMessage = ""
			}
			return a, cmd
		case screenLogs:
			var cmd tea.Cmd
			a.logViewer, cmd = a.logViewer.Update(msg, a.height)
			return a, cmd
		}

	case localDefsLoadedMsg:
		if msg.err != nil {
			a.localDefs.SetError(msg.err)
		} else {
			a.localDefs.SetData(msg.defs)
		}
		a.deriveWorkflowFiles()
		cmd := a.dashboard.SetRuns(a.runs.Data, a.localDefs.Data, a.workflowFiles)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case runsPartialMsg:
		if msg.err != nil {
			a.runs.SetError(msg.err)
			a.message = "error: " + msg.err.Error()
		} else {
			a.runs.SetPartial(msg.runs)
			a.deriveWorkflowFiles()
			cmd := a.dashboard.SetRuns(a.runs.Data, a.localDefs.Data, a.workflowFiles)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			cmds = append(cmds, loadRuns(a.client, a.config.Repos))
		}

	case runsLoadedMsg:
		if msg.err != nil {
			a.runs.SetError(msg.err)
			if !a.runs.HasData() {
				a.message = "error: " + msg.err.Error()
			}
		} else {
			a.runs.SetData(msg.runs)
			a.deriveWorkflowFiles()
			cmd := a.dashboard.SetRuns(a.runs.Data, a.localDefs.Data, a.workflowFiles)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case jobsLoadedMsg:
		if msg.err == nil {
			a.dashboard.SetJobs(msg.jobs)
		}

	case logsLoadedMsg:
		a.message = ""
		if msg.err != nil {
			a.message = "error loading logs: " + msg.err.Error()
		} else {
			a.logViewer.SetLogs(msg.logs, msg.jobName)
			a.screen = screenLogs
		}

	case actionResultMsg:
		if msg.err != nil {
			a.message = "error: " + msg.err.Error()
		} else {
			a.message = msg.message
		}
		cmds = append(cmds, clearMsg(), loadRuns(a.client, a.config.Repos))

	case dispatchResultMsg:
		if msg.err != nil {
			a.message = "error: " + msg.err.Error()
		} else {
			a.message = msg.message
		}
		cmds = append(cmds, clearMsg(), loadRuns(a.client, a.config.Repos))

	case tickMsg:
		a.runs.SetFetching()
		cmds = append(cmds, loadRuns(a.client, a.config.Repos), tick(a.config.RefreshInterval))

	case backToMainMsg:
		a.screen = screenDashboard

	case clearMsgMsg:
		a.message = ""
	}

	return a, tea.Batch(cmds...)
}

func (a App) View() string {
	if a.screen == screenLogs {
		return a.logViewer.View(a.width, a.height)
	}
	return a.dashboard.View(a.width, a.height, a.message, a.runs.IsFetching())
}

// deriveWorkflowFiles populates workflowFiles (name → filename) from
// run paths and local defs.
func (a *App) deriveWorkflowFiles() {
	for _, r := range a.runs.Data {
		if r.Path == "" {
			continue
		}
		if _, ok := a.workflowFiles[r.Name]; ok {
			continue
		}
		parts := strings.Split(r.Path, "/")
		a.workflowFiles[r.Name] = parts[len(parts)-1]
	}
	for _, def := range a.localDefs.Data {
		if _, ok := a.workflowFiles[def.Name]; !ok {
			a.workflowFiles[def.Name] = def.File
		}
	}
}

// bindingHelp renders a single key binding as a "key  desc" help item.
func bindingHelp(s styles.Styles, b key.Binding) string {
	return s.HelpKey.Render(b.Help().Key) + " " + s.HelpDesc.Render(b.Help().Desc)
}

func renderTitle(width int) string {
	title := fmt.Sprintf("ci (%s)", Version)
	return lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPurple).Render(title)
}
