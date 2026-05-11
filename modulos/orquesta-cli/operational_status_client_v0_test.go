package orquestacli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	orquestaobservability "orquesta/modulos/orquesta-observability"
)

func TestOperationalStatusCliClientV0ExitoPropagaCorrelacionYEnvelopeCanonico(t *testing.T) {
	var received orquestaobservability.OperationalStatusQueryV0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s", r.Method)
		}
		if r.URL.Path != OperationalStatusCliEndpointV0 {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if got := r.Header.Get(OperationalStatusCliCorrelationHeaderV0); got != "corr-op-1" {
			t.Fatalf("correlation=%q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("content-type=%q", got)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Fatalf("accept=%q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if received.SchemaVersion != orquestaobservability.OperationalStatusQuerySchemaVersionV0 ||
			received.RequestID != "req-op-1" ||
			received.CorrelationID != "corr-op-1" {
			t.Fatalf("request ids/schema inesperados: %+v", received)
		}
		if received.Consumer.Module != "orquesta-cli" || received.Consumer.Channel != orquestaobservability.OperationalStatusConsumerCLIChannelV0 {
			t.Fatalf("consumer inesperado: %+v", received.Consumer)
		}
		if received.Locale != "es-ES" {
			t.Fatalf("locale inesperado: %q", received.Locale)
		}
		writeOperationalStatusSuccessV0(t, w, received)
	}))
	defer server.Close()

	client := mustNewOperationalStatusCliClientV0(t, server.URL, 1200*time.Millisecond)
	env := client.ConsultarEstadoOperativo(context.Background(), CliInvocationContextV0{
		RequestID:     "req-op-1",
		CorrelationID: "corr-op-1",
		ServerURL:     server.URL,
		Timeout:       time.Second,
		OutputFormat:  CliOutputFormatJSONV0,
		Locale:        "es-ES",
	}, minimalOperationalStatusQueryForCliV0())

	if !env.OK || env.RequestID != "req-op-1" || env.CorrelationID != "corr-op-1" {
		t.Fatalf("envelope ids/ok inesperado: %+v", env)
	}
	if env.Contract != CliContractOperationalStatusV0 || env.Version != CliContractVersionOperationalV0 {
		t.Fatalf("contrato inesperado: %s %s", env.Contract, env.Version)
	}
	if env.Meta.StatusCode != http.StatusOK || env.Meta.Transporte != CliTransportRESTV0 || env.Meta.Retryable {
		t.Fatalf("meta inesperada: %+v", env.Meta)
	}
	data, ok := env.Data.(orquestaobservability.DiagnosticoCompactoV0)
	if !ok {
		t.Fatalf("data type=%T", env.Data)
	}
	if data.SchemaVersion != orquestaobservability.DiagnosticoCompactoSchemaVersionV0 ||
		data.CorrelationID != "corr-op-1" ||
		data.Scope != orquestaobservability.OperationalStatusScopeProyectoV0 ||
		data.Estado != orquestaobservability.DiagnosticoEstadoOKV0 {
		t.Fatalf("diagnostico inesperado: %+v", data)
	}
}

func TestOperationalStatusCliClientV0Respuesta400DevuelveErroresPublicos(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(orquestaobservability.OperationalStatusValidationErrorV0{
			Issues: []orquestaobservability.OperationalStatusValidationIssueV0{{
				Code:  orquestaobservability.ErrScopeNoSoportadoV0,
				Field: "scope",
			}},
		})
	}))
	defer server.Close()

	client := mustNewOperationalStatusCliClientV0(t, server.URL, time.Second)
	env := client.ConsultarEstadoOperativo(context.Background(), invocationForOperationalCliV0(server.URL), minimalOperationalStatusQueryForCliV0())

	if env.OK || env.Data != nil || env.Meta.StatusCode != http.StatusBadRequest || env.Meta.Retryable {
		t.Fatalf("envelope 400 inesperado: %+v", env)
	}
	if len(env.Errores) != 1 {
		t.Fatalf("errores=%+v", env.Errores)
	}
	if err := env.Errores[0]; err.Codigo != orquestaobservability.ErrScopeNoSoportadoV0 ||
		err.Campo != "scope" ||
		err.MensajeI18N != "orquesta_observability.errores."+orquestaobservability.ErrScopeNoSoportadoV0 {
		t.Fatalf("error publico inesperado: %+v", err)
	}
}

func TestOperationalStatusCliClientV0Status500NoFiltraBodyPrivado(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "stack privado token=secreto /home/alberto/runtime", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := mustNewOperationalStatusCliClientV0(t, server.URL, time.Second)
	env := client.ConsultarEstadoOperativo(context.Background(), invocationForOperationalCliV0(server.URL), minimalOperationalStatusQueryForCliV0())

	if env.OK || env.Meta.StatusCode != http.StatusInternalServerError || !env.Meta.Retryable {
		t.Fatalf("envelope 500 inesperado: %+v", env)
	}
	if len(env.Errores) != 1 || env.Errores[0].Codigo != CliErrErrorTransporteV0 {
		t.Fatalf("errores=%+v", env.Errores)
	}
	raw, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, forbidden := range []string{"stack privado", "token=secreto", "/home/alberto", "runtime"} {
		if bytes.Contains(raw, []byte(forbidden)) {
			t.Fatalf("envelope filtra detalle privado %q: %s", forbidden, string(raw))
		}
	}
}

