//go:build linux

package codex

import (
	"os"
	"testing"

	"golang.org/x/sys/unix"
)

func platformDescriptorCloseOnExec(descriptor int) bool {
	flags, err := unix.FcntlInt(uintptr(descriptor), unix.F_GETFD, 0)
	return err == nil && flags&unix.FD_CLOEXEC != 0
}

func TestPlatformDescriptorCloseOnExecDistinguishesInheritedDescriptors(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()
	if !platformDescriptorCloseOnExec(int(reader.Fd())) {
		t.Fatal("os.Pipe reader lacks expected FD_CLOEXEC")
	}
	flags, err := unix.FcntlInt(reader.Fd(), unix.F_GETFD, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := unix.FcntlInt(reader.Fd(), unix.F_SETFD, flags&^unix.FD_CLOEXEC); err != nil {
		t.Fatal(err)
	}
	if platformDescriptorCloseOnExec(int(reader.Fd())) {
		t.Fatal("descriptor without FD_CLOEXEC reported as close-on-exec")
	}
}
