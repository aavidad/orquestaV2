/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"orquesta/agentesapp"
	"orquesta/conectoresapp"
	"orquesta/coordinacion"
	"orquesta/db"
	"orquesta/fabricaapp"
	"orquesta/gitgobernanza"
	"orquesta/lenguajeapp"
	"orquesta/memoriaproyecto"
	"orquesta/progresoapp"
	"orquesta/propuestasapp"
	"orquesta/sesionesapp"
	"orquesta/tareasapp"
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

type apiProyectoOverviewResponse struct {
	Overview *memoriaproyecto.ProjectOverview `json:"overview"`
}

type apiProyectoDecisionCreateRequest struct {
	Categoria    string `json:"categoria"`
	Titulo       string `json:"titulo"`
	Solucion     string `json:"solucion"`
	Motivo       string `json:"motivo"`
	Alternativas string `json:"alternativas"`
	Impacto      string `json:"impacto"`
	Estado       string `json:"estado"`
	PropuestaID  *int64 `json:"propuesta_id"`
	TareaID      *int64 `json:"tarea_id"`
	MetadataJSON string `json:"metadata_json"`
}

type apiProyectoDocumentoCreateRequest struct {
	TipoDocumento string `json:"tipo_documento"`
	Titulo        string `json:"titulo"`
	RutaRef       string `json:"ruta_ref"`
	Resumen       string `json:"resumen"`
	Estado        string `json:"estado"`
	Fuente        string `json:"fuente"`
	PropuestaID   *int64 `json:"propuesta_id"`
	TareaID       *int64 `json:"tarea_id"`
	MetadataJSON  string `json:"metadata_json"`
}

type apiConteoVotosResponse struct {
	Acuerdo    int `json:"acuerdo"`
	Desacuerdo int `json:"desacuerdo"`
	Abstencion int `json:"abstencion"`
	Pendiente  int `json:"pendiente"`
}

