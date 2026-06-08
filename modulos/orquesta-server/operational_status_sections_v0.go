package orquestaserver

import (
	"strings"

	orquestaobservability "orquesta/modulos/orquesta-observability"
)

func residentOperationalHealthV0(state StateV0) []orquestaobservability.DiagnosticoSaludCheckV0 {
	health := []orquestaobservability.DiagnosticoSaludCheckV0{
		{
			Area:     "system",
			Severity: "info",
			Estado:   residentOperationalEstadoV0(state),
			I18nKey:  "server.health.resident_status",
		},
		{
			Area:     "system",
			Severity: startupSeverityV0(state),
			Estado:   startupEstadoV0(state),
			I18nKey:  "server.health.startup",
		},
		{
			Area:     "runtime",
			Severity: supervisorSeverityV0(state),
			Estado:   supervisorEstadoV0(state),
			I18nKey:  "server.health.supervisor",
		},
		{
			Area:         "system",
			Severity:     "info",
			Estado:       "ok",
			I18nKey:      "server.health.daemon_logs",
			EvidenceRefs: []string{NormalizeDaemonLogPolicyV0(state.DaemonLogPolicy).PolicyRef},
		},
	}
	if check := selfWatchdogHealthCheckV0(state); check.I18nKey != "" {
		health = append(health, check)
	}
	return health
}

func residentOperationalBlockersV0(state StateV0) []orquestaobservability.DiagnosticoBloqueoV0 {
	var blockers []orquestaobservability.DiagnosticoBloqueoV0
	if strings.TrimSpace(state.LastError) != "" || state.SupervisorErrorTicks > 0 {
		blockers = append(blockers, orquestaobservability.DiagnosticoBloqueoV0{
			BlockerRef:   "blocker-ref-server-operational-status",
			Severity:     "warning",
			OwnerArea:    "system",
			Summary:      "bloqueo residente redactado",
			EvidenceRefs: sanitizeServerEvidenceRefsV0(state.StartupEvidenceRefs),
		})
	}
	if strings.TrimSpace(state.StatePersistStatus) == "degraded" {
		blockers = append(blockers, orquestaobservability.DiagnosticoBloqueoV0{
			BlockerRef: "blocker-ref-server-state-persist",
			Severity:   "warning",
			OwnerArea:  "system",
			Summary:    "persistencia de estado degradada",
		})
	}
	if strings.TrimSpace(state.AuditStatus) == "degraded" {
		blockers = append(blockers, orquestaobservability.DiagnosticoBloqueoV0{
			BlockerRef: "blocker-ref-server-audit-write",
			Severity:   firstNonEmptyServerDiagnosticV0(state.AuditLastSeverity, "warning"),
			OwnerArea:  "system",
			Summary:    "auditoria de servidor degradada",
		})
	}
	if strings.TrimSpace(state.ResponseWriteLastCode) != "" {
		blockers = append(blockers, orquestaobservability.DiagnosticoBloqueoV0{
			BlockerRef: "blocker-ref-server-response-write",
			Severity:   "warning",
			OwnerArea:  "system",
			Summary:    "entrega HTTP degradada",
		})
	}
	return blockers
}

func residentOperationalActivityV0(state StateV0) []orquestaobservability.DiagnosticoActividadV0 {
	var out []orquestaobservability.DiagnosticoActividadV0
	appendActivity := func(ref, occurredAt, area, summary string) {
		if strings.TrimSpace(occurredAt) == "" {
			return
		}
		out = append(out, orquestaobservability.DiagnosticoActividadV0{
			ActivityRef: ref,
			OccurredAt:  occurredAt,
			Area:        area,
			Summary:     summary,
		})
	}
	appendActivity("activity-ref-server-heartbeat", state.LastHeartbeatAt, "system", "heartbeat residente observado")
	appendActivity("activity-ref-server-supervisor", state.LastSupervisorAt, "runtime", "pulso supervisor observado")
	appendActivity("activity-ref-server-startup", state.LastStartupCheckAt, "system", "startup check observado")
	appendActivity("activity-ref-server-state-persist-failed", state.StatePersistLastFailedAt, "system", "fallo de persistencia de estado observado")
	appendActivity("activity-ref-server-audit-write-failed", state.AuditLastFailedAt, "system", "fallo de escritura de auditoria observado")
	appendActivity("activity-ref-server-response-write-failed", state.ResponseWriteLastFailedAt, "system", "fallo de entrega HTTP observado")
	return out
}

func residentOperationalReferencesV0(
	query orquestaobservability.OperationalStatusQueryV0,
	state StateV0,
) []orquestaobservability.DiagnosticoReferenciaV0 {
	refs := []orquestaobservability.DiagnosticoReferenciaV0{
		{Rel: "projection", TargetType: "projection", TargetRef: "projection-ref-operational-status-resident"},
		{Rel: "query", TargetType: "query", TargetRef: query.RequestID},
	}
	if strings.TrimSpace(query.SubjectRef) != "" {
		refs = append(refs, orquestaobservability.DiagnosticoReferenciaV0{
			Rel: "subject", TargetType: "system", TargetRef: strings.TrimSpace(query.SubjectRef),
		})
	}
	if strings.TrimSpace(state.LastSupervisorQueueRef) != "" && serverEvidenceRefPatternV0.MatchString(state.LastSupervisorQueueRef) {
		refs = append(refs, orquestaobservability.DiagnosticoReferenciaV0{
			Rel: "related", TargetType: "system", TargetRef: strings.TrimSpace(state.LastSupervisorQueueRef),
		})
	}
	return refs
}

func residentOperationalWarningsV0() []orquestaobservability.DiagnosticoWarningV0 {
	return []orquestaobservability.DiagnosticoWarningV0{
		{Code: "runtime_not_available", Section: "salud", Summary: "runtime detallado no disponible por source residente"},
		{Code: "audit_not_available", Section: "actividad_reciente", Summary: "auditoria historica no agregada en query vivo"},
		{Code: "daemon-log-summary-only", Section: "salud", Summary: "stdout stderr residentes exponen resumen redactado por defecto"},
	}
}
