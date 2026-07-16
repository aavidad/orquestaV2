//go:build unix

package sqlite

import (
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func openLocalStateFile(path string) (*os.File, error) {
	descriptor, err := unix.Open(path, unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(descriptor), path), nil
}

func localFileIdentity(info os.FileInfo) (uint64, uint64, bool) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Ino == 0 {
		return 0, 0, false
	}
	return uint64(stat.Dev), uint64(stat.Ino), true
}
