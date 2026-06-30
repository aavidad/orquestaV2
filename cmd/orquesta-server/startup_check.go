package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestaserver "orquesta/modulos/orquesta-server"
)

const (
	startupCleanupModeOffV0        = "off"
	startupCleanupModeDiagnoseV0   = "diagnose"
	startupCleanupModeSelectiveV0  = "selective_project"
	startupCleanupModeForcedStopV0 = "forced_stop"
)

type serverStartupCheckV0 struct {
	Stack        orquestaappcodexstack.StackV0
	ServerConfig orquestaserver.ConfigV0
	Mode         string
	QueueLimit   int
	ScopeRefs    []string
}

func startupCheckFromEnvV0(
	stack orquestaappcodexstack.StackV0,
	serverConfig orquestaserver.ConfigV0,
) orquestaserver.StartupCheckPortV0 {
	mode := strings.TrimSpace(os.Getenv(envStartupCleanupModeV0))
	if mode == "" {
		mode = startupCleanupModeDiagnoseV0
	}
	mode = strings.ToLower(mode)
	if mode == startupCleanupModeOffV0 || mode == "disabled" || mode == "0" {
		return nil
	}
	return serverStartupCheckV0{
		Stack:        stack,
		ServerConfig: serverConfig,
		Mode:         mode,
		QueueLimit:   intEnvOrDefaultV0(envStartupQueueLimitV0, 0),
		ScopeRefs:    csvEnvOrDefaultV0(envStartupCleanupScopeRefsV0, nil),
	}
}

func (check serverStartupCheckV0) PrepareStartupV0(
	ctx context.Context,
	command orquestaserver.StartupCheckCommandV0,
) (orquestaserver.StartupCheckResultV0, error) {
	mode := strings.TrimSpace(check.Mode)
	if mode == "" {
		mode = startupCleanupModeDiagnoseV0
	}
	switch mode {
	case startupCleanupModeDiagnoseV0:
		return check.diagnoseStartupV0(ctx, command)
	case startupCleanupModeSelectiveV0, "selective", "project", "quarantine_project":
		return check.selectiveStartupCleanupV0(ctx, command)
	case startupCleanupModeForcedStopV0, "purge", "purge_forced":
		return check.forceStopStartupStateV0(ctx, command)
	default:
		return orquestaserver.StartupCheckResultV0{}, fmt.Errorf("startup_cleanup_mode_invalid")
	}
}

func (check serverStartupCheckV0) diagnoseStartupV0(
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
		return orquestaserver.StartupCheckResultV0{
			Status:       orquestaserver.StartupCheckStatusReadyV0,
			Ready:        true,
			Message:      startupAdoptionMessageV0("director: orquesta preparada; autodiagnostico sin runs transitorios ni cola sucia", adoption),
			EvidenceRefs: startupAdoptionEvidenceRefsV0([]string{"evidence-ref-orquesta-startup-diagnose-ready"}, adoption),
		}, nil
	}
	domainSessionSuppressed := startupCleanupHasDomainSessionSuppressionV0(cleanup)
	synced, err := check.reconcileStartupQueueCandidatesV0(ctx, command, cleanup)
	if err != nil {
		return orquestaserver.StartupCheckResultV0{}, err
	}
	if synced > 0 {
		cleanup, err = check.startupCleanupCandidatesV0(ctx)
		if err != nil {
			return orquestaserver.StartupCheckResultV0{}, err
		}
		if len(cleanup) == 0 {
			compaction, err := check.compactStartupStateFilesV0(command)
			if err != nil {
				return orquestaserver.StartupCheckResultV0{}, err
			}
			message := fmt.Sprintf("director: orquesta preparada; cola terminal reconciliada=%d", synced)
			evidenceRefs := []string{
				"evidence-ref-orquesta-startup-diagnose-ready",
				"evidence-ref-orquesta-startup-queue-reconciled",
			}
			if domainSessionSuppressed {
				message += "; " + startupIdleSelfImprovementDomainSessionSuppressedReasonV0
				evidenceRefs = append(evidenceRefs, startupIdleSelfImprovementDomainSessionEvidenceV0)
			}
			return orquestaserver.StartupCheckResultV0{
				Status: orquestaserver.StartupCheckStatusReadyV0,
				Ready:  true,
				Message: startupReadyMessageV0(startupAdoptionMessageV0(
					message,
					adoption,
				), compaction),
				EvidenceRefs:    startupReadyEvidenceRefsV0(startupAdoptionEvidenceRefsV0(evidenceRefs, adoption), compaction),
				StartupRevision: startupRevisionSummaryFromCompactionV0(compaction),
			}, nil
		}
	}
	active, queueDirty := countStartupCleanupCandidatesV0(cleanup)
	if active > 0 && queueDirty == 0 {
		return orquestaserver.StartupCheckResultV0{
			Status: orquestaserver.StartupCheckStatusReadyV0,
			Ready:  true,
			Message: startupAdoptionMessageV0(
				fmt.Sprintf("director: orquesta preparada; runs activos asumidos=%d sin purga", active),
				adoption,
			),
			EvidenceRefs: startupAdoptionEvidenceRefsV0([]string{
				"evidence-ref-orquesta-startup-diagnose-ready",
				"evidence-ref-orquesta-startup-active-runs-adopted",
			}, adoption),
		}, nil
	}
	return orquestaserver.StartupCheckResultV0{
		Status:       "startup_dirty_runs_detected",
		Ready:        false,
		Message:      fmt.Sprintf("runs transitorios activos=%d cola_desincronizada=%d; usar %s=forced_stop para purga logica", active, queueDirty, envStartupCleanupModeV0),
		EvidenceRefs: []string{"evidence-ref-orquesta-startup-dirty-runs"},
		Blockers:     startupCleanupBlockersV0(cleanup, command),
	}, nil
}

