package main

import (
	"cmp"
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
	apiKey := cmp.Or(os.Getenv("LLM_API_KEY"), defaultUrl)
	url := cmp.Or(os.Getenv("LLM_API_URL"), defaultUrl)
	model := cmp.Or(os.Getenv("LLM_MODEL"), defaultModel)

	agent := NewAgent(apiKey, url, model)
	if err := agent.Run(); err != nil {
		return err
	}

	return nil
}
