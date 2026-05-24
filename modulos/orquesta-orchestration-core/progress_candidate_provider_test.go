package orquestacionnucleoapp

import (
	"context"
	"errors"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestProgressSupervisionCandidateProviderV0BuildsCandidatesFromObservationPort(t *testing.T) {
	runRef := "run-nucleo-progress-observacion-001"
	provider := ProgressSupervisionCandidateProviderV0{
		Base: StaticCandidateProviderV0{Candidates: SchedulerCandidateSetV0{
			WorkCandidates: []orquestadirectorscheduler.SchedulableWorkCandidateV0{
				workCandidateWithAgentV0(runRef),
			},
			EvidenceRefs: []string{"evidence-ref-base-observacion-001"},
		}},
		ProgressSource: staticAgentProgressObservationSourceV0{Observations: []AgentProgressObservationV0{
			{
				Report:       progressObservationReportV0(runRef, "agent-ref-001", orquestaruntime.AgentLoopDetectedV0),
				TaskRef:      "task-ref-progress-observacion-001",
				EvidenceRefs: []string{"evidence-ref-progress-observacion-001"},
			},
		}},
		RequestedBy: "orquestacion-nucleo",
	}

	candidates, err := provider.BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:           mustActiveProgrammingRunV0(t, runRef),
		StepNumber:    2,
		MaxSteps:      4,
		OccurredAt:    "2026-05-09T12:10:00Z",
		CorrelationID: "corr-progress-observacion-001",
		EvidenceRefs:  []string{"evidence-ref-request-observacion-001"},
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	if len(candidates.WorkCandidates) != 1 {
		t.Fatalf("base work candidates=%d", len(candidates.WorkCandidates))
	}
	if len(candidates.ProgressSupervisionCandidates) != 1 {
		t.Fatalf("progress candidates=%+v", candidates.ProgressSupervisionCandidates)
	}
	candidate := candidates.ProgressSupervisionCandidates[0]
	if candidate.SupervisionInput.CommandMeta.OccurredAt != "2026-05-09T12:10:00Z" ||
		candidate.SupervisionInput.CommandMeta.CorrelationID != "corr-progress-observacion-001" {
		t.Fatalf("command meta=%+v", candidate.SupervisionInput.CommandMeta)
	}
	if candidate.SupervisionInput.PhaseID == "" ||
		candidate.SupervisionInput.AssessmentRef == "" ||
		candidate.SupervisionInput.Report.Status != orquestaruntime.AgentLoopDetectedV0 {
		t.Fatalf("candidate=%+v", candidate)
	}
}

func TestProgressSupervisionCandidateProviderV0LimitaPrimeraObservacionAccionable(t *testing.T) {
	runRef := "run-nucleo-progress-observacion-first-001"
	provider := ProgressSupervisionCandidateProviderV0{
		ProgressSource: staticAgentProgressObservationSourceV0{Observations: []AgentProgressObservationV0{
			{
				Report:  progressObservationReportWithIDV0(runRef, "agent-ref-progressing-001", "report-progressing-001", orquestaruntime.AgentProgressingV0),
				TaskRef: "task-ref-progressing-001",
			},
			{
				Report:  progressObservationReportWithIDV0(runRef, "agent-ref-loop-001", "report-loop-001", orquestaruntime.AgentLoopDetectedV0),
				TaskRef: "task-ref-loop-001",
			},
			{
				Report:  progressObservationReportWithIDV0(runRef, "agent-ref-stalled-001", "report-stalled-001", orquestaruntime.AgentStalledV0),
				TaskRef: "task-ref-stalled-001",
			},
		}},
		RequestedBy: "orquestacion-nucleo",
	}

	candidates, err := provider.BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:        mustActiveProgrammingRunV0(t, runRef),
		OccurredAt: "2026-05-09T12:10:00Z",
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	if len(candidates.ProgressSupervisionCandidates) != 1 {
		t.Fatalf("progress candidates=%+v", candidates.ProgressSupervisionCandidates)
	}
	got := candidates.ProgressSupervisionCandidates[0].SupervisionInput.Report.ReportID
	if got != "report-loop-001" {
		t.Fatalf("report id=%s", got)
	}
}

func TestProgressSupervisionCandidateProviderV0IgnoraStalledInformativo(t *testing.T) {
	runRef := "run-nucleo-progress-stalled-informativo-001"
	provider := ProgressSupervisionCandidateProviderV0{
		ProgressSource: staticAgentProgressObservationSourceV0{Observations: []AgentProgressObservationV0{
			{
				Report:  progressObservationReportWithIDV0(runRef, "agent-ref-stalled-info-001", "report-stalled-info-001", orquestaruntime.AgentStalledV0),
				TaskRef: "task-ref-stalled-info-001",
			},
		}},
		RequestedBy: "orquestacion-nucleo",
	}

	candidates, err := provider.BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:        mustActiveProgrammingRunV0(t, runRef),
		OccurredAt: "2026-05-09T12:10:00Z",
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	if len(candidates.ProgressSupervisionCandidates) != 0 {
		t.Fatalf("stalled informativo no debe generar candidate=%+v", candidates.ProgressSupervisionCandidates)
	}
}

func TestProgressSupervisionCandidateProviderV0CompletaQuestionIDParaStoppedSinAck(t *testing.T) {
	runRef := "run-nucleo-progress-stopped-sin-ack-001"
	report := progressObservationReportWithIDV0(
		runRef,
		"agent-ref-stopped-sin-ack-001",
		"report-stopped-sin-ack-001",
		orquestaruntime.AgentStoppedV0,
	)
	report.DecisionRequired = true
	provider := ProgressSupervisionCandidateProviderV0{
		ProgressSource: staticAgentProgressObservationSourceV0{Observations: []AgentProgressObservationV0{
			{
				Report:       report,
				TaskRef:      "task-ref-stopped-sin-ack-001",
				EvidenceRefs: []string{"evidence-ref-artifact-without-ack-001"},
			},
		}},
		RequestedBy: "orquestacion-nucleo",
	}

	candidates, err := provider.BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:        mustActiveProgrammingRunV0(t, runRef),
		OccurredAt: "2026-05-09T12:10:00Z",
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	if len(candidates.ProgressSupervisionCandidates) != 1 {
		t.Fatalf("progress candidates=%+v", candidates.ProgressSupervisionCandidates)
	}
	candidate := candidates.ProgressSupervisionCandidates[0]
	if candidate.SupervisionInput.QuestionID == "" {
		t.Fatalf("question_id vacio para stopped sin ack: %+v", candidate.SupervisionInput)
	}
	if candidate.SupervisionInput.QuestionID != "question-ref-report-stopped-sin-ack-001" {
		t.Fatalf("question_id=%s", candidate.SupervisionInput.QuestionID)
	}
}

func TestProgressSupervisionCandidateProviderV0UsaRefsEstablesParaPreguntaAdvisory(t *testing.T) {
	runRef := "run-nucleo-progress-advisory-estable-001"
	agentRef := "agent-ref-advisory-estable-001"
	first := progressObservationDecisionReportV0(runRef, agentRef, orquestaruntime.AgentStalledV0)
	first.ReportID = "agent-progress-report-ref-advisory-estable-001"
	second := first
	second.ReportID = "agent-progress-report-ref-advisory-estable-002"

	firstCandidate := progressCandidateFromSingleObservationForTestV0(t, runRef, AgentProgressObservationV0{
		Report:  first,
		TaskRef: "task-ref-advisory-estable-001",
	})
	secondCandidate := progressCandidateFromSingleObservationForTestV0(t, runRef, AgentProgressObservationV0{
		Report:  second,
		TaskRef: "task-ref-advisory-estable-001",
	})

	if firstCandidate.SupervisionInput.AssessmentRef == "" ||
		firstCandidate.SupervisionInput.QuestionID == "" {
		t.Fatalf("refs advisory vacias: %+v", firstCandidate.SupervisionInput)
	}
	if firstCandidate.SupervisionInput.AssessmentRef != secondCandidate.SupervisionInput.AssessmentRef ||
		firstCandidate.SupervisionInput.QuestionID != secondCandidate.SupervisionInput.QuestionID {
		t.Fatalf("refs advisory no estables: first=%+v second=%+v",
			firstCandidate.SupervisionInput,
			secondCandidate.SupervisionInput,
		)
	}
	if firstCandidate.SupervisionInput.QuestionID == "question-ref-"+first.ReportID {
		t.Fatalf("question_id sigue ligado al contador del reporte: %s", firstCandidate.SupervisionInput.QuestionID)
	}
}

func TestProgressSupervisionCandidateProviderV0IgnoraObservacionesDeOtroRun(t *testing.T) {
	runRef := "run-nucleo-progress-observacion-run-scope-001"
	foreignRunRef := "run-nucleo-progress-observacion-run-scope-otro"
	provider := ProgressSupervisionCandidateProviderV0{
		ProgressSource: staticAgentProgressObservationSourceV0{Observations: []AgentProgressObservationV0{
			{
				Report:  progressObservationReportWithIDV0(foreignRunRef, "agent-ref-foreign-001", "report-foreign-001", orquestaruntime.AgentLoopDetectedV0),
				TaskRef: "task-ref-foreign-001",
			},
			{
				Report:  progressObservationReportWithIDV0(runRef, "agent-ref-local-001", "report-local-001", orquestaruntime.AgentLoopDetectedV0),
				TaskRef: "task-ref-local-001",
			},
		}},
		RequestedBy: "orquestacion-nucleo",
	}

	candidates, err := provider.BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:        mustActiveProgrammingRunV0(t, runRef),
		OccurredAt: "2026-05-09T12:10:00Z",
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	if len(candidates.ProgressSupervisionCandidates) != 1 {
		t.Fatalf("progress candidates=%+v", candidates.ProgressSupervisionCandidates)
	}
	got := candidates.ProgressSupervisionCandidates[0].SupervisionInput.Report.ReportID
	if got != "report-local-001" {
		t.Fatalf("report id=%s", got)
	}
}

func progressCandidateFromSingleObservationForTestV0(
	t *testing.T,
	runRef string,
	observation AgentProgressObservationV0,
) orquestadirectorscheduler.SchedulableProgressSupervisionCandidateV0 {
	t.Helper()
	provider := ProgressSupervisionCandidateProviderV0{
		ProgressSource: staticAgentProgressObservationSourceV0{
			Observations: []AgentProgressObservationV0{observation},
		},
		RequestedBy: "orquestacion-nucleo",
	}
	candidates, err := provider.BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:        mustActiveProgrammingRunV0(t, runRef),
		OccurredAt: "2026-05-09T12:10:00Z",
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	if len(candidates.ProgressSupervisionCandidates) != 1 {
		t.Fatalf("progress candidates=%+v", candidates.ProgressSupervisionCandidates)
	}
	return candidates.ProgressSupervisionCandidates[0]
}

func TestProgressSupervisionCandidateProviderV0FiltraCandidatesBaseDeOtroRun(t *testing.T) {
	runRef := "run-nucleo-progress-base-run-scope-001"
	foreignRunRef := "run-nucleo-progress-base-run-scope-otro"
	foreignCandidate := orquestadirectorscheduler.SchedulableProgressSupervisionCandidateV0{
		CandidateRef: "progress-candidate-ref-foreign-001",
		SupervisionInput: orquestadirector.AgentProgressSupervisionInputV0{
			CommandMeta: orquestacoreworkflow.OrchestrationCommandMetaV0{RunID: foreignRunRef},
			Report:      progressObservationReportWithIDV0(foreignRunRef, "agent-ref-foreign-001", "report-foreign-001", orquestaruntime.AgentLoopDetectedV0),
		},
	}
	localCandidate := orquestadirectorscheduler.SchedulableProgressSupervisionCandidateV0{
		CandidateRef: "progress-candidate-ref-local-001",
		SupervisionInput: orquestadirector.AgentProgressSupervisionInputV0{
			CommandMeta: orquestacoreworkflow.OrchestrationCommandMetaV0{RunID: runRef},
			Report:      progressObservationReportWithIDV0(runRef, "agent-ref-local-001", "report-local-001", orquestaruntime.AgentLoopDetectedV0),
		},
	}
	noMetaCandidate := localCandidate
	noMetaCandidate.CandidateRef = "progress-candidate-ref-no-meta-001"
	noMetaCandidate.SupervisionInput.CommandMeta.RunID = ""
	noReportCandidate := localCandidate
	noReportCandidate.CandidateRef = "progress-candidate-ref-no-report-001"
	noReportCandidate.SupervisionInput.Report.RunID = ""
	provider := ProgressSupervisionCandidateProviderV0{
		Base: StaticCandidateProviderV0{Candidates: SchedulerCandidateSetV0{
			ProgressSupervisionCandidates: []orquestadirectorscheduler.SchedulableProgressSupervisionCandidateV0{
				foreignCandidate,
				noMetaCandidate,
				noReportCandidate,
				localCandidate,
			},
		}},
	}

	candidates, err := provider.BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:        mustActiveProgrammingRunV0(t, runRef),
		OccurredAt: "2026-05-09T12:10:00Z",
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	if len(candidates.ProgressSupervisionCandidates) != 1 ||
		candidates.ProgressSupervisionCandidates[0].CandidateRef != "progress-candidate-ref-local-001" {
		t.Fatalf("progress candidates=%+v", candidates.ProgressSupervisionCandidates)
	}
}

func TestProgressSupervisionCandidateProviderV0ProtectsPrimaryBrainstormingDirector(t *testing.T) {
	runRef := "run-nucleo-progress-primary-director-001"
	agentRef := "agent-spec-progress-primary-director"
	candidate := progressSupervisionCandidateForBrainstormingAgentV0(t, runRef, agentRef)

	if candidate.SupervisionInput.StopAllowed == nil ||
		*candidate.SupervisionInput.StopAllowed ||
		candidate.SupervisionInput.QuestionID == "" {
		t.Fatalf("candidate primary director=%+v", candidate.SupervisionInput)
	}
}

func TestProgressSupervisionCandidateProviderV0DoesNotProtectSpecializedBrainstormingDirector(t *testing.T) {
	runRef := "run-nucleo-progress-specialized-director-001"
	agentRef := "agent-spec-agenda-api-web-req-agenda-api-web-001-web"
	candidate := progressSupervisionCandidateForBrainstormingAgentV0(t, runRef, agentRef)

	if candidate.SupervisionInput.StopAllowed != nil {
		t.Fatalf("candidate specialized director=%+v", candidate.SupervisionInput)
	}
}

func progressSupervisionCandidateForBrainstormingAgentV0(
	t *testing.T,
	runRef string,
	agentRef string,
) orquestadirectorscheduler.SchedulableProgressSupervisionCandidateV0 {
	t.Helper()
	provider := ProgressSupervisionCandidateProviderV0{
		ProgressSource: staticAgentProgressObservationSourceV0{Observations: []AgentProgressObservationV0{{
			Report: progressObservationDecisionReportV0(
				runRef,
				agentRef,
				orquestaruntime.AgentStalledV0,
			),
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0),
			TaskRef: "task-ref-brainstorming-area-001",
		}}},
		RequestedBy: "orquestacion-nucleo-test",
	}
	candidates, err := provider.BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:        mustActiveBrainstormingRunWithStartedAgentV0(t, runRef, agentRef),
		OccurredAt: "2026-05-09T12:10:00Z",
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	if len(candidates.ProgressSupervisionCandidates) != 1 {
		t.Fatalf("progress candidates=%+v", candidates.ProgressSupervisionCandidates)
	}
	return candidates.ProgressSupervisionCandidates[0]
}

