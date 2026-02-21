# internal/ui

The UI is built with [Bubble Tea](https://github.com/charmbracelet/bubbletea), a TUI framework based on The Elm Architecture. If you haven't used it before, the short version: all state lives in `Model`, `Update()` handles incoming messages (key presses, data loads, timers), and `View()` renders to a string.

## Screens

There are two screens, toggled via the `screen` field on the model:

- **ScreenMain** — the three-panel workflow list (workflows → runs → jobs)
- **ScreenLogs** — the log viewer for a single job

## Key handling

`Update()` dispatches key messages down a chain of handlers based on current state:

```
handleBranchSelect  (branch picker input)
handleLogSearch     (log search input)
handleConfirm       (re-run confirmation)
handleDispatchConfirm
handleLogsKeys      (ScreenLogs navigation)
handleMainKeys      (ScreenMain navigation)
```

## Adding a keybinding

1. Add a `key.Binding` to the `KeyMap` struct in `keys/keys.go`
2. Add an entry in `DefaultKeyMap()` with `key.WithKeys(...)` and `key.WithHelp(...)`
3. Handle it in the appropriate function in `model.go` using `key.Matches(msg, m.keys.YourKey)`
4. Add it to `renderHelpBar()` or the log help bar in `render.go` if it should appear in the footer

## Subpackages

- **`keys/`** — all key binding definitions
- **`styles/`** — lipgloss color and style definitions
