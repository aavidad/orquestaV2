package orquestadirector

import orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"

const (
	ErrDirectorReplanFollowupsInvalidoV0 = "director_replan_followups_invalido"

	ReplanFollowupStatusCapacityAndAgentRequestedV0 = "capacity_and_agent_requested"
	ReplanFollowupStatusCapacityRequestedV0         = "capacity_requested"
	ReplanFollowupStatusMicrotasksRequestedV0       = "microtasks_requested"
	ReplanFollowupStatusAskDirectorV0               = "ask_director"
	ReplanFollowupStatusNeedsDirectorUnsupportedV0  = "needs_director/unsupported"

	ReplanFollowupSourceQualityGateBlockedV0 = "quality_gate_blocked"
	ReplanFollowupSourceReviewReworkV0       = "review_rework"
)

type ReplanFollowupsInputV0 struct {
	DecisionCommandMeta  orquestacoreworkflow.OrchestrationCommandMetaV0           `json:"decision_command_meta"`
	DecisionPayload      orquestacoreworkflow.RecordReplanDecisionCommandPayloadV0 `json:"decision_payload"`
	SourceKind           string                                                    `json:"source_kind,omitempty"`
	OpenPhaseCandidate   *ReplanOpenPhaseCandidateV0                               `json:"open_phase_candidate,omitempty"`
	MicrotaskCandidates  []ReplanMicrotaskCandidateV0                              `json:"microtask_candidates,omitempty"`
	CapacityCandidate    *ReplanCapacityCandidateV0                                `json:"capacity_candidate,omitempty"`
	AgentCandidate       *ReplanAgentCandidateV0                                   `json:"agent_candidate,omitempty"`
	AskDirectorCandidate *ReplanAskDirectorCandidateV0                             `json:"ask_director_candidate,omitempty"`
	BlockedAgentRefs     []string                                                  `json:"blocked_agent_refs,omitempty"`
}

type ReplanOpenPhaseCandidateV0 struct {
	CommandMeta orquestacoreworkflow.OrchestrationCommandMetaV0 `json:"command_meta"`
	Payload     orquestacoreworkflow.OpenPhaseCommandPayloadV0  `json:"payload"`
}

type ReplanMicrotaskCandidateV0 struct {
	CommandMeta orquestacoreworkflow.OrchestrationCommandMetaV0      `json:"command_meta"`
	Payload     orquestacoreworkflow.CreateMicrotaskCommandPayloadV0 `json:"payload"`
}

type ReplanCapacityCandidateV0 struct {
	CommandMeta orquestacoreworkflow.OrchestrationCommandMetaV0      `json:"command_meta"`
	Payload     orquestacoreworkflow.RequestCapacityCommandPayloadV0 `json:"payload"`
}

type ReplanAgentCandidateV0 struct {
	CommandMeta orquestacoreworkflow.OrchestrationCommandMetaV0   `json:"command_meta"`
	Payload     orquestacoreworkflow.RequestAgentCommandPayloadV0 `json:"payload"`
}

type ReplanAskDirectorCandidateV0 struct {
	CommandMeta orquestacoreworkflow.OrchestrationCommandMetaV0  `json:"command_meta"`
	Payload     orquestacoreworkflow.AskDirectorCommandPayloadV0 `json:"payload"`
}

type ReplanFollowupsResultV0 struct {
	RecordReplanDecisionCommand orquestacoreworkflow.OrchestrationCommandV0   `json:"record_replan_decision_command"`
	OpenPhaseCommand            *orquestacoreworkflow.OrchestrationCommandV0  `json:"open_phase_command,omitempty"`
	CreateMicrotaskCommands     []orquestacoreworkflow.OrchestrationCommandV0 `json:"create_microtask_commands,omitempty"`
	RequestCapacityCommand      *orquestacoreworkflow.OrchestrationCommandV0  `json:"request_capacity_command,omitempty"`
	RequestAgentCommand         *orquestacoreworkflow.OrchestrationCommandV0  `json:"request_agent_command,omitempty"`
	AskDirectorCommand          *orquestacoreworkflow.OrchestrationCommandV0  `json:"ask_director_command,omitempty"`
	FollowupStatus              string                                        `json:"followup_status"`
}

type ReplanFollowupsErrorV0 struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	Field         string `json:"field,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

func (err ReplanFollowupsErrorV0) Error() string {
	return err.Code
}
