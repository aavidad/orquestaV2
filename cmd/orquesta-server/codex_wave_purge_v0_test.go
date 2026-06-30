package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodexWavePurgeRuntimeRequiresExplicitConfirmationV0(t *testing.T) {
	config := testCodexWavePurgeConfigV0(t, "purge-confirm-required")
	config.PurgeRuntime = true
	config.PurgeConfirm = ""

	if _, err := runCodexLaunchWaveV0(context.Background(), config); err == nil ||
		!strings.Contains(err.Error(), "runtime_purge_confirmation_required") {
		t.Fatalf("purga sin confirmacion no bloqueada: %v", err)
	}
}

func TestCodexWavePurgeRuntimeRejectsRuntimeOutsideAllowedRootV0(t *testing.T) {
	config := testCodexWavePurgeConfigV0(t, "outside-root")
	config.RuntimeWorkDir = filepath.Join(t.TempDir(), "codex-waves", "outside-root")
	config.PurgeRuntime = true
	config.PurgeConfirm = config.WaveRef
	if err := os.MkdirAll(config.RuntimeWorkDir, 0o700); err != nil {
		t.Fatalf("crear runtime: %v", err)
	}

	if _, err := runCodexLaunchWaveV0(context.Background(), config); err == nil ||
		!strings.Contains(err.Error(), "runtime_dir_purge_root_not_allowed") {
		t.Fatalf("runtime fuera de raiz permitida no bloqueado: %v", err)
	}
}

func TestCodexWavePurgeRuntimeReportOnlyKeepsFilesAndCountsManifestV0(t *testing.T) {
	config := testCodexWavePurgeConfigV0(t, "report-only")
	config.PurgeRuntime = true
	config.PurgeConfirm = config.WaveRef
	config.PurgeReportOnly = true
	stale := filepath.Join(config.RuntimeWorkDir, "stale.txt")
	if err := os.MkdirAll(config.RuntimeWorkDir, 0o700); err != nil {
		t.Fatalf("crear runtime: %v", err)
	}
	if err := os.WriteFile(stale, []byte("stale"), 0o600); err != nil {
		t.Fatalf("crear stale: %v", err)
	}

	summary, err := runCodexLaunchWaveV0(context.Background(), config)
	if err != nil {
		t.Fatalf("purge report: %v", err)
	}
	if summary.PurgeReport == nil ||
		summary.PurgeReport.Status != "report_only" ||
		summary.PurgeReport.FileCount == 0 ||
		strings.Contains(codexWavePurgeReportJSONV0(summary.PurgeReport), config.RuntimeWorkDir) {
		t.Fatalf("reporte de purga inseguro: %+v", summary.PurgeReport)
	}
	if _, err := os.Stat(stale); err != nil {
		t.Fatalf("report-only borro archivo: %v", err)
	}
}

func TestCodexWavePurgeRuntimeBlocksLiveWorkEvidenceV0(t *testing.T) {
	config := testCodexWavePurgeConfigV0(t, "live-work")
	config.PurgeRuntime = true
	config.PurgeConfirm = config.WaveRef
	ack := filepath.Join(config.RuntimeWorkDir, "agent-01", "agent_ack.json")
	decision := filepath.Join(config.RuntimeWorkDir, "agent-01", "director_decisions.json")
	checkpoint := filepath.Join(config.RuntimeWorkDir, "agent-01", "orquesta_shutdown_request.json")
	for _, path := range []string{ack, decision, checkpoint} {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatalf("crear dir: %v", err)
		}
		if err := os.WriteFile(path, []byte("{}"), 0o600); err != nil {
			t.Fatalf("crear control: %v", err)
		}
	}

	summary, err := runCodexLaunchWaveV0(context.Background(), config)
	if err == nil || summary.PurgeReport == nil || summary.PurgeReport.Status != "blocked" {
		t.Fatalf("purga no bloqueada: summary=%+v err=%v", summary, err)
	}
	for _, want := range []string{"agent_ack_unreconciled", "director_decisions_pending", "checkpoint_pending"} {
		if !containsCodexWavePurgeStringV0(summary.PurgeReport.BlockedBy, want) {
			t.Fatalf("bloqueo %s ausente: %+v", want, summary.PurgeReport)
		}
	}
	if _, err := os.Stat(ack); err != nil {
		t.Fatalf("purga bloqueada borro ack: %v", err)
	}
}

