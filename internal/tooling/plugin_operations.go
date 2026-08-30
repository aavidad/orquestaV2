package tooling

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"orquesta/internal/goal"
)

const (
	PluginOperationsCapability = "TLS-11"

	PluginOperationInstall    PluginOperation         = "install"
	PluginOperationUpgrade    PluginOperation         = "upgrade"
	PluginOperationDisable    PluginOperation         = "disable"
	PluginOperationRemove     PluginOperation         = "remove"
	PluginDispositionEnabled  PluginDispositionStatus = "enabled"
	PluginDispositionDisabled PluginDispositionStatus = "disabled"
	PluginDispositionRemoved  PluginDispositionStatus = "removed"

	ErrorPluginOperationInvalid             = "tooling.plugin_operation_invalid"
	ErrorPluginOperationCatalogDenied       = "tooling.plugin_operation_catalog_denied"
	ErrorPluginOperationRevisionConflict    = "tooling.plugin_operation_revision_conflict"
	ErrorPluginOperationIdempotencyConflict = "tooling.plugin_operation_idempotency_conflict"
	ErrorPluginOperationTransitionDenied    = "tooling.plugin_operation_transition_denied"
)

type PluginOperation string
type PluginDispositionStatus string

// PluginOperationRequest changes only Orquesta's desired plugin disposition.
// It is not a download, activation, signature or authorization receipt.
type PluginOperationRequest struct {
	RequestRef           string
	IdempotencyKey       string
	ActorRef             goal.ActorRef
	ProjectRef           goal.ProjectRef
	Context              CuratedContext
	CuratedCatalogDigest string
	Operation            PluginOperation
	PluginID             string
	Version              string
	PluginDigest         string
	ExpectedRevision     uint64
	ReasonCode           string
	RequestedAt          time.Time
}

// PluginDisposition is a tombstoned, revision-fenced projection. Removed entries
// remain addressable so audit and the reinstall fence cannot lose history.
type PluginDisposition struct {
	ProjectRef            string                  `json:"project_ref"`
	PluginID              string                  `json:"plugin_id"`
	Version               string                  `json:"version"`
	PluginDigest          string                  `json:"plugin_digest"`
	CuratedCatalogID      string                  `json:"curated_catalog_id"`
	CuratedCatalogVersion string                  `json:"curated_catalog_version"`
	CuratedCatalogDigest  string                  `json:"curated_catalog_digest"`
	Status                PluginDispositionStatus `json:"status"`
	Revision              uint64                  `json:"revision"`
	ChangedAt             time.Time               `json:"changed_at"`
}

// PluginOperationReceipt is an audit fact for the disposition transition. It
// deliberately does not claim that bytes were downloaded or an external
// effect was attempted/applied.
type PluginOperationReceipt struct {
	Ref                  string            `json:"ref"`
	Digest               string            `json:"digest"`
	RequestRef           string            `json:"request_ref"`
	RequestDigest        string            `json:"request_digest"`
	IdempotencyKey       string            `json:"idempotency_key"`
	ActorRef             string            `json:"actor_ref"`
	ProjectRef           string            `json:"project_ref"`
	Operation            PluginOperation   `json:"operation"`
	ReasonCode           string            `json:"reason_code"`
	PluginID             string            `json:"plugin_id"`
	Before               PluginDisposition `json:"before"`
	After                PluginDisposition `json:"after"`
	CurationReviewRef    string            `json:"curation_review_ref"`
	CurationReviewDigest string            `json:"curation_review_digest"`
	RequestedAt          time.Time         `json:"requested_at"`
	CommittedAt          time.Time         `json:"committed_at"`
}

// PluginOperationState is an immutable in-process snapshot; its fence is not persistence, authorization or CAS.
type PluginOperationState struct {
	valid        bool
	dispositions map[string]PluginDisposition
	receipts     map[string]PluginOperationReceipt
	digest       string
}

