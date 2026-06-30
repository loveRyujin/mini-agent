package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const maxToolRoundsPerTurn = 25

var errToolLoopLimit = errors.New("tool loop limit exceeded")

type InferenceBackend interface {
	CallLLMStream(ctx context.Context, req map[string]any) (SSEResp, error)
}

type Agent struct {
	Backend      InferenceBackend
	Model        string
	Tools        map[string]Tool
	History      []map[string]any
	ApprovalGate ApprovalGate
	systemPrompt string
}

func NewAgent(apiKey, url, model, systemPrompt string) *Agent {
	agent := &Agent{
		Backend: &Client{
			cli:    defaultHTTPClient(),
			apiKey: apiKey,
			url:    url,
			model:  model,
		},
		Model:        model,
		Tools:        make(map[string]Tool),
		systemPrompt: systemPrompt,
	}
	agent.RegisterTool(&readFile{}, &listFile{}, &writeFile{}, &workspaceSearch{}, &runShell{})
	agent.initHistory(systemPrompt)

	return agent
}

func defaultSystemPrompt() string {
	root := WorkspaceRoot()
	display := WorkspaceDisplay()
	return fmt.Sprintf(`You are a coding agent running in the user's terminal.

Your workspace root is %s (display: %s). All tool paths must be relative to this directory. Use list_file with path "." to explore the workspace. You cannot access files outside the workspace.

Read and inspect code with read_file and workspace_search. Create or update files with write_file (full-file overwrite). Run commands with run_shell (Shell Execution; requires Approval Gate). Be concise and practical.`, root, display)
}

func (a *Agent) initHistory(systemPrompt string) {
	a.History = []map[string]any{{
		"role":    "system",
		"content": systemPrompt,
	}}
}

func (a *Agent) ClearSession() {
	if a.systemPrompt == "" {
		a.systemPrompt = defaultSystemPrompt()
	}
	a.initHistory(a.systemPrompt)
}

func (a *Agent) RunTurn(ctx context.Context, userMessage string, emit EventEmitter) error {
	a.History = append(a.History, map[string]any{
		"role":    "user",
		"content": userMessage,
	})

	req := map[string]any{
		"model":    a.Model,
		"messages": a.History,
		"stream":   true,
		"stream_options": map[string]any{
			"include_usage": true,
		},
		"tools":       a.ToolDefinitions(),
		"tool_choice": "auto",
	}

	var (
		chunks      []string
		tokenUsage  []Usage
		lastToolSig string
	)

	for round := 0; ; round++ {
		if round >= maxToolRoundsPerTurn {
			emit(Event{Kind: EventError, Err: fmt.Errorf("%w (%d rounds)", errToolLoopLimit, maxToolRoundsPerTurn)})
			return errToolLoopLimit
		}

		req["messages"] = a.History

		ch, err := a.Backend.CallLLMStream(ctx, req)
		if err != nil {
			emit(Event{Kind: EventError, Err: err})
			return err
		}

		toolAcc := newToolCallAccumulator()
		var finishReason string

		for msg := range ch {
			if len(msg.Choices) == 0 {
				tokenUsage = append(tokenUsage, msg.Usage)
				continue
			}

			choice := msg.Choices[0]
			if choice.FinishReason != "" {
				finishReason = choice.FinishReason
			}

			delta := choice.Delta
			if delta.Reasoning != "" {
				emit(Event{Kind: EventReasoningDelta, Text: delta.Reasoning})
			}
			if len(delta.ToolCalls) > 0 {
				toolAcc.Add(delta.ToolCalls)
			}
			if delta.Content != "" {
				emit(Event{Kind: EventAnswerDelta, Text: delta.Content})
				chunks = append(chunks, delta.Content)
			}

			tokenUsage = append(tokenUsage, msg.Usage)
		}

		calls := toolAcc.Calls()
		if len(calls) == 0 {
			if finishReason == "tool_calls" {
				emit(Event{
					Kind: EventError,
					Err:  fmt.Errorf("model finished with tool_calls but no valid tool call was parsed"),
				})
				return fmt.Errorf("incomplete tool call stream")
			}
			break
		}

		sig := toolSignature(calls)
		if sig == lastToolSig {
			emit(Event{
				Kind: EventError,
				Err:  fmt.Errorf("model repeated identical tool calls; stopping to avoid loop"),
			})
			return fmt.Errorf("duplicate tool call loop")
		}
		lastToolSig = sig

		a.History = append(a.History, assistantToolCallsMessage(calls))
		resp, err := a.toolCall(ctx, calls, emit)
		if err != nil {
			emit(Event{Kind: EventError, Err: err})
			return err
		}
		a.History = append(a.History, resp...)
		chunks = nil
	}

	assistantMessage := strings.Join(chunks, "")
	if assistantMessage != "" {
		a.History = append(a.History, map[string]any{
			"role":    "assistant",
			"content": assistantMessage,
		})
	}

	emit(Event{Kind: EventTurnComplete, AssistantMessage: assistantMessage})

	if len(tokenUsage) > 0 {
		var cToken, pToken int
		for _, usage := range tokenUsage {
			cToken += int(usage.CompletionToken)
			pToken += int(usage.PromptToken)
		}
		emit(Event{
			Kind: EventUsage,
			Usage: Usage{
				CompletionToken: int64(cToken),
				PromptToken:     int64(pToken),
				TotalToken:      tokenUsage[len(tokenUsage)-1].TotalToken,
			},
		})
	}

	return nil
}

