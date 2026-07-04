package orquestaappcodexstack

import orquestamcp "orquesta/modulos/orquesta-mcp"

func configProjectionSettingsV0(
	settings []orquestamcp.MCPConfigProjectionSettingV0,
) []orquestamcp.MCPConfigProjectionSettingV0 {
	if len(settings) == 0 {
		return nil
	}
	out := make([]orquestamcp.MCPConfigProjectionSettingV0, 0, len(settings))
	for _, setting := range settings {
		if setting.Key == "" {
			continue
		}
		out = append(out, setting)
	}
	return out
}

func configProjectionRequiredSettingIssuesV0(
	required []orquestamcp.MCPRequiredSettingV0,
	settings []orquestamcp.MCPConfigProjectionSettingV0,
) []orquestamcp.MCPValidationIssueV0 {
	return orquestamcp.ValidateMCPRequiredSettingsProjectionV0(
		required,
		configProjectionSettingsV0(settings),
	)
}
