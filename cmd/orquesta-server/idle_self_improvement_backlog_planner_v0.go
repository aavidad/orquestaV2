package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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
	if len(requests) < maxRequests && idleSelfImprovementShouldAddScannerRequestV0(request, requests) {
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

func (planner idleSelfImprovementBacklogPlannerV0) loadBacklogSectionsV0() ([]idleSelfImprovementBacklogSectionV0, error) {
	projectDir := strings.TrimSpace(planner.ProjectWorkDir)
	if projectDir == "" {
		return nil, errors.New("project_work_dir_required")
	}
	body, err := os.ReadFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0))
	if err != nil {
		return nil, errors.New("autoprogramming_backlog_doc_unavailable")
	}
	return parseIdleSelfImprovementBacklogSectionsV0(string(body)), nil
}
