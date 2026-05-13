package orquestaappcodexstack

import (
	"context"
	"path/filepath"
	"strings"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
	orquestaservershutdown "orquesta/modulos/orquesta-server-shutdown"
)

type stackShutdownCheckpointPreparerV0 struct {
	Config ConfigV0
}

func (preparer stackShutdownCheckpointPreparerV0) PrepareAgentShutdownV0(
	ctx context.Context,
	command orquestaservershutdown.PrepareAgentShutdownCommandV0,
) (orquestaservershutdown.PrepareAgentShutdownResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	stats, err := preparer.fullShutdownStatsV0(ctx, command)
	if err != nil {
		return orquestaservershutdown.PrepareAgentShutdownResultV0{}, err
	}
	inFlight := stackShutdownInFlightAgentRefsV0(stats)
	if len(inFlight) == 0 {
		return stackShutdownRecordedCheckpointResultV0(command), nil
	}
	descriptors, err := preparer.Config.Stores.ReceiptStore.ListCodexReceiptDescriptorsV0(
		ctx,
		orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0{
			RunID:         strings.TrimSpace(command.RunRef),
			StartedAgents: inFlight,
			CorrelationID: strings.TrimSpace(command.CorrelationID),
			EvidenceRefs:  compactStringsV0(command.EvidenceRefs),
		},
	)
	if err != nil {
		return orquestaservershutdown.PrepareAgentShutdownResultV0{}, err
	}
	return preparer.prepareInFlightAgentCheckpointsV0(ctx, command, inFlight, descriptors), nil
}

func (preparer stackShutdownCheckpointPreparerV0) fullShutdownStatsV0(
	ctx context.Context,
	command orquestaservershutdown.PrepareAgentShutdownCommandV0,
) (orquestacionnucleoapp.DirectorRunStatsV0, error) {
	run, err := preparer.Config.Stores.RunStore.LoadRunV0(ctx, command.RunRef)
	if err != nil {
		return orquestacionnucleoapp.DirectorRunStatsV0{}, err
	}
	return orquestacionnucleoapp.BuildDirectorRunStatsWithTelemetryPortsV0(
		ctx,
		run,
		preparer.Config.Stores.ProcessRegistry,
		statsProgressSourceV0(preparer.Config),
		agentUsageSourceV0(preparer.Config),
		orquestacionnucleoapp.DirectorProgressSourceRequestV0{
			CorrelationID: command.CorrelationID,
			EvidenceRefs:  command.EvidenceRefs,
		},
	), nil
}

func (preparer stackShutdownCheckpointPreparerV0) prepareInFlightAgentCheckpointsV0(
	ctx context.Context,
	command orquestaservershutdown.PrepareAgentShutdownCommandV0,
	inFlight []string,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) orquestaservershutdown.PrepareAgentShutdownResultV0 {
	byAgent := stackShutdownDescriptorsByAgentV0(descriptors)
	pending := make([]string, 0, len(inFlight))
	evidence := compactStringsV0(command.EvidenceRefs)
	for _, agentRef := range inFlight {
		descriptor, ok := byAgent[agentRef]
		if !ok {
			pending = append(pending, agentRef)
			evidence = append(evidence, "shutdown-checkpoint-descriptor-missing-"+safeStackShutdownRefPartV0(agentRef))
			continue
		}
		ready, refs := preparer.prepareAgentCheckpointV0(ctx, command, descriptor)
		evidence = append(evidence, refs...)
		if !ready {
			pending = append(pending, agentRef)
		}
	}
	if len(pending) > 0 {
		return orquestaservershutdown.PrepareAgentShutdownResultV0{
			RunRef:             strings.TrimSpace(command.RunRef),
			CheckpointRecorded: false,
			PendingAgentRefs:   compactStringsV0(pending),
			EvidenceRefs:       compactStringsV0(evidence),
		}
	}
	checkpointRef := "checkpoint-ref-shutdown-" + safeStackShutdownRefPartV0(command.RunRef)
	return orquestaservershutdown.PrepareAgentShutdownResultV0{
		RunRef:             strings.TrimSpace(command.RunRef),
		CheckpointRecorded: true,
		CheckpointRef:      checkpointRef,
		EvidenceRefs:       compactStringsV0(append(evidence, checkpointRef)),
	}
}

