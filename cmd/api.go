/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"orquesta/coordination"
	"orquesta/db"
)

type apiErrorResponse struct {
	Error string `json:"error"`
}

type apiStatusResponse struct {
	Agentes             []*db.Agente    `json:"agentes"`
	ConteoTareas        map[string]int  `json:"conteo_tareas"`
	Proyectos           []*db.Proyecto  `json:"proyectos"`
	AsignacionesActivas map[int64]int   `json:"asignaciones_activas"`
	SesionesActivas     map[int64]int   `json:"sesiones_activas"`
	PropuestasAbiertas  []*db.Propuesta `json:"propuestas_abiertas"`
}

type apiConteoVotosResponse struct {
	Acuerdo    int `json:"acuerdo"`
	Desacuerdo int `json:"desacuerdo"`
	Abstencion int `json:"abstencion"`
	Pendiente  int `json:"pendiente"`
}

type apiAsignacionActivarRequest struct {
	Agente   string `json:"agente"`
	Proyecto string `json:"proyecto"`
	Nota     string `json:"nota"`
}

type apiSesionInicioRequest struct {
	NuevoCodex        bool   `json:"nuevo_codex"`
	Agente            string `json:"agente"`
	Conector          string `json:"conector"`
	Proyecto          string `json:"proyecto"`
	CWD               string `json:"cwd"`
	Herramienta       string `json:"herramienta"`
	Branch            string `json:"branch"`
	ExternalSessionID string `json:"external_session_id"`
	ResumePayload     string `json:"resume_payload_json"`
	Resumen           string `json:"resumen_continuidad"`
	Host              string `json:"host"`
	PID               int64  `json:"pid"`
}

type apiSesionGuardarRequest struct {
	Agente            string `json:"agente"`
	Proyecto          string `json:"proyecto"`
	CWD               string `json:"cwd"`
	Herramienta       string `json:"herramienta"`
	Branch            string `json:"branch"`
	ExternalSessionID string `json:"external_session_id"`
	ResumePayload     string `json:"resume_payload_json"`
	Resumen           string `json:"resumen_continuidad"`
	Host              string `json:"host"`
	Estado            string `json:"estado"`
	PID               int64  `json:"pid"`
}

type apiSesionFinRequest struct {
	Agente string `json:"agente"`
}

type apiRespaldoBDRequest struct {
	Destino  string `json:"destino"`
	Etiqueta string `json:"etiqueta"`
	Retener  int    `json:"retener"`
}

type apiTareaCrearRequest struct {
	Titulo      string `json:"titulo"`
	Descripcion string `json:"descripcion"`
	Proyecto    string `json:"proyecto"`
	Modulo      string `json:"modulo"`
	Prioridad   string `json:"prioridad"`
	CreadoPor   string `json:"creado_por"`
	Notas       string `json:"notas"`
	Propuesta   string `json:"propuesta"`
	Agente      string `json:"agente"`
}

type apiTareaAccionRequest struct {
	Accion      string `json:"accion"`
	Agente      string `json:"agente"`
	Commit      string `json:"commit"`
	Motivo      string `json:"motivo"`
	Resolucion  string `json:"resolucion"`
	Nota        string `json:"nota"`
	NuevoAgente string `json:"nuevo_agente"`
}

type apiPropuestaCrearRequest struct {
	Codigo       string `json:"codigo"`
	Titulo       string `json:"titulo"`
	Descripcion  string `json:"descripcion"`
	Proyecto     string `json:"proyecto"`
	Tipo         string `json:"tipo"`
	PropuestoPor string `json:"propuesto_por"`
	Distribuidor string `json:"distribuidor"`
}

type apiPropuestaAccionRequest struct {
	Accion       string `json:"accion"`
	Agente       string `json:"agente"`
	Posicion     string `json:"posicion"`
	Comentario   string `json:"comentario"`
	EstadoCierre string `json:"estado_cierre"`
}

type apiLockRequest struct {
	Agente       string `json:"agente"`
	Proyecto     string `json:"proyecto"`
	TareaID      int64  `json:"tarea_id"`
	ScopeType    string `json:"scope_type"`
	ScopeKey     string `json:"scope_key"`
	Ruta         string `json:"ruta"`
	Branch       string `json:"branch"`
	Motivo       string `json:"motivo"`
	LeaseSeconds int    `json:"lease_seconds"`
	LeaseToken   string `json:"lease_token"`
}

type apiWorktreeRequest struct {
	Agente   string `json:"agente"`
	Proyecto string `json:"proyecto"`
	TareaID  int64  `json:"tarea_id"`
	LockID   int64  `json:"lock_id"`
	Nombre   string `json:"nombre"`
	Branch   string `json:"branch"`
	BaseRef  string `json:"base_ref"`
	Motivo   string `json:"motivo"`
	Eliminar bool   `json:"eliminar"`
}

type apiRuntimeOrderCreateRequest struct {
	Agente   string `json:"agente"`
	Proyecto string `json:"proyecto"`
	Tipo     string `json:"tipo"`
	Payload  string `json:"payload"`
}

type apiRuntimeMailboxCreateRequest struct {
	FromAgente     string `json:"from_agente"`
	ToAgente       string `json:"to_agente"`
	Proyecto       string `json:"proyecto"`
	RuntimeOrderID int64  `json:"runtime_order_id"`
	Kind           string `json:"kind"`
	Payload        string `json:"payload"`
}

type apiAgentePausarRequest struct {
	Agente  string `json:"agente"`
	Minutos int    `json:"minutos"`
	Motivo  string `json:"motivo"`
	Accion  string `json:"accion"`
	Entidad string `json:"entidad"`
	Detalle string `json:"detalle"`
}

type apiRuntimeHandoffRequest struct {
	AgenteOrigen      string `json:"agente_origen"`
	AgenteDestino     string `json:"agente_destino"`
	TareaID           int64  `json:"tarea_id"`
	Motivo            string `json:"motivo"`
	Resumen           string `json:"resumen_continuidad"`
	ExternalSessionID string `json:"external_session_id"`
}

func registerAPIRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/status", apiHandlerStatus)
	mux.HandleFunc("/api/diagnostico", apiHandlerDiagnostico)
	mux.HandleFunc("/api/agentes", apiHandlerAgentes)
	mux.HandleFunc("/api/agentes/", apiRouterAgentes)
	mux.HandleFunc("/api/config", apiHandlerConfig)
	mux.HandleFunc("/api/audit", apiHandlerAudit)
	mux.HandleFunc("/api/respaldo/bd", apiHandlerRespaldoBD)
	mux.HandleFunc("/api/proyectos", apiHandlerProyectos)
	mux.HandleFunc("/api/proyectos/descubrir", apiHandlerProyectoDescubrir)
	mux.HandleFunc("/api/proyectos/", apiRouterProyectos)
	mux.HandleFunc("/api/conectores", apiHandlerConectores)
	mux.HandleFunc("/api/conectores/", apiRouterConectores)
	mux.HandleFunc("/api/asignaciones", apiHandlerAsignaciones)
	mux.HandleFunc("/api/asignaciones/activar", apiHandlerAsignacionActivar)
	mux.HandleFunc("/api/locks", apiHandlerLocks)
	mux.HandleFunc("/api/locks/", apiRouterLocks)
	mux.HandleFunc("/api/tareas", apiHandlerTareas)
	mux.HandleFunc("/api/tareas/", apiRouterTareas)
	mux.HandleFunc("/api/propuestas", apiHandlerPropuestas)
	mux.HandleFunc("/api/propuestas/", apiRouterPropuestas)
	mux.HandleFunc("/api/worktrees", apiHandlerWorktrees)
	mux.HandleFunc("/api/worktrees/", apiRouterWorktrees)
	mux.HandleFunc("/api/runtimes", apiHandlerRuntimes)
	mux.HandleFunc("/api/runtimes/tree", apiHandlerRuntimesTree)
	mux.HandleFunc("/api/runtimes/", apiRouterRuntimes)
	mux.HandleFunc("/api/runtime-handles", apiHandlerRuntimeHandles)
	mux.HandleFunc("/api/runtime-orders", apiHandlerRuntimeOrders)
	mux.HandleFunc("/api/runtime-mailbox", apiHandlerRuntimeMailbox)
	mux.HandleFunc("/api/runtime-mailbox/", apiRouterRuntimeMailbox)
	mux.HandleFunc("/api/runtime-checkpoints/latest", apiHandlerRuntimeCheckpointLatest)
	mux.HandleFunc("/api/sesiones", apiHandlerSesiones)
	mux.HandleFunc("/api/sesiones/", apiRouterSesiones)
	mux.HandleFunc("/api/sesiones/inicio", apiHandlerSesionInicio)
	mux.HandleFunc("/api/sesiones/guardar", apiHandlerSesionGuardar)
	mux.HandleFunc("/api/sesiones/fin", apiHandlerSesionFin)
	mux.HandleFunc("/api/sesiones/continuar", apiHandlerSesionContinuar)
	mux.HandleFunc("/api/agente/handoff", apiHandlerAgenteHandoff)
	mux.HandleFunc("/api/agente/preparar", apiHandlerAgentePreparar)
	mux.HandleFunc("/api/agente/tick", apiHandlerAgenteTick)
	mux.HandleFunc("/api/agente/pausar", apiHandlerAgentePausar)
}

func apiHandlerStatus(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	agentes, err := db.ListarAgentes()
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	counts, err := db.ContarTareasPorEstado()
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	proyectos, err := db.ListarProyectos(db.FiltroProyectos{})
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	asignaciones, err := db.ContarAsignacionesActivasPorProyecto()
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	sesiones, err := db.ListarSesionesActivas()
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	sesionesPorProyecto := make(map[int64]int)
	for _, sesion := range sesiones {
		if sesion.ProyectoID != nil {
			sesionesPorProyecto[*sesion.ProyectoID]++
		}
	}
	estadoAb := db.PropuestaAbierta
	abiertas, err := db.ListarPropuestas(&estadoAb, nil)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiStatusResponse{
		Agentes:             agentes,
		ConteoTareas:        counts,
		Proyectos:           proyectos,
		AsignacionesActivas: asignaciones,
		SesionesActivas:     sesionesPorProyecto,
		PropuestasAbiertas:  abiertas,
	})
}

func apiHandlerDiagnostico(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	limitAudit := 20
	if limitStr := strings.TrimSpace(r.URL.Query().Get("audit_limit")); limitStr != "" {
		v, err := strconv.Atoi(limitStr)
		if err != nil || v <= 0 {
			apiError(w, http.StatusBadRequest, fmt.Errorf("audit_limit inválido"))
			return
		}
		limitAudit = v
	}
	snapshot, err := db.ConstruirSnapshotDiagnostico(limitAudit)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"diagnostico": snapshot})
}

func apiHandlerAgentes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		agentes, err := db.ListarAgentes()
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{"agentes": agentes})
	case http.MethodPost:
		var req apiAgenteRequest
		if err := apiDecodeJSON(r, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		if err := db.RegistrarAgente(strings.TrimSpace(req.Nombre), strings.TrimSpace(req.Rol)); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		apiWriteJSON(w, http.StatusCreated, map[string]any{"ok": true})
	default:
		apiMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func apiRouterAgentes(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/agentes/"), "/"), "/")
	if len(parts) != 2 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	nombre := parts[0]
	switch parts[1] {
	case "retirar":
		if err := db.RetirarAgente(nombre); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
	case "rehabilitar":
		if err := db.RehabilitarAgente(nombre); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
	case "eliminar":
		if err := db.EliminarAgente(nombre); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		db.Audit("alberto", "purgar_agente", "agente", 0, nombre)
	case "reset-reanimacion":
		if err := db.ResetReanimacion(nombre); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
	default:
		http.NotFound(w, r)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"ok": true, "agente": nombre})
}

func apiHandlerConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if clave := strings.TrimSpace(r.URL.Query().Get("clave")); clave != "" {
			valor, err := db.ConfigGet(clave)
			if err != nil {
				apiError(w, http.StatusNotFound, err)
				return
			}
			apiWriteJSON(w, http.StatusOK, map[string]any{"clave": clave, "valor": valor})
			return
		}
		config, err := db.ConfigAll()
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{"config": config})
	case http.MethodPost:
		var req apiConfigSetRequest
		if err := apiDecodeJSON(r, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		req.Clave = strings.TrimSpace(req.Clave)
		if req.Clave == "" {
			apiError(w, http.StatusBadRequest, fmt.Errorf("clave obligatoria"))
			return
		}
		if err := db.ConfigSet(req.Clave, req.Valor); err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{"ok": true, "clave": req.Clave, "valor": req.Valor})
	default:
		apiMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func apiHandlerAudit(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	limit := 50
	if limitStr := strings.TrimSpace(r.URL.Query().Get("limit")); limitStr != "" {
		v, err := strconv.Atoi(limitStr)
		if err != nil || v <= 0 {
			apiError(w, http.StatusBadRequest, fmt.Errorf("limit inválido"))
			return
		}
		limit = v
	}
	fetchLimit := limit
	if fetchLimit < 200 {
		fetchLimit = 200
	}
	entries, err := db.AuditLog(fetchLimit)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	agenteFiltro := strings.TrimSpace(r.URL.Query().Get("agente"))
	accionFiltro := strings.TrimSpace(r.URL.Query().Get("accion"))
	entidadFiltro := strings.TrimSpace(r.URL.Query().Get("entidad"))
	entidadIDFiltro := strings.TrimSpace(r.URL.Query().Get("entidad_id"))

	var entidadID int64
	var filtrarEntidadID bool
	if entidadIDFiltro != "" {
		v, err := strconv.ParseInt(entidadIDFiltro, 10, 64)
		if err != nil {
			apiError(w, http.StatusBadRequest, fmt.Errorf("entidad_id inválido"))
			return
		}
		entidadID = v
		filtrarEntidadID = true
	}

	filtered := make([]db.AuditEntry, 0, len(entries))
	for _, entry := range entries {
		if agenteFiltro != "" && entry.Agente != agenteFiltro {
			continue
		}
		if accionFiltro != "" && entry.Accion != accionFiltro {
			continue
		}
		if entidadFiltro != "" && entry.Entidad != entidadFiltro {
			continue
		}
		if filtrarEntidadID && entry.EntidadID != entidadID {
			continue
		}
		filtered = append(filtered, entry)
		if len(filtered) >= limit {
			break
		}
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"audit": filtered})
}

func apiHandlerRespaldoBD(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req apiRespaldoBDRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	ruta, err := ejecutarRespaldoBD(req.Destino, req.Etiqueta, req.Retener)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiRespaldoBDResponse{OK: true, Ruta: ruta})
}

func apiHandlerProyectos(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	proyectos, err := db.ListarProyectos(db.FiltroProyectos{})
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"proyectos": proyectos})
}

func apiHandlerProyectoDescubrir(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req apiProyectoDescubrirRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	proyectos, err := db.DescubrirProyectos(req.Ruta)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(req.Ruta) != "" {
		abs, err := filepath.Abs(req.Ruta)
		if err == nil {
			_ = db.ConfigSet("workspace_root", abs)
		}
	}
	apiWriteJSON(w, http.StatusCreated, map[string]any{"proyectos": proyectos})
}

func apiRouterProyectos(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	ref := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/proyectos/"), "/")
	if ref == "" {
		http.NotFound(w, r)
		return
	}
	proyecto, err := db.GetProyecto(ref)
	if err != nil {
		apiError(w, http.StatusNotFound, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"proyecto": proyecto})
}

func apiHandlerConectores(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		conectores, err := db.ListarConectores()
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{"conectores": conectores})
	case http.MethodPost:
		var req apiConectorUpsertRequest
		if err := apiDecodeJSON(r, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		id, err := db.UpsertConector(&db.Conector{
			Slug:         strings.TrimSpace(req.Slug),
			Nombre:       strings.TrimSpace(req.Nombre),
			Transporte:   strings.TrimSpace(req.Transporte),
			Comando:      req.Comando,
			ArgsJSON:     req.ArgsJSON,
			EnvJSON:      req.EnvJSON,
			MetadataJSON: req.MetadataJSON,
			Activo:       req.Activo,
		})
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		conector, err := db.GetConector(strconv.FormatInt(id, 10))
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusCreated, map[string]any{"ok": true, "conector": conector})
	default:
		apiMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func apiRouterConectores(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	ref := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/conectores/"), "/")
	if ref == "" {
		http.NotFound(w, r)
		return
	}
	conector, err := db.GetConector(ref)
	if err != nil {
		apiError(w, http.StatusNotFound, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"conector": conector})
}

func apiHandlerAsignaciones(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	filtro := db.FiltroAsignaciones{}
	if agente := strings.TrimSpace(r.URL.Query().Get("agente")); agente != "" {
		filtro.Agente = &agente
	}
	if proyectoRef := strings.TrimSpace(r.URL.Query().Get("proyecto")); proyectoRef != "" {
		proyecto, err := db.GetProyecto(proyectoRef)
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		filtro.ProyectoID = &proyecto.ID
	}
	if estadoStr := strings.TrimSpace(r.URL.Query().Get("estado")); estadoStr != "" {
		estado := db.EstadoAsignacion(estadoStr)
		filtro.Estado = &estado
	}
	asignaciones, err := db.ListarAsignaciones(filtro)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"asignaciones": asignaciones})
}

func apiHandlerAsignacionActivar(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req apiAsignacionActivarRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	proyecto, err := db.GetProyecto(req.Proyecto)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	if err := db.ActivarAsignacion(req.Agente, proyecto.ID, req.Nota); err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"agente":   req.Agente,
		"proyecto": proyecto,
	})
}

func apiHandlerLocks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		repo := db.SQLiteLockRepository{}
		filter := coordination.LockFilter{}
		if agente := strings.TrimSpace(r.URL.Query().Get("agente")); agente != "" {
			filter.Agent = &agente
		}
		if proyectoRef := strings.TrimSpace(r.URL.Query().Get("proyecto")); proyectoRef != "" {
			proyecto, err := db.GetProyecto(proyectoRef)
			if err != nil {
				apiError(w, http.StatusBadRequest, err)
				return
			}
			filter.ProjectID = &proyecto.ID
		}
		if estadoStr := strings.TrimSpace(r.URL.Query().Get("estado")); estadoStr != "" {
			estado := coordination.LockState(estadoStr)
			filter.State = &estado
		}
		locks, err := repo.List(filter)
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{"locks": locks})
	case http.MethodPost:
		var req apiLockRequest
		if err := apiDecodeJSON(r, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		svc := newCoordinationService()
		var projectID *int64
		if strings.TrimSpace(req.Proyecto) != "" {
			proyecto, err := db.GetProyecto(req.Proyecto)
			if err != nil {
				apiError(w, http.StatusBadRequest, err)
				return
			}
			projectID = &proyecto.ID
		}
		var taskID *int64
		if req.TareaID > 0 {
			taskID = &req.TareaID
		}
		var sessionID *int64
		if projectID != nil {
			sesion, err := db.GetSesionActiva(req.Agente, projectID)
			if err == nil && sesion != nil {
				sessionID = &sesion.ID
			}
		}
		lock, err := svc.AcquireLock(coordination.AcquireLockInput{
			ProjectID:    projectID,
			TaskID:       taskID,
			SessionID:    sessionID,
			Agent:        req.Agente,
			ScopeType:    req.ScopeType,
			ScopeKey:     req.ScopeKey,
			Path:         req.Ruta,
			Branch:       req.Branch,
			Reason:       req.Motivo,
			LeaseSeconds: req.LeaseSeconds,
		})
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		apiWriteJSON(w, http.StatusCreated, map[string]any{"ok": true, "lock": lock})
	default:
		apiMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func apiRouterLocks(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/locks/"), "/"), "/")
	if len(parts) == 1 && r.Method == http.MethodGet {
		id, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			apiError(w, http.StatusBadRequest, fmt.Errorf("id inválido"))
			return
		}
		lock, err := (db.SQLiteLockRepository{}).GetByID(id)
		if err != nil {
			apiError(w, http.StatusNotFound, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{"lock": lock})
		return
	}
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		apiError(w, http.StatusBadRequest, fmt.Errorf("id inválido"))
		return
	}
	var req apiLockRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	svc := newCoordinationService()
	switch {
	case parts[1] == "renovar" && r.Method == http.MethodPost:
		lock, err := svc.RenewLock(coordination.RenewLockInput{
			ID:           id,
			Agent:        req.Agente,
			LeaseToken:   req.LeaseToken,
			LeaseSeconds: req.LeaseSeconds,
		})
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{"ok": true, "lock": lock})
	case parts[1] == "liberar" && r.Method == http.MethodPost:
		lock, err := svc.ReleaseLock(coordination.ReleaseLockInput{
			ID:         id,
			Agent:      req.Agente,
			LeaseToken: req.LeaseToken,
			Reason:     req.Motivo,
		})
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{"ok": true, "lock": lock})
	default:
		http.NotFound(w, r)
	}
}

