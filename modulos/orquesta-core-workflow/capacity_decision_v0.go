package orquestacoreworkflow

type RegisterCapacityDecisionCommandPayloadV0 struct {
	CapacityRequestID string                                `json:"capacity_request_id"`
	DecisionRef       string                                `json:"decision_ref"`
	Tier              OrchestrationCapacityRecommendationV0 `json:"tier"`
	ReasoningEffort   OrchestrationCapacityRecommendationV0 `json:"reasoning_effort"`
	Summary           string                                `json:"summary,omitempty"`
	EvidenceRefs      []string                              `json:"evidence_refs,omitempty"`
}

type CapacityDecidedPayloadV0 RegisterCapacityDecisionCommandPayloadV0

func NewRegisterCapacityDecisionCommandV0(meta OrchestrationCommandMetaV0, payload RegisterCapacityDecisionCommandPayloadV0) (OrchestrationCommandV0, error) {
	return newOrchestrationCommandV0(meta, OrchestrationCommandRegisterCapacityDecisionV0, normalizeRegisterCapacityDecisionPayloadV0(payload))
}

func NewCapacityDecidedEventV0(meta OrchestrationEventMetaV0, payload CapacityDecidedPayloadV0) (OrchestrationEventV0, error) {
	return newOrchestrationEventV0(meta, OrchestrationEventCapacityDecidedV0, normalizeCapacityDecidedPayloadV0(payload))
}

func capacityDecidedPayloadFromCommandV0(payload RegisterCapacityDecisionCommandPayloadV0) CapacityDecidedPayloadV0 {
	return CapacityDecidedPayloadV0{
		CapacityRequestID: payload.CapacityRequestID,
		DecisionRef:       payload.DecisionRef,
		Tier:              payload.Tier,
		ReasoningEffort:   payload.ReasoningEffort,
		Summary:           payload.Summary,
		EvidenceRefs:      cloneStringsV0(payload.EvidenceRefs),
	}
}
