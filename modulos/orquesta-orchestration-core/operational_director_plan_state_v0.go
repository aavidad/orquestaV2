package orquestacionnucleoapp

import (
	"context"
	"strings"
	"sync"

	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
)

const OperationalDirectorPlanStateSchemaVersionV0 = "operational_director_plan_state.v0"

type OperationalDirectorPlanStateStatusV0 string

const (
	OperationalDirectorPlanStateActiveV0  OperationalDirectorPlanStateStatusV0 = "active"
	OperationalDirectorPlanStateBlockedV0 OperationalDirectorPlanStateStatusV0 = "blocked"
	OperationalDirectorPlanStateClosedV0  OperationalDirectorPlanStateStatusV0 = "closed"
)

type OperationalDirectorPlanStateV0 struct {
	SchemaVersion       string                                              `json:"schema_version"`
	StateRef            string                                              `json:"state_ref"`
	PlanRef             string                                              `json:"plan_ref"`
	RequestRef          string                                              `json:"request_ref,omitempty"`
	RunRef              string                                              `json:"run_ref"`
	ProjectRef          string                                              `json:"project_ref,omitempty"`
	Mode                orquestadirectoroperativo.OperationalDirectorModeV0 `json:"mode"`
	Status              OperationalDirectorPlanStateStatusV0                `json:"status"`
	ActiveStepID        string                                              `json:"active_step_id,omitempty"`
	ActiveWaveRef       string                                              `json:"active_wave_ref,omitempty"`
	ActiveCohortRef     string                                              `json:"active_cohort_ref,omitempty"`
	ActiveParentTaskRef string                                              `json:"active_parent_task_ref,omitempty"`
	Steps               []OperationalDirectorPlanStepStateV0                `json:"steps"`
	PendingAgentRefs    []string                                            `json:"pending_agent_refs,omitempty"`
	BlockerRefs         []string                                            `json:"blocker_refs,omitempty"`
	EvidenceRefs        []string                                            `json:"evidence_refs,omitempty"`
	RequiredTestRefs    []string                                            `json:"required_test_refs,omitempty"`
	ReplanAttempts      int                                                 `json:"replan_attempts,omitempty"`
	ClosureReason       string                                              `json:"closure_reason,omitempty"`
	CorrelationID       string                                              `json:"correlation_id,omitempty"`
	ObservedAt          string                                              `json:"observed_at"`
	UpdatedAt           string                                              `json:"updated_at,omitempty"`
}

type OperationalDirectorPlanStepStateV0 struct {
	StepID                   string                                                    `json:"step_id"`
	Kind                     orquestadirectoroperativo.OperationalDirectorStepKindV0   `json:"kind"`
	Status                   orquestadirectoroperativo.OperationalDirectorStepStatusV0 `json:"status"`
	WaveRef                  string                                                    `json:"wave_ref,omitempty"`
	CohortRef                string                                                    `json:"cohort_ref,omitempty"`
	ParentTaskRef            string                                                    `json:"parent_task_ref,omitempty"`
	TaskRefs                 []string                                                  `json:"task_refs,omitempty"`
	WaitRefs                 []string                                                  `json:"wait_refs,omitempty"`
	AgentRefs                []string                                                  `json:"agent_refs,omitempty"`
	PendingAgentRefs         []string                                                  `json:"pending_agent_refs,omitempty"`
	DeliveryRefs             []string                                                  `json:"delivery_refs,omitempty"`
	ReviewResultRefs         []string                                                  `json:"review_result_refs,omitempty"`
	AcceptedReviewRefs       []string                                                  `json:"accepted_review_refs,omitempty"`
	ReworkRequestRefs        []string                                                  `json:"rework_request_refs,omitempty"`
	ReplanDecisionRefs       []string                                                  `json:"replan_decision_refs,omitempty"`
	RequiredTestEvidenceRefs []string                                                  `json:"required_test_evidence_refs,omitempty"`
	EvidenceRefs             []string                                                  `json:"evidence_refs,omitempty"`
	BlockerRefs              []string                                                  `json:"blocker_refs,omitempty"`
	Attempts                 int                                                       `json:"attempts,omitempty"`
	Reason                   string                                                    `json:"reason,omitempty"`
}

type OperationalDirectorPlanStateWriterPortV0 interface {
	SaveOperationalDirectorPlanStateV0(ctx context.Context, state OperationalDirectorPlanStateV0) error
}

type OperationalDirectorPlanStateStorePortV0 interface {
	LoadOperationalDirectorPlanStateV0(ctx context.Context, runRef string, planRef string) (OperationalDirectorPlanStateV0, error)
}