func conteoVotosDesdeLista(votos []*db.Voto) apiConteoVotosResponse {
	var out apiConteoVotosResponse
	for _, voto := range votos {
		if voto == nil {
			continue
		}
		switch voto.Posicion {
		case db.VotoAcuerdo:
			out.Acuerdo++
		case db.VotoDesacuerdo:
			out.Desacuerdo++
		case db.VotoAbstencion:
			out.Abstencion++
		default:
			out.Pendiente++
		}
	}
	return out
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

type apiSesionPresupuestoRequest struct {
	SesionID          int64      `json:"sesion_id"`
	Agente            string     `json:"agente"`
	PoolID            *int64     `json:"pool_id"`
	ModelSlug         string     `json:"model_slug"`
	WindowKind        string     `json:"window_kind"`
	WindowStartedAt   *time.Time `json:"window_started_at"`
	ResetAt           *time.Time `json:"reset_at"`
	RemainingSeconds  *int64     `json:"remaining_seconds"`
	RemainingMessages *int64     `json:"remaining_messages"`
	RemainingTokens   *int64     `json:"remaining_tokens"`
	RemainingCredits  *float64   `json:"remaining_credits"`
	BudgetSource      string     `json:"budget_source"`
	RawSnapshotJSON   string     `json:"raw_snapshot_json"`
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

type apiProgresoFaseRegistrarRequest struct {
	Proyecto    string  `json:"proyecto"`
	Nombre      string  `json:"nombre"`
	Descripcion string  `json:"descripcion"`
	Orden       int64   `json:"orden"`
	Peso        float64 `json:"peso"`
	Estado      string  `json:"estado"`
}

type apiProgresoFaseActualizarRequest struct {
	Proyecto    *string  `json:"proyecto,omitempty"`
	Nombre      *string  `json:"nombre,omitempty"`
	Descripcion *string  `json:"descripcion,omitempty"`
	Orden       *int64   `json:"orden,omitempty"`
	Peso        *float64 `json:"peso,omitempty"`
	Estado      *string  `json:"estado,omitempty"`
}

type apiProgresoTareaRegistrarRequest struct {
	Proyecto       string  `json:"proyecto"`
	FaseID         *int64  `json:"fase_id"`
	ProgresoPct    float64 `json:"progreso_pct"`
	ActualizadoPor string  `json:"actualizado_por"`
}

type apiPoolSaveRequest struct {
	Slug                string `json:"slug"`
	Proveedor           string `json:"proveedor"`
	Runtime             string `json:"runtime"`
	Plan                string `json:"plan"`
	EsDePago            bool   `json:"es_de_pago"`
	CapacidadTotal      int    `json:"capacidad_total"`
	CapacidadReservada  int    `json:"capacidad_reservada"`
	PermiteHijos        bool   `json:"permite_hijos"`
	PermiteModelosMulti bool   `json:"permite_modelos_multi"`
	PermiteSobrecoste   bool   `json:"permite_sobrecoste"`
	PoliticaHandoff     string `json:"politica_handoff"`
	FuenteTelemetria    string `json:"fuente_telemetria"`
	MetadataJSON        string `json:"metadata_json"`
	Activo              bool   `json:"activo"`
}

type apiPoolModeloSaveRequest struct {
	ModelSlug          string  `json:"model_slug"`
	Activo             bool    `json:"activo"`
	Prioridad          int     `json:"prioridad"`
	CosteRelativo      float64 `json:"coste_relativo"`
	LimiteConocidoJSON string  `json:"limite_conocido_json"`
}

type apiPoliticaModeloSaveRequest struct {
	ScopeTipo       string `json:"scope_tipo"`
	ScopeRef        string `json:"scope_ref"`
	PerfilTarea     string `json:"perfil_tarea"`
	PoolSlug        string `json:"pool_slug"`
	ModelSlug       string `json:"model_slug"`
	ReasoningEffort string `json:"reasoning_effort"`
	Prioridad       int    `json:"prioridad"`
	Activa          bool   `json:"activa"`
	MetadataJSON    string `json:"metadata_json"`
}

type apiReglaCrearRequest struct {
	Actor       string `json:"actor"`
	TipoAgente  string `json:"tipo_agente"`
	Categoria   string `json:"categoria"`
	Titulo      string `json:"titulo"`
	Descripcion string `json:"descripcion"`
}

type apiSkillCrearRequest struct {
	Actor              string `json:"actor"`
	TipoAgente         string `json:"tipo_agente"`
	Nombre             string `json:"nombre"`
	Descripcion        string `json:"descripcion"`
	CuandoUsar         string `json:"cuando_usar"`
	Escenario          string `json:"escenario"`
	Prioridad          int    `json:"prioridad"`
	AliasesJSON        string `json:"aliases_json"`
	HerramientasJSON   string `json:"herramientas_json"`
	Origen             string `json:"origen"`
	NivelRiesgo        string `json:"nivel_riesgo"`
	RequiereAprobacion bool   `json:"requiere_aprobacion"`
}

type apiWorkflowCrearRequest struct {
	Actor       string `json:"actor"`
	TipoAgente  string `json:"tipo_agente"`
	Nombre      string `json:"nombre"`
	Descripcion string `json:"descripcion"`
	PasosJSON   string `json:"pasos_json"`
}

type apiPermisoCatalogoSetRequest struct {
	Actor          string `json:"actor"`
	Entidad        string `json:"entidad"`
	Rol            string `json:"rol"`
	Alcance        string `json:"alcance"`
	PuedeCrear     bool   `json:"puede_crear"`
	PuedeEditar    bool   `json:"puede_editar"`
	PuedeActivar   bool   `json:"puede_activar"`
	PuedeVersionar bool   `json:"puede_versionar"`
}

type apiCatalogoActivacionRequest struct {
	Actor  string `json:"actor"`
	Activa bool   `json:"activa"`
}

type apiReglaActualizarRequest struct {
	Actor       string `json:"actor"`
	TipoAgente  string `json:"tipo_agente"`
	Categoria   string `json:"categoria"`
	Titulo      string `json:"titulo"`
	Descripcion string `json:"descripcion"`
	Activa      bool   `json:"activa"`
}

type apiSkillActualizarRequest struct {
	Actor              string `json:"actor"`
	TipoAgente         string `json:"tipo_agente"`
	Nombre             string `json:"nombre"`
	Descripcion        string `json:"descripcion"`
	CuandoUsar         string `json:"cuando_usar"`
	Escenario          string `json:"escenario"`
	Prioridad          int    `json:"prioridad"`
	AliasesJSON        string `json:"aliases_json"`
	HerramientasJSON   string `json:"herramientas_json"`
	Origen             string `json:"origen"`
	NivelRiesgo        string `json:"nivel_riesgo"`
	RequiereAprobacion bool   `json:"requiere_aprobacion"`
	Activa             bool   `json:"activa"`
}

type apiWorkflowActualizarRequest struct {
	Actor       string `json:"actor"`
	TipoAgente  string `json:"tipo_agente"`
	Nombre      string `json:"nombre"`
	Descripcion string `json:"descripcion"`
	PasosJSON   string `json:"pasos_json"`
	Activo      bool   `json:"activo"`
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
	Accion       string  `json:"accion"`
	Agente       string  `json:"agente"`
	Posicion     string  `json:"posicion"`
	Comentario   string  `json:"comentario"`
	EstadoCierre string  `json:"estado_cierre"`
	Titulo       *string `json:"titulo"`
	Descripcion  *string `json:"descripcion"`
	AnexarDesc   *string `json:"anexar_descripcion"`
	Tipo         *string `json:"tipo"`
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

type apiRuntimeCheckpointCreateRequest struct {
	Agente         string `json:"agente"`
	Proyecto       string `json:"proyecto"`
	SesionID       int64  `json:"sesion_id"`
	RuntimeID      int64  `json:"runtime_id"`
	CheckpointKind string `json:"checkpoint_kind"`
	Resumen        string `json:"resumen"`
	Branch         string `json:"branch"`
	CWD            string `json:"cwd"`
	Payload        string `json:"payload"`
	ResumeStrategy string `json:"resume_strategy"`
	Source         string `json:"source"`
}

type apiRefineriaRequest struct {
	Agente  string `json:"agente"`
	Rama    string `json:"rama"`
	Dir     string `json:"dir"`
	CmdTest string `json:"cmd_test"`
}

type apiMemoriaEntidadRequest struct {
	Nombre        string `json:"nombre"`
	Tipo          string `json:"tipo"`
	Valor         string `json:"valor"`
	Metadata      string `json:"metadata"`
	VerificadoPor string `json:"verificado_por"`
	Proyecto      string `json:"proyecto"`
}

type apiAgentePausarRequest struct {
	Agente  string `json:"agente"`
	Minutos int    `json:"minutos"`
	Motivo  string `json:"motivo"`
	Accion  string `json:"accion"`
	Entidad string `json:"entidad"`
	Detalle string `json:"detalle"`
}

type apiAgenteFusionRequest struct {
	Destino          string `json:"destino"`
	DestinoRespaldo  string `json:"destino_respaldo"`
	EtiquetaRespaldo string `json:"etiqueta_respaldo"`
	Retener          int    `json:"retener"`
}

type apiAgenteFusionResponse struct {
	OK           bool                       `json:"ok"`
	Origen       string                     `json:"origen"`
	Destino      string                     `json:"destino"`
	RutaRespaldo string                     `json:"ruta_respaldo"`
	Resultado    *db.FusionAgentesResultado `json:"resultado"`
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
	mux.HandleFunc("/api/reglas", apiHandlerReglas)
	mux.HandleFunc("/api/reglas/", apiRouterReglas)
	mux.HandleFunc("/api/skills", apiHandlerSkills)
	mux.HandleFunc("/api/skills/remotas", apiHandlerSkillsRemotas)
	mux.HandleFunc("/api/skills/importar", apiHandlerSkillsImportar)
	mux.HandleFunc("/api/skills/detectar-carencia", apiHandlerSkillsDetectarCarencia)
	mux.HandleFunc("/api/skills/", apiRouterSkills)
	mux.HandleFunc("/api/deploy/docker-remoto/plan", apiHandlerDeployDockerRemotePlan)
	mux.HandleFunc("/api/deploy/docker-remoto/ejecutar", apiHandlerDeployDockerRemoteExecute)
	mux.HandleFunc("/api/workflows", apiHandlerWorkflows)
	mux.HandleFunc("/api/workflows/", apiRouterWorkflows)
	mux.HandleFunc("/api/permisos-catalogo", apiHandlerPermisosCatalogo)
	mux.HandleFunc("/api/lenguaje/politica", apiHandlerLenguajePolitica)
	mux.HandleFunc("/api/lenguaje/matriz", apiHandlerLenguajeMatriz)
	mux.HandleFunc("/api/lenguaje/matriz/borrar", apiHandlerLenguajeMatrizBorrar)
	mux.HandleFunc("/api/lenguaje/resolver", apiHandlerLenguajeResolver)
	mux.HandleFunc("/api/config", apiHandlerConfig)
	mux.HandleFunc("/api/audit", apiHandlerAudit)
	mux.HandleFunc("/api/respaldo/bd", apiHandlerRespaldoBD)
	mux.HandleFunc("/api/proyectos", apiHandlerProyectos)
	mux.HandleFunc("/api/proyectos/descubrir", apiHandlerProyectoDescubrir)
	mux.HandleFunc("/api/proyectos/", apiRouterProyectos)
	mux.HandleFunc("/api/pools", apiHandlerPools)
	mux.HandleFunc("/api/pools/seed-inicial", apiHandlerPoolsSeedInicial)
	mux.HandleFunc("/api/pools/modelos/seed-inicial", apiHandlerPoolsModelosSeedInicial)
	mux.HandleFunc("/api/pools/", apiRouterPools)
	mux.HandleFunc("/api/politicas-modelo", apiHandlerPoliticasModelo)
	mux.HandleFunc("/api/politicas-modelo/seed-inicial", apiHandlerPoliticasModeloSeedInicial)
	mux.HandleFunc("/api/modelo/resolver", apiHandlerModeloResolver)
	mux.HandleFunc("/api/progreso", apiHandlerProgreso)
	mux.HandleFunc("/api/progreso/fases", apiHandlerProgresoFases)
	mux.HandleFunc("/api/progreso/fases/", apiRouterProgresoFases)
	mux.HandleFunc("/api/progreso/tareas/", apiRouterProgresoTareas)
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
	mux.HandleFunc("/api/git/merges", apiHandlerGitMerges)
	mux.HandleFunc("/api/runtimes", apiHandlerRuntimes)
	mux.HandleFunc("/api/runtimes/tree", apiHandlerRuntimesTree)
	mux.HandleFunc("/api/runtimes/", apiRouterRuntimes)
	mux.HandleFunc("/api/runtime-handles", apiHandlerRuntimeHandles)
	mux.HandleFunc("/api/runtime-orders", apiHandlerRuntimeOrders)
	mux.HandleFunc("/api/runtime-mailbox", apiHandlerRuntimeMailbox)
	mux.HandleFunc("/api/runtime-mailbox/", apiRouterRuntimeMailbox)
	mux.HandleFunc("/api/runtime-checkpoints", apiHandlerRuntimeCheckpoints)
	mux.HandleFunc("/api/runtime-checkpoints/latest", apiHandlerRuntimeCheckpointLatest)
	mux.HandleFunc("/api/runtime-checkpoints/", apiRouterRuntimeCheckpoints)
	mux.HandleFunc("/api/refineria", apiHandlerRefineria)
	mux.HandleFunc("/api/refineria/", apiRouterRefineria)
	mux.HandleFunc("/api/memoria", apiHandlerMemoria)
	mux.HandleFunc("/api/memoria/", apiRouterMemoria)
	mux.HandleFunc("/api/sesiones", apiHandlerSesiones)
	mux.HandleFunc("/api/sesiones/", apiRouterSesiones)
	mux.HandleFunc("/api/sesiones/inicio", apiHandlerSesionInicio)
	mux.HandleFunc("/api/sesiones/guardar", apiHandlerSesionGuardar)
	mux.HandleFunc("/api/sesiones/fin", apiHandlerSesionFin)
	mux.HandleFunc("/api/sesiones/continuar", apiHandlerSesionContinuar)
	mux.HandleFunc("/api/sesiones/presupuesto", apiHandlerSesionPresupuesto)
	mux.HandleFunc("/api/agente/handoff", apiHandlerAgenteHandoff)
	mux.HandleFunc("/api/agente/control", apiHandlerAgenteControl)
	mux.HandleFunc("/api/agente/preparar", apiHandlerAgentePreparar)
	mux.HandleFunc("/api/agente/tick", apiHandlerAgenteTick)
	mux.HandleFunc("/api/agente/pausar", apiHandlerAgentePausar)
}

func apiHandlerStatus(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	status, err := statusService.FetchStatus()
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, status)
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
		if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("vista")), "panel") {
			rows, err := agentesService.BuildPanelRows()
			if err != nil {
				apiError(w, http.StatusInternalServerError, err)
				return
			}
			apiWriteJSON(w, http.StatusOK, apiAgentesPanelResponse{Rows: rows})
			return
		}
		agentes, err := agentesService.ListAgents()
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
		nombre := strings.TrimSpace(req.Nombre)
		rol := strings.TrimSpace(req.Rol)
		if rol == "" {
			rol = "programador"
		}
		if nombre == "" {
			nombreAuto, err := agentesService.RegisterAgentAuto(strings.TrimSpace(req.Proveedor), rol)
			if err != nil {
				apiError(w, http.StatusBadRequest, err)
				return
			}
			nombre = nombreAuto
		} else if err := agentesService.RegisterAgent(nombre, rol); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		apiWriteJSON(w, http.StatusCreated, map[string]any{"ok": true, "nombre": nombre, "rol": rol})
	default:
		apiMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func apiRouterAgentes(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/agentes/"), "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	if r.Method == http.MethodGet {
		if len(parts) == 2 && parts[1] == "overview" {
			detail, err := agentesService.BuildDetail(parts[0])
			if err != nil {
				apiError(w, http.StatusNotFound, err)
				return
			}
			apiWriteJSON(w, http.StatusOK, apiAgenteOverviewResponse{Detail: detail})
			return
		}
		http.NotFound(w, r)
		return
	}
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	nombre := parts[0]
	switch parts[1] {
	case "retirar":
		if err := agentesService.ApplyStateAction(nombre, "retirar"); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
	case "rehabilitar":
		if err := agentesService.ApplyStateAction(nombre, "rehabilitar"); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
	case "eliminar":
		var req struct {
			Actor string `json:"actor"`
		}
		if r.ContentLength > 0 {
			if err := apiDecodeJSON(r, &req); err != nil {
				apiError(w, http.StatusBadRequest, err)
				return
			}
		}
		if err := agentesService.DeleteAgent(nombre, req.Actor); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
	case "fusionar":
		var req apiAgenteFusionRequest
		if err := apiDecodeJSON(r, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		req.Destino = strings.TrimSpace(req.Destino)
		if req.Destino == "" {
			apiError(w, http.StatusBadRequest, fmt.Errorf("debes indicar el agente destino"))
			return
		}
		etiqueta := strings.TrimSpace(req.EtiquetaRespaldo)
		if etiqueta == "" {
			etiqueta = fmt.Sprintf("fusion_%s_%s", strings.ToLower(nombre), strings.ToLower(req.Destino))
		}
		rutaRespaldo, err := ejecutarRespaldoBD(req.DestinoRespaldo, etiqueta, req.Retener)
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		resultado, err := agentesService.MergeAgents(nombre, req.Destino)
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, apiAgenteFusionResponse{
			OK:           true,
			Origen:       nombre,
			Destino:      req.Destino,
			RutaRespaldo: rutaRespaldo,
			Resultado:    resultado,
		})
		return
	case "reset-reanimacion":
		if err := agentesService.ApplyStateAction(nombre, "reset-reanimacion"); err != nil {
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
			valor, err := configService.Get(clave)
			if err != nil {
				apiError(w, http.StatusNotFound, err)
				return
			}
			apiWriteJSON(w, http.StatusOK, map[string]any{"clave": clave, "valor": valor})
			return
		}
		config, err := configService.All()
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
		if err := configService.Set(req.Clave, req.Valor); err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{"ok": true, "clave": req.Clave, "valor": req.Valor})
	default:
		apiMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func apiResolverTipoAgente(r *http.Request) (string, error) {
	if tipo := strings.TrimSpace(r.URL.Query().Get("rol")); tipo != "" {
		return tipo, nil
	}
	if tipo := strings.TrimSpace(r.URL.Query().Get("tipo_agente")); tipo != "" {
		return tipo, nil
	}
	if nombre := strings.TrimSpace(r.URL.Query().Get("agente")); nombre != "" {
		agente, err := agentesService.GetAgent(nombre)
		if err != nil {
			return "", fmt.Errorf("agente '%s' no encontrado", nombre)
		}
		return strings.TrimSpace(agente.Rol), nil
	}
	return "", fmt.Errorf("debes indicar tipo_agente o agente")
}

func apiBoolOpcional(r *http.Request, key string) (*bool, error) {
	v := strings.TrimSpace(strings.ToLower(r.URL.Query().Get(key)))
	if v == "" {
		return nil, nil
	}
	switch v {
	case "true", "1", "si", "sí", "yes":
		b := true
		return &b, nil
	case "false", "0", "no":
		b := false
		return &b, nil
	default:
		return nil, fmt.Errorf("%s inválida", key)
	}
}

func apiHandlerReglas(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		activa, err := apiBoolOpcional(r, "activa")
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		tipoAgente, err := apiResolverTipoAgente(r)
		if err != nil {
			tipoAgente = ""
		}
		reglas, err := db.ListarReglas(tipoAgente, activa)
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{
			"tipo_agente": tipoAgente,
			"reglas":      reglas,
		})
	case http.MethodPost:
		var req apiReglaCrearRequest
		if err := apiDecodeJSON(r, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		id, err := db.CrearRegla(strings.TrimSpace(req.Actor), &db.Regla{
			TipoAgente:  strings.TrimSpace(req.TipoAgente),
			Categoria:   strings.TrimSpace(req.Categoria),
			Titulo:      strings.TrimSpace(req.Titulo),
			Descripcion: strings.TrimSpace(req.Descripcion),
		})
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		apiWriteJSON(w, http.StatusCreated, apiCatalogoMutationResponse{ID: id})
	default:
		apiMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func apiRouterReglas(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/reglas/"), "/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || id <= 0 {
		apiError(w, http.StatusBadRequest, fmt.Errorf("id inválido"))
		return
	}
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			regla, err := db.GetRegla(id)
			if err != nil {
				apiError(w, http.StatusNotFound, err)
				return
			}
			apiWriteJSON(w, http.StatusOK, apiReglaResponse{Regla: regla})
		case http.MethodPost:
			var req apiReglaActualizarRequest
			if err := apiDecodeJSON(r, &req); err != nil {
				apiError(w, http.StatusBadRequest, err)
				return
			}
			if err := db.ActualizarRegla(strings.TrimSpace(req.Actor), &db.Regla{
				ID:          id,
				TipoAgente:  strings.TrimSpace(req.TipoAgente),
				Categoria:   strings.TrimSpace(req.Categoria),
				Titulo:      strings.TrimSpace(req.Titulo),
				Descripcion: strings.TrimSpace(req.Descripcion),
				Activa:      req.Activa,
			}); err != nil {
				apiError(w, http.StatusBadRequest, err)
				return
			}
			apiWriteJSON(w, http.StatusOK, apiCatalogoMutationResponse{ID: id})
		default:
			apiMethodNotAllowed(w, http.MethodGet, http.MethodPost)
		}
		return
	}
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	switch parts[1] {
	case "activa":
		if !apiRequireMethod(w, r, http.MethodPost) {
			return
		}
		var req apiCatalogoActivacionRequest
		if err := apiDecodeJSON(r, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		if err := db.SetReglaActiva(strings.TrimSpace(req.Actor), id, req.Activa); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{"ok": true, "id": id})
	case "versiones":
		if !apiRequireMethod(w, r, http.MethodGet) {
			return
		}
		versiones, err := db.ListarVersionesRegla(id)
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, apiReglaVersionesResponse{Versiones: versiones})
	default:
		http.NotFound(w, r)
	}
}

