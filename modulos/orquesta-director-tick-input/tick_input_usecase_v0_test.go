package orquestadirectortickinput

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestBuildDirectorSchedulerTickInputV0ConstruyeSnapshotCanonico(t *testing.T) {
	run := tickInputProgramacionRunV0(t)
	run.CapacityRequests = []string{"capacity-ref-001"}
	run.CapacityDecisions = []string{"capacity-ref-001#capacity_decision:decision-ref-001"}
	run.ConcurrencyGates = []string{"gate-ref-001#decision:allow_request_agent#plan:plan-ref-001"}
	run.Deliveries = []string{"delivery-ref-001"}
	run.Reviews = []string{"review-request-ref-001"}
	run.ReviewResults = []string{
		"review-result-ref-001#review_result:changes_requested#review_request:review-request-ref-001#delivery:delivery-ref-001",
	}
	run.ReworkRequests = []string{
		"rework-request-ref-001#review_result:review-result-ref-001#review_request:review-request-ref-001#delivery:delivery-ref-001",
	}
	run.Agents = append(run.Agents, "agent-ref-stopped-001")
	run.StartedAgents = append(run.StartedAgents, "agent-ref-stopped-001")
	run.StoppedAgents = append(run.StoppedAgents, "agent-ref-stopped-001")
	run.ConfirmedStoppedAgents = append(run.ConfirmedStoppedAgents, "agent-ref-stopped-001")
	request := DirectorTickInputBuildRequestV0{
		TickRef:           "tick-ref-001",
		OccurredAt:        "2026-05-06T12:00:00Z",
		Run:               run,
		PendingOutboxRefs: []string{"outbox-ref-001", " outbox-ref-001 "},
		EvidenceRefs:      []string{"evidence-ref-001"},
	}

	input, err := BuildDirectorSchedulerTickInputV0(request)
	if err != nil {
		t.Fatalf("build tick input: %v", err)
	}
	if input.RunRef != run.RunID || input.Snapshot.CurrentPhaseID != string(run.CurrentPhase) {
		t.Fatalf("unexpected snapshot: %+v", input.Snapshot)
	}
	if !reflect.DeepEqual(input.Snapshot.CapacityDecisions, []string{"capacity-ref-001"}) {
		t.Fatalf("capacity decisions=%v", input.Snapshot.CapacityDecisions)
	}
	if !reflect.DeepEqual(input.Snapshot.ConcurrencyGates, []string{"gate-ref-001"}) {
		t.Fatalf("concurrency gates=%v", input.Snapshot.ConcurrencyGates)
	}
	if !reflect.DeepEqual(input.Snapshot.ReworkRequests, run.ReworkRequests) {
		t.Fatalf("rework requests=%v", input.Snapshot.ReworkRequests)
	}
	if !reflect.DeepEqual(input.Snapshot.PendingOutboxRefs, []string{"outbox-ref-001"}) {
		t.Fatalf("pending outbox=%v", input.Snapshot.PendingOutboxRefs)
	}
	if !reflect.DeepEqual(input.Snapshot.ConfirmedStoppedAgents, []string{"agent-ref-stopped-001"}) {
		t.Fatalf("confirmed_stopped_agents=%v", input.Snapshot.ConfirmedStoppedAgents)
	}
}

func TestBuildDirectorSchedulerTickInputV0DerivaQualityGatesBloqueantesPendientes(t *testing.T) {
	run := tickInputProgramacionRunV0(t)
	run.QualityGates = []string{
		"quality-gate-ref-blocked-cleared#decision:blocked#subject:subject-ref-shared",
		"quality-gate-ref-blocked-pending#decision:blocked#subject:subject-ref-pending",
		"quality-gate-ref-accepted#decision:accepted#subject:subject-ref-shared",
	}
	request := DirectorTickInputBuildRequestV0{
		TickRef:    "tick-ref-quality-gate-001",
		OccurredAt: "2026-05-06T12:00:00Z",
		Run:        run,
	}

	input, err := BuildDirectorSchedulerTickInputV0(request)
	if err != nil {
		t.Fatalf("build tick input: %v", err)
	}
	want := []string{"quality-gate-ref-blocked-pending"}
	if !reflect.DeepEqual(input.Snapshot.BlockingQualityGateRefs, want) {
		t.Fatalf("blocking quality gates=%v, want %v", input.Snapshot.BlockingQualityGateRefs, want)
	}
}

func TestBuildDirectorSchedulerTickInputV0FiltraProgressCandidatesDeOtroRun(t *testing.T) {
	run := tickInputProgramacionRunV0(t)
	request := DirectorTickInputBuildRequestV0{
		TickRef:    "tick-ref-progress-scope-001",
		OccurredAt: "2026-05-06T12:00:00Z",
		Run:        run,
		ProgressSupervisionCandidates: []orquestadirectorscheduler.SchedulableProgressSupervisionCandidateV0{
			tickInputProgressCandidateForRunV0("progress-candidate-foreign", "run-tick-input-otro", "agent-ref-foreign"),
			tickInputProgressCandidateWithoutCommandRunV0("progress-candidate-no-meta", run.RunID, "agent-ref-no-meta"),
			tickInputProgressCandidateWithoutReportRunV0("progress-candidate-no-report", run.RunID, "agent-ref-no-report"),
			tickInputProgressCandidateForRunV0("progress-candidate-local", run.RunID, "agent-ref-local"),
		},
	}

	input, err := BuildDirectorSchedulerTickInputV0(request)
	if err != nil {
		t.Fatalf("build tick input: %v", err)
	}
	if len(input.ProgressSupervisionCandidates) != 1 ||
		input.ProgressSupervisionCandidates[0].CandidateRef != "progress-candidate-local" {
		t.Fatalf("progress candidates=%+v", input.ProgressSupervisionCandidates)
	}
}

