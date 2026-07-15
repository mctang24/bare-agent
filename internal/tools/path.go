package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// resolvePath returns a path that stays inside root.
func resolvePath(root, requested string) (string, error) {
	if requested == "" {
		return "", fmt.Errorf("resolve path: requested path is empty")
	}
	if filepath.IsAbs(requested) {
		return "", fmt.Errorf("resolve path %q: absolute paths are not allowed", requested)
	}

	rootPath, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve root %q: %w", root, err)
	}
	rootPath, err = filepath.EvalSymlinks(rootPath)
	if err != nil {
		return "", fmt.Errorf("resolve root %q: %w", root, err)
	}
	rootInfo, err := os.Stat(rootPath)
	if err != nil {
		return "", fmt.Errorf("stat root %q: %w", rootPath, err)
	}
	if !rootInfo.IsDir() {
		return "", fmt.Errorf("resolve root %q: root is not a directory", rootPath)
	}

	resolved, err := filepath.EvalSymlinks(filepath.Join(rootPath, requested))
	if err != nil {
		return "", fmt.Errorf("resolve path %q: %w", requested, err)
	}
	relative, err := filepath.Rel(rootPath, resolved)
	if err != nil {
		return "", fmt.Errorf("compare path %q with root %q: %w", resolved, rootPath, err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("resolve path %q: path escapes root", requested)
	}

	return resolved, nil
}
