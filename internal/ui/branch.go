package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/turkosaurus/gh-ci/internal/gh"
	"github.com/turkosaurus/gh-ci/internal/ui/styles"
)

// BranchPickResult is returned when the picker closes with a selection.
type BranchPickResult struct {
	Chosen string
}

// BranchPicker manages the branch filter popover.
type BranchPicker struct {
	active            bool
	input             textinput.Model
	availableBranches []string
	suggestionCursor  int
}

func NewBranchPicker() BranchPicker {
	bi := textinput.New()
	bi.Placeholder = "filter branches..."
	bi.CharLimit = 100
	return BranchPicker{input: bi}
}

// Open activates the picker, resetting its state.
func (bp *BranchPicker) Open(availableBranches []string) tea.Cmd {
	bp.active = true
	bp.availableBranches = availableBranches
	bp.input.SetValue("")
	bp.input.Focus()
	bp.suggestionCursor = 0
	return textinput.Blink
}

// Active returns whether the picker is currently showing.
func (bp BranchPicker) Active() bool { return bp.active }

// Update handles key events while the picker is active.
// Returns (updated picker, cmd, result).
// result is non-nil when the picker closes with a selection.
func (bp BranchPicker) Update(msg tea.KeyMsg) (BranchPicker, tea.Cmd, *BranchPickResult) {
	switch msg.Type {
	case tea.KeyEscape:
		bp.active = false
		bp.input.Blur()
		return bp, nil, nil

	case tea.KeyEnter:
		suggestions := bp.filteredBranches()
		var chosen string
		if len(suggestions) > 0 {
			idx := bp.suggestionCursor
			if idx >= len(suggestions) {
				idx = len(suggestions) - 1
			}
			chosen = suggestions[idx]
		}
		bp.active = false
		bp.input.Blur()
		if chosen != "" {
			return bp, nil, &BranchPickResult{Chosen: chosen}
		}
		return bp, nil, nil

	case tea.KeyUp:
		if bp.suggestionCursor > 0 {
			bp.suggestionCursor--
		}
		return bp, nil, nil

	case tea.KeyDown:
		suggestions := bp.filteredBranches()
		if bp.suggestionCursor < len(suggestions)-1 {
			bp.suggestionCursor++
		}
		return bp, nil, nil

	default:
		var cmd tea.Cmd
		bp.input, cmd = bp.input.Update(msg)
		bp.suggestionCursor = 0
		return bp, cmd, nil
	}
}

func (bp BranchPicker) filteredBranches() []string {
	q := strings.ToLower(bp.input.Value())
	var out []string
	for _, b := range bp.availableBranches {
		if q == "" || strings.Contains(strings.ToLower(b), q) {
			out = append(out, b)
		}
	}
	return out
}

// View renders the branch input + suggestion list as individual rows.
func (bp BranchPicker) View(s styles.Styles, width int) []string {
	const maxSugg = 4
	selectedStyle := lipgloss.NewStyle().Bold(true).
		Background(styles.ColorBgLight).Foreground(styles.ColorWhite)

	var rows []string
	rows = append(rows, bp.input.View())
	suggestions := bp.filteredBranches()
	limit := maxSugg
	if limit > len(suggestions) {
		limit = len(suggestions)
	}
	for i, b := range suggestions[:limit] {
		if i == bp.suggestionCursor {
			rows = append(rows, selectedStyle.Render("> "+gh.TruncateString(b, width-4)))
		} else {
			rows = append(rows, s.Dimmed.Render("  "+gh.TruncateString(b, width-4)))
		}
	}
	return rows
}

// HelpView returns the help bar text when the branch picker is active.
func (bp BranchPicker) HelpView(s styles.Styles) string {
	return s.Dimmed.Render("↑/↓ navigate  ↵ confirm  esc cancel")
}
