package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	orquestaagentprocessregistrymemory "orquesta/modulos/orquesta-agent-process-registry-memory"
	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestarunmemory "orquesta/modulos/orquesta-run-memory"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
	orquestaruntimerequiredtest "orquesta/modulos/orquesta-runtime-required-test"
	orquestaweb "orquesta/modulos/orquesta-web"
)

func TestCodexStackRequiredTestRunnerFailedReplanLanzaFollowupFakeRuntimeV0(t *testing.T) {
	ctx := context.Background()
	cfg := codexStackRequiredTestLocalConfigV0(t)
	writeCodexStackRequiredTestFailingGoModuleV0(t, cfg.ProjectWorkDir)
	goCommand := codexStackRequiredTestGoCommandV0(t)
	outputDir := filepath.Join(t.TempDir(), "required-test-output")

	runtime := &codexStackRequiredTestPendingRuntimeV0{}
	evidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0()
	stack := codexStackRealRequiredTestRunnerStackV0(t, cfg, runtime, evidenceStore, goCommand, outputDir)

	refs := (codexStackRequiredTestRefsV0{
		RunRef:  "run-rt-runner-failed-replan-001",
		PlanRef: "plan-rt-runner-failed-replan-001",
		TaskRef: "task-rt-runner-failed-replan-001",
	}).withDefaultsV0()
	if err := stack.Stores.RunStore.SaveRunV0(ctx, refs.runV0()); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	if err := stack.Stores.EventSink.AppendRunEventsV0(ctx, refs.RunRef, refs.eventsV0(t)); err != nil {
		t.Fatalf("AppendRunEventsV0: %v", err)
	}
	taskWriter, ok := stack.Stores.TaskStore.(orquestacionnucleoapp.WorkflowTaskWriterPortV0)
	if !ok {
		t.Fatalf("TaskStore no escribe WorkflowTaskV0: %T", stack.Stores.TaskStore)
	}
	if err := taskWriter.SaveWorkflowTaskV0(ctx, refs.workflowTaskV0()); err != nil {
		t.Fatalf("SaveWorkflowTaskV0: %v", err)
	}
	if err := stack.Stores.OperationalPlanStateWriter.SaveOperationalDirectorPlanStateV0(ctx, refs.planStateV0()); err != nil {
		t.Fatalf("SaveOperationalDirectorPlanStateV0: %v", err)
	}

	_, err := orquestaappdirectorservice.ContinueAppDirectorV0(ctx, codexStackRequiredTestContinueRequestV0(
		refs,
		"corr-rt-runner-failed-replan-tests",
		"2026-05-22T18:00:00Z",
		1,
	), stack.Ports)
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0 required tests: %v %s", err, codexStackRequiredTestErrorDetailsV0(err))
	}
	run := mustLoadCodexStackRunForTestV0(t, stack, refs.RunRef)
	if len(run.QualityGates) != 1 || len(run.ReplanDecisions) != 1 {
		t.Fatalf("run sin quality gate/replan tras fallo: gates=%v replans=%v", run.QualityGates, run.ReplanDecisions)
	}
	replan, ok := codexStackRealSmokeParseReplanProjectionV0(run.ReplanDecisions[0])
	if !ok ||
		replan.TaskRef != refs.TaskRef ||
		replan.Action != string(orquestacoreworkflow.ReplanDecisionActionRetryTaskV0) {
		t.Fatalf("replan invalido: raw=%q parsed=%+v", run.ReplanDecisions[0], replan)
	}
	followupAgentRef := codexStackRequiredTestFirstFollowupAgentRefV0(t, replan)
	state, err := stack.Stores.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, refs.RunRef, refs.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	testsStep := codexStackRequiredTestPlanStepV0(t, state, "step-run-required-tests")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		testsStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		testsStep.Reason != "required-tests-failed" ||
		len(testsStep.RequiredTestEvidenceRefs) != 1 {
		t.Fatalf("state no bloqueo por tests fallidos: state=%+v tests=%+v", state, testsStep)
	}
	evidence, err := evidenceStore.LoadRequiredTestEvidenceV0(ctx, refs.RunRef, testsStep.RequiredTestEvidenceRefs)
	if err != nil {
		t.Fatalf("LoadRequiredTestEvidenceV0 failed: %v", err)
	}
	if len(evidence) != 1 ||
		evidence[0].Status != orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0 ||
		evidence[0].TestCommand != "go test ./..." {
		t.Fatalf("evidence failed invalida: %+v", evidence)
	}

	launched, err := orquestaappdirectorservice.ContinueAppDirectorV0(ctx, codexStackRequiredTestContinueRequestV0(
		refs,
		"corr-rt-runner-failed-replan-launch",
		"2026-05-22T18:01:00Z",
		3,
	), stack.Ports)
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0 launch followup: %v %s", err, codexStackRequiredTestErrorDetailsV0(err))
	}
	if !codexStackStringInSetForTestV0(launched.StartedAgents, followupAgentRef) || runtime.launchCountV0() != 1 {
		run = mustLoadCodexStackRunForTestV0(t, stack, refs.RunRef)
		t.Fatalf("followup no lanzado: started=%v followup=%s launches=%d run=%+v", launched.StartedAgents, followupAgentRef, runtime.launchCountV0(), run)
	}
	run = mustLoadCodexStackRunForTestV0(t, stack, refs.RunRef)
	if !codexStackStringInSetForTestV0(run.Agents, followupAgentRef) ||
		!codexStackStringInSetForTestV0(run.StartedAgents, followupAgentRef) ||
		codexStackStringInSetForTestV0(run.DeliveredAgents, followupAgentRef) {
		t.Fatalf("run followup inesperado: followup=%s agents=%v started=%v delivered=%v", followupAgentRef, run.Agents, run.StartedAgents, run.DeliveredAgents)
	}

	waitPorts := stack.Ports
	waitPorts.ExternalWaiter = nil
	waiting, err := orquestaappdirectorservice.ContinueAppDirectorV0(ctx, codexStackRequiredTestContinueRequestV0(
		refs,
		"corr-rt-runner-failed-replan-wait",
		"2026-05-22T18:02:00Z",
		1,
	), waitPorts)
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0 reopen wait: %v %s", err, codexStackRequiredTestErrorDetailsV0(err))
	}
	if waiting.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 ||
		!codexStackStringInSetForTestV0(waiting.StartedAgents, followupAgentRef) {
		t.Fatalf("continue no quedo esperando followup: loop=%s started=%v followup=%s", waiting.LoopStatus, waiting.StartedAgents, followupAgentRef)
	}
	state, err = stack.Stores.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, refs.RunRef, refs.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 final: %v", err)
	}
	waitStep := codexStackRequiredTestPlanStepV0(t, state, "step-wait-subagents")
	testsStep = codexStackRequiredTestPlanStepV0(t, state, "step-run-required-tests")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		state.ActiveStepID != "step-wait-subagents" ||
		state.ReplanAttempts != 1 ||
		!codexStackStringInSetForTestV0(state.PendingAgentRefs, followupAgentRef) ||
		waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!codexStackStringInSetForTestV0(waitStep.PendingAgentRefs, followupAgentRef) ||
		len(waitStep.WaitRefs) != 1 ||
		!codexStackStringInSetForTestV0(testsStep.ReplanDecisionRefs, strings.Split(run.ReplanDecisions[0], "#")[0]) {
		t.Fatalf("state no quedo listo para nueva espera: state=%+v wait=%+v tests=%+v", state, waitStep, testsStep)
	}

	writeCodexStackRequiredTestTinyGoModuleV0(t, cfg.ProjectWorkDir)
	followupDescriptor := codexStackRequiredTestDescriptorForAgentV0(t, stack, followupAgentRef)
	if err := writeCodexStackAckForDescriptorV0(t, followupDescriptor); err != nil {
		t.Fatalf("write followup ack: %v", err)
	}
	followupDeliveryRef := strings.TrimSpace(followupDescriptor.Spec.AgentPacket.DeliveryRefs.AckRef)
	for cycle := 1; cycle <= 6; cycle++ {
		if _, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
			RunRef:                     refs.RunRef,
			OperationalDirectorPlanRef: refs.PlanRef,
			WaitAgentRefs:              []string{followupAgentRef},
			CorrelationID:              fmt.Sprintf("corr-rt-runner-failed-replan-delivery-%03d", cycle),
			MaxBursts:                  4,
			MaxStepsPerBurst:           8,
			MaxDispatchesPerWait:       4,
			MaxCommands:                12,
			MaxOutboxPerCycle:          8,
			MaxExternalWaits:           1,
		}); err != nil {
			t.Fatalf("DrainRunV0 followup delivery ciclo=%d: %v", cycle, err)
		}
		run = mustLoadCodexStackRunForTestV0(t, stack, refs.RunRef)
		if codexStackStringInSetForTestV0(run.DeliveredAgents, followupAgentRef) &&
			codexStackRealSmokeContainsProjectionPartV0(run.Deliveries, followupDeliveryRef) {
			break
		}
	}
	run = mustLoadCodexStackRunForTestV0(t, stack, refs.RunRef)
	if !codexStackStringInSetForTestV0(run.DeliveredAgents, followupAgentRef) ||
		!codexStackRealSmokeContainsProjectionPartV0(run.Deliveries, followupDeliveryRef) {
		t.Fatalf("followup no entrego: agent=%s delivery=%s run=%+v", followupAgentRef, followupDeliveryRef, run)
	}

	for cycle := 1; cycle <= 16; cycle++ {
		if _, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
			RunRef:                     refs.RunRef,
			OperationalDirectorPlanRef: refs.PlanRef,
			WaitAgentRefs:              []string{followupAgentRef},
			CorrelationID:              fmt.Sprintf("corr-rt-runner-failed-replan-close-%03d", cycle),
			MaxBursts:                  12,
			MaxStepsPerBurst:           16,
			MaxDispatchesPerWait:       8,
			MaxCommands:                64,
			MaxOutboxPerCycle:          32,
			MaxDecisionCycles:          8,
			MaxExternalWaits:           2,
		}); err != nil {
			t.Fatalf("DrainRunV0 followup close ciclo=%d: %v", cycle, err)
		}
		state, err = stack.Stores.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, refs.RunRef, refs.PlanRef)
		if err != nil {
			t.Fatalf("LoadOperationalDirectorPlanStateV0 close ciclo=%d: %v", cycle, err)
		}
		run = mustLoadCodexStackRunForTestV0(t, stack, refs.RunRef)
		if state.Status == orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 &&
			run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
			break
		}
	}
	state, err = stack.Stores.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, refs.RunRef, refs.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 closed: %v", err)
	}
	run = mustLoadCodexStackRunForTestV0(t, stack, refs.RunRef)
	replanStep := codexStackRequiredTestPlanStepV0(t, state, "step-replan-or-close")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 ||
		run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		len(run.QualityGates) != 2 ||
		len(run.ReplanDecisions) != 1 ||
		len(replanStep.RequiredTestEvidenceRefs) != 1 {
		t.Fatalf("replan followup no cerro limpiamente: state=%+v run=%+v replan=%+v", state, run, replanStep)
	}
	if blockers := orquestacoreworkflow.PendingBlockingQualityGateRefsForSubjectV0(run, refs.TaskRef); len(blockers) != 0 {
		t.Fatalf("quality gate requerido no resuelto: blockers=%v gates=%v", blockers, run.QualityGates)
	}
	passedEvidence, err := evidenceStore.LoadRequiredTestEvidenceV0(ctx, refs.RunRef, replanStep.RequiredTestEvidenceRefs)
	if err != nil {
		t.Fatalf("LoadRequiredTestEvidenceV0 passed: %v", err)
	}
	if len(passedEvidence) != 1 ||
		passedEvidence[0].Status != orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0 ||
		passedEvidence[0].DeliveryRef != followupDeliveryRef ||
		passedEvidence[0].TaskRef != refs.TaskRef {
		t.Fatalf("evidence passed invalida: %+v delivery=%s", passedEvidence, followupDeliveryRef)
	}
}

