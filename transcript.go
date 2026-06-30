package main

import (
	"fmt"
)

type transcriptEntryKind int

const (
	entryUser transcriptEntryKind = iota
	entryReasoning
	entryAnswer
	entryError
	entryUsage
	entryToolCall
	entryToolResult
	entryApproval
	entrySystem
)

type transcriptEntry struct {
	kind     transcriptEntryKind
	text     string
	meta     string // 工具参数摘要，如 path
	toolName string
}

const noStreaming transcriptEntryKind = -1

type Transcript struct {
	entries   []transcriptEntry
	streaming transcriptEntryKind
}

func NewTranscript() *Transcript {
	return &Transcript{streaming: noStreaming}
}

func (t *Transcript) AddUserMessage(text string) {
	t.entries = append(t.entries, transcriptEntry{kind: entryUser, text: text})
}

func (t *Transcript) AddSystemMessage(text string) {
	t.endStreaming()
	t.entries = append(t.entries, transcriptEntry{kind: entrySystem, text: text})
}

func (t *Transcript) Reset() {
	t.entries = nil
	t.streaming = noStreaming
}

func (t *Transcript) Apply(e Event) {
	switch e.Kind {
	case EventReasoningDelta:
		t.appendStreaming(entryReasoning, e.Text)

	case EventAnswerDelta:
		t.appendStreaming(entryAnswer, e.Text)

	case EventToolCall:
		t.endStreaming()
		t.entries = append(t.entries, transcriptEntry{
			kind:     entryToolCall,
			text:     fmt.Sprintf("%s(%v)", e.ToolName, e.ToolArguments),
			toolName: e.ToolName,
			meta:     formatToolMeta(e.ToolName, e.ToolArguments),
		})

	case EventApprovalRequired:
		t.endStreaming()
		t.entries = append(t.entries, transcriptEntry{
			kind: entryApproval,
			text: e.Command,
		})

	case EventToolResult:
		t.endStreaming()
		t.entries = append(t.entries, transcriptEntry{
			kind: entryToolResult,
			text: e.ToolContent,
		})

	case EventTurnComplete:
		t.endStreaming()

	case EventUsage:
		t.endStreaming()
		t.entries = append(t.entries, transcriptEntry{
			kind: entryUsage,
			text: formatUsage(e.Usage),
		})

	case EventError:
		t.endStreaming()
		msg := "unknown error"
		if e.Err != nil {
			msg = e.Err.Error()
		}
		t.entries = append(t.entries, transcriptEntry{kind: entryError, text: msg})
	}
}

func formatToolMeta(name string, args map[string]any) string {
	if command, ok := args["command"].(string); ok && command != "" {
		return command
	}
	if path, ok := args["path"].(string); ok && path != "" {
		return path
	}
	if pattern, ok := args["pattern"].(string); ok && pattern != "" {
		return pattern
	}
	return ""
}

func (t *Transcript) appendStreaming(kind transcriptEntryKind, text string) {
	if t.streaming == kind {
		last := &t.entries[len(t.entries)-1]
		last.text += text
		return
	}
	t.endStreaming()
	t.entries = append(t.entries, transcriptEntry{kind: kind, text: text})
	t.streaming = kind
}

func (t *Transcript) endStreaming() {
	t.streaming = noStreaming
}

func (t *Transcript) Entries() []transcriptEntry {
	return t.entries
}

func (t *Transcript) EntryKinds() []transcriptEntryKind {
	kinds := make([]transcriptEntryKind, len(t.entries))
	for i, e := range t.entries {
		kinds[i] = e.kind
	}
	return kinds
}

func (t *Transcript) EntryText(i int) string {
	return t.entries[i].text
}

func (t *Transcript) Render(opts TranscriptRenderOpts) string {
	if opts.Expanded == nil {
		opts.Expanded = make(map[int]bool)
	}
	if opts.Theme.name == "" {
		opts.Theme = defaultTUITheme
	}
	return renderTranscriptCrush(t.entries, opts)
}

func formatUsage(u Usage) string {
	return fmt.Sprintf(
		"Token — 完成: %d, 提示: %d, 合计: %d",
		u.CompletionToken, u.PromptToken, u.TotalToken,
	)
}
