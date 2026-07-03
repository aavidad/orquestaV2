package orquestamcp

import (
	"errors"
	"strings"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

const (
	MCPObserveAppDirectorGoalToolNameV0    = "orquesta.apps.observe_director_goal.v0"
	MCPObserveAppDirectorGoalToolVersionV0 = "v0"
	MCPObserveAppDirectorGoalResourceURIV0 = "orquesta://contracts/observe-app-director-goal/v0"
	MCPObserveAppDirectorGoalEstadoOKV0    = "ok"
	MCPObserveAppDirectorGoalEstadoErrorV0 = "error"
)

type MCPObserveAppDirectorGoalToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	ResourceURI string   `json:"resource_uri"`
	Invariantes []string `json:"invariantes"`
}

type MCPObserveAppDirectorGoalToolInputV0 struct {
	RequestID     string `json:"request_id,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
	RunRef        string `json:"run_ref"`
	OccurredAt    string `json:"occurred_at,omitempty"`
	RequestedBy   string `json:"requested_by,omitempty"`
}

type MCPObserveAppDirectorGoalToolResultV0 struct {
	Estado                string                 `json:"estado"`
	RequestID             string                 `json:"request_id,omitempty"`
	CorrelationID         string                 `json:"correlation_id,omitempty"`
	Partial               bool                   `json:"partial,omitempty"`
	RunRef                string                 `json:"run_ref,omitempty"`
	RunStatus             string                 `json:"run_status,omitempty"`
	DirectorExecutionMode string                 `json:"director_execution_mode,omitempty"`
	GoalRef               string                 `json:"goal_ref,omitempty"`
	ExternalGoalRef       string                 `json:"external_goal_ref,omitempty"`
	GoalStatus            string                 `json:"goal_status,omitempty"`
	ResultRef             string                 `json:"result_ref,omitempty"`
	LastEventAt           string                 `json:"last_event_at,omitempty"`
	CurrentPhase          string                 `json:"current_phase,omitempty"`
	RetryFromPhase        string                 `json:"retry_from_phase,omitempty"`
	DomainCounters        map[string]int         `json:"domain_counters,omitempty"`
	ProcessRefs           []string               `json:"process_refs,omitempty"`
	RecommendedAction     string                 `json:"recommended_action,omitempty"`
	ClosureStatus         string                 `json:"closure_status,omitempty"`
	ClosureAccepted       bool                   `json:"closure_accepted,omitempty"`
	ClosureNeedsRework    bool                   `json:"closure_needs_rework,omitempty"`
	Summary               string                 `json:"summary,omitempty"`
	ArtifactRefs          []string               `json:"artifact_refs,omitempty"`
	DomainReceiptRefs     []string               `json:"domain_receipt_refs,omitempty"`
	ExpectedReceiptRefs   []string               `json:"expected_terminal_receipt_refs,omitempty"`
	EvidenceRefs          []string               `json:"evidence_refs,omitempty"`
	ClosureIssues         []MCPValidationIssueV0 `json:"closure_issues,omitempty"`
	Errores               []MCPValidationIssueV0 `json:"errores_publicos,omitempty"`
}

type mcpObserveAppDirectorGoalPublicIssueV0 struct {
	Code    string
	Field   string
	Message string
}

func MCPObserveAppDirectorGoalDescriptorV0() MCPObserveAppDirectorGoalToolDescriptorV0 {
	return MCPObserveAppDirectorGoalToolDescriptorV0{
		Name:        MCPObserveAppDirectorGoalToolNameV0,
		Version:     MCPObserveAppDirectorGoalToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,run_ref,occurred_at?,requested_by?}",
		Output:      "ok:{run_ref,run_status?,director_execution_mode?,goal_ref,goal_status,recommended_action?,closure_status?,closure_accepted?,artifact_refs?,evidence_refs?}|error:{errores_publicos,recommended_action?,evidence_refs?}",
		ResourceURI: MCPObserveAppDirectorGoalResourceURIV0,
		Invariantes: []string{
			"adaptador inbound fino",
			"observa un goal ya lanzado por run_ref",
			"no ejecuta loop legacy ni arranca proveedor",
			"la validacion de cierre vive en orquesta-app-director-service",
		},
	}
}

func ToObserveAppDirectorGoalRequestV0(
	input MCPObserveAppDirectorGoalToolInputV0,
) orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0 {
	return orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0{
		RunRef:        strings.TrimSpace(input.RunRef),
		OccurredAt:    strings.TrimSpace(input.OccurredAt),
		CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		RequestedBy:   firstNonEmptyMCPV0(input.RequestedBy, "orquesta-mcp-observe-director-goal"),
	}
}

func NewMCPObserveAppDirectorGoalResultV0(
	input MCPObserveAppDirectorGoalToolInputV0,
	result orquestaappdirectorservice.ObserveAppDirectorGoalResultV0,
) MCPObserveAppDirectorGoalToolResultV0 {
	goalResult := result.GoalResult
	closureStatus := strings.TrimSpace(result.Closure.Status)
	toolResult := MCPObserveAppDirectorGoalToolResultV0{
		Estado:                MCPObserveAppDirectorGoalEstadoOKV0,
		RequestID:             strings.TrimSpace(input.RequestID),
		CorrelationID:         firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		RunRef:                firstNonEmptyMCPV0(result.RunRef, input.RunRef),
		RunStatus:             strings.TrimSpace(string(result.Run.Status)),
		DirectorExecutionMode: strings.TrimSpace(result.DirectorExecutionMode),
		GoalRef:               strings.TrimSpace(result.GoalRef),
		ExternalGoalRef:       strings.TrimSpace(result.ExternalGoalRef),
		GoalStatus:            strings.TrimSpace(result.Status),
		ResultRef:             mcpObserveAppDirectorGoalResultRefV0(goalResult),
		CurrentPhase:          strings.TrimSpace(string(result.Run.CurrentPhase)),
		ClosureStatus:         closureStatus,
		ClosureAccepted:       result.Closure.Accepted,
		ClosureNeedsRework:    result.Closure.NeedsRework,
		Summary:               strings.TrimSpace(goalResult.Summary),
		ArtifactRefs:          compactStringsMCPV0(goalResult.ArtifactRefs),
		DomainReceiptRefs:     compactStringsMCPV0(goalResult.DomainReceiptRefs),
		EvidenceRefs:          compactStringsMCPV0(result.EvidenceRefs),
		ClosureIssues:         goalWorkIssuesMCPV0(append(goalResult.Issues, result.Closure.Issues...)),
		Errores:               []MCPValidationIssueV0{},
	}
	toolResult = mcpObserveAppDirectorGoalSuppressStaleClosureForRunningGoalV0(toolResult)
	toolResult.RecommendedAction = mcpObserveAppDirectorGoalRecommendedActionV0(toolResult)
	return toolResult
}

func NewMCPObserveAppDirectorGoalPartialResultFromStateV0(
	input MCPObserveAppDirectorGoalToolInputV0,
	state orquestagoal.GoalWorkStateV0,
) (MCPObserveAppDirectorGoalToolResultV0, error) {
	normalized, err := orquestagoal.NewGoalWorkStateV0(state)
	if err != nil {
		return MCPObserveAppDirectorGoalToolResultV0{}, err
	}
	toolResult := MCPObserveAppDirectorGoalToolResultV0{
		Estado:                MCPObserveAppDirectorGoalEstadoOKV0,
		RequestID:             strings.TrimSpace(input.RequestID),
		CorrelationID:         firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		Partial:               true,
		RunRef:                firstNonEmptyMCPV0(normalized.RunRef, input.RunRef),
		DirectorExecutionMode: orquestaappdirectorservice.AppDirectorExecutionModeGoalFirstV0,
		GoalRef:               strings.TrimSpace(normalized.GoalRef),
		ExternalGoalRef:       strings.TrimSpace(normalized.ExternalGoalRef),
		GoalStatus:            strings.TrimSpace(normalized.Status),
		EvidenceRefs: compactStringsMCPV0(append(
			normalized.EvidenceRefs,
			normalized.LaunchReceipt.EvidenceRefs...,
		)),
		ClosureIssues: goalWorkIssuesMCPV0(normalized.LaunchReceipt.Issues),
		Errores:       []MCPValidationIssueV0{},
	}
	if normalized.LastResult != nil {
		toolResult.ResultRef = mcpObserveAppDirectorGoalResultRefV0(*normalized.LastResult)
		toolResult.Summary = strings.TrimSpace(normalized.LastResult.Summary)
		toolResult.ArtifactRefs = compactStringsMCPV0(normalized.LastResult.ArtifactRefs)
		toolResult.DomainReceiptRefs = compactStringsMCPV0(normalized.LastResult.DomainReceiptRefs)
		toolResult.EvidenceRefs = compactStringsMCPV0(append(toolResult.EvidenceRefs, normalized.LastResult.EvidenceRefs...))
		toolResult.ClosureIssues = append(toolResult.ClosureIssues, goalWorkIssuesMCPV0(normalized.LastResult.Issues)...)
	}
	if normalized.LastClosure != nil {
		toolResult.ClosureStatus = strings.TrimSpace(normalized.LastClosure.Status)
		toolResult.ClosureAccepted = normalized.LastClosure.Accepted
		toolResult.ClosureNeedsRework = normalized.LastClosure.NeedsRework
		toolResult.ClosureIssues = append(toolResult.ClosureIssues, goalWorkIssuesMCPV0(normalized.LastClosure.Issues)...)
		toolResult.EvidenceRefs = compactStringsMCPV0(append(toolResult.EvidenceRefs, normalized.LastClosure.EvidenceRefs...))
	}
	toolResult = mcpObserveAppDirectorGoalSuppressStaleClosureForRunningGoalV0(toolResult)
	toolResult = applyMCPObserveAppDirectorGoalDomainMetadataV0(toolResult, normalized)
	toolResult.RecommendedAction = mcpObserveAppDirectorGoalRecommendedActionV0(toolResult)
	return toolResult, nil
}

func mcpObserveAppDirectorGoalSuppressStaleClosureForRunningGoalV0(
	result MCPObserveAppDirectorGoalToolResultV0,
) MCPObserveAppDirectorGoalToolResultV0 {
	status := strings.TrimSpace(result.GoalStatus)
	if status != orquestagoal.GoalStatusRunningV0 && status != orquestagoal.GoalStatusAcceptedV0 {
		return result
	}
	if !result.ClosureNeedsRework && strings.TrimSpace(result.ClosureStatus) != orquestagoal.GoalStatusBlockedV0 {
		return result
	}
	result.ClosureStatus = ""
	result.ClosureAccepted = false
	result.ClosureNeedsRework = false
	return result
}

func applyMCPObserveAppDirectorGoalDomainMetadataV0(
	result MCPObserveAppDirectorGoalToolResultV0,
	state orquestagoal.GoalWorkStateV0,
) MCPObserveAppDirectorGoalToolResultV0 {
	metadata := mcpGoalWorkStateDomainOperationalMetadataV0(state)
	result.CurrentPhase = firstNonEmptyMCPV0(result.CurrentPhase, metadata.CurrentPhase)
	result.RetryFromPhase = firstNonEmptyMCPV0(result.RetryFromPhase, metadata.RetryFromPhase)
	result.DomainCounters = mergeMCPDomainOperationalCountersV0(result.DomainCounters, metadata.DomainCounters)
	if metadata.CloseSupersededByLocalEvidence {
		result.RecommendedAction = firstNonEmptyMCPV0(result.RecommendedAction, "close_superseded_by_local_evidence")
	}
	return result
}

func mcpObserveAppDirectorGoalResultRefV0(result orquestagoal.GoalWorkResultV0) string {
	for _, ref := range compactStringsMCPV0(result.EvidenceRefs) {
		if strings.Contains(strings.ToLower(ref), "result") {
			return ref
		}
	}
	return ""
}

func mcpObserveAppDirectorGoalRecommendedActionV0(
	result MCPObserveAppDirectorGoalToolResultV0,
) string {
	if mcpObserveAppDirectorGoalHasEvidenceV0(result, mcpAutoprogrammingEvidenceCodexAppServerThreadOutputSanitizedV0) {
		return mcpQueueGlobalStatusActionReplanNarrowContextV0
	}
	closureStatus := strings.TrimSpace(result.ClosureStatus)
	switch {
	case result.ClosureAccepted || closureStatus == orquestagoal.GoalStatusAcceptedV0:
		return "no_action_closed"
	}
	if mcpObserveAppDirectorGoalLooksUsageLimitedV0(result) {
		return "inspect_goal_backend_limits"
	}
	if mcpObserveAppDirectorGoalHasIssueV0(result, MCPGoalFirstQAFailedPublicTextV0) {
		return MCPGoalFirstReworkPublicTextActionV0
	}
	if mcpObserveAppDirectorGoalHasIssueV0(result, MCPGoalFirstArtifactPathsOmittedMaterializedV0) {
		return MCPGoalFirstRepairReceiptActionV0
	}
	if mcpObserveAppDirectorGoalHasIssueV0(result, MCPGoalFirstOutOfScopeMaterializedArtifactsV0) {
		return MCPGoalFirstReworkWriteSetViolationActionV0
	}
	if mcpObserveAppDirectorGoalHasIssueV0(result, MCPGoalFirstMissingTerminalReceiptAfterArtifactsPassV0) {
		return MCPGoalFirstRepairReceiptActionV0
	}
	if mcpObserveAppDirectorGoalHasIssueV0(result, MCPGoalFirstRequiredTestEvidenceMissingV0) {
		return MCPGoalFirstRepairReceiptActionV0
	}
	if mcpObserveAppDirectorGoalHasIssueV0(result, MCPGoalFirstRepairReceiptRequiresReworkV0) {
		return "replan"
	}
	if mcpObserveAppDirectorGoalHasIssueV0(result, mcpAutoprogrammingActionWriteSetRequiresWorkspaceWriteV0) {
		return mcpQueueGlobalStatusActionConfigureWorkspaceWriteSandboxV0
	}
	if mcpObserveAppDirectorGoalHasIssueV0(result, mcpAutoprogrammingActionWriteSetGuardAllowedWriteSetMissingV0) ||
		mcpObserveAppDirectorGoalHasIssueV0(result, mcpAutoprogrammingActionWriteSetGuardAllowedWriteSetMismatchV0) {
		return mcpQueueGlobalStatusActionRepairGoalWriteSetContractV0
	}
	if mcpObserveAppDirectorGoalHasIssueV0(result, MCPGoalFirstPhase0CompleteNonPublishableV0) {
		return MCPGoalFirstContinueFromPhase0ActionV0
	}
	if mcpObserveAppDirectorGoalHasIssueV0(result, MCPGoalFirstPartialArtifactsWrittenV0) {
		return MCPGoalFirstReviewPartialArtifactsActionV0
	}
	status := strings.TrimSpace(result.GoalStatus)
	switch status {
	case orquestagoal.GoalStatusRunningV0, orquestagoal.GoalStatusAcceptedV0:
		return "observe_later"
	}
	switch {
	case result.ClosureNeedsRework || closureStatus == orquestagoal.GoalStatusBlockedV0:
		return "replan"
	}
	switch status {
	case orquestagoal.GoalStatusRunningV0, orquestagoal.GoalStatusAcceptedV0:
		return "observe_later"
	case orquestagoal.GoalStatusCompleteV0:
		return "observe_later"
	case orquestagoal.GoalStatusBlockedV0, orquestagoal.GoalStatusInvalidV0:
		return "blocked"
	default:
		return "observe_later"
	}
}

func mcpObserveAppDirectorGoalLooksUsageLimitedV0(
	result MCPObserveAppDirectorGoalToolResultV0,
) bool {
	if mcpIssueSetLooksUsageLimitedV0(result.ClosureIssues) {
		return true
	}
	summary := strings.ToLower(strings.TrimSpace(result.Summary))
	return strings.Contains(summary, "usagelimited") ||
		strings.Contains(summary, "usage_limited") ||
		strings.Contains(summary, "budgetlimited") ||
		strings.Contains(summary, "budget_limited") ||
		strings.Contains(summary, "provider_limited")
}

func mcpObserveAppDirectorGoalHasIssueV0(
	result MCPObserveAppDirectorGoalToolResultV0,
	code string,
) bool {
	return mcpObserveAppDirectorGoalHasClosureIssueCodeV0(result.ClosureIssues, code)
}

func mcpObserveAppDirectorGoalHasEvidenceV0(
	result MCPObserveAppDirectorGoalToolResultV0,
	ref string,
) bool {
	return containsStringMCPV0(result.EvidenceRefs, ref)
}

func mcpObserveAppDirectorGoalHasClosureIssueCodeV0(
	issues []MCPValidationIssueV0,
	code string,
) bool {
	code = strings.TrimSpace(code)
	if code == "" {
		return false
	}
	for _, issue := range issues {
		if strings.TrimSpace(issue.Code) == code {
			return true
		}
	}
	return false
}

func mcpIssueSetLooksUsageLimitedV0(issues []MCPValidationIssueV0) bool {
	for _, issue := range issues {
		code := strings.ToLower(strings.TrimSpace(issue.Code))
		if strings.Contains(code, "usage_limited") ||
			strings.Contains(code, "provider_limited") ||
			strings.Contains(code, "budget_limited") {
			return true
		}
	}
	return false
}

func mergeMCPObserveAppDirectorGoalPartialIntoTimeoutV0(
	timeout MCPObserveAppDirectorGoalToolResultV0,
	partial MCPObserveAppDirectorGoalToolResultV0,
) MCPObserveAppDirectorGoalToolResultV0 {
	if partial.RunRef == "" && partial.GoalRef == "" && partial.GoalStatus == "" {
		return timeout
	}
	timeout.Partial = true
	timeout.RunRef = firstNonEmptyMCPV0(partial.RunRef, timeout.RunRef)
	timeout.RunStatus = firstNonEmptyMCPV0(partial.RunStatus, timeout.RunStatus)
	timeout.DirectorExecutionMode = firstNonEmptyMCPV0(partial.DirectorExecutionMode, timeout.DirectorExecutionMode)
	timeout.GoalRef = firstNonEmptyMCPV0(partial.GoalRef, timeout.GoalRef)
	timeout.ExternalGoalRef = firstNonEmptyMCPV0(partial.ExternalGoalRef, timeout.ExternalGoalRef)
	timeout.GoalStatus = firstNonEmptyMCPV0(partial.GoalStatus, timeout.GoalStatus)
	timeout.ResultRef = firstNonEmptyMCPV0(partial.ResultRef, timeout.ResultRef)
	timeout.LastEventAt = firstNonEmptyMCPV0(partial.LastEventAt, timeout.LastEventAt)
	timeout.CurrentPhase = firstNonEmptyMCPV0(partial.CurrentPhase, timeout.CurrentPhase)
	timeout.RetryFromPhase = firstNonEmptyMCPV0(partial.RetryFromPhase, timeout.RetryFromPhase)
	timeout.DomainCounters = mergeMCPDomainOperationalCountersV0(timeout.DomainCounters, partial.DomainCounters)
	timeout.ProcessRefs = compactStringsMCPV0(append(timeout.ProcessRefs, partial.ProcessRefs...))
	timeout.RecommendedAction = firstNonEmptyMCPV0(partial.RecommendedAction, timeout.RecommendedAction)
	timeout.ClosureStatus = firstNonEmptyMCPV0(partial.ClosureStatus, timeout.ClosureStatus)
	timeout.ClosureAccepted = timeout.ClosureAccepted || partial.ClosureAccepted
	timeout.ClosureNeedsRework = timeout.ClosureNeedsRework || partial.ClosureNeedsRework
	timeout.Summary = firstNonEmptyMCPV0(partial.Summary, timeout.Summary)
	timeout.ArtifactRefs = compactStringsMCPV0(append(timeout.ArtifactRefs, partial.ArtifactRefs...))
	timeout.DomainReceiptRefs = compactStringsMCPV0(append(timeout.DomainReceiptRefs, partial.DomainReceiptRefs...))
	timeout.ExpectedReceiptRefs = compactStringsMCPV0(append(timeout.ExpectedReceiptRefs, partial.ExpectedReceiptRefs...))
	timeout.EvidenceRefs = compactStringsMCPV0(append(timeout.EvidenceRefs, partial.EvidenceRefs...))
	timeout.ClosureIssues = append(timeout.ClosureIssues, partial.ClosureIssues...)
	return timeout
}

func EnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0(
	result MCPObserveAppDirectorGoalToolResultV0,
	refs MCPDirectorGoalMaterializedRefsV0,
) MCPObserveAppDirectorGoalToolResultV0 {
	result.ArtifactRefs = compactStringsMCPV0(append(result.ArtifactRefs, refs.ArtifactRefs...))
	result.DomainReceiptRefs = compactStringsMCPV0(append(result.DomainReceiptRefs, refs.DomainReceiptRefs...))
	result.ExpectedReceiptRefs = compactStringsMCPV0(append(result.ExpectedReceiptRefs, refs.ExpectedReceiptRefs...))
	result.EvidenceRefs = compactStringsMCPV0(append(result.EvidenceRefs, refs.EvidenceRefs...))
	for _, code := range compactStringsMCPV0(refs.IssueCodes) {
		if mcpObserveAppDirectorGoalHasClosureIssueCodeV0(result.ClosureIssues, code) {
			continue
		}
		field := "goal_first.receipt"
		if code == MCPGoalFirstQAFailedPublicTextV0 {
			field = "goal_first.qa_public_text"
		} else if code == MCPGoalFirstArtifactPathsOmittedMaterializedV0 {
			field = "goal_first.artifact_paths"
		} else if code == MCPGoalFirstOutOfScopeMaterializedArtifactsV0 {
			field = "goal_first.write_set"
		} else if code == MCPGoalFirstRequiredTestEvidenceMissingV0 {
			field = "goal_first.required_tests"
		} else if code == MCPGoalFirstPhase0CompleteNonPublishableV0 {
			field = "goal_first.phase0"
		} else if code == MCPGoalFirstPartialArtifactsWrittenV0 {
			field = "goal_first.partial_artifacts"
		}
		result.ClosureIssues = append(result.ClosureIssues, MCPValidationIssueV0{
			Code:    code,
			Field:   field,
			Message: code,
		})
	}
	result.RecommendedAction = mcpObserveAppDirectorGoalRecommendedActionV0(result)
	return result
}

func NewMCPObserveAppDirectorGoalTimeoutResultWithPartialV0(
	timeout MCPObserveAppDirectorGoalToolResultV0,
	partial MCPObserveAppDirectorGoalToolResultV0,
) MCPObserveAppDirectorGoalToolResultV0 {
	return mergeMCPObserveAppDirectorGoalPartialIntoTimeoutV0(timeout, partial)
}

func NewMCPObserveAppDirectorGoalErrorResultV0(
	input MCPObserveAppDirectorGoalToolInputV0,
	code string,
	field string,
	message string,
) MCPObserveAppDirectorGoalToolResultV0 {
	code = strings.TrimSpace(code)
	if code == "" {
		code = "observe_app_director_goal_error"
	}
	return MCPObserveAppDirectorGoalToolResultV0{
		Estado:        MCPObserveAppDirectorGoalEstadoErrorV0,
		RequestID:     strings.TrimSpace(input.RequestID),
		CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		RunRef:        strings.TrimSpace(input.RunRef),
		Errores: []MCPValidationIssueV0{{
			Code:    code,
			Field:   strings.TrimSpace(field),
			Message: strings.TrimSpace(firstNonEmptyMCPV0(message, code)),
		}},
	}
}

func NewMCPObserveAppDirectorGoalErrorResultFromErrorV0(
	input MCPObserveAppDirectorGoalToolInputV0,
	err error,
) (MCPObserveAppDirectorGoalToolResultV0, bool) {
	issue, ok := mcpObserveAppDirectorGoalPublicIssueFromErrorV0(err)
	if !ok {
		return MCPObserveAppDirectorGoalToolResultV0{}, false
	}
	return NewMCPObserveAppDirectorGoalErrorResultV0(input, issue.Code, issue.Field, issue.Message), true
}

func NewMCPAutoprogrammingObserveGoalErrorResultFromErrorV0(
	input MCPAutoprogrammingObserveGoalToolInputV0,
	err error,
) (MCPAutoprogrammingObserveGoalToolResultV0, bool) {
	issue, ok := mcpObserveAppDirectorGoalPublicIssueFromErrorV0(err)
	if !ok {
		return MCPAutoprogrammingObserveGoalToolResultV0{}, false
	}
	issue.Code = strings.Replace(issue.Code, "observe_app_director_goal", "autoprogramming_observe_goal", 1)
	return NewMCPAutoprogrammingObserveGoalErrorResultV0(input, issue.Code, issue.Field, issue.Message), true
}

func mcpObserveAppDirectorGoalPublicIssueFromErrorV0(err error) (mcpObserveAppDirectorGoalPublicIssueV0, bool) {
	if err == nil {
		return mcpObserveAppDirectorGoalPublicIssueV0{}, false
	}
	var serviceIssue orquestaappdirectorservice.AppDirectorServiceIssueV0
	if errors.As(err, &serviceIssue) {
		return mcpObserveAppDirectorGoalPublicIssueForFieldV0(serviceIssue.Field), true
	}
	var lifecycleIssue orquestagoal.GoalWorkLifecycleIssueErrorV0
	if errors.As(err, &lifecycleIssue) {
		if issue, ok := mcpObserveAppDirectorGoalPublicIssueFromGoalIssuesV0(lifecycleIssue.Issues); ok {
			return issue, true
		}
		return mcpObserveAppDirectorGoalPublicIssueForFieldV0(lifecycleIssue.Field), true
	}
	var coreIssue orquestacionnucleoapp.ErrorV0
	if errors.As(err, &coreIssue) {
		return mcpObserveAppDirectorGoalPublicIssueForCoreErrorV0(coreIssue)
	}
	if mcpObserveAppDirectorGoalLooksLikeMissingStateV0(err.Error()) {
		return mcpObserveAppDirectorGoalPublicIssueV0{
			Code:    "observe_app_director_goal_state_not_found",
			Field:   "goal_state",
			Message: "goal_state_not_found",
		}, true
	}
	return mcpObserveAppDirectorGoalPublicIssueV0{}, false
}

func mcpObserveAppDirectorGoalPublicIssueFromGoalIssuesV0(
	issues []orquestagoal.GoalWorkIssueV0,
) (mcpObserveAppDirectorGoalPublicIssueV0, bool) {
	for _, issue := range issues {
		code := strings.TrimSpace(issue.Code)
		if code == "" {
			continue
		}
		return mcpObserveAppDirectorGoalPublicIssueV0{
			Code:    code,
			Field:   strings.TrimSpace(issue.Field),
			Message: code,
		}, true
	}
	return mcpObserveAppDirectorGoalPublicIssueV0{}, false
}

func mcpObserveAppDirectorGoalPublicIssueForFieldV0(field string) mcpObserveAppDirectorGoalPublicIssueV0 {
	field = strings.TrimSpace(field)
	switch field {
	case "run_ref":
		return mcpObserveAppDirectorGoalPublicIssueV0{
			Code:    "observe_app_director_goal_input_invalid",
			Field:   "run_ref",
			Message: "run_ref_requerido",
		}
	case "ports.goal_state_store":
		return mcpObserveAppDirectorGoalPublicIssueV0{
			Code:    "observe_app_director_goal_state_store_unbound",
			Field:   "goal_state_store",
			Message: "goal_state_store_not_configured",
		}
	case "ports.goal_observer":
		return mcpObserveAppDirectorGoalPublicIssueV0{
			Code:    "observe_app_director_goal_observer_unbound",
			Field:   "goal_observer",
			Message: "goal_observer_not_configured",
		}
	case "ports.goal_closure_validator":
		return mcpObserveAppDirectorGoalPublicIssueV0{
			Code:    "observe_app_director_goal_closure_validator_unbound",
			Field:   "goal_closure_validator",
			Message: "goal_closure_validator_not_configured",
		}
	case "ports.run_store", "ports.event_sink":
		return mcpObserveAppDirectorGoalPublicIssueV0{
			Code:    "observe_app_director_goal_closure_reflection_unbound",
			Field:   strings.TrimPrefix(field, "ports."),
			Message: "goal_closure_reflection_not_configured",
		}
	default:
		return mcpObserveAppDirectorGoalPublicIssueV0{
			Code:    "observe_app_director_goal_dependency_invalid",
			Field:   field,
			Message: "observe_goal_dependency_invalid",
		}
	}
}

func mcpObserveAppDirectorGoalPublicIssueForCoreErrorV0(
	coreIssue orquestacionnucleoapp.ErrorV0,
) (mcpObserveAppDirectorGoalPublicIssueV0, bool) {
	field := strings.TrimSpace(coreIssue.Field)
	message := strings.TrimSpace(coreIssue.Message)
	if field == "app_director_goal_state" && mcpObserveAppDirectorGoalLooksLikeMissingStateV0(message) {
		return mcpObserveAppDirectorGoalPublicIssueV0{
			Code:    "observe_app_director_goal_state_not_found",
			Field:   "goal_state",
			Message: "goal_state_not_found",
		}, true
	}
	if strings.HasPrefix(field, "app_director_goal_state") {
		return mcpObserveAppDirectorGoalPublicIssueV0{
			Code:    "observe_app_director_goal_state_invalid",
			Field:   "goal_state",
			Message: "goal_state_invalid",
		}, true
	}
	if coreIssue.Code == orquestacionnucleoapp.ErrNucleoOrquestacionStoreV0 &&
		mcpObserveAppDirectorGoalLooksLikeMissingStateV0(coreIssue.Error()) {
		return mcpObserveAppDirectorGoalPublicIssueV0{
			Code:    "observe_app_director_goal_state_not_found",
			Field:   "goal_state",
			Message: "goal_state_not_found",
		}, true
	}
	return mcpObserveAppDirectorGoalPublicIssueV0{}, false
}

func mcpObserveAppDirectorGoalLooksLikeMissingStateV0(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return false
	}
	return (strings.Contains(normalized, "goal_state") ||
		strings.Contains(normalized, "goal state") ||
		strings.Contains(normalized, "estado de goal")) &&
		(strings.Contains(normalized, "not found") ||
			strings.Contains(normalized, "no encontrado") ||
			strings.Contains(normalized, "no encontrada"))
}

func goalWorkIssuesMCPV0(values []orquestagoal.GoalWorkIssueV0) []MCPValidationIssueV0 {
	out := make([]MCPValidationIssueV0, 0, len(values))
	for _, value := range values {
		code := strings.TrimSpace(value.Code)
		if code == "" {
			code = "goal_issue"
		}
		out = append(out, MCPValidationIssueV0{
			Code:    code,
			Field:   strings.TrimSpace(value.Field),
			Message: code,
		})
	}
	if out == nil {
		return []MCPValidationIssueV0{}
	}
	return out
}
