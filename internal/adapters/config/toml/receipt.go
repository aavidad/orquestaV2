package toml

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"orquesta/internal/config"
)

const (
	intentSchemaVersion = 1
	intentDocumentType  = "orquesta.config_change_receipt"
	receiptRefPrefix    = "config-receipt:"
)

// durableIntent deliberately contains no source bytes. Replacement bytes live
// only in .next until rename and then in the canonical source.
type durableIntent struct {
	SchemaVersion int                  `json:"schema_version"`
	DocumentType  string               `json:"document_type"`
	Receipt       config.ChangeReceipt `json:"receipt"`
}

func newIntent(previous config.StoredDocument, request config.CommitRequest) durableIntent {
	receiptRef, _ := receiptIdentity(request.ActorRef, request.RequestRef)
	revision := revisionFor(request.Replacement)
	return durableIntent{
		SchemaVersion: intentSchemaVersion,
		DocumentType:  intentDocumentType,
		Receipt: config.ChangeReceipt{
			ReceiptRef:         receiptRef,
			ActorRef:           request.ActorRef,
			RequestRef:         request.RequestRef,
			Fingerprint:        request.Fingerprint,
			BeforeRevision:     previous.Revision,
			AfterRevision:      revision,
			ChangedKeys:        append([]config.Key{}, request.ChangedKeys...),
			PendingRestartKeys: append([]config.Key{}, request.PendingRestartKeys...),
			ChangedAt:          request.ChangedAt.UTC(),
		},
	}
}

func (intent durableIntent) result(replacement []byte, replayed bool) config.CommitResult {
	return config.CommitResult{
		Document: config.StoredDocument{
			Revision: intent.Receipt.AfterRevision,
			Content:  append([]byte(nil), replacement...),
		},
		Receipt:  cloneReceipt(intent.Receipt),
		Replayed: replayed,
	}
}

func (intent durableIntent) matchesRequest(request config.CommitRequest) bool {
	return intent.Receipt.ActorRef == request.ActorRef &&
		intent.Receipt.RequestRef == request.RequestRef &&
		intent.Receipt.Fingerprint == request.Fingerprint
}

func (store *Store) persistReplacement(revision config.Revision, replacement []byte) error {
	if _, err := os.Lstat(store.nextPath); err == nil || !errors.Is(err, os.ErrNotExist) {
		return storeError(config.DocumentStoreSourceInvalid, errors.New("toml_store_stale_replacement"))
	}
	if err := writeExclusiveSynced(store.nextPath, replacement, 0o600); err != nil {
		return err
	}
	content, found, err := readPrivateRegular(store.nextPath, store.maxSourceBytes, privateWriteMode)
	if err != nil || !found {
		return err
	}
	if revisionFor(content) != revision {
		return storeError(config.DocumentStoreSourceInvalid, errors.New("toml_store_replacement_revision_mismatch"))
	}
	return syncDirectory(filepath.Dir(store.nextPath))
}

func (store *Store) prepareIntent(intent durableIntent) ([]byte, error) {
	payload, err := marshalIntent(intent)
	if err != nil {
		return nil, err
	}
	if int64(len(payload)) > store.maxReceiptBytes {
		return nil, storeError(config.DocumentStoreReceiptTooLarge, errors.New("toml_store_receipt_too_large"))
	}
	return payload, nil
}

func (store *Store) persistIntent(payload []byte) error {
	if err := ensureReceiptDirectory(store.receiptDirectory); err != nil {
		return err
	}
	if _, err := os.Lstat(store.pendingPath); err == nil || !errors.Is(err, os.ErrNotExist) {
		return storeError(config.DocumentStoreSourceInvalid, errors.New("toml_store_stale_intent"))
	}
	if _, err := os.Lstat(store.pendingStagePath); err == nil || !errors.Is(err, os.ErrNotExist) {
		return storeError(config.DocumentStoreSourceInvalid, errors.New("toml_store_stale_intent_stage"))
	}
	if err := writeExclusiveSynced(store.pendingStagePath, payload, 0o600); err != nil {
		return err
	}
	if err := os.Rename(store.pendingStagePath, store.pendingPath); err != nil {
		return storeError(config.DocumentStoreIO, err)
	}
	return syncDirectory(store.receiptDirectory)
}

