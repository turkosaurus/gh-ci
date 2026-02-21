package gh

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/turkosaurus/gh-ci/internal/types"
)

// Client wraps the gh CLI for API calls
type Client struct{}

// NewClient creates a new GitHub API client
func NewClient() *Client {
	return &Client{}
}

// ListWorkflowRuns fetches workflow runs for a repository
func (c *Client) ListWorkflowRuns(repo string, perPage int) ([]types.WorkflowRun, error) {
	endpoint := fmt.Sprintf("repos/%s/actions/runs?per_page=%d", repo, perPage)
	output, err := c.apiCall("GET", endpoint)
	if err != nil {
		return nil, err
	}

	var response types.WorkflowRunsResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return response.WorkflowRuns, nil
}

// GetJobs fetches jobs for a workflow run
func (c *Client) GetJobs(repo string, runID int64) ([]types.Job, error) {
	endpoint := fmt.Sprintf("repos/%s/actions/runs/%d/jobs", repo, runID)
	output, err := c.apiCall("GET", endpoint)
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
func (c *Client) GetJobLogs(repo string, jobID int64) (string, error) {
	endpoint := fmt.Sprintf("repos/%s/actions/jobs/%d/logs", repo, jobID)
	output, err := c.apiCall("GET", endpoint)
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// RerunWorkflow re-runs a workflow, optionally with debug logging enabled
func (c *Client) RerunWorkflow(repo string, runID int64, debug bool) error {
	endpoint := fmt.Sprintf("repos/%s/actions/runs/%d/rerun", repo, runID)
	var extra []string
	if debug {
		extra = []string{"-F", "enable_debug_logging=true"}
	}
	_, err := c.apiCall("POST", endpoint, extra...)
	return err
}

// RerunFailedJobs re-runs only failed jobs in a workflow
func (c *Client) RerunFailedJobs(repo string, runID int64) error {
	endpoint := fmt.Sprintf("repos/%s/actions/runs/%d/rerun-failed-jobs", repo, runID)
	_, err := c.apiCall("POST", endpoint)
	return err
}

// CancelWorkflow cancels a running workflow
func (c *Client) CancelWorkflow(repo string, runID int64) error {
	endpoint := fmt.Sprintf("repos/%s/actions/runs/%d/cancel", repo, runID)
	_, err := c.apiCall("POST", endpoint)
	return err
}

// DispatchWorkflow triggers a workflow_dispatch event on the given ref.
// workflowFile is the filename, e.g. "ci.yaml".
func (c *Client) DispatchWorkflow(repo, workflowFile, ref string) error {
	endpoint := fmt.Sprintf("repos/%s/actions/workflows/%s/dispatches", repo, workflowFile)
	_, err := c.apiCall("POST", endpoint, "-f", "ref="+ref)
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
func (c *Client) OpenInBrowser(url string) error {
	cmd := exec.Command("gh", "browse", "--url", url)
	// Use open command on macOS, xdg-open on Linux
	cmd = exec.Command("open", url)
	return cmd.Start()
}

// apiCall makes an API call using the gh CLI
func (c *Client) apiCall(method, endpoint string, extraArgs ...string) ([]byte, error) {
	args := append([]string{"api", "-X", method, endpoint}, extraArgs...)
	cmd := exec.Command("gh", args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("gh api error: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to execute gh: %w", err)
	}
	return output, nil
}

// ParseRepoFromRun extracts the repo identifier from a workflow run
func ParseRepoFromRun(run types.WorkflowRun) string {
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

// SplitRepo splits a repo string into owner and name
func SplitRepo(repo string) (owner, name string) {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return "", repo
	}
	return parts[0], parts[1]
}
