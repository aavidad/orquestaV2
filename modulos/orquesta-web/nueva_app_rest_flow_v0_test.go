package orquestaweb

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestafactoryhttp "orquesta/modulos/orquesta-factory-http"
)

func TestNuevaAppRESTFlowV0ValidaWebRESTFactory(t *testing.T) {
	const requestID = "req-web-int-006a"
	server := httptest.NewServer(orquestafactoryhttp.NewAppSpecHTTPHandlerV0(fixedRESTFlowClockV0))
	defer server.Close()

	transport := &captureCorrelationTransportV0{base: http.DefaultTransport}
	client := NewRESTSolicitarNuevaAppClientV0(server.URL, time.Second)
	client.HTTPClient = &http.Client{Timeout: time.Second, Transport: transport}

	vm, err := client.SolicitarNuevaApp(context.Background(), WebNuevaAppFormV0{
		RequestID:        requestID,
		Locale:           "es",
		Nombre:           "Agenda de Ensayos",
		Objetivo:         "Coordinar ensayos y disponibilidad del equipo sin requerir proyecto previo",
		Descripcion:      "Vista compacta para preparar sesiones y avisos.",
		TipoApp:          "web",
		UsuariosObjetivo: []string{"coordinacion", "equipo"},
		Plataformas:      []string{"web"},
		Calidad:          WebNuevaAppCalidadFormV0{Pruebas: "media", Accesibilidad: "wcag_aa"},
	})
	if err != nil {
		t.Fatalf("SolicitarNuevaApp error: %v", err)
	}

	if vm.Estado != WebNuevaAppEstadoValida {
		t.Fatalf("estado=%q vm=%+v", vm.Estado, vm)
	}
	if vm.RequestID != requestID || transport.responseCorrelation != requestID {
		t.Fatalf("correlation/request_id incoherentes: vm=%q response=%q", vm.RequestID, transport.responseCorrelation)
	}
	if vm.AppSpecSchemaVersion != orquestafactory.AppSpecSchemaV0 ||
		vm.BacklogSchemaVersion != orquestafactory.BacklogInicialPropuestoSchemaV0 {
		t.Fatalf("schemas: app=%q backlog=%q", vm.AppSpecSchemaVersion, vm.BacklogSchemaVersion)
	}
	if vm.ResumenApp.Nombre != "Agenda de Ensayos" ||
		vm.ResumenApp.Slug != "agenda-de-ensayos" ||
		vm.ResumenApp.Locale != "es" ||
		vm.ResumenApp.TipoApp != "web" ||
		vm.ResumenApp.Objetivo == "" {
		t.Fatalf("resumen inesperado: %+v", vm.ResumenApp)
	}
	if len(vm.Fases) == 0 {
		t.Fatalf("fases vacias: %+v", vm.Fases)
	}
	if len(vm.Microtareas) == 0 {
		t.Fatalf("microtareas vacias: %+v", vm.Microtareas)
	}
	if vm.SpecID == "" || vm.ValidationEstadoFuente != "valida" {
		t.Fatalf("spec/validacion inesperadas: spec_id=%q validation=%q", vm.SpecID, vm.ValidationEstadoFuente)
	}
	if containsRESTFlowValueV0(vm.ContratosRequeridos, "PersistenceRepository v0") {
		t.Fatalf("flujo sin DB no debe requerir persistencia: %+v", vm.ContratosRequeridos)
	}
	for _, task := range vm.Microtareas {
		if task.ModuloSugerido == "persistence" {
			t.Fatalf("flujo sin DB no debe proponer microtarea de persistencia: %+v", task)
		}
	}
}

func TestNuevaAppRESTFlowV0InvalidaFormularioAViewModelPublico(t *testing.T) {
	const requestID = "req-web-int-invalid"
	server := httptest.NewServer(orquestafactoryhttp.NewAppSpecHTTPHandlerV0(fixedRESTFlowClockV0))
	defer server.Close()

	client := NewRESTSolicitarNuevaAppClientV0(server.URL, time.Second)
	vm, err := client.SolicitarNuevaApp(context.Background(), WebNuevaAppFormV0{
		RequestID: requestID,
		Locale:    "locale invalido",
		Nombre:    "Agenda",
		Objetivo:  "Coordinar ensayos",
		TipoApp:   "web",
	})
	if err != nil {
		t.Fatalf("errores publicos de formulario no deben ser transporte: %v", err)
	}

	if vm.Estado != WebNuevaAppEstadoInvalida || vm.RequestID != requestID || vm.Locale != "locale invalido" {
		t.Fatalf("vm invalido inesperado: %+v", vm)
	}
	if len(vm.ErroresPublicos) == 0 {
		t.Fatalf("errores_publicos vacios: %+v", vm)
	}
	if !hasRESTFlowIssueCodeV0(vm.ErroresPublicos, orquestafactory.ErrIdiomaInvalido) {
		t.Fatalf("no aparece error publico de idioma: %+v", vm.ErroresPublicos)
	}
	if len(vm.Fases) != 0 ||
		len(vm.Microtareas) != 0 ||
		len(vm.ContratosRequeridos) != 0 ||
		len(vm.Riesgos) != 0 ||
		vm.AppSpecSchemaVersion != "" ||
		vm.BacklogSchemaVersion != "" ||
		vm.SpecID != "" {
		t.Fatalf("request invalida no debe inventar backlog/spec: %+v", vm)
	}
}

type captureCorrelationTransportV0 struct {
	base                http.RoundTripper
	responseCorrelation string
}

func (transport *captureCorrelationTransportV0) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := transport.base.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	transport.responseCorrelation = resp.Header.Get(SolicitarNuevaAppCorrelationV0)
	return resp, nil
}

func fixedRESTFlowClockV0() time.Time {
	return time.Date(2026, 5, 4, 10, 30, 0, 0, time.UTC)
}

func hasRESTFlowIssueCodeV0(values []WebNuevaAppIssueV0, code string) bool {
	for _, value := range values {
		if value.Code == code {
			return true
		}
	}
	return false
}

func containsRESTFlowValueV0(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
