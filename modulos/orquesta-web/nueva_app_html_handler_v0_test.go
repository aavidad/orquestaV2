package orquestaweb

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

func TestNuevaAppHTMLHandlerV0GETMuestraFormularioUsableSinDelegar(t *testing.T) {
	client := &fakeNuevaAppClientV0{}
	handler := NewNuevaAppHTMLHandlerV0(client)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/nueva-app?locale=es", nil)

	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, body)
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("content-type=%q", rec.Header().Get("Content-Type"))
	}
	if client.calls != 0 {
		t.Fatalf("GET no debe delegar: calls=%d", client.calls)
	}
	for _, want := range []string{
		`<form method="post" action="/nueva-app" novalidate>`,
		`href="/nueva-app/guia">Guia</a>`,
		`id="nueva-app-wizard"`,
		`id="guided-assistant"`,
		`data-guided-action="analyze"`,
		`data-guided-action="mobile_both"`,
		`data-guided-action="data_management"`,
		`data-guided-action="architecture_event" data-help="Elige orientada a eventos`,
		`data-guided-action="quality_public" data-help="Eleva pruebas y accesibilidad`,
		`id="guided-active-question"`,
		`id="guided-answer"`,
		`data-guided-answer`,
		`answer_field:answerField,answer:answer`,
		`function submitGuidedAnswer()`,
		`function guidedActiveQuestionField()`,
		`function updateGuidedQuestion()`,
		`data-help="Aplica una respuesta libre`,
		`data-help="Lee la necesidad libre`,
		`data-help="Salta a la revision final`,
		`data-help="Compila una vista previa del GoalWorkSpec`,
		`data-help="Envia el contrato al Director`,
		`data-help="Vuelve al paso anterior`,
		`data-help="Avanza al siguiente paso`,
		`/api/v0/apps/intake/guided-turn`,
		`/api/v0/apps/director/goal/observe`,
		`data-goto-step="0"`,
		`role="tablist"`,
		`role="tab" id="nueva-app-step-tab-0"`,
		`role="tabpanel" id="nueva-app-step-panel-0"`,
		`aria-selected="true"`,
		`function handleTabKeydown(event,node)`,
		`event.key==='ArrowRight'`,
		`event.key==='Home'`,
		`node.tabIndex=active?0:-1`,
		`data-preset="webapp"`,
		`.help-text`,
		`function initAccessibleHelp()`,
		`let guidedSession=null`,
		`if(guidedSession)body.session=guidedSession`,
		`if(out&&out.session){guidedSession=out.session;updateGuidedQuestion();}`,
		`aria-describedby`,
		`matchMedia('(prefers-reduced-motion: reduce)')`,
		`[data-help]::after`,
		`[data-help]:hover::after`,
		`data-help="Opciones: web, API, linea de comandos`,
		`data-help="Opciones: crear app completa`,
		`data-help="Interfaz web para navegador."`,
		`data-help="Opciones: sin elegir`,
		`data-help="Perfiles de usuarios separados por comas.`,
		`Alcance de validacion`,
		`Compatibilidad historica`,
		`Forzar loop historico del Director`,
		`<input type="checkbox" name="director_execution_mode" value="legacy_director_loop">`,
		`<input type="hidden" name="director_execution_mode" value="">`,
		`data-validation-required="Completa este campo."`,
		`id="wizard-errors"`,
		`role="alert"`,
		`data-required="true" aria-required="true" data-label="Nombre"`,
		`name="request_id"`,
		`name="request_kind"`,
		`name="execution_mode"`,
		`name="locale"`,
		`name="nombre"`,
		`name="objetivo"`,
		`name="usuarios_objetivo"`,
		`name="restricciones"`,
		`name="tipo_app"`,
		`name="project_source.kind"`,
		`name="project_source.git_url"`,
		`name="project_source.branch"`,
		`name="project_source.local_path"`,
		`name="project_source.project_ref"`,
		`name="plataformas"`,
		`name="preferencias_tecnicas.arquitectura"`,
		`name="preferencias_tecnicas.restricciones"`,
		`<option value="">Sin preferencia</option>`,
		`<option value="clean_architecture">Arquitectura limpia</option>`,
		`<option value="true">Si</option>`,
		`<option value="false">No</option>`,
		`value="clean_architecture"`,
		`value="modular_monolith"`,
		`value="event_driven"`,
		`name="i18n.enabled"`,
		`name="datos.db_required"`,
		`name="datos.tipos_detallados.0.nombre"`,
		`name="datos.tipos_detallados.3.nombre"`,
		`Dato 3`,
		`Dato 4`,
		`name="datos.fuentes.0.nombre"`,
		`name="datos.fuentes.0.frecuencia"`,
		`name="datos.fuentes.1.nombre"`,
		`name="datos.fuentes.2.nombre"`,
		`name="datos.fuentes.3.nombre"`,
		`name="datos.operacion.criticidad"`,
		`name="datos.operacion.auditoria"`,
		`name="datos.storage.0.tipo"`,
		`name="datos.storage.3.tipo"`,
		`Almacenamiento 3`,
		`Almacenamiento 4`,
		`value="sin_persistencia"`,
		`value="vectorial"`,
		`value="objetos_blob"`,
		`value="clave_valor_cache"`,
		`value="eventos_auditoria"`,
		`value="mixta"`,
		`name="deploy.target"`,
		`name="calidad.pruebas"`,
		`name="calidad.accesibilidad_opciones"`,
		`type="checkbox" name="calidad.accesibilidad_opciones" value="wcag_aa"`,
		`function fieldList(el)`,
		`Array.prototype.forEach.call(el,item=>`,
		`item.checked=values.includes(item.value)`,
		`value="lectores_pantalla"`,
		`value="movimiento_reducido"`,
		`value="normal"`,
		`name="documentacion.usuario"`,
		`name="documentacion.desarrollo"`,
		`name="documentacion.sistemas"`,
		`name="documentacion.profundidad"`,
		`value="profunda"`,
		`data-help="Basica pide minimos`,
		`name="documentacion.locales"`,
		`data-help="Marca Si si la app debe entregar manual de usuario`,
		`name="agentes.autonomia"`,
		`name="integraciones.0.tipo"`,
		`name="integraciones.0.direccion"`,
		`name="integraciones.0.auth"`,
		`name="integraciones.0.data_scope"`,
		`name="integraciones.0.criticidad"`,
		`name="integraciones.1.tipo"`,
		`name="integraciones.3.tipo"`,
		`name="integraciones.4.tipo"`,
		`name="integraciones.5.tipo"`,
		`Integracion 4`,
		`Integracion 6`,
		`value="maps"`,
		`value="file_export"`,
		`value="messaging"`,
		`value="llm"`,
		`value="storage"`,
		`value="other"`,
		`data-summary-integrations="Integraciones"`,
		`data-summary-sensitivity="Sensibilidad"`,
		`data-summary-accessibility="Accesibilidad"`,
		`data-summary-documentation="Documentacion"`,
		`data-summary-deploy="Despliegue"`,
		`.help-open::after`,
		`closeHelpBubbles`,
		`event.pointerType`,
		`Resumen vivo`,
		`id="wizard-final-summary"`,
		`.wizard-ready .hidden-final`,
		`Vista previa del backlog`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET HTML no contiene %q\n%s", want, body)
		}
	}
	for _, forbidden := range []string{
		`<input name="nombre" required`,
		`<select name="tipo_app" required`,
		`<textarea name="objetivo" required`,
		`Please fill out this field`,
		`<select name="director_execution_mode"`,
		`<option value="goal_first">goal_first</option>`,
		`>true</option>`,
		`>false</option>`,
		`>clean_architecture</option>`,
		`>local_path</option>`,
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("GET HTML conserva validacion nativa del navegador %q\n%s", forbidden, body)
		}
	}
	if !strings.Contains(body, `form.addEventListener('submit'`) ||
		!strings.Contains(body, `event.preventDefault()`) ||
		!strings.Contains(body, `focusField(first)`) ||
		!strings.Contains(body, `applyGuidedNeed`) ||
		!strings.Contains(body, `applyServerGuided`) ||
		!strings.Contains(body, `function configureRentalData()`) {
		t.Fatalf("GET HTML conserva validacion nativa del navegador\n%s", body)
	}
	if strings.Contains(body, `data-help=""`) ||
		strings.Contains(body, `data-help="Texto no disponible.`) ||
		strings.Contains(body, `data-help="Text unavailable.`) {
		t.Fatalf("GET HTML contiene tooltip vacio o generico\n%s", body)
	}
}

