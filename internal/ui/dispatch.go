package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/turkosaurus/gh-ci/internal/gh"
	"github.com/turkosaurus/gh-ci/internal/ui/styles"
)

// DispatchDialog handles the dispatch confirmation prompt (y/esc).
type DispatchDialog struct {
	active bool
	repo   string
	file   string
	ref    string
}

func (dd *DispatchDialog) Open(repo, file, ref string) {
	dd.active = true
	dd.repo = repo
	dd.file = file
	dd.ref = ref
}

func (dd DispatchDialog) Active() bool { return dd.active }

// Update handles key input. Returns the updated dialog, a command, and a status message.
func (dd DispatchDialog) Update(msg tea.KeyMsg, client gh.Client) (DispatchDialog, tea.Cmd, string) {
	switch msg.String() {
	case "y":
		dd.active = false
		return dd, runDispatch(client, dd.repo, dd.file, dd.ref), "dispatching..."
	case "esc", "q":
		dd.active = false
	}
	return dd, nil, ""
}

// HelpView returns the help bar when the dispatch dialog is active.
func (dd DispatchDialog) HelpView(s styles.Styles) string {
	return s.Normal.Render("dispatch "+dd.file+" on "+dd.ref+"?") + "  " +
		s.HelpKey.Render("y") + " " + s.HelpDesc.Render("yes") + "  " +
		s.HelpKey.Render("esc") + " " + s.HelpDesc.Render("cancel")
}
