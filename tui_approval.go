package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func renderApprovalModal(m *tuiModel) string {
	t := m.theme
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.border).
		Padding(1, 2).
		Width(min(60, m.width-4))

	title := lipgloss.NewStyle().Foreground(t.gold).Bold(true).Render("Shell Execution — 需要批准")
	cmd := lipgloss.NewStyle().Foreground(t.agent).Render(m.approvalCommand)
	hint := lipgloss.NewStyle().Foreground(t.dim).Render("Y 允许  ·  N 拒绝")

	body := lipgloss.JoinVertical(lipgloss.Left, title, "", cmd, "", hint)
	return border.Render(body)
}

func overlayCenter(base, modal string, width int) string {
	baseLines := strings.Split(base, "\n")
	modalLines := strings.Split(modal, "\n")
	modalW := lipgloss.Width(modal)
	startY := max(0, (len(baseLines)-len(modalLines))/2)
	startX := max(0, (width-modalW)/2)

	for i, ml := range modalLines {
		y := startY + i
		for len(baseLines) <= y {
			baseLines = append(baseLines, "")
		}
		line := baseLines[y]
		if lipgloss.Width(line) < width {
			line += strings.Repeat(" ", width-lipgloss.Width(line))
		}
		runes := []rune(line)
		mRunes := []rune(ml)
		for j, r := range mRunes {
			x := startX + j
			if x < len(runes) {
				runes[x] = r
			}
		}
		baseLines[y] = string(runes)
	}
	return strings.Join(baseLines, "\n")
}

func (m *tuiModel) handleApprovalKeys(msg tea.KeyMsg) {
	switch strings.ToLower(msg.String()) {
	case "y":
		m.approvalReplyCh <- true
		m.clearApproval()
	case "n":
		m.approvalReplyCh <- false
		m.clearApproval()
	}
}

func (m *tuiModel) clearApproval() {
	m.approvalCommand = ""
	m.approvalReplyCh = nil
}