func NewPluginOperationState() *PluginOperationState {
	state := &PluginOperationState{
		valid: true, dispositions: map[string]PluginDisposition{},
		receipts: map[string]PluginOperationReceipt{},
	}
	state.digest = pluginOperationStateDigest(state)
	return state
}

func (state *PluginOperationState) Digest() string {
	if state == nil || !state.valid {
		return ""
	}
	return state.digest
}

func (state *PluginOperationState) Lookup(projectRef goal.ProjectRef, pluginID string) (PluginDisposition, bool) {
	if state == nil || !state.valid || projectRef.String() == "" {
		return PluginDisposition{}, false
	}
	disposition, found := state.dispositions[pluginDispositionKey(projectRef.String(), pluginID)]
	return disposition, found
}

func (state *PluginOperationState) Receipt(projectRef goal.ProjectRef, idempotencyKey string) (PluginOperationReceipt, bool) {
	if state == nil || !state.valid || projectRef.String() == "" {
		return PluginOperationReceipt{}, false
	}
	receipt, found := state.receipts[pluginReceiptKey(projectRef.String(), idempotencyKey)]
	return receipt, found
}

func (state *PluginOperationState) Apply(
	catalog *CuratedCatalog,
	request PluginOperationRequest,
	committedAt time.Time,
) (*PluginOperationState, PluginOperationReceipt, bool, error) {
	if state == nil || !state.valid {
		return state, PluginOperationReceipt{}, false, contractError(ErrorPluginOperationInvalid, "state")
	}
	requestDigest := pluginOperationRequestDigest(request)
	receiptKey := pluginReceiptKey(request.ProjectRef.String(), request.IdempotencyKey)
	if previous, exists := state.receipts[receiptKey]; exists {
		if requestDigest == "" || previous.RequestDigest != requestDigest {
			return state, PluginOperationReceipt{}, false,
				contractError(ErrorPluginOperationIdempotencyConflict, "idempotency_key")
		}
		return state, previous, false, nil
	}
	requestDigest, err := validatePluginOperationRequest(catalog, request, committedAt)
	if err != nil {
		return state, PluginOperationReceipt{}, false, err
	}
	if request.Operation == PluginOperationInstall || request.Operation == PluginOperationUpgrade {
		registration, visible, resolveErr := catalog.ResolvePlugin(request.Context, request.PluginID, request.Version)
		if resolveErr != nil || !visible || registration.Digest != request.PluginDigest {
			return state, PluginOperationReceipt{}, false,
				contractError(ErrorPluginOperationCatalogDenied, "plugin")
		}
	}
	dispositionKey := pluginDispositionKey(request.ProjectRef.String(), request.PluginID)
	before, exists := state.dispositions[dispositionKey]
	curation := catalog.Registration()
	after, err := transitionPluginDisposition(before, exists, request, committedAt, curation)
	if err != nil {
		return state, PluginOperationReceipt{}, false, err
	}
	receipt := newPluginOperationReceipt(request, requestDigest, before, after, curation, committedAt)
	next := clonePluginOperationState(state)
	next.dispositions[dispositionKey] = after
	next.receipts[receiptKey] = receipt
	next.digest = pluginOperationStateDigest(next)
	return next, receipt, true, nil
}

