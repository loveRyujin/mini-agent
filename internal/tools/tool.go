package tools

import (
	"context"

	"github.com/loveRyujin/mini-agent/internal/inference"
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
func Builtin() []Tool {
	return []Tool{
		&ReadFile{},
		&ListFile{},
		&WriteFile{},
		&WorkspaceSearch{},
		&RunShell{},
	}
}
