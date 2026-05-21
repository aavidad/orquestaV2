package orquestacoreworkflow

import (
	"errors"
	"reflect"
	"testing"
)

func TestRegisterPhaseArtifactCommandV0ProjectsDirectorArtifact(t *testing.T) {
	run := mustRunWithStartedPhaseAgentV0(t, OrchestrationPhaseBrainstormingArquitecturaV0, "agent-director-brainstorm")
	command := mustRegisterPhaseArtifactCommandV0(t, "cmd-phase-artifact", "idem-phase-artifact", "artifact-brainstorm-001", "agent-director-brainstorm")

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle RegisterPhaseArtifact: %v", err)
	}
	assertSingleEventTypeV0(t, result, OrchestrationEventPhaseArtifactRegisteredV0)
	if len(result.Outbox) != 0 {
		t.Fatalf("outbox=%d, want empty", len(result.Outbox))
	}

	applied := mustApplyReducerEventV0(t, run, result.Events[0])
	want := []string{phaseArtifactProjectionRefV0(phaseArtifactRegisteredPayloadFromCommandV0(validRegisterPhaseArtifactPayloadV0("artifact-brainstorm-001", "agent-director-brainstorm", OrchestrationPhaseBrainstormingArquitecturaV0)))}
	if !reflect.DeepEqual(applied.PhaseArtifacts, want) {
		t.Fatalf("phase_artifacts=%v, want %v", applied.PhaseArtifacts, want)
	}

	result, err = HandleCommandV0(applied, command)
	assertIdempotentNoEventsV0(t, result, err)
}

func TestReplayDurableEventsV0AcceptsPhaseArtifactRegistered(t *testing.T) {
	artifact := mustPhaseArtifactRegisteredEventWithKeyV0(t, "evt-durable-phase-artifact", 7, "idem-phase-artifact", "artifact-brainstorm-001")
	events := []OrchestrationEventV0{
		mustReplayRunStartedEventWithKeyV0(t, "evt-durable-start-phase-artifact", 1, "idem-start-phase-artifact"),
		mustReplayPhaseEventWithKeyV0(t, "evt-durable-open-phase-artifact", 2, "idem-open-phase-artifact", OrchestrationPhaseBrainstormingArquitecturaV0),
		mustCapacityRequestedPhaseEventWithKeyV0(t, "evt-durable-capacity-phase-artifact", 3, "idem-capacity-phase-artifact", "capacity-brainstorm-001", OrchestrationPhaseBrainstormingArquitecturaV0),
		mustCapacityDecidedEventWithKeyV0(t, "evt-durable-capacity-decision-phase-artifact", 4, "idem-capacity-decision-phase-artifact", "capacity-brainstorm-001"),
		mustAgentRequestedPhaseEventWithKeyV0(t, "evt-durable-agent-phase-artifact", 5, "idem-agent-phase-artifact", "agent-director-brainstorm", "capacity-brainstorm-001", OrchestrationPhaseBrainstormingArquitecturaV0),
		mustAgentStartedEventWithKeyV0(t, "evt-durable-agent-started-phase-artifact", 6, "idem-agent-started-phase-artifact", "agent-director-brainstorm"),
		artifact,
		artifact,
	}

	got, err := ReplayDurableEventsV0(events)
	if err != nil {
		t.Fatalf("replay durable PhaseArtifactRegistered: %v", err)
	}
	if got.LastSequence != 7 {
		t.Fatalf("last_sequence=%d, want 7", got.LastSequence)
	}
	if len(got.PhaseArtifacts) != 1 {
		t.Fatalf("phase_artifacts=%v, want one", got.PhaseArtifacts)
	}
}

