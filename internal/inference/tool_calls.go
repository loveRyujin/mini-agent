package inference

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type ToolCallAccumulator struct {
	byIndex map[int64]*ToolCall
	nextIdx int64
}

func NewToolCallAccumulator() *ToolCallAccumulator {
	return &ToolCallAccumulator{byIndex: make(map[int64]*ToolCall)}
}

func (a *ToolCallAccumulator) Add(delta []ToolCall) {
	for _, tc := range delta {
		idx := tc.Index
		if existing, ok := a.byIndex[idx]; ok && tc.ID != "" && existing.ID != "" && tc.ID != existing.ID {
			idx = a.nextIdx
			a.nextIdx++
		}

		existing, ok := a.byIndex[idx]
		if !ok {
			copy := tc
			a.byIndex[idx] = &copy
			if idx >= a.nextIdx {
				a.nextIdx = idx + 1
			}
			continue
		}
		if tc.ID != "" {
			existing.ID = tc.ID
		}
		if tc.Type != "" {
			existing.Type = tc.Type
		}
		if tc.Function.Name != "" {
			existing.Function.Name = tc.Function.Name
		}
		if len(tc.Function.Arguments) > 0 {
			if existing.Function.Arguments == nil {
				existing.Function.Arguments = make(map[string]any)
			}
			for k, v := range tc.Function.Arguments {
				existing.Function.Arguments[k] = v
			}
		}
		if tc.Function.ArgsRaw != "" {
			existing.Function.ArgsRaw += tc.Function.ArgsRaw
		}
	}
}

func (a *ToolCallAccumulator) Calls() []ToolCall {
	if len(a.byIndex) == 0 {
		return nil
	}
	indices := make([]int64, 0, len(a.byIndex))
	for idx := range a.byIndex {
		indices = append(indices, idx)
	}
	sort.Slice(indices, func(i, j int) bool { return indices[i] < indices[j] })

	calls := make([]ToolCall, 0, len(indices))
	for _, idx := range indices {
		tc := *a.byIndex[idx]
		if args := tc.Function.ParsedArguments(); args != nil {
			tc.Function.Arguments = args
		}
		calls = append(calls, tc)
	}
	return ValidToolCalls(calls)
}

func ValidToolCalls(calls []ToolCall) []ToolCall {
	valid := make([]ToolCall, 0, len(calls))
	for _, tc := range calls {
		name := strings.TrimSpace(tc.Function.Name)
		if tc.ID == "" || name == "" {
			continue
		}
		if tc.Function.ParsedArguments() == nil && tc.Function.ArgsRaw != "" {
			continue
		}
		tc.Function.Name = name
		if tc.Type == "" {
			tc.Type = "function"
		}
		valid = append(valid, tc)
	}
	return valid
}

func AssistantToolCallsMessage(calls []ToolCall) map[string]any {
	toolCalls := make([]map[string]any, len(calls))
	for i, tc := range calls {
		args := tc.Function.ParsedArguments()
		if args == nil {
			args = map[string]any{}
		}
		argsJSON, _ := json.Marshal(args)
		toolCalls[i] = map[string]any{
			"id":   tc.ID,
			"type": tc.Type,
			"function": map[string]any{
				"name":      tc.Function.Name,
				"arguments": string(argsJSON),
			},
		}
	}
	return map[string]any{
		"role":       "assistant",
		"content":    nil,
		"tool_calls": toolCalls,
	}
}

func ToolSignature(calls []ToolCall) string {
	if len(calls) == 0 {
		return ""
	}
	parts := make([]string, len(calls))
	for i, tc := range calls {
		args, _ := json.Marshal(tc.Function.ParsedArguments())
		parts[i] = fmt.Sprintf("%s:%s", tc.Function.Name, args)
	}
	return strings.Join(parts, "|")
}
