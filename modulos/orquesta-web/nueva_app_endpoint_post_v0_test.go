package orquestaweb

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestNuevaAppWebEndpointV0POSTJSONDelegaAlClienteYRenderizaViewModel(t *testing.T) {
	client := &fakeNuevaAppClientV0{
		vm: NewWebNuevaAppViewModelV0(validSpecForViewModelV0(), backlogForViewModelV0()),
	}
	endpoint := NewNuevaAppWebEndpointV0(client)
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(WebNuevaAppFormV0{
		RequestID: "req-web-006",
		Locale:    "es",
		Nombre:    "Agenda",
		Objetivo:  "Coordinar ensayos",
		TipoApp:   "web",
	}); err != nil {
		t.Fatalf("encode form: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/nueva-app", body)
	req.Header.Set("Content-Type", "application/json")

	endpoint.ServeHTTP(rec, req)

	page := decodeNuevaAppWebPageTestV0(t, rec)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if client.calls != 1 {
		t.Fatalf("calls=%d", client.calls)
	}
	if client.received.Locale != "es" ||
		client.received.Nombre != "Agenda" ||
		client.received.Objetivo != "Coordinar ensayos" ||
		client.received.TipoApp != "web" {
		t.Fatalf("form enviado inesperado: %+v", client.received)
	}
	if page.ViewModel.Estado != WebNuevaAppEstadoValida ||
		page.Textos.Estado != "Lista para revisar" ||
		len(page.ViewModel.Microtareas) == 0 {
		t.Fatalf("page submit inesperada: %+v", page)
	}
}

func TestNuevaAppWebEndpointV0POSTJSONArrancaDirectorSiEstaConfigurado(t *testing.T) {
	directorClient := &fakeArrancarDirectorAppClientV0{
		vm: NewWebNuevaAppDirectorViewModelV0(
			WebNuevaAppFormV0{
				RequestID: "req-web-director-001",
				Locale:    "es",
				Nombre:    "Agenda",
				Objetivo:  "Coordinar ensayos",
				TipoApp:   "mixed",
			},
			WebArrancarDirectorAppResultV0{
				Estado:    ArrancarDirectorAppEstadoOKV0,
				RequestID: "req-web-director-001",
				RunRef:    "run-ref-agenda-001",
				PhaseID:   "brainstorming_arquitectura",
				DirectorTasks: []WebNuevaAppDirectorTaskV0{{
					TaskRef:        "task-ref-director-001",
					AgentRequestID: "agent-ref-director-001",
					Capacity:       "high",
				}},
				StartedAgents: []string{"agent-ref-director-001"},
			},
		),
	}
	specClient := &fakeNuevaAppClientV0{}
	endpoint := NuevaAppWebEndpointV0{
		Client:         specClient,
		DirectorClient: directorClient,
		Catalog:        NewNuevaAppI18nCatalogV0(),
	}
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(WebNuevaAppFormV0{
		RequestID: "req-web-director-001",
		Locale:    "es",
		Nombre:    "Agenda",
		Objetivo:  "Coordinar ensayos",
		TipoApp:   "mixed",
	}); err != nil {
		t.Fatalf("encode form: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/nueva-app", body)
	req.Header.Set("Content-Type", "application/json")

	endpoint.ServeHTTP(rec, req)

	page := decodeNuevaAppWebPageTestV0(t, rec)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if directorClient.calls != 1 || specClient.calls != 0 {
		t.Fatalf("calls director=%d spec=%d", directorClient.calls, specClient.calls)
	}
	if page.ViewModel.Estado != WebNuevaAppEstadoDirector ||
		page.Textos.Estado != "Director arrancado" ||
		page.ViewModel.Director == nil ||
		page.ViewModel.Director.RunRef != "run-ref-agenda-001" {
		t.Fatalf("page director inesperada: %+v", page)
	}
	if len(page.ViewModel.Director.DirectorTasks) != 1 ||
		page.ViewModel.Director.DirectorTasks[0].AgentRequestID != "agent-ref-director-001" {
		t.Fatalf("director tasks=%+v", page.ViewModel.Director.DirectorTasks)
	}
}

func TestNuevaAppWebEndpointV0POSTJSONRenderizaBacklogPreviewCompacto(t *testing.T) {
	client := &fakeNuevaAppClientV0{
		vm: NewWebNuevaAppViewModelV0(validSpecForViewModelV0(), backlogForViewModelV0()),
	}
	endpoint := NewNuevaAppWebEndpointV0(client)
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(WebNuevaAppFormV0{
		RequestID: "req-web-007",
		Locale:    "es",
		Nombre:    "Agenda",
		Objetivo:  "Coordinar ensayos",
		TipoApp:   "web",
	}); err != nil {
		t.Fatalf("encode form: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/nueva-app", body)
	req.Header.Set("Content-Type", "application/json")

	endpoint.ServeHTTP(rec, req)

	page := decodeNuevaAppWebPageTestV0(t, rec)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if page.Textos.BacklogPreview.Titulo != "Vista previa del backlog" ||
		page.Textos.BacklogPreview.Fases != "Fases propuestas" ||
		page.Textos.BacklogPreview.Microtareas != "Microtareas propuestas" {
		t.Fatalf("textos preview sin i18n: %+v", page.Textos.BacklogPreview)
	}
	preview := page.ViewModel.BacklogPreview
	if preview.SchemaVersion != NuevaAppBacklogPreviewSchemaV0 ||
		preview.TotalFases != 1 ||
		preview.TotalMicrotareas != 1 ||
		!preview.TieneBloqueos {
		t.Fatalf("preview inesperado: %+v", preview)
	}
	if len(preview.Fases) != 1 ||
		preview.Fases[0].Key == "" ||
		preview.Fases[0].Titulo == "" ||
		preview.Fases[0].Microtareas != 1 {
		t.Fatalf("fases preview sin semantica basica: %+v", preview.Fases)
	}
	if len(preview.Microtareas) != 1 ||
		preview.Microtareas[0].Key == "" ||
		preview.Microtareas[0].Fase == "" ||
		preview.Microtareas[0].Titulo == "" ||
		preview.Microtareas[0].ModuloFrontera == "" ||
		len(preview.Microtareas[0].WriteSetPrevisto) == 0 ||
		len(preview.Microtareas[0].Bloqueos) == 0 ||
		preview.Microtareas[0].CriterioCierre == "" {
		t.Fatalf("microtareas preview sin semantica basica: %+v", preview.Microtareas)
	}
}

func TestNuevaAppWebEndpointV0POSTFormURLEncodedDelegaSinTemplates(t *testing.T) {
	client := &fakeNuevaAppClientV0{
		vm: NewWebNuevaAppViewModelV0(validSpecForViewModelV0(), backlogForViewModelV0()),
	}
	endpoint := NewNuevaAppWebEndpointV0(client)
	values := url.Values{}
	values.Set("request_id", "req-form-006")
	values.Set("locale", "es")
	values.Set("nombre", "Agenda")
	values.Set("objetivo", "Coordinar ensayos")
	values.Set("tipo_app", "web")
	values.Add("plataformas", "web,mobile")
	values.Set("integraciones.0.tipo", "api")
	values.Set("integraciones.0.nombre", "crm")
	values.Set("integraciones.0.proposito", "sincronizar ensayos")
	values.Set("integraciones.0.requerido", "true")
	values.Set("datos.db_required", "true")
	values.Set("datos.necesidad_funcional", "guardar disponibilidad")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/nueva-app", bytes.NewBufferString(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	endpoint.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if client.calls != 1 || !client.received.Datos.DBRequired || client.received.Datos.NecesidadFuncional != "guardar disponibilidad" {
		t.Fatalf("form urlencoded no delegado: calls=%d form=%+v", client.calls, client.received)
	}
	if len(client.received.Plataformas) != 2 || client.received.Plataformas[1] != "mobile" {
		t.Fatalf("plataformas=%+v", client.received.Plataformas)
	}
	if len(client.received.Integraciones) != 1 ||
		client.received.Integraciones[0].Tipo != "api" ||
		client.received.Integraciones[0].Nombre != "crm" ||
		client.received.Integraciones[0].Proposito != "sincronizar ensayos" ||
		!client.received.Integraciones[0].Requerido {
		t.Fatalf("integraciones=%+v", client.received.Integraciones)
	}
}
