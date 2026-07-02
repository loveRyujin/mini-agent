package tui

import (
	"context"
	"errors"
	"sync"

	"github.com/loveRyujin/mini-agent/internal/agent"
)

type ApprovalGate struct {
	mu      sync.Mutex
	replyCh chan bool
}

func NewApprovalGate() *ApprovalGate {
	return &ApprovalGate{}
}

func (g *ApprovalGate) RequestApproval(ctx context.Context, req agent.ApprovalRequest, emit agent.EventEmitter) (bool, error) {
	g.mu.Lock()
	if g.replyCh != nil {
		g.mu.Unlock()
		return false, errors.New("approval already pending")
	}
	ch := make(chan bool, 1)
	g.replyCh = ch
	g.mu.Unlock()

	emit(agent.Event{
		Kind:    agent.EventApprovalRequired,
		Command: req.Summary,
	})

	select {
	case allowed := <-ch:
		g.mu.Lock()
		g.replyCh = nil
		g.mu.Unlock()
		return allowed, nil
	case <-ctx.Done():
		g.mu.Lock()
		g.replyCh = nil
		g.mu.Unlock()
		return false, ctx.Err()
	}
}

func (g *ApprovalGate) Respond(allowed bool) {
	g.mu.Lock()
	ch := g.replyCh
	g.mu.Unlock()
	if ch == nil {
		return
	}
	select {
	case ch <- allowed:
	default:
	}
}

func (g *ApprovalGate) IsPending() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.replyCh != nil
}
