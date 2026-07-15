//go:build !linux

package sqlite

import (
	"errors"
	"os"
)

type recoveryRootLocks struct{}

func recoveryRootLockSupportError() error {
	return errors.New("sqlite.recovery_root_lock_unsupported")
}

func acquireRecoveryRootLocks(...string) (*recoveryRootLocks, error) {
	return nil, errors.New("sqlite.recovery_root_lock_unsupported")
}

func (*recoveryRootLocks) Close() error { return nil }

func (*recoveryRootLocks) verify(...string) error {
	return errors.New("sqlite.recovery_root_lock_unsupported")
}

func (*recoveryRootLocks) openedRoot(string) (*os.Root, error) {
	return nil, errors.New("sqlite.recovery_root_lock_unsupported")
}

func (*recoveryRootLocks) sync(string) error {
	return errors.New("sqlite.recovery_root_lock_unsupported")
}

func recoveryDescriptorPath(*os.File) (string, error) {
	return "", errors.New("sqlite.recovery_file_handle_unsupported")
}
