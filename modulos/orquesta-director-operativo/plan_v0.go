package orquestadirectoroperativo

import "strings"

func BuildOperationalDirectorPlanV0(
	request OperationalDirectorRequestV0,
) OperationalDirectorPlanResultV0 {
	request = normalizeOperationalDirectorRequestV0(request)
	if issues := validateOperationalDirectorRequestV0(request); len(issues) > 0 {
		return OperationalDirectorPlanResultV0{Accepted: false, Issues: issues}
	}
	if request.Mode == OperationalDirectorModeDomainWorkV0 &&
		request.ContextStatus == OperationalDirectorContextInsufficientV0 {
		plan := baseOperationalDirectorPlanV0(request, OperationalDirectorPlanNeedsContextV0)
		plan.Steps = operationalDirectorNeedsContextStepsV0()
		plan = addOperationalDirectorPlanStepDetailsV0(plan)
		return OperationalDirectorPlanResultV0{
			Accepted:      true,
			ReadyToLaunch: false,
			Blocked:       true,
			Plan:          plan,
		}
	}

	plan := baseOperationalDirectorPlanV0(request, OperationalDirectorPlanReadyV0)
	plan.Steps = operationalDirectorReadyStepsV0(request)
	plan = addOperationalDirectorPlanStepDetailsV0(plan)
	return OperationalDirectorPlanResultV0{
		Accepted:      true,
		ReadyToLaunch: true,
		Blocked:       false,
		Plan:          plan,
	}
}

func normalizeOperationalDirectorRequestV0(
	request OperationalDirectorRequestV0,
) OperationalDirectorRequestV0 {
	request.RequestRef = strings.TrimSpace(request.RequestRef)
	request.RunRef = strings.TrimSpace(request.RunRef)
	request.ProjectRef = strings.TrimSpace(request.ProjectRef)
	request.Objective = strings.TrimSpace(request.Objective)
	request.Mode = OperationalDirectorModeV0(strings.TrimSpace(string(request.Mode)))
	request.ContextStatus = OperationalDirectorContextStatusV0(strings.TrimSpace(string(request.ContextStatus)))
	request.DomainRefs = compactStringsV0(request.DomainRefs)
	request.MissingContext = compactStringsV0(request.MissingContext)
	request.WorktreeRef = strings.TrimSpace(request.WorktreeRef)
	request.BranchRef = strings.TrimSpace(request.BranchRef)
	request.WriteSet = compactStringsV0(request.WriteSet)
	request.RequiredTests = compactStringsV0(request.RequiredTests)
	request.MaxLoops = clampPositiveV0(request.MaxLoops, DefaultOperationalDirectorMaxLoopsV0, MaxOperationalDirectorMaxLoopsV0)
	request.MaxParallelAgents = clampPositiveV0(
		request.MaxParallelAgents,
		DefaultOperationalDirectorMaxParallelAgentsV0,
		MaxOperationalDirectorMaxParallelAgentsV0,
	)
	if request.AllowRecursiveDelegation {
		request.MaxDelegationDepth = clampPositiveV0(
			request.MaxDelegationDepth,
			DefaultOperationalDirectorMaxDelegationDepthV0,
			MaxOperationalDirectorMaxDelegationDepthV0,
		)
		request.MaxSubagentsPerAgent = clampPositiveV0(
			request.MaxSubagentsPerAgent,
			DefaultOperationalDirectorMaxSubagentsPerAgentV0,
			MaxOperationalDirectorMaxSubagentsPerAgentV0,
		)
	}
	if request.Mode == OperationalDirectorModeDomainWorkV0 && request.ContextStatus == "" {
		request.ContextStatus = OperationalDirectorContextSufficientV0
	}
	return request
}

func baseOperationalDirectorPlanV0(
	request OperationalDirectorRequestV0,
	status OperationalDirectorPlanStatusV0,
) OperationalDirectorPlanV0 {
	return OperationalDirectorPlanV0{
		PlanRef:              "operational-director-plan-" + request.RequestRef,
		RequestRef:           request.RequestRef,
		RunRef:               request.RunRef,
		ProjectRef:           request.ProjectRef,
		Mode:                 request.Mode,
		Status:               status,
		Objective:            request.Objective,
		LoopBudget:           request.MaxLoops,
		MaxParallelAgents:    request.MaxParallelAgents,
		RecursiveDelegation:  request.AllowRecursiveDelegation,
		MaxDelegationDepth:   request.MaxDelegationDepth,
		MaxSubagentsPerAgent: request.MaxSubagentsPerAgent,
		WriteSet:             append([]string(nil), request.WriteSet...),
		RequiredTests:        append([]string(nil), request.RequiredTests...),
		DomainRefs:           append([]string(nil), request.DomainRefs...),
		MissingContext:       append([]string(nil), request.MissingContext...),
	}
}

func operationalDirectorNeedsContextStepsV0() []OperationalDirectorStepV0 {
	return []OperationalDirectorStepV0{
		stepV0("step-gather-context", OperationalDirectorStepGatherContextV0, "Revisar paquete recibido", nil),
		stepV0("step-request-domain-context", OperationalDirectorStepRequestDomainContextV0, "Pedir contexto faltante al dominio", []string{"step-gather-context"}),
		stepV0("step-replan-or-close", OperationalDirectorStepReplanOrCloseV0, "Replanificar cuando llegue contexto suficiente", []string{"step-request-domain-context"}),
	}
}

