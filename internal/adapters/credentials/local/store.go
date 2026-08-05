// Package local provides the private single-file CredentialStore adapter.
package local

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"orquesta/internal/credentials"
)

const (
	documentType                     = "orquesta.credential_store"
	documentSchemaLegacy             = 1
	documentSchema                   = 2
	requestFingerprintSchema         = "orquesta.credential_request.v1"
	FailpointAfterReplacementSync    = "after_replacement_sync"
	FailpointAfterStoreRename        = "after_store_rename"
	FailpointAfterStoreDirectorySync = "after_store_directory_sync"
)

type Options struct {
	Path          string
	OwnerUID      int
	MaxStoreBytes int64
	Now           func() time.Time
	Failpoint     func(string) error
	Sync          func(*os.File) error
}

type Store struct {
	files     *fileSystem
	now       func() time.Time
	failpoint func(string) error
}

type document struct {
	SchemaVersion int                                  `json:"schema_version"`
	DocumentType  string                               `json:"document_type"`
	Revision      uint64                               `json:"revision"`
	ParentDigest  string                               `json:"parent_digest"`
	Records       map[credentials.CredentialRef]record `json:"records"`
	Ledger        map[string]ledger                    `json:"ledger"`
}

type record struct {
	Metadata credentials.Metadata `json:"metadata"`
	Material []byte               `json:"material,omitempty"`
}

type ledger struct {
	Fingerprint string               `json:"fingerprint"`
	Result      credentials.Metadata `json:"result"`
	Receipt     credentials.Receipt  `json:"receipt"`
}

func Open(options Options) (*Store, error) {
	if options.Now == nil {
		options.Now = time.Now
	}
	files, err := openFileSystem(options.Path, options.OwnerUID, options.MaxStoreBytes, options.Sync)
	if err != nil {
		return nil, mapFileError("path", err)
	}
	store := &Store{files: files, now: options.Now, failpoint: options.Failpoint}
	if err := files.withLock(context.Background(), func() error {
		doc, content, err := store.load()
		defer clearDocument(&doc)
		defer clear(content)
		return err
	}); err != nil {
		files.close()
		return nil, mapFileError("open", err)
	}
	return store, nil
}

func (store *Store) Close() error {
	if store != nil && store.files != nil {
		store.files.close()
	}
	return nil
}

func (store *Store) Create(ctx context.Context, request credentials.CreateRequest) (credentials.MutationResult, error) {
	if err := credentials.ValidateCreateRequest(request); err != nil {
		return credentials.MutationResult{}, err
	}
	fingerprint := requestFingerprint("create", request.ActorRef, request.RequestRef, request.CredentialRef.String(),
		request.OwnerRef.String(), request.PurposeRef.String(), scopesText(request.ScopeRefs), secretDigest(request.Material))
	var result credentials.MutationResult
	err := store.update(ctx, request.ActorRef, request.RequestRef, fingerprint, func(doc *document, at time.Time) error {
		if _, found := doc.Records[request.CredentialRef]; found {
			return credentials.NewError(credentials.ErrorAlreadyExists, "credential_ref")
		}
		metadata := credentials.Metadata{
			CredentialRef: request.CredentialRef, OwnerRef: request.OwnerRef, ScopeRefs: sortedScopes(request.ScopeRefs),
			PurposeRef: request.PurposeRef, Version: 1, CreatedAt: at,
		}
		doc.Records[request.CredentialRef] = record{Metadata: metadata, Material: request.Material.Bytes()}
		appendMutation(doc, &result, request.ActorRef, request.RequestRef, "create", fingerprint, metadata, at)
		return nil
	}, &result)
	if err != nil {
		return credentials.MutationResult{}, err
	}
	return result, nil
}