func validatePluginOperationRequest(catalog *CuratedCatalog, request PluginOperationRequest, committedAt time.Time) (string, error) {
	if catalog == nil || catalog.Digest() == "" || request.CuratedCatalogDigest != catalog.Digest() ||
		!validSurfaceDigest(request.CuratedCatalogDigest) {
		return "", contractError(ErrorPluginOperationCatalogDenied, "catalog_digest")
	}
	curation := catalog.Registration()
	if !validPluginOperationStrings(curation.Spec.ID, curation.Spec.Version, curation.Spec.ReviewRef,
		curation.Spec.ReviewDigest, curation.Digest) || !validSurfaceDigest(curation.Spec.ReviewDigest) ||
		!validSurfaceDigest(curation.Digest) || catalog.validateContext(request.Context) != nil {
		return "", contractError(ErrorPluginOperationCatalogDenied, "catalog")
	}
	if !validPluginOperationRef(request.RequestRef) || !validPluginOperationRef(request.IdempotencyKey) ||
		!validPluginOperationRef(request.ActorRef.String()) || !validPluginOperationRef(request.ProjectRef.String()) ||
		request.Context.capability.String() != PluginOperationsCapability || !request.Context.valid ||
		!request.Context.scope.valid || request.Context.scope.projectRef != request.ProjectRef ||
		!validToolID(request.PluginID) || !validSurfaceDigest(request.PluginDigest) ||
		!validToolID(request.ReasonCode) || !validPluginOperationTime(request.RequestedAt) ||
		!validPluginOperationTime(committedAt) || request.RequestedAt.After(committedAt) ||
		!validPluginOperationRequestUTF8(request) {
		return "", contractError(ErrorPluginOperationInvalid, "request")
	}
	if _, err := parseVersion(request.Version); err != nil {
		return "", contractError(ErrorPluginOperationInvalid, "version")
	}
	if request.ExpectedRevision == ^uint64(0) {
		return "", contractError(ErrorPluginOperationInvalid, "expected_revision")
	}
	switch request.Operation {
	case PluginOperationInstall, PluginOperationUpgrade, PluginOperationDisable, PluginOperationRemove:
	default:
		return "", contractError(ErrorPluginOperationInvalid, "operation")
	}
	request.RequestedAt = request.RequestedAt.UTC()
	digest := pluginOperationRequestDigest(request)
	if digest == "" {
		return "", contractError(ErrorPluginOperationInvalid, "request_digest")
	}
	return digest, nil
}

func validPluginOperationRef(value string) bool {
	return validSkillEvidenceRef(value) && !strings.ContainsAny(value, "\r\n")
}

func validPluginOperationTime(value time.Time) bool {
	_, err := value.UTC().MarshalJSON()
	return !value.IsZero() && value.Year() >= 0 && value.Year() <= 9999 && err == nil
}

func validPluginOperationRequestUTF8(request PluginOperationRequest) bool {
	return validPluginOperationStrings(
		request.RequestRef, request.IdempotencyKey, request.ActorRef.String(), request.ProjectRef.String(),
		request.Context.capability.String(), request.Context.scope.productID, request.Context.scope.projectRef.String(),
		string(request.Context.scope.role), request.Context.scope.goalRef.String(), request.CuratedCatalogDigest,
		string(request.Operation), request.PluginID, request.Version, request.PluginDigest, request.ReasonCode,
	)
}

func validPluginOperationStrings(values ...string) bool {
	for _, value := range values {
		if !utf8.ValidString(value) {
			return false
		}
	}
	return true
}

