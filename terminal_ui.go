package main

import "fmt"

func newTerminalEmitter() EventEmitter {
	var (
		reasoning bool
		answered  bool
	)

	return func(e Event) {
		switch e.Kind {
		case EventReasoningDelta:
			if !reasoning {
				fmt.Printf("\n%s%s Reasoning %s%s\n", colorMagenta, separator, separator, colorReset)
				reasoning = true
			}
			fmt.Printf("%s%s%s", colorGray, e.Text, colorReset)

		case EventAnswerDelta:
			if !answered {
				fmt.Printf("\n\n%s%s Answer %s%s\n", colorCyan, separator, separator, colorReset)
				answered = true
			}
			fmt.Print(e.Text)

		case EventToolCall:
			if !reasoning {
				fmt.Printf("\n%s%s Reasoning %s%s\n", colorMagenta, separator, separator, colorReset)
				reasoning = true
			}
			fmt.Printf("\n%s[Tool Call]%s %s%s%s(%s%s%s)\n",
				colorYellow, colorReset,
				colorCyan, e.ToolName, colorReset,
				colorGray, e.ToolArguments, colorReset,
			)

		case EventToolResult:
			fmt.Printf("%s[Tool Result]%s %s%s%s\n",
				colorYellow, colorReset,
				colorGray, e.ToolContent, colorReset,
			)

		case EventTurnComplete:
			if e.AssistantMessage != "" {
				fmt.Print("\n")
			}
			fmt.Println()

		case EventUsage:
			fmt.Printf("%s%s Usage %s%s\n",
				colorGray, separator, separator, colorReset)
			fmt.Printf("%sCompletion_Tokens: %d%s\n%sPrompt_Tokens: %d%s\n%sTotal_Tokens: %d%s\n\n",
				colorYellow, e.Usage.CompletionToken, colorReset,
				colorMagenta, e.Usage.PromptToken, colorReset,
				colorBlue, e.Usage.TotalToken, colorReset)

		case EventError:
			fmt.Printf("%s%s%s\n", colorRed, e.Err.Error(), colorReset)
		}
	}
}
