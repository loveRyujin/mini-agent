package tools

import (
	"encoding/json"
)

func successResp(toolID string, kv ...any) map[string]any {
	data := make(map[string]any)
	for i := 0; i < len(kv); i += 2 {
		data[kv[i].(string)] = kv[i+1]
	}
	return toolResp(toolID, data, "SUCCESS")
}

func FailResp(toolID string, err error) map[string]any {
	return toolResp(toolID, map[string]any{"error": err.Error()}, "FAILED")
}

func failResp(toolID string, err error) map[string]any {
	return FailResp(toolID, err)
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
