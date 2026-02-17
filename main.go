package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	apiKey := os.Getenv("LLM_API_KEY")
	s := bufio.NewScanner(os.Stdin)
	cli := &Client{
		apiKey: apiKey,
	}
	for {
		text, ok := getUserInput(s)
		if !ok {
			continue
		}
		resp, err := cli.CallLLM(text)
		if err != nil {
			fmt.Println(err.Error())
			continue
		}
		fmt.Println(resp.Msg)
	}
}

func getUserInput(r *bufio.Scanner) (string, bool) {
	if !r.Scan() {
		return "", false
	}
	return r.Text(), true
}

type Response struct {
	Msg string
}

type Client struct {
	apiKey string
}

func (c *Client) CallLLM(prompt string) (*Response, error) {
	// Todo...
	return &Response{
		Msg: "Call LLM successfully",
	}, nil
}
