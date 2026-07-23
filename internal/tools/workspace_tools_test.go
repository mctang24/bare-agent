package tools

import (
	"context"
	"testing"
)

func newApprovedWorkspaceTools() *WorkspaceTools {
	workspaceTools := NewWorkspaceTools()
	workspaceTools.SetWriteApprover(func(context.Context, WriteRequest) (bool, error) { return true, nil })
	return workspaceTools
}

func findTool(t *testing.T, workspaceTools *WorkspaceTools, name string) Tool {
	t.Helper()
	for _, tool := range workspaceTools.Definitions() {
		if tool.Name == name {
			return tool
		}
	}
	t.Fatalf("tool %q not found", name)
	return Tool{}
}