type InMemoryOperationalDirectorPlanStateStoreV0 struct {
	mu     sync.Mutex
	states map[string]map[string]OperationalDirectorPlanStateV0
}

var _ OperationalDirectorPlanStateWriterPortV0 = (*InMemoryOperationalDirectorPlanStateStoreV0)(nil)
var _ OperationalDirectorPlanStateStorePortV0 = (*InMemoryOperationalDirectorPlanStateStoreV0)(nil)

func NewInMemoryOperationalDirectorPlanStateStoreV0(
	states ...OperationalDirectorPlanStateV0,
) *InMemoryOperationalDirectorPlanStateStoreV0 {
	store := &InMemoryOperationalDirectorPlanStateStoreV0{
		states: map[string]map[string]OperationalDirectorPlanStateV0{},
	}
	for _, state := range states {
		_ = store.saveOperationalDirectorPlanStateV0(state)
	}
	return store
}

func (store *InMemoryOperationalDirectorPlanStateStoreV0) SaveOperationalDirectorPlanStateV0(
	ctx context.Context,
	state OperationalDirectorPlanStateV0,
) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.saveOperationalDirectorPlanStateV0(state)
}

func (store *InMemoryOperationalDirectorPlanStateStoreV0) LoadOperationalDirectorPlanStateV0(
	ctx context.Context,
	runRef string,
	planRef string,
) (OperationalDirectorPlanStateV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return OperationalDirectorPlanStateV0{}, err
	}
	runRef = strings.TrimSpace(runRef)
	planRef = strings.TrimSpace(planRef)
	if runRef == "" {
		return OperationalDirectorPlanStateV0{}, errorV0(ErrNucleoOrquestacionInvalidoV0, "run_ref", "run_ref requerido")
	}
	if planRef == "" {
		return OperationalDirectorPlanStateV0{}, errorV0(ErrNucleoOrquestacionInvalidoV0, "plan_ref", "plan_ref requerido")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	state, ok := store.states[runRef][planRef]
	if !ok {
		return OperationalDirectorPlanStateV0{}, errorV0(ErrNucleoOrquestacionStoreV0, "operational_director_plan_state", "estado de plan no encontrado")
	}
	return state, nil
}

func (store *InMemoryOperationalDirectorPlanStateStoreV0) saveOperationalDirectorPlanStateV0(
	state OperationalDirectorPlanStateV0,
) error {
	normalized, err := NewOperationalDirectorPlanStateV0(state)
	if err != nil {
		return err
	}
	if store.states[normalized.RunRef] == nil {
		store.states[normalized.RunRef] = map[string]OperationalDirectorPlanStateV0{}
	}
	store.states[normalized.RunRef][normalized.PlanRef] = normalized
	return nil
}

func NewOperationalDirectorPlanStateV0(
	state OperationalDirectorPlanStateV0,
) (OperationalDirectorPlanStateV0, error) {
	normalized := NormalizeOperationalDirectorPlanStateV0(state)
	if err := ValidateOperationalDirectorPlanStateV0(normalized); err != nil {
		return OperationalDirectorPlanStateV0{}, err
	}
	return normalized, nil
}

func NormalizeOperationalDirectorPlanStateV0(
	state OperationalDirectorPlanStateV0,
) OperationalDirectorPlanStateV0 {
	status := OperationalDirectorPlanStateStatusV0(strings.TrimSpace(string(state.Status)))
	if status == "" {
		status = OperationalDirectorPlanStateActiveV0
	}
	steps := make([]OperationalDirectorPlanStepStateV0, 0, len(state.Steps))
	for _, step := range state.Steps {
		steps = append(steps, NormalizeOperationalDirectorPlanStepStateV0(step))
	}
	return OperationalDirectorPlanStateV0{
		SchemaVersion:       strings.TrimSpace(state.SchemaVersion),
		StateRef:            strings.TrimSpace(state.StateRef),
		PlanRef:             strings.TrimSpace(state.PlanRef),
		RequestRef:          strings.TrimSpace(state.RequestRef),
		RunRef:              strings.TrimSpace(state.RunRef),
		ProjectRef:          strings.TrimSpace(state.ProjectRef),
		Mode:                orquestadirectoroperativo.OperationalDirectorModeV0(strings.TrimSpace(string(state.Mode))),
		Status:              status,
		ActiveStepID:        strings.TrimSpace(state.ActiveStepID),
		ActiveWaveRef:       strings.TrimSpace(state.ActiveWaveRef),
		ActiveCohortRef:     strings.TrimSpace(state.ActiveCohortRef),
		ActiveParentTaskRef: strings.TrimSpace(state.ActiveParentTaskRef),
		Steps:               steps,
		PendingAgentRefs:    compactStringsV0(state.PendingAgentRefs),
		BlockerRefs:         compactStringsV0(state.BlockerRefs),
		EvidenceRefs:        compactStringsV0(state.EvidenceRefs),
		RequiredTestRefs:    compactStringsV0(state.RequiredTestRefs),
		ReplanAttempts:      state.ReplanAttempts,
		ClosureReason:       strings.TrimSpace(state.ClosureReason),
		CorrelationID:       strings.TrimSpace(state.CorrelationID),
		ObservedAt:          strings.TrimSpace(state.ObservedAt),
		UpdatedAt:           strings.TrimSpace(state.UpdatedAt),
	}
}

