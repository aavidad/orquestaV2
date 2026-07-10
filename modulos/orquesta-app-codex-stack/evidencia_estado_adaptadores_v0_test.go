package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	orquestaagentprocessregistrymemory "orquesta/modulos/orquesta-agent-process-registry-memory"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaestadovivo "orquesta/modulos/orquesta-estado-vivo"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestEvidenciaEstadoRunStoreV0TraduceRunSinLogicaFase(t *testing.T) {
	ctx := context.Background()
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(orquestacoreworkflow.OrchestrationRunV0{
		RunID:          "run-ref-evidencia-runstore-001",
		Status:         orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		LastEventID:    "event-ref-runstore-001",
		Deliveries:     []string{"delivery-ref-runstore-001"},
		PhaseArtifacts: []string{"artifact-ref-runstore-001"},
	})
	source := EvidenciaEstadoRunStoreV0{Store: store}

	evidencias, err := source.ListarEvidenciasEstadoV0(ctx, orquestaestadovivo.FiltroEvidenciaEstadoV0{
		RunRef: "run-ref-evidencia-runstore-001",
	})
	if err != nil {
		t.Fatalf("ListarEvidenciasEstadoV0: %v", err)
	}
	if len(evidencias) != 1 {
		t.Fatalf("evidencias=%+v, want 1", evidencias)
	}
	got := evidencias[0]
	if got.RunRef != "run-ref-evidencia-runstore-001" ||
		got.Fuente != "run_store" ||
		got.Estado != string(orquestacoreworkflow.OrchestrationRunStatusActiveV0) ||
		got.Terminal ||
		got.ProcesoVivo {
		t.Fatalf("evidencia run_store inesperada: %+v", got)
	}
	requireEvidenceRefsV0(t, got.EvidenceRefs, "event-ref-runstore-001", "delivery-ref-runstore-001", "artifact-ref-runstore-001")
}

