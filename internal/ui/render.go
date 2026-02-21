package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"github.com/turkosaurus/gh-ci/internal/gh"
	"github.com/turkosaurus/gh-ci/internal/types"
	"github.com/turkosaurus/gh-ci/internal/ui/styles"
)

// bindingHelp renders a single key binding as a "key  desc" help item.
func bindingHelp(s styles.Styles, b key.Binding) string {
	return s.HelpKey.Render(b.Help().Key) + " " + s.HelpDesc.Render(b.Help().Desc)
}

func (m Model) View() string {
	if m.loading && len(m.allRuns) == 0 {
		return m.styles.Dimmed.Render("loading workflow runs...")
	}
	if m.screen == ScreenLogs {
		return renderLogs(m)
	}
	return renderMain(m)
}

func renderMain(m Model) string {
	w, h := m.width, m.height
	if w == 0 {
		w = 80
	}
	if h == 0 {
		h = 24
	}

	bodyH := h - 3 // title + panel-headers + help

	const workflowW = 22
	const maxDetailW = 40
	detailW := min(maxDetailW, w*30/100)
	runsW := w - workflowW - detailW - 2 // 2 separators

	sep := lipgloss.NewStyle().
		Foreground(styles.ColorSubtle).
		Render(strings.Repeat("│\n", bodyH-1) + "│")

	body := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(workflowW).Height(bodyH).Render(renderWorkflows(m, workflowW, bodyH)),
		sep,
		lipgloss.NewStyle().Width(runsW).Height(bodyH).Render(renderList(m, runsW, bodyH)),
		sep,
		lipgloss.NewStyle().Width(detailW).Height(bodyH).Render(renderDetail(m, detailW)),
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		renderTitle(m, w),
		renderPanelHeaders(m, workflowW, runsW, detailW),
		body,
		renderHelpBar(m, w),
	)
}

func renderTitle(m Model, width int) string {
	return lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPurple).
		Render("GitHub Actions Dashboard")
}

func renderPanelHeaders(m Model, workflowW, runsW, detailW int) string {
	sep := lipgloss.NewStyle().Background(styles.ColorBgLight).Foreground(styles.ColorSubtle).Render("│")
	label := func(panel int, text string, w int) string {
		style := lipgloss.NewStyle().
			Width(w).
			Align(lipgloss.Center) // Center the label text
		if m.activePanel == panel {
			style = style.Bold(true).
				Background(styles.ColorPurple).Foreground(styles.ColorBg)
		} else {
			style = style.Background(styles.ColorBgLight).Foreground(styles.ColorWhite)
		}
		return style.Render(text)
	}
	return lipgloss.NewStyle().
		Width(workflowW + runsW + detailW + 2). // 2 for separators
		Align(lipgloss.Center).
		Render(
			lipgloss.JoinHorizontal(lipgloss.Top,
				label(0, "WORKFLOWS", workflowW),
				sep,
				label(1, "RUNS", runsW),
				sep,
				label(2, "DETAIL", detailW),
			),
		)
}

