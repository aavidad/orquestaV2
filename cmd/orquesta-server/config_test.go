package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorrunner "orquesta/modulos/orquesta-director-runner"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestServerConfigFromEnvV0UsaPresupuestoDeComandosParaFronteraParalela(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", t.TempDir())

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	got := config.SupervisorCommand.DrainLimits.MaxCommands
	if got != orquestadirectorrunner.DirectorCycleMaxCommandsV0 {
		t.Fatalf("max_commands=%d want %d", got, orquestadirectorrunner.DirectorCycleMaxCommandsV0)
	}
}

func TestServerConfigFromEnvV0SupervisorResidenteNoEsperaAgenteLargo(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", t.TempDir())
	t.Setenv("ORQUESTA_SERVER_DRAIN_MAX_EXTERNAL_WAITS", "")
	t.Setenv("ORQUESTA_DIRECTOR_MAX_EXTERNAL_WAITS", "")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if config.SupervisorCommand.DrainLimits.MaxExternalWaits != defaultServerSupervisorMaxExternalWaitsV0 {
		t.Fatalf("server drain max_external_waits=%d want=%d", config.SupervisorCommand.DrainLimits.MaxExternalWaits, defaultServerSupervisorMaxExternalWaitsV0)
	}
	if config.SupervisorCommand.DrainLimits.MaxExternalWaits != 1 {
		t.Fatalf("el supervisor residente debe observar y volver al bucle, no esperar agentes largos: %d", config.SupervisorCommand.DrainLimits.MaxExternalWaits)
	}
	limits := directorLimitsV0()
	if limits.MaxExternalWaits != 120 {
		t.Fatalf("director max_external_waits=%d want=120", limits.MaxExternalWaits)
	}
}

func TestServerConfigFromEnvV0PermiteEsperaResidenteAmpliaConfiguradaV0(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", t.TempDir())
	t.Setenv("ORQUESTA_SERVER_DRAIN_MAX_EXTERNAL_WAITS", "70")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if config.SupervisorCommand.DrainLimits.MaxExternalWaits != 70 {
		t.Fatalf("max_external_waits=%d want 70", config.SupervisorCommand.DrainLimits.MaxExternalWaits)
	}
}

func TestServerConfigFromEnvV0BloqueaEsperaResidenteDescontroladaV0(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", t.TempDir())
	t.Setenv("ORQUESTA_SERVER_DRAIN_MAX_EXTERNAL_WAITS", "900")

	_, err := serverConfigFromEnvV0()
	if err == nil || !strings.Contains(err.Error(), "ORQUESTA_SERVER_DRAIN_MAX_EXTERNAL_WAITS incompatible") {
		t.Fatalf("serverConfigFromEnvV0 err=%v", err)
	}
}

func TestServerConfigFromEnvV0ExponeSupervisorDesatendidoV0(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", t.TempDir())
	t.Setenv("ORQUESTA_SERVER_SUPERVISOR_MAX_TICKS", "6")
	t.Setenv("ORQUESTA_SERVER_ALLOW_REPEATED_RUNS", "true")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if config.SupervisorCommand.MaxTicks != 6 {
		t.Fatalf("max_ticks=%d want=6", config.SupervisorCommand.MaxTicks)
	}
	if !config.SupervisorCommand.AllowRepeatedRuns {
		t.Fatalf("allow_repeated_runs=%v want=true", config.SupervisorCommand.AllowRepeatedRuns)
	}
	if config.IdleSelfImprovementPriorityScore != orquestaserver.DefaultIdleSelfImprovementPriorityScoreV0 {
		t.Fatalf("idle priority=%d", config.IdleSelfImprovementPriorityScore)
	}
	if config.IdleSelfImprovementMaxRequests != orquestaserver.DefaultIdleSelfImprovementMaxRequestsV0 {
		t.Fatalf("idle max_requests=%d", config.IdleSelfImprovementMaxRequests)
	}
	if config.IdleSelfImprovementTargetQueue != orquestaserver.DefaultIdleSelfImprovementTargetQueueV0 {
		t.Fatalf("idle target_queue=%d", config.IdleSelfImprovementTargetQueue)
	}
	if config.IdleSelfImprovementBudget.MaxGoalsPerDay != 0 ||
		config.IdleSelfImprovementBudget.MaxContextBudgetBytesPerDay != 0 {
		t.Fatalf("idle budget default=%+v", config.IdleSelfImprovementBudget)
	}
	if !containsStringV0(config.IdleSelfImprovementAcceptance, "la proyeccion publica distingue outbox pendiente, wait_external y proceso externo verificado") {
		t.Fatalf("acceptance=%v", config.IdleSelfImprovementAcceptance)
	}
}

func TestServerConfigFromEnvV0LeePresupuestoAutomejoraIdleV0(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", t.TempDir())
	t.Setenv(envServerIdleSelfImprovementDailyGoalBudgetV0, "7")
	t.Setenv(envServerIdleSelfImprovementDailyContextBudgetBytesV0, "123456")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if config.IdleSelfImprovementBudget.MaxGoalsPerDay != 7 ||
		config.IdleSelfImprovementBudget.MaxContextBudgetBytesPerDay != 123456 {
		t.Fatalf("idle budget=%+v", config.IdleSelfImprovementBudget)
	}
	if got := effectiveSettingValueForTestV0(
		config.EffectiveConfig.Settings,
		envServerIdleSelfImprovementDailyGoalBudgetV0,
	); got != "7" {
		t.Fatalf("daily goal budget effective=%q", got)
	}
}

func TestServerConfigFromEnvV0AutomejoraIdleDefaultYApagadoPorEnvV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0 default: %v", err)
	}
	if config.IdleSelfImprovementDisabled ||
		config.IdleSelfImprovementAfter != 60*time.Second {
		t.Fatalf("idle default disabled=%v after=%s", config.IdleSelfImprovementDisabled, config.IdleSelfImprovementAfter)
	}
	if len(config.IdleSelfImprovementWriteSet) == 0 || config.IdleSelfImprovementWriteSet[0] != "modulos/orquesta-server" {
		t.Fatalf("idle write-set=%v", config.IdleSelfImprovementWriteSet)
	}

	t.Setenv(envServerIdleSelfImprovementAfterV0, "0")
	config, err = serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0 disabled: %v", err)
	}
	if config.IdleSelfImprovementDisabled ||
		!config.IdleSelfImprovementIdleDisabled ||
		config.IdleSelfImprovementAfter != 0 {
		t.Fatalf("idle trigger disabled config=%+v", config)
	}
}

func TestServerConfigFromEnvV0ExponeAutomejoraGoalFirstOptInV0(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", t.TempDir())
	t.Setenv(envServerIdleSelfImprovementGoalFirstV0, "true")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if !config.IdleSelfImprovementGoalFirst {
		t.Fatalf("goal-first idle no activo: %+v", config)
	}
}

func TestServerConfigFromEnvV0ExponeTestsCongeladosOptInV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0 default: %v", err)
	}
	if config.IdleSelfImprovementFrozenTests {
		t.Fatalf("tests congelados deben estar apagados por defecto")
	}

	t.Setenv(envServerIdleSelfImprovementFrozenTestsV0, "true")
	config, err = serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0 opt-in: %v", err)
	}
	if !config.IdleSelfImprovementFrozenTests {
		t.Fatalf("tests congelados no activos: %+v", config)
	}
	setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envServerIdleSelfImprovementFrozenTestsV0)
	if setting.Value != "true" {
		t.Fatalf("setting tests congelados=%+v", setting)
	}
}

func TestServerConfigFromEnvV0LeeDirectorEscaladaV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	t.Setenv(envServerEscalationDirectorEnabledV0, "true")
	t.Setenv(envServerEscalationDirectorCommandV0, "claude,-p,--model,sonnet")
	t.Setenv(envServerEscalationDirectorTimeoutSecondsV0, "33")
	t.Setenv(envServerEscalationDirectorMaxPerDayV0, "5")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if !config.EscalationDirectorEnabled {
		t.Fatalf("director de escalada no activo: %+v", config)
	}
	wantCommand := []string{"claude", "-p", "--model", "sonnet"}
	if len(config.EscalationDirectorCommand) != len(wantCommand) {
		t.Fatalf("command=%v want=%v", config.EscalationDirectorCommand, wantCommand)
	}
	for index, want := range wantCommand {
		if config.EscalationDirectorCommand[index] != want {
			t.Fatalf("command=%v want=%v", config.EscalationDirectorCommand, wantCommand)
		}
	}
	if config.EscalationDirectorTimeout != 33*time.Second {
		t.Fatalf("timeout=%s want=33s", config.EscalationDirectorTimeout)
	}
	if config.EscalationDirectorMaxPerDay != 5 {
		t.Fatalf("max_per_day=%d want=5", config.EscalationDirectorMaxPerDay)
	}
	settings := config.EffectiveConfig.Settings
	for key, want := range map[string]string{
		envServerEscalationDirectorEnabledV0:        "true",
		envServerEscalationDirectorCommandV0:        "escalation-director-command-configured",
		envServerEscalationDirectorTimeoutSecondsV0: "33",
		envServerEscalationDirectorMaxPerDayV0:      "5",
	} {
		if got := effectiveSettingValueForTestV0(settings, key); got != want {
			t.Fatalf("%s=%q want %q settings=%+v", key, got, want, settings)
		}
	}
	commandSetting := effectiveSettingForTestV0(settings, envServerEscalationDirectorCommandV0)
	if !commandSetting.Sensitive {
		t.Fatalf("command debe quedar sensible: %+v", commandSetting)
	}
	rawEffectiveConfig, err := json.Marshal(config.EffectiveConfig)
	if err != nil {
		t.Fatalf("marshal effective config: %v", err)
	}
	if strings.Contains(string(rawEffectiveConfig), "--model") ||
		strings.Contains(string(rawEffectiveConfig), "sonnet") {
		t.Fatalf("effective_config filtra argv crudo: %s", string(rawEffectiveConfig))
	}
}

func TestServerConfigFromEnvV0DerivaAutomejoraGoalFirstDeBackendCodexGoalV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	t.Setenv(envServerIdleSelfImprovementGoalFirstV0, "")
	t.Setenv(envCodexGoalBackendV0, codexGoalBackendAppServerTmuxV0)

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if !config.IdleSelfImprovementGoalFirst {
		t.Fatalf("goal-first idle debe derivarse del backend goal: %+v", config)
	}
	setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envServerIdleSelfImprovementGoalFirstV0)
	if setting.Value != "true" || setting.Source != "derived_from_codex_goal_backend" {
		t.Fatalf("setting goal-first=%+v", setting)
	}
	if !effectiveConfigHasDiagnosticForTestV0(config.EffectiveConfig, "idle_self_improvement_goal_first_derived", envCodexGoalBackendV0) {
		t.Fatalf("diagnostico de derivacion ausente: %+v", config.EffectiveConfig.Diagnostics)
	}
}

func TestServerConfigFromEnvV0GoalFirstExplicitoFalseGanaABackendCodexGoalV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	t.Setenv(envServerIdleSelfImprovementGoalFirstV0, "false")
	t.Setenv(envCodexGoalBackendV0, codexGoalBackendAppServerTmuxV0)

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if config.IdleSelfImprovementGoalFirst {
		t.Fatalf("goal-first explicito false debe conservar compatibilidad legacy: %+v", config)
	}
	setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envServerIdleSelfImprovementGoalFirstV0)
	if setting.Value != "false" || setting.Source != "explicit" {
		t.Fatalf("setting goal-first=%+v", setting)
	}
}

func TestServerConfigFromEnvV0DerivaAutomejoraGoalFirstDeBackendClaudeGoalV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	t.Setenv(envServerIdleSelfImprovementGoalFirstV0, "")
	t.Setenv(envCodexGoalBackendV0, claudeGoalBackendFileControlV0)

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if !config.IdleSelfImprovementGoalFirst {
		t.Fatalf("goal-first idle debe derivarse del backend Claude goal: %+v", config)
	}
	setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envServerIdleSelfImprovementGoalFirstV0)
	if setting.Value != "true" || setting.Source != "derived_from_claude_goal_backend" {
		t.Fatalf("setting goal-first=%+v", setting)
	}
	if got := effectiveSettingValueForTestV0(config.EffectiveConfig.Settings, envCodexGoalBackendV0); got != claudeGoalBackendFileControlV0 {
		t.Fatalf("setting Claude goal backend=%q", got)
	}
	if !effectiveConfigHasDiagnosticForTestV0(config.EffectiveConfig, "idle_self_improvement_goal_first_derived", envCodexGoalBackendV0) {
		t.Fatalf("diagnostico de derivacion Claude ausente: %+v", config.EffectiveConfig.Diagnostics)
	}
	if effectiveConfigHasDiagnosticForTestV0(config.EffectiveConfig, orquestamcp.MCPExternalWorkRunGoalBackendRequiredV0) {
		t.Fatalf("diagnostico backend requerido no debe aparecer con Claude goal: %+v", config.EffectiveConfig.Diagnostics)
	}
}

func TestServerConfigFromEnvV0DerivaAutomejoraGoalFirstDeBackendGeminiGoalV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	t.Setenv(envServerIdleSelfImprovementGoalFirstV0, "")
	t.Setenv(envCodexGoalBackendV0, geminiGoalBackendFileControlV0)

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if !config.IdleSelfImprovementGoalFirst {
		t.Fatalf("goal-first idle debe derivarse del backend Gemini goal: %+v", config)
	}
	setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envServerIdleSelfImprovementGoalFirstV0)
	if setting.Value != "true" || setting.Source != "derived_from_gemini_goal_backend" {
		t.Fatalf("setting goal-first=%+v", setting)
	}
	if got := effectiveSettingValueForTestV0(config.EffectiveConfig.Settings, envCodexGoalBackendV0); got != geminiGoalBackendFileControlV0 {
		t.Fatalf("setting Gemini goal backend=%q", got)
	}
	if !effectiveConfigHasDiagnosticForTestV0(config.EffectiveConfig, "idle_self_improvement_goal_first_derived", envCodexGoalBackendV0) {
		t.Fatalf("diagnostico de derivacion Gemini ausente: %+v", config.EffectiveConfig.Diagnostics)
	}
	if effectiveConfigHasDiagnosticForTestV0(config.EffectiveConfig, orquestamcp.MCPExternalWorkRunGoalBackendRequiredV0) {
		t.Fatalf("diagnostico backend requerido no debe aparecer con Gemini goal: %+v", config.EffectiveConfig.Diagnostics)
	}
}

func TestServerConfigFromEnvV0LeeGoalBackendDesdeFicheroCanonicoV0(t *testing.T) {
	projectDir := t.TempDir()
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	t.Setenv(envServerIdleSelfImprovementGoalFirstV0, "")
	configFile := `{
		"schema_version":"orquesta_config.v0",
		"goal_backend":{
			"kind":"claude_file_control",
			"timeout_ms":12000,
			"preflight_timeout_ms":3400,
			"allow_app_server_proxy_diagnostic":true
		}
	}`
	if err := os.WriteFile(filepath.Join(projectDir, serverProjectConfigFileNameV0), []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if !config.IdleSelfImprovementGoalFirst {
		t.Fatalf("goal-first idle debe derivarse del backend goal del fichero: %+v", config)
	}
	for key, want := range map[string]string{
		envCodexGoalBackendV0:              claudeGoalBackendFileControlV0,
		envAllowAppServerProxyDiagnosticV0: "true",
		envCodexGoalTimeoutMSV0:            "12000",
		envCodexGoalPreflightTimeoutMSV0:   "3400",
	} {
		setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, key)
		if setting.Value != want || setting.Source != configSettingSourceConfigFileV0 {
			t.Fatalf("setting %s=%+v want value=%q source=config_file", key, setting, want)
		}
	}
	setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envServerIdleSelfImprovementGoalFirstV0)
	if setting.Value != "true" || setting.Source != "derived_from_claude_goal_backend" {
		t.Fatalf("setting goal-first=%+v", setting)
	}
	if effectiveConfigHasDiagnosticForTestV0(config.EffectiveConfig, orquestamcp.MCPExternalWorkRunGoalBackendRequiredV0) {
		t.Fatalf("diagnostico backend requerido no debe aparecer con backend en fichero: %+v", config.EffectiveConfig.Diagnostics)
	}
	if !effectiveConfigHasDiagnosticForTestV0(config.EffectiveConfig, "idle_self_improvement_goal_first_derived", envCodexGoalBackendV0) {
		t.Fatalf("diagnostico derivacion ausente: %+v", config.EffectiveConfig.Diagnostics)
	}
}

func TestServerConfigFromEnvV0EnvExplicitoGanaGoalBackendFicheroCanonicoV0(t *testing.T) {
	projectDir := t.TempDir()
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	t.Setenv(envCodexGoalBackendV0, geminiGoalBackendFileControlV0)
	t.Setenv(envCodexGoalTimeoutMSV0, "22000")
	t.Setenv(envAllowAppServerProxyDiagnosticV0, "false")
	configFile := `{
		"schema_version":"orquesta_config.v0",
		"goal_backend":{
			"kind":"claude_file_control",
			"timeout_ms":12000,
			"preflight_timeout_ms":3400,
			"allow_app_server_proxy_diagnostic":true
		}
	}`
	if err := os.WriteFile(filepath.Join(projectDir, serverProjectConfigFileNameV0), []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	expect := map[string]struct {
		value  string
		source string
	}{
		envCodexGoalBackendV0:              {value: geminiGoalBackendFileControlV0, source: "explicit"},
		envAllowAppServerProxyDiagnosticV0: {value: "false", source: "explicit"},
		envCodexGoalTimeoutMSV0:            {value: "22000", source: "explicit"},
		envCodexGoalPreflightTimeoutMSV0:   {value: "3400", source: configSettingSourceConfigFileV0},
	}
	for key, want := range expect {
		setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, key)
		if setting.Value != want.value || setting.Source != want.source {
			t.Fatalf("setting %s=%+v want=%+v", key, setting, want)
		}
	}
	setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envServerIdleSelfImprovementGoalFirstV0)
	if setting.Value != "true" || setting.Source != "derived_from_gemini_goal_backend" {
		t.Fatalf("setting goal-first=%+v", setting)
	}
}

