package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestConfirmDialog(t *testing.T) {
	t.Run("open and cancel", func(t *testing.T) {
		var cd ConfirmDialog
		assert.False(t, cd.Active())

		cd.Open("owner/repo", 123)
		assert.True(t, cd.Active())

		cd, cmd, msg := cd.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}, nil)
		assert.False(t, cd.Active())
		assert.Nil(t, cmd)
		assert.Empty(t, msg)
	})

	t.Run("confirm with y", func(t *testing.T) {
		var cd ConfirmDialog
		cd.Open("owner/repo", 123)

		cd, cmd, msg := cd.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}, nil)
		assert.False(t, cd.Active())
		assert.NotNil(t, cmd, "expected a rerun command")
		assert.Equal(t, "re-running...", msg)
	})

	t.Run("confirm with d for debug", func(t *testing.T) {
		var cd ConfirmDialog
		cd.Open("owner/repo", 456)

		cd, cmd, msg := cd.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}, nil)
		assert.False(t, cd.Active())
		assert.NotNil(t, cmd, "expected a debug rerun command")
		assert.Equal(t, "re-running with debug...", msg)
	})
}
