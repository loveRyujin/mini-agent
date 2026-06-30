package main

import "strings"

// PROTOTYPE — mock data for TUI style exploration. Delete when done.

type protoEntryKind int

const (
	protoUser protoEntryKind = iota
	protoReasoning
	protoAnswer
	protoToolCall
	protoToolResult
	protoUsage
)

type protoEntry struct {
	kind protoEntryKind
	text string
	meta string // 工具名/文件路径摘要
}

func prototypeTranscript() []protoEntry {
	return []protoEntry{
		{kind: protoUser, text: "帮我在 tui.go 里加一个 Ctrl+L 清屏快捷键"},
		{
			kind: protoReasoning,
			text: "用户想要清屏功能。需要先确认 bubbletea 是否已有现成处理，再查 viewport 重置方式。\n" +
				"若仅清空 transcript，不必动 viewport 滚动位置。",
		},
		{kind: protoToolCall, text: `Read(path="tui.go")`, meta: "Read · tui.go"},
		{
			kind: protoToolResult,
			meta: "tui.go",
			text: "package main\n\nimport (\n\t\"context\"\n\t\"strings\"\n\n\ttea \"github.com/charmbracelet/bubbletea\"\n)\n\nconst inputMinHeight = 3\n\nfunc (m *tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {\n\t// ...\n}\n\n// ... 201 lines",
		},
		{kind: protoAnswer, text: "可以在 Update 里监听 tea.KeyCtrlL，清空 transcript 并 syncViewport。需要我现在改吗？"},
		{kind: protoUsage, text: "Token — 完成: 128, 提示: 2048, 合计: 2176"},
	}
}

func (e protoEntry) lineCount() int {
	if e.text == "" {
		return 0
	}
	return strings.Count(e.text, "\n") + 1
}

const (
	protoModelName = "deepseek-r1:latest"
	protoWorkspace = "~/github/mini-agent"
)
