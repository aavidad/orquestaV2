package orquestacionnucleoapp

import (
	"context"
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

type ExternalAgentLaunchSpecResolverPortV0 interface {
	ResolveExternalAgentLaunchSpecV0(
		context.Context,
		orquestaruntime.AgentLauncherInboundV0,
	) (ExternalAgentLaunchSpecResolutionV0, error)
}

type ExternalAgentLaunchSpecResolutionV0 struct {
	Spec            orquestaruntime.ExternalAgentLaunchSpecV0
	CommandResolver orquestaruntime.ExternalAgentProcessCommandResolverV0
}

type ExternalProcessAgentLauncherV0 struct {
	SpecResolver    ExternalAgentLaunchSpecResolverPortV0
	Runtime         orquestaruntime.ExternalAgentProcessRuntimePortV0
	ProcessStopper  ProcessRuntimeStopPortV0
	Readiness       AgentReadinessProbePortV0
	ProcessRegistry AgentProcessRegistryPortV0
}

var _ AgentLauncherPortV0 = ExternalProcessAgentLauncherV0{}

func (launcher ExternalProcessAgentLauncherV0) LaunchAgentV0(
	ctx context.Context,
	inbound orquestaruntime.AgentLauncherInboundV0,
) (AgentLaunchResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return AgentLaunchResultV0{}, err
	}
	if err := launcher.validateV0(inbound); err != nil {
		return AgentLaunchResultV0{}, err
	}
	resolution, err := launcher.SpecResolver.ResolveExternalAgentLaunchSpecV0(ctx, inbound)
	if err != nil {
		return AgentLaunchResultV0{}, err
	}
	started := orquestaruntime.LaunchExternalAgentProcessV0(
		ctx,
		resolution.Spec,
		resolution.CommandResolver,
		launcher.Runtime,
	)
	if started.Status != orquestaruntime.ExternalAgentProcessLaunchStartedV0 {
		return AgentLaunchResultV0{}, externalProcessAgentLauncherBlockedErrorV0(started)
	}
	readiness, err := ProbeAgentReadinessV0(
		ctx,
		launcher.Readiness,
		agentReadinessProbeRequestFromLaunchV0(inbound, resolution.Spec, started.Snapshot),
	)
	if err != nil {
		cleanupStartedAgentProcessV0(launcher.ProcessStopper, started.Snapshot)
		return AgentLaunchResultV0{}, err
	}
	result := externalProcessAgentLaunchResultV0(inbound, resolution.Spec, started.Snapshot)
	result = mergeAgentReadinessEvidenceV0(result, readiness)
	if err := launcher.registerAgentProcessV0(ctx, inbound, started.Snapshot, result); err != nil {
		cleanupStartedAgentProcessV0(launcher.ProcessStopper, started.Snapshot)
		return AgentLaunchResultV0{}, err
	}
	return result, nil
}

func (launcher ExternalProcessAgentLauncherV0) validateV0(
	inbound orquestaruntime.AgentLauncherInboundV0,
) error {
	if launcher.SpecResolver == nil {
		return errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"external_agent_launch_spec_resolver",
			"external_agent_launch_spec_resolver requerido",
		)
	}
	if launcher.Runtime == nil {
		return errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"external_agent_process_runtime",
			"external_agent_process_runtime requerido",
		)
	}
	if launcher.ProcessStopper == nil {
		return errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"external_agent_process_stopper",
			"external_agent_process_stopper requerido",
		)
	}
	if launcher.ProcessRegistry == nil {
		return errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"agent_process_registry",
			"agent_process_registry requerido",
		)
	}
	if issues := orquestaruntime.ValidateAgentLauncherInboundV0(inbound); len(issues) > 0 {
		return errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"agent_launcher_inbound."+issues[0].Field,
			string(issues[0].Code),
		)
	}
	return nil
}

