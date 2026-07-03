package orquestaserver

import (
	"context"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

const (
	IdleSelfImprovementGoalBackendGoneWithoutResultReasonV0   = "goal_backend_gone_without_result"
	IdleSelfImprovementGoalBackendGoneWithoutResultEvidenceV0 = "evidence-ref-goal-backend-gone-without-result"
)

type IdleSelfImprovementMaterializedGoalResultRequestV0 struct {
	GoalState       orquestagoal.GoalWorkStateV0 `json:"goal_state,omitempty"`
	GoalRef         string                       `json:"goal_ref,omitempty"`
	ExternalGoalRef string                       `json:"external_goal_ref,omitempty"`
}

type IdleSelfImprovementMaterializedGoalResultV0 struct {
	Found        bool                          `json:"found"`
	Result       orquestagoal.GoalWorkResultV0 `json:"result,omitempty"`
	EvidenceRefs []string                      `json:"evidence_refs,omitempty"`
}

type IdleSelfImprovementMaterializedGoalResultPortV0 interface {
	LoadIdleSelfImprovementMaterializedGoalResultV0(
		context.Context,
		IdleSelfImprovementMaterializedGoalResultRequestV0,
	) (IdleSelfImprovementMaterializedGoalResultV0, error)
}
