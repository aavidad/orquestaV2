package orquestacontext

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	CodeContextToolLeaseSchemaVersionV0 = "code_context_tool_lease.v0"

	CodeContextToolLeaseStatusActiveV0    = "active"
	CodeContextToolLeaseStatusCompletedV0 = "completed"
	CodeContextToolLeaseStatusFailedV0    = "failed"
	CodeContextToolLeaseStatusStoppedV0   = "stopped"

	CodeContextToolLeaseCompletionCompletedV0 = "completed"
	CodeContextToolLeaseCompletionFailedV0    = "failed"
	CodeContextToolLeaseCompletionStoppedV0   = "stopped"

	CodeContextToolLeaseDecisionContinueV0    = "continue"
	CodeContextToolLeaseDecisionObserveV0     = "observe"
	CodeContextToolLeaseDecisionRequestStopV0 = "request_stop"

	CodeContextToolLeaseReasonTerminalV0            = "terminal"
	CodeContextToolLeaseReasonActiveRequestsV0      = "active_requests"
	CodeContextToolLeaseReasonLeaseCurrentV0        = "lease_current"
	CodeContextToolLeaseReasonLeaseExpiredV0        = "lease_expired"
	CodeContextToolLeaseReasonLeaseExpiredHighCPUV0 = "lease_expired_high_cpu"

	ErrCodeContextToolLeaseInvalidV0 = "code_context_tool_lease_invalid"
)

type CodeContextToolLeaseRequestV0 struct {
	RequestRef      string   `json:"request_ref,omitempty"`
	CorrelationID   string   `json:"correlation_id,omitempty"`
	RepositoryRef   string   `json:"repository_ref"`
	WorktreeRef     string   `json:"worktree_ref,omitempty"`
	CommitRef       string   `json:"commit_ref,omitempty"`
	QueryHash       string   `json:"query_hash"`
	ToolRef         string   `json:"tool_ref"`
	ProviderKind    string   `json:"provider_kind"`
	OwnerRef        string   `json:"owner_ref,omitempty"`
	StartedAt       string   `json:"started_at"`
	LeaseTTLSeconds int      `json:"lease_ttl_seconds"`
	EvidenceRefs    []string `json:"evidence_refs,omitempty"`
}

type CodeContextToolLeaseV0 struct {
	SchemaVersion   string   `json:"schema_version"`
	LeaseRef        string   `json:"lease_ref"`
	RequestRef      string   `json:"request_ref,omitempty"`
	CorrelationID   string   `json:"correlation_id,omitempty"`
	RepositoryRef   string   `json:"repository_ref"`
	WorktreeRef     string   `json:"worktree_ref,omitempty"`
	CommitRef       string   `json:"commit_ref,omitempty"`
	QueryHash       string   `json:"query_hash"`
	ToolRef         string   `json:"tool_ref"`
	ProviderKind    string   `json:"provider_kind"`
	OwnerRef        string   `json:"owner_ref,omitempty"`
	StartedAt       string   `json:"started_at"`
	LeaseUntil      string   `json:"lease_until"`
	LeaseTTLSeconds int      `json:"lease_ttl_seconds"`
	Status          string   `json:"status"`
	EvidenceRefs    []string `json:"evidence_refs,omitempty"`
}

