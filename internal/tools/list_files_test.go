package tools

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestListFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "b.txt"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.txt"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	subdirectory := filepath.Join(root, "code")
	if err := os.Mkdir(subdirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subdirectory, "main.go"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		ctx       context.Context
		requested string
		want      []string
		wantErr   bool
	}{
		{name: "root directory", ctx: context.Background(), requested: ".", want: []string{"a.txt", "b.txt", "code/"}},
		{name: "subdirectory", ctx: context.Background(), requested: "code", want: []string{"main.go"}},
		{name: "file is not directory", ctx: context.Background(), requested: "a.txt", wantErr: true},
		{name: "path escapes root", ctx: context.Background(), requested: "..", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := listFiles(tt.ctx, root, tt.requested)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("listFiles() error = nil, want an error; got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("listFiles() unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("listFiles() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExecuteListFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "file.txt"), []byte("content"), 0o600); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	result, err := executeListFiles(context.Background(), root, `{"path":"."}`)
	if err != nil {
		t.Fatalf("executeListFiles() error = %v", err)
	}
	if result != `["file.txt"]` {
		t.Errorf("executeListFiles() = %q, want %q", result, `["file.txt"]`)
	}
}

func TestExecuteListFilesErrors(t *testing.T) {
	root := t.TempDir()
	tests := []struct {
		name      string
		arguments string
		wantErr   string
	}{
		{name: "invalid JSON", arguments: `{`, wantErr: "decode arguments"},
		{name: "empty path", arguments: `{}`, wantErr: "path is empty"},
		{name: "unknown field", arguments: `{"path":".","extra":true}`, wantErr: `unknown field "extra"`},
		{name: "path escape", arguments: `{"path":".."}`, wantErr: "escapes root"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executeListFiles(context.Background(), root, tt.arguments)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("executeListFiles() error = %v, want to contain %q", err, tt.wantErr)
			}
		})
	}
}
