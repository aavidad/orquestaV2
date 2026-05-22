package orquestaappdirectorservice

import (
	"context"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func maybeCloseOperationalDirectorV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	loop orquestacionnucleoapp.ProgressiveLoopResultV0,
	loopRequest orquestacionnucleoapp.ProgressiveLoopRequestV0,
) (orquestacionnucleoapp.ProgressiveLoopResultV0, []orquestacionnucleoapp.ErrorV0, error) {
	if loop.Status != orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0 {
		return loop, nil, nil
	}
	if loop.PendingOutboxCount > 0 {
		if err := operationalDirectorPlanStateBlockedAtReplanOrCloseV0(
			ctx,
			request,
			ports,
			"operational-closure-outbox-pending",
			nil,
		); err != nil {
			return loop, nil, err
		}
		return loop, nil, nil
	}
	if ports.OperationalClosureSource == nil {
		if err := operationalDirectorPlanStateBlockedAtReplanOrCloseV0(
			ctx,
			request,
			ports,
			"operational-closure-source-unavailable",
			nil,
		); err != nil {
			return loop, nil, err
		}
		return loop, nil, nil
	}
	if ports.DirectorTaskStore == nil {
		if err := operationalDirectorPlanStateBlockedAtReplanOrCloseV0(
			ctx,
			request,
			ports,
			"operational-closure-task-store-unavailable",
			nil,
		); err != nil {
			return loop, nil, err
		}
		return loop, nil, nil
	}
	blocked, err := operationalDirectorPlanStateBlocksClosureV0(ctx, request, ports)
	if err != nil || blocked {
		return loop, nil, err
	}
	requiredTestEvidenceRefs, err := operationalDirectorPlanStateRequiredTestEvidenceRefsForClosureV0(ctx, request, ports)
	if err != nil {
		return loop, nil, err
	}
	closureRequest, ok, err := ports.OperationalClosureSource.BuildOperationalDirectorClosureRequestV0(
		ctx,
		AppDirectorOperationalClosureRequestV0{
			Run:                      loop.Run,
			LoopStatus:               loop.Status,
			OccurredAt:               request.OccurredAt,
			CorrelationID:            request.CorrelationID,
			RequestedBy:              request.RequestedBy,
			WaitAgentRefs:            append([]string(nil), loopRequest.WaitAgentRefs...),
			WaitScopeApplied:         loopRequest.WaitScopeApplied,
			WaitCohortRef:            request.WaitCohortRef,
			WaitWaveRef:              request.WaitWaveRef,
			WaitParentTaskRef:        request.WaitParentTaskRef,
			RequiredTestEvidenceRefs: requiredTestEvidenceRefs,
			EvidenceRefs:             []string{"evidence-ref-app-director-operational-closure-v0"},
		},
	)
	if err != nil || !ok {
		if blockErr := operationalDirectorPlanStateBlockedAfterClosureV0(
			ctx,
			request,
			ports,
			"operational-closure-source-unavailable",
			nil,
		); blockErr != nil {
			return loop, nil, blockErr
		}
		return loop, nil, err
	}
	closureRequest = appDirectorClosureRequestWithDefaultsV0(request, closureRequest)
	closure, err := (orquestacionnucleoapp.OperationalDirectorClosureV0{
		RunStore:                  ports.RunStore,
		EventSink:                 ports.EventSink,
		EventReader:               ports.EventReader,
		TaskStore:                 ports.DirectorTaskStore,
		RequiredTestEvidenceStore: ports.RequiredTestEvidenceStore,
		RequestedBy:               request.RequestedBy,
	}).CloseOperationalDirectorRunV0(ctx, closureRequest)
	if err != nil {
		return loop, nil, err
	}
	if len(closure.Issues) > 0 || closure.Run.RunID == "" {
		if closure.Run.RunID != "" && appDirectorClosureOnlyOpenTasksIssuesV0(closure.Issues) {
			loop.Run = closure.Run
			return loop, nil, nil
		}
		replanRefs, replanErr := operationalDirectorPlanStateReplanClosureIssuesV0(
			ctx,
			request,
			ports,
			closure.Run,
			closureRequest,
			closure.Issues,
		)
		if replanErr != nil {
			return loop, closure.Issues, replanErr
		}
		if blockErr := operationalDirectorPlanStateBlockedAfterClosureV0(
			ctx,
			request,
			ports,
			"operational-closure-issues",
			closure.Issues,
			replanRefs...,
		); blockErr != nil {
			return loop, closure.Issues, blockErr
		}
		return loop, closure.Issues, nil
	}
	if err := operationalDirectorPlanStateClosedAfterClosureV0(ctx, request, ports, closureRequest); err != nil {
		return loop, nil, err
	}
	loop.Run = closure.Run
	return loop, nil, nil
}

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
	if len(taskRefs) != 1 || strings.TrimSpace(closureRequest.TaskID) != taskRefs[0] {
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
		return false
	}
}

