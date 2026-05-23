package orquestacoreworkflow

import "strings"

func IsSupportedOrchestrationRunStatusV0(status OrchestrationRunStatusV0) bool {
	switch normalizeRunStatusV0(status) {
	case OrchestrationRunStatusPendingV0, OrchestrationRunStatusActiveV0,
		OrchestrationRunStatusBlockedV0, OrchestrationRunStatusClosedV0:
		return true
	default:
		return false
	}
}

func ValidateOrchestrationRunStatusV0(status OrchestrationRunStatusV0) error {
	if !IsSupportedOrchestrationRunStatusV0(status) {
		return OrchestrationValidationIssueV0{Code: OrchestrationEstadoInconsistenteV0, Field: "status"}
	}
	return nil
}

func IsSupportedOrchestrationPhaseStatusV0(status OrchestrationPhaseStatusV0) bool {
	switch normalizePhaseStatusV0(status) {
	case OrchestrationPhaseStatusPendingV0, OrchestrationPhaseStatusActiveV0,
		OrchestrationPhaseStatusClosedV0, OrchestrationPhaseStatusBlockedV0:
		return true
	default:
		return false
	}
}

func ValidateOrchestrationPhaseStatusV0(status OrchestrationPhaseStatusV0) error {
	if !IsSupportedOrchestrationPhaseStatusV0(status) {
		return OrchestrationValidationIssueV0{Code: OrchestrationEstadoInconsistenteV0, Field: "phase.status"}
	}
	return nil
}

func ValidateOrchestrationPhaseV0(phase OrchestrationPhaseV0) []OrchestrationValidationIssueV0 {
	var issues []OrchestrationValidationIssueV0
	if err := ValidateOrchestrationPhaseIDV0(phase.ID); err != nil {
		issues = append(issues, err.(OrchestrationValidationIssueV0))
	}
	if err := ValidateOrchestrationPhaseStatusV0(phase.Status); err != nil {
		issues = append(issues, err.(OrchestrationValidationIssueV0))
	}
	if !isSupportedCapacityRecommendationV0(phase.RecommendedCapacity) {
		issues = append(issues, issueV0(OrchestrationFaseInvalidaV0, "recommended_capacity"))
	}
	return issues
}

func ValidateOrchestrationRunV0(run OrchestrationRunV0) []OrchestrationValidationIssueV0 {
	if orchestrationRunIsEmptyV0(run) {
		return nil
	}

	var issues []OrchestrationValidationIssueV0
	issues = append(issues, validateRunRequiredFieldsV0(run)...)
	issues = append(issues, validateRunPhasesV0(run)...)
	issues = append(issues, validateRunRefsV0(run)...)
	return issues
}

func validateRunRequiredFieldsV0(run OrchestrationRunV0) []OrchestrationValidationIssueV0 {
	var issues []OrchestrationValidationIssueV0
	if strings.TrimSpace(run.SchemaVersion) != OrchestrationRunSchemaVersionV0 {
		issues = append(issues, issueV0(OrchestrationRunInvalidoV0, "schema_version"))
	}
	if strings.TrimSpace(run.RunID) == "" {
		issues = append(issues, issueV0(OrchestrationRunInvalidoV0, "run_id"))
	}
	if strings.TrimSpace(run.ProjectRef) == "" {
		issues = append(issues, issueV0(OrchestrationRunInvalidoV0, "project_ref"))
	}
	if strings.TrimSpace(run.AppSpecRef) == "" {
		issues = append(issues, issueV0(OrchestrationRunInvalidoV0, "app_spec_ref"))
	}
	if run.LastSequence < 0 {
		issues = append(issues, issueV0(OrchestrationEstadoInconsistenteV0, "last_sequence"))
	}
	if err := ValidateOrchestrationRunStatusV0(run.Status); err != nil {
		issues = append(issues, err.(OrchestrationValidationIssueV0))
	}
	return issues
}

func validateRunPhasesV0(run OrchestrationRunV0) []OrchestrationValidationIssueV0 {
	var issues []OrchestrationValidationIssueV0
	activeCount := 0
	currentFound := false
	for _, phase := range run.Phases {
		issues = append(issues, ValidateOrchestrationPhaseV0(phase)...)
		if phase.Status == OrchestrationPhaseStatusActiveV0 {
			activeCount++
		}
		if normalizePhaseIDV0(phase.ID) == normalizePhaseIDV0(run.CurrentPhase) {
			currentFound = true
		}
	}
	if activeCount > 1 {
		issues = append(issues, issueV0(OrchestrationEstadoInconsistenteV0, "phases"))
	}
	if strings.TrimSpace(string(run.CurrentPhase)) == "" || !currentFound {
		issues = append(issues, issueV0(OrchestrationFaseInvalidaV0, "current_phase"))
	}
	return issues
}

