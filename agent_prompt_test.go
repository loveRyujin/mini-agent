package main

import (
	"strings"
	"testing"
)

func TestDefaultSystemPrompt_includesWorkspace(t *testing.T) {
	workspaceDir = t.TempDir()

	prompt := defaultSystemPrompt()
	if !strings.Contains(prompt, workspaceDir) {
		t.Fatalf("prompt should include workspace root %q, got:\n%s", workspaceDir, prompt)
	}
	if !strings.Contains(prompt, "list_file") {
		t.Fatalf("prompt should mention list_file, got:\n%s", prompt)
	}
}
