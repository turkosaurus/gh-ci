package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/turkosaurus/gh-ci/internal/ui/styles"
)

func TestDispatchDialog(t *testing.T) {
	t.Run("open and cancel with esc", func(t *testing.T) {
		var dd DispatchDialog
		assert.False(t, dd.Active())

		dd.Open("owner/repo", "ci.yml", "main")
		assert.True(t, dd.Active())

		dd, cmd, msg := dd.Update(tea.KeyMsg{Type: tea.KeyEscape}, nil)
		assert.False(t, dd.Active())
		assert.Nil(t, cmd)
		assert.Empty(t, msg)
	})

	t.Run("confirm dispatch with y", func(t *testing.T) {
		var dd DispatchDialog
		dd.Open("owner/repo", "ci.yml", "main")

		dd, cmd, msg := dd.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}, nil)
		assert.False(t, dd.Active())
		assert.NotNil(t, cmd, "expected a dispatch command")
		assert.Equal(t, "dispatching...", msg)
	})

	t.Run("help view includes file and ref", func(t *testing.T) {
		var dd DispatchDialog
		dd.Open("owner/repo", "deploy.yml", "staging")

		s := styles.DefaultStyles()
		help := dd.HelpView(s)
		assert.Contains(t, help, "deploy.yml")
		assert.Contains(t, help, "staging")
	})
}
