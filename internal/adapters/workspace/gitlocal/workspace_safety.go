package gitlocal

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"orquesta/internal/ports"
)

const maxGitControlSize = 4096

func readGitControl(workspace string) (string, error) {
	value, err := readSafeGitControlFile(filepath.Join(workspace, ".git"))
	if err != nil {
		return "", err
	}
	text := string(value)
	if strings.ContainsAny(text, "\x00\r") || strings.Count(text, "\n") > 1 {
		return "", &Error{Code: CodeWorkspaceUnsafe}
	}
	text = strings.TrimSuffix(text, "\n")
	if !strings.HasPrefix(text, "gitdir: ") {
		return "", &Error{Code: CodeWorkspaceUnsafe}
	}
	gitDir := strings.TrimPrefix(text, "gitdir: ")
	if gitDir == "" || strings.TrimSpace(gitDir) != gitDir {
		return "", &Error{Code: CodeWorkspaceUnsafe}
	}
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(workspace, gitDir)
	}
	return filepath.Clean(gitDir), nil
}

func readSafeGitControlFile(path string) ([]byte, error) {
	before, err := os.Lstat(path)
	if err != nil || !safeControlInfo(before) {
		return nil, &Error{Code: CodeWorkspaceUnsafe, Cause: errUnsafeWorkspace}
	}
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, &Error{Code: CodeWorkspaceUnsafe, Cause: err}
	}
	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		_ = syscall.Close(fd)
		return nil, &Error{Code: CodeWorkspaceUnsafe}
	}
	defer file.Close()
	after, err := file.Stat()
	if err != nil || !safeControlInfo(after) || !os.SameFile(before, after) {
		return nil, &Error{Code: CodeWorkspaceUnsafe, Cause: errUnsafeWorkspace}
	}
	value, err := io.ReadAll(io.LimitReader(file, maxGitControlSize+1))
	if err != nil || len(value) == 0 || len(value) > maxGitControlSize || int64(len(value)) != after.Size() {
		return nil, &Error{Code: CodeWorkspaceUnsafe, Cause: errUnsafeWorkspace}
	}
	return value, nil
}

func hardenGitControl(workspace string) error {
	path := filepath.Join(workspace, ".git")
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return &Error{Code: CodeWorkspaceUnsafe, Cause: err}
	}
	defer syscall.Close(fd)
	var stat syscall.Stat_t
	if err := syscall.Fstat(fd, &stat); err != nil || stat.Mode&syscall.S_IFMT != syscall.S_IFREG ||
		int(stat.Uid) != os.Getuid() || stat.Nlink != 1 || stat.Size <= 0 || stat.Size > maxGitControlSize {
		return &Error{Code: CodeWorkspaceUnsafe, Cause: errUnsafeWorkspace}
	}
	if err := syscall.Fchmod(fd, 0o600); err != nil {
		return &Error{Code: CodeWorkspaceUnsafe, Cause: err}
	}
	return nil
}

func (adapter *Adapter) hardenCreatedWorkspaceGitMetadata(ctx context.Context, repository, workspace string) error {
	if err := hardenGitControl(workspace); err != nil {
		return err
	}
	commonRaw, err := adapter.gitRun(ctx, repository, nil, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return err
	}
	common, err := safeGitDirectory(trimOID(commonRaw))
	if err != nil {
		return err
	}
	gitDir, err := readGitControl(workspace)
	if err != nil || filepath.Dir(gitDir) != filepath.Join(common, "worktrees") {
		return &Error{Code: CodeWorkspaceUnsafe, Cause: errUnsafeWorkspace}
	}
	if _, err := safeGitDirectory(filepath.Dir(gitDir)); err != nil {
		return err
	}
	return hardenOwnedDirectory(gitDir)
}

func (adapter *Adapter) validateGitMetadataRoot(ctx context.Context, repository string) error {
	commonRaw, err := adapter.gitRun(ctx, repository, nil, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return err
	}
	common, err := safeGitDirectory(trimOID(commonRaw))
	if err != nil {
		return err
	}
	worktrees := filepath.Join(common, "worktrees")
	if _, err := os.Lstat(worktrees); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return &Error{Code: CodeWorkspaceUnsafe, Cause: err}
	}
	_, err = safeGitDirectory(worktrees)
	return err
}

