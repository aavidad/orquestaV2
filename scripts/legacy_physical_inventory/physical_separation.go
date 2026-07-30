// Este fichero demuestra separación física con identidades del espacio de montajes.
package main

import (
	"errors"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const mountInfoLimit = int64(8 << 20)

type physicalLocation struct {
	device uint64
	path   string
}
type physicalLocationReader func(fd int, lexicalPath string) (physicalLocation, error)

var mountPathUnescaper = strings.NewReplacer(
	`\040`, " ", `\011`, "\t", `\012`, "\n", `\134`, `\`,
)

func newPhysicalLocationReader() (physicalLocationReader, error) {
	file, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return nil, err
	}
	content, readErr := readLimited(file, mountInfoLimit)
	closeErr := file.Close()
	if readErr != nil || closeErr != nil {
		return nil, errors.Join(readErr, closeErr)
	}
	mounts := make(map[uint64][2]string)
	for _, line := range strings.Split(string(content), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		id, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			return nil, err
		}
		mounts[id] = [2]string{mountPathUnescaper.Replace(fields[3]), mountPathUnescaper.Replace(fields[4])}
	}
	return func(fd int, lexicalPath string) (physicalLocation, error) {
		id, err := mountIDAt(fd, "", unix.AT_EMPTY_PATH)
		info, ok := mounts[id]
		if err != nil || !ok {
			return physicalLocation{}, errors.Join(errMountIDUnavailable, err)
		}
		reopened, err := unix.Openat2(unix.AT_FDCWD, lexicalPath, &unix.OpenHow{
			Flags: uint64(unix.O_PATH | unix.O_DIRECTORY | unix.O_CLOEXEC),
			Resolve: unix.RESOLVE_NO_MAGICLINKS |
				unix.RESOLVE_NO_SYMLINKS,
		})
		if err != nil {
			return physicalLocation{}, errors.Join(errPhysicalOverlap, errChangedDuringScan, err)
		}
		reopenedMount, mountErr := mountIDAt(reopened, "", unix.AT_EMPTY_PATH)
		var anchoredStat, reopenedStat unix.Stat_t
		identityErr := errors.Join(
			unix.Fstat(fd, &anchoredStat),
			unix.Fstat(reopened, &reopenedStat),
			mountErr,
			unix.Close(reopened),
		)
		if identityErr != nil || anchoredStat.Dev != reopenedStat.Dev ||
			anchoredStat.Ino != reopenedStat.Ino || id != reopenedMount {
			return physicalLocation{}, errors.Join(
				errPhysicalOverlap, errChangedDuringScan, identityErr,
			)
		}
		relative, err := filepath.Rel(info[1], lexicalPath)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return physicalLocation{}, errors.Join(errPhysicalOverlap, err)
		}
		return physicalLocation{
			device: uint64(anchoredStat.Dev),
			path:   filepath.Join(info[0], relative),
		}, nil
	}, nil
}
func verifyPhysicalSeparation(
	roots []anchoredRoot,
	output *outputAnchor,
	reader physicalLocationReader,
) error {
	outputLocation, err := reader(output.fd(), output.path)
	if err != nil {
		return err
	}
	locations := make([]physicalLocation, 0, len(roots))
	for _, root := range roots {
		location, err := reader(int(root.file.Fd()), root.path)
		if err != nil {
			return err
		}
		for _, other := range locations {
			if physicalOverlap(location, other) {
				return errPhysicalOverlap
			}
		}
		if location.device == outputLocation.device {
			return errPhysicalOverlap
		}
		locations = append(locations, location)
	}
	return nil
}
func physicalOverlap(left, right physicalLocation) bool {
	return left.device == right.device &&
		(pathWithin(left.path, right.path) || pathWithin(right.path, left.path))
}
