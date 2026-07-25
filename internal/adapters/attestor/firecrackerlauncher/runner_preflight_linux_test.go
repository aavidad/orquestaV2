//go:build linux

package firecrackerlauncher

import (
	"errors"
	"testing"

	"golang.org/x/sys/unix"
)

func TestPreflightKVMRetriesOnlyInterruptedReadOnlySyscalls(t *testing.T) {
	tests := map[string]struct {
		openInterrupts  int
		fstatInterrupts int
		apiInterrupts   int
	}{
		"open":  {openInterrupts: 2},
		"fstat": {fstatInterrupts: 2},
		"ioctl": {apiInterrupts: 2},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			fake := newKVMPreflightFake()
			fake.openInterrupts = test.openInterrupts
			fake.fstatInterrupts = test.fstatInterrupts
			fake.apiInterrupts = test.apiInterrupts
			if err := preflightKVMWithSyscalls(fake.syscalls()); err != nil {
				t.Fatal(err)
			}
			if fake.openCalls != test.openInterrupts+1 ||
				fake.fstatCalls != test.fstatInterrupts+1 ||
				fake.apiCalls != test.apiInterrupts+1 ||
				fake.closeCalls != 1 {
				t.Fatalf("unexpected calls: %+v", fake)
			}
		})
	}
}

func TestPreflightKVMReturnsSafeCodeForEveryFailedSubstage(t *testing.T) {
	tests := map[string]struct {
		mutate func(*kvmPreflightFake)
		code   string
	}{
		"open": {
			mutate: func(fake *kvmPreflightFake) { fake.openErr = unix.EACCES },
			code:   CodeKVMOpenUnavailable,
		},
		"open_interrupted_exhausted": {
			mutate: func(fake *kvmPreflightFake) {
				fake.openInterrupts = maxKVMInterruptedSyscallAttempts
			},
			code: CodeKVMOpenUnavailable,
		},
		"metadata_read": {
			mutate: func(fake *kvmPreflightFake) { fake.fstatErr = unix.EIO },
			code:   CodeKVMMetadataUnsafe,
		},
		"metadata_interrupted_exhausted": {
			mutate: func(fake *kvmPreflightFake) {
				fake.fstatInterrupts = maxKVMInterruptedSyscallAttempts
			},
			code: CodeKVMMetadataUnsafe,
		},
		"metadata_type": {
			mutate: func(fake *kvmPreflightFake) { fake.stat.Mode = unix.S_IFREG | 0o660 },
			code:   CodeKVMMetadataUnsafe,
		},
		"metadata_mode": {
			mutate: func(fake *kvmPreflightFake) { fake.stat.Mode = unix.S_IFCHR | 0o666 },
			code:   CodeKVMMetadataUnsafe,
		},
		"metadata_owner": {
			mutate: func(fake *kvmPreflightFake) { fake.stat.Uid = 1000 },
			code:   CodeKVMMetadataUnsafe,
		},
		"metadata_links": {
			mutate: func(fake *kvmPreflightFake) { fake.stat.Nlink = 2 },
			code:   CodeKVMMetadataUnsafe,
		},
		"ioctl": {
			mutate: func(fake *kvmPreflightFake) { fake.apiErr = unix.EIO },
			code:   CodeKVMAPIUnavailable,
		},
		"ioctl_interrupted_exhausted": {
			mutate: func(fake *kvmPreflightFake) {
				fake.apiInterrupts = maxKVMInterruptedSyscallAttempts
			},
			code: CodeKVMAPIUnavailable,
		},
		"version": {
			mutate: func(fake *kvmPreflightFake) { fake.version = kvmAPIVersion + 1 },
			code:   CodeKVMVersionUnsupported,
		},
		"pointer_output_helper_zero": {
			mutate: func(fake *kvmPreflightFake) { fake.version = 0 },
			code:   CodeKVMVersionUnsupported,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			fake := newKVMPreflightFake()
			test.mutate(fake)
			err := preflightKVMWithSyscalls(fake.syscalls())
			if ErrorCode(err) != test.code || err.Error() != test.code {
				t.Fatalf("code=%q err=%q", ErrorCode(err), err)
			}
			if errors.Is(err, fake.openErr) ||
				errors.Is(err, fake.fstatErr) ||
				errors.Is(err, fake.apiErr) {
				t.Fatalf("raw syscall cause escaped: %v", err)
			}
		})
	}
}

func TestPreflightKVMDoesNotRetryNonInterruptedFailure(t *testing.T) {
	tests := map[string]struct {
		mutate func(*kvmPreflightFake)
		calls  [4]int
		code   string
	}{
		"open": {
			mutate: func(fake *kvmPreflightFake) { fake.openErr = unix.EPERM },
			calls:  [4]int{1, 0, 0, 0},
			code:   CodeKVMOpenUnavailable,
		},
		"fstat": {
			mutate: func(fake *kvmPreflightFake) { fake.fstatErr = unix.EIO },
			calls:  [4]int{1, 1, 0, 1},
			code:   CodeKVMMetadataUnsafe,
		},
		"ioctl": {
			mutate: func(fake *kvmPreflightFake) { fake.apiErr = unix.ENOTTY },
			calls:  [4]int{1, 1, 1, 1},
			code:   CodeKVMAPIUnavailable,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			fake := newKVMPreflightFake()
			test.mutate(fake)
			if err := preflightKVMWithSyscalls(
				fake.syscalls(),
			); ErrorCode(err) != test.code {
				t.Fatalf("unexpected error: %v", err)
			}
			got := [4]int{
				fake.openCalls,
				fake.fstatCalls,
				fake.apiCalls,
				fake.closeCalls,
			}
			if got != test.calls {
				t.Fatalf("unsafe failure retried or advanced: got=%v want=%v", got, test.calls)
			}
		})
	}
}

type kvmPreflightFake struct {
	openInterrupts  int
	fstatInterrupts int
	apiInterrupts   int
	openErr         error
	fstatErr        error
	apiErr          error
	stat            unix.Stat_t
	version         int
	openCalls       int
	fstatCalls      int
	apiCalls        int
	closeCalls      int
}

func newKVMPreflightFake() *kvmPreflightFake {
	return &kvmPreflightFake{
		stat: unix.Stat_t{
			Mode:  unix.S_IFCHR | 0o660,
			Uid:   0,
			Nlink: 1,
		},
		version: kvmAPIVersion,
	}
}

func (fake *kvmPreflightFake) syscalls() kvmPreflightSyscalls {
	return kvmPreflightSyscalls{
		open: func() (int, error) {
			fake.openCalls++
			if fake.openCalls <= fake.openInterrupts {
				return -1, unix.EINTR
			}
			if fake.openErr != nil {
				return -1, fake.openErr
			}
			return 42, nil
		},
		fstat: func(_ int, stat *unix.Stat_t) error {
			fake.fstatCalls++
			if fake.fstatCalls <= fake.fstatInterrupts {
				return unix.EINTR
			}
			if fake.fstatErr != nil {
				return fake.fstatErr
			}
			*stat = fake.stat
			return nil
		},
		apiVersion: func(int) (int, error) {
			fake.apiCalls++
			if fake.apiCalls <= fake.apiInterrupts {
				return 0, unix.EINTR
			}
			return fake.version, fake.apiErr
		},
		close: func(int) error {
			fake.closeCalls++
			return nil
		},
	}
}