func apiHandlerSkills(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		activa, err := apiBoolOpcional(r, "activa")
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		tipoAgente, err := apiResolverTipoAgente(r)
		if err != nil {
			tipoAgente = ""
		}
		skills, err := db.ListarSkills(tipoAgente, activa)
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{
			"tipo_agente": tipoAgente,
			"skills":      skills,
		})
	case http.MethodPost:
		var req apiSkillCrearRequest
		if err := apiDecodeJSON(r, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		id, err := db.CrearSkill(strings.TrimSpace(req.Actor), &db.Skill{
			TipoAgente:         strings.TrimSpace(req.TipoAgente),
			Nombre:             strings.TrimSpace(req.Nombre),
			Descripcion:        strings.TrimSpace(req.Descripcion),
			CuandoUsar:         strings.TrimSpace(req.CuandoUsar),
			Escenario:          strings.TrimSpace(req.Escenario),
			Prioridad:          req.Prioridad,
			AliasesJSON:        strings.TrimSpace(req.AliasesJSON),
			HerramientasJSON:   strings.TrimSpace(req.HerramientasJSON),
			Origen:             strings.TrimSpace(req.Origen),
			NivelRiesgo:        strings.TrimSpace(req.NivelRiesgo),
			RequiereAprobacion: req.RequiereAprobacion,
			Activa:             true,
		})
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		apiWriteJSON(w, http.StatusCreated, apiCatalogoMutationResponse{ID: id})
	default:
		apiMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func apiRouterSkills(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/skills/"), "/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || id <= 0 {
		apiError(w, http.StatusBadRequest, fmt.Errorf("id inválido"))
		return
	}
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			skill, err := db.GetSkill(id)
			if err != nil {
				apiError(w, http.StatusNotFound, err)
				return
			}
			apiWriteJSON(w, http.StatusOK, apiSkillResponse{Skill: skill})
		case http.MethodPost:
			var req apiSkillActualizarRequest
			if err := apiDecodeJSON(r, &req); err != nil {
				apiError(w, http.StatusBadRequest, err)
				return
			}
			if err := db.ActualizarSkill(strings.TrimSpace(req.Actor), &db.Skill{
				ID:                 id,
				TipoAgente:         strings.TrimSpace(req.TipoAgente),
				Nombre:             strings.TrimSpace(req.Nombre),
				Descripcion:        strings.TrimSpace(req.Descripcion),
				CuandoUsar:         strings.TrimSpace(req.CuandoUsar),
				Escenario:          strings.TrimSpace(req.Escenario),
				Prioridad:          req.Prioridad,
				AliasesJSON:        strings.TrimSpace(req.AliasesJSON),
				HerramientasJSON:   strings.TrimSpace(req.HerramientasJSON),
				Origen:             strings.TrimSpace(req.Origen),
				NivelRiesgo:        strings.TrimSpace(req.NivelRiesgo),
				RequiereAprobacion: req.RequiereAprobacion,
				Activa:             req.Activa,
			}); err != nil {
				apiError(w, http.StatusBadRequest, err)
				return
			}
			apiWriteJSON(w, http.StatusOK, apiCatalogoMutationResponse{ID: id})
		default:
			apiMethodNotAllowed(w, http.MethodGet, http.MethodPost)
		}
		return
	}
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	switch parts[1] {
	case "activa":
		if !apiRequireMethod(w, r, http.MethodPost) {
			return
		}
		var req apiCatalogoActivacionRequest
		if err := apiDecodeJSON(r, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		if err := db.SetSkillActivo(strings.TrimSpace(req.Actor), id, req.Activa); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{"ok": true, "id": id})
	case "borrar":
		if !apiRequireMethod(w, r, http.MethodPost) {
			return
		}
		var req apiSkillBorrarRequest
		if err := apiDecodeJSON(r, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		if err := db.EliminarSkill(strings.TrimSpace(req.Actor), id); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{"ok": true, "id": id})
	case "versiones":
		if !apiRequireMethod(w, r, http.MethodGet) {
			return
		}
		versiones, err := db.ListarVersionesSkill(id)
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, apiSkillVersionesResponse{Versiones: versiones})
	default:
		http.NotFound(w, r)
	}
}

