package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type Tool interface {
	Name() string
	Definition() map[string]any
	Call(context.Context, ToolCall) (map[string]any, error)
}

type getCurrentWeather struct{}

func (gcw *getCurrentWeather) Name() string {
	return "get_current_weather"
}

func (gcw *getCurrentWeather) Definition() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        gcw.Name(),
			"description": "Get the current weather in a given location",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"location": map[string]any{
						"type":        "string",
						"description": "The city and state, e.g. San Francisco, CA",
					},
					"unit": map[string]any{
						"type": "string",
						"enum": []string{"celsius", "fahrenheit"},
					},
				},
			},
			"required": []string{"location"},
		},
	}
}

func (gcw *getCurrentWeather) Call(ctx context.Context, args ToolCall) (map[string]any, error) {
	location, ok := args.Function.Arguments["location"].(string)
	if !ok {
		return nil, errors.New("unsupport argument type")
	}

	resp := struct {
		Status string
		Data   map[string]any
	}{
		Status: "Succeed",
		Data: map[string]any{
			"temperature": 30,
			"description": fmt.Sprintf("The temperature in %s is 30", location),
		},
	}

	d, err := json.Marshal(&resp)
	if err != nil {
		return map[string]any{
			"role":         "tool",
			"tool_call_id": args.ID,
			"content":      fmt.Sprintf(`{"status": "FAILED", "data": "%s"}`, err),
		}, nil
	}

	return map[string]any{
		"role":         "tool",
		"tool_call_id": args.ID,
		"content":      string(d),
	}, nil
}

type readFile struct{}

func (rf *readFile) Name() string {
	return "read_file"
}

func (rf *readFile) Definition() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        rf.Name(),
			"description": "Read the contents of a given file path or search for files containing a pattern. When searching file contents, returns line numbers where the pattern is found.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "The relative path of a file in the working directory. If pattern is provided, this can be a directory path to search in.",
					},
				},
				"required": []string{"path"},
			},
		},
	}
}

func (rf *readFile) Call(ctx context.Context, args ToolCall) (map[string]any, error) {
	dir := "."
	if args.Function.Arguments["path"] != "" {
		dir = args.Function.Arguments["path"].(string)
	}

	content, err := os.ReadFile(dir)
	if err != nil {
		return failResp(args.ID, err), nil
	}

	return successResp(args.ID, "file_content", string(content)), nil
}

func successResp(toolID string, kv ...any) map[string]any {
	data := make(map[string]any)
	for i := 0; i < len(kv); i += 2 {
		data[kv[i].(string)] = kv[i+1]
	}

	return toolResp(toolID, data, "SUCCESS")
}

func failResp(toolID string, err error) map[string]any {
	data := map[string]any{
		"error": err.Error(),
	}

	return toolResp(toolID, data, "FAILED")
}

func toolResp(toolID string, data map[string]any, status string) map[string]any {
	info := struct {
		Status string         `json:"status"`
		Data   map[string]any `json:"data"`
	}{
		Status: status,
		Data:   data,
	}

	d, err := json.Marshal(info)
	if err != nil {
		return map[string]any{
			"role":         "tool",
			"tool_call_id": toolID,
			"content":      `{"status": "FAILED", "data": "error marshaling tool response"}`,
		}
	}

	return map[string]any{
		"role":         "tool",
		"tool_call_id": toolID,
		"content":      string(d),
	}
}