func TestServerConfigFromEnvV0LeeServerIdleCanonicoYEnvDeprecatedOverrideV0(t *testing.T) {
	projectDir := t.TempDir()
	idleDir := filepath.Join(projectDir, "idle")
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	t.Setenv(envServerIdleSelfImprovementPriorityScoreV0, "88")
	configFile := `{
		"schema_version":"orquesta_config.v0",
		"server_idle":{
			"after_seconds":12,
			"disabled":false,
			"project_workdir":"` + filepath.ToSlash(idleDir) + `",
			"project_ref":"project-ref-idle-file",
			"worktree_ref":"worktree-ref-idle-file",
			"branch_ref":"branch-ref-idle-file",
			"area":"area-idle-file",
			"write_set":["cmd/orquesta-server","modulos/orquesta-server"],
			"required_tests":["go test ./cmd/orquesta-server"],
			"context_refs":["context-ref-idle-file"],
			"evidence_refs":["evidence-ref-idle-file"],
			"acceptance":["idle acceptance"],
			"goal_first_enabled":false,
			"frozen_tests_enabled":true,
			"compact_rules":["compact-rule-file"],
			"priority_score":77,
			"max_requests":4,
			"target_queue":6,
			"daily_goal_budget":3,
			"daily_context_budget_bytes":12345
		}
	}`
	if err := os.WriteFile(filepath.Join(projectDir, serverProjectConfigFileNameV0), []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if config.IdleSelfImprovementAfter != 12*time.Second ||
		config.IdleSelfImprovementDisabled ||
		config.IdleSelfImprovementProjectWorkDir != idleDir ||
		config.IdleSelfImprovementProjectRef != "project-ref-idle-file" ||
		config.IdleSelfImprovementWorktreeRef != "worktree-ref-idle-file" ||
		config.IdleSelfImprovementBranchRef != "branch-ref-idle-file" ||
		config.IdleSelfImprovementSuggestedArea != "area-idle-file" ||
		config.IdleSelfImprovementGoalFirst ||
		!config.IdleSelfImprovementFrozenTests ||
		config.IdleSelfImprovementPriorityScore != 88 ||
		config.IdleSelfImprovementMaxRequests != 4 ||
		config.IdleSelfImprovementTargetQueue != 6 ||
		config.IdleSelfImprovementBudget.MaxGoalsPerDay != 3 ||
		config.IdleSelfImprovementBudget.MaxContextBudgetBytesPerDay != 12345 {
		t.Fatalf("config idle=%+v budget=%+v", config, config.IdleSelfImprovementBudget)
	}
	if strings.Join(config.IdleSelfImprovementWriteSet, ",") != "cmd/orquesta-server,modulos/orquesta-server" ||
		strings.Join(config.IdleSelfImprovementRequiredTests, ",") != "go test ./cmd/orquesta-server" ||
		strings.Join(config.IdleSelfImprovementContextRefs, ",") != "context-ref-idle-file" ||
		strings.Join(config.IdleSelfImprovementEvidenceRefs, ",") != "evidence-ref-idle-file" ||
		strings.Join(config.IdleSelfImprovementAcceptance, ",") != "idle acceptance" ||
		strings.Join(config.IdleSelfImprovementCompactRules, ",") != "compact-rule-file" {
		t.Fatalf("idle slices write_set=%v required=%v context=%v evidence=%v acceptance=%v compact=%v",
			config.IdleSelfImprovementWriteSet,
			config.IdleSelfImprovementRequiredTests,
			config.IdleSelfImprovementContextRefs,
			config.IdleSelfImprovementEvidenceRefs,
			config.IdleSelfImprovementAcceptance,
			config.IdleSelfImprovementCompactRules)
	}

	settings := config.EffectiveConfig.Settings
	for key, want := range map[string]string{
		envServerIdleSelfImprovementAfterV0:                   "12",
		envServerIdleSelfImprovementDisabledV0:                "false",
		envServerIdleSelfImprovementProjectRefV0:              "project-ref-idle-file",
		envServerIdleSelfImprovementWorktreeRefV0:             "worktree-ref-idle-file",
		envServerIdleSelfImprovementBranchRefV0:               "branch-ref-idle-file",
		envServerIdleSelfImprovementAreaV0:                    "area-idle-file",
		envServerIdleSelfImprovementWriteSetV0:                "cmd/orquesta-server,modulos/orquesta-server",
		envServerIdleSelfImprovementRequiredTestsV0:           "go test ./cmd/orquesta-server",
		envServerIdleSelfImprovementContextRefsV0:             "context-ref-idle-file",
		envServerIdleSelfImprovementEvidenceRefsV0:            "evidence-ref-idle-file",
		envServerIdleSelfImprovementAcceptanceV0:              "idle acceptance",
		envServerIdleSelfImprovementGoalFirstV0:               "false",
		envServerIdleSelfImprovementFrozenTestsV0:             "true",
		envServerIdleSelfImprovementCompactRulesV0:            "compact-rule-file",
		envServerIdleSelfImprovementMaxRequestsV0:             "4",
		envServerIdleSelfImprovementTargetQueueV0:             "6",
		envServerIdleSelfImprovementDailyGoalBudgetV0:         "3",
		envServerIdleSelfImprovementDailyContextBudgetBytesV0: "12345",
	} {
		setting := effectiveSettingForTestV0(settings, key)
		if setting.Value != want || setting.Source != configSettingSourceConfigFileV0 {
			t.Fatalf("%s setting=%+v want value=%q source=config_file", key, setting, want)
		}
	}
	projectWorkdirSetting := effectiveSettingForTestV0(settings, envServerIdleSelfImprovementProjectWorkDirV0)
	if projectWorkdirSetting.Value != "idle-self-improvement-project-workdir-configured" ||
		projectWorkdirSetting.Source != configSettingSourceConfigFileV0 ||
		!projectWorkdirSetting.Sensitive {
		t.Fatalf("project workdir setting=%+v", projectWorkdirSetting)
	}
	priority := effectiveSettingForTestV0(settings, envServerIdleSelfImprovementPriorityScoreV0)
	if priority.Value != "88" || priority.Source != "explicit" {
		t.Fatalf("priority setting=%+v", priority)
	}
	if !effectiveConfigHasDiagnosticForTestV0(config.EffectiveConfig, "deprecated_env_used", envServerIdleSelfImprovementPriorityScoreV0, "server_idle.*") {
		t.Fatalf("diagnostico deprecated server_idle ausente: %+v", config.EffectiveConfig.Diagnostics)
	}
}

func TestServerConfigFromEnvV0ExternalWorkLegacyDirectorLoopPorDefectoFalseV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envExternalWorkLegacyDirectorLoopV0)
	if setting.Value != "false" || setting.Source != "defaulted" {
		t.Fatalf("setting external-work legacy=%+v", setting)
	}
	if !effectiveConfigHasDiagnosticForTestV0(config.EffectiveConfig, orquestamcp.MCPExternalWorkRunGoalBackendRequiredV0, envCodexGoalBackendV0, codexGoalBackendAppServerTmuxV0) {
		t.Fatalf("diagnostico goal backend requerido ausente: %+v", config.EffectiveConfig.Diagnostics)
	}
}

func TestServerConfigFromEnvV0DiagnosticaOrquestaBaseURLLegacyAliasV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	t.Setenv(envOrquestaServerURLV0, "")
	t.Setenv(envOrquestaBaseURLV0, "http://127.0.0.1:18080")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envOrquestaServerURLV0)
	if setting.Value != "orquesta-server-url-configured" ||
		setting.Source != "legacy_alias" ||
		!setting.Sensitive {
		t.Fatalf("setting Orquesta server URL legacy=%+v", setting)
	}
	if !effectiveConfigHasDiagnosticForTestV0(config.EffectiveConfig, "deprecated_env_used", envOrquestaBaseURLV0, envOrquestaServerURLV0) {
		t.Fatalf("diagnostico Orquesta URL alias ausente: %+v", config.EffectiveConfig.Diagnostics)
	}
}

func TestServerConfigFromEnvV0DiagnosticaOrquestaBaseURLPisadaV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	t.Setenv(envOrquestaServerURLV0, "http://127.0.0.1:18080")
	t.Setenv(envOrquestaBaseURLV0, "http://127.0.0.1:18081")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envOrquestaServerURLV0)
	if setting.Value != "orquesta-server-url-configured" ||
		setting.Source != "explicit" ||
		!setting.Sensitive {
		t.Fatalf("setting Orquesta server URL conflicto=%+v", setting)
	}
	if !effectiveConfigHasDiagnosticForTestV0(config.EffectiveConfig, "env_alias_conflict", envOrquestaBaseURLV0, envOrquestaServerURLV0) {
		t.Fatalf("diagnostico Orquesta URL conflicto ausente: %+v", config.EffectiveConfig.Diagnostics)
	}
}

func TestServerConfigFromEnvV0DiagnosticaOPESBaseURLLegacyAliasV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	t.Setenv(envOPESBaseURLV0, "")
	t.Setenv(envOPESBaseURLLegacyV0, "http://127.0.0.1:18082")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envOPESBaseURLV0)
	if setting.Value != "opes-base-url-configured" ||
		setting.Source != "legacy_alias" ||
		!setting.Sensitive {
		t.Fatalf("setting OPES base URL legacy=%+v", setting)
	}
	if !effectiveConfigHasDiagnosticForTestV0(config.EffectiveConfig, "deprecated_env_used", envOPESBaseURLLegacyV0, envOPESBaseURLV0) {
		t.Fatalf("diagnostico OPES alias ausente: %+v", config.EffectiveConfig.Diagnostics)
	}
}

func TestServerConfigFromEnvV0DiagnosticaOPESBaseURLPisadaV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	t.Setenv(envOPESBaseURLV0, "http://127.0.0.1:18082")
	t.Setenv(envOPESBaseURLLegacyV0, "http://127.0.0.1:18083")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envOPESBaseURLV0)
	if setting.Value != "opes-base-url-configured" ||
		setting.Source != "explicit" ||
		!setting.Sensitive {
		t.Fatalf("setting OPES base URL conflicto=%+v", setting)
	}
	if !effectiveConfigHasDiagnosticForTestV0(config.EffectiveConfig, "env_alias_conflict", envOPESBaseURLLegacyV0, envOPESBaseURLV0) {
		t.Fatalf("diagnostico OPES conflicto ausente: %+v", config.EffectiveConfig.Diagnostics)
	}
}

func TestServerConfigFromEnvV0PublicaGuardasOPESBridgeEnEffectiveConfigV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	t.Setenv(envOPESBaseURLV0, "http://127.0.0.1:18082")
	t.Setenv(envOPESTemporalConfirmV0, "true")
	t.Setenv(envOPESBridgeEnabledV0, "true")
	t.Setenv(envOPESBridgeConfirmV0, "true")
	t.Setenv(envOPESBridgeDryRunV0, "true")
	t.Setenv(envOPESBridgeJobRefV0, "job-ref-opes-real-field-test")
	t.Setenv(envOPESBridgeLimitV0, "1")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	settings := config.EffectiveConfig.Settings
	for key, want := range map[string]string{
		envOPESBaseURLV0:              "opes-base-url-configured",
		envOPESTemporalConfirmV0:      "true",
		envOPESBridgeEnabledV0:        "true",
		envOPESBridgeConfirmV0:        "true",
		envOPESBridgeDryRunV0:         "true",
		envOPESBridgeJobRefV0:         "job-ref-opes-real-field-test",
		envOPESBridgeLimitV0:          "1",
		envOPESBridgeTimeoutSecondsV0: "30",
	} {
		if got := effectiveSettingValueForTestV0(settings, key); got != want {
			t.Fatalf("%s=%q want %q settings=%+v", key, got, want, settings)
		}
	}
	if setting := effectiveSettingForTestV0(settings, envOPESBaseURLV0); !setting.Sensitive {
		t.Fatalf("OPES base URL debe quedar sensible: %+v", setting)
	}
	rawEffectiveConfig, err := json.Marshal(config.EffectiveConfig)
	if err != nil {
		t.Fatalf("marshal effective config: %v", err)
	}
	if strings.Contains(string(rawEffectiveConfig), "http://127.0.0.1:18082") {
		t.Fatalf("effective_config filtra OPES base URL cruda: %s", string(rawEffectiveConfig))
	}
}

func TestServerConfigFromEnvV0DiagnosticaCodexCodeHomeLegacyAliasV0(t *testing.T) {
	root := t.TempDir()
	t.Setenv(envCodexProjectWorkDirV0, root)
	t.Setenv(envCodexCodeHomeV0, "")
	t.Setenv(envCodexCodeHomeLegacyV0, filepath.Join(root, "legacy-codex-home"))

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envCodexCodeHomeV0)
	if setting.Value != "codex-code-home-configured" ||
		setting.Source != "legacy_alias" ||
		!setting.Sensitive {
		t.Fatalf("setting codex code home legacy=%+v", setting)
	}
	if !effectiveConfigHasDiagnosticForTestV0(config.EffectiveConfig, "deprecated_env_used", envCodexCodeHomeLegacyV0, envCodexCodeHomeV0) {
		t.Fatalf("diagnostico Codex code home alias ausente: %+v", config.EffectiveConfig.Diagnostics)
	}
}

func TestServerConfigFromEnvV0DiagnosticaOrquestaCodexHomeLegacyAliasV0(t *testing.T) {
	root := t.TempDir()
	t.Setenv(envCodexProjectWorkDirV0, root)
	t.Setenv(envCodexCodeHomeV0, "")
	t.Setenv(envCodexHomeV0, filepath.Join(root, "orquesta-codex-home"))
	t.Setenv(envCodexCodeHomeLegacyV0, "")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envCodexCodeHomeV0)
	if setting.Value != "codex-code-home-configured" ||
		setting.Source != "legacy_alias" ||
		!setting.Sensitive {
		t.Fatalf("setting codex code home ORQUESTA_CODEX_HOME legacy=%+v", setting)
	}
	if !effectiveConfigHasDiagnosticForTestV0(config.EffectiveConfig, "deprecated_env_used", envCodexHomeV0, envCodexCodeHomeV0) {
		t.Fatalf("diagnostico ORQUESTA_CODEX_HOME alias ausente: %+v", config.EffectiveConfig.Diagnostics)
	}
}

func TestServerConfigFromEnvV0DiagnosticaCodexCodeHomePisadoV0(t *testing.T) {
	root := t.TempDir()
	t.Setenv(envCodexProjectWorkDirV0, root)
	t.Setenv(envCodexCodeHomeV0, filepath.Join(root, "canonical-codex-home"))
	t.Setenv(envCodexCodeHomeLegacyV0, filepath.Join(root, "legacy-codex-home"))

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envCodexCodeHomeV0)
	if setting.Value != "codex-code-home-configured" ||
		setting.Source != "explicit" ||
		!setting.Sensitive {
		t.Fatalf("setting codex code home conflicto=%+v", setting)
	}
	if !effectiveConfigHasDiagnosticForTestV0(config.EffectiveConfig, "env_alias_conflict", envCodexCodeHomeLegacyV0, envCodexCodeHomeV0) {
		t.Fatalf("diagnostico Codex code home conflicto ausente: %+v", config.EffectiveConfig.Diagnostics)
	}
}

func TestServerConfigFromEnvV0DiagnosticaOrquestaCodexHomePisadoV0(t *testing.T) {
	root := t.TempDir()
	t.Setenv(envCodexProjectWorkDirV0, root)
	t.Setenv(envCodexCodeHomeV0, filepath.Join(root, "canonical-codex-home"))
	t.Setenv(envCodexHomeV0, filepath.Join(root, "orquesta-codex-home"))

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envCodexCodeHomeV0)
	if setting.Value != "codex-code-home-configured" ||
		setting.Source != "explicit" ||
		!setting.Sensitive {
		t.Fatalf("setting codex code home ORQUESTA_CODEX_HOME conflicto=%+v", setting)
	}
	if !effectiveConfigHasDiagnosticForTestV0(config.EffectiveConfig, "env_alias_conflict", envCodexHomeV0, envCodexCodeHomeV0) {
		t.Fatalf("diagnostico ORQUESTA_CODEX_HOME conflicto ausente: %+v", config.EffectiveConfig.Diagnostics)
	}
}

func TestServerConfigFromEnvV0ObservadorGoalFirstResidentePorDefectoV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if !config.GoalObserverEnabled ||
		config.GoalObserverMaxItems != orquestaserver.DefaultGoalObserverMaxItemsV0 ||
		config.GoalObserverInterval != config.TickInterval ||
		config.GoalObserverTimeout != orquestaserver.DefaultGoalObserverTimeoutV0 {
		t.Fatalf("goal observer config=%+v", config)
	}
	if got := effectiveSettingValueForTestV0(config.EffectiveConfig.Settings, envServerGoalObserverEnabledV0); got != "true" {
		t.Fatalf("%s=%q want true", envServerGoalObserverEnabledV0, got)
	}
	if got := effectiveSettingValueForTestV0(config.EffectiveConfig.Settings, envServerGoalObserverIntervalMSV0); got != "5000" {
		t.Fatalf("%s=%q want 5000", envServerGoalObserverIntervalMSV0, got)
	}
	if got := effectiveSettingValueForTestV0(config.EffectiveConfig.Settings, envServerGoalObserverMaxItemsV0); got != "70" {
		t.Fatalf("%s=%q want 70", envServerGoalObserverMaxItemsV0, got)
	}
	fingerprint := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envServerGoalObserverFingerprintEnabledV0)
	if fingerprint.Value != "true" || fingerprint.Source != "defaulted" {
		t.Fatalf("setting goal observer fingerprint=%+v", fingerprint)
	}
}

func TestServerConfigFromEnvV0PermiteApagarObservadorGoalFirstV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	t.Setenv(envServerGoalObserverEnabledV0, "false")
	t.Setenv(envServerGoalObserverIntervalMSV0, "1500")
	t.Setenv(envServerGoalObserverMaxItemsV0, "11")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if config.GoalObserverEnabled ||
		config.GoalObserverInterval != 1500*time.Millisecond ||
		config.GoalObserverTimeout != orquestaserver.DefaultGoalObserverTimeoutV0 ||
		config.GoalObserverMaxItems != 11 {
		t.Fatalf("goal observer config=%+v", config)
	}
	setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envServerGoalObserverEnabledV0)
	if setting.Value != "false" || setting.Source != "explicit" {
		t.Fatalf("setting goal observer=%+v", setting)
	}
	interval := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envServerGoalObserverIntervalMSV0)
	if interval.Value != "1500" || interval.Source != "explicit" {
		t.Fatalf("setting goal observer interval=%+v", interval)
	}
}

