package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestApplyDrainObservationsV0NormalizaEntregaTardiaATareaAbiertaEnFaseActual(t *testing.T) {
	ctx := context.Background()
	runRef := "run-ref-drain-late-phase-001"
	taskRef := "task-ref-drain-late-phase-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	deliveryRef := "ack-ref-drain-late-phase-001"
	stack := StackV0{Stores: StoresV0{
		RunStore:  drainObservationPhaseRepairRunStoreV0(runRef, taskRef, agentRef, false),
		EventSink: orquestacionnucleoapp.NewInMemoryEventSinkV0(),
	}}

	run, err := stack.applyDrainObservationsV0(ctx, DrainRunRequestV0{
		RunRef:        runRef,
		CorrelationID: "corr-drain-late-phase-001",
		OccurredAt:    "2026-05-26T22:00:00Z",
	}, []orquestacionnucleoapp.AgentDeliveryObservationV0{
		drainObservationPhaseRepairObservationV0(deliveryRef, taskRef, agentRef),
	})
	if err != nil {
		t.Fatalf("applyDrainObservationsV0: %v", err)
	}
	if !stringInDrainSetV0(run.Deliveries, deliveryRef) {
		t.Fatalf("deliveries=%v, falta %s", run.Deliveries, deliveryRef)
	}
	if !stringInDrainSetV0(run.DeliveredTasks, taskRef) {
		t.Fatalf("delivered_tasks=%v, falta %s", run.DeliveredTasks, taskRef)
	}
}

func TestApplyDrainObservationsV0RegistraAckTardioDeAgentePerdido(t *testing.T) {
	ctx := context.Background()
	runRef := "run-ref-drain-lost-late-ack-001"
	taskRef := "task-ref-drain-lost-late-ack-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	deliveryRef := "ack-ref-drain-lost-late-ack-001"
	stack := StackV0{Stores: StoresV0{
		RunStore:  drainObservationPhaseRepairRunStoreV0(runRef, taskRef, agentRef, true),
		EventSink: orquestacionnucleoapp.NewInMemoryEventSinkV0(),
	}}

	run, err := stack.applyDrainObservationsV0(ctx, DrainRunRequestV0{
		RunRef:        runRef,
		CorrelationID: "corr-drain-lost-late-ack-001",
		OccurredAt:    "2026-05-27T17:20:00Z",
	}, []orquestacionnucleoapp.AgentDeliveryObservationV0{
		drainObservationPhaseRepairObservationV0(deliveryRef, taskRef, agentRef),
	})
	if err != nil {
		t.Fatalf("applyDrainObservationsV0 late lost ack: %v", err)
	}
	if !stringInDrainSetV0(run.Deliveries, deliveryRef) ||
		!stringInDrainSetV0(run.DeliveredAgents, agentRef) ||
		!stringInDrainSetV0(run.DeliveredTasks, taskRef) {
		t.Fatalf("ACK valido no quedo registrado: deliveries=%v delivered_agents=%v delivered_tasks=%v", run.Deliveries, run.DeliveredAgents, run.DeliveredTasks)
	}
	if stringInDrainSetV0(run.LostAgents, agentRef) {
		t.Fatalf("ACK valido tardio debe reconciliar agente perdido: lost=%v delivered_agents=%v", run.LostAgents, run.DeliveredAgents)
	}
}

func drainObservationPhaseRepairRunStoreV0(
	runRef string,
	taskRef string,
	agentRef string,
	lost bool,
) *orquestacionnucleoapp.InMemoryRunStoreV0 {
	run := orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         runRef,
		ProjectRef:    "project-ref-drain-late-phase-001",
		AppSpecRef:    "app-spec-ref-drain-late-phase-001",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseRevisionV0,
		Phases: []orquestacoreworkflow.OrchestrationPhaseV0{{
			ID:                  orquestacoreworkflow.OrchestrationPhaseRevisionV0,
			Status:              orquestacoreworkflow.OrchestrationPhaseStatusActiveV0,
			EntryCriteria:       []string{"entry"},
			ExitCriteria:        []string{"exit"},
			EvidenceRequired:    []string{"evidence"},
			RecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
		}},
		Tasks:         []string{taskRef},
		Agents:        []string{agentRef},
		StartedAgents: []string{agentRef},
	}
	if lost {
		run.LostAgents = []string{agentRef}
		ref := "assessment-ref-drain-lost-late-ack-001#phase:brainstorming_arquitectura#agent:" + agentRef + "#task:" + taskRef + "#verdict:needs_revision#action:ask_director#severity:high"
		run.AgentAssessments = []string{ref}
		run.DirectorQuestions = []string{"question-ref-drain-lost-late-ack-001"}
	}
	return orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
}

func drainObservationPhaseRepairObservationV0(
	deliveryRef string,
	taskRef string,
	agentRef string,
) orquestacionnucleoapp.AgentDeliveryObservationV0 {
	return orquestacionnucleoapp.AgentDeliveryObservationV0{
		ArtifactRef:  deliveryRef,
		DeliveryRef:  deliveryRef,
		PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskID:       taskRef,
		AgentRef:     agentRef,
		Summary:      "Entrega tardia de agente mientras el run ya esta en revision.",
		EvidenceRefs: []string{deliveryRef},
	}
}
