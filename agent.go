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

type Agent struct {
	ApiKey string
	Cli    *Client
}

func NewAgent(apiKey, url, model string) *Agent {
	return &Agent{
		ApiKey: apiKey,
		Cli: &Client{
			cli:   http.DefaultClient,
			url:   url,
			model: model,
		},
	}
}

func (a *Agent) Run() error {
	s := bufio.NewScanner(os.Stdin)

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
			"model":    a.Cli.model,
			"messages": history,
			"stream":   true,
			"stream_options": map[string]any{
				"include_usage": true,
			},
			"tools": []map[string]any{
				{
					"type": "function",
					"function": map[string]any{
						"name":        "get_current_weather",
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
				},
			},
			"tool_choice": "auto",
		}

		ctx := context.Background()
		fmt.Printf("\n%sAgent>%s ", colorCyan, colorReset)

		var (
			chunks     []string
			tokenUsage []Usage
			reasoning  bool
			answered   bool
		)

		for {
			req["messages"] = history

			ch, err := a.Cli.CallLLMStream(ctx, req)
			if err != nil {
				fmt.Printf("%s%s%s\n", colorRed, err.Error(), colorReset)
				break
			}

			var isToolCall bool

			for msg := range ch {
				if len(msg.Choices) > 0 {
					switch {
					case msg.Choices[0].Delta.Reasoning != "":
						if !reasoning {
							fmt.Printf("\n%s%s Reasoning %s%s\n", colorMagenta, separator, separator, colorReset)
							reasoning = true
						}
						fmt.Printf("%s%s%s", colorGray, msg.Choices[0].Delta.Reasoning, colorReset)

					case len(msg.Choices[0].Delta.ToolCalls) > 0:
						if !reasoning {
							fmt.Printf("\n%s%s Reasoning %s%s\n", colorMagenta, separator, separator, colorReset)
							reasoning = true
						}

						toolCall := msg.Choices[0].Delta.ToolCalls[0]
						isToolCall = true

						fmt.Printf("\n%s[Tool Call]%s %s%s%s(%s%s%s)\n",
							colorYellow, colorReset,
							colorCyan, toolCall.Function.Name, colorReset,
							colorGray, toolCall.Function.Arguments, colorReset,
						)

						argsJson, err := json.Marshal(toolCall.Function.Arguments)
						if err != nil {
							fmt.Printf("%s%v%s\n", colorRed, err, colorReset)
						}
						history = append(history, map[string]any{
							"role": "assistant",
							"tool_calls": []map[string]any{
								{
									"id":   toolCall.ID,
									"type": "function",
									"function": map[string]any{
										"name":      toolCall.Function.Name,
										"arguments": string(argsJson),
									},
								},
							},
						})

						resp, err := getCurrentWeather(ctx, toolCall)
						if err != nil {
							fmt.Printf("%s%v%s\n", colorRed, err, colorReset)
						}
						history = append(history, resp)
						fmt.Printf("%s[Tool Result]%s %s%v%s\n",
							colorYellow, colorReset,
							colorGray, resp["content"], colorReset,
						)

					case msg.Choices[0].Delta.Content != "":
						if !answered {
							fmt.Printf("\n\n%s%s Answer %s%s\n", colorCyan, separator, separator, colorReset)
							answered = true
						}
						content := msg.Choices[0].Delta.Content
						fmt.Print(content)
						chunks = append(chunks, content)

					case msg.Choices[0].Delta.FinishReason != "":
					}
				}

				tokenUsage = append(tokenUsage, msg.Usage)
			}

			if !isToolCall {
				break
			}
		}

		if len(chunks) > 0 {
			fmt.Print("\n")
			history = append(history, map[string]any{
				"role":    "assistant",
				"content": strings.Join(chunks, " "),
			})
		}

		fmt.Println()
		if len(tokenUsage) > 0 {
			cToken, pToken := 0, 0
			for _, usage := range tokenUsage {
				cToken += int(usage.CompletionToken)
				pToken += int(usage.PromptToken)
			}
			fmt.Printf("%s%s Usage %s%s\n",
				colorGray, separator, separator, colorReset)
			fmt.Printf("%sCompletion_Tokens: %d%s\n%sPrompt_Tokens: %d%s\n%sTotal_Tokens: %d%s\n\n",
				colorYellow, cToken, colorReset,
				colorMagenta, pToken, colorReset,
				colorBlue, tokenUsage[len(tokenUsage)-1].TotalToken, colorReset)
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
			// remove prefix: [data: ]
			if err := json.Unmarshal([]byte(line[6:]), &v); err != nil {
				fmt.Println(err)
				return
			}

			ch <- v
		}
	}(ctx)

	return ch, nil
}

func getCurrentWeather(_ context.Context, args ToolCall) (map[string]any, error) {
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