func operationalDirectorClosureIssueAcceptedReviewMatchV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	activeStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	closureRequest orquestacionnucleoapp.OperationalDirectorClosureRequestV0,
) (operationalDirectorPlanAcceptedReviewMatchV0, operationalDirectorPlanReviewTraceV0, bool, error) {
	reader := operationalDirectorPlanStateEventReaderV0(ports)
	if reader == nil {
		return operationalDirectorPlanAcceptedReviewMatchV0{}, operationalDirectorPlanReviewTraceV0{}, false, nil
	}
	events, err := reader.LoadRunEventsV0(ctx, request.RunRef)
	if err != nil {
		return operationalDirectorPlanAcceptedReviewMatchV0{}, operationalDirectorPlanReviewTraceV0{}, false, err
	}
	trace := operationalDirectorPlanReviewTraceFromEventsV0(events)
	taskRef := strings.TrimSpace(closureRequest.TaskID)
	deliveryRef := strings.TrimSpace(closureRequest.DeliveryRef)
	acceptedReviewRef := strings.TrimSpace(closureRequest.AcceptedReviewRef)
	if taskRef == "" || deliveryRef == "" || acceptedReviewRef == "" {
		return operationalDirectorPlanAcceptedReviewMatchV0{}, trace, false, nil
	}
	if len(activeStep.TaskRefs) > 0 && !startAppDirectorStringInSetV0(activeStep.TaskRefs, taskRef) {
		return operationalDirectorPlanAcceptedReviewMatchV0{}, trace, false, nil
	}
	if len(activeStep.DeliveryRefs) > 0 && !startAppDirectorStringInSetV0(activeStep.DeliveryRefs, deliveryRef) {
		return operationalDirectorPlanAcceptedReviewMatchV0{}, trace, false, nil
	}
	delivery := trace.Deliveries[deliveryRef]
	if strings.TrimSpace(delivery.TaskID) != taskRef ||
		!startAppDirectorStringInSetV0(run.Deliveries, deliveryRef) ||
		!startAppDirectorStringInSetV0(run.AcceptedReviews, acceptedReviewRef) {
		return operationalDirectorPlanAcceptedReviewMatchV0{}, trace, false, nil
	}
	if agentRef := strings.TrimSpace(delivery.AgentRef); agentRef != "" &&
		len(activeStep.AgentRefs) > 0 &&
		!startAppDirectorStringInSetV0(activeStep.AgentRefs, agentRef) {
		return operationalDirectorPlanAcceptedReviewMatchV0{}, trace, false, nil
	}
	accepted := trace.AcceptedReviews[acceptedReviewRef]
	reviewRequestID := strings.TrimSpace(accepted.ReviewRequestID)
	if reviewRequestID == "" {
		return operationalDirectorPlanAcceptedReviewMatchV0{}, trace, false, nil
	}
	reviewRequest := trace.ReviewRequests[reviewRequestID]
	if strings.TrimSpace(reviewRequest.DeliveryRef) != deliveryRef {
		return operationalDirectorPlanAcceptedReviewMatchV0{}, trace, false, nil
	}
	reviewResult, ok := operationalDirectorClosureIssueAcceptedReviewResultV0(trace, reviewRequestID, deliveryRef)
	if !ok {
		return operationalDirectorPlanAcceptedReviewMatchV0{}, trace, false, nil
	}
	return operationalDirectorPlanAcceptedReviewMatchV0{
		TaskRef:           taskRef,
		AgentRef:          strings.TrimSpace(delivery.AgentRef),
		DeliveryRef:       deliveryRef,
		ReviewRequestID:   reviewRequestID,
		ReviewResultRef:   strings.TrimSpace(reviewResult.ReviewResultRef),
		AcceptedReviewRef: acceptedReviewRef,
		EvidenceRefs: compactServiceRefsV0(append(append(append(
			append([]string(nil), delivery.EvidenceRefs...),
			reviewRequest.EvidenceRefs...),
			reviewResult.EvidenceRefs...),
			acceptedReviewRef)),
	}, trace, true, nil
}