func startupCleanupBlockersV0(
	candidates []startupCandidateCleanupV0,
	command orquestaserver.StartupCheckCommandV0,
) []orquestaserver.StartupBlockerV0 {
	out := make([]orquestaserver.StartupBlockerV0, 0, len(candidates))
	for _, entry := range candidates {
		candidate := entry.Candidate
		blocker := orquestaserver.StartupBlockerV0{
			RunRef:       strings.TrimSpace(candidate.RunRef),
			AppRef:       strings.TrimSpace(candidate.AppRef),
			QueueStatus:  strings.TrimSpace(firstNonEmptyStartupStringV0(entry.QueueStatus, candidate.Status)),
			UpdatedAt:    startupPublicTimeV0(candidate.UpdatedAt),
			AgeSeconds:   startupCandidateAgeSecondsV0(candidate.UpdatedAt, command.OccurredAt),
			Active:       entry.Active,
			QueueDirty:   entry.QueueDirty,
			ProcessState: startupCleanupProcessStateV0(entry),
			Action:       startupCleanupActionV0(entry),
			EvidenceRefs: appendStartupEvidenceRefV0(candidate.EvidenceRefs, "evidence-ref-orquesta-startup-dirty-runs"),
		}
		out = append(out, blocker)
	}
	return out
}

func startupCleanupProcessStateV0(entry startupCandidateCleanupV0) string {
	if entry.Active {
		return "active_or_in_flight"
	}
	if entry.QueueDirty {
		return "no_live_process_required"
	}
	return "unknown"
}

func startupCleanupActionV0(entry startupCandidateCleanupV0) string {
	if entry.QueueDirty && !entry.Active {
		if entry.CompleteControlPending {
			return "complete_pending_control_and_reconcile_queue"
		}
		return "reconcile_queue_terminal"
	}
	if entry.Active {
		return "inspect_live_run_or_force_stop_explicit"
	}
	return "inspect_startup_state"
}

