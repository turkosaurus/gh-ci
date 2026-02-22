package types

import (
	"testing"
	"time"
)

func TestWorkflowRunGetStatus(t *testing.T) {
	tests := []struct {
		status     string
		conclusion string
		want       string
	}{
		{RunStatusCompleted, "success", "success"},
		{RunStatusCompleted, "failure", "failure"},
		{RunStatusInProgress, "", RunStatusInProgress},
		{RunStatusQueued, "", RunStatusQueued},
	}
	for _, tt := range tests {
		r := &WorkflowRun{Status: tt.status, Conclusion: tt.conclusion}
		got := r.GetStatus()
		if got != tt.want {
			t.Errorf("GetStatus() with status=%q conclusion=%q = %q, want %q",
				tt.status, tt.conclusion, got, tt.want)
		}
	}
}

func TestWorkflowRunDuration(t *testing.T) {
	start := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	end := start.Add(90 * time.Second)

	r := &WorkflowRun{
		Status:       RunStatusCompleted,
		RunStartedAt: start,
		UpdatedAt:    end,
	}
	got := r.Duration()
	want := 90 * time.Second
	if got != want {
		t.Errorf("Duration() = %v, want %v", got, want)
	}
}
