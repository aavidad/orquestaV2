package orquestaserver

import (
	"regexp"
	"strings"
	"time"

	orquestaobservability "orquesta/modulos/orquesta-observability"
)

type ResidentOperationalStatusSourceV0 struct {
	Tracker *StatusTrackerV0
	Clock   ClockPortV0
}

var serverEvidenceRefPatternV0 = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]{7,95}$`)

func (source ResidentOperationalStatusSourceV0) QueryOperationalStatusV0(
	query orquestaobservability.OperationalStatusQueryV0,
) (orquestaobservability.DiagnosticoCompactoV0, error) {
	if err := orquestaobservability.ValidateOperationalStatusQueryV0(query); err != nil {
		return orquestaobservability.DiagnosticoCompactoV0{}, err
	}
	if source.Tracker == nil {
		return orquestaobservability.DiagnosticoCompactoV0{}, operationalStatusUnavailableV0("tracker")
	}
	clock := source.Clock
	if clock == nil {
		clock = SystemClockV0{}
	}
	diagnostic := buildResidentOperationalStatusDiagnosticV0(query, source.Tracker.SnapshotV0(), clock.Now())
	return orquestaobservability.FilterDiagnosticoCompactoForQueryV0(diagnostic, query)
}

func buildResidentOperationalStatusDiagnosticV0(
	query orquestaobservability.OperationalStatusQueryV0,
	state StateV0,
	now time.Time,
) orquestaobservability.DiagnosticoCompactoV0 {
	warnings := residentOperationalWarningsV0()
	return orquestaobservability.DiagnosticoCompactoV0{
		SchemaVersion: "diagnostico_compacto.v0",
		DiagnosticID:  "diagnostic-ref-operational-status-resident",
		GeneratedAt:   formatTimeV0(now),
		CorrelationID: query.CorrelationID,
		Scope:         query.Scope,
		SubjectRef:    strings.TrimSpace(query.SubjectRef),
		ProjectionRef: "projection-ref-operational-status-resident",
		Freshness: orquestaobservability.DiagnosticoFreshnessV0{
			WatermarkRef:  "watermark-ref-server-state",
			MaxAgeSeconds: residentOperationalAgeSecondsV0(state.LastHeartbeatAt, now),
			Partial:       len(warnings) > 0,
		},
		Estado:            residentOperationalEstadoV0(state),
		Progreso:          residentOperationalProgressV0(state),
		Salud:             residentOperationalHealthV0(state),
		Bloqueos:          residentOperationalBlockersV0(state),
		ActividadReciente: residentOperationalActivityV0(state),
		Contadores:        residentOperationalCountersV0(state),
		Referencias:       residentOperationalReferencesV0(query, state),
		Warnings:          warnings,
	}
}

func operationalStatusUnavailableV0(field string) orquestaobservability.OperationalStatusValidationErrorV0 {
	return orquestaobservability.OperationalStatusValidationErrorV0{
		Issues: []orquestaobservability.OperationalStatusValidationIssueV0{{
			Code:  orquestaobservability.ErrProyeccionNoDisponibleV0,
			Field: strings.TrimSpace(field),
		}},
	}
}
