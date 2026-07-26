//go:build linux

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sys/unix"

	"orquesta/internal/adapters/attestor/firecrackerguest"
)

const (
	inputDrivePath  = "/dev/vda"
	outputDrivePath = "/dev/vdb"
	scratchRoot     = "/run/orquesta-test"
	goBinaryPath    = "/toolchain/bin/go"
	scratchMode     = 0o711
	scratchOptions  = "mode=0711,size=75%"
)

func main() {
	if exitCode, handled := firecrackerguest.DispatchPID1Supervisor(os.Args); handled {
		os.Exit(exitCode)
	}
	if os.Getpid() != 1 {
		os.Exit(2)
	}
	if err := mountScratch(); err == nil {
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
		_ = firecrackerguest.RunDevices(ctx, firecrackerguest.DeviceRequest{
			InputPath: inputDrivePath, OutputPath: outputDrivePath,
			ScratchRoot: scratchRoot, GoBinary: goBinaryPath,
		})
		stop()
	}
	unix.Sync()
	_ = unix.Reboot(unix.LINUX_REBOOT_CMD_RESTART)
	select {}
}

func mountScratch() error {
	if err := os.MkdirAll(scratchRoot, scratchMode); err != nil {
		return err
	}
	return unix.Mount(
		"tmpfs", scratchRoot, "tmpfs",
		uintptr(unix.MS_NODEV|unix.MS_NOSUID),
		scratchOptions,
	)
}