func TestCodexStackRealRequiredTestRunnerOptInV0(t *testing.T) {
	if strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_REAL_REQUIRED_TEST_RUNNER_SMOKE")) != "1" {
		t.Skip("set ORQUESTA_CODEX_REAL_REQUIRED_TEST_RUNNER_SMOKE=1")
	}
	if strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_REAL_REQUIRED_TEST_RUNNER_CONFIRM")) != "1" ||
		strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_REAL_REQUIRED_TEST_RUNNER_NO_CODEX_EXECUTION_CONFIRMED")) != "1" {
		t.Fatal("doble confirmacion requerida para smoke Codex real sin ejecucion")
	}
	if strings.TrimSpace(os.Getenv("ORQUESTA_OPES_BASE_URL")) != "" ||
		strings.TrimSpace(os.Getenv("OPES_BASE_URL")) != "" {
		t.Fatal("este smoke no debe cablear OPES")
	}

	ctx := context.Background()
	cfg := codexStackRealSmokeConfigForTestV0(t)
	writeCodexStackRequiredTestTinyGoModuleV0(t, cfg.ProjectWorkDir)
	goCommand := codexStackRequiredTestGoCommandV0(t)
	outputDir := codexStackRealSmokeEnsureDirV0(t, "ORQUESTA_REQUIRED_TEST_OUTPUT_DIR", codexStackRequiredTestOutputDirV0(t))

	processRuntime := orquestaruntime.NewProcessRuntimeConnectorV0()
	evidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0()
	stack := codexStackRealRequiredTestRunnerStackV0(t, cfg, processRuntime, evidenceStore, goCommand, outputDir)
	if stack.Ports.RequiredTestRunner == nil {
		t.Fatal("RequiredTestRunner no cableado")
	}

	refs := (codexStackRequiredTestRefsV0{}).withDefaultsV0()
	run := refs.runV0()
	if err := stack.Stores.RunStore.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	if err := stack.Stores.EventSink.AppendRunEventsV0(ctx, refs.RunRef, refs.eventsV0(t)); err != nil {
		t.Fatalf("AppendRunEventsV0: %v", err)
	}
	taskWriter, ok := stack.Stores.TaskStore.(orquestacionnucleoapp.WorkflowTaskWriterPortV0)
	if !ok {
		t.Fatalf("TaskStore no escribe WorkflowTaskV0: %T", stack.Stores.TaskStore)
	}
	if err := taskWriter.SaveWorkflowTaskV0(ctx, refs.workflowTaskV0()); err != nil {
		t.Fatalf("SaveWorkflowTaskV0: %v", err)
	}
	if err := stack.Stores.OperationalPlanStateWriter.SaveOperationalDirectorPlanStateV0(ctx, refs.planStateV0()); err != nil {
		t.Fatalf("SaveOperationalDirectorPlanStateV0: %v", err)
	}

	result, err := orquestaappdirectorservice.ContinueAppDirectorV0(ctx, orquestaappdirectorservice.ContinueAppDirectorRequestV0{
		RunRef:                     refs.RunRef,
		OperationalDirectorPlanRef: refs.PlanRef,
		OccurredAt:                 "2026-05-22T17:00:00Z",
		CorrelationID:              "corr-rt-runner-continue",
		MaxBursts:                  1,
		MaxStepsPerBurst:           1,
		MaxDispatchesPerWait:       1,
		MaxCommands:                8,
		MaxOutboxPerCycle:          8,
		MaxExternalWaits:           1,
	}, stack.Ports)
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0: %v", err)
	}
	if result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("run no cerrado: status=%s closed=%v validations=%v closures=%v", result.Run.Status, result.Run.ClosedTasks, result.Run.Validations, result.Run.Closures)
	}

	state, err := stack.Stores.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, refs.RunRef, refs.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	replanStep := codexStackRequiredTestPlanStepV0(t, state, "step-replan-or-close")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 ||
		len(replanStep.RequiredTestEvidenceRefs) != 1 {
		t.Fatalf("plan state sin cierre/evidencia: state=%+v replan=%+v", state, replanStep)
	}
	evidence, err := evidenceStore.LoadRequiredTestEvidenceV0(ctx, refs.RunRef, replanStep.RequiredTestEvidenceRefs)
	if err != nil {
		t.Fatalf("LoadRequiredTestEvidenceV0: %v", err)
	}
	if len(evidence) != 1 ||
		evidence[0].Status != orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0 ||
		evidence[0].TestCommand != "go test ./..." ||
		evidence[0].TaskRef != refs.TaskRef ||
		evidence[0].DeliveryRef != refs.DeliveryRef ||
		evidence[0].ReviewRequestID != refs.ReviewRequestRef ||
		evidence[0].ReviewResultRef != refs.ReviewResultRef ||
		evidence[0].AcceptedReviewRef != refs.AcceptedReviewRef {
		t.Fatalf("evidence invalida: %+v", evidence)
	}
	outputRef := codexStackRequiredTestOutputRefV0(evidence[0].EvidenceRefs)
	if outputRef == "" {
		t.Fatalf("evidence sin output: %+v", evidence[0])
	}
	content, err := os.ReadFile(filepath.Join(outputDir, strings.TrimPrefix(outputRef, "required-test-output-v0/")))
	if err != nil {
		t.Fatalf("leer output required test: %v", err)
	}
	if !strings.Contains(string(content), "status=passed") ||
		!strings.Contains(string(content), "test_command=go test ./...") ||
		strings.Contains(string(content), cfg.ProjectWorkDir) ||
		strings.Contains(string(content), outputDir) {
		t.Fatalf("output required test invalido: %q", string(content))
	}
	if entries := codexStackRequiredTestRuntimeEntriesV0(t, cfg.RuntimeWorkDir); len(entries) != 0 {
		t.Fatalf("Runtime Codex no debia ejecutarse; runtime_entries=%+v", entries)
	}
}