func TestRegisterPhaseArtifactCommandV0AcceptsLateArtifactForAgentLaunchPhase(t *testing.T) {
	run := mustRunWithStartedPhaseAgentV0(t, OrchestrationPhaseBrainstormingArquitecturaV0, "agent-director-brainstorm")
	run = mustApplySingleCommandEventV0(
		t,
		run,
		mustOpenPhaseCommandV0(t, "cmd-open-programacion-after-brainstorm-agent", "idem-open-programacion-after-brainstorm-agent", OrchestrationPhaseProgramacionV0),
	)
	payload := validRegisterPhaseArtifactPayloadV0(
		"artifact-brainstorm-late",
		"agent-director-brainstorm",
		OrchestrationPhaseBrainstormingArquitecturaV0,
	)
	command, err := NewRegisterPhaseArtifactCommandV0(
		validCommandMetaV0("cmd-phase-artifact-late", "idem-phase-artifact-late"),
		payload,
	)
	command = mustCommandV0(t, command, err)

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle late RegisterPhaseArtifact: %v", err)
	}
	assertSingleEventTypeV0(t, result, OrchestrationEventPhaseArtifactRegisteredV0)
	applied := mustApplyReducerEventV0(t, run, result.Events[0])
	if !phaseArtifactInRunForTestV0(applied, "artifact-brainstorm-late") {
		t.Fatalf("phase_artifacts=%v missing late artifact", applied.PhaseArtifacts)
	}
}

func TestRegisterPhaseArtifactCommandV0RejectsLateArtifactForOtherAgentPhase(t *testing.T) {
	run := mustRunWithStartedPhaseAgentV0(t, OrchestrationPhaseBrainstormingArquitecturaV0, "agent-director-brainstorm")
	run = mustApplySingleCommandEventV0(
		t,
		run,
		mustOpenPhaseCommandV0(t, "cmd-open-programacion-wrong-artifact", "idem-open-programacion-wrong-artifact", OrchestrationPhaseProgramacionV0),
	)
	payload := validRegisterPhaseArtifactPayloadV0(
		"artifact-wrong-phase",
		"agent-director-brainstorm",
		OrchestrationPhaseVotacionYDecisionV0,
	)
	command, err := NewRegisterPhaseArtifactCommandV0(
		validCommandMetaV0("cmd-phase-artifact-wrong", "idem-phase-artifact-wrong"),
		payload,
	)
	command = mustCommandV0(t, command, err)

	_, err = HandleCommandV0(run, command)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload.phase_id")
}

func TestReplayDurableEventsV0AcceptsLatePhaseArtifactForAgentLaunchPhase(t *testing.T) {
	artifact := mustPhaseArtifactRegisteredEventWithKeyV0(t, "evt-durable-phase-artifact-late", 8, "idem-phase-artifact-late", "artifact-brainstorm-late")
	events := []OrchestrationEventV0{
		mustReplayRunStartedEventWithKeyV0(t, "evt-durable-start-phase-artifact-late", 1, "idem-start-phase-artifact-late"),
		mustReplayPhaseEventWithKeyV0(t, "evt-durable-open-phase-artifact-late", 2, "idem-open-phase-artifact-late", OrchestrationPhaseBrainstormingArquitecturaV0),
		mustCapacityRequestedPhaseEventWithKeyV0(t, "evt-durable-capacity-phase-artifact-late", 3, "idem-capacity-phase-artifact-late", "capacity-brainstorm-001", OrchestrationPhaseBrainstormingArquitecturaV0),
		mustCapacityDecidedEventWithKeyV0(t, "evt-durable-capacity-decision-phase-artifact-late", 4, "idem-capacity-decision-phase-artifact-late", "capacity-brainstorm-001"),
		mustAgentRequestedPhaseEventWithKeyV0(t, "evt-durable-agent-phase-artifact-late", 5, "idem-agent-phase-artifact-late", "agent-director-brainstorm", "capacity-brainstorm-001", OrchestrationPhaseBrainstormingArquitecturaV0),
		mustAgentStartedEventWithKeyV0(t, "evt-durable-agent-started-phase-artifact-late", 6, "idem-agent-started-phase-artifact-late", "agent-director-brainstorm"),
		mustReplayPhaseEventWithKeyV0(t, "evt-durable-open-programacion-before-late-artifact", 7, "idem-open-programacion-before-late-artifact", OrchestrationPhaseProgramacionV0),
		artifact,
	}

	got, err := ReplayDurableEventsV0(events)
	if err != nil {
		t.Fatalf("replay durable late PhaseArtifactRegistered: %v", err)
	}
	if got.CurrentPhase != OrchestrationPhaseProgramacionV0 {
		t.Fatalf("current_phase=%s, want %s", got.CurrentPhase, OrchestrationPhaseProgramacionV0)
	}
	if !phaseArtifactInRunForTestV0(got, "artifact-brainstorm-late") {
		t.Fatalf("phase_artifacts=%v missing late artifact", got.PhaseArtifacts)
	}
}

