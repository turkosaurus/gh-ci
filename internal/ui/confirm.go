package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/turkosaurus/gh-ci/internal/gh"
	"github.com/turkosaurus/gh-ci/internal/ui/styles"
)

// ConfirmDialog handles the rerun confirmation prompt (y/d/esc).
type ConfirmDialog struct {
	active bool
	repo   string
	runID  int64
}

func (cd *ConfirmDialog) Open(repo string, runID int64) {
	cd.active = true
	cd.repo = repo
	cd.runID = runID
}

func (cd ConfirmDialog) Active() bool { return cd.active }

// Update handles key input. Returns the updated dialog and a command.
// A non-nil command means the user confirmed (rerun triggered).
func (cd ConfirmDialog) Update(msg tea.KeyMsg, client gh.Client) (ConfirmDialog, tea.Cmd, string) {
	switch msg.String() {
	case "y":
		cd.active = false
		return cd, rerunWorkflow(client, cd.repo, cd.runID, false), "re-running..."
	case "d":
		cd.active = false
		return cd, rerunWorkflow(client, cd.repo, cd.runID, true), "re-running with debug..."
	case "esc", "q":
		cd.active = false
	}
	return cd, nil, ""
}

// HelpView returns the help bar when the confirm dialog is active.
func (cd ConfirmDialog) HelpView(s styles.Styles) string {
	return s.Normal.Render("re-run?") + "  " +
		s.HelpKey.Render("y") + " " + s.HelpDesc.Render("normal") + "  " +
		s.HelpKey.Render("d") + " " + s.HelpDesc.Render("debug logs") + "  " +
		s.HelpKey.Render("esc") + " " + s.HelpDesc.Render("cancel")
}
