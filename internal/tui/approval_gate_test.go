package tui

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/loveRyujin/mini-agent/internal/agent"
)

func TestApprovalGate_respondAllows(t *testing.T) {
	gate := NewApprovalGate()
	req := agent.ApprovalRequest{Summary: "echo hello"}

	done := make(chan struct{})
	go func() {
		defer close(done)
		allowed, err := gate.RequestApproval(context.Background(), req, func(e agent.Event) {
			if e.Kind != agent.EventApprovalRequired {
				t.Errorf("event kind = %v, want EventApprovalRequired", e.Kind)
			}
			if e.Command != req.Summary {
				t.Errorf("command = %q, want %q", e.Command, req.Summary)
			}
		})
		if err != nil {
			t.Errorf("RequestApproval: %v", err)
		}
		if !allowed {
			t.Error("allowed = false, want true")
		}
	}()

	waitPending(t, gate)
	gate.Respond(true)
	<-done

	if gate.IsPending() {
		t.Error("IsPending = true after response")
	}
}

func TestApprovalGate_respondDenies(t *testing.T) {
	gate := NewApprovalGate()
	req := agent.ApprovalRequest{Summary: "rm -rf /"}

	done := make(chan struct{})
	go func() {
		defer close(done)
		allowed, err := gate.RequestApproval(context.Background(), req, func(agent.Event) {})
		if err != nil {
			t.Errorf("RequestApproval: %v", err)
		}
		if allowed {
			t.Error("allowed = true, want false")
		}
	}()

	waitPending(t, gate)
	gate.Respond(false)
	<-done
}

func TestApprovalGate_contextCancel(t *testing.T) {
	gate := NewApprovalGate()
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		_, err := gate.RequestApproval(ctx, agent.ApprovalRequest{Summary: "sleep"}, func(agent.Event) {})
		done <- err
	}()

	waitPending(t, gate)
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("RequestApproval did not return after ctx cancel")
	}

	if gate.IsPending() {
		t.Error("IsPending = true after ctx cancel")
	}
}

func waitPending(t *testing.T, gate *ApprovalGate) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if gate.IsPending() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("gate never became pending")
}
