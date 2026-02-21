package views

import (
	"fmt"
	"strings"

	"github.com/jay-418/gh-ci/internal/types"
	"github.com/jay-418/gh-ci/internal/ui/components"
	"github.com/jay-418/gh-ci/internal/ui/styles"
)

// MainView renders the main dashboard view
type MainView struct {
	RunList      components.RunList
	Help         components.Help
	Styles       styles.Styles
	Width        int
	Height       int
	StatusFilter types.StatusFilter
	SearchQuery  string
	ShowFilter   bool
	Message      string
}

// NewMainView creates a new main view
func NewMainView(s styles.Styles) MainView {
	return MainView{
		RunList:      components.NewRunList(s),
		Help:         components.NewHelp(s),
		Styles:       s,
		StatusFilter: types.StatusAll,
		SearchQuery:  "",
		ShowFilter:   false,
	}
}

// SetSize sets the dimensions
func (v *MainView) SetSize(width, height int) {
	v.Width = width
	v.Height = height
	v.RunList.SetSize(width, height-6) // Leave room for header, filter, and help
}

// SetMessage sets a status message
func (v *MainView) SetMessage(msg string) {
	v.Message = msg
}

// ClearMessage clears the status message
func (v *MainView) ClearMessage() {
	v.Message = ""
}

// View renders the main view
func (v *MainView) View() string {
	var sb strings.Builder

	// Title
	title := v.Styles.Title.Render("GitHub Actions Dashboard")
	sb.WriteString(title)
	sb.WriteString("\n")

	// Filter bar
	if v.ShowFilter || v.StatusFilter != types.StatusAll || v.SearchQuery != "" {
		sb.WriteString(v.renderFilterBar())
		sb.WriteString("\n")
	}

	// Status message
	if v.Message != "" {
		sb.WriteString(v.Styles.Dimmed.Render(v.Message))
		sb.WriteString("\n")
	}

	// Run list
	sb.WriteString(v.RunList.View())
	sb.WriteString("\n")

	// Help bar
	v.Help.SetView("list")
	sb.WriteString(v.Help.Render())

	return v.Styles.App.Render(sb.String())
}

// renderFilterBar renders the filter bar
func (v *MainView) renderFilterBar() string {
	var parts []string

	// Status filters
	filters := []struct {
		filter types.StatusFilter
		label  string
	}{
		{types.StatusAll, "All"},
		{types.StatusFailed, "Failed"},
		{types.StatusInProgress, "Running"},
		{types.StatusSuccess, "Success"},
	}

	for _, f := range filters {
		label := f.label
		if f.filter == v.StatusFilter {
			label = v.Styles.FilterActive.Render(label)
		} else {
			label = v.Styles.Dimmed.Render(label)
		}
		parts = append(parts, label)
	}

	filterBar := strings.Join(parts, " | ")

	// Search query
	if v.SearchQuery != "" {
		search := fmt.Sprintf(" Search: %s", v.SearchQuery)
		filterBar += v.Styles.Branch.Render(search)
	}

	return filterBar
}

// CycleFilter cycles through status filters
func (v *MainView) CycleFilter() {
	switch v.StatusFilter {
	case types.StatusAll:
		v.StatusFilter = types.StatusFailed
	case types.StatusFailed:
		v.StatusFilter = types.StatusInProgress
	case types.StatusInProgress:
		v.StatusFilter = types.StatusSuccess
	case types.StatusSuccess:
		v.StatusFilter = types.StatusAll
	}
}

// ClearFilter clears the filter
func (v *MainView) ClearFilter() {
	v.StatusFilter = types.StatusAll
	v.SearchQuery = ""
	v.ShowFilter = false
}
