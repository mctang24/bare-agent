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
	commandOutputLimit    = 50000
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

type commandOutput struct {
	head  []byte
	tail  []byte
	total int
}

func (output *commandOutput) Write(data []byte) (int, error) {
	written := len(data)
	output.total += written
	keepEach := commandOutputLimit / 2
	if len(output.head) < keepEach {
		keep := keepEach - len(output.head)
		if keep > len(data) {
			keep = len(data)
		}
		output.head = append(output.head, data[:keep]...)
		data = data[keep:]
	}
	if len(data) == 0 {
		return written, nil
	}
	combined := append(output.tail, data...)
	if len(combined) > keepEach {
		combined = combined[len(combined)-keepEach:]
	}
	output.tail = combined
	return written, nil
}

func (output *commandOutput) String() string {
	if output.total <= commandOutputLimit {
		return string(append(output.head, output.tail...))
	}
	omitted := output.total - commandOutputLimit
	var marker string
	for {
		marker = fmt.Sprintf("\n\n[... truncated %d bytes ...]\n\n", omitted)
		next := output.total - (commandOutputLimit - len(marker))
		if next == omitted {
			break
		}
		omitted = next
	}
	kept := commandOutputLimit - len(marker)
	headLength := (kept + 1) / 2
	tailLength := kept - headLength
	return string(output.head[:headLength]) + marker + string(output.tail[len(output.tail)-tailLength:])
}

func (fileTools *FileTools) executeRunCommand(ctx context.Context, root, arguments string) (string, error) {
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
	if fileTools.commandApprover == nil {
		return "", fmt.Errorf("execute run_command: command approval is not configured")
	}
	approved, err := fileTools.commandApprover(ctx, CommandRequest{Command: input.Command, Args: input.Args})
	if err != nil {
		return "", fmt.Errorf("execute run_command: request approval: %w", err)
	}
	if !approved {
		return "", fmt.Errorf("execute run_command: user denied command")
	}

	commandCtx, cancel := context.WithTimeout(ctx, fileTools.commandTimeout)
	defer cancel()
	command := exec.CommandContext(commandCtx, input.Command, input.Args...)
	command.Dir = root
	var stdout, stderr commandOutput
	command.Stdout = &stdout
	command.Stderr = &stderr
	err = command.Run()
	if commandCtx.Err() != nil {
		if errors.Is(commandCtx.Err(), context.DeadlineExceeded) {
			return "", fmt.Errorf("execute run_command: command timed out after %s", fileTools.commandTimeout)
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
