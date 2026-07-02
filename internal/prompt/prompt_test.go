package prompt

import (
	"os"
	"strings"
	"testing"

	"github.com/loveRyujin/mini-agent/internal/workspace"
)

func TestDefaultSystemPrompt_includesWorkspace(t *testing.T) {
	dir := t.TempDir()
	ws, err := workspace.New(dir)
	if err != nil {
		t.Fatal(err)
	}

	p := Default(ws)
	if !strings.Contains(p, dir) {
		t.Fatalf("prompt should include workspace root %q, got:\n%s", dir, p)
	}
	for _, tool := range []string{"list_file", "write_file", "workspace_search", "read_file", "run_shell"} {
		if !strings.Contains(p, tool) {
			t.Fatalf("prompt should mention %s, got:\n%s", tool, p)
		}
	}
}

func TestResolveSystemPrompt_default(t *testing.T) {
	t.Setenv(EnvSystemPrompt, "")
	t.Setenv(EnvSystemPromptFile, "")
	ws, err := workspace.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	p, err := Resolve(ws)
	if err != nil {
		t.Fatal(err)
	}
	if p != Default(ws) {
		t.Fatalf("got custom prompt, want default:\n%s", p)
	}
}

func TestResolveSystemPrompt_fromEnv(t *testing.T) {
	t.Setenv(EnvSystemPrompt, "  custom agent instructions  ")
	t.Setenv(EnvSystemPromptFile, "")

	ws, err := workspace.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	p, err := Resolve(ws)
	if err != nil {
		t.Fatal(err)
	}
	if p != "custom agent instructions" {
		t.Fatalf("prompt = %q, want trimmed env value", p)
	}
}

func TestResolveSystemPrompt_fromFile(t *testing.T) {
	t.Setenv(EnvSystemPrompt, "ignored when file is set")

	path := t.TempDir() + "/prompt.txt"
	if err := os.WriteFile(path, []byte("  file-based prompt\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvSystemPromptFile, path)

	ws, err := workspace.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	p, err := Resolve(ws)
	if err != nil {
		t.Fatal(err)
	}
	if p != "file-based prompt" {
		t.Fatalf("prompt = %q, want file contents", p)
	}
}

func TestResolveSystemPrompt_missingFile(t *testing.T) {
	t.Setenv(EnvSystemPrompt, "")
	t.Setenv(EnvSystemPromptFile, "/no/such/prompt.txt")

	ws, err := workspace.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	_, err = Resolve(ws)
	if err == nil {
		t.Fatal("expected error for missing prompt file")
	}
}
