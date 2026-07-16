//go:build unix

package sqlite

import (
	"errors"

	"golang.org/x/sys/unix"
	moderncsqlite "modernc.org/sqlite"
)

func validateLocalStateConnection(control moderncsqlite.FileControl, expected localStateConnectionIdentity) error {
	descriptor, err := control.FileControlFileDescriptor("main")
	if err != nil || descriptor < 0 {
		return errors.New("sqlite.local_state_connection_identity_unavailable")
	}
	var stat unix.Stat_t
	if err := unix.Fstat(descriptor, &stat); err != nil || stat.Ino == 0 {
		return errors.New("sqlite.local_state_connection_identity_unavailable")
	}
	if uint64(stat.Dev) != expected.device || uint64(stat.Ino) != expected.inode {
		return errors.New("sqlite.local_state_connection_identity_changed")
	}
	return nil
}
