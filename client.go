package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

const (
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorBlue    = "\033[34m"
	colorMagenta = "\033[35m"
	colorCyan    = "\033[36m"
	colorRed     = "\033[91m"
	colorGray    = "\033[90m"
	colorReset   = "\033[0m"

	separator = "────────────────────────────────"
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

	ch := make(SSEResp, 10)

	resp, err := c.cli.Do(r)
	if err != nil {
		return nil, err
	}

	statusCode := resp.StatusCode
	if statusCode != http.StatusOK {
		return nil, errors.New("something wrong when calling llm api")
	}

	go func(ctx context.Context) {
		defer func() {
			_ = resp.Body.Close()
			close(ch)
		}()

		s := bufio.NewScanner(resp.Body)
		for s.Scan() {
			line := s.Text()
			if line == "" || line == "data: [DONE]" {
				continue
			}

			var v Response
			if err := json.Unmarshal([]byte(line[6:]), &v); err != nil {
				fmt.Println(err)
				return
			}

			ch <- v
		}
	}(ctx)

	return ch, nil
}
