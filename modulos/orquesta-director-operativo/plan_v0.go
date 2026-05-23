package orquestadirectoroperativo

import (
	"fmt"
	"strings"
)

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
		request.MaxRecursiveAgents = clampOptionalPositiveMaxV0(
			request.MaxRecursiveAgents,
			MaxOperationalDirectorMaxRecursiveAgentsV0,
		)
	} else {
		request.MaxRecursiveAgents = 0
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
		MaxRecursiveAgents:   request.MaxRecursiveAgents,
		WriteSet:             append([]string(nil), request.WriteSet...),
		RequiredTests:        append([]string(nil), request.RequiredTests...),
		DomainRefs:           append([]string(nil), request.DomainRefs...),
		MissingContext:       append([]string(nil), request.MissingContext...),
		RepairPolicy:         DefaultOperationalDirectorRepairPolicyV0(),
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
	}
	launchSteps := operationalDirectorLaunchStepsV0(request.MaxParallelAgents)
	steps = append(steps, launchSteps...)
	steps = append(steps, stepV0(
		"step-wait-subagents",
		OperationalDirectorStepWaitSubagentsV0,
		"Esperar ACK, progreso o bloqueo de subagentes",
		operationalDirectorStepIDsV0(launchSteps),
	))
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

func operationalDirectorLaunchStepsV0(count int) []OperationalDirectorStepV0 {
	if count <= 0 {
		count = 1
	}
	steps := make([]OperationalDirectorStepV0, 0, count)
	for index := 1; index <= count; index++ {
		stepID := "step-launch-subagents"
		title := "Lanzar subagentes por piezas independientes"
		if index > 1 {
			stepID = fmt.Sprintf("step-launch-subagents-%02d", index)
		}
		if count > 1 {
			title = fmt.Sprintf("Lanzar subagente %d de %d por pieza independiente", index, count)
		}
		steps = append(steps, stepV0(stepID, OperationalDirectorStepLaunchSubagentsV0, title, []string{"step-split-work"}))
	}
	return steps
}

func operationalDirectorStepIDsV0(steps []OperationalDirectorStepV0) []string {
	out := make([]string, 0, len(steps))
	for _, step := range steps {
		out = append(out, step.StepID)
	}
	return compactStringsV0(out)
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
	launchStepCount := operationalDirectorStepKindCountV0(plan.Steps, OperationalDirectorStepLaunchSubagentsV0)
	launchStepIndex := 0
	for i := range plan.Steps {
		plan.Steps[i].WorkProfileKind = operationalDirectorWorkProfileKindV0(plan, plan.Steps[i].Kind)
		switch plan.Steps[i].Kind {
		case OperationalDirectorStepGatherContextV0,
			OperationalDirectorStepRequestDomainContextV0:
			plan.Steps[i].DomainRefs = append([]string(nil), plan.DomainRefs...)
			plan.Steps[i].EvidenceRefs = append([]string(nil), plan.MissingContext...)
		case OperationalDirectorStepSplitWorkV0,
			OperationalDirectorStepWaitSubagentsV0:
			plan.Steps[i].WriteSet = append([]string(nil), plan.WriteSet...)
			plan.Steps[i].DomainRefs = append([]string(nil), plan.DomainRefs...)
			plan.Steps[i].RequiredTests = append([]string(nil), plan.RequiredTests...)
			plan.Steps[i].AcceptanceCriteria = operationalDirectorAcceptanceCriteriaV0(plan)
		case OperationalDirectorStepLaunchSubagentsV0:
			launchStepIndex++
			plan.Steps[i].WriteSet = operationalDirectorShardedWriteSetV0(plan.WriteSet, launchStepIndex, launchStepCount)
			plan.Steps[i].DomainRefs = append([]string(nil), plan.DomainRefs...)
			plan.Steps[i].RequiredTests = append([]string(nil), plan.RequiredTests...)
			plan.Steps[i].AcceptanceCriteria = operationalDirectorAcceptanceCriteriaV0(plan)
			if launchStepCount > 1 {
				plan.Steps[i].AcceptanceCriteria = append(
					plan.Steps[i].AcceptanceCriteria,
					fmt.Sprintf("Coordinar como subagente %d de %d de la ola; el write-set asignado es ownership inicial, no limite duro.", launchStepIndex, launchStepCount),
				)
			}
		case OperationalDirectorStepGovernDelegationV0:
			plan.Steps[i].ParentStepID = "step-launch-subagents"
			plan.Steps[i].DelegationDepth = 1
			plan.Steps[i].MaxChildAgents = plan.MaxSubagentsPerAgent
			plan.Steps[i].ChildStepIDs = []string{"step-review-deliveries"}
			plan.Steps[i].DomainRefs = append([]string(nil), plan.DomainRefs...)
			plan.Steps[i].AcceptanceCriteria = []string{
				"Cada subagente hijo conserva parent_ref, objetivo, presupuesto y criterio de review.",
				"No lanzar hijos fuera de MaxDelegationDepth, MaxSubagentsPerAgent ni MaxRecursiveAgents.",
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

func operationalDirectorStepKindCountV0(
	steps []OperationalDirectorStepV0,
	kind OperationalDirectorStepKindV0,
) int {
	count := 0
	for _, step := range steps {
		if step.Kind == kind {
			count++
		}
	}
	return count
}

func operationalDirectorShardedWriteSetV0(
	writeSet []string,
	index int,
	total int,
) []string {
	writeSet = compactStringsV0(writeSet)
	if total <= 1 || len(writeSet) <= 1 || index <= 0 {
		return append([]string(nil), writeSet...)
	}
	out := make([]string, 0, len(writeSet)/total+1)
	for pathIndex, path := range writeSet {
		if pathIndex%total == index-1 {
			out = append(out, path)
		}
	}
	if len(out) == 0 {
		return []string{writeSet[(index-1)%len(writeSet)]}
	}
	return out
}
