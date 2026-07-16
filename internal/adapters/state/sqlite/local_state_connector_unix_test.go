//go:build unix

package sqlite

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	moderncsqlite "modernc.org/sqlite"
)

func TestLocalStateConnectorRejectsConnectionOpenedAcrossABASwap(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "state.sqlite")
	writeLocalIdentityProbeDatabase(t, path, 41)
	witness, _, supported, err := captureLocalStateIdentity(path)
	if err != nil {
		t.Fatal(err)
	}
	defer witness.Close()
	if !supported {
		t.Skip("host does not expose reliable local file identity")
	}

	capturedPath, replacementPath := path+".captured", path+".replacement"
	if err := os.Rename(path, capturedPath); err != nil {
		t.Fatal(err)
	}
	writeLocalIdentityProbeDatabase(t, path, 99)
	validated, err := newLocalStateConnector(buildDSN(path, 1000), witness, true)
	if err != nil {
		t.Fatal(err)
	}
	local := validated.(*localStateConnector)
	validate := local.validate
	local.validate = func(control moderncsqlite.FileControl) error {
		if err := os.Rename(path, replacementPath); err != nil {
			return err
		}
		if err := os.Rename(capturedPath, path); err != nil {
			return err
		}
		return validate(control)
	}
	database := sql.OpenDB(validated)
	defer database.Close()
	if err := database.PingContext(ctx); err == nil || !strings.Contains(err.Error(), "local_state_connection_identity_changed") {
		t.Fatalf("ABA-opened replacement entered pool: %v", err)
	}
	if version := readLocalIdentityProbeVersion(t, replacementPath); version != 99 {
		t.Fatalf("replacement changed across rejected connection: version=%d", version)
	}
	if mode := readLocalIdentityProbeJournalMode(t, replacementPath); mode != "delete" {
		t.Fatalf("replacement pragma ran before identity validation: journal_mode=%q", mode)
	}
	assertLocalIdentityProbeHasNoSidecars(t, path)
	assertLocalIdentityProbeHasNoSidecars(t, replacementPath)
}

func TestLocalStateConnectorDoesNotCreateMissingCapturedPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.sqlite")
	writeLocalIdentityProbeDatabase(t, path, 12)
	witness, _, supported, err := captureLocalStateIdentity(path)
	if err != nil {
		t.Fatal(err)
	}
	defer witness.Close()
	if !supported {
		t.Skip("host does not expose reliable local file identity")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	connector, err := newLocalStateConnector(buildDSN(path, 1000), witness, true)
	if err != nil {
		t.Fatal(err)
	}
	database := sql.OpenDB(connector)
	defer database.Close()
	if err := database.PingContext(context.Background()); err == nil {
		t.Fatal("missing captured path was recreated")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("validated connector created missing database: %v", err)
	}
	assertLocalIdentityProbeHasNoSidecars(t, path)
}

func TestLocalStateConnectorRejectsLazyConnectionAfterReplacement(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "state.sqlite")
	writeLocalIdentityProbeDatabase(t, path, 7)
	witness, _, supported, err := captureLocalStateIdentity(path)
	if err != nil {
		t.Fatal(err)
	}
	defer witness.Close()
	if !supported {
		t.Skip("host does not expose reliable local file identity")
	}
	connector, err := newLocalStateConnector(buildDSN(path, 1000), witness, true)
	if err != nil {
		t.Fatal(err)
	}
	database := sql.OpenDB(connector)
	database.SetMaxOpenConns(2)
	database.SetMaxIdleConns(2)
	defer database.Close()
	held, err := database.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()

	capturedPath := path + ".captured"
	if err := os.Rename(path, capturedPath); err != nil {
		t.Fatal(err)
	}
	writeLocalIdentityProbeDatabase(t, path, 8)
	if _, err := database.Conn(ctx); err == nil || !strings.Contains(err.Error(), "local_state_connection_identity_changed") {
		t.Fatalf("lazy replacement connection entered pool: %v", err)
	}
}

func TestLocalStateConnectorKeepsOneCanonicalWALNamespace(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "state.sqlite")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	witness, _, supported, err := captureLocalStateIdentity(path)
	if err != nil {
		t.Fatal(err)
	}
	defer witness.Close()
	connector, err := newLocalStateConnector(buildDSN(path, 1000), witness, supported)
	if err != nil {
		t.Fatal(err)
	}
	first, second := sql.OpenDB(connector), sql.OpenDB(connector)
	defer first.Close()
	defer second.Close()
	for index, database := range []*sql.DB{first, second} {
		var openedPath string
		if err := database.QueryRowContext(ctx,
			`SELECT file FROM pragma_database_list WHERE name = 'main'`).Scan(&openedPath); err != nil {
			t.Fatal(err)
		}
		if filepath.Clean(openedPath) != filepath.Clean(path) {
			t.Fatalf("pool %d opened alias %q instead of canonical %q", index, openedPath, path)
		}
	}
	if _, err := first.ExecContext(ctx, `CREATE TABLE wal_probe(value INTEGER PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	if _, err := first.ExecContext(ctx, `INSERT INTO wal_probe(value) VALUES (1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := second.ExecContext(ctx, `INSERT INTO wal_probe(value) VALUES (2)`); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := first.QueryRowContext(ctx, `SELECT COUNT(*) FROM wal_probe`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("canonical WAL lost a committed write: count=%d err=%v", count, err)
	}
}

func writeLocalIdentityProbeDatabase(t *testing.T, path string, version int) {
	t.Helper()
	database, err := sql.Open(driverName, path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`PRAGMA user_version = ` + strconv.Itoa(version)); err != nil {
		_ = database.Close()
		t.Fatal(err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
}

func readLocalIdentityProbeVersion(t *testing.T, path string) int {
	t.Helper()
	database, err := sql.Open(driverName, path)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var version int
	if err := database.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	return version
}

func readLocalIdentityProbeJournalMode(t *testing.T, path string) string {
	t.Helper()
	database, err := sql.Open(driverName, path)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var mode string
	if err := database.QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil {
		t.Fatal(err)
	}
	return strings.ToLower(mode)
}

func assertLocalIdentityProbeHasNoSidecars(t *testing.T, path string) {
	t.Helper()
	for _, suffix := range []string{"-journal", "-wal", "-shm"} {
		if _, err := os.Stat(path + suffix); !os.IsNotExist(err) {
			t.Fatalf("unexpected sqlite sidecar %q: %v", path+suffix, err)
		}
	}
}