func (preparer stackShutdownCheckpointPreparerV0) prepareAgentCheckpointV0(
	_ context.Context,
	command orquestaservershutdown.PrepareAgentShutdownCommandV0,
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) (bool, []string) {
	runtimeDir, ok := stackShutdownRuntimeDirV0(descriptor)
	if !ok {
		return false, []string{"shutdown-checkpoint-runtime-dir-invalid-" + safeStackShutdownRefPartV0(descriptor.AgentRef)}
	}
	request := stackShutdownRequestForAgentV0(command, descriptor)
	requestPath := filepath.Join(runtimeDir, orquestaruntimecodex.CodexShutdownRequestFileNameV0)
	if issues := orquestaruntimecodex.WriteCodexShutdownRequestFileV0(requestPath, request); len(issues) > 0 {
		return false, stackShutdownIssueEvidenceRefsV0(issues)
	}
	ackPath := filepath.Join(runtimeDir, orquestaruntimecodex.CodexShutdownCheckpointAckFileNameV0)
	ack, issues := orquestaruntimecodex.ReadCodexShutdownCheckpointAckFileV0(ackPath, request)
	if len(issues) > 0 {
		return false, stackShutdownIssueEvidenceRefsV0(issues)
	}
	return true, compactStringsV0(append(ack.EvidenceRefs, ack.CheckpointRef))
}

func stackShutdownRequestForAgentV0(
	command orquestaservershutdown.PrepareAgentShutdownCommandV0,
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) orquestaruntimecodex.CodexShutdownRequestV0 {
	agentRef := strings.TrimSpace(descriptor.AgentRef)
	refs := append([]string(nil), command.EvidenceRefs...)
	refs = append(refs, descriptor.DescriptorRef)
	return orquestaruntimecodex.CodexShutdownRequestV0{
		SchemaVersion: orquestaruntimecodex.CodexShutdownRequestSchemaVersionV0,
		RunRef:        strings.TrimSpace(command.RunRef),
		AgentRef:      agentRef,
		CorrelationID: strings.TrimSpace(command.CorrelationID),
		RequestedBy:   strings.TrimSpace(command.RequestedBy),
		Reason:        strings.TrimSpace(command.Reason),
		CheckpointRef: "checkpoint-ref-shutdown-" + safeStackShutdownRefPartV0(command.RunRef) + "-" + safeStackShutdownRefPartV0(agentRef),
		EvidenceRefs:  compactStringsV0(refs),
	}
}

func stackShutdownInFlightAgentRefsV0(
	stats orquestacionnucleoapp.DirectorRunStatsV0,
) []string {
	refs := make([]string, 0, len(stats.Agents))
	for _, agent := range stats.Agents {
		if agent.InFlight {
			refs = append(refs, agent.AgentRequestID)
		}
	}
	return compactStringsV0(refs)
}

func stackShutdownDescriptorsByAgentV0(
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) map[string]orquestaruntimecodexdelivery.CodexReceiptDescriptorV0 {
	byAgent := make(map[string]orquestaruntimecodexdelivery.CodexReceiptDescriptorV0, len(descriptors))
	for _, descriptor := range descriptors {
		agentRef := strings.TrimSpace(descriptor.AgentRef)
		if agentRef != "" {
			byAgent[agentRef] = descriptor
		}
	}
	return byAgent
}

func stackShutdownRuntimeDirV0(
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) (string, bool) {
	ackPath := strings.TrimSpace(descriptor.AckPath)
	if ackPath == "" || !filepath.IsAbs(ackPath) ||
		filepath.Base(ackPath) == ackPath {
		return "", false
	}
	return filepath.Dir(ackPath), true
}

func stackShutdownRecordedCheckpointResultV0(
	command orquestaservershutdown.PrepareAgentShutdownCommandV0,
) orquestaservershutdown.PrepareAgentShutdownResultV0 {
	checkpointRef := "checkpoint-ref-shutdown-" + safeStackShutdownRefPartV0(command.RunRef)
	refs := append([]string(nil), command.EvidenceRefs...)
	refs = append(refs, checkpointRef)
	return orquestaservershutdown.PrepareAgentShutdownResultV0{
		RunRef:             strings.TrimSpace(command.RunRef),
		CheckpointRecorded: true,
		CheckpointRef:      checkpointRef,
		EvidenceRefs:       compactStringsV0(refs),
	}
}

func stackShutdownIssueEvidenceRefsV0(
	issues []orquestaruntime.ExternalAgentConnectorErrorV0,
) []string {
	refs := make([]string, 0, len(issues))
	for _, issue := range issues {
		status := "invalid"
		if issue.Retryable {
			status = "pending"
		}
		ref := "shutdown-checkpoint-issue-" + status + "-" +
			safeStackShutdownRefPartV0(issue.Field)
		refs = append(refs, ref)
	}
	return compactStringsV0(refs)
}

func safeStackShutdownRefPartV0(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\\", "-")
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, " ", "-")
	if value == "" {
		return "unknown"
	}
	return value
}
