package orquestaserver

import "testing"

func TestPublicServerEffectiveConfigV0NoRedactaLimitesNumericosMaxCommandsV0(t *testing.T) {
	public, hidden := PublicServerEffectiveConfigV0(ServerEffectiveConfigV0{
		Settings: []ServerConfigSettingV0{
			{Key: "ORQUESTA_SERVER_DRAIN_MAX_COMMANDS", Value: "25", Scope: "server_supervisor"},
			{Key: "ORQUESTA_CODEX_COMMAND", Value: "/usr/bin/codex", Scope: "codex_runtime"},
		},
	})
	limit := publicConfigSettingForTestV0(t, public, "ORQUESTA_SERVER_DRAIN_MAX_COMMANDS")
	if limit.Value != "25" || limit.Sensitive {
		t.Fatalf("limite numerico redactado: %+v hidden=%+v", limit, hidden)
	}
	command := publicConfigSettingForTestV0(t, public, "ORQUESTA_CODEX_COMMAND")
	if command.Value != ServerStatusConfigHiddenValueV0 || !command.Sensitive {
		t.Fatalf("command real no redactado: %+v hidden=%+v", command, hidden)
	}
	if len(hidden) != 1 || hidden[0] != "effective_config.ORQUESTA_CODEX_COMMAND" {
		t.Fatalf("hidden=%+v", hidden)
	}
}

func publicConfigSettingForTestV0(
	t *testing.T,
	config ServerEffectiveConfigV0,
	key string,
) ServerConfigSettingV0 {
	t.Helper()
	for _, setting := range config.Settings {
		if setting.Key == key {
			return setting
		}
	}
	t.Fatalf("setting %s no encontrado en %+v", key, config.Settings)
	return ServerConfigSettingV0{}
}
