package cmd

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/db"
)

func TestWebProyectosRespetaIdiomaDelRequest(t *testing.T) {
	prepararDBTemporalCmd(t)
	var proyectoID int64
	if err := db.DB.QueryRow(
		`INSERT INTO proyectos (slug, nombre, ruta_abs, tipo, activo) VALUES (?,?,?,?,1) RETURNING id`,
		"orquestador", "Orquestador", "/tmp/orquestador", "repo",
	).Scan(&proyectoID); err != nil {
		t.Fatalf("insert proyecto: %v", err)
	}
	if _, err := db.GuardarDecisionProyecto(&db.DecisionProyecto{
		ProyectoID: proyectoID,
		Titulo:     "Arquitectura base",
		Categoria:  "general",
		Solucion:   "Puerto de almacenamiento",
	}); err != nil {
		t.Fatalf("GuardarDecisionProyecto: %v", err)
	}
	if _, err := db.GuardarDocumentoExterno(&db.DocumentoExterno{
		ProyectoID:    proyectoID,
		Titulo:        "ADR",
		TipoDocumento: "markdown",
		RutaRef:       "/tmp/adr.md",
		Resumen:       "Resumen",
	}); err != nil {
		t.Fatalf("GuardarDocumentoExterno: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/proyectos?lang=en", nil)
	rec := httptest.NewRecorder()
	webHandlerProyectos(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("proyectos status=%d cuerpo=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Projects") || !strings.Contains(body, "Project memory and traceability view.") {
		t.Fatalf("listado de proyectos sin i18n: %s", body)
	}
	if !strings.Contains(body, `<html lang="en">`) {
		t.Fatalf("lang html inesperado: %s", body)
	}
	if got := rec.Header().Get("Content-Language"); got != "en" {
		t.Fatalf("Content-Language=%q", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/proyectos/orquestador?lang=en", nil)
	rec = httptest.NewRecorder()
	webHandlerProyectoDetalle(rec, req, "orquestador")
	if rec.Code != http.StatusOK {
		t.Fatalf("proyecto detalle status=%d cuerpo=%s", rec.Code, rec.Body.String())
	}
	body = rec.Body.String()
	if !strings.Contains(body, "Back to projects") || !strings.Contains(body, "External documentation") || !strings.Contains(body, "Voting history") {
		t.Fatalf("detalle de proyectos sin i18n: %s", body)
	}
}

func TestWebProyectoFabricarAppGeneraBacklog(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

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

	form := strings.NewReader("tipo=web_api&nombre=Orquestador&descripcion=Panel+de+control&frontend=1&api=1&docker=1&i18n=1&idiomas=es,en&por=web")
	req := httptest.NewRequest(http.MethodPost, "/proyectos/orquestador/fabricar-app", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	webRouterProyectos(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	tareas, err := db.ListarTareas(db.FiltroTareas{ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar tareas: %v", err)
	}
	if len(tareas) < 4 {
		t.Fatalf("se esperaban tareas generadas, got=%d", len(tareas))
	}
	backlog := 0
	for _, tarea := range tareas {
		if tarea != nil && tarea.Estado == db.EstadoBacklog {
			backlog++
		}
	}
	if backlog == 0 {
		t.Fatalf("se esperaba backlog dependiente, tareas=%+v", tareas)
	}
}
