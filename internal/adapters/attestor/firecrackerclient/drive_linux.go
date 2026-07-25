//go:build linux

package firecrackerclient

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"

	"golang.org/x/sys/unix"

	"orquesta/internal/testattestorprotocol/rawdrive"
)

const sealedFileSeals = unix.F_SEAL_WRITE | unix.F_SEAL_GROW |
	unix.F_SEAL_SHRINK | unix.F_SEAL_SEAL

func buildInputDrive(
	ctx context.Context,
	input rawdrive.Input,
	source io.Reader,
	maxSnapshotBytes int64,
) (*os.File, string, int64, error) {
	snapshot, snapshotBytes, err := spoolSnapshot(ctx, source, maxSnapshotBytes)
	if err != nil {
		return nil, "", 0, err
	}
	defer snapshot.Close()
	driveBytes, err := rawdrive.InputDriveSize(input, uint64(snapshotBytes))
	if err != nil || driveBytes > uint64(^uint64(0)>>1) {
		return nil, "", 0, translateFailure(err)
	}
	fd, err := unix.MemfdCreate(
		"orquesta-firecracker-attestor-input",
		unix.MFD_CLOEXEC|unix.MFD_ALLOW_SEALING,
	)
	if err != nil {
		return nil, "", 0, contractError(codeInputInvalid)
	}
	drive := os.NewFile(uintptr(fd), "orquesta-firecracker-attestor-input")
	fail := func(failure error) (*os.File, string, int64, error) {
		_ = drive.Close()
		return nil, "", 0, failure
	}
	digest := sha256.New()
	if _, err := snapshot.Seek(0, io.SeekStart); err != nil {
		return fail(contractError(codeSnapshotInvalid))
	}
	if err := rawdrive.WriteInputDrive(
		io.MultiWriter(&contextWriter{ctx: ctx, destination: drive}, digest),
		driveBytes,
		input,
		&contextReader{
			ctx:    ctx,
			source: io.NewSectionReader(snapshot, 0, snapshotBytes),
		},
		uint64(snapshotBytes),
	); err != nil {
		return fail(translateInputFailure(err))
	}
	if ctx.Err() != nil {
		return fail(translateFailure(ctx.Err()))
	}
	if err := sealReadOnlyMemfd(drive, int64(driveBytes)); err != nil {
		return fail(err)
	}
	return drive, hex.EncodeToString(digest.Sum(nil)), int64(driveBytes), nil
}

func translateInputFailure(err error) error {
	if errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, context.Canceled) {
		return translateFailure(err)
	}
	switch rawdrive.ErrorCode(err) {
	case rawdrive.CodeLimit:
		return contractError(codeResourceLimit)
	case rawdrive.CodeSnapshotInvalid, rawdrive.CodeTruncated:
		return contractError(codeSnapshotInvalid)
	default:
		return contractError(codeInputInvalid)
	}
}

func spoolSnapshot(
	ctx context.Context,
	source io.Reader,
	maxBytes int64,
) (*os.File, int64, error) {
	if source == nil || maxBytes < int64(len(snapshotMagic)) {
		return nil, 0, contractError(codeSnapshotInvalid)
	}
	fd, err := unix.MemfdCreate(
		"orquesta-firecracker-attestor-snapshot",
		unix.MFD_CLOEXEC|unix.MFD_ALLOW_SEALING,
	)
	if err != nil {
		return nil, 0, contractError(codeSnapshotInvalid)
	}
	snapshot := os.NewFile(uintptr(fd), "orquesta-firecracker-attestor-snapshot")
	fail := func(failure error) (*os.File, int64, error) {
		_ = snapshot.Close()
		return nil, 0, failure
	}
	limited := &io.LimitedReader{R: &contextReader{ctx: ctx, source: source}, N: maxBytes + 1}
	buffer := make([]byte, snapshotCopyBlockSize)
	written, copyErr := io.CopyBuffer(snapshot, limited, buffer)
	switch {
	case ctx.Err() != nil:
		return fail(translateFailure(ctx.Err()))
	case copyErr != nil:
		return fail(contractError(codeSnapshotInvalid))
	case written > maxBytes:
		return fail(contractError(codeSnapshotLimit))
	case written < int64(len(snapshotMagic)):
		return fail(contractError(codeSnapshotInvalid))
	}
	if err := sealReadOnlyMemfd(snapshot, written); err != nil {
		return fail(contractError(codeSnapshotInvalid))
	}
	return snapshot, written, nil
}

