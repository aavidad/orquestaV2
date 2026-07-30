// Este fichero recorre únicamente las raíces y clasifica candidatos de fuente.
package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func scanRoot(root string, excluded []string, timeout time.Duration, collected *collector) rootRecord {
	info, err := os.Lstat(root)
	if err != nil {
		collected.add(sealSource(sourceRecord{
			Path: root, Kind: "path", Status: "error", ErrorCode: filesystemErrorCode(err),
		}))
		return rootRecord{Path: root, Status: "error", ErrorCode: filesystemErrorCode(err)}
	}
	if info.Mode()&os.ModeSymlink != 0 {
		collected.add(sealSource(sourceRecord{
			Path: root, Kind: "symbolic_link", Status: "excluded", Reason: "symbolic_link_not_followed",
		}))
		return rootRecord{Path: root, Status: "excluded", ErrorCode: "root_symbolic_link"}
	}
	if isExactlyExcluded(excluded, root) {
		return rootRecord{Path: root, Status: "excluded", ErrorCode: "operator_excluded"}
	}
	if !info.IsDir() {
		if isBundleCandidate(root) {
			collected.add(inspectBundle(root, timeout))
			return rootRecord{Path: root, Status: "scanned"}
		}
		collected.add(sealSource(sourceRecord{
			Path: root, Kind: "path", Status: "excluded", Reason: "unsupported_root_kind",
		}))
		return rootRecord{Path: root, Status: "excluded", ErrorCode: "unsupported_root_kind"}
	}

	walkErr := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		path = filepath.Clean(path)
		if walkErr != nil {
			collected.add(sealSource(sourceRecord{
				Path: path, Kind: "path", Status: "error", ErrorCode: filesystemErrorCode(walkErr),
			}))
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if path != root && isExactlyExcluded(excluded, path) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			collected.add(sealSource(sourceRecord{
				Path: path, Kind: "path", Status: "error", ErrorCode: filesystemErrorCode(err),
			}))
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 {
			collected.add(sealSource(sourceRecord{
				Path: path, Kind: "symbolic_link", Status: "excluded", Reason: "symbolic_link_not_followed",
			}))
			return nil
		}
		if entry.IsDir() {
			if path != root && entry.Name() == ".git" {
				return filepath.SkipDir
			}
			candidate, bare := repositoryCandidate(path)
			if candidate {
				collected.add(inspectRepository(path, bare, timeout))
				if bare {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if isBundleCandidate(path) {
			collected.add(inspectBundle(path, timeout))
		}
		return nil
	})
	if walkErr != nil {
		collected.add(sealSource(sourceRecord{
			Path: root, Kind: "path", Status: "error", ErrorCode: filesystemErrorCode(walkErr),
		}))
		return rootRecord{Path: root, Status: "error", ErrorCode: filesystemErrorCode(walkErr)}
	}
	return rootRecord{Path: root, Status: "scanned"}
}

func repositoryCandidate(path string) (candidate bool, bare bool) {
	marker := filepath.Join(path, ".git")
	if _, err := os.Lstat(marker); err == nil || !errors.Is(err, fs.ErrNotExist) {
		return true, false
	}
	head, headErr := os.Lstat(filepath.Join(path, "HEAD"))
	objects, objectsErr := os.Lstat(filepath.Join(path, "objects"))
	refs, refsErr := os.Lstat(filepath.Join(path, "refs"))
	packedRefs, packedErr := os.Lstat(filepath.Join(path, "packed-refs"))
	if headErr == nil && head.Mode().IsRegular() &&
		objectsErr == nil && objects.IsDir() &&
		((refsErr == nil && refs.IsDir()) || (packedErr == nil && packedRefs.Mode().IsRegular())) {
		return true, true
	}
	return false, false
}

func normalizeUniquePaths(paths []string) ([]string, error) {
	seen := make(map[string]struct{}, len(paths))
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		if strings.TrimSpace(path) == "" {
			return nil, errors.New("la ruta no puede estar vacía")
		}
		absolute, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		absolute = filepath.Clean(absolute)
		if _, exists := seen[absolute]; exists {
			continue
		}
		seen[absolute] = struct{}{}
		result = append(result, absolute)
	}
	sort.Strings(result)
	return result, nil
}

func normalizeGitPath(repository, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(repository, path))
}

func withinAnyRoot(roots []string, path string) bool {
	for _, root := range roots {
		if pathWithin(root, path) {
			return true
		}
	}
	return false
}

func pathWithin(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return relative == "." ||
		(relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}

func isExactlyExcluded(excluded []string, path string) bool {
	index := sort.SearchStrings(excluded, path)
	return index < len(excluded) && excluded[index] == path
}

func isBundleCandidate(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".bundle")
}

func filesystemErrorCode(err error) string {
	switch {
	case errors.Is(err, fs.ErrPermission):
		return "permission_denied"
	case errors.Is(err, fs.ErrNotExist):
		return "path_not_found"
	default:
		return "filesystem_error"
	}
}
