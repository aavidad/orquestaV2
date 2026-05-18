package orquestaapprunner

import (
	"context"

	orquestaappplanner "orquesta/modulos/orquesta-app-planner"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func runPreparedAppLoopAcrossPlanPhasesV0(
	ctx context.Context,
	request RunPreparedAppOrchestrationRequestV0,
	ports RunPreparedAppOrchestrationPortsV0,
	service orquestacionnucleoapp.ServiceV0,
	loopRequest orquestacionnucleoapp.ProgressiveLoopRequestV0,
) (appRunnerLoopResultV0, error) {
	var combined appRunnerLoopResultV0
	limit := appPlanPhaseTransitionLimitV0(request.Prepared.Plan)
	for attempt := 0; attempt <= limit; attempt++ {
		current, err := runPreparedSingleAppLoopV0(ctx, request, ports, service, loopRequest)
		combined = mergeAppRunnerLoopResultsV0(combined, current)
		if err != nil {
			return combined, err
		}
		if current.Managed.Status == orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 {
			return combined, nil
		}
		if current.Managed.Status != orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0 {
			return combined, nil
		}
		progress, err := orquestaappplanner.EvaluateAppPlanProgressV0(
			request.Prepared.Plan,
			current.Managed.Final.Run.Deliveries,
		)
		if err != nil || progress.Complete {
			return combined, err
		}
		nextPhase, ok := nextAppPlanPhaseToOpenV0(request.Prepared.Plan, current.Managed.Final.Run)
		if !ok {
			return combined, nil
		}
		opened, err := openAppPlanPhaseV0(ctx, request, ports, current.Managed.Final.Run, nextPhase)
		if err != nil || !opened {
			return combined, err
		}
	}
	return combined, nil
}

func mergeAppRunnerLoopResultsV0(
	left appRunnerLoopResultV0,
	right appRunnerLoopResultV0,
) appRunnerLoopResultV0 {
	if left.Managed.Final.Run.RunID == "" {
		return right
	}
	left.Managed.Status = right.Managed.Status
	left.Managed.Final = right.Managed.Final
	left.Managed.Attempts = append(left.Managed.Attempts, right.Managed.Attempts...)
	left.Managed.ExternalWaits = append(left.Managed.ExternalWaits, right.Managed.ExternalWaits...)
	if right.DirectorLoopStats != nil {
		left.DirectorLoopStats = right.DirectorLoopStats
	}
	return left
}

func appPlanPhaseTransitionLimitV0(plan orquestaappplanner.AppMicrotaskPlanV0) int {
	seen := map[orquestacoreworkflow.OrchestrationPhaseIDV0]bool{}
	for _, unit := range plan.Units {
		seen[unit.PhaseID] = true
	}
	if len(seen) == 0 {
		return 1
	}
	return len(seen) + 1
}

func nextAppPlanPhaseToOpenV0(
	plan orquestaappplanner.AppMicrotaskPlanV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) (orquestacoreworkflow.OrchestrationPhaseIDV0, bool) {
	for _, unit := range plan.Units {
		if appPlanUnitDeliveredV0(run, unit) {
			continue
		}
		if !appPlanUnitDependenciesDeliveredV0(run, unit) {
			continue
		}
		if unit.PhaseID == run.CurrentPhase {
			return "", false
		}
		return unit.PhaseID, unit.PhaseID != ""
	}
	return "", false
}

func appPlanUnitDeliveredV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	unit orquestaappplanner.AppWorkUnitV0,
) bool {
	return appRunnerStringInSetV0(run.Deliveries, unit.DeliveryRef)
}

func appPlanUnitDependenciesDeliveredV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	unit orquestaappplanner.AppWorkUnitV0,
) bool {
	for _, ref := range unit.DependsOnDeliveries {
		if !appRunnerStringInSetV0(run.Deliveries, ref) {
			return false
		}
	}
	return true
}

func openAppPlanPhaseV0(
	ctx context.Context,
	request RunPreparedAppOrchestrationRequestV0,
	ports RunPreparedAppOrchestrationPortsV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	phase orquestacoreworkflow.OrchestrationPhaseIDV0,
) (bool, error) {
	command, err := orquestacoreworkflow.NewOpenPhaseCommandV0(
		appRunPhaseCommandMetaV0(request, "open-"+string(phase)),
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: string(phase),
			Reason:  "Avanzar plan de app a la siguiente fase con trabajo listo.",
		},
	)
	if err != nil {
		return false, err
	}
	next, events, err := applyWorkflowCommandWithEventsV0(run, command)
	if err != nil {
		return false, err
	}
	if len(events) == 0 {
		return false, nil
	}
	if err := ports.EventSink.AppendRunEventsV0(ctx, run.RunID, events); err != nil {
		return false, err
	}
	if err := ports.RunStore.SaveRunV0(ctx, next); err != nil {
		return false, err
	}
	return true, nil
}

func appRunPhaseCommandMetaV0(
	request RunPreparedAppOrchestrationRequestV0,
	kind string,
) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	ref := safeAppRunnerRefPartV0(kind + "-" + request.Prepared.Run.RunID)
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      "cmd-app-run-" + ref,
		RunID:          request.Prepared.Run.RunID,
		IdempotencyKey: "idem-app-run-" + ref,
		CorrelationID:  request.CorrelationID,
		RequestedBy:    request.RequestedBy,
		OccurredAt:     request.OccurredAt,
	}
}
