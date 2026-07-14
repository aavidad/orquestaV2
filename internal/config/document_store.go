package config

import (
	"context"
	"errors"
	"time"
)

// Revision is the content-addressed identity of one exact source document.
type Revision string

// StoredDocument is one immutable view returned by a DocumentStore.
type StoredDocument struct {
	Revision Revision
	Content  []byte
}

// CommitRequest describes one causally identified replacement of the source
// document. Fingerprint is the caller-owned canonical command identity; the
// store binds it to ActorRef and RequestRef. On replay, derived replacement
// and metadata are ignored because active state may have advanced.
type CommitRequest struct {
	ExpectedRevision   Revision
	Replacement        []byte
	ActorRef           string
	RequestRef         string
	Fingerprint        string
	ChangedKeys        []Key
	PendingRestartKeys []Key
	ChangedAt          time.Time
}

// ChangeReceipt is durable proof of one committed source replacement.
type ChangeReceipt struct {
	ReceiptRef         string    `json:"receipt_ref"`
	ActorRef           string    `json:"actor_ref"`
	RequestRef         string    `json:"request_ref"`
	Fingerprint        string    `json:"fingerprint"`
	BeforeRevision     Revision  `json:"before_revision"`
	AfterRevision      Revision  `json:"after_revision"`
	ChangedKeys        []Key     `json:"changed_keys"`
	PendingRestartKeys []Key     `json:"pending_restart_keys"`
	ChangedAt          time.Time `json:"changed_at"`
}

// CommitResult returns the committed document and its immutable receipt.
// Replayed is true when the same actor/request/fingerprint was already
// committed and no filesystem mutation was repeated.
type CommitResult struct {
	Document StoredDocument
	Receipt  ChangeReceipt
	Replayed bool
}

// DocumentStore is the replaceable durable boundary for editable non-secret
// configuration. Application policy validates values before Commit.
type DocumentStore interface {
	Read(context.Context) (StoredDocument, error)
	Commit(context.Context, CommitRequest) (CommitResult, error)
}

// DocumentStoreErrorCode is stable machine-readable storage failure data.
type DocumentStoreErrorCode string

const (
	DocumentStoreSourceRequired   DocumentStoreErrorCode = "config_source_required"
	DocumentStoreSourceInvalid    DocumentStoreErrorCode = "config_source_invalid"
	DocumentStoreSourceTooLarge   DocumentStoreErrorCode = "config_source_too_large"
	DocumentStoreReceiptTooLarge  DocumentStoreErrorCode = "config_receipt_too_large"
	DocumentStoreRequestInvalid   DocumentStoreErrorCode = "config_commit_request_invalid"
	DocumentStoreRevisionConflict DocumentStoreErrorCode = "config_revision_conflict"
	DocumentStoreReplayConflict   DocumentStoreErrorCode = "config_idempotency_conflict"
	DocumentStoreIO               DocumentStoreErrorCode = "config_store_io_failed"
)

// DocumentStoreError keeps public code stable while preserving diagnostic
// cause for logs and tests.
type DocumentStoreError struct {
	Code  DocumentStoreErrorCode
	Cause error
}

func (err *DocumentStoreError) Error() string {
	if err == nil {
		return ""
	}
	return string(err.Code)
}

func (err *DocumentStoreError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Cause
}

// IsDocumentStoreError reports whether err contains the requested stable code.
func IsDocumentStoreError(err error, code DocumentStoreErrorCode) bool {
	var storeError *DocumentStoreError
	return errors.As(err, &storeError) && storeError.Code == code
}