func apiHandlerWorkflows(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if nombre := strings.TrimSpace(r.URL.Query().Get("nombre")); nombre != "" {
			tipoAgente, err := apiResolverTipoAgente(r)
			if err != nil {
				apiError(w, http.StatusBadRequest, err)
				return
			}
			workflow, err := db.GetWorkflow(tipoAgente, nombre)
			if err != nil {
				apiError(w, http.StatusNotFound, err)
				return
			}
			apiWriteJSON(w, http.StatusOK, map[string]any{
				"tipo_agente": tipoAgente,
				"workflow":    workflow,
			})
			return
		}
		activo, err := apiBoolOpcional(r, "activa")
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		tipoAgente, err := apiResolverTipoAgente(r)
		if err != nil {
			tipoAgente = ""
		}
		workflows, err := db.ListarWorkflows(tipoAgente, activo)
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{
			"tipo_agente": tipoAgente,
			"workflows":   workflows,
		})
	case http.MethodPost:
		var req apiWorkflowCrearRequest
		if err := apiDecodeJSON(r, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		id, err := db.CrearWorkflow(strings.TrimSpace(req.Actor), &db.Workflow{
			TipoAgente:  strings.TrimSpace(req.TipoAgente),
			Nombre:      strings.TrimSpace(req.Nombre),
			Descripcion: strings.TrimSpace(req.Descripcion),
			Pasos:       strings.TrimSpace(req.PasosJSON),
		})
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		apiWriteJSON(w, http.StatusCreated, apiCatalogoMutationResponse{ID: id})
	default:
		apiMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func apiRouterWorkflows(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/workflows/"), "/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || id <= 0 {
		apiError(w, http.StatusBadRequest, fmt.Errorf("id inválido"))
		return
	}
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			workflow, err := db.GetWorkflowByID(id)
			if err != nil {
				apiError(w, http.StatusNotFound, err)
				return
			}
			apiWriteJSON(w, http.StatusOK, apiWorkflowResponse{Workflow: workflow})
		case http.MethodPost:
			var req apiWorkflowActualizarRequest
			if err := apiDecodeJSON(r, &req); err != nil {
				apiError(w, http.StatusBadRequest, err)
				return
			}
			if err := db.ActualizarWorkflow(strings.TrimSpace(req.Actor), &db.Workflow{
				ID:          id,
				TipoAgente:  strings.TrimSpace(req.TipoAgente),
				Nombre:      strings.TrimSpace(req.Nombre),
				Descripcion: strings.TrimSpace(req.Descripcion),
				Pasos:       strings.TrimSpace(req.PasosJSON),
				Activo:      req.Activo,
			}); err != nil {
				apiError(w, http.StatusBadRequest, err)
				return
			}
			apiWriteJSON(w, http.StatusOK, apiCatalogoMutationResponse{ID: id})
		default:
			apiMethodNotAllowed(w, http.MethodGet, http.MethodPost)
		}
		return
	}
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	switch parts[1] {
	case "activa":
		if !apiRequireMethod(w, r, http.MethodPost) {
			return
		}
		var req apiCatalogoActivacionRequest
		if err := apiDecodeJSON(r, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		if err := db.SetWorkflowActivo(strings.TrimSpace(req.Actor), id, req.Activa); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{"ok": true, "id": id})
	case "versiones":
		if !apiRequireMethod(w, r, http.MethodGet) {
			return
		}
		versiones, err := db.ListarVersionesWorkflow(id)
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, apiWorkflowVersionesResponse{Versiones: versiones})
	default:
		http.NotFound(w, r)
	}
}

func apiHandlerPermisosCatalogo(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		entidad := strings.TrimSpace(r.URL.Query().Get("entidad"))
		permisos, err := db.ListarPermisosEdicionCatalogo(entidad)
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{"permisos": permisos})
	case http.MethodPost:
		var req apiPermisoCatalogoSetRequest
		if err := apiDecodeJSON(r, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		if err := db.GuardarPermisoEdicionCatalogo(strings.TrimSpace(req.Actor), &db.PermisoEdicionCatalogo{
			Entidad:        strings.TrimSpace(req.Entidad),
			Rol:            strings.TrimSpace(req.Rol),
			Alcance:        strings.TrimSpace(req.Alcance),
			PuedeCrear:     req.PuedeCrear,
			PuedeEditar:    req.PuedeEditar,
			PuedeActivar:   req.PuedeActivar,
			PuedeVersionar: req.PuedeVersionar,
		}); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{"ok": true})
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
	var activoPtr *bool
	if activaRaw := strings.TrimSpace(r.URL.Query().Get("activa")); activaRaw != "" {
		activa := activaRaw == "1" || strings.EqualFold(activaRaw, "true")
		activoPtr = &activa
	}
	proyectos, err := db.ListarProyectos(db.FiltroProyectos{Activo: activoPtr})
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
			_ = configService.Set("workspace_root", abs)
		}
	}
	apiWriteJSON(w, http.StatusCreated, map[string]any{"proyectos": proyectos})
}

func apiRouterProyectos(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/proyectos/"), "/")
	if path == "" {
		http.NotFound(w, r)
		return
	}
	parts := strings.Split(path, "/")
	ref := strings.TrimSpace(parts[0])
	switch {
	case len(parts) == 1 && r.Method == http.MethodGet:
		proyecto, err := db.GetProyecto(ref)
		if err != nil {
			apiError(w, http.StatusNotFound, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{"proyecto": proyecto})
	case len(parts) == 2 && parts[1] == "overview" && r.Method == http.MethodGet:
		apiHandlerProyectoOverview(w, r, ref)
	case len(parts) == 2 && parts[1] == "decisiones" && r.Method == http.MethodPost:
		apiHandlerProyectoDecisionNueva(w, r, ref)
	case len(parts) == 2 && parts[1] == "documentacion" && r.Method == http.MethodPost:
		apiHandlerProyectoDocumentoNuevo(w, r, ref)
	case len(parts) == 2 && parts[1] == "operacion" && r.Method == http.MethodGet:
		apiHandlerProyectoOperacion(w, r, ref)
	case len(parts) == 2 && parts[1] == "operacion" && r.Method == http.MethodPost:
		apiHandlerProyectoOperacionGuardar(w, r, ref)
	case len(parts) == 2 && parts[1] == "fabricar-app" && r.Method == http.MethodPost:
		apiHandlerProyectoFabricarApp(w, r, ref)
	default:
		http.NotFound(w, r)
	}
}

func apiHandlerProyectoOverview(w http.ResponseWriter, r *http.Request, ref string) {
	svc := memoriaproyecto.NewService(db.ProjectMemoryRepository{})
	overview, err := svc.Overview(strings.TrimSpace(ref))
	if err != nil {
		apiError(w, http.StatusNotFound, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiProyectoOverviewResponse{Overview: overview})
}

func apiHandlerProyectoDecisionNueva(w http.ResponseWriter, r *http.Request, ref string) {
	var req apiProyectoDecisionCreateRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	svc := memoriaproyecto.NewService(db.ProjectMemoryRepository{})
	id, err := svc.CreateDecision(memoriaproyecto.CreateDecisionInput{
		ProyectoSlug: strings.TrimSpace(ref),
		Categoria:    strings.TrimSpace(req.Categoria),
		Titulo:       strings.TrimSpace(req.Titulo),
		Solucion:     strings.TrimSpace(req.Solucion),
		Motivo:       strings.TrimSpace(req.Motivo),
		Alternativas: strings.TrimSpace(req.Alternativas),
		Impacto:      strings.TrimSpace(req.Impacto),
		Estado:       strings.TrimSpace(req.Estado),
		PropuestaID:  req.PropuestaID,
		TareaID:      req.TareaID,
		MetadataJSON: strings.TrimSpace(req.MetadataJSON),
	})
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusCreated, apiCatalogoMutationResponse{ID: id})
}

func apiHandlerProyectoDocumentoNuevo(w http.ResponseWriter, r *http.Request, ref string) {
	var req apiProyectoDocumentoCreateRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	svc := memoriaproyecto.NewService(db.ProjectMemoryRepository{})
	id, err := svc.CreateExternalDoc(memoriaproyecto.CreateExternalDocInput{
		ProyectoSlug:  strings.TrimSpace(ref),
		TipoDocumento: strings.TrimSpace(req.TipoDocumento),
		Titulo:        strings.TrimSpace(req.Titulo),
		RutaRef:       strings.TrimSpace(req.RutaRef),
		Resumen:       strings.TrimSpace(req.Resumen),
		Estado:        strings.TrimSpace(req.Estado),
		Fuente:        strings.TrimSpace(req.Fuente),
		PropuestaID:   req.PropuestaID,
		TareaID:       req.TareaID,
		MetadataJSON:  strings.TrimSpace(req.MetadataJSON),
	})
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusCreated, apiCatalogoMutationResponse{ID: id})
}

