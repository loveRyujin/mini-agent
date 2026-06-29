package main

type EventKind int

const (
	EventReasoningDelta EventKind = iota
	EventAnswerDelta
	EventToolCall
	EventToolResult
	EventTurnComplete
	EventUsage
	EventError
)

type Event struct {
	Kind EventKind

	Text              string
	ToolName          string
	ToolArguments     map[string]any
	ToolContent       string
	AssistantMessage  string
	Usage             Usage
	Err               error
}

type EventEmitter func(Event)
