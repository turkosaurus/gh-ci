package gh

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/turkosaurus/gh-ci/internal/types"
)

var ghApiMaxRetry int = 3

// RunState is a read-only view of the runs cache that the client can
// inspect before deciding how to fetch (e.g. skip if data is fresh).
type RunState interface {
	FetchedAt() time.Time
	HasData() bool
}

// Client is the interface for GitHub API operations.
type Client interface {
	ListWorkflowRuns(repo string, since time.Time) ([]types.WorkflowRun, error)
	GetJobs(repo string, runID int64) ([]types.Job, error)
	GetJobLogs(repo string, jobID int64) (string, error)
	RerunWorkflow(repo string, runID int64, debug bool) error
	RerunFailedJobs(repo string, runID int64) error
	CancelWorkflow(repo string, runID int64) error
	DispatchWorkflow(repo, workflowFile, ref string) error
	OpenInBrowser(url string) error
}

// ghClient is the concrete gh-CLI-backed implementation of Client.
type ghClient struct {
	state RunState
}

// NewClient creates a new GitHub API client backed by the gh CLI.
// state gives the client a read-only view of cached run data so it can
// optimize future requests (e.g. check freshness before fetching).
func NewClient(state RunState) Client {
	return &ghClient{state: state}
}

// ListWorkflowRuns fetches workflow runs for a repository.
// Three-way dispatch:
// - No data: fetch only first page for speed (fast startup).
// - Has data, since is zero: fetch all pages (forced refresh).
// - Has data, since is set: fetch paginated with created filter from since-2h
//   (to catch in-progress runs that started before last fetch).
func (c *ghClient) ListWorkflowRuns(repo string, since time.Time) ([]types.WorkflowRun, error) {
	var allRuns []types.WorkflowRun

	// Just get the first page on initial run for speed
	if !c.state.HasData() {
		fmtString := "2006-01-02"
		t := time.Now().Add(-52 * 7 * 24 * time.Hour).Format(fmtString)
		tStr := fmt.Sprintf(">%s", t)
		endpoint := fmt.Sprintf("repos/%s/actions/runs?page=1&per_page=40&created=%s", repo, tStr)
		output, err := c.ghApiCall(http.MethodGet, endpoint)
		if err != nil {
			return nil, err
		}

		var response types.WorkflowRunsResponse
		if err := json.Unmarshal(output, &response); err != nil {
			return nil, fmt.Errorf("failed to parse workflow runs response: %w", err)
		}
		allRuns = append(allRuns, response.WorkflowRuns...)
		slog.Debug("fetch latest runs",
			"repo", repo,
			"count", len(response.WorkflowRuns),
		)
		return allRuns, nil
	}

	// Build the created filter based on since parameter
	tStr := ""
	if !since.IsZero() {
		// Subtract 2h overlap to catch in-progress runs from before last fetch
		cutoff := since.Add(-2 * time.Hour).UTC().Format(time.RFC3339)
		tStr = fmt.Sprintf("&created=>%s", cutoff)
		slog.Debug("incremental fetch",
			"repo", repo,
			"since", since,
			"cutoff", cutoff,
		)
	} else {
		// Full fetch: use 52-week lookback
		fmtString := "2006-01-02"
		t := time.Now().Add(-52 * 7 * 24 * time.Hour).Format(fmtString)
		tStr = fmt.Sprintf("&created=>%s", t)
		slog.Debug("full fetch", "repo", repo)
	}

	// On subsequent fetches, paginate through all results
	endpoint := fmt.Sprintf("repos/%s/actions/runs?per_page=40%s", repo, tStr)
	output, err := c.ghApiPaginated(http.MethodGet, endpoint)
	if err != nil {
		return nil, err
	}

	var response types.WorkflowRunsResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse paginated workflow runs: %w", err)
	}
	allRuns = append(allRuns, response.WorkflowRuns...)
	slog.Debug("fetch runs",
		"repo", repo,
		"count", len(allRuns),
		"incremental", !since.IsZero(),
	)
	return allRuns, nil
}

// GetJobs fetches jobs for a workflow run
func (c *ghClient) GetJobs(repo string, runID int64) ([]types.Job, error) {
	endpoint := fmt.Sprintf("repos/%s/actions/runs/%d/jobs", repo, runID)
	output, err := c.ghApiCall(http.MethodGet, endpoint)
	if err != nil {
		return nil, err
	}

	var response types.JobsResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return response.Jobs, nil
}