func TestCodexStackRealRequiredTestRunnerEndToEndOptInV0(t *testing.T) {
	if strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_REAL_REQUIRED_TEST_RUNNER_E2E_SMOKE")) != "1" {
		t.Skip("set ORQUESTA_CODEX_REAL_REQUIRED_TEST_RUNNER_E2E_SMOKE=1")
	}
	if strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_REAL_REQUIRED_TEST_RUNNER_CONFIRM")) != "1" ||
		strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_REAL_REQUIRED_TEST_RUNNER_CODEX_EXECUTION_CONFIRMED")) != "1" {
		t.Fatal("doble confirmacion requerida para ejecutar Codex real")
	}
	if strings.TrimSpace(os.Getenv("ORQUESTA_OPES_BASE_URL")) != "" ||
		strings.TrimSpace(os.Getenv("OPES_BASE_URL")) != "" {
		t.Fatal("este smoke no debe cablear OPES")
	}

	cfg := codexStackRealSmokeConfigForTestV0(t)
	if cfg.Timeout < 420*time.Second {
		cfg.Timeout = 420 * time.Second
	}
	if strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_REASONING_EFFORT")) == "" {
		cfg.ReasoningEffort = "medium"
	}
	if strings.TrimSpace(cfg.ReasoningEffort) == "xhigh" {
		t.Fatal("este smoke acotado no usa xhigh")
	}
	cfg.MaxBatchReady = 1
	cfg.MaxConcurrency = 1
	writeCodexStackRequiredTestTinyGoModuleV0(t, cfg.ProjectWorkDir)
	codexStackRealSmokeWriteProjectContextV0(t, cfg.ProjectWorkDir)
	goCommand := codexStackRequiredTestGoCommandV0(t)
	outputDir := codexStackRealSmokeEnsureDirV0(t, "ORQUESTA_REQUIRED_TEST_OUTPUT_DIR", codexStackRequiredTestOutputDirV0(t))

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()
	processRuntime := orquestaruntime.NewProcessRuntimeConnectorV0()
	evidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0()
	stack := codexStackRealRequiredTestRunnerStackV0(t, cfg, processRuntime, evidenceStore, goCommand, outputDir)
	defer codexStackRealSmokeStopAllProcessesV0(
		t,
		processRuntime,
		stack.Stores.ProcessRegistry.(*orquestaagentprocessregistrymemory.InMemoryAgentProcessRegistryV0),
		stack.Stores.ReceiptStore.(*orquestaruntimecodexdelivery.InMemoryCodexReceiptDescriptorStoreV0),
	)

	refs := (codexStackRequiredTestRefsV0{
		RunRef:  "run-rt-runner-e2e-001",
		PlanRef: "plan-rt-runner-e2e-001",
		TaskRef: "task-rt-runner-e2e-001",
	}).withDefaultsV0()
	if err := stack.Stores.RunStore.SaveRunV0(ctx, refs.pendingRunV0()); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	taskWriter, ok := stack.Stores.TaskStore.(orquestacionnucleoapp.WorkflowTaskWriterPortV0)
	if !ok {
		t.Fatalf("TaskStore no escribe WorkflowTaskV0: %T", stack.Stores.TaskStore)
	}
	if err := taskWriter.SaveWorkflowTaskV0(ctx, refs.workflowTaskV0()); err != nil {
		t.Fatalf("SaveWorkflowTaskV0: %v", err)
	}

	started, err := orquestaappdirectorservice.ContinueAppDirectorV0(ctx, orquestaappdirectorservice.ContinueAppDirectorRequestV0{
		RunRef:               refs.RunRef,
		OccurredAt:           "2026-05-22T17:20:00Z",
		CorrelationID:        "corr-rt-runner-e2e-start",
		WaitAgentRefs:        []string{refs.AgentRef},
		MaxBursts:            2,
		MaxStepsPerBurst:     4,
		MaxDispatchesPerWait: 2,
		MaxCommands:          8,
		MaxOutboxPerCycle:    4,
		MaxExternalWaits:     1,
	}, stack.Ports)
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0 start: %v %s\n%s", err, codexStackRequiredTestErrorDetailsV0(err), codexStackRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir))
	}
	if !codexStackStringInSetForTestV0(started.StartedAgents, refs.AgentRef) {
		t.Fatalf("agente Codex no arrancado: started=%v wait=%s", started.StartedAgents, refs.AgentRef)
	}

	delivered := codexStackRequiredTestDrainUntilDeliveryV0(t, ctx, stack, refs.RunRef, refs.AgentRef, cfg.RuntimeWorkDir, cfg.Timeout)
	deliveryRef := strings.TrimSpace(delivered.Spec.AgentPacket.DeliveryRefs.AckRef)
	if deliveryRef == "" {
		t.Fatalf("delivery ref vacia en descriptor=%+v", delivered)
	}
	if err := stack.Stores.OperationalPlanStateWriter.SaveOperationalDirectorPlanStateV0(ctx, refs.planStateForDeliveryReviewV0(deliveryRef)); err != nil {
		t.Fatalf("SaveOperationalDirectorPlanStateV0 review: %v", err)
	}

	openCodexStackPhaseForTestV0(
		t,
		stack,
		refs.RunRef,
		orquestacoreworkflow.OrchestrationPhaseRevisionV0,
		"Smoke real required tests: revisar entrega Codex.",
	)
	causal := codexStackRequiredTestDrainUntilAcceptedReviewV0(t, ctx, stack, refs.RunRef, refs.PlanRef, deliveryRef, cfg.RuntimeWorkDir, cfg.Timeout)

	result, err := orquestaappdirectorservice.ContinueAppDirectorV0(ctx, orquestaappdirectorservice.ContinueAppDirectorRequestV0{
		RunRef:                     refs.RunRef,
		OperationalDirectorPlanRef: refs.PlanRef,
		OccurredAt:                 "2026-05-22T17:30:00Z",
		CorrelationID:              "corr-rt-runner-e2e-close",
		MaxBursts:                  1,
		MaxStepsPerBurst:           1,
		MaxDispatchesPerWait:       1,
		MaxCommands:                8,
		MaxOutboxPerCycle:          8,
		MaxExternalWaits:           1,
	}, stack.Ports)
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0 close: %v\n%s", err, codexStackRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir))
	}
	if result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("run no cerrado: status=%s closed=%v validations=%v closures=%v", result.Run.Status, result.Run.ClosedTasks, result.Run.Validations, result.Run.Closures)
	}
	state, err := stack.Stores.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, refs.RunRef, refs.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	replanStep := codexStackRequiredTestPlanStepV0(t, state, "step-replan-or-close")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 ||
		len(replanStep.RequiredTestEvidenceRefs) != 1 {
		t.Fatalf("plan state sin cierre/evidencia: state=%+v replan=%+v", state, replanStep)
	}
	evidence, err := evidenceStore.LoadRequiredTestEvidenceV0(ctx, refs.RunRef, replanStep.RequiredTestEvidenceRefs)
	if err != nil {
		t.Fatalf("LoadRequiredTestEvidenceV0: %v", err)
	}
	if len(evidence) != 1 ||
		evidence[0].Status != orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0 ||
		evidence[0].TestCommand != "go test ./..." ||
		evidence[0].DeliveryRef != deliveryRef ||
		evidence[0].ReviewRequestID != causal.ReviewRequestRef ||
		evidence[0].ReviewResultRef != causal.ReviewResultRef ||
		evidence[0].AcceptedReviewRef != causal.AcceptedReviewRef {
		t.Fatalf("evidence e2e invalida: %+v causal=%+v", evidence, causal)
	}
}

