package main

import (
	"context"
	"strings"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func (supervisor serverStackSupervisorV0) RetryableIdleSelfImprovementRunRefsV0(
	ctx context.Context,
	request orquestaserver.IdleSelfImprovementRunFreshnessRequestV0,
) (orquestaserver.IdleSelfImprovementRunFreshnessResultV0, error) {
	if supervisor.stack == nil || supervisor.stack.Stores.RunStore == nil {
		return orquestaserver.IdleSelfImprovementRunFreshnessResultV0{}, nil
	}
	result := orquestaserver.IdleSelfImprovementRunFreshnessResultV0{}
	seen := map[string]bool{}
	for _, runRef := range request.KnownRunRefs {
		trimmed := strings.TrimSpace(runRef)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		run, err := supervisor.stack.Stores.RunStore.LoadRunV0(ctx, trimmed)
		if err != nil {
			continue
		}
		if !orquestaappcodexstack.AutoprogrammingRunNeedsFreshAttemptWithRuntimeV0(
			run,
			firstNonEmptyServerStackV0(supervisor.runtimeWorkDir, supervisor.stack.CodexRuntimeWorkDir),
		) {
			continue
		}
		result.RetryableRunRefs = append(result.RetryableRunRefs, trimmed)
		result.RetryableRequestRefs = append(result.RetryableRequestRefs, idleSelfImprovementNormalizeQueuedRequestRefV0(trimmed))
	}
	if len(result.RetryableRunRefs) > 0 {
		result.EvidenceRefs = []string{"evidence-ref-idle-self-improvement-retryable-runs-filtered"}
	}
	return result, nil
}

func (supervisor serverStackSupervisorV0) withClosedKnownIdleSelfImprovementRefsV0(
	ctx context.Context,
	request orquestaserver.IdleSelfImprovementPlanRequestV0,
) orquestaserver.IdleSelfImprovementPlanRequestV0 {
	if supervisor.stack == nil || supervisor.stack.Stores.RunQueue == nil {
		return request
	}
	queueRef := strings.TrimSpace(supervisor.stack.RunQueue.QueueRef)
	if queueRef == "" {
		queueRef = orquestaappcodexstack.DefaultRunQueueRefV0
	}
	candidates, err := supervisor.stack.Stores.RunQueue.ListRunSchedulingCandidatesV0(ctx, orquestarunqueue.RunQueueReadRequestV0{
		QueueRef:             queueRef,
		IncludeNonExecutable: true,
	})
	if err != nil {
		return request
	}
	for _, candidate := range candidates {
		runRef := strings.TrimSpace(candidate.RunRef)
		if runRef == "" || !strings.HasPrefix(idleSelfImprovementNormalizeQueuedRequestRefV0(runRef), "request-ref-autoprogramming-backlog-") {
			continue
		}
		if strings.TrimSpace(candidate.Status) != orquestarunqueue.RunStatusClosedV0 &&
			!supervisor.idleSelfImprovementRunStoreStatusIsV0(ctx, runRef, orquestacoreworkflow.OrchestrationRunStatusClosedV0) {
			continue
		}
		request.KnownRunRefs = append(request.KnownRunRefs, runRef)
		request.KnownRequestRefs = append(request.KnownRequestRefs, idleSelfImprovementNormalizeQueuedRequestRefV0(runRef))
	}
	request.KnownRunRefs = compactServerStackStringsV0(request.KnownRunRefs)
	request.KnownRequestRefs = compactServerStackStringsV0(request.KnownRequestRefs)
	return request
}

func (supervisor serverStackSupervisorV0) idleSelfImprovementRunStoreStatusIsV0(
	ctx context.Context,
	runRef string,
	status orquestacoreworkflow.OrchestrationRunStatusV0,
) bool {
	if supervisor.stack == nil || supervisor.stack.Stores.RunStore == nil || strings.TrimSpace(runRef) == "" {
		return false
	}
	run, err := supervisor.stack.Stores.RunStore.LoadRunV0(ctx, strings.TrimSpace(runRef))
	return err == nil && run.Status == status
}

func (supervisor serverStackSupervisorV0) withoutRetryableKnownIdleSelfImprovementRefsV0(
	ctx context.Context,
	request orquestaserver.IdleSelfImprovementPlanRequestV0,
) orquestaserver.IdleSelfImprovementPlanRequestV0 {
	if supervisor.stack == nil || supervisor.stack.Stores.RunStore == nil {
		return request
	}
	retryableRequestRefs := map[string]bool{}
	runRefs := make([]string, 0, len(request.KnownRunRefs))
	for _, runRef := range request.KnownRunRefs {
		trimmed := strings.TrimSpace(runRef)
		if trimmed == "" {
			continue
		}
		if _, controlBlocked := supervisor.idleSelfImprovementPreparedRunControlBlocksSchedulingV0(ctx, trimmed); controlBlocked {
			runRefs = append(runRefs, trimmed)
			continue
		}
		run, err := supervisor.stack.Stores.RunStore.LoadRunV0(ctx, trimmed)
		if err == nil && orquestaappcodexstack.AutoprogrammingRunNeedsFreshAttemptWithRuntimeV0(
			run,
			firstNonEmptyServerStackV0(supervisor.runtimeWorkDir, supervisor.stack.CodexRuntimeWorkDir),
		) {
			retryableRequestRefs[idleSelfImprovementNormalizeQueuedRequestRefV0(trimmed)] = true
			continue
		}
		runRefs = append(runRefs, trimmed)
	}
	requestRefs := make([]string, 0, len(request.KnownRequestRefs))
	for _, requestRef := range request.KnownRequestRefs {
		trimmed := strings.TrimSpace(requestRef)
		if trimmed == "" || retryableRequestRefs[idleSelfImprovementNormalizeQueuedRequestRefV0(trimmed)] {
			continue
		}
		requestRefs = append(requestRefs, trimmed)
	}
	request.KnownRunRefs = compactServerStackStringsV0(runRefs)
	request.KnownRequestRefs = compactServerStackStringsV0(requestRefs)
	return request
}
