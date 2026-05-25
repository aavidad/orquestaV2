package main

import (
	"context"
	"strings"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestaserver "orquesta/modulos/orquesta-server"
)

const idleSelfImprovementBacklogDocRelV0 = "docs/autoprogramacion_orquesta_pendientes_2026-05-23.md"

func (supervisor serverStackSupervisorV0) PlanIdleSelfImprovementV0(
	ctx context.Context,
	request orquestaserver.IdleSelfImprovementPlanRequestV0,
) (orquestaserver.IdleSelfImprovementPlanResultV0, error) {
	if err := ctx.Err(); err != nil {
		return orquestaserver.IdleSelfImprovementPlanResultV0{}, err
	}
	request = supervisor.withClosedKnownIdleSelfImprovementRefsV0(ctx, request)
	request = supervisor.withoutRetryableKnownIdleSelfImprovementRefsV0(ctx, request)
	return (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: supervisor.projectWorkDir}).PlanV0(request)
}

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

type idleSelfImprovementBacklogPlannerV0 struct{ ProjectWorkDir string }

func (planner idleSelfImprovementBacklogPlannerV0) PlanV0(
	request orquestaserver.IdleSelfImprovementPlanRequestV0,
) (orquestaserver.IdleSelfImprovementPlanResultV0, error) {
	base := request.BaseRequest
	maxRequests := request.MaxRequests
	if maxRequests <= 0 {
		maxRequests = orquestaserver.DefaultIdleSelfImprovementMaxRequestsV0
	}
	sections, err := planner.loadBacklogSectionsV0()
	if err != nil {
		return orquestaserver.IdleSelfImprovementPlanResultV0{
			Requests: []orquestaserver.IdleSelfImprovementRequestV0{
				idleSelfImprovementBacklogFallbackRequestV0(base, err.Error()),
			},
			EvidenceRefs: []string{"evidence-ref-autoprogramming-backlog-planner-fallback"},
			Message:      err.Error(),
		}, nil
	}
	planner.syncBacklogSectionStateV0(sections)
	idleSelfImprovementPrioritizeBacklogSectionsV0(sections)
	excluded := idleSelfImprovementExcludedRequestRefsV0(request)
	completedRequestRefs, ackEvidenceRefs, ackCollisions := planner.completedBacklogRequestRefsV0()
	completedSections := idleSelfImprovementCompletedBacklogSectionsV0(sections, completedRequestRefs)
	for ref := range completedRequestRefs {
		excluded[ref] = true
	}
	collisions := compactBacklogScanCollisionsV0(append(
		ackCollisions,
		idleSelfImprovementBacklogSectionCollisionsV0(sections, excluded)...,
	))
	requests := make([]orquestaserver.IdleSelfImprovementRequestV0, 0, maxRequests)
	for _, section := range sections {
		if section.Completed {
			continue
		}
		if !idleSelfImprovementBacklogCanRunWithDependenciesV0(section, completedSections) {
			continue
		}
		next := idleSelfImprovementRequestForBacklogSectionV0(base, section)
		if section.NeedsDocumentReview {
			next = idleSelfImprovementDocumentReviewRequestForBacklogSectionV0(base, section)
		}
		next = planner.withBacklogSectionMergeLeaseV0(next, section)
		if excluded[next.RequestRef] {
			continue
		}
		requests = append(requests, next)
		if len(requests) >= maxRequests {
			break
		}
	}
	scannerAdded := false
	if len(requests) < maxRequests && strings.TrimSpace(request.Trigger) == "capacity_free" {
		scanner := idleSelfImprovementBacklogScannerRequestV0(base, request)
		scanner = planner.withBacklogScannerMergeLeaseV0(scanner)
		if !excluded[scanner.RequestRef] {
			requests = append(requests, scanner)
			scannerAdded = true
		}
	}
	if len(requests) == 0 && idleSelfImprovementHasKnownBacklogWorkV0(excluded) {
		return orquestaserver.IdleSelfImprovementPlanResultV0{
			Requests: []orquestaserver.IdleSelfImprovementRequestV0{},
			EvidenceRefs: compactServerStackStringsV0(append([]string{
				"evidence-ref-autoprogramming-backlog-known-work",
			}, ackEvidenceRefs...)),
			Collisions: collisions,
			Message:    "backlog_tareas_ya_visibles_en_cola",
		}, nil
	}
	if !scannerAdded && len(requests) < maxRequests && idleSelfImprovementShouldAddScannerRequestV0(request, requests) {
		scanner := idleSelfImprovementBacklogScannerRequestV0(base, request)
		scanner = planner.withBacklogScannerMergeLeaseV0(scanner)
		if !excluded[scanner.RequestRef] {
			requests = append(requests, scanner)
		}
	}
	if len(requests) == 0 {
		if len(excluded) > 0 {
			return orquestaserver.IdleSelfImprovementPlanResultV0{
				Requests: []orquestaserver.IdleSelfImprovementRequestV0{},
				EvidenceRefs: compactServerStackStringsV0(append([]string{
					"evidence-ref-autoprogramming-backlog-known-work",
				}, ackEvidenceRefs...)),
				Collisions: collisions,
				Message:    "backlog_tareas_ya_visibles_en_cola",
			}, nil
		}
		requests = append(requests, idleSelfImprovementBacklogFallbackRequestV0(base, "backlog_sin_tareas_pendientes_detectadas"))
	}
	return orquestaserver.IdleSelfImprovementPlanResultV0{
		Requests: requests,
		EvidenceRefs: compactServerStackStringsV0(append([]string{
			"evidence-ref-autoprogramming-backlog-doc",
		}, ackEvidenceRefs...)),
		Collisions: collisions,
		Message:    "backlog_autoprogramming_planned",
	}, nil
}

func idleSelfImprovementExcludedRequestRefsV0(
	request orquestaserver.IdleSelfImprovementPlanRequestV0,
) map[string]bool {
	excluded := map[string]bool{}
	for _, value := range request.KnownRequestRefs {
		if ref := idleSelfImprovementNormalizeQueuedRequestRefV0(value); ref != "" {
			excluded[ref] = true
		}
	}
	for _, value := range request.KnownRunRefs {
		if ref := idleSelfImprovementNormalizeQueuedRequestRefV0(value); ref != "" {
			excluded[ref] = true
		}
	}
	return excluded
}

func idleSelfImprovementNormalizeQueuedRequestRefV0(value string) string {
	value = strings.TrimSpace(value)
	if retryIndex := strings.Index(value, "-retry-"); retryIndex > 0 {
		value = value[:retryIndex]
	}
	return value
}

func idleSelfImprovementHasKnownBacklogWorkV0(excluded map[string]bool) bool {
	for ref := range excluded {
		if strings.HasPrefix(ref, "request-ref-autoprogramming-backlog-") {
			return true
		}
	}
	return false
}

func idleSelfImprovementShouldAddScannerRequestV0(
	request orquestaserver.IdleSelfImprovementPlanRequestV0,
	requests []orquestaserver.IdleSelfImprovementRequestV0,
) bool {
	if strings.TrimSpace(request.Trigger) == "capacity_free" {
		return true
	}
	return len(requests) == 0
}
