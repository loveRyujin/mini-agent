package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const responseContextHeight = 10

const (
	truncateFmt  = "… (%d lines hidden) [Ctrl+T 对话区 → e 展开]"
	thinkingFmt  = "… %d earlier lines hidden [Ctrl+T 对话区 → e 全文]"
)

type TranscriptRenderOpts struct {
	Theme           tuiTheme
	Expanded        map[int]bool
	FocusIdx        int
	TranscriptFocus bool
}

func isTranscriptFocusable(entries []transcriptEntry, idx int) bool {
	if idx < 0 || idx >= len(entries) {
		return false
	}
	switch entries[idx].kind {
	case entryReasoning, entryToolCall:
		return true
	default:
		return false
	}
}

func isPairedToolResultEntry(entries []transcriptEntry, idx int) bool {
	return entries[idx].kind == entryToolResult &&
		idx > 0 && entries[idx-1].kind == entryToolCall
}

func hasPairedToolResultEntry(entries []transcriptEntry, callIdx int) bool {
	return entries[callIdx].kind == entryToolCall &&
		callIdx+1 < len(entries) && entries[callIdx+1].kind == entryToolResult
}

func renderTranscriptCrush(entries []transcriptEntry, opts TranscriptRenderOpts) string {
	var blocks []string
	for i, e := range entries {
		if isPairedToolResultEntry(entries, i) {
			continue
		}
		var block string
		switch e.kind {
		case entryUser:
			block = crushUserLine(e, opts.Theme)
		case entryReasoning:
			block = crushThinkingBlock(i, e, opts)
		case entryAnswer:
			block = crushAssistantLine(e, opts.Theme)
		case entryToolCall:
			block = renderCrushToolBlockEntry(i, entries, opts)
		case entryError:
			block = lipgloss.NewStyle().Foreground(opts.Theme.error).PaddingLeft(2).Render("错误: " + e.text)
		case entryUsage:
			block = lipgloss.NewStyle().Foreground(opts.Theme.dim).PaddingLeft(2).Render(e.text)
		}
		if block != "" {
			blocks = append(blocks, renderTranscriptBlockFocus(i, block, opts))
		}
	}
	return strings.TrimRight(strings.Join(blocks, "\n"), "\n")
}

func renderTranscriptBlockFocus(idx int, content string, opts TranscriptRenderOpts) string {
	if !opts.TranscriptFocus || idx != opts.FocusIdx {
		return content
	}
	marker := lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true).Render("▶ ")
	lines := strings.Split(content, "\n")
	if len(lines) > 0 {
		lines[0] = marker + lines[0]
	}
	return strings.Join(lines, "\n")
}

func crushUserLine(e transcriptEntry, t tuiTheme) string {
	return crushLeftBar(t.user, e.text, false)
}

func crushAssistantLine(e transcriptEntry, t tuiTheme) string {
	return crushLeftBar(t.agent, e.text, true)
}

func crushLeftBar(color lipgloss.Color, text string, dim bool) string {
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

func crushToolHeaderEntry(e transcriptEntry, t tuiTheme) string {
	name := e.toolName
	param := e.meta
	if name == "" {
		name = "Tool"
	}
	if param == "" {
		param = e.text
	}
	line := lipgloss.NewStyle().Foreground(t.tool).Render("✓ "+name) + " " +
		lipgloss.NewStyle().Foreground(t.dim).Render(param)
	return lipgloss.NewStyle().PaddingLeft(2).Render(line)
}

func renderCrushToolBlockEntry(callIdx int, entries []transcriptEntry, opts TranscriptRenderOpts) string {
	header := crushToolHeaderEntry(entries[callIdx], opts.Theme)
	if !hasPairedToolResultEntry(entries, callIdx) {
		return header
	}
	body := crushToolBodyEntry(entries[callIdx+1], opts.Expanded[callIdx], opts.Theme)
	return header + "\n" + body
}

func crushToolBodyEntry(result transcriptEntry, expanded bool, t tuiTheme) string {
	lines := strings.Split(result.text, "\n")
	display := lines
	hidden := 0
	if !expanded && len(lines) > responseContextHeight {
		display = lines[:responseContextHeight]
		hidden = len(lines) - responseContextHeight
	}
	var out []string
	for i, ln := range display {
		num := lipgloss.NewStyle().Foreground(t.dim).Render(fmt.Sprintf("%2d │ ", i+1))
		code := lipgloss.NewStyle().Foreground(t.dim).Render(ln)
		out = append(out, lipgloss.NewStyle().PaddingLeft(2).Render(num+code))
	}
	if hidden > 0 {
		out = append(out, lipgloss.NewStyle().Foreground(t.dim).PaddingLeft(2).Render(
			fmt.Sprintf(truncateFmt, hidden)))
	}
	return strings.Join(out, "\n")
}

func crushThinkingBlock(idx int, e transcriptEntry, opts TranscriptRenderOpts) string {
	t := opts.Theme
	lines := strings.Split(e.text, "\n")
	expanded := opts.Expanded[idx]

	var body []string
	hidden := 0
	if expanded || len(lines) <= responseContextHeight {
		body = lines
	} else {
		hidden = len(lines) - responseContextHeight
		body = lines[len(lines)-responseContextHeight:]
	}

	head := lipgloss.NewStyle().Foreground(t.reasoning).Italic(true).PaddingLeft(2).Render("Thought")
	var out []string
	out = append(out, head)
	if hidden > 0 {
		out = append(out, lipgloss.NewStyle().Foreground(t.dim).PaddingLeft(2).Render(
			fmt.Sprintf(thinkingFmt, hidden)))
	}
	for _, ln := range body {
		out = append(out, lipgloss.NewStyle().Foreground(t.dim).Italic(true).PaddingLeft(2).Render(ln))
	}
	return strings.Join(out, "\n")
}

func firstTranscriptFocusable(entries []transcriptEntry) int {
	for i := range entries {
		if isTranscriptFocusable(entries, i) {
			return i
		}
	}
	return 0
}
