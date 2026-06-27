package orquestaappcodexstack

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestCodexStackV0AppChangeNotificaDirectorPorWorkflow(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	runRef := codexStackStartRunForChangeTestV0(t, stack)
	body := new(bytes.Buffer)
	_ = json.NewEncoder(body).Encode(orquestamcp.MCPRequestAppChangeToolInputV0{
		AppChangeRequest: validCodexStackAppChangeRequestV0(runRef),
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v0/apps/agenda-equipo/changes", body)
	rec := httptest.NewRecorder()

	stack.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPRequestAppChangeToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.DirectorQuestionRef != "question-ref-app-change-change-ref-stack-web-001" {
		t.Fatalf("result=%+v", result)
	}
	run, err := stack.Stores.RunStore.LoadRunV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("load run: %v", err)
	}
	if !codexStackStringInSetForTestV0(run.DirectorQuestions, result.DirectorQuestionRef) {
		t.Fatalf("director_questions=%v", run.DirectorQuestions)
	}
	pending, issues := stack.Stores.OutboxLedger.ListPending(
		context.Background(),
		orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0{
			RunRef:     runRef,
			TargetPort: orquestacoreworkflow.OutboxTargetDirectorV0,
		},
	)
	if len(issues) > 0 || len(pending) == 0 {
		t.Fatalf("pending=%v issues=%+v", pending, issues)
	}
	if _, err := stack.DrainRunV0(context.Background(), DrainRunRequestV0{
		RunRef:               runRef,
		CorrelationID:        "corr-stack-change-drain-001",
		MaxBursts:            16,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 8,
		MaxExternalWaits:     2,
	}); err != nil {
		t.Fatalf("DrainRunV0: %v", err)
	}
	run, err = stack.Stores.RunStore.LoadRunV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("load run after drain: %v", err)
	}
	if run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseRevisionV0 {
		t.Fatalf("current_phase=%s", run.CurrentPhase)
	}
	if len(run.Tasks) != 1 {
		t.Fatalf("tasks=%v", run.Tasks)
	}
	tasks, err := stack.Stores.TaskStore.LoadWorkflowTasksV0(context.Background(), runRef, run.Tasks)
	if err != nil {
		t.Fatalf("load tasks: %v", err)
	}
	if len(tasks) != 1 || len(tasks[0].WriteSet) != 1 || tasks[0].WriteSet[0] != "web/agenda" {
		t.Fatalf("tasks loaded=%+v", tasks)
	}
	if runtime.launchCountV0() < 5 {
		t.Fatalf("launches=%d want>=5", runtime.launchCountV0())
	}
}

func TestDrainRunV0AppChangeConProgramacionPendienteArrancaCambio(t *testing.T) {
	runtime := newPendingProgrammingAckCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	director := postDirectorAPIV0(t, stack)
	if _, err := stack.DrainRunV0(context.Background(), DrainRunRequestV0{
		RunRef:               director.RunRef,
		CorrelationID:        "corr-stack-change-pending-initial",
		MaxBursts:            16,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 8,
		MaxExternalWaits:     1,
	}); err != nil {
		t.Fatalf("DrainRunV0 inicial: %v", err)
	}
	if !codexStackHasDescriptorWriteSetV0(t, stack, "internal/agenda") {
		t.Fatalf("descriptor de programacion inicial no encontrado")
	}
	mustPostCodexStackAppChangeV0(t, stack, validCodexStackAppChangeRequestV0(director.RunRef))

	if _, err := stack.DrainRunV0(context.Background(), DrainRunRequestV0{
		RunRef:               director.RunRef,
		CorrelationID:        "corr-stack-change-pending-drain",
		MaxBursts:            16,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 8,
		MaxExternalWaits:     1,
	}); err != nil {
		t.Fatalf("DrainRunV0 cambio: %v", err)
	}
	if !codexStackHasDescriptorWriteSetV0(t, stack, "web/agenda") {
		run, err := stack.Stores.RunStore.LoadRunV0(context.Background(), director.RunRef)
		if err != nil {
			t.Fatalf("descriptor de cambio no encontrado y LoadRunV0 fallo: %v", err)
		}
		t.Fatalf(
			"descriptor de cambio no encontrado phase=%s decisions=%v questions=%v answered=%v tasks=%v contracts=%v agents=%v started=%v descriptors=%v",
			run.CurrentPhase,
			run.Decisions,
			run.DirectorQuestions,
			run.DirectorAnsweredQuestions,
			run.Tasks,
			run.FunctionContracts,
			run.Agents,
			run.StartedAgents,
			codexStackDescriptorWriteSetsV0(t, stack),
		)
	}
}

