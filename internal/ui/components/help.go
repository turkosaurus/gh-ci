package components

import (
	"strings"

	"github.com/jay-418/gh-ci/internal/ui/styles"
)

// Help is a component for displaying help information
type Help struct {
	Styles   styles.Styles
	ShowFull bool
	View     string // "list" or "logs"
}

// NewHelp creates a new help component
func NewHelp(s styles.Styles) Help {
	return Help{
		Styles:   s,
		ShowFull: false,
		View:     "list",
	}
}

// SetView sets the current view
func (h *Help) SetView(view string) {
	h.View = view
}

// Toggle toggles between short and full help
func (h *Help) Toggle() {
	h.ShowFull = !h.ShowFull
}

// Render renders the help bar
func (h *Help) Render() string {
	if h.ShowFull {
		return h.renderFull()
	}
	return h.renderShort()
}

// renderShort renders the short help bar
func (h *Help) renderShort() string {
	var items []string

	if h.View == "list" {
		items = []string{
			h.formatKey("↑/k", "up"),
			h.formatKey("↓/j", "down"),
			h.formatKey("enter", "details"),
			h.formatKey("r", "re-run"),
			h.formatKey("c", "cancel"),
			h.formatKey("l", "logs"),
			h.formatKey("o", "open"),
			h.formatKey("/", "filter"),
			h.formatKey("?", "help"),
			h.formatKey("q", "quit"),
		}
	} else {
		items = []string{
			h.formatKey("↑/k", "up"),
			h.formatKey("↓/j", "down"),
			h.formatKey("g/G", "top/bottom"),
			h.formatKey("ctrl+u/d", "½ page"),
			h.formatKey("esc", "back"),
			h.formatKey("q", "quit"),
		}
	}

	return h.Styles.Help.Render(strings.Join(items, "  "))
}

// renderFull renders the full help
func (h *Help) renderFull() string {
	var sb strings.Builder

	sb.WriteString(h.Styles.Title.Render("Keyboard Shortcuts"))
	sb.WriteString("\n\n")

	if h.View == "list" {
		sb.WriteString(h.Styles.Subtitle.Render("Navigation"))
		sb.WriteString("\n")
		sb.WriteString(h.formatKeyFull("↑/k", "Move cursor up"))
		sb.WriteString(h.formatKeyFull("↓/j", "Move cursor down"))
		sb.WriteString(h.formatKeyFull("g/Home", "Go to top"))
		sb.WriteString(h.formatKeyFull("G/End", "Go to bottom"))
		sb.WriteString(h.formatKeyFull("PgUp/ctrl+b", "Page up"))
		sb.WriteString(h.formatKeyFull("PgDn/ctrl+f", "Page down"))
		sb.WriteString("\n")

		sb.WriteString(h.Styles.Subtitle.Render("Actions"))
		sb.WriteString("\n")
		sb.WriteString(h.formatKeyFull("enter", "View run details"))
		sb.WriteString(h.formatKeyFull("l", "View logs"))
		sb.WriteString(h.formatKeyFull("o", "Open in browser"))
		sb.WriteString(h.formatKeyFull("r", "Re-run workflow"))
		sb.WriteString(h.formatKeyFull("c", "Cancel running workflow"))
		sb.WriteString(h.formatKeyFull("R", "Refresh"))
		sb.WriteString("\n")

		sb.WriteString(h.Styles.Subtitle.Render("Filtering"))
		sb.WriteString("\n")
		sb.WriteString(h.formatKeyFull("/", "Open filter/search"))
		sb.WriteString(h.formatKeyFull("tab", "Cycle status filter"))
		sb.WriteString(h.formatKeyFull("esc", "Clear filter"))
		sb.WriteString("\n")
	} else {
		sb.WriteString(h.Styles.Subtitle.Render("Navigation"))
		sb.WriteString("\n")
		sb.WriteString(h.formatKeyFull("↑/k", "Scroll up"))
		sb.WriteString(h.formatKeyFull("↓/j", "Scroll down"))
		sb.WriteString(h.formatKeyFull("g", "Go to top"))
		sb.WriteString(h.formatKeyFull("G", "Go to bottom"))
		sb.WriteString(h.formatKeyFull("ctrl+u", "Half page up"))
		sb.WriteString(h.formatKeyFull("ctrl+d", "Half page down"))
		sb.WriteString(h.formatKeyFull("PgUp", "Page up"))
		sb.WriteString(h.formatKeyFull("PgDn", "Page down"))
		sb.WriteString("\n")
	}

	sb.WriteString(h.Styles.Subtitle.Render("General"))
	sb.WriteString("\n")
	sb.WriteString(h.formatKeyFull("?", "Toggle this help"))
	sb.WriteString(h.formatKeyFull("esc", "Go back / close"))
	sb.WriteString(h.formatKeyFull("q", "Quit"))

	return sb.String()
}

// formatKey formats a key binding for short help
func (h *Help) formatKey(key, desc string) string {
	return h.Styles.HelpKey.Render(key) + " " + h.Styles.HelpDesc.Render(desc)
}

// formatKeyFull formats a key binding for full help
func (h *Help) formatKeyFull(key, desc string) string {
	return "  " + h.Styles.HelpKey.Render(padRight(key, 15)) + h.Styles.Normal.Render(desc) + "\n"
}

// padRight pads a string to the right
func padRight(s string, length int) string {
	if len(s) >= length {
		return s
	}
	return s + strings.Repeat(" ", length-len(s))
}
