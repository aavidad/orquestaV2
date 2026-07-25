//go:build linux

package firecrackerguest

import (
	"context"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

type DeviceRequest struct {
	InputPath, OutputPath string
	ScratchRoot, GoBinary string
}

func RunDevices(ctx context.Context, request DeviceRequest) error {
	input, err := os.Open(request.InputPath)
	if err != nil {
		return guestError(CodeInputInvalid, err)
	}
	defer input.Close()
	output, err := os.OpenFile(request.OutputPath, os.O_RDWR, 0)
	if err != nil {
		return guestError(CodeOutputWriteFailed, err)
	}
	defer output.Close()
	inputBytes, err := driveSize(input)
	if err != nil {
		return guestError(CodeInputInvalid, err)
	}
	outputBytes, err := driveSize(output)
	if err != nil {
		return guestError(CodeOutputWriteFailed, err)
	}
	return Run(ctx, RunRequest{
		InputDrive: input, InputDriveBytes: inputBytes,
		OutputDrive: output, OutputDriveBytes: outputBytes,
		ScratchRoot: request.ScratchRoot, GoBinary: request.GoBinary,
		Executor: OSExecutor{}, NetworkCheck: RequireNoNetwork,
		ScratchCheck: RequirePrivateTmpfs, CapacityCheck: RequireScratchCapacity,
		PrepareIdentity: PrepareNobody,
		CleanupTree:     cleanupPrivateTree,
	})
}

func driveSize(file *os.File) (uint64, error) {
	info, err := file.Stat()
	if err != nil {
		return 0, err
	}
	if info.Mode().IsRegular() {
		if info.Size() <= 0 {
			return 0, unix.EINVAL
		}
		return uint64(info.Size()), nil
	}
	if info.Mode()&os.ModeDevice == 0 || info.Mode()&os.ModeCharDevice != 0 {
		return 0, unix.ENODEV
	}
	var value uint64
	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		file.Fd(),
		uintptr(unix.BLKGETSIZE64),
		uintptr(unsafe.Pointer(&value)),
	)
	if errno != 0 {
		return 0, errno
	}
	if value == 0 {
		return 0, unix.EINVAL
	}
	return value, nil
}
