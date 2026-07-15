package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"orquesta/internal/application"

	_ "modernc.org/sqlite"
)

const driverName = "sqlite"

type Options struct {
	Path               string
	BusyTimeout        time.Duration
	MaxOpenConnections int
	// Now is the transaction clock used to fence expired outbox leases. It is
	// injectable for deterministic tests; production defaults to time.Now.
	Now func() time.Time
}

// Repository is the file-backed SQLite implementation of StateRepository.
type Repository struct {
	db     *sql.DB
	writer *sql.DB
	path   string
	now    func() time.Time
}

var _ application.StateRepository = (*Repository)(nil)
var _ application.AccessRepository = (*Repository)(nil)

func Open(ctx context.Context, options Options) (*Repository, error) {
	path, busyMilliseconds, err := validateOptions(options)
	if err != nil {
		return nil, invalid(err)
	}
	if err := preparePrivateDatabase(path); err != nil {
		return nil, invalid(err)
	}

	dsn := buildDSN(path, busyMilliseconds)
	database, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, invalid(err)
	}
	database.SetMaxOpenConns(options.MaxOpenConnections)
	database.SetMaxIdleConns(options.MaxOpenConnections)
	writer, err := sql.Open(driverName, dsn)
	if err != nil {
		_ = database.Close()
		return nil, invalid(err)
	}
	// SQLite admits one writer at a time. Keep that queue in database/sql so
	// concurrent application writes wait for the owned connection instead of
	// racing BEGIN IMMEDIATE until busy_timeout and surfacing StateConflict.
	writer.SetMaxOpenConns(1)
	writer.SetMaxIdleConns(1)
	now := options.Now
	if now == nil {
		now = time.Now
	}
	repository := &Repository{db: database, writer: writer, path: path, now: now}
	closeOnError := true
	defer func() {
		if closeOnError {
			_ = database.Close()
			_ = writer.Close()
		}
	}()

	if err := writer.PingContext(ctx); err != nil {
		return nil, mapDatabaseError(err)
	}
	if err := applyMigrations(ctx, writer); err != nil {
		return nil, err
	}
	if err := database.PingContext(ctx); err != nil {
		return nil, mapDatabaseError(err)
	}
	if err := enforceDatabaseMode(path); err != nil {
		return nil, invalid(err)
	}
	if err := syncSQLiteDirectory(filepath.Dir(path)); err != nil {
		return nil, invalid(err)
	}
	closeOnError = false
	return repository, nil
}

func (repository *Repository) Close() error {
	if repository == nil {
		return nil
	}
	var closeErrors []error
	if repository.writer != nil && repository.writer != repository.db {
		closeErrors = append(closeErrors, repository.writer.Close())
	}
	if repository.db != nil {
		closeErrors = append(closeErrors, repository.db.Close())
	}
	return mapDatabaseError(errors.Join(closeErrors...))
}

func validateOptions(options Options) (string, int64, error) {
	if strings.TrimSpace(options.Path) == "" || strings.TrimSpace(options.Path) != options.Path {
		return "", 0, errors.New("sqlite.path_invalid")
	}
	if options.Path == ":memory:" || strings.HasPrefix(strings.ToLower(options.Path), "file:") {
		return "", 0, errors.New("sqlite.file_path_required")
	}
	if options.BusyTimeout < time.Millisecond {
		return "", 0, errors.New("sqlite.busy_timeout_invalid")
	}
	if options.MaxOpenConnections <= 0 {
		return "", 0, errors.New("sqlite.max_open_connections_invalid")
	}
	busyMilliseconds := options.BusyTimeout.Milliseconds()
	if busyMilliseconds <= 0 || busyMilliseconds > math.MaxInt32 {
		return "", 0, errors.New("sqlite.busy_timeout_invalid")
	}
	absolute, err := filepath.Abs(filepath.Clean(options.Path))
	if err != nil {
		return "", 0, err
	}
	return absolute, busyMilliseconds, nil
}

func preparePrivateDatabase(path string) error {
	directory := filepath.Dir(path)
	info, err := os.Lstat(directory)
	switch {
	case errors.Is(err, os.ErrNotExist):
		if err := createPrivateSQLiteDirectoryChain(directory); err != nil {
			return err
		}
		info, err = os.Lstat(directory)
	case err != nil:
		return err
	case info.Mode()&os.ModeSymlink != 0 || !info.IsDir():
		return errors.New("sqlite.directory_invalid")
	case info.Mode().Perm()&0o077 != 0:
		return errors.New("sqlite.directory_not_private")
	}

	databaseExisted := false
	if info, err := os.Lstat(path); err == nil {
		databaseExisted = true
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return errors.New("sqlite.database_invalid")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if !databaseExisted {
		if err := syncSQLiteDirectory(directory); err != nil {
			return err
		}
	}
	return enforceDatabaseMode(path)
}

func createPrivateSQLiteDirectoryChain(directory string) error {
	missing := make([]string, 0)
	for current := filepath.Clean(directory); ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
				return errors.New("sqlite.directory_invalid")
			}
			break
		}
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		missing = append(missing, current)
		parent := filepath.Dir(current)
		if parent == current {
			return errors.New("sqlite.directory_invalid")
		}
	}
	for index := len(missing) - 1; index >= 0; index-- {
		current := missing[index]
		if err := os.Mkdir(current, 0o700); err != nil {
			return err
		}
		if err := syncSQLiteDirectory(filepath.Dir(current)); err != nil {
			return err
		}
	}
	return nil
}

func syncSQLiteDirectory(directory string) error {
	opened, err := os.Open(directory)
	if err != nil {
		return err
	}
	defer opened.Close()
	return opened.Sync()
}

func enforceDatabaseMode(path string) error {
	if err := os.Chmod(path, 0o600); err != nil {
		return err
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		if err := os.Chmod(path+suffix, 0o600); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func buildDSN(path string, busyMilliseconds int64) string {
	dsn := &url.URL{Scheme: "file", Path: path}
	query := dsn.Query()
	query.Add("_pragma", "busy_timeout("+strconv.FormatInt(busyMilliseconds, 10)+")")
	query.Add("_pragma", "foreign_keys(1)")
	query.Add("_pragma", "journal_mode(WAL)")
	query.Add("_pragma", "synchronous(FULL)")
	query.Set("_txlock", "immediate")
	dsn.RawQuery = query.Encode()
	return dsn.String()
}

func (repository *Repository) database() (*sql.DB, error) {
	if repository == nil || repository.db == nil {
		return nil, invalid(fmt.Errorf("sqlite.repository_closed"))
	}
	return repository.db, nil
}
