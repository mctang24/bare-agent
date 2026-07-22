package tools

import (
	"context"
	"testing"
)

func newApprovedFileTools() *FileTools {
	fileTools := NewFileTools()
	fileTools.SetWriteApprover(func(context.Context, WriteRequest) (bool, error) { return true, nil })
	return fileTools
}

func findTool(t *testing.T, fileTools *FileTools, name string) Tool {
	t.Helper()
	for _, tool := range fileTools.Definitions() {
		if tool.Name == name {
			return tool
		}
	}
	t.Fatalf("tool %q not found", name)
	return Tool{}
}