func (store *Store) installSource(intent durableIntent, failpoints bool) error {
	content, found, err := readPrivateRegular(store.nextPath, store.maxSourceBytes, privateWriteMode)
	if err != nil {
		return err
	}
	if !found || revisionFor(content) != intent.Receipt.AfterRevision {
		return storeError(config.DocumentStoreSourceInvalid, errors.New("toml_store_replacement_missing_or_mismatched"))
	}
	if err := os.Rename(store.nextPath, store.path); err != nil {
		return storeError(config.DocumentStoreIO, err)
	}
	if failpoints {
		if err := store.hit(FailpointAfterSourceRename); err != nil {
			return err
		}
	}
	if err := syncDirectory(filepath.Dir(store.path)); err != nil {
		return err
	}
	if failpoints {
		if err := store.hit(FailpointAfterSourceDirectory); err != nil {
			return err
		}
	}
	return nil
}

func (store *Store) finalizeReceipt(intent durableIntent, failpoints bool) error {
	finalPath := store.receiptPath(intent.Receipt.ActorRef, intent.Receipt.RequestRef)
	persisted, found, err := store.readReceipt(intent.Receipt.ActorRef, intent.Receipt.RequestRef)
	if err != nil {
		return err
	}
	if found {
		if !equalIntent(persisted, intent) {
			return storeError(config.DocumentStoreReplayConflict, errors.New("toml_store_receipt_collision"))
		}
		return store.removePendingIfExact(intent)
	}
	if err := sealReadOnly(store.pendingPath); err != nil {
		return err
	}
	if failpoints {
		if err := store.hit(FailpointAfterReceiptSeal); err != nil {
			return err
		}
	}
	if err := os.Rename(store.pendingPath, finalPath); err != nil {
		return storeError(config.DocumentStoreIO, err)
	}
	if failpoints {
		if err := store.hit(FailpointAfterReceiptRename); err != nil {
			return err
		}
	}
	if err := syncDirectory(store.receiptDirectory); err != nil {
		return err
	}
	if failpoints {
		if err := store.hit(FailpointAfterReceiptDirectory); err != nil {
			return err
		}
	}
	return nil
}

func (store *Store) recoverPending() error {
	receiptDirectoryFound, err := store.validateReceiptDirectoryIfPresent()
	if err != nil {
		return err
	}
	if !receiptDirectoryFound {
		return store.discardOrphanReplacement()
	}
	if err := store.discardOrphanIntentStage(); err != nil {
		return err
	}
	payload, found, err := readPrivateRegular(store.pendingPath, store.maxReceiptBytes, privateIntentMode)
	if err != nil {
		return err
	}
	if !found {
		if err := store.discardOrphanReplacement(); err != nil {
			return err
		}
		return store.syncReceiptDirectoriesIfPresent()
	}
	info, err := os.Lstat(store.pendingPath)
	if err != nil || (info.Mode() != 0o600 && info.Mode() != 0o400) {
		return storeError(config.DocumentStoreSourceInvalid, errors.New("toml_store_pending_mode_invalid"))
	}
	intent, err := store.decodeIntent(payload)
	if err != nil {
		return err
	}
	persisted, receiptFound, err := store.readReceipt(intent.Receipt.ActorRef, intent.Receipt.RequestRef)
	if err != nil {
		return err
	}
	if receiptFound {
		if !equalIntent(persisted, intent) {
			return storeError(config.DocumentStoreReplayConflict, errors.New("toml_store_receipt_collision"))
		}
		current, err := store.readSource()
		if err != nil {
			return err
		}
		if current.Revision != intent.Receipt.AfterRevision {
			return storeError(config.DocumentStoreSourceInvalid, errors.New("toml_store_receipt_source_mismatch"))
		}
		if err := store.discardReplacementForRevision(intent.Receipt.AfterRevision); err != nil {
			return err
		}
		return store.removePendingIfExact(intent)
	}

	current, err := store.readSource()
	if err != nil {
		return err
	}
	switch current.Revision {
	case intent.Receipt.AfterRevision:
		if err := syncDirectory(filepath.Dir(store.path)); err != nil {
			return err
		}
		if err := store.discardReplacementForRevision(intent.Receipt.AfterRevision); err != nil {
			return err
		}
	case intent.Receipt.BeforeRevision:
		if err := store.installSource(intent, false); err != nil {
			return err
		}
	default:
		return storeError(config.DocumentStoreRevisionConflict, errors.New("toml_store_recovery_revision_conflict"))
	}
	return store.finalizeReceipt(intent, false)
}

