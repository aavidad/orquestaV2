package orquestadirectortickinput

import (
	"encoding/json"
	"fmt"
	"testing"

	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
	orquestaorchestrationbudget "orquesta/modulos/orquesta-orchestration-budget"
)

func TestBuildDirectorSchedulerTickInputV0CompactaFronteraDeTrabajoSobredimensionada(t *testing.T) {
	run := tickInputProgramacionRunV0(t)
	candidates := tickInputLargeWorkCandidatesV0(run.RunID, 256)
	request := DirectorTickInputBuildRequestV0{
		TickRef:        "tick-ref-large-frontier-001",
		OccurredAt:     "2026-05-06T12:30:00Z",
		Run:            run,
		WorkCandidates: candidates,
		WorkClaims:     tickInputClaimsFromCandidatesV0(candidates),
		EvidenceRefs:   []string{"evidence-ref-large-frontier-001"},
	}
	raw := orquestadirectorscheduler.DirectorSchedulerTickInputV0{
		TickRef:        request.TickRef,
		RunRef:         run.RunID,
		OccurredAt:     request.OccurredAt,
		Snapshot:       buildRunSchedulingSnapshotV0(request),
		WorkCandidates: candidates,
		WorkClaims:     request.WorkClaims,
		EvidenceRefs:   request.EvidenceRefs,
	}
	if err := orquestadirectorscheduler.ValidateDirectorSchedulerTickInputV0(raw); err == nil {
		t.Fatal("fixture debe superar payload realmente enorme antes de compactar")
	}

	input, err := BuildDirectorSchedulerTickInputV0(request)
	if err != nil {
		t.Fatalf("BuildDirectorSchedulerTickInputV0: %v", err)
	}
	if len(input.WorkCandidates) == 0 || len(input.WorkCandidates) >= len(candidates) {
		t.Fatalf("work candidates no compactados: got=%d want 1..%d", len(input.WorkCandidates), len(candidates)-1)
	}
	if err := orquestadirectorscheduler.ValidateDirectorSchedulerTickInputV0(input); err != nil {
		t.Fatalf("input compactado invalido: %v", err)
	}
	plan, err := orquestadirectorscheduler.BuildDirectorSchedulerTickV0(input)
	if err != nil {
		t.Fatalf("scheduler no acepta input compactado: %v", err)
	}
	if plan.Status != orquestadirectorscheduler.SchedulerTickStatusCommandsReadyV0 ||
		len(plan.Commands) == 0 {
		t.Fatalf("plan inesperado: %+v", plan)
	}
}

func TestBuildDirectorSchedulerTickInputV0CompactaSnapshotSobredimensionadoSinCarrilActivo(t *testing.T) {
	run := tickInputProgramacionRunV0(t)
	run.Agents = append(run.Agents, "agent-ref-progress-target")
	run.StartedAgents = append(run.StartedAgents, "agent-ref-progress-target")
	for index := 0; index < 1700; index++ {
		suffix := tickInputLargeSuffixV0(index)
		agentRef := "agent-ref-t137-history-" + suffix
		run.Agents = append(run.Agents, agentRef)
		run.StartedAgents = append(run.StartedAgents, agentRef)
		run.StoppedAgents = append(run.StoppedAgents, agentRef)
		run.AgentAssessments = append(run.AgentAssessments, "assessment-ref-t137-history-"+suffix)
		run.DirectorQuestions = append(run.DirectorQuestions, "question-ref-t137-history-"+suffix)
	}
	request := DirectorTickInputBuildRequestV0{
		TickRef:      "tick-ref-t137-snapshot-pressure-001",
		OccurredAt:   "2026-07-07T17:29:39Z",
		Run:          run,
		EvidenceRefs: []string{"evidence-ref-t137-snapshot-pressure-001"},
	}
	rawSnapshot := buildRunSchedulingSnapshotV0(request)
	raw := orquestadirectorscheduler.DirectorSchedulerTickInputV0{
		TickRef:      request.TickRef,
		RunRef:       run.RunID,
		OccurredAt:   request.OccurredAt,
		Snapshot:     rawSnapshot,
		EvidenceRefs: request.EvidenceRefs,
	}
	if err := orquestadirectorscheduler.ValidateDirectorSchedulerTickInputV0(raw); err == nil {
		t.Fatal("fixture debe superar payload antes de compactar snapshot historico")
	}
	normalizedRaw := orquestadirectorscheduler.NormalizeDirectorSchedulerTickInputV0(raw)
	lastAssessmentRef := normalizedRaw.Snapshot.AgentAssessments[len(normalizedRaw.Snapshot.AgentAssessments)-1]
	lastQuestionRef := normalizedRaw.Snapshot.DirectorQuestions[len(normalizedRaw.Snapshot.DirectorQuestions)-1]

	input, err := BuildDirectorSchedulerTickInputV0(request)
	if err != nil {
		t.Fatalf("BuildDirectorSchedulerTickInputV0: %v", err)
	}
	if err := orquestadirectorscheduler.ValidateDirectorSchedulerTickInputV0(input); err != nil {
		t.Fatalf("input compactado invalido: %v", err)
	}
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}
	if len(data) > orquestaorchestrationbudget.SchedulerTickPayloadMaxBytesV0 {
		t.Fatalf("scheduler input no compacto: bytes=%d", len(data))
	}
	if len(input.Snapshot.AgentAssessments) > tickInputPayloadPressureKeepRefsV0 ||
		len(input.Snapshot.DirectorQuestions) > tickInputPayloadPressureKeepRefsV0 {
		t.Fatalf(
			"historial no compactado: assessments=%d questions=%d",
			len(input.Snapshot.AgentAssessments),
			len(input.Snapshot.DirectorQuestions),
		)
	}
	if !tickInputRefsContainV0(input.Snapshot.StartedAgents, "agent-ref-progress-target") {
		t.Fatalf("snapshot compactado perdio agente vivo: started_agents=%v", input.Snapshot.StartedAgents)
	}
	if !tickInputRefsContainV0(input.Snapshot.AgentAssessments, lastAssessmentRef) ||
		!tickInputRefsContainV0(input.Snapshot.DirectorQuestions, lastQuestionRef) {
		t.Fatalf(
			"snapshot compactado no conserva refs recientes: assessments_tail=%v questions_tail=%v",
			input.Snapshot.AgentAssessments,
			input.Snapshot.DirectorQuestions,
		)
	}
	plan, err := orquestadirectorscheduler.BuildDirectorSchedulerTickV0(input)
	if err != nil {
		t.Fatalf("scheduler no acepta input compactado: %v", err)
	}
	if plan.Status != orquestadirectorscheduler.SchedulerTickStatusWaitingV0 ||
		len(plan.WaitingReasons) != 1 ||
		plan.WaitingReasons[0] != orquestadirectorscheduler.SchedulerWaitingAgentDeliveryPendingV0 {
		t.Fatalf("plan inesperado: %+v", plan)
	}
}