func TestCodexWavePurgeRuntimeBlocksDurablePlansAndArtifactManifestsV0(t *testing.T) {
	config := testCodexWavePurgeConfigV0(t, "durable-plan")
	config.PurgeRuntime = true
	config.PurgeConfirm = config.WaveRef
	if err := os.MkdirAll(config.RuntimeWorkDir, 0o700); err != nil {
		t.Fatalf("crear runtime: %v", err)
	}
	if err := os.WriteFile(filepath.Join(config.RuntimeWorkDir, "plan.json"), []byte(`{"schema_version":"orquesta_plan.v0"}`), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	if err := os.WriteFile(filepath.Join(config.RuntimeWorkDir, "artifacts_manifest.json"), []byte(`{"artifacts":[]}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	summary, err := runCodexLaunchWaveV0(context.Background(), config)

	if err == nil || summary.PurgeReport == nil || summary.PurgeReport.Status != "blocked" {
		t.Fatalf("purga no bloqueada: summary=%+v err=%v", summary.PurgeReport, err)
	}
	if !containsCodexWavePurgeStringV0(summary.PurgeReport.BlockedBy, "durable_plan_or_artifact_manifest") {
		t.Fatalf("bloqueo de plan/manifiesto ausente: %+v", summary.PurgeReport)
	}
	if _, statErr := os.Stat(config.RuntimeWorkDir); statErr != nil {
		t.Fatalf("runtime durable borrado: %v", statErr)
	}
}

func TestCodexWavePurgeRuntimeBlocksRunningRegistryV0(t *testing.T) {
	config := testCodexWavePurgeConfigV0(t, "running-registry")
	config.PurgeRuntime = true
	config.PurgeConfirm = config.WaveRef
	if err := os.MkdirAll(config.RuntimeWorkDir, 0o700); err != nil {
		t.Fatalf("crear runtime: %v", err)
	}
	registry := codexWaveLaunchSummaryV0{
		SchemaVersion:  codexWaveSummarySchemaVersionV0,
		WaveRef:        config.WaveRef,
		RuntimeWorkDir: config.RuntimeWorkDir,
		RegistryPath:   codexWaveRegistryPathV0(config.RuntimeWorkDir),
		Agents: []codexWaveAgentSummaryV0{{
			AgentRef: "running-registry-agent-01",
			PID:      os.Getpid(),
			Status:   "running",
		}},
	}
	if err := codexWaveSaveRegistryV0(registry); err != nil {
		t.Fatalf("guardar registry: %v", err)
	}

	summary, err := runCodexLaunchWaveV0(context.Background(), config)
	if err == nil || summary.PurgeReport == nil ||
		!containsCodexWavePurgeStringV0(summary.PurgeReport.BlockedBy, "agent_live") {
		t.Fatalf("agente vivo no bloqueo purga: summary=%+v err=%v", summary, err)
	}
}

func testCodexWavePurgeConfigV0(t *testing.T, waveRef string) codexWaveConfigV0 {
	t.Helper()
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	runtimeDir := filepath.Join(projectDir, ".orquesta-runtime", "codex-waves", waveRef)
	fakeCommand := filepath.Join(root, "codex-fake")
	if err := os.MkdirAll(projectDir, 0o700); err != nil {
		t.Fatalf("crear project: %v", err)
	}
	if err := os.WriteFile(fakeCommand, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatalf("crear command: %v", err)
	}
	return codexWaveConfigV0{
		Agents:         1,
		WaveRef:        waveRef,
		Prompt:         "diagnostico de purga",
		ProjectWorkDir: projectDir,
		RuntimeWorkDir: runtimeDir,
		CommandPath:    fakeCommand,
		DryRun:         true,
	}
}

func containsCodexWavePurgeStringV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
