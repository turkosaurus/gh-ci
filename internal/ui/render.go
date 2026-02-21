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

	// 1 title line + 1 help line
	bodyH := h - 2
	leftW := w * 55 / 100
	rightW := w - leftW - 1 // 1 for separator

	sep := lipgloss.NewStyle().
		Foreground(styles.ColorSubtle).
		Render(strings.Repeat("│\n", bodyH-1) + "│")

	body := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(leftW).Height(bodyH).Render(renderList(m, leftW, bodyH)),
		sep,
		lipgloss.NewStyle().Width(rightW).Height(bodyH).Render(renderDetail(m, rightW)),
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		renderTitle(m, w),
		body,
		renderHelpBar(m, w),
	)
}

func renderTitle(m Model, width int) string {
	title := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPurple).
		Render("GitHub Actions Dashboard")

	tabs := []struct {
		f     types.StatusFilter
		label string
	}{
		{types.StatusAll, "All"},
		{types.StatusFailed, "Failed"},
		{types.StatusInProgress, "Running"},
		{types.StatusSuccess, "Success"},
	}

	var parts []string
	for _, t := range tabs {
		if t.f == m.filter {
			parts = append(parts, m.styles.FilterActive.Render(t.label))
		} else {
			parts = append(parts, m.styles.Dimmed.Render(t.label))
		}
	}
	filterBar := strings.Join(parts, " ")

	gap := width - lipgloss.Width(title) - lipgloss.Width(filterBar) - 1
	if gap < 1 {
		gap = 1
	}
	return title + strings.Repeat(" ", gap) + filterBar
}

func renderList(m Model, width, height int) string {
	if len(m.filteredRuns) == 0 {
		if m.searchQuery != "" || m.filter != types.StatusAll {
			return m.styles.Dimmed.Render("no runs match filter")
		}
		return m.styles.Dimmed.Render("no workflow runs")
	}

	// Widths: status(2) + spaces(5) + num(6) + dur(9) + branch(14) + workflow(18) + repo(rest)
	const colSt, colNum, colDur, colBranch, colWorkflow = 2, 6, 9, 14, 18
	colRepo := width - colSt - colNum - colDur - colBranch - colWorkflow - 5
	if colRepo < 10 {
		colRepo = 10
	}

	header := fmt.Sprintf("  %-*s  %-*s  %-*s  %*s  %-*s",
		colRepo, "REPO",
		colWorkflow, "WORKFLOW",
		colBranch, "BRANCH",
		colNum, "RUN",
		colDur, "DURATION",
	)
	rows := []string{
		lipgloss.NewStyle().Bold(true).Foreground(styles.ColorGray).Render(header),
	}

	// Scroll viewport: keep cursor visible
	listH := height - 2 // header row + scroll indicator
	startIdx := 0
	if m.cursor >= listH {
		startIdx = m.cursor - listH + 1
	}
	endIdx := min(startIdx+listH, len(m.filteredRuns))

	for i := startIdx; i < endIdx; i++ {
		rows = append(rows, renderRunRow(m, m.filteredRuns[i], i == m.cursor,
			colRepo, colWorkflow, colBranch, colNum, colDur))
	}

	if len(m.filteredRuns) > listH {
		rows = append(rows, m.styles.Dimmed.Render(
			fmt.Sprintf(" %d/%d", m.cursor+1, len(m.filteredRuns))))
	}

	return strings.Join(rows, "\n")
}

func renderRunRow(m Model, run types.WorkflowRun, selected bool, colRepo, colWorkflow, colBranch, colNum, colDur int) string {
	icon := styles.StatusIcon(run.Status, run.Conclusion)
	st := m.styles.StatusStyle(run.Status, run.Conclusion).Render(icon)
	repo := m.styles.Repo.Render(fmt.Sprintf("%-*s", colRepo, gh.TruncateString(run.Repository.FullName, colRepo)))
	workflow := fmt.Sprintf("%-*s", colWorkflow, gh.TruncateString(run.Name, colWorkflow))
	branch := m.styles.Branch.Render(fmt.Sprintf("%-*s", colBranch, gh.TruncateString(run.HeadBranch, colBranch)))
	num := fmt.Sprintf("%*s", colNum, fmt.Sprintf("#%d", run.RunNumber))
	dur := m.styles.Duration.Render(fmt.Sprintf("%-*s", colDur, gh.FormatDuration(int64(run.Duration().Seconds()))))

	row := st + " " + repo + "  " + workflow + "  " + branch + "  " + num + "  " + dur

	if selected {
		row = lipgloss.NewStyle().
			Bold(true).
			Background(styles.ColorBgLight).
			Foreground(styles.ColorWhite).
			Render(row)
	}
	return row
}

func renderDetail(m Model, width int) string {
	run := m.selectedRun()
	if run == nil {
		return m.styles.Dimmed.Render("no run selected")
	}

	var sb strings.Builder

	title := fmt.Sprintf("Run #%d: %s", run.RunNumber, gh.TruncateString(run.Name, width-10))
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPurple).Render(title))
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
	sb.WriteString(m.styles.Dimmed.Render("jobs") + "\n")

	if len(m.jobs) == 0 {
		sb.WriteString("  " + m.styles.Dimmed.Render("loading..."))
	} else {
		for _, job := range m.jobs {
			jIcon := styles.StatusIcon(job.Status, job.Conclusion)
			jStyle := m.styles.StatusStyle(job.Status, job.Conclusion)
			sb.WriteString("  " + jStyle.Render(jIcon) + " " + gh.TruncateString(job.Name, width-5) + "\n")
		}
	}

	return sb.String()
}

func renderHelpBar(m Model, width int) string {
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
		m.styles.HelpKey.Render("l") + " " + m.styles.HelpDesc.Render("logs"),
		m.styles.HelpKey.Render("r") + " " + m.styles.HelpDesc.Render("rerun"),
		m.styles.HelpKey.Render("c") + " " + m.styles.HelpDesc.Render("cancel"),
		m.styles.HelpKey.Render("o") + " " + m.styles.HelpDesc.Render("open"),
		m.styles.HelpKey.Render("/") + " " + m.styles.HelpDesc.Render("search"),
		m.styles.HelpKey.Render("q") + " " + m.styles.HelpDesc.Render("quit"),
	}
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
		m.styles.HelpKey.Render("esc") + " " + m.styles.HelpDesc.Render("back"),
		m.styles.HelpKey.Render("q") + " " + m.styles.HelpDesc.Render("quit"),
	}
	sb.WriteString(m.styles.Dimmed.Render(strings.Join(helpItems, "  ")))

	return sb.String()
}
