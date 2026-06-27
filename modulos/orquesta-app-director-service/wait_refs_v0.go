package orquestaappdirectorservice

import (
	"context"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

type appDirectorWaitFilterV0 struct {
	CohortRef     string
	WaveRef       string
	ParentTaskRef string
}

type appDirectorWaitStateMetaV0 struct {
	OccurredAt       string
	CorrelationID    string
	EvidenceRefs     []string
	MaxExternalWaits int
}

type appDirectorWaitResolutionV0 struct {
	AgentRefs    []string
	ScopeApplied bool
	Snapshot     orquestacionnucleoapp.WorkflowTaskWaitSnapshotV0
}

func appDirectorResolvedWaitV0(
	ctx context.Context,
	runRef string,
	explicitRefs []string,
	filter appDirectorWaitFilterV0,
	ports StartAppDirectorPortsV0,
	meta appDirectorWaitStateMetaV0,
) (appDirectorWaitResolutionV0, error) {
	explicitRefs = compactServiceRefsV0(explicitRefs)
	filter = normalizeAppDirectorWaitFilterV0(filter)
	if appDirectorWaitFilterEmptyV0(filter) {
		if ports.WaitStateWriter != nil && len(explicitRefs) > 0 {
			if ports.RunStore == nil {
				return appDirectorWaitResolutionV0{}, AppDirectorServiceIssueV0{Field: "ports.run_store"}
			}
			run, err := ports.RunStore.LoadRunV0(ctx, runRef)
			if err != nil {
				return appDirectorWaitResolutionV0{}, err
			}
			snapshot := appDirectorExplicitWaitSnapshotV0(run, explicitRefs)
			state, err := appDirectorWorkflowTaskWaitStateV0(runRef, filter, snapshot, meta)
			if err != nil {
				return appDirectorWaitResolutionV0{}, err
			}
			if err := ports.WaitStateWriter.SaveWorkflowTaskWaitStateV0(ctx, state); err != nil {
				return appDirectorWaitResolutionV0{}, err
			}
		}
		return appDirectorWaitResolutionV0{
			AgentRefs:    explicitRefs,
			ScopeApplied: len(explicitRefs) > 0,
		}, nil
	}
	if ports.DirectorTaskStore == nil {
		return appDirectorWaitResolutionV0{}, AppDirectorServiceIssueV0{Field: "ports.director_task_store"}
	}
	if ports.RunStore == nil {
		return appDirectorWaitResolutionV0{}, AppDirectorServiceIssueV0{Field: "ports.run_store"}
	}
	run, err := ports.RunStore.LoadRunV0(ctx, runRef)
	if err != nil {
		return appDirectorWaitResolutionV0{}, err
	}
	snapshot, err := orquestacionnucleoapp.BuildWorkflowTaskWaitSnapshotV0(
		ctx,
		ports.DirectorTaskStore,
		run,
		orquestacionnucleoapp.WorkflowTaskWaitFilterV0{
			CohortRef:     filter.CohortRef,
			WaveRef:       filter.WaveRef,
			ParentTaskRef: filter.ParentTaskRef,
		},
	)
	if err != nil {
		return appDirectorWaitResolutionV0{}, err
	}
	snapshot.AgentRefs = compactServiceRefsV0(append(snapshot.AgentRefs, explicitRefs...))
	snapshot.PendingAgentRefs = compactServiceRefsV0(append(
		snapshot.PendingAgentRefs,
		appDirectorExplicitPendingAgentRefsV0(run, explicitRefs)...,
	))
	if ports.WaitStateWriter != nil {
		state, err := appDirectorWorkflowTaskWaitStateV0(runRef, filter, snapshot, meta)
		if err != nil {
			return appDirectorWaitResolutionV0{}, err
		}
		if err := ports.WaitStateWriter.SaveWorkflowTaskWaitStateV0(ctx, state); err != nil {
			return appDirectorWaitResolutionV0{}, err
		}
	}
	return appDirectorWaitResolutionV0{
		AgentRefs:    snapshot.AgentRefs,
		ScopeApplied: true,
		Snapshot:     snapshot,
	}, nil
}

func appDirectorExplicitWaitSnapshotV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	agentRefs []string,
) orquestacionnucleoapp.WorkflowTaskWaitSnapshotV0 {
	return orquestacionnucleoapp.WorkflowTaskWaitSnapshotV0{
		AgentRefs:        compactServiceRefsV0(agentRefs),
		PendingAgentRefs: appDirectorExplicitPendingAgentRefsV0(run, agentRefs),
	}
}

func appDirectorExplicitPendingAgentRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	agentRefs []string,
) []string {
	requested := appDirectorRequestedAgentRefsV0(run)
	pending := make([]string, 0, len(agentRefs))
	for _, agentRef := range compactServiceRefsV0(agentRefs) {
		if len(requested) > 0 && !requested[agentRef] {
			continue
		}
		if startAppDirectorStringInSetV0(run.DeliveredAgents, agentRef) ||
			startAppDirectorStringInSetV0(run.FailedAgents, agentRef) ||
			startAppDirectorStringInSetV0(run.LostAgents, agentRef) ||
			startAppDirectorStringInSetV0(run.ConfirmedStoppedAgents, agentRef) ||
			startAppDirectorStringInSetV0(run.StoppedAgents, agentRef) {
			continue
		}
		pending = append(pending, agentRef)
	}
	return pending
}

func appDirectorRequestedPendingAgentRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	agentRefs []string,
) []string {
	requested := appDirectorRequestedAgentRefsV0(run)
	pending := make([]string, 0, len(agentRefs))
	for _, agentRef := range compactServiceRefsV0(agentRefs) {
		if !requested[agentRef] {
			continue
		}
		if appDirectorAgentTerminalInRunV0(run, agentRef) {
			continue
		}
		pending = append(pending, agentRef)
	}
	return pending
}

func appDirectorAgentTerminalInRunV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	agentRef string,
) bool {
	return startAppDirectorStringInSetV0(run.DeliveredAgents, agentRef) ||
		startAppDirectorStringInSetV0(run.FailedAgents, agentRef) ||
		startAppDirectorStringInSetV0(run.LostAgents, agentRef) ||
		startAppDirectorStringInSetV0(run.ConfirmedStoppedAgents, agentRef) ||
		startAppDirectorStringInSetV0(run.StoppedAgents, agentRef)
}

func appDirectorRequestedAgentRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) map[string]bool {
	requested := map[string]bool{}
	for _, values := range [][]string{
		run.Agents,
		run.StartedAgents,
	} {
		for _, agentRef := range compactServiceRefsV0(values) {
			requested[agentRef] = true
		}
	}
	return requested
}

func appDirectorRunHasRequestedAgentRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) bool {
	return len(compactServiceRefsV0(append(append([]string(nil), run.Agents...), run.StartedAgents...))) > 0
}

func appDirectorRunHasUnmaterializedAgentRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	agentRefs []string,
) bool {
	requested := appDirectorRequestedAgentRefsV0(run)
	for _, agentRef := range compactServiceRefsV0(agentRefs) {
		if requested[agentRef] || appDirectorAgentTerminalInRunV0(run, agentRef) {
			continue
		}
		return true
	}
	return false
}

func normalizeAppDirectorWaitFilterV0(filter appDirectorWaitFilterV0) appDirectorWaitFilterV0 {
	return appDirectorWaitFilterV0{
		CohortRef:     strings.TrimSpace(filter.CohortRef),
		WaveRef:       strings.TrimSpace(filter.WaveRef),
		ParentTaskRef: strings.TrimSpace(filter.ParentTaskRef),
	}
}

func appDirectorWaitFilterEmptyV0(filter appDirectorWaitFilterV0) bool {
	return strings.TrimSpace(filter.CohortRef) == "" &&
		strings.TrimSpace(filter.WaveRef) == "" &&
		strings.TrimSpace(filter.ParentTaskRef) == ""
}

func appDirectorWorkflowTaskWaitStateV0(
	runRef string,
	filter appDirectorWaitFilterV0,
	snapshot orquestacionnucleoapp.WorkflowTaskWaitSnapshotV0,
	meta appDirectorWaitStateMetaV0,
) (orquestacionnucleoapp.WorkflowTaskWaitStateV0, error) {
	state := orquestacionnucleoapp.WorkflowTaskWaitStateV0{
		SchemaVersion:    orquestacionnucleoapp.WorkflowTaskWaitStateSchemaVersionV0,
		WaitRef:          appDirectorWaitRefV0(runRef, filter, meta.CorrelationID),
		RunRef:           runRef,
		ReasonCode:       orquestacionnucleoapp.WorkflowTaskWaitReasonCohortInProgressV0,
		CohortRef:        filter.CohortRef,
		WaveRef:          filter.WaveRef,
		ParentTaskRef:    filter.ParentTaskRef,
		TaskRefs:         append([]string(nil), snapshot.TaskRefs...),
		AgentRefs:        append([]string(nil), snapshot.AgentRefs...),
		PendingAgentRefs: append([]string(nil), snapshot.PendingAgentRefs...),
		Status:           orquestacionnucleoapp.WorkflowTaskWaitStateStatusWaitingV0,
		Attempt:          1,
		MaxExternalWaits: meta.MaxExternalWaits,
		CorrelationID:    meta.CorrelationID,
		EvidenceRefs:     compactServiceRefsV0(append(meta.EvidenceRefs, "evidence-ref-app-director-wait-state-v0")),
		ObservedAt:       meta.OccurredAt,
	}
	return orquestacionnucleoapp.NewWorkflowTaskWaitStateV0(state)
}

func appDirectorWaitRefV0(
	runRef string,
	filter appDirectorWaitFilterV0,
	correlationID string,
) string {
	parts := []string{
		"wait-ref",
		runRef,
		filter.CohortRef,
		filter.WaveRef,
		filter.ParentTaskRef,
		correlationID,
	}
	return appDirectorSafeRefPartV0(strings.Join(compactServiceRefsV0(parts), "-"))
}

func appDirectorSafeRefPartV0(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\\", "-")
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, " ", "-")
	return value
}
