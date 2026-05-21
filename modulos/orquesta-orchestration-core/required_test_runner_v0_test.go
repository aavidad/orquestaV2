package orquestacionnucleoapp

import (
	"context"
	"reflect"
	"testing"
)

func TestRequiredTestRunnerV0GuardaEvidenciasPasadasConCausalidad(t *testing.T) {
	store := NewInMemoryRequiredTestEvidenceStoreV0()
	executor := &fakeRequiredTestCommandExecutorV0{
		results: map[string]RequiredTestCommandExecutionResultV0{
			"go test -count=1 ./modulos/orquesta-orchestration-core -run TestRequiredTestEvidence": {
				Status:       RequiredTestEvidenceStatusPassedV0,
				EvidenceRefs: []string{"artifact-ref-core-test-output-001"},
			},
			"go test -count=1 ./modulos/orquesta-app-director-service -run TestUpdateOperationalDirectorPlanState": {
				Status:       RequiredTestEvidenceStatusPassedV0,
				EvidenceRefs: []string{"artifact-ref-director-service-test-output-001"},
			},
		},
	}

	result, err := (RequiredTestRunnerV0{
		Executor:       executor,
		EvidenceWriter: store,
	}).RunRequiredTestsV0(context.Background(), requiredTestExecutionRequestForTestV0())
	if err != nil {
		t.Fatalf("RunRequiredTestsV0: %v", err)
	}
	if len(result.Issues) != 0 {
		t.Fatalf("issues=%+v", result.Issues)
	}
	if len(result.PassedEvidenceRefs) != 2 || len(result.FailedEvidenceRefs) != 0 {
		t.Fatalf("result=%+v", result)
	}
	if !reflect.DeepEqual(executor.commands, requiredTestExecutionRequestForTestV0().TestCommands) {
		t.Fatalf("commands=%+v", executor.commands)
	}

	evidence, err := store.LoadRequiredTestEvidenceV0(context.Background(), "run-required-tests-001", result.EvidenceRefs)
	if err != nil {
		t.Fatalf("LoadRequiredTestEvidenceV0: %v", err)
	}
	if len(evidence) != 2 {
		t.Fatalf("evidence=%+v", evidence)
	}
	for _, item := range evidence {
		if item.Status != RequiredTestEvidenceStatusPassedV0 ||
			item.TaskRef != "task-required-tests-001" ||
			item.DeliveryRef != "delivery-ref-required-tests-001" ||
			item.ReviewRequestID != "review-request-ref-required-tests-001" ||
			item.ReviewResultRef != "review-result-ref-required-tests-001" ||
			item.AcceptedReviewRef != "accepted-review-ref-required-tests-001" {
			t.Fatalf("causalidad rota: %+v", item)
		}
		if len(item.EvidenceRefs) < 2 || !requiredTestRunnerStringInSetV0(item.EvidenceRefs, "review-evidence-ref-required-tests-001") {
			t.Fatalf("evidence_refs sin artefacto real ni contexto de review: %+v", item.EvidenceRefs)
		}
	}
}

func TestRequiredTestRunnerV0GuardaEvidenciaFallidaSinMaquillarla(t *testing.T) {
	store := NewInMemoryRequiredTestEvidenceStoreV0()
	request := requiredTestExecutionRequestForTestV0()
	request.TestCommands = []string{"go test -count=1 ./modulos/orquesta-app-director-service -run TestFallaReal"}

	result, err := (RequiredTestRunnerV0{
		Executor: &fakeRequiredTestCommandExecutorV0{
			results: map[string]RequiredTestCommandExecutionResultV0{
				request.TestCommands[0]: {
					Status:       RequiredTestEvidenceStatusFailedV0,
					EvidenceRefs: []string{"artifact-ref-failing-test-output-001"},
				},
			},
		},
		EvidenceWriter: store,
	}).RunRequiredTestsV0(context.Background(), request)
	if err != nil {
		t.Fatalf("RunRequiredTestsV0: %v", err)
	}
	if len(result.FailedEvidenceRefs) != 1 || len(result.PassedEvidenceRefs) != 0 {
		t.Fatalf("result=%+v", result)
	}

	evidence, err := store.LoadRequiredTestEvidenceV0(context.Background(), request.RunRef, result.FailedEvidenceRefs)
	if err != nil {
		t.Fatalf("LoadRequiredTestEvidenceV0: %v", err)
	}
	if evidence[0].Status != RequiredTestEvidenceStatusFailedV0 ||
		evidence[0].TestCommand != request.TestCommands[0] {
		t.Fatalf("evidence=%+v", evidence[0])
	}
}

