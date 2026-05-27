package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestacoreleases "orquesta/modulos/orquesta-core-leases"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestAgentProgressLeaseBridgeV0ReusaObservacionParaLeaseCandidate(t *testing.T) {
	runRef := "run-nucleo-progress-lease-bridge-001"
	agentRef := "agent-ref-progress-lease-001"
	source := &countingProgressLeaseObservationSourceV0{
		Observations: []AgentProgressObservationV0{{
			Report: progressLeaseBridgeReportV0(runRef, agentRef),
		}},
	}
	bridge := &AgentProgressLeaseBridgeV0{
		ProgressSource: source,
		PolicySource: StaticAgentProgressLeasePolicyProviderV0{
			Policy: progressLeaseBridgePolicyV0(),
		},
	}
	provider := AgentLeaseActionCandidateProviderV0{
		Base: ProgressSupervisionCandidateProviderV0{
			ProgressSource: bridge,
			RequestedBy:    "orquesta-nucleo-test",
		},
		LeaseSource: bridge,
		RequestedBy: "orquesta-nucleo-test",
	}

	candidates, err := provider.BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:           mustActiveProgrammingRunV0(t, runRef),
		OccurredAt:    "2026-05-09T17:02:01Z",
		CorrelationID: "corr-progress-lease-bridge-001",
		EvidenceRefs:  []string{"evidence-ref-progress-lease-request-001"},
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	if source.Calls != 1 {
		t.Fatalf("progress source calls=%d want 1", source.Calls)
	}
	if len(candidates.ProgressSupervisionCandidates) != 0 {
		t.Fatalf("progress candidates=%+v", candidates.ProgressSupervisionCandidates)
	}
	if len(candidates.LeaseActionCandidates) != 1 {
		t.Fatalf("lease candidates=%+v", candidates.LeaseActionCandidates)
	}
	got := candidates.LeaseActionCandidates[0].PostLeaseActionInput
	if got.AgentRequestID != agentRef ||
		got.RecommendedAction != "stop_agent" ||
		got.ReasonCode != orquestacoreleases.AgentTimeoutReasonHeartbeatTimeoutV0 {
		t.Fatalf("lease input=%+v", got)
	}
}

func TestBuildAgentLeaseAssessmentFromProgressV0ProgressNuevoContinua(t *testing.T) {
	observation := AgentProgressObservationV0{
		Report: progressLeaseBridgeReportV0("run-nucleo-progress-lease-continue-001", "agent-ref-progress-lease-continue-001"),
	}
	observation.Report.Status = orquestaruntime.AgentProgressingV0
	observation.Report.LastActivityAt = "2026-05-09T17:01:59Z"

	assessment, ready, err := BuildAgentLeaseAssessmentFromProgressV0(AgentProgressLeaseAssessmentInputV0{
		ObservedAt: "2026-05-09T17:02:01Z",
		Report:     observation,
		Policy:     progressLeaseBridgePolicyV0(),
	})
	if err != nil {
		t.Fatalf("BuildAgentLeaseAssessmentFromProgressV0: %v", err)
	}
	if !ready || assessment.Decision != orquestacoreleases.AgentTimeoutDecisionContinueV0 {
		t.Fatalf("assessment=%+v ready=%v", assessment, ready)
	}
}

func TestBuildAgentLeaseAssessmentFromProgressV0ToleraRelojDeReportePosteriorAlTick(t *testing.T) {
	observation := AgentProgressObservationV0{
		Report: progressLeaseBridgeReportV0("run-nucleo-progress-lease-clock-skew-001", "agent-ref-progress-lease-clock-skew-001"),
	}
	observation.Report.StartedAt = "2026-05-09T17:03:00Z"
	observation.Report.LastActivityAt = "2026-05-09T17:03:00Z"

	assessment, ready, err := BuildAgentLeaseAssessmentFromProgressV0(AgentProgressLeaseAssessmentInputV0{
		ObservedAt: "2026-05-09T17:02:01Z",
		Report:     observation,
		Policy:     progressLeaseBridgePolicyV0(),
	})
	if err != nil {
		t.Fatalf("BuildAgentLeaseAssessmentFromProgressV0: %v", err)
	}
	if !ready ||
		assessment.Decision != orquestacoreleases.AgentTimeoutDecisionContinueV0 ||
		assessment.NowObservedAt != "2026-05-09T17:02:01Z" {
		t.Fatalf("assessment=%+v ready=%v", assessment, ready)
	}
}

type countingProgressLeaseObservationSourceV0 struct {
	Observations []AgentProgressObservationV0
	Calls        int
}

func (source *countingProgressLeaseObservationSourceV0) BuildAgentProgressObservationsV0(
	_ context.Context,
	_ AgentProgressObservationRequestV0,
) ([]AgentProgressObservationV0, error) {
	source.Calls++
	return source.Observations, nil
}

func progressLeaseBridgeReportV0(
	runRef string,
	agentRef string,
) orquestaruntime.AgentProgressReportV0 {
	return orquestaruntime.AgentProgressReportV0{
		ReportID:             "agent-progress-report-ref-lease-bridge-001",
		RunID:                runRef,
		AgentRequestID:       agentRef,
		Status:               orquestaruntime.AgentStalledV0,
		BudgetStatus:         orquestaruntime.AgentProgressBudgetOverBudgetNoActivityV0,
		NoProgressTicks:      8,
		RepeatedActionCount:  0,
		SecondsSinceActivity: 121,
		StartedAt:            "2026-05-09T17:00:00Z",
		LastActivityAt:       "2026-05-09T17:00:00Z",
		Summary:              "Progreso compacto sin actividad reciente.",
		EvidenceRefs:         []string{"evidence-ref-progress-lease-bridge-001"},
	}
}

func progressLeaseBridgePolicyV0() orquestacoreleases.AgentLeasePolicyV0 {
	return orquestacoreleases.AgentLeasePolicyV0{
		LeasePolicyRef:          "lease-policy-ref-progress-lease-bridge-001",
		LaunchTimeoutSeconds:    30,
		HeartbeatTimeoutSeconds: 60,
		TotalTimeoutSeconds:     900,
		TimeoutAction:           orquestacoreleases.AgentLeaseTimeoutStopAgentV0,
		EvidenceRefs:            []string{"evidence-ref-progress-lease-policy-001"},
	}
}