func codexStackRequiredTestErrorDetailsV0(err error) string {
	if err == nil {
		return ""
	}
	var commandErr orquestacoreworkflow.OrchestrationCommandErrorV0
	if errors.As(err, &commandErr) {
		return "command_error_field=" + commandErr.Field
	}
	var eventErr orquestacoreworkflow.OrchestrationEventErrorV0
	if errors.As(err, &eventErr) {
		return "event_error_field=" + eventErr.Field
	}
	var workflowErr orquestacoreworkflow.WorkflowTaskErrorV0
	if errors.As(err, &workflowErr) {
		return "workflow_task_error_field=" + workflowErr.Field
	}
	var orchestrationErr orquestacionnucleoapp.ErrorV0
	if errors.As(err, &orchestrationErr) {
		return "orchestration_error_field=" + orchestrationErr.Field + " message=" + orchestrationErr.Message
	}
	return ""
}

type codexStackRequiredTestRefsV0 struct {
	RunRef            string
	PlanRef           string
	TaskRef           string
	AgentRef          string
	DeliveryRef       string
	ReviewRequestRef  string
	ReviewResultRef   string
	AcceptedReviewRef string
}

func (refs codexStackRequiredTestRefsV0) withDefaultsV0() codexStackRequiredTestRefsV0 {
	if strings.TrimSpace(refs.TaskRef) == "" {
		refs.TaskRef = "task-rt-runner-001"
	}
	taskRef := refs.TaskRef
	if strings.TrimSpace(refs.RunRef) == "" {
		refs.RunRef = "run-rt-runner-001"
	}
	if strings.TrimSpace(refs.PlanRef) == "" {
		refs.PlanRef = "plan-rt-runner-001"
	}
	if strings.TrimSpace(refs.AgentRef) == "" {
		refs.AgentRef = orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	}
	if strings.TrimSpace(refs.DeliveryRef) == "" {
		refs.DeliveryRef = "delivery-rt-runner-001"
	}
	if strings.TrimSpace(refs.ReviewRequestRef) == "" {
		refs.ReviewRequestRef = "review-request-rt-runner-001"
	}
	if strings.TrimSpace(refs.ReviewResultRef) == "" {
		refs.ReviewResultRef = "review-result-rt-runner-001"
	}
	if strings.TrimSpace(refs.AcceptedReviewRef) == "" {
		refs.AcceptedReviewRef = "accepted-review-rt-runner-001"
	}
	return codexStackRequiredTestRefsV0{
		RunRef:            refs.RunRef,
		PlanRef:           refs.PlanRef,
		TaskRef:           refs.TaskRef,
		AgentRef:          refs.AgentRef,
		DeliveryRef:       refs.DeliveryRef,
		ReviewRequestRef:  refs.ReviewRequestRef,
		ReviewResultRef:   refs.ReviewResultRef,
		AcceptedReviewRef: refs.AcceptedReviewRef,
	}
}

func (refs codexStackRequiredTestRefsV0) pendingRunV0() orquestacoreworkflow.OrchestrationRunV0 {
	refs = refs.withDefaultsV0()
	run := refs.runV0()
	run.CurrentPhase = orquestacoreworkflow.OrchestrationPhaseProgramacionV0
	for index := range run.Phases {
		if run.Phases[index].ID == orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
			run.Phases[index].Status = orquestacoreworkflow.OrchestrationPhaseStatusActiveV0
			continue
		}
		run.Phases[index].Status = orquestacoreworkflow.OrchestrationPhaseStatusPendingV0
	}
	run.Agents = nil
	run.StartedAgents = nil
	run.DeliveredAgents = nil
	run.Deliveries = nil
	run.DeliveredTasks = nil
	run.Reviews = nil
	run.ReviewResults = nil
	run.AcceptedReviews = nil
	run.LastEventID = ""
	run.LastSequence = 0
	return run
}

func (refs codexStackRequiredTestRefsV0) runV0() orquestacoreworkflow.OrchestrationRunV0 {
	refs = refs.withDefaultsV0()
	return orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion:   orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:           refs.RunRef,
		ProjectRef:      "project-rt-runner-001",
		AppSpecRef:      "app-spec-rt-runner-001",
		Status:          orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:    orquestacoreworkflow.OrchestrationPhaseRevisionV0,
		Phases:          codexStackRequiredTestPhasesV0(),
		Tasks:           []string{refs.TaskRef},
		Agents:          []string{refs.AgentRef},
		StartedAgents:   []string{refs.AgentRef},
		DeliveredAgents: []string{refs.AgentRef},
		Deliveries:      []string{refs.DeliveryRef},
		DeliveredTasks:  []string{refs.TaskRef},
		Reviews:         []string{refs.ReviewRequestRef},
		ReviewResults: []string{
			refs.ReviewResultRef + "#review_result:accepted#review_request:" + refs.ReviewRequestRef + "#delivery:" + refs.DeliveryRef,
		},
		AcceptedReviews: []string{refs.AcceptedReviewRef},
		LastEventID:     "evt-rt-runner-review-accepted",
		LastSequence:    4,
	}
}

