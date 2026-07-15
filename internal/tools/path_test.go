package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePath(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "inside.txt")
	if err := os.WriteFile(inside, []byte("inside"), 0o600); err != nil {
		t.Fatal(err)
	}

	outsideRoot := t.TempDir()
	outside := filepath.Join(outsideRoot, "outside.txt")
	if err := os.WriteFile(outside, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "outside-link")); err != nil {
		t.Fatal(err)
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	resolvedInside, err := filepath.EvalSymlinks(inside)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		requested string
		want      string
		wantErr   bool
	}{
		{name: "file inside root", requested: "inside.txt", want: resolvedInside},
		{name: "root itself", requested: ".", want: resolvedRoot},
		{name: "empty path", requested: "", wantErr: true},
		{name: "absolute path", requested: outside, wantErr: true},
		{name: "parent traversal", requested: filepath.Join("..", filepath.Base(outsideRoot), "outside.txt"), wantErr: true},
		{name: "symlink escape", requested: "outside-link", wantErr: true},
		{name: "missing path", requested: "missing.txt", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolvePath(root, tt.requested)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("resolvePath() error = nil, want an error; got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolvePath() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("resolvePath() = %q, want %q", got, tt.want)
			}
		})
	}

	t.Run("root is a file", func(t *testing.T) {
		if _, err := resolvePath(inside, "."); err == nil {
			t.Fatal("resolvePath() error = nil, want an error")
		}
	})
}
