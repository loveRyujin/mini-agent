package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/loveRyujin/mini-agent/internal/inference"
)

type ReadFile struct{}

func (rf *ReadFile) Name() string { return "read_file" }

func (rf *ReadFile) Definition() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        rf.Name(),
			"description": "Read the contents of a given file path or search for files containing a pattern. When searching file contents, returns line numbers where the pattern is found.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "The relative path of a file in the working directory. If pattern is provided, this can be a directory path to search in.",
					},
				},
				"required": []string{"path"},
			},
		},
	}
}

func toolPathArg(args inference.ToolCall) (string, error) {
	v, ok := args.Function.Arguments["path"]
	if !ok || v == nil || v == "" {
		return ".", nil
	}
	path, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("path must be a string")
	}
	return path, nil
}

func (rf *ReadFile) Call(ctx context.Context, args inference.ToolCall) map[string]any {
	path, err := toolPathArg(args)
	if err != nil {
		return failResp(args.ID, err)
	}
	resolved, err := ResolveWorkspacePath(path)
	if err != nil {
		return failResp(args.ID, err)
	}
	content, err := os.ReadFile(resolved)
	if err != nil {
		return failResp(args.ID, err)
	}
	return successResp(args.ID, "file_content", string(content))
}

type WriteFile struct{}

func (wf *WriteFile) Name() string { return "write_file" }

func (wf *WriteFile) Definition() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        wf.Name(),
			"description": "Create or overwrite a file in the workspace with the given content.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "The relative path of the file to write.",
					},
					"content": map[string]any{
						"type":        "string",
						"description": "The full content to write to the file.",
					},
				},
				"required": []string{"path", "content"},
			},
		},
	}
}

func (wf *WriteFile) Call(ctx context.Context, args inference.ToolCall) map[string]any {
	path, ok := args.Function.Arguments["path"].(string)
	if !ok || path == "" {
		return failResp(args.ID, errors.New("path is required"))
	}
	content, ok := args.Function.Arguments["content"].(string)
	if !ok {
		return failResp(args.ID, errors.New("content must be a string"))
	}
	resolved, err := ResolveWorkspacePath(path)
	if err != nil {
		return failResp(args.ID, err)
	}
	if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
		return failResp(args.ID, err)
	}
	if err := os.WriteFile(resolved, []byte(content), 0o644); err != nil {
		return failResp(args.ID, err)
	}
	return successResp(args.ID, "path", path)
}

type ListFile struct{}

func (lf *ListFile) Name() string { return "list_file" }

func (lf *ListFile) Definition() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        lf.Name(),
			"description": "List files and directories at a given path. If no path is provided, lists files in the current directory.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "The relative path of a file in the working directory. If pattern is provided, this can be a directory path to search in.",
					},
				},
				"required": []string{"path"},
			},
		},
	}
}

func (lf *ListFile) Call(ctx context.Context, args inference.ToolCall) map[string]any {
	dir, err := toolPathArg(args)
	if err != nil {
		return failResp(args.ID, err)
	}
	resolved, err := ResolveWorkspacePath(dir)
	if err != nil {
		return failResp(args.ID, err)
	}
	var files []string
	err = filepath.Walk(resolved, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(resolved, path)
		if err != nil {
			return err
		}
		if relPath != "." {
			if info.IsDir() {
				files = append(files, relPath+"/")
			} else {
				files = append(files, relPath)
			}
		}
		return nil
	})
	if err != nil {
		return failResp(args.ID, err)
	}
	result, err := json.Marshal(files)
	if err != nil {
		return failResp(args.ID, err)
	}
	return successResp(args.ID, "files", string(result))
}

type WorkspaceSearch struct{}

func (ws *WorkspaceSearch) Name() string { return "workspace_search" }

func (ws *WorkspaceSearch) Definition() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        ws.Name(),
			"description": "Search for files or text content within the workspace by pattern.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"pattern": map[string]any{
						"type":        "string",
						"description": "Text to find in file contents, or a glob pattern (e.g. *.go) when mode is filename.",
					},
					"path": map[string]any{
						"type":        "string",
						"description": "Directory to search in, relative to the workspace. Defaults to \".\".",
					},
					"mode": map[string]any{
						"type":        "string",
						"enum":        []string{"content", "filename"},
						"description": "Search file contents (default) or match filenames by glob.",
					},
				},
				"required": []string{"pattern"},
			},
		},
	}
}

type searchMatch struct {
	File    string `json:"file"`
	Line    int    `json:"line,omitempty"`
	Content string `json:"content,omitempty"`
}

func (ws *WorkspaceSearch) Call(ctx context.Context, args inference.ToolCall) map[string]any {
	pattern, ok := args.Function.Arguments["pattern"].(string)
	if !ok || pattern == "" {
		return failResp(args.ID, errors.New("pattern is required"))
	}
	searchPath, err := toolPathArg(args)
	if err != nil {
		return failResp(args.ID, err)
	}
	mode := "content"
	if m, ok := args.Function.Arguments["mode"].(string); ok && m != "" {
		mode = m
	}
	resolved, err := ResolveWorkspacePath(searchPath)
	if err != nil {
		return failResp(args.ID, err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return failResp(args.ID, err)
	}
	if !info.IsDir() {
		return failResp(args.ID, errors.New("path must be a directory"))
	}
	var matches []searchMatch
	switch mode {
	case "filename":
		matches, err = searchFilenames(resolved, pattern)
	case "content":
		matches, err = searchFileContents(resolved, pattern)
	default:
		return failResp(args.ID, fmt.Errorf("unsupported mode %q", mode))
	}
	if err != nil {
		return failResp(args.ID, err)
	}
	result, err := json.Marshal(matches)
	if err != nil {
		return failResp(args.ID, err)
	}
	return successResp(args.ID, "matches", string(result))
}

func searchFilenames(root, pattern string) ([]searchMatch, error) {
	var matches []searchMatch
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err != nil {
			return err
		}
		if matched {
			matches = append(matches, searchMatch{File: rel})
		}
		return nil
	})
	return matches, err
}

func searchFileContents(root, pattern string) ([]searchMatch, error) {
	var matches []searchMatch
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			if strings.Contains(line, pattern) {
				matches = append(matches, searchMatch{
					File: rel, Line: lineNum, Content: line,
				})
			}
		}
		return scanner.Err()
	})
	return matches, err
}
