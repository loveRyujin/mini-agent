package main

import (
	"reflect"
	"testing"
)

func TestTranscript_conversationStream(t *testing.T) {
	tr := NewTranscript()
	tr.AddUserMessage("hi")
	tr.Apply(Event{Kind: EventReasoningDelta, Text: "think"})
	tr.Apply(Event{Kind: EventAnswerDelta, Text: "Hello"})
	tr.Apply(Event{Kind: EventAnswerDelta, Text: " world"})
	tr.Apply(Event{Kind: EventTurnComplete})
	tr.Apply(Event{Kind: EventUsage, Usage: Usage{CompletionToken: 2, PromptToken: 3, TotalToken: 5}})

	got := tr.EntryKinds()
	want := []transcriptEntryKind{
		entryUser,
		entryReasoning,
		entryAnswer,
		entryUsage,
	}
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
	tr := NewTranscript()
	tr.AddUserMessage("read files")
	tr.Apply(Event{
		Kind:          EventToolCall,
		ToolName:      "read_file",
		ToolArguments: map[string]any{"path": "main.go"},
	})
	tr.Apply(Event{
		Kind:        EventToolResult,
		ToolName:    "read_file",
		ToolContent: `{"status":"SUCCESS","data":{"file_content":"package main"}}`,
	})
	tr.Apply(Event{
		Kind:          EventToolCall,
		ToolName:      "list_file",
		ToolArguments: map[string]any{"path": "."},
	})
	tr.Apply(Event{
		Kind:        EventToolResult,
		ToolName:    "list_file",
		ToolContent: `{"status":"SUCCESS","data":{"files":["main.go"]}}`,
	})
	tr.Apply(Event{Kind: EventAnswerDelta, Text: "done"})
	tr.Apply(Event{Kind: EventTurnComplete})

	got := tr.EntryKinds()
	want := []transcriptEntryKind{
		entryUser,
		entryToolCall,
		entryToolResult,
		entryToolCall,
		entryToolResult,
		entryAnswer,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("entry kinds:\n got: %v\nwant: %v", got, want)
	}
}

func TestTranscript_errorVisible(t *testing.T) {
	tr := NewTranscript()
	tr.AddUserMessage("hi")
	tr.Apply(Event{Kind: EventError, Err: errTest})

	got := tr.EntryKinds()
	want := []transcriptEntryKind{entryUser, entryError}
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
