package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseSlashCommand(t *testing.T) {
	tests := []struct {
		input   string
		want    slashResult
		wantArg string
	}{
		{"hello", slashNone, ""},
		{"/quit", slashQuit, ""},
		{"  /quit  ", slashQuit, ""},
		{"/QUIT", slashQuit, ""},
		{"/clear", slashClear, ""},
		{"/help", slashHelp, ""},
		{"/unknown", slashUnknown, "unknown"},
		{"/foo bar", slashUnknown, "foo"},
		{"/", slashUnknown, ""},
	}

	for _, tt := range tests {
		got, arg := parseSlashCommand(tt.input)
		if got != tt.want || arg != tt.wantArg {
			t.Errorf("parseSlashCommand(%q) = (%v, %q), want (%v, %q)",
				tt.input, got, arg, tt.want, tt.wantArg)
		}
	}
}

func TestSlashHelpText(t *testing.T) {
	text := slashHelpText()
	for _, want := range []string{
		"/quit", "/clear", "/help", "Y", "N",
		"LLM_API_URL", "MINI_AGENT_SYSTEM_PROMPT",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("slashHelpText() missing %q:\n%s", want, text)
		}
	}
}

func TestAgentClearSession(t *testing.T) {
	agent := &Agent{
		Backend:      &scriptedBackend{},
		Model:        "test-model",
		Tools:        make(map[string]Tool),
		systemPrompt: "system prompt",
	}
	agent.initHistory("system prompt")

	agent.History = append(agent.History,
		map[string]any{"role": "user", "content": "hi"},
		map[string]any{"role": "assistant", "content": "hello"},
	)

	agent.ClearSession()

	if len(agent.History) != 1 {
		t.Fatalf("History len = %d, want 1", len(agent.History))
	}
	if agent.History[0]["role"] != "system" {
		t.Fatalf("History[0].role = %v, want system", agent.History[0]["role"])
	}
	if agent.History[0]["content"] != "system prompt" {
		t.Fatalf("History[0].content = %v, want system prompt", agent.History[0]["content"])
	}
}

func TestTranscriptReset(t *testing.T) {
	tr := NewTranscript()
	tr.AddUserMessage("hi")
	tr.Apply(Event{Kind: EventAnswerDelta, Text: "hello"})
	tr.Reset()

	if len(tr.Entries()) != 0 {
		t.Fatalf("entries len = %d, want 0", len(tr.Entries()))
	}
}

func TestTranscriptAddSystemMessage(t *testing.T) {
	tr := NewTranscript()
	tr.AddSystemMessage("help text")

	got := tr.EntryKinds()
	want := []transcriptEntryKind{entrySystem}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("entry kinds:\n got: %v\nwant: %v", got, want)
	}
}
