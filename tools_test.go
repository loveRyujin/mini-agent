package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func chdirWorkspace(t *testing.T, dir string) {
	t.Helper()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	workspaceDir = dir
}

func TestReadFile_withinWorkspace(t *testing.T) {
	dir := t.TempDir()
	chdirWorkspace(t, dir)

	path := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	rf := &readFile{}
	resp := rf.Call(context.Background(), ToolCall{
		ID: "call-1",
		Function: Function{
			Name:      "read_file",
			Arguments: map[string]any{"path": "hello.txt"},
		},
	})

	content, _ := resp["content"].(string)
	if !strings.Contains(content, "hello") {
		t.Fatalf("content = %q, want file contents", content)
	}
}

func TestReadFile_rejectsEscape(t *testing.T) {
	dir := t.TempDir()
	chdirWorkspace(t, dir)

	rf := &readFile{}
	resp := rf.Call(context.Background(), ToolCall{
		ID: "call-1",
		Function: Function{
			Name:      "read_file",
			Arguments: map[string]any{"path": "../outside.txt"},
		},
	})

	content, _ := resp["content"].(string)
	if !strings.Contains(content, "FAILED") {
		t.Fatalf("expected FAILED status, got %q", content)
	}
	if !strings.Contains(content, "path escapes workspace") {
		t.Fatalf("expected escape error, got %q", content)
	}
}

func TestListFile_rejectsEscape(t *testing.T) {
	dir := t.TempDir()
	chdirWorkspace(t, dir)

	lf := &listFile{}
	resp := lf.Call(context.Background(), ToolCall{
		ID: "call-1",
		Function: Function{
			Name:      "list_file",
			Arguments: map[string]any{"path": "../outside"},
		},
	})

	content, _ := resp["content"].(string)
	if !strings.Contains(content, "FAILED") || !strings.Contains(content, "path escapes workspace") {
		t.Fatalf("expected escape error, got %q", content)
	}
}

func TestListFile_withinWorkspace(t *testing.T) {
	dir := t.TempDir()
	chdirWorkspace(t, dir)

	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}

	lf := &listFile{}
	resp := lf.Call(context.Background(), ToolCall{
		ID: "call-1",
		Function: Function{
			Name:      "list_file",
			Arguments: map[string]any{"path": "."},
		},
	})

	content, _ := resp["content"].(string)
	if !strings.Contains(content, "a.txt") {
		t.Fatalf("content = %q, want a.txt listed", content)
	}
}