func apiHandlerProyectoFabricarApp(w http.ResponseWriter, r *http.Request, ref string) {
	proyecto, err := db.GetProyecto(strings.TrimSpace(ref))
	if err != nil {
		apiError(w, http.StatusNotFound, err)
		return
	}
	var req apiProyectoFabricarAppRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(req.Tipo) == "" {
		apiError(w, http.StatusBadRequest, fmt.Errorf("--tipo es obligatorio"))
		return
	}
	nombre := strings.TrimSpace(req.Nombre)
	if nombre == "" {
		nombre = strings.TrimSpace(proyecto.Nombre)
	}
	descripcion := strings.TrimSpace(req.Descripcion)
	if descripcion == "" {
		descripcion = "Backlog inicial de " + nombre + " generado por la fabrica de apps de Orquesta."
	}
	actor := strings.TrimSpace(req.Por)
	if actor == "" {
		actor = "alberto"
	}
	plan, err := newProjectAppFactory().Generate(fabricaapp.AppSpec{
		Nombre:      nombre,
		Descripcion: descripcion,
		Tipo:        strings.TrimSpace(req.Tipo),
		Frontend:    req.Frontend,
		API:         req.API,
		Auth:        req.Auth,
		Database:    req.Database,
		Docker:      req.Docker,
		I18n:        req.I18n,
		Idiomas:     req.Idiomas,
	})
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	result, err := fabricaapp.Materialize(fabricaapp.DBStore{}, proyecto.ID, actor, plan)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusCreated, apiProyectoFabricarAppResponse{
		OK:      true,
		Slug:    proyecto.Slug,
		Tipo:    strings.TrimSpace(req.Tipo),
		Created: result.Created,
		Backlog: result.Backlog,
	})
}

func apiHandlerProyectoOperacion(w http.ResponseWriter, r *http.Request, ref string) {
	proyecto, err := db.GetProyecto(strings.TrimSpace(ref))
	if err != nil {
		apiError(w, http.StatusNotFound, err)
		return
	}
	operacion, err := db.GetProyectoOperacion(proyecto.ID)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiProyectoOperacionResponse{Operacion: operacion})
}

func apiHandlerProyectoOperacionGuardar(w http.ResponseWriter, r *http.Request, ref string) {
	proyecto, err := db.GetProyecto(strings.TrimSpace(ref))
	if err != nil {
		apiError(w, http.StatusNotFound, err)
		return
	}
	var req apiProyectoOperacionSetRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	op, err := db.GetProyectoOperacion(proyecto.ID)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	op.EstadoOperativo = db.EstadoOperativoProyecto(strings.TrimSpace(req.EstadoOperativo))
	op.Motivo = strings.TrimSpace(req.Motivo)
	op.ObjetivoPct = req.ObjetivoPct
	op.MinAgentes = req.MinAgentes
	op.MaxAgentes = req.MaxAgentes
	op.Prioridad = req.Prioridad
	op.ResumeAutomatico = req.ResumeAutomatico
	if err := db.UpsertProyectoOperacion(op); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiProyectoOperacionResponse{Operacion: op})
}

