package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchText(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "code"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "code", "main.go"), []byte("first line\nfind me\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := searchText(context.Background(), root, ".", "find me")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "code/main.go:2:find me") {
		t.Fatalf("unexpected search result %q", got)
	}

	tests := []struct {
		name      string
		ctx       context.Context
		requested string
		query     string
		wantErr   bool
	}{
		{name: "没有匹配", ctx: context.Background(), requested: ".", query: "missing"},
		{name: "查询为空", ctx: context.Background(), requested: ".", wantErr: true},
		{name: "拒绝越界路径", ctx: context.Background(), requested: "..", query: "find", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := searchText(tt.ctx, root, tt.requested, tt.query)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatal(err)
			}
			if !tt.wantErr && result != "" {
				t.Fatalf("got %q, want empty result", result)
			}
		})
	}
}

func TestSearchTextMultipleFiles(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"first.go":  "package sample\nvar target = 1\n",
		"second.go": "package sample\nvar target = 2\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o600); err != nil {
			t.Fatalf("write test file: %v", err)
		}
	}

	result, err := searchText(context.Background(), root, ".", "target")
	if err != nil {
		t.Fatalf("searchText() error = %v", err)
	}
	for _, expected := range []string{
		"first.go:2:var target = 1",
		"second.go:2:var target = 2",
	} {
		if !strings.Contains(result, expected) {
			t.Errorf("searchText() result = %q, want to contain %q", result, expected)
		}
	}
}

func TestSearchTextIncludesContext(t *testing.T) {
	root := t.TempDir()
	lines := make([]string, 25)
	for index := range lines {
		lines[index] = fmt.Sprintf("line %d", index+1)
	}
	lines[12] = "target"
	if err := os.WriteFile(filepath.Join(root, "file.txt"), []byte(strings.Join(lines, "\n")), 0o600); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	result, err := searchText(context.Background(), root, ".", "target")
	if err != nil {
		t.Fatalf("searchText() error = %v", err)
	}
	for _, expected := range []string{"file.txt-3-line 3", "file.txt:13:target", "file.txt-23-line 23"} {
		if !strings.Contains(result, expected) {
			t.Errorf("searchText() result = %q, want to contain %q", result, expected)
		}
	}
}

func TestExecuteSearchText(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "file.txt"), []byte("find target\n"), 0o600); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	result, err := executeSearchText(context.Background(), root, `{"path":".","query":"target"}`)
	if err != nil {
		t.Fatalf("executeSearchText() error = %v", err)
	}
	if !strings.Contains(result, "file.txt:1:find target") {
		t.Errorf("executeSearchText() = %q, want matching line", result)
	}
}

func TestExecuteSearchTextErrors(t *testing.T) {
	root := t.TempDir()
	tests := []struct {
		name      string
		ctx       context.Context
		arguments string
		wantErr   string
	}{
		{name: "invalid JSON", ctx: context.Background(), arguments: `{`, wantErr: "decode arguments"},
		{name: "empty path", ctx: context.Background(), arguments: `{"query":"target"}`, wantErr: "path is empty"},
		{name: "empty query", ctx: context.Background(), arguments: `{"path":"."}`, wantErr: "query is empty"},
		{name: "unknown field", ctx: context.Background(), arguments: `{"path":".","query":"target","extra":true}`, wantErr: `unknown field "extra"`},
		{name: "path escape", ctx: context.Background(), arguments: `{"path":"..","query":"target"}`, wantErr: "escapes root"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executeSearchText(tt.ctx, root, tt.arguments)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("executeSearchText() error = %v, want to contain %q", err, tt.wantErr)
			}
		})
	}
}
