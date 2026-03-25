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
	"orquesta/skillsapp"
)

type fakeSkillsCatalogoWebLimpio struct {
	items []*skillsapp.SkillRemota
	err   error
}

func (f fakeSkillsCatalogoWebLimpio) Listar(context.Context, string, int) ([]*skillsapp.SkillRemota, error) {
	return f.items, f.err
}

func TestWebGobernanzaActivaBorraYDetectaSkill(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevCatalogo := skillsCatalogoFetcher
	skillsCatalogoFetcher = fakeSkillsCatalogoWebLimpio{
		items: []*skillsapp.SkillRemota{{
			Nombre:      "openai-docs",
			Repo:        "openai/skills",
			Skill:       "openai-docs",
			URLCanonica: "https://skills.sh/openai/skills/openai-docs",
			Origen:      "skills.sh",
		}},
	}
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
		Activa:           false,
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

	formEdit := url.Values{}
	formEdit.Set("tipo_agente", "programador")
	formEdit.Set("nombre", "skill-web-edicion")
	formEdit.Set("descripcion", "descripcion actualizada")
	formEdit.Set("cuando_usar", "uso actualizado")
	formEdit.Set("escenario", "investigacion")
	formEdit.Set("prioridad", "7")
	formEdit.Set("origen", "builtin")
	formEdit.Set("nivel_riesgo", "bajo")
	formEdit.Set("aliases", "uno\ndos")
	formEdit.Set("herramientas", "rg\nfzf")
	reqEdit := httptest.NewRequest(http.MethodPost, "/gobernanza/skills/"+strconv.FormatInt(id, 10)+"/editar", strings.NewReader(formEdit.Encode()))
	reqEdit.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recEdit := httptest.NewRecorder()
	webRouterGobernanza(recEdit, reqEdit)
	if recEdit.Code != http.StatusSeeOther {
		t.Fatalf("editar status=%d body=%s", recEdit.Code, recEdit.Body.String())
	}

	reqView := httptest.NewRequest(http.MethodGet, "/gobernanza?tipo_agente=programador&skill_q=openai", nil)
	recView := httptest.NewRecorder()
	webHandlerGobernanza(recView, reqView)
	body := recView.Body.String()
	for _, token := range []string{"Catálogo remoto de skills.sh", "openai/skills", "Editar skill", "Historial de versiones", "descripcion actualizada", "#2"} {
		if !strings.Contains(body, token) {
			t.Fatalf("vista sin %q:\n%s", token, body)
		}
	}

	formDetect := url.Values{}
	formDetect.Set("tipo_agente", "programador")
	formDetect.Set("nombre", "goimports")
	formDetect.Set("descripcion", "ordenar imports y formatear go")
	formDetect.Set("cuando_usar", "corregir imports en codigo go")
	formDetect.Set("escenario", "codigo")
	formDetect.Set("herramientas", "goimports")
	reqDetect := httptest.NewRequest(http.MethodPost, "/gobernanza/skills/detectar-carencia", strings.NewReader(formDetect.Encode()))
	reqDetect.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recDetect := httptest.NewRecorder()
	webRouterGobernanza(recDetect, reqDetect)
	if recDetect.Code != http.StatusOK {
		t.Fatalf("detectar status=%d body=%s", recDetect.Code, recDetect.Body.String())
	}
	if body := recDetect.Body.String(); !strings.Contains(body, "$skill-creator") {
		t.Fatalf("resultado de deteccion inesperado:\n%s", body)
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
