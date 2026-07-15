//go:build !linux

package sqlite

import (
	"errors"
	"os"
)

func validateOwner(os.FileInfo) error {
	return errors.New("sqlite.recovery_owner_metadata_unsupported")
}

func linkCount(os.FileInfo) (uint64, bool) {
	return 0, false
}
