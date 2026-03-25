/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"orquesta/db"
	"orquesta/gobernanzaapp"
	"orquesta/skillsapp"
)

type fakeSkillFetcherWeb struct {
	meta *skillsapp.RemoteSkill
	err  error
}

func (f fakeSkillFetcherWeb) Fetch(context.Context, skillsapp.SourceSpec) (*skillsapp.RemoteSkill, error) {
	return f.meta, f.err
}

type fakeSkillsCatalogoWeb struct {
	items []*skillsapp.SkillRemota
	err   error
}

func (f fakeSkillsCatalogoWeb) Listar(context.Context, string, int) ([]*skillsapp.SkillRemota, error) {
	return f.items, f.err
}

func newGobernanzaServiceTest() *gobernanzaapp.Service {
	return gobernanzaapp.NewService(db.GovernanceRepository{})
}

func TestWebGobernanzaRespetaIdiomaDelRequest(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevCatalogo := skillsCatalogoFetcher
	skillsCatalogoFetcher = fakeSkillsCatalogoWeb{
		items: []*skillsapp.SkillRemota{{
			Nombre:      "openai-docs",
			Repo:        "openai/skills",
			Skill:       "openai-docs",
			URLCanonica: "https://skills.sh/openai/skills/openai-docs",
			Origen:      "skills.sh",
		}},
	}
	defer func() { skillsCatalogoFetcher = prevCatalogo }()
	if _, err := newGobernanzaServiceTest().SaveRule(gobernanzaapp.SaveRuleInput{
		TipoAgente:  "programador",
		Categoria:   "arquitectura",
		Titulo:      "Puerto",
		Descripcion: "No acoplar",
		Activa:      true,
	}); err != nil {
		t.Fatalf("CreateRule: %v", err)
	}
	if _, err := newGobernanzaServiceTest().SaveSkill(gobernanzaapp.SaveSkillInput{
		TipoAgente:  "programador",
		Nombre:      "rg",
		Descripcion: "busqueda rapida",
		CuandoUsar:  "buscar texto",
		Activa:      true,
	}); err != nil {
		t.Fatalf("CreateSkill: %v", err)
	}
	if _, err := newGobernanzaServiceTest().SaveWorkflow(gobernanzaapp.SaveWorkflowInput{
		TipoAgente:  "programador",
		Nombre:      "inicio",
		Descripcion: "flujo base",
		Pasos:       []string{"leer", "votar"},
		Activo:      true,
	}); err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/gobernanza?tipo_agente=programador&lang=en", nil)
	rec := httptest.NewRecorder()
	webHandlerGobernanza(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("gobernanza status=%d cuerpo=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Operational governance") || !strings.Contains(body, "Rules") || !strings.Contains(body, "Workflows") {
		t.Fatalf("gobernanza sin i18n: %s", body)
	}
	if !strings.Contains(body, "Import from skills.sh") {
		t.Fatalf("falta bloque de importacion de skills: %s", body)
	}
	if !strings.Contains(body, "Remote skills.sh catalog") || !strings.Contains(body, "openai/skills") {
		t.Fatalf("falta catalogo remoto de skills: %s", body)
	}
	if !strings.Contains(body, `<html lang="en">`) {
		t.Fatalf("lang html inesperado: %s", body)
	}
	if got := rec.Header().Get("Content-Language"); got != "en" {
		t.Fatalf("Content-Language=%q", got)
	}
}

func TestWebGobernanzaImportaSkillDesdeFormulario(t *testing.T) {
	prepararDBTemporalCmd(t)

	prev := skillsImportService
	prevCatalogo := skillsCatalogoFetcher
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
	skillsCatalogoFetcher = fakeSkillsCatalogoWeb{
		items: []*skillsapp.SkillRemota{{
			Nombre:      "openai-docs",
			Repo:        "openai/skills",
			Skill:       "openai-docs",
			URLCanonica: "https://skills.sh/openai/skills/openai-docs",
			Origen:      "skills.sh",
		}},
	}
	defer func() {
		skillsImportService = prev
		skillsCatalogoFetcher = prevCatalogo
	}()

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

	skills, err := newGobernanzaServiceTest().ListSkills("programador", nil)
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

	reqView := httptest.NewRequest(http.MethodGet, "/gobernanza?tipo_agente=programador&lang=en", nil)
	recView := httptest.NewRecorder()
	webHandlerGobernanza(recView, reqView)
	body := recView.Body.String()
	if !strings.Contains(body, "Pending approval") || !strings.Contains(body, "Source: third-party") {
		t.Fatalf("la web no refleja el estado de la skill importada:\n%s", body)
	}
}

func TestWebGobernanzaCreaSkillConContratoCompleto(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevCatalogo := skillsCatalogoFetcher
	skillsCatalogoFetcher = fakeSkillsCatalogoWeb{}
	defer func() { skillsCatalogoFetcher = prevCatalogo }()

	form := url.Values{}
	form.Set("tipo_agente", "programador")
	form.Set("nombre", "goimports")
	form.Set("descripcion", "ordena imports y formatea go")
	form.Set("cuando_usar", "corregir imports go")
	form.Set("escenario", "codigo")
	form.Set("prioridad", "25")
	form.Set("origen", "builtin")
	form.Set("nivel_riesgo", "bajo")
	form.Set("aliases", "fmt\nimports")
	form.Set("herramientas", "goimports\ngofmt")
	form.Set("requiere_aprobacion", "on")
	req := httptest.NewRequest(http.MethodPost, "/gobernanza/skills", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	webRouterGobernanza(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("crear status=%d body=%s", rec.Code, rec.Body.String())
	}

	skills, err := newGobernanzaServiceTest().ListSkills("programador", nil)
	if err != nil {
		t.Fatalf("ListSkills: %v", err)
	}
	var created *db.Skill
	for _, item := range skills {
		if item != nil && item.Nombre == "goimports" {
			created = item
			break
		}
	}
	if created == nil {
		t.Fatalf("no se encontro la skill creada: %+v", skills)
	}
	if created.Escenario != "codigo" || created.Prioridad != 25 {
		t.Fatalf("faltan campos del contrato: %+v", created)
	}
	if created.Origen != "builtin" || created.NivelRiesgo != "bajo" || created.RequiereAprobacion || !created.Activa {
		t.Fatalf("flags inesperadas: %+v", created)
	}
	if created.AliasesJSON != `["fmt","imports"]` || created.HerramientasJSON != `["gofmt","goimports"]` {
		t.Fatalf("listas inesperadas: %+v", created)
	}

	reqView := httptest.NewRequest(http.MethodGet, "/gobernanza?tipo_agente=programador", nil)
	recView := httptest.NewRecorder()
	webHandlerGobernanza(recView, reqView)
	body := recView.Body.String()
	for _, token := range []string{"goimports", "codigo", "builtin", "bajo"} {
		if !strings.Contains(body, token) {
			t.Fatalf("vista sin %q:\n%s", token, body)
		}
	}
}

func TestWebGobernanzaActivaYBorraSkill(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevCatalogo := skillsCatalogoFetcher
	skillsCatalogoFetcher = fakeSkillsCatalogoWeb{}
	defer func() { skillsCatalogoFetcher = prevCatalogo }()

	id, err := db.CrearSkill("Codex1", &db.Skill{
		TipoAgente:  "programador",
		Nombre:      "skill-web-externa",
		Descripcion: "temporal",
		CuandoUsar:  "test",
		Origen:      "third_party",
		Activa:      false,
	})
	if err != nil {
		t.Fatalf("CrearSkill: %v", err)
	}

	form := url.Values{}
	form.Set("tipo_agente", "programador")
	reqAct := httptest.NewRequest(http.MethodPost, "/gobernanza/skills/"+strconv.FormatInt(id, 10)+"/activar", strings.NewReader(form.Encode()))
	reqAct.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recAct := httptest.NewRecorder()
	webRouterGobernanza(recAct, reqAct)
	if recAct.Code != http.StatusSeeOther {
		t.Fatalf("activar status=%d body=%s", recAct.Code, recAct.Body.String())
	}
	skillActiva, err := db.GetSkill(id)
	if err != nil {
		t.Fatalf("GetSkill: %v", err)
	}
	if !skillActiva.Activa || skillActiva.RequiereAprobacion {
		t.Fatalf("skill activa inesperada: %+v", skillActiva)
	}

	reqDel := httptest.NewRequest(http.MethodPost, "/gobernanza/skills/"+strconv.FormatInt(id, 10)+"/borrar", strings.NewReader(form.Encode()))
	reqDel.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recDel := httptest.NewRecorder()
	webRouterGobernanza(recDel, reqDel)
	if recDel.Code != http.StatusSeeOther {
		t.Fatalf("borrar status=%d body=%s", recDel.Code, recDel.Body.String())
	}
	if _, err := db.GetSkill(id); err == nil {
		t.Fatalf("la skill deberia haber sido borrada")
	}
}

func TestWebGobernanzaEditaSkillYMuestraVersiones(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevCatalogo := skillsCatalogoFetcher
	skillsCatalogoFetcher = fakeSkillsCatalogoWeb{}
	defer func() { skillsCatalogoFetcher = prevCatalogo }()

	id, err := db.CrearSkill("Codex1", &db.Skill{
		TipoAgente:       "programador",
		Nombre:           "skill-web-edicion",
		Descripcion:      "temporal",
		CuandoUsar:       "test",
		Escenario:        "codigo",
		Prioridad:        50,
		AliasesJSON:      `["uno"]`,
		HerramientasJSON: `["rg"]`,
		Activa:           true,
	})
	if err != nil {
		t.Fatalf("CrearSkill: %v", err)
	}

	form := url.Values{}
	form.Set("tipo_agente", "programador")
	form.Set("nombre", "skill-web-edicion")
	form.Set("descripcion", "descripcion actualizada")
	form.Set("cuando_usar", "uso actualizado")
	form.Set("escenario", "investigacion")
	form.Set("prioridad", "7")
	form.Set("origen", "builtin")
	form.Set("nivel_riesgo", "bajo")
	form.Set("aliases", "uno\ndos")
	form.Set("herramientas", "rg\nfzf")
	req := httptest.NewRequest(http.MethodPost, "/gobernanza/skills/"+strconv.FormatInt(id, 10)+"/editar", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	webRouterGobernanza(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("editar status=%d body=%s", rec.Code, rec.Body.String())
	}

	skill, err := db.GetSkill(id)
	if err != nil {
		t.Fatalf("GetSkill: %v", err)
	}
	if skill.Descripcion != "descripcion actualizada" || skill.Prioridad != 7 || skill.Escenario != "investigacion" {
		t.Fatalf("skill editada inesperada: %+v", skill)
	}

	reqView := httptest.NewRequest(http.MethodGet, "/gobernanza?tipo_agente=programador", nil)
	recView := httptest.NewRecorder()
	webHandlerGobernanza(recView, reqView)
	body := recView.Body.String()
	for _, token := range []string{"Editar habilidad", "Historial de versiones", "descripcion actualizada", "#2"} {
		if !strings.Contains(body, token) {
			t.Fatalf("vista sin %q:\n%s", token, body)
		}
	}
}

func TestWebGobernanzaDetectaCarenciaSkill(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevCatalogo := skillsCatalogoFetcher
	skillsCatalogoFetcher = fakeSkillsCatalogoWeb{}
	defer func() { skillsCatalogoFetcher = prevCatalogo }()

	form := url.Values{}
	form.Set("tipo_agente", "programador")
	form.Set("nombre", "goimports")
	form.Set("descripcion", "ordenar imports y formatear go")
	form.Set("cuando_usar", "corregir imports en codigo go")
	form.Set("escenario", "codigo")
	form.Set("herramientas", "goimports")
	req := httptest.NewRequest(http.MethodPost, "/gobernanza/skills/detectar-carencia", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	webRouterGobernanza(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("detectar status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, token := range []string{"Detectar carencia de habilidad", "$skill-creator", "goimports"} {
		if !strings.Contains(body, token) {
			t.Fatalf("resultado sin %q:\n%s", token, body)
		}
	}
}
