package orquestadirectorcandidates

import (
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadecisioncouncil "orquesta/modulos/orquesta-decision-council"
)

type CouncilPlanCandidatesInputV0 struct {
	RunRef                     string                                                     `json:"run_ref"`
	PhaseID                    string                                                     `json:"phase_id"`
	Role                       string                                                     `json:"role"`
	OccurredAt                 string                                                     `json:"occurred_at"`
	CorrelationID              string                                                     `json:"correlation_id,omitempty"`
	RequestedBy                string                                                     `json:"requested_by,omitempty"`
	MinimumRecommendedCapacity orquestacoreworkflow.OrchestrationCapacityRecommendationV0 `json:"minimum_recommended_capacity,omitempty"`
	Plan                       orquestadecisioncouncil.DecisionCouncilPlanV0              `json:"plan"`
	EvidenceRefs               []string                                                   `json:"evidence_refs,omitempty"`
}
