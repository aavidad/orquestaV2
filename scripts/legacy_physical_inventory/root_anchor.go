// Este fichero ancla raíces autorizadas y exige lectura físicamente no mutante.
package main

import (
	"errors"
	"golang.org/x/sys/unix"
	"os"
	"time"
)

type anchoredRoot struct {
	rootOption
	file           *os.File
	mountID        uint64
	directoryFlags int
	fileFlags      int
}

func anchorRootsBefore(roots []rootOption, deadline time.Time) ([]anchoredRoot, error) {
	result := make([]anchoredRoot, 0, len(roots))
	for _, root := range roots {
		if !deadline.IsZero() && !time.Now().Before(deadline) {
			return nil, errors.Join(errBudget, closeAnchoredRoots(result))
		}
		anchored, err := anchorRoot(root)
		if err != nil {
			return nil, errors.Join(err, closeAnchoredRoots(result))
		}
		result = append(result, anchored)
	}
	if !deadline.IsZero() && !time.Now().Before(deadline) {
		return nil, errors.Join(errBudget, closeAnchoredRoots(result))
	}
	return result, nil
}
func anchorRoot(root rootOption) (anchoredRoot, error) {
	pathFD, err := unix.Openat2(unix.AT_FDCWD, root.path, &unix.OpenHow{
		Flags:   uint64(unix.O_PATH | unix.O_DIRECTORY | unix.O_CLOEXEC),
		Resolve: unix.RESOLVE_NO_MAGICLINKS | unix.RESOLVE_NO_SYMLINKS,
	})
	if err != nil {
		return anchoredRoot{}, err
	}
	closePath := func(cause error) error {
		return errors.Join(cause, unix.Close(pathFD))
	}
	var anchoredStat unix.Stat_t
	if err := unix.Fstat(pathFD, &anchoredStat); err != nil {
		return anchoredRoot{}, closePath(err)
	}
	if root.afterAnchor != nil {
		root.afterAnchor()
	}
	var filesystem unix.Statfs_t
	if err := unix.Fstatfs(pathFD, &filesystem); err != nil {
		return anchoredRoot{}, closePath(err)
	}
	noAtimeRequired := filesystem.Flags&unix.ST_RDONLY == 0 &&
		filesystem.Flags&unix.ST_NOATIME == 0
	directoryFlags := unix.O_RDONLY | unix.O_DIRECTORY
	fileFlags := unix.O_RDONLY
	if noAtimeRequired {
		directoryFlags |= unix.O_NOATIME
		fileFlags |= unix.O_NOATIME
	}
	scanFD, err := openBeneath(pathFD, ".", directoryFlags)
	if err != nil {
		if noAtimeRequired && (errors.Is(err, unix.EPERM) || errors.Is(err, unix.EACCES)) {
			return anchoredRoot{}, closePath(errNoAtime)
		}
		return anchoredRoot{}, closePath(err)
	}
	var scanStat unix.Stat_t
	if err := unix.Fstat(scanFD, &scanStat); err != nil {
		return anchoredRoot{}, errors.Join(err, unix.Close(scanFD), closePath(nil))
	}
	if !sameSnapshot(&anchoredStat, &scanStat) {
		return anchoredRoot{}, errors.Join(
			errChangedDuringScan, unix.Close(scanFD), closePath(nil),
		)
	}
	mountID, err := mountIDAt(scanFD, "", unix.AT_EMPTY_PATH)
	if err != nil {
		return anchoredRoot{}, errors.Join(err, unix.Close(scanFD), closePath(nil))
	}
	if err := unix.Close(pathFD); err != nil {
		return anchoredRoot{}, errors.Join(err, unix.Close(scanFD))
	}
	return anchoredRoot{
		rootOption: root,
		file:       os.NewFile(uintptr(scanFD), "raiz-confinada"),
		mountID:    mountID, directoryFlags: directoryFlags, fileFlags: fileFlags,
	}, nil
}
func mountIDAt(directoryFD int, name string, flags int) (uint64, error) {
	var stat unix.Statx_t
	if err := unix.Statx(
		directoryFD,
		name,
		flags|unix.AT_NO_AUTOMOUNT,
		unix.STATX_MNT_ID,
		&stat,
	); err != nil {
		return 0, err
	}
	if stat.Mask&unix.STATX_MNT_ID == 0 {
		return 0, errMountIDUnavailable
	}
	return stat.Mnt_id, nil
}
func closeAnchoredRoots(roots []anchoredRoot) error {
	var failures []error
	for index := range roots {
		if roots[index].file != nil {
			failures = appendIfError(failures, roots[index].file.Close())
		}
	}
	return errors.Join(failures...)
}
