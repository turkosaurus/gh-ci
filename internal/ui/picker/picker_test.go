package picker

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPicker(t *testing.T) {
	items := []string{"feature/auth", "main", "staging"}

	t.Run("open and close with esc", func(t *testing.T) {
		p := NewPicker("filter...")
		assert.False(t, p.Active())

		p.Open(items)
		assert.True(t, p.Active())

		p, _, result := p.Update(tea.KeyMsg{Type: tea.KeyEscape})
		assert.False(t, p.Active())
		assert.Nil(t, result)
	})

	t.Run("select with enter", func(t *testing.T) {
		p := NewPicker("filter...")
		p.Open(items)

		p, _, result := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
		assert.False(t, p.Active())
		require.NotNil(t, result)
		assert.Equal(t, "feature/auth", result.Chosen)
	})

	t.Run("navigate down and select", func(t *testing.T) {
		p := NewPicker("filter...")
		p.Open(items)

		p, _, _ = p.Update(tea.KeyMsg{Type: tea.KeyDown})
		p, _, result := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
		require.NotNil(t, result)
		assert.Equal(t, "main", result.Chosen)
	})

	t.Run("up does not go below 0", func(t *testing.T) {
		p := NewPicker("filter...")
		p.Open(items)

		p, _, _ = p.Update(tea.KeyMsg{Type: tea.KeyUp})
		p, _, result := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
		require.NotNil(t, result)
		assert.Equal(t, "feature/auth", result.Chosen)
	})
}