func operationalDirectorClosureIssueAcceptedReviewResultV0(
	trace operationalDirectorPlanReviewTraceV0,
	reviewRequestID string,
	deliveryRef string,
) (orquestacoreworkflow.ReviewResultV0, bool) {
	for _, result := range trace.ReviewResults {
		if strings.TrimSpace(result.ReviewRequestID) == reviewRequestID &&
			strings.TrimSpace(result.DeliveryRef) == deliveryRef &&
			result.Status == orquestacoreworkflow.ReviewResultStatusAcceptedV0 {
			return result, true
		}
	}
	return orquestacoreworkflow.ReviewResultV0{}, false
}

func operationalDirectorPlanEmitClosureIssueReplanDecisionV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	match operationalDirectorPlanAcceptedReviewMatchV0,
	issueRefs []string,
) ([]string, error) {
	refs := operationalDirectorClosureIssueAutoReplanRefsV0(request, match, issueRefs)
	storedRun, err := ports.RunStore.LoadRunV0(ctx, request.RunRef)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(storedRun.RunID) != strings.TrimSpace(run.RunID) {
		return nil, nil
	}
	storedRun, ready, err := operationalDirectorPlanEnsureProgrammingPhaseForRequiredTestsReplanV0(ctx, request, ports, storedRun, refs)
	if err != nil || !ready || !operationalDirectorRunProgrammingPhaseActiveV0(storedRun) {
		return nil, err
	}
	gatePayload := orquestacoreworkflow.RecordQualityGateCommandPayloadV0{
		RunRef:       request.RunRef,
		GateRef:      refs.GateRef,
		PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		SubjectRef:   match.TaskRef,
		Decision:     orquestacoreworkflow.QualityGateDecisionBlockedV0,
		IssueRefs:    append([]string(nil), issueRefs...),
		Summary:      operationalDirectorClosureIssueGateSummaryV0(issueRefs),
		EvidenceRefs: operationalDirectorRequiredTestsAutoReplanEvidenceRefsV0(match, issueRefs),
	}
	gateCommand, err := orquestacoreworkflow.NewRecordQualityGateCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-" + refs.GateRef,
			RunID:          request.RunRef,
			IdempotencyKey: "idem-" + refs.GateRef,
			CorrelationID:  request.CorrelationID,
			RequestedBy:    request.RequestedBy,
			OccurredAt:     request.OccurredAt,
		},
		gatePayload,
	)
	if err != nil {
		return nil, err
	}
	if _, err := orquestacionnucleoapp.HandleStoredWorkflowCommandV0(ctx, ports.RunStore, ports.EventSink, gateCommand); err != nil {
		return nil, err
	}
	replanPayload := orquestacoreworkflow.RecordReplanDecisionCommandPayloadV0{
		ReplanRef:      refs.ReplanRef,
		RunRef:         request.RunRef,
		TaskRef:        match.TaskRef,
		SourceRef:      refs.GateRef,
		AcceptedAction: orquestacoreworkflow.ReplanDecisionActionRetryTaskV0,
		FollowupRefs:   []string{refs.CapacityRef, refs.AgentRef},
		Summary:        operationalDirectorClosureIssueReplanSummaryV0(issueRefs),
		EvidenceRefs:   compactServiceRefsV0(append([]string{refs.GateRef}, issueRefs...)),
	}
	replanCommand, err := orquestacoreworkflow.NewRecordReplanDecisionCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-" + refs.ReplanRef,
			RunID:          request.RunRef,
			IdempotencyKey: "idem-" + refs.ReplanRef,
			CorrelationID:  request.CorrelationID,
			RequestedBy:    request.RequestedBy,
			OccurredAt:     request.OccurredAt,
		},
		replanPayload,
	)
	if err != nil {
		return nil, err
	}
	if _, err := orquestacionnucleoapp.HandleStoredWorkflowCommandV0(ctx, ports.RunStore, ports.EventSink, replanCommand); err != nil {
		return nil, err
	}
	return compactServiceRefsV0([]string{refs.GateRef, refs.ReplanRef, refs.CapacityRef, refs.AgentRef}), nil
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

