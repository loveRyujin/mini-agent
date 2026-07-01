package inference

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Response struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

type Choice struct {
	Index        int64  `json:"index"`
	Delta        Delta  `json:"delta"`
	FinishReason string `json:"finish_reason"`
}

type Delta struct {
	Content      string     `json:"content"`
	Role         string     `json:"role"`
	FinishReason string     `json:"finish_reason"`
	Reasoning    string     `json:"reasoning"`
	ToolCalls    []ToolCall `json:"tool_calls"`
}

type ToolCall struct {
	Index    int64    `json:"index"`
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

type Function struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"-"`
	ArgsRaw   string         `json:"-"`
}

func (f *Function) UnmarshalJSON(data []byte) error {
	var tmp struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	f.Name = tmp.Name
	if len(tmp.Arguments) == 0 {
		return nil
	}

	var args map[string]any
	if err := json.Unmarshal(tmp.Arguments, &args); err == nil {
		f.Arguments = args
		return nil
	}

	var raw string
	if err := json.Unmarshal(tmp.Arguments, &raw); err == nil {
		f.ArgsRaw = raw
		return nil
	}

	return nil
}

func (f Function) ParsedArguments() map[string]any {
	if len(f.Arguments) > 0 {
		return f.Arguments
	}
	if f.ArgsRaw == "" {
		return nil
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(f.ArgsRaw), &args); err != nil {
		return nil
	}
	return args
}

type Usage struct {
	CompletionToken int64 `json:"completion_tokens"`
	PromptToken     int64 `json:"prompt_tokens"`
	TotalToken      int64 `json:"total_tokens"`
}

type SSEResp chan Response

// Backend streams chat-completions from an Inference Backend.
type Backend interface {
	CallLLMStream(ctx context.Context, req map[string]any) (SSEResp, error)
}

type Client struct {
	HTTPClient *http.Client
	APIKey     string
	URL        string
	Model      string
}

func DefaultHTTPClient() *http.Client {
	return http.DefaultClient
}

func (c *Client) CallLLMStream(ctx context.Context, req map[string]any) (SSEResp, error) {
	var (
		b   bytes.Buffer
		err error
	)
	if req != nil {
		if err = json.NewEncoder(&b).Encode(req); err != nil {
			return nil, err
		}
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, &b)
	if err != nil {
		return nil, err
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "text/event-stream")
	r.Header.Set("Cache-Control", "no-cache")
	if c.APIKey != "" {
		r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		msg, readErr := readAPIError(resp)
		_ = resp.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		return nil, errors.New(msg)
	}

	ch := make(SSEResp, 10)

	go func(ctx context.Context) {
		defer func() {
			_ = resp.Body.Close()
			close(ch)
		}()

		s := bufio.NewScanner(resp.Body)
		for s.Scan() {
			line := strings.TrimSpace(s.Text())
			if line == "" {
				continue
			}

			data, ok := SSEData(line)
			if !ok {
				continue
			}

			var v Response
			if err := json.Unmarshal([]byte(data), &v); err != nil {
				ch <- Response{
					Choices: []Choice{{
						Delta: Delta{Content: fmt.Sprintf("SSE 解析失败: %v", err)},
					}},
				}
				return
			}

			ch <- v
		}
	}(ctx)

	return ch, nil
}

func SSEData(line string) (string, bool) {
	if !strings.HasPrefix(line, "data:") {
		return "", false
	}
	data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
	if data == "" || data == "[DONE]" {
		return "", false
	}
	return data, true
}

func readAPIError(resp *http.Response) (string, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var apiErr struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Error.Message != "" {
		return apiErr.Error.Message, nil
	}

	if len(body) > 0 {
		return string(body), nil
	}

	return fmt.Sprintf("LLM API 错误 (HTTP %d)", resp.StatusCode), nil
}
