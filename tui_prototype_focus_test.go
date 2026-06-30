package main

import "testing"

func TestMoveFocusSkipsToolResult(t *testing.T) {
	entries := prototypeTranscript()
	m := newProtoModel()
	m.entries = entries
	m.focusIdx = firstFocusable(entries) // 推理块

	for i, e := range entries {
		if e.kind == protoToolCall {
			m.focusIdx = i - 1
			m.moveFocus(1)
			if m.focusIdx != i {
				t.Fatalf("after tool call prev, moveFocus(1) want call idx %d, got %d (kind=%v)", i, m.focusIdx, entries[m.focusIdx].kind)
			}
			if m.focusIdx+1 < len(entries) && entries[m.focusIdx+1].kind == protoToolResult {
				// ok - focus on call not result
			}
			m.moveFocus(1)
			if entries[m.focusIdx].kind == protoToolResult {
				t.Fatalf("moveFocus landed on tool result idx %d, want tool call or reasoning", m.focusIdx)
			}
			return
		}
	}
	t.Fatal("mock transcript missing tool call")
}

func TestToolExpandOnCallIndex(t *testing.T) {
	entries := prototypeTranscript()
	m := newProtoModel()
	var callIdx int
	for i, e := range entries {
		if e.kind == protoToolCall {
			callIdx = i
			break
		}
	}
	m.focusIdx = callIdx
	m.toggleFocusedExpand()
	if !m.expanded[callIdx] {
		t.Fatal("expand state should be stored on tool call index")
	}
}