func operationalDirectorPlanStateBlockedAtReplanOrCloseV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	reason string,
	issues []orquestacionnucleoapp.ErrorV0,
) error {
	atReplanOrClose, err := operationalDirectorPlanStateIsAtReplanOrCloseV0(ctx, request, ports)
	if err != nil || !atReplanOrClose {
		return err
	}
	return operationalDirectorPlanStateBlockedAfterClosureV0(ctx, request, ports, reason, issues)
}

func operationalDirectorPlanStateIsAtReplanOrCloseV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
) (bool, error) {
	planRef := continueOperationalDirectorPlanRefV0(request)
	if planRef == "" || ports.OperationalPlanStateStore == nil {
		return false, nil
	}
	state, err := ports.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, request.RunRef, planRef)
	if err != nil {
		return false, err
	}
	if state.Status == orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 {
		return false, nil
	}
	if state.Status == orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 {
		return false, nil
	}
	step, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok {
		return false, nil
	}
	return step.Kind == orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0 &&
		step.Status == orquestadirectoroperativo.OperationalDirectorStepRunningV0, nil
}

func operationalDirectorPlanStateBlocksClosureV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
) (bool, error) {
	planRef := continueOperationalDirectorPlanRefV0(request)
	if planRef == "" || ports.OperationalPlanStateStore == nil {
		return false, nil
	}
	state, err := ports.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, request.RunRef, planRef)
	if err != nil {
		return false, err
	}
	if state.Status == orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 {
		return false, nil
	}
	if state.Status == orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 {
		return true, nil
	}
	step, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok {
		return false, nil
	}
	if step.Kind == orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0 &&
		step.Status == orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
		return false, nil
	}
	return true, nil
}

func operationalDirectorPlanStateRequiredTestEvidenceRefsForClosureV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
) ([]string, error) {
	planRef := continueOperationalDirectorPlanRefV0(request)
	if planRef == "" || ports.OperationalPlanStateStore == nil {
		return nil, nil
	}
	state, err := ports.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, request.RunRef, planRef)
	if err != nil {
		return nil, err
	}
	refs := []string(nil)
	for _, step := range state.Steps {
		refs = append(refs, step.RequiredTestEvidenceRefs...)
	}
	return compactServiceRefsV0(refs), nil
}

func operationalDirectorPlanStateClosedAfterClosureV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	closureRequest orquestacionnucleoapp.OperationalDirectorClosureRequestV0,
) error {
	planRef := continueOperationalDirectorPlanRefV0(request)
	if planRef == "" || ports.OperationalPlanStateStore == nil || ports.OperationalPlanStateWriter == nil {
		return nil
	}
	state, err := ports.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, request.RunRef, planRef)
	if err != nil {
		return err
	}
	if state.Status == orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 {
		return nil
	}
	reason := "operational-closure-succeeded"
	closureRefs := operationalDirectorPlanStateClosureEvidenceRefsV0(closureRequest)
	nextSteps := make([]orquestacionnucleoapp.OperationalDirectorPlanStepStateV0, 0, len(state.Steps))
	for _, step := range state.Steps {
		nextStep := step
		if step.StepID == state.ActiveStepID ||
			(state.ActiveStepID == "" && step.Kind == orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0) {
			nextStep.Status = orquestadirectoroperativo.OperationalDirectorStepClosedV0
			nextStep.RequiredTestEvidenceRefs = compactServiceRefsV0(append(
				nextStep.RequiredTestEvidenceRefs,
				closureRequest.RequiredTestEvidenceRefs...,
			))
			nextStep.EvidenceRefs = compactServiceRefsV0(append(nextStep.EvidenceRefs, closureRefs...))
			nextStep.BlockerRefs = nil
			nextStep.Reason = reason
			if state.ActiveStepID == "" {
				state.ActiveStepID = step.StepID
			}
		}
		nextSteps = append(nextSteps, nextStep)
	}
	state.Status = orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0
	state.PendingAgentRefs = nil
	state.BlockerRefs = nil
	state.EvidenceRefs = compactServiceRefsV0(append(
		append(state.EvidenceRefs, closureRefs...),
		"evidence-ref-app-director-operational-plan-state-closure-succeeded-v0",
	))
	state.ClosureReason = reason
	state.Steps = nextSteps
	state.UpdatedAt = request.OccurredAt
	next, err := orquestacionnucleoapp.NewOperationalDirectorPlanStateV0(state)
	if err != nil {
		return err
	}
	return ports.OperationalPlanStateWriter.SaveOperationalDirectorPlanStateV0(ctx, next)
}

