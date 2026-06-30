package main

import "strings"

type slashResult int

const (
	slashNone slashResult = iota
	slashQuit
	slashClear
	slashHelp
	slashUnknown
)

func parseSlashCommand(text string) (slashResult, string) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "/") {
		return slashNone, ""
	}

	fields := strings.Fields(text)
	if len(fields) == 0 {
		return slashUnknown, ""
	}

	cmd := strings.ToLower(strings.TrimPrefix(fields[0], "/"))
	switch cmd {
	case "quit":
		return slashQuit, ""
	case "clear":
		return slashClear, ""
	case "help":
		return slashHelp, ""
	default:
		return slashUnknown, cmd
	}
}

func slashHelpText() string {
	return strings.TrimSpace(`
Slash Command：
  /quit   退出 TUI
  /clear  清空 Session 与 Transcript
  /help   显示此帮助

Approval Gate 快捷键：
  Y  允许执行 Shell 命令
  N  拒绝执行
`)
}
