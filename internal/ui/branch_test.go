package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBranchPicker(t *testing.T) {
	branches := []string{"feature/auth", "main", "staging"}

	t.Run("open and close with esc", func(t *testing.T) {
		bp := NewBranchPicker()
		assert.False(t, bp.Active())

		bp.Open(branches)
		assert.True(t, bp.Active())

		bp, _, result := bp.Update(tea.KeyMsg{Type: tea.KeyEscape})
		assert.False(t, bp.Active())
		assert.Nil(t, result)
	})

	t.Run("select with enter", func(t *testing.T) {
		bp := NewBranchPicker()
		bp.Open(branches)

		// press enter to select first suggestion (feature/auth)
		bp, _, result := bp.Update(tea.KeyMsg{Type: tea.KeyEnter})
		assert.False(t, bp.Active())
		require.NotNil(t, result)
		assert.Equal(t, "feature/auth", result.Chosen)
	})

	t.Run("navigate down and select", func(t *testing.T) {
		bp := NewBranchPicker()
		bp.Open(branches)

		// move down once then select
		bp, _, _ = bp.Update(tea.KeyMsg{Type: tea.KeyDown})
		bp, _, result := bp.Update(tea.KeyMsg{Type: tea.KeyEnter})
		require.NotNil(t, result)
		assert.Equal(t, "main", result.Chosen)
	})

	t.Run("up does not go below 0", func(t *testing.T) {
		bp := NewBranchPicker()
		bp.Open(branches)

		bp, _, _ = bp.Update(tea.KeyMsg{Type: tea.KeyUp})
		bp, _, result := bp.Update(tea.KeyMsg{Type: tea.KeyEnter})
		require.NotNil(t, result)
		assert.Equal(t, "feature/auth", result.Chosen)
	})
}
