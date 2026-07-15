package sqlite

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"orquesta/internal/application"
)

func (recovery *Recovery) VerifyBackup(
	ctx context.Context,
	backupRef application.BackupRef,
) (application.BackupVerification, error) {
	if recovery == nil {
		return application.BackupVerification{}, invalid(errors.New("sqlite.recovery_closed"))
	}
	recovery.mu.Lock()
	defer recovery.mu.Unlock()
	if err := recovery.beginOperation(ctx); err != nil {
		return application.BackupVerification{}, err
	}
	inspected, err := recovery.inspectBackup(ctx, backupRef)
	if err != nil {
		return application.BackupVerification{}, err
	}
	defer inspected.Close()
	if err := inspected.Close(); err != nil {
		return application.BackupVerification{}, invalid(err)
	}
	if err := recovery.rootLocks.verify(recovery.backupRoot, recovery.restoreRoot); err != nil {
		return application.BackupVerification{}, invalid(err)
	}
	return application.BackupVerification{
		BackupRef: backupRef, ManifestSHA256: inspected.manifestDigest,
		SchemaRef: inspected.manifest.SchemaRef, VerifiedAt: recovery.now().UTC(),
	}, nil
}

func (recovery *Recovery) inspectBackup(ctx context.Context, backupRef application.BackupRef) (inspectedBackup, error) {
	validatedRef, err := application.NewBackupRef(backupRef.String())
	if err != nil || validatedRef != backupRef {
		return inspectedBackup{}, invalid(errors.New("sqlite.backup_ref_invalid"))
	}
	digest := strings.TrimPrefix(backupRef.String(), backupRefPrefix)
	root, err := recovery.rootLocks.openedRoot(recovery.backupRoot)
	if err != nil {
		return inspectedBackup{}, invalid(err)
	}
	if _, err := validatePrivateDirectoryAt(root, digest); err != nil {
		return inspectedBackup{}, invalid(err)
	}
	payloadName := filepath.Join(digest, recoveryPayloadName)
	manifestName := filepath.Join(digest, recoveryManifestName)
	payload, payloadInfo, err := openPrivateRegularFile(root, payloadName, os.O_RDONLY)
	if err != nil {
		return inspectedBackup{}, invalid(err)
	}
	keepPayload := false
	defer func() {
		if !keepPayload {
			_ = payload.Close()
		}
	}()
	manifestContent, err := readPrivateFile(root, manifestName, 64*1024)
	if err != nil {
		return inspectedBackup{}, invalid(err)
	}
	manifest, err := decodeCanonicalManifest(manifestContent)
	if err != nil {
		return inspectedBackup{}, invalid(err)
	}
	payloadDigest, err := fileSHA256FromFile(ctx, payload, payloadInfo)
	if err != nil {
		return inspectedBackup{}, invalid(err)
	}
	if err := verifyOpenPrivateRegularFile(root, payloadName, payload, payloadInfo); err != nil {
		return inspectedBackup{}, invalid(err)
	}
	if manifest.SchemaVersion != 1 || manifest.BackupRef != backupRef.String() ||
		manifest.PayloadSHA256 != payloadDigest || backupRef.String() != backupRefPrefix+strings.TrimPrefix(payloadDigest, "sha256:") ||
		manifest.MediaType != recoveryMediaType || manifest.Size != payloadInfo.Size() ||
		manifest.CreatedAt.IsZero() || manifest.CreatedAt.Location() != time.UTC ||
		manifest.CredentialsIncluded || manifest.ArtifactBlobsIncluded {
		return inspectedBackup{}, invalid(errors.New("sqlite.backup_manifest_binding_invalid"))
	}
	database, err := openRecoveryDatabase(payload)
	if err != nil {
		return inspectedBackup{}, invalid(err)
	}
	schemaRef, logicalDigest, validateErr := validateRecoveryDatabase(ctx, database)
	closeErr := database.Close()
	if validateErr != nil {
		return inspectedBackup{}, validateErr
	}
	if closeErr != nil {
		return inspectedBackup{}, invalid(closeErr)
	}
	secondDigest, err := fileSHA256FromFile(ctx, payload, payloadInfo)
	if err != nil || secondDigest != payloadDigest {
		return inspectedBackup{}, invalid(errors.New("sqlite.backup_changed_during_verification"))
	}
	if err := verifyOpenPrivateRegularFile(root, payloadName, payload, payloadInfo); err != nil {
		return inspectedBackup{}, invalid(err)
	}
	if schemaRef != manifest.SchemaRef || logicalDigest != manifest.LogicalSHA256 {
		return inspectedBackup{}, invalid(errors.New("sqlite.backup_semantics_invalid"))
	}
	if err := recovery.hit("after_backup_read"); err != nil {
		return inspectedBackup{}, err
	}
	if err := recovery.rootLocks.verify(recovery.backupRoot); err != nil {
		return inspectedBackup{}, invalid(err)
	}
	if err := recovery.hit("after_backup_inspection"); err != nil {
		return inspectedBackup{}, err
	}
	keepPayload = true
	return inspectedBackup{
		manifest: manifest, manifestDigest: bytesSHA256(manifestContent), payloadRoot: root,
		payloadName: payloadName, payload: payload, payloadInfo: payloadInfo,
	}, nil
}

func (recovery *Recovery) receiptForExisting(ctx context.Context, refValue string) (application.BackupReceipt, error) {
	ref, err := application.NewBackupRef(refValue)
	if err != nil {
		return application.BackupReceipt{}, invalid(err)
	}
	inspected, err := recovery.inspectBackup(ctx, ref)
	if err != nil {
		return application.BackupReceipt{}, err
	}
	defer inspected.Close()
	if err := inspected.Close(); err != nil {
		return application.BackupReceipt{}, invalid(err)
	}
	if err := recovery.rootLocks.verify(recovery.backupRoot, recovery.restoreRoot); err != nil {
		return application.BackupReceipt{}, invalid(err)
	}
	return application.BackupReceipt{
		Ref: ref, ManifestSHA256: inspected.manifestDigest, MediaType: inspected.manifest.MediaType,
		Size: inspected.manifest.Size, SchemaRef: inspected.manifest.SchemaRef,
		CreatedAt: inspected.manifest.CreatedAt,
	}, nil
}

func decodeCanonicalManifest(content []byte) (backupManifest, error) {
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	var manifest backupManifest
	if err := decoder.Decode(&manifest); err != nil {
		return backupManifest{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return backupManifest{}, errors.New("sqlite.backup_manifest_trailing_data")
	}
	canonical, err := json.Marshal(manifest)
	if err != nil || !bytes.Equal(canonical, content) {
		return backupManifest{}, errors.New("sqlite.backup_manifest_not_canonical")
	}
	return manifest, nil
}