func TestProgressSupervisionCandidateProviderV0PropagatesInvalidProgressReport(t *testing.T) {
	runRef := "run-nucleo-progress-observacion-invalid-001"
	provider := ProgressSupervisionCandidateProviderV0{
		ProgressSource: staticAgentProgressObservationSourceV0{Observations: []AgentProgressObservationV0{
			{Report: orquestaruntime.AgentProgressReportV0{}},
		}},
	}

	_, err := provider.BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:        mustActiveProgrammingRunV0(t, runRef),
		OccurredAt: "2026-05-09T12:11:00Z",
	})
	if err == nil {
		t.Fatalf("esperaba error")
	}
	got, ok := err.(ErrorV0)
	if !ok || got.Field != "agent_progress_report.report_id" {
		t.Fatalf("error=%+v", err)
	}
}

func TestProgressSupervisionCandidateProviderV0PropagatesSourceError(t *testing.T) {
	want := errors.New("source unavailable")
	provider := ProgressSupervisionCandidateProviderV0{
		ProgressSource: staticAgentProgressObservationSourceV0{Err: want},
	}

	_, err := provider.BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:        mustActiveProgrammingRunV0(t, "run-nucleo-progress-observacion-error-001"),
		OccurredAt: "2026-05-09T12:12:00Z",
	})
	if !errors.Is(err, want) {
		t.Fatalf("err=%v want=%v", err, want)
	}
}

