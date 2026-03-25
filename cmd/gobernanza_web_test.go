package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/db"
	"orquesta/skillsapp"
)

type fakeSkillFetcherWeb struct {
	meta *skillsapp.RemoteSkill
	err  error
}

func (f fakeSkillFetcherWeb) Fetch(context.Context, skillsapp.SourceSpec) (*skillsapp.RemoteSkill, error) {
	return f.meta, f.err
}

func TestAPIGobernanzaReglaCrearYListar(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orquesta.db")
	prev := os.Getenv("ORQUESTA_DB")
	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	defer func() {
		_ = os.Setenv("ORQUESTA_DB", prev)
		db.Close()
	}()
	if err := db.Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}

	body := bytes.NewBufferString(`{"tipo_agente":"programador","categoria":"arquitectura","titulo":"Puerto","descripcion":"No acoplar","activa":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/gobernanza/reglas", body)
	rec := httptest.NewRecorder()
	webRouterAPIGobernanza(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status inesperado: %d cuerpo=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/gobernanza/reglas?tipo_agente=programador", nil)
	rec = httptest.NewRecorder()
	webRouterAPIGobernanza(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado listando: %d cuerpo=%s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Items []db.Regla `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(payload.Items) == 0 {
		t.Fatalf("sin reglas en payload")
	}
}

func TestAPIGobernanzaWorkflowCrear(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orquesta.db")
	prev := os.Getenv("ORQUESTA_DB")
	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	defer func() {
		_ = os.Setenv("ORQUESTA_DB", prev)
		db.Close()
	}()
	if err := db.Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}

	body := bytes.NewBufferString(`{"tipo_agente":"programador","nombre":"workflow-api","descripcion":"flujo","pasos":["uno","dos"],"activo":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/gobernanza/workflows", body)
	rec := httptest.NewRecorder()
	webRouterAPIGobernanza(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status inesperado: %d cuerpo=%s", rec.Code, rec.Body.String())
	}

	items, err := gobernanzaService.ListWorkflows("programador", nil)
	if err != nil {
		t.Fatalf("ListWorkflows: %v", err)
	}
	if len(items) == 0 || items[len(items)-1].Nombre != "workflow-api" {
		t.Fatalf("workflow no persistido: %+v", items)
	}
}

func TestWebGobernanzaImportaSkillDesdeFormulario(t *testing.T) {
	prepararDBTemporalCmd(t)

	prev := skillsImportService
	skillsImportService = skillsapp.NewService(skillsapp.Repository{}, fakeSkillFetcherWeb{
		meta: &skillsapp.RemoteSkill{
			Name:         "openai-docs",
			Description:  "Accede a documentacion actual de OpenAI",
			Repo:         "openai/skills",
			SkillRef:     "openai-docs",
			CanonicalURL: "https://skills.sh/openai/skills/openai-docs",
			SourceKind:   "skills.sh",
		},
	})
	defer func() { skillsImportService = prev }()

	form := url.Values{}
	form.Set("tipo_agente", "programador")
	form.Set("url", "https://skills.sh/openai/skills/openai-docs")
	req := httptest.NewRequest(http.MethodPost, "/gobernanza/skills/importar", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	webRouterGobernanza(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "/gobernanza?tipo_agente=programador") {
		t.Fatalf("redirect inesperado: %s", loc)
	}

	skills, err := gobernanzaService.ListSkills("programador", nil)
	if err != nil {
		t.Fatalf("ListSkills: %v", err)
	}
	var imported *db.Skill
	for _, item := range skills {
		if item != nil && item.Nombre == "openai-docs" {
			imported = item
			break
		}
	}
	if imported == nil {
		t.Fatalf("no se encontro la skill importada: %+v", skills)
	}
	if imported.Activa || !imported.RequiereAprobacion {
		t.Fatalf("skill importada inesperada: %+v", imported)
	}

	reqView := httptest.NewRequest(http.MethodGet, "/gobernanza?tipo_agente=programador", nil)
	recView := httptest.NewRecorder()
	webHandlerGobernanza(recView, reqView)
	body := recView.Body.String()
	if !strings.Contains(body, "Importar desde skills.sh") || !strings.Contains(body, "Pendiente de aprobación") {
		t.Fatalf("la web no refleja el bloque de importacion:\n%s", body)
	}
}
