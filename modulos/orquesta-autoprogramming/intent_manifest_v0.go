package orquestaautoprogramming

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"sort"
	"strings"
	"sync"
)

const AutoprogrammingIntentManifestSchemaV0 = "orquesta_autoprogramming_intent_manifest.v0"

const (
	AutoprogrammingIntentManifestContentLegacyRequestV0      = "legacy_request"
	AutoprogrammingIntentManifestContentPrepareRunEnvelopeV0 = "prepare_run_envelope"
)

// Leave framing headroom below the 512 KiB public MCP/HTTP request ceiling.
const AutoprogrammingIntentManifestMaxBytesV0 = 480 << 10

var ErrAutoprogrammingIntentManifestNotFoundV0 = errors.New("intent_manifest_not_found")

// AutoprogrammingIntentManifestV0 preserves the normalized input request bytes
// before runtime-specific markers or compact projections are introduced.
type AutoprogrammingIntentManifestV0 struct {
	SchemaVersion string `json:"schema_version"`
	ManifestRef   string `json:"manifest_ref"`
	RequestRef    string `json:"request_ref"`
	RequestSHA256 string `json:"request_sha256"`
	RequestJSON   []byte `json:"request_json"`
}

type AutoprogrammingIntentManifestContentV0 struct {
	Kind               string
	Request            AutoprogrammingRequestV0
	PrepareRunEnvelope *AutoprogrammingPrepareRunEnvelopeV0
}

type AutoprogrammingIntentManifestStorePortV0 interface {
	CreateAutoprogrammingIntentManifestIfAbsentV0(context.Context, AutoprogrammingIntentManifestV0) (AutoprogrammingIntentManifestV0, error)
	LoadAutoprogrammingIntentManifestV0(context.Context, string) (AutoprogrammingIntentManifestV0, error)
}

// InMemoryAutoprogrammingIntentManifestStoreV0 is a concurrency-safe test and
// fake-runtime adapter. Production composition must inject durable storage.
type InMemoryAutoprogrammingIntentManifestStoreV0 struct {
	mu              sync.Mutex
	byRequestRef    map[string]AutoprogrammingIntentManifestV0
	claimsByIdemKey map[string]AutoprogrammingPrepareRunIdempotencyClaimV0
}

func NewInMemoryAutoprogrammingIntentManifestStoreV0() *InMemoryAutoprogrammingIntentManifestStoreV0 {
	return &InMemoryAutoprogrammingIntentManifestStoreV0{
		byRequestRef:    map[string]AutoprogrammingIntentManifestV0{},
		claimsByIdemKey: map[string]AutoprogrammingPrepareRunIdempotencyClaimV0{},
	}
}
func (store *InMemoryAutoprogrammingIntentManifestStoreV0) CreateAutoprogrammingIntentManifestIfAbsentV0(ctx context.Context, manifest AutoprogrammingIntentManifestV0) (AutoprogrammingIntentManifestV0, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return AutoprogrammingIntentManifestV0{}, err
		}
	}
	manifest = NormalizeAutoprogrammingIntentManifestV0(manifest)
	if len(ValidateAutoprogrammingIntentManifestV0(manifest)) != 0 {
		return AutoprogrammingIntentManifestV0{}, errInMemoryIntentManifestInvalidV0
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if existing, ok := store.byRequestRef[manifest.RequestRef]; ok {
		if existing.RequestSHA256 != manifest.RequestSHA256 || !intentManifestBytesEqualV0(existing.RequestJSON, manifest.RequestJSON) {
			return AutoprogrammingIntentManifestV0{}, errInMemoryIntentManifestConflictV0
		}
		return NormalizeAutoprogrammingIntentManifestV0(existing), nil
	}
	store.byRequestRef[manifest.RequestRef] = NormalizeAutoprogrammingIntentManifestV0(manifest)
	return NormalizeAutoprogrammingIntentManifestV0(manifest), nil
}
func (store *InMemoryAutoprogrammingIntentManifestStoreV0) LoadAutoprogrammingIntentManifestV0(ctx context.Context, requestRef string) (AutoprogrammingIntentManifestV0, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return AutoprogrammingIntentManifestV0{}, err
		}
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	manifest, ok := store.byRequestRef[strings.TrimSpace(requestRef)]
	if !ok {
		return AutoprogrammingIntentManifestV0{}, ErrAutoprogrammingIntentManifestNotFoundV0
	}
	return NormalizeAutoprogrammingIntentManifestV0(manifest), nil
}

