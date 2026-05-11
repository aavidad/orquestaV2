package orquestaappdirectorservice

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/orquestacionnucleoapp"
)

func TestComposeStartAppDirectorProviderV0IncludesReviewReworkReplanSource(t *testing.T) {
	source := &recordingReviewReworkReplanSourceV0{}
	provider := composeStartAppDirectorProviderV0(
		orquestacionnucleoapp.StaticCandidateProviderV0{},
		StartAppDirectorPortsV0{ReviewReworkReplanSource: source},
		"director-service-test",
	)

	_, err := provider.BuildSchedulerCandidatesV0(context.Background(), orquestacionnucleoapp.SchedulerCandidateRequestV0{
		Run: orquestacoreworkflow.OrchestrationRunV0{
			RunID:        "run-ref-service-review-rework-001",
			CurrentPhase: orquestacoreworkflow.OrchestrationPhaseRevisionV0,
		},
		OccurredAt:    "2026-05-10T10:05:00Z",
		CorrelationID: "corr-service-review-rework-001",
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	if !source.called {
		t.Fatalf("review rework replan source no fue invocado")
	}
}

type recordingReviewReworkReplanSourceV0 struct {
	called bool
}

func (source *recordingReviewReworkReplanSourceV0) BuildReviewReworkReplanPlansV0(
	context.Context,
	orquestacionnucleoapp.ReviewReworkReplanPlanRequestV0,
) ([]orquestacionnucleoapp.ReviewReworkReplanPlanV0, error) {
	source.called = true
	return nil, nil
}
