package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// readFile reads an existing file inside root.
func readFile(_ context.Context, root, requested string) (string, error) {
	path, err := resolvePath(root, requested)
	if err != nil {
		return "", err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read file %q: %w", requested, err)
	}

	return string(content), nil
}

type readFileArguments struct {
	Path string `json:"path"`
}

// executeReadFile runs readFile from JSON tool arguments.
func executeReadFile(ctx context.Context, root, arguments string) (string, error) {
	var input readFileArguments
	decoder := json.NewDecoder(strings.NewReader(arguments))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return "", fmt.Errorf("execute read_file: decode arguments: %w", err)
	}
	if input.Path == "" {
		return "", fmt.Errorf("execute read_file: path is empty")
	}

	content, err := readFile(ctx, root, input.Path)
	if err != nil {
		return "", fmt.Errorf("execute read_file: %w", err)
	}

	return content, nil
}