func (store *InMemoryAutoprogrammingIntentManifestStoreV0) CreateAutoprogrammingPrepareRunIdempotencyClaimIfAbsentV0(
	ctx context.Context,
	claim AutoprogrammingPrepareRunIdempotencyClaimV0,
) (AutoprogrammingPrepareRunIdempotencyClaimV0, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return AutoprogrammingPrepareRunIdempotencyClaimV0{}, err
		}
	}
	claim = NormalizeAutoprogrammingPrepareRunIdempotencyClaimV0(claim)
	if len(ValidateAutoprogrammingPrepareRunIdempotencyClaimV0(claim)) != 0 {
		return AutoprogrammingPrepareRunIdempotencyClaimV0{}, ErrAutoprogrammingPrepareRunIdempotencyClaimInvalidV0
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.claimsByIdemKey == nil {
		store.claimsByIdemKey = map[string]AutoprogrammingPrepareRunIdempotencyClaimV0{}
	}
	if existing, ok := store.claimsByIdemKey[claim.IdempotencyKey]; ok {
		if !EqualAutoprogrammingPrepareRunIdempotencyClaimV0(existing, claim) {
			return AutoprogrammingPrepareRunIdempotencyClaimV0{}, ErrAutoprogrammingPrepareRunIdempotencyClaimConflictV0
		}
		return NormalizeAutoprogrammingPrepareRunIdempotencyClaimV0(existing), nil
	}
	store.claimsByIdemKey[claim.IdempotencyKey] = claim
	return claim, nil
}

var errInMemoryIntentManifestInvalidV0 = errors.New("intent_manifest_invalid")
var errInMemoryIntentManifestConflictV0 = errors.New("intent_manifest_conflict")

