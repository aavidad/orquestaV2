package orquestadirectoragent

func ValidateDirectorAgentDecisionV0(decision DirectorAgentDecisionV0) []DirectorAgentDecisionIssueV0 {
	decision = normalizeDirectorAgentDecisionV0(decision)
	v := directorAgentDecisionValidatorV0{}
	v.require("schema_version", decision.SchemaVersion, DirectorAgentDecisionSchemaVersionV0)
	v.requireRef("decision_ref", decision.DecisionRef)
	v.requireRef("run_id", decision.RunID)
	v.requireRef("phase_id", decision.PhaseID)
	v.requireRef("command_ref", decision.CommandRef)
	v.requireText("summary", decision.Summary)
	v.requireEvidence("evidence_refs", decision.EvidenceRefs)
	v.validateCommand(decision)
	return v.issues
}

func DirectorAgentDecisionValidV0(decision DirectorAgentDecisionV0) bool {
	return len(ValidateDirectorAgentDecisionV0(decision)) == 0
}

type directorAgentDecisionValidatorV0 struct {
	issues []DirectorAgentDecisionIssueV0
}

func (v *directorAgentDecisionValidatorV0) validateCommand(decision DirectorAgentDecisionV0) {
	switch decision.CommandType {
	case DirectorAgentCommandRequestBrainstormV0:
		v.validateBrainstorm(decision)
	case DirectorAgentCommandOpenPhaseV0:
		v.validateOpenPhase(decision)
	case DirectorAgentCommandRequestVoteV0:
		v.validateVote(decision)
	case DirectorAgentCommandAcceptDecisionV0:
		v.validateAcceptDecision(decision)
	case DirectorAgentCommandPublishContractV0:
		v.validatePublishContract(decision)
	case DirectorAgentCommandCreateMicrotaskV0:
		v.validateCreateMicrotask(decision)
	case DirectorAgentCommandAskDirectorV0:
		v.validateAskQuestion(decision, decision.AskDirector, "ask_director", DirectorAgentTargetDirectorV0)
	case DirectorAgentCommandAskUserV0:
		v.validateAskQuestion(decision, decision.AskUser, "ask_user", DirectorAgentTargetUserV0)
	case DirectorAgentCommandRequestCapacityV0:
		v.validateRequestCapacity(decision)
	case DirectorAgentCommandRequestAgentV0:
		v.validateRequestAgent(decision)
	case DirectorAgentCommandRequestReviewV0:
		v.validateRequestReview(decision)
	case DirectorAgentCommandRecordReviewResultV0:
		v.validateReviewResult(decision)
	case DirectorAgentCommandAcceptReviewV0:
		v.validateAcceptReview(decision)
	case DirectorAgentCommandRequestReworkV0:
		v.validateRequestRework(decision)
	case DirectorAgentCommandRecordReplanDecisionV0:
		v.validateReplanDecision(decision)
	case DirectorAgentCommandCloseTaskV0:
		v.validateCloseTask(decision)
	case DirectorAgentCommandRegisterFinalValidationV0:
		v.validateFinalValidation(decision)
	case DirectorAgentCommandCloseRunV0:
		v.validateCloseRun(decision)
	case DirectorAgentCommandProposePlanTeamV0:
		v.validatePlanTeam(decision)
	case DirectorAgentCommandAnswerQuestionV0:
		v.validateAnswerQuestion(decision)
	default:
		v.add("director_agent_command_no_soportado", "command_type")
	}
}

func (v *directorAgentDecisionValidatorV0) validateBrainstorm(decision DirectorAgentDecisionV0) {
	if decision.RequestBrainstorm == nil {
		v.add("director_agent_payload_requerido", "request_brainstorm")
		return
	}
	payload := *decision.RequestBrainstorm
	v.requireRef("request_brainstorm.brainstorm_request_id", payload.BrainstormRequestID)
	v.requireRef("request_brainstorm.phase_id", payload.PhaseID)
	v.requireRef("request_brainstorm.topic_ref", payload.TopicRef)
	v.requireText("request_brainstorm.summary", payload.Summary)
	v.requireCapacity("request_brainstorm.minimum_recommended_capacity", payload.MinimumRecommendedCapacity)
	v.requireEvidence("request_brainstorm.evidence_refs", payload.EvidenceRefs)
	if payload.PhaseID != decision.PhaseID {
		v.add("director_agent_phase_mismatch", "request_brainstorm.phase_id")
	}
}

func (v *directorAgentDecisionValidatorV0) validateOpenPhase(decision DirectorAgentDecisionV0) {
	if decision.OpenPhase == nil {
		v.add("director_agent_payload_requerido", "open_phase")
		return
	}
	payload := *decision.OpenPhase
	v.requireRef("open_phase.phase_id", payload.PhaseID)
	v.requireText("open_phase.reason", payload.Reason)
}

