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

	rootPath, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve root %q: %w", root, err)
	}
	rootInputPath := rootPath
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

	requestedPath := requested
	if filepath.IsAbs(requestedPath) {
		pathFromRoot, inside, err := relativePathInside(rootInputPath, requestedPath)
		if err != nil {
			return "", fmt.Errorf("compare path %q with root %q: %w", requestedPath, rootInputPath, err)
		}
		if !inside {
			pathFromRoot, inside, err = relativePathInside(rootPath, requestedPath)
			if err != nil {
				return "", fmt.Errorf("compare path %q with root %q: %w", requestedPath, rootPath, err)
			}
		}
		if !inside {
			return "", fmt.Errorf("resolve path %q: path escapes root", requested)
		}
		requestedPath = filepath.Join(rootPath, pathFromRoot)
	} else {
		requestedPath = filepath.Join(rootPath, requestedPath)
	}
	resolvedPath, err := filepath.EvalSymlinks(requestedPath)
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

	rootPath, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve root %q: %w", root, err)
	}
	rootInputPath := rootPath
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
	targetPath := requested
	if filepath.IsAbs(targetPath) {
		pathFromRoot, inside, err := relativePathInside(rootInputPath, targetPath)
		if err != nil {
			return "", fmt.Errorf("compare path %q with root %q: %w", targetPath, rootInputPath, err)
		}
		if !inside {
			pathFromRoot, inside, err = relativePathInside(rootPath, targetPath)
			if err != nil {
				return "", fmt.Errorf("compare path %q with root %q: %w", targetPath, rootPath, err)
			}
		}
		if !inside {
			return "", fmt.Errorf("resolve writable path %q: path escapes root", requested)
		}
		targetPath = filepath.Join(rootPath, pathFromRoot)
	} else {
		targetPath = filepath.Join(rootPath, targetPath)
	}
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

func relativePathInside(root, target string) (string, bool, error) {
	pathFromRoot, err := filepath.Rel(root, target)
	if err != nil {
		return "", false, err
	}
	inside := pathFromRoot != ".." && !strings.HasPrefix(pathFromRoot, ".."+string(filepath.Separator))
	return pathFromRoot, inside, nil
}
