package cmd

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"orquesta/db"
)

func testMuxOperacionWeb() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", webHandlerDash)
	mux.HandleFunc("/diagnostico", webHandlerDiagnostico)
	mux.HandleFunc("/auditoria", webHandlerAuditoria)
	mux.HandleFunc("/refineria", webHandlerRefineria)
	mux.HandleFunc("/refineria/", webRouterRefineria)
	mux.HandleFunc("/respaldo", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			webHandlerRespaldoCrear(w, r)
			return
		}
		webHandlerRespaldo(w, r)
	})
	registerAPIRoutes(mux)
	return mux
}

func TestWebAuditoriaListaPorAPI(t *testing.T) {
	prepararDBTemporalCmd(t)
	db.Audit("Codex1", "crear_tarea", "tarea", 7, "detalle web audit")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auditoria?lang=en&agente=Codex1", nil)
	testMuxOperacionWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("auditoria status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, token := range []string{"<html lang=\"en\">", "Audit", "Codex1", "detalle web audit"} {
		if !strings.Contains(body, token) {
			t.Fatalf("panel de auditoria incompleto, falta %q:\n%s", token, body)
		}
	}
}

func TestWebRespaldoCreaCopiaPorAPI(t *testing.T) {
	prepararDBTemporalCmd(t)
	destino := filepath.Join(t.TempDir(), "backups")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/respaldo?lang=en", nil)
	testMuxOperacionWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("respaldo status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Create backup") {
		t.Fatalf("vista respaldo inesperada:\n%s", rec.Body.String())
	}

	form := url.Values{
		"destino":  {destino},
		"etiqueta": {"web"},
		"retener":  {"1"},
	}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/respaldo", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	testMuxOperacionWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("crear respaldo status=%d body=%s", rec.Code, rec.Body.String())
	}

	patron := filepath.Join(destino, "*")
	files, err := filepath.Glob(patron)
	if err != nil {
		t.Fatalf("glob backups: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("backups inesperados: %+v", files)
	}
	if info, err := os.Stat(files[0]); err != nil || info.Size() == 0 {
		t.Fatalf("backup inválido %v %v", files, err)
	}
}

func TestWebDiagnosticoRenderizaSnapshotPorAPI(t *testing.T) {
	prepararDBTemporalCmd(t)
	if err := db.RegistrarAgente("CodexDiag", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	db.Audit("CodexDiag", "accion_diag", "demo", 1, "detalle")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/diagnostico?lang=en&audit_limit=5", nil)
	testMuxOperacionWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("diagnostico status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, token := range []string{"<html lang=\"en\">", "Diagnosis", "CodexDiag", "accion_diag"} {
		if !strings.Contains(body, token) {
			t.Fatalf("panel diagnostico incompleto, falta %q:\n%s", token, body)
		}
	}
}

func TestWebRefineriaListaYDetallePorAPI(t *testing.T) {
	prepararDBTemporalCmd(t)

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "test-refineria",
		Nombre:  "Test Refineria",
		RutaAbs: filepath.Join(t.TempDir(), "test-refineria"),
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Validar calidad",
		Descripcion: "pasar por refineria",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadMedia,
		CreadoPor:   "Codex1",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	solicitud, err := db.SolicitarRefineria(tareaID, "Codex1", "feature/refineria", t.TempDir(), "go test ./cmd")
	if err != nil {
		t.Fatalf("solicitar refineria: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/refineria?lang=en", nil)
	testMuxOperacionWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("refineria status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, token := range []string{"<html lang=\"en\">", "Refinery", "feature/refineria", "go test ./cmd"} {
		if !strings.Contains(body, token) {
			t.Fatalf("panel refineria incompleto, falta %q:\n%s", token, body)
		}
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/refineria/"+strconv.FormatInt(solicitud.ID, 10)+"?lang=en", nil)
	testMuxOperacionWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("detalle refineria status=%d body=%s", rec.Code, rec.Body.String())
	}
	detalle := rec.Body.String()
	for _, token := range []string{"Refinery request", "feature/refineria", "go test ./cmd"} {
		if !strings.Contains(detalle, token) {
			t.Fatalf("detalle refineria incompleto, falta %q:\n%s", token, detalle)
		}
	}
}
