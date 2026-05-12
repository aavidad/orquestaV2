package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func TestCodexStackRealSmokeDetectaACKFaltanteConProyectoCompilable(t *testing.T) {
	runtime := newCompilingAppMissingACKRuntimeV0("task-ref-stack-agenda-http")
	stack := mustBuildCodexStackForTestV0(t, runtime)
	director := postDirectorAPIV0(t, stack)

	result, err := codexStackRealSmokeDrainUntilProgrammingDeliveredResultV0(
		t,
		context.Background(),
		stack,
		smokeDrainStoresFromStackForTestV0(t, stack),
		director.RunRef,
		1,
	)
	if err == nil {
		t.Fatalf("expected causal missing ACK failure, got run=%+v", result.Run)
	}
	if !strings.Contains(err.Error(), "project_compiles_but_ack_missing") ||
		!strings.Contains(err.Error(), "task-ref-stack-agenda-http") {
		t.Fatalf("error no identifica causa ACK/proyecto compilable: %v", err)
	}
	if len(codexStackRealSmokeProgrammingReceiptDescriptorsV0(result.Descriptors)) < 3 {
		t.Fatalf("programming descriptors=%v", codexStackRealSmokeProgrammingDescriptorsV0(result.Descriptors))
	}
	codexStackRealSmokeAssertProgrammingParallelWaveV0(
		t,
		smokeDrainStoresFromStackForTestV0(t, stack).EventSink.EventsV0(),
		result.Descriptors,
		2,
	)
}

type compilingAppMissingACKRuntimeV0 struct {
	*fakeCodexStackRuntimeV0
	missingTaskRef string
}

func newCompilingAppMissingACKRuntimeV0(missingTaskRef string) *compilingAppMissingACKRuntimeV0 {
	return &compilingAppMissingACKRuntimeV0{
		fakeCodexStackRuntimeV0: newFakeCodexStackRuntimeV0(),
		missingTaskRef:          missingTaskRef,
	}
}

func (runtime *compilingAppMissingACKRuntimeV0) LaunchV0(
	ctx context.Context,
	req orquestaruntime.ProcessRuntimeLaunchRequestV0,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	runtimeDir := filepath.Dir(req.CommandPath)
	packet, err := codexStackPacketFromRuntimeDirForTestV0(runtimeDir)
	if err != nil {
		return orquestaruntime.ProcessRuntimeSnapshotV0{}, err
	}
	if packet.TargetModule != "orquesta-app-stack-programacion" {
		snapshot, err := runtime.fakeCodexStackRuntimeV0.LaunchV0(ctx, req)
		if err != nil || packet.TargetModule != "orquesta-app-stack-director" {
			return snapshot, err
		}
		return snapshot, writeBatchDirectorDecisionsForRuntimeDirV0(runtimeDir, packet)
	}
	snapshot, err := runtime.launchSnapshotWithoutACKV0("process-ref-app-stack-programming-missing-ack-")
	if err != nil {
		return snapshot, err
	}
	if err := writeCompilingGoAppForMissingACKSmokeV0(req.WorkingDir); err != nil {
		return snapshot, err
	}
	if packet.Task.TaskRef == runtime.missingTaskRef {
		return snapshot, nil
	}
	return snapshot, writeCodexStackACKForPacketV0(runtimeDir, packet)
}

func writeCodexStackACKForPacketV0(
	runtimeDir string,
	packet orquestaruntime.AgentStartPacketV0,
) error {
	data, err := json.Marshal(map[string]any{
		"schema_version": "codex_agent_ack.v0",
		"request_id":     packet.RequestID,
		"correlation_id": packet.CorrelationID,
		"ack_ref":        packet.DeliveryRefs.AckRef,
		"target_module":  packet.TargetModule,
		"task_ref":       packet.Task.TaskRef,
		"status":         "completed",
		"files":          packet.Task.WriteSet,
		"tests":          packet.Task.RequiredTests,
		"notes":          []string{"ack de prueba para tareas completas"},
	})
	if err != nil {
		return err
	}
	return os.WriteFile(
		filepath.Join(runtimeDir, orquestaruntimecodex.CodexAgentAckFileNameV0),
		data,
		0o600,
	)
}

func writeCompilingGoAppForMissingACKSmokeV0(projectDir string) error {
	files := map[string]string{
		"go.mod":                    "module agenda\n\ngo 1.22\n",
		"cmd/server/main.go":        "package main\n\nfunc main() {}\n",
		"internal/config/config.go": "package config\n\ntype Config struct{}\n",
		"internal/domain/placeholder.go": strings.Join([]string{
			"package domain",
			"",
			"type Placeholder struct{}",
			"",
		}, "\n"),
		"internal/application/service.go": "package application\n\ntype Service struct{}\n",
		"internal/i18n/messages.go":       "package i18n\n\nfunc DefaultLocale() string { return \"es-ES\" }\n",
		"internal/http/router.go":         "package httpapi\n\nfunc RouterName() string { return \"router\" }\n",
		"internal/http/handlers.go":       "package httpapi\n\nfunc HandlerName() string { return \"handler\" }\n",
		"internal/security/policy.go":     "package security\n\nfunc Enabled() bool { return true }\n",
		"README.md":                       "Agenda API Web\n\nSmoke fixture with a compact compilable Go project.\n",
		"docs/operacion.md":               "Operacion\n\nEjecutar go test ./... antes de entregar.\n",
		"web/static/index.html":           "<!doctype html><html><body><main>Agenda</main></body></html>\n",
		"web/static/app.js":               "console.log('agenda');\n",
		"web/static/styles.css":           "body { font-family: sans-serif; }\n",
	}
	for relPath, data := range files {
		fullPath := filepath.Join(projectDir, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(fullPath, []byte(data), 0o600); err != nil {
			return err
		}
	}
	return nil
}
