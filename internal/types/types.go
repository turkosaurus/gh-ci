package types

import "time"

// WorkflowRun represents a GitHub Actions workflow run
type WorkflowRun struct {
	ID           int64      `json:"id"`
	Name         string     `json:"name"`
	DisplayTitle string     `json:"display_title"`
	HeadBranch   string     `json:"head_branch"`
	HeadSHA      string     `json:"head_sha"`
	Status       string     `json:"status"`     // queued, in_progress, completed
	Conclusion   string     `json:"conclusion"` // success, failure, cancelled, skipped, etc.
	WorkflowID   int64      `json:"workflow_id"`
	Path         string     `json:"path"` // e.g. ".github/workflows/ci.yaml"
	RunNumber    int        `json:"run_number"`
	RunAttempt   int        `json:"run_attempt"`
	HTMLURL      string     `json:"html_url"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	RunStartedAt time.Time  `json:"run_started_at"`
	Repository   Repository `json:"repository"`
}

// Repository represents a GitHub repository
type Repository struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	HTMLURL  string `json:"html_url"`
}

// WorkflowRunsResponse is the API response for listing workflow runs
type WorkflowRunsResponse struct {
	TotalCount   int           `json:"total_count"`
	WorkflowRuns []WorkflowRun `json:"workflow_runs"`
}

// Job represents a job within a workflow run
type Job struct {
	ID          int64     `json:"id"`
	RunID       int64     `json:"run_id"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	Conclusion  string    `json:"conclusion"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	HTMLURL     string    `json:"html_url"`
	Steps       []Step    `json:"steps"`
}

// Step represents a step within a job
type Step struct {
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	Conclusion  string    `json:"conclusion"`
	Number      int       `json:"number"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
}

// JobsResponse is the API response for listing jobs
type JobsResponse struct {
	TotalCount int   `json:"total_count"`
	Jobs       []Job `json:"jobs"`
}

// WorkflowDef is a locally-discovered workflow file (may have no runs yet)
type WorkflowDef struct {
	Name string // from the "name:" YAML field; falls back to filename sans extension
	File string // e.g. "ci.yaml"
}

// StatusFilter represents the filter options for workflow run status
type StatusFilter string

const (
	StatusAll        StatusFilter = "all"
	StatusFailed     StatusFilter = "failed"
	StatusInProgress StatusFilter = "in_progress"
	StatusSuccess    StatusFilter = "success"
)

// GetStatus returns a display-friendly status string
func (r *WorkflowRun) GetStatus() string {
	if r.Status == "completed" {
		return r.Conclusion
	}
	return r.Status
}

// Duration returns the duration of the workflow run
func (r *WorkflowRun) Duration() time.Duration {
	if r.Status == "completed" {
		return r.UpdatedAt.Sub(r.RunStartedAt)
	}
	return time.Since(r.RunStartedAt)
}
