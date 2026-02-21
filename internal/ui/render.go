package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/turkosaurus/gh-ci/internal/gh"
	"github.com/turkosaurus/gh-ci/internal/types"
	"github.com/turkosaurus/gh-ci/internal/ui/styles"
)

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

	bodyH := h - 2 // title + help

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
		body,
		renderHelpBar(m, w),
	)
}

func renderTitle(m Model, width int) string {
	return lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPurple).
		Render("GitHub Actions Dashboard")
}

func renderWorkflows(m Model, width, height int) string {
	active := m.activePanel == 0

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorGray)
	if active {
		headerStyle = headerStyle.Foreground(styles.ColorPurple)
	}
	rows := []string{headerStyle.Render("WORKFLOW")}

	if len(m.workflows) == 0 {
		rows = append(rows, m.styles.Dimmed.Render("loading..."))
		return strings.Join(rows, "\n")
	}

	// Check if we have a filename to pin at the bottom (only for specific workflow selections)
	var filenameStr string
	if m.workflowCursor > 0 && m.workflowCursor <= len(m.workflows) {
		wfName := m.workflows[m.workflowCursor-1]
		if wfName != "all" {
			filenameStr = m.workflowFiles[wfName]
		}
	}

	listH := height - 1
	if filenameStr != "" {
		listH = height - 2
	}

	totalItems := 1 + len(m.workflows) // branch cell + workflow rows
	startIdx := 0
	if m.workflowCursor >= listH {
		startIdx = m.workflowCursor - listH + 1
	}
	endIdx := min(startIdx+listH, totalItems)

	// Branch display text
	branchDisplay := "all branches"
	if m.branchIdx > 0 && m.branchIdx < len(m.availableBranches) {
		branchDisplay = m.availableBranches[m.branchIdx]
	}

	for i := startIdx; i < endIdx; i++ {
		if i == 0 {
			// Branch cell (position 0)
			selected := m.workflowCursor == 0
			text := fmt.Sprintf("%-*s", width-2, gh.TruncateString(branchDisplay, width-2))
			var row string
			switch {
			case selected && active:
				row = lipgloss.NewStyle().Bold(true).Background(styles.ColorBgLight).Foreground(styles.ColorWhite).Render(text)
			case selected:
				row = lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPurple).Render(text)
			default:
				row = m.styles.Branch.Render(text)
			}
			rows = append(rows, row)
		} else {
			// Workflow row: i maps to workflows[i-1]
			wfName := m.workflows[i-1]
			selected := i == m.workflowCursor
			text := fmt.Sprintf("%-*s", width-2, gh.TruncateString(wfName, width-2))
			var row string
			switch {
			case selected && active:
				row = lipgloss.NewStyle().Bold(true).Background(styles.ColorBgLight).Foreground(styles.ColorWhite).Render(text)
			case selected:
				row = lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPurple).Render(text)
			default:
				row = m.styles.Normal.Render(text)
			}
			rows = append(rows, row)
		}
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
		if m.searchQuery != "" {
			return m.styles.Dimmed.Render("no runs match filter")
		}
		return m.styles.Dimmed.Render("no workflow runs")
	}

	const colSt, colNum, colDur, colBranch, colWorkflow = 2, 6, 9, 14, 18
	colRepo := width - colSt - colNum - colDur - colBranch - colWorkflow - 5
	if colRepo < 10 {
		colRepo = 10
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorGray)
	if active {
		headerStyle = headerStyle.Foreground(styles.ColorPurple)
	}
	header := fmt.Sprintf("  %-*s  %-*s  %-*s  %*s  %-*s",
		colRepo, "REPO",
		colWorkflow, "WORKFLOW",
		colBranch, "BRANCH",
		colNum, "RUN",
		colDur, "TIME",
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
			colRepo, colWorkflow, colBranch, colNum, colDur))
	}

	if len(m.filteredRuns) > listH {
		rows = append(rows, m.styles.Dimmed.Render(
			fmt.Sprintf(" %d/%d", m.cursor+1, len(m.filteredRuns))))
	}

	return strings.Join(rows, "\n")
}

