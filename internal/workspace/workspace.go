package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type Workspace struct {
	root string
}

func New(root string) (*Workspace, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	return &Workspace{root: abs}, nil
}

func NewFromCWD() (*Workspace, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return New(wd)
}

func (w *Workspace) Root() string {
	return w.root
}

func (w *Workspace) Display() string {
	if w.root == "" {
		return "."
	}
	if home, err := os.UserHomeDir(); err == nil {
		if rel, err := filepath.Rel(home, w.root); err == nil && !strings.HasPrefix(rel, "..") {
			return "~/" + filepath.ToSlash(rel)
		}
	}
	return w.root
}

func (w *Workspace) Resolve(rel string) (string, error) {
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
		abs, err = filepath.Abs(filepath.Join(w.root, rel))
		if err != nil {
			return "", err
		}
	}

	return validateWithinWorkspace(w.root, abs)
}

func validateWithinWorkspace(root, abs string) (string, error) {
	abs = filepath.Clean(abs)
	ws, err := filepath.Abs(root)
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
