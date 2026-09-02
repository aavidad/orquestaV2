//go:build !unix

package sqlite

import (
	"errors"
	"os"
)

func acquireStateWriterLock(*os.File) error {
	return errors.New("sqlite.state_writer_lock_unsupported")
}
