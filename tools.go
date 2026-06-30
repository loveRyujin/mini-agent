package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Tool interface {
	Name() string
	Definition() map[string]any
	Call(context.Context, ToolCall) map[string]any
}

type getCurrentWeather struct{}

func (gcw *getCurrentWeather) Name() string {
	return "get_current_weather"
}

func (gcw *getCurrentWeather) Definition() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        gcw.Name(),
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
	}
}

func (gcw *getCurrentWeather) Call(ctx context.Context, args ToolCall) map[string]any {
	location, ok := args.Function.Arguments["location"].(string)
	if !ok {
		return failResp(args.ID, errors.New("unsupport argument type"))
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
		return failResp(args.ID, err)
	}

	return successResp(args.ID, "content", string(d))
}

type readFile struct{}

func (rf *readFile) Name() string {
	return "read_file"
}

func (rf *readFile) Definition() map[string]any {
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

func toolPathArg(args ToolCall) (string, error) {
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

func (rf *readFile) Call(ctx context.Context, args ToolCall) map[string]any {
	path, err := toolPathArg(args)
	if err != nil {
		return failResp(args.ID, err)
	}

	resolved, err := resolveWorkspacePath(path)
	if err != nil {
		return failResp(args.ID, err)
	}

	content, err := os.ReadFile(resolved)
	if err != nil {
		return failResp(args.ID, err)
	}

	return successResp(args.ID, "file_content", string(content))
}

type writeFile struct{}

func (wf *writeFile) Name() string {
	return "write_file"
}

func (wf *writeFile) Definition() map[string]any {
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

func (wf *writeFile) Call(ctx context.Context, args ToolCall) map[string]any {
	path, ok := args.Function.Arguments["path"].(string)
	if !ok || path == "" {
		return failResp(args.ID, errors.New("path is required"))
	}

	content, ok := args.Function.Arguments["content"].(string)
	if !ok {
		return failResp(args.ID, errors.New("content must be a string"))
	}

	resolved, err := resolveWorkspacePath(path)
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

type listFile struct{}

func (lf *listFile) Name() string {
	return "list_file"
}

func (lf *listFile) Definition() map[string]any {
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

func (lf *listFile) Call(ctx context.Context, args ToolCall) map[string]any {
	dir, err := toolPathArg(args)
	if err != nil {
		return failResp(args.ID, err)
	}

	resolved, err := resolveWorkspacePath(dir)
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

type workspaceSearch struct{}

func (ws *workspaceSearch) Name() string {
	return "workspace_search"
}

func (ws *workspaceSearch) Definition() map[string]any {
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

func (ws *workspaceSearch) Call(ctx context.Context, args ToolCall) map[string]any {
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

	resolved, err := resolveWorkspacePath(searchPath)
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
					File:    rel,
					Line:    lineNum,
					Content: line,
				})
			}
		}
		return scanner.Err()
	})
	return matches, err
}

func successResp(toolID string, kv ...any) map[string]any {
	data := make(map[string]any)
	for i := 0; i < len(kv); i += 2 {
		data[kv[i].(string)] = kv[i+1]
	}

	return toolResp(toolID, data, "SUCCESS")
}

func failResp(toolID string, err error) map[string]any {
	data := map[string]any{
		"error": err.Error(),
	}

	return toolResp(toolID, data, "FAILED")
}

func toolResp(toolID string, data map[string]any, status string) map[string]any {
	info := struct {
		Status string         `json:"status"`
		Data   map[string]any `json:"data"`
	}{
		Status: status,
		Data:   data,
	}

	d, err := json.Marshal(info)
	if err != nil {
		return map[string]any{
			"role":         "tool",
			"tool_call_id": toolID,
			"content":      `{"status": "FAILED", "data": "error marshaling tool response"}`,
		}
	}

	return map[string]any{
		"role":         "tool",
		"tool_call_id": toolID,
		"content":      string(d),
	}
}
