//go:build linux

package firecrackerlauncher

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func copyAssetAt(
	ctx context.Context,
	directory *os.File,
	name string,
	asset *pinnedAsset,
	mode, uid, gid uint32,
) error {
	if ctx == nil || asset == nil || asset.file == nil {
		return launcherError(CodeAssetsUnsafe)
	}
	return createFileAt(
		directory,
		name,
		&contextReader{
			ctx:    ctx,
			reader: io.NewSectionReader(asset.file, 0, asset.size),
		},
		asset.size,
		mode,
		uid,
		gid,
		nil,
	)
}

func copyInputAt(
	ctx context.Context,
	directory *os.File,
	name string,
	input *os.File,
	wantDigest string,
	uid, gid uint32,
) error {
	identity, err := validateInputDriveDescriptor(
		ctx,
		input,
		maxInputDriveBytesHard,
		wantDigest,
	)
	if err != nil {
		return err
	}
	digest := sha256.New()
	err = createFileAt(
		directory,
		name,
		io.TeeReader(
			&contextReader{ctx: ctx, reader: io.NewSectionReader(input, 0, identity.size)},
			digest,
		),
		identity.size,
		0o400,
		uid,
		gid,
		nil,
	)
	if err != nil || hex.EncodeToString(digest.Sum(nil)) != wantDigest {
		return launcherError(CodeInputInvalid)
	}
	return nil
}

func createBytesAt(
	directory *os.File,
	name string,
	content []byte,
	mode, uid, gid uint32,
) error {
	return createFileAt(
		directory,
		name,
		bytes.NewReader(content),
		int64(len(content)),
		mode,
		uid,
		gid,
		nil,
	)
}

func createFileAt(
	directory *os.File,
	name string,
	source io.Reader,
	size int64,
	mode, uid, gid uint32,
	opened func(*os.File, unix.Stat_t),
) error {
	if directory == nil || !safeLeafName(name) || source == nil || size <= 0 {
		return launcherError(CodeExecutionFailed)
	}
	fd, err := unix.Openat(
		int(directory.Fd()),
		name,
		unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW,
		mode,
	)
	if err != nil {
		return launcherError(CodeExecutionFailed)
	}
	file := os.NewFile(uintptr(fd), "orquesta-firecracker-run-file")
	fail := func() error {
		_ = file.Close()
		return launcherError(CodeExecutionFailed)
	}
	copied, err := io.CopyN(file, source, size)
	if err != nil || copied != size ||
		unix.Fchmod(fd, mode) != nil ||
		unix.Fchown(fd, int(uid), int(gid)) != nil ||
		file.Sync() != nil {
		return fail()
	}
	var stat unix.Stat_t
	if unix.Fstat(fd, &stat) != nil ||
		stat.Mode&unix.S_IFMT != unix.S_IFREG ||
		stat.Uid != uid || stat.Gid != gid || stat.Nlink != 1 ||
		stat.Mode&0o777 != mode || stat.Size != size {
		return fail()
	}
	if opened != nil {
		opened(file, stat)
		return nil
	}
	if file.Close() != nil {
		return launcherError(CodeExecutionFailed)
	}
	return nil
}

func createOutputAt(
	directory *os.File,
	name string,
	size uint64,
	uid, gid uint32,
) (*os.File, unix.Stat_t, error) {
	if directory == nil || !safeLeafName(name) || !validOutputDriveBytes(size) ||
		size > uint64(^uint64(0)>>1) {
		return nil, unix.Stat_t{}, launcherError(CodeOutputInvalid)
	}
	fd, err := unix.Openat(
		int(directory.Fd()),
		name,
		unix.O_RDWR|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW,
		0o600,
	)
	if err != nil {
		return nil, unix.Stat_t{}, launcherError(CodeOutputInvalid)
	}
	file := os.NewFile(uintptr(fd), "orquesta-firecracker-jail-output")
	fail := func() (*os.File, unix.Stat_t, error) {
		_ = file.Close()
		return nil, unix.Stat_t{}, launcherError(CodeOutputInvalid)
	}
	if unix.Ftruncate(fd, int64(size)) != nil ||
		unix.Fallocate(fd, 0, 0, int64(size)) != nil ||
		unix.Fchmod(fd, 0o600) != nil ||
		unix.Fchown(fd, int(uid), int(gid)) != nil ||
		file.Sync() != nil {
		return fail()
	}
	var stat unix.Stat_t
	if unix.Fstat(fd, &stat) != nil ||
		stat.Mode&unix.S_IFMT != unix.S_IFREG ||
		stat.Uid != uid || stat.Gid != gid || stat.Nlink != 1 ||
		stat.Mode&0o777 != 0o600 || stat.Size != int64(size) {
		return fail()
	}
	return file, stat, nil
}

func safeLeafName(name string) bool {
	return name != "" && name != "." && name != ".." &&
		filepath.Base(name) == name && !bytes.ContainsRune([]byte(name), 0)
}

func (workspace *runWorkspace) copyOutput(
	destination *os.File,
	driveBytes uint64,
) error {
	if workspace == nil || workspace.output == nil || destination == nil {
		return launcherError(CodeOutputInvalid)
	}
	var stat unix.Stat_t
	if workspace.output.Sync() != nil ||
		unix.Fstat(int(workspace.output.Fd()), &stat) != nil ||
		!sameRunOutputIdentity(workspace.outputIdentity, stat) ||
		stat.Size != int64(driveBytes) {
		return launcherError(CodeOutputInvalid)
	}
	writer := io.NewOffsetWriter(destination, 0)
	copied, err := io.CopyN(
		writer,
		io.NewSectionReader(workspace.output, 0, int64(driveBytes)),
		int64(driveBytes),
	)
	if err != nil || copied != int64(driveBytes) || destination.Sync() != nil {
		return launcherError(CodeOutputInvalid)
	}
	return nil
}

func sameRunOutputIdentity(left, right unix.Stat_t) bool {
	return left.Dev == right.Dev && left.Ino == right.Ino &&
		left.Size == right.Size && left.Mode == right.Mode &&
		left.Uid == right.Uid && left.Gid == right.Gid &&
		left.Nlink == right.Nlink
}
