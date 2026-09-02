//go:build linux

package sqlite

import (
	"errors"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

type stateMigrationTarget struct {
	path       string
	name       string
	file       *os.File
	parent     *os.File
	fileStat   unix.Stat_t
	parentStat unix.Stat_t
}

func openStateMigrationTarget(path string, expectedOwner uint32) (*stateMigrationTarget, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || path == "/" {
		return nil, errors.New("sqlite.state_migration_database_invalid")
	}
	parentPath := filepath.Dir(path)
	parentFD, err := unix.Openat2(unix.AT_FDCWD, parentPath, &unix.OpenHow{
		Flags:   unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK,
		Resolve: unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS,
	})
	if err != nil {
		return nil, errors.New("sqlite.state_migration_directory_invalid")
	}
	parent := os.NewFile(uintptr(parentFD), "orquesta-state-migration-parent")
	closeParent := true
	defer func() {
		if closeParent {
			_ = parent.Close()
		}
	}()
	var parentStat unix.Stat_t
	if unix.Fstat(parentFD, &parentStat) != nil || parentStat.Mode&unix.S_IFMT != unix.S_IFDIR ||
		parentStat.Uid != expectedOwner || parentStat.Mode&0o7777 != 0o700 {
		return nil, errors.New("sqlite.state_migration_directory_invalid")
	}

	name := filepath.Base(path)
	fileFD, err := unix.Openat2(parentFD, name, &unix.OpenHow{
		Flags:   unix.O_RDWR | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK,
		Resolve: unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS,
	})
	if err != nil {
		return nil, errors.New("sqlite.state_migration_database_invalid")
	}
	file := os.NewFile(uintptr(fileFD), "orquesta-state-migration-database")
	var fileStat unix.Stat_t
	if unix.Fstat(fileFD, &fileStat) != nil || fileStat.Mode&unix.S_IFMT != unix.S_IFREG ||
		fileStat.Uid != expectedOwner || fileStat.Nlink != 1 || fileStat.Mode&0o7777 != 0o600 {
		_ = file.Close()
		return nil, errors.New("sqlite.state_migration_database_invalid")
	}
	closeParent = false
	return &stateMigrationTarget{
		path: path, name: name, file: file, parent: parent,
		fileStat: fileStat, parentStat: parentStat,
	}, nil
}

func (target *stateMigrationTarget) verify() error {
	if target == nil || target.file == nil || target.parent == nil {
		return errors.New("sqlite.state_migration_target_closed")
	}
	var fileStat, parentStat unix.Stat_t
	if unix.Fstat(int(target.file.Fd()), &fileStat) != nil ||
		unix.Fstat(int(target.parent.Fd()), &parentStat) != nil ||
		!sameStateMigrationFile(fileStat, target.fileStat) ||
		!sameStateMigrationDirectory(parentStat, target.parentStat) {
		return errors.New("sqlite.state_migration_target_changed")
	}
	parentFD, err := unix.Openat2(unix.AT_FDCWD, filepath.Dir(target.path), &unix.OpenHow{
		Flags:   unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK,
		Resolve: unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS,
	})
	if err != nil {
		return errors.New("sqlite.state_migration_target_changed")
	}
	defer unix.Close(parentFD)
	if unix.Fstat(parentFD, &parentStat) != nil || !sameStateMigrationDirectory(parentStat, target.parentStat) {
		return errors.New("sqlite.state_migration_target_changed")
	}
	fileFD, err := unix.Openat2(parentFD, target.name, &unix.OpenHow{
		Flags:   unix.O_RDWR | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK,
		Resolve: unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS,
	})
	if err != nil {
		return errors.New("sqlite.state_migration_target_changed")
	}
	defer unix.Close(fileFD)
	if unix.Fstat(fileFD, &fileStat) != nil || !sameStateMigrationFile(fileStat, target.fileStat) {
		return errors.New("sqlite.state_migration_target_changed")
	}
	return nil
}

func (target *stateMigrationTarget) rejectSidecars() error {
	if target == nil || target.parent == nil {
		return errors.New("sqlite.state_migration_target_closed")
	}
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		var stat unix.Stat_t
		err := unix.Fstatat(int(target.parent.Fd()), target.name+suffix, &stat, unix.AT_SYMLINK_NOFOLLOW)
		if err == nil {
			return errors.New("sqlite.state_migration_sidecar_present")
		}
		if !errors.Is(err, unix.ENOENT) {
			return errors.New("sqlite.state_migration_sidecar_invalid")
		}
	}
	return nil
}

func (target *stateMigrationTarget) syncParent() error {
	if target == nil || target.parent == nil {
		return errors.New("sqlite.state_migration_target_closed")
	}
	return target.parent.Sync()
}

func (target *stateMigrationTarget) Close() error {
	if target == nil {
		return nil
	}
	var result error
	if target.file != nil {
		result = errors.Join(result, target.file.Close())
		target.file = nil
	}
	if target.parent != nil {
		result = errors.Join(result, target.parent.Close())
		target.parent = nil
	}
	return result
}

func sameStateMigrationFile(left, right unix.Stat_t) bool {
	return left.Dev == right.Dev && left.Ino == right.Ino && left.Uid == right.Uid &&
		left.Nlink == 1 && right.Nlink == 1 && left.Mode == right.Mode &&
		left.Mode&unix.S_IFMT == unix.S_IFREG && left.Mode&0o7777 == 0o600
}

func sameStateMigrationDirectory(left, right unix.Stat_t) bool {
	return left.Dev == right.Dev && left.Ino == right.Ino && left.Uid == right.Uid &&
		left.Mode == right.Mode && left.Mode&unix.S_IFMT == unix.S_IFDIR && left.Mode&0o7777 == 0o700
}
