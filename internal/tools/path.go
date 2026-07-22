package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// resolveExistingPath returns an existing path that stays inside root.
func resolveExistingPath(root, requested string) (string, error) {
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

	resolvedPath, err := filepath.EvalSymlinks(filepath.Join(rootPath, requested))
	if err != nil {
		return "", fmt.Errorf("resolve path %q: %w", requested, err)
	}
	pathFromRoot, err := filepath.Rel(rootPath, resolvedPath)
	if err != nil {
		return "", fmt.Errorf("compare path %q with root %q: %w", resolvedPath, rootPath, err)
	}
	if pathFromRoot == ".." || strings.HasPrefix(pathFromRoot, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("resolve path %q: path escapes root", requested)
	}

	return resolvedPath, nil
}

// resolveWritablePath resolves a path whose final components may not exist.
func resolveWritablePath(root, requested string) (string, error) {
	if requested == "" {
		return "", fmt.Errorf("resolve writable path: requested path is empty")
	}
	if filepath.IsAbs(requested) {
		return "", fmt.Errorf("resolve writable path %q: absolute paths are not allowed", requested)
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
	targetPath := filepath.Join(rootPath, requested)
	pathFromRoot, err := filepath.Rel(rootPath, targetPath)
	if err != nil {
		return "", fmt.Errorf("compare path %q with root %q: %w", targetPath, rootPath, err)
	}
	if pathFromRoot == ".." || strings.HasPrefix(pathFromRoot, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("resolve writable path %q: path escapes root", requested)
	}

	existingPath := targetPath
	for {
		if _, err := os.Lstat(existingPath); err == nil {
			break
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("inspect path %q: %w", requested, err)
		}
		parentPath := filepath.Dir(existingPath)
		if parentPath == existingPath {
			return "", fmt.Errorf("resolve writable path %q: no existing parent", requested)
		}
		existingPath = parentPath
	}
	resolvedExistingPath, err := filepath.EvalSymlinks(existingPath)
	if err != nil {
		return "", fmt.Errorf("resolve writable path %q: %w", requested, err)
	}
	existingPathFromRoot, err := filepath.Rel(rootPath, resolvedExistingPath)
	if err != nil {
		return "", fmt.Errorf("compare path %q with root %q: %w", resolvedExistingPath, rootPath, err)
	}
	if existingPathFromRoot == ".." || strings.HasPrefix(existingPathFromRoot, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("resolve writable path %q: path escapes root", requested)
	}
	missingPath, err := filepath.Rel(existingPath, targetPath)
	if err != nil {
		return "", fmt.Errorf("resolve writable path %q: %w", requested, err)
	}
	return filepath.Join(resolvedExistingPath, missingPath), nil
}
