package main

import "testing"

func TestToolCallAccumulator_mergesStreamingChunks(t *testing.T) {
	acc := newToolCallAccumulator()
	acc.Add([]ToolCall{{
		Index: 0,
		ID:    "call-1",
		Type:  "function",
		Function: Function{
			Name:      "read_file",
			Arguments: map[string]any{"path": "a.txt"},
		},
	}})
	acc.Add([]ToolCall{{
		Index: 0,
		Function: Function{
			Name:      "read_file",
			Arguments: map[string]any{"path": "a.txt"},
		},
	}})

	calls := acc.Calls()
	if len(calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(calls))
	}
	if calls[0].ID != "call-1" {
		t.Fatalf("call id = %q, want call-1", calls[0].ID)
	}
}

func TestValidToolCalls_filtersIncomplete(t *testing.T) {
	calls := validToolCalls([]ToolCall{
		{ID: "call-1", Function: Function{Name: "list_file", Arguments: map[string]any{"path": "."}}},
		{ID: "call-2", Function: Function{Name: "  "}},
		{ID: "", Function: Function{Name: "read_file", Arguments: map[string]any{"path": "a.txt"}}},
	})
	if len(calls) != 1 {
		t.Fatalf("valid calls = %d, want 1", len(calls))
	}
}

func TestFunction_parsedArguments_fromRaw(t *testing.T) {
	f := Function{Name: "list_file", ArgsRaw: `{"path":"."}`}
	args := f.parsedArguments()
	if args["path"] != "." {
		t.Fatalf("args = %#v", args)
	}
}
