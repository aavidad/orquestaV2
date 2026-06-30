package main

import (
	"context"
	"fmt"
	"strings"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestaserver "orquesta/modulos/orquesta-server"
)

const (
	startupSelectiveCleanupEvidenceV0 = "evidence-ref-orquesta-startup-selective-cleanup"
)

type startupSelectiveCleanupSummaryV0 struct {
	QueueSynced int
	Quarantined int
	Skipped     int
}

func (check serverStartupCheckV0) selectiveStartupCleanupV0(
	ctx context.Context,
	command orquestaserver.StartupCheckCommandV0,
) (orquestaserver.StartupCheckResultV0, error) {
	adoption, err := check.adoptStartupLiveProcessesV0(ctx)
	if err != nil {
		return orquestaserver.StartupCheckResultV0{}, err
	}
	cleanup, err := check.startupCleanupCandidatesV0(ctx)
	if err != nil {
		return orquestaserver.StartupCheckResultV0{}, err
	}
	if len(cleanup) == 0 {
		return check.startupReadyAfterSelectiveCleanupV0(command, adoption, startupSelectiveCleanupSummaryV0{})
	}
	scopeRefs := compactStartupStringsV0(check.ScopeRefs)
	if len(scopeRefs) == 0 {
		return orquestaserver.StartupCheckResultV0{
			Status:       "startup_selective_cleanup_scope_required",
			Ready:        false,
			Message:      fmt.Sprintf("limpieza selectiva requiere %s con refs de proyecto/app/run", envStartupCleanupScopeRefsV0),
			EvidenceRefs: []string{startupSelectiveCleanupEvidenceV0},
			Blockers:     startupCleanupBlockersV0(cleanup, command),
		}, nil
	}
	summary, err := check.applySelectiveStartupCleanupV0(ctx, command, cleanup, scopeRefs)
	if err != nil {
		return orquestaserver.StartupCheckResultV0{}, err
	}
	if summary.QueueSynced+summary.Quarantined > 0 {
		cleanup, err = check.startupCleanupCandidatesV0(ctx)
		if err != nil {
			return orquestaserver.StartupCheckResultV0{}, err
		}
	}
	if len(cleanup) == 0 {
		return check.startupReadyAfterSelectiveCleanupV0(command, adoption, summary)
	}
	active, queueDirty := countStartupCleanupCandidatesV0(cleanup)
	return orquestaserver.StartupCheckResultV0{
		Status: "startup_dirty_runs_detected",
		Ready:  false,
		Message: fmt.Sprintf(
			"limpieza selectiva aplicada cola=%d cuarentena=%d omitidos=%d; pendientes activos=%d cola_desincronizada=%d; ampliar %s o usar forced_stop explicito",
			summary.QueueSynced,
			summary.Quarantined,
			summary.Skipped,
			active,
			queueDirty,
			envStartupCleanupScopeRefsV0,
		),
		EvidenceRefs: []string{startupSelectiveCleanupEvidenceV0, "evidence-ref-orquesta-startup-dirty-runs"},
		Blockers:     startupCleanupBlockersV0(cleanup, command),
	}, nil
}

func (check serverStartupCheckV0) startupReadyAfterSelectiveCleanupV0(
	command orquestaserver.StartupCheckCommandV0,
	adoption startupLiveProcessAdoptionSummaryV0,
	summary startupSelectiveCleanupSummaryV0,
) (orquestaserver.StartupCheckResultV0, error) {
	compaction, err := check.compactStartupStateFilesV0(command)
	if err != nil {
		return orquestaserver.StartupCheckResultV0{}, err
	}
	message := startupAdoptionMessageV0(fmt.Sprintf(
		"director: orquesta preparada; limpieza selectiva cola=%d cuarentena=%d omitidos=%d",
		summary.QueueSynced,
		summary.Quarantined,
		summary.Skipped,
	), adoption)
	return orquestaserver.StartupCheckResultV0{
		Status: orquestaserver.StartupCheckStatusReadyV0,
		Ready:  true,
		Message: startupReadyMessageV0(
			message,
			compaction,
		),
		EvidenceRefs: startupReadyEvidenceRefsV0(startupAdoptionEvidenceRefsV0([]string{
			"evidence-ref-orquesta-startup-diagnose-ready",
			startupSelectiveCleanupEvidenceV0,
		}, adoption), compaction),
		StartupRevision: startupRevisionSummaryFromCompactionV0(compaction),
	}, nil
}

