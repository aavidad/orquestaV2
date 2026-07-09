package orquestaserver

import (
	"context"
	"regexp"
	"strings"
	"time"

	orquestaestadovivo "orquesta/modulos/orquesta-estado-vivo"
	orquestaobservability "orquesta/modulos/orquesta-observability"
)

type ResidentOperationalStatusSourceV0 struct {
	Tracker          *StatusTrackerV0
	EstadoVivoSource orquestaestadovivo.FuenteEvidenciaEstadoPortV0
	Clock            ClockPortV0
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
	now := clock.Now()
	diagnostic := buildResidentOperationalStatusDiagnosticV0(query, source.Tracker.SnapshotV0(), now)
	diagnostic = source.withEstadoVivoDiagnosticV0(query, diagnostic, now)
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
		Privacy:           orquestaobservability.NewDiagnosticoPrivacyMetadataOnlyV0(),
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

const (
	residentOperationalEstadoVivoEvidenceLimitV0       = 32
	residentOperationalEstadoVivoTimeoutV0             = 500 * time.Millisecond
	residentOperationalEstadoVivoOrphanThresholdV0     = time.Hour
	residentOperationalDiagnosticoCounterBudgetV0      = 24
	residentOperationalDiagnosticoReferenceBudgetV0    = 20
	residentOperationalEstadoVivoCounterBudgetCodeV0   = "estado_vivo_counter_budget"
	residentOperationalEstadoVivoReferenceBudgetCodeV0 = "estado_vivo_reference_budget"
)

func (source ResidentOperationalStatusSourceV0) withEstadoVivoDiagnosticV0(
	query orquestaobservability.OperationalStatusQueryV0,
	diagnostic orquestaobservability.DiagnosticoCompactoV0,
	now time.Time,
) orquestaobservability.DiagnosticoCompactoV0 {
	if source.EstadoVivoSource == nil {
		return diagnostic
	}
	ctx, cancel := context.WithTimeout(context.Background(), residentOperationalEstadoVivoTimeoutV0)
	defer cancel()
	evidencias, err := source.EstadoVivoSource.ListarEvidenciasEstadoV0(
		ctx,
		residentOperationalEstadoVivoFiltroV0(query),
	)
	if err != nil {
		diagnostic.Warnings = residentOperationalAppendWarningOnceV0(
			diagnostic.Warnings,
			"estado_vivo_unavailable",
			orquestaobservability.OperationalStatusSectionSaludV0,
			"fuente de estado vivo no disponible en diagnostico residente",
		)
		return diagnostic
	}
	projection := orquestaestadovivo.ConstruirProyeccionCicloVidaV0(
		evidencias,
		now,
		residentOperationalEstadoVivoOrphanThresholdV0,
	)
	diagnostic.Contadores = residentOperationalCountersWithEstadoVivoV0(
		diagnostic.Contadores,
		evidencias,
		projection,
		&diagnostic.Warnings,
	)
	diagnostic.Referencias = residentOperationalReferencesWithEstadoVivoV0(
		diagnostic.Referencias,
		evidencias,
		&diagnostic.Warnings,
	)
	return diagnostic
}

func residentOperationalEstadoVivoFiltroV0(
	query orquestaobservability.OperationalStatusQueryV0,
) orquestaestadovivo.FiltroEvidenciaEstadoV0 {
	filter := orquestaestadovivo.FiltroEvidenciaEstadoV0{Limit: residentOperationalEstadoVivoEvidenceLimitV0}
	subject := strings.TrimSpace(query.SubjectRef)
	switch {
	case strings.HasPrefix(subject, "run-ref-") || strings.HasPrefix(subject, "run-"):
		filter.RunRef = subject
	case strings.HasPrefix(subject, "goal-ref-") || strings.HasPrefix(subject, "external-goal-ref-"):
		filter.GoalRef = subject
	}
	return filter
}
