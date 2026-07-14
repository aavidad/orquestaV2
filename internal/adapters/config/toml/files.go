package toml

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"orquesta/internal/config"
)

const lockRetryDelay = 5 * time.Millisecond

func (store *Store) withLock(ctx context.Context, operation func() error) error {
	if err := validateSourceDirectory(filepath.Dir(store.path)); err != nil {
		return err
	}
	file, err := openLockFile(store.lockPath)
	if err != nil {
		return err
	}
	defer file.Close()
	for {
		err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			break
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) && !errors.Is(err, syscall.EAGAIN) {
			return storeError(config.DocumentStoreIO, err)
		}
		select {
		case <-ctx.Done():
			return storeError(config.DocumentStoreIO, ctx.Err())
		case <-time.After(lockRetryDelay):
		}
	}
	defer syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	select {
	case <-ctx.Done():
		return storeError(config.DocumentStoreIO, ctx.Err())
	default:
		return operation()
	}
}

func openLockFile(path string) (*os.File, error) {
	file, err := openNoFollow(path, syscall.O_CREAT|syscall.O_RDWR, 0o600)
	if err != nil {
		return nil, storeError(config.DocumentStoreSourceInvalid, fmt.Errorf("toml_store_lock_invalid: %w", err))
	}
	info, err := file.Stat()
	if err != nil || !privateRegular(info, 0, 0o600) {
		file.Close()
		return nil, storeError(config.DocumentStoreSourceInvalid, errors.New("toml_store_lock_invalid"))
	}
	return file, nil
}

func (store *Store) readSource() (config.StoredDocument, error) {
	content, found, err := readPrivateRegular(store.path, store.maxSourceBytes, privateWriteMode)
	if err != nil {
		return config.StoredDocument{}, err
	}
	if !found {
		if store.requireExisting {
			return config.StoredDocument{}, storeError(config.DocumentStoreSourceRequired, errors.New("toml_store_source_missing"))
		}
		return emptyDocument(), nil
	}
	return config.StoredDocument{Revision: revisionFor(content), Content: content}, nil
}

func readPrivateRegular(path string, maximum int64, mode requiredMode) ([]byte, bool, error) {
	before, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil || !privateRegular(before, 0, mode) {
		return nil, false, storeError(config.DocumentStoreSourceInvalid, errors.New("toml_store_file_invalid"))
	}
	if before.Size() > maximum {
		return nil, false, storeError(config.DocumentStoreSourceTooLarge, errors.New("toml_store_file_too_large"))
	}
	file, err := openNoFollow(path, syscall.O_RDONLY, 0)
	if err != nil {
		return nil, false, storeError(config.DocumentStoreSourceInvalid, err)
	}
	defer file.Close()
	after, err := file.Stat()
	if err != nil || !privateRegular(after, 0, mode) || !sameFileIdentity(before, after) {
		return nil, false, storeError(config.DocumentStoreSourceInvalid, errors.New("toml_store_file_changed"))
	}
	if after.Size() > maximum {
		return nil, false, storeError(config.DocumentStoreSourceTooLarge, errors.New("toml_store_file_too_large"))
	}
	limit := maximum
	if limit < math.MaxInt64 {
		limit++
	}
	content, err := io.ReadAll(io.LimitReader(file, limit))
	if err != nil {
		return nil, false, storeError(config.DocumentStoreIO, err)
	}
	if int64(len(content)) > maximum {
		return nil, false, storeError(config.DocumentStoreSourceTooLarge, errors.New("toml_store_file_too_large"))
	}
	final, err := file.Stat()
	if err != nil || !privateRegular(final, 0, mode) || !sameFileIdentity(after, final) ||
		final.Size() != after.Size() || !final.ModTime().Equal(after.ModTime()) {
		return nil, false, storeError(config.DocumentStoreSourceInvalid, errors.New("toml_store_file_changed_during_read"))
	}
	return content, true, nil
}

type requiredMode uint32

const (
	privateIntentMode requiredMode = 0
	privateWriteMode  requiredMode = 0o600
	privateReadMode   requiredMode = 0o400
)