func apiHandlerPools(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		var activoPtr *bool
		if activoRaw := strings.TrimSpace(r.URL.Query().Get("activo")); activoRaw != "" {
			activo := activoRaw == "1" || strings.EqualFold(activoRaw, "true")
			activoPtr = &activo
		}
		pools, err := capacidadService.ListPoolsSummary(activoPtr)
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, apiPoolsResponse{Pools: pools})
	case http.MethodPost:
		var req apiPoolSaveRequest
		if err := apiDecodeJSON(r, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		id, err := capacidadService.SavePool(&db.PoolCapacidad{
			Slug:                strings.TrimSpace(req.Slug),
			Proveedor:           strings.TrimSpace(req.Proveedor),
			Runtime:             strings.TrimSpace(req.Runtime),
			Plan:                strings.TrimSpace(req.Plan),
			EsDePago:            req.EsDePago,
			CapacidadTotal:      req.CapacidadTotal,
			CapacidadReservada:  req.CapacidadReservada,
			PermiteHijos:        req.PermiteHijos,
			PermiteModelosMulti: req.PermiteModelosMulti,
			PermiteSobrecoste:   req.PermiteSobrecoste,
			PoliticaHandoff:     strings.TrimSpace(req.PoliticaHandoff),
			FuenteTelemetria:    strings.TrimSpace(req.FuenteTelemetria),
			MetadataJSON:        strings.TrimSpace(req.MetadataJSON),
			Activo:              req.Activo,
		})
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		apiWriteJSON(w, http.StatusCreated, apiPoolSaveResponse{ID: id})
	default:
		apiMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func apiRouterPools(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/pools/"), "/")
	if path == "" {
		http.NotFound(w, r)
		return
	}
	if strings.HasSuffix(path, "/modelos") {
		slug := strings.Trim(strings.TrimSuffix(path, "/modelos"), "/")
		if slug == "" {
			http.NotFound(w, r)
			return
		}
		apiHandlerPoolModelos(w, r, slug)
		return
	}
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	detalle, err := capacidadService.GetPoolDetail(path)
	if err != nil {
		apiError(w, http.StatusNotFound, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiPoolResponse{Detalle: detalle})
}

func apiHandlerPoolModelos(w http.ResponseWriter, r *http.Request, slug string) {
	switch r.Method {
	case http.MethodGet:
		modelos, err := capacidadService.ListPoolModels(slug)
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, apiPoolModelosResponse{Modelos: modelos})
	case http.MethodPost:
		var req apiPoolModeloSaveRequest
		if err := apiDecodeJSON(r, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		id, err := capacidadService.SavePoolModel(slug, &db.PoolModelo{
			ModelSlug:          strings.TrimSpace(req.ModelSlug),
			Activo:             req.Activo,
			Prioridad:          req.Prioridad,
			CosteRelativo:      req.CosteRelativo,
			LimiteConocidoJSON: strings.TrimSpace(req.LimiteConocidoJSON),
		})
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		apiWriteJSON(w, http.StatusCreated, apiPoolSaveResponse{ID: id})
	default:
		apiMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func apiHandlerPoolsSeedInicial(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	seeds := []db.PoolCapacidad{
		{
			Slug:                "codex",
			Proveedor:           "OpenAI",
			Runtime:             "codex",
			Plan:                "default",
			EsDePago:            true,
			CapacidadTotal:      4,
			CapacidadReservada:  0,
			PermiteHijos:        true,
			PermiteModelosMulti: true,
			PermiteSobrecoste:   false,
			PoliticaHandoff:     "preventivo",
			FuenteTelemetria:    "manual",
			MetadataJSON:        "{}",
			Activo:              true,
		},
		{
			Slug:                "claude",
			Proveedor:           "Anthropic",
			Runtime:             "claude",
			Plan:                "default",
			EsDePago:            true,
			CapacidadTotal:      1,
			CapacidadReservada:  0,
			PermiteHijos:        true,
			PermiteModelosMulti: true,
			PermiteSobrecoste:   false,
			PoliticaHandoff:     "preventivo",
			FuenteTelemetria:    "manual",
			MetadataJSON:        "{}",
			Activo:              true,
		},
		{
			Slug:                "android",
			Proveedor:           "Android",
			Runtime:             "android",
			Plan:                "default",
			EsDePago:            false,
			CapacidadTotal:      1,
			CapacidadReservada:  0,
			PermiteHijos:        false,
			PermiteModelosMulti: false,
			PermiteSobrecoste:   false,
			PoliticaHandoff:     "preventivo",
			FuenteTelemetria:    "manual",
			MetadataJSON:        "{}",
			Activo:              true,
		},
	}
	for _, seed := range seeds {
		if _, err := capacidadService.SavePool(&seed); err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"ok": true, "count": len(seeds)})
}

func apiHandlerPoolsModelosSeedInicial(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	if err := capacidadService.SeedInitialModels(); err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func apiHandlerPoliticasModelo(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		scopeTipo := strings.TrimSpace(r.URL.Query().Get("scope_tipo"))
		scopeRef := strings.TrimSpace(r.URL.Query().Get("scope_ref"))
		var activaPtr *bool
		if activaRaw := strings.TrimSpace(r.URL.Query().Get("activa")); activaRaw != "" {
			activa := activaRaw == "1" || strings.EqualFold(activaRaw, "true")
			activaPtr = &activa
		}
		politicas, err := capacidadService.ListModelPolicies(scopeTipo, scopeRef, activaPtr)
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, apiPoliticasModeloResponse{Politicas: politicas})
	case http.MethodPost:
		var req apiPoliticaModeloSaveRequest
		if err := apiDecodeJSON(r, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		id, err := capacidadService.SaveModelPolicy(&db.PoliticaModelo{
			ScopeTipo:       strings.TrimSpace(req.ScopeTipo),
			ScopeRef:        strings.TrimSpace(req.ScopeRef),
			PerfilTarea:     strings.TrimSpace(req.PerfilTarea),
			PoolSlug:        strings.TrimSpace(req.PoolSlug),
			ModelSlug:       strings.TrimSpace(req.ModelSlug),
			ReasoningEffort: strings.TrimSpace(req.ReasoningEffort),
			Prioridad:       req.Prioridad,
			Activa:          req.Activa,
			MetadataJSON:    strings.TrimSpace(req.MetadataJSON),
		})
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		apiWriteJSON(w, http.StatusCreated, apiPoliticaModeloSaveResponse{ID: id})
	default:
		apiMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func apiHandlerPoliticasModeloSeedInicial(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	if err := capacidadService.SeedInitialModelPolicies(); err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func apiHandlerModeloResolver(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	var tareaIDPtr *int64
	if tareaIDRaw := strings.TrimSpace(r.URL.Query().Get("tarea_id")); tareaIDRaw != "" {
		tareaID, err := strconv.ParseInt(tareaIDRaw, 10, 64)
		if err != nil || tareaID <= 0 {
			apiError(w, http.StatusBadRequest, fmt.Errorf("tarea_id inválido"))
			return
		}
		tareaIDPtr = &tareaID
	}
	resolucion, err := capacidadService.ResolveModelPolicy(db.ResolverPoliticaInput{
		TareaID:      tareaIDPtr,
		ProyectoSlug: strings.TrimSpace(r.URL.Query().Get("proyecto")),
		Fase:         strings.TrimSpace(r.URL.Query().Get("fase")),
		PerfilTarea:  strings.TrimSpace(r.URL.Query().Get("perfil")),
	})
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiResolucionModeloResponse{Resolucion: resolucion})
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
		id, err := conectoresService.SaveConnector(conectoresapp.SaveConnectorInput{
			Slug:         req.Slug,
			Nombre:       req.Nombre,
			Transporte:   req.Transporte,
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
		conector, err := conectoresService.GetConnector(strconv.FormatInt(id, 10))
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
	conector, err := conectoresService.GetConnector(ref)
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
	proyecto, err := sesionesAPIService.ActivateAssignment(req.Agente, req.Proyecto, req.Nota)
	if err != nil {
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
		repo := db.CoordinationLockRepository()
		filter := coordinacion.LockFilter{}
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
			estado := coordinacion.LockState(estadoStr)
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
			sesion, err := sesionesAPIService.GetActiveSession(req.Agente, req.Proyecto)
			if err == nil && sesion != nil {
				sessionID = &sesion.ID
			}
		}
		lock, err := svc.AcquireLock(coordinacion.AcquireLockInput{
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
		lock, err := db.CoordinationLockRepository().GetByID(id)
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
		lock, err := svc.RenewLock(coordinacion.RenewLockInput{
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
		lock, err := svc.ReleaseLock(coordinacion.ReleaseLockInput{
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
		proyectoID, err := tareasService.ResolveProjectID(proyectoRef)
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		filtro.ProyectoID = proyectoID
	}
	if propuestaCodigo := strings.TrimSpace(r.URL.Query().Get("propuesta")); propuestaCodigo != "" {
		propuestaID, err := tareasService.ResolveProposalID(propuestaCodigo)
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		filtro.PropuestaID = propuestaID
	}
	tareas, err := tareasService.List(filtro)
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
	if strings.TrimSpace(req.Titulo) == "" {
		apiError(w, http.StatusBadRequest, fmt.Errorf("el título es obligatorio"))
		return
	}
	id, err := tareasService.Create(tareasapp.CreateTaskInput{
		Titulo:          req.Titulo,
		Descripcion:     req.Descripcion,
		Modulo:          req.Modulo,
		Prioridad:       db.PrioridadTarea(req.Prioridad),
		CreadoPor:       valorConFallback(req.CreadoPor, "alberto"),
		Agente:          req.Agente,
		Proyecto:        req.Proyecto,
		PropuestaCodigo: req.Propuesta,
		Notas:           req.Notas,
	})
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	tarea, err := tareasService.Get(id)
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
		tarea, err := tareasService.Get(id)
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
	if len(parts) == 2 && parts[1] == "refineria" && r.Method == http.MethodPost {
		apiHandlerTareaRefineria(w, r, parts[0])
		return
	}
	http.NotFound(w, r)
}

func apiHandlerTareaRefineria(w http.ResponseWriter, r *http.Request, idStr string) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		apiError(w, http.StatusBadRequest, fmt.Errorf("id inválido"))
		return
	}
	var req apiRefineriaRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(req.Agente) == "" {
		apiError(w, http.StatusBadRequest, fmt.Errorf("agente requerido"))
		return
	}
	s, err := db.SolicitarRefineria(id, req.Agente, req.Rama, req.Dir, req.CmdTest)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusCreated, map[string]any{"ok": true, "solicitud": s})
}

// apiHandlerRefineria gestiona GET /api/refineria (lista solicitudes pendientes).
func apiHandlerRefineria(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	solicitudes, err := db.ListarRefineriasPendientes()
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"solicitudes": solicitudes})
}

// apiRouterRefineria gestiona GET /api/refineria/:id y GET /api/refineria/tarea/:id.
func apiRouterRefineria(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/refineria/"), "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	// GET /api/refineria/tarea/:tareaID
	if len(parts) == 2 && parts[0] == "tarea" {
		tareaID, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			apiError(w, http.StatusBadRequest, fmt.Errorf("id inválido"))
			return
		}
		ss, err := db.ListarRefineriaPorTarea(tareaID)
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{"solicitudes": ss})
		return
	}
	// GET /api/refineria/:id
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		apiError(w, http.StatusBadRequest, fmt.Errorf("id inválido"))
		return
	}
	s, err := db.GetRefineriaSolicitud(id)
	if err != nil {
		apiError(w, http.StatusNotFound, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"solicitud": s})
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
		err = tareasService.Take(id, req.Agente)
	case "iniciar":
		err = tareasService.Start(id, req.Agente)
	case "completar":
		err = tareasService.Complete(id, req.Agente, req.Commit)
	case "bloquear":
		err = tareasService.Block(id, req.Agente, req.Motivo)
	case "desbloquear":
		err = tareasService.Unblock(id, req.Agente, req.Resolucion)
	case "nota":
		err = tareasService.Note(id, req.Agente, req.Nota)
	case "contrato":
		err = db.DefinirContrato(id, req.Agente)
	case "backlog":
		err = tareasService.MoveToBacklog(id)
		if err == nil {
			db.Audit("alberto", "backlog_tarea", "tarea", id, "")
		}
	case "cancelar":
		err = db.CancelarTarea(id, req.Agente, valorConFallback(req.Motivo, "duplicado o error"))
	case "reasignar":
		err = tareasService.Reassign(id, req.NuevoAgente)
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
	tarea, err := tareasService.Get(id)
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
		repo := db.CoordinationWorktreeRepository()
		filter := coordinacion.WorktreeFilter{}
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
			estado := coordinacion.WorktreeState(estadoStr)
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
		worktree, err := svc.PrepareWorktree(coordinacion.PrepareWorktreeInput{
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

func apiHandlerGitMerges(w http.ResponseWriter, r *http.Request) {
	svc := gitgobernanza.NewService(gitgobernanza.Repository{})
	switch r.Method {
	case http.MethodGet:
		merges, err := svc.ListRequests(
			strings.TrimSpace(r.URL.Query().Get("proyecto")),
			strings.TrimSpace(r.URL.Query().Get("estado")),
		)
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, apiGitMergesResponse{Merges: merges})
	case http.MethodPost:
		var req apiGitMergeSaveRequest
		if err := apiDecodeJSON(r, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		id, err := svc.SaveRequest(req.intoInput())
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		merge, err := db.GetGitMerge(id)
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusCreated, apiGitMergeSaveResponse{OK: true, ID: id, Merge: merge})
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
	runtimes, err := runtimesService.ListRuntimes(filter)
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
	tree, err := runtimesService.BuildRuntimeTree(filter)
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
	runtime, err := runtimesService.GetRuntime(id)
	if err != nil {
		apiError(w, http.StatusNotFound, err)
		return
	}
	samples, err := runtimesService.ListRuntimeSamples(id, 20)
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
	handles, err := runtimesService.ListRuntimeHandles(filtro)
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
			p, err := runtimesService.GetProject(proyecto)
			if err != nil {
				apiError(w, http.StatusBadRequest, err)
				return
			}
			filter.ProyectoID = &p.ID
		}
		orders, err := runtimesService.ListRuntimeOrders(filter)
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
			p, err := runtimesService.GetProject(proyecto)
			if err != nil {
				apiError(w, http.StatusBadRequest, err)
				return
			}
			proyectoID = &p.ID
		}
		id, err := runtimesService.EnqueueRuntimeOrder(&db.RuntimeOrder{
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
			p, err := runtimesService.GetProject(proyecto)
			if err != nil {
				apiError(w, http.StatusBadRequest, err)
				return
			}
			filter.ProyectoID = &p.ID
		}
		mailbox, err := runtimesService.ListRuntimeMailbox(filter)
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
			p, err := runtimesService.GetProject(proyecto)
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
		id, err := runtimesService.SendRuntimeMailbox(&db.RuntimeMailboxMessage{
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
		if err := runtimesService.MarkRuntimeMailboxDelivered(id); err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
	case "consumir":
		if err := runtimesService.MarkRuntimeMailboxConsumed(id); err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
	default:
		http.NotFound(w, r)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"ok": true, "id": id})
}

func apiHandlerRuntimeCheckpoints(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		apiHandlerRuntimeCheckpointsListar(w, r)
		return
	case http.MethodPost:
	default:
		apiMethodNotAllowed(w, http.MethodGet, http.MethodPost)
		return
	}
	var req apiRuntimeCheckpointCreateRequest
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
		p, err := runtimesService.GetProject(proyecto)
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		proyectoID = &p.ID
	}
	var sesionID *int64
	if req.SesionID > 0 {
		sesionID = &req.SesionID
	}
	var runtimeID *int64
	if req.RuntimeID > 0 {
		runtimeID = &req.RuntimeID
	}
	id, err := runtimesService.CreateRuntimeCheckpoint(&db.RuntimeCheckpoint{
		Agente:         strings.TrimSpace(req.Agente),
		ProyectoID:     proyectoID,
		SesionID:       sesionID,
		RuntimeID:      runtimeID,
		CheckpointKind: strings.TrimSpace(req.CheckpointKind),
		Resumen:        strings.TrimSpace(req.Resumen),
		Branch:         strings.TrimSpace(req.Branch),
		CWD:            strings.TrimSpace(req.CWD),
		PayloadJSON:    payload,
		ResumeStrategy: strings.TrimSpace(req.ResumeStrategy),
		Source:         strings.TrimSpace(req.Source),
	})
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusCreated, map[string]any{"ok": true, "id": id})
}

func apiHandlerRuntimeCheckpointsListar(w http.ResponseWriter, r *http.Request) {
	agente := strings.TrimSpace(r.URL.Query().Get("agente"))
	if agente == "" {
		apiError(w, http.StatusBadRequest, fmt.Errorf("agente obligatorio"))
		return
	}
	filter := db.FiltroRuntimeCheckpoints{Agente: &agente}
	if proyecto := strings.TrimSpace(r.URL.Query().Get("proyecto")); proyecto != "" {
		p, err := runtimesService.GetProject(proyecto)
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		filter.ProyectoID = &p.ID
	}
	if checkpointKind := strings.TrimSpace(r.URL.Query().Get("kind")); checkpointKind != "" {
		filter.CheckpointKind = &checkpointKind
	}
	if source := strings.TrimSpace(r.URL.Query().Get("source")); source != "" {
		filter.Source = &source
	}
	if limitStr := strings.TrimSpace(r.URL.Query().Get("limit")); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			apiError(w, http.StatusBadRequest, fmt.Errorf("limit inválido"))
			return
		}
		filter.Limit = limit
	}
	checkpoints, err := runtimesService.ListRuntimeCheckpoints(filter)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"checkpoints": checkpoints})
}

