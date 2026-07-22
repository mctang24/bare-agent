package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFileCreatesAndUpdates(t *testing.T) {
	root := t.TempDir()
	fileTools := newApprovedFileTools()
	write := findTool(t, fileTools, "write_file")
	read := findTool(t, fileTools, "read_file")

	if _, err := write.Execute(context.Background(), root, `{"path":"cmd/app/main.go","content":"package main\n"}`); err != nil {
		t.Fatalf("create write_file error = %v", err)
	}
	path := filepath.Join(root, "cmd", "app", "main.go")
	if content, err := os.ReadFile(path); err != nil || string(content) != "package main\n" {
		t.Fatalf("created content = %q, error = %v", content, err)
	}
	if _, err := read.Execute(context.Background(), root, `{"path":"cmd/app/main.go"}`); err != nil {
		t.Fatal(err)
	}
	if _, err := write.Execute(context.Background(), root, `{"path":"cmd/app/main.go","content":"package main\n\nfunc main() {}\n"}`); err != nil {
		t.Fatalf("update write_file error = %v", err)
	}
}

func TestWriteApprovalAndPathSafety(t *testing.T) {
	root := t.TempDir()
	fileTools := NewFileTools()
	write := findTool(t, fileTools, "write_file")
	if _, err := write.Execute(context.Background(), root, `{"path":"new.txt","content":"data"}`); err == nil || !strings.Contains(err.Error(), "approval is not configured") {
		t.Fatalf("missing approval error = %v", err)
	}
	fileTools.SetWriteApprover(func(context.Context, WriteRequest) (bool, error) { return false, nil })
	if _, err := write.Execute(context.Background(), root, `{"path":"new.txt","content":"data"}`); err == nil || !strings.Contains(err.Error(), "user denied") {
		t.Fatalf("denied error = %v", err)
	}
	fileTools.SetWriteApprover(func(context.Context, WriteRequest) (bool, error) { return true, nil })
	if _, err := write.Execute(context.Background(), root, `{"path":"../outside.txt","content":"data"}`); err == nil || !strings.Contains(err.Error(), "escapes root") {
		t.Fatalf("path escape error = %v", err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "outside")); err != nil {
		t.Fatal(err)
	}
	if _, err := write.Execute(context.Background(), root, `{"path":"outside/file.txt","content":"data"}`); err == nil || !strings.Contains(err.Error(), "escapes root") {
		t.Fatalf("symlink escape error = %v", err)
	}
}

func TestWriteFileRechecksAfterApproval(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "file.txt")
	if err := os.WriteFile(path, []byte("before"), 0o600); err != nil {
		t.Fatal(err)
	}
	fileTools := NewFileTools()
	read := findTool(t, fileTools, "read_file")
	write := findTool(t, fileTools, "write_file")
	if _, err := read.Execute(context.Background(), root, `{"path":"file.txt"}`); err != nil {
		t.Fatal(err)
	}
	fileTools.SetWriteApprover(func(context.Context, WriteRequest) (bool, error) {
		return true, os.WriteFile(path, []byte("external"), 0o600)
	})
	if _, err := write.Execute(context.Background(), root, `{"path":"file.txt","content":"agent"}`); err == nil || !strings.Contains(err.Error(), "changed since it was read") {
		t.Fatalf("write_file error = %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil || string(content) != "external" {
		t.Fatalf("content = %q, error = %v", content, err)
	}
}

func TestWriteFileRejectsParentSymlinkCreatedDuringApproval(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	fileTools := NewFileTools()
	write := findTool(t, fileTools, "write_file")
	fileTools.SetWriteApprover(func(context.Context, WriteRequest) (bool, error) {
		return true, os.Symlink(outside, filepath.Join(root, "new"))
	})

	_, err := write.Execute(context.Background(), root, `{"path":"new/file.txt","content":"data"}`)
	if err == nil || !strings.Contains(err.Error(), "escapes root") {
		t.Fatalf("write_file error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(outside, "file.txt")); !os.IsNotExist(err) {
		t.Fatalf("outside file error = %v, want not exist", err)
	}
}
