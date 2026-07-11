package orquestaappdirectorservice

import (
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

// operationalDirectorPlanStateAfterWaitDeliveredSubsetV0 implementa el avance
// incremental por sub-ola (streaming). En vez de esperar a que TODOS los agentes
// de la ola entreguen (barrera AND de
// operationalDirectorPlanStateAfterWaitConsumedV0), avanza el SUBCONJUNTO ya
// entregado a review en cuanto llega, manteniendo en wait los pendientes. Así
// las tareas independientes fluyen por el pipeline sin esperar a las hermanas.
//
// Seguridad causal: dentro de una misma ola las tareas son independientes por
// construccion del materializador (orden topologico en
// modulos/orquesta-director-operativo/wave_work_v0.go: una tarea con DependsOn no
// resuelto se pospone a una ola posterior con wave.DependsOn). Por tanto cualquier
// agente entregado de la ola es seguro de avanzar; la dependencia ya se respeto al
// formar las olas y al lanzar (workflowTaskSchedulableV0 / DependsOn).
//
// Solo actua si:
//   - el paso activo es wait_subagents en running,
//   - hay al menos un agente entregado del scope del wait,
//   - y quedan agentes pendientes (si no quedan, la barrera AND ya cubre el caso).
//
// Es idempotente y reentrante: cada nueva entrega reduce el wait y amplia el scope
// de review. El review step acumula los agentes/tasks ya avanzados.
func operationalDirectorPlanStateAfterWaitDeliveredSubsetV0(
	request ContinueAppDirectorRequestV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) (orquestacionnucleoapp.OperationalDirectorPlanStateV0, bool, error) {
	// Localiza el paso wait_subagents en running (no exige que sea el active step:
	// tras avanzar un subconjunto, el wait sigue running aunque el active pase a
	// review; una entrega posterior debe poder sumarse igual).
	waitStep, waitOK := operationalDirectorPlanStateStepByKindRunningV0(
		state, orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0,
	)
	if !waitOK {
		return state, false, nil
	}

	waitAgentRefs := compactServiceRefsV0(append(waitStep.PendingAgentRefs, state.PendingAgentRefs...))
	if len(waitAgentRefs) == 0 {
		waitAgentRefs = compactServiceRefsV0(waitStep.AgentRefs)
	}
	delivered, pending := operationalDirectorPartitionDeliveredAgentsV0(waitAgentRefs, run.DeliveredAgents)
	// Si no hay nuevas entregas pendientes de avanzar, este camino no aplica:
	// el avance completo lo gestiona operationalDirectorPlanStateAfterWaitConsumedV0.
	if len(delivered) == 0 {
		return state, false, nil
	}

	deliveredTasks := operationalDirectorTasksForDeliveredAgentsV0(waitStep, delivered, run)
	if len(deliveredTasks) == 0 {
		return state, false, nil
	}

	// Idempotencia: si todo lo entregado ya esta en el scope de review y el wait ya
	// refleja los pendientes, no hay nada nuevo que avanzar.
	if reviewStep, ok := operationalDirectorPlanStateStepByKindV0(state, orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0); ok {
		if allServiceRefsInSetForSubsetV0(delivered, reviewStep.AgentRefs) &&
			sameServiceRefSetV0(waitStep.PendingAgentRefs, pending) {
			return state, false, nil
		}
	}

	reviewStepID := ""
	nextSteps := make([]orquestacionnucleoapp.OperationalDirectorPlanStepStateV0, 0, len(state.Steps))
	for _, step := range state.Steps {
		nextStep := step
		switch {
		case step.StepID == waitStep.StepID:
			// El wait sigue running mientras queden pendientes; si ya no quedan,
			// se acepta (consumido por streaming).
			nextStep.PendingAgentRefs = append([]string(nil), pending...)
			if len(pending) == 0 {
				nextStep.Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
				nextStep.Reason = "wait-subagents-streaming-consumed"
			} else {
				nextStep.Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
				nextStep.Reason = "wait-subagents-streaming-subset"
			}
			nextStep.EvidenceRefs = compactServiceRefsV0(append(
				nextStep.EvidenceRefs,
				"evidence-ref-app-director-wait-subagents-streaming-subset-v0",
			))
		case step.Kind == orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0 &&
			(reviewStepID == "" || step.StepID == state.ActiveStepID):
			reviewStepID = step.StepID
			nextStep.Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
			// Acumula el subconjunto entregado en el scope de review (idempotente).
			nextStep.TaskRefs = compactServiceRefsV0(append(append([]string(nil), step.TaskRefs...), deliveredTasks...))
			nextStep.AgentRefs = compactServiceRefsV0(append(append([]string(nil), step.AgentRefs...), delivered...))
			nextStep.WaveRef = waitStep.WaveRef
			nextStep.CohortRef = waitStep.CohortRef
			nextStep.ParentTaskRef = waitStep.ParentTaskRef
			nextStep.Reason = "wait-subagents-streaming-subset"
			nextStep.EvidenceRefs = compactServiceRefsV0(append(
				nextStep.EvidenceRefs,
				"evidence-ref-app-director-review-deliveries-streaming-subset-v0",
			))
		}
		nextSteps = append(nextSteps, nextStep)
	}
	if reviewStepID == "" {
		return state, false, nil
	}
	state.Status = orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0
	state.ActiveStepID = reviewStepID
	// El scope pendiente del wait se conserva en el step; el state global no debe
	// dar por consumidos los pendientes.
	state.PendingAgentRefs = append([]string(nil), pending...)
	state.Steps = nextSteps
	state.EvidenceRefs = compactServiceRefsV0(append(
		state.EvidenceRefs,
		"evidence-ref-app-director-operational-plan-state-streaming-subset-v0",
	))
	state.UpdatedAt = request.OccurredAt
	next, err := orquestacionnucleoapp.NewOperationalDirectorPlanStateV0(state)
	if err != nil {
		return state, false, err
	}
	return next, true, nil
}

func operationalDirectorPlanStateStepByKindV0(
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	kind orquestadirectoroperativo.OperationalDirectorStepKindV0,
) (orquestacionnucleoapp.OperationalDirectorPlanStepStateV0, bool) {
	for _, step := range state.Steps {
		if step.Kind == kind {
			return step, true
		}
	}
	return orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{}, false
}

func operationalDirectorPlanStateStepByKindRunningV0(
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	kind orquestadirectoroperativo.OperationalDirectorStepKindV0,
) (orquestacionnucleoapp.OperationalDirectorPlanStepStateV0, bool) {
	step, ok := operationalDirectorPlanStateStepByKindV0(state, kind)
	if !ok || step.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
		return orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{}, false
	}
	return step, true
}

