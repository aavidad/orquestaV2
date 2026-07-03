package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestStatusServerCommandV0EsEnvelopePublicoSinAddrNiPID(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", projectDir)
	t.Setenv("ORQUESTA_SERVER_STATE_DIR", stateDir)
	t.Setenv("ORQUESTA_CODEX_RUNTIME_WORKDIR", filepath.Join(t.TempDir(), "runtime"))

	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(orquestaserver.NewServerPublicStatusV0(orquestaserver.StateV0{
			SchemaVersion:   orquestaserver.StateSchemaVersionV0,
			Status:          "running",
			PID:             98765,
			Addr:            strings.TrimPrefix(r.Host, "http://"),
			ProjectWorkDir:  projectDir,
			RuntimeWorkDir:  filepath.Join(projectDir, ".orquesta-runtime"),
			SupervisorTicks: 3,
		}))
	}))
	defer server.Close()
	saveRunStatusServerStateV0(t, stateDir, strings.TrimPrefix(server.URL, "http://"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := statusServerCommandV0(&stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit=%d stderr=%s", exitCode, stderr.String())
	}

	var envelope commandPublicOutputV0
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("json publico invalido: %v\n%s", err, stdout.String())
	}
	if envelope.SchemaVersion != commandPublicOutputSchemaVersionV0 ||
		envelope.Command != "status" ||
		envelope.Freshness != commandPublicFreshnessLiveV0 ||
		envelope.RedactionLevel != commandPublicRedactionLevelV0 {
		t.Fatalf("envelope=%+v", envelope)
	}
	if strings.Contains(stdout.String(), server.URL) ||
		strings.Contains(stdout.String(), strings.TrimPrefix(server.URL, "http://")) ||
		strings.Contains(stdout.String(), projectDir) ||
		strings.Contains(stdout.String(), "98765") {
		t.Fatalf("stdout filtra detalle local: %s", stdout.String())
	}
}

