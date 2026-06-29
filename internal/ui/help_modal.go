package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/turkosaurus/gh-ci/internal/ui/keys"
	"github.com/turkosaurus/gh-ci/internal/ui/styles"
)

// HelpModal displays keybindings for the current screen context.
type HelpModal struct {
	active bool
}

// Open activates the help modal.
func (h *HelpModal) Open() {
	h.active = true
}

// Close deactivates the help modal.
func (h *HelpModal) Close() {
	h.active = false
}

// Active returns whether the help modal is currently displayed.
func (h HelpModal) Active() bool {
	return h.active
}

// View renders the help modal centered on screen.
// isDashboard=true for dashboard context, false for log viewer context.
func (h HelpModal) View(k keys.KeyMap, s styles.Styles, width, height int, isDashboard bool) string {
	const maxBoxWidth = 80
	const maxBoxHeight = 30

	boxW := width - 4
	if boxW > maxBoxWidth {
		boxW = maxBoxWidth
	}
	if boxW < 50 {
		boxW = 50
	}

	boxH := height - 4
	if boxH > maxBoxHeight {
		boxH = maxBoxHeight
	}
	if boxH < 10 {
		boxH = 10
	}

	var content string
	if isDashboard {
		content = h.dashboardBindings(k, s, boxW, boxH)
	} else {
		content = h.logviewerBindings(k, s, boxW, boxH)
	}

	box := s.Border.
		Width(boxW).
		Padding(1, 2).
		Render(content)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceBackground(s.P.Bg))
}

func (h HelpModal) dashboardBindings(k keys.KeyMap, s styles.Styles, width, height int) string {
	sectionHeader := func(title string) string {
		return lipgloss.NewStyle().Bold(true).Foreground(s.P.Accent).Render(title)
	}

	binding := func(key, desc string) string {
		return s.HelpKey.Render(key) + " " + s.HelpDesc.Render(desc)
	}

	left := []string{
		sectionHeader("NAVIGATION"),
		binding("↑/k", "up"),
		binding("↓/j", "down"),
		binding("h/←", "left"),
		binding("l/→", "right"),
		binding("enter", "select"),
		binding("pgup", "page up"),
		binding("pgdn", "page down"),
		binding("g", "top"),
		binding("G", "bottom"),
		"",
		sectionHeader("ACTIONS"),
		binding("r", "re-run"),
		binding("X", "cancel"),
		binding("d", "dispatch"),
		binding("o", "open in browser"),
		binding("R", "refresh"),
	}

	right := []string{
		sectionHeader("GENERAL"),
		binding("q", "quit"),
		binding("esc", "back"),
		binding("c", "config"),
		binding("?", "help"),
	}

	// Join columns horizontally
	leftStr := strings.Join(left, "\n")
	rightStr := strings.Join(right, "\n")

	// Pad left column to align with right
	leftLines := strings.Split(leftStr, "\n")
	rightLines := strings.Split(rightStr, "\n")
	maxLines := len(leftLines)
	if len(rightLines) > maxLines {
		maxLines = len(rightLines)
	}

	// Pad shorter column
	for len(leftLines) < maxLines {
		leftLines = append(leftLines, "")
	}
	for len(rightLines) < maxLines {
		rightLines = append(rightLines, "")
	}

	// Join column-wise
	var rows []string
	for i := 0; i < maxLines; i++ {
		// Ensure left column is padded to reasonable width
		leftW := 25
		left := lipgloss.NewStyle().Width(leftW).Render(leftLines[i])
		right := rightLines[i]
		rows = append(rows, left+"  "+right)
	}

	return strings.Join(rows, "\n")
}

func (h HelpModal) logviewerBindings(k keys.KeyMap, s styles.Styles, width, height int) string {
	sectionHeader := func(title string) string {
		return lipgloss.NewStyle().Bold(true).Foreground(s.P.Accent).Render(title)
	}

	binding := func(key, desc string) string {
		return s.HelpKey.Render(key) + " " + s.HelpDesc.Render(desc)
	}

	left := []string{
		sectionHeader("NAVIGATION"),
		binding("↑/k", "up"),
		binding("↓/j", "down"),
		binding("pgup", "page up"),
		binding("pgdn", "page down"),
		binding("ctrl+u", "½ page up"),
		binding("ctrl+d", "½ page down"),
		binding("g", "top"),
		binding("G", "bottom"),
	}

	right := []string{
		sectionHeader("SEARCH"),
		binding("/", "search"),
		binding("n", "next match"),
		binding("p", "prev match"),
		"",
		sectionHeader("GENERAL"),
		binding("q", "quit"),
		binding("esc", "back"),
		binding("?", "help"),
	}

	// Join columns horizontally
	leftStr := strings.Join(left, "\n")
	rightStr := strings.Join(right, "\n")

	// Pad left column to align with right
	leftLines := strings.Split(leftStr, "\n")
	rightLines := strings.Split(rightStr, "\n")
	maxLines := len(leftLines)
	if len(rightLines) > maxLines {
		maxLines = len(rightLines)
	}

	// Pad shorter column
	for len(leftLines) < maxLines {
		leftLines = append(leftLines, "")
	}
	for len(rightLines) < maxLines {
		rightLines = append(rightLines, "")
	}

	// Join column-wise
	var rows []string
	for i := 0; i < maxLines; i++ {
		// Ensure left column is padded to reasonable width
		leftW := 25
		left := lipgloss.NewStyle().Width(leftW).Render(leftLines[i])
		right := rightLines[i]
		rows = append(rows, left+"  "+right)
	}

	return strings.Join(rows, "\n")
}
