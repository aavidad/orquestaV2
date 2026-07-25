//go:build linux

package firecrackerlauncher

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

func OpenTrustedConfigFile(path string, owner uint32, maxBytes int64) (*os.File, error) {
	if maxBytes <= 0 || !canonicalAbsolute(path) ||
		validateTrustedAncestors(path, owner) != nil {
		return nil, launcherError(CodeConfigInvalid)
	}
	fd, err := unix.Openat2(unix.AT_FDCWD, path, &unix.OpenHow{
		Flags:   unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK,
		Resolve: unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS,
	})
	if err != nil {
		return nil, launcherError(CodeConfigInvalid)
	}
	file := os.NewFile(uintptr(fd), "orquesta-firecracker-launcher-config")
	var stat unix.Stat_t
	if unix.Fstat(fd, &stat) != nil || stat.Mode&unix.S_IFMT != unix.S_IFREG ||
		stat.Uid != owner || stat.Nlink != 1 || stat.Mode&0o022 != 0 ||
		stat.Size <= 0 || stat.Size > maxBytes {
		_ = file.Close()
		return nil, launcherError(CodeConfigInvalid)
	}
	return file, nil
}

func openTrustedRuntimeRoot(root string, owner, group uint32) (*os.File, error) {
	if !canonicalAbsolute(root) || validateTrustedAncestors(root, owner) != nil {
		return nil, launcherError(CodeRuntimeRootUnsafe)
	}
	fd, err := unix.Openat2(unix.AT_FDCWD, root, &unix.OpenHow{
		Flags:   unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK,
		Resolve: unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS,
	})
	if err != nil {
		return nil, launcherError(CodeRuntimeRootUnsafe)
	}
	directory := os.NewFile(uintptr(fd), "orquesta-firecracker-launcher-runtime-root")
	var stat unix.Stat_t
	if unix.Fstat(fd, &stat) != nil || stat.Mode&unix.S_IFMT != unix.S_IFDIR ||
		stat.Uid != owner || stat.Gid != group || stat.Mode&0o777 != 0o750 {
		_ = directory.Close()
		return nil, launcherError(CodeRuntimeRootUnsafe)
	}
	if err := validateRuntimeMarker(directory, owner); err != nil {
		_ = directory.Close()
		return nil, err
	}
	return directory, nil
}

func validateRuntimeMarker(directory *os.File, owner uint32) error {
	fd, err := unix.Openat2(int(directory.Fd()), runtimeMarkerName, &unix.OpenHow{
		Flags: unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK,
		Resolve: unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS |
			unix.RESOLVE_NO_MAGICLINKS,
	})
	if err != nil {
		return launcherError(CodeRuntimeRootUnsafe)
	}
	marker := os.NewFile(uintptr(fd), "orquesta-firecracker-launcher-runtime-marker")
	defer marker.Close()
	var stat unix.Stat_t
	if unix.Fstat(fd, &stat) != nil || stat.Mode&unix.S_IFMT != unix.S_IFREG ||
		stat.Uid != owner || stat.Nlink != 1 || stat.Mode&0o777 != 0o400 ||
		stat.Size != int64(len(runtimeMarkerContent)) {
		return launcherError(CodeRuntimeRootUnsafe)
	}
	content, err := io.ReadAll(io.LimitReader(marker, int64(len(runtimeMarkerContent))+1))
	if err != nil || string(content) != runtimeMarkerContent {
		return launcherError(CodeRuntimeRootUnsafe)
	}
	return nil
}

func validateTrustedAncestors(path string, owner uint32) error {
	if !canonicalAbsolute(path) || path == "/" {
		return launcherError(CodeRuntimeRootUnsafe)
	}
	parent := filepath.Dir(path)
	prefixes := []string{"/"}
	current := "/"
	for _, part := range strings.Split(strings.TrimPrefix(parent, "/"), "/") {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)
		prefixes = append(prefixes, current)
	}
	for _, prefix := range prefixes {
		fd, err := unix.Openat2(unix.AT_FDCWD, prefix, &unix.OpenHow{
			Flags:   unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK,
			Resolve: unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS,
		})
		if err != nil {
			return launcherError(CodeRuntimeRootUnsafe)
		}
		var stat unix.Stat_t
		statErr := unix.Fstat(fd, &stat)
		_ = unix.Close(fd)
		if statErr != nil || !trustedAncestorMetadata(stat, owner) {
			return launcherError(CodeRuntimeRootUnsafe)
		}
	}
	return nil
}

func trustedAncestorMetadata(stat unix.Stat_t, owner uint32) bool {
	if stat.Mode&unix.S_IFMT != unix.S_IFDIR ||
		stat.Uid != 0 && stat.Uid != owner {
		return false
	}
	if stat.Mode&0o022 == 0 {
		return true
	}
	return stat.Uid == 0 && stat.Mode&unix.S_ISVTX != 0
}
