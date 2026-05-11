package orquestadirectorscheduler

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestBuildDirectorSchedulerTickV0RegistersDeliveryBeforeWork(t *testing.T) {
	input := validSchedulerTickInputWithDeliveryV0()
	input.WorkCandidates = []SchedulableWorkCandidateV0{validSchedulableCandidateV0()}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 1)
	assertSchedulerCommandTypesV0(t, plan, orquestacoreworkflow.OrchestrationCommandRegisterDeliveryV0)
}

func TestBuildDirectorSchedulerTickV0DoesNotRepeatDelivery(t *testing.T) {
	input := validSchedulerTickInputWithDeliveryV0()
	input.Snapshot.Deliveries = []string{"delivery-ref-scheduler-001"}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusQuiescentV0, 0)
}

func TestBuildDirectorSchedulerTickV0DeliveryWaitsForStartedAgent(t *testing.T) {
	input := validSchedulerTickInputWithDeliveryV0()
	input.Snapshot.StartedAgents = nil

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusWaitingV0, 0)
	assertSchedulerWaitingV0(t, plan, SchedulerWaitingAgentLifecyclePendingV0)
}

func TestBuildDirectorSchedulerTickV0RejectsForeignRunDeliveryCandidate(t *testing.T) {
	input := validSchedulerTickInputWithDeliveryV0()
	input.DeliveryCandidates[0].CommandMeta.RunID = "run-externo-001"

	_, err := BuildDirectorSchedulerTickV0(input)

	assertSchedulerTickErrorV0(t, err, "delivery_candidates.command_meta.run_id")
}

func validSchedulerTickInputWithDeliveryV0() DirectorSchedulerTickInputV0 {
	input := validSchedulerTickInputV0()
	input.WorkCandidates = nil
	input.Snapshot.Tasks = []string{"task-ref-scheduler-001"}
	input.Snapshot.Agents = []string{"agent-ref-scheduler-001"}
	input.Snapshot.StartedAgents = []string{"agent-ref-scheduler-001"}
	input.DeliveryCandidates = []SchedulableDeliveryCandidateV0{
		validSchedulableDeliveryCandidateV0(),
	}
	return input
}

func validSchedulableDeliveryCandidateV0() SchedulableDeliveryCandidateV0 {
	return SchedulableDeliveryCandidateV0{
		CandidateRef: "delivery-candidate-ref-scheduler-001",
		CommandMeta:  schedulerMetaV0("cmd-delivery-scheduler-001", "delivery"),
		Payload: orquestacoreworkflow.RegisterDeliveryCommandPayloadV0{
			DeliveryRef:  "delivery-ref-scheduler-001",
			PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskID:       "task-ref-scheduler-001",
			AgentRef:     "agent-ref-scheduler-001",
			Summary:      "Entrega compacta aceptada por receipt.",
			EvidenceRefs: []string{"evidence-ref-delivery-scheduler-001"},
		},
		EvidenceRefs: []string{"evidence-ref-delivery-candidate-scheduler-001"},
	}
}
