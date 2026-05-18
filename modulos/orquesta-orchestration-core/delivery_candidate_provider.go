package orquestacionnucleoapp

import (
	"context"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

type DeliveryCandidateProviderV0 struct {
	Base           CandidateProviderPortV0
	DeliverySource AgentDeliveryObservationProviderPortV0
	RequestedBy    string
}

var _ CandidateProviderPortV0 = DeliveryCandidateProviderV0{}

func (provider DeliveryCandidateProviderV0) BuildSchedulerCandidatesV0(
	ctx context.Context,
	request SchedulerCandidateRequestV0,
) (SchedulerCandidateSetV0, error) {
	candidates, err := provider.baseCandidatesV0(ctx, request)
	if err != nil {
		return SchedulerCandidateSetV0{}, err
	}
	if provider.DeliverySource == nil {
		return candidates, nil
	}
	observations, err := provider.DeliverySource.BuildAgentDeliveryObservationsV0(
		ctx,
		agentDeliveryObservationRequestV0(request),
	)
	if err != nil {
		return SchedulerCandidateSetV0{}, err
	}
	for _, observation := range observations {
		observation = normalizeDeliveryObservationV0(request, observation)
		if !deliveryObservationMatchesWaitAgentRefsV0(request, observation) {
			continue
		}
		if observationHasTaskDeliveryV0(request.Run, observation) {
			candidate, err := provider.deliveryCandidateV0(request, observation)
			if err != nil {
				return SchedulerCandidateSetV0{}, err
			}
			candidates.DeliveryCandidates = append(candidates.DeliveryCandidates, candidate)
			candidates.EvidenceRefs = compactStringsV0(append(candidates.EvidenceRefs, candidate.EvidenceRefs...))
			return candidates, nil
		}
		candidate, err := provider.phaseArtifactCandidateV0(request, observation)
		if err != nil {
			return SchedulerCandidateSetV0{}, err
		}
		candidates.PhaseArtifactCandidates = append(candidates.PhaseArtifactCandidates, candidate)
		candidates.EvidenceRefs = compactStringsV0(append(candidates.EvidenceRefs, candidate.EvidenceRefs...))
		return candidates, nil
	}
	return candidates, nil
}

func observationHasTaskDeliveryV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	observation AgentDeliveryObservationV0,
) bool {
	return strings.TrimSpace(observation.DeliveryRef) != "" &&
		orchestrationRunHasTaskRefV0(run, observation.TaskID)
}

func orchestrationRunHasTaskRefV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	taskRef string,
) bool {
	taskRef = strings.TrimSpace(taskRef)
	for _, value := range run.Tasks {
		if strings.TrimSpace(value) == taskRef {
			return true
		}
	}
	return false
}

func (provider DeliveryCandidateProviderV0) baseCandidatesV0(
	ctx context.Context,
	request SchedulerCandidateRequestV0,
) (SchedulerCandidateSetV0, error) {
	if provider.Base == nil {
		return SchedulerCandidateSetV0{}, nil
	}
	return provider.Base.BuildSchedulerCandidatesV0(ctx, request)
}

func agentDeliveryObservationRequestV0(
	request SchedulerCandidateRequestV0,
) AgentDeliveryObservationRequestV0 {
	return AgentDeliveryObservationRequestV0{
		Run:              request.Run,
		StepNumber:       request.StepNumber,
		MaxSteps:         request.MaxSteps,
		OccurredAt:       request.OccurredAt,
		PreviousStep:     request.PreviousStep,
		CorrelationID:    request.CorrelationID,
		EvidenceRefs:     request.EvidenceRefs,
		PreviousDecision: request.PreviousDecision,
		WaitAgentRefs:    request.WaitAgentRefs,
	}
}

func (provider DeliveryCandidateProviderV0) deliveryCandidateV0(
	request SchedulerCandidateRequestV0,
	observation AgentDeliveryObservationV0,
) (orquestadirectorscheduler.SchedulableDeliveryCandidateV0, error) {
	if err := validateDeliveryObservationV0(request, observation); err != nil {
		return orquestadirectorscheduler.SchedulableDeliveryCandidateV0{}, err
	}
	return orquestadirectorscheduler.SchedulableDeliveryCandidateV0{
		CandidateRef: observation.CandidateRef,
		CommandMeta: orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-register-delivery-" + observation.DeliveryRef,
			RunID:          request.Run.RunID,
			IdempotencyKey: "idem-register-delivery-" + observation.DeliveryRef,
			CorrelationID:  strings.TrimSpace(request.CorrelationID),
			RequestedBy:    deliveryCandidateRequestedByV0(provider.RequestedBy),
			OccurredAt:     strings.TrimSpace(request.OccurredAt),
		},
		Payload: orquestacoreworkflow.RegisterDeliveryCommandPayloadV0{
			DeliveryRef:  observation.DeliveryRef,
			PhaseID:      observation.PhaseID,
			TaskID:       observation.TaskID,
			AgentRef:     observation.AgentRef,
			Summary:      observation.Summary,
			EvidenceRefs: observation.EvidenceRefs,
		},
		EvidenceRefs: compactStringsV0(append(request.EvidenceRefs, observation.EvidenceRefs...)),
	}, nil
}

