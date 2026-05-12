package orquestacionnucleoapp

import (
	"context"
	"errors"
	"testing"

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
