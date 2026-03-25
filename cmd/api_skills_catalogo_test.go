package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"orquesta/db"
	"orquesta/skillsapp"
)

type fakeSkillsCatalogoFetcher struct {
	items []*skillsapp.SkillRemota
	err   error
}

func (f fakeSkillsCatalogoFetcher) Listar(context.Context, string, int) ([]*skillsapp.SkillRemota, error) {
	return f.items, f.err
}

func TestAPISkillsRemotas(t *testing.T) {
	prev := skillsCatalogoFetcher
	skillsCatalogoFetcher = fakeSkillsCatalogoFetcher{
		items: []*skillsapp.SkillRemota{{
			Nombre:      "openai-docs",
			Repo:        "openai/skills",
			Skill:       "openai-docs",
			URLCanonica: "https://skills.sh/openai/skills/openai-docs",
			Origen:      "skills.sh",
		}},
	}
	defer func() { skillsCatalogoFetcher = prev }()

	req := httptest.NewRequest(http.MethodGet, "/api/skills/remotas?q=openai&limit=10", nil)
	rec := httptest.NewRecorder()
	apiHandlerSkillsRemotas(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	var resp apiSkillsRemotasResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].Skill != "openai-docs" {
		t.Fatalf("respuesta inesperada: %+v", resp.Items)
	}
}

func TestAPISkillBorrar(t *testing.T) {
	prepararDBTemporalCmd(t)
	id, err := db.CrearSkill("Codex1", &db.Skill{
		TipoAgente:  "programador",
		Nombre:      "skill-api-borrable",
		Descripcion: "temporal",
		CuandoUsar:  "test",
		Activa:      true,
	})
	if err != nil {
		t.Fatalf("CrearSkill: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/skills/"+strconv.FormatInt(id, 10)+"/borrar", bytes.NewReader([]byte(`{"actor":"Codex1"}`)))
	rec := httptest.NewRecorder()
	apiRouterSkills(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	if _, err := db.GetSkill(id); err == nil {
		t.Fatalf("la skill deberia haber sido borrada")
	}
}
