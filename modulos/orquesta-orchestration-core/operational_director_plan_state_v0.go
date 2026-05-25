package orquestacionnucleoapp

import (
	"context"
	"strings"

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
