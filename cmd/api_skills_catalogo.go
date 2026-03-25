package cmd

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"orquesta/skillsapp"
)

type apiSkillsRemotasResponse struct {
	Items []*skillsapp.SkillRemota `json:"items"`
}

type apiSkillBorrarRequest struct {
	Actor string `json:"actor"`
}

var skillsCatalogoFetcher skillsapp.CatalogoRemotoFetcher = skillsapp.NewSkillsSHCatalogoFetcher()

func apiHandlerSkillsRemotas(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	limit := 24
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	items, err := skillsCatalogoFetcher.Listar(context.Background(), strings.TrimSpace(r.URL.Query().Get("q")), limit)
	if err != nil {
		apiError(w, http.StatusBadGateway, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiSkillsRemotasResponse{Items: items})
}