func (store *Store) Rotate(ctx context.Context, request credentials.RotateRequest) (credentials.MutationResult, error) {
	if err := credentials.ValidateRotateRequest(request); err != nil {
		return credentials.MutationResult{}, err
	}
	fingerprint := requestFingerprint("rotate", request.ActorRef, request.RequestRef, request.CredentialRef.String(),
		request.OwnerRef.String(), strconv.FormatUint(uint64(request.ExpectedVersion), 10), secretDigest(request.Material))
	var result credentials.MutationResult
	err := store.update(ctx, request.ActorRef, request.RequestRef, fingerprint, func(doc *document, at time.Time) error {
		current, err := mutableRecord(doc, request.CredentialRef, request.OwnerRef, request.ExpectedVersion)
		if err != nil {
			return err
		}
		if current.Metadata.Version == credentials.Version(math.MaxUint64) {
			return credentials.NewError(credentials.ErrorVersionConflict, "expected_version")
		}
		current.Metadata.Version++
		current.Metadata.RotatedAt = at
		replaceRecordMaterial(&current, request.Material.Bytes())
		appendMutation(doc, &result, request.ActorRef, request.RequestRef, "rotate", fingerprint, current.Metadata, at)
		doc.Records[request.CredentialRef] = current
		return nil
	}, &result)
	if err != nil {
		return credentials.MutationResult{}, err
	}
	return result, nil
}

func (store *Store) Revoke(ctx context.Context, request credentials.RevokeRequest) (credentials.MutationResult, error) {
	if err := credentials.ValidateRevokeRequest(request); err != nil {
		return credentials.MutationResult{}, err
	}
	fingerprint := requestFingerprint("revoke", request.ActorRef, request.RequestRef, request.CredentialRef.String(),
		request.OwnerRef.String(), strconv.FormatUint(uint64(request.ExpectedVersion), 10), request.Reason)
	var result credentials.MutationResult
	err := store.update(ctx, request.ActorRef, request.RequestRef, fingerprint, func(doc *document, at time.Time) error {
		current, err := mutableRecord(doc, request.CredentialRef, request.OwnerRef, request.ExpectedVersion)
		if err != nil {
			return err
		}
		current.Metadata.Revoked, current.Metadata.RevokedAt = true, at
		replaceRecordMaterial(&current, nil)
		appendMutation(doc, &result, request.ActorRef, request.RequestRef, "revoke", fingerprint, current.Metadata, at)
		doc.Records[request.CredentialRef] = current
		return nil
	}, &result)
	if err != nil {
		return credentials.MutationResult{}, err
	}
	return result, nil
}

// DescribeUseAuthority resolves the current authorized credential version
// without materializing secret material or recording a use.
func (store *Store) DescribeUseAuthority(
	ctx context.Context,
	request credentials.DescribeUseAuthorityRequest,
) (credentials.DescribedUseAuthority, error) {
	if err := credentials.ValidateDescribeUseAuthorityRequest(request); err != nil {
		return credentials.DescribedUseAuthority{}, err
	}
	if store == nil || store.files == nil || ctx == nil {
		return credentials.DescribedUseAuthority{}, credentials.NewError(credentials.ErrorInvalidRequest, "store")
	}
	var result credentials.DescribedUseAuthority
	err := store.files.withLock(ctx, func() error {
		doc, content, err := store.load()
		if err != nil {
			return err
		}
		defer clearDocument(&doc)
		defer clear(content)
		current, found := doc.Records[request.CredentialRef]
		if !found {
			return credentials.NewError(credentials.ErrorNotFound, "credential_ref")
		}
		if err := authorizeUse(current.Metadata, credentials.UseRequest{
			ActorRef: request.ActorRef, RequestRef: request.RequestRef,
			CredentialRef: request.CredentialRef, OwnerRef: request.OwnerRef,
			ScopeRef: request.ScopeRef, PurposeRef: request.PurposeRef,
			Version: 0,
		}); err != nil {
			return err
		}
		result = credentials.DescribedUseAuthority{
			CredentialRef: request.CredentialRef, OwnerRef: request.OwnerRef,
			ScopeRef: request.ScopeRef, PurposeRef: request.PurposeRef,
			Version: current.Metadata.Version,
		}
		return credentials.ValidateDescribedUseAuthority(request, result)
	})
	if err != nil {
		return credentials.DescribedUseAuthority{}, mapFileError("describe_use_authority", err)
	}
	return result, nil
}

