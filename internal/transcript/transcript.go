package transcript

import (
	"fmt"

	"github.com/loveRyujin/mini-agent/internal/agent"
	"github.com/loveRyujin/mini-agent/internal/inference"
)

type EntryKind int

const (
	EntryUser EntryKind = iota
	EntryReasoning
	EntryAnswer
	EntryError
	EntryUsage
	EntryToolCall
	EntryToolResult
	EntryApproval
	EntrySystem
)

const noStreaming EntryKind = -1

type Entry struct {
	Kind     EntryKind
	Text     string
	Meta     string
	ToolName string
}

type Transcript struct {
	entries   []Entry
	streaming EntryKind
}

func New() *Transcript {
	return &Transcript{streaming: noStreaming}
}

func (t *Transcript) AddUserMessage(text string) {
	t.entries = append(t.entries, Entry{Kind: EntryUser, Text: text})
}

func (t *Transcript) AddSystemMessage(text string) {
	t.endStreaming()
	t.entries = append(t.entries, Entry{Kind: EntrySystem, Text: text})
}

func (t *Transcript) Reset() {
	t.entries = nil
	t.streaming = noStreaming
}

func (t *Transcript) Apply(e agent.Event) {
	switch e.Kind {
	case agent.EventReasoningDelta:
		t.appendStreaming(EntryReasoning, e.Text)
	case agent.EventAnswerDelta:
		t.appendStreaming(EntryAnswer, e.Text)
	case agent.EventToolCall:
		t.endStreaming()
		t.entries = append(t.entries, Entry{
			Kind:     EntryToolCall,
			Text:     fmt.Sprintf("%s(%v)", e.ToolName, e.ToolArguments),
			ToolName: e.ToolName,
			Meta:     formatToolMeta(e.ToolName, e.ToolArguments),
		})
	case agent.EventApprovalRequired:
		t.endStreaming()
		t.entries = append(t.entries, Entry{Kind: EntryApproval, Text: e.Command})
	case agent.EventToolResult:
		t.endStreaming()
		t.entries = append(t.entries, Entry{Kind: EntryToolResult, Text: e.ToolContent})
	case agent.EventTurnComplete:
		t.endStreaming()
	case agent.EventUsage:
		t.endStreaming()
		t.entries = append(t.entries, Entry{Kind: EntryUsage, Text: formatUsage(e.Usage)})
	case agent.EventError:
		t.endStreaming()
		msg := "unknown error"
		if e.Err != nil {
			msg = e.Err.Error()
		}
		t.entries = append(t.entries, Entry{Kind: EntryError, Text: msg})
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

func (t *Transcript) appendStreaming(kind EntryKind, text string) {
	if t.streaming == kind {
		last := &t.entries[len(t.entries)-1]
		last.Text += text
		return
	}
	t.endStreaming()
	t.entries = append(t.entries, Entry{Kind: kind, Text: text})
	t.streaming = kind
}

func (t *Transcript) endStreaming() {
	t.streaming = noStreaming
}

func (t *Transcript) Entries() []Entry {
	return t.entries
}

func (t *Transcript) EntryKinds() []EntryKind {
	kinds := make([]EntryKind, len(t.entries))
	for i, e := range t.entries {
		kinds[i] = e.Kind
	}
	return kinds
}

func (t *Transcript) EntryText(i int) string {
	return t.entries[i].Text
}

type RenderOpts struct {
	Theme           Theme
	Expanded        map[int]bool
	FocusIdx        int
	TranscriptFocus bool
}

func (t *Transcript) Render(opts RenderOpts) string {
	if opts.Expanded == nil {
		opts.Expanded = make(map[int]bool)
	}
	if opts.Theme.Name == "" {
		opts.Theme = DefaultTheme
	}
	return renderCrush(t.entries, opts)
}

func formatUsage(u inference.Usage) string {
	return fmt.Sprintf(
		"Token — 完成: %d, 提示: %d, 合计: %d",
		u.CompletionToken, u.PromptToken, u.TotalToken,
	)
}

func IsFocusable(entries []Entry, idx int) bool {
	if idx < 0 || idx >= len(entries) {
		return false
	}
	switch entries[idx].Kind {
	case EntryReasoning, EntryToolCall:
		return true
	default:
		return false
	}
}

func IsPairedToolResult(entries []Entry, idx int) bool {
	return entries[idx].Kind == EntryToolResult &&
		idx > 0 && entries[idx-1].Kind == EntryToolCall
}

func HasPairedToolResult(entries []Entry, callIdx int) bool {
	return entries[callIdx].Kind == EntryToolCall &&
		callIdx+1 < len(entries) && entries[callIdx+1].Kind == EntryToolResult
}

func FirstFocusable(entries []Entry) int {
	for i := range entries {
		if IsFocusable(entries, i) {
			return i
		}
	}
	return 0
}