func apiRouterRuntimeCheckpoints(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	idStr := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/runtime-checkpoints/"), "/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		apiError(w, http.StatusBadRequest, fmt.Errorf("id inválido"))
		return
	}
	checkpoint, err := runtimesService.GetRuntimeCheckpoint(id)
	if err != nil {
		apiError(w, http.StatusNotFound, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"checkpoint": checkpoint})
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
		p, err := runtimesService.GetProject(proyecto)
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		proyectoID = &p.ID
	}
	checkpoint, err := runtimesService.LatestRuntimeCheckpoint(agente, proyectoID)
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
		worktree, err := db.CoordinationWorktreeRepository().GetByID(id)
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
	propuestas, err := propuestasService.ListByProject(estadoPtr, strings.TrimSpace(r.URL.Query().Get("proyecto")))
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"propuestas": propuestas})
}

func apiHandlerLenguajePolitica(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		policy, err := lenguajeService.GetPolicy()
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{"politica": policy})
	case http.MethodPost:
		var req apiLenguajePoliticaSetRequest
		if err := apiDecodeJSON(r, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		if req.Politica == nil {
			apiError(w, http.StatusBadRequest, fmt.Errorf("politica requerida"))
			return
		}
		if err := lenguajeService.SetPolicy(req.Politica, strings.TrimSpace(req.Por)); err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		apiError(w, http.StatusMethodNotAllowed, fmt.Errorf("metodo no permitido"))
	}
}

func apiHandlerLenguajeMatriz(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		entries, err := lenguajeService.ListMatrixEntries()
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{"matriz": entries})
	case http.MethodPost:
		var req apiLenguajeMatrizSetRequest
		if err := apiDecodeJSON(r, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		entry, err := lenguajeService.SetMatrixEntry(lenguajeapp.SetMatrixEntryInput{
			Scope:     req.Scope,
			Selector:  req.Selector,
			Contexto:  req.Contexto,
			Language:  req.Idioma,
			Reason:    req.Razon,
			UpdatedBy: req.Por,
		})
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{"entrada": entry})
	default:
		apiError(w, http.StatusMethodNotAllowed, fmt.Errorf("metodo no permitido"))
	}
}

func apiHandlerLenguajeMatrizBorrar(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req apiLenguajeMatrizDeleteRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	if err := lenguajeService.DeleteMatrixEntry(req.Scope, req.Selector, req.Contexto); err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func apiHandlerLenguajeResolver(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	project := strings.TrimSpace(r.URL.Query().Get("proyecto"))
	contexto := strings.TrimSpace(r.URL.Query().Get("contexto"))
	var tareaPtr *int64
	if tareaStr := strings.TrimSpace(r.URL.Query().Get("tarea")); tareaStr != "" {
		tareaID, err := strconv.ParseInt(tareaStr, 10, 64)
		if err != nil || tareaID <= 0 {
			apiError(w, http.StatusBadRequest, fmt.Errorf("tarea inválida"))
			return
		}
		tareaPtr = &tareaID
	}
	res, err := lenguajeService.Resolve(lenguajeapp.ResolveInput{
		Proyecto: project,
		TareaID:  tareaPtr,
		Contexto: contexto,
	})
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"resolucion": res})
}

func apiFiltroRuntimesDesdeRequest(r *http.Request) (db.FiltroRuntimes, error) {
	filter := db.FiltroRuntimes{}
	if agente := strings.TrimSpace(r.URL.Query().Get("agente")); agente != "" {
		filter.Agente = &agente
	}
	if proyectoRef := strings.TrimSpace(r.URL.Query().Get("proyecto")); proyectoRef != "" {
		proyecto, err := runtimesService.GetProject(proyectoRef)
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
	if strings.TrimSpace(req.Titulo) == "" {
		apiError(w, http.StatusBadRequest, fmt.Errorf("el título es obligatorio"))
		return
	}
	id, _, err := propuestasService.Create(propuestasapp.CreateProposalInput{
		Codigo:       req.Codigo,
		Titulo:       req.Titulo,
		Descripcion:  req.Descripcion,
		Tipo:         valorConFallback(req.Tipo, "implementacion"),
		Proyecto:     req.Proyecto,
		PropuestoPor: valorConFallback(req.PropuestoPor, "alberto"),
		Distribuidor: valorConFallback(req.Distribuidor, "alberto"),
	})
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	detail, err := propuestasService.GetDetail(strings.TrimSpace(req.Codigo))
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	detail.Proposal.Votos = detail.Votes
	apiWriteJSON(w, http.StatusCreated, map[string]any{"ok": true, "id": id, "propuesta": detail.Proposal})
}

func apiHandlerMemoria(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		filter := db.FiltroEntidadesMemoria{}
		if proyectoRef := strings.TrimSpace(r.URL.Query().Get("proyecto")); proyectoRef != "" {
			proyecto, err := runtimesService.GetProject(proyectoRef)
			if err != nil {
				apiError(w, http.StatusBadRequest, err)
				return
			}
			filter.ProyectoID = &proyecto.ID
		}
		if tipo := strings.TrimSpace(r.URL.Query().Get("tipo")); tipo != "" {
			filter.Tipo = &tipo
		}
		entidades, err := runtimesService.ListMemoryEntities(filter)
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{"entidades": entidades})
	case http.MethodPost:
		var req apiMemoriaEntidadRequest
		if err := apiDecodeJSON(r, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		var proyectoID *int64
		if proyectoRef := strings.TrimSpace(req.Proyecto); proyectoRef != "" {
			proyecto, err := runtimesService.GetProject(proyectoRef)
			if err != nil {
				apiError(w, http.StatusBadRequest, err)
				return
			}
			proyectoID = &proyecto.ID
		}
		id, err := runtimesService.UpsertMemoryEntity(&db.EntidadMemoria{
			Nombre:        strings.TrimSpace(req.Nombre),
			Tipo:          strings.TrimSpace(req.Tipo),
			ValorJSON:     strings.TrimSpace(req.Valor),
			MetadataJSON:  strings.TrimSpace(req.Metadata),
			VerificadoPor: strings.TrimSpace(req.VerificadoPor),
			ProyectoID:    proyectoID,
		})
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		entidad, err := runtimesService.GetMemoryEntity(strings.TrimSpace(req.Nombre), proyectoID)
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusCreated, map[string]any{"ok": true, "id": id, "entidad": entidad})
	default:
		apiMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func apiRouterMemoria(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/memoria/"), "/"), "/")
	if len(parts) != 1 || parts[0] == "" || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	var proyectoID *int64
	if proyectoRef := strings.TrimSpace(r.URL.Query().Get("proyecto")); proyectoRef != "" {
		proyecto, err := runtimesService.GetProject(proyectoRef)
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		proyectoID = &proyecto.ID
	}
	entidad, err := runtimesService.GetMemoryEntity(parts[0], proyectoID)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	if entidad == nil {
		apiError(w, http.StatusNotFound, fmt.Errorf("entidad '%s' no encontrada", parts[0]))
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"entidad": entidad})
}

