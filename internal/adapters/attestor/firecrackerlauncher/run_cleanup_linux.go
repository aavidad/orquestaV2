//go:build linux

package firecrackerlauncher

import (
	"context"
	"errors"
	"io"
	"os"

	"golang.org/x/sys/unix"
)

func (workspace *runWorkspace) Cleanup() error {
	if workspace == nil {
		return nil
	}
	if workspace.cleanupTimeout <= 0 {
		return launcherError(CodeCleanupFailed)
	}
	ctx, cancel := context.WithTimeout(
		context.Background(),
		workspace.cleanupTimeout,
	)
	defer cancel()
	return workspace.CleanupContext(ctx)
}

func (workspace *runWorkspace) CleanupContext(ctx context.Context) error {
	if workspace == nil {
		return nil
	}
	if ctx == nil {
		return launcherError(CodeCleanupFailed)
	}
	workspace.cleanup.Do(func() {
		closeErr := closeFile(workspace.output)
		workspace.output = nil
		if workspace.directory != nil {
			closeErr = errors.Join(closeErr, workspace.directory.Close())
			workspace.directory = nil
		}
		removeErr := removeOwnedRun(
			ctx,
			workspace.runs,
			workspace.id,
			workspace.identity,
			workspace.marker,
			workspace.cleanupEntries,
			workspace.cleanupDepth,
		)
		if errors.Join(closeErr, removeErr) != nil {
			workspace.cleanupErr = launcherError(CodeCleanupFailed)
		}
	})
	return workspace.cleanupErr
}

func removeOwnedRun(
	ctx context.Context,
	runs *runsRoot,
	id string,
	identity descriptorIdentity,
	markerContent string,
	maxEntries, maxDepth uint32,
) error {
	if ctx.Err() != nil || runs == nil || runs.file == nil ||
		!validJailID(id) || maxEntries == 0 || maxDepth == 0 {
		return launcherError(CodeCleanupFailed)
	}
	fd, err := unix.Openat2(int(runs.file.Fd()), id, &unix.OpenHow{
		Flags: unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW,
		Resolve: unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS |
			unix.RESOLVE_NO_MAGICLINKS,
	})
	if err != nil {
		return launcherError(CodeCleanupFailed)
	}
	directory := os.NewFile(uintptr(fd), "orquesta-firecracker-cleanup-run")
	var stat unix.Stat_t
	if unix.Fstat(fd, &stat) != nil ||
		uint64(stat.Dev) != identity.device ||
		stat.Ino != identity.inode ||
		stat.Mode != identity.mode ||
		stat.Uid != runs.owner || stat.Gid != runs.owner ||
		!validRunMarker(directory, markerContent, runs.owner) {
		_ = directory.Close()
		return launcherError(CodeCleanupFailed)
	}
	removeErr := removeDirectoryContents(
		ctx,
		directory,
		maxEntries,
		maxDepth,
		runMarkerName,
	)
	if removeErr != nil || ctx.Err() != nil {
		_ = directory.Close()
		return launcherError(CodeCleanupFailed)
	}
	markerErr := unix.Unlinkat(int(directory.Fd()), runMarkerName, 0)
	closeErr := directory.Close()
	unlinkErr := unix.Unlinkat(int(runs.file.Fd()), id, unix.AT_REMOVEDIR)
	if errors.Join(markerErr, closeErr, unlinkErr) != nil {
		return launcherError(CodeCleanupFailed)
	}
	return nil
}

func validRunMarker(directory *os.File, want string, owner uint32) bool {
	fd, err := unix.Openat(
		int(directory.Fd()),
		runMarkerName,
		unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW,
		0,
	)
	if err != nil {
		return false
	}
	file := os.NewFile(uintptr(fd), "orquesta-firecracker-run-marker")
	defer file.Close()
	var stat unix.Stat_t
	if unix.Fstat(fd, &stat) != nil ||
		stat.Mode&unix.S_IFMT != unix.S_IFREG ||
		stat.Uid != owner || stat.Gid != owner ||
		stat.Nlink != 1 || stat.Mode&0o777 != 0o400 ||
		stat.Size != int64(len(want)) {
		return false
	}
	content, err := io.ReadAll(io.LimitReader(file, int64(len(want))+1))
	return err == nil && string(content) == want
}

