package orquestamcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestafactoryhttp "orquesta/modulos/orquesta-factory-http"
)

const (
	testAppSpecRequestSchemaV0    = "app_spec_request.v0"
	testAppSpecSchemaV0           = "app_spec.v0"
	testBacklogInicialPropuestoV0 = "backlog_inicial_propuesto.v0"
)

func TestMCPNuevaAppDescriptorV0Compacto(t *testing.T) {
	descriptor := MCPNuevaAppDescriptorV0()

	if descriptor.Name != MCPNuevaAppToolNameV0 || descriptor.Version != MCPNuevaAppToolVersionV0 {
		t.Fatalf("descriptor identidad: %+v", descriptor)
	}
	if descriptor.ResourceURI != MCPNuevaAppResourceURIV0 || descriptor.PromptName != MCPNuevaAppPromptNameV0 {
		t.Fatalf("descriptor resource/prompt: %+v", descriptor)
	}
	if !strings.Contains(descriptor.InputSchema, "AppSpecRequestV0") ||
		!strings.Contains(descriptor.InputSchema, "request_kind") ||
		!strings.Contains(descriptor.Output, "compact") {
		t.Fatalf("descriptor no referencia contrato compacto: %+v", descriptor)
	}
	if !mcpStringInSetForTestV0(descriptor.Invariantes, "apps generadas con arquitectura hexagonal estricta") {
		t.Fatalf("descriptor no declara hexagonalidad estricta: %+v", descriptor.Invariantes)
	}

	payload, err := json.Marshal(descriptor)
	if err != nil {
		t.Fatalf("marshal descriptor: %v", err)
	}
	text := string(payload)
	for _, forbidden := range []string{"usuarios_objetivo", "preferencias_tecnicas", "modulos_iniciales", "microtareas\":["} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("descriptor contiene dump o detalle interno %q: %s", forbidden, text)
		}
	}
	if len(text) > 1100 {
		t.Fatalf("descriptor demasiado largo: %d bytes", len(text))
	}
}

func TestToAppSpecRequestV0PoneSourceYConservaCamposClave(t *testing.T) {
	input := MCPNuevaAppToolInputV0{
		RequestID:     " req-envelope ",
		CorrelationID: " corr-1 ",
		Respuesta:     "completa",
		AppSpecRequest: orquestafactory.AppSpecRequestV0{
			SchemaVersion: testAppSpecRequestSchemaV0,
			Source:        "orquesta-web",
			Locale:        "xx-no-es-bcp47",
			Nombre:        " Agenda ",
			Objetivo:      "Coordinar ensayos",
			TipoApp:       "web",
			Plataformas:   []string{"desktop", "desktop", ""},
			Deploy:        orquestafactory.DeployRequestV0{Target: "proveedor-no-soportado"},
		},
	}

	req, correlationID := ToAppSpecRequestV0(input)

	if req.Source != MCPNuevaAppSourceV0 {
		t.Fatalf("source=%q", req.Source)
	}
	if req.RequestID != "req-envelope" || correlationID != "corr-1" {
		t.Fatalf("request/correlation: request=%q correlation=%q", req.RequestID, correlationID)
	}
	if req.Locale != "xx-no-es-bcp47" ||
		req.Nombre != " Agenda " ||
		req.Objetivo != "Coordinar ensayos" ||
		req.TipoApp != "web" ||
		req.Deploy.Target != "proveedor-no-soportado" ||
		len(req.Plataformas) != 3 {
		t.Fatalf("mapper altero campos de negocio: %+v", req)
	}
}

