package main

import (
	"bare-agent/internal/agent"
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
)

// runInteractive runs an in-memory conversation until the user exits.
func runInteractive(ctx context.Context, runner *agent.Agent, input io.Reader, output, errorOutput io.Writer) error {
	scanner := bufio.NewScanner(input)
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
			runner.Reset()
			fmt.Fprintln(output, "new conversation")
			continue
		}

		result, err := runner.Run(ctx, task)
		if err != nil {
			fmt.Fprintln(errorOutput, err)
			continue
		}
		if result.ContextUsagePercent >= 90 {
			fmt.Fprintf(errorOutput, "warning: context usage is %d%%\n", result.ContextUsagePercent)
		}
		fmt.Fprintln(output, result.Content)
	}
}