func NormalizeOperationalDirectorPlanStepStateV0(
	step OperationalDirectorPlanStepStateV0,
) OperationalDirectorPlanStepStateV0 {
	status := orquestadirectoroperativo.OperationalDirectorStepStatusV0(strings.TrimSpace(string(step.Status)))
	if status == "" {
		status = orquestadirectoroperativo.OperationalDirectorStepPendingV0
	}
	return OperationalDirectorPlanStepStateV0{
		StepID:                   strings.TrimSpace(step.StepID),
		Kind:                     orquestadirectoroperativo.OperationalDirectorStepKindV0(strings.TrimSpace(string(step.Kind))),
		Status:                   status,
		WaveRef:                  strings.TrimSpace(step.WaveRef),
		CohortRef:                strings.TrimSpace(step.CohortRef),
		ParentTaskRef:            strings.TrimSpace(step.ParentTaskRef),
		TaskRefs:                 compactStringsV0(step.TaskRefs),
		WaitRefs:                 compactStringsV0(step.WaitRefs),
		AgentRefs:                compactStringsV0(step.AgentRefs),
		PendingAgentRefs:         compactStringsV0(step.PendingAgentRefs),
		DeliveryRefs:             compactStringsV0(step.DeliveryRefs),
		ReviewResultRefs:         compactStringsV0(step.ReviewResultRefs),
		AcceptedReviewRefs:       compactStringsV0(step.AcceptedReviewRefs),
		ReworkRequestRefs:        compactStringsV0(step.ReworkRequestRefs),
		ReplanDecisionRefs:       compactStringsV0(step.ReplanDecisionRefs),
		RequiredTestEvidenceRefs: compactStringsV0(step.RequiredTestEvidenceRefs),
		EvidenceRefs:             compactStringsV0(step.EvidenceRefs),
		BlockerRefs:              compactStringsV0(step.BlockerRefs),
		Attempts:                 step.Attempts,
		Reason:                   strings.TrimSpace(step.Reason),
	}
}

func ValidateOperationalDirectorPlanStateV0(state OperationalDirectorPlanStateV0) error {
	if state.SchemaVersion != OperationalDirectorPlanStateSchemaVersionV0 {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "schema_version", "schema_version invalida")
	}
	for field, value := range map[string]string{
		"state_ref":   state.StateRef,
		"plan_ref":    state.PlanRef,
		"run_ref":     state.RunRef,
		"observed_at": state.ObservedAt,
	} {
		if strings.TrimSpace(value) == "" {
			return errorV0(ErrNucleoOrquestacionInvalidoV0, field, field+" requerido")
		}
	}
	if state.Mode != orquestadirectoroperativo.OperationalDirectorModeProgrammingV0 &&
		state.Mode != orquestadirectoroperativo.OperationalDirectorModeDomainWorkV0 {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "mode", "mode invalido")
	}
	if state.Status != OperationalDirectorPlanStateActiveV0 &&
		state.Status != OperationalDirectorPlanStateBlockedV0 &&
		state.Status != OperationalDirectorPlanStateClosedV0 {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "status", "status invalido")
	}
	if state.ReplanAttempts < 0 {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "replan_attempts", "replan_attempts invalido")
	}
	if len(state.Steps) == 0 {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "steps", "steps requerido")
	}
	steps := map[string]bool{}
	acceptedReviewRefs := map[string]bool{}
	var activeStep OperationalDirectorPlanStepStateV0
	activeStepFound := false
	for _, step := range state.Steps {
		if err := ValidateOperationalDirectorPlanStepStateV0(step); err != nil {
			return err
		}
		if steps[step.StepID] {
			return errorV0(ErrNucleoOrquestacionInvalidoV0, "steps.step_id", "step_id duplicado")
		}
		steps[step.StepID] = true
		if state.ActiveStepID != "" && step.StepID == state.ActiveStepID {
			activeStep = step
			activeStepFound = true
		}
		for _, acceptedReviewRef := range step.AcceptedReviewRefs {
			if acceptedReviewRefs[acceptedReviewRef] {
				return errorV0(ErrNucleoOrquestacionInvalidoV0, "steps.accepted_review_refs", "accepted_review_ref duplicado")
			}
			acceptedReviewRefs[acceptedReviewRef] = true
		}
	}
	if state.ActiveStepID != "" && !activeStepFound {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "active_step_id", "active_step_id no existe")
	}
	if activeStepFound {
		if err := validateOperationalDirectorPlanStateActiveStepV0(state, activeStep); err != nil {
			return err
		}
	}
	return nil
}