func TestServerConfigFromEnvV0PermiteApagarFingerprintObservadorGoalFirstV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	t.Setenv(envServerGoalObserverFingerprintEnabledV0, "false")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if serverGoalObserverFingerprintEnabledFromEnvV0() {
		t.Fatalf("%s=false debe apagar la guarda", envServerGoalObserverFingerprintEnabledV0)
	}
	setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envServerGoalObserverFingerprintEnabledV0)
	if setting.Value != "false" || setting.Source != "explicit" {
		t.Fatalf("setting goal observer fingerprint=%+v", setting)
	}
}

func TestServerConfigFromEnvV0SelfAuditBacklogOptInVisibleV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())

	defaultConfig, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0 default: %v", err)
	}
	if defaultConfig.SelfAuditBacklogEnabled {
		t.Fatalf("self audit backlog activo por defecto: %+v", defaultConfig)
	}
	defaultSetting := effectiveSettingForTestV0(defaultConfig.EffectiveConfig.Settings, envSelfAuditBacklogEnabledV0)
	if defaultSetting.Value != "false" || defaultSetting.Source != "defaulted" {
		t.Fatalf("default setting self audit=%+v", defaultSetting)
	}

	t.Setenv(envSelfAuditBacklogEnabledV0, "true")
	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0 opt-in: %v", err)
	}
	if !config.SelfAuditBacklogEnabled {
		t.Fatalf("self audit backlog no activo con opt-in: %+v", config)
	}
	setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envSelfAuditBacklogEnabledV0)
	if setting.Value != "true" || setting.Source != "explicit" {
		t.Fatalf("setting self audit=%+v", setting)
	}
}

func TestServerConfigFromEnvV0AutoprogrammingLegacyDirectorLoopPorDefectoFalseV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envAutoprogrammingLegacyDirectorLoopV0)
	if setting.Value != "false" || setting.Source != "defaulted" {
		t.Fatalf("setting autoprogramming legacy=%+v", setting)
	}
}

func TestServerConfigFromEnvV0AutoprogrammingLegacyDirectorLoopOptInVisibleV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	t.Setenv(envAutoprogrammingLegacyDirectorLoopV0, "true")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envAutoprogrammingLegacyDirectorLoopV0)
	if setting.Value != "true" || setting.Source != "explicit" {
		t.Fatalf("setting autoprogramming legacy=%+v", setting)
	}
}

func TestServerConfigFromEnvV0ExternalWorkLegacyDirectorLoopOptInVisibleV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	t.Setenv(envExternalWorkLegacyDirectorLoopV0, "true")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envExternalWorkLegacyDirectorLoopV0)
	if setting.Value != "true" || setting.Source != "explicit" {
		t.Fatalf("setting external-work legacy=%+v", setting)
	}
	if effectiveConfigHasDiagnosticForTestV0(config.EffectiveConfig, orquestamcp.MCPExternalWorkRunGoalBackendRequiredV0) {
		t.Fatalf("diagnostico goal backend no debe aparecer con legacy opt-in: %+v", config.EffectiveConfig.Diagnostics)
	}
}

func TestServerConfigFromEnvV0PromocionMaterialSinACKOptOutVisibleV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	t.Setenv(envCodexPromoteMaterializedArtifactWithoutAckV0, "false")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envCodexPromoteMaterializedArtifactWithoutAckV0)
	if setting.Value != "false" || setting.Source != "explicit" {
		t.Fatalf("setting promote without ack=%+v", setting)
	}
}

func TestServerConfigFromEnvV0PublicaUmbralCheckpointGoalConfigurableV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	t.Setenv(envAutoprogrammingCheckpointOnlyHighConsumptionTokensV0, "42000")
	t.Setenv(envAutoprogrammingCheckpointOnlyMaxWaitSecondsV0, "1200")
	t.Setenv(envAutoprogrammingNoCheckpointWarningMaxWaitSecondsV0, "900")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	policy := serverAutoprogrammingGoalProgressPolicyFromEnvV0()
	if policy.CheckpointOnlyHighConsumptionTokens != 42000 ||
		policy.CheckpointOnlyMaxWaitSeconds != 1200 ||
		policy.NoCheckpointWarningMaxWaitSeconds != 900 {
		t.Fatalf("policy=%+v", policy)
	}
	setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envAutoprogrammingCheckpointOnlyHighConsumptionTokensV0)
	if setting.Value != "42000" || setting.Source != "explicit" {
		t.Fatalf("setting checkpoint threshold=%+v", setting)
	}
	setting = effectiveSettingForTestV0(config.EffectiveConfig.Settings, envAutoprogrammingCheckpointOnlyMaxWaitSecondsV0)
	if setting.Value != "1200" || setting.Source != "explicit" {
		t.Fatalf("setting checkpoint wait=%+v", setting)
	}
	setting = effectiveSettingForTestV0(config.EffectiveConfig.Settings, envAutoprogrammingNoCheckpointWarningMaxWaitSecondsV0)
	if setting.Value != "900" || setting.Source != "explicit" {
		t.Fatalf("setting no checkpoint wait=%+v", setting)
	}
}

func TestServerConfigFromEnvV0LeeUmbralCheckpointDesdeFicheroCanonicoV0(t *testing.T) {
	projectDir := t.TempDir()
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	configFile := `{
		"schema_version":"orquesta_config.v0",
		"autoprogramming":{
			"checkpoint_only_high_consumption_tokens":450000,
			"checkpoint_only_max_wait_seconds":1500,
			"no_checkpoint_warning_max_wait_seconds":900
		}
	}`
	if err := os.WriteFile(filepath.Join(projectDir, serverProjectConfigFileNameV0), []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	policy := serverAutoprogrammingGoalProgressPolicyFromConfigV0(config)
	if policy.CheckpointOnlyHighConsumptionTokens != 450000 ||
		policy.CheckpointOnlyMaxWaitSeconds != 1500 ||
		policy.NoCheckpointWarningMaxWaitSeconds != 900 {
		t.Fatalf("policy=%+v", policy)
	}
	setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envAutoprogrammingCheckpointOnlyHighConsumptionTokensV0)
	if setting.Value != "450000" || setting.Source != configSettingSourceConfigFileV0 {
		t.Fatalf("setting checkpoint threshold=%+v", setting)
	}
	setting = effectiveSettingForTestV0(config.EffectiveConfig.Settings, envAutoprogrammingCheckpointOnlyMaxWaitSecondsV0)
	if setting.Value != "1500" || setting.Source != configSettingSourceConfigFileV0 {
		t.Fatalf("setting checkpoint wait=%+v", setting)
	}
	setting = effectiveSettingForTestV0(config.EffectiveConfig.Settings, envAutoprogrammingNoCheckpointWarningMaxWaitSecondsV0)
	if setting.Value != "900" || setting.Source != configSettingSourceConfigFileV0 {
		t.Fatalf("setting no checkpoint wait=%+v", setting)
	}
	stack, err := buildStackFromEnvWithGoalBackendV0(config, serverCodexGoalBackendV0{})
	if err != nil {
		t.Fatalf("buildStackFromEnvWithGoalBackendV0: %v", err)
	}
	if stack.MCPTransportBindings.AutoprogrammingGoalProgressPolicy.CheckpointOnlyHighConsumptionTokens != 450000 ||
		stack.MCPTransportBindings.AutoprogrammingGoalProgressPolicy.CheckpointOnlyMaxWaitSeconds != 1500 {
		t.Fatalf("stack goal_progress_policy=%+v", stack.MCPTransportBindings.AutoprogrammingGoalProgressPolicy)
	}
}

func TestServerConfigFromEnvV0EnvExplicitoGanaAlFicheroCanonicoV0(t *testing.T) {
	projectDir := t.TempDir()
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	t.Setenv(envAutoprogrammingCheckpointOnlyHighConsumptionTokensV0, "42000")
	configFile := `{
		"schema_version":"orquesta_config.v0",
		"autoprogramming":{
			"checkpoint_only_high_consumption_tokens":450000,
			"checkpoint_only_max_wait_seconds":1500
		}
	}`
	if err := os.WriteFile(filepath.Join(projectDir, serverProjectConfigFileNameV0), []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	policy := serverAutoprogrammingGoalProgressPolicyFromConfigV0(config)
	if policy.CheckpointOnlyHighConsumptionTokens != 42000 ||
		policy.CheckpointOnlyMaxWaitSeconds != 1500 {
		t.Fatalf("policy=%+v", policy)
	}
	setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envAutoprogrammingCheckpointOnlyHighConsumptionTokensV0)
	if setting.Value != "42000" || setting.Source != "explicit" {
		t.Fatalf("setting checkpoint threshold=%+v", setting)
	}
	setting = effectiveSettingForTestV0(config.EffectiveConfig.Settings, envAutoprogrammingCheckpointOnlyMaxWaitSecondsV0)
	if setting.Value != "1500" || setting.Source != configSettingSourceConfigFileV0 {
		t.Fatalf("setting checkpoint wait=%+v", setting)
	}
}

func TestServerConfigFromEnvWithProjectConfigPathV0CargaFicheroExplicito(t *testing.T) {
	root := t.TempDir()
	stateDir := filepath.Join(root, "state-from-explicit-config")
	runtimeDir := filepath.Join(root, "runtime-from-explicit-config")
	configPath := filepath.Join(root, "custom-orquesta.config.json")
	t.Setenv(envCodexProjectWorkDirV0, "")
	t.Setenv(envServerStateDirV0, "")
	t.Setenv(envCodexRuntimeWorkDirV0, "")
	t.Setenv(envAutoprogrammingCheckpointOnlyHighConsumptionTokensV0, "")
	configFile := `{
		"schema_version":"orquesta_config.v0",
		"server":{"state_dir":"` + filepath.ToSlash(stateDir) + `"},
		"codex_runtime":{"runtime_work_dir":"` + filepath.ToSlash(runtimeDir) + `"},
		"autoprogramming":{"checkpoint_only_high_consumption_tokens":450000}
	}`
	if err := os.WriteFile(configPath, []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	config, err := serverConfigFromEnvWithProjectConfigPathV0(configPath)
	if err != nil {
		t.Fatalf("serverConfigFromEnvWithProjectConfigPathV0: %v", err)
	}
	absConfigPath, err := filepath.Abs(configPath)
	if err != nil {
		t.Fatalf("abs config: %v", err)
	}
	if config.ProjectWorkDir != root ||
		config.ProjectConfigFilePath != absConfigPath ||
		config.StateDir != stateDir ||
		config.RuntimeWorkDir != runtimeDir {
		t.Fatalf("config explicita no aplicada: %+v", config)
	}
	setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envAutoprogrammingCheckpointOnlyHighConsumptionTokensV0)
	if setting.Value != "450000" || setting.Source != configSettingSourceConfigFileV0 {
		t.Fatalf("setting checkpoint threshold=%+v", setting)
	}
}

func TestServerDaemonRunArgsV0UsaSnapshotDeConfigExplicita(t *testing.T) {
	root := t.TempDir()
	stateDir := filepath.Join(root, "state")
	sourcePath := filepath.Join(root, "orquesta.config.json")
	configFile := `{
		"schema_version":"orquesta_config.v0",
		"server":{"state_dir":"` + filepath.ToSlash(stateDir) + `"},
		"autoprogramming":{"checkpoint_only_high_consumption_tokens":450000}
	}`
	if err := os.WriteFile(sourcePath, []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	args, err := serverDaemonRunArgsV0(orquestaserver.ConfigV0{
		StateDir:              stateDir,
		ProjectConfigFilePath: sourcePath,
	})
	if err != nil {
		t.Fatalf("serverDaemonRunArgsV0: %v", err)
	}
	snapshotPath := filepath.Join(stateDir, serverDaemonConfigSnapshotDirV0, serverProjectConfigFileNameV0)
	wantArgs := strings.Join([]string{"run", "--config", snapshotPath}, "\x00")
	if strings.Join(args, "\x00") != wantArgs {
		t.Fatalf("args=%v want=%v", args, wantArgs)
	}
	if err := os.WriteFile(sourcePath, []byte(`{"schema_version":"orquesta_config.v0","autoprogramming":{"checkpoint_only_high_consumption_tokens":1}}`), 0o600); err != nil {
		t.Fatalf("mutate source config: %v", err)
	}
	rawSnapshot, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if !strings.Contains(string(rawSnapshot), `"checkpoint_only_high_consumption_tokens":450000`) {
		t.Fatalf("snapshot no conserva config original: %s", string(rawSnapshot))
	}
}

func TestServerConfigFromEnvV0LeeServerDaemonRuntimeDesdeFicheroCanonicoV0(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := filepath.Join(projectDir, "state-from-file")
	runtimeDir := filepath.Join(projectDir, "runtime-from-file")
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	configFile := `{
		"schema_version":"orquesta_config.v0",
		"server":{
			"addr":"127.0.0.1:18787",
			"state_dir":"` + filepath.ToSlash(stateDir) + `",
			"audit_file":"audit-file.jsonl",
			"audit_disabled":false
		},
		"daemon_logs":{
			"max_bytes":1048576,
			"max_rotated_files":5,
			"retention_days":9,
			"local_raw_enabled":true,
			"local_raw_reason":"local_diagnostic"
		},
		"codex_runtime":{
			"runtime_work_dir":"` + filepath.ToSlash(runtimeDir) + `"
		}
	}`
	if err := os.WriteFile(filepath.Join(projectDir, serverProjectConfigFileNameV0), []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if config.Addr != "127.0.0.1:18787" ||
		config.StateDir != stateDir ||
		config.RuntimeWorkDir != runtimeDir ||
		config.AuditFile != "audit-file.jsonl" ||
		config.AuditDisabled {
		t.Fatalf("config desde fichero inesperada: addr=%q state=%q runtime=%q audit=%q disabled=%v",
			config.Addr, config.StateDir, config.RuntimeWorkDir, config.AuditFile, config.AuditDisabled)
	}
	if config.DaemonLogPolicy.MaxBytes != 1048576 ||
		config.DaemonLogPolicy.MaxRotatedFiles != 5 ||
		config.DaemonLogPolicy.RetentionDays != 9 ||
		!config.DaemonLogPolicy.LocalRawEnabled ||
		config.DaemonLogPolicy.LocalRawReason != "local_diagnostic" {
		t.Fatalf("daemon log policy=%+v", config.DaemonLogPolicy)
	}
	for _, key := range []string{
		envServerAddrV0,
		envServerStateDirV0,
		envServerAuditFileV0,
		envServerAuditDisabledV0,
		envServerDaemonLogMaxBytesV0,
		envServerDaemonLogMaxRotatedV0,
		envServerDaemonLogRetentionDaysV0,
		envServerDaemonLogRawEnabledV0,
		envServerDaemonLogRawReasonV0,
		envCodexRuntimeWorkDirV0,
	} {
		setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, key)
		if setting.Source != configSettingSourceConfigFileV0 {
			t.Fatalf("setting %s=%+v", key, setting)
		}
	}
	stateSetting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envServerStateDirV0)
	runtimeSetting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envCodexRuntimeWorkDirV0)
	if !stateSetting.Sensitive || stateSetting.Value != "server-state-dir-configured" ||
		!runtimeSetting.Sensitive || runtimeSetting.Value != "codex-runtime-workdir-configured" {
		t.Fatalf("settings sensibles state=%+v runtime=%+v", stateSetting, runtimeSetting)
	}
}

func TestServerConfigFromEnvV0LeeHTTPYLifecycleDesdeFicheroCanonicoV0(t *testing.T) {
	projectDir := t.TempDir()
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	configFile := `{
		"schema_version":"orquesta_config.v0",
		"server_http":{
			"read_header_timeout_ms":1111,
			"read_timeout_ms":2222,
			"write_timeout_ms":3333,
			"idle_timeout_ms":4444,
			"max_header_bytes":5555,
			"control_body_max_bytes":6666
		},
		"server_lifecycle":{
			"shutdown_grace_ms":7777
		}
	}`
	if err := os.WriteFile(filepath.Join(projectDir, serverProjectConfigFileNameV0), []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if config.HTTPResourceLimits.ReadHeaderTimeout != 1111*time.Millisecond ||
		config.HTTPResourceLimits.ReadTimeout != 2222*time.Millisecond ||
		config.HTTPResourceLimits.WriteTimeout != 3333*time.Millisecond ||
		config.HTTPResourceLimits.IdleTimeout != 4444*time.Millisecond ||
		config.HTTPResourceLimits.MaxHeaderBytes != 5555 ||
		config.HTTPResourceLimits.ControlBodyBytes != 6666 ||
		config.ShutdownGracePeriod != 7777*time.Millisecond {
		t.Fatalf("http/lifecycle desde fichero inesperado: http=%+v shutdown=%s",
			config.HTTPResourceLimits, config.ShutdownGracePeriod)
	}
	for _, key := range []string{
		envServerReadHeaderTimeoutMSV0,
		envServerReadTimeoutMSV0,
		envServerWriteTimeoutMSV0,
		envServerIdleTimeoutMSV0,
		envServerMaxHeaderBytesV0,
		envServerControlBodyMaxBytesV0,
		envServerShutdownGraceMSV0,
	} {
		setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, key)
		if setting.Source != configSettingSourceConfigFileV0 {
			t.Fatalf("setting %s=%+v", key, setting)
		}
	}
}