func operationalDirectorReadyStepsV0(
	request OperationalDirectorRequestV0,
) []OperationalDirectorStepV0 {
	steps := []OperationalDirectorStepV0{
		stepV0("step-gather-context", OperationalDirectorStepGatherContextV0, "Leer contexto vigente y restricciones", nil),
		stepV0("step-split-work", OperationalDirectorStepSplitWorkV0, "Dividir trabajo en piezas con write-set claro", []string{"step-gather-context"}),
		stepV0("step-launch-subagents", OperationalDirectorStepLaunchSubagentsV0, "Lanzar subagentes por piezas independientes", []string{"step-split-work"}),
		stepV0("step-wait-subagents", OperationalDirectorStepWaitSubagentsV0, "Esperar ACK, progreso o bloqueo de subagentes", []string{"step-launch-subagents"}),
	}
	if request.AllowRecursiveDelegation {
		steps = append(steps, stepV0(
			"step-govern-delegation",
			OperationalDirectorStepGovernDelegationV0,
			"Gobernar delegacion recursiva autorizada por el director",
			[]string{"step-wait-subagents"},
		))
	}
	reviewDeps := []string{"step-wait-subagents"}
	if len(steps) > 0 && steps[len(steps)-1].StepID == "step-govern-delegation" {
		reviewDeps = []string{"step-govern-delegation"}
	}
	steps = append(steps, stepV0("step-review-deliveries", OperationalDirectorStepReviewDeliveriesV0, "Revisar entregas contra contrato y evidencias", reviewDeps))
	if request.Mode == OperationalDirectorModeProgrammingV0 {
		steps = append(steps, stepV0("step-run-required-tests", OperationalDirectorStepRunRequiredTestsV0, "Ejecutar tests obligatorios antes de cerrar", []string{"step-review-deliveries"}))
		steps = append(steps, stepV0("step-replan-or-close", OperationalDirectorStepReplanOrCloseV0, "Replanificar fallos o cerrar con pruebas verdes", []string{"step-run-required-tests"}))
		return steps
	}
	return append(steps, stepV0("step-replan-or-close", OperationalDirectorStepReplanOrCloseV0, "Replanificar fallos o devolver artefacto validado", []string{"step-review-deliveries"}))
}

func stepV0(
	id string,
	kind OperationalDirectorStepKindV0,
	title string,
	dependsOn []string,
) OperationalDirectorStepV0 {
	return OperationalDirectorStepV0{
		StepID:    id,
		Kind:      kind,
		Status:    OperationalDirectorStepPendingV0,
		Title:     title,
		DependsOn: append([]string(nil), dependsOn...),
	}
}

func addOperationalDirectorPlanStepDetailsV0(
	plan OperationalDirectorPlanV0,
) OperationalDirectorPlanV0 {
	for i := range plan.Steps {
		switch plan.Steps[i].Kind {
		case OperationalDirectorStepGatherContextV0,
			OperationalDirectorStepRequestDomainContextV0:
			plan.Steps[i].DomainRefs = append([]string(nil), plan.DomainRefs...)
			plan.Steps[i].EvidenceRefs = append([]string(nil), plan.MissingContext...)
		case OperationalDirectorStepSplitWorkV0,
			OperationalDirectorStepLaunchSubagentsV0,
			OperationalDirectorStepWaitSubagentsV0:
			plan.Steps[i].WriteSet = append([]string(nil), plan.WriteSet...)
			plan.Steps[i].DomainRefs = append([]string(nil), plan.DomainRefs...)
			plan.Steps[i].RequiredTests = append([]string(nil), plan.RequiredTests...)
			plan.Steps[i].AcceptanceCriteria = operationalDirectorAcceptanceCriteriaV0(plan)
		case OperationalDirectorStepGovernDelegationV0:
			plan.Steps[i].ParentStepID = "step-launch-subagents"
			plan.Steps[i].DelegationDepth = 1
			plan.Steps[i].MaxChildAgents = plan.MaxSubagentsPerAgent
			plan.Steps[i].ChildStepIDs = []string{"step-review-deliveries"}
			plan.Steps[i].DomainRefs = append([]string(nil), plan.DomainRefs...)
			plan.Steps[i].AcceptanceCriteria = []string{
				"Cada subagente hijo conserva parent_ref, objetivo, presupuesto y criterio de review.",
				"No lanzar hijos fuera de MaxDelegationDepth ni MaxSubagentsPerAgent.",
			}
		case OperationalDirectorStepReviewDeliveriesV0,
			OperationalDirectorStepRunRequiredTestsV0,
			OperationalDirectorStepReplanOrCloseV0:
			plan.Steps[i].RequiredTests = append([]string(nil), plan.RequiredTests...)
			plan.Steps[i].AcceptanceCriteria = operationalDirectorAcceptanceCriteriaV0(plan)
		}
	}
	return plan
}

func operationalDirectorAcceptanceCriteriaV0(
	plan OperationalDirectorPlanV0,
) []string {
	criteria := []string{"Mantener el objetivo actual: " + plan.Objective}
	if plan.Mode == OperationalDirectorModeProgrammingV0 {
		return append(criteria,
			"No cambiar ficheros fuera del write-set.",
			"Ejecutar y registrar tests obligatorios antes de cerrar.",
		)
	}
	if plan.Status == OperationalDirectorPlanNeedsContextV0 {
		return append(criteria, "No lanzar agentes hasta recibir el contexto faltante.")
	}
	return append(criteria,
		"Usar solo refs de dominio y campos entregados por la app externa.",
		"No inventar contexto de dominio ausente.",
	)
}
