package orquestaobservability

import "regexp"

const (
	OperationalStatusQuerySchemaVersionV0 = "operational_status_query.v0"
	DiagnosticoCompactoSchemaVersionV0    = "diagnostico_compacto.v0"

	OperationalStatusConsumerCLIChannelV0  = "cli"
	OperationalStatusConsumerMCPChannelV0  = "mcp"
	OperationalStatusConsumerWebChannelV0  = "web"
	OperationalStatusConsumerCoreChannelV0 = "core"

	OperationalStatusScopeSistemaV0  = "sistema"
	OperationalStatusScopeProyectoV0 = "proyecto"
	OperationalStatusScopeFlujoV0    = "flujo"
	OperationalStatusScopeTareaV0    = "tarea"
	OperationalStatusScopeRuntimeV0  = "runtime"
	OperationalStatusScopeCapacityV0 = "capacity"
	OperationalStatusScopeReviewV0   = "review"

	OperationalStatusSectionEstadoV0            = "estado"
	OperationalStatusSectionProgresoV0          = "progreso"
	OperationalStatusSectionBloqueosV0          = "bloqueos"
	OperationalStatusSectionSaludV0             = "salud"
	OperationalStatusSectionActividadRecienteV0 = "actividad_reciente"
	OperationalStatusSectionContadoresV0        = "contadores"
	OperationalStatusSectionReferenciasV0       = "referencias"

	DiagnosticoEstadoOKV0       = "ok"
	DiagnosticoEstadoDegradedV0 = "degraded"
	DiagnosticoEstadoBlockedV0  = "blocked"
	DiagnosticoEstadoFailedV0   = "failed"
	DiagnosticoEstadoUnknownV0  = "unknown"

	ErrOperationalStatusQueryInvalidaV0 = "operational_status_query_invalida"
	ErrConsumidorNoAutorizadoV0         = "consumidor_no_autorizado"
	ErrScopeNoSoportadoV0               = "scope_no_soportado"
	ErrReferenciaNoOpacaV0              = "referencia_no_opaca"
	ErrConsultaDemasiadoAmpliaV0        = "consulta_demasiado_amplia"
	ErrProyeccionNoDisponibleV0         = "proyeccion_no_disponible"
	ErrDiagnosticoNoDisponibleV0        = "diagnostico_no_disponible"
	ErrFrescuraNoGarantizadaV0          = "frescura_no_garantizada"
)

const (
	maxOperationalQueryJSONBytesV0      = 4096
	maxDiagnosticoJSONBytesV0           = 16384
	maxOperationalQuerySectionsV0       = 5
	maxOperationalLimitV0               = 50
	maxOperationalTextRunesV0           = 280
	maxOperationalTokenRunesV0          = 96
	maxOperationalI18nKeyRunesV0        = 120
	maxOperationalListItemsV0           = 20
	maxOperationalWarningItemsV0        = 12
	maxOperationalReferenceItemsV0      = 20
	maxOperationalEvidenceRefsV0        = 8
	maxOperationalCountersV0            = 16
	maxOperationalFreshnessAgeSecondsV0 = 86400
)

var (
	operationalLocalePatternV0     = regexp.MustCompile(`^[a-z]{2}(-[A-Z]{2})?$`)
	operationalI18nKeyPatternV0    = regexp.MustCompile(`^[a-z][a-z0-9_.-]*$`)
	operationalTokenPatternV0      = regexp.MustCompile(`^[a-z][a-z0-9_.:-]*$`)
	operationalCounterKeyPatternV0 = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	operationalConsumerPairsV0     = map[string]string{
		"orquesta-cli":  OperationalStatusConsumerCLIChannelV0,
		"orquesta-mcp":  OperationalStatusConsumerMCPChannelV0,
		"orquesta-web":  OperationalStatusConsumerWebChannelV0,
		"orquesta-core": OperationalStatusConsumerCoreChannelV0,
	}
	allowedOperationalScopesV0 = setV0(
		OperationalStatusScopeSistemaV0,
		OperationalStatusScopeProyectoV0,
		OperationalStatusScopeFlujoV0,
		OperationalStatusScopeTareaV0,
		OperationalStatusScopeRuntimeV0,
		OperationalStatusScopeCapacityV0,
		OperationalStatusScopeReviewV0,
	)
	allowedOperationalSectionsV0 = setV0(
		OperationalStatusSectionEstadoV0,
		OperationalStatusSectionProgresoV0,
		OperationalStatusSectionBloqueosV0,
		OperationalStatusSectionSaludV0,
		OperationalStatusSectionActividadRecienteV0,
		OperationalStatusSectionContadoresV0,
		OperationalStatusSectionReferenciasV0,
	)
	allowedDiagnosticoEstadosV0 = setV0(
		DiagnosticoEstadoOKV0,
		DiagnosticoEstadoDegradedV0,
		DiagnosticoEstadoBlockedV0,
		DiagnosticoEstadoFailedV0,
		DiagnosticoEstadoUnknownV0,
	)
	allowedOperationalAreasV0 = setV0(
		"core",
		"runtime",
		"capacity",
		"review",
		"observability",
		"system",
	)
	allowedDiagnosticoReferenceRelsV0  = setV0("projection", "event", "artifact", "trace", "runtime", "query", "subject", "related")
	allowedDiagnosticoReferenceTypesV0 = setV0(
		"projection",
		"event",
		"artifact",
		"trace",
		"runtime",
		"query",
		"project",
		"task",
		"flow",
		"system",
		"capacity_decision",
		"review",
	)
	operationalSecretTextPartsV0 = []string{
		"secret", "secreto", "token", "password", "credential", "credencial", "api_key", "apikey", "oauth", "bearer",
	}
	operationalTranscriptTextPartsV0 = []string{"transcript", "transcripcion"}
	operationalForbiddenTextPartsV0  = []string{
		"prompt", "completion", "sql", "select", "insert", "update", "delete", "drop",
		"dsn", "connection", "conexion", "table", "tabla", "provider", "proveedor",
		"model_name", "modelo", "/home/", "home=", "$home", "~/", "home_path", "home_ref",
	}
)

