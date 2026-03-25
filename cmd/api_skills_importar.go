package cmd

import (
	"context"
	"net/http"
	"strings"

	"orquesta/db"
	"orquesta/skillsapp"
)

type apiSkillImportarRequest struct {
	Actor      string `json:"actor"`
	TipoAgente string `json:"tipo_agente"`
	URL        string `json:"url"`
	Repo       string `json:"repo"`
	Skill      string `json:"skill"`
}

type apiSkillImportarResponse struct {
	ID             int64     `json:"id"`
	Existente      bool      `json:"existente"`
	FuenteCanonica string    `json:"fuente_canonica"`
	Skill          *db.Skill `json:"skill"`
}

var skillsImportService = skillsapp.NewService(skillsapp.Repository{}, skillsapp.NewGitFetcher())

func apiHandlerSkillsImportar(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req apiSkillImportarRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	result, err := skillsImportService.ImportFromWeb(context.Background(), skillsapp.ImportInput{
		Actor:      strings.TrimSpace(req.Actor),
		TipoAgente: strings.TrimSpace(req.TipoAgente),
		URL:        strings.TrimSpace(req.URL),
		Repo:       strings.TrimSpace(req.Repo),
		Skill:      strings.TrimSpace(req.Skill),
	})
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	status := http.StatusCreated
	if result.Existente {
		status = http.StatusOK
	}
	apiWriteJSON(w, status, apiSkillImportarResponse{
		ID:             result.ID,
		Existente:      result.Existente,
		FuenteCanonica: result.FuenteCanonica,
		Skill:          result.Skill,
	})
}
