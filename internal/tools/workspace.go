package tools

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var workspaceDir string

func InitWorkspace() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	workspaceDir = wd
	return nil
}

// SetWorkspaceRootForTest overrides the workspace root in tests.
func SetWorkspaceRootForTest(dir string) {
	workspaceDir = dir
}

func WorkspaceRoot() string {
	if workspaceDir == "" {
		_ = InitWorkspace()
	}
	return workspaceDir
}

func WorkspaceDisplay() string {
	wd := WorkspaceRoot()
	if wd == "" {
		return "."
	}
	if home, err := os.UserHomeDir(); err == nil {
		if rel, err := filepath.Rel(home, wd); err == nil && !strings.HasPrefix(rel, "..") {
			return "~/" + filepath.ToSlash(rel)
		}
	}
	return wd
}

func ResolveWorkspacePath(rel string) (string, error) {
	if workspaceDir == "" {
		if err := InitWorkspace(); err != nil {
			return "", err
		}
	}

	if rel == "" {
		rel = "."
	}

	var abs string
	if filepath.IsAbs(rel) {
		var err error
		abs, err = filepath.Abs(rel)
		if err != nil {
			return "", err
		}
	} else {
		var err error
		abs, err = filepath.Abs(filepath.Join(workspaceDir, rel))
		if err != nil {
			return "", err
		}
	}

	return validateWithinWorkspace(abs)
}

func validateWithinWorkspace(abs string) (string, error) {
	abs = filepath.Clean(abs)
	ws, err := filepath.Abs(workspaceDir)
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(ws, abs)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("path escapes workspace")
	}
	return abs, nil
}
