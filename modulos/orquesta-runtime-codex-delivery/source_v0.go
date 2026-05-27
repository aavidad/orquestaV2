package orquestaruntimecodexdelivery

import (
	"context"
	"fmt"
	"strings"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

type CodexDeliveryObservationSourceV0 struct {
	Store            CodexReceiptDescriptorStorePortV0
	WorktreeVerifier CodexReceiptWorktreeVerifierPortV0
}

var _ orquestacionnucleoapp.AgentDeliveryObservationProviderPortV0 = CodexDeliveryObservationSourceV0{}

func (source CodexDeliveryObservationSourceV0) BuildAgentDeliveryObservationsV0(
	ctx context.Context,
	request orquestacionnucleoapp.AgentDeliveryObservationRequestV0,
) ([]orquestacionnucleoapp.AgentDeliveryObservationV0, error) {
	if source.Store == nil {
		return nil, fmt.Errorf("codex_delivery_observation_source: store requerido")
	}
	descriptors, err := source.Store.ListCodexReceiptDescriptorsV0(
		ctx,
		codexReceiptDescriptorRequestFromNucleoV0(request),
	)
	if err != nil {
		return nil, err
	}
	observations := make([]orquestacionnucleoapp.AgentDeliveryObservationV0, 0, len(descriptors))
	for _, descriptor := range descriptors {
		if !codexDeliveryObservationDescriptorMatchesWaitAgentRefsV0(request, descriptor) {
			continue
		}
		if codexReceiptDescriptorAlreadyReflectedV0(request, descriptor) {
			continue
		}
		observation, ready, err := source.observationFromDescriptorV0(ctx, descriptor)
		if err != nil {
			return nil, err
		}
		if !ready {
			continue
		}
		if !codexDeliveryObservationAgentEligibleV0(request, observation.AgentRef) {
			continue
		}
		if stringInCodexDeliverySetV0(request.Run.Deliveries, observation.DeliveryRef) {
			continue
		}
		if codexReceiptArtifactRefRegisteredV0(request.Run.PhaseArtifacts, observation.ArtifactRef) {
			continue
		}
		observations = append(observations, observation)
	}
	return observations, nil
}

func (source CodexDeliveryObservationSourceV0) observationFromDescriptorV0(
	ctx context.Context,
	descriptor CodexReceiptDescriptorV0,
) (orquestacionnucleoapp.AgentDeliveryObservationV0, bool, error) {
	codexObservation, issues := orquestaruntimecodex.ReadCodexDeliveryObservationFileV0(
		strings.TrimSpace(descriptor.AckPath),
		descriptor.Spec,
	)
	if len(issues) > 0 {
		issue := issues[0]
		if codexReceiptIssueMeansAckNotReadyV0(issue) ||
			codexReceiptIssueMeansAckWithoutDeliveryV0(issue) {
			return orquestacionnucleoapp.AgentDeliveryObservationV0{}, false, nil
		}
		return orquestacionnucleoapp.AgentDeliveryObservationV0{}, false, fmt.Errorf(
			"codex_delivery_observation: %s:%s",
			issue.Code,
			issue.Field,
		)
	}
	worktreeEvidenceRefs, err := source.verifyDescriptorWorktreeV0(ctx, descriptor)
	if err != nil {
		return orquestacionnucleoapp.AgentDeliveryObservationV0{}, false, err
	}
	evidenceRefs := codexDeliveryObservationCoreEvidenceRefsV0(
		append(codexObservation.EvidenceRefs, worktreeEvidenceRefs...),
	)
	return orquestacionnucleoapp.AgentDeliveryObservationV0{
		CandidateRef: "delivery-candidate-ref-" + codexObservation.DeliveryRef,
		ArtifactRef:  codexObservation.DeliveryRef,
		DeliveryRef:  codexObservation.DeliveryRef,
		PhaseID:      codexObservation.PhaseID,
		TaskID:       codexObservation.TaskID,
		AgentRef:     codexObservation.AgentRef,
		Summary:      codexObservation.Summary,
		EvidenceRefs: evidenceRefs,
	}, true, nil
}
