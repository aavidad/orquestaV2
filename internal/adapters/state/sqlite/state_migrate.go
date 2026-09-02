package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
)

type StateMigrationMode string

const StateMigrationModeNoRuntime StateMigrationMode = "no-runtime"

type StateMigrationState string

const (
	StateMigrationMigrated                  StateMigrationState = "migrated"
	StateMigrationAlreadyAtTarget           StateMigrationState = "already_at_target"
	StateMigrationCommittedPostcheckPending StateMigrationState = "committed_postcheck_pending"
)

type StateMigrationRequest struct {
	Options    Options
	Mode       StateMigrationMode
	ExpectFrom int
	ExpectTo   int
}

type StateMigrationReceipt struct {
	SchemaVersion       int                 `json:"schema_version"`
	Mode                StateMigrationMode  `json:"mode"`
	State               StateMigrationState `json:"state"`
	RequestedFrom       int                 `json:"requested_from"`
	TargetVersion       int                 `json:"target_version"`
	TargetSchemaRef     string              `json:"target_schema_ref"`
	TargetLogicalSHA256 string              `json:"target_logical_sha256"`
	DatabaseIdentity    string              `json:"database_identity"`
	ReadOnlyVerified    bool                `json:"read_only_verified"`
	PostCommitCauseCode string              `json:"post_commit_cause_code,omitempty"`
}

type CommittedStateMigrationError struct {
	Receipt StateMigrationReceipt
	Cause   error
}

func (err *CommittedStateMigrationError) Error() string {
	return "sqlite.state_migration_committed_postcheck_pending"
}

func (err *CommittedStateMigrationError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Cause
}

// MigrateStateOnly is the one-shot administrative owner for an exact V39 to
// V41 upgrade. It shares the runtime writer lock but never constructs a
// Repository, worker, provider, scheduler, socket or Agent MicroVM client.
func MigrateStateOnly(ctx context.Context, request StateMigrationRequest) (StateMigrationReceipt, error) {
	return migrateStateOnly(ctx, request, stateMigrationHooks{})
}

type stateMigrationHooks struct {
	afterStep   func(int) error
	afterCommit func() error
}

func migrateStateOnly(
	ctx context.Context,
	request StateMigrationRequest,
	hooks stateMigrationHooks,
) (StateMigrationReceipt, error) {
	var receipt StateMigrationReceipt
	if ctx == nil || request.Mode != StateMigrationModeNoRuntime ||
		request.ExpectFrom != recoverySchemaV38LaunchRuntimeDigests ||
		request.ExpectTo != recoverySchemaV38ExpiredLaunchContinuation ||
		recoverySchemaLatest != recoverySchemaV38ExpiredLaunchContinuation ||
		!filepath.IsAbs(request.Options.Path) {
		return receipt, invalid(errors.New("sqlite.state_migration_request_invalid"))
	}
	path, busyMilliseconds, err := validateOptions(request.Options)
	if err != nil {
		return receipt, invalid(err)
	}
	target, err := openStateMigrationTarget(path, uint32(os.Geteuid()))
	if err != nil {
		return receipt, invalid(err)
	}
	targetClosed := false
	defer func() {
		if !targetClosed {
			_ = target.Close()
		}
	}()
	if err := target.rejectSidecars(); err != nil {
		return receipt, invalid(err)
	}
	identity, identitySupported, err := localStateIdentityForHandle(target.file, path)
	if err != nil {
		return receipt, invalid(err)
	}
	receipt = StateMigrationReceipt{
		SchemaVersion:    1,
		Mode:             request.Mode,
		RequestedFrom:    request.ExpectFrom,
		TargetVersion:    request.ExpectTo,
		DatabaseIdentity: identity,
	}
	if err := acquireStateWriterLock(target.file); err != nil {
		return receipt, invalid(err)
	}
	if err := target.rejectSidecars(); err != nil {
		return receipt, invalid(err)
	}
	if err := target.verify(); err != nil {
		return receipt, invalid(err)
	}
	schemaRef, logicalDigest, current, err := verifyStateMigrationReadOnly(
		ctx, path, busyMilliseconds, target, identitySupported,
	)
	if err != nil {
		return receipt, err
	}
	if current == request.ExpectTo {
		receipt.State = StateMigrationAlreadyAtTarget
		receipt.TargetSchemaRef = schemaRef
		receipt.TargetLogicalSHA256 = logicalDigest
		receipt.ReadOnlyVerified = true
		if err := target.Close(); err != nil {
			return StateMigrationReceipt{}, invalid(err)
		}
		targetClosed = true
		return receipt, nil
	}
	if current != request.ExpectFrom {
		return receipt, invalid(errors.New("sqlite.state_migration_source_version_mismatch"))
	}

	connector, err := newLocalStateConnector(
		buildStateMigrationDSN(path, busyMilliseconds), target.file, identitySupported,
	)
	if err != nil {
		return receipt, invalid(err)
	}
	database := sql.OpenDB(connector)
	database.SetMaxOpenConns(1)
	database.SetMaxIdleConns(1)
	closed := false
	defer func() {
		if !closed {
			_ = database.Close()
		}
	}()
	if err := database.PingContext(ctx); err != nil {
		return receipt, mapDatabaseError(err)
	}

	committed := false
	policy := migrationPolicy{
		bounded: true, expectFrom: request.ExpectFrom, expectTo: request.ExpectTo,
		afterStep: hooks.afterStep,
		before: func(ctx context.Context, transaction *sql.Tx, expected int) error {
			_, _, version, err := validateRecoveryTransaction(ctx, transaction)
			if err != nil {
				return err
			}
			if version != expected {
				return invalid(errors.New("sqlite.state_migration_source_version_mismatch"))
			}
			return nil
		},
		after: func(ctx context.Context, transaction *sql.Tx, expected int) error {
			schemaRef, logicalDigest, version, err := validateRecoveryTransaction(ctx, transaction)
			if err != nil {
				return err
			}
			if version != expected {
				return invalid(errors.New("sqlite.state_migration_target_version_mismatch"))
			}
			receipt.TargetSchemaRef = schemaRef
			receipt.TargetLogicalSHA256 = logicalDigest
			return nil
		},
		afterCommit: func() { committed = true },
	}
	if err := applyMigrationsWithPolicy(ctx, database, policy); err != nil {
		if committed {
			return StateMigrationReceipt{}, committedStateMigrationError(receipt, err)
		}
		return StateMigrationReceipt{}, err
	}
	if hooks.afterCommit != nil {
		if err := hooks.afterCommit(); err != nil {
			return StateMigrationReceipt{}, committedStateMigrationError(receipt, err)
		}
	}
	if err := target.verify(); err != nil {
		return StateMigrationReceipt{}, committedStateMigrationError(receipt, invalid(err))
	}
	if err := database.Close(); err != nil {
		return StateMigrationReceipt{}, committedStateMigrationError(receipt, mapDatabaseError(err))
	}
	closed = true
	if err := target.syncParent(); err != nil {
		return StateMigrationReceipt{}, committedStateMigrationError(receipt, invalid(err))
	}
	if err := target.rejectSidecars(); err != nil {
		return StateMigrationReceipt{}, committedStateMigrationError(receipt, invalid(err))
	}

	resultSchemaRef, resultLogicalDigest, version, err := verifyStateMigrationReadOnly(
		ctx, path, busyMilliseconds, target, identitySupported,
	)
	if err != nil {
		return StateMigrationReceipt{}, committedStateMigrationError(receipt, err)
	}
	if version != request.ExpectTo || resultSchemaRef != receipt.TargetSchemaRef ||
		resultLogicalDigest != receipt.TargetLogicalSHA256 {
		return StateMigrationReceipt{}, committedStateMigrationError(
			receipt, invalid(errors.New("sqlite.state_migration_read_only_reopen_invalid")),
		)
	}

	receipt.State = StateMigrationMigrated
	receipt.ReadOnlyVerified = true
	if err := target.Close(); err != nil {
		return StateMigrationReceipt{}, committedStateMigrationError(receipt, invalid(err))
	}
	targetClosed = true
	return receipt, nil
}