func privateRegular(info os.FileInfo, maximum int64, required requiredMode) bool {
	if info == nil || !info.Mode().IsRegular() || info.Size() < 0 {
		return false
	}
	mode := info.Mode()
	if required == privateIntentMode {
		if mode != 0o600 && mode != 0o400 {
			return false
		}
	} else if mode != os.FileMode(required) {
		return false
	}
	if maximum > 0 && info.Size() > maximum {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Nlink == 1 && stat.Uid == uint32(os.Geteuid())
}

func sameFileIdentity(left, right os.FileInfo) bool {
	leftStat, leftOK := left.Sys().(*syscall.Stat_t)
	rightStat, rightOK := right.Sys().(*syscall.Stat_t)
	return leftOK && rightOK && leftStat.Dev == rightStat.Dev && leftStat.Ino == rightStat.Ino
}

func openNoFollow(path string, flags int, mode uint32) (*os.File, error) {
	descriptor, err := syscall.Open(path, flags|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, mode)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(descriptor), path), nil
}

func validateSourceDirectory(directory string) error {
	absolute, err := filepath.Abs(filepath.Clean(directory))
	if err != nil {
		return storeError(config.DocumentStoreSourceInvalid, err)
	}
	if err := rejectSymlinkComponents(absolute); err != nil {
		return err
	}
	info, err := os.Lstat(absolute)
	if err != nil || !privateDirectory(info) {
		return storeError(config.DocumentStoreSourceInvalid, errors.New("toml_store_directory_invalid"))
	}
	return nil
}

func rejectSymlinkComponents(absolute string) error {
	current := string(filepath.Separator)
	for _, component := range splitPathComponents(absolute) {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if err != nil {
			return storeError(config.DocumentStoreSourceInvalid, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return storeError(config.DocumentStoreSourceInvalid, errors.New("toml_store_directory_symlink"))
		}
	}
	return nil
}

func splitPathComponents(absolute string) []string {
	var result []string
	for current := filepath.Clean(absolute); current != string(filepath.Separator); current = filepath.Dir(current) {
		result = append(result, filepath.Base(current))
	}
	for left, right := 0, len(result)-1; left < right; left, right = left+1, right-1 {
		result[left], result[right] = result[right], result[left]
	}
	return result
}

func ensureReceiptDirectory(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.Mkdir(path, 0o700); err != nil {
			return storeError(config.DocumentStoreIO, err)
		}
		if err := syncDirectory(filepath.Dir(path)); err != nil {
			return err
		}
		info, err = os.Lstat(path)
	}
	if err != nil || !privateDirectory(info) {
		return storeError(config.DocumentStoreSourceInvalid, errors.New("toml_store_receipt_directory_invalid"))
	}
	return nil
}

func privateDirectory(info os.FileInfo) bool {
	if info == nil || info.Mode() != os.ModeDir|0o700 {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == uint32(os.Geteuid())
}

func writeExclusiveSynced(path string, content []byte, mode os.FileMode) error {
	file, err := openNoFollow(path, syscall.O_CREAT|syscall.O_EXCL|syscall.O_WRONLY, uint32(mode))
	if err != nil {
		return storeError(config.DocumentStoreIO, err)
	}
	remove := true
	defer func() {
		file.Close()
		if remove {
			_ = os.Remove(path)
		}
	}()
	if err := file.Chmod(mode); err != nil {
		return storeError(config.DocumentStoreIO, err)
	}
	if _, err := file.Write(content); err != nil {
		return storeError(config.DocumentStoreIO, err)
	}
	if err := file.Sync(); err != nil {
		return storeError(config.DocumentStoreIO, err)
	}
	if err := file.Close(); err != nil {
		return storeError(config.DocumentStoreIO, err)
	}
	remove = false
	return nil
}

func sealReadOnly(path string) error {
	file, err := openNoFollow(path, syscall.O_RDONLY, 0)
	if err != nil {
		return storeError(config.DocumentStoreIO, err)
	}
	defer file.Close()
	if err := file.Chmod(0o400); err != nil {
		return storeError(config.DocumentStoreIO, err)
	}
	if err := file.Sync(); err != nil {
		return storeError(config.DocumentStoreIO, err)
	}
	return nil
}

func syncDirectory(directory string) error {
	opened, err := os.Open(directory)
	if err != nil {
		return storeError(config.DocumentStoreIO, err)
	}
	defer opened.Close()
	if err := opened.Sync(); err != nil {
		return storeError(config.DocumentStoreIO, err)
	}
	return nil
}
