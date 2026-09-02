//go:build unix

package sqlite

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// acquireStateWriterLock makes the retained local-state identity handle also
// prove exclusive ownership of the single product writer. Maintenance and the
// runtime share this exact lock; closing the retained handle releases it.
func acquireStateWriterLock(handle *os.File) error {
	if handle == nil {
		return errors.New("sqlite.state_writer_lock_unavailable")
	}
	if err := unix.Flock(int(handle.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
			return errors.New("sqlite.state_writer_active")
		}
		return errors.New("sqlite.state_writer_lock_failed")
	}
	return nil
}
