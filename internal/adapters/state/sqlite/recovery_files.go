package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"orquesta/internal/application"
)

func recoveryTargetName(ref application.RecoveryTargetRef) string {
	digest := sha256.Sum256([]byte(ref.String()))
	return hex.EncodeToString(digest[:]) + ".sqlite"
}

func mustBackupRef(value string) application.BackupRef {
	ref, err := application.NewBackupRef(value)
	if err != nil {
		panic(err)
	}
	return ref
}

func sqliteFileURI(path string, readOnly bool) string {
	uri := &url.URL{Scheme: "file", Path: path}
	query := uri.Query()
	if readOnly {
		query.Set("mode", "ro")
		query.Set("immutable", "1")
		query.Add("_pragma", "query_only(1)")
	}
	uri.RawQuery = query.Encode()
	return uri.String()
}

func openRecoveryDatabase(path string) (*sql.DB, error) {
	database, err := sql.Open(driverName, sqliteFileURI(path, true))
	if err != nil {
		return nil, err
	}
	database.SetMaxOpenConns(1)
	database.SetMaxIdleConns(1)
	if err := database.Ping(); err != nil {
		_ = database.Close()
		return nil, err
	}
	return database, nil
}

func bytesSHA256(content []byte) string {
	digest := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func fileSHA256(ctx context.Context, path string) (string, error) {
	before, err := validatePrivateRegularFile(path)
	if err != nil {
		return "", err
	}
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(before, opened) {
		return "", errors.New("sqlite.recovery_file_replaced")
	}
	hash := sha256.New()
	buffer := make([]byte, 128*1024)
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		read, readErr := file.Read(buffer)
		if read > 0 {
			if _, err := hash.Write(buffer[:read]); err != nil {
				return "", err
			}
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return "", readErr
		}
	}
	after, err := file.Stat()
	if err != nil || after.Size() != before.Size() || !os.SameFile(before, after) {
		return "", errors.New("sqlite.recovery_file_changed")
	}
	current, err := os.Lstat(path)
	if err != nil || !os.SameFile(before, current) {
		return "", errors.New("sqlite.recovery_file_replaced")
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

func readPrivateFile(path string, maximum int64) ([]byte, error) {
	before, err := validatePrivateRegularFile(path)
	if err != nil {
		return nil, err
	}
	if before.Size() < 0 || before.Size() > maximum {
		return nil, errors.New("sqlite.recovery_file_size_invalid")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(before, opened) {
		return nil, errors.New("sqlite.recovery_file_replaced")
	}
	content, err := io.ReadAll(io.LimitReader(file, maximum+1))
	if err != nil || int64(len(content)) != before.Size() {
		return nil, errors.New("sqlite.recovery_file_changed")
	}
	current, err := os.Lstat(path)
	if err != nil || !os.SameFile(before, current) {
		return nil, errors.New("sqlite.recovery_file_replaced")
	}
	return content, nil
}

func copyExclusive(ctx context.Context, sourcePath, destinationPath string) error {
	sourceInfo, err := validatePrivateRegularFile(sourcePath)
	if err != nil {
		return invalid(err)
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		return invalid(err)
	}
	defer source.Close()
	openedSource, err := source.Stat()
	if err != nil || !os.SameFile(sourceInfo, openedSource) {
		return invalid(errors.New("sqlite.recovery_file_replaced"))
	}
	destination, err := os.OpenFile(destinationPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return invalid(err)
	}
	remove := true
	defer func() {
		_ = destination.Close()
		if remove {
			_ = os.Remove(destinationPath)
		}
	}()
	buffer := make([]byte, 64*1024)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		read, readErr := source.Read(buffer)
		if read > 0 {
			if _, err := destination.Write(buffer[:read]); err != nil {
				return invalid(err)
			}
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return invalid(readErr)
		}
	}
	if err := destination.Sync(); err != nil {
		return invalid(err)
	}
	if err := destination.Close(); err != nil {
		return invalid(err)
	}
	finalSource, err := source.Stat()
	if err != nil || finalSource.Size() != sourceInfo.Size() || !os.SameFile(sourceInfo, finalSource) {
		return invalid(errors.New("sqlite.recovery_file_changed"))
	}
	currentSource, err := os.Lstat(sourcePath)
	if err != nil || !os.SameFile(sourceInfo, currentSource) {
		return invalid(errors.New("sqlite.recovery_file_replaced"))
	}
	remove = false
	return nil
}

func writeExclusiveSynced(path string, content []byte, mode os.FileMode) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	remove := true
	defer func() {
		_ = file.Close()
		if remove {
			_ = os.Remove(path)
		}
	}()
	if _, err := file.Write(content); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	remove = false
	return nil
}

func syncPrivateFile(path string) error {
	if _, err := validatePrivateRegularFile(path); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer file.Close()
	return file.Sync()
}

func preflightRecoveryRoot(value string) (string, error) {
	if strings.TrimSpace(value) == "" || strings.TrimSpace(value) != value {
		return "", errors.New("root_required")
	}
	absolute, err := filepath.Abs(filepath.Clean(value))
	if err != nil || absolute == filepath.Dir(absolute) {
		return "", errors.New("root_invalid")
	}
	if err := rejectSymlinkComponents(absolute); err != nil {
		return "", err
	}
	if _, err := os.Lstat(absolute); err == nil {
		if err := validatePrivateDirectory(absolute); err != nil {
			return "", err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	return absolute, nil
}

func validatePrivateDirectory(path string) error {
	if err := rejectSymlinkComponents(path); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode() != os.ModeDir|0o700 {
		return errors.New("sqlite.recovery_directory_not_private")
	}
	return validateOwner(info)
}

func validatePrivateRegularFile(path string) (os.FileInfo, error) {
	lstat, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if lstat.Mode() != 0o600 {
		return nil, errors.New("sqlite.recovery_file_invalid")
	}
	if err := validateOwner(lstat); err != nil {
		return nil, err
	}
	if links, ok := linkCount(lstat); !ok || links != 1 {
		return nil, errors.New("sqlite.recovery_file_links_invalid")
	}
	opened, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer opened.Close()
	stat, err := opened.Stat()
	if err != nil {
		return nil, err
	}
	if !os.SameFile(lstat, stat) {
		return nil, errors.New("sqlite.recovery_file_replaced")
	}
	return stat, nil
}

func validateOwner(info os.FileInfo) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || int(stat.Uid) != os.Geteuid() {
		return errors.New("sqlite.recovery_owner_invalid")
	}
	return nil
}

func linkCount(info os.FileInfo) (uint64, bool) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, false
	}
	return uint64(stat.Nlink), true
}