func intentManifestBytesEqualV0(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

var autoprogrammingIntentManifestSHA256V0 = regexp.MustCompile(`^[0-9a-f]{64}$`)
var autoprogrammingIntentManifestRefTokenV0 = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

const autoprogrammingIntentManifestRequestRefMaxBytesV0 = 180

func BuildAutoprogrammingIntentManifestV0(request AutoprogrammingRequestV0) (AutoprogrammingIntentManifestV0, []AutoprogrammingRequestIssueV0) {
	canonical, canonicalIssues := CanonicalAutoprogrammingIntentRequestV0(request)
	if len(canonicalIssues) != 0 {
		return AutoprogrammingIntentManifestV0{}, canonicalIssues
	}
	if validation := ValidateAutoprogrammingRequestV0(canonical); !validation.Accepted {
		return AutoprogrammingIntentManifestV0{}, validation.Issues
	}
	raw, err := json.Marshal(canonical)
	if err != nil {
		return AutoprogrammingIntentManifestV0{}, []AutoprogrammingRequestIssueV0{autoprogrammingRequestIssueV0("intent_manifest_json_invalid", "request", "intent_manifest_json_invalid")}
	}
	return AutoprogrammingIntentManifestFromRequestJSONV0(raw)
}

// BuildAutoprogrammingIntentManifestFromPrepareRunEnvelopeV0 persists the
// complete public command receipt in the existing manifest authority. Required
// setting values must already have been removed by the inbound adapter.
func BuildAutoprogrammingIntentManifestFromPrepareRunEnvelopeV0(
	envelope AutoprogrammingPrepareRunEnvelopeV0,
) (AutoprogrammingIntentManifestV0, []AutoprogrammingRequestIssueV0) {
	canonical, issues := CanonicalAutoprogrammingPrepareRunEnvelopeV0(envelope)
	if len(issues) != 0 {
		return AutoprogrammingIntentManifestV0{}, issues
	}
	raw, err := json.Marshal(canonical)
	if err != nil {
		return AutoprogrammingIntentManifestV0{}, []AutoprogrammingRequestIssueV0{autoprogrammingRequestIssueV0("intent_manifest_json_invalid", "prepare_run_envelope", "intent_manifest_json_invalid")}
	}
	return AutoprogrammingIntentManifestFromRequestJSONV0(raw)
}

// AutoprogrammingIntentManifestFromRequestJSONV0 retains raw persisted bytes;
// it never remarshal them while checking their request identity.
func AutoprogrammingIntentManifestFromRequestJSONV0(raw []byte) (AutoprogrammingIntentManifestV0, []AutoprogrammingRequestIssueV0) {
	if len(raw) > AutoprogrammingIntentManifestMaxBytesV0 {
		return AutoprogrammingIntentManifestV0{}, []AutoprogrammingRequestIssueV0{autoprogrammingRequestIssueV0("intent_manifest_request_json_too_large", "request_json", "intent_manifest_request_json_too_large")}
	}
	requestRef, canonicalRaw, canonicalIssues := canonicalAutoprogrammingIntentManifestContentV0(raw)
	if len(canonicalIssues) != 0 {
		return AutoprogrammingIntentManifestV0{}, canonicalIssues
	}
	if !bytes.Equal(raw, canonicalRaw) {
		return AutoprogrammingIntentManifestV0{}, []AutoprogrammingRequestIssueV0{autoprogrammingRequestIssueV0("intent_manifest_request_json_noncanonical", "request_json", "intent_manifest_request_json_noncanonical")}
	}
	sum := sha256.Sum256(raw)
	manifest := AutoprogrammingIntentManifestV0{SchemaVersion: AutoprogrammingIntentManifestSchemaV0, ManifestRef: "intent-manifest-ref-" + requestRef, RequestRef: requestRef, RequestSHA256: hex.EncodeToString(sum[:]), RequestJSON: append([]byte(nil), raw...)}
	return manifest, ValidateAutoprogrammingIntentManifestV0(manifest)
}

// CanonicalAutoprogrammingIntentRequestV0 produces the exact semantic request
// that Orquesta consumes before runtime markers and compact projections.
// Ordering is preserved where it governs task/group execution; whitespace,
// duplicate scalar refs, nil slices and implicit limits are normalized.
func CanonicalAutoprogrammingIntentRequestV0(request AutoprogrammingRequestV0) (AutoprogrammingRequestV0, []AutoprogrammingRequestIssueV0) {
	if validation := ValidateAutoprogrammingRequestV0(request); !validation.Accepted {
		return AutoprogrammingRequestV0{}, validation.Issues
	}
	request.RequestRef = strings.TrimSpace(request.RequestRef)
	request.ProjectRef = strings.TrimSpace(request.ProjectRef)
	request.WorktreeRef = strings.TrimSpace(request.WorktreeRef)
	request.BranchRef = strings.TrimSpace(request.BranchRef)
	tasks := make([]AutoprogrammingTaskGroupCandidateV0, 0, len(request.Tasks))
	for _, task := range request.Tasks {
		normalized, err := normalizeAutoprogrammingTaskCandidateV0(task, strings.TrimSpace(task.TaskRef), normalizeAutoprogrammingTaskAreaV0(task.Area))
		if err != nil {
			return AutoprogrammingRequestV0{}, []AutoprogrammingRequestIssueV0{autoprogrammingRequestIssueV0("intent_manifest_request_normalization_invalid", "tasks", "intent_manifest_request_normalization_invalid")}
		}
		tasks = append(tasks, normalized)
	}
	request.Tasks = tasks
	request.WriteSet = compactStringsV0(request.WriteSet)
	request.RequiredTests = compactStringsV0(request.RequiredTests)
	aliases := make([]AutoprogrammingAreaAliasV0, 0, len(request.AreaAliases))
	seenAliases := map[string]struct{}{}
	for _, alias := range request.AreaAliases {
		alias.Alias = strings.Join(autoprogrammingCanonicalPathTokensV0(alias.Alias), "-")
		alias.Area = normalizeAutoprogrammingTaskAreaV0(alias.Area)
		if alias.Alias != "" && alias.Area != "" {
			key := alias.Area + "\x00" + alias.Alias
			if _, exists := seenAliases[key]; exists {
				continue
			}
			seenAliases[key] = struct{}{}
			aliases = append(aliases, alias)
		}
	}
	sort.Slice(aliases, func(i, j int) bool {
		if aliases[i].Area != aliases[j].Area {
			return aliases[i].Area < aliases[j].Area
		}
		return aliases[i].Alias < aliases[j].Alias
	})
	request.AreaAliases = aliases
	liveWorks := make([]AutoprogrammingLiveWorkV0, 0, len(request.LiveWorks))
	for _, live := range request.LiveWorks {
		live.WorkRef = strings.TrimSpace(live.WorkRef)
		live.TaskRef = strings.TrimSpace(live.TaskRef)
		live.AgentRef = strings.TrimSpace(live.AgentRef)
		live.Status = strings.ToLower(strings.TrimSpace(live.Status))
		live.WriteSet = compactStringsV0(live.WriteSet)
		liveWorks = append(liveWorks, live)
	}
	request.LiveWorks = liveWorks
	request.BacklogScan = normalizeAutoprogrammingBacklogScanV0(request.BacklogScan)
	request.MaxTaskRefs = autoprogrammingRequestMaxTaskRefsV0(request)
	request.MaxAreas = autoprogrammingRequestMaxAreasV0(request)
	request.MaxWriteSetEntries = autoprogrammingRequestMaxWriteSetEntriesV0(request)
	limits := autoprogrammingDelegationLimitsForRequestV0(request)
	request.MaxDelegationDepth = limits.maxDelegationDepth
	request.MaxSubagentsPerAgent = limits.maxSubagentsPerAgent
	request.MaxRecursiveAgents = limits.maxRecursiveAgents
	return request, nil
}

func NormalizeAutoprogrammingIntentManifestV0(manifest AutoprogrammingIntentManifestV0) AutoprogrammingIntentManifestV0 {
	manifest.SchemaVersion = strings.TrimSpace(manifest.SchemaVersion)
	manifest.ManifestRef = strings.TrimSpace(manifest.ManifestRef)
	manifest.RequestRef = strings.TrimSpace(manifest.RequestRef)
	// Do not lowercase hashes: upper case is malformed evidence, not a repair.
	manifest.RequestSHA256 = strings.TrimSpace(manifest.RequestSHA256)
	manifest.RequestJSON = append([]byte(nil), manifest.RequestJSON...)
	return manifest
}

func ValidateAutoprogrammingIntentManifestV0(manifest AutoprogrammingIntentManifestV0) []AutoprogrammingRequestIssueV0 {
	manifest = NormalizeAutoprogrammingIntentManifestV0(manifest)
	var issues []AutoprogrammingRequestIssueV0
	if len(manifest.RequestJSON) > AutoprogrammingIntentManifestMaxBytesV0 {
		issues = append(issues, autoprogrammingRequestIssueV0("intent_manifest_request_json_too_large", "request_json", "intent_manifest_request_json_too_large"))
	}
	if manifest.SchemaVersion != AutoprogrammingIntentManifestSchemaV0 {
		issues = append(issues, autoprogrammingRequestIssueV0("intent_manifest_schema_invalid", "schema_version", "intent_manifest_schema_invalid"))
	}
	if !autoprogrammingTypedRefV0(manifest.ManifestRef, "intent-manifest-ref-") {
		issues = append(issues, autoprogrammingRequestIssueV0("intent_manifest_ref_invalid", "manifest_ref", "intent_manifest_ref_invalid"))
	}
	if !autoprogrammingIntentManifestRefTokenV0.MatchString(manifest.RequestRef) || len(manifest.RequestRef) > autoprogrammingIntentManifestRequestRefMaxBytesV0 || strings.Contains(manifest.RequestRef, "..") {
		issues = append(issues, autoprogrammingRequestIssueV0("intent_manifest_request_ref_invalid", "request_ref", "intent_manifest_request_ref_invalid"))
	}
	if !autoprogrammingIntentManifestSHA256V0.MatchString(manifest.RequestSHA256) {
		issues = append(issues, autoprogrammingRequestIssueV0("intent_manifest_sha256_invalid", "request_sha256", "intent_manifest_sha256_invalid"))
	}
	requestRef, canonicalJSON, canonicalIssues := canonicalAutoprogrammingIntentManifestContentV0(manifest.RequestJSON)
	if len(canonicalIssues) != 0 {
		return append(issues, canonicalIssues...)
	}
	if !bytes.Equal(canonicalJSON, manifest.RequestJSON) {
		issues = append(issues, autoprogrammingRequestIssueV0("intent_manifest_request_json_noncanonical", "request_json", "intent_manifest_request_json_noncanonical"))
	}
	if requestRef != manifest.RequestRef {
		issues = append(issues, autoprogrammingRequestIssueV0("intent_manifest_request_ref_mismatch", "request_ref", "intent_manifest_request_ref_mismatch"))
	}
	if manifest.ManifestRef != "intent-manifest-ref-"+manifest.RequestRef {
		issues = append(issues, autoprogrammingRequestIssueV0("intent_manifest_ref_mismatch", "manifest_ref", "intent_manifest_ref_mismatch"))
	}
	sum := sha256.Sum256(manifest.RequestJSON)
	if manifest.RequestSHA256 != hex.EncodeToString(sum[:]) {
		issues = append(issues, autoprogrammingRequestIssueV0("intent_manifest_sha256_mismatch", "request_sha256", "intent_manifest_sha256_mismatch"))
	}
	return issues
}

// ProjectAutoprogrammingIntentManifestContentV0 exposes the canonical domain
// request without replacing the durable bytes. It is used only to accept a
// pre-envelope manifest when its legacy request is canonically identical.
func ProjectAutoprogrammingIntentManifestContentV0(
	manifest AutoprogrammingIntentManifestV0,
) (AutoprogrammingIntentManifestContentV0, []AutoprogrammingRequestIssueV0) {
	if issues := ValidateAutoprogrammingIntentManifestV0(manifest); len(issues) != 0 {
		return AutoprogrammingIntentManifestContentV0{}, issues
	}
	var discriminator struct {
		SchemaVersion string `json:"schema_version"`
	}
	if err := json.Unmarshal(manifest.RequestJSON, &discriminator); err != nil {
		return AutoprogrammingIntentManifestContentV0{}, []AutoprogrammingRequestIssueV0{
			autoprogrammingRequestIssueV0("intent_manifest_request_json_invalid", "request_json", "intent_manifest_request_json_invalid"),
		}
	}
	if strings.TrimSpace(discriminator.SchemaVersion) == AutoprogrammingPrepareRunEnvelopeSchemaV0 {
		envelope, err := decodeAutoprogrammingPrepareRunEnvelopeJSONV0(manifest.RequestJSON)
		if err != nil {
			return AutoprogrammingIntentManifestContentV0{}, []AutoprogrammingRequestIssueV0{
				autoprogrammingRequestIssueV0("intent_manifest_request_json_invalid", "request_json", "intent_manifest_request_json_invalid"),
			}
		}
		canonical, issues := CanonicalAutoprogrammingPrepareRunEnvelopeV0(envelope)
		if len(issues) != 0 {
			return AutoprogrammingIntentManifestContentV0{}, issues
		}
		return AutoprogrammingIntentManifestContentV0{
			Kind:               AutoprogrammingIntentManifestContentPrepareRunEnvelopeV0,
			Request:            canonical.AutoprogrammingRequest,
			PrepareRunEnvelope: &canonical,
		}, nil
	}
	request, err := decodeAutoprogrammingIntentRequestJSONV0(manifest.RequestJSON)
	if err != nil {
		return AutoprogrammingIntentManifestContentV0{}, []AutoprogrammingRequestIssueV0{
			autoprogrammingRequestIssueV0("intent_manifest_request_json_invalid", "request_json", "intent_manifest_request_json_invalid"),
		}
	}
	canonical, issues := CanonicalAutoprogrammingIntentRequestV0(request)
	if len(issues) != 0 {
		return AutoprogrammingIntentManifestContentV0{}, issues
	}
	return AutoprogrammingIntentManifestContentV0{
		Kind:    AutoprogrammingIntentManifestContentLegacyRequestV0,
		Request: canonical,
	}, nil
}

func canonicalAutoprogrammingIntentManifestContentV0(raw []byte) (string, []byte, []AutoprogrammingRequestIssueV0) {
	if len(raw) == 0 || !autoprogrammingIntentJSONKeysUniqueV0(raw) {
		return "", nil, []AutoprogrammingRequestIssueV0{autoprogrammingRequestIssueV0("intent_manifest_request_json_invalid", "request_json", "intent_manifest_request_json_invalid")}
	}
	var discriminator struct {
		SchemaVersion string `json:"schema_version"`
	}
	if err := json.Unmarshal(raw, &discriminator); err != nil {
		return "", nil, []AutoprogrammingRequestIssueV0{autoprogrammingRequestIssueV0("intent_manifest_request_json_invalid", "request_json", "intent_manifest_request_json_invalid")}
	}
	if strings.TrimSpace(discriminator.SchemaVersion) == AutoprogrammingPrepareRunEnvelopeSchemaV0 {
		envelope, err := decodeAutoprogrammingPrepareRunEnvelopeJSONV0(raw)
		if err != nil {
			return "", nil, []AutoprogrammingRequestIssueV0{autoprogrammingRequestIssueV0("intent_manifest_request_json_invalid", "request_json", "intent_manifest_request_json_invalid")}
		}
		canonical, canonicalIssues := CanonicalAutoprogrammingPrepareRunEnvelopeV0(envelope)
		if len(canonicalIssues) != 0 {
			return "", nil, canonicalIssues
		}
		canonicalRaw, err := json.Marshal(canonical)
		if err != nil {
			return "", nil, []AutoprogrammingRequestIssueV0{autoprogrammingRequestIssueV0("intent_manifest_request_json_invalid", "request_json", "intent_manifest_request_json_invalid")}
		}
		return strings.TrimSpace(canonical.AutoprogrammingRequest.RequestRef), canonicalRaw, nil
	}
	request, err := decodeAutoprogrammingIntentRequestJSONV0(raw)
	if err != nil {
		return "", nil, []AutoprogrammingRequestIssueV0{autoprogrammingRequestIssueV0("intent_manifest_request_json_invalid", "request_json", "intent_manifest_request_json_invalid")}
	}
	canonical, canonicalIssues := CanonicalAutoprogrammingIntentRequestV0(request)
	if len(canonicalIssues) != 0 {
		return "", nil, canonicalIssues
	}
	if validation := ValidateAutoprogrammingRequestV0(canonical); !validation.Accepted {
		return "", nil, validation.Issues
	}
	canonicalRaw, err := json.Marshal(canonical)
	if err != nil {
		return "", nil, []AutoprogrammingRequestIssueV0{autoprogrammingRequestIssueV0("intent_manifest_request_json_invalid", "request_json", "intent_manifest_request_json_invalid")}
	}
	return strings.TrimSpace(canonical.RequestRef), canonicalRaw, nil
}

func decodeAutoprogrammingIntentRequestJSONV0(raw []byte) (AutoprogrammingRequestV0, error) {
	if len(raw) == 0 || !autoprogrammingIntentJSONKeysUniqueV0(raw) {
		return AutoprogrammingRequestV0{}, errors.New("intent_manifest_request_json_invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var request AutoprogrammingRequestV0
	if err := decoder.Decode(&request); err != nil {
		return AutoprogrammingRequestV0{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return AutoprogrammingRequestV0{}, errors.New("intent_manifest_request_json_trailing")
	}
	return request, nil
}

func decodeAutoprogrammingPrepareRunEnvelopeJSONV0(raw []byte) (AutoprogrammingPrepareRunEnvelopeV0, error) {
	if len(raw) == 0 || !autoprogrammingIntentJSONKeysUniqueV0(raw) {
		return AutoprogrammingPrepareRunEnvelopeV0{}, errors.New("intent_manifest_request_json_invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var envelope AutoprogrammingPrepareRunEnvelopeV0
	if err := decoder.Decode(&envelope); err != nil {
		return AutoprogrammingPrepareRunEnvelopeV0{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return AutoprogrammingPrepareRunEnvelopeV0{}, errors.New("intent_manifest_request_json_trailing")
	}
	return envelope, nil
}

func autoprogrammingIntentJSONKeysUniqueV0(raw []byte) bool {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if !autoprogrammingIntentJSONValueKeysUniqueV0(decoder) {
		return false
	}
	_, err := decoder.Token()
	return errors.Is(err, io.EOF)
}

func autoprogrammingIntentJSONValueKeysUniqueV0(decoder *json.Decoder) bool {
	token, err := decoder.Token()
	if err != nil {
		return false
	}
	delim, compound := token.(json.Delim)
	if !compound {
		return true
	}
	switch delim {
	case '{':
		seen := map[string]struct{}{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			key, ok := keyToken.(string)
			if err != nil || !ok {
				return false
			}
			if _, duplicate := seen[key]; duplicate {
				return false
			}
			seen[key] = struct{}{}
			if !autoprogrammingIntentJSONValueKeysUniqueV0(decoder) {
				return false
			}
		}
		end, err := decoder.Token()
		return err == nil && end == json.Delim('}')
	case '[':
		for decoder.More() {
			if !autoprogrammingIntentJSONValueKeysUniqueV0(decoder) {
				return false
			}
		}
		end, err := decoder.Token()
		return err == nil && end == json.Delim(']')
	default:
		return false
	}
}

func autoprogrammingTypedRefV0(value, prefix string) bool {
	value = strings.TrimSpace(value)
	return strings.HasPrefix(value, prefix) && len(value) > len(prefix) && autoprogrammingIntentManifestRefTokenV0.MatchString(value)
}
