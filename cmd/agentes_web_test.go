package cmd

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/db"
)

func testMuxAgentesWeb() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", webHandlerDash)
	mux.HandleFunc("/tareas", webHandlerTareas)
	mux.HandleFunc("/tareas/", webRouterTareas)
	mux.HandleFunc("/propuestas", webHandlerPropuestas)
	mux.HandleFunc("/propuestas/", webRouterPropuestas)
	mux.HandleFunc("/agentes", webHandlerAgentes)
	mux.HandleFunc("/agentes/", webRouterAgentes)
	mux.HandleFunc("/runtimes", webHandlerRuntimes)
	mux.HandleFunc("/runtimes/", webRouterRuntimes)
	mux.HandleFunc("/time-travel", webHandlerTimeTravel)
	mux.HandleFunc("/time-travel/", webRouterTimeTravel)
	registerAPIRoutes(mux)
	return mux
}

func TestWebAgentesPanelMuestraEstadoVivo(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoID, "principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-agentes-001",
		ResumenContinuidad: "panel web",
		Branch:             "main",
		Host:               "worker-1",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if err := db.UpsertRuntimeHandleDesdeSesion(sesion); err != nil {
		t.Fatalf("upsert runtime handle: %v", err)
	}
	runtimeInst, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtimeInst == nil {
		t.Fatalf("get runtime: %+v err=%v", runtimeInst, err)
	}
	if _, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "arrancar",
		PayloadJSON: `{"motivo":"panel"}`,
	}); err != nil {
		t.Fatalf("encolar order: %v", err)
	}
	if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "Supervisor",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "nudge",
		PayloadJSON: `{"texto":"sigue"}`,
	}); err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	if _, err := db.CrearRuntimeCheckpoint(&db.RuntimeCheckpoint{
		Agente:         "Codex1",
		ProyectoID:     &proyectoID,
		SesionID:       &sesion.ID,
		RuntimeID:      &runtimeInst.ID,
		CheckpointKind: "handoff_prepare",
		Resumen:        "checkpoint panel",
		Branch:         "main",
		CWD:            filepath.Join(tmp, "orquestador"),
		PayloadJSON:    `{"ok":true}`,
		ResumeStrategy: "resumen_y_payload",
		Source:         "web-test",
	}); err != nil {
		t.Fatalf("crear checkpoint: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Seguir panel agentes",
		Descripcion: "Cobertura web",
		Modulo:      "web",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "tester",
		ProyectoID:  &proyectoID,
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/agentes?lang=en", nil)
	testMuxAgentesWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, token := range []string{
		"live agents panel",
		"Codex1",
		"orquestador",
		"codex-cli",
		"sess-agentes-001",
		"pending mailbox",
		"auto-refresh every 5 s",
		"/agentes/Codex1",
	} {
		if !strings.Contains(body, token) {
			t.Fatalf("panel agentes sin %q:\n%s", token, body)
		}
	}
	if !strings.Contains(body, `<html lang="en">`) {
		t.Fatalf("lang html inesperado: %s", body)
	}
	if got := rec.Header().Get("Content-Language"); got != "en" {
		t.Fatalf("Content-Language=%q", got)
	}
}

func TestWebAgenteDetalleMuestraControlPlaneYDetalleOperativo(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-agentes-detalle",
		ResumenContinuidad: "detalle agente",
		Branch:             "feature/agentes",
		Host:               "worker-2",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if err := db.UpsertRuntimeHandleDesdeSesion(sesion); err != nil {
		t.Fatalf("upsert runtime handle: %v", err)
	}
	runtimeInst, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtimeInst == nil {
		t.Fatalf("get runtime: %+v err=%v", runtimeInst, err)
	}
	orderID, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "checkpoint",
		PayloadJSON: `{"motivo":"detalle"}`,
	})
	if err != nil {
		t.Fatalf("encolar order: %v", err)
	}
	if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:     "Codex2",
		ToAgente:       "Codex1",
		ProyectoID:     &proyectoID,
		RuntimeOrderID: &orderID,
		Kind:           "discordia",
		PayloadJSON:    `{"motivo":"conflicto"}`,
	}); err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	cpID, err := db.CrearRuntimeCheckpoint(&db.RuntimeCheckpoint{
		Agente:         "Codex1",
		ProyectoID:     &proyectoID,
		SesionID:       &sesion.ID,
		RuntimeID:      &runtimeInst.ID,
		CheckpointKind: "checkpoint",
		Resumen:        "detalle",
		Branch:         "feature/agentes",
		CWD:            filepath.Join(tmp, "orquestador"),
		PayloadJSON:    `{"ok":true}`,
		ResumeStrategy: "payload",
		Source:         "detalle-test",
	})
	if err != nil {
		t.Fatalf("crear checkpoint: %v", err)
	}
	if _, err := db.RegistrarRuntimeTranscript(&db.RuntimeTranscriptEntry{
		RuntimeID:      runtimeInst.ID,
		Agente:         "Codex1",
		ProyectoID:     &proyectoID,
		Stream:         "pty_out",
		Text:           "¿me dejas seguir con el refactor?",
		NormalizedText: "me dejas seguir con el refactor",
		Classification: "approval_request",
	}); err != nil {
		t.Fatalf("crear transcript: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/agentes/Codex1?lang=en", nil)
	testMuxAgentesWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, token := range []string{
		"Control",
		"Admin",
		"feature/agentes",
		"codex-cli",
		"#" + itoa(orderID),
		"discordia",
		"/time-travel/" + itoa(cpID),
		"detalle-test",
		"Conversation",
		"approval_request",
		"¿me dejas seguir con el refactor?",
		"Reset reanimation",
	} {
		if !strings.Contains(body, token) {
			t.Fatalf("detalle agente sin %q:\n%s", token, body)
		}
	}
}

func TestWebAgenteNuevoAutoNombrePorProveedor(t *testing.T) {
	prepararDBTemporalCmd(t)

	form := strings.NewReader("proveedor=claude&rol=programador")
	req := httptest.NewRequest(http.MethodPost, "/agentes/nuevo", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	testMuxAgentesWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	if agente, err := db.GetAgente("Claude1"); err != nil || agente == nil {
		t.Fatalf("agente auto no creado: %+v err=%v", agente, err)
	}
}

func TestWebAgenteAsignacionActivaProyecto(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if _, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	}); err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	form := strings.NewReader("proyecto=orquestador&nota=frente+principal")
	req := httptest.NewRequest(http.MethodPost, "/agentes/Codex1/asignacion", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	testMuxAgentesWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	agente := "Codex1"
	asignaciones, err := db.ListarAsignaciones(db.FiltroAsignaciones{Agente: &agente})
	if err != nil {
		t.Fatalf("listar asignaciones: %v", err)
	}
	if len(asignaciones) != 1 || asignaciones[0].ProyectoSlug != "orquestador" || asignaciones[0].Estado != db.AsignacionActiva {
		t.Fatalf("asignaciones inesperadas: %+v", asignaciones)
	}
}

func TestWebAgenteControlEncolaOrden(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if _, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	}); err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	form := strings.NewReader("accion=arrancar&proyecto=orquestador&conector=codex-cli&modelo=gpt-5.4&motivo=web-agentes")
	req := httptest.NewRequest(http.MethodPost, "/agentes/Codex1/control", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	testMuxAgentesWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "/agentes/Codex1?ok=") {
		t.Fatalf("redirect inesperado: %s", location)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != agenteControlAccionStart {
		t.Fatalf("runtime orders inesperadas: %+v", orders)
	}
}
