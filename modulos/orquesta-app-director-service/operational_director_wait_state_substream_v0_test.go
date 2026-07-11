package orquestaappdirectorservice

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

// Verifica el avance incremental por sub-ola: con 2 agentes en wait y solo 1
// entregado, review se abre SOLO para el entregado y el wait sigue running para
// el pendiente. Es el comportamiento de streaming que evita esperar a la ola.
func TestOperationalDirectorPlanStateAfterWaitDeliveredSubsetV0AvanzaSubconjunto(t *testing.T) {
	runRef := "run-streaming-subset"
	planRef := "plan-streaming-subset"
	taskA := serviceOperationalClosureTaskRefsForTestV0{TaskRef: "task-stream-a"}
	taskB := serviceOperationalClosureTaskRefsForTestV0{TaskRef: "task-stream-b"}
	agentA := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskA.TaskRef)
	agentB := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskB.TaskRef)

	state := serviceOperationalDirectorWideWaveWaitStateForTestV0(
		runRef, planRef, "parent-stream", "wave-stream", "cohort-stream", taskA, taskB,
	)
	run := orquestacoreworkflow.OrchestrationRunV0{
		RunID:           runRef,
		DeliveredAgents: []string{agentA},
		DeliveredTasks:  []string{taskA.TaskRef},
	}

	next, changed, err := operationalDirectorPlanStateAfterWaitDeliveredSubsetV0(
		ContinueAppDirectorRequestV0{RunRef: runRef, OccurredAt: "2026-05-22T14:00:00Z"},
		state,
		run,
	)
	if err != nil {
		t.Fatalf("subset: %v", err)
	}
	if !changed {
		t.Fatalf("debe avanzar el subconjunto entregado")
	}
	if next.ActiveStepID != "step-review-deliveries" {
		t.Fatalf("active step=%q want step-review-deliveries", next.ActiveStepID)
	}

	reviewStep := serviceOperationalDirectorPlanStateStepForTestV0(t, next, "step-review-deliveries")
	if reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
		t.Fatalf("review status=%q want running", reviewStep.Status)
	}
	if !sameStringSetForSubsetTestV0(reviewStep.AgentRefs, []string{agentA}) {
		t.Fatalf("review agents=%v want solo %s", reviewStep.AgentRefs, agentA)
	}
	if !sameStringSetForSubsetTestV0(reviewStep.TaskRefs, []string{taskA.TaskRef}) {
		t.Fatalf("review tasks=%v want solo %s", reviewStep.TaskRefs, taskA.TaskRef)
	}

	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, next, "step-wait-subagents")
	if waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
		t.Fatalf("wait status=%q want running (pendiente)", waitStep.Status)
	}
	if !sameStringSetForSubsetTestV0(waitStep.PendingAgentRefs, []string{agentB}) {
		t.Fatalf("wait pending=%v want solo %s", waitStep.PendingAgentRefs, agentB)
	}
}

// Con todos entregados en una sola pasada, en el flujo real la barrera de ola
// completa (operationalDirectorPlanStateAfterWaitConsumedV0) corre ANTES y consume
// el wait, por lo que este camino no llega a verlo en running. Aislado, el helper
// puede avanzar la ola completa a review (mismo resultado, wait aceptado): es
// inofensivo. Verificamos ese resultado equivalente, no inaccion.
func TestOperationalDirectorPlanStateAfterWaitDeliveredSubsetV0OlaCompletaAvanzaIgual(t *testing.T) {
	runRef := "run-streaming-full"
	planRef := "plan-streaming-full"
	taskA := serviceOperationalClosureTaskRefsForTestV0{TaskRef: "task-full-a"}
	taskB := serviceOperationalClosureTaskRefsForTestV0{TaskRef: "task-full-b"}
	agentA := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskA.TaskRef)
	agentB := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskB.TaskRef)

	state := serviceOperationalDirectorWideWaveWaitStateForTestV0(
		runRef, planRef, "parent-full", "wave-full", "cohort-full", taskA, taskB,
	)
	run := orquestacoreworkflow.OrchestrationRunV0{
		RunID:           runRef,
		DeliveredAgents: []string{agentA, agentB},
		DeliveredTasks:  []string{taskA.TaskRef, taskB.TaskRef},
	}

	next, changed, err := operationalDirectorPlanStateAfterWaitDeliveredSubsetV0(
		ContinueAppDirectorRequestV0{RunRef: runRef, OccurredAt: "2026-05-22T14:00:00Z"},
		state,
		run,
	)
	if err != nil {
		t.Fatalf("subset full: %v", err)
	}
	if !changed {
		t.Fatalf("aislado, con todo entregado debe avanzar a review")
	}
	reviewStep := serviceOperationalDirectorPlanStateStepForTestV0(t, next, "step-review-deliveries")
	if !sameStringSetForSubsetTestV0(reviewStep.AgentRefs, []string{agentA, agentB}) {
		t.Fatalf("review agents=%v want ambos", reviewStep.AgentRefs)
	}
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, next, "step-wait-subagents")
	if waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepAcceptedV0 {
		t.Fatalf("wait status=%q want accepted (consumido por streaming)", waitStep.Status)
	}
}

