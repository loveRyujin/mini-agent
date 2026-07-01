package transcript

import (
	"reflect"
	"testing"

	"github.com/loveRyujin/mini-agent/internal/agent"
	"github.com/loveRyujin/mini-agent/internal/inference"
)

func TestTranscript_conversationStream(t *testing.T) {
	tr := New()
	tr.AddUserMessage("hi")
	tr.Apply(agent.Event{Kind: agent.EventReasoningDelta, Text: "think"})
	tr.Apply(agent.Event{Kind: agent.EventAnswerDelta, Text: "Hello"})
	tr.Apply(agent.Event{Kind: agent.EventAnswerDelta, Text: " world"})
	tr.Apply(agent.Event{Kind: agent.EventTurnComplete})
	tr.Apply(agent.Event{Kind: agent.EventUsage, Usage: inference.Usage{CompletionToken: 2, PromptToken: 3, TotalToken: 5}})

	got := tr.EntryKinds()
	want := []EntryKind{EntryUser, EntryReasoning, EntryAnswer, EntryUsage}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("entry kinds:\n got: %v\nwant: %v", got, want)
	}
	if tr.EntryText(1) != "think" {
		t.Fatalf("reasoning text = %q", tr.EntryText(1))
	}
	if tr.EntryText(2) != "Hello world" {
		t.Fatalf("answer text = %q", tr.EntryText(2))
	}
}

func TestTranscript_toolCallOrder(t *testing.T) {
	tr := New()
	tr.AddUserMessage("read files")
	tr.Apply(agent.Event{Kind: agent.EventToolCall, ToolName: "read_file", ToolArguments: map[string]any{"path": "main.go"}})
	tr.Apply(agent.Event{Kind: agent.EventToolResult, ToolName: "read_file", ToolContent: `{"status":"SUCCESS","data":{"file_content":"package main"}}`})
	tr.Apply(agent.Event{Kind: agent.EventToolCall, ToolName: "list_file", ToolArguments: map[string]any{"path": "."}})
	tr.Apply(agent.Event{Kind: agent.EventToolResult, ToolName: "list_file", ToolContent: `{"status":"SUCCESS","data":{"files":["main.go"]}}`})
	tr.Apply(agent.Event{Kind: agent.EventAnswerDelta, Text: "done"})
	tr.Apply(agent.Event{Kind: agent.EventTurnComplete})

	got := tr.EntryKinds()
	want := []EntryKind{EntryUser, EntryToolCall, EntryToolResult, EntryToolCall, EntryToolResult, EntryAnswer}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("entry kinds:\n got: %v\nwant: %v", got, want)
	}
}

func TestTranscript_errorVisible(t *testing.T) {
	tr := New()
	tr.AddUserMessage("hi")
	tr.Apply(agent.Event{Kind: agent.EventError, Err: errTest})

	got := tr.EntryKinds()
	want := []EntryKind{EntryUser, EntryError}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("entry kinds:\n got: %v\nwant: %v", got, want)
	}
	if tr.EntryText(1) != "backend failed" {
		t.Fatalf("error text = %q", tr.EntryText(1))
	}
}

var errTest = &testError{msg: "backend failed"}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }
