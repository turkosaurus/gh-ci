package ui

import (
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/turkosaurus/gh-ci/internal/config"
	"github.com/turkosaurus/gh-ci/internal/gh"
	"github.com/turkosaurus/gh-ci/internal/types"
	"github.com/turkosaurus/gh-ci/internal/ui/keys"
	"github.com/turkosaurus/gh-ci/internal/ui/picker"
	"github.com/turkosaurus/gh-ci/internal/ui/styles"
)

// NewDashboard creates a Dashboard with the given dependencies.
func NewDashboard(cfg *config.Config, client gh.Client, runs *Fetchable[types.RunMap], s styles.Styles, k keys.KeyMap, defaultBranch, localBranch string) Dashboard {
	return Dashboard{
		born:           time.Now(),
		config:         cfg,
		client:         client,
		runs:           runs,
		styles:         s,
		keys:           k,
		repoPicker:     picker.NewPicker("filter repos..."),
		branchPicker:   picker.NewPicker("filter branches..."),
		workflowCursor: 1, // start on workflowAll (0=branch, 1=workflows[0])
		workflowFiles:  make(map[string]string),
		defaultBranch:  defaultBranch,
		localBranch:    localBranch,
		localRepo:      strings.Join(cfg.Repos, ", "),
	}
}

// SetRuns updates the dashboard with fresh run data and re-derives filters.
// workflowFiles is passed from App (which populates it from run paths + localDefs).
func (d *Dashboard) SetRuns(allRuns types.RunMap, localDefs []types.WorkflowDef, workflowFiles map[string]string) tea.Cmd {
	d.allRuns = allRuns
	d.localDefs = localDefs
	d.workflowFiles = workflowFiles

	// preserve cursors by name before re-deriving lists
	prevBranch := ""
	if d.availableBranches == nil {
		prevBranch = d.localBranch
	} else if d.branchIdx < len(d.availableBranches) {
		prevBranch = d.availableBranches[d.branchIdx]
	}
	prevWf := d.selectedWorkflow()

	d.workflows, d.availableBranches = listsFromRuns(d.localDefs, flattenRunMap(d.allRuns))

	// ensure both the configured primary branch and the local checkout are
	// always present, even when they have no runs yet
	for _, branch := range []string{d.defaultBranch, d.localBranch} {
		if branch == "" {
			continue
		}
		found := false
		for _, b := range d.availableBranches {
			if b == branch {
				found = true
				break
			}
		}
		if !found {
			d.availableBranches = append(d.availableBranches, branch)
			sort.Strings(d.availableBranches)
		}
	}

	// restore branch cursor
	d.branchIdx = 0
	for i, b := range d.availableBranches {
		if b == prevBranch {
			d.branchIdx = i
			break
		}
	}

	// restore workflow cursor
	if prevWf != "" {
		d.workflowCursor = 1
		for i, w := range d.workflows {
			if w == prevWf {
				d.workflowCursor = i + 1
				break
			}
		}
	} else if d.workflowCursor > len(d.workflows) {
		d.workflowCursor = 0
	}

	d.applyFilter()
	if run := d.selectedRun(); run != nil {
		return loadJobs(d.client, run.Repository.FullName, run.ID)
	}
	return nil
}

// SetJobs updates the dashboard with fresh job data.
func (d *Dashboard) SetJobs(jobs []types.Job) {
	d.jobs = jobs
	if d.jobCursor >= len(d.jobs) {
		d.jobCursor = 0
	}
}

func (d Dashboard) repoPickerNames() []string {
	names := make([]string, 0, len(d.nearbyRepos)+1)
	names = append(names, ".")
	for _, r := range d.nearbyRepos {
		names = append(names, r.Name)
	}
	return names
}

func (d Dashboard) filteredBranches() []string {
	q := strings.ToLower(d.branchPicker.Query())
	var out []string
	for _, b := range d.availableBranches {
		if q == "" || strings.Contains(strings.ToLower(b), q) {
			out = append(out, b)
		}
	}
	return out
}

func (d Dashboard) selectedBranch() string {
	if d.branchIdx < len(d.availableBranches) {
		return d.availableBranches[d.branchIdx]
	}
	return d.defaultBranch
}

func (d *Dashboard) applyFilter() {
	// filter by branch: collect runs for the selected branch across all repos
	var runs []types.WorkflowRun
	if d.branchIdx < len(d.availableBranches) {
		branch := d.availableBranches[d.branchIdx]
		switch branch {
		case branchAll:
			runs = flattenRunMap(d.allRuns)
		default:
			for _, repoMap := range d.allRuns {
				runs = append(runs, repoMap[branch]...)
				if branch == d.defaultBranch {
					for tagBranch, tagRuns := range repoMap {
						if isTagBranch(tagBranch) {
							runs = append(runs, tagRuns...)
						}
					}
				}
			}
			sort.Slice(runs, func(i, j int) bool {
				return runs[i].CreatedAt.After(runs[j].CreatedAt)
			})
		}
	} else {
		runs = flattenRunMap(d.allRuns)
	}

	// filter by workflow
	if wfName := d.selectedWorkflow(); wfName != "" && wfName != workflowAll {
		var filtered []types.WorkflowRun
		for _, r := range runs {
			if r.Name == wfName {
				filtered = append(filtered, r)
			}
		}
		runs = filtered
	}

	d.filteredRuns = runs
	if d.cursor >= len(d.filteredRuns) {
		d.cursor = max(0, len(d.filteredRuns)-1)
	}
}

