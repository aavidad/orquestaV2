package orquestaappdirectorservice

import (
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	"strings"
)

func operationalDirectorClosureIssueReplanReflectedV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	refs operationalDirectorRequiredTestsAutoReplanRefSetV0,
	match operationalDirectorPlanAcceptedReviewMatchV0,
) bool {
	taskRef := strings.TrimSpace(match.TaskRef)
	gateRef := strings.TrimSpace(refs.GateRef)
	replanRef := strings.TrimSpace(refs.ReplanRef)
	if taskRef == "" || gateRef == "" || replanRef == "" {
		return false
	}
	expectedGate := gateRef +
		"#decision:" + string(orquestacoreworkflow.QualityGateDecisionBlockedV0) +
		"#subject:" + taskRef
	expectedReplan := replanRef +
		"#source:" + gateRef +
		"#task:" + taskRef +
		"#action:" + string(orquestacoreworkflow.ReplanDecisionActionRetryTaskV0) +
		"#followups:" + strings.TrimSpace(refs.CapacityRef) + "+" + strings.TrimSpace(refs.AgentRef)
	return startAppDirectorStringInSetV0(run.QualityGates, expectedGate) &&
		startAppDirectorStringInSetV0(run.ReplanDecisions, expectedReplan)
}

func operationalDirectorClosureIssueGateSummaryV0(issueRefs []string) string {
	if startAppDirectorStringInSetV0(issueRefs, "operational_closure_insufficient") {
		return "Cierre operativo bloqueado por evidencia de cierre insuficiente."
	}
	return "Cierre operativo bloqueado por tests requeridos sin evidencia causal."
}

func operationalDirectorClosureIssueReplanSummaryV0(issueRefs []string) string {
	if startAppDirectorStringInSetV0(issueRefs, "operational_closure_insufficient") {
		return "Reintentar tarea tras cierre operativo insuficiente."
	}
	return "Reintentar tarea tras cierre bloqueado por tests requeridos."
}

func operationalDirectorClosureIssueAutoReplanRefsV0(
	request ContinueAppDirectorRequestV0,
	match operationalDirectorPlanAcceptedReviewMatchV0,
	issueRefs []string,
) operationalDirectorRequiredTestsAutoReplanRefSetV0 {
	stem := compactRecoveryRefV0(strings.TrimSpace(match.TaskRef))
	if len(stem) > 72 {
		stem = strings.Trim(stem[:72], "-")
	}
	digest := operationalDirectorRequiredTestsAutoReplanDigestV0(
		request.RunRef,
		match.TaskRef,
		match.DeliveryRef,
		match.ReviewRequestID,
		match.ReviewResultRef,
		match.AcceptedReviewRef,
		strings.Join(sortedServiceRefsV0(issueRefs), "|"),
		"operational-closure",
	)
	base := stem + "-" + digest
	return operationalDirectorRequiredTestsAutoReplanRefSetV0{
		GateRef:     "quality-gate-ref-app-director-operational-closure-blocked-" + base,
		ReplanRef:   "replan-ref-app-director-operational-closure-retry-" + base,
		CapacityRef: "capacity-ref-app-director-operational-closure-retry-" + base,
		AgentRef:    "agent-ref-app-director-operational-closure-retry-" + base,
	}
}
