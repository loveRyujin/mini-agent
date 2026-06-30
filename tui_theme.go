package main

import "github.com/charmbracelet/lipgloss"

type tuiTheme struct {
	name      string
	gold      lipgloss.Color
	border    lipgloss.Color
	dim       lipgloss.Color
	user      lipgloss.Color
	agent     lipgloss.Color
	tool      lipgloss.Color
	reasoning lipgloss.Color
	error     lipgloss.Color
}

var defaultTUITheme = tuiTheme{
	name:      "Maple",
	gold:      "220",
	border:    "130",
	dim:       "244",
	user:      "82",
	agent:     "51",
	tool:      "214",
	reasoning: "244",
	error:     "9",
}
