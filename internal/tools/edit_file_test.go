package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEditFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "main.go")
	if err := os.WriteFile(path, []byte("package main\n\nvar value = 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	fileTools := newApprovedFileTools()
	read := findTool(t, fileTools, "read_file")
	edit := findTool(t, fileTools, "edit_file")
	if _, err := read.Execute(context.Background(), root, `{"path":"main.go"}`); err != nil {
		t.Fatalf("read_file error = %v", err)
	}
	result, err := edit.Execute(context.Background(), root, `{"path":"main.go","old_string":"value = 1","new_string":"value = 2"}`)
	if err != nil {
		t.Fatalf("edit_file error = %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "package main\n\nvar value = 2\n" || !strings.Contains(result, "replaced 1 occurrence") {
		t.Fatalf("content = %q, result = %q", content, result)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v, want 0600", info.Mode().Perm())
	}
}

func TestEditFileRejectsUnsafeChanges(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "file.txt")
	if err := os.WriteFile(path, []byte("same\nsame\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	fileTools := newApprovedFileTools()
	read := findTool(t, fileTools, "read_file")
	edit := findTool(t, fileTools, "edit_file")

	tests := []struct {
		name      string
		prepare   func()
		arguments string
		wantErr   string
	}{
		{name: "must read first", arguments: `{"path":"file.txt","old_string":"same","new_string":"new"}`, wantErr: "must be read"},
		{name: "empty old string", arguments: `{"path":"file.txt","old_string":"","new_string":"new"}`, wantErr: "old_string is empty"},
		{name: "multiple matches", prepare: func() { _, _ = read.Execute(context.Background(), root, `{"path":"file.txt"}`) }, arguments: `{"path":"file.txt","old_string":"same","new_string":"new"}`, wantErr: "occurs 2 times"},
		{name: "external change", prepare: func() {
			_, _ = read.Execute(context.Background(), root, `{"path":"file.txt"}`)
			_ = os.WriteFile(path, []byte("changed\n"), 0o600)
		}, arguments: `{"path":"file.txt","old_string":"changed","new_string":"new"}`, wantErr: "changed since it was read"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileTools.ResetReadState()
			if err := os.WriteFile(path, []byte("same\nsame\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if tt.prepare != nil {
				tt.prepare()
			}
			_, err := edit.Execute(context.Background(), root, tt.arguments)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}