// GetJobLogs fetches logs for a specific job
func (c *ghClient) GetJobLogs(repo string, jobID int64) (string, error) {
	endpoint := fmt.Sprintf("repos/%s/actions/jobs/%d/logs", repo, jobID)
	output, err := c.ghApiCall(http.MethodGet, endpoint)
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// RerunWorkflow re-runs a workflow, optionally with debug logging enabled
func (c *ghClient) RerunWorkflow(repo string, runID int64, debug bool) error {
	endpoint := fmt.Sprintf("repos/%s/actions/runs/%d/rerun", repo, runID)
	var extra []string
	if debug {
		extra = []string{"-F", "enable_debug_logging=true"}
	}
	_, err := c.ghApiCall(http.MethodPost, endpoint, extra...)
	return err
}

// RerunFailedJobs re-runs only failed jobs in a workflow
func (c *ghClient) RerunFailedJobs(repo string, runID int64) error {
	endpoint := fmt.Sprintf("repos/%s/actions/runs/%d/rerun-failed-jobs", repo, runID)
	_, err := c.ghApiCall(http.MethodPost, endpoint)
	return err
}

// CancelWorkflow cancels a running workflow
func (c *ghClient) CancelWorkflow(repo string, runID int64) error {
	endpoint := fmt.Sprintf("repos/%s/actions/runs/%d/cancel", repo, runID)
	_, err := c.ghApiCall(http.MethodPost, endpoint)
	return err
}

// DispatchWorkflow triggers a workflow_dispatch event on the given ref.
// workflowFile is the filename, e.g. "ci.yaml".
func (c *ghClient) DispatchWorkflow(repo, workflowFile, ref string) error {
	endpoint := fmt.Sprintf("repos/%s/actions/workflows/%s/dispatches", repo, workflowFile)
	_, err := c.ghApiCall(http.MethodPost, endpoint, "-f", "ref="+ref)
	if err != nil {
		msg := err.Error()
		lower := strings.ToLower(msg)
		if strings.Contains(lower, "404") ||
			strings.Contains(lower, "not found") ||
			strings.Contains(lower, "no workflow") ||
			strings.Contains(lower, "could not find") {
			return fmt.Errorf("%s\nhint: workflow file must exist on the default branch to be dispatched", msg)
		}
		return err
	}
	return nil
}

// OpenInBrowser opens a URL in the default browser
func (c *ghClient) OpenInBrowser(url string) error {
	cmd := exec.Command("gh", "browse", "--url", url)
	// Use open command on macOS, xdg-open on Linux
	cmd = exec.Command("open", url)
	return cmd.Start()
}

// ghApiCall makes an API call using the gh CLI
func (c *ghClient) ghApiCall(method, endpoint string, extraArgs ...string) ([]byte, error) {
	args := append([]string{"api", "-X", method, endpoint}, extraArgs...)
	cmd := exec.Command("gh", args...)
	var err error
	var output []byte
	for i := range ghApiMaxRetry {
		output, err = cmd.Output()
		if err == nil {
			// slog.Debug("api called",
			// 	"method", method,
			// 	"endpoint", endpoint,
			// 	"extraArgs", extraArgs,
			// )
			return output, nil
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := string(exitErr.Stderr)
			if strings.Contains(stderr, "rate limit") || strings.Contains(stderr, "abuse detection") {
				waitTime := time.Duration((i+1)*2) * time.Second
				slog.Warn("rate limited",
					"method", method,
					"endpoint", endpoint,
					"extraArgs", extraArgs,
					"error", err,
					"attempt_current", i+1,
					"attempt_max", ghApiMaxRetry,
				)
				time.Sleep(waitTime)
				continue
			}
			return nil, fmt.Errorf("gh api error: %s", stderr)
		}
		return nil, fmt.Errorf("failed to execute gh: %w", err)
	}
	return nil, fmt.Errorf("gh api error after %d retries: %w", ghApiMaxRetry, err)
}

// ghApiPaginated makes an API call with --paginate to fetch all pages.
// The gh CLI handles pagination automatically and returns concatenated results.
func (c *ghClient) ghApiPaginated(method, endpoint string, extraArgs ...string) ([]byte, error) {
	args := append([]string{"api", "--paginate", "-X", method, endpoint}, extraArgs...)
	cmd := exec.Command("gh", args...)
	var err error
	var output []byte
	for i := range ghApiMaxRetry {
		output, err = cmd.Output()
		if err == nil {
			slog.Debug("paginated api call",
				"method", method,
				"endpoint", endpoint,
				"extraArgs", extraArgs,
			)
			return output, nil
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := string(exitErr.Stderr)
			if strings.Contains(stderr, "rate limit") || strings.Contains(stderr, "abuse detection") {
				waitTime := time.Duration((i+1)*2) * time.Second
				slog.Warn("rate limited on paginated call",
					"method", method,
					"endpoint", endpoint,
					"error", err,
					"attempt_current", i+1,
					"attempt_max", ghApiMaxRetry,
				)
				time.Sleep(waitTime)
				continue
			}
			return nil, fmt.Errorf("gh api error: %s", stderr)
		}
		return nil, fmt.Errorf("failed to execute gh: %w", err)
	}
	return nil, fmt.Errorf("gh api error after %d retries: %w", ghApiMaxRetry, err)
}

// RepoName extracts the repo identifier from a workflow run
func RepoName(run types.WorkflowRun) string {
	return run.Repository.FullName
}

// FormatDuration formats a duration in a human-readable way
func FormatDuration(d int64) string {
	seconds := d
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	minutes := seconds / 60
	seconds = seconds % 60
	if minutes < 60 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	hours := minutes / 60
	minutes = minutes % 60
	return fmt.Sprintf("%dh %dm", hours, minutes)
}

// TruncateString truncates a string to a maximum length
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// RepoParts splits a repo string into owner and name
func RepoParts(repo string) (owner, name string) {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return "", repo
	}
	return parts[0], parts[1]
}