func startupPublicTimeV0(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func startupCandidateAgeSecondsV0(updatedAt time.Time, now time.Time) int64 {
	if updatedAt.IsZero() || now.IsZero() {
		return 0
	}
	age := now.Sub(updatedAt)
	if age < 0 {
		return 0
	}
	return int64(age.Seconds())
}

func startupCleanupHasDomainSessionSuppressionV0(candidates []startupCandidateCleanupV0) bool {
	for _, candidate := range candidates {
		if candidate.DomainSessionSuppressed {
			return true
		}
	}
	return false
}

func (check serverStartupCheckV0) forceStopStartupStateV0(
	ctx context.Context,
	command orquestaserver.StartupCheckCommandV0,
) (orquestaserver.StartupCheckResultV0, error) {
	cleanup, err := check.startupCleanupCandidatesV0(ctx)
	if err != nil {
		return orquestaserver.StartupCheckResultV0{}, err
	}
	if len(cleanup) == 0 {
		compaction, err := check.compactStartupStateFilesV0(command)
		if err != nil {
			return orquestaserver.StartupCheckResultV0{}, err
		}
		return orquestaserver.StartupCheckResultV0{
			Status:          orquestaserver.StartupCheckStatusReadyV0,
			Ready:           true,
			Message:         startupReadyMessageV0("director: orquesta preparada; no habia runs transitorios activos", compaction),
			EvidenceRefs:    startupReadyEvidenceRefsV0([]string{"evidence-ref-orquesta-startup-cleanup-empty"}, compaction),
			StartupRevision: startupRevisionSummaryFromCompactionV0(compaction),
		}, nil
	}
	completed := 0
	queueSynced := 0
	blocked := 0
	for _, entry := range cleanup {
		if entry.QueueDirty && !entry.Active {
			if err := check.markStartupQueueCandidateTerminalV0(ctx, command, entry.Candidate, entry.QueueStatus); err != nil {
				return orquestaserver.StartupCheckResultV0{}, err
			}
			queueSynced++
			continue
		}
		ready, err := check.forceStopStartupCandidateV0(ctx, command, entry.Candidate)
		if err != nil {
			return orquestaserver.StartupCheckResultV0{}, err
		}
		if ready {
			completed++
			continue
		}
		blocked++
	}
	if blocked == 0 {
		compaction, err := check.compactStartupStateFilesV0(command)
		if err != nil {
			return orquestaserver.StartupCheckResultV0{}, err
		}
		return orquestaserver.StartupCheckResultV0{
			Status: orquestaserver.StartupCheckStatusReadyV0,
			Ready:  true,
			Message: startupReadyMessageV0(fmt.Sprintf(
				"director: orquesta preparada; purga logica completada runs=%d cola=%d",
				completed,
				queueSynced,
			), compaction),
			EvidenceRefs:    startupReadyEvidenceRefsV0([]string{"evidence-ref-orquesta-startup-ready-after-cleanup"}, compaction),
			StartupRevision: startupRevisionSummaryFromCompactionV0(compaction),
		}, nil
	}
	return orquestaserver.StartupCheckResultV0{
		Status: "startup_waiting_drain",
		Ready:  false,
		Message: fmt.Sprintf(
			"startup no listo: runs_completados=%d cola_sincronizada=%d runs_bloqueados=%d",
			completed,
			queueSynced,
			blocked,
		),
		EvidenceRefs: []string{"evidence-ref-orquesta-startup-cleanup-pending"},
	}, nil
}

func (check serverStartupCheckV0) forceStopStartupCandidateV0(
	ctx context.Context,
	command orquestaserver.StartupCheckCommandV0,
	candidate orquestarunqueue.RunSchedulingCandidateV0,
) (bool, error) {
	runRef := strings.TrimSpace(candidate.RunRef)
	if runRef == "" {
		return true, nil
	}
	_, err := check.Stack.Stores.RunControl.StopRunV0(ctx, orquestaruncontrol.StopRunCommandV0{
		RunRef:         runRef,
		RequestedBy:    "orquesta-server-startup",
		Reason:         "purga logica de arranque antes de activar supervisor",
		Forced:         true,
		IdempotencyKey: "idem-orquesta-server-startup-stop-" + startupSafeRefPartV0(runRef),
		EvidenceRefs:   []string{"evidence-ref-orquesta-startup-forced-stop"},
	})
	if err != nil {
		return false, err
	}
	ready, err := check.startupCandidateCanBeCompletedByStartupV0(ctx, runRef)
	if err != nil {
		return false, err
	}
	if !ready {
		return false, nil
	}
	_, err = check.Stack.Stores.RunControl.CompleteRunControlV0(
		ctx,
		orquestaruncontrol.CompleteRunControlCommandV0{
			RunRef:         runRef,
			TargetStatus:   orquestaruncontrol.RunControlStatusStoppedV0,
			RequestedBy:    "orquesta-server-startup",
			Reason:         "purga logica completada sin agentes en vuelo",
			IdempotencyKey: "idem-orquesta-server-startup-complete-" + startupSafeRefPartV0(runRef),
			EvidenceRefs: []string{
				"evidence-ref-orquesta-startup-control-stopped",
			},
		},
	)
	if err != nil {
		return false, err
	}
	if err := check.markStartupQueueCandidateTerminalV0(ctx, command, candidate, orquestarunqueue.RunStatusStoppedV0); err != nil {
		return false, err
	}
	return true, nil
}

func (check serverStartupCheckV0) markStartupQueueCandidateTerminalV0(
	ctx context.Context,
	command orquestaserver.StartupCheckCommandV0,
	candidate orquestarunqueue.RunSchedulingCandidateV0,
	status string,
) error {
	writer, ok := any(check.Stack.Stores.RunQueue).(startupQueueCandidateWriterV0)
	if !ok || writer == nil {
		return nil
	}
	_, err := writer.UpsertRunSchedulingCandidateV0(
		ctx,
		check.Stack.RunQueue.QueueRef,
		terminalStartupQueueCandidateV0(candidate, command, status),
	)
	return err
}
