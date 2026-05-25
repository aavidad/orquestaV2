package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCodexWaveTailReportV0SummaryRedactaYAcota(t *testing.T) {
	root := t.TempDir()
	runtimeDir := filepath.Join(root, "runtime", "wave-tail")
	agentDir := filepath.Join(runtimeDir, "agent-01")
	if err := os.MkdirAll(agentDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	stdoutPath := filepath.Join(agentDir, "codex_stdout.log")
	if err := os.WriteFile(stdoutPath, []byte("linea 1\naccess_token=secreto\n/home/alberto/x\n"), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	report, err := codexWaveBuildTailReportV0(codexWaveLaunchSummaryV0{
		WaveRef:        "wave-tail",
		RuntimeWorkDir: runtimeDir,
		Agents: []codexWaveAgentSummaryV0{{
			AgentRef:       "wave-tail-agent-01",
			RuntimeWorkDir: agentDir,
			StdoutPath:     stdoutPath,
			Status:         "stopped",
		}},
	}, codexWaveControlConfigV0{
		RuntimeWorkDir: runtimeDir,
		LogKind:        "stdout",
		Mode:           "summary",
		Reason:         "diagnostico-test",
		Lines:          2,
		MaxBytes:       64,
	})
	if err != nil {
		t.Fatalf("tail report: %v", err)
	}
	if len(report.Agents) != 1 || len(report.Agents[0].Fragment) != 0 {
		t.Fatalf("summary no debe incluir fragmento: %+v", report)
	}
	if !report.Agents[0].RedactionApplied || report.Agents[0].LinesReturned != 2 {
		t.Fatalf("redaccion/limite inesperado: %+v", report.Agents[0])
	}
}

func TestCodexWaveTailReportV0ExigeRazon(t *testing.T) {
	_, err := codexWaveBuildTailReportV0(codexWaveLaunchSummaryV0{}, codexWaveControlConfigV0{})
	if err == nil {
		t.Fatalf("err nil")
	}
}
