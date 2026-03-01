package ui

import (
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/turkosaurus/gh-ci/internal/types"
	"github.com/turkosaurus/gh-ci/internal/ui/picker"
)

// Update handles a key event and returns the updated dashboard and command.
func (d Dashboard) Update(msg tea.KeyMsg) (Dashboard, tea.Cmd) {
	if d.helpModal.Active() {
		d.helpModal.Close()
		return d, nil
	}
	if d.repoPicker.Active() {
		return d.handleRepoSelect(msg)
	}
	if d.branchPicker.Active() {
		return d.handleBranchSelect(msg)
	}
	if d.dispatchDialog.Active() {
		return d.handleDispatchConfirm(msg)
	}
	if d.confirmDialog.Active() {
		return d.handleConfirm(msg)
	}
	return d.handleMainKeys(msg)
}

func (d Dashboard) handleRepoSelect(msg tea.KeyMsg) (Dashboard, tea.Cmd) {
	var cmd tea.Cmd
	var result *picker.PickResult
	d.repoPicker, cmd, result = d.repoPicker.Update(msg)
	if result != nil {
		chosen := result.Chosen
		branch := ""
		if chosen == "." {
			chosen = d.localRepo
			if len(d.nearbyRepos) > 0 {
				branch = d.nearbyRepos[0].Branch
			}
		} else {
			for _, r := range d.nearbyRepos {
				if r.Name == chosen {
					branch = r.Branch
					break
				}
			}
		}
		d.config.Repos = []string{chosen}
		d.allRuns = nil
		d.filteredRuns = nil
		d.workflows = nil
		d.availableBranches = nil
		d.branchIdx = 0
		d.localBranch = branch
		d.jobs = nil
		d.cursor = 0
		d.jobCursor = 0
		d.workflowCursor = 1
		return d, refreshRuns(d.client, d.config.Repos, time.Time{})
	}
	return d, cmd
}

func (d Dashboard) handleBranchSelect(msg tea.KeyMsg) (Dashboard, tea.Cmd) {
	var cmd tea.Cmd
	var result *picker.PickResult
	d.branchPicker, cmd, result = d.branchPicker.Update(msg)
	if result != nil {
		for i, b := range d.availableBranches {
			if b == result.Chosen {
				d.branchIdx = i
				break
			}
		}
		d.workflowCursor = 1 // land on workflowAll so next Enter goes right, not re-opens selector
		d.applyFilter()
		d.cursor = 0
	}
	return d, cmd
}

func (d Dashboard) handleConfirm(msg tea.KeyMsg) (Dashboard, tea.Cmd) {
	var cmd tea.Cmd
	var statusMsg string
	d.confirmDialog, cmd, statusMsg = d.confirmDialog.Update(msg, d.client)
	if statusMsg != "" {
		d.PendingMessage = statusMsg
	}
	return d, cmd
}

func (d Dashboard) handleDispatchConfirm(msg tea.KeyMsg) (Dashboard, tea.Cmd) {
	var cmd tea.Cmd
	var statusMsg string
	d.dispatchDialog, cmd, statusMsg = d.dispatchDialog.Update(msg, d.client)
	if statusMsg != "" {
		d.PendingMessage = statusMsg
	}
	return d, cmd
}

func (d Dashboard) handleMainKeys(msg tea.KeyMsg) (Dashboard, tea.Cmd) {
	switch {
	case key.Matches(msg, d.keys.Quit):
		return d, tea.Quit

	case key.Matches(msg, d.keys.Up):
		return d.moveCursor(-1)

	case key.Matches(msg, d.keys.Down):
		return d.moveCursor(1)

	case key.Matches(msg, d.keys.PageUp):
		return d.moveCursorPage(-1)

	case key.Matches(msg, d.keys.PageDown):
		return d.moveCursorPage(1)

	case key.Matches(msg, d.keys.Top):
		return d.moveCursorEdge(true)

	case key.Matches(msg, d.keys.Bottom):
		return d.moveCursorEdge(false)

	case key.Matches(msg, d.keys.Right): // l — move right between panels
		if d.activePanel < panelDetail {
			d.activePanel++
		}

	case key.Matches(msg, d.keys.Enter):
		if d.activePanel == panelWorkflows && d.workflowCursor == -1 {
			if len(d.nearbyRepos) > 0 {
				cmd := d.repoPicker.Open(d.repoPickerNames())
				return d, cmd
			}
			return d, discoverRepos()
		} else if d.activePanel == panelWorkflows && d.workflowCursor == 0 {
			cmd := d.branchPicker.Open(d.availableBranches)
			return d, cmd
		} else if d.activePanel < panelDetail {
			d.activePanel++
		} else if d.jobCursor < len(d.jobs) {
			// detail panel: enter opens logs for the selected job
			if run := d.selectedRun(); run != nil {
				job := d.jobs[d.jobCursor]
				d.PendingMessage = "loading logs..."
				return d, loadLogs(d.client, run.Repository.FullName, job.ID, job.Name)
			}
		}

	case key.Matches(msg, d.keys.Left): // move left between panels
		if d.activePanel > panelWorkflows {
			d.activePanel--
		}

	case key.Matches(msg, d.keys.Open):
		if url := d.openURL(); url != "" {
			d.client.OpenInBrowser(url)
		}

	case key.Matches(msg, d.keys.Rerun):
		if run := d.selectedRun(); run != nil {
			d.confirmDialog.Open(run.Repository.FullName, run.ID)
		}

	case key.Matches(msg, d.keys.Cancel):
		if run := d.selectedRun(); run != nil && run.Status == types.RunStatusInProgress {
			d.PendingMessage = "cancelling..."
			return d, cancelWorkflow(d.client, run.Repository.FullName, run.ID)
		}

	case key.Matches(msg, d.keys.Dispatch):
		if d.activePanel == panelWorkflows {
			if wfName := d.selectedWorkflow(); wfName != "" && wfName != workflowAll {
				if file, ok := d.workflowFiles[wfName]; ok {
					var repo string
					if run := d.selectedRun(); run != nil {
						// Derive repo from the currently visible run (already filtered by
						// branch + workflow), so dispatch always targets the right repo.
						repo = run.Repository.FullName
					} else if len(d.config.Repos) == 1 {
						// Local-only workflow (no runs yet) with a single configured repo.
						repo = d.config.Repos[0]
					} else {
						d.PendingMessage = "cannot dispatch: no runs for this workflow on this branch"
						return d, clearMsg(time.Duration(d.config.MsgTimeout) * time.Second)
					}
					d.dispatchDialog.Open(repo, file, d.selectedBranch())
				}
			}
		}

	case key.Matches(msg, d.keys.Edit):
		if d.activePanel == panelWorkflows {
			if wfName := d.selectedWorkflow(); wfName != "" && wfName != workflowAll {
				if file, ok := d.workflowFiles[wfName]; ok {
					fullPath := filepath.Join(gitRoot(), workflowDirGitHub, file)
					return d, editFile(fullPath)
				}
			}
		}

	case key.Matches(msg, d.keys.Back):
		if d.activePanel > 0 {
			d.activePanel--
		}

	case key.Matches(msg, d.keys.Refresh):
		d.PendingMessage = "refreshing..."
		return d, refreshRuns(d.client, d.config.Repos, time.Time{})

	case key.Matches(msg, d.keys.Help):
		d.helpModal.Open()
		return d, nil
	}

	return d, nil
}

