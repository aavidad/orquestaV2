package orquestaappdirectorservice

import (
	"crypto/sha256"
	"encoding/hex"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"sort"
	"strings"
)

type operationalDirectorRequiredTestsAutoReplanRefSetV0 struct {
	GateRef     string
	ReplanRef   string
	CapacityRef string
	AgentRef    string
}

func operationalDirectorRequiredTestsAutoReplanRefsV0(
	request ContinueAppDirectorRequestV0,
	match operationalDirectorPlanAcceptedReviewMatchV0,
	failedRefs []string,
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
		strings.Join(sortedServiceRefsV0(failedRefs), "|"),
	)
	base := stem + "-" + digest
	return operationalDirectorRequiredTestsAutoReplanRefSetV0{
		GateRef:     "quality-gate-ref-app-director-required-tests-failed-" + base,
		ReplanRef:   "replan-ref-app-director-required-tests-failed-" + base,
		CapacityRef: "capacity-ref-app-director-required-tests-retry-" + base,
		AgentRef:    "agent-ref-app-director-required-tests-retry-" + base,
	}
}

func operationalDirectorRequiredTestsPassedGateRefsV0(
	request ContinueAppDirectorRequestV0,
	match operationalDirectorPlanAcceptedReviewMatchV0,
	passedRefs []string,
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
		strings.Join(sortedServiceRefsV0(passedRefs), "|"),
		"required-tests-passed",
	)
	return operationalDirectorRequiredTestsAutoReplanRefSetV0{
		GateRef: "quality-gate-ref-app-director-required-tests-passed-" + stem + "-" + digest,
	}
}

func operationalDirectorRequiredTestsPassedGateEvidenceRefsV0(
	match operationalDirectorPlanAcceptedReviewMatchV0,
	passedRefs []string,
) []string {
	return compactServiceRefsV0(append([]string{
		match.DeliveryRef,
		match.ReviewRequestID,
		match.ReviewResultRef,
		match.AcceptedReviewRef,
	}, append(passedRefs, match.EvidenceRefs...)...))
}

func operationalDirectorRequiredTestsEvidenceMissingGateRefsV0(
	request ContinueAppDirectorRequestV0,
	match operationalDirectorPlanAcceptedReviewMatchV0,
	requiredTests []string,
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
		strings.Join(sortedServiceRefsV0(requiredTests), "|"),
		"required-tests-evidence-missing",
	)
	base := stem + "-" + digest
	return operationalDirectorRequiredTestsAutoReplanRefSetV0{
		GateRef:   "quality-gate-ref-app-director-required-tests-evidence-missing-" + base,
		ReplanRef: "replan-ref-app-director-required-tests-evidence-missing-" + base,
	}
}

func operationalDirectorRequiredTestsAutoReplanDigestV0(parts ...string) string {
	hash := sha256.New()
	for _, part := range parts {
		hash.Write([]byte(strings.TrimSpace(part)))
		hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))[:16]
}

func sortedServiceRefsV0(values []string) []string {
	refs := compactServiceRefsV0(values)
	sort.Strings(refs)
	return refs
}

func operationalDirectorRequiredTestsAutoReplanEvidenceRefsV0(
	match operationalDirectorPlanAcceptedReviewMatchV0,
	failedRefs []string,
) []string {
	return compactServiceRefsV0(append([]string{
		match.DeliveryRef,
		match.ReviewRequestID,
		match.ReviewResultRef,
		match.AcceptedReviewRef,
	}, append(failedRefs, match.EvidenceRefs...)...))
}

func operationalDirectorRunProgrammingPhaseActiveV0(run orquestacoreworkflow.OrchestrationRunV0) bool {
	return operationalDirectorRunPhaseActiveV0(run, orquestacoreworkflow.OrchestrationPhaseProgramacionV0)
}

func operationalDirectorRunPhaseActiveV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	phaseID orquestacoreworkflow.OrchestrationPhaseIDV0,
) bool {
	if strings.TrimSpace(string(run.CurrentPhase)) != string(phaseID) {
		return false
	}
	for _, phase := range run.Phases {
		if strings.TrimSpace(string(phase.ID)) == string(phaseID) &&
			phase.Status == orquestacoreworkflow.OrchestrationPhaseStatusActiveV0 {
			return true
		}
	}
	return false
}

func operationalDirectorRunContainsPhaseV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	phaseID orquestacoreworkflow.OrchestrationPhaseIDV0,
) bool {
	for _, phase := range run.Phases {
		if strings.TrimSpace(string(phase.ID)) == string(phaseID) {
			return true
		}
	}
	return false
}

func operationalDirectorPlanRequiredTestsQualityGateV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	trace operationalDirectorPlanReviewTraceV0,
	activeStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	taskRef string,
	failedRefs []string,
) (orquestacoreworkflow.QualityGateRecordedPayloadV0, bool) {
	taskRef = strings.TrimSpace(taskRef)
	for _, gateRef := range trace.QualityGateRefs {
		gate := trace.QualityGates[gateRef]
		if strings.TrimSpace(gate.RunRef) != strings.TrimSpace(run.RunID) ||
			strings.TrimSpace(gate.SubjectRef) != taskRef ||
			gate.Decision != orquestacoreworkflow.QualityGateDecisionBlockedV0 ||
			!operationalDirectorPlanProjectionReflectedV0(gate.GateRef, run.QualityGates) ||
			!operationalDirectorPlanRefsIntersectV0(failedRefs, append(gate.IssueRefs, gate.EvidenceRefs...)) {
			continue
		}
		if len(activeStep.TaskRefs) > 0 && !startAppDirectorStringInSetV0(activeStep.TaskRefs, taskRef) {
			continue
		}
		return gate, true
	}
	return orquestacoreworkflow.QualityGateRecordedPayloadV0{}, false
}

func operationalDirectorPlanReplanForQualityGateV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	trace operationalDirectorPlanReviewTraceV0,
	gateRef string,
	taskRef string,
) (orquestacoreworkflow.ReplanDecisionRecordedPayloadV0, bool) {
	for _, replanDecisionRef := range trace.ReplanDecisionRefs {
		replan := trace.ReplanDecisions[replanDecisionRef]
		if strings.TrimSpace(replan.SourceRef) == strings.TrimSpace(gateRef) &&
			strings.TrimSpace(replan.TaskRef) == strings.TrimSpace(taskRef) &&
			strings.TrimSpace(replan.RunRef) == strings.TrimSpace(run.RunID) &&
			operationalDirectorPlanProjectionReflectedV0(replan.ReplanRef, run.ReplanDecisions) {
			return replan, true
		}
	}
	return orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{}, false
}

func operationalDirectorPlanRefsIntersectV0(left []string, right []string) bool {
	for _, candidate := range compactServiceRefsV0(left) {
		if startAppDirectorStringInSetV0(right, candidate) {
			return true
		}
	}
	return false
}
