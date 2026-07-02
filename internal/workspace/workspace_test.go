package workspace

import (
	"path/filepath"
	"testing"
)

func TestResolve_withinWorkspace(t *testing.T) {
	dir := t.TempDir()
	ws, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	got, err := ws.Resolve("sub/file.txt")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want := filepath.Join(dir, "sub", "file.txt")
	if got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
}

func TestResolve_rejectsEscape(t *testing.T) {
	dir := t.TempDir()
	ws, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ws.Resolve("../outside.txt")
	if err == nil {
		t.Fatal("expected error for path escape")
	}
}