func TestBuildDirectorSchedulerTickInputV0CompactaAssessmentProyectadoLargo(t *testing.T) {
	run := tickInputProgramacionRunV0(t)
	payload := orquestacoreworkflow.AgentWorkAssessedPayloadV0{
		AssessmentRef:  "assessment-ref-progress-scheduler-compact-001",
		PhaseID:        string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		AgentRequestID: "agent-ref-" + strings.Repeat("assessment-agent-", 20),
		TaskRef:        "workflow-task-" + strings.Repeat("programacion-", 12),
		DeliveryRef:    "ack-ref-" + strings.Repeat("assessment-delivery-", 20),
		Verdict:        orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0,
		Action:         orquestacoreworkflow.AgentAssessmentActionAskDirectorV0,
		Severity:       orquestacoreworkflow.AgentAssessmentSeverityHighV0,
	}
	projection := orquestacoreworkflow.AgentAssessmentProjectionRefV0(payload)
	if len(projection) <= maxTickInputStringV0 {
		t.Fatalf("fixture no reproduce projection larga: len=%d", len(projection))
	}
	run.AgentAssessments = []string{projection}

	input, err := BuildDirectorSchedulerTickInputV0(DirectorTickInputBuildRequestV0{
		TickRef:    "tick-ref-assessment-compact-001",
		OccurredAt: "2026-05-06T12:00:00Z",
		Run:        run,
	})
	if err != nil {
		t.Fatalf("build tick input: %v", err)
	}
	if !reflect.DeepEqual(input.Snapshot.AgentAssessments, []string{payload.AssessmentRef}) {
		t.Fatalf("agent_assessments=%v, want [%s]", input.Snapshot.AgentAssessments, payload.AssessmentRef)
	}
}

func TestBuildDirectorSchedulerTickInputV0AceptaRefsArtefactoSinCortar(t *testing.T) {
	run := tickInputProgramacionRunV0(t)

	input, err := BuildDirectorSchedulerTickInputV0(DirectorTickInputBuildRequestV0{
		TickRef:           "tick-ref-artifact-refs-open-001",
		OccurredAt:        "2026-05-06T12:00:00Z",
		Run:               run,
		PendingOutboxRefs: []string{"README.md"},
		EvidenceRefs: []string{
			"README.md",
			"docs/arquitectura.md",
			"artifact-ref-" + strings.Repeat("x", 900),
		},
	})
	if err != nil {
		t.Fatalf("build tick input: %v", err)
	}
	if !reflect.DeepEqual(input.Snapshot.PendingOutboxRefs, []string{"README.md"}) {
		t.Fatalf("pending_outbox_refs=%v", input.Snapshot.PendingOutboxRefs)
	}
	if len(input.EvidenceRefs) != 3 {
		t.Fatalf("evidence_refs=%v", input.EvidenceRefs)
	}
}

func TestBuildDirectorSchedulerTickInputV0RejectsIncompleto(t *testing.T) {
	_, err := BuildDirectorSchedulerTickInputV0(DirectorTickInputBuildRequestV0{})
	if err == nil {
		t.Fatal("expected validation error")
	}
	var publicErr DirectorTickInputBuildErrorV0
	if !errors.As(err, &publicErr) || publicErr.Code != ErrDirectorTickInputBuildInvalidoV0 {
		t.Fatalf("unexpected error: %v", err)
	}
}

func tickInputProgressCandidateWithoutCommandRunV0(
	candidateRef string,
	runRef string,
	agentRef string,
) orquestadirectorscheduler.SchedulableProgressSupervisionCandidateV0 {
	candidate := tickInputProgressCandidateForRunV0(candidateRef, runRef, agentRef)
	candidate.SupervisionInput.CommandMeta.RunID = ""
	return candidate
}

func tickInputProgressCandidateWithoutReportRunV0(
	candidateRef string,
	runRef string,
	agentRef string,
) orquestadirectorscheduler.SchedulableProgressSupervisionCandidateV0 {
	candidate := tickInputProgressCandidateForRunV0(candidateRef, runRef, agentRef)
	candidate.SupervisionInput.Report.RunID = ""
	return candidate
}

func tickInputProgressCandidateForRunV0(
	candidateRef string,
	runRef string,
	agentRef string,
) orquestadirectorscheduler.SchedulableProgressSupervisionCandidateV0 {
	return orquestadirectorscheduler.SchedulableProgressSupervisionCandidateV0{
		CandidateRef: candidateRef,
		SupervisionInput: orquestadirector.AgentProgressSupervisionInputV0{
			CommandMeta: orquestacoreworkflow.OrchestrationCommandMetaV0{RunID: runRef},
			Report: orquestaruntime.AgentProgressReportV0{
				ReportID:            "progress-report-ref-" + agentRef,
				RunID:               runRef,
				AgentRequestID:      agentRef,
				Status:              orquestaruntime.AgentLoopDetectedV0,
				NoProgressTicks:     4,
				RepeatedActionCount: 3,
				Summary:             "Evidencia compacta de progreso del agente.",
				EvidenceRefs:        []string{"evidence-ref-progress-scope-001"},
			},
			PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			AssessmentRef: "assessment-ref-" + agentRef,
		},
		EvidenceRefs: []string{"evidence-ref-progress-scope-001"},
	}
}
