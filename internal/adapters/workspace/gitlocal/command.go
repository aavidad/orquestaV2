package gitlocal

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func (adapter *Adapter) gitRun(ctx context.Context, directory string, input []byte, args ...string) ([]byte, error) {
	if adapter == nil {
		return nil, &Error{Code: CodeUnavailable}
	}
	argv := gitArguments(directory, args...)
	command, cleanup, err := adapter.pinnedGitCommand(ctx, nil, argv...)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	command.Env = gitEnvironment(adapter.root)
	return adapter.runGitCommand(ctx, command, input)
}

func gitArguments(directory string, args ...string) []string {
	argv := []string{"--no-replace-objects",
		"-c", "core.hooksPath=/dev/null",
		"-c", "core.attributesFile=/dev/null",
		"-c", "core.sharedRepository=0600",
		"-c", "core.pager=cat",
		"-c", "credential.helper=",
		"-c", "core.fsmonitor=false",
		"-c", "diff.external=",
		"-c", "commit.gpgSign=false",
		"-c", "fetch.ifMissing=false"}
	if directory != "" {
		argv = append(argv, "-C", directory)
	}
	argv = append(argv, args...)
	return argv
}

func (adapter *Adapter) runGitCommand(ctx context.Context, command *exec.Cmd, input []byte) ([]byte, error) {
	limit := adapter.gitOutputLimit()
	if int64(len(input)) > limit {
		return nil, &Error{Code: CodeSnapshotLimit}
	}
	output := boundedGitOutput{remaining: limit}
	output.kill = func() { _ = killPinnedProcessGroup(command) }
	command.Stdin = bytes.NewReader(input)
	command.Stdout, command.Stderr = &output, &output
	err := runPinnedCommand(command)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if output.overflow {
		return nil, &Error{Code: CodeSnapshotLimit}
	}
	if err != nil {
		return nil, &Error{Code: CodeGitFailed, Cause: err}
	}
	return output.buffer.Bytes(), nil
}

func (adapter *Adapter) gitOutputLimit() int64 {
	limit := adapter.maxSnapshotBytes
	if limit > 0 && limit < maxSnapshotStreamFieldBytes {
		limit = maxSnapshotStreamFieldBytes
	}
	if limit < 1 || limit > maxSnapshotMetadataBytes {
		limit = maxSnapshotMetadataBytes
	}
	return limit
}

type boundedGitOutput struct {
	buffer    bytes.Buffer
	remaining int64
	overflow  bool
	kill      func()
}

func (output *boundedGitOutput) Write(value []byte) (int, error) {
	count := len(value)
	if int64(count) > output.remaining {
		value = value[:output.remaining]
		if !output.overflow && output.kill != nil {
			defer output.kill()
		}
		output.overflow = true
	}
	output.remaining -= int64(len(value))
	_, _ = output.buffer.Write(value)
	return count, nil
}

func gitEnvironment(root string) []string {
	return []string{
		"PATH=/usr/bin:/bin", "HOME=" + root, "LANG=C", "LC_ALL=C",
		"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_TERMINAL_PROMPT=0", "GIT_PAGER=cat", "GIT_EDITOR=/bin/false", "GIT_SEQUENCE_EDITOR=/bin/false",
		"GIT_ASKPASS=/bin/false", "GIT_SSH_COMMAND=/bin/false", "GIT_EXTERNAL_DIFF=", "GIT_OPTIONAL_LOCKS=0",
	}
}

func trimOID(output []byte) string { return strings.TrimSpace(string(output)) }

func (adapter *Adapter) gitOID(ctx context.Context, directory, ref string) (string, error) {
	output, err := adapter.gitRun(ctx, directory, nil, "rev-parse", "--verify", ref+"^{commit}")
	if err != nil {
		return "", err
	}
	return trimOID(output), nil
}

// gitRefOID distinguishes an absent idempotency marker (normal replay
// boundary) from a broken Git invocation. It never exposes Git output.
func (adapter *Adapter) gitRefOID(ctx context.Context, directory, ref string) (string, bool, error) {
	command, cleanup, err := adapter.pinnedGitCommand(ctx, nil, "-C", directory, "show-ref", "--verify", "--quiet", ref)
	if err != nil {
		return "", false, err
	}
	defer cleanup()
	command.Env = gitEnvironment(adapter.root)
	if err := runPinnedCommand(command); err != nil {
		if ctx.Err() != nil {
			return "", false, ctx.Err()
		}
		var exit interface{ ExitCode() int }
		if errors.As(err, &exit) && exit.ExitCode() == 1 {
			return "", false, nil
		}
		return "", false, &Error{Code: CodeGitFailed, Cause: err}
	}
	oid, err := adapter.gitOID(ctx, directory, ref)
	if err != nil {
		return "", false, err
	}
	return oid, true, nil
}

func privateRoot(value string) (string, error) {
	root, err := filepath.Abs(value)
	if err != nil {
		return "", &Error{Code: CodeConfigInvalid, Cause: err}
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return "", &Error{Code: CodeConfigInvalid, Cause: err}
	}
	if err := checkPrivateAncestry(root); err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Join(root, "workspaces"), 0o700); err != nil {
		return "", &Error{Code: CodeConfigInvalid, Cause: err}
	}
	if err := checkPrivateDirectory(filepath.Join(root, "workspaces")); err != nil {
		return "", err
	}
	return root, nil
}

func checkPrivateDirectory(path string) error {
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o077 != 0 || !ownedByCurrentUID(info) {
		return &Error{Code: CodeWorkspaceUnsafe, Cause: errUnsafeWorkspace}
	}
	return nil
}

func checkPrivateAncestry(path string) error {
	return checkOwnedAncestry(path, 0o077)
}

func checkRepositoryAncestry(path string) error {
	return checkOwnedAncestry(path, 0o022)
}

func checkOwnedAncestry(path string, leafForbidden os.FileMode) error {
	path = filepath.Clean(path)
	leaf := path
	type ancestor struct{ info os.FileInfo }
	var ancestors []ancestor
	for {
		info, err := os.Lstat(path)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || !ownedByCurrentOrRoot(info) {
			return &Error{Code: CodeWorkspaceUnsafe, Cause: errUnsafeWorkspace}
		}
		if path == leaf {
			if info.Mode().Perm()&leafForbidden != 0 || !ownedByCurrentUID(info) {
				return &Error{Code: CodeWorkspaceUnsafe, Cause: errUnsafeWorkspace}
			}
		} else {
			ancestors = append(ancestors, ancestor{info: info})
		}
		parent := filepath.Dir(path)
		if parent == path {
			break
		}
		path = parent
	}
	barrier := -1
	for index, value := range ancestors {
		if ownedByCurrentUID(value.info) && value.info.Mode().Perm()&0o077 == 0 {
			barrier = index
			break
		}
	}
	for index, value := range ancestors {
		if index < barrier && ownedByCurrentUID(value.info) {
			continue
		}
		if value.info.Mode().Perm()&0o022 != 0 && value.info.Mode()&os.ModeSticky == 0 {
			return &Error{Code: CodeWorkspaceUnsafe, Cause: errUnsafeWorkspace}
		}
	}
	return nil
}

func pathsOverlap(left, right string) bool {
	within := func(parent, child string) bool {
		relative, err := filepath.Rel(parent, child)
		return err == nil && (relative == "." ||
			(relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))))
	}
	return within(left, right) || within(right, left)
}

func ownedByCurrentUID(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && int(stat.Uid) == os.Getuid()
}

func ownedByCurrentOrRoot(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && (int(stat.Uid) == os.Getuid() || stat.Uid == 0)
}
