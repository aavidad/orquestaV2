package orquestaappdirectorservice

import (
	"context"
	"strings"

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

func appDirectorResolvedWaitAgentRefsV0(
	ctx context.Context,
	runRef string,
	explicitRefs []string,
	filter appDirectorWaitFilterV0,
	ports StartAppDirectorPortsV0,
) ([]string, error) {
	resolution, err := appDirectorResolvedWaitV0(ctx, runRef, explicitRefs, filter, ports, appDirectorWaitStateMetaV0{})
	if err != nil {
		return nil, err
	}
	return resolution.AgentRefs, nil
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
		return appDirectorWaitResolutionV0{AgentRefs: explicitRefs}, nil
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
		AgentRefs:    compactServiceRefsV0(append(explicitRefs, snapshot.AgentRefs...)),
		ScopeApplied: true,
		Snapshot:     snapshot,
	}, nil
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