type staticAgentProgressObservationSourceV0 struct {
	Observations []AgentProgressObservationV0
	Err          error
}

func (source staticAgentProgressObservationSourceV0) BuildAgentProgressObservationsV0(
	context.Context,
	AgentProgressObservationRequestV0,
) ([]AgentProgressObservationV0, error) {
	if source.Err != nil {
		return nil, source.Err
	}
	return source.Observations, nil
}

func progressObservationReportV0(
	runRef string,
	agentRef string,
	status orquestaruntime.AgentProgressStatusV0,
) orquestaruntime.AgentProgressReportV0 {
	return progressObservationReportWithIDV0(
		runRef,
		agentRef,
		"agent-progress-report-ref-observacion-001",
		status,
	)
}

func progressObservationReportWithIDV0(
	runRef string,
	agentRef string,
	reportRef string,
	status orquestaruntime.AgentProgressStatusV0,
) orquestaruntime.AgentProgressReportV0 {
	return orquestaruntime.AgentProgressReportV0{
		ReportID:            reportRef,
		RunID:               runRef,
		AgentRequestID:      agentRef,
		Status:              status,
		NoProgressTicks:     4,
		RepeatedActionCount: 3,
		Summary:             "Evidencia compacta de progreso del agente.",
		EvidenceRefs:        []string{"evidence-ref-progress-observacion-report-001"},
	}
}

func progressObservationDecisionReportV0(
	runRef string,
	agentRef string,
	status orquestaruntime.AgentProgressStatusV0,
) orquestaruntime.AgentProgressReportV0 {
	report := progressObservationReportV0(runRef, agentRef, status)
	report.DecisionRequired = true
	return report
}