func (refs codexStackRequiredTestRefsV0) eventsV0(t *testing.T) []orquestacoreworkflow.OrchestrationEventV0 {
	t.Helper()
	refs = refs.withDefaultsV0()
	delivery, err := orquestacoreworkflow.NewDeliveryRegisteredEventV0(
		codexStackRequiredTestEventMetaV0(refs.RunRef, 1, "delivery"),
		orquestacoreworkflow.DeliveryRegisteredPayloadV0{
			DeliveryRef:  refs.DeliveryRef,
			PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskID:       refs.TaskRef,
			AgentRef:     refs.AgentRef,
			Summary:      "Entrega de programacion lista para revision.",
			EvidenceRefs: []string{"artifact-ref-rt-delivery"},
		},
	)
	if err != nil {
		t.Fatalf("NewDeliveryRegisteredEventV0: %v", err)
	}
	requested, err := orquestacoreworkflow.NewReviewRequestedEventV0(
		codexStackRequiredTestEventMetaV0(refs.RunRef, 2, "review-request"),
		orquestacoreworkflow.ReviewRequestedPayloadV0{
			ReviewRequestID: refs.ReviewRequestRef,
			PhaseID:         string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			DeliveryRef:     refs.DeliveryRef,
			Summary:         "Revision requerida para entrega de programacion.",
			EvidenceRefs:    []string{"artifact-ref-rt-review-request"},
		},
	)
	if err != nil {
		t.Fatalf("NewReviewRequestedEventV0: %v", err)
	}
	recorded, err := orquestacoreworkflow.NewReviewResultRecordedEventV0(
		codexStackRequiredTestEventMetaV0(refs.RunRef, 3, "review-result"),
		orquestacoreworkflow.ReviewResultV0{
			ReviewResultRef: refs.ReviewResultRef,
			ReviewRequestID: refs.ReviewRequestRef,
			DeliveryRef:     refs.DeliveryRef,
			Status:          orquestacoreworkflow.ReviewResultStatusAcceptedV0,
			Summary:         "Revision aceptada; ejecutar tests requeridos.",
			EvidenceRefs:    []string{"artifact-ref-rt-review-result"},
		},
	)
	if err != nil {
		t.Fatalf("NewReviewResultRecordedEventV0: %v", err)
	}
	accepted, err := orquestacoreworkflow.NewReviewAcceptedEventV0(
		codexStackRequiredTestEventMetaV0(refs.RunRef, 4, "review-accepted"),
		orquestacoreworkflow.ReviewAcceptedPayloadV0{
			AcceptedReviewRef: refs.AcceptedReviewRef,
			PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			ReviewRequestID:   refs.ReviewRequestRef,
			DeliveryRef:       refs.DeliveryRef,
			Summary:           "Revision aceptada con cadena causal completa.",
			EvidenceRefs:      []string{"artifact-ref-rt-review-accepted"},
		},
	)
	if err != nil {
		t.Fatalf("NewReviewAcceptedEventV0: %v", err)
	}
	return []orquestacoreworkflow.OrchestrationEventV0{delivery, requested, recorded, accepted}
}

func (refs codexStackRequiredTestRefsV0) workflowTaskV0() orquestacoreworkflow.WorkflowTaskV0 {
	refs = refs.withDefaultsV0()
	return orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:      orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:             refs.TaskRef,
		RunID:              refs.RunRef,
		PhaseID:            orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		WorkProfileKind:    orquestacoreworkflow.WorkProfileImplementationV0,
		Title:              "Implementar modulo Go con tests",
		Summary:            "Microtarea de programacion con cierre causal.",
		WriteSet:           []string{"go.mod", "calc.go", "calc_test.go"},
		AcceptanceCriteria: []string{"operational_director.required_tests passed", "codigo Go probado"},
		RequiredTests:      []string{"go test ./..."},
		WaveRef:            "wave-rt-runner-001",
		CohortRef:          "cohort-rt-runner-001",
	}
}

func (refs codexStackRequiredTestRefsV0) planStateV0() orquestacionnucleoapp.OperationalDirectorPlanStateV0 {
	refs = refs.withDefaultsV0()
	return orquestacionnucleoapp.OperationalDirectorPlanStateV0{
		SchemaVersion:    orquestacionnucleoapp.OperationalDirectorPlanStateSchemaVersionV0,
		StateRef:         "state-rt-runner-001",
		PlanRef:          refs.PlanRef,
		RequestRef:       "request-rt-runner-001",
		RunRef:           refs.RunRef,
		ProjectRef:       "project-rt-runner-001",
		Mode:             orquestadirectoroperativo.OperationalDirectorModeProgrammingV0,
		Status:           orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0,
		ActiveStepID:     "step-run-required-tests",
		ActiveWaveRef:    "wave-rt-runner-001",
		ActiveCohortRef:  "cohort-rt-runner-001",
		RequiredTestRefs: []string{"go test ./..."},
		ObservedAt:       "2026-05-22T16:59:00Z",
		Steps: []orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{
			{
				StepID:    "step-launch-subagents",
				Kind:      orquestadirectoroperativo.OperationalDirectorStepLaunchSubagentsV0,
				Status:    orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
				WaveRef:   "wave-rt-runner-001",
				CohortRef: "cohort-rt-runner-001",
				TaskRefs:  []string{refs.TaskRef},
				AgentRefs: []string{refs.AgentRef},
			},
			{
				StepID:           "step-wait-subagents",
				Kind:             orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0,
				Status:           orquestadirectoroperativo.OperationalDirectorStepPendingV0,
				WaveRef:          "wave-rt-runner-001",
				CohortRef:        "cohort-rt-runner-001",
				TaskRefs:         []string{refs.TaskRef},
				AgentRefs:        []string{refs.AgentRef},
				PendingAgentRefs: []string{refs.AgentRef},
			},
			{
				StepID:             "step-review-deliveries",
				Kind:               orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0,
				Status:             orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
				WaveRef:            "wave-rt-runner-001",
				CohortRef:          "cohort-rt-runner-001",
				TaskRefs:           []string{refs.TaskRef},
				AgentRefs:          []string{refs.AgentRef},
				DeliveryRefs:       []string{refs.DeliveryRef},
				ReviewResultRefs:   []string{refs.ReviewResultRef},
				AcceptedReviewRefs: []string{refs.AcceptedReviewRef},
			},
			{
				StepID:           "step-run-required-tests",
				Kind:             orquestadirectoroperativo.OperationalDirectorStepRunRequiredTestsV0,
				Status:           orquestadirectoroperativo.OperationalDirectorStepRunningV0,
				WaveRef:          "wave-rt-runner-001",
				CohortRef:        "cohort-rt-runner-001",
				TaskRefs:         []string{refs.TaskRef},
				AgentRefs:        []string{refs.AgentRef},
				DeliveryRefs:     []string{refs.DeliveryRef},
				ReviewResultRefs: []string{refs.ReviewResultRef},
			},
			{
				StepID:    "step-replan-or-close",
				Kind:      orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0,
				Status:    orquestadirectoroperativo.OperationalDirectorStepPendingV0,
				WaveRef:   "wave-rt-runner-001",
				CohortRef: "cohort-rt-runner-001",
			},
		},
	}
}