func TestStatusServerCommandV0FallbackStatefileNoPublicaEstadoCrudo(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", projectDir)
	t.Setenv("ORQUESTA_SERVER_STATE_DIR", stateDir)
	t.Setenv("ORQUESTA_CODEX_RUNTIME_WORKDIR", filepath.Join(t.TempDir(), "runtime"))

	store, err := orquestaserver.NewFileStateStoreV0(filepath.Join(stateDir, orquestaserver.DefaultStateFileV0))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	if err := store.SaveServerStateV0(context.Background(), orquestaserver.StateV0{
		Status:         "running",
		PID:            os.Getpid(),
		Addr:           "127.0.0.1:1",
		ProjectWorkDir: projectDir,
		RuntimeWorkDir: filepath.Join(projectDir, ".orquesta-runtime"),
		LastError:      "open " + filepath.Join(projectDir, "state.json") + ": permission denied",
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := statusServerCommandV0(&stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit=%d stderr=%s", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"command":"status"`) ||
		!strings.Contains(stdout.String(), `"status":"degraded"`) {
		t.Fatalf("stdout sin envelope publico: %s", stdout.String())
	}
	if strings.Contains(stdout.String(), projectDir) ||
		strings.Contains(stdout.String(), "45678") ||
		strings.Contains(stdout.String(), "127.0.0.1:1") {
		t.Fatalf("stdout filtra state crudo: %s", stdout.String())
	}
}

func TestStatusServerCommandV0ReconciliaStatefileConPIDMuerto(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", projectDir)
	t.Setenv("ORQUESTA_SERVER_STATE_DIR", stateDir)
	t.Setenv("ORQUESTA_CODEX_RUNTIME_WORKDIR", filepath.Join(t.TempDir(), "runtime"))

	deadPID := 99999999
	if processAliveV0(deadPID) {
		t.Skip("pid de prueba vivo en este sistema")
	}
	store, err := orquestaserver.NewFileStateStoreV0(filepath.Join(stateDir, orquestaserver.DefaultStateFileV0))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	if err := store.SaveServerStateV0(context.Background(), orquestaserver.StateV0{
		Status:               "running",
		PID:                  deadPID,
		Addr:                 "127.0.0.1:1",
		ProjectWorkDir:       projectDir,
		RuntimeWorkDir:       filepath.Join(projectDir, ".orquesta-runtime"),
		LastHeartbeatAt:      "2026-07-01T00:00:00Z",
		StartupReady:         true,
		StartupStatus:        "startup_ready",
		LastSupervisorStatus: "ok",
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := statusServerCommandV0(&stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit=%d stderr=%s", exitCode, stderr.String())
	}
	var envelope commandPublicOutputV0
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("json publico invalido: %v\n%s", err, stdout.String())
	}
	if envelope.Status != "stale" ||
		envelope.Freshness != commandPublicFreshnessSnapshotV0 ||
		envelope.DiagnosticsMode != "statefile_snapshot_reconciled_process_not_alive" {
		t.Fatalf("envelope no reconciliado: %+v", envelope)
	}
	var payload orquestaserver.ServerPublicStatusV0
	rawPayload, err := json.Marshal(envelope.Payload)
	if err != nil {
		t.Fatalf("payload marshal: %v", err)
	}
	if err := json.Unmarshal(rawPayload, &payload); err != nil {
		t.Fatalf("payload publico invalido: %v\n%+v", err, envelope.Payload)
	}
	if payload.AvailabilityStatus != "crashed" ||
		payload.AvailabilityReason != "server_crashed_after_readiness" {
		t.Fatalf("payload no expone disponibilidad accionable: %+v", payload)
	}
	reconciled, err := store.LoadServerStateV0(context.Background())
	if err != nil {
		t.Fatalf("load reconciled: %v", err)
	}
	if reconciled.Status != "stale" ||
		reconciled.StartupReady ||
		reconciled.StartupStatus != orquestaserver.ServerProcessStaleReasonCodeV0 ||
		reconciled.LastHeartbeatAt != "2026-07-01T00:00:00Z" ||
		reconciled.LastError != "server_process_not_alive" {
		t.Fatalf("state no reconciliado: %+v", reconciled)
	}
	if len(reconciled.RecentErrors) == 0 ||
		reconciled.RecentErrors[0].Code != orquestaserver.ServerProcessStaleReasonCodeV0 {
		t.Fatalf("recent_errors sin stale: %+v", reconciled.RecentErrors)
	}
	if strings.Contains(stdout.String(), projectDir) ||
		strings.Contains(stdout.String(), "127.0.0.1:1") ||
		strings.Contains(stdout.String(), "99999999") {
		t.Fatalf("stdout filtra detalle local: %s", stdout.String())
	}
}

func TestStatusServerCommandV0ReconciliaStoppedConStartupReadyHeredado(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", projectDir)
	t.Setenv("ORQUESTA_SERVER_STATE_DIR", stateDir)
	t.Setenv("ORQUESTA_CODEX_RUNTIME_WORKDIR", filepath.Join(t.TempDir(), "runtime"))

	store, err := orquestaserver.NewFileStateStoreV0(filepath.Join(stateDir, orquestaserver.DefaultStateFileV0))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	if err := store.SaveServerStateV0(context.Background(), orquestaserver.StateV0{
		Status:         "stopped",
		PID:            os.Getpid(),
		Addr:           "127.0.0.1:18787",
		ProjectWorkDir: projectDir,
		RuntimeWorkDir: filepath.Join(projectDir, ".orquesta-runtime"),
		StartupReady:   false,
		StartupStatus:  "stopped",
		StartupMessage: "director: orquesta preparada",
		StartupOperationalMessage: &orquestaserver.ServerOperationalMessageV0{
			SchemaVersion: "orquesta_server_operational_message.v0",
			Scope:         "startup",
			ReasonCode:    "startup_ready",
			Status:        "startup_ready",
			Message:       "director: orquesta preparada",
		},
		StartupEvidenceRefs: []string{"evidence-ref-startup-ready-stale"},
		LastHeartbeatAt:     "2026-07-02T22:33:36Z",
		ShutdownReady:       true,
		ShutdownStatus:      "stopped",
		LastStartupCheckAt:  "2026-07-02T22:33:36Z",
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := statusServerCommandV0(&stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit=%d stderr=%s", exitCode, stderr.String())
	}
	var envelope commandPublicOutputV0
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("json publico invalido: %v\n%s", err, stdout.String())
	}
	if envelope.Status != "stopped" ||
		envelope.Freshness != commandPublicFreshnessSnapshotV0 ||
		envelope.DiagnosticsMode != "statefile_snapshot_reconciled" {
		t.Fatalf("envelope no reconciliado: %+v", envelope)
	}
	var payload orquestaserver.ServerPublicStatusV0
	rawPayload, err := json.Marshal(envelope.Payload)
	if err != nil {
		t.Fatalf("payload marshal: %v", err)
	}
	if err := json.Unmarshal(rawPayload, &payload); err != nil {
		t.Fatalf("payload publico invalido: %v\n%+v", err, envelope.Payload)
	}
	if payload.StartupReady ||
		payload.StartupStatus != "stopped" ||
		payload.StartupMessage != "" ||
		payload.StartupOperationalMessage != nil {
		t.Fatalf("payload conserva startup heredado: %+v", payload)
	}
	reconciled, err := store.LoadServerStateV0(context.Background())
	if err != nil {
		t.Fatalf("load reconciled: %v", err)
	}
	if reconciled.Status != "stopped" ||
		reconciled.StartupReady ||
		reconciled.StartupStatus != "stopped" ||
		reconciled.StartupMessage != "" ||
		reconciled.StartupOperationalMessage != nil ||
		len(reconciled.StartupEvidenceRefs) != 0 ||
		reconciled.LastHeartbeatAt != "2026-07-02T22:33:36Z" {
		t.Fatalf("state no reconciliado: %+v", reconciled)
	}
	if strings.Contains(stdout.String(), projectDir) ||
		strings.Contains(stdout.String(), "127.0.0.1:18787") {
		t.Fatalf("stdout filtra detalle local: %s", stdout.String())
	}
}

func TestStatusServerCommandV0FalloStdoutDevuelveReasonCode(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", projectDir)
	t.Setenv("ORQUESTA_SERVER_STATE_DIR", stateDir)
	t.Setenv("ORQUESTA_CODEX_RUNTIME_WORKDIR", filepath.Join(t.TempDir(), "runtime"))

	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(orquestaserver.NewServerPublicStatusV0(orquestaserver.StateV0{
			SchemaVersion:  orquestaserver.StateSchemaVersionV0,
			Status:         "running",
			Addr:           strings.TrimPrefix(r.Host, "http://"),
			ProjectWorkDir: projectDir,
		}))
	}))
	defer server.Close()
	saveRunStatusServerStateV0(t, stateDir, strings.TrimPrefix(server.URL, "http://"))

	var stderr bytes.Buffer
	exitCode := statusServerCommandV0(failingCommandWriterV0{}, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit=%d stderr=%s", exitCode, stderr.String())
	}
	if !strings.Contains(stderr.String(), "command_output_write_failed") ||
		strings.Contains(stderr.String(), projectDir) {
		t.Fatalf("stderr sin fallo redactado: %s", stderr.String())
	}
}

func TestCommandPublicOPESDrainPayloadV0RedactaURLsYCodigosExternos(t *testing.T) {
	payload := commandPublicOPESDrainPayloadV0(opesDrainSummaryV0{
		OPESBaseURL:     "http://127.0.0.1:18080/private",
		OrquestaBaseURL: "http://127.0.0.1:8787",
		Limit:           1,
		JobType:         "plan_temario",
		Destination: opesDrainDestinationPolicyV0{
			OPESDestination:        opesDrainDestinationV0{Category: "loopback", URLRef: "opes-base-url-ref-abc"},
			OrquestaDestination:    opesDrainDestinationV0{Category: "loopback", URLRef: "orquesta-base-url-ref-def"},
			DestinationEvidenceRef: "opes-destination-evidence-ref-abc",
		},
		Seen: 1,
		Errors: []opesDrainPublicErrorV0{{
			JobRef: "job-ref-001",
			Code:   `Get "http://127.0.0.1/private": token=abc`,
		}},
	})

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if payload.OPESBaseURLRef == "" ||
		payload.OrquestaBaseURLRef == "" ||
		payload.OPESDestination != "loopback" ||
		payload.OrquestaDestination != "loopback" ||
		payload.DestinationEvidenceRef == "" ||
		payload.Filter.Mode != "job_type" ||
		len(payload.ErrorCodes) != 1 ||
		payload.ErrorCodes[0] != "external_error" {
		t.Fatalf("payload=%+v", payload)
	}
	if strings.Contains(string(data), "127.0.0.1") ||
		strings.Contains(string(data), "token=abc") {
		t.Fatalf("payload filtra detalle local: %s", string(data))
	}
}

type failingCommandWriterV0 struct{}

func (failingCommandWriterV0) Write(_ []byte) (int, error) {
	return 0, errCommandWriterFailedV0{}
}

type errCommandWriterFailedV0 struct{}

func (errCommandWriterFailedV0) Error() string {
	return "writer_failed_private_path_/tmp/orquesta"
}
