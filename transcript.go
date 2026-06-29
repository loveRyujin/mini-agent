package main

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
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
)

type transcriptEntry struct {
	kind transcriptEntryKind
	text string
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

func (t *Transcript) Apply(e Event) {
	switch e.Kind {
	case EventReasoningDelta:
		t.appendStreaming(entryReasoning, e.Text)

	case EventAnswerDelta:
		t.appendStreaming(entryAnswer, e.Text)

	case EventToolCall:
		t.endStreaming()
		t.entries = append(t.entries, transcriptEntry{
			kind: entryToolCall,
			text: fmt.Sprintf("%s(%v)", e.ToolName, e.ToolArguments),
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

func (t *Transcript) Render() string {
	var lines []string
	for _, e := range t.entries {
		switch e.kind {
		case entryUser:
			lines = append(lines, userStyle.Render("你: "+e.text))
		case entryReasoning:
			lines = append(lines, reasoningStyle.Render("推理: "+e.text))
		case entryAnswer:
			lines = append(lines, answerStyle.Render(e.text))
		case entryError:
			lines = append(lines, errorStyle.Render("错误: "+e.text))
		case entryUsage:
			lines = append(lines, usageStyle.Render(e.text))
		case entryToolCall:
			lines = append(lines, toolStyle.Render("工具调用: "+e.text))
		case entryToolResult:
			lines = append(lines, toolStyle.Render("工具结果: "+e.text))
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func formatUsage(u Usage) string {
	return fmt.Sprintf(
		"Token — 完成: %d, 提示: %d, 合计: %d",
		u.CompletionToken, u.PromptToken, u.TotalToken,
	)
}

var (
	userStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
	reasoningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
	answerStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	errorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	usageStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	toolStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
)
