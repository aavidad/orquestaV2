// Package toml provides the local transactional document store for editable
// Orquesta TOML configuration. It stores bytes; registry parsing and policy
// remain outside this adapter.
package toml

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"orquesta/internal/config"
)

const (
	revisionPrefix      = "sha256:"
	maximumIdentitySize = 1024
)

const (
	FailpointAfterReplacementSync  = "after_replacement_sync"
	FailpointAfterIntentSync       = "after_intent_sync"
	FailpointAfterSourceRename     = "after_source_rename"
	FailpointAfterSourceDirectory  = "after_source_directory_sync"
	FailpointAfterReceiptSeal      = "after_receipt_seal"
	FailpointAfterReceiptRename    = "after_receipt_rename"
	FailpointAfterReceiptDirectory = "after_receipt_directory_sync"
)

// Options configures one local source and its adjacent transaction metadata.
// Failpoint is test-only injection; production leaves it nil.
type Options struct {
	Path            string
	MaxSourceBytes  int64
	MaxReceiptBytes int64
	RequireExisting bool
	Failpoint       func(string) error
}

// Store serializes all readers and writers with a process-shared file lock.
type Store struct {
	path             string
	lockPath         string
	pendingPath      string
	pendingStagePath string
	nextPath         string
	receiptDirectory string
	maxSourceBytes   int64
	maxReceiptBytes  int64
	requireExisting  bool
	failpoint        func(string) error
}

// ReservedPaths reports adapter-owned sidecar namespaces adjacent to source.
// Composition passes these refs to neutral configuration validation; callers
// must not place runtime state, artifacts, credentials, or effective output
// at or below them.
func ReservedPaths(source string) []string {
	if strings.TrimSpace(source) == "" {
		return []string{}
	}
	return []string{source + ".lock", source + ".next", source + ".receipts"}
}

// Open validates options without creating the configured source. Empty Path
// is a valid defaults-only reader; Commit then returns config_source_required.
func Open(options Options) (*Store, error) {
	if options.MaxSourceBytes <= 0 || options.MaxSourceBytes > int64(math.MaxInt) ||
		options.MaxReceiptBytes <= 0 || options.MaxReceiptBytes > int64(math.MaxInt) {
		return nil, storeError(config.DocumentStoreRequestInvalid, errors.New("toml_store_limits_invalid"))
	}
	path := strings.TrimSpace(options.Path)
	if path != options.Path {
		return nil, storeError(config.DocumentStoreSourceInvalid, errors.New("toml_store_path_invalid"))
	}
	if path != "" {
		absolute, err := filepath.Abs(filepath.Clean(path))
		if err != nil || filepath.Base(absolute) == "." || filepath.Base(absolute) == string(filepath.Separator) {
			return nil, storeError(config.DocumentStoreSourceInvalid, errors.New("toml_store_path_invalid"))
		}
		path = absolute
		if options.RequireExisting {
			_, found, err := readPrivateRegular(path, options.MaxSourceBytes, privateWriteMode)
			if err != nil {
				return nil, err
			}
			if !found {
				return nil, storeError(config.DocumentStoreSourceRequired, errors.New("toml_store_source_missing"))
			}
		}
	}
	receiptDirectory := path + ".receipts"
	pendingPath := filepath.Join(receiptDirectory, ".pending")
	return &Store{
		path:             path,
		lockPath:         path + ".lock",
		pendingPath:      pendingPath,
		pendingStagePath: pendingPath + ".next",
		nextPath:         path + ".next",
		receiptDirectory: receiptDirectory,
		maxSourceBytes:   options.MaxSourceBytes,
		maxReceiptBytes:  options.MaxReceiptBytes,
		requireExisting:  options.RequireExisting,
		failpoint:        options.Failpoint,
	}, nil
}

// Read returns exact source bytes after completing any durable pending intent.
func (store *Store) Read(ctx context.Context) (config.StoredDocument, error) {
	if store == nil {
		return config.StoredDocument{}, storeError(config.DocumentStoreSourceInvalid, errors.New("toml_store_nil"))
	}
	if ctx == nil {
		return config.StoredDocument{}, storeError(config.DocumentStoreRequestInvalid, errors.New("toml_store_context_required"))
	}
	if store.path == "" {
		return emptyDocument(), nil
	}
	var document config.StoredDocument
	err := store.withLock(ctx, func() error {
		if err := store.recoverPending(); err != nil {
			return err
		}
		var err error
		document, err = store.readSource()
		return err
	})
	if err != nil {
		return config.StoredDocument{}, err
	}
	return cloneDocument(document), nil
}

