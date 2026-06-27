package main

import (
	"context"
	"strings"

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
		return planner.degradedBacklogPlanResultV0(base, request, err.Error()), nil
	}
	planner.syncBacklogSectionStateV0(sections)
	excluded := idleSelfImprovementExcludedRequestRefsV0(request)
	knownAttempts := idleSelfImprovementKnownAttemptsByBaseRefV0(request)
	sections, canonicalCollisions, blockedCanonicalRefs := idleSelfImprovementCanonicalizeBacklogSectionsV0(sections, excluded)
	sections, preflightCollisions, blockedPreflightRefs := idleSelfImprovementCanonicalPreflightBacklogSectionsV0(sections, excluded)
	sections, proposalCollisions, blockedProposalRefs := idleSelfImprovementDedupeBacklogProposalsV0(sections, excluded)
	canonicalPreflightBlocked := idleSelfImprovementBacklogCanonicalPreflightBlockedV0(preflightCollisions)
	idleSelfImprovementPrioritizeBacklogSectionsV0(sections)
	if len(sections) == 0 {
		return planner.emptyBacklogSectionsPlanResultV0(base, request, excluded), nil
	}
	completedRequestRefs, ackEvidenceRefs, ackCollisions := planner.completedBacklogRequestRefsV0()
	idleSelfImprovementMarkReconciledPendingSectionsClosedV0(sections, completedRequestRefs, knownAttempts)
	idleSelfImprovementDropExplicitPendingRuntimeCompletionsV0(sections, completedRequestRefs)
	idleSelfImprovementDropExplicitPendingKnownExclusionsV0(sections, excluded)
	completedSections := idleSelfImprovementCompletedBacklogSectionsV0(sections, completedRequestRefs)
	for ref := range completedRequestRefs {
		excluded[ref] = true
	}
	taskIDSections := planner.backlogExecutableTaskIDSectionsV0()
	collisions := planner.backlogPlanCollisionsV0(ackCollisions, canonicalCollisions,
		preflightCollisions, proposalCollisions, sections, excluded, taskIDSections)
	if canonicalPreflightBlocked {
		return orquestaserver.IdleSelfImprovementPlanResultV0{
			Requests: []orquestaserver.IdleSelfImprovementRequestV0{},
			EvidenceRefs: compactServerStackStringsV0(append([]string{
				"evidence-ref-autoprogramming-backlog-scanner-canonical-preflight",
			}, ackEvidenceRefs...)),
			Collisions: collisions,
			Message:    "backlog_scanner_canonical_preflight_required",
		}, nil
	}
	if collision, ok := idleSelfImprovementAmbiguousTaskIDCollisionV0(request, taskIDSections); ok {
		return idleSelfImprovementAmbiguousTaskIDPlanResultV0(ackEvidenceRefs, collisions, collision), nil
	}
	if scanner, ok := planner.backlogACKScanRecoveryRequestV0(base, request, collisions); ok {
		return scanner, nil
	}
	requests := make([]orquestaserver.IdleSelfImprovementRequestV0, 0, maxRequests)
	for _, section := range sections {
		if section.Completed {
			continue
		}
		if !idleSelfImprovementBacklogCanRunWithDependenciesV0(section, completedSections) {
			continue
		}
		if blockedCanonicalRefs[idleSelfImprovementCanonicalSectionRefV0(section.Ref)] {
			continue
		}
		if blockedPreflightRefs[idleSelfImprovementCanonicalSectionRefV0(section.Ref)] {
			continue
		}
		if blockedProposalRefs[idleSelfImprovementCanonicalSectionRefV0(section.Ref)] {
			continue
		}
		next := idleSelfImprovementRequestForBacklogSectionV0(base, section)
		if section.NeedsDocumentReview {
			next = idleSelfImprovementDocumentReviewRequestForBacklogSectionV0(base, section)
		}
		next = idleSelfImprovementPendingKnownRequestV0(next, section, knownAttempts)
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
	if len(requests) < maxRequests &&
		strings.TrimSpace(request.Trigger) == "capacity_free" &&
		!idleSelfImprovementHasExecutableBacklogRequestsV0(requests) &&
		!canonicalPreflightBlocked {
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
	if !scannerAdded && len(requests) < maxRequests &&
		!canonicalPreflightBlocked &&
		idleSelfImprovementShouldAddScannerRequestV0(request, requests) {
		scanner := idleSelfImprovementBacklogScannerRequestV0(base, request)
		if len(requests) == 0 {
			scanner = idleSelfImprovementBacklogFallbackRequestV0(base, request, "backlog_sin_tareas_pendientes_detectadas")
		}
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
		fallback := idleSelfImprovementBacklogFallbackRequestV0(base, request, "backlog_sin_tareas_pendientes_detectadas")
		fallback = planner.withBacklogScannerMergeLeaseV0(fallback)
		requests = append(requests, fallback)
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
