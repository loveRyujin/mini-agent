package main

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type scriptedBackend struct {
	scripts [][]Response
	calls   int
}

func (s *scriptedBackend) CallLLMStream(_ context.Context, _ map[string]any) (SSEResp, error) {
	if s.calls >= len(s.scripts) {
		ch := make(SSEResp)
		close(ch)
		return ch, nil
	}

	script := s.scripts[s.calls]
	s.calls++

	ch := make(SSEResp, len(script))
	go func() {
		for _, resp := range script {
			ch <- resp
		}
		close(ch)
	}()

	return ch, nil
}

func collectEmitter() (EventEmitter, func() []Event) {
	var events []Event
	emit := func(e Event) {
		events = append(events, e)
	}
	return emit, func() []Event { return events }
}

func TestRunTurn_conversationOnly(t *testing.T) {
	backend := &scriptedBackend{
		scripts: [][]Response{
			{
				{Choices: []Choice{{Delta: Delta{Reasoning: "think"}}}},
				{Choices: []Choice{{Delta: Delta{Content: "Hello"}}}},
				{Choices: []Choice{{Delta: Delta{Content: " world"}}}},
				{Usage: Usage{CompletionToken: 2, PromptToken: 3, TotalToken: 5}},
			},
		},
	}

	agent := &Agent{
		Backend: backend,
		Model:   "test-model",
		Tools:   make(map[string]Tool),
	}
	agent.initHistory("system prompt")

	emit, events := collectEmitter()
	if err := agent.RunTurn(context.Background(), "hi", emit); err != nil {
		t.Fatalf("RunTurn: %v", err)
	}

	got := eventKinds(events())
	want := []EventKind{
		EventReasoningDelta,
		EventAnswerDelta,
		EventAnswerDelta,
		EventTurnComplete,
		EventUsage,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("event kinds:\n got: %v\nwant: %v", got, want)
	}

	last := agent.History[len(agent.History)-1]
	if last["role"] != "assistant" || last["content"] != "Hello world" {
		t.Fatalf("assistant history: %#v", last)
	}
}

func TestRunTurn_toolLoop(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	workspaceDir = dir

	backend := &scriptedBackend{
		scripts: [][]Response{
			{
				{Choices: []Choice{{Delta: Delta{
					ToolCalls: []ToolCall{{
						ID:   "call-1",
						Type: "function",
						Function: Function{
							Name:      "read_file",
							Arguments: map[string]any{"path": "hello.txt"},
						},
					}},
				}}}},
			},
			{
				{Choices: []Choice{{Delta: Delta{Content: "done"}}}},
			},
		},
	}

	agent := &Agent{
		Backend: backend,
		Model:   "test-model",
		Tools:   make(map[string]Tool),
	}
	agent.RegisterTool(&readFile{})
	agent.initHistory("system prompt")

	emit, events := collectEmitter()
	if err := agent.RunTurn(context.Background(), "read hello.txt", emit); err != nil {
		t.Fatalf("RunTurn: %v", err)
	}

	got := eventKinds(events())
	want := []EventKind{
		EventToolCall,
		EventToolResult,
		EventAnswerDelta,
		EventTurnComplete,
		EventUsage,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("event kinds:\n got: %v\nwant: %v", got, want)
	}

	if backend.calls != 2 {
		t.Fatalf("backend calls = %d, want 2", backend.calls)
	}
}

func TestRunTurn_multiToolLoop(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	workspaceDir = dir

	backend := &scriptedBackend{
		scripts: [][]Response{
			{
				{Choices: []Choice{{Delta: Delta{
					ToolCalls: []ToolCall{{
						ID:   "call-1",
						Type: "function",
						Function: Function{
							Name:      "read_file",
							Arguments: map[string]any{"path": "hello.txt"},
						},
					}},
				}}}},
			},
			{
				{Choices: []Choice{{Delta: Delta{
					ToolCalls: []ToolCall{{
						ID:   "call-2",
						Type: "function",
						Function: Function{
							Name:      "list_file",
							Arguments: map[string]any{"path": "."},
						},
					}},
				}}}},
			},
			{
				{Choices: []Choice{{Delta: Delta{Content: "done"}}}},
			},
		},
	}

	agent := &Agent{
		Backend: backend,
		Model:   "test-model",
		Tools:   make(map[string]Tool),
	}
	agent.RegisterTool(&readFile{}, &listFile{})
	agent.initHistory("system prompt")

	emit, events := collectEmitter()
	if err := agent.RunTurn(context.Background(), "inspect project", emit); err != nil {
		t.Fatalf("RunTurn: %v", err)
	}

	got := eventKinds(events())
	want := []EventKind{
		EventToolCall,
		EventToolResult,
		EventToolCall,
		EventToolResult,
		EventAnswerDelta,
		EventTurnComplete,
		EventUsage,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("event kinds:\n got: %v\nwant: %v", got, want)
	}

	if backend.calls != 3 {
		t.Fatalf("backend calls = %d, want 3", backend.calls)
	}
}

func TestRunTurn_toolPathEscape(t *testing.T) {
	dir := t.TempDir()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	workspaceDir = dir

	backend := &scriptedBackend{
		scripts: [][]Response{
			{
				{Choices: []Choice{{Delta: Delta{
					ToolCalls: []ToolCall{{
						ID:   "call-1",
						Type: "function",
						Function: Function{
							Name:      "read_file",
							Arguments: map[string]any{"path": "../outside.txt"},
						},
					}},
				}}}},
			},
			{
				{Choices: []Choice{{Delta: Delta{Content: "cannot read"}}}},
			},
		},
	}

	agent := &Agent{
		Backend: backend,
		Model:   "test-model",
		Tools:   make(map[string]Tool),
	}
	agent.RegisterTool(&readFile{})
	agent.initHistory("system prompt")

	emit, events := collectEmitter()
	if err := agent.RunTurn(context.Background(), "read outside", emit); err != nil {
		t.Fatalf("RunTurn: %v", err)
	}

	got := events()
	var foundResult bool
	for _, e := range got {
		if e.Kind == EventToolResult {
			foundResult = true
			if !strings.Contains(e.ToolContent, "path escapes workspace") {
				t.Fatalf("tool result should mention escape: %q", e.ToolContent)
			}
		}
	}
	if !foundResult {
		t.Fatal("expected EventToolResult for path escape")
	}
}

func eventKinds(events []Event) []EventKind {
	kinds := make([]EventKind, len(events))
	for i, e := range events {
		kinds[i] = e.Kind
	}
	return kinds
}
