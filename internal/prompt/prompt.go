package prompt

import (
	"fmt"
	"os"
	"strings"

	"github.com/loveRyujin/mini-agent/internal/tools"
)

const (
	EnvSystemPrompt     = "MINI_AGENT_SYSTEM_PROMPT"
	EnvSystemPromptFile = "MINI_AGENT_SYSTEM_PROMPT_FILE"
)

func Resolve() (string, error) {
	if path := os.Getenv(EnvSystemPromptFile); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read system prompt file %q: %w", path, err)
		}
		return strings.TrimSpace(string(data)), nil
	}
	if p := os.Getenv(EnvSystemPrompt); p != "" {
		return strings.TrimSpace(p), nil
	}
	return Default(), nil
}

func Default() string {
	root := tools.WorkspaceRoot()
	display := tools.WorkspaceDisplay()
	return fmt.Sprintf(`You are a coding agent running in the user's terminal.

Your workspace root is %s (display: %s). All tool paths must be relative to this directory. Use list_file with path "." to explore the workspace. You cannot access files outside the workspace.

Read and inspect code with read_file and workspace_search. Create or update files with write_file (full-file overwrite). Run commands with run_shell (Shell Execution; requires Approval Gate). Be concise and practical.`, root, display)
}