// Cuando el estado persistido del wait solo conserva AgentRefs, el streaming
// reconstruye cada task entregada por su AgentRequestRef causal. Una entrega de
// otra ola presente en el run no puede ampliar el review actual.
func TestOperationalDirectorPlanStateAfterWaitDeliveredSubsetV0SinTaskRefsNoAmpliaRunCompleto(t *testing.T) {
	runRef := "run-streaming-agent-only"
	planRef := "plan-streaming-agent-only"
	taskA := serviceOperationalClosureTaskRefsForTestV0{TaskRef: "task-agent-only-a"}
	taskB := serviceOperationalClosureTaskRefsForTestV0{TaskRef: "task-agent-only-b"}
	otherTask := "task-agent-only-other-wave"
	agentA := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskA.TaskRef)
	otherAgent := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(otherTask)

	state := serviceOperationalDirectorWideWaveWaitStateForTestV0(
		runRef, planRef, "parent-agent-only", "wave-agent-only", "cohort-agent-only", taskA, taskB,
	)
	for index := range state.Steps {
		if state.Steps[index].StepID == "step-wait-subagents" {
			state.Steps[index].TaskRefs = nil
		}
	}
	run := orquestacoreworkflow.OrchestrationRunV0{
		RunID:           runRef,
		DeliveredAgents: []string{agentA, otherAgent},
		DeliveredTasks:  []string{taskA.TaskRef, otherTask},
	}

	next, changed, err := operationalDirectorPlanStateAfterWaitDeliveredSubsetV0(
		ContinueAppDirectorRequestV0{RunRef: runRef, OccurredAt: "2026-05-22T14:00:00Z"},
		state,
		run,
	)
	if err != nil {
		t.Fatalf("subset agent-only: %v", err)
	}
	if !changed {
		t.Fatalf("debe avanzar la entrega causal del agente en espera")
	}
	reviewStep := serviceOperationalDirectorPlanStateStepForTestV0(t, next, "step-review-deliveries")
	if !sameStringSetForSubsetTestV0(reviewStep.AgentRefs, []string{agentA}) {
		t.Fatalf("review agents=%v want solo %s", reviewStep.AgentRefs, agentA)
	}
	if !sameStringSetForSubsetTestV0(reviewStep.TaskRefs, []string{taskA.TaskRef}) {
		t.Fatalf("review tasks=%v want solo %s", reviewStep.TaskRefs, taskA.TaskRef)
	}
}

// Sin una relacion causal entre una task entregada y el agente del wait, no se
// consume el wait ni se abre review sobre tareas del run ajenas al scope.
func TestOperationalDirectorPlanStateAfterWaitDeliveredSubsetV0SinMappingCausalEspera(t *testing.T) {
	runRef := "run-streaming-agent-only-unmapped"
	planRef := "plan-streaming-agent-only-unmapped"
	taskA := serviceOperationalClosureTaskRefsForTestV0{TaskRef: "task-agent-only-unmapped-a"}
	taskB := serviceOperationalClosureTaskRefsForTestV0{TaskRef: "task-agent-only-unmapped-b"}

	state := serviceOperationalDirectorWideWaveWaitStateForTestV0(
		runRef, planRef, "parent-agent-only-unmapped", "wave-agent-only-unmapped", "cohort-agent-only-unmapped", taskA, taskB,
	)
	for index := range state.Steps {
		if state.Steps[index].StepID == "step-wait-subagents" {
			state.Steps[index].TaskRefs = nil
			state.Steps[index].AgentRefs = []string{"agent-ref-without-causal-task"}
			state.Steps[index].PendingAgentRefs = []string{"agent-ref-without-causal-task"}
		}
	}
	state.PendingAgentRefs = []string{"agent-ref-without-causal-task"}
	run := orquestacoreworkflow.OrchestrationRunV0{
		RunID:           runRef,
		DeliveredAgents: []string{"agent-ref-without-causal-task"},
		DeliveredTasks:  []string{taskA.TaskRef},
	}

	_, changed, err := operationalDirectorPlanStateAfterWaitDeliveredSubsetV0(
		ContinueAppDirectorRequestV0{RunRef: runRef, OccurredAt: "2026-05-22T14:00:00Z"},
		state,
		run,
	)
	if err != nil {
		t.Fatalf("subset agent-only unmapped: %v", err)
	}
	if changed {
		t.Fatalf("sin mapping causal no debe ampliar review ni consumir el wait")
	}
}