func codexStackStartRunForChangeTestV0(t *testing.T, stack StackV0) string {
	t.Helper()
	body := new(bytes.Buffer)
	_ = json.NewEncoder(body).Encode(orquestamcp.MCPArrancarDirectorAppToolInputV0{
		RequestID:             "req-stack-change-start-001",
		CorrelationID:         "corr-stack-change-start-001",
		DirectorExecutionMode: orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0,
		AppSpecRequest:        codexStackAppSpecRequestV0(),
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v0/apps/director", body)
	rec := httptest.NewRecorder()
	stack.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("start status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPArrancarDirectorAppToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode start: %v", err)
	}
	if result.RunRef == "" {
		t.Fatalf("start result=%+v", result)
	}
	return result.RunRef
}

func mustPostCodexStackAppChangeV0(
	t *testing.T,
	stack StackV0,
	change orquestaappchange.AppChangeRequestV0,
) {
	t.Helper()
	body := new(bytes.Buffer)
	_ = json.NewEncoder(body).Encode(orquestamcp.MCPRequestAppChangeToolInputV0{
		AppChangeRequest: change,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v0/apps/agenda-equipo/changes", body)
	rec := httptest.NewRecorder()
	stack.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("change status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func codexStackHasDescriptorWriteSetV0(t *testing.T, stack StackV0, path string) bool {
	t.Helper()
	for _, writeSet := range codexStackDescriptorWriteSetsV0(t, stack) {
		for _, entry := range writeSet {
			if entry == path {
				return true
			}
		}
	}
	return false
}

func codexStackDescriptorWriteSetsV0(t *testing.T, stack StackV0) [][]string {
	t.Helper()
	store := stack.Stores.ReceiptStore.(*orquestaruntimecodexdelivery.InMemoryCodexReceiptDescriptorStoreV0)
	out := [][]string{}
	for _, descriptor := range codexStackRealSmokeDescriptorsV0(t, store) {
		out = append(out, append([]string(nil), descriptor.Spec.AgentPacket.Task.WriteSet...))
	}
	return out
}

type pendingProgrammingAckCodexStackRuntimeV0 struct {
	*decisionWritingFakeCodexStackRuntimeV0
}

func newPendingProgrammingAckCodexStackRuntimeV0() *pendingProgrammingAckCodexStackRuntimeV0 {
	return &pendingProgrammingAckCodexStackRuntimeV0{
		decisionWritingFakeCodexStackRuntimeV0: newDecisionWritingFakeCodexStackRuntimeV0(),
	}
}

func (runtime *pendingProgrammingAckCodexStackRuntimeV0) LaunchV0(
	ctx context.Context,
	req orquestaruntime.ProcessRuntimeLaunchRequestV0,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	snapshot, err := runtime.decisionWritingFakeCodexStackRuntimeV0.LaunchV0(ctx, req)
	if err != nil {
		return snapshot, err
	}
	runtimeDir := filepath.Dir(req.CommandPath)
	packet, err := codexStackPacketFromRuntimeDirForTestV0(runtimeDir)
	if err != nil {
		return snapshot, err
	}
	if packet.TargetModule == "orquesta-app-stack-programacion" {
		_ = os.Remove(filepath.Join(runtimeDir, orquestaruntimecodex.CodexAgentAckFileNameV0))
	}
	return snapshot, nil
}

func validCodexStackAppChangeRequestV0(runRef string) orquestaappchange.AppChangeRequestV0 {
	return orquestaappchange.AppChangeRequestV0{
		RunRef:             runRef,
		ChangeRef:          "change-ref-stack-web-001",
		UserIntent:         "Cambiar la web para mostrar una vista semanal.",
		TargetArea:         "web",
		CurrentStateRefs:   []string{"delivery-ref-web-001"},
		AcceptanceCriteria: []string{"vista semanal visible"},
		AllowedWriteSet:    []string{"web/agenda"},
	}
}