func (store *Store) Use(ctx context.Context, request credentials.UseRequest, consume func(credentials.Secret) error) (credentials.Receipt, error) {
	if err := credentials.ValidateUseRequest(request); err != nil {
		return credentials.Receipt{}, err
	}
	if store == nil || store.files == nil || ctx == nil || consume == nil {
		return credentials.Receipt{}, credentials.NewError(credentials.ErrorInvalidRequest, "consumer")
	}
	fingerprint := requestFingerprint("use", request.ActorRef, request.RequestRef, request.CredentialRef.String(),
		request.OwnerRef.String(), request.ScopeRef.String(), request.PurposeRef.String(), strconv.FormatUint(uint64(request.Version), 10))
	var receipt credentials.Receipt
	var secret credentials.Secret
	err := store.files.withLock(ctx, func() error {
		doc, content, err := store.load()
		if err != nil {
			return err
		}
		defer clearDocument(&doc)
		defer clear(content)
		current, found := doc.Records[request.CredentialRef]
		if !found {
			return credentials.NewError(credentials.ErrorNotFound, "credential_ref")
		}
		if err := authorizeUse(current.Metadata, request); err != nil {
			return err
		}
		if replay, found := doc.Ledger[ledgerKey(request.ActorRef, request.RequestRef)]; found {
			if replay.Fingerprint != fingerprint {
				return credentials.NewError(credentials.ErrorIdempotencyConflict, "request_ref")
			}
			if replay.Receipt.Version != current.Metadata.Version {
				return credentials.NewError(credentials.ErrorVersionConflict, "version")
			}
			receipt = replay.Receipt
			secret, err = credentials.NewSecret(current.Material)
			return err
		}
		at, err := store.operationTime()
		if err != nil {
			return err
		}
		receipt = newReceipt(request.ActorRef, request.RequestRef, "use", current.Metadata, request.ScopeRef, at)
		doc.Ledger[ledgerKey(request.ActorRef, request.RequestRef)] = ledger{Fingerprint: fingerprint, Result: cloneMetadata(current.Metadata), Receipt: receipt}
		gate, err := newDurableProjectionGate(doc)
		if err != nil {
			return err
		}
		defer gate.Destroy()
		if err := store.persist(doc, content, gate); err != nil {
			return err
		}
		secret, err = credentials.NewSecret(current.Material)
		return err
	})
	if err != nil {
		return credentials.Receipt{}, mapFileError("use", err)
	}
	if err := consumeSecret(secret, consume); err != nil {
		return receipt, err
	}
	return receipt, nil
}