func apiHandlerTareas(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		apiHandlerTareasListar(w, r)
	case http.MethodPost:
		apiHandlerTareasCrear(w, r)
	default:
		apiMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func apiHandlerTareasListar(w http.ResponseWriter, r *http.Request) {
	filtro := db.FiltroTareas{}
	if agente := strings.TrimSpace(r.URL.Query().Get("agente")); agente != "" {
		filtro.Agente = &agente
	}
	if modulo := strings.TrimSpace(r.URL.Query().Get("modulo")); modulo != "" {
		filtro.Modulo = &modulo
	}
	if estadoStr := strings.TrimSpace(r.URL.Query().Get("estado")); estadoStr != "" {
		estado := db.EstadoTarea(estadoStr)
		filtro.Estado = &estado
	}
	if proyectoRef := strings.TrimSpace(r.URL.Query().Get("proyecto")); proyectoRef != "" {
		proyecto, err := db.GetProyecto(proyectoRef)
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		filtro.ProyectoID = &proyecto.ID
	}
	if propuestaCodigo := strings.TrimSpace(r.URL.Query().Get("propuesta")); propuestaCodigo != "" {
		propuesta, err := db.GetPropuesta(propuestaCodigo)
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		filtro.PropuestaID = &propuesta.ID
	}
	tareas, err := db.ListarTareas(filtro)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"tareas": tareas})
}

func apiHandlerTareasCrear(w http.ResponseWriter, r *http.Request) {
	var req apiTareaCrearRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	t := &db.Tarea{
		Titulo:      strings.TrimSpace(req.Titulo),
		Descripcion: req.Descripcion,
		Modulo:      req.Modulo,
		Prioridad:   db.PrioridadTarea(req.Prioridad),
		CreadoPor:   valorConFallback(req.CreadoPor, "alberto"),
		Notas:       req.Notas,
	}
	if t.Titulo == "" {
		apiError(w, http.StatusBadRequest, fmt.Errorf("el título es obligatorio"))
		return
	}
	if proyectoRef := strings.TrimSpace(req.Proyecto); proyectoRef != "" {
		proyecto, err := db.GetProyecto(proyectoRef)
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		t.ProyectoID = &proyecto.ID
	}
	if propuestaRef := strings.TrimSpace(req.Propuesta); propuestaRef != "" {
		propuesta, err := db.GetPropuesta(propuestaRef)
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		t.PropuestaID = &propuesta.ID
	}
	id, err := db.CrearTarea(t)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	if agente := strings.TrimSpace(req.Agente); agente != "" {
		if err := db.TomarTarea(id, agente); err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
	}
	tarea, err := db.GetTarea(id)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusCreated, map[string]any{"ok": true, "tarea": tarea})
}

func apiRouterTareas(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/tareas/"), "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	if len(parts) == 1 && r.Method == http.MethodGet {
		id, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			apiError(w, http.StatusBadRequest, fmt.Errorf("id inválido"))
			return
		}
		tarea, err := db.GetTarea(id)
		if err != nil {
			apiError(w, http.StatusNotFound, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{"tarea": tarea})
		return
	}
	if len(parts) == 2 && parts[1] == "accion" && r.Method == http.MethodPost {
		apiHandlerTareaAccion(w, r, parts[0])
		return
	}
	http.NotFound(w, r)
}

func apiHandlerTareaAccion(w http.ResponseWriter, r *http.Request, idStr string) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		apiError(w, http.StatusBadRequest, fmt.Errorf("id inválido"))
		return
	}
	var req apiTareaAccionRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	switch req.Accion {
	case "tomar":
		err = db.TomarTarea(id, req.Agente)
	case "iniciar":
		err = db.IniciarTarea(id, req.Agente)
	case "completar":
		err = db.CompletarTarea(id, req.Agente, req.Commit)
	case "bloquear":
		err = db.BloquearTarea(id, req.Agente, req.Motivo)
	case "desbloquear":
		err = db.DesbloquearTarea(id, req.Agente, req.Resolucion)
	case "nota":
		err = db.AnotarTarea(id, req.Agente, req.Nota)
	case "contrato":
		err = db.DefinirContrato(id, req.Agente)
	case "backlog":
		_, err = db.DB.Exec(`UPDATE tareas SET estado='backlog', agente=NULL WHERE id=?`, id)
		if err == nil {
			db.Audit("alberto", "backlog_tarea", "tarea", id, "")
		}
	case "cancelar":
		err = db.CancelarTarea(id, req.Agente, valorConFallback(req.Motivo, "duplicado o error"))
	case "reasignar":
		_, err = db.DB.Exec(`UPDATE tareas SET agente=?, estado='asignada' WHERE id=?`, req.NuevoAgente, id)
		if err == nil {
			db.Audit("alberto", "reasignar_tarea", "tarea", id, req.NuevoAgente)
		}
	default:
		apiError(w, http.StatusBadRequest, fmt.Errorf("acción desconocida"))
		return
	}
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	tarea, err := db.GetTarea(id)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"ok": true, "tarea": tarea})
}

