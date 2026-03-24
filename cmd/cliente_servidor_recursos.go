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
	"strings"

	"orquesta/coordinacion"
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
	Lock *coordinacion.Lock `json:"lock"`
}

type apiLocksResponse struct {
	Locks []*coordinacion.Lock `json:"locks"`
}

type apiWorktreeResponse struct {
	Worktree *coordinacion.Worktree `json:"worktree"`
}

type apiWorktreesResponse struct {
	Worktrees []*coordinacion.Worktree `json:"worktrees"`
}

type apiAgenteHandoffResponse struct {
	ID      int64 `json:"id"`
	OrderID int64 `json:"order_id"`
}

type apiRuntimeMailboxCreateResponse struct {
	OK bool  `json:"ok"`
	ID int64 `json:"id"`
}

type apiProyectoDescubrirRequest struct {
	Ruta string `json:"ruta"`
}

type apiLenguajePoliticaResponse struct {
	Politica *db.LanguagePolicy `json:"politica"`
}

type apiLenguajeMatrizResponse struct {
	Matriz []*db.LanguageMatrixEntry `json:"matriz"`
}

type apiLenguajeResolucionResponse struct {
	Resolucion *db.LanguageResolution `json:"resolucion"`
}

type apiLenguajePoliticaSetRequest struct {
	Politica *db.LanguagePolicy `json:"politica"`
	Por      string             `json:"por"`
}

type apiLenguajeMatrizSetRequest struct {
	Scope    string `json:"scope"`
	Selector string `json:"selector"`
	Contexto string `json:"contexto"`
	Idioma   string `json:"idioma"`
	Razon    string `json:"razon"`
	Por      string `json:"por"`
}

