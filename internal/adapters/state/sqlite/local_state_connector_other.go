//go:build !unix

package sqlite

import (
	"errors"

	moderncsqlite "modernc.org/sqlite"
)

func validateLocalStateConnection(moderncsqlite.FileControl, localStateConnectionIdentity) error {
	return errors.New("sqlite.local_state_connection_identity_unsupported")
}
