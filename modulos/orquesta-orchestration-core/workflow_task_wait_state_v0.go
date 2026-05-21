package orquestacionnucleoapp

import (
	"context"
	"strings"
	"sync"
)

const WorkflowTaskWaitStateSchemaVersionV0 = "workflow_task_wait_state.v0"

type WorkflowTaskWaitStateStatusV0 string

const (
	WorkflowTaskWaitStateStatusWaitingV0   WorkflowTaskWaitStateStatusV0 = "waiting"
	WorkflowTaskWaitStateStatusContinuedV0 WorkflowTaskWaitStateStatusV0 = "continued"
	WorkflowTaskWaitStateStatusExpiredV0   WorkflowTaskWaitStateStatusV0 = "expired"
	WorkflowTaskWaitStateStatusClearedV0   WorkflowTaskWaitStateStatusV0 = "cleared"
)

const (
	WorkflowTaskWaitReasonCohortInProgressV0 = "cohort_in_progress"
	WorkflowTaskWaitReasonReviewPendingV0    = "review_pending"
	WorkflowTaskWaitReasonTimeoutPendingV0   = "timeout_pending"
	WorkflowTaskWaitReasonBudgetExhaustedV0  = "budget_exhausted"
	WorkflowTaskWaitReasonExternalBlockedV0  = "external_blocked"
	WorkflowTaskWaitReasonContextMissingV0   = "context_missing"
)

type WorkflowTaskWaitStateV0 struct {
	SchemaVersion    string                        `json:"schema_version"`
	WaitRef          string                        `json:"wait_ref"`
	RunRef           string                        `json:"run_ref"`
	ReasonCode       string                        `json:"reason_code"`
	CohortRef        string                        `json:"cohort_ref,omitempty"`
	WaveRef          string                        `json:"wave_ref,omitempty"`
	ParentTaskRef    string                        `json:"parent_task_ref,omitempty"`
	TaskRefs         []string                      `json:"task_refs,omitempty"`
	AgentRefs        []string                      `json:"agent_refs,omitempty"`
	PendingAgentRefs []string                      `json:"pending_agent_refs,omitempty"`
	Status           WorkflowTaskWaitStateStatusV0 `json:"status"`
	Attempt          int                           `json:"attempt,omitempty"`
	MaxExternalWaits int                           `json:"max_external_waits,omitempty"`
	CorrelationID    string                        `json:"correlation_id,omitempty"`
	EvidenceRefs     []string                      `json:"evidence_refs,omitempty"`
	ObservedAt       string                        `json:"observed_at"`
	DeadlineAt       string                        `json:"deadline_at,omitempty"`
}

type WorkflowTaskWaitStateWriterPortV0 interface {
	SaveWorkflowTaskWaitStateV0(ctx context.Context, state WorkflowTaskWaitStateV0) error
}

type WorkflowTaskWaitStateStorePortV0 interface {
	LoadWorkflowTaskWaitStateV0(ctx context.Context, runRef string, waitRef string) (WorkflowTaskWaitStateV0, error)
}

type InMemoryWorkflowTaskWaitStateStoreV0 struct {
	mu     sync.Mutex
	states map[string]map[string]WorkflowTaskWaitStateV0
}

var _ WorkflowTaskWaitStateWriterPortV0 = (*InMemoryWorkflowTaskWaitStateStoreV0)(nil)
var _ WorkflowTaskWaitStateStorePortV0 = (*InMemoryWorkflowTaskWaitStateStoreV0)(nil)

func NewInMemoryWorkflowTaskWaitStateStoreV0(
	states ...WorkflowTaskWaitStateV0,
) *InMemoryWorkflowTaskWaitStateStoreV0 {
	store := &InMemoryWorkflowTaskWaitStateStoreV0{
		states: map[string]map[string]WorkflowTaskWaitStateV0{},
	}
	for _, state := range states {
		_ = store.saveWorkflowTaskWaitStateV0(state)
	}
	return store
}

func (store *InMemoryWorkflowTaskWaitStateStoreV0) SaveWorkflowTaskWaitStateV0(
	ctx context.Context,
	state WorkflowTaskWaitStateV0,
) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.saveWorkflowTaskWaitStateV0(state)
}

func (store *InMemoryWorkflowTaskWaitStateStoreV0) LoadWorkflowTaskWaitStateV0(
	ctx context.Context,
	runRef string,
	waitRef string,
) (WorkflowTaskWaitStateV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return WorkflowTaskWaitStateV0{}, err
	}
	runRef = strings.TrimSpace(runRef)
	waitRef = strings.TrimSpace(waitRef)
	if runRef == "" {
		return WorkflowTaskWaitStateV0{}, errorV0(ErrNucleoOrquestacionInvalidoV0, "run_ref", "run_ref requerido")
	}
	if waitRef == "" {
		return WorkflowTaskWaitStateV0{}, errorV0(ErrNucleoOrquestacionInvalidoV0, "wait_ref", "wait_ref requerido")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	state, ok := store.states[runRef][waitRef]
	if !ok {
		return WorkflowTaskWaitStateV0{}, errorV0(ErrNucleoOrquestacionStoreV0, "workflow_task_wait_state", "espera no encontrada")
	}
	return state, nil
}

