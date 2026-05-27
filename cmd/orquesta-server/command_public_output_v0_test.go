package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
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
		PID:            45678,
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
