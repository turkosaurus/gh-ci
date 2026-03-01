package ui

import (
	"time"

	"github.com/turkosaurus/gh-ci/internal/config"
	"github.com/turkosaurus/gh-ci/internal/types"
)

type (
	runsUpdatedMsg struct {
		runs        types.RunMap
		err         error
		incremental bool
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
	localDefsLoadedMsg struct {
		defs []types.WorkflowDef
		err  error
	}
	reposDiscoveredMsg struct {
		repos []config.RepoInfo
	}
	tickMsg      time.Time
	clearMsgMsg  struct{}
	editFileMsg  struct {
		err error
	}
)

// backToMainMsg signals that the log viewer wants to return to the dashboard.
type backToMainMsg struct{}
