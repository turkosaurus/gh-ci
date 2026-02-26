package ui

import (
	"log/slog"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/turkosaurus/gh-ci/internal/gh"
	"github.com/turkosaurus/gh-ci/internal/types"
)

func loadLocalDefs() tea.Cmd {
	return func() tea.Msg {
		defs, err := scanLocalWorkflows()
		if err != nil {
			slog.Error("fetch local", "error", err)
		}
		slog.Debug("scanned local workflow definitions",
			"count", len(defs),
		)
		for _, def := range defs {
			slog.Debug("found workflow definition",
				"name", def.Name,
			)
		}

		return localDefsLoadedMsg{defs: defs, err: err}
	}
}

func loadRunsPartial(client gh.Client, repos []string) tea.Cmd {
	return func() tea.Msg {
		var all []types.WorkflowRun
		for _, repo := range repos {
			runs, err := client.ListWorkflowRuns(repo, 1)
			if err != nil {
				return runsPartialMsg{err: err}
			}
			all = append(all, runs...)
		}
		return runsPartialMsg{runs: all}
	}
}

func loadRuns(client gh.Client, repos []string) tea.Cmd {
	return func() tea.Msg {
		var all []types.WorkflowRun
		for _, repo := range repos {
			// TODO: paginate, but also filter by type and filter by date (max age, etc)
			runs, err := client.ListWorkflowRuns(repo, 10)
			if err != nil {
				return runsLoadedMsg{err: err}
			}
			all = append(all, runs...)
		}
		return runsLoadedMsg{runs: all}
	}
}

func loadJobs(client gh.Client, repo string, runID int64) tea.Cmd {
	return func() tea.Msg {
		jobs, err := client.GetJobs(repo, runID)
		if err != nil {
			return jobsLoadedMsg{err: err}
		}
		return jobsLoadedMsg{jobs: jobs}
	}
}

func loadLogs(client gh.Client, repo string, jobID int64, jobName string) tea.Cmd {
	return func() tea.Msg {
		logs, err := client.GetJobLogs(repo, jobID)
		if err != nil {
			return logsLoadedMsg{err: err, jobName: jobName}
		}
		return logsLoadedMsg{logs: logs, jobName: jobName}
	}
}

func rerunWorkflow(client gh.Client, repo string, runID int64, debug bool) tea.Cmd {
	return func() tea.Msg {
		err := client.RerunWorkflow(repo, runID, debug)
		if err != nil {
			return actionResultMsg{err: err}
		}
		if debug {
			return actionResultMsg{message: "re-run triggered (debug logging enabled)"}
		}
		return actionResultMsg{message: "re-run triggered"}
	}
}

func cancelWorkflow(client gh.Client, repo string, runID int64) tea.Cmd {
	return func() tea.Msg {
		err := client.CancelWorkflow(repo, runID)
		if err != nil {
			return actionResultMsg{err: err}
		}
		return actionResultMsg{message: "workflow cancelled"}
	}
}

func runDispatch(client gh.Client, repo, file, ref string) tea.Cmd {
	return func() tea.Msg {
		err := client.DispatchWorkflow(repo, file, ref)
		if err != nil {
			return dispatchResultMsg{err: err}
		}
		return dispatchResultMsg{message: "dispatched " + file + " on " + ref}
	}
}

func tick(interval int) tea.Cmd {
	return tea.Tick(time.Duration(interval)*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func clearMsg() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return clearMsgMsg{}
	})
}
