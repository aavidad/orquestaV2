package main

import (
	"path/filepath"
	"strings"
	"testing"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestServerDaemonStartEnvironmentV0UsaAllowlistYDerivaRuntime(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".orquesta-runtime")
	config := orquestaserver.ConfigV0{
		Addr:           "127.0.0.1:19091",
		ProjectWorkDir: projectDir,
		RuntimeWorkDir: runtimeDir,
		StateDir:       filepath.Join(projectDir, "state"),
		AuditFile:      "audit.jsonl",
	}
	parent := []string{
		"HOME=/home/alberto",
		"PATH=/tmp/bin",
		"GITHUB_TOKEN=secret",
		"ORQUESTA_SERVER_MAX_RUNS_PER_TICK=3",
		"ORQUESTA_CODEX_MODEL=model-ref-local",
		"ORQUESTA_CODEX_COMMAND=/usr/bin/codex",
	}

	got := serverDaemonStartEnvironmentV0(parent, config)
	for _, forbidden := range []string{"HOME=", "PATH=", "GITHUB_TOKEN="} {
		if daemonStartEnvHasKeyForTestV0(got, strings.TrimSuffix(forbidden, "=")) {
			t.Fatalf("env proyecto contiene %s: %v", forbidden, got)
		}
	}
	body := strings.Join(got, "\n")
	for _, want := range []string{
		"ORQUESTA_SERVER_MAX_RUNS_PER_TICK=3",
		"ORQUESTA_CODEX_MODEL=model-ref-local",
		"ORQUESTA_CODEX_COMMAND=/usr/bin/codex",
		"ORQUESTA_CODEX_PROJECT_WORKDIR=" + projectDir,
		"ORQUESTA_CODEX_RUNTIME_WORKDIR=" + runtimeDir,
		envStartupCleanupModeV0 + "=" + startupCleanupModeDiagnoseV0,
		envRailsModeV0 + "=" + railsModeServerDefaultV0,
		detailProhibitedRailsEnvV0 + "=" + detailProhibitedRailsServerDefaultV0,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("env falta %q en %v", want, got)
		}
	}
}

func daemonStartEnvHasKeyForTestV0(env []string, key string) bool {
	for _, item := range env {
		got, _, ok := strings.Cut(item, "=")
		if ok && got == key {
			return true
		}
	}
	return false
}

func TestServerDaemonStartEnvPolicyV0PublicaCategoriasSinValoresCrudos(t *testing.T) {
	projectDir := t.TempDir()
	config := orquestaserver.ConfigV0{
		Addr:           "127.0.0.1:19092",
		ProjectWorkDir: projectDir,
		RuntimeWorkDir: filepath.Join(projectDir, "runtime"),
		StateDir:       filepath.Join(projectDir, "state"),
		AuditFile:      "audit.jsonl",
	}
	parent := []string{
		"ORQUESTA_CODEX_HOME=/home/alberto/.codex-home",
		"ORQUESTA_SERVER_CONTROL_TOKEN=secret-control",
		"ORQUESTA_DOMAIN_WORK_HTTP_BASE_URL=http://127.0.0.1:19093",
		"ORQUESTA_PRIVATE_API_KEY=blocked",
	}

	policy := serverDaemonStartEnvPolicyV0(parent, config)
	settings := daemonStartEnvSettingsV0(policy)
	body := strings.TrimSpace(effectiveSettingValueForTestV0(settings, daemonStartEnvCategoriesSettingV0) + "\n" +
		effectiveSettingValueForTestV0(settings, daemonStartEnvCountsSettingV0) + "\n" +
		effectiveSettingValueForTestV0(settings, daemonStartEnvIssuesSettingV0))

	for _, forbidden := range []string{"/home/alberto", "secret-control", "blocked", "127.0.0.1:19093"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("policy expone valor crudo %q: %s", forbidden, body)
		}
	}
	for _, want := range []string{"codex_runtime", "domain_work", "server"} {
		if !strings.Contains(body, want) {
			t.Fatalf("policy falta categoria %q: %s", want, body)
		}
	}
	if !strings.Contains(body, "projected=") || !strings.Contains(body, "explicit=") {
		t.Fatalf("policy sin conteos: %s", body)
	}
}

