package styles

import "github.com/charmbracelet/lipgloss"

// MessageType represents the type of status message
type MessageType int

const (
	MessageTypeInfo MessageType = iota
	MessageTypeSuccess
	MessageTypeError
)

// Styles contains all the lipgloss styles for the UI
type Styles struct {
	P              Palette        // embedded palette for ad-hoc color access
	App            lipgloss.Style
	Title          lipgloss.Style
	Subtitle       lipgloss.Style
	StatusSuccess  lipgloss.Style
	StatusFailure  lipgloss.Style
	StatusPending  lipgloss.Style
	StatusRunning  lipgloss.Style
	Selected       lipgloss.Style
	Normal         lipgloss.Style
	Dimmed         lipgloss.Style
	Help           lipgloss.Style
	HelpKey        lipgloss.Style
	HelpDesc       lipgloss.Style
	Error          lipgloss.Style
	Branch         lipgloss.Style
	Repo           lipgloss.Style
	Duration       lipgloss.Style
	LogLine        lipgloss.Style
	LogLineNumber  lipgloss.Style
	FilterActive   lipgloss.Style
	Header         lipgloss.Style
	Border         lipgloss.Style
	MessageInfo    lipgloss.Style
	MessageSuccess lipgloss.Style
	MessageError   lipgloss.Style
	DialogPrompt   lipgloss.Style
}

// NewStyles creates a Styles instance from a Palette
func NewStyles(p Palette) Styles {
	return Styles{
		P: p,
		App: lipgloss.NewStyle().
			Padding(1, 2),

		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.Accent).
			MarginBottom(1),

		Subtitle: lipgloss.NewStyle().
			Foreground(p.FgDim),

		StatusSuccess: lipgloss.NewStyle().
			Foreground(p.Success),

		StatusFailure: lipgloss.NewStyle().
			Foreground(p.Failure),

		StatusPending: lipgloss.NewStyle().
			Foreground(p.FgDim),

		StatusRunning: lipgloss.NewStyle().
			Foreground(p.Running),

		Selected: lipgloss.NewStyle().
			Bold(true).
			Background(p.BgLight).
			Foreground(p.Fg),

		Normal: lipgloss.NewStyle().
			Foreground(p.Fg),

		Dimmed: lipgloss.NewStyle().
			Foreground(p.FgDim),

		Help: lipgloss.NewStyle().
			Foreground(p.FgDim).
			MarginTop(1),

		HelpKey: lipgloss.NewStyle().
			Foreground(p.Key).
			Bold(true),

		HelpDesc: lipgloss.NewStyle().
			Foreground(p.FgDim),

		Error: lipgloss.NewStyle().
			Foreground(p.Failure).
			Bold(true),

		Branch: lipgloss.NewStyle().
			Foreground(p.Branch),

		Repo: lipgloss.NewStyle().
			Foreground(p.Repo),

		Duration: lipgloss.NewStyle().
			Foreground(p.Duration),

		LogLine: lipgloss.NewStyle().
			Foreground(p.Fg),

		LogLineNumber: lipgloss.NewStyle().
			Foreground(p.FgDim).
			Width(6).
			Align(lipgloss.Right),

		FilterActive: lipgloss.NewStyle().
			Background(p.Accent).
			Foreground(p.Fg).
			Padding(0, 1),

		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.Accent).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(p.Subtle).
			MarginBottom(1),

		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(p.Subtle),

		MessageInfo: lipgloss.NewStyle().
			Background(p.Accent).
			Foreground(p.Bg).
			Padding(0, 1),

		MessageSuccess: lipgloss.NewStyle().
			Background(p.Success).
			Foreground(p.Bg).
			Padding(0, 1),

		MessageError: lipgloss.NewStyle().
			Background(p.Failure).
			Foreground(p.Bg).
			Padding(0, 1),

		DialogPrompt: lipgloss.NewStyle().
			Bold(true).
			Background(p.Accent).
			Foreground(p.Bg).
			Padding(0, 1),
	}
}

// DefaultStyles returns the default styles for the UI (Dracula theme)
func DefaultStyles() Styles {
	return NewStyles(ThemeByName("dracula").Palette)
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

// MessageStyle returns the appropriate style for a message type
func (s Styles) MessageStyle(t MessageType) lipgloss.Style {
	switch t {
	case MessageTypeSuccess:
		return s.MessageSuccess
	case MessageTypeError:
		return s.MessageError
	default:
		return s.MessageInfo
	}
}