func rejectSymlinkComponents(path string) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	volume := filepath.VolumeName(absolute)
	remainder := strings.TrimPrefix(absolute, volume)
	current := volume + string(os.PathSeparator)
	for _, component := range strings.Split(strings.TrimPrefix(remainder, string(os.PathSeparator)), string(os.PathSeparator)) {
		if component == "" {
			continue
		}
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("sqlite.recovery_symlink_component")
		}
	}
	return nil
}

func pathsOverlap(left, right string) bool {
	left, _ = filepath.Abs(filepath.Clean(left))
	right, _ = filepath.Abs(filepath.Clean(right))
	within := func(parent, child string) bool {
		relative, err := filepath.Rel(parent, child)
		return err == nil && (relative == "." || relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator)))
	}
	return within(left, right) || within(right, left)
}

func cleanupRecoveryStages(locks *recoveryRootLocks, backupRoot, restoreRoot string) error {
	if err := locks.verify(backupRoot, restoreRoot); err != nil {
		return err
	}
	backup, err := os.OpenRoot(backupRoot)
	if err != nil {
		return err
	}
	defer backup.Close()
	backupEntries, err := os.ReadDir(backupRoot)
	if err != nil {
		return err
	}
	backupChanged := false
	for _, entry := range backupEntries {
		if !isBackupStageName(entry.Name()) {
			continue
		}
		path := filepath.Join(backupRoot, entry.Name())
		if err := locks.verify(backupRoot); err != nil {
			return err
		}
		if err := validatePrivateDirectory(path); err != nil {
			return err
		}
		if err := backup.RemoveAll(entry.Name()); err != nil {
			return err
		}
		backupChanged = true
	}
	if backupChanged {
		if err := locks.sync(backupRoot); err != nil {
			return err
		}
	}

	restore, err := os.OpenRoot(restoreRoot)
	if err != nil {
		return err
	}
	defer restore.Close()
	restoreEntries, err := os.ReadDir(restoreRoot)
	if err != nil {
		return err
	}
	restoreChanged := false
	for _, entry := range restoreEntries {
		if !isRestoreStageName(entry.Name()) {
			continue
		}
		path := filepath.Join(restoreRoot, entry.Name())
		if err := locks.verify(restoreRoot); err != nil {
			return err
		}
		if _, err := validatePrivateRegularFile(path); err != nil {
			return err
		}
		if err := restore.Remove(entry.Name()); err != nil {
			return err
		}
		restoreChanged = true
	}
	if restoreChanged {
		return locks.sync(restoreRoot)
	}
	return nil
}

func isBackupStageName(name string) bool {
	if !strings.HasPrefix(name, ".backup-") || !strings.HasSuffix(name, ".next") {
		return false
	}
	identity := strings.TrimSuffix(strings.TrimPrefix(name, ".backup-"), ".next")
	separator := strings.LastIndexByte(identity, '-')
	if separator <= 0 || separator == len(identity)-1 {
		return false
	}
	nanos, nanosErr := strconv.ParseInt(identity[:separator], 10, 64)
	sequence, sequenceErr := strconv.ParseUint(identity[separator+1:], 10, 64)
	return nanosErr == nil && sequenceErr == nil && nanos != 0 && sequence > 0
}

func isRestoreStageName(name string) bool {
	if len(name) != 1+64+len(".partial") || name[0] != '.' || !strings.HasSuffix(name, ".partial") {
		return false
	}
	digest := strings.TrimSuffix(name[1:], ".partial")
	decoded, err := hex.DecodeString(digest)
	return err == nil && hex.EncodeToString(decoded) == digest
}