func apiHandlerPropuestas(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		apiHandlerPropuestasListar(w, r)
	case http.MethodPost:
		apiHandlerPropuestasCrear(w, r)
	default:
		apiMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func apiHandlerWorktrees(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		repo := db.SQLiteWorktreeRepository{}
		filter := coordination.WorktreeFilter{}
		if agente := strings.TrimSpace(r.URL.Query().Get("agente")); agente != "" {
			filter.Agent = &agente
		}
		if proyectoRef := strings.TrimSpace(r.URL.Query().Get("proyecto")); proyectoRef != "" {
			proyecto, err := db.GetProyecto(proyectoRef)
			if err != nil {
				apiError(w, http.StatusBadRequest, err)
				return
			}
			filter.ProjectID = &proyecto.ID
		}
		if estadoStr := strings.TrimSpace(r.URL.Query().Get("estado")); estadoStr != "" {
			estado := coordination.WorktreeState(estadoStr)
			filter.State = &estado
		}
		worktrees, err := repo.List(filter)
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{"worktrees": worktrees})
	case http.MethodPost:
		var req apiWorktreeRequest
		if err := apiDecodeJSON(r, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		svc := newCoordinationService()
		var taskID *int64
		if req.TareaID > 0 {
			taskID = &req.TareaID
		}
		var lockID *int64
		if req.LockID > 0 {
			lockID = &req.LockID
		}
		worktree, err := svc.PrepareWorktree(coordination.PrepareWorktreeInput{
			ProjectRef: req.Proyecto,
			Agent:      req.Agente,
			TaskID:     taskID,
			LockID:     lockID,
			Name:       req.Nombre,
			Branch:     req.Branch,
			BaseRef:    req.BaseRef,
			Reason:     req.Motivo,
		})
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		apiWriteJSON(w, http.StatusCreated, map[string]any{"ok": true, "worktree": worktree})
	default:
		apiMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func apiHandlerRuntimes(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	filter, err := apiFiltroRuntimesDesdeRequest(r)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	runtimes, err := db.ListarRuntimes(filter)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"runtimes": runtimes})
}

func apiHandlerRuntimesTree(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	filter, err := apiFiltroRuntimesDesdeRequest(r)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	tree, err := db.ConstruirArbolRuntimes(filter)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"runtimes": tree})
}

func apiRouterRuntimes(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/runtimes/"), "/"), "/")
	if len(parts) != 1 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		apiError(w, http.StatusBadRequest, fmt.Errorf("id inválido"))
		return
	}
	runtime, err := db.GetRuntime(id)
	if err != nil {
		apiError(w, http.StatusNotFound, err)
		return
	}
	samples, err := db.ListarMuestrasRuntime(id, 20)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"runtime": runtime, "samples": samples})
}

func apiHandlerRuntimeHandles(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	agente := strings.TrimSpace(r.URL.Query().Get("agente"))
	var filtro *string
	if agente != "" {
		filtro = &agente
	}
	handles, err := db.ListarRuntimeHandles(filtro)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"handles": handles})
}

func apiHandlerRuntimeOrders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		filter := db.FiltroRuntimeOrders{}
		if agente := strings.TrimSpace(r.URL.Query().Get("agente")); agente != "" {
			filter.Agente = &agente
		}
		if estado := strings.TrimSpace(r.URL.Query().Get("estado")); estado != "" {
			filter.Estado = &estado
		}
		if proyecto := strings.TrimSpace(r.URL.Query().Get("proyecto")); proyecto != "" {
			p, err := db.GetProyecto(proyecto)
			if err != nil {
				apiError(w, http.StatusBadRequest, err)
				return
			}
			filter.ProyectoID = &p.ID
		}
		orders, err := db.ListarRuntimeOrders(filter)
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{"orders": orders})
	case http.MethodPost:
		var req apiRuntimeOrderCreateRequest
		if err := apiDecodeJSON(r, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		payload := strings.TrimSpace(req.Payload)
		if payload == "" {
			payload = "{}"
		}
		var proyectoID *int64
		if proyecto := strings.TrimSpace(req.Proyecto); proyecto != "" {
			p, err := db.GetProyecto(proyecto)
			if err != nil {
				apiError(w, http.StatusBadRequest, err)
				return
			}
			proyectoID = &p.ID
		}
		id, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
			Agente:      strings.TrimSpace(req.Agente),
			ProyectoID:  proyectoID,
			Tipo:        strings.TrimSpace(req.Tipo),
			PayloadJSON: payload,
		})
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		apiWriteJSON(w, http.StatusCreated, map[string]any{"ok": true, "id": id})
	default:
		apiMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func apiHandlerRuntimeMailbox(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		filter := db.FiltroRuntimeMailbox{}
		if toAgente := strings.TrimSpace(r.URL.Query().Get("to_agente")); toAgente != "" {
			filter.ToAgente = &toAgente
		}
		if fromAgente := strings.TrimSpace(r.URL.Query().Get("from_agente")); fromAgente != "" {
			filter.FromAgente = &fromAgente
		}
		if estado := strings.TrimSpace(r.URL.Query().Get("estado")); estado != "" {
			filter.Estado = &estado
		}
		if proyecto := strings.TrimSpace(r.URL.Query().Get("proyecto")); proyecto != "" {
			p, err := db.GetProyecto(proyecto)
			if err != nil {
				apiError(w, http.StatusBadRequest, err)
				return
			}
			filter.ProyectoID = &p.ID
		}
		mailbox, err := db.ListarRuntimeMailbox(filter)
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{"mailbox": mailbox})
	case http.MethodPost:
		var req apiRuntimeMailboxCreateRequest
		if err := apiDecodeJSON(r, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		payload := strings.TrimSpace(req.Payload)
		if payload == "" {
			payload = "{}"
		}
		var proyectoID *int64
		if proyecto := strings.TrimSpace(req.Proyecto); proyecto != "" {
			p, err := db.GetProyecto(proyecto)
			if err != nil {
				apiError(w, http.StatusBadRequest, err)
				return
			}
			proyectoID = &p.ID
		}
		var runtimeOrderID *int64
		if req.RuntimeOrderID > 0 {
			runtimeOrderID = &req.RuntimeOrderID
		}
		id, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
			FromAgente:     strings.TrimSpace(req.FromAgente),
			ToAgente:       strings.TrimSpace(req.ToAgente),
			ProyectoID:     proyectoID,
			RuntimeOrderID: runtimeOrderID,
			Kind:           strings.TrimSpace(req.Kind),
			PayloadJSON:    payload,
		})
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		apiWriteJSON(w, http.StatusCreated, map[string]any{"ok": true, "id": id})
	default:
		apiMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func apiRouterRuntimeMailbox(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/runtime-mailbox/"), "/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		http.NotFound(w, r)
		return
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || id <= 0 {
		apiError(w, http.StatusBadRequest, fmt.Errorf("id inválido"))
		return
	}
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	switch parts[1] {
	case "entregar":
		if err := db.MarcarRuntimeMailboxEntregado(id); err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
	case "consumir":
		if err := db.MarcarRuntimeMailboxConsumido(id); err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
	default:
		http.NotFound(w, r)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"ok": true, "id": id})
}