func hardenOwnedDirectory(path string) error {
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW|syscall.O_DIRECTORY, 0)
	if err != nil {
		return &Error{Code: CodeWorkspaceUnsafe, Cause: err}
	}
	defer syscall.Close(fd)
	var stat syscall.Stat_t
	if err := syscall.Fstat(fd, &stat); err != nil || stat.Mode&syscall.S_IFMT != syscall.S_IFDIR ||
		int(stat.Uid) != os.Getuid() {
		return &Error{Code: CodeWorkspaceUnsafe, Cause: errUnsafeWorkspace}
	}
	if err := syscall.Fchmod(fd, 0o700); err != nil {
		return &Error{Code: CodeWorkspaceUnsafe, Cause: err}
	}
	return nil
}

func safeControlInfo(info os.FileInfo) bool {
	if info == nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 ||
		info.Mode().Perm()&0o022 != 0 || info.Size() <= 0 || info.Size() > maxGitControlSize {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && int(stat.Uid) == os.Getuid() && stat.Nlink == 1
}

func (adapter *Adapter) verifyWorkspace(
	ctx context.Context,
	repository, path, base string,
	ref ports.ExecutionWorkspaceRef,
) (string, error) {
	if err := ensureSafeNewWorkspacePath(adapter.root, path); err != nil {
		return "", err
	}
	if info, err := os.Lstat(path); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", &Error{Code: CodeWorkspaceUnsafe, Cause: errUnsafeWorkspace}
	}
	if err := hardenGitControl(path); err != nil {
		return "", err
	}
	gitDir, err := adapter.verifyWorkspaceGitBinding(ctx, repository, path)
	if err != nil {
		return "", err
	}
	if actual, err := adapter.gitOID(ctx, path, "HEAD"); err != nil || (base != "" && actual != base) {
		return "", &Error{Code: CodeBaseStale, Cause: err}
	}
	branch, err := adapter.gitRun(ctx, path, nil, "rev-parse", "--symbolic-full-name", "HEAD")
	if err != nil || trimOID(branch) != "refs/heads/"+workspaceBranch(ref) {
		return "", &Error{Code: CodeWorkspaceUnsafe, Cause: errUnsafeWorkspace}
	}
	return gitDir, nil
}

func (adapter *Adapter) verifyWorkspaceGitBinding(ctx context.Context, repository, workspace string) (string, error) {
	commonRaw, err := adapter.gitRun(ctx, repository, nil, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return "", err
	}
	common, err := safeGitDirectory(trimOID(commonRaw))
	if err != nil {
		return "", err
	}
	gitDir, err := readGitControl(workspace)
	if err != nil {
		return "", err
	}
	gitDir, err = safeGitDirectory(gitDir)
	if err != nil || filepath.Dir(gitDir) != filepath.Join(common, "worktrees") {
		return "", &Error{Code: CodeWorkspaceUnsafe, Cause: errUnsafeWorkspace}
	}
	actualGit, err := adapter.workspaceGitPath(ctx, workspace, "--git-dir")
	if err != nil || actualGit != gitDir {
		return "", &Error{Code: CodeWorkspaceUnsafe, Cause: errUnsafeWorkspace}
	}
	actualCommon, err := adapter.workspaceGitPath(ctx, workspace, "--git-common-dir")
	if err != nil || actualCommon != common {
		return "", &Error{Code: CodeWorkspaceUnsafe, Cause: errUnsafeWorkspace}
	}
	return gitDir, nil
}

func (adapter *Adapter) workspaceGitPath(ctx context.Context, workspace, selector string) (string, error) {
	value, err := adapter.gitRun(ctx, workspace, nil, "rev-parse", "--path-format=absolute", selector)
	if err != nil {
		return "", err
	}
	return safeGitDirectory(trimOID(value))
}

func safeGitDirectory(value string) (string, error) {
	if value == "" || !filepath.IsAbs(value) {
		return "", &Error{Code: CodeWorkspaceUnsafe}
	}
	clean := filepath.Clean(value)
	if err := checkGitDirectoryAncestry(clean); err != nil {
		return "", err
	}
	return clean, nil
}

func checkGitDirectoryAncestry(path string) error {
	leaf := filepath.Clean(path)
	for current := leaf; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 ||
			!ownedByCurrentOrRoot(info) || (current == leaf && !ownedByCurrentUID(info)) {
			return &Error{Code: CodeWorkspaceUnsafe, Cause: errUnsafeWorkspace}
		}
		if info.Mode().Perm()&0o022 != 0 && (current == leaf || info.Mode()&os.ModeSticky == 0) {
			return &Error{Code: CodeWorkspaceUnsafe, Cause: errUnsafeWorkspace}
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
	}
}