type CodeContextToolLeaseCompletionV0 struct {
	LeaseRef     string   `json:"lease_ref"`
	ToolRef      string   `json:"tool_ref"`
	CompletedAt  string   `json:"completed_at"`
	Status       string   `json:"status"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type CodeContextToolLeaseListFilterV0 struct {
	RepositoryRef string
	ToolRef       string
	Status        string
}

type CodeContextToolLeaseObservationV0 struct {
	ObservationRef string                 `json:"observation_ref,omitempty"`
	ObservedAt     string                 `json:"observed_at"`
	Lease          CodeContextToolLeaseV0 `json:"lease"`
	LastRequestAt  string                 `json:"last_request_at,omitempty"`
	CPUPercent     int                    `json:"cpu_percent"`
	ActiveRequests int                    `json:"active_requests"`
	CPUHighPercent int                    `json:"cpu_high_percent"`
	EvidenceRefs   []string               `json:"evidence_refs,omitempty"`
}

type CodeContextToolLeaseAssessmentV0 struct {
	AssessmentRef     string   `json:"assessment_ref,omitempty"`
	LeaseRef          string   `json:"lease_ref"`
	ToolRef           string   `json:"tool_ref"`
	ObservedAt        string   `json:"observed_at"`
	Decision          string   `json:"decision"`
	ReasonCode        string   `json:"reason_code"`
	CPUPercent        int      `json:"cpu_percent"`
	ShouldRequestStop bool     `json:"should_request_stop"`
	EvidenceRefs      []string `json:"evidence_refs,omitempty"`
}

type CodeContextToolLeaseListPortV0 interface {
	ListCodeContextToolLeasesV0(context.Context, CodeContextToolLeaseListFilterV0) ([]CodeContextToolLeaseV0, error)
}

type InMemoryCodeContextToolLeaseStoreV0 struct {
	mu     sync.Mutex
	leases map[string]CodeContextToolLeaseV0
}

func NewInMemoryCodeContextToolLeaseStoreV0() *InMemoryCodeContextToolLeaseStoreV0 {
	return &InMemoryCodeContextToolLeaseStoreV0{leases: map[string]CodeContextToolLeaseV0{}}
}

func (store *InMemoryCodeContextToolLeaseStoreV0) BeginCodeContextToolLeaseV0(
	ctx context.Context,
	request CodeContextToolLeaseRequestV0,
) (CodeContextToolLeaseV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return CodeContextToolLeaseV0{}, err
	}
	request = normalizeCodeContextToolLeaseRequestV0(request)
	if err := validateCodeContextToolLeaseRequestV0(request); err != nil {
		return CodeContextToolLeaseV0{}, err
	}
	startedAt, _ := parseCodeContextToolLeaseTimeV0(request.StartedAt)
	lease := CodeContextToolLeaseV0{
		SchemaVersion:   CodeContextToolLeaseSchemaVersionV0,
		LeaseRef:        codeContextToolLeaseRefV0(request),
		RequestRef:      request.RequestRef,
		CorrelationID:   request.CorrelationID,
		RepositoryRef:   request.RepositoryRef,
		WorktreeRef:     request.WorktreeRef,
		CommitRef:       request.CommitRef,
		QueryHash:       request.QueryHash,
		ToolRef:         request.ToolRef,
		ProviderKind:    request.ProviderKind,
		OwnerRef:        request.OwnerRef,
		StartedAt:       request.StartedAt,
		LeaseUntil:      startedAt.Add(time.Duration(request.LeaseTTLSeconds) * time.Second).UTC().Format(time.RFC3339),
		LeaseTTLSeconds: request.LeaseTTLSeconds,
		Status:          CodeContextToolLeaseStatusActiveV0,
		EvidenceRefs:    compactContextStringsV0(request.EvidenceRefs),
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.ensureLeasesLockedV0()
	store.leases[lease.LeaseRef] = lease
	return lease, nil
}

func (store *InMemoryCodeContextToolLeaseStoreV0) FinishCodeContextToolLeaseV0(
	ctx context.Context,
	completion CodeContextToolLeaseCompletionV0,
) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	completion = normalizeCodeContextToolLeaseCompletionV0(completion)
	if err := validateCodeContextToolLeaseCompletionV0(completion); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.ensureLeasesLockedV0()
	lease, ok := store.leases[completion.LeaseRef]
	if !ok {
		return errors.New(ErrCodeContextToolLeaseInvalidV0)
	}
	if lease.ToolRef != completion.ToolRef {
		return errors.New(ErrCodeContextToolLeaseInvalidV0)
	}
	switch completion.Status {
	case CodeContextToolLeaseCompletionCompletedV0:
		lease.Status = CodeContextToolLeaseStatusCompletedV0
	case CodeContextToolLeaseCompletionFailedV0:
		lease.Status = CodeContextToolLeaseStatusFailedV0
	case CodeContextToolLeaseCompletionStoppedV0:
		lease.Status = CodeContextToolLeaseStatusStoppedV0
	default:
		return errors.New(ErrCodeContextToolLeaseInvalidV0)
	}
	lease.EvidenceRefs = compactContextStringsV0(append(lease.EvidenceRefs, completion.EvidenceRefs...))
	store.leases[completion.LeaseRef] = lease
	return nil
}

func (store *InMemoryCodeContextToolLeaseStoreV0) ListCodeContextToolLeasesV0(
	ctx context.Context,
	filter CodeContextToolLeaseListFilterV0,
) ([]CodeContextToolLeaseV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	filter.RepositoryRef = trimContextV0(filter.RepositoryRef)
	filter.ToolRef = trimContextV0(filter.ToolRef)
	filter.Status = trimContextV0(filter.Status)
	store.mu.Lock()
	defer store.mu.Unlock()
	store.ensureLeasesLockedV0()
	out := make([]CodeContextToolLeaseV0, 0, len(store.leases))
	for _, lease := range store.leases {
		if filter.RepositoryRef != "" && lease.RepositoryRef != filter.RepositoryRef {
			continue
		}
		if filter.ToolRef != "" && lease.ToolRef != filter.ToolRef {
			continue
		}
		if filter.Status != "" && lease.Status != filter.Status {
			continue
		}
		out = append(out, lease)
	}
	sort.Slice(out, func(i int, j int) bool {
		return out[i].LeaseRef < out[j].LeaseRef
	})
	return out, nil
}

func EvaluateCodeContextToolLeaseV0(
	observation CodeContextToolLeaseObservationV0,
) (CodeContextToolLeaseAssessmentV0, error) {
	observation = normalizeCodeContextToolLeaseObservationV0(observation)
	if err := validateCodeContextToolLeaseObservationV0(observation); err != nil {
		return CodeContextToolLeaseAssessmentV0{}, err
	}
	observedAt, _ := parseCodeContextToolLeaseTimeV0(observation.ObservedAt)
	leaseUntil, _ := parseCodeContextToolLeaseTimeV0(observation.Lease.LeaseUntil)
	decision := CodeContextToolLeaseDecisionContinueV0
	reason := CodeContextToolLeaseReasonLeaseCurrentV0
	shouldStop := false
	if isCodeContextToolLeaseTerminalV0(observation.Lease.Status) {
		reason = CodeContextToolLeaseReasonTerminalV0
	} else if observation.ActiveRequests > 0 {
		reason = CodeContextToolLeaseReasonActiveRequestsV0
	} else if observedAt.After(leaseUntil) || observedAt.Equal(leaseUntil) {
		decision = CodeContextToolLeaseDecisionRequestStopV0
		reason = CodeContextToolLeaseReasonLeaseExpiredV0
		shouldStop = true
		if observation.CPUPercent >= observation.CPUHighPercent {
			reason = CodeContextToolLeaseReasonLeaseExpiredHighCPUV0
		}
	}
	assessment := CodeContextToolLeaseAssessmentV0{
		AssessmentRef:     firstNonEmptyContextV0(observation.ObservationRef, "code-context-tool-lease-assessment"),
		LeaseRef:          observation.Lease.LeaseRef,
		ToolRef:           observation.Lease.ToolRef,
		ObservedAt:        observation.ObservedAt,
		Decision:          decision,
		ReasonCode:        reason,
		CPUPercent:        observation.CPUPercent,
		ShouldRequestStop: shouldStop,
		EvidenceRefs:      compactContextStringsV0(append(observation.EvidenceRefs, observation.Lease.EvidenceRefs...)),
	}
	return assessment, nil
}

func normalizeCodeContextToolLeaseRequestV0(
	request CodeContextToolLeaseRequestV0,
) CodeContextToolLeaseRequestV0 {
	request.RequestRef = trimContextV0(request.RequestRef)
	request.CorrelationID = trimContextV0(request.CorrelationID)
	request.RepositoryRef = trimContextV0(request.RepositoryRef)
	request.WorktreeRef = trimContextV0(request.WorktreeRef)
	request.CommitRef = trimContextV0(request.CommitRef)
	request.QueryHash = trimContextV0(request.QueryHash)
	request.ToolRef = trimContextV0(request.ToolRef)
	request.ProviderKind = trimContextV0(request.ProviderKind)
	request.OwnerRef = trimContextV0(request.OwnerRef)
	request.StartedAt = trimContextV0(request.StartedAt)
	request.EvidenceRefs = compactContextStringsV0(request.EvidenceRefs)
	if request.LeaseTTLSeconds <= 0 {
		request.LeaseTTLSeconds = int(defaultCodeContextToolLeaseTTLV0.Seconds())
	}
	if request.LeaseTTLSeconds <= 0 {
		request.LeaseTTLSeconds = 1
	}
	return request
}

func validateCodeContextToolLeaseRequestV0(request CodeContextToolLeaseRequestV0) error {
	for _, required := range []string{
		request.RepositoryRef,
		request.QueryHash,
		request.ToolRef,
		request.ProviderKind,
		request.StartedAt,
	} {
		if strings.TrimSpace(required) == "" {
			return errors.New(ErrCodeContextToolLeaseInvalidV0)
		}
	}
	if _, err := parseCodeContextToolLeaseTimeV0(request.StartedAt); err != nil {
		return errors.New(ErrCodeContextToolLeaseInvalidV0)
	}
	if request.LeaseTTLSeconds <= 0 || request.LeaseTTLSeconds > 3600 {
		return errors.New(ErrCodeContextToolLeaseInvalidV0)
	}
	return nil
}

func normalizeCodeContextToolLeaseCompletionV0(
	completion CodeContextToolLeaseCompletionV0,
) CodeContextToolLeaseCompletionV0 {
	completion.LeaseRef = trimContextV0(completion.LeaseRef)
	completion.ToolRef = trimContextV0(completion.ToolRef)
	completion.CompletedAt = trimContextV0(completion.CompletedAt)
	completion.Status = trimContextV0(completion.Status)
	completion.EvidenceRefs = compactContextStringsV0(completion.EvidenceRefs)
	return completion
}

func validateCodeContextToolLeaseCompletionV0(completion CodeContextToolLeaseCompletionV0) error {
	if completion.LeaseRef == "" || completion.ToolRef == "" || completion.CompletedAt == "" {
		return errors.New(ErrCodeContextToolLeaseInvalidV0)
	}
	if _, err := parseCodeContextToolLeaseTimeV0(completion.CompletedAt); err != nil {
		return errors.New(ErrCodeContextToolLeaseInvalidV0)
	}
	switch completion.Status {
	case CodeContextToolLeaseCompletionCompletedV0, CodeContextToolLeaseCompletionFailedV0, CodeContextToolLeaseCompletionStoppedV0:
	default:
		return errors.New(ErrCodeContextToolLeaseInvalidV0)
	}
	return nil
}

func normalizeCodeContextToolLeaseObservationV0(
	observation CodeContextToolLeaseObservationV0,
) CodeContextToolLeaseObservationV0 {
	observation.ObservationRef = trimContextV0(observation.ObservationRef)
	observation.ObservedAt = trimContextV0(observation.ObservedAt)
	observation.LastRequestAt = trimContextV0(observation.LastRequestAt)
	observation.CPUPercent = nonNegativeContextIntV0(observation.CPUPercent)
	observation.ActiveRequests = nonNegativeContextIntV0(observation.ActiveRequests)
	observation.CPUHighPercent = nonNegativeContextIntV0(observation.CPUHighPercent)
	if observation.CPUHighPercent == 0 {
		observation.CPUHighPercent = 75
	}
	observation.EvidenceRefs = compactContextStringsV0(observation.EvidenceRefs)
	return observation
}

func validateCodeContextToolLeaseObservationV0(observation CodeContextToolLeaseObservationV0) error {
	if observation.ObservedAt == "" ||
		observation.Lease.LeaseRef == "" ||
		observation.Lease.ToolRef == "" ||
		observation.Lease.LeaseUntil == "" {
		return errors.New(ErrCodeContextToolLeaseInvalidV0)
	}
	if _, err := parseCodeContextToolLeaseTimeV0(observation.ObservedAt); err != nil {
		return errors.New(ErrCodeContextToolLeaseInvalidV0)
	}
	if _, err := parseCodeContextToolLeaseTimeV0(observation.Lease.LeaseUntil); err != nil {
		return errors.New(ErrCodeContextToolLeaseInvalidV0)
	}
	return nil
}

func isCodeContextToolLeaseTerminalV0(status string) bool {
	switch strings.TrimSpace(status) {
	case CodeContextToolLeaseStatusCompletedV0, CodeContextToolLeaseStatusFailedV0, CodeContextToolLeaseStatusStoppedV0:
		return true
	default:
		return false
	}
}

func parseCodeContextToolLeaseTimeV0(value string) (time.Time, error) {
	return time.Parse(time.RFC3339, strings.TrimSpace(value))
}

func codeContextToolLeaseRefV0(request CodeContextToolLeaseRequestV0) string {
	raw := strings.Join([]string{
		request.RepositoryRef,
		request.WorktreeRef,
		request.CommitRef,
		request.QueryHash,
		request.ToolRef,
		request.StartedAt,
	}, "\x1f")
	sum := sha256.Sum256([]byte(raw))
	return "code-context-tool-lease-" + hex.EncodeToString(sum[:])[:24]
}

func nonNegativeContextIntV0(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

func (store *InMemoryCodeContextToolLeaseStoreV0) ensureLeasesLockedV0() {
	if store.leases == nil {
		store.leases = map[string]CodeContextToolLeaseV0{}
	}
}