func apiHandlerRuntimeCheckpointLatest(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	agente := strings.TrimSpace(r.URL.Query().Get("agente"))
	if agente == "" {
		apiError(w, http.StatusBadRequest, fmt.Errorf("agente obligatorio"))
		return
	}
	var proyectoID *int64
	if proyecto := strings.TrimSpace(r.URL.Query().Get("proyecto")); proyecto != "" {
		p, err := db.GetProyecto(proyecto)
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		proyectoID = &p.ID
	}
	checkpoint, err := db.UltimoRuntimeCheckpoint(agente, proyectoID)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"checkpoint": checkpoint})
}

func apiHandlerAgenteHandoff(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req apiRuntimeHandoffRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	var tareaID *int64
	if req.TareaID > 0 {
		tareaID = &req.TareaID
	}
	id, err := db.CrearHandoffAgenteVivo(
		strings.TrimSpace(req.AgenteOrigen),
		strings.TrimSpace(req.AgenteDestino),
		tareaID,
		strings.TrimSpace(req.Motivo),
		strings.TrimSpace(req.Resumen),
		strings.TrimSpace(req.ExternalSessionID),
	)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusCreated, map[string]any{"ok": true, "id": id, "order_id": id})
}

func apiRouterWorktrees(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/worktrees/"), "/"), "/")
	if len(parts) == 1 && r.Method == http.MethodGet {
		id, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			apiError(w, http.StatusBadRequest, fmt.Errorf("id inválido"))
			return
		}
		worktree, err := (db.SQLiteWorktreeRepository{}).GetByID(id)
		if err != nil {
			apiError(w, http.StatusNotFound, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{"worktree": worktree})
		return
	}
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		apiError(w, http.StatusBadRequest, fmt.Errorf("id inválido"))
		return
	}
	if parts[1] != "cerrar" || r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	var req apiWorktreeRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	worktree, err := newCoordinationService().CloseWorktree(id, req.Eliminar, req.Motivo)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"ok": true, "worktree": worktree})
}

func apiHandlerPropuestasListar(w http.ResponseWriter, r *http.Request) {
	var estadoPtr *db.EstadoPropuesta
	if estadoStr := strings.TrimSpace(r.URL.Query().Get("estado")); estadoStr != "" {
		estado := db.EstadoPropuesta(estadoStr)
		estadoPtr = &estado
	}
	var proyectoID *int64
	if proyectoRef := strings.TrimSpace(r.URL.Query().Get("proyecto")); proyectoRef != "" {
		proyecto, err := db.GetProyecto(proyectoRef)
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		proyectoID = &proyecto.ID
	}
	propuestas, err := db.ListarPropuestas(estadoPtr, proyectoID)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"propuestas": propuestas})
}

func apiFiltroRuntimesDesdeRequest(r *http.Request) (db.FiltroRuntimes, error) {
	filter := db.FiltroRuntimes{}
	if agente := strings.TrimSpace(r.URL.Query().Get("agente")); agente != "" {
		filter.Agente = &agente
	}
	if proyectoRef := strings.TrimSpace(r.URL.Query().Get("proyecto")); proyectoRef != "" {
		proyecto, err := db.GetProyecto(proyectoRef)
		if err != nil {
			return filter, err
		}
		filter.ProyectoID = &proyecto.ID
	}
	if activosStr := strings.TrimSpace(r.URL.Query().Get("activos")); activosStr != "" {
		switch activosStr {
		case "true":
			v := true
			filter.Activos = &v
		case "false":
			v := false
			filter.Activos = &v
		default:
			return filter, fmt.Errorf("activos inválido")
		}
	}
	return filter, nil
}

func apiHandlerPropuestasCrear(w http.ResponseWriter, r *http.Request) {
	var req apiPropuestaCrearRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	p := &db.Propuesta{
		Codigo:       strings.TrimSpace(req.Codigo),
		Titulo:       strings.TrimSpace(req.Titulo),
		Descripcion:  req.Descripcion,
		Tipo:         valorConFallback(req.Tipo, "implementacion"),
		PropuestoPor: valorConFallback(req.PropuestoPor, "alberto"),
		Distribuidor: valorConFallback(req.Distribuidor, "alberto"),
	}
	if p.Titulo == "" {
		apiError(w, http.StatusBadRequest, fmt.Errorf("el título es obligatorio"))
		return
	}
	if proyectoRef := strings.TrimSpace(req.Proyecto); proyectoRef != "" {
		proyecto, err := db.GetProyecto(proyectoRef)
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		p.ProyectoID = &proyecto.ID
	}
	id, err := db.CrearPropuesta(p)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	propuesta, err := db.GetPropuesta(p.Codigo)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusCreated, map[string]any{"ok": true, "id": id, "propuesta": propuesta})
}

func apiRouterPropuestas(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/propuestas/"), "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	if len(parts) == 1 && r.Method == http.MethodGet {
		propuesta, err := db.GetPropuesta(parts[0])
		if err != nil {
			apiError(w, http.StatusNotFound, err)
			return
		}
		votos, err := db.ResumenVotos(propuesta.ID)
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		propuesta.Votos = votos
		acuerdo, desacuerdo, abstencion, pendiente, err := db.ContarVotos(propuesta.ID)
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{
			"propuesta": propuesta,
			"conteo_votos": apiConteoVotosResponse{
				Acuerdo: acuerdo, Desacuerdo: desacuerdo, Abstencion: abstencion, Pendiente: pendiente,
			},
		})
		return
	}
	if len(parts) == 2 && parts[1] == "accion" && r.Method == http.MethodPost {
		apiHandlerPropuestaAccion(w, r, parts[0])
		return
	}
	http.NotFound(w, r)
}