// La normalizacion de WorkflowTaskAgentRequestRefV0 puede colisionar para IDs
// distintos. Sin TaskRefs persistidas, esa ambiguedad no permite atribuir una
// entrega al wait ni avanzar su estado.
func TestOperationalDirectorPlanStateAfterWaitDeliveredSubsetV0SinTaskRefsColisionDeAgentRefEspera(t *testing.T) {
	runRef := "run-streaming-agent-only-collision"
	planRef := "plan-streaming-agent-only-collision"
	taskA := serviceOperationalClosureTaskRefsForTestV0{TaskRef: "a/b"}
	taskB := serviceOperationalClosureTaskRefsForTestV0{TaskRef: "a-b"}
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskA.TaskRef)
	if otherAgentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskB.TaskRef); otherAgentRef != agentRef {
		t.Fatalf("agent refs must collide: %q != %q", agentRef, otherAgentRef)
	}

	state := serviceOperationalDirectorWideWaveWaitStateForTestV0(
		runRef, planRef, "parent-agent-only-collision", "wave-agent-only-collision", "cohort-agent-only-collision", taskA, taskB,
	)
	for index := range state.Steps {
		if state.Steps[index].StepID == "step-wait-subagents" {
			state.Steps[index].TaskRefs = nil
		}
	}
	run := orquestacoreworkflow.OrchestrationRunV0{
		RunID:           runRef,
		DeliveredAgents: []string{agentRef},
		DeliveredTasks:  []string{taskA.TaskRef, taskB.TaskRef},
	}

	_, changed, err := operationalDirectorPlanStateAfterWaitDeliveredSubsetV0(
		ContinueAppDirectorRequestV0{RunRef: runRef, OccurredAt: "2026-05-22T14:00:00Z"},
		state,
		run,
	)
	if err != nil {
		t.Fatalf("subset agent-only collision: %v", err)
	}
	if changed {
		t.Fatalf("con mapping ambiguo no debe incorporar tareas ni consumir el wait")
	}
}

// Sin ninguna entrega, no hay nada que avanzar.
func TestOperationalDirectorPlanStateAfterWaitDeliveredSubsetV0NoActuaSinEntregas(t *testing.T) {
	runRef := "run-streaming-none"
	planRef := "plan-streaming-none"
	taskA := serviceOperationalClosureTaskRefsForTestV0{TaskRef: "task-none-a"}
	taskB := serviceOperationalClosureTaskRefsForTestV0{TaskRef: "task-none-b"}

	state := serviceOperationalDirectorWideWaveWaitStateForTestV0(
		runRef, planRef, "parent-none", "wave-none", "cohort-none", taskA, taskB,
	)
	run := orquestacoreworkflow.OrchestrationRunV0{RunID: runRef}

	_, changed, err := operationalDirectorPlanStateAfterWaitDeliveredSubsetV0(
		ContinueAppDirectorRequestV0{RunRef: runRef, OccurredAt: "2026-05-22T14:00:00Z"},
		state,
		run,
	)
	if err != nil {
		t.Fatalf("subset none: %v", err)
	}
	if changed {
		t.Fatalf("sin entregas no debe avanzar")
	}
}

func sameStringSetForSubsetTestV0(got []string, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for _, w := range want {
		if !startAppDirectorStringInSetV0(got, w) {
			return false
		}
	}
	return true
}
