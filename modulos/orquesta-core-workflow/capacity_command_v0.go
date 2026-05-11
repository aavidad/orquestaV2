package orquestacoreworkflow

type RequestCapacityCommandPayloadV0 struct {
	CapacityRequestID          string                                `json:"capacity_request_id"`
	PhaseID                    string                                `json:"phase_id"`
	TaskRef                    string                                `json:"task_ref,omitempty"`
	ReasonCode                 string                                `json:"reason_code"`
	Summary                    string                                `json:"summary"`
	MinimumRecommendedCapacity OrchestrationCapacityRecommendationV0 `json:"minimum_recommended_capacity,omitempty"`
	EvidenceRefs               []string                              `json:"evidence_refs,omitempty"`
}

type CapacityRequestedPayloadV0 struct {
	CapacityRequestID          string                                `json:"capacity_request_id"`
	PhaseID                    string                                `json:"phase_id"`
	TaskRef                    string                                `json:"task_ref,omitempty"`
	ReasonCode                 string                                `json:"reason_code"`
	Summary                    string                                `json:"summary"`
	MinimumRecommendedCapacity OrchestrationCapacityRecommendationV0 `json:"minimum_recommended_capacity,omitempty"`
	EvidenceRefs               []string                              `json:"evidence_refs,omitempty"`
}

func NewRequestCapacityCommandV0(meta OrchestrationCommandMetaV0, payload RequestCapacityCommandPayloadV0) (OrchestrationCommandV0, error) {
	return newOrchestrationCommandV0(meta, OrchestrationCommandRequestCapacityV0, payload)
}

func NewCapacityRequestedEventV0(meta OrchestrationEventMetaV0, payload CapacityRequestedPayloadV0) (OrchestrationEventV0, error) {
	return newOrchestrationEventV0(meta, OrchestrationEventCapacityRequestedV0, payload)
}

func capacityRequestedPayloadFromCommandV0(payload RequestCapacityCommandPayloadV0) CapacityRequestedPayloadV0 {
	return CapacityRequestedPayloadV0{
		CapacityRequestID:          payload.CapacityRequestID,
		PhaseID:                    payload.PhaseID,
		TaskRef:                    payload.TaskRef,
		ReasonCode:                 payload.ReasonCode,
		Summary:                    payload.Summary,
		MinimumRecommendedCapacity: payload.MinimumRecommendedCapacity,
		EvidenceRefs:               cloneStringsV0(payload.EvidenceRefs),
	}
}