func (store *InMemoryWorkflowTaskWaitStateStoreV0) saveWorkflowTaskWaitStateV0(
	state WorkflowTaskWaitStateV0,
) error {
	normalized, err := NewWorkflowTaskWaitStateV0(state)
	if err != nil {
		return err
	}
	if store.states[normalized.RunRef] == nil {
		store.states[normalized.RunRef] = map[string]WorkflowTaskWaitStateV0{}
	}
	store.states[normalized.RunRef][normalized.WaitRef] = normalized
	return nil
}

func NewWorkflowTaskWaitStateV0(state WorkflowTaskWaitStateV0) (WorkflowTaskWaitStateV0, error) {
	normalized := NormalizeWorkflowTaskWaitStateV0(state)
	if err := ValidateWorkflowTaskWaitStateV0(normalized); err != nil {
		return WorkflowTaskWaitStateV0{}, err
	}
	return normalized, nil
}

func NormalizeWorkflowTaskWaitStateV0(state WorkflowTaskWaitStateV0) WorkflowTaskWaitStateV0 {
	status := WorkflowTaskWaitStateStatusV0(strings.TrimSpace(string(state.Status)))
	if status == "" {
		status = WorkflowTaskWaitStateStatusWaitingV0
	}
	return WorkflowTaskWaitStateV0{
		SchemaVersion:    strings.TrimSpace(state.SchemaVersion),
		WaitRef:          strings.TrimSpace(state.WaitRef),
		RunRef:           strings.TrimSpace(state.RunRef),
		ReasonCode:       strings.TrimSpace(state.ReasonCode),
		CohortRef:        strings.TrimSpace(state.CohortRef),
		WaveRef:          strings.TrimSpace(state.WaveRef),
		ParentTaskRef:    strings.TrimSpace(state.ParentTaskRef),
		TaskRefs:         compactStringsV0(state.TaskRefs),
		AgentRefs:        compactStringsV0(state.AgentRefs),
		PendingAgentRefs: compactStringsV0(state.PendingAgentRefs),
		Status:           status,
		Attempt:          state.Attempt,
		MaxExternalWaits: state.MaxExternalWaits,
		CorrelationID:    strings.TrimSpace(state.CorrelationID),
		EvidenceRefs:     compactStringsV0(state.EvidenceRefs),
		ObservedAt:       strings.TrimSpace(state.ObservedAt),
		DeadlineAt:       strings.TrimSpace(state.DeadlineAt),
	}
}

func ValidateWorkflowTaskWaitStateV0(state WorkflowTaskWaitStateV0) error {
	if state.SchemaVersion != WorkflowTaskWaitStateSchemaVersionV0 {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "schema_version", "schema_version invalida")
	}
	for field, value := range map[string]string{
		"wait_ref":    state.WaitRef,
		"run_ref":     state.RunRef,
		"reason_code": state.ReasonCode,
		"observed_at": state.ObservedAt,
	} {
		if strings.TrimSpace(value) == "" {
			return errorV0(ErrNucleoOrquestacionInvalidoV0, field, field+" requerido")
		}
	}
	if state.Status != WorkflowTaskWaitStateStatusWaitingV0 &&
		state.Status != WorkflowTaskWaitStateStatusContinuedV0 &&
		state.Status != WorkflowTaskWaitStateStatusExpiredV0 &&
		state.Status != WorkflowTaskWaitStateStatusClearedV0 {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "status", "status invalido")
	}
	if !workflowTaskWaitReasonCodeValidV0(state.ReasonCode) {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "reason_code", "reason_code invalido")
	}
	if state.Attempt < 0 || state.MaxExternalWaits < 0 {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "attempt", "limite de espera invalido")
	}
	if strings.TrimSpace(state.CohortRef) == "" &&
		strings.TrimSpace(state.WaveRef) == "" &&
		strings.TrimSpace(state.ParentTaskRef) == "" &&
		len(state.AgentRefs) == 0 {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "wait_scope", "scope de espera requerido")
	}
	return nil
}

func workflowTaskWaitReasonCodeValidV0(reason string) bool {
	switch strings.TrimSpace(reason) {
	case WorkflowTaskWaitReasonCohortInProgressV0,
		WorkflowTaskWaitReasonReviewPendingV0,
		WorkflowTaskWaitReasonTimeoutPendingV0,
		WorkflowTaskWaitReasonBudgetExhaustedV0,
		WorkflowTaskWaitReasonExternalBlockedV0,
		WorkflowTaskWaitReasonContextMissingV0:
		return true
	default:
		return false
	}
}
