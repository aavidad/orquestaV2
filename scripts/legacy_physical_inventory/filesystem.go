// Este fichero reúne primitivas pequeñas de filesystem que no deciden el recorrido.
package main

import (
	"errors"
	"golang.org/x/sys/unix"
	"io"
	"os"
)

func appendPart(parent []string, name string) []string {
	result := make([]string, len(parent)+1)
	copy(result, parent)
	result[len(parent)] = name
	return result
}
func boundedChildPath(
	parent []string,
	parentBytes int64,
	name string,
	budget budgetOptions,
) ([]string, int64, string) {
	if len(parent)+1 > budget.maxDepth {
		return nil, parentBytes, "depth_limit"
	}
	pathBytes := parentBytes + int64(len(name))
	if len(parent) != 0 {
		pathBytes++
	}
	if pathBytes > budget.maxPathBytes {
		return nil, parentBytes, "path_bytes_limit"
	}
	return appendPart(parent, name), pathBytes, ""
}
func readDirectoryEntries(fd int, limit int64) ([]os.DirEntry, error) {
	duplicate, err := unix.Dup(fd)
	if err != nil {
		return nil, err
	}
	directory := os.NewFile(uintptr(duplicate), "directorio-confinado")
	entries, readErr := directory.ReadDir(int(limit) + 1)
	if errors.Is(readErr, io.EOF) {
		readErr = nil
	}
	closeErr := directory.Close()
	if readErr != nil || closeErr != nil {
		return nil, errors.Join(readErr, closeErr)
	}
	if int64(len(entries)) > limit {
		return entries, errDirectoryBudget
	}
	return entries, nil
}
func openBeneath(parentFD int, name string, flags int) (int, error) {
	how := &unix.OpenHow{
		Flags: uint64(flags | unix.O_CLOEXEC),
		Resolve: unix.RESOLVE_BENEATH | unix.RESOLVE_NO_MAGICLINKS |
			unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_XDEV,
	}
	return unix.Openat2(parentFD, name, how)
}
func verifyEntrySnapshot(
	parentFD int,
	name string,
	expectedMountID uint64,
	before *unix.Stat_t,
) error {
	mountID, err := mountIDAt(parentFD, name, unix.AT_SYMLINK_NOFOLLOW)
	if err != nil {
		return err
	}
	if mountID != expectedMountID {
		return unix.EXDEV
	}
	var after unix.Stat_t
	if err := unix.Fstatat(parentFD, name, &after, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return err
	}
	if !sameSnapshot(before, &after) {
		return errChangedDuringScan
	}
	return nil
}
func denied(parts []string, rules [][]string) bool {
	for _, rule := range rules {
		if len(rule) > len(parts) {
			continue
		}
		match := true
		for index := range rule {
			match = match && rule[index] == parts[index]
		}
		if match {
			return true
		}
	}
	return false
}
func specialType(mode uint32) string {
	switch mode & unix.S_IFMT {
	case unix.S_IFIFO:
		return "fifo"
	case unix.S_IFSOCK:
		return "socket"
	case unix.S_IFCHR:
		return "character_device"
	case unix.S_IFBLK:
		return "block_device"
	default:
		return "unknown"
	}
}
func errorsIs(err, target error) bool { return errors.Is(err, target) }