// UseOnce claims one causal request durably before exposing material to its
// callback. Distinct actor/request pairs remain independent even when they use
// the same credential version.
func (store *Store) UseOnce(ctx context.Context, request credentials.OneShotUseRequest, consume func(credentials.Secret) error) (credentials.OneShotUseResult, error) {
	if err := credentials.ValidateOneShotUseRequest(request); err != nil {
		return credentials.OneShotUseResult{}, err
	}
	if store == nil || store.files == nil || ctx == nil || consume == nil {
		return credentials.OneShotUseResult{}, credentials.NewError(credentials.ErrorInvalidRequest, "consumer")
	}
	useRequest := credentials.UseRequest(request)
	fingerprint := requestFingerprint(credentials.OperationUseOnce, request.CredentialRef.String(),
		request.OwnerRef.String(), request.ScopeRef.String(), request.PurposeRef.String(), strconv.FormatUint(uint64(request.Version), 10))
	var result credentials.OneShotUseResult
	var secret credentials.Secret
	err := store.files.withLock(ctx, func() error {
		doc, content, err := store.load()
		if err != nil {
			return err
		}
		defer clearDocument(&doc)
		defer clear(content)
		key := ledgerKey(request.ActorRef, request.RequestRef)
		if replay, found := doc.Ledger[key]; found {
			if replay.Fingerprint != fingerprint {
				return credentials.NewError(credentials.ErrorIdempotencyConflict, "request_ref")
			}
			result = credentials.OneShotUseResult{Receipt: replay.Receipt, Replayed: true}
			if err := credentials.ValidateOneShotUseResult(request, result); err != nil {
				return errUnsafeFile
			}
			return credentials.NewError(credentials.ErrorAlreadyConsumed, "request_ref")
		}
		current, found := doc.Records[request.CredentialRef]
		if !found {
			return credentials.NewError(credentials.ErrorNotFound, "credential_ref")
		}
		if err := authorizeUse(current.Metadata, useRequest); err != nil {
			return err
		}
		at, err := store.operationTime()
		if err != nil {
			return err
		}
		result = credentials.OneShotUseResult{Receipt: newReceipt(request.ActorRef, request.RequestRef,
			credentials.OperationUseOnce, current.Metadata, request.ScopeRef, at)}
		doc.SchemaVersion = documentSchema
		doc.Ledger[key] = ledger{Fingerprint: fingerprint, Result: cloneMetadata(current.Metadata), Receipt: result.Receipt}
		gate, err := newDurableProjectionGate(doc)
		if err != nil {
			return err
		}
		defer gate.Destroy()
		if err := store.persist(doc, content, gate); err != nil {
			return err
		}
		secret, err = credentials.NewSecret(current.Material)
		return err
	})
	if err != nil {
		if credentials.HasErrorCode(err, credentials.ErrorAlreadyConsumed) {
			return result, err
		}
		return credentials.OneShotUseResult{}, mapFileError("use_once", err)
	}
	if err := consumeSecret(secret, consume); err != nil {
		return result, err
	}
	return result, nil
}

func consumeSecret(secret credentials.Secret, consume func(credentials.Secret) error) error {
	defer secret.Destroy()
	guard, err := credentials.NewLeakGuard(secret)
	if err != nil {
		return err
	}
	defer guard.Destroy()
	consumeErr := consume(secret)
	if consumeErr == nil {
		return nil
	}
	projection := []byte(consumeErr.Error())
	defer clear(projection)
	if err := guard.Scan([]credentials.LeakSurface{{Name: "consumer_error", Content: projection}}); err != nil {
		return credentials.NewError(credentials.ErrorSecretLeak, "consumer")
	}
	return safeConsumerError(consumeErr)
}

func safeConsumerError(err error) error {
	switch {
	case errors.Is(err, context.Canceled):
		return context.Canceled
	case errors.Is(err, context.DeadlineExceeded):
		return context.DeadlineExceeded
	}
	var credentialError *credentials.Error
	if errors.As(err, &credentialError) {
		return credentials.NewError(credentialError.Code, credentialError.Field)
	}
	return credentials.NewError(credentials.ErrorConsumerFailed, "consumer")
}

func (store *Store) update(ctx context.Context, actor, request, fingerprint string, mutate func(*document, time.Time) error, result *credentials.MutationResult) error {
	if store == nil || store.files == nil || ctx == nil {
		return credentials.NewError(credentials.ErrorInvalidRequest, "store")
	}
	err := store.files.withLock(ctx, func() error {
		doc, content, err := store.load()
		if err != nil {
			return err
		}
		defer clearDocument(&doc)
		defer clear(content)
		if replay, found := doc.Ledger[ledgerKey(actor, request)]; found {
			if replay.Fingerprint != fingerprint {
				return credentials.NewError(credentials.ErrorIdempotencyConflict, "request_ref")
			}
			*result = credentials.MutationResult{Metadata: cloneMetadata(replay.Result), Receipt: replay.Receipt, Replayed: true}
			return nil
		}
		gate, err := newDurableProjectionGate(doc)
		if err != nil {
			return err
		}
		defer gate.Destroy()
		at, err := store.operationTime()
		if err != nil {
			return err
		}
		if err := mutate(&doc, at); err != nil {
			return err
		}
		return store.persist(doc, content, gate)
	})
	return mapFileError("update", err)
}