func TestNuevaAppFormFromValuesV0PriorizaLegacyExplicitoAunqueHiddenVacioV0(t *testing.T) {
	values := url.Values{}
	values.Add("director_execution_mode", "")
	values.Add("director_execution_mode", "legacy_director_loop")

	form := nuevaAppFormFromValuesV0(values)

	if form.DirectorExecutionMode != "legacy_director_loop" {
		t.Fatalf("director_execution_mode=%q", form.DirectorExecutionMode)
	}
}

func TestNuevaAppHTMLV0RenderizaPanelGoalFirstConActualizacion(t *testing.T) {
	endpoint := NewNuevaAppWebEndpointV0(&fakeNuevaAppClientV0{})
	vm := NewWebNuevaAppViewModelV0(validSpecForViewModelV0(), backlogForViewModelV0())
	vm.Estado = WebNuevaAppEstadoDirector
	vm.Director = &WebNuevaAppDirectorV0{
		RunRef:                "run-ref-web-goal-001",
		DirectorExecutionMode: "goal_first",
		GoalRef:               "goal-ref-web-goal-001",
		ExternalGoalRef:       "thread-ref-web-goal-001",
		GoalStatus:            "running",
	}
	page := endpoint.page("es", vm)
	rec := httptest.NewRecorder()

	writeNuevaAppHTMLPageV0(rec, http.StatusOK, page)

	body := rec.Body.String()
	for _, want := range []string{
		`data-goal-panel`,
		`data-run-ref="run-ref-web-goal-001"`,
		`data-goal-ref="goal-ref-web-goal-001"`,
		`data-goal-auto-poll="true"`,
		`data-goal-poll-interval-ms="5000"`,
		`data-goal-max-polls="60"`,
		`Director y goal`,
		`Modo director`,
		`goal_first`,
		`thread-ref-web-goal-001`,
		`data-goal-observe`,
		`Actualizar goal`,
		`function startGoalAutoPoll()`,
		`orquesta-web-nueva-app-auto`,
		`runStatus==='cerrada'||runStatus==='bloqueada'`,
		`window.setTimeout(tick,interval)`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("HTML goal-first no contiene %q\n%s", want, body)
		}
	}
}

