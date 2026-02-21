package styles

import "github.com/charmbracelet/lipgloss"

// Colors
var (
	ColorRed     = lipgloss.Color("#FF5555")
	ColorGreen   = lipgloss.Color("#50FA7B")
	ColorYellow  = lipgloss.Color("#F1FA8C")
	ColorBlue    = lipgloss.Color("#8BE9FD")
	ColorPurple  = lipgloss.Color("#BD93F9")
	ColorCyan    = lipgloss.Color("#8BE9FD")
	ColorOrange  = lipgloss.Color("#FFB86C")
	ColorPink    = lipgloss.Color("#FF79C6")
	ColorGray    = lipgloss.Color("#6272A4")
	ColorWhite   = lipgloss.Color("#F8F8F2")
	ColorSubtle  = lipgloss.Color("#44475A")
	ColorBg      = lipgloss.Color("#282A36")
	ColorBgLight = lipgloss.Color("#44475A")
)

// Styles contains all the lipgloss styles for the UI
type Styles struct {
	App           lipgloss.Style
	Title         lipgloss.Style
	Subtitle      lipgloss.Style
	StatusSuccess lipgloss.Style
	StatusFailure lipgloss.Style
	StatusPending lipgloss.Style
	StatusRunning lipgloss.Style
	Selected      lipgloss.Style
	Normal        lipgloss.Style
	Dimmed        lipgloss.Style
	Help          lipgloss.Style
	HelpKey       lipgloss.Style
	HelpDesc      lipgloss.Style
	Error         lipgloss.Style
	Branch        lipgloss.Style
	Repo          lipgloss.Style
	Duration      lipgloss.Style
	LogLine       lipgloss.Style
	LogLineNumber lipgloss.Style
	FilterActive  lipgloss.Style
	Header        lipgloss.Style
	Border        lipgloss.Style
}

// DefaultStyles returns the default styles for the UI
func DefaultStyles() Styles {
	return Styles{
		App: lipgloss.NewStyle().
			Padding(1, 2),

		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPurple).
			MarginBottom(1),

		Subtitle: lipgloss.NewStyle().
			Foreground(ColorGray),

		StatusSuccess: lipgloss.NewStyle().
			Foreground(ColorGreen),

		StatusFailure: lipgloss.NewStyle().
			Foreground(ColorRed),

		StatusPending: lipgloss.NewStyle().
			Foreground(ColorGray),

		StatusRunning: lipgloss.NewStyle().
			Foreground(ColorYellow),

		Selected: lipgloss.NewStyle().
			Bold(true).
			Background(ColorBgLight).
			Foreground(ColorWhite),

		Normal: lipgloss.NewStyle().
			Foreground(ColorWhite),

		Dimmed: lipgloss.NewStyle().
			Foreground(ColorGray),

		Help: lipgloss.NewStyle().
			Foreground(ColorGray).
			MarginTop(1),

		HelpKey: lipgloss.NewStyle().
			Foreground(ColorCyan).
			Bold(true),

		HelpDesc: lipgloss.NewStyle().
			Foreground(ColorGray),

		Error: lipgloss.NewStyle().
			Foreground(ColorRed).
			Bold(true),

		Branch: lipgloss.NewStyle().
			Foreground(ColorPink),

		Repo: lipgloss.NewStyle().
			Foreground(ColorBlue),

		Duration: lipgloss.NewStyle().
			Foreground(ColorOrange),

		LogLine: lipgloss.NewStyle().
			Foreground(ColorWhite),

		LogLineNumber: lipgloss.NewStyle().
			Foreground(ColorGray).
			Width(6).
			Align(lipgloss.Right),

		FilterActive: lipgloss.NewStyle().
			Background(ColorPurple).
			Foreground(ColorWhite).
			Padding(0, 1),

		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPurple).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(ColorSubtle).
			MarginBottom(1),

		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorSubtle),
	}
}

// StatusIcon returns the appropriate icon for a workflow status
func StatusIcon(status, conclusion string) string {
	if status == "completed" {
		switch conclusion {
		case "success":
			return "✓"
		case "failure":
			return "✗"
		case "cancelled":
			return "⊘"
		case "skipped":
			return "⊖"
		default:
			return "?"
		}
	}
	switch status {
	case "in_progress":
		return "●"
	case "queued":
		return "◷"
	case "pending":
		return "○"
	case "waiting":
		return "⚇"
	default:
		return "?"
	}
}

// StatusStyle returns the appropriate style for a workflow status
func (s Styles) StatusStyle(status, conclusion string) lipgloss.Style {
	if status == "completed" {
		switch conclusion {
		case "success":
			return s.StatusSuccess
		case "failure":
			return s.StatusFailure
		default:
			return s.StatusPending
		}
	}
	switch status {
	case "in_progress":
		return s.StatusRunning
	default:
		return s.StatusPending
	}
}
