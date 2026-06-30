package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	protoTruncateLines = 10
	protoTruncateFmt   = "… (%d lines hidden) [Ctrl+T 对话区 → e 展开]"
	protoThinkingFmt   = "… %d earlier lines hidden [Ctrl+T 对话区 → e 全文]"
)

// 可选块：推理、工具调用（header 行）；工具结果与调用合并为一块（对齐 crush）。
func isFocusable(entries []protoEntry, idx int) bool {
	if idx < 0 || idx >= len(entries) {
		return false
	}
	switch entries[idx].kind {
	case protoReasoning, protoToolCall:
		return true
	default:
		return false
	}
}

func isPairedToolResult(entries []protoEntry, idx int) bool {
	return entries[idx].kind == protoToolResult &&
		idx > 0 && entries[idx-1].kind == protoToolCall
}

func hasPairedToolResult(entries []protoEntry, callIdx int) bool {
	return entries[callIdx].kind == protoToolCall &&
		callIdx+1 < len(entries) && entries[callIdx+1].kind == protoToolResult
}

func renderProtoTranscriptCrush(entries []protoEntry, opts transcriptRenderOpts) string {
	var blocks []string
	for i, e := range entries {
		if isPairedToolResult(entries, i) {
			continue
		}
		var block string
		switch e.kind {
		case protoUser:
			block = crushUser(e, opts.theme)
		case protoReasoning:
			block = crushThinking(i, e, opts)
		case protoAnswer:
			block = crushAssistant(e, opts.theme)
		case protoToolCall:
			block = renderCrushToolBlock(i, entries, opts)
		case protoUsage:
			block = lipgloss.NewStyle().Foreground(opts.theme.dim).PaddingLeft(2).Render(e.text)
		}
		if block != "" {
			blocks = append(blocks, renderBlockFocus(i, block, opts))
		}
	}
	return strings.TrimRight(strings.Join(blocks, "\n"), "\n")
}

func renderBlockFocus(idx int, content string, opts transcriptRenderOpts) string {
	if !opts.transcriptFocus || idx != opts.focusIdx {
		return content
	}
	marker := lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true).Render("▶ ")
	lines := strings.Split(content, "\n")
	if len(lines) > 0 {
		lines[0] = marker + lines[0]
	}
	return strings.Join(lines, "\n")
}

func crushUser(e protoEntry, t protoTheme) string {
	return protoCrushLeftBar(t.user, e.text, false)
}

func crushAssistant(e protoEntry, t protoTheme) string {
	return protoCrushLeftBar(t.agent, e.text, true)
}

func protoCrushLeftBar(color lipgloss.Color, text string, dim bool) string {
	bar := lipgloss.NewStyle().Foreground(color).Render("▌")
	style := lipgloss.NewStyle()
	if dim {
		style = style.Foreground(color)
	}
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		lines = append(lines, bar+style.Render(line))
	}
	return strings.Join(lines, "\n")
}

func crushToolHeader(e protoEntry, t protoTheme) string {
	name := "Read"
	param := e.meta
	if param == "" {
		param = e.text
	}
	if parts := strings.SplitN(param, "·", 2); len(parts) == 2 {
		name = strings.TrimSpace(parts[0])
		param = strings.TrimSpace(parts[1])
	}
	line := lipgloss.NewStyle().Foreground(t.tool).Render("✓ "+name) + " " +
		lipgloss.NewStyle().Foreground(t.dim).Render(param)
	return lipgloss.NewStyle().PaddingLeft(2).Render(line)
}

func renderCrushToolBlock(callIdx int, entries []protoEntry, opts transcriptRenderOpts) string {
	header := crushToolHeader(entries[callIdx], opts.theme)
	if !hasPairedToolResult(entries, callIdx) {
		return header
	}
	body := crushToolBodyContent(entries[callIdx+1], opts.expanded[callIdx], opts.theme)
	return header + "\n" + body
}