func validateRunRefsV0(run OrchestrationRunV0) []OrchestrationValidationIssueV0 {
	refs := map[string][]string{
		"project_ref":                 []string{run.ProjectRef},
		"app_spec_ref":                []string{run.AppSpecRef},
		"brainstorms":                 run.Brainstorms,
		"votes":                       run.Votes,
		"tasks":                       run.Tasks,
		"function_contracts":          run.FunctionContracts,
		"decisions":                   run.Decisions,
		"capacity_requests":           run.CapacityRequests,
		"capacity_decisions":          run.CapacityDecisions,
		"agents":                      run.Agents,
		"agent_phase_refs":            run.AgentPhaseRefs,
		"started_agents":              run.StartedAgents,
		"failed_agents":               run.FailedAgents,
		"lost_agents":                 run.LostAgents,
		"stopped_agents":              run.StoppedAgents,
		"agent_stop_requests":         run.AgentStopRequests,
		"confirmed_stopped_agents":    run.ConfirmedStoppedAgents,
		"agent_assessments":           run.AgentAssessments,
		"agent_lease_expirations":     run.AgentLeaseExpirations,
		"quality_gates":               run.QualityGates,
		"phase_artifacts":             run.PhaseArtifacts,
		"deliveries":                  run.Deliveries,
		"delivered_tasks":             run.DeliveredTasks,
		"delivered_agents":            run.DeliveredAgents,
		"reviews":                     run.Reviews,
		"review_results":              run.ReviewResults,
		"rework_requests":             run.ReworkRequests,
		"replan_decisions":            run.ReplanDecisions,
		"accepted_reviews":            run.AcceptedReviews,
		"closed_tasks":                run.ClosedTasks,
		"validations":                 run.Validations,
		"closures":                    run.Closures,
		"director_questions":          run.DirectorQuestions,
		"director_answers":            run.DirectorAnswers,
		"director_answered_questions": run.DirectorAnsweredQuestions,
		"blockers":                    run.Blockers,
	}
	for field, values := range refs {
		if refsContainForbiddenDetailsV0(values) {
			return []OrchestrationValidationIssueV0{issueV0(OrchestrationDetalleProhibidoV0, field)}
		}
	}
	if stoppedAgentRefsInvalidV0(run) {
		return []OrchestrationValidationIssueV0{issueV0(OrchestrationEstadoInconsistenteV0, "stopped_agents")}
	}
	if lostAgentRefsInvalidV0(run) {
		return []OrchestrationValidationIssueV0{issueV0(OrchestrationEstadoInconsistenteV0, "lost_agents")}
	}
	if agentPhaseRefsInvalidV0(run) {
		return []OrchestrationValidationIssueV0{issueV0(OrchestrationEstadoInconsistenteV0, "agent_phase_refs")}
	}
	if agentStopRequestRefsInvalidV0(run) {
		return []OrchestrationValidationIssueV0{issueV0(OrchestrationEstadoInconsistenteV0, "agent_stop_requests")}
	}
	if confirmedStoppedAgentRefsInvalidV0(run) {
		return []OrchestrationValidationIssueV0{issueV0(OrchestrationEstadoInconsistenteV0, "confirmed_stopped_agents")}
	}
	if capacityDecisionRefsInvalidV0(run) {
		return []OrchestrationValidationIssueV0{issueV0(OrchestrationEstadoInconsistenteV0, "capacity_decisions")}
	}
	if directorAnsweredQuestionRefsInvalidV0(run) {
		return []OrchestrationValidationIssueV0{issueV0(OrchestrationEstadoInconsistenteV0, "director_answered_questions")}
	}
	if reviewResultRefsInvalidV0(run) {
		return []OrchestrationValidationIssueV0{issueV0(OrchestrationEstadoInconsistenteV0, "review_results")}
	}
	if reworkRequestRefsInvalidV0(run) {
		return []OrchestrationValidationIssueV0{issueV0(OrchestrationEstadoInconsistenteV0, "rework_requests")}
	}
	if replanDecisionRefsInvalidV0(run) {
		return []OrchestrationValidationIssueV0{issueV0(OrchestrationEstadoInconsistenteV0, "replan_decisions")}
	}
	if agentLeaseExpirationRefsInvalidV0(run) {
		return []OrchestrationValidationIssueV0{issueV0(OrchestrationEstadoInconsistenteV0, "agent_lease_expirations")}
	}
	if concurrencyGateRefsInvalidV0(run) {
		return []OrchestrationValidationIssueV0{issueV0(OrchestrationEstadoInconsistenteV0, "concurrency_gates")}
	}
	if qualityGateRefsInvalidV0(run) {
		return []OrchestrationValidationIssueV0{issueV0(OrchestrationEstadoInconsistenteV0, "quality_gates")}
	}
	if phaseArtifactRefsInvalidV0(run) {
		return []OrchestrationValidationIssueV0{issueV0(OrchestrationEstadoInconsistenteV0, "phase_artifacts")}
	}
	if commandEffectRefsInvalidV0(run) {
		return []OrchestrationValidationIssueV0{issueV0(OrchestrationEstadoInconsistenteV0, "command_effects")}
	}
	return nil
}