func TestEvidenciaEstadoGoalStateV0TraduceAppGoalStateStore(t *testing.T) {
	ctx := context.Background()
	store := newGoalFirstQueueStateStoreForTestV0()
	state := evidenciaEstadoGoalStateForTestV0("run-ref-evidencia-goalstate-001", "goal-ref-evidencia-goalstate-001")
	state.EvidenceRefs = []string{"evidence-ref-goalstate-state-001"}
	state.LaunchReceipt.EvidenceRefs = []string{"evidence-ref-goalstate-launch-001"}
	if err := store.SaveGoalWorkStateV0(ctx, state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	source := EvidenciaEstadoGoalStateV0{Store: store}

	evidencias, err := source.ListarEvidenciasEstadoV0(ctx, orquestaestadovivo.FiltroEvidenciaEstadoV0{
		GoalRef: "goal-ref-evidencia-goalstate-001",
	})
	if err != nil {
		t.Fatalf("ListarEvidenciasEstadoV0: %v", err)
	}
	if len(evidencias) != 1 {
		t.Fatalf("evidencias=%+v, want 1", evidencias)
	}
	got := evidencias[0]
	if got.RunRef != state.RunRef ||
		got.GoalRef != state.GoalRef ||
		got.Fuente != "goal_state" ||
		got.Estado != orquestagoal.GoalStatusRunningV0 ||
		got.Terminal ||
		got.Aceptado {
		t.Fatalf("evidencia goal_state inesperada: %+v", got)
	}
	requireEvidenceRefsV0(t, got.EvidenceRefs, "evidence-ref-goalstate-state-001", "evidence-ref-goalstate-launch-001")
}

func TestEvidenciaEstadoMarkerV0TraduceGoalWorkRunMarkerStore(t *testing.T) {
	ctx := context.Background()
	store := newGoalFirstQueueStateStoreForTestV0()
	marker := orquestagoal.GoalWorkRunMarkerV0{
		RunRef:          "run-ref-evidencia-marker-001",
		GoalRef:         "goal-ref-evidencia-marker-001",
		ExternalGoalRef: "external-goal-ref-evidencia-marker-001",
		DirectorKind:    orquestagoal.GoalDirectorKindCodexGoalV0,
		Status:          orquestagoal.GoalStatusRunningV0,
		EvidenceRefs:    []string{"evidence-ref-marker-001"},
		LaunchReceipt: &orquestagoal.GoalLaunchReceiptV0{
			Status:       orquestagoal.GoalStatusAcceptedV0,
			GoalRef:      "goal-ref-evidencia-marker-001",
			EvidenceRefs: []string{"evidence-ref-marker-launch-001"},
		},
	}
	if err := store.SaveGoalWorkRunMarkerV0(ctx, marker); err != nil {
		t.Fatalf("SaveGoalWorkRunMarkerV0: %v", err)
	}
	source := EvidenciaEstadoMarkerV0{Store: store}

	evidencias, err := source.ListarEvidenciasEstadoV0(ctx, orquestaestadovivo.FiltroEvidenciaEstadoV0{
		RunRef: marker.RunRef,
	})
	if err != nil {
		t.Fatalf("ListarEvidenciasEstadoV0: %v", err)
	}
	if len(evidencias) != 1 {
		t.Fatalf("evidencias=%+v, want 1", evidencias)
	}
	got := evidencias[0]
	if got.RunRef != marker.RunRef ||
		got.GoalRef != marker.GoalRef ||
		got.Fuente != "run_marker" ||
		got.Estado != orquestagoal.GoalStatusRunningV0 ||
		got.ProcesoVivo ||
		got.Terminal {
		t.Fatalf("evidencia marker inesperada: %+v", got)
	}
	requireEvidenceRefsV0(t, got.EvidenceRefs, "evidence-ref-marker-001", "evidence-ref-marker-launch-001")
}

func TestEvidenciaEstadoProcesosV0MarcaProcesoVivoSoloConSnapshotConfirmado(t *testing.T) {
	ctx := context.Background()
	registry := orquestaagentprocessregistrymemory.NewInMemoryAgentProcessRegistryV0()
	mustRecordAgentProcessForEvidenciaTestV0(t, registry, "run-ref-evidencia-proceso-001", "agent-ref-proceso-live", "process-ref-live")
	mustRecordAgentProcessForEvidenciaTestV0(t, registry, "run-ref-evidencia-proceso-001", "agent-ref-proceso-stopped", "process-ref-stopped")
	snapshots := evidenciaEstadoSnapshotSourceForTestV0{
		snapshots: map[string]orquestaruntime.ProcessRuntimeSnapshotV0{
			"process-ref-live": {
				SchemaVersion: orquestaruntime.ProcessRuntimeConnectorVersionV0,
				ProcessRef:    "process-ref-live",
				SessionRef:    "session-ref-process-ref-live",
				LaunchRef:     "launch-ref-process-ref-live",
				Status:        orquestaruntime.ProcessRuntimeRunningV0,
			},
			"process-ref-stopped": {
				SchemaVersion: orquestaruntime.ProcessRuntimeConnectorVersionV0,
				ProcessRef:    "process-ref-stopped",
				SessionRef:    "session-ref-process-ref-stopped",
				LaunchRef:     "launch-ref-process-ref-stopped",
				Status:        orquestaruntime.ProcessRuntimeStoppedV0,
			},
		},
	}
	source := EvidenciaEstadoProcesosV0{Registry: registry, SnapshotSource: snapshots}

	evidencias, err := source.ListarEvidenciasEstadoV0(ctx, orquestaestadovivo.FiltroEvidenciaEstadoV0{
		RunRef: "run-ref-evidencia-proceso-001",
	})
	if err != nil {
		t.Fatalf("ListarEvidenciasEstadoV0: %v", err)
	}
	if len(evidencias) != 4 {
		t.Fatalf("evidencias=%+v, want registry+snapshot por proceso", evidencias)
	}
	live := evidenciaPorFuenteYEstadoForTestV0(evidencias, "process_snapshot", string(orquestaruntime.ProcessRuntimeRunningV0))
	if live == nil || !live.ProcesoVivo || !live.RuntimeObservado || !live.RuntimeObservationAttempted ||
		live.Scope != orquestaestadovivo.ScopeGoalExecutionV0 || live.RuntimeIdentityRef != "process-ref-live" {
		t.Fatalf("snapshot running debe confirmar proceso vivo: %+v", live)
	}
	stopped := evidenciaPorFuenteYEstadoForTestV0(evidencias, "process_snapshot", string(orquestaruntime.ProcessRuntimeStoppedV0))
	if stopped == nil || stopped.ProcesoVivo || !stopped.RuntimeObservado || !stopped.RuntimeObservationAttempted ||
		stopped.Scope != orquestaestadovivo.ScopeGoalExecutionV0 || stopped.RuntimeIdentityRef != "process-ref-stopped" {
		t.Fatalf("snapshot stopped no debe marcar proceso vivo: %+v", stopped)
	}
	registryLive := evidenciaPorFuenteYEstadoForTestV0(evidencias, "process_registry", "registered")
	if registryLive == nil || registryLive.Scope != orquestaestadovivo.ScopeGoalExecutionV0 ||
		registryLive.RuntimeIdentityRef == "" || registryLive.RuntimeGenerationRef == "" || registryLive.RuntimeObservado {
		t.Fatalf("registry debe aportar identidad esperada sin fingir observacion: %+v", registryLive)
	}
	requireEvidenceRefsV0(t, live.EvidenceRefs, "process-ref-live", "agent-ref-proceso-live", "evidence-ref-process-agent-ref-proceso-live")
}

func TestEvidenciaEstadoProcesosV0SnapshotFallidoEmiteObservacionIndeterminadaTipadaV0(t *testing.T) {
	registry := orquestaagentprocessregistrymemory.NewInMemoryAgentProcessRegistryV0()
	mustRecordAgentProcessForEvidenciaTestV0(t, registry, "run-ref-timeout-001", "agent-ref-timeout-001", "process-ref-timeout-001")
	source := EvidenciaEstadoProcesosV0{
		Registry: registry,
		SnapshotSource: evidenciaEstadoSnapshotSourceForTestV0{
			errs: map[string]error{"process-ref-timeout-001": errors.New("timeout privado")},
		},
	}

	evidencias, err := source.ListarEvidenciasEstadoV0(context.Background(), orquestaestadovivo.FiltroEvidenciaEstadoV0{RunRef: "run-ref-timeout-001"})
	if err != nil {
		t.Fatalf("ListarEvidenciasEstadoV0: %v", err)
	}
	indeterminate := evidenciaPorFuenteYEstadoForTestV0(evidencias, "process_snapshot", "observation_indeterminate")
	if indeterminate == nil || !indeterminate.RuntimeObservationAttempted || indeterminate.RuntimeObservado ||
		indeterminate.ProcesoVivo || indeterminate.RuntimeIdentityRef != "process-ref-timeout-001" {
		t.Fatalf("observacion fallida incompleta o falsamente viva: %+v", indeterminate)
	}
	body, _ := json.Marshal(evidencias)
	if string(body) == "" || strings.Contains(string(body), "timeout privado") {
		t.Fatalf("evidencia no debe filtrar error privado: %s", body)
	}
}

func TestEvidenciaEstadoProcesosV0SnapshotGeneracionDistintaEmiteDivergenciaCausalV0(t *testing.T) {
	registry := orquestaagentprocessregistrymemory.NewInMemoryAgentProcessRegistryV0()
	mustRecordAgentProcessForEvidenciaTestV0(t, registry, "run-ref-mismatch-001", "agent-ref-mismatch-001", "process-ref-mismatch-001")
	source := EvidenciaEstadoProcesosV0{
		Registry: registry,
		SnapshotSource: evidenciaEstadoSnapshotSourceForTestV0{snapshots: map[string]orquestaruntime.ProcessRuntimeSnapshotV0{
			"process-ref-mismatch-001": {
				ProcessRef: "process-ref-mismatch-001", SessionRef: "session-ref-otra-generacion",
				LaunchRef: "launch-ref-otra-generacion", Status: orquestaruntime.ProcessRuntimeRunningV0,
			},
		}},
	}
	evidencias, err := source.ListarEvidenciasEstadoV0(context.Background(), orquestaestadovivo.FiltroEvidenciaEstadoV0{RunRef: "run-ref-mismatch-001"})
	if err != nil {
		t.Fatalf("ListarEvidenciasEstadoV0: %v", err)
	}
	veredicto := orquestaestadovivo.DerivarVeredictoCausalV0(evidencias)
	if veredicto.Clase != orquestaestadovivo.VeredictoDivergentNeedsRepairV0 ||
		veredicto.ReasonCode != orquestaestadovivo.RazonVeredictoIdentidadNoCoincidenteV0 || veredicto.PublicarRunning {
		t.Fatalf("snapshot de otra generacion no puede confirmar running: evidencias=%+v veredicto=%+v", evidencias, veredicto)
	}
}

func TestEvidenciaEstadoReceiptsV0TraduceGoalWorkResultYReceiptCodex(t *testing.T) {
	ctx := context.Background()
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	state := evidenciaEstadoGoalStateForTestV0("run-ref-evidencia-receipt-001", "goal-ref-evidencia-receipt-001")
	state.LastResult = &orquestagoal.GoalWorkResultV0{
		Status:       orquestagoal.GoalStatusCompleteV0,
		GoalRef:      state.GoalRef,
		ArtifactRefs: []string{"artifact-ref-goal-result-001"},
		EvidenceRefs: []string{"evidence-ref-goal-result-001"},
		RequiredTestResults: []orquestagoal.GoalRequiredTestResultV0{{
			TestRef:      "required-test-ref-goal-result-001",
			Status:       "passed",
			EvidenceRefs: []string{"evidence-ref-goal-result-test-001"},
		}},
	}
	state.LastClosure = &orquestagoal.GoalClosureValidationV0{
		Status:       orquestagoal.GoalStatusAcceptedV0,
		Accepted:     true,
		EvidenceRefs: []string{"evidence-ref-goal-closure-001"},
	}
	if err := goalStates.SaveGoalWorkStateV0(ctx, state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	receiptStore := orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(
		evidenciaEstadoCodexReceiptDescriptorForTestV0(t, state.RunRef),
	)
	source := EvidenciaEstadoReceiptsV0{GoalStateStore: goalStates, ReceiptStore: receiptStore}

	evidencias, err := source.ListarEvidenciasEstadoV0(ctx, orquestaestadovivo.FiltroEvidenciaEstadoV0{
		RunRef: state.RunRef,
	})
	if err != nil {
		t.Fatalf("ListarEvidenciasEstadoV0: %v", err)
	}
	if len(evidencias) != 2 {
		t.Fatalf("evidencias=%+v, want GoalWorkResult y ACK Codex", evidencias)
	}
	goalResult := evidenciaPorGoalForTestV0(evidencias, state.GoalRef)
	if goalResult == nil || !goalResult.Terminal || !goalResult.Aceptado || goalResult.Estado != orquestagoal.GoalStatusCompleteV0 {
		t.Fatalf("GoalWorkResult debe marcar terminal aceptado validado: %+v", goalResult)
	}
	requireEvidenceRefsV0(t, goalResult.EvidenceRefs, "evidence-ref-goal-result-001", "evidence-ref-goal-result-test-001", "evidence-ref-goal-closure-001")
	ack := evidenciaPorFuenteYEstadoForTestV0(evidencias, "receipt", "completed")
	if ack == nil || ack.Terminal || ack.Aceptado || ack.GoalRef != "" {
		t.Fatalf("ACK Codex debe ser receipt no terminal: %+v", ack)
	}
	requireEvidenceRefsV0(t, ack.EvidenceRefs, "ack-ref-evidencia-receipt-001", "required-test-receipt-ref-evidencia-001")
}

func TestAgregadorEvidenciaEstadoV0CombinaFuentesSinPerderRefs(t *testing.T) {
	ctx := context.Background()
	agregador := EvidenciaEstadoAgregadorV0{
		Fuentes: []orquestaestadovivo.FuenteEvidenciaEstadoPortV0{
			fuenteEvidenciaEstadoForTestV0{evidencias: []orquestaestadovivo.EvidenciaEstadoV0{{
				RunRef:       "run-ref-evidencia-agregador-001",
				GoalRef:      "goal-ref-evidencia-agregador-001",
				Fuente:       "run_store",
				Estado:       "activa",
				EvidenceRefs: []string{"evidence-ref-agregador-run-001"},
			}}},
			nil,
			fuenteEvidenciaEstadoForTestV0{evidencias: []orquestaestadovivo.EvidenciaEstadoV0{{
				RunRef:       "run-ref-evidencia-agregador-001",
				GoalRef:      "goal-ref-evidencia-agregador-001",
				Fuente:       "receipt",
				Estado:       "complete",
				Terminal:     true,
				Aceptado:     true,
				EvidenceRefs: []string{"evidence-ref-agregador-receipt-001"},
			}}},
		},
	}

	evidencias, err := agregador.ListarEvidenciasEstadoV0(ctx, orquestaestadovivo.FiltroEvidenciaEstadoV0{
		RunRef: "run-ref-evidencia-agregador-001",
	})
	if err != nil {
		t.Fatalf("ListarEvidenciasEstadoV0: %v", err)
	}
	if len(evidencias) != 2 {
		t.Fatalf("evidencias=%+v, want 2", evidencias)
	}
	for _, evidencia := range evidencias {
		if evidencia.RunRef != "run-ref-evidencia-agregador-001" ||
			evidencia.GoalRef != "goal-ref-evidencia-agregador-001" {
			t.Fatalf("agregador perdio refs: %+v", evidencia)
		}
	}
	requireEvidenceRefsV0(t, evidencias[0].EvidenceRefs, "evidence-ref-agregador-run-001")
	requireEvidenceRefsV0(t, evidencias[1].EvidenceRefs, "evidence-ref-agregador-receipt-001")
}

func evidenciaEstadoGoalStateForTestV0(runRef, goalRef string) orquestagoal.GoalWorkStateV0 {
	return orquestagoal.GoalWorkStateV0{
		RunRef:          runRef,
		GoalRef:         goalRef,
		ExternalGoalRef: "external-" + goalRef,
		Status:          orquestagoal.GoalStatusRunningV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Objective:    "validar adaptadores de estado vivo",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet: []orquestagoal.GoalWriteScopeV0{{
				Path: "modulos/orquesta-app-codex-stack",
			}},
			EvidenceRefs: []string{"evidence-ref-spec-" + goalRef},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			Status:          orquestagoal.GoalStatusAcceptedV0,
			GoalRef:         goalRef,
			ExternalGoalRef: "external-" + goalRef,
			EvidenceRefs:    []string{"evidence-ref-launch-" + goalRef},
		},
	}
}

func mustRecordAgentProcessForEvidenciaTestV0(
	t *testing.T,
	registry *orquestaagentprocessregistrymemory.InMemoryAgentProcessRegistryV0,
	runRef string,
	agentRef string,
	processRef string,
) {
	t.Helper()
	if err := registry.RecordAgentProcessV0(context.Background(), orquestacionnucleoapp.AgentProcessRegistryRecordV0{
		RunID:          runRef,
		AgentRequestID: agentRef,
		ProcessRef:     processRef,
		SessionRef:     "session-ref-" + processRef,
		LaunchRef:      "launch-ref-" + processRef,
		ReadinessRef:   "readiness-ref-" + agentRef,
		EvidenceRefs:   []string{"evidence-ref-process-" + agentRef},
	}); err != nil {
		t.Fatalf("RecordAgentProcessV0: %v", err)
	}
}

type evidenciaEstadoSnapshotSourceForTestV0 struct {
	snapshots map[string]orquestaruntime.ProcessRuntimeSnapshotV0
	errs      map[string]error
}

func (source evidenciaEstadoSnapshotSourceForTestV0) SnapshotV0(
	processRef string,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	if err := source.errs[processRef]; err != nil {
		return orquestaruntime.ProcessRuntimeSnapshotV0{}, err
	}
	return source.snapshots[processRef], nil
}

func evidenciaEstadoCodexReceiptDescriptorForTestV0(
	t *testing.T,
	runRef string,
) orquestaruntimecodexdelivery.CodexReceiptDescriptorV0 {
	t.Helper()
	spec := orquestaruntime.ExternalAgentLaunchSpecV0{
		RequestID:     "agent-ref-evidencia-receipt-001",
		CorrelationID: "correlation-ref-evidencia-receipt-001",
		AgentPacket: orquestaruntime.AgentStartPacketV0{
			RequestID:     "agent-ref-evidencia-receipt-001",
			CorrelationID: "correlation-ref-evidencia-receipt-001",
			TargetModule:  "modulos/orquesta-app-codex-stack",
			Phase:         "programacion",
			Task: orquestaruntime.AgentStartTaskV0{
				TaskRef: "task-ref-evidencia-receipt-001",
			},
			DeliveryRefs: orquestaruntime.AgentStartDeliveryRefsV0{
				AckRef:       "ack-ref-evidencia-receipt-001",
				MailboxRef:   "mailbox-ref-evidencia-receipt-001",
				ReadinessRef: "readiness-ref-evidencia-receipt-001",
			},
		},
	}
	ackPath := filepath.Join(t.TempDir(), orquestaruntimecodex.CodexAgentAckFileNameV0)
	exitCode := 0
	outputRedacted := true
	ack := orquestaruntimecodex.CodexAgentAckV0{
		SchemaVersion: orquestaruntimecodex.CodexAgentAckSchemaVersionV0,
		RequestID:     spec.RequestID,
		CorrelationID: spec.CorrelationID,
		AckRef:        spec.AgentPacket.DeliveryRefs.AckRef,
		TargetModule:  spec.AgentPacket.TargetModule,
		TaskRef:       spec.AgentPacket.Task.TaskRef,
		Status:        "completed",
		TestReceipts: []orquestaruntimecodex.CodexRequiredTestReceiptV0{{
			SchemaVersion:  orquestaruntimecodex.CodexRequiredTestReceiptSchemaVersionV0,
			Command:        "go test -count=1 ./modulos/orquesta-app-codex-stack",
			Status:         "passed",
			ExitCode:       &exitCode,
			EvidenceRefs:   []string{"required-test-receipt-ref-evidencia-001"},
			OccurredAt:     "2026-07-03T10:30:00Z",
			Sequence:       1,
			OutputRedacted: &outputRedacted,
		}},
	}
	data, err := json.Marshal(ack)
	if err != nil {
		t.Fatalf("Marshal ack: %v", err)
	}
	if err := os.WriteFile(ackPath, data, 0o600); err != nil {
		t.Fatalf("WriteFile ack: %v", err)
	}
	return orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
		DescriptorRef: "receipt-ref-evidencia-receipt-001",
		RunID:         runRef,
		AgentRef:      spec.RequestID,
		Spec:          spec,
		AckPath:       ackPath,
	}
}

