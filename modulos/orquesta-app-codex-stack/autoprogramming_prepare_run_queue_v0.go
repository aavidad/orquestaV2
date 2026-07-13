package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func (executor CodexStackAutoprogrammingPrepareRunExecutorV0) enqueuePreparedRunV0(
	ctx context.Context,
	input orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0,
	result AutoprogrammingBridgeResultV0,
) error {
	if executor.QueueWriter == nil {
		return fmt.Errorf("run_queue.writer requerido")
	}
	runRef := strings.TrimSpace(result.Run.RunID)
	if runRef == "" {
		return fmt.Errorf("run_ref preparado requerido")
	}
	if terminalStatus := autoprogrammingPreparedRunTerminalQueueStatusV0(result.Run); terminalStatus != "" {
		return executor.markPreparedRunTerminalV0(ctx, input, result, terminalStatus)
	}
	if blockedStatus, blocked, err := executor.queueStatusForPreparedRunControlV0(ctx, runRef); err != nil {
		return err
	} else if blocked {
		return executor.markPreparedRunControlBlockedV0(ctx, input, result, blockedStatus)
	}
	_, err := executor.QueueWriter.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        runRef,
		QueueRef:      executor.Queue.QueueRef,
		AppRef:        autoprogrammingQueueAppRefV0(result),
		Status:        orquestarunqueue.RunStatusReadyV0,
		PriorityScore: autoprogrammingPrepareRunPriorityScoreV0(input.PriorityScore, executor.Queue.DefaultPriorityScore),
		UpdatedAt:     stackNowV0(executor.Clock),
		RequestedBy: firstNonEmptyAutoprogrammingStackV0(
			input.RequestedBy,
			result.Continue.RequestedBy,
			executor.DefaultRequestedBy,
			"orquesta-app-codex-stack-autoprogramming",
		),
		Reason:         "autoprogramming_prepare_run",
		IdempotencyKey: "idem-run-queue-autoprogramming-prepare-" + runRef,
		EvidenceRefs:   []string{"evidence-ref-autoprogramming-prepare-run-enqueued"},
	})
	if err != nil {
		return err
	}
	return executor.ensurePreparedRunQueueVisibleV0(ctx, runRef)
}

func autoprogrammingPreparedRunTerminalQueueStatusV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) string {
	if run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		return orquestarunqueue.RunStatusClosedV0
	}
	return ""
}

func (executor CodexStackAutoprogrammingPrepareRunExecutorV0) ensurePreparedRunQueueVisibleV0(
	ctx context.Context,
	runRef string,
) error {
	runRef = strings.TrimSpace(runRef)
	if runRef == "" {
		return fmt.Errorf("run_ref preparado requerido")
	}
	reader := executor.preparedRunQueueReaderV0()
	if reader == nil {
		return fmt.Errorf("autoprogramming_prepare_run_visibility_error: run_queue.reader requerido")
	}
	candidates, err := reader.ListRunSchedulingCandidatesV0(ctx, orquestarunqueue.RunQueueReadRequestV0{
		QueueRef:             executor.Queue.QueueRef,
		RunRef:               runRef,
		Limit:                1,
		IncludeNonExecutable: true,
	})
	if err != nil {
		return fmt.Errorf("autoprogramming_prepare_run_visibility_error: %w", err)
	}
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate.RunRef) == runRef {
			return nil
		}
	}
	return fmt.Errorf("autoprogramming_prepare_run_visibility_error: run_ref no visible en cola: %s", runRef)
}

func (executor CodexStackAutoprogrammingPrepareRunExecutorV0) preparedRunQueueReaderV0() orquestarunqueue.RunQueueReaderPortV0 {
	if reader, ok := executor.QueueWriter.(orquestarunqueue.RunQueueReaderPortV0); ok && reader != nil {
		return reader
	}
	if executor.Stack != nil && executor.Stack.Stores.RunQueue != nil {
		return executor.Stack.Stores.RunQueue
	}
	return nil
}

func (executor CodexStackAutoprogrammingPrepareRunExecutorV0) markPreparedRunTerminalV0(
	ctx context.Context,
	input orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0,
	result AutoprogrammingBridgeResultV0,
	status string,
) error {
	runRef := strings.TrimSpace(result.Run.RunID)
	_, err := executor.QueueWriter.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        runRef,
		QueueRef:      executor.Queue.QueueRef,
		AppRef:        autoprogrammingQueueAppRefV0(result),
		Status:        status,
		PriorityScore: 0,
		UpdatedAt:     stackNowV0(executor.Clock),
		RequestedBy: firstNonEmptyAutoprogrammingStackV0(
			input.RequestedBy,
			result.Continue.RequestedBy,
			executor.DefaultRequestedBy,
			"orquesta-app-codex-stack-autoprogramming",
		),
		Reason:         "autoprogramming_prepare_run_terminal_reconciled",
		IdempotencyKey: "idem-run-queue-autoprogramming-terminal-" + runRef,
		EvidenceRefs: []string{
			"evidence-ref-autoprogramming-prepare-run-terminal-reconciled",
			"evidence-ref-autoprogramming-run-not-requeued",
		},
	})
	return err
}