func TestServerConfigFromEnvV0EnvExplicitoGanaCampoAFicheroCanonicoV0(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := filepath.Join(projectDir, "state-from-file")
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	t.Setenv(envServerAddrV0, "127.0.0.1:28787")
	t.Setenv(envServerReadHeaderTimeoutMSV0, "9876")
	configFile := `{
		"schema_version":"orquesta_config.v0",
		"server":{
			"addr":"127.0.0.1:18787",
			"state_dir":"` + filepath.ToSlash(stateDir) + `"
		},
		"server_http":{
			"read_header_timeout_ms":1111,
			"read_timeout_ms":2222
		}
	}`
	if err := os.WriteFile(filepath.Join(projectDir, serverProjectConfigFileNameV0), []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if config.Addr != "127.0.0.1:28787" || config.StateDir != stateDir {
		t.Fatalf("config=%+v", config)
	}
	if config.HTTPResourceLimits.ReadHeaderTimeout != 9876*time.Millisecond ||
		config.HTTPResourceLimits.ReadTimeout != 2222*time.Millisecond {
		t.Fatalf("http limits=%+v", config.HTTPResourceLimits)
	}
	addrSetting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envServerAddrV0)
	stateSetting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envServerStateDirV0)
	readHeaderSetting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envServerReadHeaderTimeoutMSV0)
	readSetting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envServerReadTimeoutMSV0)
	if addrSetting.Source != "explicit" ||
		stateSetting.Source != configSettingSourceConfigFileV0 ||
		readHeaderSetting.Source != "explicit" ||
		readSetting.Source != configSettingSourceConfigFileV0 {
		t.Fatalf("addr=%+v state=%+v read_header=%+v read=%+v", addrSetting, stateSetting, readHeaderSetting, readSetting)
	}
}

func TestServerConfigFromEnvV0LeeLimitesCodexYSupervisorDesdeFicheroCanonicoV0(t *testing.T) {
	projectDir := t.TempDir()
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	for _, key := range []string{
		envServerMaxRunsPerTickV0,
		envServerMaxExecutionsPerTickV0,
		envServerQueueLimitV0,
		envServerDrainMaxDispatchesV0,
		envServerDrainMaxCommandsV0,
		envServerDrainMaxOutboxV0,
		envServerDrainMaxExternalWaitsV0,
		envServerIdleSelfImprovementMaxRequestsV0,
		envServerResidentDirectorMaxActionsV0,
		envCodexExecutionModeV0,
		envCodexMaxBatchReadyV0,
		envCodexMaxConcurrencyV0,
		envCodexReasoningEffortV0,
		envCodexMaxExpectedSecondsV0,
		envCodexDirectorWaveAgentsV0,
		envCodexDirectorMaxSubagentsPerAgentV0,
		envCodexDirectorRecursiveAgentBudgetV0,
	} {
		t.Setenv(key, "")
	}
	configFile := `{
		"schema_version":"orquesta_config.v0",
		"server_supervisor":{
			"max_runs_per_tick":31,
			"max_executions_per_tick":32,
			"queue_limit":33,
			"drain_max_dispatches":34,
			"drain_max_commands":35,
			"drain_max_outbox":36,
			"drain_max_external_waits":37
		},
		"server_idle_self_improvement":{"max_requests":5},
		"server_resident_director":{"max_actions":6},
		"codex_runtime":{
			"execution_mode":"parallel",
			"reasoning_effort":"high",
			"max_expected_seconds":41,
			"max_batch_ready":42,
			"max_concurrency":43
		},
		"codex_director":{
			"wave_agents":44,
			"max_subagents_per_agent":7,
			"recursive_agent_budget":45
		}
	}`
	if err := os.WriteFile(filepath.Join(projectDir, serverProjectConfigFileNameV0), []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if config.SupervisorCommand.MaxRunsPerTick != 31 ||
		config.SupervisorCommand.MaxExecutions != 32 ||
		config.SupervisorCommand.DrainLimits.MaxDispatchesPerWait != 34 ||
		config.SupervisorCommand.DrainLimits.MaxCommands != 35 ||
		config.SupervisorCommand.DrainLimits.MaxOutboxPerCycle != 36 ||
		config.SupervisorCommand.DrainLimits.MaxExternalWaits != 37 ||
		config.IdleSelfImprovementMaxRequests != 5 ||
		config.ResidentDirectorMaxActions != 6 {
		t.Fatalf("config limites=%+v idle=%d resident=%d", config.SupervisorCommand, config.IdleSelfImprovementMaxRequests, config.ResidentDirectorMaxActions)
	}
	runtimeConfig := codexRuntimeConfigV0(config, nil)
	if runtimeConfig.ReasoningEffort != "high" ||
		runtimeConfig.MaxBatchReady != 42 ||
		runtimeConfig.MaxConcurrency != 43 ||
		runtimeConfig.ProgressBudget.MaxExpected != 41*time.Second {
		t.Fatalf("runtime_config=%+v", runtimeConfig)
	}
	for key, want := range map[string]string{
		envServerMaxRunsPerTickV0:                 "31",
		envServerMaxExecutionsPerTickV0:           "32",
		envServerQueueLimitV0:                     "33",
		envServerDrainMaxDispatchesV0:             "34",
		envServerDrainMaxCommandsV0:               "35",
		envServerDrainMaxOutboxV0:                 "36",
		envServerDrainMaxExternalWaitsV0:          "37",
		envServerIdleSelfImprovementMaxRequestsV0: "5",
		envServerResidentDirectorMaxActionsV0:     "6",
		envCodexExecutionModeV0:                   codexExecutionModeParallelV0,
		envCodexMaxBatchReadyV0:                   "42",
		envCodexMaxConcurrencyV0:                  "43",
		envCodexReasoningEffortV0:                 "high",
		envCodexMaxExpectedSecondsV0:              "41",
		envCodexDirectorWaveAgentsV0:              "44",
		envCodexDirectorMaxSubagentsPerAgentV0:    "7",
		envCodexDirectorRecursiveAgentBudgetV0:    "45",
	} {
		setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, key)
		if setting.Value != want || setting.Source != configSettingSourceConfigFileV0 {
			t.Fatalf("setting %s=%+v want value=%q source=config_file", key, setting, want)
		}
	}
}

func TestServerConfigFromEnvV0EnvExplicitoGanaLimitesCodexYSupervisorV0(t *testing.T) {
	projectDir := t.TempDir()
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	t.Setenv(envServerMaxRunsPerTickV0, "9")
	t.Setenv(envCodexReasoningEffortV0, "low")
	t.Setenv(envCodexMaxExpectedSecondsV0, "19")
	t.Setenv(envCodexDirectorWaveAgentsV0, "8")
	configFile := `{
		"schema_version":"orquesta_config.v0",
		"server_supervisor":{"max_runs_per_tick":31,"max_executions_per_tick":32},
		"codex_runtime":{"reasoning_effort":"high","max_expected_seconds":41,"max_batch_ready":42},
		"codex_director":{"wave_agents":44}
	}`
	if err := os.WriteFile(filepath.Join(projectDir, serverProjectConfigFileNameV0), []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	runtimeConfig := codexRuntimeConfigV0(config, nil)
	if config.SupervisorCommand.MaxRunsPerTick != 9 ||
		config.SupervisorCommand.MaxExecutions != 32 ||
		runtimeConfig.ReasoningEffort != "low" ||
		runtimeConfig.ProgressBudget.MaxExpected != 19*time.Second {
		t.Fatalf("config=%+v runtime=%+v", config.SupervisorCommand, runtimeConfig)
	}
	for key, wantSource := range map[string]string{
		envServerMaxRunsPerTickV0:       "explicit",
		envServerMaxExecutionsPerTickV0: configSettingSourceConfigFileV0,
		envCodexReasoningEffortV0:       "explicit",
		envCodexMaxExpectedSecondsV0:    "explicit",
		envCodexMaxBatchReadyV0:         configSettingSourceConfigFileV0,
		envCodexDirectorWaveAgentsV0:    "explicit",
	} {
		setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, key)
		if setting.Source != wantSource {
			t.Fatalf("setting %s=%+v want source=%s", key, setting, wantSource)
		}
	}
}

func TestServerConfigFromEnvV0RechazaAuditFileInvalidoDesdeFicheroCanonicoV0(t *testing.T) {
	projectDir := t.TempDir()
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	if err := os.WriteFile(
		filepath.Join(projectDir, serverProjectConfigFileNameV0),
		[]byte(`{"schema_version":"orquesta_config.v0","server":{"audit_file":"logs/audit.json"}}`),
		0o600,
	); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	_, err := serverConfigFromEnvV0()
	if err == nil || !strings.Contains(err.Error(), "audit_file invalido") {
		t.Fatalf("serverConfigFromEnvV0 err=%v", err)
	}
}

func TestServerConfigFromEnvV0RechazaFicheroCanonicoConSchemaInvalidoV0(t *testing.T) {
	projectDir := t.TempDir()
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	if err := os.WriteFile(
		filepath.Join(projectDir, serverProjectConfigFileNameV0),
		[]byte(`{"schema_version":"otro","autoprogramming":{"checkpoint_only_high_consumption_tokens":450000}}`),
		0o600,
	); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	_, err := serverConfigFromEnvV0()
	if err == nil || !strings.Contains(err.Error(), configFileUnsupportedSchemaCodeV0) {
		t.Fatalf("serverConfigFromEnvV0 err=%v", err)
	}
}

func TestBuildStackFromEnvV0CableaPromocionMaterialSinACKV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0 default: %v", err)
	}
	stack, err := buildStackFromEnvWithGoalBackendV0(config, serverCodexGoalBackendV0{})
	if err != nil {
		t.Fatalf("buildStackFromEnvWithGoalBackendV0 default: %v", err)
	}
	if !stack.PromoteMaterializedArtifactWithoutAck {
		t.Fatalf("promote without ack default=%v", stack.PromoteMaterializedArtifactWithoutAck)
	}

	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	t.Setenv(envCodexPromoteMaterializedArtifactWithoutAckV0, "false")
	config, err = serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0 optout: %v", err)
	}
	stack, err = buildStackFromEnvWithGoalBackendV0(config, serverCodexGoalBackendV0{})
	if err != nil {
		t.Fatalf("buildStackFromEnvWithGoalBackendV0 optout: %v", err)
	}
	if stack.PromoteMaterializedArtifactWithoutAck {
		t.Fatalf("promote without ack optout=%v", stack.PromoteMaterializedArtifactWithoutAck)
	}
}

func TestBuildStackFromEnvV0CableaWatcherResultMaterializadoConBackendGoalV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	relay := &serverSupervisorWakeupRelayV0{}
	backend := serverCodexGoalBackendV0{
		Observer: serverCodexAppServerGoalBackendV0{Protocol: &fakeCodexAppServerProtocolV0{}},
	}
	stack, err := buildStackFromEnvWithGoalBackendV0(config, backend, relay)
	if err != nil {
		t.Fatalf("buildStackFromEnvWithGoalBackendV0: %v", err)
	}
	if stack.GoalMaterializedResultWatcher == nil {
		t.Fatalf("watcher result materializado no cableado")
	}
	if workers := serverBackgroundWorkersFromStackV0(stack); len(workers) != 1 {
		t.Fatalf("background workers=%d", len(workers))
	}

	config.GoalObserverEnabledConfigured = true
	config.GoalObserverEnabled = false
	stack, err = buildStackFromEnvWithGoalBackendV0(config, backend, relay)
	if err != nil {
		t.Fatalf("buildStackFromEnvWithGoalBackendV0 disabled: %v", err)
	}
	if stack.GoalMaterializedResultWatcher != nil {
		t.Fatalf("watcher cableado pese a goal observer disabled")
	}
}

func TestServerConfigFromEnvV0PerfilAutonomiaActivaDirectorResidenteV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	t.Setenv(envServerAutonomyEnabledV0, "true")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if !config.ResidentDirectorEnabled {
		t.Fatalf("perfil autonomia no activo director residente: %+v", config)
	}
	if got := effectiveSettingValueForTestV0(config.EffectiveConfig.Settings, envServerAutonomyEnabledV0); got != "true" {
		t.Fatalf("%s=%q want true", envServerAutonomyEnabledV0, got)
	}
	if got := effectiveSettingValueForTestV0(config.EffectiveConfig.Settings, envServerResidentDirectorEnabledV0); got != "true" {
		t.Fatalf("%s=%q want true", envServerResidentDirectorEnabledV0, got)
	}
}

func TestServerConfigFromEnvV0ContextoOPESActivaDirectorResidenteV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	t.Setenv(envOPESProjectWorkDirV0, "/tmp/opes-workspace")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if !config.ResidentDirectorEnabled {
		t.Fatalf("contexto OPES no activo director residente: %+v", config)
	}
	if got := effectiveSettingValueForTestV0(config.EffectiveConfig.Settings, envServerAutonomyEnabledV0); got != "true" {
		t.Fatalf("%s=%q want true", envServerAutonomyEnabledV0, got)
	}
	if got := effectiveSettingValueForTestV0(config.EffectiveConfig.Settings, envServerResidentDirectorEnabledV0); got != "true" {
		t.Fatalf("%s=%q want true", envServerResidentDirectorEnabledV0, got)
	}
}

func TestServerConfigFromEnvV0PermiteOPESConDirectorResidenteApagadoExplicitoV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	t.Setenv(envOPESProjectWorkDirV0, "/tmp/opes-workspace")
	t.Setenv(envServerResidentDirectorEnabledV0, "false")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if config.ResidentDirectorEnabled {
		t.Fatalf("override explicito no desactivo director residente en OPES: %+v", config)
	}
	if got := effectiveSettingValueForTestV0(config.EffectiveConfig.Settings, envServerAutonomyEnabledV0); got != "true" {
		t.Fatalf("%s=%q want true", envServerAutonomyEnabledV0, got)
	}
	if got := effectiveSettingValueForTestV0(config.EffectiveConfig.Settings, envServerResidentDirectorEnabledV0); got != "false" {
		t.Fatalf("%s=%q want false", envServerResidentDirectorEnabledV0, got)
	}
}

func TestServerConfigFromEnvV0DirectorResidenteExplicitoGanaAlPerfilAutonomiaV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	t.Setenv(envServerAutonomyEnabledV0, "true")
	t.Setenv(envServerResidentDirectorEnabledV0, "false")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if config.ResidentDirectorEnabled {
		t.Fatalf("override explicito no desactivo director residente: %+v", config)
	}
	if got := effectiveSettingValueForTestV0(config.EffectiveConfig.Settings, envServerAutonomyEnabledV0); got != "true" {
		t.Fatalf("%s=%q want true", envServerAutonomyEnabledV0, got)
	}
	if got := effectiveSettingValueForTestV0(config.EffectiveConfig.Settings, envServerResidentDirectorEnabledV0); got != "false" {
		t.Fatalf("%s=%q want false", envServerResidentDirectorEnabledV0, got)
	}
}

func TestServerConfigFromEnvV0PublicaConfiguracionEfectivaCanonica(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", t.TempDir())
	t.Setenv("ORQUESTA_SERVER_MAX_RUNS_PER_TICK", "10")
	t.Setenv("ORQUESTA_SERVER_MAX_EXECUTIONS_PER_TICK", "10")
	t.Setenv("ORQUESTA_SERVER_QUEUE_LIMIT", "10")
	t.Setenv("ORQUESTA_SERVER_DRAIN_MAX_DISPATCHES", "10")
	t.Setenv("ORQUESTA_SERVER_DRAIN_MAX_OUTBOX", "10")
	t.Setenv("ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_TARGET_QUEUE", "25")
	t.Setenv("ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_MAX_REQUESTS", "10")
	t.Setenv("ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED", "true")
	t.Setenv("ORQUESTA_SERVER_RESIDENT_DIRECTOR_MAX_ACTIONS", "7")
	t.Setenv("ORQUESTA_SERVER_SELF_WATCHDOG_CPU_HIGH_PERCENT", "88")
	t.Setenv("ORQUESTA_SERVER_SELF_WATCHDOG_SUSTAINED_SECONDS", "180")
	t.Setenv("ORQUESTA_SERVER_SELF_WATCHDOG_NO_PROGRESS_SECONDS", "90")
	t.Setenv("ORQUESTA_CODEX_MAX_BATCH_READY", "10")
	t.Setenv("ORQUESTA_CODEX_MAX_CONCURRENCY", "10")
	t.Setenv("ORQUESTA_CODEX_REASONING_EFFORT", "high")
	t.Setenv("ORQUESTA_CODEX_DIRECTOR_MAX_SUBAGENTS_PER_AGENT", "6")
	t.Setenv(envCodebaseBrokerProviderKindV0, "fallback_rg")
	t.Setenv(envCodebaseBrokerExternalIndexerEnabledV0, "false")
	t.Setenv(envCodebaseBrokerMaxConcurrentV0, "3")
	t.Setenv(envCodebaseBrokerTimeoutMSV0, "1500")
	t.Setenv(envCodebaseBrokerCommandV0, "codebase-memory-mcp-test")
	t.Setenv(envCodebaseBrokerProjectNameV0, "orquesta-test-index")
	t.Setenv(envOPESBridgeWaitResidentSecondsV0, "15")
	t.Setenv(envOPESBridgeWaitResidentIntervalMSV0, "250")
	t.Setenv("ORQUESTA_HERMES_ENABLED", "1")
	t.Setenv("ORQUESTA_HERMES_BASE_URL", "https://hermes.local/mcp")
	t.Setenv("ORQUESTA_HERMES_API_KEY", "secret-hermes-test")
	t.Setenv("ORQUESTA_HERMES_STATUS_TOOL", "hermes.status")
	t.Setenv("ORQUESTA_HERMES_STATUS_CONNECTOR_REF", "hermes-status-ref")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	settings := config.EffectiveConfig.Settings
	for key, want := range map[string]string{
		"ORQUESTA_SERVER_MAX_RUNS_PER_TICK":                  "10",
		"ORQUESTA_SERVER_MAX_EXECUTIONS_PER_TICK":            "10",
		"ORQUESTA_SERVER_QUEUE_LIMIT":                        "10",
		"ORQUESTA_SERVER_DRAIN_MAX_DISPATCHES":               "10",
		"ORQUESTA_SERVER_DRAIN_MAX_OUTBOX":                   "10",
		"ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_TARGET_QUEUE": "25",
		"ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_MAX_REQUESTS": "10",
		"ORQUESTA_SERVER_AUTONOMY_ENABLED":                   "false",
		"ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED":          "true",
		"ORQUESTA_SERVER_RESIDENT_DIRECTOR_MAX_ACTIONS":      "7",
		"ORQUESTA_SERVER_SELF_WATCHDOG_DISABLED":             "false",
		"ORQUESTA_SERVER_SELF_WATCHDOG_CPU_HIGH_PERCENT":     "88",
		"ORQUESTA_SERVER_SELF_WATCHDOG_SUSTAINED_SECONDS":    "180",
		"ORQUESTA_SERVER_SELF_WATCHDOG_NO_PROGRESS_SECONDS":  "90",
		"ORQUESTA_CODEX_MAX_BATCH_READY":                     "10",
		"ORQUESTA_CODEX_MAX_CONCURRENCY":                     "10",
		"ORQUESTA_CODEX_REASONING_EFFORT":                    "high",
		envCodexPromoteMaterializedArtifactWithoutAckV0:      "true",
		"ORQUESTA_CODEX_DIRECTOR_MAX_SUBAGENTS_PER_AGENT":    "6",
		envCodebaseBrokerProviderKindV0:                      "fallback_rg",
		envCodebaseBrokerExternalIndexerEnabledV0:            "false",
		envCodebaseBrokerMaxConcurrentV0:                     "3",
		envCodebaseBrokerTimeoutMSV0:                         "1500",
		envCodebaseBrokerCommandV0:                           codebaseBrokerConfiguredCommandRefV0,
		envCodebaseBrokerProjectNameV0:                       "orquesta-test-index",
		envOPESBridgeWaitResidentSecondsV0:                   "15",
		envOPESBridgeWaitResidentIntervalMSV0:                "250",
		"ORQUESTA_HERMES_ENABLED":                            "true",
		"ORQUESTA_HERMES_BASE_URL":                           "hermes-base-url-configured",
		"ORQUESTA_HERMES_API_KEY":                            "hermes-api-key-configured",
		"ORQUESTA_HERMES_STATUS_TOOL":                        "hermes.status",
		"ORQUESTA_HERMES_STATUS_CONNECTOR_REF":               "hermes-status-ref",
	} {
		got := effectiveSettingValueForTestV0(settings, key)
		if got != want {
			t.Fatalf("%s=%q want %q settings=%+v", key, got, want, settings)
		}
	}
	for _, key := range []string{"ORQUESTA_HERMES_BASE_URL", "ORQUESTA_HERMES_API_KEY"} {
		setting := effectiveSettingForTestV0(settings, key)
		if !setting.Sensitive {
			t.Fatalf("%s debe quedar marcado como sensible: %+v", key, setting)
		}
	}
}

func TestServerConfigFromEnvV0SeparaProyectoPrincipalDeAutomejoraIdleV0(t *testing.T) {
	root := t.TempDir()
	opesDir := filepath.Join(root, "OPES")
	orquestaDir := filepath.Join(root, "orquesta")
	t.Setenv(envCodexProjectWorkDirV0, opesDir)
	t.Setenv(envOPESProjectWorkDirV0, opesDir)
	t.Setenv(envServerIdleSelfImprovementProjectWorkDirV0, orquestaDir)

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if config.ProjectWorkDir != opesDir {
		t.Fatalf("project_work_dir=%q want %q", config.ProjectWorkDir, opesDir)
	}
	if config.IdleSelfImprovementProjectWorkDir != orquestaDir {
		t.Fatalf("idle_self_improvement_project_work_dir=%q want %q", config.IdleSelfImprovementProjectWorkDir, orquestaDir)
	}
	if config.IdleSelfImprovementDisabled {
		t.Fatalf("automejora idle no debe desactivarse si tiene workdir separado: %+v", config)
	}
	setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envServerIdleSelfImprovementProjectWorkDirV0)
	if setting.Value != "idle-self-improvement-project-workdir-configured" || !setting.Sensitive {
		t.Fatalf("idle self improvement project workdir setting=%+v", setting)
	}
	rawEffectiveConfig, err := json.Marshal(config.EffectiveConfig)
	if err != nil {
		t.Fatalf("marshal effective config: %v", err)
	}
	if strings.Contains(string(rawEffectiveConfig), opesDir) ||
		strings.Contains(string(rawEffectiveConfig), orquestaDir) {
		t.Fatalf("effective_config filtra rutas locales: %s", string(rawEffectiveConfig))
	}
}

func TestServerStackSupervisorV0PreparaAutomejoraConWorkdirSeparadoSinMutarStackV0(t *testing.T) {
	root := t.TempDir()
	opesDir := filepath.Join(root, "OPES")
	orquestaDir := filepath.Join(root, "orquesta")
	stack := orquestaappcodexstack.StackV0{
		Codex: orquestaappcodexstack.CodexRuntimeConfigV0{
			ProjectWorkDir: opesDir,
		},
	}
	supervisor := serverStackSupervisorV0{
		stack:          &stack,
		projectWorkDir: orquestaDir,
	}

	prepareStack := supervisor.autoprogrammingPrepareStackV0()

	if prepareStack == nil {
		t.Fatalf("prepare stack nil")
	}
	if prepareStack.Codex.ProjectWorkDir != orquestaDir {
		t.Fatalf("prepare project_work_dir=%q want %q", prepareStack.Codex.ProjectWorkDir, orquestaDir)
	}
	if stack.Codex.ProjectWorkDir != opesDir {
		t.Fatalf("stack principal mutado: %q want %q", stack.Codex.ProjectWorkDir, opesDir)
	}
}

func TestServerConfigFromEnvV0PublicaEgressSanitizerCanonicoRedactadoV0(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", t.TempDir())
	rawModelPath := "/home/alberto/private/models/openai-privacy-filter.gguf"
	rawSidecarCommand := "/home/alberto/bin/privacy-filter --model " + rawModelPath
	rawSidecarEndpoint := "http://127.0.0.1:17777/filter?token=secret-endpoint"
	t.Setenv("ORQUESTA_EGRESS_SANITIZER_ENABLED", "true")
	t.Setenv("ORQUESTA_EGRESS_SANITIZER_REF", "sanitizer-ref-server-test")
	t.Setenv("ORQUESTA_EGRESS_SANITIZER_LOCAL_MODEL_ENABLED", "true")
	t.Setenv("ORQUESTA_EGRESS_SANITIZER_LOCAL_MODEL_REF", "model-ref-sensitive-test")
	t.Setenv("ORQUESTA_EGRESS_SANITIZER_LOCAL_MODEL_PATH", rawModelPath)
	t.Setenv("ORQUESTA_EGRESS_SANITIZER_LOCAL_RUNTIME_REF", "runtime-ref-sensitive-test")
	t.Setenv("ORQUESTA_EGRESS_SANITIZER_LOCAL_EVIDENCE_REF", "evidence-ref-sensitive-test")
	t.Setenv("ORQUESTA_EGRESS_SANITIZER_SIDECAR_ENABLED", "true")
	t.Setenv("ORQUESTA_EGRESS_SANITIZER_SIDECAR_REF", "sidecar-ref-server-test")
	t.Setenv("ORQUESTA_EGRESS_SANITIZER_SIDECAR_ADAPTER_REF", "adapter-ref-server-test")
	t.Setenv("ORQUESTA_EGRESS_SANITIZER_SIDECAR_TRANSPORT_REF", "transport-ref-server-test")
	t.Setenv("ORQUESTA_EGRESS_SANITIZER_SIDECAR_EVIDENCE_REF", "evidence-ref-sidecar-test")
	t.Setenv("ORQUESTA_EGRESS_SANITIZER_SIDECAR_COMMAND", rawSidecarCommand)
	t.Setenv("ORQUESTA_EGRESS_SANITIZER_SIDECAR_LOCAL_ENDPOINT", rawSidecarEndpoint)

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	settings := config.EffectiveConfig.Settings
	for key, want := range map[string]string{
		"ORQUESTA_EGRESS_SANITIZER_ENABLED":                "true",
		"ORQUESTA_EGRESS_SANITIZER_REF":                    "sanitizer-ref-server-test",
		"ORQUESTA_EGRESS_SANITIZER_LOCAL_MODEL_ENABLED":    "true",
		"ORQUESTA_EGRESS_SANITIZER_LOCAL_MODEL_REF":        "egress-sanitizer-local-model-ref-configured",
		"ORQUESTA_EGRESS_SANITIZER_LOCAL_MODEL_PATH":       "egress-sanitizer-local-model-path-configured",
		"ORQUESTA_EGRESS_SANITIZER_LOCAL_RUNTIME_REF":      "egress-sanitizer-local-runtime-ref-configured",
		"ORQUESTA_EGRESS_SANITIZER_LOCAL_EVIDENCE_REF":     "egress-sanitizer-local-evidence-ref-configured",
		"ORQUESTA_EGRESS_SANITIZER_SIDECAR_ENABLED":        "true",
		"ORQUESTA_EGRESS_SANITIZER_SIDECAR_REF":            "sidecar-ref-server-test",
		"ORQUESTA_EGRESS_SANITIZER_SIDECAR_ADAPTER_REF":    "adapter-ref-server-test",
		"ORQUESTA_EGRESS_SANITIZER_SIDECAR_TRANSPORT_REF":  "transport-ref-server-test",
		"ORQUESTA_EGRESS_SANITIZER_SIDECAR_EVIDENCE_REF":   "evidence-ref-sidecar-test",
		"ORQUESTA_EGRESS_SANITIZER_SIDECAR_COMMAND":        "egress-sanitizer-sidecar-command-configured",
		"ORQUESTA_EGRESS_SANITIZER_SIDECAR_LOCAL_ENDPOINT": "egress-sanitizer-sidecar-local-endpoint-configured",
	} {
		if got := effectiveSettingValueForTestV0(settings, key); got != want {
			t.Fatalf("%s=%q want %q settings=%+v", key, got, want, settings)
		}
	}
	for _, key := range []string{
		"ORQUESTA_EGRESS_SANITIZER_LOCAL_MODEL_REF",
		"ORQUESTA_EGRESS_SANITIZER_LOCAL_MODEL_PATH",
		"ORQUESTA_EGRESS_SANITIZER_LOCAL_RUNTIME_REF",
		"ORQUESTA_EGRESS_SANITIZER_LOCAL_EVIDENCE_REF",
		"ORQUESTA_EGRESS_SANITIZER_SIDECAR_COMMAND",
		"ORQUESTA_EGRESS_SANITIZER_SIDECAR_LOCAL_ENDPOINT",
	} {
		if setting := effectiveSettingForTestV0(settings, key); !setting.Sensitive {
			t.Fatalf("%s debe quedar marcado como sensible: %+v", key, setting)
		}
	}
	rawEffectiveConfig, err := json.Marshal(config.EffectiveConfig)
	if err != nil {
		t.Fatalf("marshal effective config: %v", err)
	}
	for _, forbidden := range []string{rawModelPath, rawSidecarCommand, rawSidecarEndpoint} {
		if strings.Contains(string(rawEffectiveConfig), forbidden) {
			t.Fatalf("effective_config filtra valor crudo %q: %s", forbidden, string(rawEffectiveConfig))
		}
	}
	publicConfig, hidden := orquestaserver.PublicServerEffectiveConfigV0(config.EffectiveConfig)
	if got := effectiveSettingValueForTestV0(publicConfig.Settings, "ORQUESTA_EGRESS_SANITIZER_LOCAL_MODEL_REF"); got != orquestaserver.ServerStatusConfigHiddenValueV0 {
		t.Fatalf("public model ref=%q want hidden", got)
	}
	for _, wantHidden := range []string{
		"effective_config.ORQUESTA_EGRESS_SANITIZER_LOCAL_MODEL_REF",
		"effective_config.ORQUESTA_EGRESS_SANITIZER_LOCAL_MODEL_PATH",
		"effective_config.ORQUESTA_EGRESS_SANITIZER_SIDECAR_COMMAND",
		"effective_config.ORQUESTA_EGRESS_SANITIZER_SIDECAR_LOCAL_ENDPOINT",
	} {
		if !containsStringV0(hidden, wantHidden) {
			t.Fatalf("hidden no contiene %s: %v", wantHidden, hidden)
		}
	}
}

