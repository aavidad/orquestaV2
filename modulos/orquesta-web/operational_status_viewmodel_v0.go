package orquestaweb

import orquestaobservability "orquesta/modulos/orquesta-observability"

const (
	WebOperationalStatusPanelSchemaV0 = "web_operational_status_panel.v0"

	WebOperationalStatusEstadoOKV0       = "ok"
	WebOperationalStatusEstadoDegradedV0 = "degraded"
	WebOperationalStatusEstadoBlockedV0  = "blocked"
	WebOperationalStatusEstadoFailedV0   = "failed"
	WebOperationalStatusEstadoUnknownV0  = "unknown"
)

type WebOperationalStatusPanelV0 struct {
	SchemaVersion     string                           `json:"schema_version"`
	Locale            string                           `json:"locale"`
	Estado            string                           `json:"estado"`
	EstadoVisual      string                           `json:"estado_visual"`
	Scope             string                           `json:"scope"`
	SubjectRef        string                           `json:"subject_ref,omitempty"`
	CorrelationID     string                           `json:"correlation_id"`
	DiagnosticID      string                           `json:"diagnostic_id"`
	ProjectionRef     string                           `json:"projection_ref"`
	FaseActual        string                           `json:"fase_actual,omitempty"`
	Progreso          WebOperationalStatusProgressV0   `json:"progreso"`
	Fases             []WebOperationalStatusPhaseV0    `json:"fases"`
	Bloqueos          []WebOperationalStatusBlockerV0  `json:"bloqueos"`
	Salud             []WebOperationalStatusHealthV0   `json:"salud"`
	ActividadReciente []WebOperationalStatusActivityV0 `json:"actividad_reciente"`
	Warnings          []WebOperationalStatusWarningV0  `json:"warnings"`
	Frescura          WebOperationalStatusFreshnessV0  `json:"frescura"`
	TieneBloqueos     bool                             `json:"tiene_bloqueos"`
	PrivacyOK         bool                             `json:"privacy_ok"`
}

type WebOperationalStatusProgressV0 struct {
	Completed int      `json:"completed"`
	Total     int      `json:"total,omitempty"`
	Percent   *float64 `json:"percent,omitempty"`
	Summary   string   `json:"summary,omitempty"`
}

type WebOperationalStatusPhaseV0 struct {
	Key      string                         `json:"key"`
	Estado   string                         `json:"estado"`
	Actual   bool                           `json:"actual"`
	Progress WebOperationalStatusProgressV0 `json:"progress"`
}

type WebOperationalStatusBlockerV0 struct {
	Ref          string   `json:"ref"`
	Severity     string   `json:"severity"`
	OwnerArea    string   `json:"owner_area"`
	Summary      string   `json:"summary"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type WebOperationalStatusHealthV0 struct {
	Area         string   `json:"area"`
	Severity     string   `json:"severity"`
	Estado       string   `json:"estado"`
	I18nKey      string   `json:"i18n_key"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type WebOperationalStatusActivityV0 struct {
	Ref         string `json:"ref"`
	OccurredAt  string `json:"occurred_at"`
	Area        string `json:"area"`
	Summary     string `json:"summary"`
	EventRef    string `json:"event_ref,omitempty"`
	ArtifactRef string `json:"artifact_ref,omitempty"`
}

type WebOperationalStatusWarningV0 struct {
	Code    string `json:"code"`
	Section string `json:"section,omitempty"`
	Summary string `json:"summary,omitempty"`
}

type WebOperationalStatusFreshnessV0 struct {
	WatermarkRef  string `json:"watermark_ref"`
	MaxAgeSeconds int    `json:"max_age_seconds"`
	Partial       bool   `json:"partial"`
	Stale         bool   `json:"stale"`
}

func NewWebOperationalStatusPanelV0(locale string, diagnostic orquestaobservability.DiagnosticoCompactoV0) WebOperationalStatusPanelV0 {
	progress := webOperationalStatusProgressV0(diagnostic.Progreso)
	faseActual := trimOperationalStatusV0(diagnostic.Progreso.Phase)
	return WebOperationalStatusPanelV0{
		SchemaVersion:     WebOperationalStatusPanelSchemaV0,
		Locale:            trimOperationalStatusV0(locale),
		Estado:            webOperationalStatusEstadoV0(diagnostic.Estado),
		EstadoVisual:      webOperationalStatusVisualV0(diagnostic.Estado),
		Scope:             trimOperationalStatusV0(diagnostic.Scope),
		SubjectRef:        trimOperationalStatusV0(diagnostic.SubjectRef),
		CorrelationID:     trimOperationalStatusV0(diagnostic.CorrelationID),
		DiagnosticID:      trimOperationalStatusV0(diagnostic.DiagnosticID),
		ProjectionRef:     trimOperationalStatusV0(diagnostic.ProjectionRef),
		FaseActual:        faseActual,
		Progreso:          progress,
		Fases:             webOperationalStatusPhasesV0(faseActual, diagnostic.Estado, progress),
		Bloqueos:          webOperationalStatusBlockersV0(diagnostic.Bloqueos),
		Salud:             webOperationalStatusHealthV0(diagnostic.Salud),
		ActividadReciente: webOperationalStatusActivitiesV0(diagnostic.ActividadReciente),
		Warnings:          webOperationalStatusWarningsV0(diagnostic.Warnings),
		Frescura:          webOperationalStatusFreshnessV0(diagnostic.Freshness),
		TieneBloqueos:     len(diagnostic.Bloqueos) > 0,
		PrivacyOK:         webOperationalStatusPrivacyOKV0(diagnostic.Privacy),
	}
}
