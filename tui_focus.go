package main

import (
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *tuiModel) toggleTranscriptFocus() {
	m.transcriptFocus = !m.transcriptFocus
	if m.transcriptFocus {
		m.textarea.Blur()
		if m.focusIdx < 0 {
			m.focusIdx = firstTranscriptFocusable(m.transcript.Entries())
		}
	} else {
		m.textarea.Focus()
	}
}

func (m *tuiModel) toggleFocusedExpand() {
	entries := m.transcript.Entries()
	if !isTranscriptFocusable(entries, m.focusIdx) {
		return
	}
	m.expanded[m.focusIdx] = !m.expanded[m.focusIdx]
}

func (m *tuiModel) moveFocus(delta int) {
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
		if isTranscriptFocusable(entries, idx) {
			m.focusIdx = idx
			return
		}
	}
}

func (m *tuiModel) toggleExpandKind(kind transcriptEntryKind) {
	switch kind {
	case entryReasoning:
		expand := !m.allExpandedKind(entryReasoning)
		for i, e := range m.transcript.Entries() {
			if e.kind == entryReasoning {
				m.expanded[i] = expand
			}
		}
	case entryToolCall, entryToolResult:
		expand := !m.allToolBlocksExpanded()
		entries := m.transcript.Entries()
		for i := range entries {
			if hasPairedToolResultEntry(entries, i) {
				m.expanded[i] = expand
			}
		}
	}
}

func (m *tuiModel) allExpandedKind(kind transcriptEntryKind) bool {
	any := false
	for i, e := range m.transcript.Entries() {
		if e.kind != kind {
			continue
		}
		any = true
		if !m.expanded[i] {
			return false
		}
	}
	return any
}

func (m *tuiModel) allToolBlocksExpanded() bool {
	entries := m.transcript.Entries()
	any := false
	for i := range entries {
		if !hasPairedToolResultEntry(entries, i) {
			continue
		}
		any = true
		if !m.expanded[i] {
			return false
		}
	}
	return any
}

func (m *tuiModel) setAllExpanded(expanded bool) {
	for i, e := range m.transcript.Entries() {
		if e.kind == entryReasoning {
			m.expanded[i] = expanded
		}
		if hasPairedToolResultEntry(m.transcript.Entries(), i) {
			m.expanded[i] = expanded
		}
	}
}

func (m *tuiModel) handleTranscriptKeys(msg tea.KeyMsg) bool {
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

func (m *tuiModel) copyFocusedEntry() {
	text := m.focusedEntryPlainText()
	if text == "" {
		return
	}
	_ = clipboard.WriteAll(text)
}

func (m *tuiModel) focusedEntryPlainText() string {
	entries := m.transcript.Entries()
	if m.focusIdx < 0 || m.focusIdx >= len(entries) {
		return ""
	}
	e := entries[m.focusIdx]
	switch e.kind {
	case entryToolCall:
		if hasPairedToolResultEntry(entries, m.focusIdx) {
			return entries[m.focusIdx+1].text
		}
		return strings.TrimSpace(e.meta + "\n" + e.text)
	default:
		return e.text
	}
}