func (store *Store) load() (document, []byte, error) {
	content, found, err := store.files.read(store.files.name)
	if err != nil {
		clear(content)
		return document{}, nil, err
	}
	current := emptyDocument()
	if found {
		current, err = decodeDocument(content)
		if err != nil {
			clear(content)
			return document{}, nil, err
		}
	}
	nextContent, nextFound, err := store.files.read(store.files.next)
	if err != nil {
		clearDocument(&current)
		clear(content)
		clear(nextContent)
		return document{}, nil, err
	}
	if !nextFound {
		return current, content, nil
	}
	next, err := decodeDocument(nextContent)
	if err != nil || next.Revision != current.Revision+1 || next.ParentDigest != contentDigest(content) {
		clearDocument(&current)
		clearDocument(&next)
		clear(content)
		clear(nextContent)
		return document{}, nil, errUnsafeFile
	}
	if err := store.files.installNext(); err != nil {
		clearDocument(&current)
		clearDocument(&next)
		clear(content)
		clear(nextContent)
		return document{}, nil, err
	}
	clearDocument(&current)
	clear(content)
	return next, nextContent, nil
}

func (store *Store) persist(next document, previous []byte, gate *durableProjectionGate) error {
	if next.Revision == math.MaxUint64 {
		return credentials.NewError(credentials.ErrorStoreIO, "revision")
	}
	next.Revision++
	next.ParentDigest = contentDigest(previous)
	if gate == nil {
		return credentials.NewError(credentials.ErrorStoreIO, "durable_projection_gate")
	}
	if err := gate.AddDocument(next); err != nil {
		return err
	}
	if err := gate.Validate(next); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return credentials.WrapError(credentials.ErrorStoreIO, "encode", err)
	}
	payload = append(payload, '\n')
	defer clear(payload)
	if int64(len(payload)) > store.files.maximum {
		return credentials.NewError(credentials.ErrorStoreIO, "maximum")
	}
	return store.files.replace(payload, store.hit)
}

func (store *Store) hit(point string) error {
	if store.failpoint == nil {
		return nil
	}
	if err := store.failpoint(point); err != nil {
		return credentials.WrapError(credentials.ErrorStoreIO, point, err)
	}
	return nil
}

func (store *Store) operationTime() (time.Time, error) {
	value := store.now().Round(0).UTC()
	if value.IsZero() {
		return time.Time{}, credentials.NewError(credentials.ErrorStoreIO, "clock")
	}
	return value, nil
}

func authorizeUse(metadata credentials.Metadata, request credentials.UseRequest) error {
	switch {
	case metadata.OwnerRef != request.OwnerRef:
		return credentials.NewError(credentials.ErrorOwnerMismatch, "owner_ref")
	case metadata.Revoked:
		return credentials.NewError(credentials.ErrorRevoked, "credential_ref")
	case metadata.PurposeRef != request.PurposeRef:
		return credentials.NewError(credentials.ErrorPurposeDenied, "purpose_ref")
	case !hasScope(metadata.ScopeRefs, request.ScopeRef):
		return credentials.NewError(credentials.ErrorScopeDenied, "scope_ref")
	case request.Version != 0 && metadata.Version != request.Version:
		return credentials.NewError(credentials.ErrorVersionConflict, "version")
	default:
		return nil
	}
}

func decodeDocument(content []byte) (document, error) {
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	var doc document
	if err := decoder.Decode(&doc); err != nil {
		clearDocument(&doc)
		return document{}, errUnsafeFile
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) || validateDocument(doc) != nil {
		clearDocument(&doc)
		return document{}, errUnsafeFile
	}
	return doc, nil
}

