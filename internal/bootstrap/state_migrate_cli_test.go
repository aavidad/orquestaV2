package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"orquesta/internal/adapters/state/sqlite"
	"orquesta/internal/config"
	"orquesta/internal/i18n"
)

func TestStateMigrationCLIAlreadyTargetWALDoesNotMutateOrCreateSidecars(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "orquesta.toml")
	if err := os.WriteFile(configPath, []byte("# fixture\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	databasePath := filepath.Join(root, "state", "orquesta.sqlite")
	repository, err := sqlite.Open(context.Background(), sqlite.Options{
		Path: databasePath, BusyTimeout: time.Second, MaxOpenConnections: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.Close(); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(databasePath)
	if err != nil {
		t.Fatal(err)
	}

	snapshot := stateMigrationTestSnapshot(t, configPath, databasePath)
	catalog, err := i18n.LoadBundled()
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runStateMigration(context.Background(), []string{
		"--config", configPath, "--mode", "no-runtime", "--expect-from", "39", "--expect-to", "41",
	}, catalog, &stdout, &stderr, stateMigrationDependencies{
		loadSnapshot: func(context.Context, string) (config.Snapshot, error) { return snapshot, nil },
		migrate:      sqlite.MigrateStateOnly,
	})
	if code != 0 || stderr.Len() != 0 ||
		!strings.Contains(stdout.String(), `"state":"already_at_target"`) ||
		!strings.Contains(stdout.String(), `"read_only_verified":true`) {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	after, err := os.ReadFile(databasePath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, before) {
		t.Fatal("CLI already-at-target verification changed database bytes")
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		if _, err := os.Lstat(databasePath + suffix); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("sidecar %s err=%v", suffix, err)
		}
	}
}

func TestStateMigrationCLIComposesOnlyConfigAndMigrationAdapter(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "orquesta.toml")
	if err := os.WriteFile(configPath, []byte("# fixture\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	databasePath := filepath.Join(root, "state", "orquesta.sqlite")
	snapshot := stateMigrationTestSnapshot(t, configPath, databasePath)
	loadCalls, migrationCalls := 0, 0
	dependencies := stateMigrationDependencies{
		loadSnapshot: func(_ context.Context, got string) (config.Snapshot, error) {
			loadCalls++
			if got != configPath {
				t.Fatalf("config path=%q", got)
			}
			return snapshot, nil
		},
		migrate: func(_ context.Context, request sqlite.StateMigrationRequest) (sqlite.StateMigrationReceipt, error) {
			migrationCalls++
			if request.Options.Path != databasePath || request.Options.BusyTimeout != snapshot.StateSQLiteBusyTimeout() ||
				request.Options.MaxOpenConnections != int(snapshot.StateSQLiteMaxOpenConnections()) ||
				request.Mode != sqlite.StateMigrationModeNoRuntime || request.ExpectFrom != 39 || request.ExpectTo != 41 {
				t.Fatalf("migration request=%+v", request)
			}
			return sqlite.StateMigrationReceipt{
				SchemaVersion: 1, Mode: sqlite.StateMigrationModeNoRuntime,
				State: sqlite.StateMigrationMigrated, RequestedFrom: 39, TargetVersion: 41,
				ReadOnlyVerified: true,
			}, nil
		},
	}
	catalog, err := i18n.LoadBundled()
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runStateMigration(context.Background(), []string{
		"--config", configPath, "--mode", "no-runtime", "--expect-from", "39", "--expect-to", "41",
	}, catalog, &stdout, &stderr, dependencies)
	if code != 0 || loadCalls != 1 || migrationCalls != 1 || stderr.Len() != 0 ||
		!strings.Contains(stdout.String(), `"read_only_verified":true`) {
		t.Fatalf("code=%d load=%d migration=%d stdout=%q stderr=%q",
			code, loadCalls, migrationCalls, stdout.String(), stderr.String())
	}
}

func TestStateMigrationCLIPrintsDurableReceiptWhenPostcommitCheckIsPending(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "orquesta.toml")
	if err := os.WriteFile(configPath, []byte("# fixture\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	databasePath := filepath.Join(root, "state", "orquesta.sqlite")
	snapshot := stateMigrationTestSnapshot(t, configPath, databasePath)
	receipt := sqlite.StateMigrationReceipt{
		SchemaVersion: 1, Mode: sqlite.StateMigrationModeNoRuntime,
		State:         sqlite.StateMigrationCommittedPostcheckPending,
		RequestedFrom: 39, TargetVersion: 41,
		TargetSchemaRef: "schema:fixture", TargetLogicalSHA256: "sha256:fixture",
		DatabaseIdentity: "local-state:sha256:fixture", ReadOnlyVerified: false,
		PostCommitCauseCode: "sqlite.state_migration_postcommit_check_failed",
	}
	catalog, err := i18n.LoadBundled()
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runStateMigration(context.Background(), []string{
		"--config", configPath, "--mode", "no-runtime", "--expect-from", "39", "--expect-to", "41",
	}, catalog, &stdout, &stderr, stateMigrationDependencies{
		loadSnapshot: func(context.Context, string) (config.Snapshot, error) { return snapshot, nil },
		migrate: func(context.Context, sqlite.StateMigrationRequest) (sqlite.StateMigrationReceipt, error) {
			return sqlite.StateMigrationReceipt{}, &sqlite.CommittedStateMigrationError{
				Receipt: receipt, Cause: errors.New("test.private_cause"),
			}
		},
	})
	if code != 1 || !strings.Contains(stdout.String(), `"state":"committed_postcheck_pending"`) ||
		!strings.Contains(stdout.String(), `"read_only_verified":false`) ||
		!strings.Contains(stderr.String(), "code=cli.state_migration_committed_postcheck_pending") ||
		!strings.Contains(stderr.String(), "cause_code=sqlite.state_migration_committed_postcheck_pending") ||
		strings.Contains(stderr.String(), "test.private_cause") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestStateMigrationCLIRequiresExactNoRuntimeArgumentsAndAbsoluteRegularConfig(t *testing.T) {
	catalog, err := i18n.LoadBundled()
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runStateMigration(context.Background(), []string{"--help"}, catalog, &stdout, &stderr,
		stateMigrationDependencies{})
	if code != 0 || stderr.Len() != 0 || !strings.Contains(stdout.String(), "orquesta state migrate") {
		t.Fatalf("help code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}

	for _, arguments := range [][]string{
		{"--config", "relative.toml", "--mode", "no-runtime", "--expect-from", "39", "--expect-to", "41"},
		{"--config", "/missing/config.toml", "--mode", "runtime", "--expect-from", "39", "--expect-to", "41"},
		{"--config", "/missing/config.toml", "--mode", "no-runtime", "--expect-from", "40", "--expect-to", "41"},
		{"--config", "/missing/config.toml", "--mode", "no-runtime", "--expect-from", "39", "--expect-to", "42"},
	} {
		stdout.Reset()
		stderr.Reset()
		calls := 0
		code := runStateMigration(context.Background(), arguments, catalog, &stdout, &stderr,
			stateMigrationDependencies{
				loadSnapshot: func(context.Context, string) (config.Snapshot, error) {
					calls++
					return config.Snapshot{}, nil
				},
				migrate: func(context.Context, sqlite.StateMigrationRequest) (sqlite.StateMigrationReceipt, error) {
					calls++
					return sqlite.StateMigrationReceipt{}, nil
				},
			})
		if code != 2 || calls != 0 || stdout.Len() != 0 ||
			!strings.Contains(stderr.String(), "code=cli.state_migration_arguments_invalid") {
			t.Fatalf("args=%q code=%d calls=%d stdout=%q stderr=%q",
				arguments, code, calls, stdout.String(), stderr.String())
		}
	}
}

func TestStateMigrationCLIFailsClosedWithSafeSQLiteCause(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "orquesta.toml")
	if err := os.WriteFile(configPath, []byte("# fixture\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	databasePath := filepath.Join(root, "state", "orquesta.sqlite")
	snapshot := stateMigrationTestSnapshot(t, configPath, databasePath)
	catalog, err := i18n.LoadBundled()
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runStateMigration(context.Background(), []string{
		"--config", configPath, "--mode", "no-runtime", "--expect-from", "39", "--expect-to", "41",
	}, catalog, &stdout, &stderr, stateMigrationDependencies{
		loadSnapshot: func(context.Context, string) (config.Snapshot, error) { return snapshot, nil },
		migrate: func(context.Context, sqlite.StateMigrationRequest) (sqlite.StateMigrationReceipt, error) {
			return sqlite.StateMigrationReceipt{}, errors.New("sqlite.state_writer_active")
		},
	})
	if code != 1 || stdout.Len() != 0 ||
		!strings.Contains(stderr.String(), "code=cli.state_migration_failed") ||
		!strings.Contains(stderr.String(), "cause_code=sqlite.state_writer_active") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func stateMigrationTestSnapshot(t *testing.T, configPath, databasePath string) config.Snapshot {
	t.Helper()
	source := "[state.sqlite]\npath = " + strconv.Quote(databasePath) + "\n"
	snapshot, err := config.Resolve(config.ResolveOptions{
		TOML: []byte(source), Environment: map[string]string{}, SourcePath: configPath,
	})
	if err != nil {
		t.Fatalf("config.Resolve() error=%v", err)
	}
	return snapshot
}
