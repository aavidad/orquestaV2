package orquestamcp

import (
	"strings"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func diagnosticsFromRunProgressIssuesMCPAutoprogrammingV0(
	stats *orquestacionnucleoapp.DirectorRunStatsV0,
) []MCPAutoprogrammingDiagnosticV0 {
	if stats == nil {
		return nil
	}
	out := make([]MCPAutoprogrammingDiagnosticV0, 0, len(stats.Progress.Issues))
	for _, issue := range stats.Progress.Issues {
		code := strings.TrimSpace(issue.Code)
		if code == "" {
			continue
		}
		out = append(out, MCPAutoprogrammingDiagnosticV0{
			Code:    code,
			Scope:   strings.Join(compactStringsMCPV0([]string{"run:" + strings.TrimSpace(stats.RunRef), "field:" + strings.TrimSpace(issue.Field)}), " "),
			Message: strings.TrimSpace(issue.Message),
		})
	}
	return out
}