func (executor CodexStackAutoprogrammingPrepareRunExecutorV0) queueStatusForPreparedRunControlV0(
	ctx context.Context,
	runRef string,
) (string, bool, error) {
	if executor.Stack == nil || executor.Stack.Stores.RunControl == nil {
		return "", false, nil
	}
	state, err := executor.Stack.Stores.RunControl.ReadRunControlStateV0(
		ctx,
		orquestaruncontrol.RunControlReadRequestV0{RunRef: runRef},
	)
	if err != nil {
		var notFound orquestaruncontrol.RunControlStateNotFoundErrorV0
		if errorAsRunControlNotFoundV0(err, &notFound) {
			return "", false, nil
		}
		return "", false, err
	}
	switch orquestaruncontrol.NormalizeRunControlStatusV0(state.Status) {
	case orquestaruncontrol.RunControlStatusRunningV0:
		return "", false, nil
	case orquestaruncontrol.RunControlStatusPausedV0:
		return orquestarunqueue.RunStatusPausedV0, true, nil
	case orquestaruncontrol.RunControlStatusStopRequestedV0, orquestaruncontrol.RunControlStatusStoppedV0:
		return orquestarunqueue.RunStatusStoppedV0, true, nil
	case orquestaruncontrol.RunControlStatusCancelRequestedV0, orquestaruncontrol.RunControlStatusCanceledV0:
		return orquestarunqueue.RunStatusCanceledV0, true, nil
	default:
		evaluation := orquestaruncontrol.EvaluateRunControlV0(state)
		if evaluation.DispatchAllowed {
			return "", false, nil
		}
		return "", false, nil
	}
}

func (executor CodexStackAutoprogrammingPrepareRunExecutorV0) markPreparedRunControlBlockedV0(
	ctx context.Context,
	input orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0,
	result AutoprogrammingBridgeResultV0,
	status string,
) error {
	runRef := strings.TrimSpace(result.Run.RunID)
	_, err := executor.QueueWriter.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        runRef,
		QueueRef:      executor.Queue.QueueRef,
		AppRef:        autoprogrammingQueueAppRefV0(result),
		Status:        status,
		PriorityScore: 0,
		UpdatedAt:     stackNowV0(executor.Clock),
		RequestedBy: firstNonEmptyAutoprogrammingStackV0(
			input.RequestedBy,
			result.Continue.RequestedBy,
			executor.DefaultRequestedBy,
			"orquesta-app-codex-stack-autoprogramming",
		),
		Reason:         "autoprogramming_prepare_run_control_blocked",
		IdempotencyKey: "idem-run-queue-autoprogramming-control-blocked-" + runRef,
		EvidenceRefs: []string{
			"evidence-ref-autoprogramming-prepare-run-control-blocked",
			"evidence-ref-autoprogramming-run-not-requeued",
		},
	})
	return err
}

func autoprogrammingPrepareRunPriorityScoreV0(requested int, fallback int) int {
	if requested > 0 {
		return requested
	}
	return fallback
}

func autoprogrammingQueueAppRefV0(result AutoprogrammingBridgeResultV0) string {
	return firstNonEmptyAutoprogrammingStackV0(
		result.Work.ProjectRef,
		result.Run.ProjectRef,
		result.Run.AppSpecRef,
		result.Run.RunID,
	)
}

func codexStackAutoprogrammingBridgeRequestFromMCPV0(
	input orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0,
	defaultOccurredAt string,
	defaultRequestedBy string,
) (AutoprogrammingBridgeRequestV0, []orquestamcp.MCPValidationIssueV0) {
	request := input.AutoprogrammingRequest
	if strings.TrimSpace(request.RequestRef) == "" {
		request.RequestRef = firstNonEmptyAutoprogrammingStackV0(input.RequestID, input.CorrelationID)
	}
	input.AutoprogrammingRequest = request
	input.OccurredAt = firstNonEmptyAutoprogrammingStackV0(input.OccurredAt, defaultOccurredAt)
	input.CorrelationID = firstNonEmptyAutoprogrammingStackV0(input.CorrelationID, input.RequestID, request.RequestRef)
	input.RequestedBy = firstNonEmptyAutoprogrammingStackV0(input.RequestedBy, defaultRequestedBy)
	envelope, issues := orquestamcp.CanonicalMCPAutoprogrammingPrepareRunEnvelopeV0(input)
	if len(issues) != 0 {
		return AutoprogrammingBridgeRequestV0{}, issues
	}
	return AutoprogrammingBridgeRequestV0{
		PrepareRunEnvelope: &envelope,
	}, nil
}

