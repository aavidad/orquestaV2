package orquestacli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	orquestagovernance "orquesta/modulos/orquesta-governance"
)

func TestGovernanceCatalogCliReaderV0ExitoPropagaCorrelacionYEnvelopeCanonico(t *testing.T) {
	t.Parallel()

	var gotMethod string
	var gotPath string
	var gotHeader string
	var gotQuery governanceCatalogQueryHTTPRequestV0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotHeader = r.Header.Get(GovernanceCatalogCliCorrelationHeaderV0)
		if err := json.NewDecoder(r.Body).Decode(&gotQuery); err != nil {
			t.Fatalf("Decode(request) error = %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(testGovernanceCatalogResultV0()); err != nil {
			t.Fatalf("Encode(response) error = %v", err)
		}
	}))
	defer server.Close()

	client, err := NewGovernanceCatalogCliReaderV0(server.URL, 0)
	if err != nil {
		t.Fatalf("NewGovernanceCatalogCliReaderV0() error = %v", err)
	}

	inv := NormalizeCliInvocationContextV0(CliInvocationContextV0{
		ServerURL:     server.URL,
		CorrelationID: "corr-gov-001",
	})
	query := orquestagovernance.GovernanceCatalogQueryV0{
		Module: "orquesta-cli",
		Role:   "director",
		Phase:  "brainstorming",
		Tags:   []string{"hexagonal", "i18n", "hexagonal"},
	}

	out := client.ConsultarCatalogo(context.Background(), inv, query)

	if !out.OK {
		t.Fatalf("OK = false, errores = %#v", out.Errores)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method = %q, want %q", gotMethod, http.MethodPost)
	}
	if gotPath != GovernanceCatalogCliEndpointV0 {
		t.Fatalf("path = %q, want %q", gotPath, GovernanceCatalogCliEndpointV0)
	}
	if gotHeader != inv.CorrelationID {
		t.Fatalf("X-Correlation-ID = %q, want %q", gotHeader, inv.CorrelationID)
	}
	if gotQuery.Module != "orquesta-cli" || gotQuery.Role != "director" || gotQuery.Phase != "brainstorming" {
		t.Fatalf("query enviada = %#v", gotQuery)
	}
	if len(gotQuery.Tags) != 2 || gotQuery.Tags[0] != "hexagonal" || gotQuery.Tags[1] != "i18n" {
		t.Fatalf("tags = %#v", gotQuery.Tags)
	}
	if out.Contract != CliContractGovernanceCatalogV0 || out.Version != CliContractVersionGovernanceV0 {
		t.Fatalf("contract/version = %q/%q", out.Contract, out.Version)
	}

	data, ok := out.Data.(orquestagovernance.GovernanceCatalogQueryResultV0)
	if !ok {
		t.Fatalf("data type = %T", out.Data)
	}
	if len(data.Effective) != 1 || data.Counters.Effective != 1 || data.Counters.Proposed != 2 || data.Counters.Quarantine != 1 {
		t.Fatalf("data = %#v", data)
	}
}

func TestGovernanceCatalogCliReaderV0Respuesta400DevuelveErroresPublicos(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(governanceCatalogHTTPErrorV0{
			Errores: []governanceCatalogIssueV0{
				{Codigo: "governance_catalog_invalid_source", Campo: "module"},
			},
		})
	}))
	defer server.Close()

	client, err := NewGovernanceCatalogCliReaderV0(server.URL, 0)
	if err != nil {
		t.Fatalf("NewGovernanceCatalogCliReaderV0() error = %v", err)
	}

	out := client.ConsultarCatalogo(context.Background(), NormalizeCliInvocationContextV0(CliInvocationContextV0{ServerURL: server.URL}), orquestagovernance.GovernanceCatalogQueryV0{})

	if out.OK {
		t.Fatalf("OK = true, want false")
	}
	if len(out.Errores) != 1 {
		t.Fatalf("len(errores) = %d, want 1", len(out.Errores))
	}
	if out.Errores[0].Codigo != "governance_catalog_invalid_source" || out.Errores[0].Campo != "module" {
		t.Fatalf("errores[0] = %#v", out.Errores[0])
	}
}

func TestGovernanceCatalogCliReaderV0Status500NoFiltraBodyPrivado(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "secret stacktrace", http.StatusInternalServerError)
	}))
	defer server.Close()

	client, err := NewGovernanceCatalogCliReaderV0(server.URL, 0)
	if err != nil {
		t.Fatalf("NewGovernanceCatalogCliReaderV0() error = %v", err)
	}

	out := client.ConsultarCatalogo(context.Background(), NormalizeCliInvocationContextV0(CliInvocationContextV0{ServerURL: server.URL}), orquestagovernance.GovernanceCatalogQueryV0{})

	if out.OK {
		t.Fatalf("OK = true, want false")
	}
	if len(out.Errores) != 1 || out.Errores[0].Codigo != CliErrErrorTransporteV0 {
		t.Fatalf("errores = %#v", out.Errores)
	}
	if out.Meta.StatusCode != http.StatusInternalServerError || !out.Meta.Retryable {
		t.Fatalf("meta = %#v", out.Meta)
	}
	if out.Errores[0].Detalle == "secret stacktrace" {
		t.Fatalf("detalle filtro body privado = %#v", out.Errores[0])
	}
}