type apiLenguajeMatrizDeleteRequest struct {
	Scope    string `json:"scope"`
	Selector string `json:"selector"`
	Contexto string `json:"contexto"`
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

func cargarPoliticaLenguajeDesdeAPI() (*db.LanguagePolicy, bool, error) {
	var resp apiLenguajePoliticaResponse
	ok, err := apiGet("/api/lenguaje/politica", &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Politica, true, nil
}

func cargarMatrizLenguajeDesdeAPI() ([]*db.LanguageMatrixEntry, bool, error) {
	var resp apiLenguajeMatrizResponse
	ok, err := apiGet("/api/lenguaje/matriz", &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Matriz, true, nil
}

func resolverLenguajePorAPI(proyecto string, tareaID *int64, contexto string) (*db.LanguageResolution, bool, error) {
	query := url.Values{}
	if strings.TrimSpace(proyecto) != "" {
		query.Set("proyecto", strings.TrimSpace(proyecto))
	}
	if tareaID != nil && *tareaID > 0 {
		query.Set("tarea", fmt.Sprintf("%d", *tareaID))
	}
	if strings.TrimSpace(contexto) != "" {
		query.Set("contexto", strings.TrimSpace(contexto))
	}
	var resp apiLenguajeResolucionResponse
	ok, err := apiGetQuery("/api/lenguaje/resolver", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Resolucion, true, nil
}

func fijarPoliticaLenguajePorAPI(policy *db.LanguagePolicy, por string) (bool, error) {
	ok, err := apiPost("/api/lenguaje/politica", apiLenguajePoliticaSetRequest{
		Politica: policy,
		Por:      strings.TrimSpace(por),
	}, nil)
	return ok, err
}

func fijarEntradaMatrizLenguajePorAPI(scope, selector, contexto, idioma, razon, por string) (bool, error) {
	ok, err := apiPost("/api/lenguaje/matriz", apiLenguajeMatrizSetRequest{
		Scope:    strings.TrimSpace(scope),
		Selector: strings.TrimSpace(selector),
		Contexto: strings.TrimSpace(contexto),
		Idioma:   strings.TrimSpace(idioma),
		Razon:    strings.TrimSpace(razon),
		Por:      strings.TrimSpace(por),
	}, nil)
	return ok, err
}

func borrarEntradaMatrizLenguajePorAPI(scope, selector, contexto string) (bool, error) {
	ok, err := apiPost("/api/lenguaje/matriz/borrar", apiLenguajeMatrizDeleteRequest{
		Scope:    strings.TrimSpace(scope),
		Selector: strings.TrimSpace(selector),
		Contexto: strings.TrimSpace(contexto),
	}, nil)
	return ok, err
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

func eliminarAgentePorAPI(nombre string) (bool, error) {
	ok, err := apiPost(fmt.Sprintf("/api/agentes/%s/eliminar", url.PathEscape(nombre)), map[string]any{}, nil)
	return ok, err
}

func resetReanimacionAgentePorAPI(nombre string) (bool, error) {
	ok, err := apiPost(fmt.Sprintf("/api/agentes/%s/reset-reanimacion", url.PathEscape(nombre)), map[string]any{}, nil)
	return ok, err
}

func pausarAgentePorAPI(nombre string, minutos int, motivo string) (bool, error) {
	return registrarPausaAgentePorAPI(nombre, minutos, motivo, "", "", "")
}

func registrarPausaAgentePorAPI(nombre string, minutos int, motivo, accion, entidad, detalle string) (bool, error) {
	ok, err := apiPost("/api/agente/pausar", apiAgentePausarRequest{
		Agente:  nombre,
		Minutos: minutos,
		Motivo:  motivo,
		Accion:  accion,
		Entidad: entidad,
		Detalle: detalle,
	}, nil)
	return ok, err
}

func crearHandoffAgentePorAPI(origen, destino string, tareaID *int64, motivo, resumen, externalSessionID string) (int64, bool, error) {
	req := apiRuntimeHandoffRequest{
		AgenteOrigen:      origen,
		AgenteDestino:     destino,
		Motivo:            motivo,
		Resumen:           resumen,
		ExternalSessionID: externalSessionID,
	}
	if tareaID != nil {
		req.TareaID = *tareaID
	}
	var resp apiAgenteHandoffResponse
	ok, err := apiPost("/api/agente/handoff", req, &resp)
	if !ok || err != nil {
		return 0, ok, err
	}
	if resp.OrderID == 0 {
		resp.OrderID = resp.ID
	}
	return resp.OrderID, true, nil
}

func cargarRuntimeMailboxDesdeAPI(query url.Values) ([]*db.RuntimeMailboxMessage, bool, error) {
	var resp apiRuntimeMailboxResponse
	ok, err := apiGetQuery("/api/runtime-mailbox", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Mailbox, true, nil
}

func crearRuntimeMailboxDesdeAPI(fromAgente, toAgente, proyecto string, runtimeOrderID int64, kind, payload string) (int64, bool, error) {
	var resp apiRuntimeMailboxCreateResponse
	ok, err := apiPost("/api/runtime-mailbox", apiRuntimeMailboxCreateRequest{
		FromAgente:     fromAgente,
		ToAgente:       toAgente,
		Proyecto:       proyecto,
		RuntimeOrderID: runtimeOrderID,
		Kind:           kind,
		Payload:        payload,
	}, &resp)
	if !ok || err != nil {
		return 0, ok, err
	}
	return resp.ID, true, nil
}

func crearRuntimeCheckpointDesdeAPI(agente, proyecto string, sesionID, runtimeID int64, checkpointKind, resumen, branch, cwd, payload, resumeStrategy, source string) (int64, bool, error) {
	var resp apiRuntimeCheckpointCreateResponse
	ok, err := apiPost("/api/runtime-checkpoints", apiRuntimeCheckpointCreateRequest{
		Agente:         agente,
		Proyecto:       proyecto,
		SesionID:       sesionID,
		RuntimeID:      runtimeID,
		CheckpointKind: checkpointKind,
		Resumen:        resumen,
		Branch:         branch,
		CWD:            cwd,
		Payload:        payload,
		ResumeStrategy: resumeStrategy,
		Source:         source,
	}, &resp)
	if !ok || err != nil {
		return 0, ok, err
	}
	return resp.ID, true, nil
}

func marcarRuntimeMailboxEntregadoPorAPI(id int64) (bool, error) {
	ok, err := apiPost(fmt.Sprintf("/api/runtime-mailbox/%d/entregar", id), map[string]any{}, nil)
	return ok, err
}

func marcarRuntimeMailboxConsumidoPorAPI(id int64) (bool, error) {
	ok, err := apiPost(fmt.Sprintf("/api/runtime-mailbox/%d/consumir", id), map[string]any{}, nil)
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

func cargarLocksDesdeAPI(query url.Values) ([]*coordinacion.Lock, bool, error) {
	var resp apiLocksResponse
	ok, err := apiGetQuery("/api/locks", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Locks, true, nil
}

func tomarLockPorAPI(req apiLockRequest) (*coordinacion.Lock, bool, error) {
	var resp apiLockResponse
	ok, err := apiPost("/api/locks", req, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Lock, true, nil
}

func renovarLockPorAPI(id int64, req apiLockRequest) (*coordinacion.Lock, bool, error) {
	var resp apiLockResponse
	ok, err := apiPost(fmt.Sprintf("/api/locks/%d/renovar", id), req, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Lock, true, nil
}

func liberarLockPorAPI(id int64, req apiLockRequest) (*coordinacion.Lock, bool, error) {
	var resp apiLockResponse
	ok, err := apiPost(fmt.Sprintf("/api/locks/%d/liberar", id), req, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Lock, true, nil
}

func cargarWorktreesDesdeAPI(query url.Values) ([]*coordinacion.Worktree, bool, error) {
	var resp apiWorktreesResponse
	ok, err := apiGetQuery("/api/worktrees", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Worktrees, true, nil
}

func crearWorktreePorAPI(req apiWorktreeRequest) (*coordinacion.Worktree, bool, error) {
	var resp apiWorktreeResponse
	ok, err := apiPost("/api/worktrees", req, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Worktree, true, nil
}

func cerrarWorktreePorAPI(id int64, req apiWorktreeRequest) (*coordinacion.Worktree, bool, error) {
	var resp apiWorktreeResponse
	ok, err := apiPost(fmt.Sprintf("/api/worktrees/%d/cerrar", id), req, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Worktree, true, nil
}
