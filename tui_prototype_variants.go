package main

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

type protoVariant struct {
	key  string
	name string
}

var protoVariants = []protoVariant{
	{key: "A", name: "像素对话"},
	{key: "B", name: "冒险岛面板"},
	{key: "C", name: "HUD 日志"},
}

type protoRenderCtx struct {
	width, height  int
	viewport       viewport.Model
	textarea       textarea.Model
	entries        []protoEntry
	turnActive     bool
	theme          protoTheme
	transcriptMode int
	expanded       map[int]bool
	focusIdx       int
	transcriptFocus bool
}

const protoInputHintText = "输入消息试试…"

func renderProtoInput(variantIdx int, ctx protoRenderCtx) string {
	t := ctx.theme
	if strings.TrimSpace(ctx.textarea.Value()) == "" {
		hint := lipgloss.NewStyle().Foreground(t.dim).Italic(true).Render(protoInputHintText)
		switch variantIdx {
		case 2:
			return lipgloss.JoinHorizontal(lipgloss.Top,
				lipgloss.NewStyle().Foreground(t.tool).Bold(true).Render("❯ "), hint)
		default:
			return lipgloss.JoinHorizontal(lipgloss.Top,
				lipgloss.NewStyle().Foreground(t.tool).Render("> "), hint)
		}
	}
	if variantIdx == 2 {
		return lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Foreground(t.tool).Bold(true).Render("❯ "), ctx.textarea.View())
	}
	return ctx.textarea.View()
}

func renderProtoVariantA(ctx protoRenderCtx) string {
	t := ctx.theme
	divider := lipgloss.NewStyle().Foreground(t.border).Render(strings.Repeat("═", ctx.width))
	meta := lipgloss.NewStyle().Foreground(t.dim).Render("Enter 发送 · Ctrl+C 退出")
	status := ""
	if ctx.turnActive {
		status = lipgloss.NewStyle().Foreground(t.tool).Render("  ◆ 生成中…")
	}
	return lipgloss.JoinVertical(lipgloss.Left, ctx.viewport.View(), divider, meta, renderProtoInput(0, ctx)+status)
}

