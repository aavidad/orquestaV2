//go:build !linux

package codex

import (
	"os"
)

func openAccountProfileLease(config Config) (*os.File, string, error) {
	if config.AccountProfile == "" {
		return nil, "", nil
	}
	return nil, "", &Error{Code: CodeAccountProfileUnavailable}
}

func closeAccountProfileLease(lock *os.File) error {
	if lock == nil {
		return nil
	}
	return lock.Close()
}
