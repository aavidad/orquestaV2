package orquestadirectoragent

func (v *directorAgentDecisionValidatorV0) validateAskQuestion(
	decision DirectorAgentDecisionV0,
	payload *DirectorAgentAskQuestionCommandV0,
	field string,
	target string,
) {
	if payload == nil {
		v.add("director_agent_payload_requerido", field)
		return
	}
	v.requireRef(field+".question_id", payload.QuestionID)
	v.requireRef(field+".source_group", payload.SourceGroup)
	v.requireTargetGroup(field+".target_group", payload.TargetGroup, target)
	v.requireText(field+".summary", payload.Summary)
	v.requireOptionalTextList(field+".options", payload.Options)
	v.requireEvidence(field+".evidence_refs", payload.EvidenceRefs)
	_ = decision
}

func (v *directorAgentDecisionValidatorV0) validateRequestCapacity(
	decision DirectorAgentDecisionV0,
) {
	if decision.RequestCapacity == nil {
		v.add("director_agent_payload_requerido", "request_capacity")
		return
	}
	payload := *decision.RequestCapacity
	v.requireRef("request_capacity.capacity_request_id", payload.CapacityRequestID)
	v.requireRef("request_capacity.phase_id", payload.PhaseID)
	if payload.TaskRef != "" {
		v.requireRef("request_capacity.task_ref", payload.TaskRef)
	}
	v.requireRef("request_capacity.reason_code", payload.ReasonCode)
	v.requireText("request_capacity.summary", payload.Summary)
	v.requireCapacity("request_capacity.minimum_recommended_capacity", payload.MinimumRecommendedCapacity)
	v.requireEvidence("request_capacity.evidence_refs", payload.EvidenceRefs)
	if payload.PhaseID != decision.PhaseID {
		v.add("director_agent_phase_mismatch", "request_capacity.phase_id")
	}
}

func (v *directorAgentDecisionValidatorV0) validateRequestAgent(
	decision DirectorAgentDecisionV0,
) {
	if decision.RequestAgent == nil {
		v.add("director_agent_payload_requerido", "request_agent")
		return
	}
	payload := *decision.RequestAgent
	v.requireRef("request_agent.agent_request_id", payload.AgentRequestID)
	v.requireRef("request_agent.phase_id", payload.PhaseID)
	if payload.TaskRef != "" {
		v.requireRef("request_agent.task_ref", payload.TaskRef)
	}
	v.requireRef("request_agent.capacity_request_ref", payload.CapacityRequestRef)
	v.requireText("request_agent.role", payload.Role)
	v.requireText("request_agent.summary", payload.Summary)
	v.requireEvidence("request_agent.evidence_refs", payload.EvidenceRefs)
	if payload.PhaseID != decision.PhaseID {
		v.add("director_agent_phase_mismatch", "request_agent.phase_id")
	}
}

func (v *directorAgentDecisionValidatorV0) validateRequestRework(
	decision DirectorAgentDecisionV0,
) {
	if decision.RequestRework == nil {
		v.add("director_agent_payload_requerido", "request_rework")
		return
	}
	payload := *decision.RequestRework
	v.requireRef("request_rework.rework_request_ref", payload.ReworkRequestRef)
	v.requireRef("request_rework.phase_id", payload.PhaseID)
	v.requireRef("request_rework.review_result_ref", payload.ReviewResultRef)
	v.requireRef("request_rework.review_request_id", payload.ReviewRequestID)
	v.requireRef("request_rework.delivery_ref", payload.DeliveryRef)
	v.requireText("request_rework.summary", payload.Summary)
	v.requireEvidence("request_rework.evidence_refs", payload.EvidenceRefs)
	if payload.PhaseID != directorAgentRevisionPhaseV0 {
		v.add("director_agent_phase_invalida", "request_rework.phase_id")
	}
	if payload.PhaseID != decision.PhaseID {
		v.add("director_agent_phase_mismatch", "request_rework.phase_id")
	}
}

func (v *directorAgentDecisionValidatorV0) validateReplanDecision(
	decision DirectorAgentDecisionV0,
) {
	if decision.RecordReplanDecision == nil {
		v.add("director_agent_payload_requerido", "record_replan_decision")
		return
	}
	payload := *decision.RecordReplanDecision
	v.requireRef("record_replan_decision.replan_ref", payload.ReplanRef)
	v.requireRef("record_replan_decision.run_ref", payload.RunRef)
	v.requireRef("record_replan_decision.task_ref", payload.TaskRef)
	v.requireRef("record_replan_decision.source_ref", payload.SourceRef)
	v.requireReplanAction("record_replan_decision.accepted_action", payload.AcceptedAction)
	v.requireTextList("record_replan_decision.followup_refs", payload.FollowupRefs)
	v.requireText("record_replan_decision.summary", payload.Summary)
	v.requireEvidence("record_replan_decision.evidence_refs", payload.EvidenceRefs)
	if payload.RunRef != decision.RunID {
		v.add("director_agent_run_mismatch", "record_replan_decision.run_ref")
	}
}

func (v *directorAgentDecisionValidatorV0) requireOptionalTextList(field string, values []string) {
	if len(values) == 0 {
		return
	}
	v.requireTextList(field, values)
}

func (v *directorAgentDecisionValidatorV0) requireTargetGroup(field string, value string, want string) {
	if value == "" && want == DirectorAgentTargetDirectorV0 {
		return
	}
	if value != want {
		v.add("director_agent_target_invalido", field)
	}
}

func (v *directorAgentDecisionValidatorV0) requireReviewStatus(field string, value string) {
	switch value {
	case DirectorAgentReviewStatusAcceptedV0,
		DirectorAgentReviewStatusChangesRequestedV0,
		DirectorAgentReviewStatusRejectedV0:
		return
	default:
		v.add("director_agent_review_status_invalido", field)
	}
}

func (v *directorAgentDecisionValidatorV0) requireReplanAction(field string, value string) {
	switch value {
	case DirectorAgentReplanActionSplitTaskV0,
		DirectorAgentReplanActionRetryTaskV0,
		DirectorAgentReplanActionReplaceAgentV0,
		DirectorAgentReplanActionEscalateCapacityV0,
		DirectorAgentReplanActionAskDirectorV0,
		DirectorAgentReplanActionAbortTaskV0:
		return
	default:
		v.add("director_agent_replan_action_invalida", field)
	}
}
