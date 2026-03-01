package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/turkosaurus/gh-ci/internal/gh"
	"github.com/turkosaurus/gh-ci/internal/types"
	"github.com/turkosaurus/gh-ci/internal/ui/styles"
)

// View renders the complete dashboard (title + panels + help bar).
func (d Dashboard) View(width, height int, message Message, loading bool) string {
	w, h := width, height
	if w == 0 {
		w = 80
	}
	if h == 0 {
		h = 24
	}

	bodyH := h - 5 // 3-line title + panel-headers + help
	// set minimum to avoid rendering issues when tiny
	if bodyH < 5 {
		bodyH = 5
	}

	lay := computeDashLayout(w)
	workflowW, runsW, detailW := lay.workflowW, lay.runsW, lay.detailW

	sep := lipgloss.NewStyle().
		Foreground(d.styles.P.Subtle).
		Render(strings.Repeat("│\n", bodyH-1) + "│")

	body := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(workflowW).Height(bodyH).Render(d.renderWorkflows(workflowW, bodyH)),
		sep,
		lipgloss.NewStyle().Width(runsW).Height(bodyH).Render(d.renderList(runsW, bodyH, loading)),
		sep,
		lipgloss.NewStyle().Width(detailW).Height(bodyH).Render(d.renderDetail(detailW)),
	)

	base := lipgloss.JoinVertical(lipgloss.Left,
		renderTitle(w, d.styles.P),
		d.renderPanelHeaders(workflowW, runsW, detailW),
		body,
		d.renderHelpBar(w, message),
	)

	if d.helpModal.Active() {
		return d.helpModal.View(d.keys, d.styles, w, h, true)
	}
	return base
}

func (d Dashboard) renderPanelHeaders(workflowW, runsW, detailW int) string {
	sep := lipgloss.NewStyle().Background(d.styles.P.BgLight).Foreground(d.styles.P.Subtle).Render("│")
	label := func(panel int, text string, w int) string {
		style := lipgloss.NewStyle().
			Width(w).
			Align(lipgloss.Center) // Center the label text
		if d.activePanel == panel {
			style = style.Bold(true).
				Background(d.styles.P.Accent).Foreground(d.styles.P.Bg)
		} else {
			style = style.Background(d.styles.P.BgLight).Foreground(d.styles.P.Fg)
		}
		return style.Render(text)
	}
	return lipgloss.NewStyle().
		Width(workflowW + runsW + detailW + 2). // 2 for separators
		Align(lipgloss.Center).
		Render(
			lipgloss.JoinHorizontal(lipgloss.Top,
				label(panelWorkflows, "WORKFLOWS", workflowW),
				sep,
				label(panelRuns, "RUNS", runsW),
				sep,
				label(panelDetail, "DETAIL", detailW),
			),
		)
}