func validateDocument(doc document) error {
	if doc.SchemaVersion != documentSchemaLegacy && doc.SchemaVersion != documentSchema || doc.DocumentType != documentType ||
		doc.Revision == 0 || !validDigest(doc.ParentDigest) || doc.Records == nil || doc.Ledger == nil {
		return errUnsafeFile
	}
	for ref, item := range doc.Records {
		metadata := item.Metadata
		if ref != metadata.CredentialRef || !validMetadata(metadata) || metadata.Revoked != (len(item.Material) == 0) {
			return errUnsafeFile
		}
	}
	for key, item := range doc.Ledger {
		if key != ledgerKey(item.Receipt.ActorRef, item.Receipt.RequestRef) || !validLedger(doc.SchemaVersion, item) {
			return errUnsafeFile
		}
	}
	gate, err := newDurableProjectionGate(doc)
	if err != nil {
		return errUnsafeFile
	}
	defer gate.Destroy()
	if err := gate.Validate(doc); err != nil {
		return errUnsafeFile
	}
	return nil
}

func validMetadata(value credentials.Metadata) bool {
	return credentials.ValidateCredentialRef(value.CredentialRef) == nil && credentials.ValidateOwnerRef(value.OwnerRef) == nil &&
		credentials.ValidatePurposeRef(value.PurposeRef) == nil && validScopes(value.ScopeRefs) && value.Version > 0 && !value.CreatedAt.IsZero() &&
		(value.Version > 1) == !value.RotatedAt.IsZero() && value.Revoked == !value.RevokedAt.IsZero()
}

func validLedger(schema int, item ledger) bool {
	result, receipt := item.Result, item.Receipt
	scope := receipt.ScopeRef
	if scope == "" && len(result.ScopeRefs) > 0 {
		scope = result.ScopeRefs[0]
	}
	request := credentials.UseRequest{ActorRef: receipt.ActorRef, RequestRef: receipt.RequestRef, CredentialRef: receipt.CredentialRef,
		OwnerRef: receipt.OwnerRef, ScopeRef: scope, PurposeRef: receipt.PurposeRef, Version: receipt.Version}
	if !validDigest(item.Fingerprint) || !validMetadata(result) || credentials.ValidateUseRequest(request) != nil || receipt.OccurredAt.IsZero() ||
		result.CredentialRef != receipt.CredentialRef || result.OwnerRef != receipt.OwnerRef || result.PurposeRef != receipt.PurposeRef || result.Version != receipt.Version {
		return false
	}
	switch receipt.Operation {
	case "create":
		return receipt.ScopeRef == "" && result.Version == 1 && !result.Revoked
	case "rotate":
		return receipt.ScopeRef == "" && result.Version > 1 && !result.Revoked
	case "revoke":
		return receipt.ScopeRef == "" && result.Revoked
	case "use":
		return !result.Revoked && hasScope(result.ScopeRefs, receipt.ScopeRef)
	case credentials.OperationUseOnce:
		oneShot := credentials.OneShotUseRequest{
			ActorRef: receipt.ActorRef, RequestRef: receipt.RequestRef, CredentialRef: receipt.CredentialRef,
			OwnerRef: receipt.OwnerRef, ScopeRef: receipt.ScopeRef, PurposeRef: receipt.PurposeRef, Version: receipt.Version,
		}
		fingerprint := requestFingerprint(credentials.OperationUseOnce, receipt.CredentialRef.String(), receipt.OwnerRef.String(),
			receipt.ScopeRef.String(), receipt.PurposeRef.String(), strconv.FormatUint(uint64(receipt.Version), 10))
		return schema == documentSchema && !result.Revoked && hasScope(result.ScopeRefs, receipt.ScopeRef) &&
			credentials.ValidateOneShotUseRequest(oneShot) == nil && item.Fingerprint == fingerprint
	default:
		return false
	}
}

func emptyDocument() document {
	return document{SchemaVersion: documentSchemaLegacy, DocumentType: documentType,
		Records: map[credentials.CredentialRef]record{}, Ledger: map[string]ledger{}}
}