func codexStackAutoprogrammingPrepareRunInputFromEnvelopeV0(
	input orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0,
	envelope orquestaautoprogramming.AutoprogrammingPrepareRunEnvelopeV0,
) orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0 {
	input.RequestID = envelope.RequestID
	input.CorrelationID = envelope.CorrelationID
	input.IdempotencyKey = envelope.IdempotencyKey
	input.OccurredAt = envelope.OccurredAt
	input.RequestedBy = envelope.RequestedBy
	input.DirectorExecutionMode = envelope.DirectorExecutionMode
	input.AutoprogrammingRequest = envelope.AutoprogrammingRequest
	input.MaxBursts = envelope.MaxBursts
	input.MaxStepsPerBurst = envelope.MaxStepsPerBurst
	input.MaxDispatchesPerWait = envelope.MaxDispatchesPerWait
	input.MaxCommands = envelope.MaxCommands
	input.MaxOutboxPerCycle = envelope.MaxOutboxPerCycle
	input.PriorityScore = envelope.PriorityScore
	return input
}

func codexStackAutoprogrammingPrepareRunResultMCPV0(
	input orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0,
	result AutoprogrammingBridgeResultV0,
) orquestamcp.MCPAutoprogrammingPrepareRunToolResultV0 {
	if !result.Accepted {
		return orquestamcp.NewMCPAutoprogrammingPrepareRunIssuesResultV0(
			input,
			codexStackAutoprogrammingIssuesMCPV0(result.Issues),
		)
	}
	runRef := strings.TrimSpace(result.Run.RunID)
	out := orquestamcp.MCPAutoprogrammingPrepareRunToolResultV0{
		Estado:           orquestamcp.MCPAutoprogrammingPrepareRunEstadoOKV0,
		RequestID:        firstNonEmptyAutoprogrammingStackV0(input.RequestID, result.Work.RequestRef),
		CorrelationID:    firstNonEmptyAutoprogrammingStackV0(input.CorrelationID, input.RequestID, result.Work.RequestRef),
		Accepted:         true,
		RunRef:           runRef,
		ProjectRef:       strings.TrimSpace(result.Work.ProjectRef),
		WorktreeRef:      strings.TrimSpace(result.Work.WorktreeRef),
		BranchRef:        strings.TrimSpace(result.Work.BranchRef),
		WorkflowTaskRefs: codexStackAutoprogrammingWorkflowTaskRefsMCPV0(result.Tasks),
		WaitAgentRefs:    compactStringsV0(result.WaitAgentRefs),
		GoalSpecs:        codexStackAutoprogrammingGoalSpecsMCPV0(result.Work.GoalSpecs, runRef),
		Goal:             codexStackAutoprogrammingGoalRunMCPV0(result),
		Goals:            codexStackAutoprogrammingGoalRunsMCPV0(result),
		Errores:          []orquestamcp.MCPValidationIssueV0{},
	}
	if runRef != "" {
		out.PhaseID = string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0)
	}
	if strings.TrimSpace(result.Continue.RunRef) != "" {
		out.Continue = &orquestamcp.MCPAutoprogrammingContinueRequestV0{
			RunRef:                     strings.TrimSpace(result.Continue.RunRef),
			OperationalDirectorPlanRef: strings.TrimSpace(result.Continue.OperationalDirectorPlanRef),
			OccurredAt:                 strings.TrimSpace(result.Continue.OccurredAt),
			CorrelationID:              strings.TrimSpace(result.Continue.CorrelationID),
			RequestedBy:                strings.TrimSpace(result.Continue.RequestedBy),
			MaxBursts:                  result.Continue.MaxBursts,
			MaxStepsPerBurst:           result.Continue.MaxStepsPerBurst,
			MaxDispatchesPerWait:       result.Continue.MaxDispatchesPerWait,
			WaitAgentRefs:              compactStringsV0(result.Continue.WaitAgentRefs),
			MaxCommands:                result.Continue.MaxCommands,
			MaxOutboxPerCycle:          result.Continue.MaxOutboxPerCycle,
		}
	}
	return out
}

func autoprogrammingBridgeHasPreparedLegacyRunV0(result AutoprogrammingBridgeResultV0) bool {
	return strings.TrimSpace(result.Run.RunID) != "" && !autoprogrammingBridgeResultIsGoalFirstV0(result)
}