func renderWorkflows(m Model, width, height int) string {
	active := m.activePanel == 0

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorGray)
	if active {
		headerStyle = headerStyle.Foreground(styles.ColorPurple)
	}
	selectedStyle := lipgloss.NewStyle().Bold(true).Background(styles.ColorBgLight).Foreground(styles.ColorWhite)

	const maxSugg = 4
	var rows []string

	// ── REPO section (display only) ──────────────────────────────────────────
	rows = append(rows, headerStyle.Render("REPO"))
	repoDisplay := strings.Join(m.config.Repos, ", ")
	rows = append(rows, m.styles.Repo.Render(fmt.Sprintf("%-*s", width-2, gh.TruncateString(repoDisplay, width-2))))

	// Separator
	rows = append(rows, m.styles.Dimmed.Render(strings.Repeat("─", width-1)))

	// ── BRANCH section ──────────────────────────────────────────────────────
	rows = append(rows, headerStyle.Render("BRANCH"))

	branchDisplay := m.defaultBranch
	if m.branchIdx < len(m.availableBranches) {
		branchDisplay = m.availableBranches[m.branchIdx]
	}

	if m.branchSelecting {
		rows = append(rows, m.branchInput.View())
		suggestions := m.filteredBranches()
		limit := maxSugg
		if limit > len(suggestions) {
			limit = len(suggestions)
		}
		for i, b := range suggestions[:limit] {
			if i == m.branchSuggestionCursor {
				rows = append(rows, selectedStyle.Render("> "+gh.TruncateString(b, width-4)))
			} else {
				rows = append(rows, m.styles.Dimmed.Render("  "+gh.TruncateString(b, width-4)))
			}
		}
	} else {
		text := fmt.Sprintf("%-*s", width-2, gh.TruncateString(branchDisplay, width-2))
		if m.workflowCursor == 0 && active {
			rows = append(rows, selectedStyle.Render(text))
		} else {
			rows = append(rows, m.styles.Branch.Render(text))
		}
	}

	// Separator
	rows = append(rows, m.styles.Dimmed.Render(strings.Repeat("─", width-1)))

	// ── NAME section ─────────────────────────────────────────────────────────
	rows = append(rows, headerStyle.Render("NAME"))

	if len(m.workflows) == 0 {
		rows = append(rows, m.styles.Dimmed.Render("loading..."))
		return strings.Join(rows, "\n")
	}

	// Check if we have a filename to pin at the bottom
	var filenameStr string
	if m.workflowCursor > 0 && m.workflowCursor <= len(m.workflows) {
		wfName := m.workflows[m.workflowCursor-1]
		if wfName != workflowAll {
			filenameStr = m.workflowFiles[wfName]
		}
	}

	branchSectionH := len(rows)
	workflowListH := height - branchSectionH
	if filenameStr != "" {
		workflowListH--
	}
	if workflowListH < 1 {
		workflowListH = 1
	}

	// wfCursor: index within m.workflows for scroll calculation
	// cursor scheme: 0=branch, 1..N=workflows[0..N-1]
	wfCursor := 0
	if m.workflowCursor > 0 {
		wfCursor = m.workflowCursor - 1
	}
	startIdx := 0
	if wfCursor >= workflowListH {
		startIdx = wfCursor - workflowListH + 1
	}
	endIdx := min(startIdx+workflowListH, len(m.workflows))

	for i := startIdx; i < endIdx; i++ {
		wfName := m.workflows[i]
		selected := (i + 1) == m.workflowCursor
		text := fmt.Sprintf("%-*s", width-2, gh.TruncateString(wfName, width-2))
		var row string
		switch {
		case selected && active:
			row = selectedStyle.Render(text)
		case selected:
			row = lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPurple).Render(text)
		default:
			row = m.styles.Normal.Render(text)
		}
		rows = append(rows, row)
	}

	if filenameStr != "" {
		for len(rows) < height-1 {
			rows = append(rows, "")
		}
		rows = append(rows, m.styles.Dimmed.Render(gh.TruncateString(filenameStr, width-2)))
	}

	return strings.Join(rows, "\n")
}

func renderList(m Model, width, height int) string {
	active := m.activePanel == 1

	if len(m.filteredRuns) == 0 {
		return m.styles.Dimmed.Render("no workflow runs")
	}

	const colOk, colNum, colDur, colFile, colDispatched = 2, 6, 7, 14, 16
	colWorkflow := width - colOk - colNum - colDur - colFile - colDispatched - 10
	if colWorkflow < 10 {
		colWorkflow = 10
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorGray)
	if active {
		headerStyle = headerStyle.Foreground(styles.ColorPurple)
	}
	header := fmt.Sprintf("%-*s  %-*s  %-*s  %*s  %-*s  %-*s",
		colDispatched, "DISPATCHED",
		colFile, "FILE",
		colWorkflow, "NAME",
		colNum, "RUN",
		colDur, "TIME",
		colOk, "OK",
	)
	rows := []string{headerStyle.Render(header)}

	listH := height - 2
	startIdx := 0
	if m.cursor >= listH {
		startIdx = m.cursor - listH + 1
	}
	endIdx := min(startIdx+listH, len(m.filteredRuns))

	for i := startIdx; i < endIdx; i++ {
		rows = append(rows, renderRunRow(m, m.filteredRuns[i], i == m.cursor, active,
			width, colWorkflow, colFile, colNum, colDur, colDispatched, colOk))
	}

	if len(m.filteredRuns) > listH {
		rows = append(rows, m.styles.Dimmed.Render(
			fmt.Sprintf(" %d/%d", m.cursor+1, len(m.filteredRuns))))
	}

	return strings.Join(rows, "\n")
}