type codexStackRequiredTestCausalReviewRefsV0 struct {
	DeliveryRef       string
	ReviewRequestRef  string
	ReviewResultRef   string
	AcceptedReviewRef string
}

func (refs codexStackRequiredTestRefsV0) planStateForCausalReviewV0(
	causal codexStackRequiredTestCausalReviewRefsV0,
) orquestacionnucleoapp.OperationalDirectorPlanStateV0 {
	refs = refs.withDefaultsV0()
	refs.DeliveryRef = causal.DeliveryRef
	refs.ReviewRequestRef = causal.ReviewRequestRef
	refs.ReviewResultRef = causal.ReviewResultRef
	refs.AcceptedReviewRef = causal.AcceptedReviewRef
	return refs.planStateV0()
}

func (refs codexStackRequiredTestRefsV0) planStateForDeliveryReviewV0(
	deliveryRef string,
) orquestacionnucleoapp.OperationalDirectorPlanStateV0 {
	refs = refs.withDefaultsV0()
	refs.DeliveryRef = strings.TrimSpace(deliveryRef)
	state := refs.planStateV0()
	state.ActiveStepID = "step-review-deliveries"
	for index := range state.Steps {
		switch state.Steps[index].StepID {
		case "step-wait-subagents":
			state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
			state.Steps[index].PendingAgentRefs = nil
			state.Steps[index].DeliveryRefs = []string{refs.DeliveryRef}
		case "step-review-deliveries":
			state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
			state.Steps[index].DeliveryRefs = []string{refs.DeliveryRef}
			state.Steps[index].ReviewResultRefs = nil
			state.Steps[index].AcceptedReviewRefs = nil
		case "step-run-required-tests":
			state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepPendingV0
			state.Steps[index].DeliveryRefs = nil
			state.Steps[index].ReviewResultRefs = nil
			state.Steps[index].AcceptedReviewRefs = nil
			state.Steps[index].BlockerRefs = nil
		case "step-replan-or-close":
			state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepPendingV0
		}
	}
	return state
}

func codexStackRealRequiredTestRunnerStackV0(
	t *testing.T,
	cfg codexStackRealSmokeConfigV0,
	processRuntime codexStackRuntimeForTestV0,
	evidenceStore orquestacionnucleoapp.RequiredTestEvidenceStorePortV0,
	goCommand string,
	outputDir string,
) StackV0 {
	t.Helper()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	waitStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskWaitStateStoreV0()
	planStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0()
	runMemory := orquestarunmemory.NewRunMemoryStoreV0()
	stack, err := BuildStackV0(ConfigV0{
		Enabled: true,
		Timeout: cfg.Timeout,
		DirectorLimits: orquestaweb.WebArrancarDirectorAppLimitsV0{
			MaxBursts:            4,
			MaxStepsPerBurst:     4,
			MaxDispatchesPerWait: 2,
			MaxCommands:          8,
			MaxOutboxPerCycle:    8,
			MaxExternalWaits:     1,
		},
		Stores: StoresV0{
			RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(),
			EventSink:                  orquestacionnucleoapp.NewInMemoryEventSinkV0(),
			OutboxLedger:               orquestacionnucleoapp.NewInMemoryOutboxLedgerV0(),
			TaskStore:                  taskStore,
			WaitStateStore:             waitStore,
			OperationalPlanStateWriter: planStore,
			OperationalPlanStateStore:  planStore,
			RequiredTestEvidenceStore:  evidenceStore,
			AppChangeStore:             orquestaappchange.NewInMemoryAppChangeStoreV0(),
			ReceiptStore:               orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(),
			ProgressState:              orquestaruntimecodexdelivery.NewInMemoryCodexProgressStateStoreV0(),
			ProcessRegistry:            orquestaagentprocessregistrymemory.NewInMemoryAgentProcessRegistryV0(),
			RunControl:                 runMemory,
			RunQueue:                   runMemory,
		},
		Codex: CodexRuntimeConfigV0{
			CommandPath:     cfg.CommandPath,
			ProjectWorkDir:  cfg.ProjectWorkDir,
			RuntimeWorkDir:  cfg.RuntimeWorkDir,
			CodeHomeDir:     cfg.CodeHomeDir,
			HomeDir:         cfg.HomeDir,
			PathEnv:         cfg.PathEnv,
			Model:           cfg.Model,
			ReasoningEffort: cfg.ReasoningEffort,
			Profile:         cfg.Profile,
			Sandbox:         cfg.Sandbox,
			ApprovalPolicy:  cfg.ApprovalPolicy,
			ExtraArgs:       cfg.ExtraArgs,
			PromptHints: []string{
				"Smoke opt-in: tarea pequena, comunicacion compacta, terminar con ACK valido.",
				"Si el proyecto ya compila, evita ampliar alcance; deja evidencia breve.",
			},
			Runtime:        processRuntime,
			ProcessStopper: processRuntime,
			SnapshotSource: processRuntime,
			MaxBatchReady:  cfg.MaxBatchReady,
			MaxConcurrency: cfg.MaxConcurrency,
			WaitInterval:   time.Second,
		},
		Capacity: CapacityConfigV0{
			Tier:            orquestacoreworkflow.OrchestrationCapacityHighV0,
			ReasoningEffort: orquestacoreworkflow.OrchestrationCapacityMediumV0,
			OccurredAt:      "2026-05-22T17:00:00Z",
			RequestedBy:     "orquesta-app-stack-required-test-smoke",
			Summary:         "Capacidad opt-in para tests requeridos con ejecucion controlada.",
			EvidenceRefs:    []string{"evidence-ref-rt-runner"},
		},
		ReviewGate: ReviewGateConfigV0{
			FileEvidence: orquestaruntimecodexdelivery.CodexReviewGateProjectFileEvidenceV0{},
		},
		RequiredTests: orquestacionnucleoapp.RequiredTestRunnerV0{
			Executor: orquestaruntimerequiredtest.LocalCommandExecutorV0{
				ProjectWorkDir: cfg.ProjectWorkDir,
				OutputDir:      outputDir,
				AllowedCommands: map[string]string{
					"go": goCommand,
				},
				Env:            []string{"CGO_ENABLED=0", "GOCACHE=" + filepath.Join(outputDir, "go-cache")},
				MaxOutputBytes: 1024 * 1024,
			},
			EvidenceWriter: evidenceStore,
		},
	})
	if err != nil {
		t.Fatalf("BuildStackV0: %v", err)
	}
	return stack
}

