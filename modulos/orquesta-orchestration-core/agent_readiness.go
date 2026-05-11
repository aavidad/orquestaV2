package orquestacionnucleoapp

import (
	"context"
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

type AgentReadinessProbeStatusV0 string

const (
	AgentReadinessReadyV0    AgentReadinessProbeStatusV0 = "ready"
	AgentReadinessNotReadyV0 AgentReadinessProbeStatusV0 = "not_ready"
)

type AgentReadinessProbePortV0 interface {
	AwaitAgentReadinessV0(
		context.Context,
		AgentReadinessProbeRequestV0,
	) (AgentReadinessProbeResultV0, error)
}

type AgentReadinessProbeRequestV0 struct {
	RunID          string
	AgentRequestID string
	ProcessRef     string
	SessionRef     string
	LaunchRef      string
	ReadinessRef   string
	EvidenceRefs   []string
}

type AgentReadinessProbeResultV0 struct {
	Status       AgentReadinessProbeStatusV0
	ReadinessRef string
	EvidenceRefs []string
}

func ProbeAgentReadinessV0(
	ctx context.Context,
	probe AgentReadinessProbePortV0,
	request AgentReadinessProbeRequestV0,
) (AgentReadinessProbeResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return AgentReadinessProbeResultV0{}, err
	}
	request = normalizeAgentReadinessProbeRequestV0(request)
	if probe == nil {
		return agentReadinessProbeSkippedV0(request), nil
	}
	if err := validateAgentReadinessProbeRequestV0(request); err != nil {
		return AgentReadinessProbeResultV0{}, err
	}
	result, err := probe.AwaitAgentReadinessV0(ctx, request)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return AgentReadinessProbeResultV0{}, ctxErr
		}
		return AgentReadinessProbeResultV0{}, errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"readiness_probe",
			err.Error(),
		)
	}
	result = normalizeAgentReadinessProbeResultV0(request, result)
	if err := validateAgentReadinessProbeResultV0(request, result); err != nil {
		return AgentReadinessProbeResultV0{}, err
	}
	return result, nil
}

func agentReadinessProbeRequestFromLaunchV0(
	inbound orquestaruntime.AgentLauncherInboundV0,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
	snapshot orquestaruntime.ProcessRuntimeSnapshotV0,
) AgentReadinessProbeRequestV0 {
	delivery := spec.AgentPacket.DeliveryRefs
	request := AgentReadinessProbeRequestV0{
		ProcessRef:   snapshot.ProcessRef,
		SessionRef:   snapshot.SessionRef,
		LaunchRef:    snapshot.LaunchRef,
		ReadinessRef: delivery.ReadinessRef,
		EvidenceRefs: []string{snapshot.ProcessRef, snapshot.SessionRef, snapshot.LaunchRef, delivery.ReadinessRef},
	}
	if inbound.Payload != nil {
		request.RunID = inbound.Payload.RunID
		request.AgentRequestID = inbound.Payload.AgentRequestID
	}
	return normalizeAgentReadinessProbeRequestV0(request)
}

func mergeAgentReadinessEvidenceV0(
	launch AgentLaunchResultV0,
	readiness AgentReadinessProbeResultV0,
) AgentLaunchResultV0 {
	if readiness.ReadinessRef != "" {
		launch.ReadinessRef = readiness.ReadinessRef
	}
	launch.EvidenceRefs = compactStringsV0(append(launch.EvidenceRefs, readiness.EvidenceRefs...))
	return launch
}

func agentReadinessProbeSkippedV0(
	request AgentReadinessProbeRequestV0,
) AgentReadinessProbeResultV0 {
	return AgentReadinessProbeResultV0{
		Status:       AgentReadinessReadyV0,
		ReadinessRef: request.ReadinessRef,
		EvidenceRefs: request.EvidenceRefs,
	}
}

func normalizeAgentReadinessProbeRequestV0(
	request AgentReadinessProbeRequestV0,
) AgentReadinessProbeRequestV0 {
	request.RunID = strings.TrimSpace(request.RunID)
	request.AgentRequestID = strings.TrimSpace(request.AgentRequestID)
	request.ProcessRef = strings.TrimSpace(request.ProcessRef)
	request.SessionRef = strings.TrimSpace(request.SessionRef)
	request.LaunchRef = strings.TrimSpace(request.LaunchRef)
	request.ReadinessRef = strings.TrimSpace(request.ReadinessRef)
	request.EvidenceRefs = compactStringsV0(request.EvidenceRefs)
	return request
}

func normalizeAgentReadinessProbeResultV0(
	request AgentReadinessProbeRequestV0,
	result AgentReadinessProbeResultV0,
) AgentReadinessProbeResultV0 {
	result.ReadinessRef = strings.TrimSpace(result.ReadinessRef)
	if result.ReadinessRef == "" {
		result.ReadinessRef = request.ReadinessRef
	}
	result.EvidenceRefs = compactStringsV0(result.EvidenceRefs)
	return result
}
