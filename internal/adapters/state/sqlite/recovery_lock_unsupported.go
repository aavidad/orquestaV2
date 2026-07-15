//go:build !linux

package sqlite

import "errors"

type recoveryRootLocks struct{}

func acquireRecoveryRootLocks(...string) (*recoveryRootLocks, error) {
	return nil, errors.New("sqlite.recovery_root_lock_unsupported")
}

func (*recoveryRootLocks) Close() error { return nil }

func (*recoveryRootLocks) verify(...string) error {
	return errors.New("sqlite.recovery_root_lock_unsupported")
}

func (*recoveryRootLocks) sync(string) error {
	return errors.New("sqlite.recovery_root_lock_unsupported")
}
