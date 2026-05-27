package main

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

func validateGuardianArtifactPathV0(path string) error {
	if filepath.Clean(path) != path {
		return guardianArtifactPolicyErrorV0{Code: "guardian_artifact_path_not_clean", Err: nil}
	}
	if !filepath.IsAbs(path) {
		return guardianArtifactPolicyErrorV0{Code: "guardian_artifact_path_not_absolute", Err: nil}
	}
	return nil
}

func validateGuardianArtifactRootV0(config guardianConfigV0, path string) error {
	rootValue := strings.TrimSpace(config.ArtifactRoot)
	if rootValue == "" {
		return nil
	}
	root := filepath.Clean(rootValue)
	rel, err := filepath.Rel(root, filepath.Clean(path))
	if err != nil {
		return guardianArtifactPolicyErrorV0{Code: "guardian_artifact_root_invalid", Err: err}
	}
	if rel == "." {
		return nil
	}
	if strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || rel == ".." || filepath.IsAbs(rel) {
		return guardianArtifactPolicyErrorV0{Code: "guardian_artifact_outside_root", Err: nil}
	}
	return nil
}

func rejectGuardianSymlinkPathV0(path string) error {
	path = filepath.Clean(path)
	volume := filepath.VolumeName(path)
	rest := path[len(volume):]
	current := volume
	if filepath.IsAbs(path) {
		current += string(os.PathSeparator)
		rest = rest[1:]
	}
	for _, part := range splitGuardianPathPartsV0(rest) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return guardianArtifactPolicyErrorV0{Code: "guardian_artifact_symlink_path_blocked", Err: nil}
		}
	}
	return nil
}

func splitGuardianPathPartsV0(path string) []string {
	parts := []string{}
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

func hasUnsafeHardlinksV0(info os.FileInfo) bool {
	if runtime.GOOS == "windows" {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Nlink > 1
}
