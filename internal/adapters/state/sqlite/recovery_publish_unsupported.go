//go:build !linux

package sqlite

import "errors"

func (*recoveryRootLocks) publishNoReplace(_, _, _ string) error {
	return errors.New("sqlite.recovery_atomic_publish_unsupported")
}
