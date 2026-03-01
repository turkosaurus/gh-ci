package ui

import (
	"log/slog"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/turkosaurus/gh-ci/internal/config"
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

func discoverRepos() tea.Cmd {
	return func() tea.Msg {
		return reposDiscoveredMsg{repos: config.DiscoverRepos()}
	}
}

func refreshRuns(client gh.Client, repos []string) tea.Cmd {
	return func() tea.Msg {
		runMap := make(types.RunMap)
		for _, repo := range repos {
			repoRuns, err := client.ListWorkflowRuns(repo, time.Time{})
			if err != nil {
				return runsUpdatedMsg{err: err}
			}
			repoMap := make(map[string][]types.WorkflowRun)
			for _, r := range repoRuns {
				repoMap[r.HeadBranch] = append(repoMap[r.HeadBranch], r)
			}
			runMap[repo] = repoMap
		}
		return runsUpdatedMsg{runs: runMap}
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

func clearMsg(timeout time.Duration) tea.Cmd {
	return tea.Tick(timeout, func(t time.Time) tea.Msg {
		return clearMsgMsg{}
	})
}