func transitionPluginDisposition(
	before PluginDisposition,
	exists bool,
	request PluginOperationRequest,
	committedAt time.Time,
	curation CuratedRegistration,
) (PluginDisposition, error) {
	if (!exists && request.ExpectedRevision != 0) ||
		(exists && request.ExpectedRevision != before.Revision) {
		return PluginDisposition{}, contractError(ErrorPluginOperationRevisionConflict, "expected_revision")
	}
	if exists && committedAt.Before(before.ChangedAt) {
		return PluginDisposition{}, contractError(ErrorPluginOperationTransitionDenied, "committed_at")
	}
	if exists {
		beforeCatalogVersion, beforeErr := parseVersion(before.CuratedCatalogVersion)
		currentCatalogVersion, currentErr := parseVersion(curation.Spec.Version)
		if beforeErr != nil || currentErr != nil || before.CuratedCatalogID != curation.Spec.ID ||
			currentCatalogVersion < beforeCatalogVersion ||
			currentCatalogVersion == beforeCatalogVersion && before.CuratedCatalogDigest != curation.Digest {
			return PluginDisposition{}, contractError(ErrorPluginOperationCatalogDenied, "catalog_revision")
		}
	}
	after := before
	switch request.Operation {
	case PluginOperationInstall:
		if exists && before.Status != PluginDispositionRemoved {
			return PluginDisposition{}, contractError(ErrorPluginOperationTransitionDenied, "install")
		}
		if exists {
			beforeVersion, beforeErr := parseVersion(before.Version)
			targetVersion, targetErr := parseVersion(request.Version)
			if beforeErr != nil || targetErr != nil || targetVersion < beforeVersion ||
				targetVersion == beforeVersion && request.PluginDigest != before.PluginDigest {
				return PluginDisposition{}, contractError(ErrorPluginOperationTransitionDenied, "install_history")
			}
		}
		after.Status = PluginDispositionEnabled
	case PluginOperationUpgrade:
		currentVersion, currentErr := parseVersion(before.Version)
		targetVersion, targetErr := parseVersion(request.Version)
		if !exists || before.Status == PluginDispositionRemoved || currentErr != nil || targetErr != nil ||
			targetVersion <= currentVersion {
			return PluginDisposition{}, contractError(ErrorPluginOperationTransitionDenied, "upgrade")
		}
	case PluginOperationDisable:
		if !exists || before.Status != PluginDispositionEnabled ||
			before.Version != request.Version || before.PluginDigest != request.PluginDigest {
			return PluginDisposition{}, contractError(ErrorPluginOperationTransitionDenied, "disable")
		}
		after.Status = PluginDispositionDisabled
	case PluginOperationRemove:
		if !exists || before.Status != PluginDispositionDisabled ||
			before.Version != request.Version || before.PluginDigest != request.PluginDigest {
			return PluginDisposition{}, contractError(ErrorPluginOperationTransitionDenied, "remove")
		}
		after.Status = PluginDispositionRemoved
	}
	after.ProjectRef = request.ProjectRef.String()
	after.PluginID = request.PluginID
	after.Version = request.Version
	after.PluginDigest = request.PluginDigest
	after.CuratedCatalogID = curation.Spec.ID
	after.CuratedCatalogVersion = curation.Spec.Version
	after.CuratedCatalogDigest = curation.Digest
	after.Revision = request.ExpectedRevision + 1
	after.ChangedAt = committedAt.UTC()
	return after, nil
}

func newPluginOperationReceipt(
	request PluginOperationRequest,
	requestDigest string,
	before PluginDisposition,
	after PluginDisposition,
	curation CuratedRegistration,
	committedAt time.Time,
) PluginOperationReceipt {
	receipt := PluginOperationReceipt{
		RequestRef: request.RequestRef, RequestDigest: requestDigest,
		IdempotencyKey: request.IdempotencyKey, ActorRef: request.ActorRef.String(),
		ProjectRef: request.ProjectRef.String(), Operation: request.Operation,
		ReasonCode: request.ReasonCode, PluginID: request.PluginID,
		Before: before, After: after,
		CurationReviewRef: curation.Spec.ReviewRef, CurationReviewDigest: curation.Spec.ReviewDigest,
		RequestedAt: request.RequestedAt.UTC(), CommittedAt: committedAt.UTC(),
	}
	receipt.Digest = pluginOperationReceiptDigest(receipt)
	receipt.Ref = "receipt:plugin-operation:" + receipt.Digest[len("sha256:"):]
	return receipt
}

func clonePluginOperationState(source *PluginOperationState) *PluginOperationState {
	next := &PluginOperationState{
		valid: true, digest: source.digest,
		dispositions: make(map[string]PluginDisposition, len(source.dispositions)),
		receipts:     make(map[string]PluginOperationReceipt, len(source.receipts)),
	}
	for key, value := range source.dispositions {
		next.dispositions[key] = value
	}
	for key, value := range source.receipts {
		next.receipts[key] = value
	}
	return next
}

func pluginDispositionKey(projectRef, pluginID string) string { return projectRef + "\x00" + pluginID }
func pluginReceiptKey(projectRef, idempotencyKey string) string {
	return projectRef + "\x00" + idempotencyKey
}

