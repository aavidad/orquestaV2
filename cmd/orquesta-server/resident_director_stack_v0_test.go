package main

import (
	"testing"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestNewServerResidentDirectorV0EsOptInV0(t *testing.T) {
	stack := &orquestaappcodexstack.StackV0{}
	if got := newServerResidentDirectorV0(stack, orquestaserver.ConfigV0{}); got != nil {
		t.Fatalf("director residente inyectado sin opt-in: %T", got)
	}
	if got := newServerResidentDirectorV0(nil, orquestaserver.ConfigV0{
		ResidentDirectorEnabled: true,
	}); got != nil {
		t.Fatalf("director residente inyectado sin stack: %T", got)
	}
	if got := newServerResidentDirectorV0(stack, orquestaserver.ConfigV0{
		ResidentDirectorEnabled: true,
	}); got == nil {
		t.Fatalf("director residente no inyectado con opt-in")
	}
}

func TestServerConfigFromEnvV0ConfiguraResidentDirectorV0(t *testing.T) {
	t.Setenv(envServerResidentDirectorEnabledV0, "true")
	t.Setenv(envServerResidentDirectorMaxActionsV0, "9")
	t.Setenv(envServerStateDirV0, t.TempDir())
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if !config.ResidentDirectorEnabled || config.ResidentDirectorMaxActions != 9 {
		t.Fatalf("resident director config=%+v", config)
	}
	if !containsEffectiveConfigSettingForTestV0(config.EffectiveConfig, envServerResidentDirectorEnabledV0, "true") ||
		!containsEffectiveConfigSettingForTestV0(config.EffectiveConfig, envServerResidentDirectorMaxActionsV0, "9") {
		t.Fatalf("effective_config no publica resident director: %+v", config.EffectiveConfig)
	}
}
