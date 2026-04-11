package cmd

import (
	"net/url"
	"strconv"
	"strings"

	"orquesta/microprogramacionapp"
)

func cargarEspecificacionesFuncionDesdeAPI(filtro microprogramacionapp.FiltroEspecificaciones) ([]*microprogramacionapp.EspecificacionFuncion, bool, error) {
	query := url.Values{}
	if filtro.TareaID != nil {
		query.Set("tarea_id", strconv.FormatInt(*filtro.TareaID, 10))
	}
	if filtro.Estado != nil {
		query.Set("estado", string(*filtro.Estado))
	}
	if filtro.Limit > 0 {
		query.Set("limit", strconv.Itoa(filtro.Limit))
	}
	var resp apiEspecificacionesFuncionResponse
	ok, err := apiGetQuery("/api/microprogramacion/especificaciones", query, &resp)
	return resp.Especificaciones, ok, err
}

func cargarEspecificacionFuncionDesdeAPI(id int64) (*microprogramacionapp.EspecificacionFuncion, bool, error) {
	var resp apiEspecificacionFuncionResponse
	ok, err := apiGet("/api/microprogramacion/especificaciones/"+strconv.FormatInt(id, 10), &resp)
	return resp.Especificacion, ok, err
}

func registrarEspecificacionFuncionPorAPI(req apiEspecificacionFuncionCreateRequest) (*apiEspecificacionFuncionCreateResponse, bool, error) {
	var resp apiEspecificacionFuncionCreateResponse
	ok, err := apiPost("/api/microprogramacion/especificaciones", req, &resp)
	return &resp, ok, err
}

func emitirMicrotareaPorAPI(id int64, contexto string) (*apiMicrotareaEmitidaResponse, bool, error) {
	var resp apiMicrotareaEmitidaResponse
	ok, err := apiPost("/api/microprogramacion/especificaciones/"+strconv.FormatInt(id, 10)+"/emitir", apiEmitirMicrotareaRequest{
		Contexto: strings.TrimSpace(contexto),
	}, &resp)
	return &resp, ok, err
}

func despacharMicrotareaPorAPI(id int64, req apiDespacharMicrotareaRequest) (*apiMicrotareaDespachadaResponse, bool, error) {
	var resp apiMicrotareaDespachadaResponse
	ok, err := apiPost("/api/microprogramacion/especificaciones/"+strconv.FormatInt(id, 10)+"/despachar", req, &resp)
	return &resp, ok, err
}

func validarEntregaMicrotareaPorAPI(id int64, req apiValidarEntregaRequest) (*apiValidacionEntregaResponse, bool, error) {
	var resp apiValidacionEntregaResponse
	ok, err := apiPost("/api/microprogramacion/especificaciones/"+strconv.FormatInt(id, 10)+"/validar-entrega", req, &resp)
	return &resp, ok, err
}

func listarEspecificacionesFuncionDesdeAPI(tareaID *int64, proyecto string, estado string, limit int) ([]*microprogramacionapp.EspecificacionFuncion, bool, error) {
	query := url.Values{}
	if tareaID != nil && *tareaID > 0 {
		query.Set("tarea_id", strconv.FormatInt(*tareaID, 10))
	}
	if strings.TrimSpace(proyecto) != "" {
		query.Set("proyecto", strings.TrimSpace(proyecto))
	}
	if strings.TrimSpace(estado) != "" {
		query.Set("estado", strings.TrimSpace(estado))
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	var resp apiEspecificacionesFuncionResponse
	ok, err := apiGetQuery("/api/microprogramacion/especificaciones", query, &resp)
	return resp.Especificaciones, ok, err
}
