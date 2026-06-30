package main

import (
	"strings"
	"unicode/utf8"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type textPos struct {
	line int
	col  int
}

func splitPlainLines(content string) []string {
	lines := strings.Split(content, "\n")
	plain := make([]string, len(lines))
	for i, line := range lines {
		plain[i] = ansi.Strip(line)
	}
	return plain
}

func normalizeSelection(a, b textPos) (textPos, textPos) {
	if a.line > b.line || (a.line == b.line && a.col > b.col) {
		a, b = b, a
	}
	return a, b
}

func extractSelectionText(plainLines []string, start, end textPos) string {
	start, end = normalizeSelection(start, end)
	if len(plainLines) == 0 {
		return ""
	}

	var b strings.Builder
	for line := start.line; line <= end.line && line < len(plainLines); line++ {
		if line > start.line {
			b.WriteByte('\n')
		}
		runes := []rune(plainLines[line])
		from := 0
		to := len(runes)
		if line == start.line {
			from = min(start.col, len(runes))
		}
		if line == end.line {
			to = min(end.col+1, len(runes))
		}
		if from < to {
			b.WriteString(string(runes[from:to]))
		}
	}
	return b.String()
}

func highlightSelection(content string, plainLines []string, start, end textPos) string {
	start, end = normalizeSelection(start, end)
	if start.line >= len(plainLines) {
		return content
	}

	styledLines := strings.Split(content, "\n")
	hi := lipgloss.NewStyle().Background(lipgloss.Color("237")).Foreground(lipgloss.Color("255"))
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	out := make([]string, len(styledLines))
	for i, styled := range styledLines {
		if i >= len(plainLines) {
			out[i] = styled
			continue
		}
		plain := plainLines[i]
		if plain == "" {
			out[i] = styled
			continue
		}
		if i < start.line || i > end.line {
			out[i] = styled
			continue
		}

		runes := []rune(plain)
		from := 0
		to := len(runes)
		if i == start.line {
			from = min(start.col, len(runes))
		}
		if i == end.line {
			to = min(end.col+1, len(runes))
		}
		if from >= to {
			out[i] = styled
			continue
		}

		var line strings.Builder
		if from > 0 {
			line.WriteString(dim.Render(string(runes[:from])))
		}
		line.WriteString(hi.Render(string(runes[from:to])))
		if to < len(runes) {
			line.WriteString(dim.Render(string(runes[to:])))
		}
		out[i] = line.String()
	}
	return strings.Join(out, "\n")
}

func (m *tuiModel) hasSelection() bool {
	return m.selActive()
}

func (m *tuiModel) selActive() bool {
	return m.selStart.line >= 0 && m.selEnd.line >= 0 &&
		(m.selStart != m.selEnd || m.selecting)
}

func (m *tuiModel) clearSelection() {
	m.selecting = false
	m.selStart = textPos{line: -1, col: -1}
	m.selEnd = textPos{line: -1, col: -1}
}

func (m *tuiModel) copySelection() {
	if !m.selActive() && len(m.plainLines) == 0 {
		return
	}
	text := extractSelectionText(m.plainLines, m.selStart, m.selEnd)
	if strings.TrimSpace(text) == "" {
		return
	}
	_ = clipboard.WriteAll(text)
	m.copyNotice = "已复制到剪贴板"
}

func (m *tuiModel) updateTranscriptHitbox() {
	t := m.theme
	titleStyle := lipgloss.NewStyle().Background(t.gold).Foreground(lipgloss.Color("16")).Bold(true).Padding(0, 1)
	status := lipgloss.NewStyle().Foreground(t.user).Bold(true).Render("● 就绪")
	title := titleStyle.Width(m.width).Render(
		lipgloss.JoinHorizontal(lipgloss.Top,
			" mini-agent ",
			lipgloss.PlaceHorizontal(m.width-lipgloss.Width(" mini-agent ")-lipgloss.Width(status), lipgloss.Right, status),
		),
	)
	subtitle := lipgloss.NewStyle().Foreground(t.dim).Width(m.width).Padding(0, 1).
		Render(m.agent.Model + "  ·  " + m.workspace + "  ·  " + t.name)

	m.transcriptOriginY = lipgloss.Height(title) + lipgloss.Height(subtitle) + 1
	m.transcriptOriginX = 3
	m.transcriptHitH = m.viewport.Height
	m.transcriptHitW = m.viewport.Width
}

func (m *tuiModel) mouseInTranscript(msg tea.MouseMsg) bool {
	if msg.Y < m.transcriptOriginY || msg.Y >= m.transcriptOriginY+m.transcriptHitH {
		return false
	}
	if msg.X < m.transcriptOriginX || msg.X >= m.transcriptOriginX+m.transcriptHitW {
		return false
	}
	return true
}

func (m *tuiModel) mouseToContentPos(msg tea.MouseMsg) textPos {
	relY := msg.Y - m.transcriptOriginY
	if relY < 0 {
		relY = 0
	}
	if relY >= m.transcriptHitH {
		relY = m.transcriptHitH - 1
	}

	relX := msg.X - m.transcriptOriginX
	if relX < 0 {
		relX = 0
	}

	line := m.viewport.YOffset + relY
	if len(m.plainLines) == 0 {
		return textPos{line: 0, col: 0}
	}
	if line >= len(m.plainLines) {
		line = len(m.plainLines) - 1
	}

	plain := m.plainLines[line]
	col := relX
	if col > utf8.RuneCountInString(plain) {
		col = utf8.RuneCountInString(plain)
	}
	return textPos{line: line, col: col}
}

func (m *tuiModel) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.approvalReplyCh != nil {
		return m, nil
	}

	if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
		if m.mouseInTranscript(msg) {
			m.viewport, _ = m.viewport.Update(msg)
			m.followTail = m.viewport.AtBottom()
			m.clearSelection()
			m.copyNotice = ""
		}
		return m, nil
	}

	if !m.mouseInTranscript(msg) {
		return m, nil
	}

	pos := m.mouseToContentPos(msg)
	switch msg.Action {
	case tea.MouseActionPress:
		if msg.Button == tea.MouseButtonLeft {
			m.selecting = true
			m.selStart = pos
			m.selEnd = pos
			m.copyNotice = ""
			m.syncViewport()
		}
	case tea.MouseActionMotion:
		if m.selecting {
			m.selEnd = pos
			m.syncViewport()
		}
	case tea.MouseActionRelease:
		if m.selecting {
			m.selecting = false
			m.selEnd = pos
			if m.selStart != m.selEnd {
				m.copySelection()
			} else {
				m.clearSelection()
			}
			m.syncViewport()
		}
	}
	return m, nil
}
