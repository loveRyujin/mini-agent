package slash

import (
	"strings"
	"testing"
)

func TestParseSlashCommand(t *testing.T) {
	tests := []struct {
		input   string
		want    Result
		wantArg string
	}{
		{"hello", None, ""},
		{"/quit", Quit, ""},
		{"  /quit  ", Quit, ""},
		{"/QUIT", Quit, ""},
		{"/clear", Clear, ""},
		{"/help", Help, ""},
		{"/unknown", Unknown, "unknown"},
		{"/foo bar", Unknown, "foo"},
		{"/", Unknown, ""},
	}

	for _, tt := range tests {
		got, arg := Parse(tt.input)
		if got != tt.want || arg != tt.wantArg {
			t.Errorf("Parse(%q) = (%v, %q), want (%v, %q)",
				tt.input, got, arg, tt.want, tt.wantArg)
		}
	}
}

func TestSlashHelpText(t *testing.T) {
	text := HelpText()
	for _, want := range []string{
		"/quit", "/clear", "/help", "Y", "N",
		"LLM_API_URL", "MINI_AGENT_SYSTEM_PROMPT",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("HelpText() missing %q:\n%s", want, text)
		}
	}
}
