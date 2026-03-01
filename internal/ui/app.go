package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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

// Message represents a status message with a type
type Message struct {
	text      string
	msgType   styles.MessageType
}

func newMessage(text string, msgType styles.MessageType) Message {
	return Message{text: text, msgType: msgType}
}

// messageConfirming creates a message for immediate user action feedback (purple).
// Used for: "re-running...", "dispatching...", etc. - confirmation that action is queued.
func messageConfirming(text string) Message {
	return newMessage(text, styles.MessageTypeInfo)
}

// messageCompleted creates a message for async operation completion (green).
// Used for: "re-run triggered", "workflow cancelled", "dispatched ...", etc. - action finished.
func messageCompleted(text string) Message {
	return newMessage(text, styles.MessageTypeSuccess)
}

// messageError creates an error message (red).
func messageError(text string) Message {
	return newMessage(text, styles.MessageTypeError)
}

// App is the top-level tea.Model.
type App struct {
	config *config.Config
	client gh.Client
	styles styles.Styles
	keys   keys.KeyMap

	screen screen

	runs      *Fetchable[types.RunMap]
	localDefs *Fetchable[[]types.WorkflowDef]

	workflowFiles map[string]string
	defaultBranch string
	localBranch   string

	width, height int
	message       Message

	dashboard Dashboard
	logViewer LogViewer
}

func NewApp(cfg *config.Config) App {
	s := styles.DefaultStyles()
	k := keys.DefaultKeyMap()
	runs := &Fetchable[types.RunMap]{Fetching: true}
	client := gh.NewClient(runs)
	defaultBranch := cfg.PrimaryBranch
	localBranch := gitBranch()
	return App{
		config:        cfg,
		client:        client,
		styles:        s,
		keys:          k,
		runs:          runs,
		localDefs:     &Fetchable[[]types.WorkflowDef]{Fetching: true},
		workflowFiles: make(map[string]string),
		defaultBranch: defaultBranch,
		localBranch:   localBranch,
		dashboard:     NewDashboard(cfg, client, runs, s, k, defaultBranch, localBranch),
		logViewer:     NewLogViewer(s, k, cfg.LogContext),
	}
}

func (a App) Init() tea.Cmd {
	return tea.Batch(
		loadLocalDefs(),
		refreshRuns(a.client, a.config.Repos, time.Time{}),
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
				a.message = messageConfirming(a.dashboard.PendingMessage)
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

	case runsUpdatedMsg:
		if msg.err != nil {
			a.runs.SetError(msg.err)
			if !a.runs.HasData() {
				a.message = messageError("error: " + msg.err.Error())
			}
		} else {
			if msg.incremental {
				// Merge updated runs into existing RunMap
				merged := mergeRunMap(a.runs.Data, flattenRunMap(msg.runs))
				a.runs.SetData(merged)
			} else {
				// Replace entire RunMap (full fetch)
				a.runs.SetData(msg.runs)
			}
			a.deriveWorkflowFiles()
			cmd := a.dashboard.SetRuns(a.runs.Data, a.localDefs.Data, a.workflowFiles)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case reposDiscoveredMsg:
		a.dashboard.nearbyRepos = msg.repos
		cmd := a.dashboard.repoPicker.Open(a.dashboard.repoPickerNames())
		cmds = append(cmds, cmd)

	case jobsLoadedMsg:
		if msg.err == nil {
			a.dashboard.SetJobs(msg.jobs)
		}

	case logsLoadedMsg:
		a.message = messageConfirming("")
		if msg.err != nil {
			a.message = messageError("error loading logs: " + msg.err.Error())
		} else {
			a.logViewer.SetLogs(msg.logs, msg.jobName)
			a.screen = screenLogs
		}

	case actionResultMsg:
		if msg.err != nil {
			a.message = messageError("error: " + msg.err.Error())
		} else {
			a.message = messageCompleted(msg.message)
		}
		cmds = append(cmds, clearMsg(time.Duration(a.config.MsgTimeout)*time.Second), refreshRuns(a.client, a.config.Repos, time.Time{}))

	case dispatchResultMsg:
		if msg.err != nil {
			a.message = messageError("error: " + msg.err.Error())
		} else {
			a.message = messageCompleted(msg.message)
		}
		cmds = append(cmds, clearMsg(time.Duration(a.config.MsgTimeout)*time.Second), refreshRuns(a.client, a.config.Repos, time.Time{}))

	case tickMsg:
		a.runs.SetFetching()
		cmds = append(cmds, refreshRuns(a.client, a.config.Repos, a.runs.FetchedAt()), tick(a.config.RefreshInterval))

	case backToMainMsg:
		a.screen = screenDashboard

	case clearMsgMsg:
		a.message = messageConfirming("")
	}

	return a, tea.Batch(cmds...)
}

const (
	minWidth  = 60
	minHeight = 14
)

func (a App) View() string {
	if a.width > 0 && a.height > 0 && (a.width < minWidth || a.height < minHeight) {
		msg := fmt.Sprintf("terminal too small (%dx%d, need %dx%d)", a.width, a.height, minWidth, minHeight)
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, msg)
	}
	if a.screen == screenLogs {
		return a.logViewer.View(a.width, a.height)
	}
	return a.dashboard.View(a.width, a.height, a.message, a.runs.IsFetching())
}

// deriveWorkflowFiles populates workflowFiles (name → filename) from
// run paths and local defs.
func (a *App) deriveWorkflowFiles() {
	for _, repoMap := range a.runs.Data {
		for _, branchRuns := range repoMap {
			for _, r := range branchRuns {
				if r.Path == "" {
					continue
				}
				if _, ok := a.workflowFiles[r.Name]; ok {
					continue
				}
				parts := strings.Split(r.Path, "/")
				a.workflowFiles[r.Name] = parts[len(parts)-1]
			}
		}
	}
	for _, def := range a.localDefs.Data {
		if _, ok := a.workflowFiles[def.Name]; !ok {
			a.workflowFiles[def.Name] = def.File
		}
	}
}
