package main

import (
	"cmp"
	"fmt"
	"os"
)

const (
	// use ollama api, reference: https://docs.ollama.com/api/openai-compatibility
	defaultUrl   = "http://localhost:11434/v1/chat/completions"
	defaultModel = "deepseek-r1:latest"
)

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	if err := initWorkspace(); err != nil {
		return fmt.Errorf("init workspace: %w", err)
	}

	apiKey := os.Getenv("LLM_API_KEY")
	url := cmp.Or(os.Getenv("LLM_API_URL"), defaultUrl)
	model := cmp.Or(os.Getenv("LLM_MODEL"), defaultModel)

	systemPrompt, err := resolveSystemPrompt()
	if err != nil {
		return fmt.Errorf("system prompt: %w", err)
	}

	agent := NewAgent(apiKey, url, model, systemPrompt)
	return runTUI(agent)
}
