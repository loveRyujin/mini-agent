package agent

import "context"

type ApprovalRequest struct {
	ToolCallID string
	ToolName   string
	Summary    string
}

type ApprovalGate interface {
	RequestApproval(ctx context.Context, req ApprovalRequest, emit EventEmitter) (allowed bool, err error)
}

type staticApprovalGate struct {
	allowed bool
}

func NewStaticApprovalGate(allowed bool) ApprovalGate {
	return &staticApprovalGate{allowed: allowed}
}

func (g *staticApprovalGate) RequestApproval(_ context.Context, req ApprovalRequest, emit EventEmitter) (bool, error) {
	emit(Event{
		Kind:    EventApprovalRequired,
		Command: req.Summary,
	})
	return g.allowed, nil
}
