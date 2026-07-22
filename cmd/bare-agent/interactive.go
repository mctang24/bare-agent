package main

import (
	"bare-agent/internal/agent"
	"bare-agent/internal/tools"
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
)

func runTask(ctx context.Context, runner *agent.Agent, task string, output io.Writer) error {
	_, err := runner.Run(ctx, task, func(delta string) error {
		_, err := io.WriteString(output, delta)
		return err
	})
	return err
}

// runInteractive runs an in-memory conversation until the user exits.
func runInteractive(ctx context.Context, runner *agent.Agent, input io.Reader, output, errorOutput io.Writer) error {
	scanner := bufio.NewScanner(input)
	runner.SetWriteApprover(newScannerWriteApprover(scanner, output))
	for {
		fmt.Fprint(output, "> ")
		if !scanner.Scan() {
			return scanner.Err()
		}
		task := strings.TrimSpace(scanner.Text())
		switch task {
		case "":
			continue
		case "/exit":
			return nil
		case "/new":
			if err := runner.Reset(); err != nil {
				return err
			}
			fmt.Fprintln(output, "new conversation")
			continue
		}

		err := runTask(ctx, runner, task, output)
		fmt.Fprintln(output)
		if err != nil {
			fmt.Fprintln(errorOutput, err)
			continue
		}
	}
}

func newScannerWriteApprover(scanner *bufio.Scanner, output io.Writer) tools.WriteApprover {
	return func(_ context.Context, request tools.WriteRequest) (bool, error) {
		for {
			fmt.Fprintf(output, "允许 %s 写入 %q？[Y/n] ", request.Tool, request.Path)
			if !scanner.Scan() {
				return false, scanner.Err()
			}
			switch strings.TrimSpace(strings.ToLower(scanner.Text())) {
			case "", "y", "yes":
				return true, nil
			case "n", "no":
				return false, nil
			}
		}
	}
}
