package tools

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"unicode/utf8"
)

type editFileArguments struct {
	Path      string `json:"path"`
	OldString string `json:"old_string"`
	NewString string `json:"new_string"`
}

func (fileTools *FileTools) executeEditFile(ctx context.Context, root, arguments string) (string, error) {
	var input editFileArguments
	if err := decodeArguments(arguments, &input); err != nil {
		return "", fmt.Errorf("execute edit_file: decode arguments: %w", err)
	}
	if input.Path == "" {
		return "", fmt.Errorf("execute edit_file: path is empty")
	}
	if input.OldString == "" {
		return "", fmt.Errorf("execute edit_file: old_string is empty")
	}
	if !utf8.ValidString(input.NewString) {
		return "", fmt.Errorf("execute edit_file: new_string is not valid UTF-8")
	}

	path, content, mode, err := fileTools.loadUnchangedFile(root, input.Path)
	if err != nil {
		return "", fmt.Errorf("execute edit_file: %w", err)
	}
	count := strings.Count(string(content), input.OldString)
	if count != 1 {
		return "", fmt.Errorf("execute edit_file: old_string occurs %d times, want exactly once", count)
	}
	updated := strings.Replace(string(content), input.OldString, input.NewString, 1)
	if updated == string(content) {
		return "", fmt.Errorf("execute edit_file: replacement does not change the file")
	}
	if err := fileTools.requireWriteApproval(ctx, "edit_file", input.Path); err != nil {
		return "", fmt.Errorf("execute edit_file: %w", err)
	}
	path, content, mode, err = fileTools.loadUnchangedFile(root, input.Path)
	if err != nil {
		return "", fmt.Errorf("execute edit_file: %w", err)
	}
	updated = strings.Replace(string(content), input.OldString, input.NewString, 1)
	if err := writeFileAtomically(path, []byte(updated), mode); err != nil {
		return "", fmt.Errorf("execute edit_file: write %q: %w", input.Path, err)
	}
	fileTools.readHashes[path] = sha256.Sum256([]byte(updated))
	return fmt.Sprintf("Edited %s: replaced 1 occurrence", input.Path), nil
}