func operationalDirectorPlanStateBlockedAfterClosureV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	reason string,
	issues []orquestacionnucleoapp.ErrorV0,
	extraBlockerRefs ...string,
) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "operational-closure-blocked"
	}
	planRef := continueOperationalDirectorPlanRefV0(request)
	if planRef == "" || ports.OperationalPlanStateStore == nil || ports.OperationalPlanStateWriter == nil {
		return nil
	}
	state, err := ports.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, request.RunRef, planRef)
	if err != nil {
		return err
	}
	if state.Status == orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 {
		return nil
	}
	blockerRefs := operationalDirectorPlanStateClosureBlockerRefsV0(reason, issues)
	blockerRefs = compactServiceRefsV0(append(blockerRefs, extraBlockerRefs...))
	nextSteps := make([]orquestacionnucleoapp.OperationalDirectorPlanStepStateV0, 0, len(state.Steps))
	for _, step := range state.Steps {
		nextStep := step
		if step.StepID == state.ActiveStepID ||
			(state.ActiveStepID == "" && step.Kind == orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0) {
			nextStep.Status = orquestadirectoroperativo.OperationalDirectorStepBlockedV0
			nextStep.BlockerRefs = compactServiceRefsV0(append(nextStep.BlockerRefs, blockerRefs...))
			nextStep.Reason = reason
			if state.ActiveStepID == "" {
				state.ActiveStepID = step.StepID
			}
		}
		nextSteps = append(nextSteps, nextStep)
	}
	state.Status = orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0
	state.BlockerRefs = compactServiceRefsV0(append(state.BlockerRefs, blockerRefs...))
	state.EvidenceRefs = compactServiceRefsV0(append(
		append(state.EvidenceRefs, extraBlockerRefs...),
		"evidence-ref-app-director-operational-plan-state-closure-blocked-v0",
	))
	state.ClosureReason = reason
	state.Steps = nextSteps
	state.UpdatedAt = request.OccurredAt
	next, err := orquestacionnucleoapp.NewOperationalDirectorPlanStateV0(state)
	if err != nil {
		return err
	}
	return ports.OperationalPlanStateWriter.SaveOperationalDirectorPlanStateV0(ctx, next)
}

func operationalDirectorPlanStateClosureEvidenceRefsV0(
	request orquestacionnucleoapp.OperationalDirectorClosureRequestV0,
) []string {
	refs := []string{
		request.DeliveryRef,
		request.AcceptedReviewRef,
		request.ValidationRef,
		request.ClosureRef,
	}
	refs = append(refs, request.RequiredTestEvidenceRefs...)
	refs = append(refs, request.EvidenceRefs...)
	return compactServiceRefsV0(refs)
}

func operationalDirectorPlanStateClosureBlockerRefsV0(
	reason string,
	issues []orquestacionnucleoapp.ErrorV0,
) []string {
	refs := []string{reason}
	for _, issue := range issues {
		if field := strings.TrimSpace(issue.Field); field != "" {
			refs = append(refs, field)
		}
		if code := strings.TrimSpace(issue.Code); code != "" {
			refs = append(refs, code)
		}
	}
	return compactServiceRefsV0(refs)
}

func appDirectorClosureRequestWithDefaultsV0(
	request ContinueAppDirectorRequestV0,
	closureRequest orquestacionnucleoapp.OperationalDirectorClosureRequestV0,
) orquestacionnucleoapp.OperationalDirectorClosureRequestV0 {
	if closureRequest.RunRef == "" {
		closureRequest.RunRef = request.RunRef
	}
	if closureRequest.OccurredAt == "" {
		closureRequest.OccurredAt = request.OccurredAt
	}
	if closureRequest.CorrelationID == "" {
		closureRequest.CorrelationID = request.CorrelationID
	}
	if closureRequest.RequestedBy == "" {
		closureRequest.RequestedBy = request.RequestedBy
	}
	return closureRequest
}