func TestNuevaAppHTMLV0DefensivoGoalFirstSinRunNoObserva(t *testing.T) {
	endpoint := NewNuevaAppWebEndpointV0(&fakeNuevaAppClientV0{})
	vm := NewWebNuevaAppViewModelV0(validSpecForViewModelV0(), backlogForViewModelV0())
	vm.Estado = WebNuevaAppEstadoDirector
	vm.Director = &WebNuevaAppDirectorV0{
		DirectorExecutionMode: "goal_first",
		GoalRef:               "goal-ref-web-goal-sin-run-001",
		ExternalGoalRef:       "thread-ref-web-goal-sin-run-001",
		GoalStatus:            "running",
	}
	page := endpoint.page("es", vm)
	rec := httptest.NewRecorder()

	writeNuevaAppHTMLPageV0(rec, http.StatusOK, page)

	body := rec.Body.String()
	for _, want := range []string{
		`data-goal-panel`,
		`data-goal-auto-poll="false"`,
		`data-run-ref=""`,
		`goal-ref-web-goal-sin-run-001`,
		`thread-ref-web-goal-sin-run-001`,
		`goal_first`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("HTML goal-first sin run no contiene %q\n%s", want, body)
		}
	}
	if strings.Contains(body, `<div class="goal-actions"><button`) ||
		strings.Contains(body, `data-goal-panel data-goal-auto-poll="true"`) {
		t.Fatalf("HTML goal-first sin run no debe activar polling/boton de goal\n%s", body)
	}
}

