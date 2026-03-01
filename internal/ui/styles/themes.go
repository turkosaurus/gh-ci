package styles

import "github.com/charmbracelet/lipgloss"

// Palette holds all semantic color roles for the UI theme
type Palette struct {
	Accent   lipgloss.Color // primary highlight (purple in Dracula)
	Bg       lipgloss.Color // terminal background
	BgLight  lipgloss.Color // selected-item background
	Subtle   lipgloss.Color // separators, borders
	Fg       lipgloss.Color // primary foreground
	FgDim    lipgloss.Color // secondary/dimmed text
	Success  lipgloss.Color // green
	Failure  lipgloss.Color // red
	Running  lipgloss.Color // yellow
	Branch   lipgloss.Color // pink / branch label color
	Repo     lipgloss.Color // blue / repo label color
	Duration lipgloss.Color // orange
	Key      lipgloss.Color // cyan / help key color
}

// Theme bundles a name and palette
type Theme struct {
	Name    string
	Palette Palette
}

// Themes is a map of available themes by name
var Themes = map[string]Theme{
	"dracula": {
		Name: "dracula",
		Palette: Palette{
			Accent:   lipgloss.Color("#BD93F9"),
			Bg:       lipgloss.Color("#282A36"),
			BgLight:  lipgloss.Color("#44475A"),
			Subtle:   lipgloss.Color("#44475A"),
			Fg:       lipgloss.Color("#F8F8F2"),
			FgDim:    lipgloss.Color("#6272A4"),
			Success:  lipgloss.Color("#50FA7B"),
			Failure:  lipgloss.Color("#FF5555"),
			Running:  lipgloss.Color("#F1FA8C"),
			Branch:   lipgloss.Color("#FF79C6"),
			Repo:     lipgloss.Color("#8BE9FD"),
			Duration: lipgloss.Color("#FFB86C"),
			Key:      lipgloss.Color("#8BE9FD"),
		},
	},
	"tokyo-night": {
		Name: "tokyo-night",
		Palette: Palette{
			Accent:   lipgloss.Color("#7AA2F7"),
			Bg:       lipgloss.Color("#1A1B26"),
			BgLight:  lipgloss.Color("#2B2D42"),
			Subtle:   lipgloss.Color("#565F89"),
			Fg:       lipgloss.Color("#C0CAF5"),
			FgDim:    lipgloss.Color("#565F89"),
			Success:  lipgloss.Color("#9ECE6A"),
			Failure:  lipgloss.Color("#F7768E"),
			Running:  lipgloss.Color("#E0AF68"),
			Branch:   lipgloss.Color("#BB9AF7"),
			Repo:     lipgloss.Color("#7AA2F7"),
			Duration: lipgloss.Color("#FF9E64"),
			Key:      lipgloss.Color("#7AA2F7"),
		},
	},
	"tokyo-night-alt": {
		Name: "tokyo-night-alt",
		Palette: Palette{
			Accent:   lipgloss.Color("#BB9AF7"),
			Bg:       lipgloss.Color("#1A1B26"),
			BgLight:  lipgloss.Color("#2B2D42"),
			Subtle:   lipgloss.Color("#565F89"),
			Fg:       lipgloss.Color("#C0CAF5"),
			FgDim:    lipgloss.Color("#565F89"),
			Success:  lipgloss.Color("#9ECE6A"),
			Failure:  lipgloss.Color("#F7768E"),
			Running:  lipgloss.Color("#E0AF68"),
			Branch:   lipgloss.Color("#7AA2F7"),
			Repo:     lipgloss.Color("#BB9AF7"),
			Duration: lipgloss.Color("#FF9E64"),
			Key:      lipgloss.Color("#BB9AF7"),
		},
	},
	"light": {
		Name: "light",
		Palette: Palette{
			Accent:   lipgloss.Color("#0366D6"),
			Bg:       lipgloss.Color("#FAFBFC"),
			BgLight:  lipgloss.Color("#E1E4E8"),
			Subtle:   lipgloss.Color("#D1D5DA"),
			Fg:       lipgloss.Color("#24292E"),
			FgDim:    lipgloss.Color("#6A737D"),
			Success:  lipgloss.Color("#28A745"),
			Failure:  lipgloss.Color("#D73A49"),
			Running:  lipgloss.Color("#B08800"),
			Branch:   lipgloss.Color("#6F42C1"),
			Repo:     lipgloss.Color("#0366D6"),
			Duration: lipgloss.Color("#E36209"),
			Key:      lipgloss.Color("#0366D6"),
		},
	},
}

// ThemeByName returns the theme with the given name, or Dracula if not found
func ThemeByName(name string) Theme {
	if t, ok := Themes[name]; ok {
		return t
	}
	return Themes["dracula"]
}