func renderRunRow(m Model, run types.WorkflowRun, selected, active bool, width, colWorkflow, colFile, colNum, colDur, colDispatched, colOk int) string {
	icon := styles.StatusIcon(run.Status, run.Conclusion)
	iconS := fmt.Sprintf("%-*s", colOk, icon)
	wfS := fmt.Sprintf("%-*s", colWorkflow, gh.TruncateString(run.Name, colWorkflow))
	fileS := fmt.Sprintf("%-*s", colFile, gh.TruncateString(m.workflowFiles[run.Name], colFile))
	numS := fmt.Sprintf("%*s", colNum, fmt.Sprintf("#%d", run.RunNumber))
	durS := fmt.Sprintf("%-*s", colDur, gh.FormatDuration(int64(run.Duration().Seconds())))
	dispS := fmt.Sprintf("%-*s", colDispatched, run.CreatedAt.Format("2006-01-02 15:04"))

	if selected && active {
		// Per-element styles with shared background so status/duration colors are preserved.
		bg := lipgloss.NewStyle().Background(styles.ColorBgLight)
		sep := bg.Render("  ")
		disp := m.styles.Dimmed.Background(styles.ColorBgLight).Render(dispS)
		wf := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorWhite).Background(styles.ColorBgLight).Render(wfS)
		file := m.styles.Dimmed.Background(styles.ColorBgLight).Render(fileS)
		num := lipgloss.NewStyle().Foreground(styles.ColorWhite).Background(styles.ColorBgLight).Render(numS)
		dur := m.styles.Duration.Background(styles.ColorBgLight).Render(durS)
		st := m.styles.StatusStyle(run.Status, run.Conclusion).Background(styles.ColorBgLight).Render(iconS)
		row := disp + sep + file + sep + wf + sep + num + sep + dur + sep + st
		// pad remaining width with background so the bar extends to the edge
		used := colDispatched + 2 + colWorkflow + 2 + colFile + 2 + colNum + 2 + colDur + 2 + colOk
		if pad := width - used; pad > 0 {
			row += bg.Render(strings.Repeat(" ", pad))
		}
		return row
	}

	if selected {
		plainRow := dispS + "  " + fileS + "  " + wfS + "  " + numS + "  " + durS + "  " + iconS
		return lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPurple).Render(plainRow)
	}

	// normal: per-element styles
	st := m.styles.StatusStyle(run.Status, run.Conclusion).Render(iconS)
	file := m.styles.Dimmed.Render(fileS)
	dur := m.styles.Duration.Render(durS)
	disp := m.styles.Dimmed.Render(dispS)
	return disp + "  " + file + "  " + wfS + "  " + numS + "  " + dur + "  " + st
}

func renderDetail(m Model, width int) string {
	active := m.activePanel == 2

	run := m.selectedRun()
	if run == nil {
		return m.styles.Dimmed.Render("no run selected")
	}

	var sb strings.Builder

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorGray)
	if active {
		headerStyle = headerStyle.Foreground(styles.ColorPurple)
	}
	sb.WriteString(headerStyle.Render(fmt.Sprintf("[#%d] %s", run.RunNumber, gh.TruncateString(run.Name, width-10))))
	sb.WriteString("\n\n")

	field := func(label, value string) {
		sb.WriteString(m.styles.Dimmed.Render(fmt.Sprintf("%-8s", label)))
		sb.WriteString(value + "\n")
	}

	sha := run.HeadSHA
	if len(sha) > 8 {
		sha = sha[:8]
	}
	statusStyle := m.styles.StatusStyle(run.Status, run.Conclusion)
	icon := styles.StatusIcon(run.Status, run.Conclusion)
	dur := gh.FormatDuration(int64(run.Duration().Seconds()))

	field("repo", m.styles.Repo.Render(gh.TruncateString(run.Repository.FullName, width-10)))
	field("branch", m.styles.Branch.Render(run.HeadBranch))
	field("commit", m.styles.Normal.Render(sha))
	field("status", statusStyle.Render(icon+" "+run.GetStatus())+"  "+m.styles.Duration.Render(dur))

	sb.WriteString("\n")

	jobsHeaderStyle := m.styles.Dimmed
	if active {
		jobsHeaderStyle = lipgloss.NewStyle().Foreground(styles.ColorPurple)
	}
	sb.WriteString(jobsHeaderStyle.Render("jobs") + "\n")

	if len(m.jobs) == 0 {
		sb.WriteString("  " + m.styles.Dimmed.Render("loading..."))
	} else {
		for i, job := range m.jobs {
			jIcon := styles.StatusIcon(job.Status, job.Conclusion)
			name := gh.TruncateString(job.Name, width-5)
			var line string
			switch {
			case i == m.jobCursor && active:
				line = lipgloss.NewStyle().Bold(true).Background(styles.ColorBgLight).Foreground(styles.ColorWhite).
					Render(fmt.Sprintf("  %s %s", jIcon, name))
			case i == m.jobCursor:
				line = lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPurple).
					Render(fmt.Sprintf("  %s %s", jIcon, name))
			default:
				jStyle := m.styles.StatusStyle(job.Status, job.Conclusion)
				line = "  " + jStyle.Render(jIcon) + " " + name
			}
			sb.WriteString(line + "\n")
		}
	}

	return sb.String()
}