func verifyStateMigrationReadOnly(
	ctx context.Context,
	path string,
	busyMilliseconds int64,
	target *stateMigrationTarget,
	identitySupported bool,
) (string, string, int, error) {
	if err := target.verify(); err != nil {
		return "", "", 0, invalid(err)
	}
	readOnly, err := openStateMigrationReadOnly(path, busyMilliseconds, target.file, identitySupported)
	if err != nil {
		return "", "", 0, err
	}
	resultSchemaRef, resultLogicalDigest, validationErr := validateRecoveryDatabase(ctx, readOnly)
	var queryOnly, version int
	if validationErr == nil {
		validationErr = readOnly.QueryRowContext(ctx, "PRAGMA query_only").Scan(&queryOnly)
	}
	if validationErr == nil {
		validationErr = readOnly.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version)
	}
	closeErr := readOnly.Close()
	if validationErr != nil {
		return "", "", 0, validationErr
	}
	if closeErr != nil {
		return "", "", 0, mapDatabaseError(closeErr)
	}
	if queryOnly != 1 {
		return "", "", 0, invalid(errors.New("sqlite.state_migration_read_only_reopen_invalid"))
	}
	if err := target.verify(); err != nil {
		return "", "", 0, invalid(err)
	}
	if err := target.rejectSidecars(); err != nil {
		return "", "", 0, invalid(err)
	}
	return resultSchemaRef, resultLogicalDigest, version, nil
}

func committedStateMigrationError(receipt StateMigrationReceipt, cause error) error {
	receipt.State = StateMigrationCommittedPostcheckPending
	receipt.ReadOnlyVerified = false
	receipt.PostCommitCauseCode = "sqlite.state_migration_postcommit_check_failed"
	return &CommittedStateMigrationError{Receipt: receipt, Cause: cause}
}

func buildStateMigrationDSN(path string, busyMilliseconds int64) string {
	dsn := &url.URL{Scheme: "file", Path: path}
	query := dsn.Query()
	query.Add("_pragma", "busy_timeout("+strconv.FormatInt(busyMilliseconds, 10)+")")
	query.Add("_pragma", "foreign_keys(1)")
	query.Add("_pragma", "synchronous("+string(fullSQLiteDurability)+")")
	query.Set("_txlock", "exclusive")
	dsn.RawQuery = query.Encode()
	return dsn.String()
}

func openStateMigrationReadOnly(
	path string,
	busyMilliseconds int64,
	identityHandle *os.File,
	identitySupported bool,
) (*sql.DB, error) {
	dsn := &url.URL{Scheme: "file", Path: path}
	query := dsn.Query()
	query.Set("mode", "ro")
	query.Set("immutable", "1")
	query.Add("_pragma", "busy_timeout("+strconv.FormatInt(busyMilliseconds, 10)+")")
	query.Add("_pragma", "foreign_keys(1)")
	query.Add("_pragma", "query_only(1)")
	dsn.RawQuery = query.Encode()
	connector, err := newLocalStateConnector(dsn.String(), identityHandle, identitySupported)
	if err != nil {
		return nil, invalid(err)
	}
	database := sql.OpenDB(connector)
	database.SetMaxOpenConns(1)
	database.SetMaxIdleConns(1)
	return database, nil
}
