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

func openRecoveryDatabase(file *os.File) (*sql.DB, error) {
	descriptorPath, err := recoveryDescriptorPath(file)
	if err != nil {
		return nil, err
	}
	database, err := sql.Open(driverName, sqliteFileURI(descriptorPath, true))
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

func fileSHA256FromFile(ctx context.Context, file *os.File, before os.FileInfo) (string, error) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
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
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

func readPrivateFile(root *os.Root, name string, maximum int64) ([]byte, error) {
	file, before, err := openPrivateRegularFile(root, name, os.O_RDONLY)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	if before.Size() < 0 || before.Size() > maximum {
		return nil, errors.New("sqlite.recovery_file_size_invalid")
	}
	content, err := io.ReadAll(io.LimitReader(file, maximum+1))
	if err != nil || int64(len(content)) != before.Size() {
		return nil, errors.New("sqlite.recovery_file_changed")
	}
	if err := verifyOpenPrivateRegularFile(root, name, file, before); err != nil {
		return nil, err
	}
	return content, nil
}

func copyExclusive(
	ctx context.Context,
	sourceRoot *os.Root,
	sourceName string,
	source *os.File,
	sourceInfo os.FileInfo,
	destinationRoot *os.Root,
	destinationName string,
) (*os.File, os.FileInfo, error) {
	if source == nil || sourceInfo == nil {
		return nil, nil, invalid(errors.New("sqlite.recovery_source_handle_required"))
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return nil, nil, invalid(err)
	}
	destination, err := destinationRoot.OpenFile(destinationName, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err != nil {
		return nil, nil, invalid(err)
	}
	remove := true
	defer func() {
		if remove {
			_ = destination.Close()
			_ = destinationRoot.Remove(destinationName)
		}
	}()
	buffer := make([]byte, 64*1024)
	for {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		read, readErr := source.Read(buffer)
		if read > 0 {
			if _, err := destination.Write(buffer[:read]); err != nil {
				return nil, nil, invalid(err)
			}
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return nil, nil, invalid(readErr)
		}
	}
	if err := destination.Sync(); err != nil {
		return nil, nil, invalid(err)
	}
	if err := verifyOpenPrivateRegularFile(sourceRoot, sourceName, source, sourceInfo); err != nil {
		return nil, nil, invalid(err)
	}
	destinationInfo, err := bindOpenPrivateRegularFile(destinationRoot, destinationName, destination)
	if err != nil {
		return nil, nil, invalid(err)
	}
	remove = false
	return destination, destinationInfo, nil
}

func writeExclusiveSynced(root *os.Root, name string, content []byte, mode os.FileMode) error {
	file, err := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	remove := true
	defer func() {
		_ = file.Close()
		if remove {
			_ = root.Remove(name)
		}
	}()
	if _, err := file.Write(content); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if _, err := bindOpenPrivateRegularFile(root, name, file); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	remove = false
	return nil
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

func validatePrivateDirectoryAt(root *os.Root, name string) (os.FileInfo, error) {
	directory, info, err := openPrivateDirectory(root, name)
	if err != nil {
		return nil, err
	}
	defer directory.Close()
	if err := verifyOpenPrivateDirectory(root, name, directory, info); err != nil {
		return nil, err
	}
	return info, nil
}

func openPrivateDirectory(root *os.Root, name string) (*os.File, os.FileInfo, error) {
	if root == nil {
		return nil, nil, errors.New("sqlite.recovery_root_handle_required")
	}
	info, err := root.Lstat(name)
	if err != nil {
		return nil, nil, err
	}
	if info.Mode() != os.ModeDir|0o700 {
		return nil, nil, errors.New("sqlite.recovery_directory_not_private")
	}
	if err := validateOwner(info); err != nil {
		return nil, nil, err
	}
	directory, err := root.Open(name)
	if err != nil {
		return nil, nil, err
	}
	opened, err := directory.Stat()
	if err != nil || !os.SameFile(info, opened) {
		_ = directory.Close()
		return nil, nil, errors.New("sqlite.recovery_directory_replaced")
	}
	return directory, info, nil
}

func verifyOpenPrivateDirectory(root *os.Root, name string, directory *os.File, before os.FileInfo) error {
	opened, err := directory.Stat()
	if err != nil || !os.SameFile(before, opened) || opened.Mode() != os.ModeDir|0o700 {
		return errors.New("sqlite.recovery_directory_changed")
	}
	if err := validateOwner(opened); err != nil {
		return err
	}
	current, err := root.Lstat(name)
	if err != nil || !os.SameFile(before, current) {
		return errors.New("sqlite.recovery_directory_replaced")
	}
	if current.Mode() != os.ModeDir|0o700 {
		return errors.New("sqlite.recovery_directory_not_private")
	}
	return validateOwner(current)
}

func readPrivateDirectory(root *os.Root, name string) ([]os.DirEntry, error) {
	directory, info, err := openPrivateDirectory(root, name)
	if err != nil {
		return nil, err
	}
	defer directory.Close()
	entries, err := directory.ReadDir(-1)
	if err != nil {
		return nil, err
	}
	if err := verifyOpenPrivateDirectory(root, name, directory, info); err != nil {
		return nil, err
	}
	return entries, nil
}

func syncPrivateDirectory(root *os.Root, name string) error {
	directory, info, err := openPrivateDirectory(root, name)
	if err != nil {
		return err
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return err
	}
	return verifyOpenPrivateDirectory(root, name, directory, info)
}

func validatePrivateRegularFileAt(root *os.Root, name string) (os.FileInfo, error) {
	file, info, err := openPrivateRegularFile(root, name, os.O_RDONLY)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	if err := verifyOpenPrivateRegularFile(root, name, file, info); err != nil {
		return nil, err
	}
	return info, nil
}

func openPrivateRegularFile(root *os.Root, name string, flag int) (*os.File, os.FileInfo, error) {
	if root == nil {
		return nil, nil, errors.New("sqlite.recovery_root_handle_required")
	}
	info, err := root.Lstat(name)
	if err != nil {
		return nil, nil, err
	}
	if err := validatePrivateRegularFileInfo(info); err != nil {
		return nil, nil, err
	}
	file, err := root.OpenFile(name, flag, 0)
	if err != nil {
		return nil, nil, err
	}
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) {
		_ = file.Close()
		return nil, nil, errors.New("sqlite.recovery_file_replaced")
	}
	return file, info, nil
}

func verifyOpenPrivateRegularFile(root *os.Root, name string, file *os.File, before os.FileInfo) error {
	opened, err := bindOpenPrivateRegularFile(root, name, file)
	if err != nil {
		return err
	}
	if opened.Size() != before.Size() || !os.SameFile(before, opened) {
		return errors.New("sqlite.recovery_file_changed")
	}
	return nil
}

func bindOpenPrivateRegularFile(root *os.Root, name string, file *os.File) (os.FileInfo, error) {
	if root == nil || file == nil {
		return nil, errors.New("sqlite.recovery_file_handle_required")
	}
	opened, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if err := validatePrivateRegularFileInfo(opened); err != nil {
		return nil, err
	}
	current, err := root.Lstat(name)
	if err != nil || !os.SameFile(opened, current) {
		return nil, errors.New("sqlite.recovery_file_replaced")
	}
	if err := validatePrivateRegularFileInfo(current); err != nil {
		return nil, err
	}
	return opened, nil
}

func validatePrivateRegularFileInfo(info os.FileInfo) error {
	if info.Mode() != 0o600 {
		return errors.New("sqlite.recovery_file_invalid")
	}
	if err := validateOwner(info); err != nil {
		return err
	}
	if links, ok := linkCount(info); !ok || links != 1 {
		return errors.New("sqlite.recovery_file_links_invalid")
	}
	return nil
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
	backup, err := locks.openedRoot(backupRoot)
	if err != nil {
		return err
	}
	backupEntries, err := readPrivateDirectory(backup, ".")
	if err != nil {
		return err
	}
	backupChanged := false
	for _, entry := range backupEntries {
		if !isBackupStageName(entry.Name()) {
			continue
		}
		if _, err := validatePrivateDirectoryAt(backup, entry.Name()); err != nil {
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

	restore, err := locks.openedRoot(restoreRoot)
	if err != nil {
		return err
	}
	restoreEntries, err := readPrivateDirectory(restore, ".")
	if err != nil {
		return err
	}
	restoreChanged := false
	for _, entry := range restoreEntries {
		if !isRestoreStageName(entry.Name()) {
			continue
		}
		if _, err := validatePrivateRegularFileAt(restore, entry.Name()); err != nil {
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
	return locks.verify(backupRoot, restoreRoot)
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
