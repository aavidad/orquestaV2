package orquestadirectoragent

const directorAgentRevisionPhaseV0 = "revision"

func (v *directorAgentDecisionValidatorV0) validateCloseTask(
	decision DirectorAgentDecisionV0,
) {
	if decision.CloseTask == nil {
		v.add("director_agent_payload_requerido", "close_task")
		return
	}
	payload := *decision.CloseTask
	v.requireRef("close_task.task_id", payload.TaskID)
	v.requireRef("close_task.phase_id", payload.PhaseID)
	v.requireRef("close_task.delivery_ref", payload.DeliveryRef)
	v.requireRef("close_task.accepted_review_ref", payload.AcceptedReviewRef)
	v.requireText("close_task.summary", payload.Summary)
	v.requireRequiredEvidence("close_task.evidence_refs", payload.EvidenceRefs)
	if payload.PhaseID != directorAgentRevisionPhaseV0 {
		v.add("director_agent_phase_invalida", "close_task.phase_id")
	}
	if payload.PhaseID != decision.PhaseID {
		v.add("director_agent_phase_mismatch", "close_task.phase_id")
	}
}

func (v *directorAgentDecisionValidatorV0) validateFinalValidation(
	decision DirectorAgentDecisionV0,
) {
	if decision.RegisterFinalValidation == nil {
		v.add("director_agent_payload_requerido", "register_final_validation")
		return
	}
	payload := *decision.RegisterFinalValidation
	v.requireRef("register_final_validation.validation_ref", payload.ValidationRef)
	v.requireRef("register_final_validation.phase_id", payload.PhaseID)
	v.requireRef("register_final_validation.closed_task_ref", payload.ClosedTaskRef)
	v.requireText("register_final_validation.summary", payload.Summary)
	v.requirePolicyText("register_final_validation.request_kind", payload.RequestKind)
	v.requirePolicyText("register_final_validation.execution_mode", payload.ExecutionMode)
	v.requirePolicyList("register_final_validation.minimum_deliverables", payload.MinimumDeliverables)
	v.requireEvidence("register_final_validation.evidence_refs", payload.EvidenceRefs)
	if payload.PhaseID != decision.PhaseID {
		v.add("director_agent_phase_mismatch", "register_final_validation.phase_id")
	}
}

func (v *directorAgentDecisionValidatorV0) validateCloseRun(
	decision DirectorAgentDecisionV0,
) {
	if decision.CloseRun == nil {
		v.add("director_agent_payload_requerido", "close_run")
		return
	}
	payload := *decision.CloseRun
	v.requireRef("close_run.closure_ref", payload.ClosureRef)
	v.requireRef("close_run.phase_id", payload.PhaseID)
	v.requireRef("close_run.validation_ref", payload.ValidationRef)
	v.requireText("close_run.summary", payload.Summary)
	v.requirePolicyText("close_run.request_kind", payload.RequestKind)
	v.requirePolicyText("close_run.execution_mode", payload.ExecutionMode)
	v.requirePolicyList("close_run.minimum_deliverables", payload.MinimumDeliverables)
	v.requireEvidence("close_run.evidence_refs", payload.EvidenceRefs)
	if payload.PhaseID != decision.PhaseID {
		v.add("director_agent_phase_mismatch", "close_run.phase_id")
	}
}

func (v *directorAgentDecisionValidatorV0) requirePolicyText(
	field string,
	value string,
) {
	if value == "" {
		return
	}
	v.requireText(field, value)
}

func (v *directorAgentDecisionValidatorV0) requirePolicyList(
	field string,
	values []string,
) {
	if len(values) == 0 {
		return
	}
	v.requireTextList(field, values)
}

func (v *directorAgentDecisionValidatorV0) requireRequiredEvidence(
	field string,
	values []string,
) {
	if len(values) == 0 {
		v.add("director_agent_evidence_invalida", field)
		return
	}
	v.requireEvidence(field, values)
}
