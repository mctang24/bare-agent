package tools

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"
)

type WriteRequest struct {
	Tool string
	Path string
}

type WriteApprover func(context.Context, WriteRequest) (bool, error)

type FileTools struct {
	readHashes map[string][sha256.Size]byte
	approver   WriteApprover
}

func NewFileTools() *FileTools {
	return &FileTools{readHashes: make(map[string][sha256.Size]byte)}
}

func (fileTools *FileTools) SetWriteApprover(approver WriteApprover) {
	fileTools.approver = approver
}

func (fileTools *FileTools) ResetReadState() {
	clear(fileTools.readHashes)
}

func (fileTools *FileTools) Definitions() []Tool {
	pathProperty := map[string]any{
		"type":        "string",
		"description": "Path must be relative to the agent working directory. Use \".\" for the root. Do not use absolute paths.",
	}
	return []Tool{
		{
			Name:        "list_files",
			Description: "List the direct children of a directory.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": pathProperty,
				},
				"required":             []string{"path"},
				"additionalProperties": false,
			},
			Execute: executeListFiles,
		},
		{
			Name:        "read_file",
			Description: "Read the complete contents of a file. When multiple independent files are known, call read_file for all of them in the same tool round. Only the path parameter is supported; line ranges, offsets, and partial reads are not supported.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": pathProperty,
				},
				"required":             []string{"path"},
				"additionalProperties": false,
			},
			Execute: fileTools.executeReadFile,
		},
		{
			Name:        "search_text",
			Description: "Search for exact text in a file or directory and return 10 lines of context before and after each match. Submit all independent searches together in the same tool round.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": pathProperty,
					"query": map[string]any{
						"type":        "string",
						"description": "Exact text to search for.",
					},
				},
				"required":             []string{"path", "query"},
				"additionalProperties": false,
			},
			Execute: executeSearchText,
		},
		Tool{
			Name:        "edit_file",
			Description: "Edit an existing file by replacing one exact, unique text match. Read the file first.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path":       pathProperty,
					"old_string": map[string]any{"type": "string", "description": "Exact text to replace; it must occur once."},
					"new_string": map[string]any{"type": "string", "description": "Replacement text."},
				},
				"required":             []string{"path", "old_string", "new_string"},
				"additionalProperties": false,
			},
			Execute: fileTools.executeEditFile,
		},
		Tool{
			Name:        "write_file",
			Description: "Create a text file or replace an existing file. Read an existing file first.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path":    pathProperty,
					"content": map[string]any{"type": "string", "description": "Complete file content."},
				},
				"required":             []string{"path", "content"},
				"additionalProperties": false,
			},
			Execute: fileTools.executeWriteFile,
		},
	}
}

func (fileTools *FileTools) loadUnchangedFile(root, requested string) (string, []byte, os.FileMode, error) {
	path, err := resolveExistingPath(root, requested)
	if err != nil {
		return "", nil, 0, err
	}
	expected, ok := fileTools.readHashes[path]
	if !ok {
		return "", nil, 0, fmt.Errorf("%q must be read before writing", requested)
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", nil, 0, fmt.Errorf("stat %q: %w", requested, err)
	}
	if !info.Mode().IsRegular() {
		return "", nil, 0, fmt.Errorf("%q is not a regular file", requested)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", nil, 0, fmt.Errorf("read %q: %w", requested, err)
	}
	if sha256.Sum256(content) != expected {
		return "", nil, 0, fmt.Errorf("%q changed since it was read", requested)
	}
	if !utf8.Valid(content) {
		return "", nil, 0, fmt.Errorf("%q is not valid UTF-8 text", requested)
	}
	return path, content, info.Mode().Perm(), nil
}

func (fileTools *FileTools) requireWriteApproval(ctx context.Context, tool, path string) error {
	if fileTools.approver == nil {
		return fmt.Errorf("write approval is not configured")
	}
	approved, err := fileTools.approver(ctx, WriteRequest{Tool: tool, Path: path})
	if err != nil {
		return fmt.Errorf("request approval: %w", err)
	}
	if !approved {
		return fmt.Errorf("user denied writing %q", path)
	}
	return nil
}

func decodeArguments(arguments string, target any) error {
	decoder := json.NewDecoder(strings.NewReader(arguments))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}
