package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "note.txt"), []byte("hello agent"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "docs"), 0o700); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		ctx       context.Context
		requested string
		want      string
		wantErr   bool
	}{
		{name: "读取文件", ctx: context.Background(), requested: "note.txt", want: "hello agent"},
		{name: "文件不存在", ctx: context.Background(), requested: "missing.txt", wantErr: true},
		{name: "拒绝越界路径", ctx: context.Background(), requested: "../outside.txt", wantErr: true},
		{name: "拒绝目录", ctx: context.Background(), requested: "docs", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, got, err := readFile(tt.ctx, root, tt.requested)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
			wantPath, err := filepath.EvalSymlinks(filepath.Join(root, tt.requested))
			if err != nil {
				t.Fatal(err)
			}
			if path != wantPath {
				t.Fatalf("path = %q, want %q", path, wantPath)
			}
		})
	}
}

func TestExecuteReadFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "file.txt"), []byte("hello"), 0o600); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	result, err := NewFileTools().executeReadFile(context.Background(), root, `{"path":"file.txt"}`)
	if err != nil {
		t.Fatalf("executeReadFile() error = %v", err)
	}
	if result != "hello" {
		t.Errorf("executeReadFile() = %q, want hello", result)
	}
}

func TestExecuteReadFileErrors(t *testing.T) {
	root := t.TempDir()
	tests := []struct {
		name      string
		ctx       context.Context
		arguments string
		wantErr   string
	}{
		{name: "invalid JSON", ctx: context.Background(), arguments: `{`, wantErr: "decode arguments"},
		{name: "empty path", ctx: context.Background(), arguments: `{}`, wantErr: "path is empty"},
		{name: "unknown field", ctx: context.Background(), arguments: `{"path":"file.txt","extra":true}`, wantErr: `unknown field "extra"`},
		{name: "path escape", ctx: context.Background(), arguments: `{"path":".."}`, wantErr: "escapes root"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewFileTools().executeReadFile(tt.ctx, root, tt.arguments)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("executeReadFile() error = %v, want to contain %q", err, tt.wantErr)
			}
		})
	}
}
