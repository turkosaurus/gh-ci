package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/turkosaurus/gh-ci/internal/gh"
	"github.com/turkosaurus/gh-ci/internal/ui/keys"
	"github.com/turkosaurus/gh-ci/internal/ui/styles"
)

// LogViewer is the full-screen log viewing screen.
type LogViewer struct {
	logs         string
	jobName      string
	logOffset    int
	logQuery     string
	searching    bool
	textInput    textinput.Model
	contextLines []logContextLine
	matchGroups  []int
	matchIdx     int

	styles styles.Styles
	keys   keys.KeyMap
}

func NewLogViewer(s styles.Styles, k keys.KeyMap) LogViewer {
	ti := textinput.New()
	ti.Placeholder = "search logs..."
	ti.CharLimit = 100
	return LogViewer{
		styles:    s,
		keys:      k,
		textInput: ti,
	}
}

// SetLogs populates the viewer with new log data.
func (lv *LogViewer) SetLogs(logs, jobName string) {
	lv.logs = logs
	lv.jobName = jobName
	lv.logOffset = 0
	lv.logQuery = ""
	lv.searching = false
	lv.contextLines = nil
	lv.matchGroups = nil
	lv.matchIdx = 0
}

// Update handles key events for the log viewer.
func (lv LogViewer) Update(msg tea.KeyMsg, height int) (LogViewer, tea.Cmd) {
	if lv.searching {
		return lv.handleSearch(msg)
	}

	var displayLen int
	if lv.logQuery != "" {
		displayLen = len(lv.contextLines)
	} else {
		displayLen = len(strings.Split(lv.logs, "\n"))
	}
	visibleLines := height - logViewOverhead
	if visibleLines < 1 {
		visibleLines = 1
	}

	switch {
	case key.Matches(msg, lv.keys.Quit):
		return lv, tea.Quit

	case key.Matches(msg, lv.keys.Back), key.Matches(msg, lv.keys.Left), msg.Type == tea.KeyBackspace:
		lv.logQuery = ""
		lv.searching = false
		lv.contextLines = nil
		lv.matchGroups = nil
		lv.matchIdx = 0
		return lv, func() tea.Msg { return backToMainMsg{} }

	case key.Matches(msg, lv.keys.Search):
		lv.searching = true
		lv.textInput.SetValue("")
		lv.textInput.Focus()
		return lv, textinput.Blink

	case key.Matches(msg, lv.keys.SearchNext):
		if lv.logQuery != "" && lv.matchIdx < len(lv.matchGroups)-1 {
			lv.matchIdx++
			lv.logOffset = lv.matchGroups[lv.matchIdx]
		}

	case key.Matches(msg, lv.keys.SearchPrev):
		if lv.logQuery != "" && lv.matchIdx > 0 {
			lv.matchIdx--
			lv.logOffset = lv.matchGroups[lv.matchIdx]
		}

	case key.Matches(msg, lv.keys.Up):
		if lv.logOffset > 0 {
			lv.logOffset--
		}

	case key.Matches(msg, lv.keys.Down):
		if maxOffset := displayLen - visibleLines; lv.logOffset < maxOffset {
			lv.logOffset++
		}

	case key.Matches(msg, lv.keys.PageUp):
		lv.logOffset = max(0, lv.logOffset-visibleLines)

	case key.Matches(msg, lv.keys.PageDown):
		lv.logOffset = min(max(0, displayLen-visibleLines), lv.logOffset+visibleLines)

	case key.Matches(msg, lv.keys.HalfPageUp):
		lv.logOffset = max(0, lv.logOffset-visibleLines/2)

	case key.Matches(msg, lv.keys.HalfPageDown):
		lv.logOffset = min(max(0, displayLen-visibleLines), lv.logOffset+visibleLines/2)

	case key.Matches(msg, lv.keys.Top):
		lv.logOffset = 0

	case key.Matches(msg, lv.keys.Bottom):
		lv.logOffset = max(0, displayLen-visibleLines)
	}

	return lv, nil
}

func (lv LogViewer) handleSearch(msg tea.KeyMsg) (LogViewer, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		lv.searching = false
		lv.textInput.Blur()
		return lv, nil
	case tea.KeyEnter:
		lv.logQuery = lv.textInput.Value()
		lv.searching = false
		lv.textInput.Blur()
		lv.logOffset = 0
		lv.matchIdx = 0
		if lv.logQuery != "" {
			lines := strings.Split(lv.logs, "\n")
			lv.contextLines, lv.matchGroups = buildLogContext(lines, lv.logQuery, 3)
		} else {
			lv.contextLines = nil
			lv.matchGroups = nil
		}
		return lv, nil
	}
	var cmd tea.Cmd
	lv.textInput, cmd = lv.textInput.Update(msg)
	return lv, cmd
}