type fuenteEvidenciaEstadoForTestV0 struct {
	evidencias []orquestaestadovivo.EvidenciaEstadoV0
}

func (source fuenteEvidenciaEstadoForTestV0) ListarEvidenciasEstadoV0(
	_ context.Context,
	_ orquestaestadovivo.FiltroEvidenciaEstadoV0,
) ([]orquestaestadovivo.EvidenciaEstadoV0, error) {
	return append([]orquestaestadovivo.EvidenciaEstadoV0(nil), source.evidencias...), nil
}

func evidenciaPorFuenteYEstadoForTestV0(
	evidencias []orquestaestadovivo.EvidenciaEstadoV0,
	fuente string,
	estado string,
) *orquestaestadovivo.EvidenciaEstadoV0 {
	for index := range evidencias {
		if evidencias[index].Fuente == fuente && evidencias[index].Estado == estado {
			return &evidencias[index]
		}
	}
	return nil
}

func evidenciaPorGoalForTestV0(
	evidencias []orquestaestadovivo.EvidenciaEstadoV0,
	goalRef string,
) *orquestaestadovivo.EvidenciaEstadoV0 {
	for index := range evidencias {
		if evidencias[index].GoalRef == goalRef {
			return &evidencias[index]
		}
	}
	return nil
}

func requireEvidenceRefsV0(t *testing.T, got []string, wants ...string) {
	t.Helper()
	sort.Strings(got)
	for _, want := range wants {
		if !stringInSetV0(got, want) {
			t.Fatalf("evidence_refs=%v missing %s", got, want)
		}
	}
}
