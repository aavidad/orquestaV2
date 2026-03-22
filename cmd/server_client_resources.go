/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"fmt"
	"net/url"

	"orquesta/coordination"
	"orquesta/db"
)

type apiProyectoResponse struct {
	Proyecto *db.Proyecto `json:"proyecto"`
}

type apiConectorResponse struct {
	Conector *db.Conector `json:"conector"`
}

type apiConfigResponse struct {
	Config map[string]string `json:"config"`
	Clave  string            `json:"clave"`
	Valor  string            `json:"valor"`
}

type apiAsignacionesResponse struct {
	Asignaciones []*db.Asignacion `json:"asignaciones"`
}

type apiLockResponse struct {
	Lock *coordination.Lock `json:"lock"`
}

type apiLocksResponse struct {
	Locks []*coordination.Lock `json:"locks"`
}

type apiWorktreeResponse struct {
	Worktree *coordination.Worktree `json:"worktree"`
}

type apiWorktreesResponse struct {
	Worktrees []*coordination.Worktree `json:"worktrees"`
}

type apiProyectoDescubrirRequest struct {
	Ruta string `json:"ruta"`
}

type apiAgenteRequest struct {
	Nombre string `json:"nombre"`
	Rol    string `json:"rol"`
}

type apiConfigSetRequest struct {
	Clave string `json:"clave"`
	Valor string `json:"valor"`
}

type apiConectorUpsertRequest struct {
	Slug         string `json:"slug"`
	Nombre       string `json:"nombre"`
	Transporte   string `json:"transporte"`
	Comando      string `json:"comando"`
	ArgsJSON     string `json:"args_json"`
	EnvJSON      string `json:"env_json"`
	MetadataJSON string `json:"metadata_json"`
	Activo       bool   `json:"activo"`
}

func cargarConfigDesdeAPI(clave string) (*apiConfigResponse, bool, error) {
	var resp apiConfigResponse
	if clave == "" {
		ok, err := apiGet("/api/config", &resp)
		if !ok || err != nil {
			return nil, ok, err
		}
		return &resp, true, nil
	}
	query := url.Values{}
	query.Set("clave", clave)
	ok, err := apiGetQuery("/api/config", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return &resp, true, nil
}

func configurarValorPorAPI(clave, valor string) (bool, error) {
	ok, err := apiPost("/api/config", apiConfigSetRequest{Clave: clave, Valor: valor}, nil)
	return ok, err
}

func cargarProyectosDesdeAPI() ([]*db.Proyecto, bool, error) {
	var resp apiProyectosResponse
	ok, err := apiGet("/api/proyectos", &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Proyectos, true, nil
}

func descubrirProyectosPorAPI(ruta string) ([]*db.Proyecto, bool, error) {
	var resp apiProyectosResponse
	ok, err := apiPost("/api/proyectos/descubrir", apiProyectoDescubrirRequest{Ruta: ruta}, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Proyectos, true, nil
}

func cargarProyectoDesdeAPI(ref string) (*db.Proyecto, bool, error) {
	var resp apiProyectoResponse
	ok, err := apiGet(fmt.Sprintf("/api/proyectos/%s", url.PathEscape(ref)), &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Proyecto, true, nil
}

func cargarConectoresDesdeAPI() ([]*db.Conector, bool, error) {
	var resp apiConectoresResponse
	ok, err := apiGet("/api/conectores", &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Conectores, true, nil
}

func cargarConectorDesdeAPI(ref string) (*db.Conector, bool, error) {
	var resp apiConectorResponse
	ok, err := apiGet(fmt.Sprintf("/api/conectores/%s", url.PathEscape(ref)), &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Conector, true, nil
}

func registrarConectorPorAPI(req apiConectorUpsertRequest) (int64, bool, error) {
	var resp apiConectorResponse
	ok, err := apiPost("/api/conectores", req, &resp)
	if !ok || err != nil {
		return 0, ok, err
	}
	if resp.Conector == nil {
		return 0, true, fmt.Errorf("respuesta sin conector")
	}
	return resp.Conector.ID, true, nil
}

func registrarAgentePorAPI(nombre, rol string) (bool, error) {
	ok, err := apiPost("/api/agentes", apiAgenteRequest{Nombre: nombre, Rol: rol}, nil)
	return ok, err
}

func retirarAgentePorAPI(nombre string) (bool, error) {
	ok, err := apiPost(fmt.Sprintf("/api/agentes/%s/retirar", url.PathEscape(nombre)), map[string]any{}, nil)
	return ok, err
}

func rehabilitarAgentePorAPI(nombre string) (bool, error) {
	ok, err := apiPost(fmt.Sprintf("/api/agentes/%s/rehabilitar", url.PathEscape(nombre)), map[string]any{}, nil)
	return ok, err
}

func cargarAsignacionesDesdeAPI(query url.Values) ([]*db.Asignacion, bool, error) {
	var resp apiAsignacionesResponse
	ok, err := apiGetQuery("/api/asignaciones", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Asignaciones, true, nil
}

func activarAsignacionPorAPI(req apiAsignacionActivarRequest) (bool, error) {
	ok, err := apiPost("/api/asignaciones/activar", req, nil)
	return ok, err
}

func cargarLocksDesdeAPI(query url.Values) ([]*coordination.Lock, bool, error) {
	var resp apiLocksResponse
	ok, err := apiGetQuery("/api/locks", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Locks, true, nil
}

func tomarLockPorAPI(req apiLockRequest) (*coordination.Lock, bool, error) {
	var resp apiLockResponse
	ok, err := apiPost("/api/locks", req, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Lock, true, nil
}

func renovarLockPorAPI(id int64, req apiLockRequest) (*coordination.Lock, bool, error) {
	var resp apiLockResponse
	ok, err := apiPost(fmt.Sprintf("/api/locks/%d/renovar", id), req, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Lock, true, nil
}

func liberarLockPorAPI(id int64, req apiLockRequest) (*coordination.Lock, bool, error) {
	var resp apiLockResponse
	ok, err := apiPost(fmt.Sprintf("/api/locks/%d/liberar", id), req, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Lock, true, nil
}

func cargarWorktreesDesdeAPI(query url.Values) ([]*coordination.Worktree, bool, error) {
	var resp apiWorktreesResponse
	ok, err := apiGetQuery("/api/worktrees", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Worktrees, true, nil
}

func crearWorktreePorAPI(req apiWorktreeRequest) (*coordination.Worktree, bool, error) {
	var resp apiWorktreeResponse
	ok, err := apiPost("/api/worktrees", req, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Worktree, true, nil
}

func cerrarWorktreePorAPI(id int64, req apiWorktreeRequest) (*coordination.Worktree, bool, error) {
	var resp apiWorktreeResponse
	ok, err := apiPost(fmt.Sprintf("/api/worktrees/%d/cerrar", id), req, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Worktree, true, nil
}
