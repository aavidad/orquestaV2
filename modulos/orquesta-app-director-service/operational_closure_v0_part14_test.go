package orquestaappdirectorservice

import (
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"testing"
)

func serviceOperationalClosureRequiredTestEvidenceForTaskRefsV0(
	runRef string,
	refs serviceOperationalClosureTaskRefsForTestV0,
) orquestacionnucleoapp.RequiredTestEvidenceV0 {
	return orquestacionnucleoapp.RequiredTestEvidenceV0{
		SchemaVersion:     orquestacionnucleoapp.RequiredTestEvidenceSchemaVersionV0,
		EvidenceRef:       refs.TestEvidenceRef,
		RunRef:            runRef,
		TaskRef:           refs.TaskRef,
		TestCommand:       "go test ./...",
		Status:            orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0,
		DeliveryRef:       refs.DeliveryRef,
		ReviewRequestID:   refs.ReviewRequestID,
		ReviewResultRef:   refs.ReviewResultRef,
		AcceptedReviewRef: refs.AcceptedRef,
		OccurredAt:        "2026-05-22T12:00:00Z",
		EvidenceRefs:      []string{refs.DeliveryRef, refs.TestEvidenceRef},
	}
}

func mustServiceActiveProgrammingRunForClosureV0(
	t *testing.T,
	runRef string,
) orquestacoreworkflow.OrchestrationRunV0 {
	t.Helper()
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseProgramacionV0)
	run.Tasks = nil
	return run
}
