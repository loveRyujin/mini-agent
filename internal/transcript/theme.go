package transcript

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	Name      string
	Gold      lipgloss.Color
	Border    lipgloss.Color
	Dim       lipgloss.Color
	User      lipgloss.Color
	Agent     lipgloss.Color
	Tool      lipgloss.Color
	Reasoning lipgloss.Color
	Error     lipgloss.Color
}

var DefaultTheme = Theme{
	Name:      "Maple",
	Gold:      "220",
	Border:    "130",
	Dim:       "244",
	User:      "82",
	Agent:     "51",
	Tool:      "214",
	Reasoning: "244",
	Error:     "9",
}