func TestGovernanceCatalogCliReaderV0TimeoutDevuelveErrorTransporte(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewGovernanceCatalogCliReaderV0(server.URL, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("NewGovernanceCatalogCliReaderV0() error = %v", err)
	}

	out := client.ConsultarCatalogo(context.Background(), NormalizeCliInvocationContextV0(CliInvocationContextV0{ServerURL: server.URL}), orquestagovernance.GovernanceCatalogQueryV0{})

	if out.OK {
		t.Fatalf("OK = true, want false")
	}
	if len(out.Errores) != 1 || out.Errores[0].Codigo != CliErrErrorTransporteV0 || out.Errores[0].Detalle != "timeout" {
		t.Fatalf("errores = %#v", out.Errores)
	}
}

func TestGovernanceCatalogCliReaderV0RespuestaInvalida(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"effective":[{"name":"bad"}],"counters":{"effective":1,"proposed":0,"quarantine":0}}`))
	}))
	defer server.Close()

	client, err := NewGovernanceCatalogCliReaderV0(server.URL, 0)
	if err != nil {
		t.Fatalf("NewGovernanceCatalogCliReaderV0() error = %v", err)
	}

	out := client.ConsultarCatalogo(context.Background(), NormalizeCliInvocationContextV0(CliInvocationContextV0{ServerURL: server.URL}), orquestagovernance.GovernanceCatalogQueryV0{})

	if out.OK {
		t.Fatalf("OK = true, want false")
	}
	if len(out.Errores) != 1 || out.Errores[0].Codigo != CliErrRespuestaInvalidaV0 {
		t.Fatalf("errores = %#v", out.Errores)
	}
}

func TestGovernanceCatalogCliReaderV0RechazaServerURLConCredenciales(t *testing.T) {
	t.Parallel()

	_, err := NewGovernanceCatalogCliReaderV0("https://user:secret@example.test", 0)
	if !IsCliClientErrorCodeV0(err, CliErrConfiguracionInvalidaV0) {
		t.Fatalf("error = %v, want %q", err, CliErrConfiguracionInvalidaV0)
	}
}

func testGovernanceCatalogResultV0() orquestagovernance.GovernanceCatalogQueryResultV0 {
	return orquestagovernance.GovernanceCatalogQueryResultV0{
		Effective: []orquestagovernance.GovernanceCatalogEntryV0{
			testEffectiveGovernanceEntryV0(),
		},
		Counters: orquestagovernance.GovernanceCatalogCountersV0{
			Effective:  1,
			Proposed:   2,
			Quarantine: 1,
		},
	}
}

func testEffectiveGovernanceEntryV0() orquestagovernance.GovernanceCatalogEntryV0 {
	decisionRef := "decision://director/architecture-vote"
	promotedAt := "2026-05-04T10:00:00Z"
	return orquestagovernance.GovernanceCatalogEntryV0{
		Kind:    "rule",
		Name:    "Hexagonal por defecto",
		Summary: "Mantener conectores y puertos por modulo.",
		Scope: orquestagovernance.GovernanceScopeV0{
			Level:   "module",
			Modules: []string{"orquesta-cli"},
			Roles:   []string{"director"},
			Phases:  []string{"brainstorming"},
			Tags:    []string{"hexagonal", "i18n"},
		},
		Status: orquestagovernance.GovernanceStatusV0{
			CatalogState: orquestagovernance.GovernanceCatalogStateEffectiveV0,
			ReviewState:  orquestagovernance.GovernanceReviewStateApprovedEffectiveV0,
		},
		Version: orquestagovernance.GovernanceVersionV0{
			ContractVersion:       orquestagovernance.GovernanceCatalogVersionV0,
			DocumentVersion:       "2026-05-04",
			SourceVersionEvidence: "decision_v2",
		},
		Origin: orquestagovernance.GovernanceOriginV0{
			SourceType:           "director_decision",
			SourceRef:            "docs/decisiones.md#hexagonal",
			ForensicInventoryRef: "GOV-001",
			LegacyPublicIDPolicy: "no_legacy_ids_as_canon",
		},
		Promotion: orquestagovernance.GovernancePromotionV0{
			Criterion:                "decision_v2_aprobada",
			DecisionRef:              &decisionRef,
			PromotedAt:               &promotedAt,
			HistoricalReviewRequired: true,
		},
		SecretControls: orquestagovernance.GovernanceSecretControlsV0{
			ContainsSecretMaterial:   false,
			ReviewMethod:             "structured_review",
			StructuredMarkersChecked: []string{"no_tokens", "no_home_paths"},
		},
	}
}