func (check serverStartupCheckV0) applySelectiveStartupCleanupV0(
	ctx context.Context,
	command orquestaserver.StartupCheckCommandV0,
	candidates []startupCandidateCleanupV0,
	scopeRefs []string,
) (startupSelectiveCleanupSummaryV0, error) {
	summary := startupSelectiveCleanupSummaryV0{}
	for _, entry := range candidates {
		if !startupCleanupCandidateMatchesScopeV0(entry.Candidate, scopeRefs) {
			summary.Skipped++
			continue
		}
		if entry.QueueDirty && !entry.Active {
			if entry.CompleteControlPending {
				if err := check.completeStartupRunControlV0(ctx, entry); err != nil {
					return summary, err
				}
			}
			if err := check.markStartupQueueCandidateTerminalV0(ctx, command, entry.Candidate, entry.QueueStatus); err != nil {
				return summary, err
			}
			summary.QueueSynced++
			continue
		}
		if !entry.Active {
			summary.Skipped++
			continue
		}
		ready, err := check.startupCandidateCanBeCompletedByStartupV0(ctx, entry.Candidate.RunRef)
		if err != nil {
			return summary, err
		}
		if !ready {
			summary.Skipped++
			continue
		}
		if err := check.markStartupScopedCandidateStoppedV0(ctx, command, entry.Candidate); err != nil {
			return summary, err
		}
		summary.Quarantined++
	}
	return summary, nil
}

func (check serverStartupCheckV0) markStartupScopedCandidateStoppedV0(
	ctx context.Context,
	command orquestaserver.StartupCheckCommandV0,
	candidate orquestarunqueue.RunSchedulingCandidateV0,
) error {
	runRef := strings.TrimSpace(candidate.RunRef)
	if runRef == "" {
		return nil
	}
	_, err := check.Stack.Stores.RunControl.CompleteRunControlV0(
		ctx,
		orquestaruncontrol.CompleteRunControlCommandV0{
			RunRef:         runRef,
			TargetStatus:   orquestaruncontrol.RunControlStatusStoppedV0,
			RequestedBy:    "orquesta-server-startup",
			Reason:         "cuarentena selectiva de arranque sin agentes vivos pendientes",
			IdempotencyKey: "idem-orquesta-server-startup-selective-" + startupSafeRefPartV0(runRef),
			EvidenceRefs: []string{
				startupSelectiveCleanupEvidenceV0,
				"evidence-ref-orquesta-startup-control-stopped",
			},
		},
	)
	if err != nil {
		return err
	}
	candidate.EvidenceRefs = appendStartupEvidenceRefV0(candidate.EvidenceRefs, startupSelectiveCleanupEvidenceV0)
	return check.markStartupQueueCandidateTerminalV0(ctx, command, candidate, orquestarunqueue.RunStatusStoppedV0)
}

func startupCleanupCandidateMatchesScopeV0(
	candidate orquestarunqueue.RunSchedulingCandidateV0,
	scopeRefs []string,
) bool {
	values := startupCleanupCandidateScopeValuesV0(candidate)
	for _, scope := range scopeRefs {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			continue
		}
		for _, value := range values {
			if strings.EqualFold(value, scope) {
				return true
			}
		}
	}
	return false
}

func startupCleanupCandidateScopeValuesV0(
	candidate orquestarunqueue.RunSchedulingCandidateV0,
) []string {
	values := []string{
		candidate.RunRef,
		candidate.AppRef,
		candidate.FairnessGroupRef,
		candidate.ParentRunRef,
		candidate.SupersedesRunRef,
		candidate.AttemptGroup.GroupRef,
		candidate.AttemptGroup.ConsumerRef,
		candidate.AttemptGroup.ObjectiveRef,
		candidate.AttemptGroup.WorkItemRef,
	}
	values = append(values, candidate.AttemptGroup.WriteSetRefs...)
	return compactStartupStringsV0(values)
}
