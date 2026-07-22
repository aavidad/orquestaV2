//go:build linux

package filesystem

import (
	"os"
	"syscall"
)

type privateMetadataSnapshot struct {
	device, inode, links uint64
	mode, uid, gid       uint32
	size                 int64
	modified, changed    syscall.Timespec
}

func snapshotPrivateMetadata(info os.FileInfo) (privateMetadataSnapshot, bool) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return privateMetadataSnapshot{}, false
	}
	return privateMetadataSnapshot{uint64(stat.Dev), stat.Ino, uint64(stat.Nlink), stat.Mode,
		stat.Uid, stat.Gid, stat.Size, stat.Mtim, stat.Ctim}, true
}

func samePrivateMetadata(left, right os.FileInfo) bool {
	leftSnapshot, leftOK := snapshotPrivateMetadata(left)
	rightSnapshot, rightOK := snapshotPrivateMetadata(right)
	return leftOK && rightOK && leftSnapshot == rightSnapshot && os.SameFile(left, right)
}

func privateDirectoryValid(info os.FileInfo, euid int) bool {
	stat, ok := privateStat(info)
	return ok && stat.Uid == uint32(euid) && info.Mode() == os.ModeDir|0o700
}

func privateFileValid(info os.FileInfo, size int64, euid int) bool {
	stat, ok := privateStat(info)
	return ok && stat.Uid == uint32(euid) && info.Mode() == 0o600 && stat.Nlink == 1 && info.Size() == size
}

func privateStat(info os.FileInfo) (*syscall.Stat_t, bool) {
	if info == nil || info.Mode()&os.ModeSymlink != 0 {
		return nil, false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return stat, ok
}
