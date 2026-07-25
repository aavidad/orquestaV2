//go:build linux && kvm_real_e2e

package firecrackerlauncher

import (
	"errors"
	"os"
	"testing"

	"golang.org/x/sys/unix"
)

func TestRealNonRootKVMAPIVersionUsesIoctlReturn(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("requires a non-root identity with explicit KVM access")
	}
	fd, err := retryKVMOpen(func() (int, error) {
		return unix.Openat2(unix.AT_FDCWD, "/dev/kvm", &unix.OpenHow{
			Flags:   unix.O_RDWR | unix.O_CLOEXEC | unix.O_NOFOLLOW,
			Resolve: unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS,
		})
	})
	if errors.Is(err, unix.ENOENT) || errors.Is(err, unix.EACCES) ||
		errors.Is(err, unix.EPERM) {
		t.Skipf("non-root KVM access unavailable: %v", err)
	}
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(fd)

	pointerValue, pointerErr := unix.IoctlGetInt(fd, kvmGetAPIVersionIOCTL)
	if pointerErr == nil && pointerValue == kvmAPIVersion {
		t.Fatal("pointer-output helper unexpectedly captured ioctl return value")
	}
	returnValue, err := retryKVMAPIVersion(fd, func(fd int) (int, error) {
		return unix.IoctlRetInt(fd, kvmGetAPIVersionIOCTL)
	})
	if err != nil {
		t.Fatal(err)
	}
	if returnValue != kvmAPIVersion {
		t.Fatalf(
			"ioctl semantics pointer_value=%d pointer_error=%v return_value=%d",
			pointerValue,
			pointerErr,
			returnValue,
		)
	}
	if err := preflightKVM(); err != nil {
		t.Fatalf("real non-root KVM preflight failed: %v", err)
	}
}