// Commit atomically replaces source bytes under expected-revision CAS. A
// durable actor/request/fingerprint receipt makes replay side-effect free.
func (store *Store) Commit(ctx context.Context, request config.CommitRequest) (config.CommitResult, error) {
	if store == nil {
		return config.CommitResult{}, storeError(config.DocumentStoreSourceInvalid, errors.New("toml_store_nil"))
	}
	if ctx == nil {
		return config.CommitResult{}, storeError(config.DocumentStoreRequestInvalid, errors.New("toml_store_context_required"))
	}
	if store.path == "" {
		return config.CommitResult{}, storeError(config.DocumentStoreSourceRequired, nil)
	}
	normalized, err := store.normalizeRequest(request)
	if err != nil {
		return config.CommitResult{}, err
	}

	var result config.CommitResult
	err = store.withLock(ctx, func() error {
		if err := store.recoverPending(); err != nil {
			return err
		}
		persisted, found, err := store.readReceipt(normalized.ActorRef, normalized.RequestRef)
		if err != nil {
			return err
		}
		if found {
			if !persisted.matchesRequest(normalized) {
				return storeError(config.DocumentStoreReplayConflict, errors.New("toml_store_fingerprint_conflict"))
			}
			current, err := store.readSource()
			if err != nil {
				return err
			}
			result = config.CommitResult{Document: current, Receipt: cloneReceipt(persisted.Receipt), Replayed: true}
			return nil
		}

		current, err := store.readSource()
		if err != nil {
			return err
		}
		if current.Revision != normalized.ExpectedRevision {
			return storeError(config.DocumentStoreRevisionConflict, fmt.Errorf("expected %s got %s", normalized.ExpectedRevision, current.Revision))
		}
		intent := newIntent(current, normalized)
		intentPayload, err := store.prepareIntent(intent)
		if err != nil {
			return err
		}
		if err := store.persistReplacement(intent.Receipt.AfterRevision, normalized.Replacement); err != nil {
			return err
		}
		if err := store.hit(FailpointAfterReplacementSync); err != nil {
			return err
		}
		if err := store.persistIntent(intentPayload); err != nil {
			return err
		}
		if err := store.hit(FailpointAfterIntentSync); err != nil {
			return err
		}
		if err := store.installSource(intent, true); err != nil {
			return err
		}
		if err := store.finalizeReceipt(intent, true); err != nil {
			return err
		}
		result = intent.result(normalized.Replacement, false)
		return nil
	})
	if err != nil {
		return config.CommitResult{}, err
	}
	return cloneResult(result), nil
}

func (store *Store) normalizeRequest(request config.CommitRequest) (config.CommitRequest, error) {
	if !validRevision(request.ExpectedRevision) || int64(len(request.Replacement)) > store.maxSourceBytes ||
		!validIdentity(request.ActorRef) || !validIdentity(request.RequestRef) || !validRevision(config.Revision(request.Fingerprint)) ||
		request.ChangedAt.IsZero() {
		code := config.DocumentStoreRequestInvalid
		if int64(len(request.Replacement)) > store.maxSourceBytes {
			code = config.DocumentStoreSourceTooLarge
		}
		return config.CommitRequest{}, storeError(code, errors.New("toml_store_request_invalid"))
	}
	changed, err := normalizeKeys(request.ChangedKeys)
	if err != nil {
		return config.CommitRequest{}, err
	}
	pending, err := normalizeKeys(request.PendingRestartKeys)
	if err != nil {
		return config.CommitRequest{}, err
	}
	request.Replacement = append([]byte(nil), request.Replacement...)
	request.ChangedKeys = changed
	request.PendingRestartKeys = pending
	request.ChangedAt = request.ChangedAt.UTC()
	return request, nil
}

func normalizeKeys(keys []config.Key) ([]config.Key, error) {
	result := append([]config.Key(nil), keys...)
	sort.Slice(result, func(left, right int) bool { return result[left] < result[right] })
	for index, key := range result {
		if !validCanonicalKey(string(key)) || (index > 0 && result[index-1] == key) {
			return nil, storeError(config.DocumentStoreRequestInvalid, errors.New("toml_store_keys_invalid"))
		}
	}
	if result == nil {
		result = []config.Key{}
	}
	return result, nil
}

func validCanonicalKey(value string) bool {
	if value == "" || strings.HasPrefix(value, ".") || strings.HasSuffix(value, ".") || strings.Contains(value, "..") {
		return false
	}
	for _, character := range value {
		if (character >= 'a' && character <= 'z') || (character >= '0' && character <= '9') || character == '_' || character == '.' {
			continue
		}
		return false
	}
	return true
}

func validIdentity(value string) bool {
	if value == "" || strings.TrimSpace(value) != value || len(value) > maximumIdentitySize || !utf8.ValidString(value) {
		return false
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return false
		}
	}
	return true
}

func revisionFor(content []byte) config.Revision {
	digest := sha256.Sum256(content)
	return config.Revision(revisionPrefix + hex.EncodeToString(digest[:]))
}

func validRevision(revision config.Revision) bool {
	value := string(revision)
	if len(value) != len(revisionPrefix)+sha256.Size*2 || !strings.HasPrefix(value, revisionPrefix) {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, revisionPrefix))
	return err == nil && value == strings.ToLower(value)
}

func emptyDocument() config.StoredDocument {
	return config.StoredDocument{Revision: revisionFor(nil), Content: []byte{}}
}

func cloneDocument(document config.StoredDocument) config.StoredDocument {
	document.Content = append([]byte(nil), document.Content...)
	if document.Content == nil {
		document.Content = []byte{}
	}
	return document
}

func cloneReceipt(receipt config.ChangeReceipt) config.ChangeReceipt {
	receipt.ChangedKeys = append([]config.Key(nil), receipt.ChangedKeys...)
	receipt.PendingRestartKeys = append([]config.Key(nil), receipt.PendingRestartKeys...)
	if receipt.ChangedKeys == nil {
		receipt.ChangedKeys = []config.Key{}
	}
	if receipt.PendingRestartKeys == nil {
		receipt.PendingRestartKeys = []config.Key{}
	}
	return receipt
}

func cloneResult(result config.CommitResult) config.CommitResult {
	result.Document = cloneDocument(result.Document)
	result.Receipt = cloneReceipt(result.Receipt)
	return result
}

func storeError(code config.DocumentStoreErrorCode, cause error) error {
	return &config.DocumentStoreError{Code: code, Cause: cause}
}

func (store *Store) hit(point string) error {
	if store.failpoint == nil {
		return nil
	}
	if err := store.failpoint(point); err != nil {
		return storeError(config.DocumentStoreIO, fmt.Errorf("%s: %w", point, err))
	}
	return nil
}

var _ config.DocumentStore = (*Store)(nil)
