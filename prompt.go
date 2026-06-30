package main

import (
	"fmt"
	"os"
	"strings"
)

const (
	envSystemPrompt     = "MINI_AGENT_SYSTEM_PROMPT"
	envSystemPromptFile = "MINI_AGENT_SYSTEM_PROMPT_FILE"
)

func resolveSystemPrompt() (string, error) {
	if path := os.Getenv(envSystemPromptFile); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read system prompt file %q: %w", path, err)
		}
		return strings.TrimSpace(string(data)), nil
	}

	if prompt := os.Getenv(envSystemPrompt); prompt != "" {
		return strings.TrimSpace(prompt), nil
	}

	return defaultSystemPrompt(), nil
}
