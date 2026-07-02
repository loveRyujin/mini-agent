package main

import (
	"cmp"
	"fmt"
	"os"

	"github.com/loveRyujin/mini-agent/internal/agent"
	"github.com/loveRyujin/mini-agent/internal/prompt"
	"github.com/loveRyujin/mini-agent/internal/tui"
	"github.com/loveRyujin/mini-agent/internal/workspace"
)

const (
	defaultURL   = "http://localhost:11434/v1/chat/completions"
	defaultModel = "deepseek-r1:latest"
)

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	ws, err := workspace.NewFromCWD()
	if err != nil {
		return fmt.Errorf("init workspace: %w", err)
	}

	apiKey := os.Getenv("LLM_API_KEY")
	url := cmp.Or(os.Getenv("LLM_API_URL"), defaultURL)
	model := cmp.Or(os.Getenv("LLM_MODEL"), defaultModel)

	systemPrompt, err := prompt.Resolve(ws)
	if err != nil {
		return fmt.Errorf("system prompt: %w", err)
	}

	a := agent.NewAgent(apiKey, url, model, systemPrompt, ws)
	gate := tui.NewApprovalGate()
	a.ApprovalGate = gate
	return tui.Run(a, gate, ws.Display())
}