func renderHelpBar(m Model, width int) string {
	if m.confirming {
		return m.styles.Normal.Render("re-run?") + "  " +
			m.styles.HelpKey.Render("y") + " " + m.styles.HelpDesc.Render("normal") + "  " +
			m.styles.HelpKey.Render("d") + " " + m.styles.HelpDesc.Render("debug logs") + "  " +
			m.styles.HelpKey.Render("esc") + " " + m.styles.HelpDesc.Render("cancel")
	}

	if m.dispatchConfirming {
		return m.styles.Normal.Render("dispatch "+m.dispatchFile+" on "+m.dispatchRef+"?") + "  " +
			m.styles.HelpKey.Render("y") + " " + m.styles.HelpDesc.Render("yes") + "  " +
			m.styles.HelpKey.Render("esc") + " " + m.styles.HelpDesc.Render("cancel")
	}

	if m.branchSelecting {
		return m.styles.Dimmed.Render("↑/↓ navigate  ↵ confirm  esc cancel")
	}

	if m.message != "" {
		return m.styles.Dimmed.Render(m.message)
	}

	var items []string
	if run := m.selectedRun(); run != nil {
		items = append(items, bindingHelp(m.styles, m.keys.Rerun))
		if run.Status == "in_progress" {
			items = append(items, bindingHelp(m.styles, m.keys.Cancel))
		}
	}
	if m.activePanel == 0 && m.workflowCursor > 0 && m.workflowCursor <= len(m.workflows) {
		if wfName := m.workflows[m.workflowCursor-1]; wfName != workflowAll {
			if _, ok := m.workflowFiles[wfName]; ok {
				items = append(items, bindingHelp(m.styles, m.keys.Dispatch))
			}
		}
	}
	items = append(items, bindingHelp(m.styles, m.keys.Open))

	left := strings.Join(items, "  ")
	right := bindingHelp(m.styles, m.keys.Quit)
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 2 {
		gap = 2
	}
	return left + strings.Repeat(" ", gap) + right
}

