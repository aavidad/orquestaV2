package orquestaappcodexstack

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
)

func (executor CodexStackAutoprogrammingPrepareRunExecutorV0) freshAttemptForStaleAutoprogrammingRunV0(
	ctx context.Context,
	input orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0,
) orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0 {
	if executor.Stack == nil || executor.Stack.Stores.RunStore == nil {
		return input
	}
	requestRef := firstNonEmptyAutoprogrammingStackV0(
		input.AutoprogrammingRequest.RequestRef,
		input.RequestID,
		input.CorrelationID,
	)
	if strings.TrimSpace(requestRef) == "" {
		return input
	}
	run, err := executor.Stack.Stores.RunStore.LoadRunV0(ctx, requestRef)
	if err != nil {
		return input
	}
	if executor.autoprogrammingPrepareRunHasRegisteredLiveOrAmbiguousProcessV0(ctx, run) {
		return input
	}
	if !autoprogrammingPrepareRunNeedsFreshAttemptWithRuntimeV0(run, executor.RuntimeWorkDir) &&
		!executor.autoprogrammingPrepareRunControlNeedsFreshAttemptV0(ctx, run) {
		return input
	}
	_ = executor.markStaleAutoprogrammingQueueCandidateV0(ctx, run)
	nextAttempt := autoprogrammingPrepareRetryAttemptV0(requestRef) + 1
	nextRef := autoprogrammingPrepareRetryRefV0(requestRef, input.OccurredAt, stackNowV0(executor.Clock))
	input.RequestID = nextRef
	input.AutoprogrammingRequest.RequestRef = nextRef
	input.AutoprogrammingRequest = reframeAutoprogrammingRequestAfterRepeatedRetriesV0(
		input.AutoprogrammingRequest,
		requestRef,
		nextAttempt,
	)
	if strings.TrimSpace(input.CorrelationID) == "" ||
		strings.TrimSpace(input.CorrelationID) == "corr-"+requestRef ||
		strings.TrimSpace(input.CorrelationID) == requestRef {
		input.CorrelationID = "corr-" + nextRef
	}
	return input
}

func (executor CodexStackAutoprogrammingPrepareRunExecutorV0) autoprogrammingPrepareRunControlNeedsFreshAttemptV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
) bool {
	if executor.Stack == nil || executor.Stack.Stores.RunControl == nil ||
		strings.TrimSpace(run.RunID) == "" ||
		run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		return false
	}
	state, err := executor.Stack.Stores.RunControl.ReadRunControlStateV0(
		ctx,
		orquestaruncontrol.RunControlReadRequestV0{RunRef: run.RunID},
	)
	if err != nil || !orquestaruncontrol.IsTerminalRunControlStatusV0(state.Status) {
		return false
	}
	return autoprogrammingPrepareRunHasFailureSignalV0(run) ||
		len(AutoprogrammingRunPendingAgentRefsV0(run)) == 0
}

func (executor CodexStackAutoprogrammingPrepareRunExecutorV0) autoprogrammingPrepareRunHasRegisteredLiveOrAmbiguousProcessV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
) bool {
	if executor.Stack == nil || strings.TrimSpace(run.RunID) == "" {
		return false
	}
	liveness, err := executor.Stack.queuedRunningStaleProcessLivenessV0(ctx, run.RunID)
	if err != nil {
		return true
	}
	return liveness.RecordCount > 0 && (!liveness.Verifiable || liveness.Live)
}

func autoprogrammingPrepareRunNeedsFreshAttemptWithRuntimeV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	runtimeWorkDir string,
) bool {
	return AutoprogrammingRunNeedsFreshAttemptWithRuntimeV0(run, runtimeWorkDir)
}

func AutoprogrammingRunNeedsFreshAttemptV0(run orquestacoreworkflow.OrchestrationRunV0) bool {
	if strings.TrimSpace(run.RunID) == "" ||
		run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		autoprogrammingPrepareRunIsBacklogScannerV0(run.RunID) ||
		!autoprogrammingPrepareRunHasFailureSignalV0(run) {
		return false
	}
	return !autoprogrammingPrepareRunHasLivePendingAgentV0(run)
}