func apiHandlerSesiones(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	filtro := db.FiltroSesionesInspeccion{}
	limit := 0
	if limitStr := strings.TrimSpace(r.URL.Query().Get("limit")); limitStr != "" {
		v, err := strconv.Atoi(limitStr)
		if err != nil || v <= 0 {
			apiError(w, http.StatusBadRequest, fmt.Errorf("limit inválido"))
			return
		}
		limit = v
	}
	agenteFiltro := strings.TrimSpace(r.URL.Query().Get("agente"))
	if agenteFiltro != "" {
		filtro.Agente = &agenteFiltro
	}
	if proyectoFiltro := strings.TrimSpace(r.URL.Query().Get("proyecto")); proyectoFiltro != "" {
		proyecto, err := db.GetProyecto(proyectoFiltro)
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		filtro.ProyectoID = &proyecto.ID
	}
	if activaStr := strings.TrimSpace(r.URL.Query().Get("activa")); activaStr != "" {
		switch activaStr {
		case "true":
			v := true
			filtro.Activa = &v
		case "false":
			v := false
			filtro.Activa = &v
		default:
			apiError(w, http.StatusBadRequest, fmt.Errorf("activa inválido"))
			return
		}
	}
	if estado := strings.TrimSpace(r.URL.Query().Get("estado")); estado != "" {
		filtro.Estado = &estado
	}
	sesiones, err := db.ListarSesionesInspeccion(filtro)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	if limit > 0 && len(sesiones) > limit {
		sesiones = sesiones[:limit]
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"sesiones": sesiones})
}

func apiRouterSesiones(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/sesiones/"), "/"), "/")
	if len(parts) != 1 || parts[0] == "" || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		apiError(w, http.StatusBadRequest, fmt.Errorf("id inválido"))
		return
	}
	sesion, err := db.GetSesionInspeccionByID(id)
	if err != nil {
		apiError(w, http.StatusNotFound, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"sesion": sesion})
}

func apiHandlerPropuestaAccion(w http.ResponseWriter, r *http.Request, codigo string) {
	propuesta, err := db.GetPropuesta(codigo)
	if err != nil {
		apiError(w, http.StatusNotFound, err)
		return
	}
	var req apiPropuestaAccionRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	switch req.Accion {
	case "votar":
		_, err = db.Votar(propuesta.ID, req.Agente, db.PosicionVoto(req.Posicion), req.Comentario)
	case "cerrar":
		err = db.CerrarPropuesta(codigo, req.EstadoCierre, valorConFallback(req.Agente, "alberto"))
	case "reabrir":
		_, err = db.ReabrirPropuesta(codigo, valorConFallback(req.Agente, "alberto"))
	case "reparar_votos", "reparar-votos":
		_, err = db.RepararVotosPendientesPropuesta(codigo, valorConFallback(req.Agente, "alberto"))
	default:
		apiError(w, http.StatusBadRequest, fmt.Errorf("acción desconocida"))
		return
	}
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	propuesta, err = db.GetPropuesta(codigo)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"ok": true, "propuesta": propuesta})
}

func apiHandlerSesionInicio(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req apiSesionInicioRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	resp, err := construirSesionInicioResponse(req)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusCreated, resp)
}

func apiHandlerSesionGuardar(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req apiSesionGuardarRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	var proyectoID *int64
	if strings.TrimSpace(req.Proyecto) != "" {
		proyecto, err := db.GetProyecto(req.Proyecto)
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		proyectoID = &proyecto.ID
	}
	upd := db.SesionUpdate{Heartbeat: true}
	if req.CWD != "" {
		upd.CWD = &req.CWD
	}
	if req.Herramienta != "" {
		upd.Herramienta = &req.Herramienta
	}
	if req.Branch != "" {
		upd.Branch = &req.Branch
	}
	if req.ExternalSessionID != "" {
		upd.ExternalSessionID = &req.ExternalSessionID
	}
	if req.ResumePayload != "" {
		upd.ResumePayloadJSON = &req.ResumePayload
	}
	if req.Resumen != "" {
		upd.ResumenContinuidad = &req.Resumen
	}
	if req.Host != "" {
		upd.Host = &req.Host
	}
	if req.Estado != "" {
		upd.Estado = &req.Estado
	}
	if req.PID > 0 {
		upd.PID = &req.PID
	}
	if err := db.GuardarSesionActiva(req.Agente, proyectoID, upd); err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	sesion, err := db.GetSesionActiva(req.Agente, proyectoID)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"ok": true, "sesion": sesion})
}