func protoTranscriptA(entries []protoEntry, t protoTheme) string {
	var lines []string
	for _, e := range entries {
		switch e.kind {
		case protoUser:
			lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(t.user).Render("你: "+e.text))
		case protoReasoning:
			lines = append(lines, lipgloss.NewStyle().Foreground(t.reasoning).Italic(true).Render("推理: "+e.text))
		case protoAnswer:
			lines = append(lines, lipgloss.NewStyle().Foreground(t.agent).Render(e.text))
		case protoToolCall:
			lines = append(lines, lipgloss.NewStyle().Foreground(t.tool).Render("▣ 工具: "+e.text))
		case protoToolResult:
			lines = append(lines, lipgloss.NewStyle().Foreground(t.dim).Render("  └ "+e.text))
		case protoUsage:
			lines = append(lines, lipgloss.NewStyle().Foreground(t.dim).Render(e.text))
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func renderProtoVariantB(ctx protoRenderCtx) string {
	t := ctx.theme
	titleStyle := lipgloss.NewStyle().Background(t.gold).Foreground(lipgloss.Color("16")).Bold(true).Padding(0, 1)
	panelStyle := lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).BorderForeground(t.border).Padding(0, 1)
	footerStyle := lipgloss.NewStyle().Foreground(t.dim).Padding(0, 1)

	status := lipgloss.NewStyle().Foreground(t.user).Bold(true).Render("● 就绪")
	if ctx.turnActive {
		status = lipgloss.NewStyle().Foreground(t.tool).Bold(true).Render("● 生成中")
	}
	title := titleStyle.Width(ctx.width).Render(
		lipgloss.JoinHorizontal(lipgloss.Top,
			" mini-agent ",
			lipgloss.PlaceHorizontal(ctx.width-lipgloss.Width(" mini-agent ")-lipgloss.Width(status), lipgloss.Right, status),
		),
	)
	subtitle := lipgloss.NewStyle().Foreground(t.dim).Width(ctx.width).Padding(0, 1).
		Render(protoModelName + "  ·  " + protoWorkspace + "  ·  对话:" + transcriptModes[ctx.transcriptMode].name + "  ·  " + t.name)
	transcriptPanel := panelStyle.Width(ctx.width - 2).Render(ctx.viewport.View())
	inputPanel := panelStyle.Width(ctx.width - 2).Render(renderProtoInput(1, ctx))
	footer := footerStyle.Width(ctx.width).Render("Ctrl+T 对话区  ·  j/k 选块  ·  e/Space 展开  ·  Esc 回输入  ·  Enter 发送")
	return lipgloss.JoinVertical(lipgloss.Left, title, subtitle, transcriptPanel, inputPanel, footer)
}

func renderProtoVariantC(ctx protoRenderCtx) string {
	t := ctx.theme
	statusBar := lipgloss.NewStyle().Background(t.gold).Foreground(lipgloss.Color("16")).Bold(true).Width(ctx.width).Padding(0, 1).
		Render(" MAPLE · " + protoModelName + " · " + protoWorkspace)
	if ctx.turnActive {
		statusBar += lipgloss.NewStyle().Foreground(t.tool).Render("  ◆")
	}
	keyHints := lipgloss.NewStyle().Foreground(t.dim).Width(ctx.width).Padding(0, 1).
		Render("F9/F10 布局  ·  F11/F12 配色  ·  Enter 发送  ·  Ctrl+C 退出")
	return lipgloss.JoinVertical(lipgloss.Left, statusBar, ctx.viewport.View(), renderProtoInput(2, ctx), keyHints)
}

func protoTranscriptC(entries []protoEntry, t protoTheme) string {
	tag := lipgloss.NewStyle().Bold(true)
	var lines []string
	for _, e := range entries {
		switch e.kind {
		case protoUser:
			lines = append(lines, tag.Foreground(t.user).Render("[你]")+" "+e.text)
		case protoReasoning:
			for _, line := range strings.Split(e.text, "\n") {
				lines = append(lines, lipgloss.NewStyle().Foreground(t.dim).Render("   │ "+line))
			}
		case protoAnswer:
			lines = append(lines, tag.Foreground(t.agent).Render("[Agent]")+" "+e.text)
		case protoToolCall:
			lines = append(lines, tag.Foreground(t.tool).Render("[工具]")+" "+e.text)
		case protoToolResult:
			for _, line := range strings.Split(e.text, "\n") {
				lines = append(lines, lipgloss.NewStyle().Foreground(t.dim).Render("   > "+line))
			}
		case protoUsage:
			lines = append(lines, lipgloss.NewStyle().Foreground(t.dim).Render("// "+e.text))
		}
	}
	return strings.Join(lines, "\n")
}

func renderProtoTranscript(variantIdx int, entries []protoEntry, theme protoTheme, modeIdx int, expanded map[int]bool, focusIdx int, transcriptFocus bool) string {
	opts := transcriptRenderOpts{theme: theme, modeIdx: modeIdx, expanded: expanded, focusIdx: focusIdx, transcriptFocus: transcriptFocus}
	switch variantIdx {
	case 0:
		return protoTranscriptA(entries, theme)
	case 1:
		return renderTranscriptB(entries, opts)
	default:
		return protoTranscriptC(entries, theme)
	}
}

func renderProtoView(variantIdx int, ctx protoRenderCtx) string {
	switch variantIdx {
	case 0:
		return renderProtoVariantA(ctx)
	case 1:
		return renderProtoVariantB(ctx)
	default:
		return renderProtoVariantC(ctx)
	}
}

func protoReservedLines(variantIdx, inputH int) int {
	const banner, switcher = 1, 2 // switcher 留余量防窄屏折行
	switch variantIdx {
	case 0:
		return banner + switcher + 2 + inputH
	case 1:
		return banner + switcher + 1 + 2 + 3 + 3 + 1 // +1 副标题窄屏折行
	case 2:
		return banner + switcher + 1 + 1 + 2 // +1 底栏折行
	default:
		return banner + switcher + 2 + inputH
	}
}
