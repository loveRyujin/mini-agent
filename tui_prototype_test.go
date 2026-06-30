package main

import (
	"strings"
	"testing"
)

func TestProtoPlaceholderRendersCleanly(t *testing.T) {
	for i := range protoVariants {
		m := newProtoModel()
		if m.textarea.Placeholder != "" {
			t.Fatal("textarea.Placeholder must stay empty to avoid CJK rendering bug")
		}
		m.ready = true
		m.width = 80
		m.height = 24
		m.variantIdx = i
		m.layout()
		body := renderProtoView(i, protoRenderCtx{
			width: 80, viewport: m.viewport, textarea: m.textarea,
			theme: protoThemes[0], transcriptMode: 0, expanded: m.expanded, focusIdx: -1, transcriptFocus: false,
		})
		if !strings.Contains(body, protoInputHintText) {
			t.Errorf("variant %s: external hint %q not found", protoVariants[i].key, protoInputHintText)
		}
	}
}

func TestProtoViewFitsTerminalHeight(t *testing.T) {
	sizes := [][2]int{{80, 24}, {120, 40}, {60, 20}}
	for _, sz := range sizes {
		w, h := sz[0], sz[1]
		for i := range protoVariants {
			m := newProtoModel()
			m.ready = true
			m.width = w
			m.height = h
			m.variantIdx = i
			m.layout()
			view := m.View()
			lines := strings.Count(view, "\n") + 1
			if lines > h {
				t.Errorf("variant %s (%s) at %dx%d: rendered %d lines, want <= %d",
					protoVariants[i].key, protoVariants[i].name, w, h, lines, h)
			}
		}
	}
}