func (v *directorAgentDecisionValidatorV0) validateVote(decision DirectorAgentDecisionV0) {
	if decision.RequestVote == nil {
		v.add("director_agent_payload_requerido", "request_vote")
		return
	}
	payload := *decision.RequestVote
	v.requireRef("request_vote.vote_request_id", payload.VoteRequestID)
	v.requireRef("request_vote.phase_id", payload.PhaseID)
	v.requireRef("request_vote.decision_topic_ref", payload.DecisionTopicRef)
	v.requireRef("request_vote.brainstorm_ref", payload.BrainstormRef)
	v.requireText("request_vote.summary", payload.Summary)
	v.requireCapacity("request_vote.minimum_recommended_capacity", payload.MinimumRecommendedCapacity)
	v.requireEvidence("request_vote.evidence_refs", payload.EvidenceRefs)
	if payload.PhaseID != decision.PhaseID {
		v.add("director_agent_phase_mismatch", "request_vote.phase_id")
	}
}

func (v *directorAgentDecisionValidatorV0) validateAcceptDecision(decision DirectorAgentDecisionV0) {
	if decision.AcceptDecision == nil {
		v.add("director_agent_payload_requerido", "accept_decision")
		return
	}
	payload := *decision.AcceptDecision
	v.requireRef("accept_decision.decision_ref", payload.DecisionRef)
	v.requireRef("accept_decision.phase_id", payload.PhaseID)
	v.requireRef("accept_decision.vote_ref", payload.VoteRef)
	v.requireRef("accept_decision.accepted_option_ref", payload.AcceptedOptionRef)
	v.requireText("accept_decision.summary", payload.Summary)
	v.requireEvidence("accept_decision.evidence_refs", payload.EvidenceRefs)
	if payload.PhaseID != decision.PhaseID {
		v.add("director_agent_phase_mismatch", "accept_decision.phase_id")
	}
}

func (v *directorAgentDecisionValidatorV0) validatePublishContract(decision DirectorAgentDecisionV0) {
	if decision.PublishContract == nil {
		v.add("director_agent_payload_requerido", "publish_function_contract")
		return
	}
	payload := *decision.PublishContract
	v.requireRef("publish_function_contract.contract_ref", payload.ContractRef)
	v.requireRef("publish_function_contract.phase_id", payload.PhaseID)
	v.requireRef("publish_function_contract.decision_ref", payload.DecisionRef)
	v.requireText("publish_function_contract.summary", payload.Summary)
	v.requireTextList("publish_function_contract.function_names", payload.FunctionNames)
	v.requireEvidence("publish_function_contract.evidence_refs", payload.EvidenceRefs)
	if payload.PhaseID != decision.PhaseID {
		v.add("director_agent_phase_mismatch", "publish_function_contract.phase_id")
	}
}

func (v *directorAgentDecisionValidatorV0) validateRequestReview(decision DirectorAgentDecisionV0) {
	if decision.RequestReview == nil {
		v.add("director_agent_payload_requerido", "request_review")
		return
	}
	payload := *decision.RequestReview
	v.requireRef("request_review.review_request_id", payload.ReviewRequestID)
	v.requireRef("request_review.phase_id", payload.PhaseID)
	v.requireRef("request_review.delivery_ref", payload.DeliveryRef)
	v.requireText("request_review.summary", payload.Summary)
	v.requireEvidence("request_review.evidence_refs", payload.EvidenceRefs)
	if payload.PhaseID != decision.PhaseID {
		v.add("director_agent_phase_mismatch", "request_review.phase_id")
	}
}

func (v *directorAgentDecisionValidatorV0) validateReviewResult(decision DirectorAgentDecisionV0) {
	if decision.RecordReviewResult == nil {
		v.add("director_agent_payload_requerido", "record_review_result")
		return
	}
	payload := *decision.RecordReviewResult
	v.requireRef("record_review_result.review_result_ref", payload.ReviewResultRef)
	v.requireRef("record_review_result.review_request_id", payload.ReviewRequestID)
	v.requireRef("record_review_result.delivery_ref", payload.DeliveryRef)
	v.requireReviewStatus("record_review_result.status", payload.Status)
	v.requireText("record_review_result.summary", payload.Summary)
	v.requireEvidence("record_review_result.evidence_refs", payload.EvidenceRefs)
	if payload.QualityGateRef != "" {
		v.requireRef("record_review_result.quality_gate_ref", payload.QualityGateRef)
	}
}

func (v *directorAgentDecisionValidatorV0) validateAcceptReview(decision DirectorAgentDecisionV0) {
	if decision.AcceptReview == nil {
		v.add("director_agent_payload_requerido", "accept_review")
		return
	}
	payload := *decision.AcceptReview
	v.requireRef("accept_review.accepted_review_ref", payload.AcceptedReviewRef)
	v.requireRef("accept_review.phase_id", payload.PhaseID)
	v.requireRef("accept_review.review_request_id", payload.ReviewRequestID)
	v.requireRef("accept_review.delivery_ref", payload.DeliveryRef)
	v.requireText("accept_review.summary", payload.Summary)
	v.requireEvidence("accept_review.evidence_refs", payload.EvidenceRefs)
	if payload.PhaseID != decision.PhaseID {
		v.add("director_agent_phase_mismatch", "accept_review.phase_id")
	}
}

