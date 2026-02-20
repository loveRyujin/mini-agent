package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
)

const (
	// use deepseek api
	url   = "https://api.deepseek.com/v1/chat/completions"
	model = "deepseek-chat"

	colorGreen = "\033[32m"
	colorCyan  = "\033[36m"
	colorReset = "\033[0m"
)

func main() {
	apiKey := os.Getenv("LLM_API_KEY")
	s := bufio.NewScanner(os.Stdin)
	cli := &Client{
		apiKey: apiKey,
		cli:    http.DefaultClient,
	}
	history := make([]map[string]any, 0, 1)
	history = append(history, map[string]any{
		"role":    "system",
		"content": "You are a helpful assistant.",
	})

	for {
		fmt.Printf("%sYou>%s ", colorGreen, colorReset)
		text, ok := getUserInput(s)
		if !ok {
			continue
		}
		history = append(history, map[string]any{
			"role":    "user",
			"content": text,
		})

		req := map[string]any{
			"model":    model,
			"messages": history,
			"stream":   true,
		}

		ctx := context.Background()
		ch, err := cli.CallLLMStream(ctx, req)
		if err != nil {
			fmt.Printf("\u001b[91m%s\u001b[0m\n", err.Error())
			continue
		}

		fmt.Printf("%sAgent>%s ", colorCyan, colorReset)

		var chunks []string

		for msg := range ch {
			if len(msg.Choices) == 0 {
				continue
			}

			switch {
			case msg.Choices[0].Delta.Content != "":
				content := msg.Choices[0].Delta.Content
				fmt.Print(content)
				chunks = append(chunks, content)
			case msg.Choices[0].Delta.FinishReason != "":
				fmt.Printf("\u001b[91m%s\u001b[0m", msg.Choices[0].Delta.FinishReason)
			}
		}

		if len(chunks) > 0 {
			fmt.Print("\n")
			history = append(history, map[string]any{
				"role":    "assistant",
				"content": strings.Join(chunks, " "),
			})
		}
	}
}

func getUserInput(r *bufio.Scanner) (string, bool) {
	if !r.Scan() {
		return "", false
	}
	text := strings.TrimSpace(r.Text())
	if text == "" {
		return "", false
	}
	return r.Text(), true
}

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
	Content      string `json:"content"`
	Role         string `json:"role"`
	FinishReason string `json:"finish_reason"`
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
}

func (c *Client) CallLLMStream(ctx context.Context, req map[string]any) (SSEResp, error) {
	if c.apiKey == "" {
		return nil, errors.New("empty api key")
	}

	var (
		b   bytes.Buffer
		err error
	)
	if req != nil {
		if err = json.NewEncoder(&b).Encode(req); err != nil {
			return nil, err
		}
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &b)
	if err != nil {
		return nil, err
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
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