func (d Dashboard) renderWorkflows(width, height int) string {
	active := d.activePanel == panelWorkflows

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(d.styles.P.FgDim)
	if active {
		headerStyle = headerStyle.Foreground(d.styles.P.Accent)
	}
	selectedStyle := lipgloss.NewStyle().Bold(true).Background(d.styles.P.BgLight).Foreground(d.styles.P.Fg)

	var rows []string

	// ── REPO section ────────────────────────────────────────────────────────
	rows = append(rows, headerStyle.Render("REPO"))
	if d.repoPicker.Active() {
		for _, row := range d.repoPicker.View(d.styles, width) {
			rows = append(rows, row)
		}
	} else {
		repoDisplay := strings.Join(d.config.Repos, ", ")
		text := fmt.Sprintf("%-*s", width-2, gh.TruncateString(repoDisplay, width-2))
		if d.workflowCursor == -1 && active {
			rows = append(rows, selectedStyle.Render(text))
		} else {
			rows = append(rows, d.styles.Repo.Render(text))
		}
	}

	// Separator
	rows = append(rows, d.styles.Dimmed.Render(strings.Repeat("─", width-1)))

	// ── BRANCH section ──────────────────────────────────────────────────────
	rows = append(rows, headerStyle.Render("BRANCH"))

	branchDisplay := d.defaultBranch
	if d.branchIdx < len(d.availableBranches) {
		branchDisplay = d.availableBranches[d.branchIdx]
	}

	if d.branchPicker.Active() {
		for _, row := range d.branchPicker.View(d.styles, width) {
			rows = append(rows, row)
		}
	} else {
		text := fmt.Sprintf("%-*s", width-2, gh.TruncateString(branchDisplay, width-2))
		if d.workflowCursor == 0 && active {
			rows = append(rows, selectedStyle.Render(text))
		} else {
			rows = append(rows, d.styles.Branch.Render(text))
		}
	}

	// Separator
	rows = append(rows, d.styles.Dimmed.Render(strings.Repeat("─", width-1)))

	// ── NAME section ─────────────────────────────────────────────────────────
	rows = append(rows, headerStyle.Render("NAME"))

	if len(d.workflows) == 0 {
		rows = append(rows, d.styles.Dimmed.Render("loading..."))
		return strings.Join(rows, "\n")
	}

	// Check if we have a filename to pin at the bottom
	var filenameStr string
	if wfName := d.selectedWorkflow(); wfName != "" && wfName != workflowAll {
		filenameStr = d.workflowFiles[wfName]
	}

	branchSectionH := len(rows)
	workflowListH := height - branchSectionH
	if filenameStr != "" {
		workflowListH--
	}
	if workflowListH < 1 {
		workflowListH = 1
	}

	// wfCursor: index within d.workflows for scroll calculation
	// cursor scheme: 0=branch, 1..N=workflows[0..N-1]
	wfCursor := 0
	if d.workflowCursor > 0 {
		wfCursor = d.workflowCursor - 1
	}
	startIdx := 0
	if wfCursor >= workflowListH {
		startIdx = wfCursor - workflowListH + 1
	}
	endIdx := min(startIdx+workflowListH, len(d.workflows))

	for i := startIdx; i < endIdx; i++ {
		wfName := d.workflows[i]
		selected := (i + 1) == d.workflowCursor
		text := fmt.Sprintf("%-*s", width-2, gh.TruncateString(wfName, width-2))
		var row string
		switch {
		case selected && active:
			row = selectedStyle.Render(text)
		case selected:
			row = lipgloss.NewStyle().Bold(true).Foreground(d.styles.P.Accent).Render(text)
		default:
			row = d.styles.Normal.Render(text)
		}
		rows = append(rows, row)
	}

	if filenameStr != "" {
		for len(rows) < height-1 {
			rows = append(rows, "")
		}
		rows = append(rows, lipgloss.NewStyle().Foreground(d.styles.P.Branch).Render(gh.TruncateString(filenameStr, width-2)))
	}

	return strings.Join(rows, "\n")
}