func renderLogs(m Model) string {
	w, h := m.width, m.height
	if w == 0 {
		w = 80
	}
	if h == 0 {
		h = 24
	}

	visibleLines := h - 4
	maxLineW := w - 8
	if maxLineW < 40 {
		maxLineW = 40
	}

	var sb strings.Builder

	if m.logQuery != "" && len(m.logContextLines) > 0 {
		// ── context-window mode ──────────────────────────────────────────────
		total := len(m.logContextLines)
		end := min(m.logOffset+visibleLines, total)
		scrollInfo := fmt.Sprintf("%d-%d / %d", m.logOffset+1, end, total)
		matchInfo := fmt.Sprintf("[/%s  match %d/%d]", m.logQuery, m.logMatchIdx+1, len(m.logMatchGroups))
		header := fmt.Sprintf("Logs: %s  %s", m.logJobName, m.styles.Dimmed.Render(matchInfo))
		hGap := w - lipgloss.Width(header) - len(scrollInfo) - 2
		if hGap < 1 {
			hGap = 1
		}
		sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPurple).
			Render("Logs: "+m.logJobName) + "  " + m.styles.Dimmed.Render(matchInfo) +
			strings.Repeat(" ", hGap) + m.styles.Dimmed.Render(scrollInfo))
		sb.WriteString("\n\n")

		for i := m.logOffset; i < end; i++ {
			cl := m.logContextLines[i]
			if cl.lineNo == 0 {
				sb.WriteString("\n")
				continue
			}
			text := cl.text
			if len(text) > maxLineW {
				text = text[:maxLineW-3] + "..."
			}
			numStr := m.styles.LogLineNumber.Render(fmt.Sprintf("%5d ", cl.lineNo))
			if cl.isMatch {
				sb.WriteString(numStr)
				sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(styles.ColorYellow).Render(text))
			} else {
				sb.WriteString(numStr)
				sb.WriteString(m.styles.Dimmed.Render(text))
			}
			sb.WriteString("\n")
		}
	} else if m.logQuery != "" {
		// query active but no matches
		sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPurple).
			Render("Logs: " + m.logJobName))
		sb.WriteString("\n\n")
		sb.WriteString(m.styles.Dimmed.Render(fmt.Sprintf("no matches for /%s", m.logQuery)))
		sb.WriteString("\n")
	} else {
		// ── normal (no filter) mode ──────────────────────────────────────────
		logLines := strings.Split(m.logs, "\n")
		end := min(m.logOffset+visibleLines, len(logLines))
		scrollInfo := fmt.Sprintf("%d-%d / %d", m.logOffset+1, end, len(logLines))
		header := fmt.Sprintf("Logs: %s", m.logJobName)
		hGap := w - len(header) - len(scrollInfo) - 2
		if hGap < 1 {
			hGap = 1
		}
		sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPurple).
			Render(header + strings.Repeat(" ", hGap) + scrollInfo))
		sb.WriteString("\n\n")

		for i := m.logOffset; i < end; i++ {
			line := logLines[i]
			if len(line) > maxLineW {
				line = line[:maxLineW-3] + "..."
			}
			sb.WriteString(m.styles.LogLineNumber.Render(fmt.Sprintf("%5d ", i+1)))
			sb.WriteString(m.styles.LogLine.Render(line))
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n")
	if m.logSearching {
		prompt := m.styles.HelpKey.Render("/") + " " + m.textInput.View()
		esc := m.styles.Dimmed.Render("esc to cancel")
		gap := w - lipgloss.Width(prompt) - lipgloss.Width(esc) - 2
		if gap < 1 {
			gap = 1
		}
		sb.WriteString(prompt + strings.Repeat(" ", gap) + esc)
	} else if m.logQuery != "" {
		helpItems := []string{
			bindingHelp(m.styles, m.keys.SearchNext),
			bindingHelp(m.styles, m.keys.SearchPrev),
			m.styles.HelpKey.Render("↑/↓") + " " + m.styles.HelpDesc.Render("scroll"),
			m.styles.HelpKey.Render(m.keys.Search.Help().Key) + " " + m.styles.HelpDesc.Render("new search"),
			m.styles.HelpKey.Render("h/esc") + " " + m.styles.HelpDesc.Render("back"),
			bindingHelp(m.styles, m.keys.Quit),
		}
		sb.WriteString(m.styles.Dimmed.Render(strings.Join(helpItems, "  ")))
	} else {
		helpItems := []string{
			bindingHelp(m.styles, m.keys.Up),
			bindingHelp(m.styles, m.keys.Down),
			m.styles.HelpKey.Render("g/G") + " " + m.styles.HelpDesc.Render("top/bottom"),
			m.styles.HelpKey.Render("ctrl+u/d") + " " + m.styles.HelpDesc.Render("½ page"),
			bindingHelp(m.styles, m.keys.Search),
			m.styles.HelpKey.Render("h/esc/⌫") + " " + m.styles.HelpDesc.Render("back"),
			bindingHelp(m.styles, m.keys.Quit),
		}
		sb.WriteString(m.styles.Dimmed.Render(strings.Join(helpItems, "  ")))
	}

	return sb.String()
}