func codexStackAutoprogrammingGoalRunMCPV0(
	result AutoprogrammingBridgeResultV0,
) *orquestamcp.MCPAutoprogrammingGoalRunV0 {
	goals := codexStackAutoprogrammingGoalRunsMCPV0(result)
	if len(goals) != 1 {
		return nil
	}
	return &goals[0]
}

func codexStackAutoprogrammingGoalRunsMCPV0(
	result AutoprogrammingBridgeResultV0,
) []orquestamcp.MCPAutoprogrammingGoalRunV0 {
	if !autoprogrammingBridgeResultIsGoalFirstV0(result) {
		return nil
	}
	if len(result.GoalStates) > 0 {
		out := make([]orquestamcp.MCPAutoprogrammingGoalRunV0, 0, len(result.GoalStates))
		for index, state := range result.GoalStates {
			var receipt *orquestagoal.GoalLaunchReceiptV0
			if index < len(result.GoalReceipts) {
				receipt = &result.GoalReceipts[index]
			}
			out = append(out, codexStackAutoprogrammingGoalRunFromStateMCPV0(state, receipt))
		}
		return out
	}
	return []orquestamcp.MCPAutoprogrammingGoalRunV0{
		codexStackAutoprogrammingGoalRunFromStateMCPV0(result.GoalState, result.GoalReceipt),
	}
}

func codexStackAutoprogrammingGoalRunFromStateMCPV0(
	state orquestagoal.GoalWorkStateV0,
	receipt *orquestagoal.GoalLaunchReceiptV0,
) orquestamcp.MCPAutoprogrammingGoalRunV0 {
	goalRef := strings.TrimSpace(state.GoalRef)
	externalGoalRef := strings.TrimSpace(state.ExternalGoalRef)
	goalStatus := strings.TrimSpace(state.Status)
	evidenceRefs := compactStringsV0(state.EvidenceRefs)
	if receipt != nil {
		goalRef = firstNonEmptyAutoprogrammingStackV0(goalRef, receipt.GoalRef)
		externalGoalRef = firstNonEmptyAutoprogrammingStackV0(externalGoalRef, receipt.ExternalGoalRef)
		goalStatus = firstNonEmptyAutoprogrammingStackV0(goalStatus, receipt.Status)
		evidenceRefs = compactStringsV0(append(evidenceRefs, receipt.EvidenceRefs...))
	}
	return orquestamcp.MCPAutoprogrammingGoalRunV0{
		DirectorExecutionMode: orquestaappdirectorservice.AppDirectorExecutionModeGoalFirstV0,
		RunRef:                strings.TrimSpace(state.RunRef),
		GoalRef:               goalRef,
		ExternalGoalRef:       externalGoalRef,
		GoalStatus:            goalStatus,
		EvidenceRefs:          evidenceRefs,
	}
}

func codexStackAutoprogrammingWorkflowTaskRefsMCPV0(
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) []string {
	refs := make([]string, 0, len(tasks))
	for _, task := range tasks {
		refs = append(refs, task.TaskID)
	}
	return compactStringsV0(refs)
}

func codexStackAutoprogrammingGoalSpecsMCPV0(
	specs []orquestagoal.GoalWorkSpecV0,
	runRef string,
) []orquestagoal.GoalWorkSpecV0 {
	out := make([]orquestagoal.GoalWorkSpecV0, 0, len(specs))
	runRef = strings.TrimSpace(runRef)
	for _, spec := range specs {
		spec = orquestagoal.NormalizeGoalWorkSpecV0(spec)
		if strings.TrimSpace(spec.RunRef) == "" {
			spec.RunRef = runRef
		}
		out = append(out, spec)
	}
	if out == nil {
		return []orquestagoal.GoalWorkSpecV0{}
	}
	return out
}

func codexStackAutoprogrammingIssuesMCPV0(
	issues []orquestaautoprogramming.AutoprogrammingRequestIssueV0,
) []orquestamcp.MCPValidationIssueV0 {
	out := make([]orquestamcp.MCPValidationIssueV0, 0, len(issues))
	for _, issue := range issues {
		out = append(out, orquestamcp.MCPValidationIssueV0{
			Code:    strings.TrimSpace(issue.Code),
			Field:   strings.TrimSpace(issue.Field),
			Message: orquestamcp.MCPAutoprogrammingPrepareRunIssueMessageV0(issue.Code, issue.Message),
		})
	}
	if out == nil {
		return []orquestamcp.MCPValidationIssueV0{}
	}
	return out
}

func firstNonEmptyAutoprogrammingStackV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
