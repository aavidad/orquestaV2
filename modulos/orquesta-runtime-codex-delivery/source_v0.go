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
	// PromoteMaterializedArtifactWithoutAck activa la recuperacion de trabajo: si
	// un agente NO escribio agent_ack.json pero materializo su write-set en disco,
	// la entrega se promueve igualmente con un gate-issue de revision humana, en
	// vez de descartarse. Opt-in: por defecto el ACK ausente sigue siendo pending.
	PromoteMaterializedArtifactWithoutAck bool
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
		if codexReceiptIssueMeansAckNotReadyV0(issue) {
			// Recuperacion de trabajo: ACK ausente pero artefacto materializado.
			// Promovemos la entrega con gate-issue de revision humana en vez de
			// descartarla, para que un run no se cuelgue por falta del acuse.
			if source.PromoteMaterializedArtifactWithoutAck &&
				codexProgressWriteSetMaterializedV0(descriptor) {
				return source.promotedObservationFromMaterializedArtifactV0(ctx, descriptor)
			}
			return orquestacionnucleoapp.AgentDeliveryObservationV0{}, false, nil
		}
		if codexReceiptIssueMeansAckWithoutDeliveryV0(issue) {
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

// promotedObservationFromMaterializedArtifactV0 sintetiza una entrega cuando el
// agente materializo su write-set pero no escribio agent_ack.json. Deriva las
// refs del packet del spec (no del ACK, que no existe), pasa por la verificacion
// de worktree (que aporta evidencia/soft-gates del artefacto real) y marca un
// gate-issue de revision humana para que el cierre no se de por bueno sin que un
// humano valide el artefacto sin acuse.
func (source CodexDeliveryObservationSourceV0) promotedObservationFromMaterializedArtifactV0(
	ctx context.Context,
	descriptor CodexReceiptDescriptorV0,
) (orquestacionnucleoapp.AgentDeliveryObservationV0, bool, error) {
	packet := descriptor.Spec.AgentPacket
	deliveryRef := strings.TrimSpace(packet.DeliveryRefs.AckRef)
	if deliveryRef == "" {
		// Sin ref de acuse derivable no podemos construir una entrega causal.
		return orquestacionnucleoapp.AgentDeliveryObservationV0{}, false, nil
	}
	worktreeEvidenceRefs, err := source.verifyDescriptorWorktreeMaterializedV0(ctx, descriptor)
	if err != nil {
		return orquestacionnucleoapp.AgentDeliveryObservationV0{}, false, err
	}
	evidenceRefs := codexDeliveryObservationCoreEvidenceRefsV0(append([]string{
		deliveryRef,
		strings.TrimSpace(packet.DeliveryRefs.MailboxRef),
		strings.TrimSpace(packet.DeliveryRefs.ReadinessRef),
		"evidence-ref-artifact-without-ack",
		"gate-issue:artifact_without_ack_requires_review",
	}, worktreeEvidenceRefs...))
	return orquestacionnucleoapp.AgentDeliveryObservationV0{
		CandidateRef: "delivery-candidate-ref-" + deliveryRef,
		ArtifactRef:  deliveryRef,
		DeliveryRef:  deliveryRef,
		PhaseID:      strings.TrimSpace(packet.Phase),
		TaskID:       strings.TrimSpace(packet.Task.TaskRef),
		AgentRef:     strings.TrimSpace(packet.RequestID),
		Summary:      "Entrega promovida desde artefacto materializado sin ACK; requiere revision (agente primero, humano solo si el Director no puede resolverlo).",
		EvidenceRefs: evidenceRefs,
	}, true, nil
}

// verifyDescriptorWorktreeMaterializedV0 ejecuta la verificacion de worktree para
// la entrega promovida. Si el verificador no esta disponible, la promocion sigue
// adelante solo con la evidencia de "artefacto sin ACK" (no es un corte duro).
func (source CodexDeliveryObservationSourceV0) verifyDescriptorWorktreeMaterializedV0(
	ctx context.Context,
	descriptor CodexReceiptDescriptorV0,
) ([]string, error) {
	if source.WorktreeVerifier == nil {
		return nil, nil
	}
	request := CodexReceiptWorktreeVerificationRequestV0{
		DescriptorRef:       descriptor.DescriptorRef,
		RunID:               descriptor.RunID,
		AgentRef:            descriptor.AgentRef,
		ProjectWorkDir:      descriptor.ProjectWorkDir,
		WorktreeBaselineRef: descriptor.WorktreeBaselineRef,
		Spec:                descriptor.Spec,
		AckFiles:            append([]string(nil), descriptor.Spec.AgentPacket.Task.WriteSet...),
	}
	if verifier, ok := source.WorktreeVerifier.(CodexReceiptWorktreeEvidenceVerifierPortV0); ok {
		return verifier.VerifyCodexReceiptWorktreeEvidenceRefsV0(ctx, request)
	}
	return nil, source.WorktreeVerifier.VerifyCodexReceiptWorktreeV0(ctx, request)
}
