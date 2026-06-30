package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveWorkspacePath_withinWorkspace(t *testing.T) {
	dir := t.TempDir()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	workspaceDir = dir

	got, err := resolveWorkspacePath("sub/file.txt")
	if err != nil {
		t.Fatalf("resolveWorkspacePath: %v", err)
	}
	want := filepath.Join(dir, "sub", "file.txt")
	if got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
}

func TestResolveWorkspacePath_rejectsEscape(t *testing.T) {
	dir := t.TempDir()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	workspaceDir = dir

	_, err = resolveWorkspacePath("../outside.txt")
	if err == nil {
		t.Fatal("expected error for path escape")
	}
}
