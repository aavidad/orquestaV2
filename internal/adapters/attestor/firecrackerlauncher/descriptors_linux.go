//go:build linux

package firecrackerlauncher

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"

	"golang.org/x/sys/unix"
)

const (
	inputSeals  = unix.F_SEAL_WRITE | unix.F_SEAL_GROW | unix.F_SEAL_SHRINK | unix.F_SEAL_SEAL
	outputSeals = unix.F_SEAL_GROW | unix.F_SEAL_SHRINK
)

type descriptorIdentity struct {
	device uint64
	inode  uint64
	size   int64
	mode   uint32
}

func NewSealedInput(payload []byte, maxBytes int64) (*os.File, string, error) {
	return newSealedInputFromReaderContext(
		context.Background(),
		bytes.NewReader(payload),
		int64(len(payload)),
		maxBytes,
	)
}

// NewSealedInputFromReader is a compatibility helper for finite readers. It
// streams in constant memory, but cannot cancel an arbitrary blocking Read.
// Production adapters must use NewSealedInputFromFileContext.
func NewSealedInputFromReader(reader io.Reader, size, maxBytes int64) (*os.File, string, error) {
	return newSealedInputFromReaderContext(context.Background(), reader, size, maxBytes)
}

func NewSealedInputFromFileContext(
	ctx context.Context,
	source *os.File,
	size, maxBytes int64,
) (*os.File, string, error) {
	var before unix.Stat_t
	if source == nil || unix.Fstat(int(source.Fd()), &before) != nil ||
		before.Mode&unix.S_IFMT != unix.S_IFREG || before.Size != size {
		return nil, "", launcherError(CodeInputInvalid)
	}
	sealed, digest, err := newSealedInputFromReaderContext(
		ctx,
		io.NewSectionReader(source, 0, size),
		size,
		maxBytes,
	)
	if err != nil {
		return nil, "", err
	}
	var after unix.Stat_t
	if unix.Fstat(int(source.Fd()), &after) != nil ||
		!sameSourceMetadata(before, after) {
		_ = sealed.Close()
		return nil, "", launcherError(CodeInputInvalid)
	}
	return sealed, digest, nil
}

func newSealedInputFromReaderContext(
	ctx context.Context,
	reader io.Reader,
	size, maxBytes int64,
) (*os.File, string, error) {
	if reader == nil || size <= 0 || size > maxBytes {
		return nil, "", launcherError(CodeInputInvalid)
	}
	fd, err := unix.MemfdCreate("orquesta-firecracker-input", unix.MFD_CLOEXEC|unix.MFD_ALLOW_SEALING)
	if err != nil {
		return nil, "", launcherError(CodeInputInvalid)
	}
	file := os.NewFile(uintptr(fd), "orquesta-firecracker-input")
	fail := func() (*os.File, string, error) {
		_ = file.Close()
		return nil, "", launcherError(CodeInputInvalid)
	}
	digest := sha256.New()
	copied, err := io.CopyN(io.MultiWriter(file, digest), &contextReader{ctx: ctx, reader: reader}, size)
	if err != nil || copied != size || unix.Fchmod(fd, 0o400) != nil {
		if ctx.Err() != nil {
			_ = file.Close()
			return nil, "", launcherError(CodeUnavailable)
		}
		return fail()
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fail()
	}
	if _, err := unix.FcntlInt(file.Fd(), unix.F_ADD_SEALS, inputSeals); err != nil {
		return fail()
	}
	digestHex := hex.EncodeToString(digest.Sum(nil))
	if _, err := validateInputDescriptor(ctx, file, maxBytes, digestHex); err != nil {
		_ = file.Close()
		return nil, "", err
	}
	return file, digestHex, nil
}