func apiHandlerSesionFin(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req apiSesionFinRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	if err := db.FinSesion(req.Agente); err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func apiHandlerSesionContinuar(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	agente := strings.TrimSpace(r.URL.Query().Get("agente"))
	if agente == "" {
		apiError(w, http.StatusBadRequest, fmt.Errorf("debes indicar agente"))
		return
	}
	var proyectoID *int64
	cwd := strings.TrimSpace(r.URL.Query().Get("cwd"))
	if proyectoRef := strings.TrimSpace(r.URL.Query().Get("proyecto")); proyectoRef != "" {
		proyecto, err := db.GetProyecto(proyectoRef)
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		proyectoID = &proyecto.ID
	}
	sesion, err := db.ObtenerUltimaSesionConFiltro(agente, proyectoID, cwd)
	if err != nil {
		apiError(w, http.StatusNotFound, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"sesion": sesion})
}

func apiHandlerAgentePreparar(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	agenteNombre := strings.TrimSpace(r.URL.Query().Get("agente"))
	proyectoRef := strings.TrimSpace(r.URL.Query().Get("proyecto"))
	conectorRef := strings.TrimSpace(r.URL.Query().Get("conector"))
	modelo := strings.TrimSpace(r.URL.Query().Get("modelo"))
	razonamiento := strings.TrimSpace(r.URL.Query().Get("razonamiento"))
	perfilTarea := strings.TrimSpace(r.URL.Query().Get("perfil"))
	if agenteNombre == "" || proyectoRef == "" {
		apiError(w, http.StatusBadRequest, fmt.Errorf("debes indicar agente y proyecto"))
		return
	}

	agente, err := db.GetAgente(agenteNombre)
	if err != nil {
		apiError(w, http.StatusNotFound, err)
		return
	}
	proyecto, err := db.GetProyecto(proyectoRef)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	ultima, err := db.ObtenerUltimaSesion(agenteNombre, &proyecto.ID)
	if err != nil && err != sql.ErrNoRows {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	conector, err := resolverConectorPreparacion(agenteNombre, conectorRef, ultima)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	out, err := construirAgentePrepararOutputDesdeDatos(agente, proyecto, conector, ultima, modelo, razonamiento, perfilTarea)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, out)
}

func apiHandlerAgenteTick(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req struct {
		Agente     string `json:"agente"`
		Proyecto   string `json:"proyecto"`
		Host       string `json:"host"`
		PID        int64  `json:"pid"`
		CuotaPct   int    `json:"cuota_pct"`
		Finalizado bool   `json:"finalizado"`
		Motivo     string `json:"motivo"`
	}
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	if req.Agente == "" || req.Proyecto == "" {
		apiError(w, http.StatusBadRequest, fmt.Errorf("debes indicar agente y proyecto"))
		return
	}
	proyecto, err := db.GetProyecto(req.Proyecto)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	sesionActiva, err := db.GetSesionActiva(req.Agente, &proyecto.ID)
	if err != nil && err != sql.ErrNoRows {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	if sesionActiva != nil {
		upd := db.SesionUpdate{Heartbeat: true}
		if req.Host != "" {
			upd.Host = &req.Host
		}
		if req.PID > 0 {
			upd.PID = &req.PID
		}
		if err := db.GuardarSesionActiva(req.Agente, &proyecto.ID, upd); err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		// Si el agente informa que termina por cuota, lo pausamos automáticamente (OP-084)
		if req.Finalizado && debeAutoPausarPorAgotamiento(req.CuotaPct, req.Motivo) {
			if err := registrarAutoPausaLocal(
				req.Agente,
				60,
				"Auto-pausa por agotamiento: "+req.Motivo,
				"Detección de agotamiento en tick final: "+req.Motivo,
			); err != nil {
				apiError(w, http.StatusInternalServerError, err)
				return
			}
		}
		sesionActiva, err = db.GetSesionActiva(req.Agente, &proyecto.ID)
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
	}
	out, err := construirAgenteTickOutput(req.Agente, proyecto, sesionActiva, req.CuotaPct)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, out)
}

func apiHandlerAgentePausar(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req apiAgentePausarRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	req.Agente = strings.TrimSpace(req.Agente)
	req.Motivo = strings.TrimSpace(req.Motivo)
	if req.Agente == "" || req.Minutos <= 0 || req.Motivo == "" {
		apiError(w, http.StatusBadRequest, fmt.Errorf("debes indicar agente, minutos positivos y motivo"))
		return
	}
	if err := db.PausarAgente(req.Agente, req.Minutos, req.Motivo); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	accion := valorConFallback(strings.TrimSpace(req.Accion), "pausa_externa")
	entidad := valorConFallback(strings.TrimSpace(req.Entidad), "agente")
	detalle := strings.TrimSpace(req.Detalle)
	if detalle == "" {
		detalle = fmt.Sprintf("Bloqueado %d min por: %s", req.Minutos, req.Motivo)
	}
	db.Audit(req.Agente, accion, entidad, 0, detalle)
	apiWriteJSON(w, http.StatusOK, map[string]any{"ok": true, "agente": req.Agente})
}

func iniciarSesionDesdeRequest(req apiSesionInicioRequest) (*db.Sesion, error) {
	var proyectoID *int64
	if strings.TrimSpace(req.Proyecto) != "" {
		proyecto, err := db.GetProyecto(req.Proyecto)
		if err != nil {
			return nil, err
		}
		proyectoID = &proyecto.ID
		if strings.TrimSpace(req.CWD) == "" {
			req.CWD = proyecto.RutaAbs
		}
		if err := db.ActivarAsignacion(req.Agente, proyecto.ID, "asignación automática al iniciar sesión"); err != nil {
			return nil, err
		}
	}
	var conectorID *int64
	if strings.TrimSpace(req.Conector) != "" {
		conector, err := db.GetConector(req.Conector)
		if err != nil {
			return nil, err
		}
		conectorID = &conector.ID
		if strings.TrimSpace(req.Herramienta) == "" {
			req.Herramienta = conector.Slug
		}
	}
	var pid *int64
	if req.PID > 0 {
		pid = &req.PID
	}
	return db.IniciarSesionContexto(db.SesionInicio{
		Agente:             req.Agente,
		ConectorID:         conectorID,
		ProyectoID:         proyectoID,
		CWD:                req.CWD,
		Herramienta:        req.Herramienta,
		ExternalSessionID:  req.ExternalSessionID,
		ResumePayloadJSON:  req.ResumePayload,
		ResumenContinuidad: req.Resumen,
		Branch:             req.Branch,
		Host:               req.Host,
		PID:                pid,
	})
}

func construirSesionInicioResponse(req apiSesionInicioRequest) (*apiSesionInicioResponse, error) {
	if req.NuevoCodex {
		nombre, err := db.RegistrarCodex()
		if err != nil {
			return nil, fmt.Errorf("registrando codex: %w", err)
		}
		req.Agente = nombre
	}
	if strings.TrimSpace(req.Agente) == "" {
		return nil, fmt.Errorf("indica el nombre del agente o usa --nuevo-codex")
	}

	var previo *db.Sesion
	if strings.TrimSpace(req.Proyecto) != "" {
		proyecto, err := db.GetProyecto(req.Proyecto)
		if err != nil {
			return nil, err
		}
		previo, err = db.ObtenerUltimaSesion(req.Agente, &proyecto.ID)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
	}

	sesion, err := iniciarSesionDesdeRequest(req)
	if err != nil {
		return nil, err
	}

	agentes, err := db.ListarAgentes()
	if err != nil {
		return nil, err
	}
	rol := ""
	for _, agente := range agentes {
		if agente.Nombre == sesion.Agente {
			rol = agente.Rol
			break
		}
	}

	resp := &apiSesionInicioResponse{
		Sesion:       sesion,
		SesionPrevia: previo,
		Rol:          rol,
	}
	if rol == "" || rol == "admin" {
		return resp, nil
	}

	pendientes, err := db.PropuestasPendientesVoto(sesion.Agente)
	if err != nil {
		return nil, err
	}
	resp.PropuestasPendientes = pendientes

	reglas, err := db.GetReglasAgente(rol)
	if err != nil {
		return nil, err
	}
	resp.Reglas = reglas

	skills, err := db.GetSkillsAgente(rol)
	if err != nil {
		return nil, err
	}
	resp.Skills = skills

	workflow, err := db.GetWorkflow(rol, "inicio-sesion")
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	resp.Workflow = workflow
	return resp, nil
}

func apiDecodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func apiWriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func apiError(w http.ResponseWriter, status int, err error) {
	if err == nil {
		err = fmt.Errorf("error desconocido")
	}
	apiWriteJSON(w, status, apiErrorResponse{Error: err.Error()})
}

func apiMethodNotAllowed(w http.ResponseWriter, methods ...string) {
	if len(methods) > 0 {
		w.Header().Set("Allow", strings.Join(methods, ", "))
	}
	apiError(w, http.StatusMethodNotAllowed, fmt.Errorf("método no permitido"))
}

func apiRequireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		apiMethodNotAllowed(w, method)
		return false
	}
	return true
}

func valorConFallback(v, fallback string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return fallback
	}
	return v
}