// ── Mouse handling ──────────────────────────────────────────────────────────

// handleMouse processes a left-click at the given terminal coordinates.
func (d Dashboard) handleMouse(x, y, w, h int) (Dashboard, tea.Cmd) {
	if d.helpModal.Active() || d.repoPicker.Active() || d.branchPicker.Active() ||
		d.confirmDialog.Active() || d.dispatchDialog.Active() {
		return d, nil
	}

	bodyTop := 4     // 3-line title + 1 panel header row
	bodyBot := h - 1 // help bar
	if y < bodyTop || y >= bodyBot {
		return d, nil
	}
	bodyY := y - bodyTop
	bodyH := bodyBot - bodyTop

	lay := computeDashLayout(w)

	if x < lay.workflowW {
		d.activePanel = panelWorkflows
		return d.handleWorkflowClick(bodyY, bodyH)
	}
	if x < lay.workflowW+1+lay.runsW {
		d.activePanel = panelRuns
		return d.handleRunClick(bodyY, bodyH)
	}
	d.activePanel = panelDetail
	return d.handleDetailClick(bodyY)
}

func (d Dashboard) handleWorkflowClick(bodyY, bodyH int) (Dashboard, tea.Cmd) {
	// Row layout: 0=REPO hdr, 1=repo, 2=sep, 3=BRANCH hdr, 4=branch, 5=sep, 6=NAME hdr, 7+=items
	switch {
	case bodyY == 1:
		d.workflowCursor = -1
		d.applyFilter()
		d.cursor = 0
	case bodyY == 4:
		d.workflowCursor = 0
		d.applyFilter()
		d.cursor = 0
	case bodyY >= 7 && len(d.workflows) > 0:
		// Compute scroll offset (mirrors renderWorkflows)
		filenameShown := false
		if wfName := d.selectedWorkflow(); wfName != "" && wfName != workflowAll {
			if _, ok := d.workflowFiles[wfName]; ok {
				filenameShown = true
			}
		}
		workflowListH := bodyH - 7
		if filenameShown {
			workflowListH--
		}
		if workflowListH < 1 {
			workflowListH = 1
		}
		wfCursor := 0
		if d.workflowCursor > 0 {
			wfCursor = d.workflowCursor - 1
		}
		startIdx := 0
		if wfCursor >= workflowListH {
			startIdx = wfCursor - workflowListH + 1
		}
		clickedIdx := startIdx + (bodyY - 7)
		if clickedIdx >= 0 && clickedIdx < len(d.workflows) {
			d.workflowCursor = clickedIdx + 1
			d.applyFilter()
			d.cursor = 0
			d.jobs = nil
			d.jobCursor = 0
			if run := d.selectedRun(); run != nil {
				return d, loadJobs(d.client, run.Repository.FullName, run.ID)
			}
		}
	}
	return d, nil
}

func (d Dashboard) handleRunClick(bodyY, bodyH int) (Dashboard, tea.Cmd) {
	// Row 0: column header, Row 1+: run items
	if bodyY < 1 || len(d.filteredRuns) == 0 {
		return d, nil
	}
	listH := bodyH - 2
	startIdx := 0
	if d.cursor >= listH {
		startIdx = d.cursor - listH + 1
	}
	clickedIdx := startIdx + (bodyY - 1)
	if clickedIdx >= 0 && clickedIdx < len(d.filteredRuns) {
		d.cursor = clickedIdx
		d.jobs = nil
		d.jobCursor = 0
		if run := d.selectedRun(); run != nil {
			return d, loadJobs(d.client, run.Repository.FullName, run.ID)
		}
	}
	return d, nil
}

func (d Dashboard) handleDetailClick(bodyY int) (Dashboard, tea.Cmd) {
	// Rows 0-7: header fields, Row 8+: job items
	if bodyY < 8 || len(d.jobs) == 0 {
		return d, nil
	}
	clickedIdx := bodyY - 8
	if clickedIdx >= 0 && clickedIdx < len(d.jobs) {
		d.jobCursor = clickedIdx
	}
	return d, nil
}
