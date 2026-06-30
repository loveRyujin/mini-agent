package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type protoTheme struct {
	name      string
	gold      lipgloss.Color
	border    lipgloss.Color
	dim       lipgloss.Color
	user      lipgloss.Color
	agent     lipgloss.Color
	tool      lipgloss.Color
	reasoning lipgloss.Color
}

var protoThemes = []protoTheme{
	{name: "Maple", gold: "220", border: "130", dim: "244", user: "82", agent: "51", tool: "214", reasoning: "244"},
	{name: "Mono", gold: "245", border: "240", dim: "245", user: "252", agent: "250", tool: "248", reasoning: "240"},
	{name: "Ocean", gold: "117", border: "37", dim: "109", user: "42", agent: "45", tool: "216", reasoning: "109"},
	{name: "Vivid", gold: "226", border: "201", dim: "246", user: "46", agent: "39", tool: "208", reasoning: "246"},
}

type transcriptMode struct {
	key  string
	name string
}

// Crush 为默认；混合/标签保留作对比
var transcriptModes = []transcriptMode{
	{key: "C", name: "Crush"},
	{key: "M", name: "混合"},
	{key: "L", name: "标签"},
}

type transcriptRenderOpts struct {
	theme           protoTheme
	modeIdx         int
	expanded        map[int]bool
	focusIdx        int
	transcriptFocus bool
}

func renderTranscriptB(entries []protoEntry, opts transcriptRenderOpts) string {
	switch opts.modeIdx {
	case 1:
		return renderTranscriptCompact(entries, opts, compactHybrid)
	case 2:
		return renderTranscriptCompact(entries, opts, compactLabeled)
	default:
		return renderTranscriptCrush(entries, opts)
	}
}

type compactStyle int

const (
	compactHybrid compactStyle = iota
	compactLabeled
)

func renderTranscriptCompact(entries []protoEntry, opts transcriptRenderOpts, style compactStyle) string {
	var blocks []string
	for i, e := range entries {
		if isPairedToolResult(entries, i) {
			continue
		}
		var block string
		if e.kind == protoToolCall {
			block = renderCompactToolBlock(i, entries, opts, style)
		} else if style == compactHybrid {
			block = renderCompactHybrid(i, e, opts)
		} else {
			block = renderCompactLabeled(i, e, opts)
		}
		if block != "" {
			blocks = append(blocks, renderBlockFocus(i, block, opts))
		}
	}
	return strings.TrimRight(strings.Join(blocks, "\n"), "\n")
}

func renderCompactToolBlock(callIdx int, entries []protoEntry, opts transcriptRenderOpts, style compactStyle) string {
	t := opts.theme
	call := entries[callIdx]
	label := call.meta
	if label == "" {
		label = call.text
	}
	var head string
	if style == compactHybrid {
		head = styleLine(t.tool, "✓ ", label)
	} else {
		head = styleLine(t.tool, "工具  ", label)
	}
	if !hasPairedToolResult(entries, callIdx) {
		return head
	}
	body := crushToolBodyContent(entries[callIdx+1], opts.expanded[callIdx], t)
	return head + "\n" + body
}

func renderCompactHybrid(i int, e protoEntry, opts transcriptRenderOpts) string {
	t := opts.theme
	switch e.kind {
	case protoUser:
		return styleLine(t.user, "› 你  ", e.text)
	case protoReasoning:
		return renderTruncatable(i, e, opts, "… 推理", t.reasoning, true)
	case protoAnswer:
		return styleLine(t.agent, "« Agent  ", e.text)
	case protoToolCall:
		label := e.meta
		if label == "" {
			label = e.text
		}
		return styleLine(t.tool, "✓ ", label)
	case protoToolResult:
		return "" // 已与 tool call 合并
	case protoUsage:
		return lipgloss.NewStyle().Foreground(t.dim).Render("─ " + e.text)
	default:
		return e.text
	}
}

func renderCompactLabeled(i int, e protoEntry, opts transcriptRenderOpts) string {
	t := opts.theme
	switch e.kind {
	case protoUser:
		return styleLine(t.user, "你  ", e.text)
	case protoReasoning:
		return renderTruncatable(i, e, opts, "推理", t.reasoning, true)
	case protoAnswer:
		return styleLine(t.agent, "Agent  ", e.text)
	case protoToolCall:
		label := e.meta
		if label == "" {
			label = e.text
		}
		return styleLine(t.tool, "工具  ", label)
	case protoToolResult:
		return ""
	case protoUsage:
		return lipgloss.NewStyle().Foreground(t.dim).Render(e.text)
	default:
		return e.text
	}
}

func styleLine(color lipgloss.Color, prefix, text string) string {
	return lipgloss.NewStyle().Foreground(color).Render(prefix) + text
}

func renderTruncatable(idx int, e protoEntry, opts transcriptRenderOpts, label string, color lipgloss.Color, italic bool) string {
	lines := strings.Split(e.text, "\n")
	expanded := opts.expanded[idx]
	display, hidden := truncateHead(lines, expanded)

	head := lipgloss.NewStyle().Foreground(color).Render(label)
	if italic {
		head = lipgloss.NewStyle().Foreground(color).Italic(true).Render(label)
	}
	var out []string
	out = append(out, head)
	if hidden > 0 {
		out = append(out, lipgloss.NewStyle().Foreground(opts.theme.dim).Render(
			fmt.Sprintf(protoTruncateFmt, hidden)))
	}
	bodyStyle := lipgloss.NewStyle().PaddingLeft(2)
	if italic {
		bodyStyle = bodyStyle.Foreground(opts.theme.dim).Italic(true)
	}
	for _, ln := range display {
		out = append(out, bodyStyle.Render(ln))
	}
	return strings.Join(out, "\n")
}

func renderTruncatableCode(idx int, e protoEntry, opts transcriptRenderOpts) string {
	lines := strings.Split(e.text, "\n")
	expanded := opts.expanded[idx]
	display, hidden := truncateHead(lines, expanded)

	head := lipgloss.NewStyle().Foreground(opts.theme.tool).PaddingLeft(2).Render("✓ " + e.meta)
	var out []string
	out = append(out, head)
	for i, ln := range display {
		num := lipgloss.NewStyle().Foreground(opts.theme.dim).Render(fmt.Sprintf("%2d │ ", i+1))
		out = append(out, lipgloss.NewStyle().PaddingLeft(2).Render(num+ln))
	}
	if hidden > 0 {
		out = append(out, lipgloss.NewStyle().Foreground(opts.theme.dim).PaddingLeft(2).Render(
			fmt.Sprintf(protoTruncateFmt, hidden)))
	}
	return strings.Join(out, "\n")
}

func truncateHead(lines []string, expanded bool) (display []string, hidden int) {
	if expanded || len(lines) <= protoTruncateLines {
		return lines, 0
	}
	return lines[:protoTruncateLines], len(lines) - protoTruncateLines
}
