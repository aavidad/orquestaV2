package sqlite

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"orquesta/internal/application"
)

func (recovery *Recovery) RestoreBackup(
	ctx context.Context,
	backupRef application.BackupRef,
	targetRef application.RecoveryTargetRef,
) (application.RestoreReceipt, error) {
	if recovery == nil {
		return application.RestoreReceipt{}, invalid(errors.New("sqlite.recovery_closed"))
	}
	recovery.mu.Lock()
	defer recovery.mu.Unlock()
	if err := recovery.beginOperation(ctx); err != nil {
		return application.RestoreReceipt{}, err
	}
	if _, err := application.NewRecoveryTargetRef(targetRef.String()); err != nil {
		return application.RestoreReceipt{}, invalid(err)
	}
	inspected, err := recovery.inspectBackup(ctx, backupRef)
	if err != nil {
		return application.RestoreReceipt{}, err
	}
	root, err := os.OpenRoot(recovery.restoreRoot)
	if err != nil {
		return application.RestoreReceipt{}, invalid(err)
	}
	defer root.Close()
	if err := recovery.rootLocks.verify(recovery.restoreRoot); err != nil {
		return application.RestoreReceipt{}, invalid(err)
	}
	targetName := recoveryTargetName(targetRef)
	if _, err := root.Lstat(targetName); err == nil {
		return application.RestoreReceipt{}, conflict(errors.New("sqlite.restore_target_exists"))
	} else if !errors.Is(err, os.ErrNotExist) {
		return application.RestoreReceipt{}, invalid(err)
	}
	stageName := "." + strings.TrimSuffix(targetName, ".sqlite") + ".partial"
	if _, err := root.Lstat(stageName); err == nil {
		return application.RestoreReceipt{}, conflict(errors.New("sqlite.restore_stage_exists"))
	} else if !errors.Is(err, os.ErrNotExist) {
		return application.RestoreReceipt{}, invalid(err)
	}
	stagePath := filepath.Join(recovery.restoreRoot, stageName)
	if err := copyExclusive(ctx, inspected.payloadPath, stagePath); err != nil {
		return application.RestoreReceipt{}, err
	}
	published := false
	defer func() {
		if !published {
			_ = root.Remove(stageName)
		}
	}()
	digest, err := fileSHA256(ctx, stagePath)
	if err != nil {
		return application.RestoreReceipt{}, invalid(fmt.Errorf("sqlite.restore_copy_invalid: %w", err))
	}
	if digest != inspected.manifest.PayloadSHA256 {
		return application.RestoreReceipt{}, invalid(errors.New("sqlite.restore_copy_digest_invalid"))
	}
	database, err := openRecoveryDatabase(stagePath)
	if err != nil {
		return application.RestoreReceipt{}, invalid(err)
	}
	schemaRef, logicalDigest, validateErr := validateRecoveryDatabase(ctx, database)
	closeErr := database.Close()
	if validateErr != nil {
		return application.RestoreReceipt{}, validateErr
	}
	if closeErr != nil {
		return application.RestoreReceipt{}, invalid(closeErr)
	}
	if schemaRef != inspected.manifest.SchemaRef || logicalDigest != inspected.manifest.LogicalSHA256 {
		return application.RestoreReceipt{}, invalid(errors.New("sqlite.restore_semantics_changed"))
	}
	if err := recovery.hit("after_restore_sync"); err != nil {
		return application.RestoreReceipt{}, err
	}
	if err := recovery.rootLocks.publishNoReplace(recovery.restoreRoot, stageName, targetName); err != nil {
		return application.RestoreReceipt{}, invalid(err)
	}
	published = true
	if err := recovery.rootLocks.sync(recovery.restoreRoot); err != nil {
		recovery.removePrivateTree(root, targetName, recovery.restoreRoot)
		return application.RestoreReceipt{}, invalid(err)
	}
	if err := recovery.hit("after_restore_publish"); err != nil {
		recovery.removePrivateTree(root, targetName, recovery.restoreRoot)
		return application.RestoreReceipt{}, err
	}
	return application.RestoreReceipt{
		BackupRef: backupRef, TargetRef: targetRef, ManifestSHA256: inspected.manifestDigest,
		SchemaRef: inspected.manifest.SchemaRef, RestoredAt: recovery.now().UTC(),
	}, nil
}
