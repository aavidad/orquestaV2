package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestServerConfigFromEnvV0ConfiguraAutomejoraIdleV0(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", t.TempDir())
	t.Setenv("ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS", "")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if config.IdleSelfImprovementAfter != 60*time.Second ||
		config.IdleSelfImprovementDisabled ||
		config.IdleSelfImprovementMaxRequests != 10 ||
		config.IdleSelfImprovementTargetQueue != 10 ||
		len(config.IdleSelfImprovementWriteSet) == 0 ||
		config.IdleSelfImprovementRequiredTests[0] != "go test -count=1 ./..." {
		t.Fatalf("idle config=%+v", config)
	}

	t.Setenv("ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS", "0")
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

func TestServerConfigFromEnvV0DesactivaAutomejoraIdlePorEnvCanonicaV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	t.Setenv(envServerIdleSelfImprovementDisabledV0, "true")
	t.Setenv(envServerIdleSelfImprovementAfterV0, "")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if !config.IdleSelfImprovementDisabled ||
		!config.IdleSelfImprovementIdleDisabled ||
		config.IdleSelfImprovementAfter != 0 {
		t.Fatalf("automejora idle debe quedar desactivada por env canonica: %+v", config)
	}
	setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envServerIdleSelfImprovementDisabledV0)
	if setting.Value != "true" || setting.Source != "explicit" || !setting.Canonical {
		t.Fatalf("setting disabled=%+v", setting)
	}
}

func TestServerConfigFromEnvV0AceptaAliasLegacyDeAutomejoraIdleConDiagnosticoV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	t.Setenv(envServerIdleSelfImprovementAfterV0, "")
	t.Setenv(envServerIdleSelfImprovementAfterLegacyV0, "0")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if config.IdleSelfImprovementDisabled ||
		!config.IdleSelfImprovementIdleDisabled ||
		config.IdleSelfImprovementAfter != 0 {
		t.Fatalf("alias legacy no desactiva trigger idle: %+v", config)
	}
	setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envServerIdleSelfImprovementAfterV0)
	if setting.Value != "0" || setting.Source != "legacy_alias" || !setting.Canonical {
		t.Fatalf("setting after=%+v", setting)
	}
	if !effectiveConfigHasDiagnosticForTestV0(config.EffectiveConfig, "legacy_env_alias", envServerIdleSelfImprovementAfterLegacyV0, envServerIdleSelfImprovementAfterV0) {
		t.Fatalf("diagnostico legacy ausente: %+v", config.EffectiveConfig.Diagnostics)
	}
}

func TestServerConfigFromEnvV0CanonicaGanaAAliasLegacyDeAutomejoraIdleV0(t *testing.T) {
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	t.Setenv(envServerIdleSelfImprovementAfterV0, "30")
	t.Setenv(envServerIdleSelfImprovementAfterLegacyV0, "0")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if config.IdleSelfImprovementDisabled || config.IdleSelfImprovementAfter != 30*time.Second {
		t.Fatalf("la canonica debe ganar al alias legacy: %+v", config)
	}
	setting := effectiveSettingForTestV0(config.EffectiveConfig.Settings, envServerIdleSelfImprovementAfterV0)
	if setting.Value != "30" || setting.Source != "explicit" {
		t.Fatalf("setting after=%+v", setting)
	}
	if !effectiveConfigHasDiagnosticForTestV0(config.EffectiveConfig, "legacy_env_ignored", envServerIdleSelfImprovementAfterLegacyV0, envServerIdleSelfImprovementAfterV0) {
		t.Fatalf("diagnostico legacy ignored ausente: %+v", config.EffectiveConfig.Diagnostics)
	}
}

func TestServerConfigFromEnvV0DesactivaAutomejoraIdleEnOPESSinWorkdirSeparadoV0(t *testing.T) {
	root := t.TempDir()
	opesDir := filepath.Join(root, "OPES")
	t.Setenv(envCodexProjectWorkDirV0, opesDir)
	t.Setenv(envOPESProjectWorkDirV0, opesDir)
	t.Setenv(envServerIdleSelfImprovementAfterV0, "")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if !config.IdleSelfImprovementDisabled ||
		!config.IdleSelfImprovementIdleDisabled ||
		config.IdleSelfImprovementAfter != 0 {
		t.Fatalf("automejora idle debe quedar desactivada en OPES sin workdir separado: %+v", config)
	}
	if config.IdleSelfImprovementProjectWorkDir != opesDir {
		t.Fatalf("idle_self_improvement_project_work_dir=%q want %q", config.IdleSelfImprovementProjectWorkDir, opesDir)
	}
}

func TestServerConfigFromEnvV0DetectaOPESPorProjectWorkdirCodexV0(t *testing.T) {
	root := t.TempDir()
	opesOutputDir := filepath.Join(root, "OPES", "opes-salidas", "curso-demo")
	t.Setenv(envCodexProjectWorkDirV0, opesOutputDir)
	t.Setenv(envServerIdleSelfImprovementAfterV0, "")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if !config.IdleSelfImprovementDisabled ||
		!config.IdleSelfImprovementIdleDisabled ||
		config.IdleSelfImprovementAfter != 0 {
		t.Fatalf("automejora idle debe quedar desactivada si CODEX_PROJECT_WORKDIR apunta a salida OPES: %+v", config)
	}
}

func TestServerConfigFromEnvV0DesactivaAutomejoraIdleSiWorkdirExplicitoEsOPESV0(t *testing.T) {
	root := t.TempDir()
	opesDir := filepath.Join(root, "OPES")
	orquestaDir := filepath.Join(root, "orquesta")
	t.Setenv(envCodexProjectWorkDirV0, orquestaDir)
	t.Setenv(envOPESProjectWorkDirV0, opesDir)
	t.Setenv(envServerIdleSelfImprovementProjectWorkDirV0, opesDir)
	t.Setenv(envServerIdleSelfImprovementAfterV0, "")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if !config.IdleSelfImprovementDisabled ||
		!config.IdleSelfImprovementIdleDisabled ||
		config.IdleSelfImprovementAfter != 0 {
		t.Fatalf("automejora idle debe quedar desactivada si apunta al workdir OPES: %+v", config)
	}
}

func effectiveConfigHasDiagnosticForTestV0(config orquestaserver.ServerEffectiveConfigV0, code string, parts ...string) bool {
	for _, diagnostic := range config.Diagnostics {
		if diagnostic.Code != code {
			continue
		}
		message := diagnostic.Message
		ok := true
		for _, part := range parts {
			if !strings.Contains(message, part) {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}
