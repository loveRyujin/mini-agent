package main

import (
	"fmt"
	"os"
)

const (
	// use ollama api, reference: https://docs.ollama.com/api/openai-compatibility
	defaultUrl   = "http://localhost:11434/v1/chat/completions"
	defaultModel = "gpt-oss:latest"
)

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	apiKey := os.Getenv("LLM_API_KEY")
	url := os.Getenv("LLM_API_URL")
	if url == "" {
		url = defaultUrl
	}
	model := os.Getenv("LLM_MODEL")
	if model == "" {
		model = defaultModel
	}

	agent := NewAgent(apiKey, url, model)
	if err := agent.Run(); err != nil {
		return err
	}

	return nil
}
