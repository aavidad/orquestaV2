package sqlite

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
)

const localStateIdentityDomain = "orquesta.sqlite.local-state-identity.v1"

func captureLocalStateIdentity(path string) (*os.File, string, bool, error) {
	handle, err := openLocalStateFile(path)
	if err != nil {
		return nil, "", false, errors.New("sqlite.local_state_identity_open_failed")
	}
	identity, supported, err := localStateIdentityForHandle(handle, path)
	if err != nil {
		_ = handle.Close()
		return nil, "", false, err
	}
	return handle, identity, supported, nil
}

func verifyLocalStateIdentity(handle *os.File, path, expected string, expectedSupported bool) error {
	identity, supported, err := localStateIdentityForHandle(handle, path)
	if err != nil {
		return err
	}
	if expectedSupported && (!supported || identity != expected) {
		return errors.New("sqlite.local_state_identity_changed")
	}
	if !expectedSupported && expected != "" {
		return errors.New("sqlite.local_state_identity_changed")
	}
	return nil
}

func localStateIdentityForHandle(handle *os.File, path string) (string, bool, error) {
	if handle == nil {
		return "", false, errors.New("sqlite.local_state_identity_unavailable")
	}
	openedInfo, err := handle.Stat()
	if err != nil || !openedInfo.Mode().IsRegular() {
		return "", false, errors.New("sqlite.local_state_identity_invalid")
	}
	pathInfo, err := os.Stat(path)
	if err != nil || !os.SameFile(openedInfo, pathInfo) {
		return "", false, errors.New("sqlite.local_state_identity_replaced")
	}
	canonical, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", false, errors.New("sqlite.local_state_identity_path_invalid")
	}
	canonical, err = filepath.Abs(filepath.Clean(canonical))
	if err != nil {
		return "", false, errors.New("sqlite.local_state_identity_path_invalid")
	}
	device, inode, supported := localFileIdentity(openedInfo)
	if !supported {
		return "", false, nil
	}
	digest := sha256.New()
	writeLocalStateIdentityField(digest, localStateIdentityDomain)
	writeLocalStateIdentityField(digest, canonical)
	var fileIdentity [16]byte
	binary.BigEndian.PutUint64(fileIdentity[:8], device)
	binary.BigEndian.PutUint64(fileIdentity[8:], inode)
	_, _ = digest.Write(fileIdentity[:])
	return "local-state:sha256:" + hex.EncodeToString(digest.Sum(nil)), true, nil
}

func writeLocalStateIdentityField(digest interface{ Write([]byte) (int, error) }, value string) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(value)))
	_, _ = digest.Write(size[:])
	_, _ = digest.Write([]byte(value))
}

// LocalStateIdentity returns the immutable host-local identity captured while
// opening this repository. The active path is checked against the retained
// file handle on every read so bootstrap fails closed after replacement.
func (repository *Repository) LocalStateIdentity() (string, bool, error) {
	if repository == nil {
		return "", false, invalid(errors.New("sqlite.local_state_identity_unavailable"))
	}
	repository.localIdentityMu.Lock()
	defer repository.localIdentityMu.Unlock()
	if repository.localIdentityHandle == nil {
		return "", false, invalid(errors.New("sqlite.local_state_identity_unavailable"))
	}
	if err := verifyLocalStateIdentity(
		repository.localIdentityHandle,
		repository.path,
		repository.localIdentity,
		repository.localIdentitySupported,
	); err != nil {
		return "", false, invalid(err)
	}
	return repository.localIdentity, repository.localIdentitySupported, nil
}
