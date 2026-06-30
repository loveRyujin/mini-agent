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

func TestWriteFile_createsFile(t *testing.T) {
	dir := t.TempDir()
	chdirWorkspace(t, dir)

	wf := &writeFile{}
	resp := wf.Call(context.Background(), ToolCall{
		ID: "call-1",
		Function: Function{
			Name: "write_file",
			Arguments: map[string]any{
				"path":    "out.txt",
				"content": "hello world",
			},
		},
	})

	content, _ := resp["content"].(string)
	if !strings.Contains(content, "SUCCESS") {
		t.Fatalf("expected SUCCESS, got %q", content)
	}

	got, err := os.ReadFile(filepath.Join(dir, "out.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello world" {
		t.Fatalf("file content = %q, want %q", got, "hello world")
	}
}

func TestWriteFile_overwritesExisting(t *testing.T) {
	dir := t.TempDir()
	chdirWorkspace(t, dir)

	path := filepath.Join(dir, "out.txt")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	wf := &writeFile{}
	resp := wf.Call(context.Background(), ToolCall{
		ID: "call-1",
		Function: Function{
			Name: "write_file",
			Arguments: map[string]any{
				"path":    "out.txt",
				"content": "new",
			},
		},
	})

	content, _ := resp["content"].(string)
	if !strings.Contains(content, "SUCCESS") {
		t.Fatalf("expected SUCCESS, got %q", content)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Fatalf("file content = %q, want %q", got, "new")
	}
}

func TestWriteFile_rejectsEscape(t *testing.T) {
	dir := t.TempDir()
	chdirWorkspace(t, dir)

	wf := &writeFile{}
	resp := wf.Call(context.Background(), ToolCall{
		ID: "call-1",
		Function: Function{
			Name: "write_file",
			Arguments: map[string]any{
				"path":    "../outside.txt",
				"content": "nope",
			},
		},
	})

	content, _ := resp["content"].(string)
	if !strings.Contains(content, "FAILED") || !strings.Contains(content, "path escapes workspace") {
		t.Fatalf("expected escape error, got %q", content)
	}
}

func TestWorkspaceSearch_contentMatch(t *testing.T) {
	dir := t.TempDir()
	chdirWorkspace(t, dir)

	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc foo() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ws := &workspaceSearch{}
	resp := ws.Call(context.Background(), ToolCall{
		ID: "call-1",
		Function: Function{
			Name: "workspace_search",
			Arguments: map[string]any{
				"pattern": "func foo",
				"path":    ".",
			},
		},
	})

	content, _ := resp["content"].(string)
	if !strings.Contains(content, "SUCCESS") {
		t.Fatalf("expected SUCCESS, got %q", content)
	}
	if !strings.Contains(content, "main.go") {
		t.Fatalf("expected main.go in results, got %q", content)
	}
}

func TestWorkspaceSearch_filenameMatch(t *testing.T) {
	dir := t.TempDir()
	chdirWorkspace(t, dir)

	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	ws := &workspaceSearch{}
	resp := ws.Call(context.Background(), ToolCall{
		ID: "call-1",
		Function: Function{
			Name: "workspace_search",
			Arguments: map[string]any{
				"pattern": "*.go",
				"path":    ".",
				"mode":    "filename",
			},
		},
	})

	content, _ := resp["content"].(string)
	if !strings.Contains(content, "SUCCESS") {
		t.Fatalf("expected SUCCESS, got %q", content)
	}
	if !strings.Contains(content, "main.go") {
		t.Fatalf("expected main.go in results, got %q", content)
	}
	if strings.Contains(content, "readme.txt") {
		t.Fatalf("did not expect readme.txt in filename results, got %q", content)
	}
}

func TestWorkspaceSearch_rejectsEscape(t *testing.T) {
	dir := t.TempDir()
	chdirWorkspace(t, dir)

	ws := &workspaceSearch{}
	resp := ws.Call(context.Background(), ToolCall{
		ID: "call-1",
		Function: Function{
			Name: "workspace_search",
			Arguments: map[string]any{
				"pattern": "foo",
				"path":    "../outside",
			},
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
