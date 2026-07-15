package sqlite

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"orquesta/internal/application"
)

const (
	recoveryMediaType    = "application/vnd.sqlite3"
	recoveryManifestName = "backup.manifest.json"
	recoveryPayloadName  = "state.sqlite"
	backupRefPrefix      = "backup:sha256:"
	schemaRefPrefix      = "state-schema:sqlite:sha256:"
)

type RecoveryOptions struct {
	Repository  *Repository
	BackupRoot  string
	RestoreRoot string
	Now         func() time.Time
	Failpoint   func(string) error
}

type Recovery struct {
	mu          sync.Mutex
	repository  *Repository
	backupRoot  string
	restoreRoot string
	now         func() time.Time
	failpoint   func(string) error
	rootLocks   *recoveryRootLocks
	closed      bool
	sequence    uint64
}

var _ application.StateRecovery = (*Recovery)(nil)

type backupManifest struct {
	SchemaVersion         int       `json:"schema_version"`
	BackupRef             string    `json:"backup_ref"`
	PayloadSHA256         string    `json:"payload_sha256"`
	LogicalSHA256         string    `json:"logical_sha256"`
	MediaType             string    `json:"media_type"`
	Size                  int64     `json:"size"`
	SchemaRef             string    `json:"schema_ref"`
	CreatedAt             time.Time `json:"created_at"`
	CredentialsIncluded   bool      `json:"credentials_included"`
	ArtifactBlobsIncluded bool      `json:"artifact_blobs_included"`
}

type inspectedBackup struct {
	manifest       backupManifest
	manifestDigest string
	payloadPath    string
}

func NewRecovery(options RecoveryOptions) (*Recovery, error) {
	if options.Repository == nil {
		return nil, invalid(errors.New("sqlite.recovery_repository_required"))
	}
	if _, err := options.Repository.database(); err != nil {
		return nil, err
	}
	// Preflight both roots before creating either, avoiding partial setup when
	// one supplied namespace is unsafe.
	backupRoot, err := preflightRecoveryRoot(options.BackupRoot)
	if err != nil {
		return nil, invalid(fmt.Errorf("sqlite.backup_root_invalid: %w", err))
	}
	restoreRoot, err := preflightRecoveryRoot(options.RestoreRoot)
	if err != nil {
		return nil, invalid(fmt.Errorf("sqlite.restore_root_invalid: %w", err))
	}
	activeDirectory := filepath.Dir(options.Repository.path)
	if pathsOverlap(backupRoot, restoreRoot) || pathsOverlap(backupRoot, activeDirectory) ||
		pathsOverlap(restoreRoot, activeDirectory) {
		return nil, invalid(errors.New("sqlite.recovery_root_overlap"))
	}
	if err := createPrivateSQLiteDirectoryChain(backupRoot); err != nil {
		return nil, invalid(err)
	}
	if err := createPrivateSQLiteDirectoryChain(restoreRoot); err != nil {
		return nil, invalid(err)
	}
	if err := validatePrivateDirectory(backupRoot); err != nil {
		return nil, invalid(err)
	}
	if err := validatePrivateDirectory(restoreRoot); err != nil {
		return nil, invalid(err)
	}
	rootLocks, err := acquireRecoveryRootLocks(backupRoot, restoreRoot)
	if err != nil {
		return nil, conflict(fmt.Errorf("sqlite.recovery_root_locked: %w", err))
	}
	if err := cleanupRecoveryStages(rootLocks, backupRoot, restoreRoot); err != nil {
		_ = rootLocks.Close()
		return nil, invalid(err)
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}
	return &Recovery{
		repository: options.Repository, backupRoot: backupRoot, restoreRoot: restoreRoot,
		now: now, failpoint: options.Failpoint, rootLocks: rootLocks,
	}, nil
}

func (recovery *Recovery) Close() error {
	if recovery == nil {
		return nil
	}
	recovery.mu.Lock()
	defer recovery.mu.Unlock()
	if recovery.closed {
		return nil
	}
	recovery.closed = true
	locks := recovery.rootLocks
	recovery.rootLocks = nil
	return locks.Close()
}

// TargetPath is a concrete adapter diagnostic used by offline operators. It is
// deliberately absent from the neutral StateRecovery port.
func (recovery *Recovery) TargetPath(targetRef application.RecoveryTargetRef) (string, error) {
	if recovery == nil {
		return "", invalid(errors.New("sqlite.recovery_closed"))
	}
	recovery.mu.Lock()
	defer recovery.mu.Unlock()
	if recovery.closed {
		return "", invalid(errors.New("sqlite.recovery_closed"))
	}
	if _, err := application.NewRecoveryTargetRef(targetRef.String()); err != nil {
		return "", invalid(err)
	}
	if err := recovery.rootLocks.verify(recovery.restoreRoot); err != nil {
		return "", invalid(err)
	}
	return filepath.Join(recovery.restoreRoot, recoveryTargetName(targetRef)), nil
}

func (recovery *Recovery) beginOperation(ctx context.Context) error {
	if recovery.closed {
		return invalid(errors.New("sqlite.recovery_closed"))
	}
	if ctx == nil {
		return invalid(errors.New("sqlite.recovery_context_required"))
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if recovery.now().IsZero() {
		return invalid(errors.New("sqlite.recovery_clock_invalid"))
	}
	if err := recovery.rootLocks.verify(recovery.backupRoot, recovery.restoreRoot); err != nil {
		return invalid(err)
	}
	return nil
}

func (recovery *Recovery) hit(stage string) error {
	if recovery.failpoint == nil {
		return nil
	}
	return recovery.failpoint(stage)
}

func (recovery *Recovery) removePrivateTree(root *os.Root, name, directory string) {
	_ = root.RemoveAll(name)
	_ = recovery.rootLocks.sync(directory)
}
