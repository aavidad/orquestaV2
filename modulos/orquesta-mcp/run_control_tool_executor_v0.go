package orquestamcp

import (
	"context"
	"strings"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
)

type MCPRunControlToolExecutorV0 struct {
	Port orquestaruncontrol.RunControlWriterPortV0
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
	if strings.TrimSpace(input.RunRef) == "" {
		return newMCPRunControlErrorV0(input, "run_ref_requerido", "run_ref"), nil
	}
	if executor.Port == nil {
		return newMCPRunControlErrorV0(input, "run_control_port_no_disponible", "port"), nil
	}
	if !isMCPRunControlActionSupportedV0(input.Action) {
		return newMCPRunControlErrorV0(input, "action_no_soportada", "action"), nil
	}
	state, err := executor.executeActionV0(ctx, input)
	if err != nil {
		return MCPRunControlToolResultV0{}, err
	}
	return newMCPRunControlResultV0(input, state), nil
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
