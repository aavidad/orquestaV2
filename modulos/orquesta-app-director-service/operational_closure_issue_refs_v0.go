package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"strings"
)

func appDirectorClosureOnlyOpenTasksIssuesV0(issues []orquestacionnucleoapp.ErrorV0) bool {
	if len(issues) == 0 {
		return false
	}
	for _, issue := range issues {
		if issue.Code != orquestacionnucleoapp.ErrNucleoOrquestacionInvalidoV0 ||
			issue.Field != "run.open_tasks" {
			return false
		}
	}
	return true
}

func appDirectorClosureClosedTaskProgressV0(
	before orquestacoreworkflow.OrchestrationRunV0,
	after orquestacoreworkflow.OrchestrationRunV0,
) bool {
	beforeClosed := map[string]bool{}
	for _, taskRef := range compactServiceRefsV0(before.ClosedTasks) {
		beforeClosed[taskRef] = true
	}
	for _, taskRef := range compactServiceRefsV0(after.ClosedTasks) {
		if !beforeClosed[taskRef] {
			return true
		}
	}
	return false
}

func operationalDirectorPlanStateReplanClosureIssuesV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	closureRequest orquestacionnucleoapp.OperationalDirectorClosureRequestV0,
	issues []orquestacionnucleoapp.ErrorV0,
) ([]string, error) {
	issueRefs := operationalDirectorClosureReplannableIssueRefsV0(issues)
	if len(issueRefs) == 0 || ports.RunStore == nil || ports.EventSink == nil {
		return nil, nil
	}
	planRef := continueOperationalDirectorPlanRefV0(request)
	if planRef == "" || ports.OperationalPlanStateStore == nil {
		return nil, nil
	}
	state, err := ports.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, request.RunRef, planRef)
	if err != nil {
		return nil, err
	}
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 {
		return nil, nil
	}
	activeStep, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok ||
		activeStep.Kind != orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0 ||
		activeStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
		return nil, nil
	}
	if strings.TrimSpace(run.RunID) == "" {
		loadedRun, err := ports.RunStore.LoadRunV0(ctx, request.RunRef)
		if err != nil {
			return nil, err
		}
		run = loadedRun
	}
	taskRefs := compactServiceRefsV0(activeStep.TaskRefs)
	taskRef := strings.TrimSpace(closureRequest.TaskID)
	if taskRef == "" || !startAppDirectorStringInSetV0(taskRefs, taskRef) {
		return nil, nil
	}
	match, trace, ok, err := operationalDirectorClosureIssueAcceptedReviewMatchV0(ctx, request, ports, activeStep, run, closureRequest)
	if err != nil || !ok {
		return nil, err
	}
	if gate, ok := operationalDirectorPlanRequiredTestsQualityGateV0(run, trace, activeStep, match.TaskRef, issueRefs); ok {
		if replan, ok := operationalDirectorPlanReplanForQualityGateV0(run, trace, gate.GateRef, match.TaskRef); ok {
			return compactServiceRefsV0(append([]string{gate.GateRef, replan.ReplanRef}, issueRefs...)), nil
		}
	}
	replanRefs, err := operationalDirectorPlanEmitClosureIssueReplanDecisionV0(ctx, request, ports, run, match, issueRefs)
	return compactServiceRefsV0(append(replanRefs, issueRefs...)), err
}

func operationalDirectorClosureReplannableIssueRefsV0(
	issues []orquestacionnucleoapp.ErrorV0,
) []string {
	refs := make([]string, 0, len(issues)*2)
	closureInsufficient := false
	for _, issue := range issues {
		field := strings.TrimSpace(issue.Field)
		if !operationalDirectorClosureIssueReplannableV0(field) {
			continue
		}
		refs = append(refs, field)
		if code := strings.TrimSpace(issue.Code); code != "" {
			refs = append(refs, code)
		}
		if field != "required_test_evidence_refs" {
			closureInsufficient = true
		}
	}
	if closureInsufficient {
		refs = append(refs, "operational_closure_insufficient")
	}
	return compactServiceRefsV0(refs)
}

func operationalDirectorClosureIssueReplannableV0(field string) bool {
	switch strings.TrimSpace(field) {
	case "required_test_evidence_refs",
		"operational_closure_insufficient",
		"validation_ref",
		"closure_ref":
		return true
	default:
		return operationalDirectorClosureIssueRepairableV0(field)
	}
}

func operationalDirectorClosureIssueRepairableV0(field string) bool {
	switch strings.ReplaceAll(strings.TrimSpace(field), "-", "_") {
	case "artifact_ref",
		"artifact_refs",
		"domain_artifact_ref",
		"domain_artifact_refs",
		"external_artifact_ref",
		"external_artifact_refs",
		"validation_evidence_ref",
		"validation_evidence_refs",
		"closure_evidence_ref",
		"closure_evidence_refs",
		"domain_validation_ref",
		"domain_validation_refs":
		return true
	default:
		return false
	}
}
