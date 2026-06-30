package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func TestOverlayModal_preservesStyledBase(t *testing.T) {
	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Background(lipgloss.Color("235")).
		Render("MODAL")

	const width, height = 40, 5
	result := overlayModal(modal, width, height)

	if !strings.Contains(ansi.Strip(result), "MODAL") {
		t.Fatalf("modal not visible in output:\n%s", result)
	}
	if lipgloss.Height(result) != height {
		t.Fatalf("output height = %d, want %d", lipgloss.Height(result), height)
	}
}

func TestOverlayModal_fillsTerminalHeight(t *testing.T) {
	modal := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Render("OK")
	result := overlayModal(modal, 30, 10)
	if lipgloss.Height(result) != 10 {
		t.Fatalf("output height = %d, want 10", lipgloss.Height(result))
	}
}
