package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

type cachedTestDatabaseSeed struct {
	sync.Once
	content []byte
	err     error
}

var fastSchemaSeedState cachedTestDatabaseSeed

// seedFastTestDatabase copies an empty database already migrated by the same
// production migration runner. Every test still enters through Open afterwards.
func seedFastTestDatabase(path string) error {
	if _, err := os.Lstat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	content, err := fastSchemaSeedContent()
	if err != nil {
		return err
	}
	return writePrivateTestDatabaseSeed(path, content)
}

func writePrivateTestDatabaseSeed(path string, content []byte) error {
	directory := filepath.Dir(path)
	if err := createPrivateSQLiteDirectoryChain(directory); err != nil {
		return err
	}
	info, err := os.Lstat(directory)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || info.Mode().Perm()&0o077 != 0 {
		return errors.New("sqlite.test_seed_directory_not_private")
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	written, writeErr := file.Write(content)
	if writeErr == nil && written != len(content) {
		writeErr = io.ErrShortWrite
	}
	closeErr := file.Close()
	if writeErr == nil && closeErr == nil {
		return nil
	}
	return errors.Join(writeErr, closeErr, os.Remove(path))
}

// seedCachedTestDatabase amortizes only construction of a canonical fixture.
// Every caller still opens its private copy through the production Open path.
func seedCachedTestDatabase(
	t *testing.T,
	path string,
	label string,
	state *cachedTestDatabaseSeed,
	build func(*testing.T, string),
) {
	t.Helper()
	state.Do(func() {
		root, err := os.MkdirTemp("", "orquesta-sqlite-"+label+"-")
		if err != nil {
			state.err = err
			return
		}
		defer func() { state.err = errors.Join(state.err, os.RemoveAll(root)) }()
		templatePath := filepath.Join(root, "state", "template.sqlite")
		build(t, templatePath)
		if state.err = checkpointCachedTestDatabase(templatePath); state.err != nil {
			return
		}
		state.content, state.err = os.ReadFile(templatePath)
		if state.err == nil && len(state.content) == 0 {
			state.err = errors.New("empty cached test database")
		}
	})
	if state.err != nil {
		t.Fatalf("build cached %s database: %v", label, state.err)
	}
	if len(state.content) == 0 {
		t.Fatalf("build cached %s database: empty template", label)
	}
	if err := writePrivateTestDatabaseSeed(path, state.content); err != nil {
		t.Fatalf("copy cached %s database: %v", label, err)
	}
}

func checkpointCachedTestDatabase(path string) error {
	database, err := sql.Open(driverName, buildDSNWithDurability(
		path, testBusyTimeout.Milliseconds(), fastSQLiteTestDurability,
	))
	if err != nil {
		return err
	}
	_, checkpointErr := database.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	return errors.Join(checkpointErr, database.Close())
}

func fastSchemaSeedContent() ([]byte, error) {
	fastSchemaSeedState.Do(func() {
		root, err := os.MkdirTemp("", "orquesta-sqlite-fast-schema-")
		if err != nil {
			fastSchemaSeedState.err = err
			return
		}
		defer func() {
			fastSchemaSeedState.err = errors.Join(fastSchemaSeedState.err, os.RemoveAll(root))
		}()
		path := filepath.Join(root, "state", "template.sqlite")
		repository, err := openWithDurability(context.Background(), Options{
			Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 1,
		}, fastSQLiteTestDurability)
		if err != nil {
			fastSchemaSeedState.err = fmt.Errorf("migrate fast schema seed: %w", err)
			return
		}
		if _, err := repository.writer.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
			_ = repository.Close()
			fastSchemaSeedState.err = fmt.Errorf("checkpoint fast schema seed: %w", err)
			return
		}
		if err := repository.Close(); err != nil {
			fastSchemaSeedState.err = fmt.Errorf("close fast schema seed: %w", err)
			return
		}
		fastSchemaSeedState.content, fastSchemaSeedState.err = os.ReadFile(path)
		if fastSchemaSeedState.err == nil && len(fastSchemaSeedState.content) == 0 {
			fastSchemaSeedState.err = errors.New("empty fast schema seed")
		}
	})
	if fastSchemaSeedState.err != nil {
		return nil, fastSchemaSeedState.err
	}
	return append([]byte(nil), fastSchemaSeedState.content...), nil
}

func TestFastSchemaSeedNeverReplacesAnExistingDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "existing.sqlite")
	want := []byte("existing database sentinel")
	if err := os.WriteFile(path, want, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := seedFastTestDatabase(path); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(want) {
		t.Fatalf("existing database changed: got=%q err=%v", got, err)
	}
}

func TestFastSchemaSeedRejectsAnExistingPublicDirectory(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "public")
	if err := os.Mkdir(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "state.sqlite")
	if err := seedFastTestDatabase(path); err == nil {
		t.Fatal("fast schema seed repaired an existing public directory")
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("rejected seed left database behind: %v", err)
	}
}

func TestFastSchemaSeedCreatesIndependentLatestEmptyDatabases(t *testing.T) {
	first := filepath.Join(t.TempDir(), "first", "state.sqlite")
	second := filepath.Join(t.TempDir(), "second", "state.sqlite")
	if err := seedFastTestDatabase(first); err != nil {
		t.Fatal(err)
	}
	if err := seedFastTestDatabase(second); err != nil {
		t.Fatal(err)
	}
	firstInfo, err := os.Stat(first)
	if err != nil {
		t.Fatal(err)
	}
	secondInfo, err := os.Stat(second)
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(firstInfo, secondInfo) || firstInfo.Mode().Perm() != 0o600 || secondInfo.Mode().Perm() != 0o600 {
		t.Fatalf("seed files are not private independent copies: first=%v second=%v", firstInfo, secondInfo)
	}
	repository := openFastTestRepository(t, Options{
		Path: first, BusyTimeout: testBusyTimeout, MaxOpenConnections: 2,
	})
	var version, migrations, goals int
	if err := repository.db.QueryRow(`SELECT user_version,
 (SELECT COUNT(*) FROM schema_migrations),
 (SELECT COUNT(*) FROM goals) FROM pragma_user_version`).Scan(&version, &migrations, &goals); err != nil {
		t.Fatal(err)
	}
	if version != recoverySchemaLatest || migrations != recoverySchemaLatest || goals != 0 {
		t.Fatalf("seed state version=%d migrations=%d goals=%d", version, migrations, goals)
	}
}