func AutoprogrammingRunNeedsFreshAttemptWithRuntimeV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	runtimeWorkDir string,
) bool {
	if autoprogrammingPrepareRunIsBacklogScannerV0(run.RunID) {
		return false
	}
	if AutoprogrammingRunNeedsFreshAttemptV0(run) {
		return true
	}
	pendingAgents := AutoprogrammingRunPendingAgentRefsV0(run)
	return AutoprogrammingRunHasFreshAttemptFailureSignalV0(run) &&
		len(pendingAgents) > 0 &&
		!autoprogrammingRunHasRuntimeDirForAnyAgentV0(runtimeWorkDir, run.RunID, pendingAgents)
}

func autoprogrammingPrepareRunIsBacklogScannerV0(runRef string) bool {
	runRef = strings.TrimSpace(runRef)
	if retryIndex := strings.Index(runRef, "-retry-"); retryIndex > 0 {
		runRef = runRef[:retryIndex]
	}
	return strings.HasPrefix(runRef, "request-ref-autoprogramming-backlog-scanner-")
}

func AutoprogrammingRunHasFreshAttemptFailureSignalV0(run orquestacoreworkflow.OrchestrationRunV0) bool {
	return autoprogrammingPrepareRunHasFailureSignalV0(run)
}

func AutoprogrammingRunPendingAgentRefsV0(run orquestacoreworkflow.OrchestrationRunV0) []string {
	delivered := autoprogrammingPrepareStringSetV0(run.DeliveredAgents)
	failed := autoprogrammingPrepareStringSetV0(run.FailedAgents)
	lost := autoprogrammingPrepareStringSetV0(run.LostAgents)
	stopped := autoprogrammingPrepareStringSetV0(run.StoppedAgents)
	confirmedStopped := autoprogrammingPrepareStringSetV0(run.ConfirmedStoppedAgents)
	out := make([]string, 0)
	for _, agentRef := range compactStringsV0(run.StartedAgents) {
		if delivered[agentRef] || failed[agentRef] || lost[agentRef] ||
			stopped[agentRef] || confirmedStopped[agentRef] {
			continue
		}
		out = append(out, agentRef)
	}
	return out
}

func autoprogrammingPrepareRunHasFailureSignalV0(run orquestacoreworkflow.OrchestrationRunV0) bool {
	return len(compactStringsV0(run.FailedAgents)) > 0 ||
		len(compactStringsV0(run.LostAgents)) > 0 ||
		autoprogrammingPrepareRunHasTerminalAssessmentV0(run)
}

func autoprogrammingPrepareRunHasTerminalAssessmentV0(run orquestacoreworkflow.OrchestrationRunV0) bool {
	for _, raw := range compactStringsV0(run.AgentAssessments) {
		projection, ok := orquestacoreworkflow.ParseAgentAssessmentProjectionV0(raw)
		if !ok {
			continue
		}
		if projection.Action == orquestacoreworkflow.AgentAssessmentActionStopAgentV0 {
			return true
		}
		switch projection.Verdict {
		case orquestacoreworkflow.AgentAssessmentVerdictGarbageV0,
			orquestacoreworkflow.AgentAssessmentVerdictLoopDetectedV0,
			orquestacoreworkflow.AgentAssessmentVerdictCapacityLimitedV0,
			orquestacoreworkflow.AgentAssessmentVerdictTimeoutV0:
			return true
		}
	}
	return false
}

func autoprogrammingPrepareRunHasLivePendingAgentV0(run orquestacoreworkflow.OrchestrationRunV0) bool {
	return len(AutoprogrammingRunPendingAgentRefsV0(run)) > 0
}

func autoprogrammingRunHasRuntimeDirForAnyAgentV0(
	runtimeWorkDir string,
	runRef string,
	agentRefs []string,
) bool {
	runtimeWorkDir = strings.TrimSpace(runtimeWorkDir)
	runRef = strings.TrimSpace(runRef)
	if runtimeWorkDir == "" || runRef == "" {
		return false
	}
	for _, agentRef := range compactStringsV0(agentRefs) {
		info, err := os.Stat(filepath.Join(runtimeWorkDir, runRef, agentRef))
		if err == nil && info.IsDir() {
			return true
		}
	}
	return false
}

func autoprogrammingPrepareStringSetV0(values []string) map[string]bool {
	out := map[string]bool{}
	for _, value := range compactStringsV0(values) {
		out[value] = true
	}
	return out
}