func TestOperationalStatusCliClientV0TimeoutDevuelveErrorTransporte(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(validOperationalStatusDiagnosticoForCliV0("corr-timeout", "project_20260504_000001"))
	}))
	defer server.Close()

	client := mustNewOperationalStatusCliClientV0(t, server.URL, 2*time.Millisecond)
	env := client.ConsultarEstadoOperativo(context.Background(), invocationForOperationalCliV0(server.URL), minimalOperationalStatusQueryForCliV0())

	if env.OK || len(env.Errores) != 1 {
		t.Fatalf("timeout envelope inesperado: %+v", env)
	}
	if env.Errores[0].Codigo != CliErrErrorTransporteV0 || env.Errores[0].Detalle != "timeout" || !env.Meta.Retryable {
		t.Fatalf("timeout error inesperado: errores=%+v meta=%+v", env.Errores, env.Meta)
	}
}

func TestOperationalStatusCliClientV0RespuestaInvalida(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"schema_version": "diagnostico_compacto.v0"})
	}))
	defer server.Close()

	client := mustNewOperationalStatusCliClientV0(t, server.URL, time.Second)
	env := client.ConsultarEstadoOperativo(context.Background(), invocationForOperationalCliV0(server.URL), minimalOperationalStatusQueryForCliV0())

	if env.OK || len(env.Errores) == 0 {
		t.Fatalf("respuesta invalida no detectada: %+v", env)
	}
	if env.Errores[0].Codigo != orquestaobservability.ErrOperationalStatusQueryInvalidaV0 {
		t.Fatalf("codigo inesperado: %+v", env.Errores)
	}
	if env.Data != nil || env.Meta.StatusCode != http.StatusOK {
		t.Fatalf("respuesta invalida con data/status inesperado: %+v", env)
	}
}

func TestOperationalStatusCliClientV0RechazaServerURLConCredenciales(t *testing.T) {
	_, err := NewOperationalStatusCliClientV0("https://usuario:secreto@example.test", time.Second)
	if !IsCliClientErrorCodeV0(err, CliErrConfiguracionInvalidaV0) {
		t.Fatalf("error constructor=%v", err)
	}
	var cliErr CliClientErrorV0
	if !errors.As(err, &cliErr) || cliErr.Field != "server_url" || cliErr.Detail != "server_url_no_debe_contener_credenciales" {
		t.Fatalf("error publico inesperado: %+v", cliErr)
	}
	if strings.Contains(cliErr.Detail, "usuario") || strings.Contains(cliErr.Detail, "secreto") {
		t.Fatalf("detalle filtra credenciales: %+v", cliErr)
	}
}

func mustNewOperationalStatusCliClientV0(t *testing.T, serverURL string, timeout time.Duration) *OperationalStatusCliClientV0 {
	t.Helper()
	client, err := NewOperationalStatusCliClientV0(serverURL, timeout)
	if err != nil {
		t.Fatalf("NewOperationalStatusCliClientV0: %v", err)
	}
	return client
}

func invocationForOperationalCliV0(serverURL string) CliInvocationContextV0 {
	return CliInvocationContextV0{
		RequestID:     "req-op-test",
		CorrelationID: "corr-op-test",
		ServerURL:     serverURL,
		Timeout:       time.Second,
		OutputFormat:  CliOutputFormatJSONV0,
	}
}

func minimalOperationalStatusQueryForCliV0() orquestaobservability.OperationalStatusQueryV0 {
	return orquestaobservability.OperationalStatusQueryV0{
		Scope:           orquestaobservability.OperationalStatusScopeProyectoV0,
		SubjectRef:      "project_20260504_000001",
		IncludeSections: []string{orquestaobservability.OperationalStatusSectionEstadoV0, orquestaobservability.OperationalStatusSectionProgresoV0},
		Limit:           10,
	}
}

func writeOperationalStatusSuccessV0(t *testing.T, w http.ResponseWriter, query orquestaobservability.OperationalStatusQueryV0) {
	t.Helper()
	if err := orquestaobservability.ValidateDiagnosticoCompactoV0(validOperationalStatusDiagnosticoForCliV0(query.CorrelationID, query.SubjectRef)); err != nil {
		t.Fatalf("fixture diagnostico invalida: %v", err)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(validOperationalStatusDiagnosticoForCliV0(query.CorrelationID, query.SubjectRef))
}

func validOperationalStatusDiagnosticoForCliV0(correlationID string, subjectRef string) orquestaobservability.DiagnosticoCompactoV0 {
	percent := 60.0
	return orquestaobservability.DiagnosticoCompactoV0{
		SchemaVersion: orquestaobservability.DiagnosticoCompactoSchemaVersionV0,
		DiagnosticID:  "diagnostic_20260504_000001",
		GeneratedAt:   "2026-05-04T10:30:00Z",
		CorrelationID: correlationID,
		Scope:         orquestaobservability.OperationalStatusScopeProyectoV0,
		SubjectRef:    subjectRef,
		ProjectionRef: "projection_20260504_000001",
		Freshness: orquestaobservability.DiagnosticoFreshnessV0{
			WatermarkRef:  "watermark_20260504_000001",
			MaxAgeSeconds: 60,
			Partial:       false,
			Stale:         false,
		},
		Estado: orquestaobservability.DiagnosticoEstadoOKV0,
		Progreso: orquestaobservability.DiagnosticoProgresoV0{
			Completed: 3,
			Total:     5,
			Percent:   &percent,
			Phase:     "programacion",
			Summary:   "Progreso agregado compacto",
		},
		Salud: []orquestaobservability.DiagnosticoSaludCheckV0{{
			Area:     "core",
			Severity: orquestaobservability.OrquestaEventSeverityInfoV0,
			Estado:   orquestaobservability.DiagnosticoEstadoOKV0,
			I18nKey:  "observability.health.core_ok",
		}},
		Referencias: []orquestaobservability.DiagnosticoReferenciaV0{{
			Rel:        "projection",
			TargetType: "projection",
			TargetRef:  "projection_20260504_000001",
		}},
		Privacy: orquestaobservability.DiagnosticoPrivacyV0{},
	}
}