func validateOperationalDirectorPlanStateActiveStepV0(
	state OperationalDirectorPlanStateV0,
	step OperationalDirectorPlanStepStateV0,
) error {
	if state.ActiveWaveRef != "" && step.WaveRef != "" && state.ActiveWaveRef != step.WaveRef {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "active_wave_ref", "active_wave_ref inconsistente")
	}
	if state.ActiveCohortRef != "" && step.CohortRef != "" && state.ActiveCohortRef != step.CohortRef {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "active_cohort_ref", "active_cohort_ref inconsistente")
	}
	if state.ActiveParentTaskRef != "" && step.ParentTaskRef != "" && state.ActiveParentTaskRef != step.ParentTaskRef {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "active_parent_task_ref", "active_parent_task_ref inconsistente")
	}
	if step.Kind == orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0 &&
		step.Status == orquestadirectoroperativo.OperationalDirectorStepRunningV0 &&
		len(step.WaitRefs) != 1 {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "steps.wait_refs", "wait_subagents activo requiere wait_ref unico")
	}
	return nil
}

func ValidateOperationalDirectorPlanStepStateV0(step OperationalDirectorPlanStepStateV0) error {
	if strings.TrimSpace(step.StepID) == "" {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "steps.step_id", "step_id requerido")
	}
	if !operationalDirectorPlanStepKindValidV0(step.Kind) {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "steps.kind", "kind invalido")
	}
	if !operationalDirectorPlanStepStatusValidV0(step.Status) {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "steps.status", "status invalido")
	}
	if step.Attempts < 0 {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "steps.attempts", "attempts invalido")
	}
	if len(step.AcceptedReviewRefs) > 0 &&
		step.Kind != orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0 {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "steps.accepted_review_refs", "accepted_review_refs solo aplica a review_deliveries")
	}
	if step.Kind == orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0 &&
		(step.Status == orquestadirectoroperativo.OperationalDirectorStepAcceptedV0 ||
			step.Status == orquestadirectoroperativo.OperationalDirectorStepClosedV0) &&
		(len(step.DeliveryRefs) == 0 || len(step.ReviewResultRefs) == 0 || len(step.AcceptedReviewRefs) == 0) {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "steps.accepted_review_refs", "review_deliveries aceptado requiere refs causales")
	}
	return nil
}

func operationalDirectorPlanStepKindValidV0(kind orquestadirectoroperativo.OperationalDirectorStepKindV0) bool {
	switch kind {
	case orquestadirectoroperativo.OperationalDirectorStepGatherContextV0,
		orquestadirectoroperativo.OperationalDirectorStepRequestDomainContextV0,
		orquestadirectoroperativo.OperationalDirectorStepSplitWorkV0,
		orquestadirectoroperativo.OperationalDirectorStepLaunchSubagentsV0,
		orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0,
		orquestadirectoroperativo.OperationalDirectorStepGovernDelegationV0,
		orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0,
		orquestadirectoroperativo.OperationalDirectorStepRunRequiredTestsV0,
		orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0:
		return true
	default:
		return false
	}
}

func operationalDirectorPlanStepStatusValidV0(status orquestadirectoroperativo.OperationalDirectorStepStatusV0) bool {
	switch status {
	case orquestadirectoroperativo.OperationalDirectorStepPendingV0,
		orquestadirectoroperativo.OperationalDirectorStepRunningV0,
		orquestadirectoroperativo.OperationalDirectorStepBlockedV0,
		orquestadirectoroperativo.OperationalDirectorStepChangesRequestedV0,
		orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
		orquestadirectoroperativo.OperationalDirectorStepClosedV0:
		return true
	default:
		return false
	}
}
