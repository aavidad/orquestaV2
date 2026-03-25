package cmd

import (
	"net/http"
	"strings"

	"orquesta/db"
)

type apiSkillDeteccionRequest struct {
	TipoAgente       string `json:"tipo_agente"`
	Nombre           string `json:"nombre"`
	Descripcion      string `json:"descripcion"`
	CuandoUsar       string `json:"cuando_usar"`
	Escenario        string `json:"escenario"`
	AliasesJSON      string `json:"aliases_json"`
	HerramientasJSON string `json:"herramientas_json"`
}

type apiSkillDeteccionResponse struct {
	Resultado *db.ResultadoDeteccionSkill `json:"resultado"`
}

func apiHandlerSkillsDetectarCarencia(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req apiSkillDeteccionRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	resultado, err := db.DetectarCarenciaSkill(&db.SolicitudDeteccionSkill{
		TipoAgente:       strings.TrimSpace(req.TipoAgente),
		Nombre:           strings.TrimSpace(req.Nombre),
		Descripcion:      strings.TrimSpace(req.Descripcion),
		CuandoUsar:       strings.TrimSpace(req.CuandoUsar),
		Escenario:        strings.TrimSpace(req.Escenario),
		AliasesJSON:      strings.TrimSpace(req.AliasesJSON),
		HerramientasJSON: strings.TrimSpace(req.HerramientasJSON),
	})
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiSkillDeteccionResponse{Resultado: resultado})
}
