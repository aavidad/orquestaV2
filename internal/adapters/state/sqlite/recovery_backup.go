package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"orquesta/internal/application"

	moderncsqlite "modernc.org/sqlite"
)

func (recovery *Recovery) CreateBackup(ctx context.Context) (application.BackupReceipt, error) {
	if recovery == nil {
		return application.BackupReceipt{}, invalid(errors.New("sqlite.recovery_closed"))
	}
	recovery.mu.Lock()
	defer recovery.mu.Unlock()
	if err := recovery.beginOperation(ctx); err != nil {
		return application.BackupReceipt{}, err
	}

	root, err := recovery.rootLocks.openedRoot(recovery.backupRoot)
	if err != nil {
		return application.BackupReceipt{}, invalid(err)
	}
	recovery.sequence++
	stageName := ".backup-" + strconv.FormatInt(recovery.now().UTC().UnixNano(), 10) + "-" +
		strconv.FormatUint(recovery.sequence, 10) + ".next"
	if err := root.Mkdir(stageName, 0o700); err != nil {
		return application.BackupReceipt{}, invalid(err)
	}
	stagePublished := false
	defer func() {
		if !stagePublished {
			_ = root.RemoveAll(stageName)
		}
	}()
	payloadName := filepath.Join(stageName, recoveryPayloadName)
	file, err := root.OpenFile(payloadName, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err != nil {
		return application.BackupReceipt{}, invalid(err)
	}
	fileOpen := true
	defer func() {
		if fileOpen {
			_ = file.Close()
		}
	}()
	if err := recovery.onlineBackup(ctx, file); err != nil {
		return application.BackupReceipt{}, err
	}
	if err := file.Sync(); err != nil {
		return application.BackupReceipt{}, invalid(err)
	}
	payloadInfo, err := bindOpenPrivateRegularFile(root, payloadName, file)
	if err != nil {
		return application.BackupReceipt{}, invalid(err)
	}
	manifest, payloadDigest, err := recovery.buildManifest(ctx, root, payloadName, file, payloadInfo)
	if err != nil {
		return application.BackupReceipt{}, err
	}
	if err := file.Close(); err != nil {
		return application.BackupReceipt{}, invalid(err)
	}
	fileOpen = false
	manifestContent, err := json.Marshal(manifest)
	if err != nil {
		return application.BackupReceipt{}, invalid(err)
	}
	if err := writeExclusiveSynced(root, filepath.Join(stageName, recoveryManifestName), manifestContent, 0o600); err != nil {
		return application.BackupReceipt{}, invalid(err)
	}
	if err := syncPrivateDirectory(root, stageName); err != nil {
		return application.BackupReceipt{}, invalid(err)
	}
	if err := recovery.hit("after_backup_sync"); err != nil {
		return application.BackupReceipt{}, err
	}

	finalName := strings.TrimPrefix(payloadDigest, "sha256:")
	if _, err := root.Lstat(finalName); err == nil {
		return recovery.receiptForExisting(ctx, manifest.BackupRef)
	} else if !errors.Is(err, os.ErrNotExist) {
		return application.BackupReceipt{}, invalid(err)
	}
	if err := recovery.rootLocks.publishNoReplace(recovery.backupRoot, stageName, finalName); err != nil {
		return application.BackupReceipt{}, invalid(err)
	}
	stagePublished = true
	if err := recovery.rootLocks.sync(recovery.backupRoot); err != nil {
		recovery.removePrivateTree(root, finalName, recovery.backupRoot)
		return application.BackupReceipt{}, invalid(err)
	}
	if err := recovery.hit("after_backup_publish"); err != nil {
		recovery.removePrivateTree(root, finalName, recovery.backupRoot)
		return application.BackupReceipt{}, err
	}
	if err := recovery.rootLocks.verify(recovery.backupRoot, recovery.restoreRoot); err != nil {
		recovery.removePrivateTree(root, finalName, recovery.backupRoot)
		return application.BackupReceipt{}, invalid(err)
	}
	return application.BackupReceipt{
		Ref: mustBackupRef(manifest.BackupRef), ManifestSHA256: bytesSHA256(manifestContent),
		MediaType: manifest.MediaType, Size: manifest.Size, SchemaRef: manifest.SchemaRef,
		CreatedAt: manifest.CreatedAt,
	}, nil
}

// onlineBackup keeps modernc's driver connection and Backup object entirely
// inside one Raw callback. Finish always runs there; Commit is never used.
func (recovery *Recovery) onlineBackup(ctx context.Context, destination *os.File) error {
	database, err := recovery.repository.database()
	if err != nil {
		return err
	}
	connection, err := database.Conn(ctx)
	if err != nil {
		return mapDatabaseError(err)
	}
	defer connection.Close()
	destinationPath, err := recoveryDescriptorPath(destination)
	if err != nil {
		return invalid(err)
	}
	destinationURI := sqliteFileURI(destinationPath, false)
	return mapDatabaseError(connection.Raw(func(driverConnection any) (finalErr error) {
		provider, ok := driverConnection.(interface {
			NewBackup(string) (*moderncsqlite.Backup, error)
		})
		if !ok {
			return errors.New("sqlite.online_backup_unsupported")
		}
		backup, err := provider.NewBackup(destinationURI)
		if err != nil {
			return err
		}
		defer func() {
			if finishErr := backup.Finish(); finalErr == nil {
				finalErr = finishErr
			}
		}()
		first := true
		for {
			if err := ctx.Err(); err != nil {
				return err
			}
			more, err := backup.Step(128)
			if err != nil {
				return err
			}
			if first {
				first = false
				if err := recovery.hit("after_first_backup_step"); err != nil {
					return err
				}
			}
			if !more {
				return nil
			}
		}
	}))
}

func (recovery *Recovery) buildManifest(
	ctx context.Context,
	root *os.Root,
	payloadName string,
	payload *os.File,
	info os.FileInfo,
) (backupManifest, string, error) {
	payloadDigest, err := fileSHA256FromFile(ctx, payload, info)
	if err != nil {
		return backupManifest{}, "", invalid(err)
	}
	if err := verifyOpenPrivateRegularFile(root, payloadName, payload, info); err != nil {
		return backupManifest{}, "", invalid(err)
	}
	database, err := openRecoveryDatabase(payload)
	if err != nil {
		return backupManifest{}, "", invalid(err)
	}
	schemaRef, logicalDigest, err := validateRecoveryDatabase(ctx, database)
	closeErr := database.Close()
	if err != nil {
		return backupManifest{}, "", err
	}
	if closeErr != nil {
		return backupManifest{}, "", invalid(closeErr)
	}
	if err := verifyOpenPrivateRegularFile(root, payloadName, payload, info); err != nil {
		return backupManifest{}, "", invalid(err)
	}
	createdAt := recovery.now().UTC()
	return backupManifest{
		SchemaVersion: 1, BackupRef: backupRefPrefix + strings.TrimPrefix(payloadDigest, "sha256:"),
		PayloadSHA256: payloadDigest, LogicalSHA256: logicalDigest, MediaType: recoveryMediaType,
		Size: info.Size(), SchemaRef: schemaRef, CreatedAt: createdAt,
		CredentialsIncluded: false, ArtifactBlobsIncluded: false,
	}, payloadDigest, nil
}