func TestServerConfigFromEnvV0PublicaReciboDaemonStartEnvV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	t.Setenv(envCodexCommandV0, "/usr/bin/codex")
	t.Setenv("HOME", "/home/alberto")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	settings := config.EffectiveConfig.Settings
	if got := effectiveSettingValueForTestV0(settings, daemonStartEnvProfileSettingV0); got != "allowlist" {
		t.Fatalf("daemon profile=%q", got)
	}
	categories := effectiveSettingValueForTestV0(settings, daemonStartEnvCategoriesSettingV0)
	if !strings.Contains(categories, "codex_runtime") || strings.Contains(categories, "/home/alberto") {
		t.Fatalf("daemon categories=%q", categories)
	}
	counts := effectiveSettingValueForTestV0(settings, daemonStartEnvCountsSettingV0)
	if !strings.Contains(counts, "projected=") || strings.Contains(counts, "/home/alberto") {
		t.Fatalf("daemon counts=%q", counts)
	}
}

func TestServerDaemonStartEnvPolicyV0DistingueExplicitDefaultedDerived(t *testing.T) {
	projectDir := t.TempDir()
	config := orquestaserver.ConfigV0{
		Addr:           "127.0.0.1:19094",
		ProjectWorkDir: projectDir,
		RuntimeWorkDir: filepath.Join(projectDir, "runtime"),
		StateDir:       filepath.Join(projectDir, "state"),
		AuditFile:      "audit.jsonl",
	}
	parent := []string{
		"ORQUESTA_SERVER_ADDR=127.0.0.1:19094",
		"ORQUESTA_STARTUP_CLEANUP_MODE=disabled",
	}

	policy := serverDaemonStartEnvPolicyV0(parent, config)
	counts := daemonStartEnvCountsV0(policy)
	for _, want := range []string{"explicit=2", "defaulted=", "derived="} {
		if !strings.Contains(counts, want) {
			t.Fatalf("counts falta %q: %s", want, counts)
		}
	}
}

func TestServerDaemonStartEnvironmentV0NoFuerzaPurgasPorDefecto(t *testing.T) {
	projectDir := t.TempDir()
	config := orquestaserver.ConfigV0{
		Addr:           "127.0.0.1:19095",
		ProjectWorkDir: projectDir,
		RuntimeWorkDir: filepath.Join(projectDir, "runtime"),
		StateDir:       filepath.Join(projectDir, "state"),
		AuditFile:      "audit.jsonl",
	}

	got := serverDaemonStartEnvironmentV0(nil, config)

	if !daemonStartEnvHasPairForTestV0(got, envStartupCleanupModeV0, startupCleanupModeDiagnoseV0) {
		t.Fatalf("startup cleanup debe arrancar en diagnose por defecto: %v", got)
	}
	if daemonStartEnvHasPairForTestV0(got, envStartupCleanupModeV0, startupCleanupModeForcedStopV0) {
		t.Fatalf("startup cleanup no debe proyectar forced_stop sin opt-in: %v", got)
	}
}

func TestServerDaemonStartEnvironmentV0RespetaForcedStopExplicito(t *testing.T) {
	projectDir := t.TempDir()
	config := orquestaserver.ConfigV0{
		Addr:           "127.0.0.1:19096",
		ProjectWorkDir: projectDir,
		RuntimeWorkDir: filepath.Join(projectDir, "runtime"),
		StateDir:       filepath.Join(projectDir, "state"),
		AuditFile:      "audit.jsonl",
	}

	got := serverDaemonStartEnvironmentV0([]string{
		envStartupCleanupModeV0 + "=" + startupCleanupModeForcedStopV0,
	}, config)

	if !daemonStartEnvHasPairForTestV0(got, envStartupCleanupModeV0, startupCleanupModeForcedStopV0) {
		t.Fatalf("startup cleanup forced_stop explicito no respetado: %v", got)
	}
}

func daemonStartEnvHasPairForTestV0(env []string, key string, value string) bool {
	for _, item := range env {
		gotKey, gotValue, ok := strings.Cut(item, "=")
		if ok && gotKey == key && gotValue == value {
			return true
		}
	}
	return false
}
