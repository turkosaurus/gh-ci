//go:build integration

package gh

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/turkosaurus/gh-ci/internal/config"
	"github.com/turkosaurus/gh-ci/internal/types"
)

func TestMain(m *testing.M) {
	if err := exec.Command("gh", "auth", "status").Run(); err != nil {
		fmt.Println("skipping integration tests: gh not available or not authenticated")
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func testRepo(t *testing.T) string {
	t.Helper()
	if r := os.Getenv("GH_CI_TEST_REPO"); r != "" {
		return r
	}
	cfg, err := config.Load()
	if err != nil || len(cfg.Repos) == 0 {
		t.Skip("no repo configured: set GH_CI_TEST_REPO or configure ~/.config/gh-ci/config.yml")
	}
	return cfg.Repos[0]
}

func TestCLIClientListWorkflowRuns(t *testing.T) {
	repo := testRepo(t)
	client := NewClient()

	runs, err := client.ListWorkflowRuns(repo, 5)
	if err != nil {
		t.Fatalf("ListWorkflowRuns(%q, 5) error: %v", repo, err)
	}

	// result must be the right type (may be empty)
	_ = []types.WorkflowRun(runs)

	if len(runs) > 0 {
		if runs[0].ID <= 0 {
			t.Errorf("runs[0].ID = %d, want > 0", runs[0].ID)
		}
		if runs[0].Status == "" {
			t.Errorf("runs[0].Status is empty")
		}
	}
}

func TestCLIClientGetJobs(t *testing.T) {
	repo := testRepo(t)
	client := NewClient()

	runs, err := client.ListWorkflowRuns(repo, 5)
	if err != nil {
		t.Fatalf("ListWorkflowRuns(%q, 5) error: %v", repo, err)
	}
	if len(runs) == 0 {
		t.Skip("no workflow runs found in repo")
	}

	jobs, err := client.GetJobs(repo, runs[0].ID)
	if err != nil {
		t.Fatalf("GetJobs(%q, %d) error: %v", repo, runs[0].ID, err)
	}

	if len(jobs) > 0 {
		if jobs[0].ID <= 0 {
			t.Errorf("jobs[0].ID = %d, want > 0", jobs[0].ID)
		}
	}
}

func TestCLIClientGetJobLogs(t *testing.T) {
	repo := testRepo(t)
	client := NewClient()

	runs, err := client.ListWorkflowRuns(repo, 5)
	if err != nil {
		t.Fatalf("ListWorkflowRuns(%q, 5) error: %v", repo, err)
	}
	if len(runs) == 0 {
		t.Skip("no workflow runs found in repo")
	}

	var completedJob *types.Job
	for i := range runs {
		jobs, err := client.GetJobs(repo, runs[i].ID)
		if err != nil {
			continue
		}
		for j := range jobs {
			if jobs[j].Status == "completed" {
				completedJob = &jobs[j]
				break
			}
		}
		if completedJob != nil {
			break
		}
	}
	if completedJob == nil {
		t.Skip("no completed jobs found in recent runs")
	}

	logs, err := client.GetJobLogs(repo, completedJob.ID)
	if err != nil {
		t.Fatalf("GetJobLogs(%q, %d) error: %v", repo, completedJob.ID, err)
	}
	if logs == "" {
		t.Errorf("GetJobLogs returned empty string")
	}
}
