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
	"strconv"
	"strings"

	"orquesta/capacidadapp"
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

func cargarPoolsDesdeAPI(activo *bool) ([]*db.PoolCapacidadResumen, bool, error) {
	var resp apiPoolsResponse
	query := url.Values{}
	if activo != nil {
		query.Set("activo", strconv.FormatBool(*activo))
	}
	ok, err := apiGetQuery("/api/pools", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Pools, true, nil
}

func cargarPoolDetalleDesdeAPI(slug string) (*capacidadapp.PoolDetail, bool, error) {
	var resp apiPoolResponse
	ok, err := apiGet(fmt.Sprintf("/api/pools/%s", url.PathEscape(strings.TrimSpace(slug))), &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Detalle, true, nil
}

func guardarPoolPorAPI(pool *db.PoolCapacidad) (int64, bool, error) {
	if pool == nil {
		return 0, false, fmt.Errorf("pool obligatorio")
	}
	var resp apiPoolSaveResponse
	ok, err := apiPost("/api/pools", apiPoolSaveRequest{
		Slug:                strings.TrimSpace(pool.Slug),
		Proveedor:           strings.TrimSpace(pool.Proveedor),
		Runtime:             strings.TrimSpace(pool.Runtime),
		Plan:                strings.TrimSpace(pool.Plan),
		EsDePago:            pool.EsDePago,
		CapacidadTotal:      pool.CapacidadTotal,
		CapacidadReservada:  pool.CapacidadReservada,
		PermiteHijos:        pool.PermiteHijos,
		PermiteModelosMulti: pool.PermiteModelosMulti,
		PermiteSobrecoste:   pool.PermiteSobrecoste,
		PoliticaHandoff:     strings.TrimSpace(pool.PoliticaHandoff),
		FuenteTelemetria:    strings.TrimSpace(pool.FuenteTelemetria),
		MetadataJSON:        strings.TrimSpace(pool.MetadataJSON),
		Activo:              pool.Activo,
	}, &resp)
	if !ok || err != nil {
		return 0, ok, err
	}
	return resp.ID, true, nil
}

func seedInicialPoolsPorAPI() (bool, error) {
	return apiPost("/api/pools/seed-inicial", map[string]any{}, nil)
}

func cargarModelosPoolDesdeAPI(slug string) ([]*db.PoolModelo, bool, error) {
	var resp apiPoolModelosResponse
	ok, err := apiGet(fmt.Sprintf("/api/pools/%s/modelos", url.PathEscape(strings.TrimSpace(slug))), &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Modelos, true, nil
}

func guardarModeloPoolPorAPI(poolSlug string, modelo *db.PoolModelo) (int64, bool, error) {
	if modelo == nil {
		return 0, false, fmt.Errorf("modelo obligatorio")
	}
	var resp apiPoolSaveResponse
	ok, err := apiPost(fmt.Sprintf("/api/pools/%s/modelos", url.PathEscape(strings.TrimSpace(poolSlug))), apiPoolModeloSaveRequest{
		ModelSlug:          strings.TrimSpace(modelo.ModelSlug),
		Activo:             modelo.Activo,
		Prioridad:          modelo.Prioridad,
		CosteRelativo:      modelo.CosteRelativo,
		LimiteConocidoJSON: strings.TrimSpace(modelo.LimiteConocidoJSON),
	}, &resp)
	if !ok || err != nil {
		return 0, ok, err
	}
	return resp.ID, true, nil
}

func seedInicialModelosPoolPorAPI() (bool, error) {
	return apiPost("/api/pools/modelos/seed-inicial", map[string]any{}, nil)
}

func cargarPoliticasModeloDesdeAPI(scopeTipo, scopeRef string, activa *bool) ([]*db.PoliticaModelo, bool, error) {
	var resp apiPoliticasModeloResponse
	query := url.Values{}
	if strings.TrimSpace(scopeTipo) != "" {
		query.Set("scope_tipo", strings.TrimSpace(scopeTipo))
	}
	if strings.TrimSpace(scopeRef) != "" {
		query.Set("scope_ref", strings.TrimSpace(scopeRef))
	}
	if activa != nil {
		query.Set("activa", strconv.FormatBool(*activa))
	}
	ok, err := apiGetQuery("/api/politicas-modelo", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Politicas, true, nil
}

func guardarPoliticaModeloPorAPI(policy *db.PoliticaModelo) (int64, bool, error) {
	if policy == nil {
		return 0, false, fmt.Errorf("politica obligatoria")
	}
	var resp apiPoliticaModeloSaveResponse
	ok, err := apiPost("/api/politicas-modelo", apiPoliticaModeloSaveRequest{
		ScopeTipo:       strings.TrimSpace(policy.ScopeTipo),
		ScopeRef:        strings.TrimSpace(policy.ScopeRef),
		PerfilTarea:     strings.TrimSpace(policy.PerfilTarea),
		PoolSlug:        strings.TrimSpace(policy.PoolSlug),
		ModelSlug:       strings.TrimSpace(policy.ModelSlug),
		ReasoningEffort: strings.TrimSpace(policy.ReasoningEffort),
		Prioridad:       policy.Prioridad,
		Activa:          policy.Activa,
		MetadataJSON:    strings.TrimSpace(policy.MetadataJSON),
	}, &resp)
	if !ok || err != nil {
		return 0, ok, err
	}
	return resp.ID, true, nil
}

func seedInicialPoliticasModeloPorAPI() (bool, error) {
	return apiPost("/api/politicas-modelo/seed-inicial", map[string]any{}, nil)
}

func resolverModeloPorAPI(input db.ResolverPoliticaInput) (*db.ResolucionModelo, bool, error) {
	var resp apiResolucionModeloResponse
	query := url.Values{}
	if input.TareaID != nil && *input.TareaID > 0 {
		query.Set("tarea_id", strconv.FormatInt(*input.TareaID, 10))
	}
	if strings.TrimSpace(input.ProyectoSlug) != "" {
		query.Set("proyecto", strings.TrimSpace(input.ProyectoSlug))
	}
	if strings.TrimSpace(input.Fase) != "" {
		query.Set("fase", strings.TrimSpace(input.Fase))
	}
	if strings.TrimSpace(input.PerfilTarea) != "" {
		query.Set("perfil", strings.TrimSpace(input.PerfilTarea))
	}
	ok, err := apiGetQuery("/api/modelo/resolver", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Resolucion, true, nil
}

func cargarResumenProgresoDesdeAPI(proyecto string) (*db.ResumenProgresoProyecto, bool, error) {
	var resp apiProgresoResumenResponse
	query := url.Values{}
	query.Set("proyecto", strings.TrimSpace(proyecto))
	ok, err := apiGetQuery("/api/progreso", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Resumen, true, nil
}

func cargarFasesProgresoDesdeAPI(proyecto string) ([]*db.FaseProyecto, bool, error) {
	var resp apiProgresoFasesResponse
	query := url.Values{}
	query.Set("proyecto", strings.TrimSpace(proyecto))
	ok, err := apiGetQuery("/api/progreso/fases", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Fases, true, nil
}

func registrarFaseProgresoPorAPI(proyecto string, req apiProgresoFaseRegistrarRequest) (*apiProgresoFaseResponse, bool, error) {
	req.Proyecto = strings.TrimSpace(proyecto)
	var resp apiProgresoFaseResponse
	ok, err := apiPost("/api/progreso/fases", req, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return &resp, true, nil
}

func actualizarFaseProgresoPorAPI(id int64, req apiProgresoFaseActualizarRequest) (*apiProgresoFaseResponse, bool, error) {
	var resp apiProgresoFaseResponse
	ok, err := apiPost(fmt.Sprintf("/api/progreso/fases/%d", id), req, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return &resp, true, nil
}

func registrarAvanceProgresoPorAPI(tareaID int64, req apiProgresoTareaRegistrarRequest) (bool, error) {
	ok, err := apiPost(fmt.Sprintf("/api/progreso/tareas/%d", tareaID), req, nil)
	return ok, err
}

func cargarPresupuestoSesionDesdeAPI(sesionID int64, agente string) (*apiSesionPresupuestoResponse, bool, error) {
	var resp apiSesionPresupuestoResponse
	query := url.Values{}
	if sesionID > 0 {
		query.Set("sesion", strconv.FormatInt(sesionID, 10))
	}
	if strings.TrimSpace(agente) != "" {
		query.Set("agente", strings.TrimSpace(agente))
	}
	ok, err := apiGetQuery("/api/sesiones/presupuesto", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return &resp, true, nil
}

func registrarPresupuestoSesionPorAPI(req apiSesionPresupuestoRequest) (*apiSesionPresupuestoResponse, bool, error) {
	var resp apiSesionPresupuestoResponse
	ok, err := apiPost("/api/sesiones/presupuesto", req, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return &resp, true, nil
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

func cargarReglasDesdeAPI(rol string, activa *bool) ([]*db.Regla, bool, error) {
	var resp apiReglasResponse
	query := url.Values{}
	if strings.TrimSpace(rol) != "" {
		query.Set("rol", strings.TrimSpace(rol))
	}
	if activa != nil {
		query.Set("activa", fmt.Sprintf("%t", *activa))
	}
	ok, err := apiGetQuery("/api/reglas", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Reglas, true, nil
}

func cargarSkillsDesdeAPI(rol string, activa *bool) ([]*db.Skill, bool, error) {
	var resp apiSkillsResponse
	query := url.Values{}
	if strings.TrimSpace(rol) != "" {
		query.Set("rol", strings.TrimSpace(rol))
	}
	if activa != nil {
		query.Set("activa", fmt.Sprintf("%t", *activa))
	}
	ok, err := apiGetQuery("/api/skills", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Skills, true, nil
}

func cargarWorkflowsDesdeAPI(rol string, activo *bool) ([]*db.Workflow, bool, error) {
	var resp apiWorkflowsResponse
	query := url.Values{}
	if strings.TrimSpace(rol) != "" {
		query.Set("rol", strings.TrimSpace(rol))
	}
	if activo != nil {
		query.Set("activa", fmt.Sprintf("%t", *activo))
	}
	ok, err := apiGetQuery("/api/workflows", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Workflows, true, nil
}

func cargarPermisosCatalogoDesdeAPI(entidad string) ([]*db.PermisoEdicionCatalogo, bool, error) {
	var resp apiPermisosCatalogoResponse
	query := url.Values{}
	if strings.TrimSpace(entidad) != "" {
		query.Set("entidad", strings.TrimSpace(entidad))
	}
	ok, err := apiGetQuery("/api/permisos-catalogo", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Permisos, true, nil
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

func fusionarAgentePorAPI(origen, destino, destinoRespaldo, etiqueta string, retener int) (*apiAgenteFusionResponse, bool, error) {
	var resp apiAgenteFusionResponse
	ok, err := apiPost(fmt.Sprintf("/api/agentes/%s/fusionar", url.PathEscape(origen)), apiAgenteFusionRequest{
		Destino:          strings.TrimSpace(destino),
		DestinoRespaldo:  strings.TrimSpace(destinoRespaldo),
		EtiquetaRespaldo: strings.TrimSpace(etiqueta),
		Retener:          retener,
	}, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return &resp, true, nil
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
