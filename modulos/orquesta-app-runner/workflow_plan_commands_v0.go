package orquestaapprunner

import (
	"strings"

	orquestaappplanner "orquesta/modulos/orquesta-app-planner"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func appRunVotePayloadV0(
	request PrepareAppOrchestrationRequestV0,
) orquestacoreworkflow.RequestVoteCommandPayloadV0 {
	return orquestacoreworkflow.RequestVoteCommandPayloadV0{
		VoteRequestID:    appRunVoteRefV0(request),
		PhaseID:          string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0),
		DecisionTopicRef: "topic-" + safeAppRunnerRefPartV0(request.AppSpec.App.Slug),
		BrainstormRef:    "brainstorm-" + safeAppRunnerRefPartV0(request.AppSpec.App.Slug),
		Summary:          "Elegir plan compacto de app.",
		EvidenceRefs:     []string{"evidence-ref-app-runner-vote-v0"},
	}
}

func appRunAcceptDecisionPayloadV0(
	request PrepareAppOrchestrationRequestV0,
) orquestacoreworkflow.AcceptDecisionCommandPayloadV0 {
	return orquestacoreworkflow.AcceptDecisionCommandPayloadV0{
		DecisionRef:       appRunDecisionRefV0(request),
		PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0),
		VoteRef:           appRunVoteRefV0(request),
		AcceptedOptionRef: "option-" + safeAppRunnerRefPartV0(request.AppSpec.App.Slug),
		Summary:           "Plan compacto aceptado.",
		EvidenceRefs:      []string{"evidence-ref-app-runner-decision-v0"},
	}
}

func appRunFunctionContractPayloadV0(
	request PrepareAppOrchestrationRequestV0,
	plan orquestaappplanner.AppMicrotaskPlanV0,
) orquestacoreworkflow.PublishFunctionContractCommandPayloadV0 {
	return orquestacoreworkflow.PublishFunctionContractCommandPayloadV0{
		ContractRef:   appRunContractRefV0(plan),
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0),
		DecisionRef:   appRunDecisionRefV0(request),
		Summary:       "Contrato funcional de app.",
		FunctionNames: appRunFunctionNamesV0(plan),
		EvidenceRefs:  []string{"evidence-ref-app-runner-contract-v0"},
	}
}

func appRunCreateMicrotaskCommandsV0(
	request PrepareAppOrchestrationRequestV0,
	plan orquestaappplanner.AppMicrotaskPlanV0,
) ([]orquestacoreworkflow.OrchestrationCommandV0, error) {
	commands := make([]orquestacoreworkflow.OrchestrationCommandV0, 0, len(plan.Units))
	contractRef := appRunContractRefV0(plan)
	for _, unit := range plan.Units {
		command, err := orquestacoreworkflow.NewCreateMicrotaskCommandV0(
			appRunCommandMetaV0(request, "create-"+unit.TaskRef),
			orquestacoreworkflow.CreateMicrotaskCommandPayloadV0{
				Task: appRunWorkflowTaskV0(request.RunRef, contractRef, unit),
			},
		)
		if err != nil {
			return nil, err
		}
		commands = append(commands, command)
	}
	return commands, nil
}

func appRunWorkflowTaskV0(
	runRef string,
	contractRef string,
	unit orquestaappplanner.AppWorkUnitV0,
) orquestacoreworkflow.WorkflowTaskV0 {
	return orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:      orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:             unit.TaskRef,
		RunID:              runRef,
		PhaseID:            unit.PhaseID,
		Title:              unit.Title,
		Summary:            "Ejecutar microtarea planificada con alcance acotado.",
		WriteSet:           unit.WriteSet,
		AcceptanceCriteria: appRunTaskAcceptanceV0(),
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
			ContractRef:  contractRef,
			FunctionName: appRunFunctionNameV0(unit.TaskRef),
		}},
	}
}

func appRunTaskAcceptanceV0() []string {
	return []string{
		"Entrega compacta registrada.",
		"Evidencia externa con validacion.",
	}
}

func appRunFunctionNamesV0(plan orquestaappplanner.AppMicrotaskPlanV0) []string {
	names := make([]string, 0, len(plan.Units))
	for _, unit := range plan.Units {
		names = append(names, appRunFunctionNameV0(unit.TaskRef))
	}
	return compactAppRunnerRefsV0(names)
}

func appRunFunctionNameV0(taskRef string) string {
	return "app_unit_" + strings.ReplaceAll(safeAppRunnerRefPartV0(taskRef), "-", "_")
}
