package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
	if err := os.WriteFile(filepath.Join(root, "file.txt"), []byte("hello\nagent\n"), 0o600); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	result, err := NewWorkspaceTools().executeReadFile(context.Background(), root, `{"path":"file.txt"}`)
	if err != nil {
		t.Fatalf("executeReadFile() error = %v", err)
	}
	want := "   1 | hello\n   2 | agent\n   3 | "
	if result != want {
		t.Errorf("executeReadFile() = %q, want %q", result, want)
	}
}

func TestExecuteReadEmptyFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "empty.txt"), nil, 0o600); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	result, err := NewWorkspaceTools().executeReadFile(context.Background(), root, `{"path":"empty.txt"}`)
	if err != nil {
		t.Fatalf("executeReadFile() error = %v", err)
	}
	if result != "   1 | " {
		t.Errorf("executeReadFile() = %q, want one empty numbered line", result)
	}
}

func TestExecuteReadFileLineNumbersMatchSource(t *testing.T) {
	_, sourcePath, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test source")
	}
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read source file: %v", err)
	}
	result, err := NewWorkspaceTools().executeReadFile(
		context.Background(),
		filepath.Dir(sourcePath),
		`{"path":"read_file_test.go"}`,
	)
	if err != nil {
		t.Fatalf("executeReadFile() error = %v", err)
	}

	sourceLines := strings.Split(string(content), "\n")
	resultLines := strings.Split(result, "\n")
	if len(resultLines) != len(sourceLines) {
		t.Fatalf("result line count = %d, want %d", len(resultLines), len(sourceLines))
	}
	for index := range sourceLines {
		want := fmt.Sprintf("%4d | %s", index+1, sourceLines[index])
		if resultLines[index] != want {
			t.Fatalf("line %d = %q, want %q", index+1, resultLines[index], want)
		}
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
		{name: "unsupported range", ctx: context.Background(), arguments: `{"path":"file.txt","range":{"start":1,"end":2}}`, wantErr: `unknown field "range"`},
		{name: "path escape", ctx: context.Background(), arguments: `{"path":".."}`, wantErr: "escapes root"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewWorkspaceTools().executeReadFile(tt.ctx, root, tt.arguments)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("executeReadFile() error = %v, want to contain %q", err, tt.wantErr)
			}
		})
	}
}
