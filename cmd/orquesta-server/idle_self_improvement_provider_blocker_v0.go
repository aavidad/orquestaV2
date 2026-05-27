package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestaserver "orquesta/modulos/orquesta-server"
)

const (
	idleSelfImprovementProviderAuthBlockedReasonV0     = "provider_auth_blocked"
	idleSelfImprovementProviderAuthRecoveredReasonV0   = "provider_auth_recovered"
	idleSelfImprovementProviderAuthRecoveredEvidenceV0 = "evidence-ref-provider-auth-recovered"
	idleSelfImprovementProviderAuthRecoveryActionV0    = "restore_provider_credentials_then_resume_run"
)

type idleSelfImprovementRunDocumentV0 struct {
	RunRef string                                  `json:"run_ref"`
	Run    orquestacoreworkflow.OrchestrationRunV0 `json:"run"`
}

func (supervisor serverStackSupervisorV0) IdleSelfImprovementBlockersV0(
	ctx context.Context,
	request orquestaserver.IdleSelfImprovementBlockerRequestV0,
) (orquestaserver.IdleSelfImprovementBlockerResultV0, error) {
	runRefs, err := supervisor.idleSelfImprovementProviderBlockedRunRefsV0(ctx, request)
	if err != nil {
		return orquestaserver.IdleSelfImprovementBlockerResultV0{}, err
	}
	if len(runRefs) == 0 {
		return orquestaserver.IdleSelfImprovementBlockerResultV0{}, nil
	}
	return orquestaserver.IdleSelfImprovementBlockerResultV0{
		Blocked: true,
		Reason:  idleSelfImprovementProviderAuthBlockedReasonV0,
		RunRefs: limitServerStackStringsV0(runRefs, 20),
		EvidenceRefs: []string{
			"evidence-ref-auth-config-blocker",
			"evidence-ref-orquesta-provider-auth-blocker",
		},
		Message:        "provider_auth_blocked: restaurar proveedor y reanudar run antes de reintentar automejora.",
		RecoveryAction: idleSelfImprovementProviderAuthRecoveryActionV0,
		NextActions: []string{
			"restore_provider_credentials",
			"confirm_provider_available",
			"resume_run_with_provider_auth_recovered_evidence",
		},
	}, nil
}

func (supervisor serverStackSupervisorV0) idleSelfImprovementProviderBlockedRunRefsV0(
	ctx context.Context,
	request orquestaserver.IdleSelfImprovementBlockerRequestV0,
) ([]string, error) {
	stateDir := strings.TrimSpace(supervisor.stateDir)
	if stateDir == "" {
		return nil, nil
	}
	scope := idleSelfImprovementProviderBlockerScopeV0(request)
	runsDir := filepath.Join(stateDir, "orchestration-state", "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	runRefs := []string{}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		run, ok := readIdleSelfImprovementRunDocumentV0(filepath.Join(runsDir, entry.Name()))
		if !ok || !idleSelfImprovementRunHasProviderAuthBlockerV0(run) {
			continue
		}
		if len(scope) > 0 && !scope[idleSelfImprovementNormalizeQueuedRequestRefV0(run.RunID)] {
			continue
		}
		if supervisor.idleSelfImprovementProviderBlockerSuppressedV0(ctx, run.RunID) {
			continue
		}
		runRefs = append(runRefs, run.RunID)
	}
	return compactServerStackStringsV0(runRefs), nil
}

func idleSelfImprovementProviderBlockerScopeV0(
	request orquestaserver.IdleSelfImprovementBlockerRequestV0,
) map[string]bool {
	scope := map[string]bool{}
	for _, ref := range append(append([]string(nil), request.KnownRunRefs...), request.KnownRequestRefs...) {
		normalized := idleSelfImprovementNormalizeQueuedRequestRefV0(ref)
		if normalized != "" {
			scope[normalized] = true
		}
	}
	return scope
}

func (supervisor serverStackSupervisorV0) idleSelfImprovementProviderBlockerSuppressedV0(
	ctx context.Context,
	runRef string,
) bool {
	if strings.TrimSpace(runRef) == "" {
		return true
	}
	controlFound, controlSuppressed := supervisor.idleSelfImprovementProviderBlockerControlStateV0(ctx, runRef)
	queueFound, queueSuppressed := supervisor.idleSelfImprovementProviderBlockerQueueStateV0(ctx, runRef)
	if supervisor.idleSelfImprovementProviderBlockerHasLiveStoresV0() && !controlFound && !queueFound {
		return true
	}
	return controlSuppressed || queueSuppressed
}

