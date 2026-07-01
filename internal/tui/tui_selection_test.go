package tui

import "testing"

func TestExtractSelectionText(t *testing.T) {
	lines := []string{"hello world", "second line", "third"}

	got := extractSelectionText(lines, textPos{0, 6}, textPos{1, 5})
	want := "world\nsecond"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestExtractSelectionText_singleLine(t *testing.T) {
	lines := []string{"abcdef"}
	got := extractSelectionText(lines, textPos{0, 1}, textPos{0, 3})
	if got != "bcd" {
		t.Fatalf("got %q, want bcd", got)
	}
}
