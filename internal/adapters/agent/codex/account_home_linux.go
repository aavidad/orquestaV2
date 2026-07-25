//go:build linux

package codex

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

const accountProfileLockName = ".orquesta-profile.lock"

func openAccountProfileLease(config Config) (*os.File, string, error) {
	if config.AccountProfile == "" {
		return nil, "", nil
	}
	if err := validateAccountPathAncestors(config.AccountHomeRoot); err != nil {
		return nil, "", err
	}
	rootInfo, err := os.Lstat(config.AccountHomeRoot)
	if err != nil || !secureAccountDirectory(rootInfo) {
		return nil, "", &Error{Code: CodeAccountProfileUnavailable, Cause: err}
	}
	accountRoot, err := os.OpenRoot(config.AccountHomeRoot)
	if err != nil {
		return nil, "", &Error{Code: CodeAccountProfileUnavailable, Cause: err}
	}
	defer accountRoot.Close()

	before, err := accountRoot.Lstat(config.AccountProfile)
	if err != nil || !secureAccountDirectory(before) {
		return nil, "", &Error{Code: CodeAccountProfileUnavailable, Cause: err}
	}
	profileRoot, err := accountRoot.OpenRoot(config.AccountProfile)
	if err != nil {
		return nil, "", &Error{Code: CodeAccountProfileUnavailable, Cause: err}
	}
	defer profileRoot.Close()
	opened, err := profileRoot.Stat(".")
	after, afterErr := accountRoot.Lstat(config.AccountProfile)
	if err != nil || afterErr != nil || !os.SameFile(before, opened) || !os.SameFile(opened, after) {
		return nil, "", &Error{Code: CodeAccountProfileUnavailable, Cause: errors.Join(err, afterErr)}
	}
	lockBefore, beforeErr := profileRoot.Lstat(accountProfileLockName)
	if beforeErr != nil && !errors.Is(beforeErr, os.ErrNotExist) {
		return nil, "", &Error{Code: CodeAccountProfileUnavailable, Cause: beforeErr}
	}
	if beforeErr == nil && !secureAccountAuthFile(lockBefore) {
		return nil, "", &Error{Code: CodeAccountProfileUnavailable}
	}
	lock, err := profileRoot.OpenFile(accountProfileLockName, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, "", &Error{Code: CodeAccountProfileUnavailable, Cause: err}
	}
	closeLock := true
	defer func() {
		if closeLock {
			_ = lock.Close()
		}
	}()
	lockInfo, statErr := lock.Stat()
	lockAfter, afterErr := profileRoot.Lstat(accountProfileLockName)
	if statErr != nil || afterErr != nil || !secureAccountAuthFile(lockInfo) ||
		!secureAccountAuthFile(lockAfter) || !os.SameFile(lockInfo, lockAfter) ||
		beforeErr == nil && !os.SameFile(lockBefore, lockInfo) {
		return nil, "", &Error{Code: CodeAccountProfileUnavailable, Cause: errors.Join(statErr, afterErr)}
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return nil, "", &Error{Code: CodeAccountProfileUnavailable, Cause: err}
	}
	if err := validateAccountAuth(profileRoot, config.AccountAuthMaxDocumentBytes); err != nil {
		return nil, "", err
	}
	closeLock = false
	return lock, filepath.Join(config.AccountHomeRoot, config.AccountProfile), nil
}

func validateAccountPathAncestors(accountHomeRoot string) error {
	current := filepath.Clean(accountHomeRoot)
	for {
		info, err := os.Lstat(current)
		if err != nil || !secureAccountAncestor(info) {
			return &Error{Code: CodeAccountProfileUnavailable, Cause: err}
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
		current = parent
	}
}

func validateAccountAuth(profileRoot *os.Root, maximumBytes int64) error {
	before, err := profileRoot.Lstat(accountAuthFileName)
	if err != nil || !secureAccountAuthFile(before) ||
		before.Size() <= 0 || before.Size() > maximumBytes {
		return &Error{Code: CodeAccountProfileUnavailable, Cause: err}
	}
	file, err := profileRoot.Open(accountAuthFileName)
	if err != nil {
		return &Error{Code: CodeAccountProfileUnavailable, Cause: err}
	}
	defer file.Close()
	openedBefore, err := file.Stat()
	if err != nil || !secureAccountAuthFile(openedBefore) || !os.SameFile(before, openedBefore) {
		return &Error{Code: CodeAccountProfileUnavailable, Cause: err}
	}
	payload, readErr := io.ReadAll(io.LimitReader(file, maximumBytes+1))
	defer clearBytes(payload)
	openedAfter, statErr := file.Stat()
	after, afterErr := profileRoot.Lstat(accountAuthFileName)
	if readErr != nil || statErr != nil || afterErr != nil ||
		int64(len(payload)) != before.Size() || int64(len(payload)) > maximumBytes ||
		!secureAccountAuthFile(openedAfter) || !secureAccountAuthFile(after) ||
		!os.SameFile(openedBefore, openedAfter) || !os.SameFile(openedAfter, after) ||
		openedAfter.Size() != before.Size() {
		return &Error{Code: CodeAccountProfileUnavailable, Cause: errors.Join(readErr, statErr, afterErr)}
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	var document map[string]json.RawMessage
	if err := decoder.Decode(&document); err != nil || len(document) == 0 {
		return &Error{Code: CodeAccountProfileUnavailable, Cause: err}
	}
	if err := requireJSONEOF(decoder); err != nil {
		return &Error{Code: CodeAccountProfileUnavailable, Cause: err}
	}
	return nil
}

func secureAccountDirectory(info os.FileInfo) bool {
	return info != nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0 &&
		info.Mode().Perm() == 0o700 && ownedByCurrentUser(info)
}

func secureAccountAncestor(info os.FileInfo) bool {
	if info == nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o022 != 0 {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && (int(stat.Uid) == os.Geteuid() || stat.Uid == 0)
}

func secureAccountAuthFile(info os.FileInfo) bool {
	return info != nil && info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0 &&
		info.Mode().Perm()&0o077 == 0 && !hardlinkedAccountFile(info) && ownedByCurrentUser(info)
}

func ownedByCurrentUser(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && int(stat.Uid) == os.Geteuid()
}

func hardlinkedAccountFile(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return !ok || stat.Nlink != 1
}

func closeAccountProfileLease(lock *os.File) error {
	if lock == nil {
		return nil
	}
	unlockErr := syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	return errors.Join(unlockErr, lock.Close())
}
