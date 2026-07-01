package tui

import (
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/loveRyujin/mini-agent/internal/transcript"
)

func (m *model) toggleTranscriptFocus() {
	m.transcriptFocus = !m.transcriptFocus
	if m.transcriptFocus {
		m.textarea.Blur()
		if m.focusIdx < 0 {
			m.focusIdx = transcript.FirstFocusable(m.transcript.Entries())
		}
	} else {
		m.textarea.Focus()
	}
}

func (m *model) toggleFocusedExpand() {
	entries := m.transcript.Entries()
	if !transcript.IsFocusable(entries, m.focusIdx) {
		return
	}
	m.expanded[m.focusIdx] = !m.expanded[m.focusIdx]
}

func (m *model) moveFocus(delta int) {
	entries := m.transcript.Entries()
	if len(entries) == 0 {
		m.focusIdx = -1
		return
	}
	start := m.focusIdx
	if start < 0 {
		start = 0
	}
	for step := 1; step <= len(entries); step++ {
		idx := start + delta*step
		for idx < 0 {
			idx += len(entries)
		}
		idx %= len(entries)
		if transcript.IsFocusable(entries, idx) {
			m.focusIdx = idx
			return
		}
	}
}

func (m *model) toggleExpandKind(kind transcript.EntryKind) {
	switch kind {
	case transcript.EntryReasoning:
		expand := !m.allExpandedKind(transcript.EntryReasoning)
		for i, e := range m.transcript.Entries() {
			if e.Kind == transcript.EntryReasoning {
				m.expanded[i] = expand
			}
		}
	case transcript.EntryToolCall, transcript.EntryToolResult:
		expand := !m.allToolBlocksExpanded()
		entries := m.transcript.Entries()
		for i := range entries {
			if transcript.HasPairedToolResult(entries, i) {
				m.expanded[i] = expand
			}
		}
	}
}

func (m *model) allExpandedKind(kind transcript.EntryKind) bool {
	any := false
	for i, e := range m.transcript.Entries() {
		if e.Kind != kind {
			continue
		}
		any = true
		if !m.expanded[i] {
			return false
		}
	}
	return any
}

func (m *model) allToolBlocksExpanded() bool {
	entries := m.transcript.Entries()
	any := false
	for i := range entries {
		if !transcript.HasPairedToolResult(entries, i) {
			continue
		}
		any = true
		if !m.expanded[i] {
			return false
		}
	}
	return any
}

func (m *model) setAllExpanded(expanded bool) {
	for i, e := range m.transcript.Entries() {
		if e.Kind == transcript.EntryReasoning {
			m.expanded[i] = expanded
		}
		if transcript.HasPairedToolResult(m.transcript.Entries(), i) {
			m.expanded[i] = expanded
		}
	}
}

func (m *model) handleTranscriptKeys(msg tea.KeyMsg) bool {
	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlT:
		m.transcriptFocus = false
		m.textarea.Focus()
		return true
	case tea.KeyEnter, tea.KeySpace:
		m.toggleFocusedExpand()
		return true
	case tea.KeyUp:
		m.moveFocus(-1)
		return true
	case tea.KeyDown:
		m.moveFocus(1)
		return true
	case tea.KeyPgUp, tea.KeyPgDown:
		m.scrollViewport(msg)
		return true
	case tea.KeyCtrlY:
		m.copyFocusedEntry()
		return true
	}
	if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 {
		switch msg.Runes[0] {
		case 'j':
			m.moveFocus(1)
			return true
		case 'k':
			m.moveFocus(-1)
			return true
		case 'e', 'E':
			m.toggleFocusedExpand()
			return true
		case 'G':
			m.viewport.GotoBottom()
			m.followTail = true
			return true
		}
	}
	return false
}

func (m *model) copyFocusedEntry() {
	text := m.focusedEntryPlainText()
	if text == "" {
		return
	}
	_ = clipboard.WriteAll(text)
}

func (m *model) focusedEntryPlainText() string {
	entries := m.transcript.Entries()
	if m.focusIdx < 0 || m.focusIdx >= len(entries) {
		return ""
	}
	e := entries[m.focusIdx]
	switch e.Kind {
	case transcript.EntryToolCall:
		if transcript.HasPairedToolResult(entries, m.focusIdx) {
			return entries[m.focusIdx+1].Text
		}
		return strings.TrimSpace(e.Meta + "\n" + e.Text)
	default:
		return e.Text
	}
}
