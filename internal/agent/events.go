package agent

import "github.com/loveRyujin/mini-agent/internal/inference"

type EventKind int

const (
	EventReasoningDelta EventKind = iota
	EventAnswerDelta
	EventToolCall
	EventApprovalRequired
	EventToolResult
	EventTurnComplete
	EventUsage
	EventError
)

type Event struct {
	Kind EventKind

	Text             string
	Command          string
	ToolName         string
	ToolArguments    map[string]any
	ToolContent      string
	AssistantMessage string
	Usage            inference.Usage
	Err              error
	ApprovalReplyCh  chan<- bool
}

type EventEmitter func(Event)
