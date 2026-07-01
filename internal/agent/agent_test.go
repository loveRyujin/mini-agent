package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/loveRyujin/mini-agent/internal/inference"
	"github.com/loveRyujin/mini-agent/internal/tools"
)


type scriptedBackend struct {
	scripts [][]inference.Response
	calls   int
}

func (s *scriptedBackend) CallLLMStream(_ context.Context, _ map[string]any) (inference.SSEResp, error) {
	if s.calls >= len(s.scripts) {
		ch := make(inference.SSEResp)
		close(ch)
		return ch, nil
	}

	script := s.scripts[s.calls]
	s.calls++

	ch := make(inference.SSEResp, len(script))
	go func() {
		for _, resp := range script {
			ch <- resp
		}
		close(ch)
	}()

	return ch, nil
}


func chdirWorkspace(t *testing.T, dir string) {
	t.Helper()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	tools.SetWorkspaceRootForTest(dir)
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
		scripts: [][]inference.Response{
			{
				{Choices: []inference.Choice{{Delta: inference.Delta{Reasoning: "think"}}}},
				{Choices: []inference.Choice{{Delta: inference.Delta{Content: "Hello"}}}},
				{Choices: []inference.Choice{{Delta: inference.Delta{Content: " world"}}}},
				{Usage: inference.Usage{CompletionToken: 2, PromptToken: 3, TotalToken: 5}},
			},
		},
	}

	agent := &Agent{
		Backend: backend,
		Model:   "test-model",
		Tools:   make(map[string]tools.Tool),
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
	tools.SetWorkspaceRootForTest(dir)

	backend := &scriptedBackend{
		scripts: [][]inference.Response{
			{
				{Choices: []inference.Choice{{Delta: inference.Delta{
					ToolCalls: []inference.ToolCall{{
						ID:   "call-1",
						Type: "function",
						Function: inference.Function{
							Name:      "read_file",
							Arguments: map[string]any{"path": "hello.txt"},
						},
					}},
				}}}},
			},
			{
				{Choices: []inference.Choice{{Delta: inference.Delta{Content: "done"}}}},
			},
		},
	}

	agent := &Agent{
		Backend: backend,
		Model:   "test-model",
		Tools:   make(map[string]tools.Tool),
	}
	agent.RegisterTool(&tools.ReadFile{})
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
	tools.SetWorkspaceRootForTest(dir)

	backend := &scriptedBackend{
		scripts: [][]inference.Response{
			{
				{Choices: []inference.Choice{{Delta: inference.Delta{
					ToolCalls: []inference.ToolCall{{
						ID:   "call-1",
						Type: "function",
						Function: inference.Function{
							Name:      "read_file",
							Arguments: map[string]any{"path": "hello.txt"},
						},
					}},
				}}}},
			},
			{
				{Choices: []inference.Choice{{Delta: inference.Delta{
					ToolCalls: []inference.ToolCall{{
						ID:   "call-2",
						Type: "function",
						Function: inference.Function{
							Name:      "list_file",
							Arguments: map[string]any{"path": "."},
						},
					}},
				}}}},
			},
			{
				{Choices: []inference.Choice{{Delta: inference.Delta{Content: "done"}}}},
			},
		},
	}

	agent := &Agent{
		Backend: backend,
		Model:   "test-model",
		Tools:   make(map[string]tools.Tool),
	}
	agent.RegisterTool(&tools.ReadFile{}, &tools.ListFile{})
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
	tools.SetWorkspaceRootForTest(dir)

	backend := &scriptedBackend{
		scripts: [][]inference.Response{
			{
				{Choices: []inference.Choice{{Delta: inference.Delta{
					ToolCalls: []inference.ToolCall{{
						ID:   "call-1",
						Type: "function",
						Function: inference.Function{
							Name:      "read_file",
							Arguments: map[string]any{"path": "../outside.txt"},
						},
					}},
				}}}},
			},
			{
				{Choices: []inference.Choice{{Delta: inference.Delta{Content: "cannot read"}}}},
			},
		},
	}

	agent := &Agent{
		Backend: backend,
		Model:   "test-model",
		Tools:   make(map[string]tools.Tool),
	}
	agent.RegisterTool(&tools.ReadFile{})
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

func TestRunTurn_shellAllowed(t *testing.T) {
	dir := t.TempDir()
	chdirWorkspace(t, dir)

	backend := &scriptedBackend{
		scripts: [][]inference.Response{
			{
				{Choices: []inference.Choice{{Delta: inference.Delta{
					ToolCalls: []inference.ToolCall{{
						ID:   "call-1",
						Type: "function",
						Function: inference.Function{
							Name:      "run_shell",
							Arguments: map[string]any{"command": "echo allowed"},
						},
					}},
				}}}},
			},
			{
				{Choices: []inference.Choice{{Delta: inference.Delta{Content: "done"}}}},
			},
		},
	}

	agent := &Agent{
		Backend:      backend,
		Model:        "test-model",
		Tools:        make(map[string]tools.Tool),
		ApprovalGate: NewStaticApprovalGate(true),
	}
	agent.RegisterTool(&tools.RunShell{})
	agent.initHistory("system prompt")

	emit, events := collectEmitter()
	if err := agent.RunTurn(context.Background(), "run echo", emit); err != nil {
		t.Fatalf("RunTurn: %v", err)
	}

	got := eventKinds(events())
	want := []EventKind{
		EventToolCall,
		EventApprovalRequired,
		EventToolResult,
		EventAnswerDelta,
		EventTurnComplete,
		EventUsage,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("event kinds:\n got: %v\nwant: %v", got, want)
	}

	for _, e := range events() {
		if e.Kind == EventToolResult {
			if !strings.Contains(e.ToolContent, "allowed") {
				t.Fatalf("tool result should contain command output: %q", e.ToolContent)
			}
		}
	}
}

func TestRunTurn_shellDenied(t *testing.T) {
	dir := t.TempDir()
	chdirWorkspace(t, dir)

	backend := &scriptedBackend{
		scripts: [][]inference.Response{
			{
				{Choices: []inference.Choice{{Delta: inference.Delta{
					ToolCalls: []inference.ToolCall{{
						ID:   "call-1",
						Type: "function",
						Function: inference.Function{
							Name:      "run_shell",
							Arguments: map[string]any{"command": "echo denied"},
						},
					}},
				}}}},
			},
			{
				{Choices: []inference.Choice{{Delta: inference.Delta{Content: "ok"}}}},
			},
		},
	}

	agent := &Agent{
		Backend:      backend,
		Model:        "test-model",
		Tools:        make(map[string]tools.Tool),
		ApprovalGate: NewStaticApprovalGate(false),
	}
	agent.RegisterTool(&tools.RunShell{})
	agent.initHistory("system prompt")

	emit, events := collectEmitter()
	if err := agent.RunTurn(context.Background(), "run echo", emit); err != nil {
		t.Fatalf("RunTurn: %v", err)
	}

	got := eventKinds(events())
	want := []EventKind{
		EventToolCall,
		EventApprovalRequired,
		EventToolResult,
		EventAnswerDelta,
		EventTurnComplete,
		EventUsage,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("event kinds:\n got: %v\nwant: %v", got, want)
	}

	for _, e := range events() {
		if e.Kind == EventToolResult {
			if !strings.Contains(e.ToolContent, "FAILED") {
				t.Fatalf("expected FAILED tool result, got %q", e.ToolContent)
			}
			if !strings.Contains(e.ToolContent, "denied") {
				t.Fatalf("expected denial message, got %q", e.ToolContent)
			}
		}
	}
}

func TestRunTurn_toolLoopLimit(t *testing.T) {
	dir := t.TempDir()
	chdirWorkspace(t, dir)

	scripts := make([][]inference.Response, maxToolRoundsPerTurn+1)
	for i := range scripts {
		scripts[i] = []inference.Response{
			{Choices: []inference.Choice{{Delta: inference.Delta{
				ToolCalls: []inference.ToolCall{{
					Index: 0,
					ID:    fmt.Sprintf("call-loop-%d", i),
					Type:  "function",
					Function: inference.Function{
						Name:      "list_file",
						Arguments: map[string]any{"path": fmt.Sprintf("dir%d", i)},
					},
				}},
			}}}},
		}
	}

	backend := &scriptedBackend{scripts: scripts}
	agent := &Agent{
		Backend: backend,
		Model:   "test-model",
		Tools:   make(map[string]tools.Tool),
	}
	agent.RegisterTool(&tools.ListFile{})
	agent.initHistory("system prompt")

	emit, events := collectEmitter()
	err := agent.RunTurn(context.Background(), "loop forever", emit)
	if !errors.Is(err, errToolLoopLimit) {
		t.Fatalf("RunTurn err = %v, want %v", err, errToolLoopLimit)
	}

	var foundError bool
	for _, e := range events() {
		if e.Kind == EventError && errors.Is(e.Err, errToolLoopLimit) {
			foundError = true
		}
	}
	if !foundError {
		t.Fatal("expected EventError for tool loop limit")
	}
	if backend.calls != maxToolRoundsPerTurn {
		t.Fatalf("backend calls = %d, want %d", backend.calls, maxToolRoundsPerTurn)
	}
}

func TestRunTurn_streamingToolCallDeduplicated(t *testing.T) {
	dir := t.TempDir()
	chdirWorkspace(t, dir)

	backend := &scriptedBackend{
		scripts: [][]inference.Response{
			{
				{Choices: []inference.Choice{{Delta: inference.Delta{
					ToolCalls: []inference.ToolCall{{
						Index: 0,
						ID:    "call-1",
						Type:  "function",
						Function: inference.Function{
							Name:      "list_file",
							Arguments: map[string]any{"path": "."},
						},
					}},
				}}}},
				{Choices: []inference.Choice{{Delta: inference.Delta{
					ToolCalls: []inference.ToolCall{{
						Index: 0,
						Function: inference.Function{
							Name:      "list_file",
							Arguments: map[string]any{"path": "."},
						},
					}},
				}}}},
			},
			{
				{Choices: []inference.Choice{{Delta: inference.Delta{Content: "done"}}}},
			},
		},
	}

	agent := &Agent{
		Backend: backend,
		Model:   "test-model",
		Tools:   make(map[string]tools.Tool),
	}
	agent.RegisterTool(&tools.ListFile{})
	agent.initHistory("system prompt")

	emit, events := collectEmitter()
	if err := agent.RunTurn(context.Background(), "list files", emit); err != nil {
		t.Fatalf("RunTurn: %v", err)
	}

	toolCallEvents := 0
	for _, e := range events() {
		if e.Kind == EventToolCall {
			toolCallEvents++
		}
	}
	if toolCallEvents != 1 {
		t.Fatalf("tool call events = %d, want 1", toolCallEvents)
	}
}

func TestRunTurn_toolHistoryIncludesAssistantToolCalls(t *testing.T) {
	dir := t.TempDir()
	chdirWorkspace(t, dir)

	backend := &scriptedBackend{
		scripts: [][]inference.Response{
			{
				{Choices: []inference.Choice{{Delta: inference.Delta{
					ToolCalls: []inference.ToolCall{{
						Index: 0,
						ID:    "call-1",
						Type:  "function",
						Function: inference.Function{
							Name:      "list_file",
							Arguments: map[string]any{"path": "."},
						},
					}},
				}}}},
			},
			{
				{Choices: []inference.Choice{{Delta: inference.Delta{Content: "done"}}}},
			},
		},
	}

	agent := &Agent{
		Backend: backend,
		Model:   "test-model",
		Tools:   make(map[string]tools.Tool),
	}
	agent.RegisterTool(&tools.ListFile{})
	agent.initHistory("system prompt")

	emit, _ := collectEmitter()
	if err := agent.RunTurn(context.Background(), "list files", emit); err != nil {
		t.Fatalf("RunTurn: %v", err)
	}

	var assistantIdx, toolIdx = -1, -1
	for i, msg := range agent.History {
		if msg["role"] == "assistant" && msg["tool_calls"] != nil {
			assistantIdx = i
		}
		if msg["role"] == "tool" {
			toolIdx = i
		}
	}
	if assistantIdx < 0 {
		t.Fatal("expected assistant message with tool_calls in history")
	}
	if toolIdx < 0 {
		t.Fatal("expected tool result in history")
	}
	if assistantIdx > toolIdx {
		t.Fatalf("assistant tool_calls (idx %d) should precede tool result (idx %d)", assistantIdx, toolIdx)
	}
}

func TestRunTurn_unknownToolStillRecordsResult(t *testing.T) {
	backend := &scriptedBackend{
		scripts: [][]inference.Response{
			{
				{Choices: []inference.Choice{{Delta: inference.Delta{
					ToolCalls: []inference.ToolCall{{
						Index: 0,
						ID:    "call-1",
						Type:  "function",
						Function: inference.Function{
							Name:      "missing_tool",
							Arguments: map[string]any{},
						},
					}},
				}}}},
			},
			{
				{Choices: []inference.Choice{{Delta: inference.Delta{Content: "ok"}}}},
			},
		},
	}

	agent := &Agent{
		Backend: backend,
		Model:   "test-model",
		Tools:   make(map[string]tools.Tool),
	}
	agent.initHistory("system prompt")

	emit, _ := collectEmitter()
	if err := agent.RunTurn(context.Background(), "call missing", emit); err != nil {
		t.Fatalf("RunTurn: %v", err)
	}

	var toolResults int
	for _, msg := range agent.History {
		if msg["role"] == "tool" {
			toolResults++
		}
	}
	if toolResults != 1 {
		t.Fatalf("tool results in history = %d, want 1", toolResults)
	}
}

func TestRunTurn_duplicateToolCallsStop(t *testing.T) {
	dir := t.TempDir()
	chdirWorkspace(t, dir)

	alwaysTool := []inference.Response{
		{Choices: []inference.Choice{{Delta: inference.Delta{
			ToolCalls: []inference.ToolCall{{
				Index: 0,
				ID:    "call-1",
				Type:  "function",
				Function: inference.Function{
					Name:      "list_file",
					Arguments: map[string]any{"path": "."},
				},
			}},
		}}}},
	}

	backend := &scriptedBackend{
		scripts: [][]inference.Response{alwaysTool, alwaysTool},
	}
	agent := &Agent{
		Backend: backend,
		Model:   "test-model",
		Tools:   make(map[string]tools.Tool),
	}
	agent.RegisterTool(&tools.ListFile{})
	agent.initHistory("system prompt")

	emit, _ := collectEmitter()
	err := agent.RunTurn(context.Background(), "loop", emit)
	if err == nil {
		t.Fatal("expected duplicate tool loop error")
	}
	if backend.calls != 2 {
		t.Fatalf("backend calls = %d, want 2", backend.calls)
	}
}

func TestRunTurn_reasoningAndToolCallsSameDelta(t *testing.T) {
	dir := t.TempDir()
	chdirWorkspace(t, dir)

	backend := &scriptedBackend{
		scripts: [][]inference.Response{
			{
				{Choices: []inference.Choice{{Delta: inference.Delta{
					Reasoning: "think",
					ToolCalls: []inference.ToolCall{{
						Index: 0,
						ID:    "call-1",
						Type:  "function",
						Function: inference.Function{
							Name:      "list_file",
							Arguments: map[string]any{"path": "."},
						},
					}},
				}}}},
			},
			{
				{Choices: []inference.Choice{{Delta: inference.Delta{Content: "done"}}}},
			},
		},
	}

	agent := &Agent{
		Backend: backend,
		Model:   "test-model",
		Tools:   make(map[string]tools.Tool),
	}
	agent.RegisterTool(&tools.ListFile{})
	agent.initHistory("system prompt")

	emit, events := collectEmitter()
	if err := agent.RunTurn(context.Background(), "list", emit); err != nil {
		t.Fatalf("RunTurn: %v", err)
	}

	got := eventKinds(events())
	want := []EventKind{
		EventReasoningDelta,
		EventToolCall,
		EventToolResult,
		EventAnswerDelta,
		EventTurnComplete,
		EventUsage,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("event kinds:\n got: %v\nwant: %v", got, want)
	}
}

func eventKinds(events []Event) []EventKind {
	kinds := make([]EventKind, len(events))
	for i, e := range events {
		kinds[i] = e.Kind
	}
	return kinds
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
		Tools:        make(map[string]tools.Tool),
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
