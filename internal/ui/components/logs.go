package components

import (
	"fmt"
	"strings"

	"github.com/jay-418/gh-ci/internal/ui/styles"
)

// LogViewer is a component for viewing job logs
type LogViewer struct {
	Lines   []string
	Offset  int
	Styles  styles.Styles
	Width   int
	Height  int
	JobName string
	Loading bool
	Error   string
}

// NewLogViewer creates a new log viewer
func NewLogViewer(s styles.Styles) LogViewer {
	return LogViewer{
		Lines:   []string{},
		Offset:  0,
		Styles:  s,
		Loading: false,
	}
}

// SetLogs sets the log content
func (l *LogViewer) SetLogs(content string, jobName string) {
	l.Lines = strings.Split(content, "\n")
	l.JobName = jobName
	l.Offset = 0
	l.Loading = false
	l.Error = ""
}

// SetError sets an error message
func (l *LogViewer) SetError(err string) {
	l.Error = err
	l.Loading = false
}

// SetLoading sets the loading state
func (l *LogViewer) SetLoading(loading bool) {
	l.Loading = loading
}

// SetSize sets the dimensions
func (l *LogViewer) SetSize(width, height int) {
	l.Width = width
	l.Height = height
}

// ScrollUp scrolls up by one line
func (l *LogViewer) ScrollUp() {
	if l.Offset > 0 {
		l.Offset--
	}
}

// ScrollDown scrolls down by one line
func (l *LogViewer) ScrollDown() {
	maxOffset := len(l.Lines) - l.visibleLines()
	if l.Offset < maxOffset {
		l.Offset++
	}
}

// PageUp scrolls up by a page
func (l *LogViewer) PageUp() {
	l.Offset = max(0, l.Offset-l.visibleLines())
}

// PageDown scrolls down by a page
func (l *LogViewer) PageDown() {
	maxOffset := len(l.Lines) - l.visibleLines()
	l.Offset = min(maxOffset, l.Offset+l.visibleLines())
	if l.Offset < 0 {
		l.Offset = 0
	}
}

// HalfPageUp scrolls up by half a page
func (l *LogViewer) HalfPageUp() {
	l.Offset = max(0, l.Offset-l.visibleLines()/2)
}

// HalfPageDown scrolls down by half a page
func (l *LogViewer) HalfPageDown() {
	maxOffset := len(l.Lines) - l.visibleLines()
	l.Offset = min(maxOffset, l.Offset+l.visibleLines()/2)
	if l.Offset < 0 {
		l.Offset = 0
	}
}

// GoToTop goes to the top
func (l *LogViewer) GoToTop() {
	l.Offset = 0
}

// GoToBottom goes to the bottom
func (l *LogViewer) GoToBottom() {
	maxOffset := len(l.Lines) - l.visibleLines()
	if maxOffset < 0 {
		maxOffset = 0
	}
	l.Offset = maxOffset
}

// visibleLines returns the number of visible lines
func (l *LogViewer) visibleLines() int {
	if l.Height < 4 {
		return 10
	}
	return l.Height - 4 // Leave room for header and footer
}

// View renders the log viewer
func (l *LogViewer) View() string {
	var sb strings.Builder

	// Header
	header := fmt.Sprintf("Logs: %s", l.JobName)
	sb.WriteString(l.Styles.Header.Render(header))
	sb.WriteString("\n\n")

	if l.Loading {
		sb.WriteString(l.Styles.Dimmed.Render("Loading logs..."))
		return sb.String()
	}

	if l.Error != "" {
		sb.WriteString(l.Styles.Error.Render("Error: " + l.Error))
		return sb.String()
	}

	if len(l.Lines) == 0 {
		sb.WriteString(l.Styles.Dimmed.Render("No logs available"))
		return sb.String()
	}

	// Log content
	visible := l.visibleLines()
	endIdx := min(l.Offset+visible, len(l.Lines))

	for i := l.Offset; i < endIdx; i++ {
		line := l.Lines[i]
		// Truncate long lines
		maxLineWidth := l.Width - 8
		if maxLineWidth < 40 {
			maxLineWidth = 80
		}
		if len(line) > maxLineWidth {
			line = line[:maxLineWidth-3] + "..."
		}

		// Line number
		lineNum := l.Styles.LogLineNumber.Render(fmt.Sprintf("%5d ", i+1))
		lineContent := l.Styles.LogLine.Render(line)
		sb.WriteString(lineNum + lineContent + "\n")
	}

	// Scroll position
	if len(l.Lines) > visible {
		scrollInfo := fmt.Sprintf("\nLine %d-%d of %d", l.Offset+1, endIdx, len(l.Lines))
		sb.WriteString(l.Styles.Dimmed.Render(scrollInfo))
	}

	return sb.String()
}
