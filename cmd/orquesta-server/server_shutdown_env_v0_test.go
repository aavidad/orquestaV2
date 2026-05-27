package main

import (
	"strconv"
	"testing"
	"time"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestServerConfigFromEnvV0ShutdownGraceConfigurableV0(t *testing.T) {
	t.Setenv(envServerShutdownGraceMSV0, "1234")
	t.Setenv(envServerStateDirV0, t.TempDir())
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if config.ShutdownGracePeriod != 1234*time.Millisecond {
		t.Fatalf("ShutdownGracePeriod=%s", config.ShutdownGracePeriod)
	}
	if !containsEffectiveConfigSettingForTestV0(
		config.EffectiveConfig,
		envServerShutdownGraceMSV0,
		strconv.Itoa(int(1234)),
	) {
		t.Fatalf("effective_config no incluye %s: %+v", envServerShutdownGraceMSV0, config.EffectiveConfig)
	}
}

func TestServerConfigFromEnvV0ShutdownGraceDefaultV0(t *testing.T) {
	t.Setenv(envServerStateDirV0, t.TempDir())
	t.Setenv(envCodexProjectWorkDirV0, t.TempDir())
	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if config.ShutdownGracePeriod != orquestaserver.DefaultShutdownGracePeriodV0 {
		t.Fatalf("ShutdownGracePeriod=%s", config.ShutdownGracePeriod)
	}
}

func containsEffectiveConfigSettingForTestV0(
	config orquestaserver.ServerEffectiveConfigV0,
	key string,
	value string,
) bool {
	for _, setting := range config.Settings {
		if setting.Key == key && setting.Value == value {
			return true
		}
	}
	return false
}
