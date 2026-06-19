package orquestaappplanner

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestAppPlanCandidateProviderV0EmiteSoloOlaLista(t *testing.T) {
	plan := mustAppPlanForTestV0(t)
	provider := AppPlanCandidateProviderV0{Plan: plan}

	first := mustBuildAppCandidatesForTestV0(t, provider, appRunForTestV0(nil, nil))
	if len(first.WorkCandidates) != 1 || first.WorkCandidates[0].AgentCandidate.Payload.TaskRef != "task-agenda-bootstrap" {
		t.Fatalf("first candidates=%+v", first.WorkCandidates)
	}
	if first.WorkCandidates[0].AgentCandidate.Payload.Role != "analisis" ||
		first.WorkCandidates[0].CapacityCandidate.Payload.ReasonCode != "work_profile_code_study" {
		t.Fatalf("first profile payload capacity=%+v agent=%+v",
			first.WorkCandidates[0].CapacityCandidate.Payload,
			first.WorkCandidates[0].AgentCandidate.Payload,
		)
	}

	second := mustBuildAppCandidatesForTestV0(t, provider, appRunForTestV0([]string{"ack-agenda-bootstrap"}, nil))
	if len(second.WorkCandidates) != 2 {
		t.Fatalf("second candidates=%+v", second.WorkCandidates)
	}
	if len(second.WorkClaims) != 2 {
		t.Fatalf("work claims de ola incompletos: %+v", second.WorkClaims)
	}
	for _, candidate := range second.WorkCandidates {
		if len(candidate.Claims) != 1 || candidate.Claims[0].ClaimRef != candidate.SubjectClaimRefs[0] {
			t.Fatalf("candidate claim no acotado: %+v", candidate.Claims)
		}
	}
}

func TestAppPlanCandidateProviderV0PropagaSkillRefsDeclaradas(t *testing.T) {
	plan := mustAppPlanForTestV0(t)
	for i := range plan.Units {
		if plan.Units[i].TaskRef == "task-agenda-web" {
			plan.Units[i].SkillRefs = []string{"skill-ref-catalogo-declarado-v0"}
		}
	}
	provider := AppPlanCandidateProviderV0{Plan: plan}
	candidates := mustBuildAppCandidatesForTestV0(t, provider, appRunForTestV0([]string{"ack-agenda-bootstrap"}, nil))

	for _, candidate := range candidates.WorkCandidates {
		if candidate.AgentCandidate.Payload.AgentRequestID != "agent-agenda-web" {
			continue
		}
		if !appPlannerStringInSetV0(candidate.AgentCandidate.Payload.SkillRefs, "skill-ref-catalogo-declarado-v0") {
			t.Fatalf("web skill_refs=%v", candidate.AgentCandidate.Payload.SkillRefs)
		}
		return
	}
	t.Fatalf("candidate web no encontrado: %+v", candidates.WorkCandidates)
}

func TestAppPlanCandidateProviderV0UsaPerfilNeutralParaDocs(t *testing.T) {
	plan := mustAppPlanForTestV0(t)
	provider := AppPlanCandidateProviderV0{Plan: plan}
	run := appRunForTestV0([]string{"ack-agenda-api"}, nil)
	run.CurrentPhase = orquestacoreworkflow.OrchestrationPhaseDocumentacionV0

	candidates := mustBuildAppCandidatesForTestV0(t, provider, run)
	if len(candidates.WorkCandidates) != 1 {
		t.Fatalf("candidates=%+v", candidates.WorkCandidates)
	}
	candidate := candidates.WorkCandidates[0]
	if candidate.AgentCandidate.Payload.Role != "documentacion" ||
		candidate.AgentCandidate.Payload.Summary != "Documentacion acotada lista." ||
		candidate.CapacityCandidate.Payload.ReasonCode != "work_profile_documentation" {
		t.Fatalf("profile payload capacity=%+v agent=%+v",
			candidate.CapacityCandidate.Payload,
			candidate.AgentCandidate.Payload,
		)
	}
}

