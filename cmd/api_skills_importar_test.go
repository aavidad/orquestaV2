package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"orquesta/skillsapp"
)

type fakeSkillFetcherCmd struct {
	meta *skillsapp.RemoteSkill
	err  error
}

func (f fakeSkillFetcherCmd) Fetch(context.Context, skillsapp.SourceSpec) (*skillsapp.RemoteSkill, error) {
	return f.meta, f.err
}

func TestAPISkillImportarDesdeSkillsSh(t *testing.T) {
	prepararDBTemporalCmd(t)

	prev := skillsImportService
	skillsImportService = skillsapp.NewService(skillsapp.Repository{}, fakeSkillFetcherCmd{
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

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/skills/importar", bytes.NewReader([]byte(`{"actor":"Codex1","tipo_agente":"programador","url":"https://skills.sh/openai/skills/openai-docs"}`)))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status importar inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp apiSkillImportarResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode importar: %v", err)
	}
	if resp.ID <= 0 || resp.Skill == nil {
		t.Fatalf("respuesta importar inesperada: %+v", resp)
	}
	if resp.Skill.Origen != "third_party" || resp.Skill.Activa || !resp.Skill.RequiereAprobacion {
		t.Fatalf("skill importada inesperada: %+v", resp.Skill)
	}
}
