package orquestamcp

import (
	"errors"
	"strings"

	orquestaappplanner "orquesta/modulos/orquesta-app-planner"
	orquestaapprunner "orquesta/modulos/orquesta-app-runner"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func ExecuteMCPPrepararOrquestacionAppToolV0(
	input MCPPrepararOrquestacionAppToolInputV0,
) MCPPrepararOrquestacionAppToolResultV0 {
	request := ToPrepareAppOrchestrationRequestMCPV0(input)
	prepared, err := orquestaapprunner.PrepareAppOrchestrationV0(request)
	if err != nil {
		return NewMCPPrepararOrquestacionAppErrorResultV0(input, err)
	}
	return NewMCPPrepararOrquestacionAppOKResultV0(input, prepared)
}

func NewMCPPrepararOrquestacionAppOKResultV0(
	input MCPPrepararOrquestacionAppToolInputV0,
	prepared orquestaapprunner.AppOrchestrationPreparedV0,
) MCPPrepararOrquestacionAppToolResultV0 {
	return MCPPrepararOrquestacionAppToolResultV0{
		Estado:        MCPPrepararOrquestacionAppEstadoOKV0,
		RequestID:     firstNonEmptyMCPV0(input.RequestID, prepared.Run.AppSpecRef, input.AppSpec.RequestID),
		CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, "corr-"+prepared.Run.RunID),
		AppSpec:       compactAppSpecV0(input.AppSpec),
		RunRef:        strings.TrimSpace(prepared.Run.RunID),
		PhaseID:       strings.TrimSpace(string(prepared.Run.CurrentPhase)),
		Plan:          compactAppPlanMCPV0(prepared.Plan),
		Progress:      compactAppPlanProgressMCPV0(prepared.InitialProgress),
		EvidenceRefs:  compactStringsMCPV0(prepared.EvidenceRefs),
		Errores:       []MCPValidationIssueV0{},
	}
}

func NewMCPPrepararOrquestacionAppErrorResultV0(
	input MCPPrepararOrquestacionAppToolInputV0,
	err error,
) MCPPrepararOrquestacionAppToolResultV0 {
	return MCPPrepararOrquestacionAppToolResultV0{
		Estado:        MCPPrepararOrquestacionAppEstadoErrorV0,
		RequestID:     firstNonEmptyMCPV0(input.RequestID, input.AppSpec.RequestID),
		CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		Errores:       []MCPValidationIssueV0{publicPrepareOrchestrationIssueMCPV0(err)},
	}
}

func compactAppPlanMCPV0(plan orquestaappplanner.AppMicrotaskPlanV0) MCPAppPlanCompactV0 {
	return MCPAppPlanCompactV0{
		SchemaVersion: strings.TrimSpace(plan.SchemaVersion),
		RunRef:        strings.TrimSpace(plan.RunRef),
		AppRef:        strings.TrimSpace(plan.AppRef),
		Units:         len(plan.Units),
		UnitRefs:      compactAppPlanUnitsMCPV0(plan.Units),
		EvidenceRefs:  compactStringsMCPV0(plan.EvidenceRefs),
	}
}

func compactAppPlanUnitsMCPV0(units []orquestaappplanner.AppWorkUnitV0) []MCPAppPlanUnitMCPV0 {
	out := make([]MCPAppPlanUnitMCPV0, 0, len(units))
	for _, unit := range units {
		out = append(out, MCPAppPlanUnitMCPV0{
			TaskRef:             strings.TrimSpace(unit.TaskRef),
			Title:               strings.TrimSpace(unit.Title),
			PhaseID:             strings.TrimSpace(string(unit.PhaseID)),
			Role:                strings.TrimSpace(unit.Role),
			Capacity:            strings.TrimSpace(string(unit.Capacity)),
			DeliveryRef:         strings.TrimSpace(unit.DeliveryRef),
			DependsOnDeliveries: compactStringsMCPV0(unit.DependsOnDeliveries),
			WriteSet:            compactStringsMCPV0(unit.WriteSet),
		})
	}
	if out == nil {
		return []MCPAppPlanUnitMCPV0{}
	}
	return out
}

func compactAppPlanProgressMCPV0(
	progress orquestaappplanner.AppPlanProgressV0,
) MCPAppPlanProgressMCPV0 {
	return MCPAppPlanProgressMCPV0{
		TotalUnits:        progress.TotalUnits,
		DeliveredUnits:    progress.DeliveredUnits,
		Complete:          progress.Complete,
		ReadyTaskRefs:     compactStringsMCPV0(progress.ReadyTaskRefs),
		BlockedTaskRefs:   compactStringsMCPV0(progress.BlockedTaskRefs),
		PendingTaskRefs:   compactStringsMCPV0(progress.PendingTaskRefs),
		DeliveredTaskRefs: compactStringsMCPV0(progress.DeliveredTaskRefs),
	}
}

func publicPrepareOrchestrationIssueMCPV0(err error) MCPValidationIssueV0 {
	var runnerIssue orquestaapprunner.AppRunnerIssueV0
	field := ""
	if errors.As(err, &runnerIssue) {
		field = runnerIssue.Field
	}
	var coreIssue orquestacionnucleoapp.ErrorV0
	if field == "" && errors.As(err, &coreIssue) {
		field = coreIssue.Field
	}
	return MCPValidationIssueV0{
		Code:    MCPPrepararOrquestacionAppErrorCodeV0,
		Field:   strings.TrimSpace(field),
		Message: MCPPrepararOrquestacionAppErrorMsgV0,
	}
}
