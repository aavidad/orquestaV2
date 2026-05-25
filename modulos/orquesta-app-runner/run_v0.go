package orquestaapprunner

import (
	"context"

	orquestaappplanner "orquesta/modulos/orquesta-app-planner"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func RunPreparedAppOrchestrationV0(
	ctx context.Context,
	request RunPreparedAppOrchestrationRequestV0,
	ports RunPreparedAppOrchestrationPortsV0,
) (AppOrchestrationRunResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	request = normalizeRunPreparedAppOrchestrationRequestV0(request)
	if err := validateRunPreparedAppOrchestrationRequestV0(request); err != nil {
		return AppOrchestrationRunResultV0{}, err
	}
	if err := validateDirectorV2RequirementForAppRunnerV0(request); err != nil {
		return AppOrchestrationRunResultV0{}, err
	}
	if err := validateRunPreparedAppOrchestrationPortsV0(ports); err != nil {
		return AppOrchestrationRunResultV0{}, err
	}
	if err := ensurePreparedAppRunStoredV0(ctx, request, ports); err != nil {
		return AppOrchestrationRunResultV0{}, err
	}
	loop, err := runPreparedAppLoopV0(ctx, request, ports)
	if err != nil {
		return AppOrchestrationRunResultV0{}, err
	}
	progress, err := orquestaappplanner.EvaluateAppPlanProgressV0(
		request.Prepared.Plan,
		loop.Managed.Final.Run.Deliveries,
	)
	if err != nil {
		return AppOrchestrationRunResultV0{}, err
	}
	return appRunResultV0(request, loop, progress), nil
}

func normalizeRunPreparedAppOrchestrationRequestV0(
	request RunPreparedAppOrchestrationRequestV0,
) RunPreparedAppOrchestrationRequestV0 {
	request.OccurredAt = firstAppRunnerValueV0(request.OccurredAt)
	request.CorrelationID = firstAppRunnerValueV0(
		request.CorrelationID,
		"corr-"+request.Prepared.Run.RunID,
	)
	request.RequestedBy = firstAppRunnerValueV0(request.RequestedBy, "orquesta-app-runner")
	return normalizeRunPreparedLimitsV0(request)
}

func ensurePreparedAppRunStoredV0(
	ctx context.Context,
	request RunPreparedAppOrchestrationRequestV0,
	ports RunPreparedAppOrchestrationPortsV0,
) error {
	existing, err := ports.RunStore.LoadRunV0(ctx, request.Prepared.Run.RunID)
	if err == nil && existing.RunID == request.Prepared.Run.RunID {
		return nil
	}
	if err != nil && !orquestacionnucleoapp.IsRunNotFoundErrorV0(err) {
		return err
	}
	return ports.RunStore.SaveRunV0(ctx, request.Prepared.Run)
}

func appRunResultV0(
	request RunPreparedAppOrchestrationRequestV0,
	loop appRunnerLoopResultV0,
	progress orquestaappplanner.AppPlanProgressV0,
) AppOrchestrationRunResultV0 {
	status := AppOrchestrationRunStatusRunningV0
	if progress.Complete {
		status = AppOrchestrationRunStatusCompleteV0
	}
	return AppOrchestrationRunResultV0{
		SchemaVersion:     AppOrchestrationRunResultSchemaVersionV0,
		Status:            status,
		Run:               loop.Managed.Final.Run,
		Progress:          progress,
		LoopStatus:        loop.Managed.Status,
		StartedAgents:     append([]string(nil), loop.Managed.Final.Run.StartedAgents...),
		Attempts:          len(loop.Managed.Attempts),
		ExternalWaits:     len(loop.Managed.ExternalWaits),
		DirectorLoopStats: loop.DirectorLoopStats,
		RoutePolicy:       AppRunnerPreviewRoutePolicyV0(AppRunnerLegacyEntrypointExecuteV0),
		EvidenceRefs: compactAppRunnerRefsV0(append(
			request.Prepared.EvidenceRefs,
			"evidence-ref-app-runner-run-v0",
		)),
	}
}