func crushToolBodyContent(result protoEntry, expanded bool, t protoTheme) string {
	lines := strings.Split(result.text, "\n")
	display := lines
	hidden := 0
	if !expanded && len(lines) > protoTruncateLines {
		display = lines[:protoTruncateLines]
		hidden = len(lines) - protoTruncateLines
	}
	var out []string
	for i, ln := range display {
		num := lipgloss.NewStyle().Foreground(t.dim).Render(fmt.Sprintf("%2d │ ", i+1))
		code := lipgloss.NewStyle().Foreground(t.dim).Render(ln)
		out = append(out, lipgloss.NewStyle().PaddingLeft(2).Render(num+code))
	}
	if hidden > 0 {
		out = append(out, lipgloss.NewStyle().Foreground(t.dim).PaddingLeft(2).Render(
			fmt.Sprintf(protoTruncateFmt, hidden)))
	}
	return strings.Join(out, "\n")
}

func crushThinking(idx int, e protoEntry, opts transcriptRenderOpts) string {
	t := opts.theme
	lines := strings.Split(e.text, "\n")
	expanded := opts.expanded[idx]

	var body []string
	hidden := 0
	if expanded || len(lines) <= protoTruncateLines {
		body = lines
	} else {
		hidden = len(lines) - protoTruncateLines
		body = lines[len(lines)-protoTruncateLines:]
	}

	head := lipgloss.NewStyle().Foreground(t.reasoning).Italic(true).PaddingLeft(2).Render("Thought")
	var out []string
	out = append(out, head)
	if hidden > 0 {
		out = append(out, lipgloss.NewStyle().Foreground(t.dim).PaddingLeft(2).Render(
			fmt.Sprintf(protoThinkingFmt, hidden)))
	}
	for _, ln := range body {
		out = append(out, lipgloss.NewStyle().Foreground(t.dim).Italic(true).PaddingLeft(2).Render(ln))
	}
	return strings.Join(out, "\n")
}

func initProtoExpanded(entries []protoEntry) map[int]bool {
	return make(map[int]bool)
}

func (m *protoModel) toggleExpandKind(kind protoEntryKind) {
	switch kind {
	case protoReasoning:
		expand := !m.allExpandedKind(protoReasoning)
		for i, e := range m.entries {
			if e.kind == protoReasoning {
				m.expanded[i] = expand
			}
		}
	case protoToolResult, protoToolCall:
		expand := !m.allToolBlocksExpanded()
		for i := range m.entries {
			if hasPairedToolResult(m.entries, i) {
				m.expanded[i] = expand
			}
		}
	}
}

func (m *protoModel) allExpandedKind(kind protoEntryKind) bool {
	any := false
	for i, e := range m.entries {
		if e.kind != kind {
			continue
		}
		any = true
		if !m.expanded[i] {
			return false
		}
	}
	return any
}

func (m *protoModel) allToolBlocksExpanded() bool {
	any := false
	for i := range m.entries {
		if !hasPairedToolResult(m.entries, i) {
			continue
		}
		any = true
		if !m.expanded[i] {
			return false
		}
	}
	return any
}

func (m *protoModel) setAllExpanded(expanded bool) {
	for i, e := range m.entries {
		if e.kind == protoReasoning {
			m.expanded[i] = expanded
		}
		if hasPairedToolResult(m.entries, i) {
			m.expanded[i] = expanded
		}
	}
}

func (m *protoModel) toggleFocusedExpand() {
	if !isFocusable(m.entries, m.focusIdx) {
		return
	}
	m.expanded[m.focusIdx] = !m.expanded[m.focusIdx]
}

func (m *protoModel) moveFocus(delta int) {
	if len(m.entries) == 0 {
		m.focusIdx = -1
		return
	}
	start := m.focusIdx
	if start < 0 {
		start = 0
	}
	for step := 1; step <= len(m.entries); step++ {
		idx := start + delta*step
		for idx < 0 {
			idx += len(m.entries)
		}
		idx %= len(m.entries)
		if isFocusable(m.entries, idx) {
			m.focusIdx = idx
			return
		}
	}
}

func firstFocusable(entries []protoEntry) int {
	for i := range entries {
		if isFocusable(entries, i) {
			return i
		}
	}
	return 0
}
