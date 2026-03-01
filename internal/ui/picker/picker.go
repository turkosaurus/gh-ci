package picker

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/turkosaurus/gh-ci/internal/gh"
	"github.com/turkosaurus/gh-ci/internal/ui/styles"
)

type PickResult struct {
	Chosen string
}

type Picker struct {
	active           bool
	input            textinput.Model
	items            []string
	suggestionCursor int
}

func NewPicker(placeholder string) Picker {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = 100
	return Picker{input: ti}
}

func (p *Picker) Open(items []string) tea.Cmd {
	p.active = true
	p.items = items
	p.input.SetValue("")
	p.input.Focus()
	p.suggestionCursor = 0
	return textinput.Blink
}

func (p Picker) Active() bool { return p.active }

func (p Picker) Update(msg tea.KeyMsg) (Picker, tea.Cmd, *PickResult) {
	switch msg.Type {
	case tea.KeyEscape:
		p.active = false
		p.input.Blur()
		return p, nil, nil

	case tea.KeyEnter:
		suggestions := p.filtered()
		var chosen string
		if len(suggestions) > 0 {
			idx := p.suggestionCursor
			if idx >= len(suggestions) {
				idx = len(suggestions) - 1
			}
			chosen = suggestions[idx]
		}
		p.active = false
		p.input.Blur()
		if chosen != "" {
			return p, nil, &PickResult{Chosen: chosen}
		}
		return p, nil, nil

	case tea.KeyUp:
		if p.suggestionCursor > 0 {
			p.suggestionCursor--
		}
		return p, nil, nil

	case tea.KeyDown:
		suggestions := p.filtered()
		if p.suggestionCursor < len(suggestions)-1 {
			p.suggestionCursor++
		}
		return p, nil, nil

	default:
		var cmd tea.Cmd
		p.input, cmd = p.input.Update(msg)
		p.suggestionCursor = 0
		return p, cmd, nil
	}
}

func (p Picker) filtered() []string {
	q := strings.ToLower(p.input.Value())
	var out []string
	for _, item := range p.items {
		if q == "" || strings.Contains(strings.ToLower(item), q) {
			out = append(out, item)
		}
	}
	return out
}

func (p Picker) View(s styles.Styles, width int) []string {
	const maxSugg = 4
	selectedStyle := lipgloss.NewStyle().Bold(true).
		Background(s.P.BgLight).Foreground(s.P.Fg)

	var rows []string
	rows = append(rows, p.input.View())
	suggestions := p.filtered()
	limit := maxSugg
	if limit > len(suggestions) {
		limit = len(suggestions)
	}
	for i, item := range suggestions[:limit] {
		if i == p.suggestionCursor {
			rows = append(rows, selectedStyle.Render("> "+gh.TruncateString(item, width-4)))
		} else {
			rows = append(rows, s.Dimmed.Render("  "+gh.TruncateString(item, width-4)))
		}
	}
	return rows
}

func (p Picker) HelpView(s styles.Styles) string {
	return s.Dimmed.Render("↑/↓ navigate  ↵ confirm  esc cancel")
}

// Query returns the current search input value.
func (p Picker) Query() string {
	return p.input.Value()
}
