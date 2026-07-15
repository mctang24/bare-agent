package main

import (
	"bare-agent/internal/agent"
	"bare-agent/internal/deepseek"
	"context"
	"fmt"
	"os"
)

func main() {
	config, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	client, err := deepseek.NewClient(os.Getenv("DEEPSEEK_API_KEY"), "", "")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	runner, err := agent.NewAgent(config.root, client, "Inspect the working directory with tools before answering. Answer concisely and state the conclusion directly.")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if config.task == "" {
		if err := runInteractive(context.Background(), runner, os.Stdin, os.Stdout, os.Stderr); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	result, err := runner.Run(context.Background(), config.task)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if result.ContextUsagePercent >= 90 {
		fmt.Fprintf(os.Stderr, "warning: context usage is %d%%\n", result.ContextUsagePercent)
	}
	fmt.Println(result.Content)
}
