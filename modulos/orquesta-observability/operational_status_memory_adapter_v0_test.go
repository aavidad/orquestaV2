package orquestaobservability

import "testing"

func TestOperationalStatusMemoryAdapterV0LookupScopeSubjectCorrelation(t *testing.T) {
	first := validDiagnosticoCompactoV0()
	second := validDiagnosticoCompactoV0()
	second.DiagnosticID = "diagnostic_20260504_000002"
	second.CorrelationID = "corr_20260504_000002"
	second.ProjectionRef = "projection_20260504_000002"
	second.Freshness.WatermarkRef = "watermark_20260504_000002"
	second.Bloqueos = append(second.Bloqueos, DiagnosticoBloqueoV0{
		BlockerRef:   "blocker_20260504_000002",
		Severity:     OrquestaEventSeverityErrorV0,
		OwnerArea:    "runtime",
		Summary:      "Bloqueo agregado alternativo",
		EvidenceRefs: []string{"event_20260504_000012"},
	})
	second.Referencias = append(second.Referencias, DiagnosticoReferenciaV0{
		Rel:        "projection",
		TargetType: "projection",
		TargetRef:  "projection_20260504_000002",
	})

	adapter, err := NewOperationalStatusMemoryAdapterV0([]DiagnosticoCompactoV0{first, second})
	if err != nil {
		t.Fatalf("create adapter: %v", err)
	}

	query := validOperationalStatusQueryV0()
	query.CorrelationID = second.CorrelationID
	query.IncludeSections = []string{
		OperationalStatusSectionEstadoV0,
		OperationalStatusSectionBloqueosV0,
	}
	query.Limit = 1
	query.Freshness = &OperationalStatusFreshnessRequestV0{
		MaxAgeSeconds: 120,
		WatermarkRef:  second.Freshness.WatermarkRef,
	}

	got, err := adapter.QueryOperationalStatusV0(query)
	if err != nil {
		t.Fatalf("query adapter: %v", err)
	}
	if got.DiagnosticID != second.DiagnosticID {
		t.Fatalf("diagnostic_id=%q, want %q", got.DiagnosticID, second.DiagnosticID)
	}
	if len(got.Bloqueos) != 1 {
		t.Fatalf("bloqueos len=%d, want 1", len(got.Bloqueos))
	}
	if got.Salud != nil || got.ActividadReciente != nil || got.Contadores != nil || got.Referencias != nil {
		t.Fatalf("unrequested sections should be omitted: %+v", got)
	}
	if got.Privacy.ContainsSecret || got.Privacy.ContainsTranscript || got.Privacy.ContainsPrompt || got.Privacy.ContainsCompletion || got.Privacy.ContainsConnectionDetail {
		t.Fatalf("privacy flags should remain false: %+v", got.Privacy)
	}
}

func TestOperationalStatusMemoryAdapterV0ValidaQueryYDiagnosticos(t *testing.T) {
	t.Run("query invalida", func(t *testing.T) {
		adapter, err := NewOperationalStatusMemoryAdapterV0([]DiagnosticoCompactoV0{validDiagnosticoCompactoV0()})
		if err != nil {
			t.Fatalf("create adapter: %v", err)
		}
		query := validOperationalStatusQueryV0()
		query.SubjectRef = "/home/alberto/proyecto"

		_, err = adapter.QueryOperationalStatusV0(query)
		assertOperationalStatusIssueV0(t, err, ErrReferenciaNoOpacaV0)
	})

	t.Run("diagnostico invalido en constructor", func(t *testing.T) {
		diagnostic := validDiagnosticoCompactoV0()
		diagnostic.Bloqueos[0].Summary = "access_token=abc123456"

		_, err := NewOperationalStatusMemoryAdapterV0([]DiagnosticoCompactoV0{diagnostic})
		assertOperationalStatusIssueV0(t, err, ErrSecretoDetectadoV0)
	})
}

func TestOperationalStatusMemoryAdapterV0ErroresPublicosDisponibilidadYFrescura(t *testing.T) {
	t.Run("proyeccion no disponible", func(t *testing.T) {
		adapter, err := NewOperationalStatusMemoryAdapterV0(nil)
		if err != nil {
			t.Fatalf("create adapter: %v", err)
		}

		_, err = adapter.QueryOperationalStatusV0(validOperationalStatusQueryV0())
		assertOperationalStatusIssueV0(t, err, ErrProyeccionNoDisponibleV0)
	})

	t.Run("diagnostico no disponible", func(t *testing.T) {
		adapter, err := NewOperationalStatusMemoryAdapterV0([]DiagnosticoCompactoV0{validDiagnosticoCompactoV0()})
		if err != nil {
			t.Fatalf("create adapter: %v", err)
		}
		query := validOperationalStatusQueryV0()
		query.SubjectRef = "project_20260504_999999"

		_, err = adapter.QueryOperationalStatusV0(query)
		assertOperationalStatusIssueV0(t, err, ErrDiagnosticoNoDisponibleV0)
	})

	t.Run("frescura no garantizada", func(t *testing.T) {
		diagnostic := validDiagnosticoCompactoV0()
		diagnostic.Freshness.MaxAgeSeconds = 300
		adapter, err := NewOperationalStatusMemoryAdapterV0([]DiagnosticoCompactoV0{diagnostic})
		if err != nil {
			t.Fatalf("create adapter: %v", err)
		}
		query := validOperationalStatusQueryV0()
		query.Freshness = &OperationalStatusFreshnessRequestV0{
			MaxAgeSeconds: 120,
			WatermarkRef:  diagnostic.Freshness.WatermarkRef,
		}

		_, err = adapter.QueryOperationalStatusV0(query)
		assertOperationalStatusIssueV0(t, err, ErrFrescuraNoGarantizadaV0)
	})
}

func TestOperationalStatusMemoryAdapterV0CopiaDefensiva(t *testing.T) {
	diagnostic := validDiagnosticoCompactoV0()
	adapter, err := NewOperationalStatusMemoryAdapterV0([]DiagnosticoCompactoV0{diagnostic})
	if err != nil {
		t.Fatalf("create adapter: %v", err)
	}

	diagnostic.Bloqueos[0].Summary = "client_secret mutado fuera"
	query := validOperationalStatusQueryV0()
	query.IncludeSections = []string{
		OperationalStatusSectionEstadoV0,
		OperationalStatusSectionBloqueosV0,
	}
	query.Freshness = nil

	first, err := adapter.QueryOperationalStatusV0(query)
	if err != nil {
		t.Fatalf("query first: %v", err)
	}
	first.Bloqueos[0].Summary = "client_secret mutado en respuesta"

	second, err := adapter.QueryOperationalStatusV0(query)
	if err != nil {
		t.Fatalf("query second: %v", err)
	}
	if second.Bloqueos[0].Summary != "Espera evidencia agregada" {
		t.Fatalf("summary=%q, want immutable adapter copy", second.Bloqueos[0].Summary)
	}
}
