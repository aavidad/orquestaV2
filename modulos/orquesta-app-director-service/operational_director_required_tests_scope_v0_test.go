package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"testing"
)

func TestOperationalDirectorPlanRequiredTestsSeAcotanPorWorkflowTaskV0(t *testing.T) {
	runRef := "run-ref-required-tests-scope"
	taskA := serviceRequiredTestsScopeTaskForTestV0(t, runRef, "task-required-tests-scope-a", []string{
		"go test ./...",
		"git diff --check",
	})
	taskB := serviceRequiredTestsScopeTaskForTestV0(t, runRef, "task-required-tests-scope-b", []string{
		"go test ./...",
		"git diff --check",
		"smoke HTTP completo",
	})
	store := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(taskA, taskB)
	requiredTests := []string{"go test ./...", "git diff --check", "smoke HTTP completo"}
	activeStep := orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{
		TaskRefs: []string{taskA.TaskID, taskB.TaskID},
	}
	matches := []operationalDirectorPlanAcceptedReviewMatchV0{
		serviceRequiredTestsScopeMatchForTestV0(taskA.TaskID),
		serviceRequiredTestsScopeMatchForTestV0(taskB.TaskID),
	}

	requiredByTask, err := operationalDirectorPlanRequiredTestsByTaskV0(
		context.Background(),
		runRef,
		store,
		activeStep,
		requiredTests,
		matches,
	)
	if err != nil {
		t.Fatalf("operationalDirectorPlanRequiredTestsByTaskV0: %v", err)
	}
	evidence := []orquestacionnucleoapp.RequiredTestEvidenceV0{
		serviceRequiredTestsScopeEvidenceForTestV0(runRef, matches[0], "go test ./..."),
		serviceRequiredTestsScopeEvidenceForTestV0(runRef, matches[0], "git diff --check"),
		serviceRequiredTestsScopeEvidenceForTestV0(runRef, matches[1], "go test ./..."),
		serviceRequiredTestsScopeEvidenceForTestV0(runRef, matches[1], "git diff --check"),
		serviceRequiredTestsScopeEvidenceForTestV0(runRef, matches[1], "smoke HTTP completo"),
	}

	evaluation := operationalDirectorPlanEvaluateRequiredTestEvidenceV0(
		runRef,
		requiredTests,
		matches,
		evidence,
		requiredByTask,
	)

	if !evaluation.Complete ||
		len(evaluation.FailedRefs) != 0 ||
		len(evaluation.PassedRefs) != 5 ||
		serviceStringInSetV0(requiredByTask[taskA.TaskID], "smoke HTTP completo") {
		t.Fatalf("evaluation=%+v requiredByTask=%+v", evaluation, requiredByTask)
	}
}

func TestOperationalDirectorPlanRequiredTestsNoHeredaFallbackEnWorkflowTaskSinTestsV0(t *testing.T) {
	runRef := "run-ref-required-tests-review-without-tests"
	codeTask := serviceRequiredTestsScopeTaskForTestV0(t, runRef, "task-required-tests-code", []string{
		"go test ./...",
	})
	reviewTask := serviceRequiredTestsScopeTaskForTestV0(t, runRef, "task-required-tests-review", nil)
	store := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(codeTask, reviewTask)
	requiredTests := []string{"go test ./..."}
	activeStep := orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{
		TaskRefs: []string{codeTask.TaskID, reviewTask.TaskID},
	}
	matches := []operationalDirectorPlanAcceptedReviewMatchV0{
		serviceRequiredTestsScopeMatchForTestV0(codeTask.TaskID),
		serviceRequiredTestsScopeMatchForTestV0(reviewTask.TaskID),
	}

	requiredByTask, err := operationalDirectorPlanRequiredTestsByTaskV0(
		context.Background(),
		runRef,
		store,
		activeStep,
		requiredTests,
		matches,
	)
	if err != nil {
		t.Fatalf("operationalDirectorPlanRequiredTestsByTaskV0: %v", err)
	}
	evidence := []orquestacionnucleoapp.RequiredTestEvidenceV0{
		serviceRequiredTestsScopeEvidenceForTestV0(runRef, matches[0], "go test ./..."),
	}

	evaluation := operationalDirectorPlanEvaluateRequiredTestEvidenceV0(
		runRef,
		requiredTests,
		matches,
		evidence,
		requiredByTask,
	)

	if !evaluation.Complete ||
		len(evaluation.FailedRefs) != 0 ||
		len(evaluation.PassedRefs) != 1 ||
		len(requiredByTask[reviewTask.TaskID]) != 0 {
		t.Fatalf("evaluation=%+v requiredByTask=%+v", evaluation, requiredByTask)
	}
}