func TestNewMCPNuevaAppOKResultV0ContieneSpecYBacklogCompactos(t *testing.T) {
	spec := validMCPAppSpecV0()
	backlog := validMCPBacklogV0()

	result := NewMCPNuevaAppOKResultV0(spec, backlog, "corr-ok")

	if result.Estado != MCPNuevaAppEstadoOKV0 || result.RequestID != "req-ok" || result.CorrelationID != "corr-ok" {
		t.Fatalf("result identidad: %+v", result)
	}
	if result.AppSpec.SchemaVersion != testAppSpecSchemaV0 ||
		result.AppSpec.SpecID != "spec-agenda" ||
		result.AppSpec.Nombre != "Agenda" ||
		result.AppSpec.Slug != "agenda" ||
		result.AppSpec.RequestKind != orquestafactory.RequestKindDocumentarAppV0 ||
		result.AppSpec.ExecutionMode != orquestafactory.ExecutionModeDebugV0 ||
		result.AppSpec.DeployTarget != "local" ||
		result.AppSpec.ArchitecturePolicy != "hexagonal_estricta" {
		t.Fatalf("app_spec compacto: %+v", result.AppSpec)
	}
	if result.Backlog.SchemaVersion != testBacklogInicialPropuestoV0 ||
		result.Backlog.SpecID != "spec-agenda" ||
		result.Backlog.Estado != orquestafactory.BacklogInicialEstadoPreviewNoEjecutableV0 ||
		result.Backlog.DirectorHandoff != orquestafactory.BacklogDirectorHandoffContractV0 ||
		result.Backlog.DirectorHandoffRef != "app_spec:spec-agenda" ||
		result.Backlog.Fases != 2 ||
		result.Backlog.Microtareas != 1 ||
		len(result.Backlog.ContratosRequeridos) != 2 {
		t.Fatalf("backlog compacto: %+v", result.Backlog)
	}
	if len(result.Preguntas) != 1 || result.Preguntas[0] != "Confirmar calendario inicial" {
		t.Fatalf("preguntas=%+v", result.Preguntas)
	}

	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	if strings.Contains(string(payload), "modulos_iniciales") || strings.Contains(string(payload), "write_set_previsto") {
		t.Fatalf("resultado no compacto: %s", payload)
	}
}

func TestNewMCPNuevaAppErrorResultV0PublicoNoInventaBacklog(t *testing.T) {
	result := NewMCPNuevaAppErrorResultV0("req-bad", "corr-bad", []orquestafactory.ValidationIssue{
		{Code: orquestafactory.ErrIdiomaInvalido, Field: "locale", Message: "locale BCP 47 invalido"},
		{Code: "stack_privado", Field: "internal.trace"},
	})

	if result.Estado != MCPNuevaAppEstadoErrorV0 || result.RequestID != "req-bad" || result.CorrelationID != "corr-bad" {
		t.Fatalf("result identidad: %+v", result)
	}
	if result.AppSpec.SchemaVersion != "" || result.AppSpec.SpecID != "" || result.Backlog.SchemaVersion != "" || result.Backlog.SpecID != "" ||
		result.Backlog.Fases != 0 || result.Backlog.Microtareas != 0 {
		t.Fatalf("error no debe inventar spec/backlog: %+v", result)
	}
	if len(result.Errores) != 2 {
		t.Fatalf("errores=%+v", result.Errores)
	}
	if result.Errores[0].Code != orquestafactory.ErrIdiomaInvalido || result.Errores[0].Field != "locale" {
		t.Fatalf("error publico conocido: %+v", result.Errores[0])
	}
	if result.Errores[1].Code != orquestafactory.ErrAppSpecInvalida || result.Errores[1].Message != MCPNuevaAppDefaultErrorV0 {
		t.Fatalf("error publico sanitizado: %+v", result.Errores[1])
	}
	if len(result.Preguntas) != 0 || len(result.Supuestos) != 0 {
		t.Fatalf("error no debe inventar preguntas/supuestos: %+v", result)
	}
}

func TestNewMCPNuevaAppToolExecutorV0RejectsCredentialsInServerURL(t *testing.T) {
	_, err := NewMCPNuevaAppToolExecutorV0("https://user:secret@example.com", time.Second)
	if err == nil || !strings.Contains(err.Error(), "credenciales") {
		t.Fatalf("error credenciales esperado: %v", err)
	}
}

