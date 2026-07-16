package main

import (
	"bare-agent/internal/agent"
	"bare-agent/internal/deepseek"
	"bare-agent/internal/trace"
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
	runner, err := agent.NewAgent(config.root, client, "")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if config.tracePath != "" {
		if err := runner.EnableTrace(trace.Writer{Path: config.tracePath}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
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
	fmt.Println(result.Content)
}