func NewOutputDescriptor(size uint64) (*os.File, error) {
	if !validOutputDriveBytes(size) {
		return nil, launcherError(CodeOutputInvalid)
	}
	fd, err := unix.MemfdCreate("orquesta-firecracker-output", unix.MFD_CLOEXEC|unix.MFD_ALLOW_SEALING)
	if err != nil {
		return nil, launcherError(CodeOutputInvalid)
	}
	file := os.NewFile(uintptr(fd), "orquesta-firecracker-output")
	fail := func() (*os.File, error) {
		_ = file.Close()
		return nil, launcherError(CodeOutputInvalid)
	}
	if unix.Ftruncate(fd, int64(size)) != nil || unix.Fchmod(fd, 0o600) != nil {
		return fail()
	}
	if _, err := unix.FcntlInt(file.Fd(), unix.F_ADD_SEALS, outputSeals); err != nil {
		return fail()
	}
	if _, err := validateEmptyOutputDescriptor(file, size); err != nil {
		return fail()
	}
	return file, nil
}

func validateInputDescriptor(
	ctx context.Context,
	file *os.File,
	maxBytes int64,
	wantDigest string,
) (descriptorIdentity, error) {
	if ctx.Err() != nil {
		return descriptorIdentity{}, launcherError(CodeUnavailable)
	}
	identity, seals, err := descriptorMetadata(file)
	if err != nil || identity.size <= 0 || identity.size > maxBytes ||
		identity.mode&unix.S_IFMT != unix.S_IFREG || identity.mode&0o777 != 0o400 ||
		seals&inputSeals != inputSeals {
		return descriptorIdentity{}, launcherError(CodeInputInvalid)
	}
	digest, err := descriptorDigestContext(ctx, file, identity.size)
	if err != nil || digest != wantDigest {
		if ctx.Err() != nil {
			return descriptorIdentity{}, launcherError(CodeUnavailable)
		}
		return descriptorIdentity{}, launcherError(CodeInputInvalid)
	}
	after, afterSeals, err := descriptorMetadata(file)
	if err != nil || after != identity || afterSeals != seals {
		return descriptorIdentity{}, launcherError(CodeInputInvalid)
	}
	return identity, nil
}

func validateInputDriveDescriptor(
	ctx context.Context,
	file *os.File,
	maxBytes int64,
	wantDigest string,
) (descriptorIdentity, error) {
	identity, err := validateInputDescriptor(ctx, file, maxBytes, wantDigest)
	if err != nil {
		return descriptorIdentity{}, err
	}
	if identity.size < int64(firecrackerSectorBytes) ||
		identity.size%int64(firecrackerSectorBytes) != 0 ||
		identity.size > maxInputDriveBytesHard {
		return descriptorIdentity{}, launcherError(CodeInputInvalid)
	}
	return identity, nil
}

func validateEmptyOutputDescriptor(file *os.File, wantBytes uint64) (descriptorIdentity, error) {
	identity, err := validateWritableOutputDescriptor(file, wantBytes)
	if err != nil {
		return descriptorIdentity{}, err
	}
	buffer := make([]byte, 32*1024)
	reader := io.NewSectionReader(file, 0, identity.size)
	zeroes := make([]byte, len(buffer))
	for {
		count, readErr := reader.Read(buffer)
		if count > 0 && !bytes.Equal(buffer[:count], zeroes[:count]) {
			return descriptorIdentity{}, launcherError(CodeOutputInvalid)
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return descriptorIdentity{}, launcherError(CodeOutputInvalid)
		}
	}
	return identity, nil
}

func validateWritableOutputDescriptor(file *os.File, wantBytes uint64) (descriptorIdentity, error) {
	identity, seals, err := descriptorMetadata(file)
	if err != nil || !validOutputDriveBytes(wantBytes) || identity.size != int64(wantBytes) ||
		identity.mode&unix.S_IFMT != unix.S_IFREG || identity.mode&0o777 != 0o600 ||
		seals&outputSeals != outputSeals || seals&(unix.F_SEAL_WRITE|unix.F_SEAL_SEAL) != 0 {
		return descriptorIdentity{}, launcherError(CodeOutputInvalid)
	}
	return identity, nil
}

