/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/db"
)

func TestWebRuntimesPageYFiltros(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente 1: %v", err)
	}
	if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar agente 2: %v", err)
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

	// Runtime activo.
	sesionActiva, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-web-001",
		ResumenContinuidad: "runtime activo",
		Branch:             "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion activa: %v", err)
	}
	runtimeActivo, err := db.GetRuntimeBySesionID(sesionActiva.ID)
	if err != nil || runtimeActivo == nil {
		t.Fatalf("get runtime activo: %v", err)
	}

	// Runtime cerrado.
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex2",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-web-002",
		ResumenContinuidad: "runtime cerrado",
		Branch:             "main",
	}); err != nil {
		t.Fatalf("iniciar sesion cerrada: %v", err)
	}
	if err := db.FinSesion("Codex2"); err != nil {
		t.Fatalf("cerrar sesion codex2: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", webHandlerDash)
	mux.HandleFunc("/tareas", webHandlerTareas)
	mux.HandleFunc("/tareas/", webRouterTareas)
	mux.HandleFunc("/propuestas", webHandlerPropuestas)
	mux.HandleFunc("/propuestas/", webRouterPropuestas)
	mux.HandleFunc("/runtimes", webHandlerRuntimes)
	mux.HandleFunc("/runtimes/", webRouterRuntimes)
	registerAPIRoutes(mux)

	assertContains := func(path, needle string) {
		t.Helper()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status inesperado %s: %d body=%s", path, rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), needle) {
			t.Fatalf("respuesta %s no contiene %q: %s", path, needle, rec.Body.String())
		}
	}

	assertContains("/runtimes", "Runtimes")
	assertContains("/runtimes", "Codex1")
	assertContains("/runtimes?activos=true", "Codex1")
	assertContains("/runtimes?activos=false", "Codex2")
	assertContains("/runtimes/"+itoa(runtimeActivo.ID), "Runtime #")
}

func TestWebRuntimesControlEncolaOrden(t *testing.T) {
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

	mux := http.NewServeMux()
	mux.HandleFunc("/", webHandlerDash)
	mux.HandleFunc("/tareas", webHandlerTareas)
	mux.HandleFunc("/tareas/", webRouterTareas)
	mux.HandleFunc("/propuestas", webHandlerPropuestas)
	mux.HandleFunc("/propuestas/", webRouterPropuestas)
	mux.HandleFunc("/runtimes", webHandlerRuntimes)
	mux.HandleFunc("/runtimes/", webRouterRuntimes)
	registerAPIRoutes(mux)

	form := strings.NewReader("agente=Codex1&proyecto=orquestador&accion=arrancar&conector=codex-cli&modelo=gpt-5.4&motivo=web")
	req := httptest.NewRequest(http.MethodPost, "/runtimes/control", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	if location := rec.Header().Get("Location"); !strings.Contains(location, "/runtimes?ok=") {
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

func TestWebRuntimeDetalleMuestraTimelineOperativa(t *testing.T) {
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
		ExternalSessionID:  "sess-web-timeline",
		ResumenContinuidad: "runtime con timeline",
		Branch:             "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	runtimeInst, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtimeInst == nil {
		t.Fatalf("get runtime: runtime=%+v err=%v", runtimeInst, err)
	}
	orderID, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "checkpoint",
		PayloadJSON: `{"motivo":"timeline-web"}`,
	})
	if err != nil {
		t.Fatalf("encolar runtime order: %v", err)
	}
	if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "Codex0",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "handoff_note",
		PayloadJSON: `{"texto":"revisa el handoff"}`,
	}); err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	cpID, err := db.CrearRuntimeCheckpoint(&db.RuntimeCheckpoint{
		Agente:         "Codex1",
		ProyectoID:     &proyectoID,
		SesionID:       &sesion.ID,
		CheckpointKind: "handoff_prepare",
		Resumen:        "checkpoint web",
		Branch:         "feature/timeline",
		CWD:            filepath.Join(tmp, "orquestador"),
		PayloadJSON:    `{"ok":true}`,
		ResumeStrategy: "resumen_y_payload",
		Source:         "web-test",
	})
	if err != nil {
		t.Fatalf("crear checkpoint: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", webHandlerDash)
	mux.HandleFunc("/tareas", webHandlerTareas)
	mux.HandleFunc("/tareas/", webRouterTareas)
	mux.HandleFunc("/propuestas", webHandlerPropuestas)
	mux.HandleFunc("/propuestas/", webRouterPropuestas)
	mux.HandleFunc("/runtimes", webHandlerRuntimes)
	mux.HandleFunc("/runtimes/", webRouterRuntimes)
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/runtimes/"+itoa(runtimeInst.ID), nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, token := range []string{
		"Timeline operativa",
		"runtime_order",
		"#" + itoa(orderID) + " checkpoint",
		"mailbox",
		"handoff_note",
		"checkpoint",
		"#" + itoa(cpID) + " handoff_prepare",
		"web-test",
	} {
		if !strings.Contains(body, token) {
			t.Fatalf("detalle runtime sin %q:\n%s", token, body)
		}
	}
}
