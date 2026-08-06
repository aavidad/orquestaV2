package credentials

import (
	"context"
	"crypto/ed25519"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
)

// Ed25519PrivateKeySource generates or obtains one Ed25519 private key for a
// single callback. It never returns private material. Implementations must
// destroy callback-scoped material after the callback finishes.
type Ed25519PrivateKeySource interface {
	WithPrivateKey(context.Context, func(Secret) error) error
}

// ProvisionStore guarantees that discovery and mutation observe one credential
// authority. It composes existing narrow ports without widening either one.
type ProvisionStore interface {
	Store
	UseAuthorityReader
}

// Ed25519PrivateKeySourceFunc adapts an injected callback-scoped generator.
type Ed25519PrivateKeySourceFunc func(context.Context, func(Secret) error) error

func (source Ed25519PrivateKeySourceFunc) WithPrivateKey(
	ctx context.Context,
	consume func(Secret) error,
) error {
	return source(ctx, consume)
}

// ProvisionCredentialRequest identifies one independently resumable
// credential. RequestRef is a deterministic, material-free base reference;
// operation and concrete-version suffixes are derived internally.
type ProvisionCredentialRequest struct {
	RequestRef    string
	CredentialRef CredentialRef
	ScopeRefs     []ScopeRef
	PurposeRef    PurposeRef
}

// ProvisionCodexMicroVMCredentialsRequest explicitly binds both credentials
// needed by the first Codex microVM to one actor and owner.
type ProvisionCodexMicroVMCredentialsRequest struct {
	ActorRef     string
	OwnerRef     OwnerRef
	SigningKeyID string
	Signing      ProvisionCredentialRequest
	Auth         ProvisionCredentialRequest
}

type CredentialProvisionState string

const (
	CredentialProvisionPending CredentialProvisionState = "pending"
	CredentialProvisionCreated CredentialProvisionState = "created"
	CredentialProvisionResumed CredentialProvisionState = "resumed"
)

// RequestedCredentialAuthority is the material-free requested authority.
// ScopeRefs records exact requested tuples confirmed by DescribeUseAuthority
// or by a validated successful Create result. It does not claim that the
// stored credential has no additional authorized scopes. Version zero appears
// only while that credential remains pending.
type RequestedCredentialAuthority struct {
	CredentialRef CredentialRef
	OwnerRef      OwnerRef
	ScopeRefs     []ScopeRef
	PurposeRef    PurposeRef
	Version       Version
}

// CredentialProvisionStatus makes partial progress explicit and retryable.
// Receipts are create/use receipts only and never contain credential material.
type CredentialProvisionStatus struct {
	State              CredentialProvisionState
	RequestedAuthority RequestedCredentialAuthority
	Receipts           []Receipt
}

// ProvisionCodexMicroVMCredentialsResult exposes only resumable status and the
// public half of the signing identity. SigningKeyID is the exact, public ID
// supplied by the caller's canonical configuration authority.
type ProvisionCodexMicroVMCredentialsResult struct {
	Signing                CredentialProvisionStatus
	Auth                   CredentialProvisionStatus
	SigningKeyID           string
	SigningPublicKeyBase64 string
}