func publishOutput(file *os.File, driveBytes uint64) (string, error) {
	before, err := validateWritableOutputDescriptor(file, driveBytes)
	if err != nil {
		return "", err
	}
	if file.Sync() != nil || unix.Fchmod(int(file.Fd()), 0o400) != nil {
		return "", launcherError(CodeOutputInvalid)
	}
	if _, err := unix.FcntlInt(file.Fd(), unix.F_ADD_SEALS, unix.F_SEAL_WRITE|unix.F_SEAL_SEAL); err != nil {
		return "", launcherError(CodeOutputInvalid)
	}
	after, seals, err := descriptorMetadata(file)
	if err != nil || after.device != before.device || after.inode != before.inode ||
		after.size != before.size || after.mode&0o777 != 0o400 ||
		seals&(inputSeals) != inputSeals {
		return "", launcherError(CodeOutputInvalid)
	}
	digest, err := descriptorDigest(file, after.size)
	if err != nil {
		return "", launcherError(CodeOutputInvalid)
	}
	return digest, nil
}

func validatePublishedOutput(file *os.File, wantBytes uint64, wantDigest string) (descriptorIdentity, error) {
	identity, seals, err := descriptorMetadata(file)
	if err != nil || !validOutputDriveBytes(wantBytes) || identity.size != int64(wantBytes) ||
		identity.mode&unix.S_IFMT != unix.S_IFREG || identity.mode&0o777 != 0o400 ||
		seals&inputSeals != inputSeals {
		return descriptorIdentity{}, launcherError(CodeOutputInvalid)
	}
	digest, err := descriptorDigest(file, identity.size)
	if err != nil || digest != wantDigest {
		return descriptorIdentity{}, launcherError(CodeOutputInvalid)
	}
	return identity, nil
}

func descriptorMetadata(file *os.File) (descriptorIdentity, int, error) {
	var stat unix.Stat_t
	if file == nil || unix.Fstat(int(file.Fd()), &stat) != nil || stat.Nlink != 0 {
		return descriptorIdentity{}, 0, launcherError(CodeDescriptorInvalid)
	}
	seals, err := unix.FcntlInt(file.Fd(), unix.F_GET_SEALS, 0)
	if err != nil {
		return descriptorIdentity{}, 0, launcherError(CodeDescriptorInvalid)
	}
	return descriptorIdentity{
		device: uint64(stat.Dev), inode: stat.Ino, size: stat.Size, mode: stat.Mode,
	}, seals, nil
}

func descriptorDigest(file *os.File, size int64) (string, error) {
	return descriptorDigestContext(context.Background(), file, size)
}

func descriptorDigestContext(ctx context.Context, file *os.File, size int64) (string, error) {
	digest := sha256.New()
	reader := io.NewSectionReader(file, 0, size)
	buffer := make([]byte, 128*1024)
	var copied int64
	for copied < size {
		if ctx.Err() != nil {
			return "", launcherError(CodeUnavailable)
		}
		want := min(int64(len(buffer)), size-copied)
		count, err := io.ReadFull(reader, buffer[:want])
		if err != nil {
			return "", launcherError(CodeDescriptorInvalid)
		}
		if _, err := digest.Write(buffer[:count]); err != nil {
			return "", launcherError(CodeDescriptorInvalid)
		}
		copied += int64(count)
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func sameSourceMetadata(before, after unix.Stat_t) bool {
	return before.Dev == after.Dev && before.Ino == after.Ino &&
		before.Size == after.Size && before.Mode == after.Mode &&
		before.Mtim == after.Mtim && before.Ctim == after.Ctim
}

func (reader *contextReader) Read(buffer []byte) (int, error) {
	if err := reader.ctx.Err(); err != nil {
		return 0, err
	}
	return reader.reader.Read(buffer)
}