func (v *directorAgentDecisionValidatorV0) validateAnswerQuestion(decision DirectorAgentDecisionV0) {
	if decision.AnswerQuestion == nil {
		v.add("director_agent_payload_requerido", "answer_director_question")
		return
	}
	payload := *decision.AnswerQuestion
	v.requireRef("answer_director_question.answer_id", payload.AnswerID)
	v.requireRef("answer_director_question.question_id", payload.QuestionID)
	v.requireDirectorAnswer("answer_director_question.decision", payload.Decision)
	v.requireText("answer_director_question.summary", payload.Summary)
	v.requireEvidence("answer_director_question.evidence_refs", payload.EvidenceRefs)
}

func (v *directorAgentDecisionValidatorV0) require(field string, got string, want string) {
	if got != want {
		v.add("director_agent_valor_invalido", field)
	}
}

func (v *directorAgentDecisionValidatorV0) requireRef(field string, value string) {
	if !directorAgentRefCompactV0(value) || directorAgentHasForbiddenDetailV0(value) {
		v.add("director_agent_ref_invalida", field)
	}
}

func (v *directorAgentDecisionValidatorV0) requireOptionalRef(field string, value string) {
	if value == "" {
		return
	}
	v.requireRef(field, value)
}

func (v *directorAgentDecisionValidatorV0) requireText(field string, value string) {
	if value == "" || len(value) > maxDirectorAgentStringV0 || directorAgentHasForbiddenDetailV0(value) {
		v.add("director_agent_texto_invalido", field)
	}
}

func (v *directorAgentDecisionValidatorV0) requireEvidence(field string, values []string) {
	if len(values) > maxDirectorAgentEvidenceRefsV0 {
		v.add("director_agent_evidence_invalida", field)
		return
	}
	for _, value := range values {
		v.requireRef(field, value)
	}
}

func (v *directorAgentDecisionValidatorV0) requireOptionalRefs(field string, values []string) {
	if len(values) > maxDirectorAgentEvidenceRefsV0 {
		v.add("director_agent_evidence_invalida", field)
		return
	}
	for _, value := range values {
		v.requireRef(field, value)
	}
}

func (v *directorAgentDecisionValidatorV0) requireOptionalWorkRefs(field string, values []string) {
	if len(values) > maxDirectorAgentRecursionLimitV0 {
		v.add("director_agent_lista_invalida", field)
		return
	}
	for _, value := range values {
		v.requireRef(field, value)
	}
}

func (v *directorAgentDecisionValidatorV0) requireOptionalContextRefs(field string, values []string) {
	if len(values) > maxDirectorAgentEvidenceRefsV0 {
		v.add("director_agent_evidence_invalida", field)
		return
	}
	for _, value := range values {
		if !directorAgentContextRefCompactV0(value) || directorAgentHasForbiddenDetailV0(value) {
			v.add("director_agent_ref_invalida", field)
		}
	}
}

func (v *directorAgentDecisionValidatorV0) requireTextList(field string, values []string) {
	if len(values) == 0 || len(values) > maxDirectorAgentEvidenceRefsV0 {
		v.add("director_agent_lista_invalida", field)
		return
	}
	for _, value := range values {
		v.requireText(field, value)
	}
}

func (v *directorAgentDecisionValidatorV0) validateFunctionRefs(
	field string,
	refs []DirectorAgentFunctionContractRefV0,
) {
	if len(refs) == 0 || len(refs) > maxDirectorAgentEvidenceRefsV0 {
		v.add("director_agent_lista_invalida", field)
		return
	}
	for _, ref := range refs {
		if ref.ContractRef == "" {
			v.add("director_agent_ref_invalida", field+".contract_ref")
			continue
		}
		v.requireRef(field+".contract_ref", ref.ContractRef)
		if ref.FunctionName != "" {
			v.requireText(field+".function_name", ref.FunctionName)
		}
	}
}

func (v *directorAgentDecisionValidatorV0) requireCapacity(field string, value string) {
	if value == "" {
		return
	}
	switch value {
	case DirectorAgentCapacityLowV0, DirectorAgentCapacityMediumV0,
		DirectorAgentCapacityHighV0, DirectorAgentCapacityXHighV0:
		return
	default:
		v.add("director_agent_capacidad_invalida", field)
	}
}

func (v *directorAgentDecisionValidatorV0) requireDirectorAnswer(field string, value string) {
	switch value {
	case DirectorAgentAnswerContinueV0, DirectorAgentAnswerReplanV0, DirectorAgentAnswerStopAgentV0:
		return
	default:
		v.add("director_agent_answer_invalida", field)
	}
}

func (v *directorAgentDecisionValidatorV0) add(code string, field string) {
	v.issues = append(v.issues, DirectorAgentDecisionIssueV0{Code: code, Field: field})
}
