package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	orquestaserver "orquesta/modulos/orquesta-server"
)

const serverProjectSecretFileMaxBytesV0 = 64 * 1024

func readServerProjectSecretFileV0(config orquestaserver.ConfigV0, ref string) (string, error) {
	parts, err := serverProjectSecretFilePartsV0(ref)
	if err != nil {
		return "", err
	}
	root, err := openServerProjectSecretRootV0(config.ProjectWorkDir)
	if err != nil {
		return "", err
	}
	defer root.Close()

	for index := range parts[:len(parts)-1] {
		name := filepath.Join(parts[:index+1]...)
		if err := validateServerProjectSecretIntermediateV0(root, name); err != nil {
			return "", err
		}
	}
	name := filepath.Join(parts...)
	before, err := root.Lstat(name)
	if err != nil {
		return "", fmt.Errorf("secret_file_unavailable")
	}
	if err := validateServerProjectSecretFileInfoV0(before); err != nil {
		return "", err
	}
	file, err := root.Open(name)
	if err != nil {
		return "", fmt.Errorf("secret_file_unavailable")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("secret_file_unavailable")
	}
	after, err := root.Lstat(name)
	if err != nil || !os.SameFile(info, after) || !os.SameFile(before, info) {
		return "", fmt.Errorf("secret_file_changed")
	}
	if err := validateServerProjectSecretFileInfoV0(info); err != nil {
		return "", err
	}
	raw, err := io.ReadAll(io.LimitReader(file, serverProjectSecretFileMaxBytesV0+1))
	if err != nil {
		return "", fmt.Errorf("secret_file_unavailable")
	}
	if len(raw) > serverProjectSecretFileMaxBytesV0 {
		return "", fmt.Errorf("secret_file_size_invalid")
	}
	if token := strings.TrimSpace(string(raw)); token != "" {
		return token, nil
	}
	return "", fmt.Errorf("secret_file_empty")
}

func serverProjectSecretFilePartsV0(ref string) ([]string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" || filepath.IsAbs(ref) {
		return nil, fmt.Errorf("secret_file_path_invalid")
	}
	clean := filepath.Clean(ref)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("secret_file_path_outside_project")
	}
	parts := strings.Split(clean, string(filepath.Separator))
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return nil, fmt.Errorf("secret_file_path_outside_project")
		}
	}
	return parts, nil
}

func openServerProjectSecretRootV0(projectWorkDir string) (*os.Root, error) {
	projectWorkDir = strings.TrimSpace(projectWorkDir)
	if projectWorkDir == "" {
		return nil, fmt.Errorf("secret_file_root_required")
	}
	rootPath, err := filepath.Abs(projectWorkDir)
	if err != nil {
		return nil, fmt.Errorf("secret_file_root_invalid")
	}
	canonical, err := filepath.EvalSymlinks(rootPath)
	if err != nil || filepath.Clean(canonical) != filepath.Clean(rootPath) {
		return nil, fmt.Errorf("secret_file_root_symlink_forbidden")
	}
	before, err := os.Lstat(rootPath)
	if err != nil {
		return nil, fmt.Errorf("secret_file_root_invalid")
	}
	if err := validateServerProjectSecretRootInfoV0(before); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return nil, fmt.Errorf("secret_file_root_invalid")
	}
	info, err := root.Stat(".")
	if err != nil || !os.SameFile(before, info) {
		root.Close()
		return nil, fmt.Errorf("secret_file_root_changed")
	}
	if err := validateServerProjectSecretRootInfoV0(info); err != nil {
		root.Close()
		return nil, err
	}
	return root, nil
}

func validateServerProjectSecretIntermediateV0(root *os.Root, name string) error {
	before, err := root.Lstat(name)
	if err != nil {
		return fmt.Errorf("secret_file_unavailable")
	}
	if before.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("secret_file_symlink_forbidden")
	}
	if err := validateServerProjectSecretOwnerOnlyInfoV0(before, true); err != nil {
		return err
	}
	directory, err := root.Open(name)
	if err != nil {
		return fmt.Errorf("secret_file_unavailable")
	}
	defer directory.Close()
	info, err := directory.Stat()
	if err != nil || !os.SameFile(before, info) {
		return fmt.Errorf("secret_file_changed")
	}
	return validateServerProjectSecretOwnerOnlyInfoV0(info, true)
}

func validateServerProjectSecretRootInfoV0(info os.FileInfo) error {
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("secret_file_root_symlink_forbidden")
	}
	return validateServerProjectSecretOwnerOnlyInfoV0(info, true)
}

func validateServerProjectSecretFileInfoV0(info os.FileInfo) error {
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("secret_file_symlink_forbidden")
	}
	return validateServerProjectSecretOwnerOnlyInfoV0(info, false)
}

func validateServerProjectSecretOwnerOnlyInfoV0(info os.FileInfo, directory bool) error {
	if directory {
		if !info.IsDir() {
			return fmt.Errorf("secret_file_parent_invalid")
		}
	} else if !info.Mode().IsRegular() {
		return fmt.Errorf("secret_file_not_regular")
	}
	owner, ok := serverProjectSecretFileOwnerUIDV0(info)
	if !ok || owner != os.Geteuid() {
		return fmt.Errorf("secret_file_owner_invalid: owner=%d expected=%d", owner, os.Geteuid())
	}
	if info.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("secret_file_permissions_insecure: mode=%04o", info.Mode().Perm())
	}
	if !directory && info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("secret_file_permissions_insecure: mode=%04o", info.Mode().Perm())
	}
	return nil
}
