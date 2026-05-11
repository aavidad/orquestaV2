package orquestaappplanner

import (
	"context"
	"fmt"

	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
	orquestacionnucleoapp "orquesta/orquestacionnucleoapp"
)

type AppPlanCandidateProviderV0 struct {
	Plan        AppMicrotaskPlanV0
	RequestedBy string
}

var _ orquestacionnucleoapp.CandidateProviderPortV0 = AppPlanCandidateProviderV0{}

func (provider AppPlanCandidateProviderV0) BuildSchedulerCandidatesV0(
	ctx context.Context,
	request orquestacionnucleoapp.SchedulerCandidateRequestV0,
) (orquestacionnucleoapp.SchedulerCandidateSetV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return orquestacionnucleoapp.SchedulerCandidateSetV0{}, err
	}
	if err := validateAppMicrotaskPlanV0(provider.Plan); err != nil {
		return orquestacionnucleoapp.SchedulerCandidateSetV0{}, err
	}
	ready := provider.readyUnitsV0(request)
	claims, err := claimsForReadyUnitsV0(ready, request.Run.RunID)
	if err != nil {
		return orquestacionnucleoapp.SchedulerCandidateSetV0{}, err
	}
	candidates := make([]orquestadirectorscheduler.SchedulableWorkCandidateV0, 0, len(ready))
	for index, unit := range ready {
		candidates = append(candidates, provider.workCandidateV0(request, unit, claims[index]))
	}
	return orquestacionnucleoapp.SchedulerCandidateSetV0{
		WorkClaims:     claims,
		WorkCandidates: candidates,
		EvidenceRefs:   []string{"evidence-ref-app-plan-candidates-v0"},
	}, nil
}

func (provider AppPlanCandidateProviderV0) readyUnitsV0(
	request orquestacionnucleoapp.SchedulerCandidateRequestV0,
) []AppWorkUnitV0 {
	ready := make([]AppWorkUnitV0, 0, len(provider.Plan.Units))
	for _, unit := range provider.Plan.Units {
		if unit.PhaseID != request.Run.CurrentPhase {
			continue
		}
		if unitAlreadyClosedOrRunningV0(unit, request.Run) {
			continue
		}
		if !unitDependenciesDeliveredV0(unit, request.Run.Deliveries) {
			continue
		}
		ready = append(ready, unit)
	}
	return ready
}

func unitAlreadyClosedOrRunningV0(
	unit AppWorkUnitV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) bool {
	return appPlannerStringInSetV0(run.Deliveries, unit.DeliveryRef) ||
		appPlannerStringInSetV0(run.Agents, unit.AgentRequestID) ||
		appPlannerStringInSetV0(run.StartedAgents, unit.AgentRequestID) ||
		appPlannerStringInSetV0(run.FailedAgents, unit.AgentRequestID) ||
		appPlannerStringInSetV0(run.StoppedAgents, unit.AgentRequestID)
}

func unitDependenciesDeliveredV0(unit AppWorkUnitV0, deliveries []string) bool {
	for _, ref := range unit.DependsOnDeliveries {
		if !appPlannerStringInSetV0(deliveries, ref) {
			return false
		}
	}
	return true
}

func claimsForReadyUnitsV0(
	units []AppWorkUnitV0,
	runRef string,
) ([]orquestacoreconcurrency.WorksetClaimV0, error) {
	claims := make([]orquestacoreconcurrency.WorksetClaimV0, 0, len(units))
	for _, unit := range units {
		writeSet, err := scopeRefsForUnitV0(unit)
		if err != nil {
			return nil, err
		}
		claims = append(claims, orquestacoreconcurrency.WorksetClaimV0{
			SchemaVersion:  orquestacoreconcurrency.WorksetClaimSchemaVersionV0,
			ClaimRef:       unit.ClaimRef,
			RunRef:         runRef,
			TaskRef:        unit.TaskRef,
			GroupRef:       "group-" + unit.TaskRef,
			AgentRequestID: unit.AgentRequestID,
			WriteSet:       writeSet,
			EvidenceRefs:   unit.EvidenceRefs,
		})
	}
	return claims, nil
}

