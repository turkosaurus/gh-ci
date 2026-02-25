package ui

import (
	"time"

	"github.com/turkosaurus/gh-ci/internal/types"
)

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

// backToMainMsg signals that the log viewer wants to return to the dashboard.
type backToMainMsg struct{}