func renderRunRow(m Model, run types.WorkflowRun, selected, active bool, colRepo, colWorkflow, colBranch, colNum, colDur int) string {
	icon := styles.StatusIcon(run.Status, run.Conclusion)
	st := m.styles.StatusStyle(run.Status, run.Conclusion).Render(icon)
	repo := m.styles.Repo.Render(fmt.Sprintf("%-*s", colRepo, gh.TruncateString(run.Repository.FullName, colRepo)))
	workflow := fmt.Sprintf("%-*s", colWorkflow, gh.TruncateString(run.Name, colWorkflow))
	branch := m.styles.Branch.Render(fmt.Sprintf("%-*s", colBranch, gh.TruncateString(run.HeadBranch, colBranch)))
	num := fmt.Sprintf("%*s", colNum, fmt.Sprintf("#%d", run.RunNumber))
	dur := m.styles.Duration.Render(fmt.Sprintf("%-*s", colDur, gh.FormatDuration(int64(run.Duration().Seconds()))))

	row := st + " " + repo + "  " + workflow + "  " + branch + "  " + num + "  " + dur

	switch {
	case selected && active:
		row = lipgloss.NewStyle().Bold(true).Background(styles.ColorBgLight).Foreground(styles.ColorWhite).Render(row)
	case selected:
		row = lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPurple).Render(row)
	}
	return row
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
	sb.WriteString(headerStyle.Render(fmt.Sprintf("Run #%d: %s", run.RunNumber, gh.TruncateString(run.Name, width-10))))
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

	if m.searching {
		prompt := m.styles.HelpKey.Render("/") + " " + m.textInput.View()
		esc := m.styles.Dimmed.Render("esc to cancel")
		gap := width - lipgloss.Width(prompt) - lipgloss.Width(esc) - 2
		if gap < 1 {
			gap = 1
		}
		return prompt + strings.Repeat(" ", gap) + esc
	}

	if m.message != "" {
		return m.styles.Dimmed.Render(m.message)
	}

	items := []string{
		m.styles.HelpKey.Render("↑/k") + " " + m.styles.HelpDesc.Render("up"),
		m.styles.HelpKey.Render("↓/j") + " " + m.styles.HelpDesc.Render("down"),
		m.styles.HelpKey.Render("h/l") + " " + m.styles.HelpDesc.Render("panels"),
	}
	if m.activePanel == 2 {
		items = append(items, m.styles.HelpKey.Render("↵")+" "+m.styles.HelpDesc.Render("logs"))
	} else if m.activePanel == 0 && m.workflowCursor == 0 {
		items = append(items, m.styles.HelpKey.Render("↵")+" "+m.styles.HelpDesc.Render("branch"))
	}
	items = append(items,
		m.styles.HelpKey.Render("r")+" "+m.styles.HelpDesc.Render("rerun"),
		m.styles.HelpKey.Render("c")+" "+m.styles.HelpDesc.Render("cancel"),
	)
	if m.activePanel == 0 && m.workflowCursor > 0 && m.workflowCursor <= len(m.workflows) {
		if wfName := m.workflows[m.workflowCursor-1]; wfName != "all" {
			if _, ok := m.workflowFiles[wfName]; ok {
				items = append(items, m.styles.HelpKey.Render("d")+" "+m.styles.HelpDesc.Render("dispatch"))
			}
		}
	}
	items = append(items,
		m.styles.HelpKey.Render("o")+" "+m.styles.HelpDesc.Render("open"),
		m.styles.HelpKey.Render("/")+" "+m.styles.HelpDesc.Render("search"),
		m.styles.HelpKey.Render("q")+" "+m.styles.HelpDesc.Render("quit"),
	)
	return m.styles.Dimmed.Render(strings.Join(items, "  "))
}

func renderLogs(m Model) string {
	w, h := m.width, m.height
	if w == 0 {
		w = 80
	}
	if h == 0 {
		h = 24
	}

	logLines := strings.Split(m.logs, "\n")
	visibleLines := h - 4

	end := min(m.logOffset+visibleLines, len(logLines))
	scrollInfo := fmt.Sprintf("%d-%d / %d", m.logOffset+1, end, len(logLines))
	header := fmt.Sprintf("Logs: %s", m.logJobName)
	hGap := w - len(header) - len(scrollInfo) - 2
	if hGap < 1 {
		hGap = 1
	}

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPurple).
		Render(header + strings.Repeat(" ", hGap) + scrollInfo))
	sb.WriteString("\n\n")

	maxLineW := w - 8
	if maxLineW < 40 {
		maxLineW = 40
	}
	for i := m.logOffset; i < end; i++ {
		line := logLines[i]
		if len(line) > maxLineW {
			line = line[:maxLineW-3] + "..."
		}
		sb.WriteString(m.styles.LogLineNumber.Render(fmt.Sprintf("%5d ", i+1)))
		sb.WriteString(m.styles.LogLine.Render(line))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	helpItems := []string{
		m.styles.HelpKey.Render("↑/k") + " " + m.styles.HelpDesc.Render("up"),
		m.styles.HelpKey.Render("↓/j") + " " + m.styles.HelpDesc.Render("down"),
		m.styles.HelpKey.Render("g/G") + " " + m.styles.HelpDesc.Render("top/bottom"),
		m.styles.HelpKey.Render("ctrl+u/d") + " " + m.styles.HelpDesc.Render("½ page"),
		m.styles.HelpKey.Render("h/esc/⌫") + " " + m.styles.HelpDesc.Render("back"),
		m.styles.HelpKey.Render("q") + " " + m.styles.HelpDesc.Render("quit"),
	}
	sb.WriteString(m.styles.Dimmed.Render(strings.Join(helpItems, "  ")))

	return sb.String()
}
