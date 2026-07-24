package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

const (
	defaultCommandTimeout = 60 * time.Second
)

type runCommandInput struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

type runCommandResult struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

func (workspaceTools *WorkspaceTools) executeRunCommand(ctx context.Context, root, arguments string) (string, error) {
	var input runCommandInput
	if err := decodeArguments(arguments, &input); err != nil {
		return "", fmt.Errorf("execute run_command: decode arguments: %w", err)
	}
	if input.Command == "" {
		return "", fmt.Errorf("execute run_command: command is empty")
	}
	if input.Args == nil {
		return "", fmt.Errorf("execute run_command: args is required")
	}
	if workspaceTools.commandApprover == nil {
		return "", fmt.Errorf("execute run_command: command approval is not configured")
	}
	approved, err := workspaceTools.commandApprover(ctx, CommandRequest{Command: input.Command, Args: input.Args})
	if err != nil {
		return "", fmt.Errorf("execute run_command: request approval: %w", err)
	}
	if !approved {
		return "", fmt.Errorf("execute run_command: user denied command")
	}

	commandCtx, cancel := context.WithTimeout(ctx, workspaceTools.commandTimeout)
	defer cancel()
	command := exec.CommandContext(commandCtx, input.Command, input.Args...)
	command.Dir = root
	var stdout, stderr limitedOutput
	command.Stdout = &stdout
	command.Stderr = &stderr
	err = command.Run()
	if commandCtx.Err() != nil {
		if errors.Is(commandCtx.Err(), context.DeadlineExceeded) {
			return "", fmt.Errorf("execute run_command: command timed out after %s", workspaceTools.commandTimeout)
		}
		return "", fmt.Errorf("execute run_command: command cancelled: %w", commandCtx.Err())
	}

	exitCode := 0
	if err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) {
			return "", fmt.Errorf("execute run_command: start command: %w", err)
		}
		exitCode = exitError.ExitCode()
	}
	result, err := json.Marshal(runCommandResult{
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	})
	if err != nil {
		return "", fmt.Errorf("execute run_command: encode result: %w", err)
	}
	return string(result), nil
}