func TestRegisterPhaseArtifactCommandV0RequiresStartedAgent(t *testing.T) {
	run := mustRunWithRequestedPhaseAgentV0(t, OrchestrationPhaseBrainstormingArquitecturaV0, "agent-director-not-started")
	command := mustRegisterPhaseArtifactCommandV0(t, "cmd-phase-artifact-agent", "idem-phase-artifact-agent", "artifact-agent-001", "agent-director-not-started")

	_, err := HandleCommandV0(run, command)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload.agent_ref")
}

func TestRegisterPhaseArtifactCommandV0RejectsProgrammingPhase(t *testing.T) {
	payload := validRegisterPhaseArtifactPayloadV0("artifact-programacion", "agent-director-programacion", OrchestrationPhaseProgramacionV0)

	_, err := NewRegisterPhaseArtifactCommandV0(validCommandMetaV0("cmd-phase-artifact-programacion", "idem-phase-artifact-programacion"), payload)

	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != ErrTransicionInvalidaV0 || publicErr.Field != "payload.phase_id" {
		t.Fatalf("error=%+v, want invalid payload.phase_id", publicErr)
	}
}

func TestValidateOrchestrationRunV0RejectsInvalidPhaseArtifactProjection(t *testing.T) {
	run := mustRunWithStartedPhaseAgentV0(t, OrchestrationPhaseBrainstormingArquitecturaV0, "agent-director-valid")
	run.PhaseArtifacts = []string{"artifact-orphan#phase:brainstorming_arquitectura#agent:agent-missing"}

	issues := ValidateOrchestrationRunV0(run)
	if len(issues) == 0 || issues[0].Field != "phase_artifacts" {
		t.Fatalf("issues=%+v, want phase_artifacts invalid", issues)
	}
}

func mustRunWithStartedPhaseAgentV0(t *testing.T, phase OrchestrationPhaseIDV0, requestID string) OrchestrationRunV0 {
	t.Helper()
	run := mustRunWithRequestedPhaseAgentV0(t, phase, requestID)
	return mustApplySingleCommandEventV0(t, run, mustRegisterAgentStartedCommandV0(t, "cmd-start-"+requestID, "idem-start-"+requestID, requestID))
}

func mustRunWithRequestedPhaseAgentV0(t *testing.T, phase OrchestrationPhaseIDV0, requestID string) OrchestrationRunV0 {
	t.Helper()
	capacityID := "capacity-" + requestID
	run := mustHandlerRunInPhaseV0(t, phase)
	run = mustApplySingleCommandEventV0(t, run, mustRequestCapacityPhaseCommandV0(t, "cmd-"+capacityID, "idem-"+capacityID, capacityID, phase))
	run = mustApplySingleCommandEventV0(t, run, mustRegisterCapacityDecisionCommandV0(t, "cmd-decision-"+capacityID, "idem-decision-"+capacityID, capacityID))
	return mustApplySingleCommandEventV0(t, run, mustRequestAgentPhaseCommandV0(t, "cmd-"+requestID, "idem-"+requestID, requestID, capacityID, phase))
}

func mustRegisterPhaseArtifactCommandV0(t *testing.T, commandID string, idempotencyKey string, artifactRef string, agentRef string) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewRegisterPhaseArtifactCommandV0(validCommandMetaV0(commandID, idempotencyKey), validRegisterPhaseArtifactPayloadV0(artifactRef, agentRef, OrchestrationPhaseBrainstormingArquitecturaV0))
	return mustCommandV0(t, command, err)
}