func apiRouterPropuestas(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/propuestas/"), "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	if len(parts) == 1 && r.Method == http.MethodGet {
		detail, err := propuestasService.GetDetail(parts[0])
		if err != nil {
			apiError(w, http.StatusNotFound, err)
			return
		}
		detail.Proposal.Votos = detail.Votes
		conteo := conteoVotosDesdeLista(detail.Votes)
		apiWriteJSON(w, http.StatusOK, map[string]any{
			"propuesta":    detail.Proposal,
			"conteo_votos": conteo,
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
		proyectoID, err := sesionesAPIService.ResolveProjectID(proyectoFiltro)
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		filtro.ProyectoID = proyectoID
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
	sesiones, err := sesionesAPIService.ListInspectionSessions(filtro)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	if limit > 0 && len(sesiones) > limit {
		sesiones = sesiones[:limit]
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"sesiones": sesiones})
}

func apiHandlerProgreso(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	proyecto := strings.TrimSpace(r.URL.Query().Get("proyecto"))
	if proyecto == "" {
		apiError(w, http.StatusBadRequest, fmt.Errorf("proyecto obligatorio"))
		return
	}
	resumen, err := progresoService.GetSummary(proyecto)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiProgresoResumenResponse{Resumen: resumen})
}

func apiHandlerProgresoFases(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		proyecto := strings.TrimSpace(r.URL.Query().Get("proyecto"))
		if proyecto == "" {
			apiError(w, http.StatusBadRequest, fmt.Errorf("proyecto obligatorio"))
			return
		}
		fases, err := progresoService.ListPhases(proyecto)
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, apiProgresoFasesResponse{Fases: fases})
	case http.MethodPost:
		var req apiProgresoFaseRegistrarRequest
		if err := apiDecodeJSON(r, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		id, fase, err := progresoService.RegisterPhase(progresoapp.RegisterPhaseInput{
			Proyecto:    req.Proyecto,
			Nombre:      req.Nombre,
			Descripcion: req.Descripcion,
			Orden:       req.Orden,
			Peso:        req.Peso,
			Estado:      req.Estado,
		})
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		apiWriteJSON(w, http.StatusCreated, apiProgresoFaseResponse{ID: id, Fase: fase})
	default:
		apiMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func apiRouterProgresoFases(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	idStr := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/progreso/fases/"), "/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		apiError(w, http.StatusBadRequest, fmt.Errorf("id inválido"))
		return
	}
	var req apiProgresoFaseActualizarRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	fase, err := progresoService.UpdatePhase(progresoapp.UpdatePhaseInput{
		ID:          id,
		Proyecto:    req.Proyecto,
		Nombre:      req.Nombre,
		Descripcion: req.Descripcion,
		Orden:       req.Orden,
		Peso:        req.Peso,
		Estado:      req.Estado,
	})
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiProgresoFaseResponse{ID: fase.ID, Fase: fase})
}

func apiRouterProgresoTareas(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	idStr := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/progreso/tareas/"), "/")
	tareaID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || tareaID <= 0 {
		apiError(w, http.StatusBadRequest, fmt.Errorf("tarea-id inválido"))
		return
	}
	var req apiProgresoTareaRegistrarRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	if err := progresoService.RegisterTaskProgress(progresoapp.RegisterTaskProgressInput{
		TareaID:        tareaID,
		Proyecto:       req.Proyecto,
		FaseID:         req.FaseID,
		ProgresoPct:    req.ProgresoPct,
		ActualizadoPor: req.ActualizadoPor,
	}); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"ok": true})
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
	sesion, err := sesionesAPIService.GetInspectionSession(id)
	if err != nil {
		apiError(w, http.StatusNotFound, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"sesion": sesion})
}

func apiHandlerPropuestaAccion(w http.ResponseWriter, r *http.Request, codigo string) {
	var req apiPropuestaAccionRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	var err error
	switch req.Accion {
	case "votar":
		result, err := propuestasService.VoteDetail(codigo, req.Agente, db.PosicionVoto(req.Posicion), req.Comentario)
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{
			"ok":        true,
			"propuesta": result.Proposal,
			"conteo_votos": apiConteoVotosResponse{
				Acuerdo: result.Acuerdo, Desacuerdo: result.Desacuerdo, Abstencion: result.Abstencion, Pendiente: result.Pendiente,
			},
		})
		return
	case "actualizar":
		err = db.ActualizarPropuesta(codigo, db.PropuestaPatch{
			Titulo:            req.Titulo,
			Descripcion:       req.Descripcion,
			AnexarDescripcion: req.AnexarDesc,
			Tipo:              req.Tipo,
		}, valorConFallback(req.Agente, "alberto"))
	case "cerrar":
		err = propuestasService.Close(codigo, req.EstadoCierre, valorConFallback(req.Agente, "alberto"))
	case "reabrir":
		_, err = propuestasService.Reopen(codigo, valorConFallback(req.Agente, "alberto"))
	case "reparar_votos", "reparar-votos":
		_, err = propuestasService.RepairPendingVotes(codigo, valorConFallback(req.Agente, "alberto"))
	default:
		apiError(w, http.StatusBadRequest, fmt.Errorf("acción desconocida"))
		return
	}
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	detail, err := propuestasService.GetDetail(codigo)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	detail.Proposal.Votos = detail.Votes
	apiWriteJSON(w, http.StatusOK, map[string]any{"ok": true, "propuesta": detail.Proposal, "conteo_votos": conteoVotosDesdeLista(detail.Votes)})
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
	result, err := sesionesAPIService.StartContext(sesionesapp.StartContextInput{
		Agente:            req.Agente,
		NuevoCodex:        req.NuevoCodex,
		Proyecto:          req.Proyecto,
		Conector:          req.Conector,
		CWD:               req.CWD,
		Herramienta:       req.Herramienta,
		ExternalSessionID: req.ExternalSessionID,
		ResumePayload:     req.ResumePayload,
		Resumen:           req.Resumen,
		Branch:            req.Branch,
		Host:              req.Host,
		PID:               req.PID,
	})
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusCreated, &apiSesionInicioResponse{
		Sesion:               result.Sesion,
		SesionPrevia:         result.SesionPrevia,
		Rol:                  result.Rol,
		PropuestasPendientes: result.PropuestasPendientes,
		Reglas:               result.Reglas,
		Skills:               result.Skills,
		Workflow:             result.Workflow,
	})
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
	sesion, err := sesionesAPIService.SaveActiveSession(req.Agente, req.Proyecto, upd)
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
	if _, err := sesionesAPIService.Finish(req.Agente); err != nil {
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
	cwd := strings.TrimSpace(r.URL.Query().Get("cwd"))
	sesion, err := sesionesAPIService.Continue(agente, strings.TrimSpace(r.URL.Query().Get("proyecto")), cwd)
	if err != nil {
		apiError(w, http.StatusNotFound, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"sesion": sesion})
}

func apiHandlerSesionPresupuesto(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		sesionID, err := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("sesion")), 10, 64)
		if strings.TrimSpace(r.URL.Query().Get("sesion")) != "" && (err != nil || sesionID <= 0) {
			apiError(w, http.StatusBadRequest, fmt.Errorf("sesion inválida"))
			return
		}
		result, err := sesionesAPIService.GetBudget(sesionID, strings.TrimSpace(r.URL.Query().Get("agente")))
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, apiSesionPresupuestoResponse{
			Sesion:      result.Sesion,
			Presupuesto: result.Presupuesto,
			Evaluacion:  result.Evaluacion,
		})
	case http.MethodPost:
		var req apiSesionPresupuestoRequest
		if err := apiDecodeJSON(r, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		result, err := sesionesAPIService.RegisterBudget(req.SesionID, req.Agente, &db.PresupuestoSesion{
			PoolID:            req.PoolID,
			ModelSlug:         strings.TrimSpace(req.ModelSlug),
			WindowKind:        strings.TrimSpace(req.WindowKind),
			WindowStartedAt:   req.WindowStartedAt,
			ResetAt:           req.ResetAt,
			RemainingSeconds:  req.RemainingSeconds,
			RemainingMessages: req.RemainingMessages,
			RemainingTokens:   req.RemainingTokens,
			RemainingCredits:  req.RemainingCredits,
			BudgetSource:      strings.TrimSpace(req.BudgetSource),
			RawSnapshotJSON:   strings.TrimSpace(req.RawSnapshotJSON),
		})
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		apiWriteJSON(w, http.StatusCreated, apiSesionPresupuestoResponse{
			ID:          result.ID,
			Sesion:      result.Sesion,
			Presupuesto: result.Presupuesto,
			Evaluacion:  result.Evaluacion,
		})
	default:
		apiMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
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

	agente, err := agentesService.GetAgent(agenteNombre)
	if err != nil {
		apiError(w, http.StatusNotFound, err)
		return
	}
	out, err := agentesService.BuildPrepare(agentesapp.PrepareInput{
		Agente:       agente.Nombre,
		Proyecto:     proyectoRef,
		Conector:     conectorRef,
		Modelo:       modelo,
		Razonamiento: razonamiento,
		Perfil:       perfilTarea,
	})
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
	out, err := agentesService.ProcessTick(agentesapp.TickInput{
		Agente:     req.Agente,
		Proyecto:   req.Proyecto,
		Host:       req.Host,
		PID:        req.PID,
		CuotaPct:   req.CuotaPct,
		Finalizado: req.Finalizado,
		Motivo:     req.Motivo,
	})
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
	if err := agentesService.PauseTemporarily(req.Agente, req.Minutos, req.Motivo, req.Accion, req.Entidad, req.Detalle); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"ok": true, "agente": req.Agente})
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