func tickInputLargeWorkCandidatesV0(runRef string, count int) []orquestadirectorscheduler.SchedulableWorkCandidateV0 {
	candidates := make([]orquestadirectorscheduler.SchedulableWorkCandidateV0, 0, count)
	for index := 0; index < count; index++ {
		candidate := tickInputWorkCandidateV0(runRef)
		suffix := tickInputLargeSuffixV0(index)
		candidate.CandidateRef = "candidate-ref-tick-input-large-" + suffix
		candidate.SubjectClaimRefs = []string{"claim-ref-tick-input-large-" + suffix}
		candidate.Claims[0].ClaimRef = candidate.SubjectClaimRefs[0]
		candidate.Claims[0].TaskRef = "task-ref-tick-input-large-" + suffix
		candidate.Claims[0].AgentRequestID = "agent-ref-tick-input-large-" + suffix
		candidate.Claims[0].WriteSet[0].Ref = "modulos/app/" + suffix + "/internal/application/usecase_" + suffix + ".go"
		candidate.CapacityCandidate.CommandMeta.CommandID = "cmd-tick-input-capacity-" + suffix
		candidate.CapacityCandidate.CommandMeta.IdempotencyKey = "idem-tick-input-capacity-" + suffix
		candidate.CapacityCandidate.Payload.CapacityRequestID = "capacity-ref-tick-input-large-" + suffix
		candidate.CapacityCandidate.Payload.TaskRef = candidate.Claims[0].TaskRef
		candidate.AgentCandidate.ClaimRef = candidate.SubjectClaimRefs[0]
		candidate.AgentCandidate.CommandMeta.CommandID = "cmd-tick-input-agent-" + suffix
		candidate.AgentCandidate.CommandMeta.IdempotencyKey = "idem-tick-input-agent-" + suffix
		candidate.AgentCandidate.Payload.AgentRequestID = candidate.Claims[0].AgentRequestID
		candidate.AgentCandidate.Payload.TaskRef = candidate.Claims[0].TaskRef
		candidate.AgentCandidate.Payload.CapacityRequestRef = candidate.CapacityCandidate.Payload.CapacityRequestID
		candidate.GateCommandMeta.CommandID = "cmd-tick-input-gate-" + suffix
		candidate.GateCommandMeta.IdempotencyKey = "idem-tick-input-gate-" + suffix
		candidates = append(candidates, candidate)
	}
	return candidates
}

func tickInputClaimsFromCandidatesV0(
	candidates []orquestadirectorscheduler.SchedulableWorkCandidateV0,
) []orquestacoreconcurrency.WorksetClaimV0 {
	claims := make([]orquestacoreconcurrency.WorksetClaimV0, 0, len(candidates))
	for _, candidate := range candidates {
		claims = append(claims, candidate.Claims...)
	}
	return claims
}

func tickInputLargeSuffixV0(index int) string {
	return fmt.Sprintf("%03d-%s", index, "objetivo-actual-agenda-equipo-hexagonal-i18n-programacion-paralela")
}