func (d Dashboard) renderList(width, height int, loading bool) string {
	active := d.activePanel == panelRuns

	if len(d.filteredRuns) == 0 {
		if loading && time.Since(d.born) > 1500*time.Millisecond {
			return d.styles.Dimmed.Render("🔶 workflow runs loading")
		}
		return d.styles.Dimmed.Render("🟧 workflow runs empty")
	}

	const colOk, colNum, colDur = 2, 6, 7
	colFile, colDispatched := 14, 16

	// Number of separators depends on visible columns (always 3 for ok/num/dur/workflow, +1 each for file/dispatched)
	numSeps := 3
	usedFixed := colOk + colNum + colDur
	if colDispatched > 0 {
		usedFixed += colDispatched
		numSeps++
	}
	if colFile > 0 {
		usedFixed += colFile
		numSeps++
	}
	colWorkflow := width - usedFixed - numSeps*colSep

	// Collapse FILE column first if too narrow
	if colWorkflow < 10 {
		colFile = 0
		numSeps = 3
		usedFixed = colOk + colNum + colDur + colDispatched
		numSeps++
		colWorkflow = width - usedFixed - numSeps*colSep
	}
	// Collapse DISPATCHED column if still too narrow
	if colWorkflow < 10 {
		colDispatched = 0
		numSeps = 3
		usedFixed = colOk + colNum + colDur
		colWorkflow = width - usedFixed - numSeps*colSep
	}
	if colWorkflow < 4 {
		colWorkflow = 4
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(d.styles.P.FgDim)
	if active {
		headerStyle = headerStyle.Foreground(d.styles.P.Accent)
	}

	var headerParts []string
	if colDispatched > 0 {
		headerParts = append(headerParts, fmt.Sprintf("%-*s", colDispatched, "DISPATCHED (UTC)"))
	}
	if colFile > 0 {
		headerParts = append(headerParts, fmt.Sprintf("%-*s", colFile, "FILE"))
	}
	headerParts = append(headerParts,
		fmt.Sprintf("%-*s", colWorkflow, "NAME"),
		fmt.Sprintf("%*s", colNum, "RUN"),
		fmt.Sprintf("%-*s", colDur, "TIME"),
		fmt.Sprintf("%-*s", colOk, "OK"),
	)
	header := strings.Join(headerParts, "  ")
	rows := []string{headerStyle.Render(header)}

	listH := height - 2
	startIdx := 0
	if d.cursor >= listH {
		startIdx = d.cursor - listH + 1
	}
	endIdx := min(startIdx+listH, len(d.filteredRuns))

	for i := startIdx; i < endIdx; i++ {
		rows = append(rows, d.renderRunRow(d.filteredRuns[i], i == d.cursor, active,
			width, colWorkflow, colFile, colNum, colDur, colDispatched, colOk))
	}

	if len(d.filteredRuns) > listH {
		rows = append(rows, d.styles.Dimmed.Render(
			fmt.Sprintf(" %d/%d", d.cursor+1, len(d.filteredRuns))))
	}

	return strings.Join(rows, "\n")
}

func (d Dashboard) renderRunRow(run types.WorkflowRun, selected, active bool, width, colWorkflow, colFile, colNum, colDur, colDispatched, colOk int) string {
	icon := styles.StatusIcon(run.Status, run.Conclusion)
	iconS := fmt.Sprintf("%-*s", colOk, icon)
	wfS := fmt.Sprintf("%-*s", colWorkflow, gh.TruncateString(run.Name, colWorkflow))
	numS := fmt.Sprintf("%*s", colNum, fmt.Sprintf("#%d", run.RunNumber))
	durS := fmt.Sprintf("%-*s", colDur, gh.FormatDuration(int64(run.Duration().Seconds())))

	// Build column list based on which columns are visible
	type col struct {
		plain  string
		styled func(lipgloss.Style) string // for selected+active row
		dim    bool                        // use dimmed style in normal mode
	}
	var cols []col

	if colDispatched > 0 {
		dispS := fmt.Sprintf("%-*s", colDispatched, run.CreatedAt.Format(timestampFormat))
		cols = append(cols, col{plain: dispS, styled: func(bg lipgloss.Style) string {
			return d.styles.Dimmed.Background(bg.GetBackground()).Render(dispS)
		}, dim: true})
	}
	if colFile > 0 {
		fileS := fmt.Sprintf("%-*s", colFile, gh.TruncateString(d.workflowFiles[run.Name], colFile))
		cols = append(cols, col{plain: fileS, styled: func(bg lipgloss.Style) string {
			return d.styles.Dimmed.Background(bg.GetBackground()).Render(fileS)
		}, dim: true})
	}
	cols = append(cols,
		col{plain: wfS, styled: func(bg lipgloss.Style) string {
			return lipgloss.NewStyle().Bold(true).Foreground(d.styles.P.Fg).Background(bg.GetBackground()).Render(wfS)
		}},
		col{plain: numS, styled: func(bg lipgloss.Style) string {
			return lipgloss.NewStyle().Foreground(d.styles.P.Fg).Background(bg.GetBackground()).Render(numS)
		}},
		col{plain: durS, styled: func(bg lipgloss.Style) string {
			return d.styles.Duration.Background(bg.GetBackground()).Render(durS)
		}},
		col{plain: iconS, styled: func(bg lipgloss.Style) string {
			return d.styles.StatusStyle(run.Status, run.Conclusion).Background(bg.GetBackground()).Render(iconS)
		}},
	)

	if selected && active {
		bg := lipgloss.NewStyle().Background(d.styles.P.BgLight)
		sep := bg.Render("  ")
		var parts []string
		for _, c := range cols {
			parts = append(parts, c.styled(bg))
		}
		row := strings.Join(parts, sep)
		// pad remaining width with background
		used := colWorkflow + colNum + colDur + colOk
		numSeps := len(cols) - 1
		if colDispatched > 0 {
			used += colDispatched
		}
		if colFile > 0 {
			used += colFile
		}
		used += numSeps * colSep
		if pad := width - used; pad > 0 {
			row += bg.Render(strings.Repeat(" ", pad))
		}
		return row
	}

	if selected {
		var parts []string
		for _, c := range cols {
			parts = append(parts, c.plain)
		}
		plainRow := strings.Join(parts, "  ")
		return lipgloss.NewStyle().Bold(true).Foreground(d.styles.P.Accent).Render(plainRow)
	}

	// normal: per-element styles
	var parts []string
	for _, c := range cols {
		if c.dim {
			parts = append(parts, d.styles.Dimmed.Render(c.plain))
		} else if c.plain == iconS {
			parts = append(parts, d.styles.StatusStyle(run.Status, run.Conclusion).Render(c.plain))
		} else if c.plain == durS {
			parts = append(parts, d.styles.Duration.Render(c.plain))
		} else {
			parts = append(parts, c.plain)
		}
	}
	return strings.Join(parts, "  ")
}

func (d Dashboard) renderDetail(width int) string {
	active := d.activePanel == panelDetail

	run := d.selectedRun()
	if run == nil {
		return d.styles.Dimmed.Render("no run selected")
	}

	var sb strings.Builder

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(d.styles.P.FgDim)
	if active {
		headerStyle = headerStyle.Foreground(d.styles.P.Accent)
	}
	sb.WriteString(headerStyle.Render(fmt.Sprintf("[#%d] %s", run.RunNumber, gh.TruncateString(run.Name, width-10))))
	sb.WriteString("\n\n")

	field := func(label, value string) {
		sb.WriteString(d.styles.Dimmed.Render(fmt.Sprintf("%-8s", label)))
		sb.WriteString(value + "\n")
	}

	sha := run.HeadSHA
	if len(sha) > 8 {
		sha = sha[:8]
	}
	statusStyle := d.styles.StatusStyle(run.Status, run.Conclusion)
	icon := styles.StatusIcon(run.Status, run.Conclusion)
	dur := gh.FormatDuration(int64(run.Duration().Seconds()))

	field("repo", d.styles.Repo.Render(gh.TruncateString(run.Repository.FullName, width-10)))
	field("branch", d.styles.Branch.Render(run.HeadBranch))
	field("commit", d.styles.Normal.Render(sha))
	field("status", statusStyle.Render(icon+" "+run.GetStatus())+"  "+d.styles.Duration.Render(dur))

	sb.WriteString("\n")

	jobsHeaderStyle := d.styles.Dimmed
	if active {
		jobsHeaderStyle = lipgloss.NewStyle().Foreground(d.styles.P.Accent)
	}
	sb.WriteString(jobsHeaderStyle.Render("jobs") + "\n")

	if len(d.jobs) == 0 {
		sb.WriteString("  " + d.styles.Dimmed.Render("loading..."))
	} else {
		for i, job := range d.jobs {
			jIcon := styles.StatusIcon(job.Status, job.Conclusion)
			name := gh.TruncateString(job.Name, width-5)
			var line string
			switch {
			case i == d.jobCursor && active:
				line = lipgloss.NewStyle().Bold(true).Background(d.styles.P.BgLight).Foreground(d.styles.P.Fg).
					Render(fmt.Sprintf("  %s %s", jIcon, name))
			case i == d.jobCursor:
				line = lipgloss.NewStyle().Bold(true).Foreground(d.styles.P.Accent).
					Render(fmt.Sprintf("  %s %s", jIcon, name))
			default:
				jStyle := d.styles.StatusStyle(job.Status, job.Conclusion)
				line = "  " + jStyle.Render(jIcon) + " " + name
			}
			sb.WriteString(line + "\n")
		}
	}

	return sb.String()
}

func (d Dashboard) renderHelpBar(width int, message Message) string {
	if d.confirmDialog.Active() {
		return d.confirmDialog.HelpView(d.styles)
	}

	if d.dispatchDialog.Active() {
		return d.dispatchDialog.HelpView(d.styles)
	}

	if d.repoPicker.Active() {
		return d.repoPicker.HelpView(d.styles)
	}

	if d.branchPicker.Active() {
		return d.branchPicker.HelpView(d.styles)
	}

	if message.text != "" {
		return d.styles.MessageStyle(message.msgType).Render(message.text)
	}

	var items []string
	if run := d.selectedRun(); run != nil && d.activePanel != panelWorkflows {
		items = append(items, bindingHelp(d.styles, d.keys.Rerun))
		if run.Status == types.RunStatusInProgress {
			items = append(items, bindingHelp(d.styles, d.keys.Cancel))
		}
	}
	if d.activePanel == panelWorkflows {
		if wfName := d.selectedWorkflow(); wfName != "" && wfName != workflowAll {
			if _, ok := d.workflowFiles[wfName]; ok {
				items = append(items, bindingHelp(d.styles, d.keys.Edit))
				items = append(items, bindingHelp(d.styles, d.keys.Dispatch))
			}
		}
	}
	items = append(items, bindingHelp(d.styles, d.keys.Open))
	if d.activePanel == panelDetail && d.jobCursor < len(d.jobs) {
		items = append(items, d.styles.HelpKey.Render("↵")+" "+d.styles.HelpDesc.Render("view logs"))
	}

	left := strings.Join(items, "  ")
	right := bindingHelp(d.styles, d.keys.Quit) + "  " + bindingHelp(d.styles, d.keys.Help)
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 2 {
		gap = 2
	}
	return left + strings.Repeat(" ", gap) + right
}