func validRegisterPhaseArtifactPayloadV0(artifactRef string, agentRef string, phase OrchestrationPhaseIDV0) RegisterPhaseArtifactCommandPayloadV0 {
	return RegisterPhaseArtifactCommandPayloadV0{
		ArtifactRef:  artifactRef,
		PhaseID:      string(phase),
		AgentRef:     agentRef,
		Summary:      "Registrar artefacto compacto de fase generado por director.",
		EvidenceRefs: []string{"docs/arquitectura.md"},
	}
}

func mustRequestCapacityPhaseCommandV0(t *testing.T, commandID string, idempotencyKey string, requestID string, phase OrchestrationPhaseIDV0) OrchestrationCommandV0 {
	t.Helper()
	payload := validRequestCapacityPayloadV0(requestID)
	payload.PhaseID = string(phase)
	payload.TaskRef = ""
	command, err := NewRequestCapacityCommandV0(validCommandMetaV0(commandID, idempotencyKey), payload)
	return mustCommandV0(t, command, err)
}

func mustRequestAgentPhaseCommandV0(t *testing.T, commandID string, idempotencyKey string, requestID string, capacityID string, phase OrchestrationPhaseIDV0) OrchestrationCommandV0 {
	t.Helper()
	payload := validRequestAgentPayloadV0(requestID)
	payload.PhaseID = string(phase)
	payload.TaskRef = ""
	payload.CapacityRequestRef = capacityID
	payload.Role = "director"
	payload.Summary = "Coordinar una fase y devolver evidencias compactas."
	command, err := NewRequestAgentCommandV0(validCommandMetaV0(commandID, idempotencyKey), payload)
	return mustCommandV0(t, command, err)
}

func mustCapacityRequestedPhaseEventWithKeyV0(t *testing.T, eventID string, sequence int64, idempotencyKey string, requestID string, phase OrchestrationPhaseIDV0) OrchestrationEventV0 {
	t.Helper()
	payload := validRequestCapacityPayloadV0(requestID)
	payload.PhaseID = string(phase)
	payload.TaskRef = ""
	meta := reducerEventMetaV0(eventID, sequence)
	meta.IdempotencyKey = idempotencyKey
	event, err := NewCapacityRequestedEventV0(meta, capacityRequestedPayloadFromCommandV0(payload))
	return mustReducerEventV0(t, event, err)
}

func mustAgentRequestedPhaseEventWithKeyV0(t *testing.T, eventID string, sequence int64, idempotencyKey string, requestID string, capacityID string, phase OrchestrationPhaseIDV0) OrchestrationEventV0 {
	t.Helper()
	payload := validRequestAgentPayloadV0(requestID)
	payload.PhaseID = string(phase)
	payload.TaskRef = ""
	payload.CapacityRequestRef = capacityID
	payload.Role = "director"
	payload.Summary = "Coordinar una fase y devolver evidencias compactas."
	meta := reducerEventMetaV0(eventID, sequence)
	meta.IdempotencyKey = idempotencyKey
	event, err := NewAgentRequestedEventV0(meta, agentRequestedPayloadFromCommandV0(payload))
	return mustReducerEventV0(t, event, err)
}

func mustPhaseArtifactRegisteredEventWithKeyV0(t *testing.T, eventID string, sequence int64, idempotencyKey string, artifactRef string) OrchestrationEventV0 {
	t.Helper()
	meta := reducerEventMetaV0(eventID, sequence)
	meta.IdempotencyKey = idempotencyKey
	payload := validRegisterPhaseArtifactPayloadV0(artifactRef, "agent-director-brainstorm", OrchestrationPhaseBrainstormingArquitecturaV0)
	event, err := NewPhaseArtifactRegisteredEventV0(meta, phaseArtifactRegisteredPayloadFromCommandV0(payload))
	return mustReducerEventV0(t, event, err)
}

func phaseArtifactInRunForTestV0(run OrchestrationRunV0, artifactRef string) bool {
	_, ok := phaseArtifactProjectionForRefV0(run, artifactRef)
	return ok
}