func TestMCPNuevaAppToolExecutorV0RechazaEndpointAmbiguo(t *testing.T) {
	executor, err := NewMCPNuevaAppToolExecutorV0("http://operator-mcp-test.local/base", time.Second)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}
	executor.Endpoint = "https://evil.test/api"
	_, err = executor.Execute(context.Background(), MCPNuevaAppToolInputV0{
		RequestID: "req-endpoint",
		AppSpecRequest: orquestafactory.AppSpecRequestV0{
			SchemaVersion: orquestafactory.AppSpecRequestSchemaV0,
			Locale:        "es",
			Nombre:        "Agenda",
			Objetivo:      "Coordinar ensayos",
			TipoApp:       "web",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "endpoint debe ser path relativo") {
		t.Fatalf("error endpoint esperado: %v", err)
	}
}

func TestMCPNuevaAppToolExecutorV0ReturnsOKResultFromFactoryHTTPPort(t *testing.T) {
	handler := orquestafactoryhttp.NewAppSpecHTTPHandlerV0(func() time.Time {
		return time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	})
	executor, err := newLocalMCPNuevaAppToolExecutorV0(handler)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := executor.Execute(context.Background(), MCPNuevaAppToolInputV0{
		RequestID:     "req-envelope",
		CorrelationID: "corr-1",
		AppSpecRequest: orquestafactory.AppSpecRequestV0{
			SchemaVersion: orquestafactory.AppSpecRequestSchemaV0,
			Locale:        "es",
			Nombre:        "Agenda",
			Objetivo:      "Coordinar ensayos",
			TipoApp:       "web",
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Estado != MCPNuevaAppEstadoOKV0 || result.RequestID != "req-envelope" || result.CorrelationID != "corr-1" {
		t.Fatalf("result identidad: %+v", result)
	}
	if result.AppSpec.SchemaVersion != orquestafactory.AppSpecSchemaV0 || result.AppSpec.SpecID == "" {
		t.Fatalf("app spec invalida: %+v", result.AppSpec)
	}
	if result.AppSpec.ArchitecturePolicy != "hexagonal_estricta" {
		t.Fatalf("architecture policy=%q", result.AppSpec.ArchitecturePolicy)
	}
	if result.Backlog.SchemaVersion == "" || result.Backlog.Microtareas == 0 || result.Backlog.Fases == 0 {
		t.Fatalf("backlog invalido: %+v", result.Backlog)
	}
}

func mcpStringInSetForTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestMCPNuevaAppToolExecutorV0ReturnsPublicErrorResultFromFactoryHTTPPort(t *testing.T) {
	handler := orquestafactoryhttp.NewAppSpecHTTPHandlerV0(func() time.Time { return time.Now().UTC() })
	executor, err := newLocalMCPNuevaAppToolExecutorV0(handler)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := executor.Execute(context.Background(), MCPNuevaAppToolInputV0{
		RequestID:     "req-bad",
		CorrelationID: "corr-bad",
		AppSpecRequest: orquestafactory.AppSpecRequestV0{
			SchemaVersion: orquestafactory.AppSpecRequestSchemaV0,
			Locale:        "xx-no-es-bcp47",
			Nombre:        "Agenda",
			Objetivo:      "Coordinar ensayos",
			TipoApp:       "web",
		},
	})
	if err != nil {
		t.Fatalf("execute error inesperado: %v", err)
	}
	if result.Estado != MCPNuevaAppEstadoErrorV0 || len(result.Errores) == 0 {
		t.Fatalf("result error esperado: %+v", result)
	}
	if result.Errores[0].Code != orquestafactory.ErrAppSpecInvalida && result.Errores[0].Code != orquestafactory.ErrIdiomaInvalido {
		t.Fatalf("codigo error inesperado: %+v", result.Errores)
	}
}

func TestMCPNuevaAppToolExecutorV0ReturnsTransportErrorOnNon2XX(t *testing.T) {
	executor, err := newLocalMCPNuevaAppToolExecutorV0(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"debug":"privado"}`))
	}))
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	_, err = executor.Execute(context.Background(), MCPNuevaAppToolInputV0{
		RequestID: "req-500",
		AppSpecRequest: orquestafactory.AppSpecRequestV0{
			SchemaVersion: orquestafactory.AppSpecRequestSchemaV0,
			Locale:        "es",
			Nombre:        "Agenda",
			Objetivo:      "Coordinar ensayos",
			TipoApp:       "web",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "status 500") {
		t.Fatalf("error 500 esperado: %v", err)
	}
}

func newLocalMCPNuevaAppToolExecutorV0(handler http.Handler) (*MCPNuevaAppToolExecutorV0, error) {
	executor, err := NewMCPNuevaAppToolExecutorV0("http://operator-mcp-test.local", time.Second)
	if err != nil {
		return nil, err
	}
	executor.HTTPClient = &http.Client{Transport: localMCPNuevaAppRoundTripperV0{handler: handler}}
	return executor, nil
}

type localMCPNuevaAppRoundTripperV0 struct {
	handler http.Handler
}

func (transport localMCPNuevaAppRoundTripperV0) RoundTrip(req *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	transport.handler.ServeHTTP(rec, req)
	return rec.Result(), nil
}

func validMCPAppSpecV0() orquestafactory.AppSpecV0 {
	return orquestafactory.AppSpecV0{
		SchemaVersion: testAppSpecSchemaV0,
		SpecID:        "spec-agenda",
		RequestID:     "req-ok",
		Locale:        "es",
		RequestKind:   orquestafactory.RequestKindDocumentarAppV0,
		ExecutionMode: orquestafactory.ExecutionModeDebugV0,
		App: orquestafactory.AppInfoV0{
			Nombre:   "Agenda",
			Slug:     "agenda",
			Objetivo: "Coordinar ensayos",
			TipoApp:  "web",
		},
		Scope: orquestafactory.ScopeV0{
			Supuestos:         []string{"Arquitectura hexagonal estricta"},
			PreguntasAbiertas: []string{"Confirmar calendario inicial"},
		},
		Architecture: orquestafactory.ArchitectureV0{
			Patron:             "hexagonal",
			ContratosEsperados: []string{"ArquitecturaHexagonalEstricta v0"},
		},
		I18N: orquestafactory.I18NSpecV0{
			Locales: []string{"es", "en", "es"},
		},
		Deploy: orquestafactory.DeploySpecV0{Target: "local"},
	}
}

func validMCPBacklogV0() orquestafactory.BacklogInicialPropuestoV0 {
	return orquestafactory.BacklogInicialPropuestoV0{
		SchemaVersion: testBacklogInicialPropuestoV0,
		SpecID:        "spec-agenda",
		Estado:        orquestafactory.BacklogInicialEstadoPreviewNoEjecutableV0,
		Freshness: orquestafactory.BacklogFreshnessV0{
			SourceRef: "app_spec:spec-agenda",
		},
		DirectorHandoff: orquestafactory.BacklogDirectorHandoffV0{
			Status:           orquestafactory.BacklogDirectorHandoffStatusPendienteV0,
			RequiredContract: orquestafactory.BacklogDirectorHandoffContractV0,
			RequiredInputRef: "app_spec:spec-agenda",
		},
		Fases: []orquestafactory.FaseInicialV0{
			{ID: "discovery"},
			{ID: "implementacion"},
		},
		Microtareas: []orquestafactory.MicrotareaPropuestaV0{
			{ID: "BLG-001", WriteSetPrevisto: []string{"docs/app_spec.md"}},
		},
		ContratosRequeridos: []string{"AppSpecV0", "FunctionContract v0", "AppSpecV0"},
		Riesgos:             []string{"Cerrar contratos"},
		PreguntasAbiertas:   []string{"Confirmar calendario inicial"},
	}
}
