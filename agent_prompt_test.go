package main

import (
	"os"
	"strings"
	"testing"
)

func TestDefaultSystemPrompt_includesWorkspace(t *testing.T) {
	workspaceDir = t.TempDir()

	prompt := defaultSystemPrompt()
	if !strings.Contains(prompt, workspaceDir) {
		t.Fatalf("prompt should include workspace root %q, got:\n%s", workspaceDir, prompt)
	}
	for _, tool := range []string{"list_file", "write_file", "workspace_search", "read_file", "run_shell"} {
		if !strings.Contains(prompt, tool) {
			t.Fatalf("prompt should mention %s, got:\n%s", tool, prompt)
		}
	}
}

func TestResolveSystemPrompt_default(t *testing.T) {
	t.Setenv(envSystemPrompt, "")
	t.Setenv(envSystemPromptFile, "")
	workspaceDir = t.TempDir()

	prompt, err := resolveSystemPrompt()
	if err != nil {
		t.Fatal(err)
	}
	if prompt != defaultSystemPrompt() {
		t.Fatalf("got custom prompt, want default:\n%s", prompt)
	}
}

func TestResolveSystemPrompt_fromEnv(t *testing.T) {
	t.Setenv(envSystemPrompt, "  custom agent instructions  ")
	t.Setenv(envSystemPromptFile, "")

	prompt, err := resolveSystemPrompt()
	if err != nil {
		t.Fatal(err)
	}
	if prompt != "custom agent instructions" {
		t.Fatalf("prompt = %q, want trimmed env value", prompt)
	}
}

func TestResolveSystemPrompt_fromFile(t *testing.T) {
	t.Setenv(envSystemPrompt, "ignored when file is set")

	path := t.TempDir() + "/prompt.txt"
	if err := os.WriteFile(path, []byte("  file-based prompt\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(envSystemPromptFile, path)

	prompt, err := resolveSystemPrompt()
	if err != nil {
		t.Fatal(err)
	}
	if prompt != "file-based prompt" {
		t.Fatalf("prompt = %q, want file contents", prompt)
	}
}

func TestResolveSystemPrompt_missingFile(t *testing.T) {
	t.Setenv(envSystemPrompt, "")
	t.Setenv(envSystemPromptFile, "/no/such/prompt.txt")

	_, err := resolveSystemPrompt()
	if err == nil {
		t.Fatal("expected error for missing prompt file")
	}
}

func TestNewAgent_builtinTools(t *testing.T) {
	agent := NewAgent("", "", "test", "system")

	want := []string{"read_file", "list_file", "write_file", "workspace_search", "run_shell"}
	for _, name := range want {
		if _, ok := agent.Tools[name]; !ok {
			t.Fatalf("missing built-in tool %q", name)
		}
	}
	if len(agent.Tools) != len(want) {
		t.Fatalf("Tools = %d, want exactly %d coding tools", len(agent.Tools), len(want))
	}
}

func TestAgentClearSession_preservesConfiguredPrompt(t *testing.T) {
	agent := &Agent{
		Backend:      &scriptedBackend{},
		Model:        "test-model",
		Tools:        make(map[string]Tool),
		systemPrompt: "configured prompt",
	}
	agent.initHistory(agent.systemPrompt)
	agent.History = append(agent.History,
		map[string]any{"role": "user", "content": "hi"},
		map[string]any{"role": "assistant", "content": "hello"},
	)

	agent.ClearSession()

	if len(agent.History) != 1 {
		t.Fatalf("History len = %d, want 1", len(agent.History))
	}
	if agent.History[0]["content"] != "configured prompt" {
		t.Fatalf("History[0].content = %v, want configured prompt", agent.History[0]["content"])
	}
}