func TestNuevaAppHTMLV0RenderizaDirectorLegacySinPollingGoal(t *testing.T) {
	endpoint := NewNuevaAppWebEndpointV0(&fakeNuevaAppClientV0{})
	vm := NewWebNuevaAppViewModelV0(validSpecForViewModelV0(), backlogForViewModelV0())
	vm.Estado = WebNuevaAppEstadoDirector
	vm.Director = &WebNuevaAppDirectorV0{
		RunRef:                "run-ref-web-legacy-001",
		DirectorExecutionMode: "legacy_director_loop",
		LoopStatus:            "wait_external",
	}
	page := endpoint.page("es", vm)
	rec := httptest.NewRecorder()

	writeNuevaAppHTMLPageV0(rec, http.StatusOK, page)

	body := rec.Body.String()
	for _, want := range []string{
		`data-goal-panel`,
		`data-goal-auto-poll="false"`,
		`run-ref-web-legacy-001`,
		`legacy_director_loop`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("HTML legacy no contiene %q\n%s", want, body)
		}
	}
	if strings.Contains(body, `<div class="goal-actions"><button`) ||
		strings.Contains(body, `data-goal-panel data-goal-auto-poll="true"`) {
		t.Fatalf("HTML legacy no debe activar polling/boton de goal\n%s", body)
	}
}

func TestNuevaAppHTMLHandlerV0GETLocalizaValidacionEnInglesV0(t *testing.T) {
	client := &fakeNuevaAppClientV0{}
	handler := NewNuevaAppHTMLHandlerV0(client)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/nueva-app?locale=en", nil)

	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, body)
	}
	if !strings.Contains(body, `data-validation-required="Complete this field."`) ||
		!strings.Contains(body, `data-validation-summary-title="Required fields missing"`) {
		t.Fatalf("validacion inglesa no localizada\n%s", body)
	}
	for _, want := range []string{
		`Expert data mode`,
		`Data set 1`,
		`Additional integrations`,
		`Options: api, webhook, email, calendar, maps, file_import, file_export, messaging, llm, storage, payments, auth, analytics, search, notifications, or other.`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET ingles no contiene %q\n%s", want, body)
		}
	}
	for _, forbidden := range []string{
		`Modo experto de datos`,
		`Integraciones adicionales`,
		`Dato 1`,
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("GET ingles conserva texto espanol %q\n%s", forbidden, body)
		}
	}
}

func TestNuevaAppHTMLHandlerV0POSTValidoDelegaYRenderizaResultado(t *testing.T) {
	client := &fakeNuevaAppClientV0{
		vm: NewWebNuevaAppViewModelV0(validSpecForViewModelV0(), backlogForViewModelV0()),
	}
	handler := NewNuevaAppHTMLHandlerV0(client)
	values := nuevaAppHTMLValidFormValuesV0()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/nueva-app", bytes.NewBufferString(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, body)
	}
	if client.calls != 1 {
		t.Fatalf("calls=%d", client.calls)
	}
	if client.received.Nombre != "Agenda" ||
		client.received.RequestKind != "crear_app_completa" ||
		client.received.ExecutionMode != "normal" ||
		client.received.ProjectSource.Kind != "github" ||
		client.received.ProjectSource.GitURL != "https://example.test/agenda.git" ||
		client.received.PreferenciasTecnicas.Arquitectura != "event_driven" ||
		!client.received.Datos.DBRequired ||
		len(client.received.Datos.TiposDetallados) != 1 ||
		client.received.Datos.TiposDetallados[0].Nombre != "Pisos" ||
		len(client.received.Datos.Storage) != 1 ||
		client.received.Datos.Storage[0].Tipo != "relacional" ||
		client.received.Deploy.Target != "contenedor" ||
		len(client.received.Calidad.AccesibilidadOpciones) != 2 ||
		client.received.Agentes.Autonomia != "media" ||
		len(client.received.Integraciones) != 2 ||
		client.received.Integraciones[0].Tipo != "api" ||
		client.received.Integraciones[1].Tipo != "maps" {
		t.Fatalf("form delegado inesperado: %+v", client.received)
	}
	for _, want := range []string{"Lista para revisar", "Agenda", "BLG-001", "producto / AppSpecV0"} {
		if !strings.Contains(body, want) {
			t.Fatalf("POST valido no contiene %q\n%s", want, body)
		}
	}
}