func (store *Store) syncReceiptDirectoriesIfPresent() error {
	found, err := store.validateReceiptDirectoryIfPresent()
	if err != nil || !found {
		return err
	}
	if err := syncDirectory(store.receiptDirectory); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(store.receiptDirectory))
}

func (store *Store) validateReceiptDirectoryIfPresent() (bool, error) {
	info, err := os.Lstat(store.receiptDirectory)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil || !privateDirectory(info) {
		return false, storeError(config.DocumentStoreSourceInvalid, errors.New("toml_store_receipt_directory_invalid"))
	}
	return true, nil
}

func (store *Store) discardOrphanIntentStage() error {
	_, found, err := readPrivateRegular(store.pendingStagePath, store.maxReceiptBytes, privateWriteMode)
	if err != nil || !found {
		return err
	}
	if _, err := os.Lstat(store.pendingPath); err == nil || !errors.Is(err, os.ErrNotExist) {
		return storeError(config.DocumentStoreSourceInvalid, errors.New("toml_store_intent_stage_collision"))
	}
	if err := os.Remove(store.pendingStagePath); err != nil {
		return storeError(config.DocumentStoreIO, err)
	}
	return syncDirectory(store.receiptDirectory)
}

func (store *Store) discardOrphanReplacement() error {
	_, found, err := readPrivateRegular(store.nextPath, store.maxSourceBytes, privateWriteMode)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	if err := os.Remove(store.nextPath); err != nil {
		return storeError(config.DocumentStoreIO, err)
	}
	return syncDirectory(filepath.Dir(store.nextPath))
}

func (store *Store) discardReplacementForRevision(revision config.Revision) error {
	content, found, err := readPrivateRegular(store.nextPath, store.maxSourceBytes, privateWriteMode)
	if err != nil || !found {
		return err
	}
	if revisionFor(content) != revision {
		return storeError(config.DocumentStoreSourceInvalid, errors.New("toml_store_replacement_mismatch"))
	}
	if err := os.Remove(store.nextPath); err != nil {
		return storeError(config.DocumentStoreIO, err)
	}
	return syncDirectory(filepath.Dir(store.nextPath))
}

func (store *Store) removePendingIfExact(intent durableIntent) error {
	payload, found, err := readPrivateRegular(store.pendingPath, store.maxReceiptBytes, privateIntentMode)
	if err != nil || !found {
		return err
	}
	var persisted durableIntent
	if err := strictJSON(payload, &persisted); err != nil || !equalIntent(persisted, intent) {
		return storeError(config.DocumentStoreSourceInvalid, errors.New("toml_store_pending_mismatch"))
	}
	if err := os.Remove(store.pendingPath); err != nil {
		return storeError(config.DocumentStoreIO, err)
	}
	return syncDirectory(store.receiptDirectory)
}

