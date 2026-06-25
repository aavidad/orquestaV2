package orquestaserver

import "strings"

const ServerEffectiveConfigSchemaVersionV0 = "orquesta_server_effective_config.v0"

func NormalizeServerEffectiveConfigV0(config ServerEffectiveConfigV0) ServerEffectiveConfigV0 {
	config.SchemaVersion = strings.TrimSpace(config.SchemaVersion)
	if config.SchemaVersion == "" && len(config.Settings) > 0 {
		config.SchemaVersion = ServerEffectiveConfigSchemaVersionV0
	}
	config.RestartNote = strings.TrimSpace(config.RestartNote)
	settings := make([]ServerConfigSettingV0, 0, len(config.Settings))
	seen := map[string]bool{}
	for _, setting := range config.Settings {
		setting = NormalizeServerConfigSettingV0(setting)
		if setting.Key == "" || seen[setting.Key] {
			continue
		}
		seen[setting.Key] = true
		settings = append(settings, setting)
	}
	config.Settings = settings
	config.Diagnostics = normalizeServerEffectiveConfigDiagnosticsV0(config.Diagnostics)
	if config.RestartNote == "" && len(config.Settings) > 0 {
		config.RestartNote = "Los cambios de variables de entorno no se aplican al proceso actual; quedan pendientes hasta reinicio."
	}
	return config
}

func NormalizeServerConfigSettingV0(setting ServerConfigSettingV0) ServerConfigSettingV0 {
	setting.Key = strings.TrimSpace(setting.Key)
	setting.Value = strings.TrimSpace(setting.Value)
	setting.Source = strings.TrimSpace(setting.Source)
	setting.Scope = strings.TrimSpace(setting.Scope)
	setting.Label = strings.TrimSpace(setting.Label)
	setting.Description = strings.TrimSpace(setting.Description)
	setting.RestartBehavior = strings.TrimSpace(setting.RestartBehavior)
	if setting.RestartBehavior == "" && setting.Editable {
		setting.RestartBehavior = "restart_required"
	}
	if setting.Canonical || setting.Key != "" {
		setting.Canonical = true
	}
	return setting
}

func normalizeServerEffectiveConfigDiagnosticsV0(diagnostics []ServerDiagnosticV0) []ServerDiagnosticV0 {
	out := make([]ServerDiagnosticV0, 0, len(diagnostics))
	seen := map[string]bool{}
	for _, diagnostic := range diagnostics {
		diagnostic.Code = strings.TrimSpace(diagnostic.Code)
		diagnostic.Scope = strings.TrimSpace(diagnostic.Scope)
		diagnostic.Message = strings.TrimSpace(diagnostic.Message)
		diagnostic.EvidenceRefs = compactConfigStringsV0(diagnostic.EvidenceRefs)
		if diagnostic.Code == "" && diagnostic.Message == "" {
			continue
		}
		key := diagnostic.Code + "|" + diagnostic.Scope + "|" + diagnostic.Message
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, diagnostic)
	}
	return out
}
