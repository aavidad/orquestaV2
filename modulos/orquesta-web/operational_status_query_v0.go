package orquestaweb

import orquestaobservability "orquesta/modulos/orquesta-observability"

type WebOperationalStatusQueryInputV0 struct {
	RequestID     string
	CorrelationID string
	Locale        string
	Scope         string
	SubjectRef    string
	TraceRef      string
	Limit         int
}

func NewWebOperationalStatusQueryV0(input WebOperationalStatusQueryInputV0) orquestaobservability.OperationalStatusQueryV0 {
	locale := trimOperationalStatusV0(input.Locale)
	if locale == "" {
		locale = "es-ES"
	}
	scope := trimOperationalStatusV0(input.Scope)
	if scope == "" {
		scope = orquestaobservability.OperationalStatusScopeFlujoV0
	}
	limit := input.Limit
	if limit <= 0 {
		limit = 10
	}
	return orquestaobservability.OperationalStatusQueryV0{
		SchemaVersion: orquestaobservability.OperationalStatusQuerySchemaVersionV0,
		RequestID:     trimOperationalStatusV0(input.RequestID),
		CorrelationID: trimOperationalStatusV0(input.CorrelationID),
		Consumer: orquestaobservability.OperationalStatusConsumerV0{
			Module:  "orquesta-web",
			Channel: orquestaobservability.OperationalStatusConsumerWebChannelV0,
		},
		Locale:     locale,
		Scope:      scope,
		SubjectRef: trimOperationalStatusV0(input.SubjectRef),
		TraceRef:   trimOperationalStatusV0(input.TraceRef),
		IncludeSections: []string{
			orquestaobservability.OperationalStatusSectionEstadoV0,
			orquestaobservability.OperationalStatusSectionProgresoV0,
			orquestaobservability.OperationalStatusSectionBloqueosV0,
			orquestaobservability.OperationalStatusSectionSaludV0,
			orquestaobservability.OperationalStatusSectionActividadRecienteV0,
		},
		Limit: limit,
	}
}
