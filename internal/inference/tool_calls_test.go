package inference

import "testing"

func TestToolCallAccumulator_mergesStreamingChunks(t *testing.T) {
	acc := NewToolCallAccumulator()
	acc.Add([]ToolCall{{
		Index: 0,
		ID:    "call_1",
		Type:  "function",
		Function: Function{
			Name: "read_file",
			Arguments: map[string]any{
				"path": "foo.go",
			},
		},
	}})
	acc.Add([]ToolCall{{
		Index: 0,
		Function: Function{
			ArgsRaw: `{"path":"foo.go"}`,
		},
	}})

	calls := acc.Calls()
	if len(calls) != 1 {
		t.Fatalf("got %d calls, want 1", len(calls))
	}
	if calls[0].Function.Name != "read_file" {
		t.Fatalf("name = %q", calls[0].Function.Name)
	}
}

func TestValidToolCalls_filtersIncomplete(t *testing.T) {
	calls := ValidToolCalls([]ToolCall{
		{ID: "1", Function: Function{Name: "ok", Arguments: map[string]any{}}},
		{ID: "", Function: Function{Name: "no-id"}},
		{ID: "3", Function: Function{Name: "", Arguments: map[string]any{}}},
	})
	if len(calls) != 1 || calls[0].ID != "1" {
		t.Fatalf("got %v", calls)
	}
}

func TestFunction_parsedArguments_fromRaw(t *testing.T) {
	f := Function{ArgsRaw: `{"path":"x.go"}`}
	args := f.ParsedArguments()
	if args["path"] != "x.go" {
		t.Fatalf("args = %v", args)
	}
}