func (supervisor serverStackSupervisorV0) idleSelfImprovementProviderBlockerHasLiveStoresV0() bool {
	return supervisor.stack != nil &&
		(supervisor.stack.Stores.RunControl != nil || supervisor.stack.Stores.RunQueue != nil)
}

func (supervisor serverStackSupervisorV0) idleSelfImprovementProviderBlockerControlStateV0(
	ctx context.Context,
	runRef string,
) (bool, bool) {
	if supervisor.stack == nil || supervisor.stack.Stores.RunControl == nil || strings.TrimSpace(runRef) == "" {
		return false, false
	}
	state, err := supervisor.stack.Stores.RunControl.ReadRunControlStateV0(ctx, orquestaruncontrol.RunControlReadRequestV0{RunRef: runRef})
	if err != nil {
		var notFound orquestaruncontrol.RunControlStateNotFoundErrorV0
		if errors.As(err, &notFound) {
			return false, false
		}
		return true, false
	}
	if idleSelfImprovementProviderAuthRecoveredControlV0(state) {
		return true, true
	}
	switch state.Status {
	case orquestaruncontrol.RunControlStatusPausedV0,
		orquestaruncontrol.RunControlStatusStopRequestedV0,
		orquestaruncontrol.RunControlStatusStoppedV0,
		orquestaruncontrol.RunControlStatusCancelRequestedV0,
		orquestaruncontrol.RunControlStatusCanceledV0:
		return true, true
	default:
		return true, false
	}
}

func idleSelfImprovementProviderAuthRecoveredControlV0(
	state orquestaruncontrol.RunControlStateV0,
) bool {
	if orquestaruncontrol.NormalizeRunControlStatusV0(state.Status) !=
		orquestaruncontrol.RunControlStatusRunningV0 {
		return false
	}
	if strings.TrimSpace(state.Meta.Reason) == idleSelfImprovementProviderAuthRecoveredReasonV0 {
		return true
	}
	for _, ref := range compactServerStackStringsV0(state.EvidenceRefs) {
		if ref == idleSelfImprovementProviderAuthRecoveredEvidenceV0 {
			return true
		}
	}
	return false
}

func (supervisor serverStackSupervisorV0) idleSelfImprovementProviderBlockerQueueStateV0(
	ctx context.Context,
	runRef string,
) (bool, bool) {
	if supervisor.stack == nil || supervisor.stack.Stores.RunQueue == nil || strings.TrimSpace(runRef) == "" {
		return false, false
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
		return true, false
	}
	for _, candidate := range candidates {
		if idleSelfImprovementNormalizeQueuedRequestRefV0(candidate.RunRef) != idleSelfImprovementNormalizeQueuedRequestRefV0(runRef) {
			continue
		}
		switch strings.TrimSpace(candidate.Status) {
		case orquestarunqueue.RunStatusPausedV0,
			orquestarunqueue.RunStatusStoppedV0,
			orquestarunqueue.RunStatusCanceledV0,
			orquestarunqueue.RunStatusClosedV0:
			return true, true
		default:
			return true, false
		}
	}
	return false, false
}

func readIdleSelfImprovementRunDocumentV0(path string) (
	orquestacoreworkflow.OrchestrationRunV0,
	bool,
) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return orquestacoreworkflow.OrchestrationRunV0{}, false
	}
	var document idleSelfImprovementRunDocumentV0
	if err := json.Unmarshal(raw, &document); err != nil {
		return orquestacoreworkflow.OrchestrationRunV0{}, false
	}
	if strings.TrimSpace(document.Run.RunID) == "" {
		return orquestacoreworkflow.OrchestrationRunV0{}, false
	}
	return document.Run, true
}

func idleSelfImprovementRunHasProviderAuthBlockerV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) bool {
	if run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		return false
	}
	if len(compactServerStackStringsV0(run.LostAgents)) == 0 &&
		len(compactServerStackStringsV0(run.DirectorQuestions)) == 0 {
		return false
	}
	for _, raw := range compactServerStackStringsV0(run.AgentAssessments) {
		projection, ok := orquestacoreworkflow.ParseAgentAssessmentProjectionV0(raw)
		if !ok {
			continue
		}
		if projection.Verdict == orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0 &&
			projection.Action == orquestacoreworkflow.AgentAssessmentActionAskDirectorV0 &&
			projection.Severity == orquestacoreworkflow.AgentAssessmentSeverityCriticalV0 {
			return true
		}
	}
	return false
}

func limitServerStackStringsV0(values []string, limit int) []string {
	values = compactServerStackStringsV0(values)
	if limit > 0 && len(values) > limit {
		return values[:limit]
	}
	return values
}