func capacityDecisionRefsInvalidV0(run OrchestrationRunV0) bool {
	seen := map[string]bool{}
	for _, decision := range run.CapacityDecisions {
		requestID := capacityDecisionProjectionRequestIDV0(decision)
		if strings.TrimSpace(requestID) == "" || !capacityRequestAlreadyReflectedV0(run, requestID) {
			return true
		}
		if seen[requestID] {
			return true
		}
		seen[requestID] = true
	}
	return false
}

func directorAnsweredQuestionRefsInvalidV0(run OrchestrationRunV0) bool {
	for _, questionID := range run.DirectorAnsweredQuestions {
		if !directorQuestionRefAlreadyReflectedV0(run, questionID) {
			return true
		}
	}
	return false
}

func reviewResultRefsInvalidV0(run OrchestrationRunV0) bool {
	seen := map[string]bool{}
	for _, projection := range run.ReviewResults {
		parts, ok := reviewResultProjectionPartsFromRefV0(projection)
		if !ok || !IsSupportedReviewResultStatusV0(parts.Status) {
			return true
		}
		if !reviewRequestAlreadyReflectedV0(run, parts.ReviewRequestID) ||
			!deliveryAlreadyReflectedV0(run, parts.DeliveryRef) ||
			seen[parts.ReviewResultRef] {
			return true
		}
		seen[parts.ReviewResultRef] = true
	}
	return false
}

func stoppedAgentRefsInvalidV0(run OrchestrationRunV0) bool {
	for _, stopped := range run.StoppedAgents {
		if !agentRequestAlreadyReflectedV0(run, stopped) {
			return true
		}
	}
	return false
}

func lostAgentRefsInvalidV0(run OrchestrationRunV0) bool {
	for _, lost := range run.LostAgents {
		if !agentRequestAlreadyReflectedV0(run, lost) ||
			!agentStartedAlreadyReflectedV0(run, lost) ||
			agentFailedAlreadyReflectedV0(run, lost) ||
			agentStopConfirmedAlreadyReflectedV0(run, lost) ||
			agentDeliveredAlreadyReflectedV0(run, lost) {
			return true
		}
	}
	return false
}

func agentStopRequestRefsInvalidV0(run OrchestrationRunV0) bool {
	seen := map[string]bool{}
	for _, projectionRef := range run.AgentStopRequests {
		projection, ok := ParseAgentStopRequestProjectionV0(projectionRef)
		if !ok {
			return true
		}
		if !agentStopAlreadyReflectedV0(run, projection.AgentRequestID) {
			return true
		}
		if seen[projection.AgentRequestID] {
			return true
		}
		seen[projection.AgentRequestID] = true
	}
	return false
}

func confirmedStoppedAgentRefsInvalidV0(run OrchestrationRunV0) bool {
	for _, stopped := range run.ConfirmedStoppedAgents {
		if !agentStopAlreadyReflectedV0(run, stopped) {
			return true
		}
	}
	return false
}

func refsContainForbiddenDetailsV0(values []string) bool {
	for _, value := range values {
		if stringHasForbiddenRunDetailV0(value) {
			return true
		}
	}
	return false
}

func stringHasForbiddenRunDetailV0(value string) bool {
	lower := strings.ToLower(value)
	for _, fragment := range forbiddenEventFragmentsV0 {
		if containsForbiddenFragmentV0(lower, fragment) {
			return true
		}
	}
	return false
}