func pluginOperationRequestDigest(request PluginOperationRequest) string {
	if !validPluginOperationRequestUTF8(request) {
		return ""
	}
	return digestPluginOperationFact("orquesta.tooling.plugin-operation-request.v2", struct {
		RequestRef, IdempotencyKey, ActorRef, ProjectRef                             string
		Capability, ContextProductID, ContextProjectRef, ContextRole, ContextGoalRef string
		CatalogDigest, Operation, PluginID, Version, PluginDigest, ReasonCode        string
		RequestedAt                                                                  string
		ExpectedRevision                                                             uint64
		ContextValid, ScopeValid                                                     bool
	}{
		request.RequestRef, request.IdempotencyKey, request.ActorRef.String(), request.ProjectRef.String(),
		request.Context.capability.String(), request.Context.scope.productID,
		request.Context.scope.projectRef.String(), string(request.Context.scope.role),
		request.Context.scope.goalRef.String(), request.CuratedCatalogDigest, string(request.Operation),
		request.PluginID, request.Version, request.PluginDigest, request.ReasonCode,
		request.RequestedAt.UTC().Format(time.RFC3339Nano), request.ExpectedRevision,
		request.Context.valid, request.Context.scope.valid,
	})
}

func pluginOperationReceiptDigest(receipt PluginOperationReceipt) string {
	if !validPluginOperationReceiptUTF8(receipt) {
		return ""
	}
	receipt.Ref, receipt.Digest = "", ""
	return digestPluginOperationFact("orquesta.tooling.plugin-operation-receipt.v2", receipt)
}

func validPluginOperationDispositionUTF8(disposition PluginDisposition) bool {
	return validPluginOperationStrings(
		disposition.ProjectRef, disposition.PluginID, disposition.Version, disposition.PluginDigest,
		disposition.CuratedCatalogID, disposition.CuratedCatalogVersion, disposition.CuratedCatalogDigest,
		string(disposition.Status),
	)
}

func validPluginOperationReceiptUTF8(receipt PluginOperationReceipt) bool {
	return validPluginOperationStrings(
		receipt.Ref, receipt.Digest, receipt.RequestRef, receipt.RequestDigest, receipt.IdempotencyKey,
		receipt.ActorRef, receipt.ProjectRef, string(receipt.Operation), receipt.ReasonCode, receipt.PluginID,
		receipt.CurationReviewRef, receipt.CurationReviewDigest,
	) && validPluginOperationDispositionUTF8(receipt.Before) && validPluginOperationDispositionUTF8(receipt.After)
}

func pluginOperationStateDigest(state *PluginOperationState) string {
	dispositions := make([]PluginDisposition, 0, len(state.dispositions))
	for _, disposition := range state.dispositions {
		if !validPluginOperationDispositionUTF8(disposition) {
			return ""
		}
		dispositions = append(dispositions, disposition)
	}
	sort.Slice(dispositions, func(left, right int) bool {
		return pluginDispositionKey(dispositions[left].ProjectRef, dispositions[left].PluginID) <
			pluginDispositionKey(dispositions[right].ProjectRef, dispositions[right].PluginID)
	})
	receipts := make([]PluginOperationReceipt, 0, len(state.receipts))
	for _, receipt := range state.receipts {
		if !validPluginOperationReceiptUTF8(receipt) {
			return ""
		}
		receipts = append(receipts, receipt)
	}
	sort.Slice(receipts, func(left, right int) bool {
		return pluginReceiptKey(receipts[left].ProjectRef, receipts[left].IdempotencyKey) <
			pluginReceiptKey(receipts[right].ProjectRef, receipts[right].IdempotencyKey)
	})
	return digestPluginOperationFact("orquesta.tooling.plugin-operation-state.v2", struct {
		Dispositions []PluginDisposition      `json:"dispositions"`
		Receipts     []PluginOperationReceipt `json:"receipts"`
	}{dispositions, receipts})
}

func digestPluginOperationFact(contract string, value any) string {
	if !utf8.ValidString(contract) {
		return ""
	}
	encoded, err := json.Marshal(struct {
		Contract string `json:"contract"`
		Value    any    `json:"value"`
	}{contract, value})
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:])
}
