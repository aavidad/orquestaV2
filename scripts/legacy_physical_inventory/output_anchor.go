// Este fichero ancla un directorio privado y mantiene el bloqueo exclusivo del par.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
)

type outputAnchor struct {
	file          *os.File
	lock          *os.File
	path          string
	jsonlFinal    string
	manifestFinal string
	pairID        string
}

func openOutputAnchor(jsonlPath, manifestPath string) (*outputAnchor, error) {
	return openOutputAnchorWithHook(jsonlPath, manifestPath, nil)
}
func openOutputAnchorWithHook(
	jsonlPath, manifestPath string,
	afterAnchor func(),
) (*outputAnchor, error) {
	directoryPath := filepath.Dir(jsonlPath)
	if directoryPath != filepath.Dir(manifestPath) {
		return nil, errOutputDirectory
	}
	fd, err := unix.Openat2(unix.AT_FDCWD, directoryPath, &unix.OpenHow{
		Flags:   uint64(unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC),
		Resolve: unix.RESOLVE_NO_MAGICLINKS | unix.RESOLVE_NO_SYMLINKS,
	})
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), "salida-anclada")
	closeOnError := func(cause error) (*outputAnchor, error) {
		return nil, errors.Join(cause, file.Close())
	}
	var anchored unix.Stat_t
	if err := unix.Fstat(fd, &anchored); err != nil {
		return closeOnError(err)
	}
	if afterAnchor != nil {
		afterAnchor()
	}
	if anchored.Uid != uint32(os.Geteuid()) || anchored.Mode&0o077 != 0 {
		return closeOnError(errOutputDirectory)
	}
	jsonlFinal, manifestFinal := filepath.Base(jsonlPath), filepath.Base(manifestPath)
	pairSum := sha256.Sum256([]byte("physical-inventory-pair-v1\x00" + jsonlFinal + "\x00" + manifestFinal))
	anchor := &outputAnchor{
		file: file, path: directoryPath, jsonlFinal: jsonlFinal,
		manifestFinal: manifestFinal, pairID: hex.EncodeToString(pairSum[:12]),
	}
	return anchor, nil
}
func (anchor *outputAnchor) openLock() (*os.File, error) {
	fd, err := unix.Openat(
		int(anchor.file.Fd()), ".legacy-physical-inventory.lock",
		unix.O_RDWR|unix.O_CREAT|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600,
	)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), "bloqueo-salida")
	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil {
		return nil, errors.Join(err, file.Close())
	}
	if stat.Mode&unix.S_IFMT != unix.S_IFREG || stat.Uid != uint32(os.Geteuid()) ||
		stat.Mode&0o077 != 0 {
		return nil, errors.Join(errOutputDirectory, file.Close())
	}
	if err := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		return nil, errors.Join(errOutputLocked, file.Close())
	}
	return file, nil
}
func (anchor *outputAnchor) close() error {
	var failures []error
	if anchor.lock != nil {
		failures = appendIfError(failures, unix.Flock(int(anchor.lock.Fd()), unix.LOCK_UN))
		failures = appendIfError(failures, anchor.lock.Close())
		anchor.lock = nil
	}
	if anchor.file != nil {
		failures = appendIfError(failures, anchor.file.Close())
		anchor.file = nil
	}
	return errors.Join(failures...)
}
func (anchor *outputAnchor) sync() error {
	return anchor.file.Sync()
}
func (anchor *outputAnchor) fd() int {
	return int(anchor.file.Fd())
}