func TestRequiredTestRunnerV0EsIdempotenteConMismaEvidencia(t *testing.T) {
	store := NewInMemoryRequiredTestEvidenceStoreV0()
	request := requiredTestExecutionRequestForTestV0()
	request.TestCommands = []string{"go test -count=1 ./modulos/orquesta-orchestration-core -run TestRequiredTestRunner"}
	runner := RequiredTestRunnerV0{
		Executor: &fakeRequiredTestCommandExecutorV0{
			results: map[string]RequiredTestCommandExecutionResultV0{
				request.TestCommands[0]: {
					Status:       RequiredTestEvidenceStatusPassedV0,
					EvidenceRefs: []string{"artifact-ref-idempotent-test-output-001"},
				},
			},
		},
		EvidenceWriter: store,
	}

	first, err := runner.RunRequiredTestsV0(context.Background(), request)
	if err != nil {
		t.Fatalf("RunRequiredTestsV0 first: %v", err)
	}
	second, err := runner.RunRequiredTestsV0(context.Background(), request)
	if err != nil {
		t.Fatalf("RunRequiredTestsV0 second: %v", err)
	}
	if !reflect.DeepEqual(first.EvidenceRefs, second.EvidenceRefs) {
		t.Fatalf("refs no estables: first=%+v second=%+v", first.EvidenceRefs, second.EvidenceRefs)
	}
}

func TestRequiredTestRunnerV0RechazaResultadoSinArtefacto(t *testing.T) {
	store := NewInMemoryRequiredTestEvidenceStoreV0()
	request := requiredTestExecutionRequestForTestV0()
	request.TestCommands = []string{"go test -count=1 ./modulos/orquesta-orchestration-core -run TestSinArtefacto"}

	result, err := (RequiredTestRunnerV0{
		Executor: &fakeRequiredTestCommandExecutorV0{
			results: map[string]RequiredTestCommandExecutionResultV0{
				request.TestCommands[0]: {
					Status: RequiredTestEvidenceStatusPassedV0,
				},
			},
		},
		EvidenceWriter: store,
	}).RunRequiredTestsV0(context.Background(), request)
	if err != nil {
		t.Fatalf("RunRequiredTestsV0: %v", err)
	}
	if len(result.Issues) != 1 ||
		result.Issues[0].Field != "required_test.evidence_refs" ||
		len(result.EvidenceRefs) != 0 {
		t.Fatalf("result=%+v", result)
	}
}

type fakeRequiredTestCommandExecutorV0 struct {
	results  map[string]RequiredTestCommandExecutionResultV0
	commands []string
}

func (executor *fakeRequiredTestCommandExecutorV0) RunRequiredTestCommandV0(
	ctx context.Context,
	request RequiredTestCommandExecutionRequestV0,
) (RequiredTestCommandExecutionResultV0, error) {
	if err := ctx.Err(); err != nil {
		return RequiredTestCommandExecutionResultV0{}, err
	}
	executor.commands = append(executor.commands, request.TestCommand)
	return executor.results[request.TestCommand], nil
}

func requiredTestExecutionRequestForTestV0() RequiredTestExecutionRequestV0 {
	return RequiredTestExecutionRequestV0{
		RunRef:  "run-required-tests-001",
		TaskRef: "task-required-tests-001",
		TestCommands: []string{
			"go test -count=1 ./modulos/orquesta-orchestration-core -run TestRequiredTestEvidence",
			"go test -count=1 ./modulos/orquesta-app-director-service -run TestUpdateOperationalDirectorPlanState",
		},
		DeliveryRef:       "delivery-ref-required-tests-001",
		ReviewRequestID:   "review-request-ref-required-tests-001",
		ReviewResultRef:   "review-result-ref-required-tests-001",
		AcceptedReviewRef: "accepted-review-ref-required-tests-001",
		OccurredAt:        "2026-05-21T11:00:00Z",
		CorrelationID:     "correlation-required-tests-001",
		EvidenceRefs:      []string{"review-evidence-ref-required-tests-001"},
	}
}

func requiredTestRunnerStringInSetV0(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
