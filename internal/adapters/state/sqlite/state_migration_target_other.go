//go:build !linux

package sqlite

import (
	"errors"
	"os"
)

type stateMigrationTarget struct {
	file *os.File
}

func openStateMigrationTarget(string, uint32) (*stateMigrationTarget, error) {
	return nil, errors.New("sqlite.state_migration_target_unsupported")
}

func (*stateMigrationTarget) verify() error {
	return errors.New("sqlite.state_migration_target_unsupported")
}

func (*stateMigrationTarget) rejectSidecars() error {
	return errors.New("sqlite.state_migration_target_unsupported")
}

func (*stateMigrationTarget) syncParent() error {
	return errors.New("sqlite.state_migration_target_unsupported")
}

func (*stateMigrationTarget) Close() error { return nil }