type cleanupDirectoryFrame struct {
	directory *os.File
	name      string
	depth     uint32
	pending   []os.DirEntry
	next      int
}

func removeDirectoryContents(
	ctx context.Context,
	directory *os.File,
	maxEntries, maxDepth uint32,
	protectedRootLeaf string,
) error {
	if ctx == nil || directory == nil || maxEntries == 0 || maxDepth == 0 ||
		!safeLeafName(protectedRootLeaf) {
		return launcherError(CodeCleanupFailed)
	}
	stack := []cleanupDirectoryFrame{{directory: directory}}
	defer func() {
		for index := 1; index < len(stack); index++ {
			_ = stack[index].directory.Close()
		}
	}()
	var visited uint32
	for len(stack) > 0 {
		if ctx.Err() != nil {
			return launcherError(CodeCleanupFailed)
		}
		frame := &stack[len(stack)-1]
		if frame.next < len(frame.pending) {
			entry := frame.pending[frame.next]
			frame.next++
			if visited >= maxEntries {
				return launcherError(CodeCleanupFailed)
			}
			visited++
			name := entry.Name()
			if !safeLeafName(name) {
				return launcherError(CodeCleanupFailed)
			}
			if frame.depth == 0 && name == protectedRootLeaf {
				continue
			}
			var before unix.Stat_t
			if unix.Fstatat(
				int(frame.directory.Fd()),
				name,
				&before,
				unix.AT_SYMLINK_NOFOLLOW,
			) != nil {
				return launcherError(CodeCleanupFailed)
			}
			if before.Mode&unix.S_IFMT != unix.S_IFDIR {
				if unix.Unlinkat(int(frame.directory.Fd()), name, 0) != nil {
					return launcherError(CodeCleanupFailed)
				}
				continue
			}
			if frame.depth >= maxDepth {
				return launcherError(CodeCleanupFailed)
			}
			fd, err := unix.Openat(
				int(frame.directory.Fd()),
				name,
				unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW,
				0,
			)
			if err != nil {
				return launcherError(CodeCleanupFailed)
			}
			child := os.NewFile(uintptr(fd), "orquesta-firecracker-cleanup-directory")
			var after unix.Stat_t
			if unix.Fstat(fd, &after) != nil ||
				after.Mode&unix.S_IFMT != unix.S_IFDIR ||
				after.Dev != before.Dev || after.Ino != before.Ino {
				_ = child.Close()
				return launcherError(CodeCleanupFailed)
			}
			stack = append(stack, cleanupDirectoryFrame{
				directory: child,
				name:      name,
				depth:     frame.depth + 1,
			})
			continue
		}
		entries, err := frame.directory.ReadDir(32)
		if len(entries) > 0 {
			frame.pending = entries
			frame.next = 0
			continue
		}
		if !errors.Is(err, io.EOF) {
			return launcherError(CodeCleanupFailed)
		}
		if len(stack) == 1 {
			return nil
		}
		child := *frame
		stack = stack[:len(stack)-1]
		parent := &stack[len(stack)-1]
		if child.directory.Close() != nil ||
			unix.Unlinkat(
				int(parent.directory.Fd()),
				child.name,
				unix.AT_REMOVEDIR,
			) != nil {
			return launcherError(CodeCleanupFailed)
		}
	}
	return nil
}

func (runs *runsRoot) Close() error {
	if runs == nil || runs.file == nil {
		return nil
	}
	emptyErr := runs.verifyEmpty()
	return errors.Join(emptyErr, runs.file.Close())
}

func (runs *runsRoot) verifyEmpty() error {
	if _, err := runs.file.Seek(0, io.SeekStart); err != nil {
		return launcherError(CodeCleanupFailed)
	}
	entries, err := runs.file.ReadDir(1)
	if err != nil && !errors.Is(err, io.EOF) || len(entries) != 0 {
		return launcherError(CodeCleanupFailed)
	}
	return nil
}
