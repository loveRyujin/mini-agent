package transcript

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const responseContextHeight = 10

const (
	truncateFmt = "… (%d lines hidden) [Ctrl+T 对话区 → e 展开]"
	thinkingFmt = "… %d earlier lines hidden [Ctrl+T 对话区 → e 全文]"
)

func renderCrush(entries []Entry, opts RenderOpts) string {
	var blocks []string
	for i, e := range entries {
		if IsPairedToolResult(entries, i) {
			continue
		}
		var block string
		switch e.Kind {
		case EntryUser:
			block = crushUserLine(e, opts.Theme)
		case EntryReasoning:
			block = crushThinkingBlock(i, e, opts)
		case EntryAnswer:
			block = crushAssistantLine(e, opts.Theme)
		case EntryToolCall:
			block = renderCrushToolBlockEntry(i, entries, opts)
		case EntryApproval:
			block = lipgloss.NewStyle().Foreground(opts.Theme.Tool).PaddingLeft(2).
				Render("⏸ 等待批准: " + lipgloss.NewStyle().Foreground(opts.Theme.Agent).Render(e.Text))
		case EntryError:
			block = lipgloss.NewStyle().Foreground(opts.Theme.Error).PaddingLeft(2).Render("错误: " + e.Text)
		case EntryUsage:
			block = lipgloss.NewStyle().Foreground(opts.Theme.Dim).PaddingLeft(2).Render(e.Text)
		case EntrySystem:
			block = lipgloss.NewStyle().Foreground(opts.Theme.Dim).PaddingLeft(2).Render(e.Text)
		}
		if block != "" {
			blocks = append(blocks, renderBlockFocus(i, block, opts))
		}
	}
	return strings.TrimRight(strings.Join(blocks, "\n"), "\n")
}

func renderBlockFocus(idx int, content string, opts RenderOpts) string {
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

func crushUserLine(e Entry, t Theme) string {
	return crushLeftBar(t.User, e.Text, false)
}

func crushAssistantLine(e Entry, t Theme) string {
	return crushLeftBar(t.Agent, e.Text, true)
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

func crushToolHeaderEntry(e Entry, t Theme) string {
	name := e.ToolName
	param := e.Meta
	if name == "" {
		name = "Tool"
	}
	if param == "" {
		param = e.Text
	}
	line := lipgloss.NewStyle().Foreground(t.Tool).Render("✓ "+name) + " " +
		lipgloss.NewStyle().Foreground(t.Dim).Render(param)
	return lipgloss.NewStyle().PaddingLeft(2).Render(line)
}

func renderCrushToolBlockEntry(callIdx int, entries []Entry, opts RenderOpts) string {
	header := crushToolHeaderEntry(entries[callIdx], opts.Theme)
	if !HasPairedToolResult(entries, callIdx) {
		return header
	}
	body := crushToolBodyEntry(entries[callIdx+1], opts.Expanded[callIdx], opts.Theme)
	return header + "\n" + body
}

func crushToolBodyEntry(result Entry, expanded bool, t Theme) string {
	lines := strings.Split(result.Text, "\n")
	display := lines
	hidden := 0
	if !expanded && len(lines) > responseContextHeight {
		display = lines[:responseContextHeight]
		hidden = len(lines) - responseContextHeight
	}
	var out []string
	for i, ln := range display {
		num := lipgloss.NewStyle().Foreground(t.Dim).Render(fmt.Sprintf("%2d │ ", i+1))
		code := lipgloss.NewStyle().Foreground(t.Dim).Render(ln)
		out = append(out, lipgloss.NewStyle().PaddingLeft(2).Render(num+code))
	}
	if hidden > 0 {
		out = append(out, lipgloss.NewStyle().Foreground(t.Dim).PaddingLeft(2).Render(
			fmt.Sprintf(truncateFmt, hidden)))
	}
	return strings.Join(out, "\n")
}

func crushThinkingBlock(idx int, e Entry, opts RenderOpts) string {
	t := opts.Theme
	lines := strings.Split(e.Text, "\n")
	expanded := opts.Expanded[idx]
	var body []string
	hidden := 0
	if expanded || len(lines) <= responseContextHeight {
		body = lines
	} else {
		hidden = len(lines) - responseContextHeight
		body = lines[len(lines)-responseContextHeight:]
	}
	head := lipgloss.NewStyle().Foreground(t.Reasoning).Italic(true).PaddingLeft(2).Render("Thought")
	var out []string
	out = append(out, head)
	if hidden > 0 {
		out = append(out, lipgloss.NewStyle().Foreground(t.Dim).PaddingLeft(2).Render(
			fmt.Sprintf(thinkingFmt, hidden)))
	}
	for _, ln := range body {
		out = append(out, lipgloss.NewStyle().Foreground(t.Dim).Italic(true).PaddingLeft(2).Render(ln))
	}
	return strings.Join(out, "\n")
}
