package orquestaobservability

import "testing"

func TestValidateOperationalStatusQueryV0Valida(t *testing.T) {
	query := validOperationalStatusQueryV0()

	if err := ValidateOperationalStatusQueryV0(query); err != nil {
		t.Fatalf("query should validate: %v", err)
	}
}

func TestDecodeOperationalStatusQueryV0ReadOnlyEstricto(t *testing.T) {
	data := []byte(`{
		"schema_version": "operational_status_query.v0",
		"request_id": "req_20260504_000020",
		"correlation_id": "corr_20260504_000020",
		"consumer": {"module": "orquesta-cli", "channel": "cli"},
		"locale": "es-ES",
		"scope": "proyecto",
		"subject_ref": "project_20260504_000020",
		"include_sections": ["estado", "progreso"],
		"limit": 10,
		"recovery_action": "retry_runtime"
	}`)

	_, err := DecodeOperationalStatusQueryV0(data)
	assertOperationalStatusIssueV0(t, err, ErrOperationalStatusQueryInvalidaV0)
}

func TestValidateOperationalStatusQueryV0ConsumidorScopesYSecciones(t *testing.T) {
	base := validOperationalStatusQueryV0()

	t.Run("consumidor no autorizado", func(t *testing.T) {
		query := cloneOperationalStatusQueryV0(t, base)
		query.Consumer = OperationalStatusConsumerV0{Module: "orquesta-runtime", Channel: "runtime"}
		assertOperationalStatusIssueV0(t, ValidateOperationalStatusQueryV0(query), ErrConsumidorNoAutorizadoV0)
	})

	t.Run("scope no soportado", func(t *testing.T) {
		query := cloneOperationalStatusQueryV0(t, base)
		query.Scope = "db"
		assertOperationalStatusIssueV0(t, ValidateOperationalStatusQueryV0(query), ErrScopeNoSoportadoV0)
	})

	t.Run("seccion no soportada", func(t *testing.T) {
		query := cloneOperationalStatusQueryV0(t, base)
		query.IncludeSections = []string{"estado", "sql"}
		assertOperationalStatusIssueV0(t, ValidateOperationalStatusQueryV0(query), ErrOperationalStatusQueryInvalidaV0)
	})

	t.Run("demasiadas secciones", func(t *testing.T) {
		query := cloneOperationalStatusQueryV0(t, base)
		query.IncludeSections = []string{
			"estado",
			"progreso",
			"bloqueos",
			"salud",
			"actividad_reciente",
			"contadores",
			"referencias",
		}
		assertOperationalStatusIssueV0(t, ValidateOperationalStatusQueryV0(query), ErrConsultaDemasiadoAmpliaV0)
	})

	t.Run("limite demasiado amplio", func(t *testing.T) {
		query := cloneOperationalStatusQueryV0(t, base)
		query.Limit = maxOperationalLimitV0 + 1
		assertOperationalStatusIssueV0(t, ValidateOperationalStatusQueryV0(query), ErrConsultaDemasiadoAmpliaV0)
	})
}

func TestValidateOperationalStatusQueryV0ReferenciasOpacasYContenidoProhibido(t *testing.T) {
	base := validOperationalStatusQueryV0()

	t.Run("referencia no opaca", func(t *testing.T) {
		query := cloneOperationalStatusQueryV0(t, base)
		query.SubjectRef = "/home/alberto/proyecto"
		assertOperationalStatusIssueV0(t, ValidateOperationalStatusQueryV0(query), ErrReferenciaNoOpacaV0)
	})

	t.Run("home en referencia opaca permitida", func(t *testing.T) {
		query := cloneOperationalStatusQueryV0(t, base)
		query.TraceRef = "trace_home_ref_000001"
		if err := ValidateOperationalStatusQueryV0(query); err != nil {
			t.Fatalf("query should allow opaque home ref: %v", err)
		}
	})

	t.Run("policy ref con token permitida", func(t *testing.T) {
		query := cloneOperationalStatusQueryV0(t, base)
		query.RequestID = "token_policy_ref_20260504_000001"
		if err := ValidateOperationalStatusQueryV0(query); err != nil {
			t.Fatalf("query should allow opaque policy ref: %v", err)
		}
	})

	t.Run("transcript en watermark", func(t *testing.T) {
		query := cloneOperationalStatusQueryV0(t, base)
		query.Freshness = &OperationalStatusFreshnessRequestV0{WatermarkRef: "watermark_transcript_000001"}
		assertOperationalStatusIssueV0(t, ValidateOperationalStatusQueryV0(query), ErrTranscriptNoPermitidoV0)
	})
}