func TestServerConfigFromEnvV0LeeEgressSanitizerDesdeFicheroCanonicoV0(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := filepath.Join(projectDir, "state")
	runtimeDir := filepath.Join(projectDir, "runtime")
	rawModelPath := "/home/alberto/private/models/openai-privacy-filter.gguf"
	rawSidecarCommand := "/home/alberto/bin/privacy-filter --model " + rawModelPath
	rawSidecarEndpoint := "http://127.0.0.1:17777/filter"
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	configFile := `{
		"schema_version":"orquesta_config.v0",
		"server":{"state_dir":"` + filepath.ToSlash(stateDir) + `"},
		"codex_runtime":{"runtime_work_dir":"` + filepath.ToSlash(runtimeDir) + `"},
		"egress_sanitizer":{
			"enabled":true,
			"sanitizer_ref":"sanitizer-ref-file-test",
			"local_model":{
				"enabled":true,
				"model_ref":"model-ref-sensitive-file",
				"model_path":"` + rawModelPath + `",
				"runtime_ref":"runtime-ref-sensitive-file",
				"evidence_ref":"evidence-ref-sensitive-file"
			},
			"sidecar":{
				"enabled":true,
				"sidecar_ref":"sidecar-ref-file-test",
				"adapter_ref":"adapter-ref-file-test",
				"transport_ref":"transport-ref-file-test",
				"evidence_ref":"evidence-ref-sidecar-file",
				"command":"` + rawSidecarCommand + `",
				"local_endpoint":"` + rawSidecarEndpoint + `"
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(projectDir, serverProjectConfigFileNameV0), []byte(configFile), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	settings := config.EffectiveConfig.Settings
	for key, want := range map[string]string{
		envEgressSanitizerEnabledV0:              "true",
		envEgressSanitizerRefV0:                  "sanitizer-ref-file-test",
		envEgressSanitizerLocalModelEnabledV0:    "true",
		envEgressSanitizerLocalModelRefV0:        "egress-sanitizer-local-model-ref-configured",
		envEgressSanitizerLocalModelPathV0:       "egress-sanitizer-local-model-path-configured",
		envEgressSanitizerLocalRuntimeRefV0:      "egress-sanitizer-local-runtime-ref-configured",
		envEgressSanitizerLocalEvidenceRefV0:     "egress-sanitizer-local-evidence-ref-configured",
		envEgressSanitizerSidecarEnabledV0:       "true",
		envEgressSanitizerSidecarRefV0:           "sidecar-ref-file-test",
		envEgressSanitizerSidecarAdapterRefV0:    "adapter-ref-file-test",
		envEgressSanitizerSidecarTransportRefV0:  "transport-ref-file-test",
		envEgressSanitizerSidecarEvidenceRefV0:   "evidence-ref-sidecar-file",
		envEgressSanitizerSidecarCommandV0:       "egress-sanitizer-sidecar-command-configured",
		envEgressSanitizerSidecarLocalEndpointV0: "egress-sanitizer-sidecar-local-endpoint-configured",
	} {
		setting := effectiveSettingForTestV0(settings, key)
		if setting.Value != want || setting.Source != configSettingSourceConfigFileV0 {
			t.Fatalf("setting %s=%+v want value=%q source=config_file", key, setting, want)
		}
	}
	rawEffectiveConfig, err := json.Marshal(config.EffectiveConfig)
	if err != nil {
		t.Fatalf("marshal effective config: %v", err)
	}
	for _, forbidden := range []string{rawModelPath, rawSidecarCommand, rawSidecarEndpoint} {
		if strings.Contains(string(rawEffectiveConfig), forbidden) {
			t.Fatalf("effective_config filtra valor crudo %q: %s", forbidden, string(rawEffectiveConfig))
		}
	}
	stack, err := buildStackFromEnvV0(config)
	if err != nil {
		t.Fatalf("buildStackFromEnvV0: %v", err)
	}
	if !stack.EgressSanitizer.Enabled ||
		!stack.EgressSanitizer.Sidecar.Enabled ||
		stack.EgressSanitizer.SanitizerRef != "sanitizer-ref-file-test" ||
		stack.EgressSanitizer.Sidecar.SidecarRef != "sidecar-ref-file-test" ||
		!stack.EgressSanitizer.Sidecar.CommandConfigured ||
		!stack.EgressSanitizer.Sidecar.LocalEndpointConfigured ||
		stack.EgressSanitizer.Sidecar.Port == nil {
		t.Fatalf("egress sanitizer desde fichero=%+v", stack.EgressSanitizer)
	}
}

func TestServerConfigFromEnvV0PublicaOPESSpeechSynthesisPreflightRedactadoV0(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", t.TempDir())
	rawWorkdir := "/home/alberto/Trabajo/USO/web"
	rawCommand := "/home/alberto/bin/opes_audio_private --token secreto --help"
	t.Setenv(envOPESBridgeSpeechSynthesisToolWorkDirV0, rawWorkdir)
	t.Setenv(envOPESBridgeSpeechSynthesisToolCommandV0, rawCommand)
	t.Setenv(envOPESBridgeSpeechSynthesisToolPreflightTimeoutSecondsV0, "7")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	settings := config.EffectiveConfig.Settings
	if got := effectiveSettingValueForTestV0(settings, envOPESBridgeSpeechSynthesisToolWorkDirV0); got != "opes-speech-synthesis-tool-workdir-configured" {
		t.Fatalf("tool workdir=%q settings=%+v", got, settings)
	}
	if got := effectiveSettingValueForTestV0(settings, envOPESBridgeSpeechSynthesisToolCommandV0); got != "opes-speech-synthesis-tool-command-configured" {
		t.Fatalf("tool command=%q settings=%+v", got, settings)
	}
	if got := effectiveSettingValueForTestV0(settings, envOPESBridgeSpeechSynthesisToolPreflightTimeoutSecondsV0); got != "7" {
		t.Fatalf("tool preflight timeout=%q settings=%+v", got, settings)
	}
	for _, key := range []string{
		envOPESBridgeSpeechSynthesisToolWorkDirV0,
		envOPESBridgeSpeechSynthesisToolCommandV0,
	} {
		if setting := effectiveSettingForTestV0(settings, key); !setting.Sensitive {
			t.Fatalf("%s debe quedar marcado como sensible: %+v", key, setting)
		}
	}
	rawEffectiveConfig, err := json.Marshal(config.EffectiveConfig)
	if err != nil {
		t.Fatalf("marshal effective config: %v", err)
	}
	for _, forbidden := range []string{rawWorkdir, rawCommand} {
		if strings.Contains(string(rawEffectiveConfig), forbidden) {
			t.Fatalf("effective_config filtra valor crudo %q: %s", forbidden, string(rawEffectiveConfig))
		}
	}
}

func TestBuildStackFromEnvV0CableaEgressSanitizerCanonicoV0(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", t.TempDir())
	t.Setenv("ORQUESTA_SERVER_STATE_DIR", t.TempDir())
	t.Setenv("ORQUESTA_CODEX_RUNTIME_WORKDIR", t.TempDir())
	t.Setenv("ORQUESTA_EGRESS_SANITIZER_ENABLED", "true")
	t.Setenv("ORQUESTA_EGRESS_SANITIZER_REF", "sanitizer-ref-stack-test")
	t.Setenv("ORQUESTA_EGRESS_SANITIZER_SIDECAR_ENABLED", "true")
	t.Setenv("ORQUESTA_EGRESS_SANITIZER_SIDECAR_REF", "sidecar-ref-stack-test")
	t.Setenv("ORQUESTA_EGRESS_SANITIZER_SIDECAR_ADAPTER_REF", "adapter-ref-stack-test")
	t.Setenv("ORQUESTA_EGRESS_SANITIZER_SIDECAR_TRANSPORT_REF", "transport-ref-stack-test")
	t.Setenv("ORQUESTA_EGRESS_SANITIZER_SIDECAR_EVIDENCE_REF", "evidence-ref-stack-test")
	t.Setenv("ORQUESTA_EGRESS_SANITIZER_SIDECAR_COMMAND", "/home/alberto/bin/privacy-filter")
	t.Setenv("ORQUESTA_EGRESS_SANITIZER_SIDECAR_LOCAL_ENDPOINT", "http://127.0.0.1:17777/filter")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	stack, err := buildStackFromEnvV0(config)
	if err != nil {
		t.Fatalf("buildStackFromEnvV0: %v", err)
	}
	if !stack.EgressSanitizer.Enabled ||
		stack.EgressSanitizer.SanitizerRef != "sanitizer-ref-stack-test" ||
		stack.EgressSanitizer.ProviderRef != orquestaappcodexstack.EgressSanitizerProviderOpenAIPrivacyLocalV0 ||
		!stack.EgressSanitizer.Sidecar.Enabled ||
		stack.EgressSanitizer.Sidecar.SidecarRef != "sidecar-ref-stack-test" ||
		!stack.EgressSanitizer.Sidecar.CommandConfigured ||
		!stack.EgressSanitizer.Sidecar.LocalEndpointConfigured {
		t.Fatalf("egress sanitizer=%+v", stack.EgressSanitizer)
	}
	if stack.EgressSanitizer.Sidecar.Port == nil {
		t.Fatalf("BuildStackFromEnvV0 debe cablear el puerto HTTP local del sidecar")
	}
	provider := runtimeProviderForTestV0(stack.ProviderRuntimes, orquestaappcodexstack.EgressSanitizerProviderOpenAIPrivacyLocalV0)
	if !provider.Enabled || !provider.CommandConfigured || !provider.LocalEndpointConfigured || provider.ModelRef != "" {
		t.Fatalf("provider egress sanitizer=%+v", provider)
	}
	if stack.Codex.ContextSanitizer == nil {
		t.Fatalf("BuildStackV0 debe activar sanitizer local en contexto/egress")
	}
	if _, ok := stack.Codex.ContextSanitizer.(orquestaappcodexstack.EgressSanitizerContextSanitizerV0); !ok {
		t.Fatalf("BuildStackV0 debe inyectar el wrapper egress sanitizer, got %T", stack.Codex.ContextSanitizer)
	}
}

func TestServerConfigFromEnvV0UsaCapacidadCanonicaAmpliaSetentaPadresSeisHijos(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", t.TempDir())
	t.Setenv("ORQUESTA_SERVER_MAX_RUNS_PER_TICK", "")
	t.Setenv("ORQUESTA_SERVER_MAX_EXECUTIONS_PER_TICK", "")
	t.Setenv("ORQUESTA_SERVER_QUEUE_LIMIT", "")
	t.Setenv("ORQUESTA_CODEX_MAX_BATCH_READY", "")
	t.Setenv("ORQUESTA_CODEX_MAX_CONCURRENCY", "")
	t.Setenv("ORQUESTA_CODEX_DIRECTOR_WAVE_AGENTS", "")
	t.Setenv("ORQUESTA_CODEX_DIRECTOR_MAX_SUBAGENTS_PER_AGENT", "")
	t.Setenv("ORQUESTA_CODEX_DIRECTOR_RECURSIVE_AGENT_BUDGET", "")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if config.SupervisorCommand.MaxRunsPerTick != 70 ||
		config.SupervisorCommand.MaxExecutions != 70 {
		t.Fatalf("supervisor capacity=%d/%d want 70/70", config.SupervisorCommand.MaxRunsPerTick, config.SupervisorCommand.MaxExecutions)
	}
	settings := config.EffectiveConfig.Settings
	for key, want := range map[string]string{
		"ORQUESTA_CODEX_MAX_BATCH_READY":                  "70",
		"ORQUESTA_CODEX_MAX_CONCURRENCY":                  "70",
		"ORQUESTA_SERVER_QUEUE_LIMIT":                     "70",
		"ORQUESTA_CODEX_DIRECTOR_WAVE_AGENTS":             "70",
		"ORQUESTA_CODEX_DIRECTOR_MAX_SUBAGENTS_PER_AGENT": "6",
		"ORQUESTA_CODEX_DIRECTOR_RECURSIVE_AGENT_BUDGET":  "70",
	} {
		if got := effectiveSettingValueForTestV0(settings, key); got != want {
			t.Fatalf("%s=%q want %q settings=%+v", key, got, want, settings)
		}
	}
}

func TestServerConfigFromEnvV0ModoSerialCapaAgentesCodexV0(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", t.TempDir())
	t.Setenv("ORQUESTA_CODEX_EXECUTION_MODE", "serial")
	t.Setenv("ORQUESTA_SERVER_MAX_RUNS_PER_TICK", "20")
	t.Setenv("ORQUESTA_SERVER_MAX_EXECUTIONS_PER_TICK", "20")
	t.Setenv("ORQUESTA_SERVER_DRAIN_MAX_DISPATCHES", "20")
	t.Setenv("ORQUESTA_SERVER_DRAIN_MAX_COMMANDS", "20")
	t.Setenv("ORQUESTA_SERVER_DRAIN_MAX_OUTBOX", "20")
	t.Setenv("ORQUESTA_CODEX_MAX_BATCH_READY", "20")
	t.Setenv("ORQUESTA_CODEX_MAX_CONCURRENCY", "20")
	t.Setenv("ORQUESTA_CODEX_DIRECTOR_WAVE_AGENTS", "20")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if config.SupervisorCommand.MaxRunsPerTick != 1 ||
		config.SupervisorCommand.MaxExecutions != 1 ||
		config.SupervisorCommand.DrainLimits.MaxDispatchesPerWait != 1 ||
		config.SupervisorCommand.DrainLimits.MaxCommands != 1 ||
		config.SupervisorCommand.DrainLimits.MaxOutboxPerCycle != 1 {
		t.Fatalf("serial supervisor=%+v", config.SupervisorCommand)
	}
	runtimeConfig := codexRuntimeConfigV0(config, nil)
	if runtimeConfig.MaxBatchReady != 1 || runtimeConfig.MaxConcurrency != 1 {
		t.Fatalf("serial codex limits batch=%d concurrency=%d", runtimeConfig.MaxBatchReady, runtimeConfig.MaxConcurrency)
	}
	settings := config.EffectiveConfig.Settings
	for key, want := range map[string]string{
		"ORQUESTA_CODEX_EXECUTION_MODE":       "serial",
		"ORQUESTA_CODEX_MAX_BATCH_READY":      "1",
		"ORQUESTA_CODEX_MAX_CONCURRENCY":      "1",
		"ORQUESTA_CODEX_DIRECTOR_WAVE_AGENTS": "1",
	} {
		if got := effectiveSettingValueForTestV0(settings, key); got != want {
			t.Fatalf("%s=%q want %q settings=%+v", key, got, want, settings)
		}
	}
}

func TestServerStackIdleSelfImprovementFiltraSeccionesNoTxxV0(t *testing.T) {
	supervisor := serverStackSupervisorV0{}
	result, err := supervisor.FilterIdleSelfImprovementRequestsV0(context.Background(), orquestaserver.IdleSelfImprovementRequestFilterRequestV0{
		Requests: []orquestaserver.IdleSelfImprovementRequestV0{{
			RequestRef: "request-ref-autoprogramming-backlog-tareas-futuras-tras-estabilizar-la-automejora-0cdbd385",
		}, {
			RequestRef:    "request-ref-autoprogramming-backlog-t33-alias-reparado-a1b2c3d4",
			SuggestedArea: "t33-alias-reparado",
			ContextRefs:   []string{"backlog_section:tareas-futuras-tras-estabilizar-la-automejora"},
		}, {
			RequestRef: "request-ref-autoprogramming-backlog-t33-autoprogramming-backlog-ack-correlation-a1b2c3d4",
		}, {
			RequestRef: "request-ref-autoprogramming-backlog-t260-corregir-estados-falsos-running-en-agentes-externos-a1b2c3d4",
		}, {
			RequestRef:    "request-ref-autoprogramming-backlog-t260-contexto-narrativo-a1b2c3d4",
			SuggestedArea: "t260-contexto-narrativo",
			ContextRefs:   []string{"backlog_section:tareas-futuras-tras-estabilizar-la-automejora"},
		}, {
			RequestRef:    "request-ref-autoprogramming-backlog-scanner-narrativo",
			SuggestedArea: "backlog-scan",
			ContextRefs:   []string{"backlog_section:tareas-futuras-tras-estabilizar-la-automejora"},
		}, {
			RequestRef: "request-ref-autoprogramming-backlog-scanner-abc123",
		}},
	})
	if err != nil {
		t.Fatalf("FilterIdleSelfImprovementRequestsV0: %v", err)
	}
	if len(result.Requests) != 3 ||
		result.Requests[0].RequestRef != "request-ref-autoprogramming-backlog-t33-autoprogramming-backlog-ack-correlation-a1b2c3d4" ||
		result.Requests[1].RequestRef != "request-ref-autoprogramming-backlog-t260-corregir-estados-falsos-running-en-agentes-externos-a1b2c3d4" ||
		result.Requests[2].RequestRef != "request-ref-autoprogramming-backlog-scanner-abc123" ||
		!containsStringForTestV0(result.EvidenceRefs, "evidence-ref-idle-self-improvement-backlog-non-task-section-filtered") {
		t.Fatalf("result=%+v", result)
	}
}

func TestIdleSelfImprovementBacklogPlannerV0IgnoraSeccionesNarrativasV0(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	content := `# Backlog

## Tareas futuras tras estabilizar la automejora

Texto narrativo que no debe convertirse en run.

## Escaneo backlog 2026-07-02

Evidencia revisada sin tarea ejecutable propia.

## T260 goal-first-codex-loop-delgado

Objetivo: adelgazar el loop residente con Codex Goal.
`
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write backlog: %v", err)
	}

	result, err := (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}).PlanV0(
		orquestaserver.IdleSelfImprovementPlanRequestV0{
			MaxRequests: 3,
			Trigger:     "capacity_free",
			BaseRequest: orquestaserver.IdleSelfImprovementRequestV0{
				RequestRef:    "request-ref-base",
				CorrelationID: "corr-request-ref-base",
				ProjectRef:    "project-ref-orquesta",
				WriteSet:      []string{"cmd/orquesta-server"},
				RequiredTests: []string{"go test -count=1 ./cmd/orquesta-server"},
			},
		},
	)
	if err != nil {
		t.Fatalf("PlanV0: %v", err)
	}
	if len(result.Requests) != 1 ||
		result.Requests[0].SuggestedArea != "t260-goal-first-codex-loop-delgado" ||
		result.Requests[0].FailureKind != "backlog_autoprogramming" {
		t.Fatalf("requests=%+v", result.Requests)
	}
	for _, request := range result.Requests {
		if strings.Contains(request.RequestRef, "tareas-futuras") ||
			strings.Contains(request.RequestRef, "escaneo-backlog") ||
			request.FailureKind == "backlog_scan" {
			t.Fatalf("seccion narrativa convertida en request: %+v", request)
		}
	}
}

func TestIdleSelfImprovementBacklogPlannerV0BloqueaT260AmbiguoPorNumeroHumanoV0(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteBacklogTaskIDDocV0(t, projectDir, `# Backlog

## T260 goal-first-codex-loop-delgado

Objetivo: adelgazar goal-first.

## T260 corregir-estados-falsos-running-en-agentes-externos

Objetivo: corregir proyeccion externa.
`)
	result := planBacklogTaskIDReadModelForTestV0(t, projectDir, []string{"T260"})
	if len(result.Requests) != 0 || result.Message != "backlog_duplicate_task_id_ambiguous" {
		t.Fatalf("result=%+v", result)
	}
	collision := backlogCollisionForTestV0(result.Collisions, "backlog_duplicate_task_id_ambiguous")
	if collision.TaskID != "T260" || len(collision.InstanceRefs) != 2 {
		t.Fatalf("collision=%+v", collision)
	}
}

func TestServerStackIdleSelfImprovementPermiteAliasesFederadosV0(t *testing.T) {
	supervisor := serverStackSupervisorV0{}
	result, err := supervisor.FilterIdleSelfImprovementRequestsV0(context.Background(), orquestaserver.IdleSelfImprovementRequestFilterRequestV0{
		Requests: []orquestaserver.IdleSelfImprovementRequestV0{{
			RequestRef:  "request-ref-autoprogramming-backlog-apg-001-doc-loader",
			ContextRefs: []string{"backlog_local_alias:APG-001"},
		}, {
			RequestRef:  "request-ref-autoprogramming-backlog-srv-task-006-capacidad-libre",
			ContextRefs: []string{"backlog_local_alias:SRV-TASK-006"},
		}, {
			RequestRef:  "request-ref-autoprogramming-backlog-mcp-012-roadmap",
			ContextRefs: []string{"backlog_local_alias:MCP-012"},
		}, {
			RequestRef: "request-ref-autoprogramming-backlog-tareas-futuras-001",
		}},
	})
	if err != nil {
		t.Fatalf("FilterIdleSelfImprovementRequestsV0: %v", err)
	}
	if len(result.Requests) != 3 ||
		result.Requests[0].RequestRef != "request-ref-autoprogramming-backlog-apg-001-doc-loader" ||
		result.Requests[1].RequestRef != "request-ref-autoprogramming-backlog-srv-task-006-capacidad-libre" ||
		result.Requests[2].RequestRef != "request-ref-autoprogramming-backlog-mcp-012-roadmap" {
		t.Fatalf("result=%+v", result)
	}
}

func TestServerConfigFromEnvV0AislaEstadoYDejaRuntimeEscribiblePorDefecto(t *testing.T) {
	projectDir := t.TempDir()
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", projectDir)
	t.Setenv("ORQUESTA_SERVER_STATE_DIR", "")
	t.Setenv("ORQUESTA_CODEX_RUNTIME_WORKDIR", "")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if pathIsInsideForTestV0(t, config.StateDir, projectDir) {
		t.Fatalf("state dir dentro del proyecto: %s", config.StateDir)
	}
	if filepath.Dir(filepath.Dir(config.StateDir)) != filepath.Join(filepath.Dir(projectDir), ".orquesta-control") {
		t.Fatalf("state dir inesperado: %s", config.StateDir)
	}
	if !pathIsInsideForTestV0(t, config.RuntimeWorkDir, projectDir) {
		t.Fatalf("runtime dir fuera del proyecto: %s", config.RuntimeWorkDir)
	}
	if filepath.Base(config.RuntimeWorkDir) != ".orquesta-runtime" {
		t.Fatalf("runtime dir inesperado: %s", config.RuntimeWorkDir)
	}
}

func TestCodexDirectorWaveConfigFromEnvV0UsaSubagentesCanonicos(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_DIRECTOR_MAX_SUBAGENTS_PER_AGENT", "6")
	t.Setenv("ORQUESTA_CODEX_DIRECTOR_RECURSIVE_AGENT_BUDGET", "42")
	stderr := strings.Builder{}

	config, err := codexDirectorWaveConfigFromArgsV0([]string{
		"--dry-run",
		"--allow-recursive-delegation",
		"--max-delegation-depth", "1",
		"--wave-ref", "wave-env-subagents",
		"--project-dir", t.TempDir(),
		"--runtime-dir", filepath.Join(t.TempDir(), "runtime"),
		"--command", os.Args[0],
		"--objective", "probar fanout configurable",
		"--write-set", "cmd/orquesta-server",
		"--required-tests", "go test -count=1 ./cmd/orquesta-server",
		"--branch-ref", "branch-env-subagents",
		"--worktree-ref", "worktree-env-subagents",
	}, &stderr)
	if err != nil {
		t.Fatalf("codexDirectorWaveConfigFromArgsV0: %v stderr=%s", err, stderr.String())
	}
	if config.MaxSubagentsPerAgent != 6 || config.RecursiveAgentBudget != 42 {
		t.Fatalf("subagentes=%d budget=%d", config.MaxSubagentsPerAgent, config.RecursiveAgentBudget)
	}
}

func effectiveSettingValueForTestV0(settings []orquestaserver.ServerConfigSettingV0, key string) string {
	return effectiveSettingForTestV0(settings, key).Value
}

func effectiveSettingForTestV0(settings []orquestaserver.ServerConfigSettingV0, key string) orquestaserver.ServerConfigSettingV0 {
	for _, setting := range settings {
		if setting.Key == key {
			return setting
		}
	}
	return orquestaserver.ServerConfigSettingV0{}
}

func runtimeProviderForTestV0(
	providers []orquestaappcodexstack.RuntimeProviderConfigV0,
	kind orquestaappcodexstack.RuntimeProviderKindV0,
) orquestaappcodexstack.RuntimeProviderConfigV0 {
	for _, provider := range providers {
		if provider.Kind == kind {
			return provider
		}
	}
	return orquestaappcodexstack.RuntimeProviderConfigV0{}
}

func pathIsInsideForTestV0(t *testing.T, child string, parent string) bool {
	t.Helper()
	rel, err := filepath.Rel(filepath.Clean(parent), filepath.Clean(child))
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func TestCodexRuntimeConfigV0UsaUmbralesConservadoresPorDefecto(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_STALLED_TICKS", "")
	t.Setenv("ORQUESTA_CODEX_LOOP_TICKS", "")
	t.Setenv("ORQUESTA_CODEX_WAIT_INTERVAL_MS", "")
	t.Setenv("ORQUESTA_CODEX_NO_ACTIVITY_SECONDS", "")
	t.Setenv("ORQUESTA_CODEX_MAX_EXPECTED_SECONDS", "")
	t.Setenv("ORQUESTA_CODEX_REASONING_EFFORT", "")

	config := codexRuntimeConfigV0(orquestaserver.ConfigV0{
		ProjectWorkDir: t.TempDir(),
		RuntimeWorkDir: t.TempDir(),
	}, nil)

	if config.WaitInterval != 2*time.Second {
		t.Fatalf("wait_interval=%s want 2s", config.WaitInterval)
	}
	if config.ProgressPolicy.StalledAfterNoProgressTicks != defaultCodexStalledTicksV0 {
		t.Fatalf(
			"stalled_ticks=%d want %d",
			config.ProgressPolicy.StalledAfterNoProgressTicks,
			defaultCodexStalledTicksV0,
		)
	}
	if config.ProgressPolicy.LoopAfterRepeatedActions != defaultCodexLoopTicksV0 {
		t.Fatalf(
			"loop_ticks=%d want %d",
			config.ProgressPolicy.LoopAfterRepeatedActions,
			defaultCodexLoopTicksV0,
		)
	}
	if config.ProgressBudget.NoActivityLimit != 10*time.Minute {
		t.Fatalf("no_activity=%s want 10m", config.ProgressBudget.NoActivityLimit)
	}
	if config.ProgressBudget.MaxExpected != 20*time.Minute {
		t.Fatalf("max_expected=%s want 20m", config.ProgressBudget.MaxExpected)
	}
	if config.ReasoningEffort != "medium" {
		t.Fatalf("reasoning_effort=%q want medium", config.ReasoningEffort)
	}
}

func TestCodexRuntimeConfigV0PermiteSobrescribirUmbralesDeProgreso(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_STALLED_TICKS", "7")
	t.Setenv("ORQUESTA_CODEX_LOOP_TICKS", "11")
	t.Setenv("ORQUESTA_CODEX_WAIT_INTERVAL_MS", "500")
	t.Setenv("ORQUESTA_CODEX_NO_ACTIVITY_SECONDS", "17")
	t.Setenv("ORQUESTA_CODEX_MAX_EXPECTED_SECONDS", "19")

	config := codexRuntimeConfigV0(orquestaserver.ConfigV0{
		ProjectWorkDir: t.TempDir(),
		RuntimeWorkDir: t.TempDir(),
	}, nil)

	if config.WaitInterval != 500*time.Millisecond {
		t.Fatalf("wait_interval=%s want 500ms", config.WaitInterval)
	}
	if config.ProgressPolicy.StalledAfterNoProgressTicks != 7 {
		t.Fatalf("stalled_ticks=%d want 7", config.ProgressPolicy.StalledAfterNoProgressTicks)
	}
	if config.ProgressPolicy.LoopAfterRepeatedActions != 11 {
		t.Fatalf("loop_ticks=%d want 11", config.ProgressPolicy.LoopAfterRepeatedActions)
	}
	if config.ProgressBudget.NoActivityLimit != 17*time.Second {
		t.Fatalf("no_activity=%s want 17s", config.ProgressBudget.NoActivityLimit)
	}
	if config.ProgressBudget.MaxExpected != 19*time.Second {
		t.Fatalf("max_expected=%s want 19s", config.ProgressBudget.MaxExpected)
	}
}

func TestCodexRuntimeConfigV0PermiteSobrescribirReasoningEffort(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_REASONING_EFFORT", "high")

	config := codexRuntimeConfigV0(orquestaserver.ConfigV0{
		ProjectWorkDir: t.TempDir(),
		RuntimeWorkDir: t.TempDir(),
	}, nil)

	if config.ReasoningEffort != "high" {
		t.Fatalf("reasoning_effort=%q want high", config.ReasoningEffort)
	}
}

func TestCodexRuntimeConfigV0ConservaReasoningEffortMedium(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_REASONING_EFFORT", "medium")

	config := codexRuntimeConfigV0(orquestaserver.ConfigV0{
		ProjectWorkDir: t.TempDir(),
		RuntimeWorkDir: t.TempDir(),
	}, nil)

	if config.ReasoningEffort != "medium" {
		t.Fatalf("reasoning_effort=%q want medium", config.ReasoningEffort)
	}
}

func TestCodexRuntimeConfigV0InyectaToolbeltOperativoDelServidor(t *testing.T) {
	config := codexRuntimeConfigV0(orquestaserver.ConfigV0{
		Addr:           "127.0.0.1:18787",
		ProjectWorkDir: t.TempDir(),
		RuntimeWorkDir: t.TempDir(),
	}, nil)

	joined := strings.Join(config.PromptHints, "\n")
	for _, want := range []string{
		"http://127.0.0.1:18787",
		"/api/v0/autoprogramming/supervise",
		"orquesta.operator.operations.v0",
		"artifact_submission",
		"codigo reutilizable",
		"palabras genericas como clave",
		"$caveman full",
		"razonamiento medium",
		"checkpoint/handoff durable",
		"no metas HTTP, MCP, Codex, OPES",
		"centraliza configuracion/env",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("prompt hints no contienen %q: %s", want, joined)
		}
	}
}

func TestCodexRuntimeConfigV0LeeSkillInstructionsDesdeComposicion(t *testing.T) {
	t.Setenv(envCodexSkillInstructionsJSONV0, `[
		{"skill_ref":" skill-ref-catalogo-declarado-v0 ","text":" Catalogo declarado: usa workspace administrativo denso. "},
		{"skill_ref":"skill-ref-catalogo-declarado-v0","text":"duplicada"}
	]`)

	config := codexRuntimeConfigV0(orquestaserver.ConfigV0{
		ProjectWorkDir: t.TempDir(),
		RuntimeWorkDir: t.TempDir(),
	}, nil)

	if len(config.SkillInstructions) != 1 {
		t.Fatalf("skill_instructions=%+v", config.SkillInstructions)
	}
	if got := config.SkillInstructions[0]; got.SkillRef != "skill-ref-catalogo-declarado-v0" ||
		got.Text != "Catalogo declarado: usa workspace administrativo denso." {
		t.Fatalf("skill_instruction=%+v", got)
	}
}

func TestCodexStackCapacityConfigFromEnvV0ConservaReasoningCodexMedium(t *testing.T) {
	t.Setenv("ORQUESTA_CAPACITY_REASONING_EFFORT", "")
	t.Setenv("ORQUESTA_CODEX_REASONING_EFFORT", "medium")

	config := codexStackCapacityConfigFromEnvV0()

	if config.ReasoningEffort != orquestacoreworkflow.OrchestrationCapacityMediumV0 {
		t.Fatalf("reasoning_effort=%q want medium", config.ReasoningEffort)
	}
	if config.Tier != orquestacoreworkflow.OrchestrationCapacityMediumV0 {
		t.Fatalf("tier=%q want medium", config.Tier)
	}
	if config.OccurredAt == "" || config.RequestedBy != "orquesta-server" {
		t.Fatalf("capacity config incompleta: %+v", config)
	}
}

func TestCodexStackCapacityConfigFromEnvV0PermiteSobrescribirCapacidad(t *testing.T) {
	t.Setenv("ORQUESTA_CAPACITY_TIER", "high")
	t.Setenv("ORQUESTA_CAPACITY_REASONING_EFFORT", "low")
	t.Setenv("ORQUESTA_CODEX_REASONING_EFFORT", "medium")

	config := codexStackCapacityConfigFromEnvV0()

	if config.Tier != orquestacoreworkflow.OrchestrationCapacityHighV0 {
		t.Fatalf("tier=%q want high", config.Tier)
	}
	if config.ReasoningEffort != orquestacoreworkflow.OrchestrationCapacityLowV0 {
		t.Fatalf("reasoning_effort=%q want low", config.ReasoningEffort)
	}
}

func TestCodexRuntimeConfigV0PermiteSandboxAmplioConfigurable(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_SANDBOX", "workspace-write")
	t.Setenv("ORQUESTA_CODEX_APPROVAL_POLICY", "never")
	t.Setenv("ORQUESTA_CODEX_DIRECTOR_SANDBOX", "danger-full-access")
	t.Setenv("ORQUESTA_CODEX_DIRECTOR_APPROVAL_POLICY", "on-request")

	config := codexRuntimeConfigV0(orquestaserver.ConfigV0{
		ProjectWorkDir: t.TempDir(),
		RuntimeWorkDir: t.TempDir(),
	}, nil)

	if config.Sandbox != "workspace-write" ||
		config.ApprovalPolicy != "never" ||
		config.DirectorSandbox != "danger-full-access" ||
		config.DirectorApprovalPolicy != "on-request" {
		t.Fatalf("config=%+v", config)
	}
}

func TestCodexRuntimeConfigV0RespetaSandboxAmplioDeOperador(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_SANDBOX", "danger-full-access")

	config := codexRuntimeConfigV0(orquestaserver.ConfigV0{
		ProjectWorkDir: t.TempDir(),
		RuntimeWorkDir: t.TempDir(),
	}, nil)

	if config.Sandbox != "danger-full-access" {
		t.Fatalf("sandbox=%q want danger-full-access", config.Sandbox)
	}
}

func TestCodexRuntimeConfigV0UsaDangerFullAccessPorDefecto(t *testing.T) {
	config := codexRuntimeConfigV0(orquestaserver.ConfigV0{
		ProjectWorkDir: t.TempDir(),
		RuntimeWorkDir: t.TempDir(),
	}, nil)

	if config.Sandbox != "danger-full-access" {
		t.Fatalf("sandbox=%q want danger-full-access", config.Sandbox)
	}
}

func TestServerConfigFromEnvV0NoMutaStartupCleanupGlobalPorDefecto(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", t.TempDir())
	t.Setenv("ORQUESTA_STARTUP_CLEANUP_MODE", "")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if got := os.Getenv("ORQUESTA_STARTUP_CLEANUP_MODE"); got != "" {
		t.Fatalf("startup cleanup global mutado=%q", got)
	}
	counts := daemonStartEnvCountsV0(serverDaemonStartEnvPolicyV0(os.Environ(), config))
	if !strings.Contains(counts, "defaulted=") {
		t.Fatalf("daemon policy sin defaults: %s", counts)
	}
}

func TestServerConfigFromEnvV0LecturasRepetidasNoContaminanDefaultsV0(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", t.TempDir())
	t.Setenv("ORQUESTA_STARTUP_CLEANUP_MODE", "")
	t.Setenv("ORQUESTA_DETAIL_PROHIBITED_RAILS", "")

	first, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("first serverConfigFromEnvV0: %v", err)
	}
	second, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("second serverConfigFromEnvV0: %v", err)
	}
	if got := os.Getenv("ORQUESTA_STARTUP_CLEANUP_MODE"); got != "" {
		t.Fatalf("startup cleanup global mutado=%q", got)
	}
	if got := os.Getenv("ORQUESTA_DETAIL_PROHIBITED_RAILS"); got != "" {
		t.Fatalf("detail rails global mutado=%q", got)
	}
	if first.EffectiveConfig.Settings[0].Key != second.EffectiveConfig.Settings[0].Key {
		t.Fatalf("lecturas no estables")
	}
}

func TestDomainWorkExecutorFromEnvV0ConectaOPESOptIn(t *testing.T) {
	var received struct {
		CorrelationID  string `json:"correlation_id"`
		IdempotencyKey string `json:"idempotency_key"`
		RequestedBy    string `json:"requested_by"`
		JobType        string `json:"job_type"`
	}
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs" || r.Method != http.MethodPost {
			t.Fatalf("request=%s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":              "job-ref-opes-env-001",
			"status":          "accepted",
			"correlation_id":  "corr-opes-env-001",
			"idempotency_key": "idem-opes-env-001",
		})
	}))
	defer server.Close()
	t.Setenv("ORQUESTA_OPES_BASE_URL", server.URL)
	t.Setenv("ORQUESTA_OPES_TEMPORAL_CONFIRM", "1")
	t.Setenv("ORQUESTA_OPES_TIMEOUT_SECONDS", "1")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_ENABLED", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_DIR", "")

	executor, err := domainWorkExecutorFromEnvV0(orquestaserver.ConfigV0{
		StateDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("domainWorkExecutorFromEnvV0: %v", err)
	}
	if executor == nil {
		t.Fatalf("executor OPES no configurado")
	}
	result, err := executor.Execute(context.Background(), orquestamcp.MCPDomainWorkToolInputV0{
		RequestID:     "request-ref-opes-env-001",
		CorrelationID: "corr-opes-env-001",
		Action:        orquestamcp.MCPDomainWorkActionCreateJobV0,
		JobRequest: orquestadomainwork.DomainWorkJobRequestV0{
			SchemaVersion:  orquestadomainwork.DomainWorkJobRequestSchemaV0,
			RequestID:      "request-ref-opes-env-001",
			CorrelationID:  "corr-opes-env-001",
			IdempotencyKey: "idem-opes-env-001",
			RequestedBy:    "orquesta",
			DomainRef:      "opes",
			WorkKind:       "draft_content_block",
			Objective:      "crear bloque documental",
		},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != orquestamcp.MCPDomainWorkEstadoOKV0 ||
		result.Job == nil ||
		result.Job.JobRef != "job-ref-opes-env-001" {
		t.Fatalf("result=%+v", result)
	}
	if received.JobType != "draft_content_block" ||
		received.RequestedBy != "orquesta" {
		t.Fatalf("received=%+v", received)
	}
}

func TestDomainWorkExecutorFromEnvV0RechazaOPESProductivo(t *testing.T) {
	t.Setenv("ORQUESTA_OPES_BASE_URL", "https://opes.example.com")
	t.Setenv("ORQUESTA_OPES_BRIDGE_PRODUCTIVE_CONFIRM", "1")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_ENABLED", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_DIR", "")

	if _, err := domainWorkExecutorFromEnvV0(orquestaserver.ConfigV0{
		StateDir: t.TempDir(),
	}); err == nil || !strings.Contains(err.Error(), "productive_not_allowed") {
		t.Fatalf("err=%v", err)
	}
}

func TestDomainWorkExecutorFromEnvV0SinOPESQuedaApagado(t *testing.T) {
	t.Setenv("ORQUESTA_OPES_BASE_URL", "")
	t.Setenv("OPES_BASE_URL", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_ENABLED", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_DIR", "")

	executor, err := domainWorkExecutorFromEnvV0(orquestaserver.ConfigV0{
		StateDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("domainWorkExecutorFromEnvV0: %v", err)
	}
	if executor != nil {
		t.Fatalf("executor debe ser nil sin ORQUESTA_OPES_BASE_URL ni OPES_BASE_URL")
	}
}

func TestDomainWorkExecutorFromEnvV0ConectaOPESBaseURLFallback(t *testing.T) {
	var received struct {
		JobType string `json:"job_type"`
	}
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs" || r.Method != http.MethodPost {
			t.Fatalf("request=%s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":              "job-ref-opes-fallback-001",
			"status":          "accepted",
			"correlation_id":  "corr-opes-fallback-001",
			"idempotency_key": "idem-opes-fallback-001",
		})
	}))
	defer server.Close()
	t.Setenv("ORQUESTA_OPES_BASE_URL", "")
	t.Setenv("OPES_BASE_URL", server.URL)
	t.Setenv("ORQUESTA_OPES_TEMPORAL_CONFIRM", "1")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_ENABLED", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_DIR", "")

	executor, err := domainWorkExecutorFromEnvV0(orquestaserver.ConfigV0{
		StateDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("domainWorkExecutorFromEnvV0: %v", err)
	}
	if executor == nil {
		t.Fatalf("executor OPES fallback no configurado")
	}
	result, err := executor.Execute(context.Background(), orquestamcp.MCPDomainWorkToolInputV0{
		RequestID:     "request-ref-opes-fallback-001",
		CorrelationID: "corr-opes-fallback-001",
		Action:        orquestamcp.MCPDomainWorkActionCreateJobV0,
		JobRequest: orquestadomainwork.DomainWorkJobRequestV0{
			SchemaVersion:  orquestadomainwork.DomainWorkJobRequestSchemaV0,
			RequestID:      "request-ref-opes-fallback-001",
			CorrelationID:  "corr-opes-fallback-001",
			IdempotencyKey: "idem-opes-fallback-001",
			RequestedBy:    "orquesta",
			DomainRef:      "opes",
			WorkKind:       "plan_temario",
			Objective:      "crear plan de temario",
		},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != orquestamcp.MCPDomainWorkEstadoOKV0 ||
		result.Job == nil ||
		result.Job.JobRef != "job-ref-opes-fallback-001" ||
		received.JobType != "plan_temario" {
		t.Fatalf("result=%+v received=%+v", result, received)
	}
}

func TestDomainWorkExecutorFromEnvV0ConectaFileCreatorOptIn(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("ORQUESTA_OPES_BASE_URL", "")
	t.Setenv("OPES_BASE_URL", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_ENABLED", "1")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_DIR", "")

	executor, err := domainWorkExecutorFromEnvV0(orquestaserver.ConfigV0{
		StateDir: stateDir,
	})
	if err != nil {
		t.Fatalf("domainWorkExecutorFromEnvV0: %v", err)
	}
	if executor == nil {
		t.Fatalf("executor file no configurado")
	}
	input := orquestamcp.MCPDomainWorkToolInputV0{
		RequestID:     "request-ref-file-env-001",
		CorrelationID: "corr-file-env-001",
		Action:        orquestamcp.MCPDomainWorkActionCreateJobV0,
		JobRequest: orquestadomainwork.DomainWorkJobRequestV0{
			SchemaVersion:  orquestadomainwork.DomainWorkJobRequestSchemaV0,
			RequestID:      "request-ref-file-env-001",
			CorrelationID:  "corr-file-env-001",
			IdempotencyKey: "idem-file-env-001",
			RequestedBy:    "orquesta",
			DomainRef:      "dominio-demo",
			WorkKind:       "generate_content_package",
			Objective:      "crear paquete generico",
		},
	}
	first, err := executor.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute first: %v", err)
	}
	second, err := executor.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute second: %v", err)
	}
	if first.Estado != orquestamcp.MCPDomainWorkEstadoOKV0 ||
		first.Job == nil ||
		first.Job.JobRef == "" ||
		second.Job == nil ||
		second.Job.JobRef != first.Job.JobRef {
		t.Fatalf("first=%+v second=%+v", first, second)
	}
	if !pathExistsForTestV0(filepath.Join(stateDir, "domain-work-jobs", "domain_work_jobs_v0.json")) {
		t.Fatalf("snapshot domain-work file no creado")
	}

	submitResult, err := executor.Execute(context.Background(), orquestamcp.MCPDomainWorkToolInputV0{
		RequestID:     "request-ref-file-env-artifact-001",
		CorrelationID: "corr-file-env-001",
		Action:        orquestamcp.MCPDomainWorkActionSubmitArtifactV0,
		ArtifactSubmission: orquestadomainwork.DomainWorkArtifactSubmissionV0{
			SchemaVersion:  orquestadomainwork.DomainWorkArtifactSubmissionSchemaV0,
			RequestID:      "request-ref-file-env-artifact-001",
			CorrelationID:  "corr-file-env-001",
			IdempotencyKey: "idem-file-env-artifact-001",
			RequestedBy:    "orquesta",
			DomainRef:      "dominio-demo",
			JobRef:         first.Job.JobRef,
			ArtifactRef:    "artifact-ref-file-env-001",
			ArtifactType:   "content_package",
			PayloadFields: []orquestadomainwork.DomainWorkFieldV0{{
				Name:  "body",
				Value: "contenido",
			}},
			CompleteJob: true,
		},
	})
	if err != nil {
		t.Fatalf("Execute submit: %v", err)
	}
	if submitResult.Estado != orquestamcp.MCPDomainWorkEstadoOKV0 ||
		submitResult.Receipt == nil ||
		submitResult.Receipt.ReceiptRef == "" {
		t.Fatalf("submitResult=%+v", submitResult)
	}
	if !pathExistsForTestV0(filepath.Join(stateDir, "domain-work-jobs", "domain_work_artifacts_v0.json")) {
		t.Fatalf("snapshot artifact domain-work file no creado")
	}
}

func TestDomainWorkExecutorFromProjectConfigV0ConectaFileCreatorOptIn(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	fileDir := filepath.Join(stateDir, "domain-work-from-config")
	writeServerProjectConfigForDomainWorkTestV0(t, projectDir, serverProjectConfigFileV0{
		DomainWork: serverProjectConfigDomainWorkV0{
			FileEnabled: boolPointerForDomainWorkTestV0(true),
			FileDir:     stringPointerForDomainWorkTestV0(fileDir),
		},
	})
	clearDomainWorkEnvForTestV0(t)

	executor, err := domainWorkExecutorFromEnvV0(orquestaserver.ConfigV0{
		ProjectWorkDir: projectDir,
		StateDir:       stateDir,
	})
	if err != nil {
		t.Fatalf("domainWorkExecutorFromEnvV0: %v", err)
	}
	if executor == nil {
		t.Fatalf("executor file no configurado")
	}
	result, err := executor.Execute(context.Background(), orquestamcp.MCPDomainWorkToolInputV0{
		RequestID:     "request-ref-file-config-001",
		CorrelationID: "corr-file-config-001",
		Action:        orquestamcp.MCPDomainWorkActionCreateJobV0,
		JobRequest: orquestadomainwork.DomainWorkJobRequestV0{
			SchemaVersion:  orquestadomainwork.DomainWorkJobRequestSchemaV0,
			RequestID:      "request-ref-file-config-001",
			CorrelationID:  "corr-file-config-001",
			IdempotencyKey: "idem-file-config-001",
			RequestedBy:    "orquesta",
			DomainRef:      "dominio-demo",
			WorkKind:       "generate_content_package",
			Objective:      "crear paquete desde config file",
		},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != orquestamcp.MCPDomainWorkEstadoOKV0 ||
		result.Job == nil ||
		result.Job.JobRef == "" {
		t.Fatalf("result=%+v", result)
	}
	if !pathExistsForTestV0(filepath.Join(fileDir, "domain_work_jobs_v0.json")) {
		t.Fatalf("snapshot domain-work file no creado en file_dir configurado")
	}
}

func TestDomainWorkExecutorFromEnvV0ConectaHTTPNeutralOptIn(t *testing.T) {
	var gotJobRequest orquestadomainwork.DomainWorkJobRequestV0
	var gotSubmission orquestadomainwork.DomainWorkArtifactSubmissionV0
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/jobs":
			if err := json.NewDecoder(r.Body).Decode(&gotJobRequest); err != nil {
				t.Fatalf("decode job: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"job": map[string]any{
					"schema_version":  orquestadomainwork.DomainWorkJobSchemaV0,
					"status":          orquestadomainwork.DomainWorkStatusAcceptedV0,
					"job_ref":         "job-ref-http-neutral-001",
					"domain_ref":      gotJobRequest.DomainRef,
					"work_kind":       gotJobRequest.WorkKind,
					"correlation_id":  gotJobRequest.CorrelationID,
					"idempotency_key": gotJobRequest.IdempotencyKey,
				},
			})
		case "/artifacts":
			if err := json.NewDecoder(r.Body).Decode(&gotSubmission); err != nil {
				t.Fatalf("decode submission: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"receipt": map[string]any{
					"schema_version":  orquestadomainwork.DomainWorkArtifactReceiptSchemaV0,
					"status":          orquestadomainwork.DomainWorkStatusAcceptedV0,
					"job_ref":         gotSubmission.JobRef,
					"artifact_ref":    gotSubmission.ArtifactRef,
					"receipt_ref":     "receipt-ref-http-neutral-001",
					"correlation_id":  gotSubmission.CorrelationID,
					"idempotency_key": gotSubmission.IdempotencyKey,
				},
			})
		default:
			t.Fatalf("path=%s", r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("ORQUESTA_OPES_BASE_URL", "")
	t.Setenv("OPES_BASE_URL", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_ENABLED", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_DIR", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_HTTP_BASE_URL", server.URL)
	t.Setenv("ORQUESTA_DOMAIN_WORK_HTTP_EGRESS_MODE", "smoke_local")
	t.Setenv("ORQUESTA_DOMAIN_WORK_HTTP_CREATE_PATH", "/jobs")
	t.Setenv("ORQUESTA_DOMAIN_WORK_HTTP_SUBMIT_PATH", "/artifacts")

	executor, err := domainWorkExecutorFromEnvV0(orquestaserver.ConfigV0{
		StateDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("domainWorkExecutorFromEnvV0: %v", err)
	}
	if executor == nil {
		t.Fatalf("executor HTTP neutral no configurado")
	}
	createResult, err := executor.Execute(context.Background(), orquestamcp.MCPDomainWorkToolInputV0{
		Action: orquestamcp.MCPDomainWorkActionCreateJobV0,
		JobRequest: orquestadomainwork.DomainWorkJobRequestV0{
			RequestID:      "request-ref-http-neutral-001",
			CorrelationID:  "corr-http-neutral-001",
			IdempotencyKey: "idem-http-neutral-001",
			RequestedBy:    "test",
			DomainRef:      "domain-ref-http-neutral-001",
			WorkKind:       "compose_external_summary",
			Objective:      "crear job por HTTP neutral",
		},
	})
	if err != nil {
		t.Fatalf("Execute create: %v", err)
	}
	if createResult.Estado != orquestamcp.MCPDomainWorkEstadoOKV0 ||
		createResult.Job == nil ||
		createResult.Job.JobRef != "job-ref-http-neutral-001" {
		t.Fatalf("createResult=%+v", createResult)
	}
	submitResult, err := executor.Execute(context.Background(), orquestamcp.MCPDomainWorkToolInputV0{
		Action: orquestamcp.MCPDomainWorkActionSubmitArtifactV0,
		ArtifactSubmission: orquestadomainwork.DomainWorkArtifactSubmissionV0{
			RequestID:      "request-ref-http-neutral-submit-001",
			CorrelationID:  "corr-http-neutral-001",
			IdempotencyKey: "idem-http-neutral-submit-001",
			RequestedBy:    "test",
			DomainRef:      "domain-ref-http-neutral-001",
			JobRef:         createResult.Job.JobRef,
			ArtifactRef:    "artifact-ref-http-neutral-001",
			ArtifactType:   "work_delivery",
		},
	})
	if err != nil {
		t.Fatalf("Execute submit: %v", err)
	}
	if submitResult.Estado != orquestamcp.MCPDomainWorkEstadoOKV0 ||
		submitResult.Receipt == nil ||
		submitResult.Receipt.ReceiptRef != "receipt-ref-http-neutral-001" ||
		gotSubmission.JobRef != "job-ref-http-neutral-001" {
		t.Fatalf("submitResult=%+v got=%+v", submitResult, gotSubmission)
	}
}

func TestDomainWorkExecutorFromProjectConfigV0ConectaHTTPNeutralOptIn(t *testing.T) {
	var gotJobRequest orquestadomainwork.DomainWorkJobRequestV0
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/jobs-config" || r.Method != http.MethodPost {
			t.Fatalf("request=%s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotJobRequest); err != nil {
			t.Fatalf("decode job: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"job": map[string]any{
				"schema_version":  orquestadomainwork.DomainWorkJobSchemaV0,
				"status":          orquestadomainwork.DomainWorkStatusAcceptedV0,
				"job_ref":         "job-ref-http-config-001",
				"domain_ref":      gotJobRequest.DomainRef,
				"work_kind":       gotJobRequest.WorkKind,
				"correlation_id":  gotJobRequest.CorrelationID,
				"idempotency_key": gotJobRequest.IdempotencyKey,
			},
		})
	}))
	defer server.Close()
	projectDir := t.TempDir()
	writeServerProjectConfigForDomainWorkTestV0(t, projectDir, serverProjectConfigFileV0{
		DomainWork: serverProjectConfigDomainWorkV0{
			HTTPBaseURL:        stringPointerForDomainWorkTestV0(server.URL),
			HTTPCreatePath:     stringPointerForDomainWorkTestV0("/jobs-config"),
			HTTPTimeoutSeconds: intPointerForDomainWorkTestV0(3),
			HTTPEgressMode:     stringPointerForDomainWorkTestV0("smoke_local"),
		},
	})
	clearDomainWorkEnvForTestV0(t)

	executor, err := domainWorkExecutorFromEnvV0(orquestaserver.ConfigV0{
		ProjectWorkDir: projectDir,
		StateDir:       t.TempDir(),
	})
	if err != nil {
		t.Fatalf("domainWorkExecutorFromEnvV0: %v", err)
	}
	if executor == nil {
		t.Fatalf("executor HTTP neutral no configurado")
	}
	createResult, err := executor.Execute(context.Background(), orquestamcp.MCPDomainWorkToolInputV0{
		Action: orquestamcp.MCPDomainWorkActionCreateJobV0,
		JobRequest: orquestadomainwork.DomainWorkJobRequestV0{
			RequestID:      "request-ref-http-config-001",
			CorrelationID:  "corr-http-config-001",
			IdempotencyKey: "idem-http-config-001",
			RequestedBy:    "test",
			DomainRef:      "domain-ref-http-config-001",
			WorkKind:       "compose_external_summary",
			Objective:      "crear job por HTTP neutral desde config file",
		},
	})
	if err != nil {
		t.Fatalf("Execute create: %v", err)
	}
	if createResult.Estado != orquestamcp.MCPDomainWorkEstadoOKV0 ||
		createResult.Job == nil ||
		createResult.Job.JobRef != "job-ref-http-config-001" ||
		gotJobRequest.WorkKind != "compose_external_summary" {
		t.Fatalf("createResult=%+v got=%+v", createResult, gotJobRequest)
	}
}

func TestServerEffectiveConfigV0PublicaDomainWorkDesdeProjectConfigV0(t *testing.T) {
	projectDir := t.TempDir()
	fileDir := filepath.Join(projectDir, "domain-work-jobs")
	ledgerPath := filepath.Join(projectDir, "domain-work-ledger.json")
	allowedHosts := []string{"127.0.0.1", "localhost"}
	writeServerProjectConfigForDomainWorkTestV0(t, projectDir, serverProjectConfigFileV0{
		DomainWork: serverProjectConfigDomainWorkV0{
			HTTPBaseURL:        stringPointerForDomainWorkTestV0("http://127.0.0.1:18080"),
			HTTPDomainRef:      stringPointerForDomainWorkTestV0("demo"),
			FileDir:            stringPointerForDomainWorkTestV0(fileDir),
			FileEnabled:        boolPointerForDomainWorkTestV0(true),
			HTTPCreatePath:     stringPointerForDomainWorkTestV0("/jobs"),
			HTTPSubmitPath:     stringPointerForDomainWorkTestV0("/artifacts"),
			HTTPTimeoutSeconds: intPointerForDomainWorkTestV0(9),
			HTTPEgressMode:     stringPointerForDomainWorkTestV0("allowlist"),
			HTTPAllowedHosts:   stringSlicePointerForDomainWorkTestV0(allowedHosts),
			DeliveryLedgerPath: stringPointerForDomainWorkTestV0(ledgerPath),
		},
	})
	clearDomainWorkEnvForTestV0(t)
	t.Setenv(envCodexProjectWorkDirV0, projectDir)

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	settings := config.EffectiveConfig.Settings
	for key, want := range map[string]string{
		envDomainWorkHTTPBaseURLV0:        domainWorkHTTPBaseURLConfiguredRefV0,
		envDomainWorkHTTPDomainRefV0:      "demo",
		envDomainWorkFileDirV0:            domainWorkFileDirConfiguredRefV0,
		envDomainWorkFileEnabledV0:        "true",
		envDomainWorkHTTPCreatePathV0:     "/jobs",
		envDomainWorkHTTPSubmitPathV0:     "/artifacts",
		envDomainWorkHTTPTimeoutSecondsV0: "9",
		envDomainWorkHTTPEgressModeV0:     "allowlist",
		envDomainWorkHTTPAllowedHostsV0:   "127.0.0.1,localhost",
		envDomainDeliveryLedgerPathV0:     domainWorkDeliveryLedgerPathConfiguredRefV0,
	} {
		setting := effectiveSettingForTestV0(settings, key)
		if setting.Value != want ||
			setting.Source != configSettingSourceConfigFileV0 {
			t.Fatalf("%s setting=%+v want value=%q source=%s", key, setting, want, configSettingSourceConfigFileV0)
		}
	}
	for _, key := range []string{envDomainWorkHTTPBaseURLV0, envDomainWorkFileDirV0, envDomainDeliveryLedgerPathV0} {
		setting := effectiveSettingForTestV0(settings, key)
		if !setting.Sensitive {
			t.Fatalf("%s debe quedar marcado como sensible: %+v", key, setting)
		}
	}
}

func TestDomainWorkDeliveryEnabledFromEnvV0IncluyeBackendFile(t *testing.T) {
	t.Setenv("ORQUESTA_OPES_BASE_URL", "")
	t.Setenv("OPES_BASE_URL", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_HTTP_BASE_URL", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_ENABLED", "1")

	if !domainWorkDeliveryEnabledFromEnvV0() {
		t.Fatalf("domain work delivery debe activarse con backend file")
	}
}

func TestDomainWorkExecutorFromEnvV0RechazaBackendAmbiguo(t *testing.T) {
	t.Setenv("ORQUESTA_OPES_BASE_URL", "http://127.0.0.1:1")
	t.Setenv("OPES_BASE_URL", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_ENABLED", "1")

	executor, err := domainWorkExecutorFromEnvV0(orquestaserver.ConfigV0{
		StateDir: t.TempDir(),
	})
	if err == nil || err.Error() != "domain_work_backend_ambiguous" || executor != nil {
		t.Fatalf("executor=%v err=%v", executor, err)
	}
}

func TestDomainWorkExecutorFromEnvV0RechazaFallbackAmbiguo(t *testing.T) {
	t.Setenv("ORQUESTA_OPES_BASE_URL", "")
	t.Setenv("OPES_BASE_URL", "http://127.0.0.1:1")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_ENABLED", "1")

	executor, err := domainWorkExecutorFromEnvV0(orquestaserver.ConfigV0{
		StateDir: t.TempDir(),
	})
	if err == nil || err.Error() != "domain_work_backend_ambiguous" || executor != nil {
		t.Fatalf("executor=%v err=%v", executor, err)
	}
}

func TestDomainWorkExecutorFromEnvV0RechazaHTTPConOtroBackend(t *testing.T) {
	t.Setenv("ORQUESTA_OPES_BASE_URL", "")
	t.Setenv("OPES_BASE_URL", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_ENABLED", "1")
	t.Setenv("ORQUESTA_DOMAIN_WORK_HTTP_BASE_URL", "http://127.0.0.1:1")

	executor, err := domainWorkExecutorFromEnvV0(orquestaserver.ConfigV0{
		StateDir: t.TempDir(),
	})
	if err == nil || err.Error() != "domain_work_backend_ambiguous" || executor != nil {
		t.Fatalf("executor=%v err=%v", executor, err)
	}
}

func writeServerProjectConfigForDomainWorkTestV0(
	t *testing.T,
	root string,
	config serverProjectConfigFileV0,
) {
	t.Helper()
	config.SchemaVersion = serverProjectConfigSchemaVersionV0
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("marshal project config: %v", err)
	}
	if err := os.WriteFile(serverProjectConfigFilePathV0(root), data, 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}
}

func clearDomainWorkEnvForTestV0(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		envOPESBaseURLV0,
		envOPESBaseURLLegacyV0,
		envDomainWorkHTTPBaseURLV0,
		envDomainWorkHTTPDomainRefV0,
		envDomainWorkFileDirV0,
		envDomainWorkFileEnabledV0,
		envDomainWorkHTTPCreatePathV0,
		envDomainWorkHTTPSubmitPathV0,
		envDomainWorkHTTPTimeoutSecondsV0,
		envDomainWorkHTTPEgressModeV0,
		envDomainWorkHTTPAllowedHostsV0,
		envDomainDeliveryLedgerPathV0,
	} {
		t.Setenv(key, "")
	}
}

func stringPointerForDomainWorkTestV0(value string) *string {
	return &value
}

func boolPointerForDomainWorkTestV0(value bool) *bool {
	return &value
}

func intPointerForDomainWorkTestV0(value int) *int {
	return &value
}

func stringSlicePointerForDomainWorkTestV0(value []string) *[]string {
	return &value
}

func pathExistsForTestV0(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
