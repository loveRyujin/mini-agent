package tools

import (
	"context"

	"github.com/loveRyujin/mini-agent/internal/inference"
	"github.com/loveRyujin/mini-agent/internal/workspace"
)

type Tool interface {
	Name() string
	Definition() map[string]any
	Call(context.Context, inference.ToolCall) map[string]any
}

type GatedTool interface {
	Tool
	ApprovalSummary(args inference.ToolCall) string
}

// Builtin returns all Built-in Tools shipped with the application.
func Builtin(ws *workspace.Workspace) []Tool {
	return []Tool{
		&ReadFile{WS: ws},
		&ListFile{WS: ws},
		&WriteFile{WS: ws},
		&WorkspaceSearch{WS: ws},
		&RunShell{WS: ws},
	}
}