type OperationalStatusQueryV0 struct {
	SchemaVersion   string                               `json:"schema_version"`
	RequestID       string                               `json:"request_id"`
	CorrelationID   string                               `json:"correlation_id"`
	Consumer        OperationalStatusConsumerV0          `json:"consumer"`
	Locale          string                               `json:"locale"`
	Scope           string                               `json:"scope"`
	SubjectRef      string                               `json:"subject_ref,omitempty"`
	TraceRef        string                               `json:"trace_ref,omitempty"`
	TimeWindow      *OperationalStatusTimeWindowV0       `json:"time_window,omitempty"`
	IncludeSections []string                             `json:"include_sections"`
	Limit           int                                  `json:"limit"`
	Freshness       *OperationalStatusFreshnessRequestV0 `json:"freshness,omitempty"`
}

type OperationalStatusConsumerV0 struct {
	Module  string `json:"module"`
	Channel string `json:"channel"`
}

type OperationalStatusTimeWindowV0 struct {
	From   string `json:"from,omitempty"`
	To     string `json:"to,omitempty"`
	Preset string `json:"preset,omitempty"`
}

type OperationalStatusFreshnessRequestV0 struct {
	MaxAgeSeconds int    `json:"max_age_seconds,omitempty"`
	WatermarkRef  string `json:"watermark_ref,omitempty"`
}

type DiagnosticoCompactoV0 struct {
	SchemaVersion     string                    `json:"schema_version"`
	DiagnosticID      string                    `json:"diagnostic_id"`
	GeneratedAt       string                    `json:"generated_at"`
	CorrelationID     string                    `json:"correlation_id"`
	Scope             string                    `json:"scope"`
	SubjectRef        string                    `json:"subject_ref,omitempty"`
	ProjectionRef     string                    `json:"projection_ref"`
	Freshness         DiagnosticoFreshnessV0    `json:"freshness"`
	Estado            string                    `json:"estado"`
	Progreso          DiagnosticoProgresoV0     `json:"progreso,omitempty"`
	Salud             []DiagnosticoSaludCheckV0 `json:"salud,omitempty"`
	Bloqueos          []DiagnosticoBloqueoV0    `json:"bloqueos,omitempty"`
	ActividadReciente []DiagnosticoActividadV0  `json:"actividad_reciente,omitempty"`
	Contadores        map[string]float64        `json:"contadores,omitempty"`
	Referencias       []DiagnosticoReferenciaV0 `json:"referencias,omitempty"`
	Warnings          []DiagnosticoWarningV0    `json:"warnings,omitempty"`
	Privacy           DiagnosticoPrivacyV0      `json:"privacy"`
}

type DiagnosticoFreshnessV0 struct {
	WatermarkRef  string `json:"watermark_ref"`
	MaxAgeSeconds int    `json:"max_age_seconds"`
	Partial       bool   `json:"partial"`
	Stale         bool   `json:"stale"`
}

type DiagnosticoProgresoV0 struct {
	Completed int      `json:"completed"`
	Total     int      `json:"total,omitempty"`
	Percent   *float64 `json:"percent,omitempty"`
	Phase     string   `json:"phase,omitempty"`
	Summary   string   `json:"summary,omitempty"`
}

type DiagnosticoSaludCheckV0 struct {
	Area         string   `json:"area"`
	Severity     string   `json:"severity"`
	Estado       string   `json:"estado"`
	I18nKey      string   `json:"i18n_key"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type DiagnosticoBloqueoV0 struct {
	BlockerRef   string   `json:"blocker_ref"`
	Severity     string   `json:"severity"`
	OwnerArea    string   `json:"owner_area"`
	Summary      string   `json:"summary"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type DiagnosticoActividadV0 struct {
	ActivityRef string `json:"activity_ref"`
	OccurredAt  string `json:"occurred_at"`
	Area        string `json:"area"`
	Summary     string `json:"summary"`
	EventRef    string `json:"event_ref,omitempty"`
	ArtifactRef string `json:"artifact_ref,omitempty"`
}

type DiagnosticoReferenciaV0 struct {
	Rel        string `json:"rel"`
	TargetType string `json:"target_type"`
	TargetRef  string `json:"target_ref"`
}

type DiagnosticoWarningV0 struct {
	Code    string `json:"code"`
	Section string `json:"section,omitempty"`
	Summary string `json:"summary,omitempty"`
}

type DiagnosticoPrivacyV0 struct {
	ContainsSecret           bool `json:"contains_secret"`
	ContainsTranscript       bool `json:"contains_transcript"`
	ContainsPrompt           bool `json:"contains_prompt"`
	ContainsCompletion       bool `json:"contains_completion"`
	ContainsConnectionDetail bool `json:"contains_connection_detail"`
}

type OperationalStatusValidationIssueV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

type OperationalStatusValidationErrorV0 struct {
	Issues []OperationalStatusValidationIssueV0 `json:"issues"`
}
