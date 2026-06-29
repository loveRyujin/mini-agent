package main

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
	Index int64 `json:"index"`
	Delta Delta `json:"delta"`
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
	Arguments map[string]any `json:"arguments"`
}

func (f *Function) UnmarshalJSON(data []byte) error {
	var tmpF struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	}

	if err := json.Unmarshal(data, &tmpF); err != nil {
		return err
	}

	args := make(map[string]any)
	if err := json.Unmarshal([]byte(tmpF.Arguments), &args); err != nil {
		return err
	}

	*f = Function{
		Name:      tmpF.Name,
		Arguments: args,
	}
	return nil
}

type Usage struct {
	CompletionToken int64 `json:"completion_tokens"`
	PromptToken     int64 `json:"prompt_tokens"`
	TotalToken      int64 `json:"total_tokens"`
}

type SSEResp chan Response

type Client struct {
	cli    *http.Client
	apiKey string
	url    string
	model  string
}

func defaultHTTPClient() *http.Client {
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

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, &b)
	if err != nil {
		return nil, err
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "text/event-stream")
	r.Header.Set("Cache-Control", "no-cache")
	if c.apiKey != "" {
		r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}

	resp, err := c.cli.Do(r)
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

			data, ok := sseData(line)
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

func sseData(line string) (string, bool) {
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
