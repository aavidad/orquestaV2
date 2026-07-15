package application

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

// BackupRef identifies an immutable, content-addressed state backup.
type BackupRef struct{ value string }

// RecoveryTargetRef identifies a new offline restore target. It is opaque to
// the application; concrete paths remain owned by the state adapter.
type RecoveryTargetRef struct{ value string }

func NewBackupRef(value string) (BackupRef, error) {
	const prefix = "backup:sha256:"
	digest := strings.TrimPrefix(value, prefix)
	if len(digest) != 64 || prefix+digest != value {
		return BackupRef{}, errors.New("state_recovery.backup_ref_invalid")
	}
	decoded, err := hex.DecodeString(digest)
	if err != nil || hex.EncodeToString(decoded) != digest {
		return BackupRef{}, errors.New("state_recovery.backup_ref_invalid")
	}
	return BackupRef{value: value}, nil
}

func (ref BackupRef) String() string { return ref.value }

func NewRecoveryTargetRef(value string) (RecoveryTargetRef, error) {
	const prefix = "recovery-target:"
	name := strings.TrimPrefix(value, prefix)
	if value != prefix+name || len(name) == 0 || len(name) > 128 || strings.Contains(name, "..") {
		return RecoveryTargetRef{}, errors.New("state_recovery.target_ref_invalid")
	}
	for _, character := range name {
		if character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' ||
			character >= '0' && character <= '9' || strings.ContainsRune("._-", character) {
			continue
		}
		return RecoveryTargetRef{}, errors.New("state_recovery.target_ref_invalid")
	}
	return RecoveryTargetRef{value: value}, nil
}

func (ref RecoveryTargetRef) String() string { return ref.value }

type BackupReceipt struct {
	Ref            BackupRef
	ManifestSHA256 string
	MediaType      string
	Size           int64
	SchemaRef      string
	CreatedAt      time.Time
}

type BackupVerification struct {
	BackupRef      BackupRef
	ManifestSHA256 string
	SchemaRef      string
	VerifiedAt     time.Time
}

type RestoreReceipt struct {
	BackupRef      BackupRef
	TargetRef      RecoveryTargetRef
	ManifestSHA256 string
	SchemaRef      string
	RestoredAt     time.Time
}

// StateRecovery is the operational recovery boundary. StateRepository remains
// solely responsible for application state mutations and lifecycle fencing.
type StateRecovery interface {
	CreateBackup(context.Context) (BackupReceipt, error)
	VerifyBackup(context.Context, BackupRef) (BackupVerification, error)
	RestoreBackup(context.Context, BackupRef, RecoveryTargetRef) (RestoreReceipt, error)
}