func TestAppPlanCandidateProviderV0NoReemiteAgenteYaArrancado(t *testing.T) {
	plan := mustAppPlanForTestV0(t)
	provider := AppPlanCandidateProviderV0{Plan: plan}
	run := appRunForTestV0([]string{"ack-agenda-bootstrap"}, []string{"agent-agenda-agenda-core"})

	candidates := mustBuildAppCandidatesForTestV0(t, provider, run)
	if len(candidates.WorkCandidates) != 1 {
		t.Fatalf("candidates=%+v", candidates.WorkCandidates)
	}
	got := candidates.WorkCandidates[0].AgentCandidate.Payload.AgentRequestID
	if got != "agent-agenda-web" {
		t.Fatalf("agent=%s", got)
	}
}

func TestAppPlanCandidateProviderV0ProduceComandosSchedulerValidos(t *testing.T) {
	plan := mustAppPlanForTestV0(t)
	provider := AppPlanCandidateProviderV0{Plan: plan}
	run := appRunForTestV0([]string{"ack-agenda-bootstrap"}, nil)
	candidates := mustBuildAppCandidatesForTestV0(t, provider, run)

	tick, err := orquestadirectorscheduler.BuildDirectorSchedulerTickV0(orquestadirectorscheduler.DirectorSchedulerTickInputV0{
		TickRef:        "tick-ref-app-001",
		RunRef:         run.RunID,
		OccurredAt:     "2026-05-09T20:00:00Z",
		Snapshot:       schedulerSnapshotForAppRunV0(run, nil),
		WorkClaims:     candidates.WorkClaims,
		WorkCandidates: candidates.WorkCandidates,
		EvidenceRefs:   []string{"evidence-ref-app-scheduler-001"},
	})
	if err != nil {
		t.Fatalf("BuildDirectorSchedulerTickV0 capacity: %v", err)
	}
	if tick.Status != orquestadirectorscheduler.SchedulerTickStatusCommandsReadyV0 ||
		len(tick.Commands) != 2 {
		t.Fatalf("tick=%+v", tick)
	}

	decided := []string{"capacity-agenda-agenda-core", "capacity-agenda-web"}
	tick, err = orquestadirectorscheduler.BuildDirectorSchedulerTickV0(orquestadirectorscheduler.DirectorSchedulerTickInputV0{
		TickRef:        "tick-ref-app-002",
		RunRef:         run.RunID,
		OccurredAt:     "2026-05-09T20:01:00Z",
		Snapshot:       schedulerSnapshotForAppRunV0(run, decided),
		WorkClaims:     candidates.WorkClaims,
		WorkCandidates: candidates.WorkCandidates,
		EvidenceRefs:   []string{"evidence-ref-app-scheduler-002"},
	})
	if err != nil {
		t.Fatalf("BuildDirectorSchedulerTickV0 agent: %v", err)
	}
	if tick.Status != orquestadirectorscheduler.SchedulerTickStatusCommandsReadyV0 ||
		len(tick.Commands) != 4 {
		t.Fatalf("tick=%+v", tick)
	}
}

func mustBuildAppCandidatesForTestV0(
	t *testing.T,
	provider AppPlanCandidateProviderV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) orquestacionnucleoapp.SchedulerCandidateSetV0 {
	t.Helper()
	candidates, err := provider.BuildSchedulerCandidatesV0(context.Background(), orquestacionnucleoapp.SchedulerCandidateRequestV0{
		Run:           run,
		OccurredAt:    "2026-05-09T20:00:00Z",
		CorrelationID: "corr-app-planner-001",
		EvidenceRefs:  []string{"evidence-ref-app-planner-001"},
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	return candidates
}

func appRunForTestV0(deliveries []string, started []string) orquestacoreworkflow.OrchestrationRunV0 {
	return orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         "run-ref-app-001",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Deliveries:    deliveries,
		StartedAgents: started,
	}
}

func schedulerSnapshotForAppRunV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	capacityDecisions []string,
) orquestadirectorscheduler.RunSchedulingSnapshotV0 {
	return orquestadirectorscheduler.RunSchedulingSnapshotV0{
		RunRef:            run.RunID,
		CurrentPhaseID:    string(run.CurrentPhase),
		CapacityDecisions: capacityDecisions,
		StartedAgents:     run.StartedAgents,
		Deliveries:        run.Deliveries,
	}
}