func scopeRefsForUnitV0(unit AppWorkUnitV0) ([]orquestacoreconcurrency.ScopeRefV0, error) {
	refs := make([]orquestacoreconcurrency.ScopeRefV0, 0, len(unit.WriteSet))
	for _, value := range unit.WriteSet {
		ref, issues := orquestacoreconcurrency.NewScopeRefV0(value)
		if len(issues) > 0 {
			return nil, fmt.Errorf("write_set invalido: %s", value)
		}
		refs = append(refs, ref)
	}
	return refs, nil
}

func (provider AppPlanCandidateProviderV0) workCandidateV0(
	request orquestacionnucleoapp.SchedulerCandidateRequestV0,
	unit AppWorkUnitV0,
	claim orquestacoreconcurrency.WorksetClaimV0,
) orquestadirectorscheduler.SchedulableWorkCandidateV0 {
	return orquestadirectorscheduler.SchedulableWorkCandidateV0{
		CandidateRef:     "candidate-" + unit.TaskRef,
		SubjectClaimRefs: []string{unit.ClaimRef},
		Claims:           []orquestacoreconcurrency.WorksetClaimV0{claim},
		CapacityCandidate: &orquestadirectorscheduler.SchedulerCapacityCommandCandidateV0{
			CommandMeta: commandMetaForUnitV0(request, unit, "capacity", provider.requestedByV0()),
			Payload:     capacityPayloadForUnitV0(unit),
		},
		AgentCandidate: &orquestadirectorscheduler.SchedulerAgentCommandCandidateV0{
			ClaimRef:    unit.ClaimRef,
			CommandMeta: commandMetaForUnitV0(request, unit, "agent", provider.requestedByV0()),
			Payload:     agentPayloadForUnitV0(unit),
		},
		GateCommandMeta:  commandMetaForUnitV0(request, unit, "gate", provider.requestedByV0()),
		GateEvidenceRefs: unit.EvidenceRefs,
		EvidenceRefs:     unit.EvidenceRefs,
	}
}

func capacityPayloadForUnitV0(unit AppWorkUnitV0) orquestacoreworkflow.RequestCapacityCommandPayloadV0 {
	return orquestacoreworkflow.RequestCapacityCommandPayloadV0{
		CapacityRequestID:          capacityRefV0(unit),
		PhaseID:                    string(unit.PhaseID),
		TaskRef:                    unit.TaskRef,
		ReasonCode:                 "app_microtask_ready",
		Summary:                    "Capacidad para microtarea de app.",
		MinimumRecommendedCapacity: unit.Capacity,
		EvidenceRefs:               unit.EvidenceRefs,
	}
}

func agentPayloadForUnitV0(unit AppWorkUnitV0) orquestacoreworkflow.RequestAgentCommandPayloadV0 {
	return orquestacoreworkflow.RequestAgentCommandPayloadV0{
		AgentRequestID:     unit.AgentRequestID,
		PhaseID:            string(unit.PhaseID),
		TaskRef:            unit.TaskRef,
		CapacityRequestRef: capacityRefV0(unit),
		Role:               unit.Role,
		Summary:            "Ejecutar microtarea de app con write-set acotado.",
		EvidenceRefs:       unit.EvidenceRefs,
	}
}

func commandMetaForUnitV0(
	request orquestacionnucleoapp.SchedulerCandidateRequestV0,
	unit AppWorkUnitV0,
	kind string,
	requestedBy string,
) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      commandRefV0(unit, kind),
		RunID:          request.Run.RunID,
		IdempotencyKey: idempotencyRefV0(unit, kind),
		CorrelationID:  request.CorrelationID,
		RequestedBy:    requestedBy,
		OccurredAt:     request.OccurredAt,
	}
}

func (provider AppPlanCandidateProviderV0) requestedByV0() string {
	if provider.RequestedBy != "" {
		return provider.RequestedBy
	}
	return "orquesta-app-planner"
}
