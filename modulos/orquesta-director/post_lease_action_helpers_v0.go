package orquestadirector

import orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"

func registerLeaseExpiredPayloadFromPostLeaseV0(
	input PostLeaseActionInputV0,
) orquestacoreworkflow.RegisterAgentLeaseExpiredCommandPayloadV0 {
	return orquestacoreworkflow.RegisterAgentLeaseExpiredCommandPayloadV0{
		RunRef:            input.RunRef,
		AgentRequestID:    input.AgentRequestID,
		LeaseRef:          input.LeaseRef,
		ReasonCode:        input.ReasonCode,
		ObservedAt:        input.ObservedAt,
		RecommendedAction: input.RecommendedAction,
		EvidenceRefs:      input.EvidenceRefs,
	}
}

func stopAgentPayloadFromPostLeaseV0(input PostLeaseActionInputV0) orquestacoreworkflow.StopAgentCommandPayloadV0 {
	return orquestacoreworkflow.StopAgentCommandPayloadV0{
		AgentRequestID: input.AgentRequestID,
		ReasonCode:     input.ReasonCode,
		Summary:        "Lease expirado; detener agente logico.",
		EvidenceRefs:   input.EvidenceRefs,
	}
}

func askDirectorPayloadFromPostLeaseV0(input PostLeaseActionInputV0) orquestacoreworkflow.AskDirectorCommandPayloadV0 {
	return orquestacoreworkflow.AskDirectorCommandPayloadV0{
		QuestionID:   input.QuestionID,
		SourceGroup:  postLeaseActionSourceGroupV0,
		TargetGroup:  orquestacoreworkflow.DirectorQuestionTargetDirectorV0,
		Summary:      "Lease expirado; decidir accion posterior.",
		Options:      []string{"retry", "stop_agent", "mark_failed", "mark_stopped", "replan_task"},
		EvidenceRefs: input.EvidenceRefs,
		Blocking:     true,
	}
}

func postLeaseFollowupMetaV0(
	meta orquestacoreworkflow.OrchestrationCommandMetaV0,
	suffix string,
) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	meta.CommandID += suffix
	meta.IdempotencyKey += suffix
	return meta
}
