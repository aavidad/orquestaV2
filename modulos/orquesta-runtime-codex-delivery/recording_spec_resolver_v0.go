package orquestaruntimecodexdelivery

import (
	"context"
	"fmt"
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestacionnucleoapp "orquesta/orquestacionnucleoapp"
)

type CodexReceiptAckPathResolverPortV0 interface {
	ResolveCodexReceiptAckPathV0(
		context.Context,
		CodexReceiptAckPathRequestV0,
	) (CodexReceiptAckPathResolutionV0, error)
}

type CodexReceiptAckPathRequestV0 struct {
	DescriptorRef string
	RunID         string
	AgentRef      string
	Spec          orquestaruntime.ExternalAgentLaunchSpecV0
}

type CodexReceiptAckPathResolutionV0 struct {
	AckPath        string
	ProjectWorkDir string
}

type CodexReceiptRecordingSpecResolverV0 struct {
	Inner                    orquestacionnucleoapp.ExternalAgentLaunchSpecResolverPortV0
	Recorder                 CodexReceiptDescriptorRecorderPortV0
	AckPathResolver          CodexReceiptAckPathResolverPortV0
	WorktreeBaselineRecorder CodexReceiptWorktreeBaselineRecorderPortV0
}

var _ orquestacionnucleoapp.ExternalAgentLaunchSpecResolverPortV0 = CodexReceiptRecordingSpecResolverV0{}

func (resolver CodexReceiptRecordingSpecResolverV0) ResolveExternalAgentLaunchSpecV0(
	ctx context.Context,
	inbound orquestaruntime.AgentLauncherInboundV0,
) (orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0, error) {
	if resolver.Inner == nil {
		return orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0{},
			fmt.Errorf("codex_receipt_recording_spec_resolver: inner requerido")
	}
	if resolver.Recorder == nil {
		return orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0{},
			fmt.Errorf("codex_receipt_recording_spec_resolver: recorder requerido")
	}
	if resolver.AckPathResolver == nil {
		return orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0{},
			fmt.Errorf("codex_receipt_recording_spec_resolver: ack_path_resolver requerido")
	}
	resolution, err := resolver.Inner.ResolveExternalAgentLaunchSpecV0(ctx, inbound)
	if err != nil {
		return resolution, err
	}
	descriptor, err := resolver.descriptorFromResolutionV0(ctx, inbound, resolution)
	if err != nil {
		return resolution, err
	}
	if err := resolver.Recorder.RecordCodexReceiptDescriptorV0(ctx, descriptor); err != nil {
		return resolution, fmt.Errorf("codex_receipt_descriptor_record: failed")
	}
	return resolution, nil
}

func (resolver CodexReceiptRecordingSpecResolverV0) descriptorFromResolutionV0(
	ctx context.Context,
	inbound orquestaruntime.AgentLauncherInboundV0,
	resolution orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0,
) (CodexReceiptDescriptorV0, error) {
	runID, agentRef := codexReceiptRunAndAgentFromInboundV0(inbound)
	descriptorRef := codexReceiptDescriptorRefV0(runID, agentRef)
	path, err := resolver.AckPathResolver.ResolveCodexReceiptAckPathV0(
		ctx,
		CodexReceiptAckPathRequestV0{
			DescriptorRef: descriptorRef,
			RunID:         runID,
			AgentRef:      agentRef,
			Spec:          resolution.Spec,
		},
	)
	if err != nil {
		return CodexReceiptDescriptorV0{}, fmt.Errorf("codex_receipt_ack_path: failed")
	}
	descriptor := CodexReceiptDescriptorV0{
		DescriptorRef:  descriptorRef,
		RunID:          runID,
		AgentRef:       agentRef,
		Spec:           resolution.Spec,
		AckPath:        strings.TrimSpace(path.AckPath),
		ProjectWorkDir: strings.TrimSpace(path.ProjectWorkDir),
	}
	if resolver.WorktreeBaselineRecorder == nil {
		return descriptor, nil
	}
	baseline, err := resolver.WorktreeBaselineRecorder.CaptureCodexReceiptWorktreeBaselineV0(
		ctx,
		CodexReceiptWorktreeBaselineRequestV0{
			DescriptorRef:  descriptorRef,
			RunID:          runID,
			AgentRef:       agentRef,
			ProjectWorkDir: descriptor.ProjectWorkDir,
			Spec:           resolution.Spec,
		},
	)
	if err != nil {
		return CodexReceiptDescriptorV0{}, err
	}
	descriptor.WorktreeBaselineRef = strings.TrimSpace(baseline.BaselineRef)
	return descriptor, nil
}

func codexReceiptRunAndAgentFromInboundV0(
	inbound orquestaruntime.AgentLauncherInboundV0,
) (string, string) {
	if inbound.Payload == nil {
		return "", ""
	}
	return strings.TrimSpace(inbound.Payload.RunID), strings.TrimSpace(inbound.Payload.AgentRequestID)
}

func codexReceiptDescriptorRefV0(runID string, agentRef string) string {
	runID = strings.TrimSpace(runID)
	agentRef = strings.TrimSpace(agentRef)
	if runID == "" {
		return "codex-receipt-ref-" + agentRef
	}
	return "codex-receipt-ref-" + runID + "-" + agentRef
}

type StaticCodexReceiptAckPathResolverV0 struct {
	AckPath        string
	ProjectWorkDir string
}

func (resolver StaticCodexReceiptAckPathResolverV0) ResolveCodexReceiptAckPathV0(
	context.Context,
	CodexReceiptAckPathRequestV0,
) (CodexReceiptAckPathResolutionV0, error) {
	if strings.TrimSpace(resolver.AckPath) == "" {
		return CodexReceiptAckPathResolutionV0{}, fmt.Errorf("ack_path requerido")
	}
	return CodexReceiptAckPathResolutionV0{
		AckPath:        strings.TrimSpace(resolver.AckPath),
		ProjectWorkDir: strings.TrimSpace(resolver.ProjectWorkDir),
	}, nil
}
