package main

import (
	"flag"
	"fmt"
	"strings"
)

type config struct {
	root string
	task string
}

// parseArgs parses the working directory and task.
func parseArgs(args []string) (config, error) {
	flags := flag.NewFlagSet("bare-agent", flag.ContinueOnError)
	root := flags.String("root", ".", "agent working directory")
	if err := flags.Parse(args); err != nil {
		return config{}, fmt.Errorf("parse arguments: %w", err)
	}
	task := strings.Join(flags.Args(), " ")
	return config{root: *root, task: task}, nil
}