func externalProcessAgentLaunchResultV0(
	inbound orquestaruntime.AgentLauncherInboundV0,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
	snapshot orquestaruntime.ProcessRuntimeSnapshotV0,
) AgentLaunchResultV0 {
	delivery := spec.AgentPacket.DeliveryRefs
	result := AgentLaunchResultV0{
		AgentRequestID: externalProcessAgentRequestIDV0(inbound),
		LaunchRef:      strings.TrimSpace(snapshot.LaunchRef),
		AckRef:         strings.TrimSpace(delivery.AckRef),
		ReadinessRef:   strings.TrimSpace(delivery.ReadinessRef),
		EvidenceRefs: compactStringsV0([]string{
			snapshot.ProcessRef,
			snapshot.SessionRef,
			snapshot.LaunchRef,
			delivery.MailboxRef,
			delivery.AckRef,
			delivery.ReadinessRef,
			delivery.CheckpointRef,
		}),
	}
	result.EvidenceRefs = compactStringsV0(append(
		result.EvidenceRefs,
		orquestaruntime.ProcessRuntimeSnapshotEvidenceRefsV0(snapshot)...,
	))
	return result
}

func externalProcessAgentRequestIDV0(
	inbound orquestaruntime.AgentLauncherInboundV0,
) string {
	if inbound.Payload == nil {
		return ""
	}
	return strings.TrimSpace(inbound.Payload.AgentRequestID)
}

func (launcher ExternalProcessAgentLauncherV0) registerAgentProcessV0(
	ctx context.Context,
	inbound orquestaruntime.AgentLauncherInboundV0,
	snapshot orquestaruntime.ProcessRuntimeSnapshotV0,
	launch AgentLaunchResultV0,
) error {
	if launcher.ProcessRegistry == nil {
		return nil
	}
	record := agentProcessRegistryRecordFromLaunchV0(inbound, snapshot, launch)
	if err := launcher.ProcessRegistry.RecordAgentProcessV0(ctx, record); err != nil {
		return errorV0(
			ErrNucleoOrquestacionStoreV0,
			"agent_process_registry",
			err.Error(),
		)
	}
	return nil
}

func agentProcessRegistryRecordFromLaunchV0(
	inbound orquestaruntime.AgentLauncherInboundV0,
	snapshot orquestaruntime.ProcessRuntimeSnapshotV0,
	launch AgentLaunchResultV0,
) AgentProcessRegistryRecordV0 {
	record := AgentProcessRegistryRecordV0{
		ProcessRef:   strings.TrimSpace(snapshot.ProcessRef),
		SessionRef:   strings.TrimSpace(snapshot.SessionRef),
		LaunchRef:    strings.TrimSpace(launch.LaunchRef),
		PID:          snapshot.PID,
		ReadinessRef: strings.TrimSpace(launch.ReadinessRef),
		EvidenceRefs: compactStringsV0(append([]string{
			snapshot.ProcessRef,
			snapshot.SessionRef,
			launch.LaunchRef,
			launch.ReadinessRef,
		}, launch.EvidenceRefs...)),
	}
	if inbound.Payload != nil {
		record.RunID = strings.TrimSpace(inbound.Payload.RunID)
		record.AgentRequestID = strings.TrimSpace(inbound.Payload.AgentRequestID)
	}
	return record
}

func externalProcessAgentLauncherBlockedErrorV0(
	result orquestaruntime.ExternalAgentProcessLaunchResultV0,
) error {
	if len(result.Issues) == 0 {
		return errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"external_agent_process",
			string(result.Status),
		)
	}
	issue := result.Issues[0]
	field := strings.TrimSpace(issue.Field)
	if field == "" {
		field = "external_agent_process"
	}
	return errorV0(
		ErrNucleoOrquestacionInvalidoV0,
		field,
		externalProcessAgentIssueMessageV0(issue),
	)
}

func externalProcessAgentIssueMessageV0(
	issue orquestaruntime.ExternalAgentConnectorErrorV0,
) string {
	message := string(issue.Code)
	evidence := compactStringsV0(issue.Evidence)
	if len(evidence) == 0 {
		return message
	}
	return message + ": " + strings.Join(evidence, ",")
}