func TestNuevaAppHTMLHandlerV0POSTInvalidoRenderizaErrorPublico(t *testing.T) {
	client := &fakeNuevaAppClientV0{
		vm: NewWebNuevaAppErrorViewModelV0("req-invalid", "es", []orquestafactory.ValidationIssue{{
			Code:    orquestafactory.ErrAppSpecInvalida,
			Field:   "nombre",
			Message: "campo obligatorio",
		}}),
	}
	handler := NewNuevaAppHTMLHandlerV0(client)
	values := nuevaAppHTMLValidFormValuesV0()
	values.Set("request_id", "req-invalid")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/nueva-app", bytes.NewBufferString(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, body)
	}
	if client.calls != 1 {
		t.Fatalf("calls=%d", client.calls)
	}
	for _, want := range []string{"Necesita correcciones", "app_spec_invalida", "nombre", "campo obligatorio"} {
		if !strings.Contains(body, want) {
			t.Fatalf("POST invalido no contiene %q\n%s", want, body)
		}
	}
}

func TestNuevaAppHTMLHandlerV0OptionsNoRenderizaNiDelega(t *testing.T) {
	client := &fakeNuevaAppClientV0{}
	handler := NewNuevaAppHTMLHandlerV0(client)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/nueva-app?locale=es", nil)

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Allow") != webPublicHTTPAllowHeaderV0(http.MethodGet, http.MethodPost) {
		t.Fatalf("allow=%q", rec.Header().Get("Allow"))
	}
	if client.calls != 0 || rec.Body.Len() != 0 {
		t.Fatalf("options con efectos calls=%d body=%q", client.calls, rec.Body.String())
	}
}

func nuevaAppHTMLValidFormValuesV0() url.Values {
	values := url.Values{}
	values.Set("request_id", "req-html-001")
	values.Set("locale", "es")
	values.Set("request_kind", "crear_app_completa")
	values.Set("execution_mode", "normal")
	values.Set("nombre", "Agenda")
	values.Set("objetivo", "Coordinar ensayos")
	values.Set("descripcion", "Gestion operativa")
	values.Set("tipo_app", "web")
	values.Set("project_source.kind", "github")
	values.Set("project_source.git_url", "https://example.test/agenda.git")
	values.Set("project_source.branch", "main")
	values.Set("project_source.project_ref", "project-ref-agenda")
	values.Add("plataformas", "web")
	values.Set("preferencias_tecnicas.arquitectura", "hexagonal")
	values.Set("preferencias_tecnicas.arquitectura", "event_driven")
	values.Set("i18n.enabled", "true")
	values.Set("i18n.default_locale", "es")
	values.Set("datos.db_required", "true")
	values.Set("datos.necesidad_funcional", "guardar disponibilidad")
	values.Set("datos.tipos_detallados.0.nombre", "Pisos")
	values.Set("datos.tipos_detallados.0.proposito", "Mostrar alquileres cercanos")
	values.Set("datos.tipos_detallados.0.sensibilidad", "publica")
	values.Set("datos.storage.0.tipo", "relacional")
	values.Set("datos.storage.0.proposito", "consultas transaccionales")
	values.Set("datos.storage.0.requerido", "true")
	values.Set("deploy.target", "contenedor")
	values.Set("calidad.pruebas", "alta")
	values.Set("calidad.accesibilidad", "normal")
	values.Set("calidad.accesibilidad_opciones", "normal,wcag_aa")
	values.Set("calidad.observabilidad", "true")
	values.Set("agentes.revision_humana", "true")
	values.Set("agentes.autonomia", "media")
	values.Set("integraciones.0.tipo", "api")
	values.Set("integraciones.0.nombre", "crm")
	values.Set("integraciones.0.proposito", "sincronizar ensayos")
	values.Set("integraciones.1.tipo", "maps")
	values.Set("integraciones.1.nombre", "capacidad de mapas")
	values.Set("integraciones.1.proposito", "mostrar ubicaciones")
	return values
}
