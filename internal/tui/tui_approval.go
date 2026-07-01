package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
)

const approvalScrimColor = "235"

func renderApprovalModal(m *model) string {
	t := m.theme
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Background(lipgloss.Color("234")).
		Padding(1, 2).
		Width(min(60, m.width-4))

	title := lipgloss.NewStyle().Foreground(t.Gold).Bold(true).Render("Shell Execution — 需要批准")
	cmd := lipgloss.NewStyle().Foreground(t.Agent).Render(m.approvalCommand)
	hint := lipgloss.NewStyle().Foreground(t.Dim).Render("Y 允许  ·  N 拒绝")

	body := lipgloss.JoinVertical(lipgloss.Left, title, "", cmd, "", hint)
	return border.Render(body)
}

func renderDimScrim(width, height int) string {
	if width < 1 || height < 1 {
		return ""
	}
	line := lipgloss.NewStyle().
		Background(lipgloss.Color(approvalScrimColor)).
		Width(width).
		Render("")
	lines := make([]string, height)
	for i := range lines {
		lines[i] = line
	}
	return strings.Join(lines, "\n")
}

func overlayModal(modal string, width, height int) string {
	buf := cellbuf.NewBuffer(width, height)
	cellbuf.SetContent(buf, renderDimScrim(width, height))

	modalW := lipgloss.Width(modal)
	modalH := lipgloss.Height(modal)
	x := max(0, (width-modalW)/2)
	y := max(0, (height-modalH)/2)
	cellbuf.SetContentRect(buf, modal, cellbuf.Rect(x, y, modalW, modalH))
	return cellbuf.Render(buf)
}

func (m *model) handleApprovalKeys(msg tea.KeyMsg) {
	switch strings.ToLower(msg.String()) {
	case "y":
		m.approvalReplyCh <- true
		m.clearApproval()
	case "n":
		m.approvalReplyCh <- false
		m.clearApproval()
	}
}

func (m *model) clearApproval() {
	m.approvalCommand = ""
	m.approvalReplyCh = nil
	m.textarea.Focus()
}
