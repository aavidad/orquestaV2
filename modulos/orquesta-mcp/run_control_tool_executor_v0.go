package orquestamcp

import (
	"context"
	"strings"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
)

type MCPRunControlToolExecutorV0 struct {
	Port              orquestaruncontrol.RunControlWriterPortV0
	ExternalJobSource MCPDirectorExternalJobStatsSourcePortV0
}

func NewMCPRunControlToolExecutorV0(
	port orquestaruncontrol.RunControlWriterPortV0,
) MCPRunControlToolExecutorV0 {
	return MCPRunControlToolExecutorV0{Port: port}
}

func (executor MCPRunControlToolExecutorV0) Execute(
	ctx context.Context,
	input MCPRunControlToolInputV0,
) (MCPRunControlToolResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	resolved, ok, err := executor.resolveRunControlInputV0(ctx, input)
	if err != nil {
		return newMCPRunControlErrorV0(input, "external_job_run_ref_error", "external_job_ref"), nil
	}
	if !ok || strings.TrimSpace(resolved.RunRef) == "" {
		return newMCPRunControlErrorV0(input, "run_ref_requerido", "run_ref"), nil
	}
	if executor.Port == nil {
		return newMCPRunControlErrorV0(resolved, "run_control_port_no_disponible", "port"), nil
	}
	if !isMCPRunControlActionSupportedV0(input.Action) {
		return newMCPRunControlErrorV0(resolved, "action_no_soportada", "action"), nil
	}
	state, err := executor.executeActionV0(ctx, resolved)
	if err != nil {
		return MCPRunControlToolResultV0{}, err
	}
	return newMCPRunControlResultV0(resolved, state), nil
}

func (executor MCPRunControlToolExecutorV0) resolveRunControlInputV0(
	ctx context.Context,
	input MCPRunControlToolInputV0,
) (MCPRunControlToolInputV0, bool, error) {
	input.RunRef = strings.TrimSpace(input.RunRef)
	if input.RunRef != "" {
		return input, true, nil
	}
	if strings.TrimSpace(input.ExternalJobRef) == "" {
		return input, false, nil
	}
	if executor.ExternalJobSource == nil {
		return input, false, nil
	}
	stats, ok, err := executor.ExternalJobSource.ResolveDirectorExternalJobStatsV0(
		ctx,
		MCPDirectorExternalJobStatsRequestV0{
			AppRef:         strings.TrimSpace(input.AppRef),
			ExternalJobRef: strings.TrimSpace(input.ExternalJobRef),
			CorrelationID:  firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		},
	)
	if err != nil || !ok || strings.TrimSpace(stats.RunRef) == "" {
		return input, ok, err
	}
	input.RunRef = strings.TrimSpace(stats.RunRef)
	input.EvidenceRefs = compactStringsMCPV0(append(input.EvidenceRefs, stats.JobRef, stats.TaskRef, stats.AgentRef))
	return input, true, nil
}

func (executor MCPRunControlToolExecutorV0) executeActionV0(
	ctx context.Context,
	input MCPRunControlToolInputV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	switch normalizeMCPRunControlActionV0(input.Action) {
	case "pause":
		return executor.Port.PauseRunV0(ctx, orquestaruncontrol.PauseRunCommandV0{
			RunRef:         strings.TrimSpace(input.RunRef),
			RequestedBy:    strings.TrimSpace(input.RequestedBy),
			Reason:         strings.TrimSpace(input.Reason),
			IdempotencyKey: strings.TrimSpace(input.IdempotencyKey),
			EvidenceRefs:   compactStringsMCPV0(input.EvidenceRefs),
		})
	case "resume":
		return executor.Port.ResumeRunV0(ctx, orquestaruncontrol.ResumeRunCommandV0{
			RunRef:         strings.TrimSpace(input.RunRef),
			RequestedBy:    strings.TrimSpace(input.RequestedBy),
			Reason:         strings.TrimSpace(input.Reason),
			IdempotencyKey: strings.TrimSpace(input.IdempotencyKey),
			EvidenceRefs:   compactStringsMCPV0(input.EvidenceRefs),
		})
	case "stop":
		return executor.Port.StopRunV0(ctx, orquestaruncontrol.StopRunCommandV0{
			RunRef:         strings.TrimSpace(input.RunRef),
			RequestedBy:    strings.TrimSpace(input.RequestedBy),
			Reason:         strings.TrimSpace(input.Reason),
			Forced:         input.Forced,
			IdempotencyKey: strings.TrimSpace(input.IdempotencyKey),
			EvidenceRefs:   compactStringsMCPV0(input.EvidenceRefs),
		})
	case "cancel":
		return executor.Port.CancelRunV0(ctx, orquestaruncontrol.CancelRunCommandV0{
			RunRef:         strings.TrimSpace(input.RunRef),
			RequestedBy:    strings.TrimSpace(input.RequestedBy),
			Reason:         strings.TrimSpace(input.Reason),
			Forced:         input.Forced,
			IdempotencyKey: strings.TrimSpace(input.IdempotencyKey),
			EvidenceRefs:   compactStringsMCPV0(input.EvidenceRefs),
		})
	default:
		return orquestaruncontrol.RunControlStateV0{}, nil
	}
}

func isMCPRunControlActionSupportedV0(action string) bool {
	switch normalizeMCPRunControlActionV0(action) {
	case "pause", "resume", "stop", "cancel":
		return true
	default:
		return false
	}
}