func TestOperationalDirectorPlanRequiredTestsAceptaEvidenciaRunScopedCondicionalV0(t *testing.T) {
	runRef := "run-ref-required-tests-run-scoped"
	requiredTests := []string{"smoke HTTP completo"}
	matches := []operationalDirectorPlanAcceptedReviewMatchV0{
		serviceRequiredTestsScopeMatchForTestV0("task-required-tests-run-scoped-a"),
		serviceRequiredTestsScopeMatchForTestV0("task-required-tests-run-scoped-b"),
	}
	evidence := []orquestacionnucleoapp.RequiredTestEvidenceV0{
		serviceRequiredTestsScopeEvidenceForTestV0(runRef, matches[1], "smoke HTTP completo"),
	}

	evaluation := operationalDirectorPlanEvaluateRequiredTestEvidenceV0(
		runRef,
		requiredTests,
		matches,
		evidence,
		nil,
	)

	if !evaluation.Complete ||
		len(evaluation.FailedRefs) != 0 ||
		len(evaluation.PassedRefs) != 1 {
		t.Fatalf("evaluation=%+v", evaluation)
	}
}

func serviceRequiredTestsScopeTaskForTestV0(
	t *testing.T,
	runRef string,
	taskRef string,
	requiredTests []string,
) orquestacoreworkflow.WorkflowTaskV0 {
	t.Helper()
	task, err := orquestacoreworkflow.NewWorkflowTaskV0(orquestacoreworkflow.WorkflowTaskV0{
		TaskID:             taskRef,
		RunID:              runRef,
		PhaseID:            orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Title:              "Tarea con tests requeridos acotados",
		Summary:            "Fixture para scope de required tests.",
		WriteSet:           []string{"internal/app"},
		AcceptanceCriteria: []string{"tests por tarea"},
		RequiredTests:      requiredTests,
	})
	if err != nil {
		t.Fatalf("NewWorkflowTaskV0: %v", err)
	}
	return task
}

func serviceRequiredTestsScopeMatchForTestV0(taskRef string) operationalDirectorPlanAcceptedReviewMatchV0 {
	return operationalDirectorPlanAcceptedReviewMatchV0{
		TaskRef:           taskRef,
		AgentRef:          "agent-ref-" + taskRef,
		DeliveryRef:       "delivery-ref-" + taskRef,
		ReviewRequestID:   "review-request-ref-" + taskRef,
		ReviewResultRef:   "review-result-ref-" + taskRef,
		AcceptedReviewRef: "accepted-review-ref-" + taskRef,
		EvidenceRefs:      []string{"evidence-ref-" + taskRef},
	}
}

func serviceRequiredTestsScopeEvidenceForTestV0(
	runRef string,
	match operationalDirectorPlanAcceptedReviewMatchV0,
	required string,
) orquestacionnucleoapp.RequiredTestEvidenceV0 {
	return orquestacionnucleoapp.RequiredTestEvidenceV0{
		SchemaVersion:     orquestacionnucleoapp.RequiredTestEvidenceSchemaVersionV0,
		EvidenceRef:       "test-evidence-ref-" + match.TaskRef + "-" + appDirectorSafeRefPartV0(required),
		RunRef:            runRef,
		TaskRef:           match.TaskRef,
		TestCommand:       required,
		Status:            orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0,
		DeliveryRef:       match.DeliveryRef,
		ReviewRequestID:   match.ReviewRequestID,
		ReviewResultRef:   match.ReviewResultRef,
		AcceptedReviewRef: match.AcceptedReviewRef,
		OccurredAt:        "2026-05-27T21:30:00Z",
	}
}