func (provider DeliveryCandidateProviderV0) phaseArtifactCandidateV0(
	request SchedulerCandidateRequestV0,
	observation AgentDeliveryObservationV0,
) (orquestadirectorscheduler.SchedulablePhaseArtifactCandidateV0, error) {
	if err := validatePhaseArtifactObservationV0(request, observation); err != nil {
		return orquestadirectorscheduler.SchedulablePhaseArtifactCandidateV0{}, err
	}
	return orquestadirectorscheduler.SchedulablePhaseArtifactCandidateV0{
		CandidateRef: observation.CandidateRef,
		CommandMeta: orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-register-phase-artifact-" + observation.ArtifactRef,
			RunID:          request.Run.RunID,
			IdempotencyKey: "idem-register-phase-artifact-" + observation.ArtifactRef,
			CorrelationID:  strings.TrimSpace(request.CorrelationID),
			RequestedBy:    phaseArtifactCandidateRequestedByV0(provider.RequestedBy),
			OccurredAt:     strings.TrimSpace(request.OccurredAt),
		},
		Payload: orquestacoreworkflow.RegisterPhaseArtifactCommandPayloadV0{
			ArtifactRef:  observation.ArtifactRef,
			PhaseID:      observation.PhaseID,
			AgentRef:     observation.AgentRef,
			Summary:      phaseArtifactSummaryV0(observation.Summary),
			EvidenceRefs: observation.EvidenceRefs,
		},
		EvidenceRefs: compactStringsV0(append(request.EvidenceRefs, observation.EvidenceRefs...)),
	}, nil
}

func normalizeDeliveryObservationV0(
	request SchedulerCandidateRequestV0,
	observation AgentDeliveryObservationV0,
) AgentDeliveryObservationV0 {
	observation.CandidateRef = strings.TrimSpace(observation.CandidateRef)
	observation.ArtifactRef = strings.TrimSpace(observation.ArtifactRef)
	observation.DeliveryRef = strings.TrimSpace(observation.DeliveryRef)
	observation.PhaseID = strings.TrimSpace(observation.PhaseID)
	observation.TaskID = strings.TrimSpace(observation.TaskID)
	observation.AgentRef = strings.TrimSpace(observation.AgentRef)
	observation.Summary = strings.TrimSpace(observation.Summary)
	observation.EvidenceRefs = compactStringsV0(observation.EvidenceRefs)
	if observation.CandidateRef == "" {
		observation.CandidateRef = "delivery-candidate-ref-" + observation.DeliveryRef
	}
	if observation.ArtifactRef == "" {
		observation.ArtifactRef = observation.DeliveryRef
	}
	if observation.PhaseID == "" {
		observation.PhaseID = string(request.Run.CurrentPhase)
	}
	return observation
}

func validateDeliveryObservationV0(
	request SchedulerCandidateRequestV0,
	observation AgentDeliveryObservationV0,
) error {
	if strings.TrimSpace(request.OccurredAt) == "" {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "occurred_at", "occurred_at requerido")
	}
	command, err := orquestacoreworkflow.NewRegisterDeliveryCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-register-delivery-validate-" + observation.DeliveryRef,
			RunID:          request.Run.RunID,
			IdempotencyKey: "idem-register-delivery-validate-" + observation.DeliveryRef,
			OccurredAt:     request.OccurredAt,
		},
		orquestacoreworkflow.RegisterDeliveryCommandPayloadV0{
			DeliveryRef:  observation.DeliveryRef,
			PhaseID:      observation.PhaseID,
			TaskID:       observation.TaskID,
			AgentRef:     observation.AgentRef,
			Summary:      observation.Summary,
			EvidenceRefs: observation.EvidenceRefs,
		},
	)
	if err != nil {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "delivery_observation", err.Error())
	}
	if command.RunID != request.Run.RunID {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "run_ref", "run_id no coincide")
	}
	return nil
}

func validatePhaseArtifactObservationV0(
	request SchedulerCandidateRequestV0,
	observation AgentDeliveryObservationV0,
) error {
	if strings.TrimSpace(request.OccurredAt) == "" {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "occurred_at", "occurred_at requerido")
	}
	command, err := orquestacoreworkflow.NewRegisterPhaseArtifactCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-register-phase-artifact-validate-" + observation.ArtifactRef,
			RunID:          request.Run.RunID,
			IdempotencyKey: "idem-register-phase-artifact-validate-" + observation.ArtifactRef,
			OccurredAt:     request.OccurredAt,
		},
		orquestacoreworkflow.RegisterPhaseArtifactCommandPayloadV0{
			ArtifactRef:  observation.ArtifactRef,
			PhaseID:      observation.PhaseID,
			AgentRef:     observation.AgentRef,
			Summary:      phaseArtifactSummaryV0(observation.Summary),
			EvidenceRefs: observation.EvidenceRefs,
		},
	)
	if err != nil {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "phase_artifact_observation", err.Error())
	}
	if command.RunID != request.Run.RunID {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "run_ref", "run_id no coincide")
	}
	return nil
}

func deliveryCandidateRequestedByV0(value string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return "orquesta-nucleo-delivery"
}

func phaseArtifactCandidateRequestedByV0(value string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return "orquesta-nucleo-artifact"
}

func phaseArtifactSummaryV0(value string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return "Artefacto compacto validado por recibo de agente."
}

func deliveryObservationMatchesWaitAgentRefsV0(
	request SchedulerCandidateRequestV0,
	observation AgentDeliveryObservationV0,
) bool {
	waitAgentRefs := compactStringsV0(request.WaitAgentRefs)
	if len(waitAgentRefs) == 0 {
		return true
	}
	return stringInDeliveryCandidateSetV0(waitAgentRefs, observation.AgentRef)
}

func stringInDeliveryCandidateSetV0(values []string, want string) bool {
	want = strings.TrimSpace(want)
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}
