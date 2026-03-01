package ui

import (
	"time"

	"github.com/turkosaurus/gh-ci/internal/config"
	"github.com/turkosaurus/gh-ci/internal/gh"
	"github.com/turkosaurus/gh-ci/internal/types"
	"github.com/turkosaurus/gh-ci/internal/ui/keys"
	"github.com/turkosaurus/gh-ci/internal/ui/picker"
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
	born time.Time

	activePanel    int
	workflowCursor int
	cursor         int
	jobCursor      int

	filteredRuns      []types.WorkflowRun
	workflows         []string
	availableBranches []string
	branchIdx         int
	jobs              []types.Job

	repoPicker     picker.Picker
	branchPicker   picker.Picker
	confirmDialog  ConfirmDialog
	dispatchDialog DispatchDialog
	helpModal      HelpModal

	nearbyRepos []config.RepoInfo

	PendingMessage string // App reads and clears after each Update

	config        *config.Config
	client        gh.Client
	runs          *Fetchable[types.RunMap]
	styles        styles.Styles
	keys          keys.KeyMap
	allRuns       types.RunMap
	localDefs     []types.WorkflowDef
	workflowFiles map[string]string
	defaultBranch string
	localBranch   string
	localRepo     string
}