func writeCodexStackRequiredTestTinyGoModuleV0(t *testing.T, dir string) {
	t.Helper()
	files := map[string]string{
		"go.mod": "module example.com/orquesta-rt-runner\n\ngo 1.22\n",
		"calc.go": strings.Join([]string{
			"package calc",
			"",
			"func Add(a int, b int) int {",
			"\treturn a + b",
			"}",
			"",
		}, "\n"),
		"calc_test.go": strings.Join([]string{
			"package calc",
			"",
			"import \"testing\"",
			"",
			"func TestAdd(t *testing.T) {",
			"\tif Add(2, 3) != 5 {",
			"\t\tt.Fatalf(\"Add fallo\")",
			"\t}",
			"}",
			"",
		}, "\n"),
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
}

func writeCodexStackRequiredTestFailingGoModuleV0(t *testing.T, dir string) {
	t.Helper()
	writeCodexStackRequiredTestTinyGoModuleV0(t, dir)
	path := filepath.Join(dir, "calc_test.go")
	content := strings.Join([]string{
		"package calc",
		"",
		"import \"testing\"",
		"",
		"func TestAdd(t *testing.T) {",
		"\tif Add(2, 3) != 99 {",
		"\t\tt.Fatalf(\"Add esperado fallo controlado\")",
		"\t}",
		"}",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write failing calc_test.go: %v", err)
	}
}

func codexStackRequiredTestLocalConfigV0(t *testing.T) codexStackRealSmokeConfigV0 {
	t.Helper()
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	homeDir := filepath.Join(t.TempDir(), "home")
	codeHomeDir := filepath.Join(t.TempDir(), "code-home")
	for _, dir := range []string{runtimeDir, homeDir, codeHomeDir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatalf("mkdir local required-test cfg: %v", err)
		}
	}
	return codexStackRealSmokeConfigV0{
		CommandPath:     filepath.Join(projectDir, "codex-bin"),
		ProjectWorkDir:  projectDir,
		RuntimeWorkDir:  runtimeDir,
		CodeHomeDir:     codeHomeDir,
		HomeDir:         homeDir,
		PathEnv:         os.Getenv("PATH"),
		Model:           "fake-codex",
		ReasoningEffort: "medium",
		Sandbox:         "workspace-write",
		ApprovalPolicy:  "never",
		MaxBatchReady:   1,
		MaxConcurrency:  1,
		Timeout:         5 * time.Second,
	}
}

func codexStackRequiredTestGoCommandV0(t *testing.T) string {
	t.Helper()
	raw := strings.TrimSpace(os.Getenv("ORQUESTA_REQUIRED_TEST_GO_COMMAND"))
	if raw == "" {
		raw = "go"
	}
	if filepath.IsAbs(raw) {
		return raw
	}
	path, err := exec.LookPath(raw)
	if err != nil {
		t.Fatalf("go command no encontrado: %s", raw)
	}
	return path
}

func codexStackRequiredTestContinueRequestV0(
	refs codexStackRequiredTestRefsV0,
	correlationID string,
	occurredAt string,
	maxBursts int,
) orquestaappdirectorservice.ContinueAppDirectorRequestV0 {
	refs = refs.withDefaultsV0()
	return orquestaappdirectorservice.ContinueAppDirectorRequestV0{
		RunRef:                     refs.RunRef,
		OperationalDirectorPlanRef: refs.PlanRef,
		OccurredAt:                 occurredAt,
		CorrelationID:              correlationID,
		RequestedBy:                "orquesta-app-stack-required-test-failed-replan-test",
		MaxBursts:                  maxBursts,
		MaxStepsPerBurst:           8,
		MaxDispatchesPerWait:       2,
		MaxCommands:                12,
		MaxOutboxPerCycle:          8,
		MaxExternalWaits:           1,
	}
}

func codexStackRequiredTestFirstFollowupAgentRefV0(
	t *testing.T,
	replan codexStackRealSmokeReplanProjectionV0,
) string {
	t.Helper()
	for _, ref := range replan.FollowupRefs {
		if strings.HasPrefix(ref, "agent-ref-") {
			return ref
		}
	}
	t.Fatalf("replan sin followup agent: %+v", replan)
	return ""
}

func codexStackRequiredTestDescriptorForAgentV0(
	t *testing.T,
	stack StackV0,
	agentRef string,
) orquestaruntimecodexdelivery.CodexReceiptDescriptorV0 {
	t.Helper()
	for _, descriptor := range codexStackRequiredTestReceiptDescriptorsV0(t, stack) {
		if strings.TrimSpace(descriptor.AgentRef) == strings.TrimSpace(agentRef) {
			return descriptor
		}
	}
	t.Fatalf("descriptor no encontrado agent_ref=%s descriptors=%+v", agentRef, codexStackRequiredTestReceiptDescriptorsV0(t, stack))
	return orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{}
}

type codexStackRequiredTestPendingRuntimeV0 struct {
	mu        sync.Mutex
	next      int
	snapshots map[string]orquestaruntime.ProcessRuntimeSnapshotV0
}

func (runtime *codexStackRequiredTestPendingRuntimeV0) LaunchV0(
	_ context.Context,
	_ orquestaruntime.ProcessRuntimeLaunchRequestV0,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	if runtime.snapshots == nil {
		runtime.snapshots = map[string]orquestaruntime.ProcessRuntimeSnapshotV0{}
	}
	runtime.next++
	ref := fmt.Sprintf("%03d", runtime.next)
	snapshot := orquestaruntime.ProcessRuntimeSnapshotV0{
		SchemaVersion: orquestaruntime.ProcessRuntimeConnectorVersionV0,
		ProcessRef:    "process-ref-rt-pending-" + ref,
		SessionRef:    "session-ref-rt-pending-" + ref,
		LaunchRef:     "launch-ref-rt-pending-" + ref,
		Status:        orquestaruntime.ProcessRuntimeRunningV0,
	}
	runtime.snapshots[snapshot.ProcessRef] = snapshot
	return snapshot, nil
}

func (runtime *codexStackRequiredTestPendingRuntimeV0) StopV0(
	_ context.Context,
	processRef string,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	snapshot := runtime.snapshots[processRef]
	snapshot.Status = orquestaruntime.ProcessRuntimeStoppedV0
	snapshot.StopRef = "stop-ref-" + processRef
	runtime.snapshots[processRef] = snapshot
	return snapshot, nil
}

func (runtime *codexStackRequiredTestPendingRuntimeV0) SnapshotV0(
	processRef string,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	return runtime.snapshots[processRef], nil
}

func (runtime *codexStackRequiredTestPendingRuntimeV0) launchCountV0() int {
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	return runtime.next
}

func codexStackRequiredTestOutputDirV0(t *testing.T) string {
	t.Helper()
	value := strings.TrimSpace(os.Getenv("ORQUESTA_REQUIRED_TEST_OUTPUT_DIR"))
	if value == "" {
		return filepath.Join(t.TempDir(), "required-test-output")
	}
	return value
}

func codexStackRequiredTestPhasesV0() []orquestacoreworkflow.OrchestrationPhaseV0 {
	capacity := orquestacoreworkflow.OrchestrationCapacityHighV0
	return []orquestacoreworkflow.OrchestrationPhaseV0{
		{ID: orquestacoreworkflow.OrchestrationPhaseProgramacionV0, Status: orquestacoreworkflow.OrchestrationPhaseStatusPendingV0, RecommendedCapacity: capacity},
		{ID: orquestacoreworkflow.OrchestrationPhaseRevisionV0, Status: orquestacoreworkflow.OrchestrationPhaseStatusActiveV0, RecommendedCapacity: capacity},
		{ID: orquestacoreworkflow.OrchestrationPhaseValidacionFinalV0, Status: orquestacoreworkflow.OrchestrationPhaseStatusPendingV0, RecommendedCapacity: capacity},
		{ID: orquestacoreworkflow.OrchestrationPhaseCierreV0, Status: orquestacoreworkflow.OrchestrationPhaseStatusPendingV0, RecommendedCapacity: capacity},
	}
}

func codexStackRequiredTestEventMetaV0(
	runRef string,
	sequence int64,
	suffix string,
) orquestacoreworkflow.OrchestrationEventMetaV0 {
	return orquestacoreworkflow.OrchestrationEventMetaV0{
		EventID:       "evt-rt-runner-" + suffix,
		RunID:         runRef,
		Sequence:      sequence,
		CorrelationID: "corr-rt-runner-seed",
		OccurredAt:    "2026-05-22T16:59:00Z",
	}
}

func codexStackRequiredTestPlanStepV0(
	t *testing.T,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	stepID string,
) orquestacionnucleoapp.OperationalDirectorPlanStepStateV0 {
	t.Helper()
	for _, step := range state.Steps {
		if step.StepID == stepID {
			return step
		}
	}
	t.Fatalf("step no encontrado: %s en %+v", stepID, state.Steps)
	return orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{}
}

func codexStackRequiredTestOutputRefV0(refs []string) string {
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if strings.HasPrefix(ref, "required-test-output-v0/") {
			return ref
		}
	}
	return ""
}

