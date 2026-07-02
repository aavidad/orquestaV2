package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodexWaveCodeHomeProjectionV0DesactivaCodebaseMemoryMCPSinOptInCentral(t *testing.T) {
	root := t.TempDir()
	sourceCodeHome := filepath.Join(root, "source-codex-home")
	runtimeDir := filepath.Join(root, "runtime", "agent-01")

	mustMkdirV0(t, sourceCodeHome)
	mustWriteV0(t, filepath.Join(sourceCodeHome, "auth.json"), `{"ok":true}`, 0o600)
	mustWriteV0(t, filepath.Join(sourceCodeHome, "AGENTS.md"), "# Base\n", 0o600)
	mustWriteV0(t, filepath.Join(sourceCodeHome, "config.toml"), strings.Join([]string{
		`model = "test"`,
		``,
		`[mcp_servers.codebase-memory-mcp]`,
		`command = "codebase-memory-mcp"`,
		`args = ["--repo", "."]`,
		``,
		`[mcp_servers.codebase-memory-mcp.env]`,
		`ORQUESTA = "1"`,
		``,
		`[mcp_servers.other]`,
		`command = "other-mcp"`,
		``,
	}, "\n"), 0o600)

	_, codeHomeDir, receipt, err := codexWaveCopyCodeHomeV0(
		sourceCodeHome,
		runtimeDir,
		codexWaveCredentialProjectionPolicyWithBoundsV0(true, false, 20, 4096, 65536),
	)
	if err != nil {
		t.Fatalf("codexWaveCopyCodeHomeV0: %v", err)
	}
	configRaw := mustReadStringV0(t, filepath.Join(codeHomeDir, "config.toml"))
	if strings.Contains(configRaw, "codebase-memory-mcp") ||
		strings.Contains(configRaw, "mcp_servers.codebase-memory-mcp") {
		t.Fatalf("config conserva codebase-memory-mcp sin opt-in:\n%s", configRaw)
	}
	if !strings.Contains(configRaw, `[mcp_servers.other]`) ||
		!strings.Contains(configRaw, `command = "other-mcp"`) ||
		!strings.Contains(configRaw, `model = "test"`) {
		t.Fatalf("config filtro contenido no relacionado:\n%s", configRaw)
	}
	agentsRaw := mustReadStringV0(t, filepath.Join(codeHomeDir, "AGENTS.md"))
	if !strings.Contains(agentsRaw, "# Base") ||
		!strings.Contains(agentsRaw, codexWaveCodebaseMemoryMCPGuardStartV0) ||
		!strings.Contains(agentsRaw, "direct_mcp=true") {
		t.Fatalf("AGENTS.md no contiene guarda Orquesta:\n%s", agentsRaw)
	}
	if receipt == nil || !hasProjectionOmissionReasonForTestV0(receipt.Omissions, "codebase_memory_mcp_disabled_by_default") {
		t.Fatalf("receipt no registra saneado de config: %+v", receipt)
	}
}

func TestCodexWaveCodeHomeProjectionV0DesactivaCodebaseMemoryMCPConEnabledSinOptInDirecto(t *testing.T) {
	root := t.TempDir()
	sourceCodeHome := filepath.Join(root, "source-codex-home")
	runtimeDir := filepath.Join(root, "runtime", "agent-01")

	mustMkdirV0(t, filepath.Join(sourceCodeHome, "log"))
	mustWriteV0(t, filepath.Join(sourceCodeHome, "auth.json"), `{"ok":true}`, 0o600)
	mustWriteV0(t, filepath.Join(sourceCodeHome, "config.toml"), strings.Join([]string{
		`model = "test"`,
		``,
		`[mcp_servers.codebase-memory-mcp]`,
		`command = "codebase-memory-mcp"`,
		``,
	}, "\n"), 0o600)
	mustWriteV0(t, filepath.Join(sourceCodeHome, codexWaveAgentToolingLedgerPathV0), strings.Join([]string{
		`tool=codebase-memory-mcp`,
		`enabled=1`,
		`ui=false`,
		`direct_mcp=false`,
		`broker_policy=central_only`,
		``,
	}, "\n"), 0o600)

	_, codeHomeDir, receipt, err := codexWaveCopyCodeHomeV0(
		sourceCodeHome,
		runtimeDir,
		codexWaveCredentialProjectionPolicyWithBoundsV0(true, false, 20, 4096, 65536),
	)
	if err != nil {
		t.Fatalf("codexWaveCopyCodeHomeV0: %v", err)
	}
	configRaw := mustReadStringV0(t, filepath.Join(codeHomeDir, "config.toml"))
	if strings.Contains(configRaw, "codebase-memory-mcp") {
		t.Fatalf("config conserva codebase-memory-mcp con enabled pero sin direct_mcp=true:\n%s", configRaw)
	}
	if receipt == nil || !hasProjectionOmissionReasonForTestV0(receipt.Omissions, "codebase_memory_mcp_disabled_by_default") {
		t.Fatalf("receipt no registra saneado sin opt-in directo: %+v", receipt)
	}
}

func TestCodexWaveCodeHomeProjectionV0RespetaCodebaseMemoryMCPConOptInDirecto(t *testing.T) {
	root := t.TempDir()
	sourceCodeHome := filepath.Join(root, "source-codex-home")
	runtimeDir := filepath.Join(root, "runtime", "agent-01")

	mustMkdirV0(t, filepath.Join(sourceCodeHome, "log"))
	mustWriteV0(t, filepath.Join(sourceCodeHome, "auth.json"), `{"ok":true}`, 0o600)
	mustWriteV0(t, filepath.Join(sourceCodeHome, "config.toml"), strings.Join([]string{
		`model = "test"`,
		``,
		`[mcp_servers.codebase-memory-mcp]`,
		`command = "codebase-memory-mcp"`,
		``,
	}, "\n"), 0o600)
	mustWriteV0(t, filepath.Join(sourceCodeHome, codexWaveAgentToolingLedgerPathV0), strings.Join([]string{
		`tool=codebase-memory-mcp`,
		`enabled=1`,
		`ui=false`,
		`direct_mcp=true`,
		`broker_policy=central_only`,
		``,
	}, "\n"), 0o600)

	_, codeHomeDir, receipt, err := codexWaveCopyCodeHomeV0(
		sourceCodeHome,
		runtimeDir,
		codexWaveCredentialProjectionPolicyWithBoundsV0(true, false, 20, 4096, 65536),
	)
	if err != nil {
		t.Fatalf("codexWaveCopyCodeHomeV0: %v", err)
	}
	configRaw := mustReadStringV0(t, filepath.Join(codeHomeDir, "config.toml"))
	if !strings.Contains(configRaw, "codebase-memory-mcp") {
		t.Fatalf("config no conserva opt-in directo:\n%s", configRaw)
	}
	if receipt == nil || hasProjectionOmissionReasonForTestV0(receipt.Omissions, "codebase_memory_mcp_disabled_by_default") {
		t.Fatalf("receipt no debe registrar saneado con opt-in: %+v", receipt)
	}
	agentsRaw := mustReadStringV0(t, filepath.Join(codeHomeDir, "AGENTS.md"))
	if !strings.Contains(agentsRaw, codexWaveCodebaseMemoryMCPGuardStartV0) {
		t.Fatalf("AGENTS.md no contiene guarda informativa:\n%s", agentsRaw)
	}
}

func mustReadStringV0(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(raw)
}
