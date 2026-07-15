package sqlite

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
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
	directoryPath := filepath.Join(recovery.backupRoot, digest)
	if err := validatePrivateDirectory(directoryPath); err != nil {
		return inspectedBackup{}, invalid(err)
	}
	payloadPath := filepath.Join(directoryPath, recoveryPayloadName)
	manifestPath := filepath.Join(directoryPath, recoveryManifestName)
	payloadInfo, err := validatePrivateRegularFile(payloadPath)
	if err != nil {
		return inspectedBackup{}, invalid(err)
	}
	manifestContent, err := readPrivateFile(manifestPath, 64*1024)
	if err != nil {
		return inspectedBackup{}, invalid(err)
	}
	manifest, err := decodeCanonicalManifest(manifestContent)
	if err != nil {
		return inspectedBackup{}, invalid(err)
	}
	payloadDigest, err := fileSHA256(ctx, payloadPath)
	if err != nil {
		return inspectedBackup{}, invalid(err)
	}
	if manifest.SchemaVersion != 1 || manifest.BackupRef != backupRef.String() ||
		manifest.PayloadSHA256 != payloadDigest || backupRef.String() != backupRefPrefix+strings.TrimPrefix(payloadDigest, "sha256:") ||
		manifest.MediaType != recoveryMediaType || manifest.Size != payloadInfo.Size() ||
		manifest.CreatedAt.IsZero() || manifest.CreatedAt.Location() != time.UTC ||
		manifest.CredentialsIncluded || manifest.ArtifactBlobsIncluded {
		return inspectedBackup{}, invalid(errors.New("sqlite.backup_manifest_binding_invalid"))
	}
	database, err := openRecoveryDatabase(payloadPath)
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
	secondDigest, err := fileSHA256(ctx, payloadPath)
	if err != nil || secondDigest != payloadDigest {
		return inspectedBackup{}, invalid(errors.New("sqlite.backup_changed_during_verification"))
	}
	if schemaRef != manifest.SchemaRef || logicalDigest != manifest.LogicalSHA256 {
		return inspectedBackup{}, invalid(errors.New("sqlite.backup_semantics_invalid"))
	}
	return inspectedBackup{
		manifest: manifest, manifestDigest: bytesSHA256(manifestContent), payloadPath: payloadPath,
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