func allServiceRefsInSetForSubsetV0(values []string, available []string) bool {
	for _, value := range compactServiceRefsV0(values) {
		if !startAppDirectorStringInSetV0(available, value) {
			return false
		}
	}
	return true
}

func sameServiceRefSetV0(a []string, b []string) bool {
	ca := compactServiceRefsV0(a)
	cb := compactServiceRefsV0(b)
	if len(ca) != len(cb) {
		return false
	}
	for _, value := range ca {
		if !startAppDirectorStringInSetV0(cb, value) {
			return false
		}
	}
	return true
}

// operationalDirectorPartitionDeliveredAgentsV0 separa los agentes del scope del
// wait en entregados y pendientes segun run.DeliveredAgents.
func operationalDirectorPartitionDeliveredAgentsV0(
	waitAgentRefs []string,
	deliveredAgents []string,
) (delivered []string, pending []string) {
	for _, ref := range compactServiceRefsV0(waitAgentRefs) {
		if startAppDirectorStringInSetV0(deliveredAgents, ref) {
			delivered = append(delivered, ref)
		} else {
			pending = append(pending, ref)
		}
	}
	return delivered, pending
}

// operationalDirectorTasksForDeliveredAgentsV0 devuelve las task refs del scope del
// wait que corresponden a agentes ya entregados. Cuando el paso no conserva
// TaskRefs, reconstruye ese scope unicamente mediante la relacion causal
// TaskRef -> AgentRequestRef de las microtareas materializadas. No puede usar
// todas las DeliveredTasks del run: pueden pertenecer a otra ola o a otro wait.
// La reconstruccion exige una relacion inequivoca: si varias tareas entregadas
// normalizan al mismo AgentRequestRef, no se atribuye ninguna al wait.
func operationalDirectorTasksForDeliveredAgentsV0(
	activeStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	deliveredAgents []string,
	run orquestacoreworkflow.OrchestrationRunV0,
) []string {
	stepTasks := compactServiceRefsV0(activeStep.TaskRefs)
	if len(stepTasks) == 0 {
		tasksByAgentRef := make(map[string][]string)
		for _, taskRef := range compactServiceRefsV0(run.DeliveredTasks) {
			agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
			if startAppDirectorStringInSetV0(deliveredAgents, agentRef) {
				tasksByAgentRef[agentRef] = append(tasksByAgentRef[agentRef], taskRef)
			}
		}
		out := make([]string, 0, len(tasksByAgentRef))
		for _, agentRef := range compactServiceRefsV0(deliveredAgents) {
			tasks := tasksByAgentRef[agentRef]
			if len(tasks) == 1 {
				out = append(out, tasks[0])
			}
		}
		return compactServiceRefsV0(out)
	}
	out := make([]string, 0, len(stepTasks))
	for _, taskRef := range stepTasks {
		if startAppDirectorStringInSetV0(run.DeliveredTasks, taskRef) {
			out = append(out, taskRef)
		}
	}
	return compactServiceRefsV0(out)
}
