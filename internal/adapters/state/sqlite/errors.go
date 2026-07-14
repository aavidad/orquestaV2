package sqlite

import (
	"database/sql"
	"errors"

	"orquesta/internal/application"

	sqlitedriver "modernc.org/sqlite"
)

const (
	sqliteBusy       = 5
	sqliteLocked     = 6
	sqliteConstraint = 19
)

func stateError(code application.StateErrorCode, cause error) error {
	return &application.StateError{Code: code, Cause: cause}
}

func invalid(cause error) error {
	return stateError(application.StateInvalid, cause)
}

func conflict(cause error) error {
	return stateError(application.StateConflict, cause)
}

func mapDatabaseError(err error) error {
	if err == nil {
		return nil
	}
	var stateErr *application.StateError
	if errors.As(err, &stateErr) {
		return err
	}
	if errors.Is(err, sql.ErrNoRows) {
		return stateError(application.StateNotFound, err)
	}
	var sqliteErr *sqlitedriver.Error
	if errors.As(err, &sqliteErr) {
		switch sqliteErr.Code() & 0xff {
		case sqliteBusy, sqliteLocked, sqliteConstraint:
			return conflict(err)
		}
	}
	return invalid(err)
}
