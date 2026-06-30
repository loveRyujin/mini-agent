package main

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
)

var errShellDenied = errors.New("shell execution denied by user")

type runShell struct{}

func (rs *runShell) Name() string {
	return "run_shell"
}

func (rs *runShell) Definition() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        rs.Name(),
			"description": "Run a Shell Execution command in the workspace. Requires Approval Gate before running.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"command": map[string]any{
						"type":        "string",
						"description": "The command to run via Shell Execution.",
					},
				},
				"required": []string{"command"},
			},
		},
	}
}

func (rs *runShell) ApprovalSummary(args ToolCall) string {
	cmd, _ := args.Function.Arguments["command"].(string)
	return cmd
}

func (rs *runShell) Call(ctx context.Context, args ToolCall) map[string]any {
	command, ok := args.Function.Arguments["command"].(string)
	if !ok || strings.TrimSpace(command) == "" {
		return failResp(args.ID, errors.New("command is required"))
	}

	stdout, stderr, exitCode, err := executeShell(ctx, command)
	if err != nil {
		return failResp(args.ID, err)
	}

	return successResp(args.ID,
		"stdout", stdout,
		"stderr", stderr,
		"exit_code", exitCode,
	)
}

func executeShell(ctx context.Context, command string) (stdout, stderr string, exitCode int, err error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = WorkspaceRoot()

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	runErr := cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()

	if runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			return stdout, stderr, exitErr.ExitCode(), nil
		}
		return stdout, stderr, -1, runErr
	}

	return stdout, stderr, 0, nil
}
