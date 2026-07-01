package tui

import (
	"fmt"
	"testing"

	"github.com/loveRyujin/mini-agent/internal/agent"
	"github.com/loveRyujin/mini-agent/internal/tools"
)

func TestSyncViewport_preservesScrollWhenNotFollowingTail(t *testing.T) {
	a := &agent.Agent{Model: "test-model", Tools: make(map[string]tools.Tool)}
	m := newModel(a)
	m.ready = true
	m.width = 80
	m.height = 24
	m.layout()

	for i := range 80 {
		m.transcript.AddUserMessage(fmt.Sprintf("line %d", i))
	}
	m.syncViewport()
	if !m.viewport.AtBottom() {
		t.Fatal("expected initial sync to follow tail")
	}

	m.followTail = false
	m.viewport.SetYOffset(0)
	if m.viewport.AtBottom() {
		t.Fatal("expected manual scroll away from bottom")
	}

	m.transcript.AddUserMessage("new message")
	m.syncViewport()

	if m.viewport.AtBottom() {
		t.Fatal("syncViewport jumped back to bottom while followTail is false")
	}
	if m.viewport.YOffset != 0 {
		t.Fatalf("YOffset = %d, want 0", m.viewport.YOffset)
	}
}
