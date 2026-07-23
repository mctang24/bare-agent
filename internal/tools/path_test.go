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
		{name: "absolute file inside root", requested: inside, want: resolvedInside},
		{name: "absolute root itself", requested: root, want: resolvedRoot},
		{name: "empty path", requested: "", wantErr: true},
		{name: "absolute path outside root", requested: outside, wantErr: true},
		{name: "parent traversal", requested: filepath.Join("..", filepath.Base(outsideRoot), "outside.txt"), wantErr: true},
		{name: "symlink escape", requested: "outside-link", wantErr: true},
		{name: "missing path", requested: "missing.txt", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveExistingPath(root, tt.requested)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("resolveExistingPath() error = nil, want an error; got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveExistingPath() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("resolveExistingPath() = %q, want %q", got, tt.want)
			}
		})
	}

	t.Run("root is a file", func(t *testing.T) {
		if _, err := resolveExistingPath(inside, "."); err == nil {
			t.Fatal("resolveExistingPath() error = nil, want an error")
		}
	})
}

func TestResolveWritablePathAcceptsOnlyAbsolutePathsInsideRoot(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "new", "file.txt")
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	got, err := resolveWritablePath(root, inside)
	if err != nil {
		t.Fatalf("resolveWritablePath() unexpected error: %v", err)
	}
	want := filepath.Join(resolvedRoot, "new", "file.txt")
	if got != want {
		t.Fatalf("resolveWritablePath() = %q, want %q", got, want)
	}

	outside := filepath.Join(t.TempDir(), "file.txt")
	if _, err := resolveWritablePath(root, outside); err == nil {
		t.Fatal("resolveWritablePath() error = nil, want an error")
	}
}