func (a *Agent) RegisterTool(tools ...Tool) {
	for _, tool := range tools {
		a.Tools[tool.Name()] = tool
	}
}

func (a *Agent) ToolDefinitions() []map[string]any {
	defs := make([]map[string]any, 0, len(a.Tools))
	for _, tool := range a.Tools {
		defs = append(defs, tool.Definition())
	}
	return defs
}

func (a *Agent) toolCall(ctx context.Context, toolCalls []ToolCall, emit EventEmitter) ([]map[string]any, error) {
	results := make([]map[string]any, 0, len(toolCalls))

	for _, toolCall := range toolCalls {
		emit(Event{
			Kind:          EventToolCall,
			ToolName:      toolCall.Function.Name,
			ToolArguments: toolCall.Function.Arguments,
		})

		tool, exist := a.Tools[toolCall.Function.Name]
		if !exist {
			resp := failResp(toolCall.ID, fmt.Errorf("unknown tool %q", toolCall.Function.Name))
			content, _ := resp["content"].(string)
			emit(Event{
				Kind:        EventToolResult,
				ToolName:    toolCall.Function.Name,
				ToolContent: content,
			})
			results = append(results, resp)
			continue
		}

		var resp map[string]any
		if gt, ok := tool.(GatedTool); ok {
			allowed, err := a.requestApproval(ctx, gt, toolCall, emit)
			if err != nil {
				return nil, err
			}
			if !allowed {
				resp = failResp(toolCall.ID, errShellDenied)
			} else {
				resp = tool.Call(ctx, toolCall)
			}
		} else {
			resp = tool.Call(ctx, toolCall)
		}

		content, _ := resp["content"].(string)
		emit(Event{
			Kind:        EventToolResult,
			ToolName:    toolCall.Function.Name,
			ToolContent: content,
		})

		results = append(results, resp)
	}

	return results, nil
}

func (a *Agent) requestApproval(ctx context.Context, gt GatedTool, toolCall ToolCall, emit EventEmitter) (bool, error) {
	req := ApprovalRequest{
		ToolCallID: toolCall.ID,
		ToolName:   toolCall.Function.Name,
		Summary:    gt.ApprovalSummary(toolCall),
	}

	if a.ApprovalGate != nil {
		return a.ApprovalGate.RequestApproval(ctx, req, emit)
	}

	ch := make(chan bool, 1)
	emit(Event{
		Kind:            EventApprovalRequired,
		Command:         req.Summary,
		ApprovalReplyCh: ch,
	})

	select {
	case allowed := <-ch:
		return allowed, nil
	case <-ctx.Done():
		return false, ctx.Err()
	}
}

