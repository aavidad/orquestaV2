package main

import (
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

func codexAppServerThreadStatusToGoalWorkStatusV0(status serverCodexAppServerThreadStatusV0) string {
	switch strings.TrimSpace(string(status)) {
	case "systemError":
		return orquestagoal.GoalStatusBlockedV0
	default:
		return orquestagoal.GoalStatusRunningV0
	}
}

func (backend serverCodexAppServerGoalBackendV0) codexAppServerThreadStatusIssueCodeV0(status serverCodexAppServerThreadStatusV0) string {
	switch strings.TrimSpace(string(status)) {
	case "systemError":
		if issueCode := codexAppServerIssueCodeFromLogFileV0(backend.DiagnosticLogPath); issueCode != "" {
			return issueCode
		}
		if issueCode := strings.TrimSpace(backend.AuthIssueCode); issueCode != "" {
			return issueCode
		}
		return "codex_app_server_thread_system_error"
	default:
		return ""
	}
}

func codexAppServerIssueEvidenceRefV0(issueCode string) string {
	switch strings.TrimSpace(issueCode) {
	case "codex_app_server_provider_unauthorized":
		return "evidence-ref-codex-app-server-provider-unauthorized"
	case "codex_app_server_auth_missing":
		return "evidence-ref-codex-app-server-auth-missing"
	default:
		return ""
	}
}
