package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/loveRyujin/mini-agent/internal/inference"
)

func TestRunShell_executesInWorkspace(t *testing.T) {
	dir := t.TempDir()
	chdirWorkspace(t, dir)

	rs := &RunShell{}
	resp := rs.Call(context.Background(), inference.ToolCall{
		ID: "call-1",
		Function: inference.Function{
			Name:      "run_shell",
			Arguments: map[string]any{"command": "echo hello"},
		},
	})

	content, _ := resp["content"].(string)
	if !strings.Contains(content, "SUCCESS") {
		t.Fatalf("expected SUCCESS, got %q", content)
	}
	if !strings.Contains(content, "hello") {
		t.Fatalf("expected stdout hello, got %q", content)
	}
}

func TestRunShell_rejectsMissingCommand(t *testing.T) {
	dir := t.TempDir()
	chdirWorkspace(t, dir)

	rs := &RunShell{}
	resp := rs.Call(context.Background(), inference.ToolCall{
		ID: "call-1",
		Function: inference.Function{
			Name:      "run_shell",
			Arguments: map[string]any{},
		},
	})

	content, _ := resp["content"].(string)
	if !strings.Contains(content, "FAILED") {
		t.Fatalf("expected FAILED, got %q", content)
	}
}

func TestRunShell_runsInWorkspaceDir(t *testing.T) {
	dir := t.TempDir()
	chdirWorkspace(t, dir)

	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	rs := &RunShell{}
	resp := rs.Call(context.Background(), inference.ToolCall{
		ID: "call-1",
		Function: inference.Function{
			Name:      "run_shell",
			Arguments: map[string]any{"command": "pwd"},
		},
	})

	content, _ := resp["content"].(string)
	if !strings.Contains(content, dir) {
		t.Fatalf("expected workspace dir in output, got %q", content)
	}
}