func mutableRecord(doc *document, ref credentials.CredentialRef, owner credentials.OwnerRef, version credentials.Version) (record, error) {
	current, found := doc.Records[ref]
	if !found {
		return record{}, credentials.NewError(credentials.ErrorNotFound, "credential_ref")
	}
	switch {
	case current.Metadata.OwnerRef != owner:
		return record{}, credentials.NewError(credentials.ErrorOwnerMismatch, "owner_ref")
	case current.Metadata.Revoked:
		return record{}, credentials.NewError(credentials.ErrorRevoked, "credential_ref")
	case current.Metadata.Version != version:
		return record{}, credentials.NewError(credentials.ErrorVersionConflict, "expected_version")
	}
	return current, nil
}

func newReceipt(actor, request, operation string, metadata credentials.Metadata, scope credentials.ScopeRef, at time.Time) credentials.Receipt {
	return credentials.Receipt{CredentialRef: metadata.CredentialRef, OwnerRef: metadata.OwnerRef, ScopeRef: scope,
		PurposeRef: metadata.PurposeRef, Version: metadata.Version, RequestRef: request, ActorRef: actor,
		Operation: operation, OccurredAt: at}
}

func appendMutation(doc *document, result *credentials.MutationResult, actor, request, operation, fingerprint string, metadata credentials.Metadata, at time.Time) {
	receipt := newReceipt(actor, request, operation, metadata, "", at)
	*result = credentials.MutationResult{Metadata: cloneMetadata(metadata), Receipt: receipt}
	doc.Ledger[ledgerKey(actor, request)] = ledger{Fingerprint: fingerprint, Result: cloneMetadata(metadata), Receipt: receipt}
}

func ledgerKey(actor, request string) string { return actor + "\x00" + request }

func requestFingerprint(fields ...string) string {
	versioned := make([]string, 1, len(fields)+1)
	versioned[0] = requestFingerprintSchema
	versioned = append(versioned, fields...)
	payload, _ := json.Marshal(versioned)
	defer clear(payload)
	return contentDigest(payload)
}

func secretDigest(secret credentials.Secret) string {
	material := secret.Bytes()
	defer clear(material)
	return contentDigest(material)
}
func contentDigest(content []byte) string { return fmt.Sprintf("sha256:%x", sha256.Sum256(content)) }
func validDigest(value string) bool {
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return len(value) == len("sha256:")+sha256.Size*2 && strings.HasPrefix(value, "sha256:") && err == nil
}

func sortedScopes(values []credentials.ScopeRef) []credentials.ScopeRef {
	result := append([]credentials.ScopeRef(nil), values...)
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}
func scopesText(values []credentials.ScopeRef) string {
	payload, _ := json.Marshal(sortedScopes(values))
	return string(payload)
}
func validScopes(values []credentials.ScopeRef) bool {
	if len(values) == 0 {
		return false
	}
	for index, value := range values {
		if credentials.ValidateScopeRef(value) != nil || index > 0 && values[index-1] >= value {
			return false
		}
	}
	return true
}
func hasScope(values []credentials.ScopeRef, wanted credentials.ScopeRef) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func cloneMetadata(value credentials.Metadata) credentials.Metadata {
	value.ScopeRefs = append([]credentials.ScopeRef(nil), value.ScopeRefs...)
	return value
}

func replaceRecordMaterial(target *record, material []byte) {
	clear(target.Material)
	target.Material = material
}

func clearDocument(doc *document) {
	if doc == nil {
		return
	}
	for ref, item := range doc.Records {
		clear(item.Material)
		item.Material = nil
		doc.Records[ref] = item
	}
}
func mapFileError(field string, err error) error {
	if err == nil {
		return nil
	}
	var credentialError *credentials.Error
	if errors.As(err, &credentialError) {
		return err
	}
	if errors.Is(err, errUnsafeFile) {
		return credentials.WrapError(credentials.ErrorUnsafeFile, field, err)
	}
	return credentials.WrapError(credentials.ErrorStoreIO, field, err)
}

var _ credentials.Store = (*Store)(nil)
var _ credentials.OneShotStore = (*Store)(nil)
