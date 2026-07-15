package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// searchText searches an existing path inside root for exact text.
func searchText(_ context.Context, root, requested, query string) (string, error) {
	if query == "" {
		return "", fmt.Errorf("search query is empty")
	}

	safeRoot, err := resolvePath(root, ".")
	if err != nil {
		return "", err
	}
	safePath, err := resolvePath(root, requested)
	if err != nil {
		return "", err
	}
	target, err := filepath.Rel(safeRoot, safePath)
	if err != nil {
		return "", fmt.Errorf("make search path relative: %w", err)
	}

	command := exec.Command("rg", "--line-number", "--with-filename", "--no-heading", "--color=never", "--fixed-strings", "--", query, target)
	command.Dir = safeRoot
	output, err := command.Output()
	if err == nil {
		return string(output), nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return "", nil
	}
	return "", fmt.Errorf("search text: %w", err)
}

type searchTextArguments struct {
	Path  string `json:"path"`
	Query string `json:"query"`
}

// executeSearchText runs searchText from JSON tool arguments.
func executeSearchText(ctx context.Context, root, arguments string) (string, error) {
	var input searchTextArguments
	decoder := json.NewDecoder(strings.NewReader(arguments))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return "", fmt.Errorf("execute search_text: decode arguments: %w", err)
	}
	if input.Path == "" {
		return "", fmt.Errorf("execute search_text: path is empty")
	}
	if input.Query == "" {
		return "", fmt.Errorf("execute search_text: query is empty")
	}

	result, err := searchText(ctx, root, input.Path, input.Query)
	if err != nil {
		return "", fmt.Errorf("execute search_text: %w", err)
	}

	return result, nil
}