// View renders the complete log screen.
func (lv LogViewer) View(width, height int) string {
	w, h := width, height
	if w == 0 {
		w = 80
	}
	if h == 0 {
		h = 24
	}

	visibleLines := h - logViewOverhead
	if visibleLines < 1 {
		visibleLines = 1
	}
	maxLineW := w - 8
	if maxLineW < 40 {
		maxLineW = 40
	}

	var sb strings.Builder

	if lv.logQuery != "" && len(lv.contextLines) > 0 {
		// context-window mode
		total := len(lv.contextLines)
		end := min(lv.logOffset+visibleLines, total)
		scrollInfo := fmt.Sprintf("%d-%d / %d", lv.logOffset+1, end, total)
		matchInfo := fmt.Sprintf("[/%s  match %d/%d]", lv.logQuery, lv.matchIdx+1, len(lv.matchGroups))
		header := fmt.Sprintf("Logs: %s  %s", lv.jobName, lv.styles.Dimmed.Render(matchInfo))
		hGap := w - lipgloss.Width(header) - len(scrollInfo) - 2
		if hGap < 1 {
			hGap = 1
		}
		sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPurple).
			Render("Logs: "+lv.jobName) + "  " + lv.styles.Dimmed.Render(matchInfo) +
			strings.Repeat(" ", hGap) + lv.styles.Dimmed.Render(scrollInfo))
		sb.WriteString("\n\n")

		for i := lv.logOffset; i < end; i++ {
			cl := lv.contextLines[i]
			if cl.lineNo == 0 {
				sb.WriteString("\n")
				continue
			}
			text := gh.TruncateString(cl.text, maxLineW)
			numStr := lv.styles.LogLineNumber.Render(fmt.Sprintf("%5d ", cl.lineNo))
			if cl.isMatch {
				sb.WriteString(numStr)
				sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(styles.ColorYellow).Render(text))
			} else {
				sb.WriteString(numStr)
				sb.WriteString(lv.styles.Dimmed.Render(text))
			}
			sb.WriteString("\n")
		}
	} else if lv.logQuery != "" {
		// query active but no matches
		sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPurple).
			Render("Logs: " + lv.jobName))
		sb.WriteString("\n\n")
		sb.WriteString(lv.styles.Dimmed.Render(fmt.Sprintf("no matches for /%s", lv.logQuery)))
		sb.WriteString("\n")
	} else {
		// normal (no filter) mode
		logLines := strings.Split(lv.logs, "\n")
		end := min(lv.logOffset+visibleLines, len(logLines))
		scrollInfo := fmt.Sprintf("%d-%d / %d", lv.logOffset+1, end, len(logLines))
		header := fmt.Sprintf("Logs: %s", lv.jobName)
		hGap := w - len(header) - len(scrollInfo) - 2
		if hGap < 1 {
			hGap = 1
		}
		sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPurple).
			Render(header + strings.Repeat(" ", hGap) + scrollInfo))
		sb.WriteString("\n\n")

		for i := lv.logOffset; i < end; i++ {
			line := gh.TruncateString(logLines[i], maxLineW)
			sb.WriteString(lv.styles.LogLineNumber.Render(fmt.Sprintf("%5d ", i+1)))
			sb.WriteString(lv.styles.LogLine.Render(line))
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n")
	if lv.searching {
		prompt := lv.styles.HelpKey.Render("/") + " " + lv.textInput.View()
		esc := lv.styles.Dimmed.Render("esc to cancel")
		gap := w - lipgloss.Width(prompt) - lipgloss.Width(esc) - 2
		if gap < 1 {
			gap = 1
		}
		sb.WriteString(prompt + strings.Repeat(" ", gap) + esc)
	} else if lv.logQuery != "" {
		helpItems := []string{
			bindingHelp(lv.styles, lv.keys.SearchNext),
			bindingHelp(lv.styles, lv.keys.SearchPrev),
			lv.styles.HelpKey.Render("↑/↓") + " " + lv.styles.HelpDesc.Render("scroll"),
			lv.styles.HelpKey.Render(lv.keys.Search.Help().Key) + " " + lv.styles.HelpDesc.Render("new search"),
			lv.styles.HelpKey.Render("h/esc") + " " + lv.styles.HelpDesc.Render("back"),
			bindingHelp(lv.styles, lv.keys.Quit),
		}
		sb.WriteString(lv.styles.Dimmed.Render(strings.Join(helpItems, "  ")))
	} else {
		helpItems := []string{
			bindingHelp(lv.styles, lv.keys.Up),
			bindingHelp(lv.styles, lv.keys.Down),
			lv.styles.HelpKey.Render("g/G") + " " + lv.styles.HelpDesc.Render("top/bottom"),
			lv.styles.HelpKey.Render("ctrl+u/d") + " " + lv.styles.HelpDesc.Render("½ page"),
			bindingHelp(lv.styles, lv.keys.Search),
			lv.styles.HelpKey.Render("h/esc/⌫") + " " + lv.styles.HelpDesc.Render("back"),
			bindingHelp(lv.styles, lv.keys.Quit),
		}
		sb.WriteString(lv.styles.Dimmed.Render(strings.Join(helpItems, "  ")))
	}

	return sb.String()
}