// ProvisionCodexMicroVMCredentials creates or resumes the signing and provider
// auth credentials independently. Signing is intentionally first: after its
// Create becomes durable, any crash is repairable by describing the exact
// version and deriving only its public key inside Store.Use. Auth is therefore
// never materialized merely to repair signing. This use case never rotates,
// revokes, or rolls back either credential.
func ProvisionCodexMicroVMCredentials(
	ctx context.Context,
	store ProvisionStore,
	authSource MaterialSource,
	keySource Ed25519PrivateKeySource,
	request ProvisionCodexMicroVMCredentialsRequest,
) (result ProvisionCodexMicroVMCredentialsResult, err error) {
	defer func() {
		if recover() != nil {
			err = NewError(ErrorConsumerFailed, "provision")
		}
	}()

	if ctx == nil || nilInterfaceValue(store) || nilInterfaceValue(authSource) || nilInterfaceValue(keySource) {
		return result, NewError(ErrorInvalidRequest, "dependency")
	}
	if err := ValidateProvisionCodexMicroVMCredentialsRequest(request); err != nil {
		return result, err
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}

	result.Signing = pendingProvisionStatus(request.OwnerRef, request.Signing)
	result.Auth = pendingProvisionStatus(request.OwnerRef, request.Auth)

	result.Signing, result.SigningKeyID, result.SigningPublicKeyBase64, err = provisionSigningCredential(
		ctx, store, keySource, request.ActorRef, request.OwnerRef, request.SigningKeyID, request.Signing,
	)
	if err != nil {
		return result, err
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}

	result.Auth, err = provisionAuthCredential(
		ctx, store, authSource, request.ActorRef, request.OwnerRef, request.Auth,
	)
	if err != nil {
		return result, err
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	return result, nil
}

func ValidateProvisionCodexMicroVMCredentialsRequest(
	request ProvisionCodexMicroVMCredentialsRequest,
) error {
	if request.Signing.CredentialRef == request.Auth.CredentialRef {
		return NewError(ErrorInvalidRequest, "credential_refs")
	}
	if request.Signing.RequestRef == request.Auth.RequestRef {
		return NewError(ErrorInvalidRequest, "request_refs")
	}
	if !validProvisionSigningKeyID(request.SigningKeyID) {
		return NewError(ErrorInvalidRequest, "signing_key_id")
	}
	for _, candidate := range []ProvisionCredentialRequest{request.Signing, request.Auth} {
		scopes, err := canonicalProvisionScopeRefs(candidate.ScopeRefs)
		if err != nil {
			return err
		}
		for index, scope := range scopes {
			describe := describeRequest(request.ActorRef, request.OwnerRef, candidate, scope, index)
			if err := ValidateDescribeUseAuthorityRequest(describe); err != nil {
				return err
			}
		}
		for _, suffix := range []string{"create", "use:v1"} {
			derived := operationRequestRef(candidate.RequestRef, suffix)
			if err := validateRef(derived, "request:"); err != nil {
				return err
			}
		}
	}
	return nil
}

func provisionSigningCredential(
	ctx context.Context,
	store ProvisionStore,
	source Ed25519PrivateKeySource,
	actor string,
	owner OwnerRef,
	keyID string,
	spec ProvisionCredentialRequest,
) (CredentialProvisionStatus, string, string, error) {
	status := pendingProvisionStatus(owner, spec)
	authority, found, err := describeProvisionAuthority(ctx, store, actor, owner, spec)
	created := false
	if err != nil {
		return status, "", "", err
	}
	if !found {
		mutation, createErr := createFromCallbackSource(ctx, store, actor, owner, spec, source.WithPrivateKey, validateEd25519PrivateMaterial)
		if createErr != nil {
			if !creationCollision(createErr) {
				return status, "", "", createErr
			}
			authority, found, err = describeProvisionAuthority(ctx, store, actor, owner, spec)
			if err != nil || !found {
				if err != nil {
					return status, "", "", err
				}
				return status, "", "", safeProvisionStoreError(createErr, "create_race")
			}
		} else {
			authority = authorityFromMutation(mutation, canonicalFirstScope(spec))
			status.Receipts = append(status.Receipts, mutation.Receipt)
			created = !mutation.Replayed
		}
	}

	status.RequestedAuthority = requestedAuthorityFromDescription(authority, spec)
	publicKey, useReceipt, err := deriveStoredEd25519PublicKey(ctx, store, actor, spec, authority)
	if err != nil {
		return status, "", "", err
	}
	status.Receipts = append(status.Receipts, useReceipt)
	if created {
		status.State = CredentialProvisionCreated
	} else {
		status.State = CredentialProvisionResumed
	}
	return status, keyID, publicKey, nil
}

func provisionAuthCredential(
	ctx context.Context,
	store ProvisionStore,
	source MaterialSource,
	actor string,
	owner OwnerRef,
	spec ProvisionCredentialRequest,
) (CredentialProvisionStatus, error) {
	status := pendingProvisionStatus(owner, spec)
	authority, found, err := describeProvisionAuthority(ctx, store, actor, owner, spec)
	if err != nil {
		return status, err
	}
	if found {
		status.RequestedAuthority = requestedAuthorityFromDescription(authority, spec)
		status.State = CredentialProvisionResumed
		return status, nil
	}

	mutation, createErr := createFromCallbackSource(ctx, store, actor, owner, spec, source.WithSecret, nil)
	if createErr != nil {
		if !creationCollision(createErr) {
			return status, createErr
		}
		authority, found, err = describeProvisionAuthority(ctx, store, actor, owner, spec)
		if err != nil || !found {
			if err != nil {
				return status, err
			}
			return status, safeProvisionStoreError(createErr, "create_race")
		}
		status.RequestedAuthority = requestedAuthorityFromDescription(authority, spec)
		status.State = CredentialProvisionResumed
		return status, nil
	}

	status.RequestedAuthority = requestedAuthorityFromDescription(authorityFromMutation(mutation, canonicalFirstScope(spec)), spec)
	status.Receipts = append(status.Receipts, mutation.Receipt)
	if mutation.Replayed {
		status.State = CredentialProvisionResumed
	} else {
		status.State = CredentialProvisionCreated
	}
	return status, nil
}

type callbackSecretSource func(context.Context, func(Secret) error) error

type provisionCallbackGate struct {
	open     atomic.Bool
	claimed  atomic.Bool
	rejected atomic.Uint32
	done     chan struct{}
}

func newProvisionCallbackGate() *provisionCallbackGate {
	gate := &provisionCallbackGate{done: make(chan struct{})}
	gate.open.Store(true)
	return gate
}

// claim uses no mutex: a reentrant or concurrent second callback fails
// immediately while the first callback remains active.
func (gate *provisionCallbackGate) claim() (func(), bool) {
	if !gate.open.Load() || !gate.claimed.CompareAndSwap(false, true) {
		gate.rejected.Add(1)
		return nil, false
	}
	if !gate.open.Load() {
		gate.rejected.Add(1)
		close(gate.done)
		return nil, false
	}
	return func() { close(gate.done) }, true
}

// closeAndWait first makes every late callback inert, then joins the sole
// callback only when it already won the claim before closure.
func (gate *provisionCallbackGate) closeAndWait() {
	gate.open.Store(false)
	if gate.claimed.Load() {
		<-gate.done
	}
}

func (gate *provisionCallbackGate) wasClaimed() bool { return gate.claimed.Load() }
func (gate *provisionCallbackGate) rejectedCount() uint32 {
	return gate.rejected.Load()
}

func createFromCallbackSource(
	ctx context.Context,
	store Store,
	actor string,
	owner OwnerRef,
	spec ProvisionCredentialRequest,
	withSecret callbackSecretSource,
	validateMaterial func([]byte) error,
) (MutationResult, error) {
	if err := ctx.Err(); err != nil {
		return MutationResult{}, err
	}
	gate := newProvisionCallbackGate()
	var owned Secret
	defer func() { owned.Destroy() }()
	var callbackErr error

	consume := func(secret Secret) error {
		defer secret.Destroy()
		finish, claimed := gate.claim()
		if !claimed {
			return NewError(ErrorConsumerFailed, "source_callback")
		}
		defer finish()
		if err := ctx.Err(); err != nil {
			callbackErr = err
			return err
		}
		material := secret.Bytes()
		defer clear(material)
		if validateMaterial != nil {
			if err := validateMaterial(material); err != nil {
				callbackErr = err
				return err
			}
		}
		var err error
		owned, err = NewSecret(material)
		if err != nil {
			callbackErr = err
			return err
		}
		return nil
	}

	var sourceErr error
	func() {
		defer gate.closeAndWait()
		sourceErr = withSecret(ctx, consume)
	}()
	if err := ctx.Err(); err != nil {
		return MutationResult{}, err
	}
	if !gate.wasClaimed() && sourceErr != nil {
		return MutationResult{}, safeProvisionSourceError(sourceErr, "source")
	}
	if !gate.wasClaimed() || gate.rejectedCount() != 0 {
		return MutationResult{}, NewError(ErrorConsumerFailed, "source_callback")
	}
	if callbackErr != nil {
		return MutationResult{}, safeProvisionSourceError(callbackErr, "source_callback")
	}
	if sourceErr != nil {
		return MutationResult{}, safeProvisionSourceError(sourceErr, "source")
	}

	create := CreateRequest{
		ActorRef: actor, RequestRef: operationRequestRef(spec.RequestRef, "create"),
		CredentialRef: spec.CredentialRef, OwnerRef: owner,
		ScopeRefs: canonicalScopeRefsUnchecked(spec.ScopeRefs), PurposeRef: spec.PurposeRef, Material: owned,
	}
	if err := ValidateCreateRequest(create); err != nil {
		return MutationResult{}, err
	}
	mutation, createErr := store.Create(ctx, create)
	if err := ctx.Err(); err != nil {
		return MutationResult{}, err
	}
	if createErr != nil {
		return MutationResult{}, safeProvisionStoreError(createErr, "create")
	}
	if err := validateCreateMutation(create, mutation); err != nil {
		return MutationResult{}, err
	}
	return mutation, nil
}

func describeProvisionAuthority(
	ctx context.Context,
	reader UseAuthorityReader,
	actor string,
	owner OwnerRef,
	spec ProvisionCredentialRequest,
) (DescribedUseAuthority, bool, error) {
	if err := ctx.Err(); err != nil {
		return DescribedUseAuthority{}, false, err
	}
	scopes := canonicalScopeRefsUnchecked(spec.ScopeRefs)
	var authority DescribedUseAuthority
	for index, scope := range scopes {
		if err := ctx.Err(); err != nil {
			return DescribedUseAuthority{}, false, err
		}
		request := describeRequest(actor, owner, spec, scope, index)
		current, err := reader.DescribeUseAuthority(ctx, request)
		if contextErr := ctx.Err(); contextErr != nil {
			return DescribedUseAuthority{}, false, contextErr
		}
		if err != nil {
			if HasErrorCode(err, ErrorNotFound) && index == 0 {
				return DescribedUseAuthority{}, false, nil
			}
			return DescribedUseAuthority{}, false, safeProvisionStoreError(err, "describe")
		}
		if err := ValidateDescribedUseAuthority(request, current); err != nil {
			return DescribedUseAuthority{}, false, err
		}
		if index > 0 && current.Version != authority.Version {
			return DescribedUseAuthority{}, false, NewError(ErrorVersionConflict, "described_scope_versions")
		}
		if index == 0 {
			authority = current
		}
	}
	return authority, true, nil
}

func deriveStoredEd25519PublicKey(
	ctx context.Context,
	store Store,
	actor string,
	spec ProvisionCredentialRequest,
	authority DescribedUseAuthority,
) (string, Receipt, error) {
	if err := ctx.Err(); err != nil {
		return "", Receipt{}, err
	}
	request := UseRequest{
		ActorRef:      actor,
		RequestRef:    operationRequestRef(spec.RequestRef, "use:v"+strconv.FormatUint(uint64(authority.Version), 10)),
		CredentialRef: authority.CredentialRef, OwnerRef: authority.OwnerRef,
		ScopeRef: canonicalFirstScope(spec), PurposeRef: authority.PurposeRef, Version: authority.Version,
	}
	if err := ValidateUseRequest(request); err != nil {
		return "", Receipt{}, err
	}

	gate := newProvisionCallbackGate()
	var encodedPublic string
	var callbackErr error
	consume := func(secret Secret) error {
		defer secret.Destroy()
		finish, claimed := gate.claim()
		if !claimed {
			return NewError(ErrorConsumerFailed, "use_callback")
		}
		defer finish()
		if err := ctx.Err(); err != nil {
			callbackErr = err
			return err
		}
		encodedPublic, callbackErr = ed25519PublicProjection(secret)
		return callbackErr
	}

	var receipt Receipt
	var useErr error
	func() {
		defer gate.closeAndWait()
		receipt, useErr = store.Use(ctx, request, consume)
	}()
	if err := ctx.Err(); err != nil {
		return "", Receipt{}, err
	}
	if !gate.wasClaimed() && useErr != nil {
		return "", Receipt{}, safeProvisionStoreError(useErr, "use")
	}
	if !gate.wasClaimed() || gate.rejectedCount() != 0 {
		return "", Receipt{}, NewError(ErrorConsumerFailed, "use_callback")
	}
	if callbackErr != nil {
		return "", Receipt{}, callbackErr
	}
	if useErr != nil {
		return "", Receipt{}, safeProvisionStoreError(useErr, "use")
	}
	if err := validateUseReceipt(request, receipt); err != nil {
		return "", Receipt{}, err
	}
	return encodedPublic, receipt, nil
}

func ed25519PublicProjection(secret Secret) (string, error) {
	material := secret.Bytes()
	defer clear(material)
	if err := validateEd25519PrivateMaterial(material); err != nil {
		return "", err
	}
	canonical := ed25519.NewKeyFromSeed(material[:ed25519.SeedSize])
	defer clear(canonical)
	public := append([]byte(nil), canonical[ed25519.SeedSize:]...)
	defer clear(public)
	return base64.StdEncoding.EncodeToString(public), nil
}

func validateEd25519PrivateMaterial(material []byte) error {
	if len(material) != ed25519.PrivateKeySize {
		return NewError(ErrorInvalidRequest, "ed25519_private_key")
	}
	canonical := ed25519.NewKeyFromSeed(material[:ed25519.SeedSize])
	defer clear(canonical)
	if subtle.ConstantTimeCompare(canonical, material) != 1 {
		return NewError(ErrorInvalidRequest, "ed25519_private_key")
	}
	return nil
}

func validateCreateMutation(request CreateRequest, result MutationResult) error {
	metadata, receipt := result.Metadata, result.Receipt
	if metadata.CredentialRef != request.CredentialRef || metadata.OwnerRef != request.OwnerRef ||
		metadata.PurposeRef != request.PurposeRef || metadata.Version != 1 || metadata.Revoked ||
		!metadata.RotatedAt.IsZero() || !metadata.RevokedAt.IsZero() ||
		!equalScopeRefs(metadata.ScopeRefs, request.ScopeRefs) || metadata.CreatedAt.IsZero() ||
		receipt.CredentialRef != request.CredentialRef || receipt.OwnerRef != request.OwnerRef ||
		receipt.ScopeRef != "" || receipt.PurposeRef != request.PurposeRef || receipt.Version != metadata.Version ||
		receipt.RequestRef != request.RequestRef || receipt.ActorRef != request.ActorRef ||
		receipt.Operation != "create" || receipt.OccurredAt.IsZero() {
		return NewError(ErrorInvalidRequest, "create_result")
	}
	return nil
}

func validateUseReceipt(request UseRequest, receipt Receipt) error {
	if receipt.CredentialRef != request.CredentialRef || receipt.OwnerRef != request.OwnerRef ||
		receipt.ScopeRef != request.ScopeRef || receipt.PurposeRef != request.PurposeRef ||
		receipt.Version != request.Version || receipt.RequestRef != request.RequestRef ||
		receipt.ActorRef != request.ActorRef || receipt.Operation != "use" || receipt.OccurredAt.IsZero() {
		return NewError(ErrorInvalidRequest, "use_result")
	}
	return nil
}

func describeRequest(
	actor string,
	owner OwnerRef,
	spec ProvisionCredentialRequest,
	scope ScopeRef,
	index int,
) DescribeUseAuthorityRequest {
	return DescribeUseAuthorityRequest{
		ActorRef: actor, RequestRef: operationRequestRef(spec.RequestRef, "describe:scope:"+strconv.Itoa(index)),
		CredentialRef: spec.CredentialRef, OwnerRef: owner,
		ScopeRef: scope, PurposeRef: spec.PurposeRef,
	}
}

func operationRequestRef(base, operation string) string { return base + ":" + operation }

func pendingProvisionStatus(owner OwnerRef, spec ProvisionCredentialRequest) CredentialProvisionStatus {
	return CredentialProvisionStatus{
		State: CredentialProvisionPending,
		RequestedAuthority: RequestedCredentialAuthority{
			CredentialRef: spec.CredentialRef, OwnerRef: owner,
			ScopeRefs: canonicalScopeRefsUnchecked(spec.ScopeRefs), PurposeRef: spec.PurposeRef,
		},
	}
}

func requestedAuthorityFromDescription(
	authority DescribedUseAuthority,
	spec ProvisionCredentialRequest,
) RequestedCredentialAuthority {
	return RequestedCredentialAuthority{
		CredentialRef: authority.CredentialRef, OwnerRef: authority.OwnerRef,
		ScopeRefs: canonicalScopeRefsUnchecked(spec.ScopeRefs), PurposeRef: authority.PurposeRef, Version: authority.Version,
	}
}

func authorityFromMutation(result MutationResult, scope ScopeRef) DescribedUseAuthority {
	return DescribedUseAuthority{
		CredentialRef: result.Metadata.CredentialRef, OwnerRef: result.Metadata.OwnerRef,
		ScopeRef: scope, PurposeRef: result.Metadata.PurposeRef,
		Version: result.Metadata.Version,
	}
}

func canonicalProvisionScopeRefs(values []ScopeRef) ([]ScopeRef, error) {
	if len(values) == 0 {
		return nil, NewError(ErrorInvalidRequest, "scope_refs")
	}
	result := append([]ScopeRef(nil), values...)
	for _, value := range result {
		if err := ValidateScopeRef(value); err != nil {
			return nil, err
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	for index := 1; index < len(result); index++ {
		if result[index] == result[index-1] {
			return nil, NewError(ErrorInvalidRequest, "duplicate_scope_ref")
		}
	}
	return result, nil
}

func canonicalScopeRefsUnchecked(values []ScopeRef) []ScopeRef {
	result, _ := canonicalProvisionScopeRefs(values)
	return result
}

func canonicalFirstScope(spec ProvisionCredentialRequest) ScopeRef {
	return canonicalScopeRefsUnchecked(spec.ScopeRefs)[0]
}

func equalScopeRefs(left, right []ScopeRef) bool {
	canonicalLeft, err := canonicalProvisionScopeRefs(left)
	if err != nil {
		return false
	}
	canonicalRight, err := canonicalProvisionScopeRefs(right)
	if err != nil || len(canonicalLeft) != len(canonicalRight) {
		return false
	}
	for index := range canonicalLeft {
		if canonicalLeft[index] != canonicalRight[index] {
			return false
		}
	}
	return true
}

// The configuration registry remains the authority for the stricter product
// syntax. This boundary only guarantees one bounded, non-control public ID in
// the established namespace and returns it byte-for-byte unchanged.
func validProvisionSigningKeyID(value string) bool {
	const prefix = "clave-publica:"
	if !strings.HasPrefix(value, prefix) || len(value) == len(prefix) || !safeText(value, 160) {
		return false
	}
	for _, character := range []byte(value) {
		if character < 0x21 || character > 0x7e {
			return false
		}
	}
	return true
}

func creationCollision(err error) bool {
	return HasErrorCode(err, ErrorAlreadyExists) || HasErrorCode(err, ErrorIdempotencyConflict)
}

func safeProvisionStoreError(err error, field string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	var credentialError *Error
	if errors.As(err, &credentialError) && knownProvisionErrorCode(credentialError.Code) {
		return NewError(credentialError.Code, field)
	}
	return NewError(ErrorStoreIO, field)
}

func safeProvisionSourceError(err error, field string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	var credentialError *Error
	if errors.As(err, &credentialError) && knownProvisionErrorCode(credentialError.Code) {
		return NewError(credentialError.Code, field)
	}
	return NewError(ErrorConsumerFailed, field)
}

func knownProvisionErrorCode(code ErrorCode) bool {
	switch code {
	case ErrorInvalidRequest, ErrorInvalidRef, ErrorAlreadyExists, ErrorNotFound,
		ErrorOwnerMismatch, ErrorScopeDenied, ErrorPurposeDenied, ErrorVersionConflict,
		ErrorIdempotencyConflict, ErrorAlreadyConsumed, ErrorRevoked, ErrorUnsafeFile,
		ErrorSecretLeak, ErrorChildEnvironment, ErrorConsumerFailed, ErrorStoreIO:
		return true
	default:
		return false
	}
}

func nilInterfaceValue(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}