func sealReadOnlyMemfd(file *os.File, wantSize int64) error {
	var stat unix.Stat_t
	if file == nil || unix.Fstat(int(file.Fd()), &stat) != nil ||
		stat.Mode&unix.S_IFMT != unix.S_IFREG || stat.Size != wantSize ||
		stat.Nlink != 0 || unix.Fchmod(int(file.Fd()), 0o400) != nil {
		return contractError(codeInputInvalid)
	}
	if _, err := unix.FcntlInt(file.Fd(), unix.F_ADD_SEALS, sealedFileSeals); err != nil {
		return contractError(codeInputInvalid)
	}
	seals, err := unix.FcntlInt(file.Fd(), unix.F_GET_SEALS, 0)
	if err != nil || seals&sealedFileSeals != sealedFileSeals {
		return contractError(codeInputInvalid)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return contractError(codeInputInvalid)
	}
	return nil
}

func readOutputDrive(
	ctx context.Context,
	file *os.File,
	driveBytes uint64,
	wantDigest string,
) (rawdrive.Output, error) {
	if err := validatePrivateOutput(file, driveBytes); err != nil {
		return rawdrive.Output{}, err
	}
	digest := sha256.New()
	reader := io.TeeReader(
		&contextReader{
			ctx:    ctx,
			source: io.NewSectionReader(file, 0, int64(driveBytes)),
		},
		digest,
	)
	output, err := rawdrive.ReadOutputDrive(reader, driveBytes)
	if err != nil {
		return rawdrive.Output{}, translateFailure(err)
	}
	if ctx.Err() != nil {
		return rawdrive.Output{}, translateFailure(ctx.Err())
	}
	if hex.EncodeToString(digest.Sum(nil)) != wantDigest {
		return rawdrive.Output{}, contractError(codeOutputInvalid)
	}
	return output, nil
}

func validatePrivateOutput(file *os.File, wantBytes uint64) error {
	if file == nil || wantBytes == 0 || wantBytes > rawdrive.MaxOutputDriveBytes ||
		wantBytes%rawdrive.SectorSize != 0 {
		return contractError(codeOutputInvalid)
	}
	var stat unix.Stat_t
	if unix.Fstat(int(file.Fd()), &stat) != nil ||
		stat.Mode&unix.S_IFMT != unix.S_IFREG ||
		stat.Mode&0o777 != 0o400 ||
		stat.Nlink != 0 ||
		stat.Size != int64(wantBytes) {
		return contractError(codeOutputInvalid)
	}
	seals, err := unix.FcntlInt(file.Fd(), unix.F_GET_SEALS, 0)
	if err != nil || seals&sealedFileSeals != sealedFileSeals {
		return contractError(codeOutputInvalid)
	}
	return nil
}

type contextReader struct {
	ctx    context.Context
	source io.Reader
}

func (reader *contextReader) Read(value []byte) (int, error) {
	if err := reader.ctx.Err(); err != nil {
		return 0, err
	}
	count, err := reader.source.Read(value)
	if err == nil && count == 0 {
		return 0, io.ErrNoProgress
	}
	return count, err
}

type contextWriter struct {
	ctx         context.Context
	destination io.Writer
}

func (writer *contextWriter) Write(value []byte) (int, error) {
	if err := writer.ctx.Err(); err != nil {
		return 0, err
	}
	return writer.destination.Write(value)
}

func closeWithContract(file *os.File) error {
	if file == nil {
		return nil
	}
	if err := file.Close(); err != nil {
		return contractError(codeCleanupFailed)
	}
	return nil
}