func (d Dashboard) selectedRun() *types.WorkflowRun {
	if d.cursor >= 0 && d.cursor < len(d.filteredRuns) {
		return &d.filteredRuns[d.cursor]
	}
	return nil
}

// selectedWorkflow returns the workflow name at the current workflow cursor,
// or "" when the branch row (cursor 0) or an out-of-range position is selected.
func (d Dashboard) selectedWorkflow() string {
	if d.workflowCursor > 0 && d.workflowCursor <= len(d.workflows) {
		return d.workflows[d.workflowCursor-1]
	}
	return ""
}

func (d Dashboard) moveCursor(delta int) (Dashboard, tea.Cmd) {
	switch d.activePanel {
	case panelWorkflows:
		n := d.workflowCursor + delta
		if n >= -1 && n <= len(d.workflows) {
			d.workflowCursor = n
			d.applyFilter()
			d.cursor = 0
			d.jobs = nil
			d.jobCursor = 0
			if run := d.selectedRun(); run != nil {
				return d, loadJobs(d.client, run.Repository.FullName, run.ID)
			}
		}
	case panelRuns:
		n := d.cursor + delta
		if n >= 0 && n < len(d.filteredRuns) {
			d.cursor = n
			d.jobs = nil
			d.jobCursor = 0
			if run := d.selectedRun(); run != nil {
				return d, loadJobs(d.client, run.Repository.FullName, run.ID)
			}
		}
	case panelDetail:
		n := d.jobCursor + delta
		if n >= 0 && n < len(d.jobs) {
			d.jobCursor = n
		}
	}
	return d, nil
}

func (d Dashboard) moveCursorPage(dir int) (Dashboard, tea.Cmd) {
	const pageSize = 10
	switch d.activePanel {
	case panelWorkflows:
		n := max(0, min(len(d.workflows), d.workflowCursor+dir*pageSize))
		if n != d.workflowCursor {
			d.workflowCursor = n
			d.applyFilter()
			d.cursor = 0
			d.jobs = nil
			d.jobCursor = 0
			if run := d.selectedRun(); run != nil {
				return d, loadJobs(d.client, run.Repository.FullName, run.ID)
			}
		}
	case panelRuns:
		n := max(0, min(len(d.filteredRuns)-1, d.cursor+dir*pageSize))
		if n != d.cursor {
			d.cursor = n
			d.jobs = nil
			d.jobCursor = 0
			if run := d.selectedRun(); run != nil {
				return d, loadJobs(d.client, run.Repository.FullName, run.ID)
			}
		}
	}
	return d, nil
}

func (d Dashboard) moveCursorEdge(top bool) (Dashboard, tea.Cmd) {
	switch d.activePanel {
	case panelWorkflows:
		if top {
			d.workflowCursor = 0
		} else {
			d.workflowCursor = len(d.workflows)
		}
		d.applyFilter()
		d.cursor = 0
		d.jobs = nil
		d.jobCursor = 0
		if run := d.selectedRun(); run != nil {
			return d, loadJobs(d.client, run.Repository.FullName, run.ID)
		}
	case panelRuns:
		if top {
			d.cursor = 0
		} else {
			d.cursor = max(0, len(d.filteredRuns)-1)
		}
		d.jobs = nil
		d.jobCursor = 0
		if run := d.selectedRun(); run != nil {
			return d, loadJobs(d.client, run.Repository.FullName, run.ID)
		}
	case panelDetail:
		if top {
			d.jobCursor = 0
		} else {
			d.jobCursor = max(0, len(d.jobs)-1)
		}
	}
	return d, nil
}

func (d Dashboard) openURL() string {
	switch d.activePanel {
	case panelWorkflows:
		wfName := d.selectedWorkflow()
		if wfName == "" || wfName == workflowAll {
			// branch cell or "all workflows" — open repo actions page
			if run := d.selectedRun(); run != nil {
				return run.Repository.HTMLURL + "/actions"
			}
			if len(d.config.Repos) > 0 {
				return "https://github.com/" + d.config.Repos[0] + "/actions"
			}
		} else {
			// specific workflow — open its actions/workflows page
			for _, r := range flattenRunMap(d.allRuns) {
				if r.Name == wfName {
					if filename, ok := d.workflowFiles[wfName]; ok {
						return r.Repository.HTMLURL + "/actions/workflows/" + filename
					}
					return r.HTMLURL
				}
			}
		}
	case panelRuns:
		if run := d.selectedRun(); run != nil {
			return run.HTMLURL
		}
	case panelDetail:
		if d.jobCursor < len(d.jobs) {
			return d.jobs[d.jobCursor].HTMLURL
		}
	}
	return ""
}
