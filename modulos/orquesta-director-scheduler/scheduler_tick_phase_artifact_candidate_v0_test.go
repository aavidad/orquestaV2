package orquestadirectorscheduler

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestBuildDirectorSchedulerTickV0RegistersPhaseArtifactBeforeWork(t *testing.T) {
	input := validSchedulerTickInputWithPhaseArtifactV0()
	input.WorkCandidates = []SchedulableWorkCandidateV0{validSchedulableCandidateV0()}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 1)
	assertSchedulerCommandTypesV0(t, plan, orquestacoreworkflow.OrchestrationCommandRegisterPhaseArtifactV0)
}

func TestBuildDirectorSchedulerTickV0DoesNotRepeatPhaseArtifact(t *testing.T) {
	input := validSchedulerTickInputWithPhaseArtifactV0()
	input.Snapshot.PhaseArtifacts = []string{"artifact-ref-scheduler-001"}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusQuiescentV0, 0)
}

func TestBuildDirectorSchedulerTickV0PhaseArtifactWaitsForStartedAgent(t *testing.T) {
	input := validSchedulerTickInputWithPhaseArtifactV0()
	input.Snapshot.StartedAgents = nil

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusWaitingV0, 0)
	assertSchedulerWaitingV0(t, plan, SchedulerWaitingAgentLifecyclePendingV0)
}

func TestBuildDirectorSchedulerTickV0RejectsForeignRunPhaseArtifactCandidate(t *testing.T) {
	input := validSchedulerTickInputWithPhaseArtifactV0()
	input.PhaseArtifactCandidates[0].CommandMeta.RunID = "run-externo-001"

	_, err := BuildDirectorSchedulerTickV0(input)

	assertSchedulerTickErrorV0(t, err, "phase_artifact_candidates.command_meta.run_id")
}

func validSchedulerTickInputWithPhaseArtifactV0() DirectorSchedulerTickInputV0 {
	input := validSchedulerTickInputV0()
	input.WorkCandidates = nil
	input.Snapshot.CurrentPhaseID = string(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0)
	input.Snapshot.Agents = []string{"agent-ref-scheduler-001"}
	input.Snapshot.StartedAgents = []string{"agent-ref-scheduler-001"}
	input.PhaseArtifactCandidates = []SchedulablePhaseArtifactCandidateV0{
		validSchedulablePhaseArtifactCandidateV0(),
	}
	return input
}

func validSchedulablePhaseArtifactCandidateV0() SchedulablePhaseArtifactCandidateV0 {
	return SchedulablePhaseArtifactCandidateV0{
		CandidateRef: "phase-artifact-candidate-ref-scheduler-001",
		CommandMeta:  schedulerMetaV0("cmd-phase-artifact-scheduler-001", "phase-artifact"),
		Payload: orquestacoreworkflow.RegisterPhaseArtifactCommandPayloadV0{
			ArtifactRef:  "artifact-ref-scheduler-001",
			PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0),
			AgentRef:     "agent-ref-scheduler-001",
			Summary:      "Artefacto compacto de fase aceptado por receipt.",
			EvidenceRefs: []string{"evidence-ref-phase-artifact-scheduler-001"},
		},
		EvidenceRefs: []string{"evidence-ref-phase-artifact-candidate-scheduler-001"},
	}
}