func (store *Store) readReceipt(actorRef, requestRef string) (durableIntent, bool, error) {
	info, err := os.Lstat(store.receiptDirectory)
	if errors.Is(err, os.ErrNotExist) {
		return durableIntent{}, false, nil
	}
	if err != nil || !privateDirectory(info) {
		return durableIntent{}, false, storeError(config.DocumentStoreSourceInvalid, errors.New("toml_store_receipt_directory_invalid"))
	}
	payload, found, err := readPrivateRegular(store.receiptPath(actorRef, requestRef), store.maxReceiptBytes, privateReadMode)
	if err != nil || !found {
		return durableIntent{}, found, err
	}
	intent, err := store.decodeIntent(payload)
	if err != nil {
		return durableIntent{}, false, err
	}
	if intent.Receipt.ActorRef != actorRef || intent.Receipt.RequestRef != requestRef {
		return durableIntent{}, false, storeError(config.DocumentStoreSourceInvalid, errors.New("toml_store_receipt_identity_mismatch"))
	}
	return intent, true, nil
}

func (store *Store) decodeIntent(payload []byte) (durableIntent, error) {
	var intent durableIntent
	if err := strictJSON(payload, &intent); err != nil {
		return durableIntent{}, storeError(config.DocumentStoreSourceInvalid, err)
	}
	if err := validateIntent(intent); err != nil {
		return durableIntent{}, err
	}
	return intent, nil
}

func validateIntent(intent durableIntent) error {
	wantRef, _ := receiptIdentity(intent.Receipt.ActorRef, intent.Receipt.RequestRef)
	if intent.SchemaVersion != intentSchemaVersion || intent.DocumentType != intentDocumentType ||
		!validIdentity(intent.Receipt.ActorRef) || !validIdentity(intent.Receipt.RequestRef) ||
		!validRevision(config.Revision(intent.Receipt.Fingerprint)) || intent.Receipt.ReceiptRef != wantRef ||
		!validRevision(intent.Receipt.BeforeRevision) || !validRevision(intent.Receipt.AfterRevision) ||
		intent.Receipt.ChangedAt.IsZero() || intent.Receipt.ChangedAt.Location() != time.UTC {
		return storeError(config.DocumentStoreSourceInvalid, errors.New("toml_store_receipt_invalid"))
	}
	changed, err := normalizeKeys(intent.Receipt.ChangedKeys)
	if err != nil || !reflect.DeepEqual(changed, intent.Receipt.ChangedKeys) {
		return storeError(config.DocumentStoreSourceInvalid, errors.New("toml_store_receipt_changed_keys_invalid"))
	}
	pending, err := normalizeKeys(intent.Receipt.PendingRestartKeys)
	if err != nil || !reflect.DeepEqual(pending, intent.Receipt.PendingRestartKeys) {
		return storeError(config.DocumentStoreSourceInvalid, errors.New("toml_store_receipt_pending_keys_invalid"))
	}
	return nil
}

func marshalIntent(intent durableIntent) ([]byte, error) {
	payload, err := json.MarshalIndent(intent, "", "  ")
	if err != nil {
		return nil, storeError(config.DocumentStoreIO, err)
	}
	return append(payload, '\n'), nil
}

func strictJSON(payload []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("toml_store_trailing_json")
		}
		return err
	}
	return nil
}

func (store *Store) receiptPath(actorRef, requestRef string) string {
	_, identity := receiptIdentity(actorRef, requestRef)
	return filepath.Join(store.receiptDirectory, identity+".json")
}

func receiptIdentity(actorRef, requestRef string) (string, string) {
	digest := sha256.New()
	writeDigestString(digest, actorRef)
	writeDigestString(digest, requestRef)
	identity := hex.EncodeToString(digest.Sum(nil))
	return receiptRefPrefix + identity, identity
}

func writeDigestString(digest io.Writer, value string) {
	writeDigestBytes(digest, []byte(value))
}

func writeDigestBytes(digest io.Writer, value []byte) {
	writeDigestUint64(digest, uint64(len(value)))
	_, _ = digest.Write(value)
}

func writeDigestUint64(digest io.Writer, value uint64) {
	_ = binary.Write(digest, binary.BigEndian, value)
}

func equalIntent(left, right durableIntent) bool {
	return reflect.DeepEqual(left, right)
}