func codexStackRequiredTestDrainUntilDeliveryV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	runRef string,
	agentRef string,
	runtimeWorkDir string,
	timeout time.Duration,
) orquestaruntimecodexdelivery.CodexReceiptDescriptorV0 {
	t.Helper()
	deadline := time.Now().Add(timeout)
	pollInterval := 2 * time.Second
	maxCycles := codexStackRealSmokeMaxExternalWaitsV0(timeout, pollInterval)
	var lastRun orquestacoreworkflow.OrchestrationRunV0
	for cycle := 1; cycle <= maxCycles && time.Now().Before(deadline); cycle++ {
		if _, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
			RunRef:               runRef,
			CorrelationID:        "corr-rt-runner-e2e-delivery",
			WaitAgentRefs:        []string{agentRef},
			MaxBursts:            4,
			MaxStepsPerBurst:     8,
			MaxDispatchesPerWait: 4,
			MaxCommands:          12,
			MaxOutboxPerCycle:    6,
			MaxExternalWaits:     1,
		}); err != nil {
			t.Fatalf("DrainRunV0 delivery ciclo=%d: %v\n%s", cycle, err, codexStackRealSmokeDiagnosticsV0(runtimeWorkDir))
		}
		run := mustLoadCodexStackRunForTestV0(t, stack, runRef)
		lastRun = run
		for _, descriptor := range codexStackRequiredTestReceiptDescriptorsV0(t, stack) {
			if strings.TrimSpace(descriptor.AgentRef) != agentRef {
				continue
			}
			ack, ok := codexStackRealSmokeCompletedAckV0(descriptor)
			if ok && codexStackRealSmokeContainsProjectionPartV0(run.Deliveries, ack.AckRef) {
				return descriptor
			}
		}
		select {
		case <-ctx.Done():
			t.Fatalf("contexto cancelado esperando entrega Codex real: %v\n%s", ctx.Err(), codexStackRealSmokeDiagnosticsV0(runtimeWorkDir))
		case <-time.After(pollInterval):
		}
	}
	t.Fatalf("sin entrega Codex real: run=%+v descriptors=%v\n%s",
		lastRun,
		codexStackRequiredTestReceiptDescriptorsV0(t, stack),
		codexStackRealSmokeDiagnosticsV0(runtimeWorkDir),
	)
	return orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{}
}

func codexStackRequiredTestDrainUntilAcceptedReviewV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	runRef string,
	planRef string,
	deliveryRef string,
	runtimeWorkDir string,
	timeout time.Duration,
) codexStackRequiredTestCausalReviewRefsV0 {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for cycle := 1; cycle <= 12 && time.Now().Before(deadline); cycle++ {
		if _, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
			RunRef:                     runRef,
			CorrelationID:              "corr-rt-runner-e2e-review",
			MaxBursts:                  4,
			MaxStepsPerBurst:           8,
			MaxDispatchesPerWait:       4,
			MaxCommands:                12,
			MaxOutboxPerCycle:          6,
			MaxExternalWaits:           1,
			OperationalDirectorPlanRef: planRef,
		}); err != nil {
			t.Fatalf("DrainRunV0 review ciclo=%d: %v\n%s", cycle, err, codexStackRealSmokeDiagnosticsV0(runtimeWorkDir))
		}
		causal, ok := codexStackRequiredTestAcceptedReviewRefsV0(t, stack, runRef, deliveryRef)
		if ok {
			return causal
		}
		time.Sleep(time.Second)
	}
	run := mustLoadCodexStackRunForTestV0(t, stack, runRef)
	t.Fatalf("sin review aceptada causal delivery=%s reviews=%v results=%v accepted=%v\n%s",
		deliveryRef,
		run.Reviews,
		run.ReviewResults,
		run.AcceptedReviews,
		codexStackRealSmokeDiagnosticsV0(runtimeWorkDir),
	)
	return codexStackRequiredTestCausalReviewRefsV0{}
}

func codexStackRequiredTestAcceptedReviewRefsV0(
	t *testing.T,
	stack StackV0,
	runRef string,
	deliveryRef string,
) (codexStackRequiredTestCausalReviewRefsV0, bool) {
	t.Helper()
	reader, ok := stack.Stores.EventSink.(orquestacionnucleoapp.RunEventReaderPortV0)
	if !ok {
		t.Fatalf("EventSink no lee eventos: %T", stack.Stores.EventSink)
	}
	events, err := reader.LoadRunEventsV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadRunEventsV0: %v", err)
	}
	var causal codexStackRequiredTestCausalReviewRefsV0
	causal.DeliveryRef = strings.TrimSpace(deliveryRef)
	for _, event := range events {
		switch event.EventType {
		case orquestacoreworkflow.OrchestrationEventReviewRequestedV0:
			var payload orquestacoreworkflow.ReviewRequestedPayloadV0
			if json.Unmarshal(event.Payload, &payload) == nil &&
				strings.TrimSpace(payload.DeliveryRef) == deliveryRef {
				causal.ReviewRequestRef = strings.TrimSpace(payload.ReviewRequestID)
			}
		case orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0:
			var payload orquestacoreworkflow.ReviewResultV0
			if json.Unmarshal(event.Payload, &payload) == nil &&
				strings.TrimSpace(payload.DeliveryRef) == deliveryRef &&
				payload.Status == orquestacoreworkflow.ReviewResultStatusAcceptedV0 {
				causal.ReviewResultRef = strings.TrimSpace(payload.ReviewResultRef)
				if causal.ReviewRequestRef == "" {
					causal.ReviewRequestRef = strings.TrimSpace(payload.ReviewRequestID)
				}
			}
		case orquestacoreworkflow.OrchestrationEventReviewAcceptedV0:
			var payload orquestacoreworkflow.ReviewAcceptedPayloadV0
			if json.Unmarshal(event.Payload, &payload) == nil &&
				strings.TrimSpace(payload.DeliveryRef) == deliveryRef {
				causal.AcceptedReviewRef = strings.TrimSpace(payload.AcceptedReviewRef)
				if causal.ReviewRequestRef == "" {
					causal.ReviewRequestRef = strings.TrimSpace(payload.ReviewRequestID)
				}
			}
		}
	}
	return causal, causal.DeliveryRef != "" &&
		causal.ReviewRequestRef != "" &&
		causal.ReviewResultRef != "" &&
		causal.AcceptedReviewRef != ""
}

func codexStackRequiredTestReceiptDescriptorsV0(
	t *testing.T,
	stack StackV0,
) []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0 {
	t.Helper()
	store, ok := stack.Stores.ReceiptStore.(*orquestaruntimecodexdelivery.InMemoryCodexReceiptDescriptorStoreV0)
	if !ok {
		t.Fatalf("ReceiptStore inesperado: %T", stack.Stores.ReceiptStore)
	}
	return codexStackRealSmokeDescriptorsV0(t, store)
}

func codexStackRequiredTestRuntimeEntriesV0(t *testing.T, runtimeDir string) []string {
	t.Helper()
	entries := []string{}
	err := filepath.WalkDir(runtimeDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == runtimeDir {
			return nil
		}
		rel, err := filepath.Rel(runtimeDir, path)
		if err != nil {
			return err
		}
		entries = append(entries, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatalf("walk runtime dir: %v", err)
	}
	return entries
}
