package orquestaruntimecodexdelivery

import (
	"context"
	"fmt"
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

type CodexReceiptDescriptorStorePortV0 interface {
	ListCodexReceiptDescriptorsV0(
		context.Context,
		CodexReceiptDescriptorRequestV0,
	) ([]CodexReceiptDescriptorV0, error)
}

type CodexReceiptDescriptorRecorderPortV0 interface {
	RecordCodexReceiptDescriptorV0(context.Context, CodexReceiptDescriptorV0) error
}

type CodexReceiptDescriptorRequestV0 struct {
	RunID          string
	StartedAgents  []string
	Deliveries     []string
	PhaseArtifacts []string
	CorrelationID  string
	EvidenceRefs   []string
}

type CodexReceiptDescriptorV0 struct {
	DescriptorRef       string
	RunID               string
	AgentRef            string
	Spec                orquestaruntime.ExternalAgentLaunchSpecV0
	AckPath             string
	ProjectWorkDir      string
	WorktreeBaselineRef string
}

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

func codexDeliveryObservationAgentEligibleV0(
	request orquestacionnucleoapp.AgentDeliveryObservationRequestV0,
	agentRef string,
) bool {
	return stringInCodexDeliverySetV0(request.Run.Agents, agentRef) &&
		stringInCodexDeliverySetV0(request.Run.StartedAgents, agentRef) &&
		!stringInCodexDeliverySetV0(request.Run.FailedAgents, agentRef) &&
		!stringInCodexDeliverySetV0(request.Run.StoppedAgents, agentRef)
}

func codexReceiptDescriptorRequestFromNucleoV0(
	request orquestacionnucleoapp.AgentDeliveryObservationRequestV0,
) CodexReceiptDescriptorRequestV0 {
	return CodexReceiptDescriptorRequestV0{
		RunID:          strings.TrimSpace(request.Run.RunID),
		StartedAgents:  compactCodexDeliveryRefsV0(request.Run.StartedAgents),
		Deliveries:     compactCodexDeliveryRefsV0(request.Run.Deliveries),
		PhaseArtifacts: compactCodexDeliveryRefsV0(request.Run.PhaseArtifacts),
		CorrelationID:  strings.TrimSpace(request.CorrelationID),
		EvidenceRefs:   compactCodexDeliveryRefsV0(request.EvidenceRefs),
	}
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
			return orquestacionnucleoapp.AgentDeliveryObservationV0{}, false, nil
		}
		return orquestacionnucleoapp.AgentDeliveryObservationV0{}, false, fmt.Errorf(
			"codex_delivery_observation: %s:%s",
			issue.Code,
			issue.Field,
		)
	}
	if err := source.verifyDescriptorWorktreeV0(ctx, descriptor); err != nil {
		return orquestacionnucleoapp.AgentDeliveryObservationV0{}, false, err
	}
	return orquestacionnucleoapp.AgentDeliveryObservationV0{
		CandidateRef: "delivery-candidate-ref-" + codexObservation.DeliveryRef,
		ArtifactRef:  codexObservation.DeliveryRef,
		DeliveryRef:  codexObservation.DeliveryRef,
		PhaseID:      codexObservation.PhaseID,
		TaskID:       codexObservation.TaskID,
		AgentRef:     codexObservation.AgentRef,
		Summary:      codexObservation.Summary,
		EvidenceRefs: compactCodexDeliveryRefsV0(codexObservation.EvidenceRefs),
	}, true, nil
}

func codexReceiptIssueMeansAckNotReadyV0(
	issue orquestaruntime.ExternalAgentConnectorErrorV0,
) bool {
	return string(issue.Code) == string(orquestaruntimecodex.CodexConnectorAckInvalidV0) &&
		strings.TrimSpace(issue.Field) == orquestaruntimecodex.CodexAgentAckFileNameV0 &&
		issue.Retryable &&
		stringInCodexDeliverySetV0(issue.Evidence, "ack_not_ready")
}

func compactCodexDeliveryRefsV0(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func stringInCodexDeliverySetV0(values []string, want string) bool {
	want = strings.TrimSpace(want)
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}

func codexReceiptArtifactRefRegisteredV0(values []string, artifactRef string) bool {
	artifactRef = strings.TrimSpace(artifactRef)
	if artifactRef == "" {
		return false
	}
	for _, value := range values {
		if codexReceiptProjectionArtifactRefV0(value) == artifactRef {
			return true
		}
	}
	return false
}

func codexReceiptProjectionArtifactRefV0(value string) string {
	value = strings.TrimSpace(value)
	artifactRef, _, ok := strings.Cut(value, "#phase:")
	if ok {
		return strings.TrimSpace(artifactRef)
	}
	return value
}
