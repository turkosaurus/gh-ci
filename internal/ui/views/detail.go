package views

import (
	"fmt"
	"strings"

	"github.com/jay-418/gh-ci/internal/gh"
	"github.com/jay-418/gh-ci/internal/types"
	"github.com/jay-418/gh-ci/internal/ui/components"
	"github.com/jay-418/gh-ci/internal/ui/styles"
)

// DetailView renders the run detail/logs view
type DetailView struct {
	Run       *types.WorkflowRun
	Jobs      []types.Job
	LogViewer components.LogViewer
	Help      components.Help
	Styles    styles.Styles
	Width     int
	Height    int
	ShowLogs  bool
	Loading   bool
	Error     string
}

// NewDetailView creates a new detail view
func NewDetailView(s styles.Styles) DetailView {
	return DetailView{
		LogViewer: components.NewLogViewer(s),
		Help:      components.NewHelp(s),
		Styles:    s,
		ShowLogs:  false,
		Loading:   false,
	}
}

// SetRun sets the workflow run to display
func (v *DetailView) SetRun(run *types.WorkflowRun) {
	v.Run = run
	v.Jobs = nil
	v.ShowLogs = false
	v.Loading = true
	v.Error = ""
}

// SetJobs sets the jobs for the run
func (v *DetailView) SetJobs(jobs []types.Job) {
	v.Jobs = jobs
	v.Loading = false
}

// SetError sets an error
func (v *DetailView) SetError(err string) {
	v.Error = err
	v.Loading = false
}

// SetSize sets the dimensions
func (v *DetailView) SetSize(width, height int) {
	v.Width = width
	v.Height = height
	v.LogViewer.SetSize(width, height-4)
}

// View renders the detail view
func (v *DetailView) View() string {
	if v.ShowLogs {
		return v.renderLogs()
	}
	return v.renderDetail()
}

// renderDetail renders the run details
func (v *DetailView) renderDetail() string {
	if v.Run == nil {
		return v.Styles.Dimmed.Render("No run selected")
	}

	var sb strings.Builder

	// Header
	header := fmt.Sprintf("Run #%d: %s", v.Run.RunNumber, v.Run.Name)
	sb.WriteString(v.Styles.Header.Render(header))
	sb.WriteString("\n\n")

	// Run info
	sb.WriteString(v.renderField("Repository", v.Styles.Repo.Render(v.Run.Repository.FullName)))
	sb.WriteString(v.renderField("Branch", v.Styles.Branch.Render(v.Run.HeadBranch)))
	sb.WriteString(v.renderField("Commit", v.Styles.Normal.Render(v.Run.HeadSHA[:8])))

	statusStyle := v.Styles.StatusStyle(v.Run.Status, v.Run.Conclusion)
	status := v.Run.GetStatus()
	sb.WriteString(v.renderField("Status", statusStyle.Render(styles.StatusIcon(v.Run.Status, v.Run.Conclusion)+" "+status)))

	duration := v.Run.Duration()
	durationStr := gh.FormatDuration(int64(duration.Seconds()))
	sb.WriteString(v.renderField("Duration", v.Styles.Duration.Render(durationStr)))
	sb.WriteString("\n")

	// Jobs
	if v.Loading {
		sb.WriteString(v.Styles.Dimmed.Render("Loading jobs..."))
	} else if v.Error != "" {
		sb.WriteString(v.Styles.Error.Render("Error: " + v.Error))
	} else if len(v.Jobs) > 0 {
		sb.WriteString(v.Styles.Subtitle.Render("Jobs:"))
		sb.WriteString("\n")
		for _, job := range v.Jobs {
			sb.WriteString(v.renderJob(job))
		}
	}

	sb.WriteString("\n")

	// Help bar
	v.Help.SetView("list")
	sb.WriteString(v.Help.Render())

	return v.Styles.App.Render(sb.String())
}

// renderLogs renders the log viewer
func (v *DetailView) renderLogs() string {
	var sb strings.Builder
	sb.WriteString(v.LogViewer.View())
	sb.WriteString("\n\n")

	v.Help.SetView("logs")
	sb.WriteString(v.Help.Render())

	return v.Styles.App.Render(sb.String())
}

// renderField renders a single field
func (v *DetailView) renderField(label, value string) string {
	return fmt.Sprintf("  %s: %s\n",
		v.Styles.Dimmed.Render(label),
		value,
	)
}

// renderJob renders a job
func (v *DetailView) renderJob(job types.Job) string {
	icon := styles.StatusIcon(job.Status, job.Conclusion)
	style := v.Styles.StatusStyle(job.Status, job.Conclusion)
	return fmt.Sprintf("  %s %s\n", style.Render(icon), job.Name)
}
