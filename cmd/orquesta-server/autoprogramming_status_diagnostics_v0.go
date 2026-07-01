package main

import (
	"strings"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func serverAutoprogrammingStatusDiagnosticsFromEffectiveConfigV0(
	config orquestaserver.ServerEffectiveConfigV0,
) []orquestamcp.MCPAutoprogrammingDiagnosticV0 {
	config = orquestaserver.NormalizeServerEffectiveConfigV0(config)
	out := make([]orquestamcp.MCPAutoprogrammingDiagnosticV0, 0, len(config.Diagnostics))
	for _, diagnostic := range config.Diagnostics {
		if strings.TrimSpace(diagnostic.Code) != serverCodexGoalBackendDegradedDiagnosticCodeV0 {
			continue
		}
		out = append(out, orquestamcp.MCPAutoprogrammingDiagnosticV0{
			Code:         strings.TrimSpace(diagnostic.Code),
			Scope:        strings.TrimSpace(diagnostic.Scope),
			Message:      strings.TrimSpace(diagnostic.Message),
			EvidenceRefs: compactServerStringsV0(diagnostic.EvidenceRefs),
		})
	}
	return out
}

func compactServerStringsV0(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}
