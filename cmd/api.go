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
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"orquesta/agentesapp"
	"orquesta/capacidadapp"
	"orquesta/conectoresapp"
	"orquesta/coordinacion"
	"orquesta/db"
	"orquesta/fabricaapp"
	"orquesta/gitgobernanza"
	"orquesta/gobernanzaapp"
	"orquesta/i18n"
	"orquesta/internal/a2ui"
	"orquesta/lenguajeapp"
	"orquesta/memoriaproyecto"
	"orquesta/notificaciones"
	"orquesta/progresoapp"
	"orquesta/propuestasapp"
	"orquesta/reviewapp"
	"orquesta/runtimesapp"
	"orquesta/sesionesapp"
	"orquesta/supervisionapp"
	"orquesta/tareasapp"
)

type apiErrorResponse struct {
	Error string `json:"error"`
}

type apiStatusResponse struct {
	Agentes             []*db.Agente                        `json:"agentes"`
	ConteoTareas        map[string]int                      `json:"conteo_tareas"`
	ResumenTareas       map[string]int                      `json:"resumenTareas,omitempty"`
	Proyectos           []*db.Proyecto                      `json:"proyectos"`
	AsignacionesActivas map[int64]int                       `json:"asignaciones_activas"`
	SesionesActivas     map[int64]int                       `json:"sesiones_activas"`
	PropuestasAbiertas  []*db.Propuesta                     `json:"propuestas_abiertas"`
	Generado            string                              `json:"generado,omitempty"`
	TareasPorEstado     map[string]int                      `json:"tareasPorEstado,omitempty"`
	AgentesActivos      []*db.Agente                        `json:"agentesActivos,omitempty"`
	AgentesTrabajando   []*db.Agente                        `json:"agentesTrabajando,omitempty"`
	AgentesSaturados    []*db.Agente                        `json:"agentesSaturados,omitempty"`
	AgentesAtascados    []*db.Agente                        `json:"agentesAtascados,omitempty"`
	AgentesAuthManual   []*db.Agente                        `json:"agentesAuthManual,omitempty"`
	AgentesQuotaBlocked []*db.Agente                        `json:"agentesQuotaBlocked,omitempty"`
	PropuestasResumen   []propuestaLite                     `json:"propuestasAbiertas,omitempty"`
	TareasActivas       []tareaLite                         `json:"tareasActivas,omitempty"`
	TareasEnProgreso    []tareaLite                         `json:"tareasEnProgreso,omitempty"`
	TareasReservadas    []tareaLite                         `json:"tareasReservadas,omitempty"`
	PoolsLocales        []*capacidadapp.PoolLocalCompartido `json:"poolsLocales,omitempty"`
	DeudaDispatch       deudaDispatchResumen                `json:"deudaDispatch,omitempty"`
	Autonomia           autonomiaResumen                    `json:"autonomia,omitempty"`
	AutonomySurface     *autonomySurface                    `json:"autonomySurface,omitempty"`
	AutonomyHighlights  []string                            `json:"autonomyHighlights,omitempty"`
	CriticalProjectRisk *workspaceAutonomyProjectSummary    `json:"criticalProjectRisk,omitempty"`
	WorkersConectados   int                                 `json:"workersConectados,omitempty"`
	WorkersTrabajando   int                                 `json:"workersTrabajando,omitempty"`
	SupervisoresActivos int                                 `json:"supervisoresActivos,omitempty"`
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

type apiProyectoFusionRequest struct {
	Origen         string `json:"origen"`
	ArchivarOrigen bool   `json:"archivar_origen"`
}

type apiProyectoAutonomiaSaveRequest struct {
	Enabled              bool   `json:"enabled"`
	ObjetivoGeneral      string `json:"objetivo_general"`
	DefinitionOfDoneJSON string `json:"definition_of_done_json"`
	MaxWorkers           int    `json:"max_workers"`
	SupervisorAgente     string `json:"supervisor_agente"`
	ReviewerAgente       string `json:"reviewer_agente"`
	ReserveReviewer      bool   `json:"reserve_reviewer"`
	ReserveSupervisor    bool   `json:"reserve_supervisor"`
	ReviewRequired       bool   `json:"review_required"`
	AutoCreateTasks      bool   `json:"auto_create_tasks"`
	AutoCloseProject     bool   `json:"auto_close_project"`
	EstadoAutonomia      string `json:"estado_autonomia"`
}

type apiProyectoAutonomiaResponse struct {
	Policy *supervisionapp.Policy  `json:"policy"`
	Cycles []*supervisionapp.Cycle `json:"cycles,omitempty"`
}

var (
	apiStatusFetchTimeout           = 3 * time.Second
	apiServerOperationalTimeout     = 3 * time.Second
	apiAuditTimeout                 = 2 * time.Second
	apiAgentsPanelTimeout           = 3 * time.Second
	apiAgentOverviewTimeout         = 3 * time.Second
	apiAgentBudgetRefreshTimeout    = 3 * time.Second
	apiConfigTimeout                = 3 * time.Second
	apiProjectListTimeout           = 3 * time.Second
	apiTaskListTimeout              = 3 * time.Second
	apiRuntimeProcessOrdersTimeout  = 4 * time.Second
	apiRuntimeProcessMailboxTimeout = 4 * time.Second
	apiAgentPrepareTimeout          = 15 * time.Second
	apiAgentTickTimeout             = 10 * time.Second
	apiAgentPrepareLimiter          = make(chan struct{}, 4)
	apiAgentTickLimiter             = make(chan struct{}, 4)
	apiActivePrepareRequests        atomic.Int32
	apiActiveTickRequests           atomic.Int32
	apiConfigGetFn                  = func(clave string) (string, error) {
		return configService.Get(clave)
	}
	apiConfigListFn = func() (map[string]string, error) {
		return configService.List()
	}
	apiConfigSetFn = func(clave, valor string) error {
		if err := configService.Set(clave, valor); err != nil {
			return err
		}
		resetControlPlaneConfigCache()
		return nil
	}
	apiRuntimeProcessMailboxProjectFn = func(ref string) (*db.Proyecto, error) {
		return runtimesService.GetProject(ref)
	}
	apiRuntimeProcessOrdersExecutor = func() (int, error) {
		return (dbAutomationService{}).ProcesarRuntimeOrdersBatch()
	}
	apiRuntimeProcessMailboxExecutor = func(filter db.FiltroRuntimeMailbox) (int, error) {
		return procesarRuntimeMailboxBatchConFiltro(filter)
	}
	apiRuntimePurgeTranscriptNoiseExecutor = func() (int, error) {
		return db.PurgarRuntimeTranscriptRuidoHistorico()
	}
	apiRuntimePurgeTranscriptNoiseAllExecutor = func() (int, error) {
		return db.PurgarRuntimeTranscriptRuidoHistoricoCompleto()
	}
	apiAgentPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return agentesService.BuildPanelRows()
	}
	apiAgentDetailBuilder = func(nombre string, compact bool) (*agentesapp.Detail, error) {
		if compact {
			return agentesService.BuildDetailCompact(nombre)
		}
		return agentesService.BuildDetail(nombre)
	}
	apiAgentBudgetRefreshExecutor = refrescarPresupuestoSesionObservadoAgente
	apiAgentPrepareBuilder        = func(input agentesapp.PrepareInput) (*agentesapp.PrepareOutput, error) {
		return agentesService.BuildPrepare(input)
	}
	apiAgentTickProcessor = func(input agentesapp.TickInput) (*agentesapp.TickOutput, error) {
		return agentesService.ProcessTick(input)
	}
	apiListRuntimeOrdersFn = func(filter db.FiltroRuntimeOrders) ([]*db.RuntimeOrder, error) {
		return runtimesService.ListRuntimeOrders(filter)
	}
	apiListRuntimeTranscriptFn = func(filter db.FiltroRuntimeTranscript) ([]*db.RuntimeTranscriptEntry, error) {
		return runtimesService.ListRuntimeTranscript(filter)
	}
	apiListRuntimeMailboxFn = func(filter db.FiltroRuntimeMailbox) ([]*db.RuntimeMailboxMessage, error) {
		return runtimesService.ListRuntimeMailbox(filter)
	}
	apiGetAgentFn = func(ref string) (*db.Agente, error) {
		return agentesService.GetAgent(ref)
	}
	apiGetRuntimeProjectFn = func(ref string) (*db.Proyecto, error) {
		return runtimesService.GetProject(ref)
	}
	apiProjectLookupFn = func(ref string) (*db.Proyecto, error) {
		return db.GetProyecto(ref)
	}
	apiListProjectsFn = func(filtro db.FiltroProyectos, cwdHint string) ([]*db.Proyecto, error) {
		return db.ListarProyectosConRutaEfectiva(filtro, cwdHint)
	}
	apiListProjectsLiteFn = func(filtro db.FiltroProyectos) ([]*db.Proyecto, error) {
		return db.ListarProyectos(filtro)
	}
	apiProjectLookupPrepareLiteFn = func(ref string) (*db.Proyecto, error) {
		return db.GetProyectoPrepareLite(ref)
	}
	apiProjectLookupWithRouteFn = func(ref, cwdHint string) (*db.Proyecto, error) {
		return db.GetProyectoConRutaEfectiva(ref, cwdHint)
	}
	apiProjectLookupWithRoutePrepareLiteFn = func(ref, cwdHint string) (*db.Proyecto, error) {
		proyecto, err := db.GetProyectoPrepareLite(ref)
		if err != nil || proyecto == nil {
			return proyecto, err
		}
		return db.ProyectoPrepareLiteConRutaEfectiva(proyecto, ""), nil
	}
	apiListAuditFn                     = db.ListarAuditoria
	apiListTasksFn                     = tareasService.List
	apiServerOperationalBuilder        = buildServerOperationalInfoFastFromDB
	apiServerOperationalStatusFallback = func() (apiStatusResponse, error) {
		if status, ok := fetchStatusReadOnlyLiteDirect(statusFastTimeout); ok {
			return status, nil
		}
		if status, ok := fetchStatusForOperationalFallback(statusFastTimeout); ok {
			return status, nil
		}
		return apiStatusResponse{}, errStatusFetchTimeout
	}
	apiServerOperationalDegradedBuilder = func() serverOperationalInfo {
		return buildServerOperationalInfo(degradedAPIStatusResponse())
	}
	apiStatusUltraLiteFetcher      = fetchStatusUltraLiteFallback
	apiStatusReadOnlyLiteFetcher   = db.ListarEstadoLigeroReadOnly
	apiStatusFallbackRichGrace     = 200 * time.Millisecond
	apiListRuntimeOrdersTimeout    = 2 * time.Second
	apiListRuntimeMailboxTimeout   = 2 * time.Second
	apiRuntimeMailboxDefaultLimit  = 100
	apiRuntimeProjectLookupTimeout = 1500 * time.Millisecond
	apiAgentCanonicalCacheTTL      = 30 * time.Second
	apiAgentCanonicalCacheMu       sync.Mutex
	apiAgentCanonicalCache         = map[string]apiCanonicalAgentCacheEntry{}
)

func init() {
	agentesapp.ShouldPersistTickHeartbeatDB = func() bool {
		return !agentAPIHotPathBursting()
	}
}

type apiCanonicalAgentCacheEntry struct {
	nombre  string
	expires time.Time
}

func apiCanonicalAgentCacheKey(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func apiCanonicalAgentCacheGet(raw string) (string, bool) {
	key := apiCanonicalAgentCacheKey(raw)
	if key == "" {
		return "", false
	}
	apiAgentCanonicalCacheMu.Lock()
	defer apiAgentCanonicalCacheMu.Unlock()
	item, ok := apiAgentCanonicalCache[key]
	if !ok || strings.TrimSpace(item.nombre) == "" || time.Now().After(item.expires) {
		if ok {
			delete(apiAgentCanonicalCache, key)
		}
		return "", false
	}
	return item.nombre, true
}

func apiCanonicalAgentCachePut(raw, nombre string) {
	key := apiCanonicalAgentCacheKey(raw)
	nombre = strings.TrimSpace(nombre)
	if key == "" || nombre == "" {
		return
	}
	apiAgentCanonicalCacheMu.Lock()
	defer apiAgentCanonicalCacheMu.Unlock()
	apiAgentCanonicalCache[key] = apiCanonicalAgentCacheEntry{
		nombre:  nombre,
		expires: time.Now().Add(apiAgentCanonicalCacheTTL),
	}
}

func apiResetAgentCanonicalCache() {
	apiAgentCanonicalCacheMu.Lock()
	defer apiAgentCanonicalCacheMu.Unlock()
	apiAgentCanonicalCache = map[string]apiCanonicalAgentCacheEntry{}
}

func runAPIAgentPrepareLimited(timeout time.Duration, fn func() (*agentesapp.PrepareOutput, error)) (*agentesapp.PrepareOutput, error) {
	if fn == nil {
		return nil, fmt.Errorf("prepare builder no inicializado")
	}
	if timeout <= 0 {
		timeout = apiAgentPrepareTimeout
	}
	deadline := time.Now().Add(timeout)
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case apiAgentPrepareLimiter <- struct{}{}:
	case <-timer.C:
		return nil, errStatusFetchTimeout
	}
	defer func() { <-apiAgentPrepareLimiter }()
	remaining := time.Until(deadline)
	if remaining <= 0 {
		return nil, errStatusFetchTimeout
	}
	type result struct {
		out *agentesapp.PrepareOutput
		err error
	}
	done := make(chan result, 1)
	go func() {
		apiActivePrepareRequests.Add(1)
		defer apiActivePrepareRequests.Add(-1)
		out, err := fn()
		done <- result{out: out, err: err}
	}()
	select {
	case res := <-done:
		return res.out, res.err
	case <-time.After(remaining):
		return nil, errStatusFetchTimeout
	}
}

func runAPIAgentTickLimited(timeout time.Duration, fn func() (*agentesapp.TickOutput, error)) (*agentesapp.TickOutput, error) {
	if fn == nil {
		return nil, fmt.Errorf("tick processor no inicializado")
	}
	if timeout <= 0 {
		timeout = apiAgentTickTimeout
	}
	deadline := time.Now().Add(timeout)
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case apiAgentTickLimiter <- struct{}{}:
	case <-timer.C:
		return nil, errStatusFetchTimeout
	}
	defer func() { <-apiAgentTickLimiter }()
	remaining := time.Until(deadline)
	if remaining <= 0 {
		return nil, errStatusFetchTimeout
	}
	type result struct {
		out *agentesapp.TickOutput
		err error
	}
	done := make(chan result, 1)
	go func() {
		apiActiveTickRequests.Add(1)
		defer apiActiveTickRequests.Add(-1)
		out, err := fn()
		done <- result{out: out, err: err}
	}()
	select {
	case res := <-done:
		return res.out, res.err
	case <-time.After(remaining):
		return nil, errStatusFetchTimeout
	}
}

func agentAPIHotPathBusy() bool {
	return apiActivePrepareRequests.Load() > 0 || apiActiveTickRequests.Load() > 0
}

func agentAPIHotPathBursting() bool {
	return apiActivePrepareRequests.Load()+apiActiveTickRequests.Load() > 1
}

type apiProyectoMicrocicloRequest struct {
	Agente               string `json:"agente"`
	ObjetivoGeneral      string `json:"objetivo_general"`
	DefinitionOfDoneJSON string `json:"definition_of_done_json"`
	Titulo               string `json:"titulo"`
	Descripcion          string `json:"descripcion"`
	Modulo               string `json:"modulo"`
	Notas                string `json:"notas"`
	LimpiarPruebas       bool   `json:"limpiar_pruebas"`
}

type apiProyectoMicrocicloResponse struct {
	Resultado *proyectoMicrocicloResult `json:"resultado"`
}

type apiReviewGateCreateRequest struct {
	Proyecto       string `json:"proyecto"`
	TareaID        *int64 `json:"tarea_id"`
	WorktreeID     *int64 `json:"worktree_id"`
	RequestedBy    string `json:"requested_by"`
	ReviewerAgente string `json:"reviewer_agente"`
	SeverityMax    string `json:"severity_max"`
	FindingsJSON   string `json:"findings_json"`
}

type apiReviewGateUpdateRequest struct {
	Estado         string  `json:"estado"`
	ReviewerAgente *string `json:"reviewer_agente"`
	SeverityMax    *string `json:"severity_max"`
	FindingsJSON   *string `json:"findings_json"`
}

type apiReviewGateResolveRequest struct {
	Estado         string `json:"estado"`
	ReviewerAgente string `json:"reviewer_agente"`
	FindingsJSON   string `json:"findings_json"`
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
	ArranqueLimpio    bool   `json:"arranque_limpio"`
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
	Agente             string `json:"agente"`
	Proyecto           string `json:"proyecto"`
	LimpiarContinuidad bool   `json:"limpiar_continuidad"`
	CWD                string `json:"cwd"`
	Herramienta        string `json:"herramienta"`
	Branch             string `json:"branch"`
	ExternalSessionID  string `json:"external_session_id"`
	ResumePayload      string `json:"resume_payload_json"`
	Resumen            string `json:"resumen_continuidad"`
	Host               string `json:"host"`
	Estado             string `json:"estado"`
	PID                int64  `json:"pid"`
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

type apiTareaLimpiarFrenteRequest struct {
	Agente   string  `json:"agente"`
	Proyecto string  `json:"proyecto"`
	KeepIDs  []int64 `json:"keep_ids"`
}

type apiTareaLimpiarFrenteResponse struct {
	OK        bool                              `json:"ok"`
	Resultado *tareasapp.CleanActiveFrontResult `json:"resultado"`
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

type apiPoolLocalCompartidoSaveRequest struct {
	PoolSlug               string                         `json:"pool_slug"`
	Proveedor              string                         `json:"proveedor"`
	Runtime                string                         `json:"runtime"`
	ModeloPreferente       string                         `json:"modelo_preferente"`
	SlotsMaximos           int                            `json:"slots_maximos"`
	ConectorCanonico       string                         `json:"conector_canonico"`
	ConectorCompatibilidad string                         `json:"conector_compatibilidad"`
	ExperimentalCompat     bool                           `json:"experimental_compat"`
	Perfiles               []capacidadapp.PerfilPoolLocal `json:"perfiles"`
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

type apiGovernanceOverrideSaveRequest struct {
	Actor      string `json:"actor"`
	TipoAgente string `json:"tipo_agente"`
	ScopeTipo  string `json:"scope_tipo"`
	ScopeRef   string `json:"scope_ref"`
	Entidad    string `json:"entidad"`
	EntidadID  int64  `json:"entidad_id"`
	Accion     string `json:"accion"`
}

type apiGovernanceCatalogResponse struct {
	Catalogo *db.GovernanceCatalog `json:"catalogo"`
}

type apiGovernanceOverridesResponse struct {
	Overrides []*db.GovernanceOverride `json:"overrides"`
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

type apiRuntimeOrdersPurgeRequest struct {
	Agente           string   `json:"agente"`
	Proyecto         string   `json:"proyecto"`
	Estados          []string `json:"estados"`
	Tipos            []string `json:"tipos"`
	OlderThanMinutes int      `json:"older_than_minutes"`
	Actor            string   `json:"actor"`
}

type apiRuntimeResidualCloseRequest struct {
	Agente   string `json:"agente"`
	Proyecto string `json:"proyecto"`
	Actor    string `json:"actor"`
}

type apiRuntimeMailboxCreateRequest struct {
	FromAgente     string `json:"from_agente"`
	ToAgente       string `json:"to_agente"`
	Proyecto       string `json:"proyecto"`
	RuntimeOrderID int64  `json:"runtime_order_id"`
	Kind           string `json:"kind"`
	Payload        string `json:"payload"`
}

type apiRuntimeWakeRequest struct {
	Orders  bool `json:"orders"`
	Mailbox bool `json:"mailbox"`
	Warm    bool `json:"warm"`
}

type apiRuntimeWakeResponse struct {
	Orders  bool `json:"orders"`
	Mailbox bool `json:"mailbox"`
	Warm    bool `json:"warm"`
}

type apiRuntimeProcessMailboxResponse struct {
	OK    bool `json:"ok"`
	Count int  `json:"count"`
}

type apiRuntimeProcessOrdersResponse struct {
	OK    bool `json:"ok"`
	Count int  `json:"count"`
}

type apiRuntimeProcessAutonomiaResponse struct {
	OK                        bool `json:"ok"`
	Count                     int  `json:"count"`
	Accepted                  bool `json:"accepted,omitempty"`
	Running                   bool `json:"running,omitempty"`
	GhostAssignmentsCompacted int  `json:"ghost_assignments_compacted,omitempty"`
	ReactivatedWithoutRuntime int  `json:"reactivated_without_runtime,omitempty"`
	IdleAutoassigned          int  `json:"idle_autoassigned,omitempty"`
}

type apiRuntimeProcessAutonomiaRequest struct {
	Wait bool `json:"wait"`
}

type apiRuntimeProcessTranscriptRequest struct {
	Wait     bool   `json:"wait"`
	HandleID int64  `json:"handle_id,omitempty"`
	Agente   string `json:"agente,omitempty"`
	Proyecto string `json:"proyecto,omitempty"`
}

type apiRuntimePurgeTranscriptNoiseRequest struct {
	All bool `json:"all"`
}

type apiRuntimeProcessReanimationsResponse struct {
	OK                bool `json:"ok"`
	Count             int  `json:"count"`
	Accepted          bool `json:"accepted,omitempty"`
	Running           bool `json:"running,omitempty"`
	Candidates        int  `json:"candidates,omitempty"`
	Reactivated       int  `json:"reactivated,omitempty"`
	CooldownSustained int  `json:"cooldown_sustained,omitempty"`
	CapacityBlocked   int  `json:"capacity_blocked,omitempty"`
	Errors            int  `json:"errors,omitempty"`
}

var runtimeProcessAutonomiaBatch = func() (int, error) {
	return (dbAutomationService{}).ProcesarAutonomiaAgentesBatch()
}

var runtimeProcessDegradadosBatch = func() (int, error) {
	return procesarAgentesDegradadosAutonomiaBatchFn()
}

var runtimeProcessHygieneBatch = func() (int, error) {
	return procesarRuntimeHygieneBatch(true)
}

var runtimeProcessTranscriptBatch = func() (int, error) {
	return (dbAutomationService{}).ProcesarRuntimeTranscriptBatch()
}

var runtimeProcessDegradadosBatchDetailed = func() (runtimeProcessDegradadosSummary, error) {
	return procesarAgentesDegradadosAutonomiaBatchDetallado()
}

var runtimeProcessDegradadosWaitTimeout = 3 * time.Second

type apiAgenteResetReanimacionResponse struct {
	OK       bool   `json:"ok"`
	Agente   string `json:"agente"`
	Accepted bool   `json:"accepted,omitempty"`
	Running  bool   `json:"running,omitempty"`
}

var runtimeProcessAutonomiaEnCurso atomic.Bool
var runtimeProcessDegradadosEnCurso atomic.Bool
var runtimeProcessTranscriptEnCurso atomic.Bool
var runtimeProcessReanimacionesEnCurso atomic.Bool
var runtimeProcessReanimationsBatchFn = ejecutarRuntimeProcessReanimationsBatch
var apiAgentResetReanimacionEnCurso sync.Map

type apiRuntimeProcessMailboxRequest struct {
	ToAgente string `json:"to_agente,omitempty"`
	Proyecto string `json:"proyecto,omitempty"`
}

type apiRuntimeMailboxClearRequest struct {
	ToAgente   string   `json:"to_agente"`
	FromAgente string   `json:"from_agente"`
	Proyecto   string   `json:"proyecto"`
	Estados    []string `json:"estados"`
	Kinds      []string `json:"kinds"`
}

type apiRuntimeMailboxClearResponse struct {
	OK         bool    `json:"ok"`
	Cleared    int     `json:"cleared"`
	ClearedIDs []int64 `json:"cleared_ids"`
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

type apiRuntimeA2UIResponse struct {
	Runtime *db.RuntimeInstance     `json:"runtime"`
	A2UI    []apiRuntimeA2UIMessage `json:"a2ui"`
}

type apiRuntimeA2UIMessage struct {
	Creado     string                   `json:"creado"`
	FromAgente string                   `json:"from_agente"`
	Estado     string                   `json:"estado"`
	Component  string                   `json:"component"`
	Error      string                   `json:"error,omitempty"`
	DataTable  *apiRuntimeA2UITable     `json:"data_table,omitempty"`
	Chart      *apiRuntimeA2UIChart     `json:"chart,omitempty"`
	Approval   *a2ui.ApprovalFormProps  `json:"approval,omitempty"`
	Markdown   *a2ui.MarkdownBlockProps `json:"markdown,omitempty"`
}

type apiRuntimeA2UITable struct {
	Title   string     `json:"title"`
	Columns []string   `json:"columns"`
	Rows    [][]string `json:"rows"`
}

type apiRuntimeA2UIChart struct {
	Title     string                   `json:"title"`
	ChartType string                   `json:"chart_type"`
	Series    []string                 `json:"series"`
	Rows      []apiRuntimeA2UIChartRow `json:"rows"`
}

type apiRuntimeA2UIChartRow struct {
	Label  string   `json:"label"`
	Values []string `json:"values"`
}

func registerAPIRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/server", apiHandlerServer)
	mux.HandleFunc("/api/server/operational", apiHandlerServerOperational)
	mux.HandleFunc("/api/mcp", apiHandlerMCP)
	mux.HandleFunc("/api/status", apiHandlerStatus)
	mux.HandleFunc("/api/diagnostico", apiHandlerDiagnostico)
	mux.HandleFunc("/api/agentes", apiHandlerAgentes)
	mux.HandleFunc("/api/agentes/", apiRouterAgentes)
	mux.HandleFunc("/api/agentes/presupuesto/refrescar", apiHandlerAgentesPresupuestoRefrescar)
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
	mux.HandleFunc("/api/gobernanza/catalogo", apiHandlerGobernanzaCatalogo)
	mux.HandleFunc("/api/gobernanza/overrides", apiHandlerGobernanzaOverrides)
	mux.HandleFunc("/api/lenguaje/politica", apiHandlerLenguajePolitica)
	mux.HandleFunc("/api/lenguaje/matriz", apiHandlerLenguajeMatriz)
	mux.HandleFunc("/api/lenguaje/matriz/borrar", apiHandlerLenguajeMatrizBorrar)
	mux.HandleFunc("/api/lenguaje/resolver", apiHandlerLenguajeResolver)
	mux.HandleFunc("/api/config", apiHandlerConfig)
	mux.HandleFunc("/api/notificaciones", apiHandlerNotificaciones)
	mux.HandleFunc("/api/workspace/control", apiHandlerWorkspaceControl)
	mux.HandleFunc("/api/repos/materializar", apiHandlerRepoMaterializar)
	mux.HandleFunc("/api/repos/revisar", apiHandlerRepoRevisar)
	mux.HandleFunc("/api/repos/mejorar", apiHandlerRepoMejorar)
	mux.HandleFunc("/api/openclaw/operator", apiHandlerOpenClawOperator)
	mux.HandleFunc("/api/openclaw/threads", apiHandlerOpenClawThreads)
	mux.HandleFunc("/api/openclaw/pipeline", apiHandlerOpenClawPipeline)
	mux.HandleFunc("/api/audit", apiHandlerAudit)
	mux.HandleFunc("/api/respaldo/bd", apiHandlerRespaldoBD)
	mux.HandleFunc("/api/persistencia/verificar", apiHandlerPersistenciaVerificar)
	mux.HandleFunc("/api/proyectos", apiHandlerProyectos)
	mux.HandleFunc("/api/proyectos/descubrir", apiHandlerProyectoDescubrir)
	mux.HandleFunc("/api/proyectos/", apiRouterProyectos)
	mux.HandleFunc("/api/review-gates", apiHandlerReviewGates)
	mux.HandleFunc("/api/review-gates/", apiRouterReviewGates)
	mux.HandleFunc("/api/pools", apiHandlerPools)
	mux.HandleFunc("/api/pools/seed-inicial", apiHandlerPoolsSeedInicial)
	mux.HandleFunc("/api/pools/modelos/seed-inicial", apiHandlerPoolsModelosSeedInicial)
	mux.HandleFunc("/api/pools/", apiRouterPools)
	mux.HandleFunc("/api/politicas-modelo", apiHandlerPoliticasModelo)
	mux.HandleFunc("/api/politicas-modelo/seed-inicial", apiHandlerPoliticasModeloSeedInicial)
	mux.HandleFunc("/api/modelo/resolver", apiHandlerModeloResolver)
	mux.HandleFunc("/api/modelo/pipeline-local", apiHandlerModeloPipelineLocal)
	mux.HandleFunc("/api/modelo/pipeline-local/paso", apiHandlerModeloPipelineLocalPaso)
	mux.HandleFunc("/api/modelo/pipeline-local/ejecutar", apiHandlerModeloPipelineLocalEjecutar)
	mux.HandleFunc("/api/modelo/pipeline-local/despachar", apiHandlerModeloPipelineLocalDespachar)
	mux.HandleFunc("/api/modelo/runtime-activos", apiHandlerModeloRuntimeActivos)
	mux.HandleFunc("/api/modelo/runtime-descargar", apiHandlerModeloRuntimeDescargar)
	mux.HandleFunc("/api/progreso", apiHandlerProgreso)
	mux.HandleFunc("/api/progreso/fases", apiHandlerProgresoFases)
	mux.HandleFunc("/api/progreso/fases/", apiRouterProgresoFases)
	mux.HandleFunc("/api/progreso/tareas/", apiRouterProgresoTareas)
	mux.HandleFunc("/api/conectores", apiHandlerConectores)
	mux.HandleFunc("/api/conectores/", apiRouterConectores)
	mux.HandleFunc("/api/microprogramacion/especificaciones", apiHandlerMicroprogramacionEspecificaciones)
	mux.HandleFunc("/api/microprogramacion/especificaciones/", apiRouterMicroprogramacionEspecificaciones)
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
	mux.HandleFunc("/api/git/stats", apiHandlerGitStats)
	mux.HandleFunc("/api/git/merges", apiHandlerGitMerges)
	mux.HandleFunc("/api/runtimes", apiHandlerRuntimes)
	mux.HandleFunc("/api/runtimes/tree", apiHandlerRuntimesTree)
	mux.HandleFunc("/api/runtimes/", apiRouterRuntimes)
	mux.HandleFunc("/api/runtime-handles", apiHandlerRuntimeHandles)
	mux.HandleFunc("/api/runtime-handles/cerrar", apiHandlerRuntimeHandlesCerrar)
	mux.HandleFunc("/api/runtime-handles/purgar", apiHandlerRuntimeHandlesPurgar)
	mux.HandleFunc("/api/runtimes/cerrar", apiHandlerRuntimesCerrar)
	mux.HandleFunc("/api/runtime-orders/purgar", apiHandlerRuntimeOrdersPurgar)
	mux.HandleFunc("/api/runtime-orders/cancelar", apiHandlerRuntimeOrdersCancelar)
	mux.HandleFunc("/api/runtime-orders/", apiRouterRuntimeOrders)
	mux.HandleFunc("/api/runtime-trace", apiHandlerRuntimeTrace)
	mux.HandleFunc("/api/runtime-events", apiHandlerRuntimeEvents)
	mux.HandleFunc("/api/runtime-transcript", apiHandlerRuntimeTranscript)
	mux.HandleFunc("/api/runtime-orders", apiHandlerRuntimeOrders)
	mux.HandleFunc("/api/runtime-mailbox", apiHandlerRuntimeMailbox)
	mux.HandleFunc("/api/runtime-mailbox/limpiar", apiHandlerRuntimeMailboxClear)
	mux.HandleFunc("/api/runtime-mailbox/", apiRouterRuntimeMailbox)
	mux.HandleFunc("/api/runtime-checkpoints", apiHandlerRuntimeCheckpoints)
	mux.HandleFunc("/api/runtime-checkpoints/latest", apiHandlerRuntimeCheckpointLatest)
	mux.HandleFunc("/api/runtime-checkpoints/", apiRouterRuntimeCheckpoints)
	mux.HandleFunc("/api/runtime/wake", apiHandlerRuntimeWake)
	mux.HandleFunc("/api/runtime/process-orders", apiHandlerRuntimeProcessOrders)
	mux.HandleFunc("/api/runtime/process-mailbox", apiHandlerRuntimeProcessMailbox)
	mux.HandleFunc("/api/runtime/process-transcript", apiHandlerRuntimeProcessTranscript)
	mux.HandleFunc("/api/runtime/purge-transcript-noise", apiHandlerRuntimePurgeTranscriptNoise)
	mux.HandleFunc("/api/runtime/process-autonomia", apiHandlerRuntimeProcessAutonomia)
	mux.HandleFunc("/api/runtime/process-degradados", apiHandlerRuntimeProcessDegradados)
	mux.HandleFunc("/api/runtime/process-hygiene", apiHandlerRuntimeProcessHygiene)
	mux.HandleFunc("/api/runtime/process-reanimations", apiHandlerRuntimeProcessReanimaciones)
	mux.HandleFunc("/api/runtime/ollama-pool/launch", apiHandlerOllamaPoolLaunch)
	mux.HandleFunc("/api/runtime/ollama-pool/input", apiHandlerOllamaPoolInput)
	mux.HandleFunc("/api/runtime/ollama-pool/status", apiHandlerOllamaPoolStatus)
	mux.HandleFunc("/api/runtime/ollama-pool/stop", apiHandlerOllamaPoolStop)
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
	mux.HandleFunc("/api/agente/lanzar", apiHandlerAgenteLanzar)
	mux.HandleFunc("/api/agente/investigar", apiHandlerAgenteInvestigar)
	mux.HandleFunc("/api/agente/preparar", apiHandlerAgentePreparar)
	mux.HandleFunc("/api/agente/adoptar-contexto", apiHandlerAgenteAdoptarContexto)
	mux.HandleFunc("/api/agente/tick", apiHandlerAgenteTick)
	mux.HandleFunc("/api/agente/pausar", apiHandlerAgentePausar)
}

func apiHandlerMCP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		apiWriteJSON(w, http.StatusOK, map[string]any{
			"name":               mcpServerName,
			"title":              "Orquesta MCP",
			"version":            mcpServerVersion,
			"transport":          "http_jsonrpc",
			"endpoint":           "/api/mcp",
			"protocol_version":   mcpProtocolLatest,
			"supported_versions": mcpSupportedProtocols,
			"modes":              []string{"initialize", "resources", "prompts", "tools"},
			"stateless":          true,
		})
	case http.MethodPost:
		var req mcpRequest
		if err := apiDecodeJSON(r, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		if req.JSONRPC != "2.0" || strings.TrimSpace(req.Method) == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(mcpResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &mcpError{Code: -32600, Message: "Request inválida"},
			})
			return
		}
		result, rpcErr := handleMCPRequest(req, true, mcpHandleOptions{RequireInitialize: false})
		status := http.StatusOK
		resp := mcpResponse{JSONRPC: "2.0", ID: req.ID}
		if rpcErr != nil {
			resp.Error = rpcErr
			if rpcErr.Code == -32600 || rpcErr.Code == -32602 {
				status = http.StatusBadRequest
			}
		} else if req.hasID() {
			resp.Result = result
		} else {
			resp.Result = map[string]any{}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(resp)
	default:
		apiMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func apiHandlerServer(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	info := serverInfo{
		Name:            "orquesta",
		Version:         mcpServerVersion,
		StorageMode:     "single-process",
		StorageDriver:   db.CurrentStorageDriver(),
		SQLPlaceholder:  db.PlaceholderStyle(),
		BootstrapSchema: db.CurrentBootstrapSchemaEnabled(),
		QueryRebinding:  db.QueryRebindingEnabled(),
		Capabilities: []string{
			"status",
			"api",
			"runtimes",
			"control-plane",
			"operational-status",
		},
	}
	apiWriteJSON(w, http.StatusOK, info)
}

func apiHandlerServerOperational(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	if snapshot, ok := readStatusSnapshotFreshUsable(); ok {
		reconcileStatusSnapshotWithFreshPanel(&snapshot)
		apiWriteServerOperational(w, r, buildServerOperationalInfo(snapshot))
		return
	}
	if apiServerOperationalStrictProbe(r) {
		if snapshot, ok := readStatusSnapshotAny(); ok {
			reconcileStatusSnapshotWithFreshPanel(&snapshot)
			ensureStatusRefreshAsync()
			apiWriteServerOperational(w, r, buildServerOperationalInfo(snapshot))
			return
		}
	}
	if apiServerOperationalStatusFallback != nil {
		if status, err := apiServerOperationalStatusFallback(); err == nil {
			reconcileStatusSnapshotWithFreshPanel(&status)
			apiWriteServerOperational(w, r, buildServerOperationalInfo(status))
			return
		}
	}
	if apiServerOperationalBuilder != nil {
		if info, err := runAPITimeboxed(apiServerOperationalTimeout, apiServerOperationalBuilder, errStatusFetchTimeout); err == nil {
			apiWriteServerOperational(w, r, info)
			return
		}
	}
	ensureStatusRefreshAsync()
	if apiServerOperationalDegradedBuilder != nil {
		apiWriteServerOperational(w, r, apiServerOperationalDegradedBuilder())
		return
	}
	apiWriteServerOperational(w, r, degradedServerOperationalInfo())
}

func apiWriteServerOperational(w http.ResponseWriter, r *http.Request, info serverOperationalInfo) {
	info = normalizeServerOperationalInfo(info)
	status := http.StatusOK
	if apiServerOperationalStrictProbe(r) && !info.Operational {
		status = http.StatusServiceUnavailable
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Orquesta-Operational", strconv.FormatBool(info.Operational))
	if state := strings.TrimSpace(info.State); state != "" {
		w.Header().Set("X-Orquesta-Operational-State", state)
	}
	if reason := strings.TrimSpace(info.Reason); reason != "" {
		w.Header().Set("X-Orquesta-Operational-Reason", reason)
	}
	apiWriteJSON(w, status, info)
}

func apiServerOperationalStrictProbe(r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	query := r.URL.Query()
	return parseBoolDebug(query.Get("strict"), false) || parseBoolDebug(query.Get("probe"), false)
}

func apiHandlerWorkspaceControl(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	since, err := parseWorkspaceControlSince(r.URL.Query().Get("desde"))
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	report, err := buildWorkspaceControlReportSince(since)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiWorkspaceControlResponse{Control: report})
}

func apiHandlerStatus(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	if status, err := fetchStatusForAPIAllowDirectFallback(apiStatusFetchTimeout, statusFastTimeout); err == nil {
		writeAPIStatusPayload(w, status)
		return
	}
	ensureStatusRefreshAsync()
	writeAPIStatusPayload(w, degradedAPIStatusResponse())
}

func writeAPIStatusPayload(w http.ResponseWriter, status apiStatusResponse) {
	tareasActivas := normalizarTareasLiteVisibles(status.TareasActivas)
	agentesActivos := status.AgentesActivos
	agentesTrabajando := status.AgentesTrabajando
	autonomia := status.Autonomia
	if autonomia.ByKind == nil {
		autonomia.ByKind = map[string]int{}
	}
	tareasEnProgreso := filtrarOpenClawTareasPorEstado(status.TareasEnProgreso, db.TareaEnProgreso)
	if len(tareasEnProgreso) == 0 {
		tareasEnProgreso = filtrarOpenClawTareasPorEstado(tareasActivas, db.TareaEnProgreso)
	}
	tareasReservadas := filtrarOpenClawTareasPorEstado(status.TareasReservadas, db.TareaAsignada)
	if len(tareasReservadas) == 0 {
		tareasReservadas = filtrarOpenClawTareasPorEstado(tareasActivas, db.TareaAsignada)
	}
	if len(status.Agentes) > 0 && (len(agentesActivos) == 0 || len(agentesTrabajando) == 0) {
		if activos, trabajando, _ := agentesVisiblesLigero(status.Agentes, tareasEnProgreso); len(activos) > 0 || len(trabajando) > 0 {
			activos, trabajando, _ = normalizarAgentesVisiblesStatus(activos, trabajando, nil)
			if len(agentesActivos) == 0 {
				agentesActivos = activos
			}
			if len(agentesTrabajando) == 0 {
				agentesTrabajando = trabajando
			}
		}
	}
	autonomySurface, autonomyHighlights, criticalProjectRisk := normalizeStatusAutonomyPayload(status.AutonomySurface, status.AutonomyHighlights, status.CriticalProjectRisk)
	workersConectados, workersTrabajando, supervisoresActivos := statusVisibleWorkerCounters(agentesActivos, agentesTrabajando, autonomia)
	payload := map[string]any{
		"agentes":              status.Agentes,
		"conteo_tareas":        status.ConteoTareas,
		"resumenTareas":        status.TareasPorEstado,
		"proyectos":            status.Proyectos,
		"asignaciones_activas": status.AsignacionesActivas,
		"sesiones_activas":     status.SesionesActivas,
		"propuestas_abiertas":  status.PropuestasAbiertas,
		"generado":             status.Generado,
		"tareasPorEstado":      status.TareasPorEstado,
		"agentesActivos":       agentesActivos,
		"agentesTrabajando":    agentesTrabajando,
		"agentesSaturados":     status.AgentesSaturados,
		"agentesAtascados":     status.AgentesAtascados,
		"agentesAuthManual":    status.AgentesAuthManual,
		"agentesQuotaBlocked":  status.AgentesQuotaBlocked,
		"propuestasAbiertas":   status.PropuestasResumen,
		"tareasActivas":        tareasActivas,
		"tareasEnProgreso":     tareasEnProgreso,
		"tareasReservadas":     tareasReservadas,
		"poolsLocales":         status.PoolsLocales,
		"deudaDispatch":        status.DeudaDispatch,
		"autonomia":            autonomia,
		"autonomySurface":      autonomySurface,
		"autonomyHighlights":   autonomyHighlights,
		"criticalProjectRisk":  criticalProjectRisk,
		"workersConectados":    workersConectados,
		"workersTrabajando":    workersTrabajando,
		"supervisoresActivos":  supervisoresActivos,
	}
	apiWriteJSON(w, http.StatusOK, payload)
}

func degradedAPIStatusResponse() apiStatusResponse {
	now := time.Now().UTC().Format(time.RFC3339)
	return apiStatusResponse{
		Agentes:             []*db.Agente{},
		ConteoTareas:        map[string]int{},
		ResumenTareas:       map[string]int{},
		Proyectos:           []*db.Proyecto{},
		AsignacionesActivas: map[int64]int{},
		SesionesActivas:     map[int64]int{},
		PropuestasAbiertas:  []*db.Propuesta{},
		Generado:            now,
		TareasPorEstado:     map[string]int{},
		AgentesActivos:      []*db.Agente{},
		AgentesTrabajando:   []*db.Agente{},
		AgentesSaturados:    []*db.Agente{},
		AgentesAtascados:    []*db.Agente{},
		AgentesAuthManual:   []*db.Agente{},
		AgentesQuotaBlocked: []*db.Agente{},
		PropuestasResumen:   []propuestaLite{},
		TareasActivas:       []tareaLite{},
		TareasEnProgreso:    []tareaLite{},
		TareasReservadas:    []tareaLite{},
		PoolsLocales:        []*capacidadapp.PoolLocalCompartido{},
		DeudaDispatch:       deudaDispatchResumen{},
		Autonomia:           autonomiaResumen{ByKind: map[string]int{}},
		AutonomyHighlights:  []string{},
	}
}

func degradedServerOperationalInfo() serverOperationalInfo {
	if apiServerOperationalStatusFallback != nil {
		if snapshot, err := apiServerOperationalStatusFallback(); err == nil {
			return normalizeServerOperationalInfo(buildServerOperationalInfo(snapshot))
		}
	}
	if apiServerOperationalDegradedBuilder != nil {
		return normalizeServerOperationalInfo(apiServerOperationalDegradedBuilder())
	}
	return normalizeServerOperationalInfo(serverOperationalInfo{
		State:       "degraded",
		Operational: false,
		Reason:      "status_temporarily_degraded",
	})
}

func fetchStatusForOperationalFallback(timeout time.Duration) (apiStatusResponse, bool) {
	if snapshot, ok := readStatusSnapshotFreshUsable(); ok {
		return snapshot, true
	}
	if snapshot, ok := readStatusSnapshotAny(); ok && !statusSnapshotNeedsImmediateRefresh(snapshot) {
		return snapshot, true
	}
	if status, err := fetchStatusFallbackRace(timeout); err == nil {
		storeStatusSnapshot(status, statusNowFunc().UTC())
		ensureStatusRefreshAsync()
		return status, true
	}
	if status, ok := fetchStatusReadOnlyLiteDirect(timeout); ok {
		storeStatusSnapshotWithTTL(status, statusNowFunc().UTC(), statusFallbackTTL)
		ensureStatusRefreshAsync()
		return status, true
	}
	return apiStatusResponse{}, false
}

func fetchStatusForAPI(timeout time.Duration) (apiStatusResponse, error) {
	if statusService == nil {
		return apiStatusResponse{}, fmt.Errorf("status service nil")
	}
	return runAPITimeboxed(timeout, statusService.FetchStatus, errStatusFetchTimeout)
}

func fetchStatusForAPIAllowDirectFallback(timeout, fallbackTimeout time.Duration) (apiStatusResponse, error) {
	if cached, ok := readStatusSnapshotFreshUsable(); ok {
		reconcileStatusSnapshotWithFreshPanel(&cached)
		return cached, nil
	}
	if cached, ok := readStatusSnapshotAny(); ok && !statusSnapshotNeedsImmediateRefresh(cached) {
		reconcileStatusSnapshotWithFreshPanel(&cached)
		return cached, nil
	}
	if status, fallbackErr := fetchStatusFallbackRace(fallbackTimeout); fallbackErr == nil {
		storeStatusSnapshot(status, statusNowFunc().UTC())
		ensureStatusRefreshAsync()
		reconcileStatusSnapshotWithFreshPanel(&status)
		return status, nil
	}
	if status, ok := fetchStatusReadOnlyLiteDirect(fallbackTimeout); ok {
		storeStatusSnapshotWithTTL(status, statusNowFunc().UTC(), statusFallbackTTL)
		ensureStatusRefreshAsync()
		reconcileStatusSnapshotWithFreshPanel(&status)
		return status, nil
	}
	if cached, ok := readStatusSnapshotAny(); ok {
		reconcileStatusSnapshotWithFreshPanel(&cached)
		return cached, nil
	}
	return apiStatusResponse{}, errStatusFetchTimeout
}

func fetchStatusFallbackRace(timeout time.Duration) (apiStatusResponse, error) {
	type result struct {
		status apiStatusResponse
		ok     bool
		rich   bool
	}
	if timeout <= 0 {
		timeout = time.Second
	}
	ch := make(chan result, 2)
	launched := 0
	if fetcher := statusFastFetcher; fetcher != nil {
		launched++
		go func() {
			status, err := runStatusFetcherWithTimeout(fetcher, timeout)
			ch <- result{status: status, ok: err == nil, rich: true}
		}()
	}
	if fetcher := apiStatusUltraLiteFetcher; fetcher != nil {
		launched++
		go func() {
			status, ok := fetcher(timeout)
			ch <- result{status: status, ok: ok, rich: false}
		}()
	}
	if launched == 0 {
		return apiStatusResponse{}, errStatusFetchTimeout
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	var graceTimer *time.Timer
	var graceC <-chan time.Time
	defer func() {
		if graceTimer != nil {
			graceTimer.Stop()
		}
	}()
	failures := 0
	var fallback *apiStatusResponse
	for failures < launched {
		select {
		case res := <-ch:
			if res.ok && res.rich {
				return res.status, nil
			}
			if res.ok {
				if fallback == nil {
					status := res.status
					fallback = &status
					if failures+1 >= launched {
						return *fallback, nil
					}
					if apiStatusFallbackRichGrace <= 0 {
						return *fallback, nil
					}
					graceTimer = time.NewTimer(apiStatusFallbackRichGrace)
					graceC = graceTimer.C
				}
				continue
			}
			failures++
			if failures >= launched && fallback != nil {
				return *fallback, nil
			}
		case <-graceC:
			if fallback != nil {
				return *fallback, nil
			}
		case <-timer.C:
			if fallback != nil {
				return *fallback, nil
			}
			return apiStatusResponse{}, errStatusFetchTimeout
		}
	}
	if fallback != nil {
		return *fallback, nil
	}
	return apiStatusResponse{}, errStatusFetchTimeout
}

func fetchStatusUltraLiteFallback(timeout time.Duration) (apiStatusResponse, bool) {
	now := time.Now().UTC()
	type result struct {
		status apiStatusResponse
		ok     bool
	}
	ch := make(chan result, 2)
	launched := 0
	if fetcher := apiStatusReadOnlyLiteFetcher; fetcher != nil {
		launched++
		go func() {
			agentes, cuentas, tareas, err := fetcher()
			if err != nil {
				ch <- result{}
				return
			}
			ch <- result{status: buildUltraLiteStatus(now, agentes, cuentas, tareaLiteFromDB(tareas)), ok: true}
		}()
	}
	launched++
	go func() {
		status, ok := fetchStatusUltraLiteFallbackPrimary(now, timeout)
		ch <- result{status: status, ok: ok}
	}()
	if launched == 0 {
		return apiStatusResponse{}, false
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	failures := 0
	for failures < launched {
		select {
		case res := <-ch:
			if res.ok {
				return res.status, true
			}
			failures++
		case <-timer.C:
			return apiStatusResponse{}, false
		}
	}
	return apiStatusResponse{}, false
}

func fetchStatusReadOnlyLiteDirect(timeout time.Duration) (apiStatusResponse, bool) {
	fetcher := apiStatusReadOnlyLiteFetcher
	if fetcher == nil {
		return apiStatusResponse{}, false
	}
	startedAt := time.Now()
	type liteResult struct {
		agentes []*db.Agente
		cuentas map[string]int
		tareas  []*db.Tarea
	}
	result, err := runAPITimeboxed(timeout, func() (liteResult, error) {
		agentes, cuentas, tareas, err := fetcher()
		if err != nil {
			return liteResult{}, err
		}
		return liteResult{agentes: agentes, cuentas: cuentas, tareas: tareas}, nil
	}, errStatusFetchTimeout)
	if err != nil {
		return apiStatusResponse{}, false
	}
	if statusVisibleSessionsFetcher != nil && statusListAgentsWithSessionsFetcher != nil {
		if remaining := timeout - time.Since(startedAt); remaining > 0 {
			type sessionAgentsResult struct {
				value []*db.Agente
			}
			if enriched, err := runAPITimeboxed(remaining, func() (sessionAgentsResult, error) {
				sesiones, err := statusVisibleSessionsFetcher()
				if err != nil || len(sesiones) == 0 {
					return sessionAgentsResult{}, err
				}
				agentesConSesiones, err := statusListAgentsWithSessionsFetcher(sesiones)
				if err != nil {
					return sessionAgentsResult{}, err
				}
				return sessionAgentsResult{value: agentesConSesiones}, nil
			}, errStatusFetchTimeout); err == nil && len(enriched.value) > 0 {
				result.agentes = enriched.value
			}
		}
	}
	now := time.Now().UTC()
	status := buildUltraLiteStatusReadOnly(now, result.agentes, result.cuentas, tareaLiteFromDB(result.tareas))
	if rows, ok := readAgentPanelSnapshotFresh(); ok {
		activos, trabajando, saturados, atascados, authManual, quotaBlocked, _ := agentesVisiblesPorEstadoOperativoRows(status.Agentes, rows)
		activos, trabajando, saturados = normalizarAgentesVisiblesStatus(activos, trabajando, saturados)
		status.AgentesActivos = activos
		status.AgentesTrabajando = trabajando
		status.AgentesSaturados = saturados
		status.AgentesAtascados = atascados
		status.AgentesAuthManual = authManual
		status.AgentesQuotaBlocked = quotaBlocked
		status.Autonomia = resumirAutonomiaRows(rows, now)
	}
	return status, true
}

func fetchStatusUltraLiteFallbackPrimary(now time.Time, timeout time.Duration) (apiStatusResponse, bool) {
	type agentesResult struct {
		value []*db.Agente
		ok    bool
	}
	type cuentasResult struct {
		value map[string]int
		ok    bool
	}
	agentesCh := make(chan agentesResult, 1)
	cuentasCh := make(chan cuentasResult, 1)
	go func() {
		value, ok := runStatusOptional(timeout, statusListAgentsFetcher)
		agentesCh <- agentesResult{value: value, ok: ok}
	}()
	go func() {
		value, ok := runStatusOptional(timeout, statusCountTasksFetcher)
		cuentasCh <- cuentasResult{value: value, ok: ok}
	}()
	agentesRes := <-agentesCh
	cuentasRes := <-cuentasCh
	agentes, agentesOK := agentesRes.value, agentesRes.ok
	cuentas, cuentasOK := cuentasRes.value, cuentasRes.ok
	if !agentesOK && !cuentasOK {
		return apiStatusResponse{}, false
	}
	return buildUltraLiteStatus(now, agentes, cuentas, nil), true
}

func tareaLiteFromDB(items []*db.Tarea) []tareaLite {
	if len(items) == 0 {
		return nil
	}
	out := make([]tareaLite, 0, len(items))
	for _, tarea := range items {
		if tarea == nil {
			continue
		}
		lite := tareaLite{
			ID:        tarea.ID,
			Titulo:    tarea.Titulo,
			Estado:    tarea.Estado,
			Modulo:    tarea.Modulo,
			Prioridad: tarea.Prioridad,
		}
		if tarea.Agente != nil {
			lite.Agente = *tarea.Agente
		}
		out = append(out, lite)
	}
	return normalizarTareasLiteVisibles(out)
}

func buildUltraLiteStatus(now time.Time, agentes []*db.Agente, cuentas map[string]int, tareasActivas []tareaLite) apiStatusResponse {
	return buildUltraLiteStatusWithAutonomy(now, agentes, cuentas, tareasActivas, true)
}

func buildUltraLiteStatusReadOnly(now time.Time, agentes []*db.Agente, cuentas map[string]int, tareasActivas []tareaLite) apiStatusResponse {
	return buildUltraLiteStatusWithAutonomy(now, agentes, cuentas, tareasActivas, true)
}

func buildUltraLiteStatusWithAutonomy(now time.Time, agentes []*db.Agente, cuentas map[string]int, tareasActivas []tareaLite, includeSupervisorConfig bool) apiStatusResponse {
	status := apiStatusResponse{
		Agentes:             []*db.Agente{},
		ConteoTareas:        map[string]int{},
		ResumenTareas:       map[string]int{},
		Proyectos:           []*db.Proyecto{},
		AsignacionesActivas: map[int64]int{},
		SesionesActivas:     map[int64]int{},
		PropuestasAbiertas:  []*db.Propuesta{},
		Generado:            now.Format(time.RFC3339),
		TareasPorEstado:     map[string]int{},
		AgentesActivos:      []*db.Agente{},
		AgentesTrabajando:   []*db.Agente{},
		AgentesSaturados:    []*db.Agente{},
		AgentesAtascados:    []*db.Agente{},
		AgentesAuthManual:   []*db.Agente{},
		AgentesQuotaBlocked: []*db.Agente{},
		PropuestasResumen:   []propuestaLite{},
		TareasActivas:       []tareaLite{},
		TareasEnProgreso:    []tareaLite{},
		TareasReservadas:    []tareaLite{},
		PoolsLocales:        []*capacidadapp.PoolLocalCompartido{},
		DeudaDispatch:       deudaDispatchResumen{},
		Autonomia:           autonomiaResumen{ByKind: map[string]int{}},
	}
	if agentes != nil {
		status.Agentes = agentes
		status.AgentesQuotaBlocked = agentesNoActivosConCuotaConResumen(agentes, nil)
	}
	if cuentas != nil {
		status.ConteoTareas = cuentas
		status.ResumenTareas = cuentas
		status.TareasPorEstado = cuentas
	}
	if len(tareasActivas) > 0 {
		tareasActivas = normalizarTareasLiteVisibles(tareasActivas)
		status.TareasActivas = tareasActivas
		status.TareasEnProgreso = filtrarOpenClawTareasPorEstado(tareasActivas, db.TareaEnProgreso)
		status.TareasReservadas = filtrarOpenClawTareasPorEstado(tareasActivas, db.TareaAsignada)
		status.TareasPorEstado = reconciliarConteoTareasActivasVisible(status.TareasPorEstado, tareasActivas)
		status.ConteoTareas = status.TareasPorEstado
		status.ResumenTareas = status.TareasPorEstado
	}
	if agentes != nil {
		activos, trabajando, quota := agentesVisiblesLigero(status.Agentes, status.TareasEnProgreso)
		activos, trabajando, _ = normalizarAgentesVisiblesStatus(activos, trabajando, nil)
		status.AgentesActivos = activos
		status.AgentesTrabajando = trabajando
		if len(quota) > 0 {
			status.AgentesQuotaBlocked = quota
		}
		status.Autonomia = resumirAutonomiaLigeraConSupervisor(activos, trabajando, status.TareasEnProgreso, now, includeSupervisorConfig)
		if enriched, err := resumirAutonomiaEventosRecientes(status.Autonomia, now); err == nil {
			status.Autonomia = enriched
		} else if status.Autonomia.ByKind == nil {
			status.Autonomia.ByKind = map[string]int{}
		}
		status.WorkersConectados, status.WorkersTrabajando, status.SupervisoresActivos = statusVisibleWorkerCounters(activos, trabajando, status.Autonomia)
	}
	riskSummary := buildStatusWorkspaceRiskSummary(status.AutonomySurface)
	if out, err := runAPITimeboxed(statusOptionalSectionTimeout, fetchStatusWorkspaceRiskSummary, errStatusFetchTimeout); err == nil {
		riskSummary = mergeStatusWorkspaceRiskSummary(riskSummary, out)
	}
	status.AutonomySurface, riskSummary = canonicalizeStatusAutonomyRisk(status.AutonomySurface, riskSummary)
	status.CriticalProjectRisk = riskSummary.CriticalProjectRisk
	status.AutonomyHighlights = riskSummary.Highlights
	return status
}

func runAPITimeboxed[T any](timeout time.Duration, fn func() (T, error), timeoutErr error) (T, error) {
	var zero T
	if fn == nil {
		return zero, fmt.Errorf("api fn nil")
	}
	if timeout <= 0 {
		return fn()
	}
	type result struct {
		value T
		err   error
	}
	ch := make(chan result, 1)
	go func() {
		value, err := fn()
		ch <- result{value: value, err: err}
	}()
	select {
	case res := <-ch:
		return res.value, res.err
	case <-time.After(timeout):
		if timeoutErr != nil {
			return zero, timeoutErr
		}
		return zero, fmt.Errorf("api timeout")
	}
}

func apiResolveProjectIDTimeboxed(proyectoRef string) (*int64, error) {
	proyectoRef = strings.TrimSpace(proyectoRef)
	if proyectoRef == "" {
		return nil, nil
	}
	proyecto, err := runAPITimeboxed(apiRuntimeProjectLookupTimeout, func() (*db.Proyecto, error) {
		return apiGetRuntimeProjectFn(proyectoRef)
	}, errStatusFetchTimeout)
	if errors.Is(err, errStatusFetchTimeout) {
		proyecto, err = apiProjectLookupPrepareLiteFn(proyectoRef)
	}
	if err != nil {
		return nil, err
	}
	if proyecto == nil {
		return nil, fmt.Errorf("proyecto no encontrado")
	}
	return &proyecto.ID, nil
}

func apiGetProyectoTimeboxed(ref string) (*db.Proyecto, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, fmt.Errorf("referencia de proyecto vacía")
	}
	proyecto, err := runAPITimeboxed(apiRuntimeProjectLookupTimeout, func() (*db.Proyecto, error) {
		return apiProjectLookupFn(ref)
	}, errStatusFetchTimeout)
	if errors.Is(err, errStatusFetchTimeout) {
		return apiProjectLookupPrepareLiteFn(ref)
	}
	return proyecto, err
}

func apiGetProyectoConRutaEfectivaTimeboxed(ref, cwdHint string) (*db.Proyecto, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, fmt.Errorf("referencia de proyecto vacía")
	}
	proyecto, err := runAPITimeboxed(apiRuntimeProjectLookupTimeout, func() (*db.Proyecto, error) {
		return apiProjectLookupWithRouteFn(ref, cwdHint)
	}, errStatusFetchTimeout)
	if errors.Is(err, errStatusFetchTimeout) {
		return apiProjectLookupWithRoutePrepareLiteFn(ref, cwdHint)
	}
	return proyecto, err
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
			rows, err := fetchAgentPanelRowsCached(apiAgentsPanelTimeout)
			if err != nil {
				if errors.Is(err, errStatusFetchTimeout) {
					apiError(w, http.StatusServiceUnavailable, fmt.Errorf("panel temporalmente degradado"))
					return
				}
				apiError(w, http.StatusInternalServerError, err)
				return
			}
			now := time.Now().UTC()
			rows = normalizeAgentPanelRowsForAPI(rows, now)
			if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("schema")), "canonical") {
				apiWriteJSON(w, http.StatusOK, apiAgentesPanelCanonicalResponse{Agents: agentesService.BuildPanelEntities(rows, now)})
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
		agentes = reconcileAPIAgentesWithFreshPanel(agentes)
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
		invalidateAgentStatusCaches()
		apiWriteJSON(w, http.StatusCreated, map[string]any{"ok": true, "nombre": nombre, "rol": rol})
	default:
		apiMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func reconcileAPIAgentesWithFreshPanel(agentes []*db.Agente) []*db.Agente {
	if len(agentes) == 0 {
		return agentes
	}
	rows, ok := readAgentPanelSnapshotFresh()
	if !ok || len(rows) == 0 {
		return agentes
	}
	rowByName := make(map[string]agentesapp.Row, len(rows))
	for _, row := range rows {
		if row.Agente == nil {
			continue
		}
		nombre := nombreAgenteCanonico(row.Agente.Nombre)
		if nombre == "" {
			continue
		}
		rowByName[nombre] = row
	}
	out := make([]*db.Agente, 0, len(agentes))
	for _, agente := range agentes {
		if agente == nil {
			continue
		}
		merged := *agente
		if row, ok := rowByName[nombreAgenteCanonico(agente.Nombre)]; ok && row.Agente != nil {
			if strings.TrimSpace(row.Agente.EstadoCuota) != "" {
				merged.EstadoCuota = row.Agente.EstadoCuota
			}
			if row.Agente.ReanimarAt != nil {
				merged.ReanimarAt = row.Agente.ReanimarAt
			}
			if strings.TrimSpace(row.Agente.MotivoPausa) != "" {
				merged.MotivoPausa = row.Agente.MotivoPausa
			}
			if strings.TrimSpace(row.Agente.EstadoSesion) != "" {
				merged.EstadoSesion = row.Agente.EstadoSesion
			}
			if row.Agente.UltimaSesion != nil && (merged.UltimaSesion == nil || row.Agente.UltimaSesion.After(*merged.UltimaSesion)) {
				merged.UltimaSesion = row.Agente.UltimaSesion
			}
		}
		out = append(out, &merged)
	}
	sanitizeServerOperationalQuotaFromPanelRows(out, rows)
	aplicarVisibilidadOperativaAgentes(out, rows)
	return out
}

func invalidateAgentStatusCaches() {
	resetStatusSnapshotCache()
	resetAgentPanelSnapshotCache()
}

func normalizeAgentPanelRowsForAPI(rows []agentesapp.Row, now time.Time) []agentesapp.Row {
	if len(rows) == 0 {
		return rows
	}
	out := make([]agentesapp.Row, len(rows))
	copy(out, rows)
	for i := range out {
		out[i].MailboxPending = compactMailboxPendingVisibleForAPI(out[i], now)
		if !out[i].EffectiveContinuityPending(now) {
			out[i].MailboxContinuityPending = 0
		}
	}
	return out
}

func compactMailboxPendingVisibleForAPI(row agentesapp.Row, now time.Time) int {
	pending := row.MailboxPending
	if pending <= 0 {
		return 0
	}
	if row.MailboxContinuityPending > 0 && !row.EffectiveContinuityPending(now) {
		pending -= row.MailboxContinuityPending
		if pending < 0 {
			pending = 0
		}
	}
	return pending
}

func apiRouterAgentes(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/agentes/"), "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	if r.Method == http.MethodGet {
		if len(parts) == 1 && parts[0] == "presupuesto" {
			apiHandlerAgentesPresupuesto(w, r)
			return
		}
		if len(parts) == 1 && parts[0] == "cuentas" {
			apiHandlerAgentesCuentas(w, r)
			return
		}
		if len(parts) == 1 && parts[0] == "ranking-cuentas" {
			apiHandlerAgentesRankingCuentas(w, r)
			return
		}
		if len(parts) == 1 && parts[0] == "reanimaciones" {
			apiHandlerAgentesReanimaciones(w, r)
			return
		}
		if len(parts) == 2 && parts[1] == "overview" {
			compact := parseBoolDebug(r.URL.Query().Get("compact"), false)
			detail, err := runAPITimeboxed(apiAgentOverviewTimeout, func() (*agentesapp.Detail, error) {
				return apiAgentDetailBuilder(parts[0], compact)
			}, errStatusFetchTimeout)
			if err != nil {
				if errors.Is(err, errStatusFetchTimeout) {
					apiError(w, http.StatusServiceUnavailable, fmt.Errorf("overview temporalmente degradado"))
					return
				}
				apiError(w, http.StatusNotFound, err)
				return
			}
			apiWriteJSON(w, http.StatusOK, apiAgenteOverviewResponse{Detail: detail})
			return
		}
		if len(parts) == 2 && parts[1] == "actividad" {
			apiHandlerAgenteActividad(w, r, parts[0])
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
	case "observar-cuenta":
		var req apiAgenteObservarCuentaRequest
		if err := apiDecodeJSON(r, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		fuente := strings.TrimSpace(req.Fuente)
		if fuente == "" {
			fuente = "manual_observed_identity"
		}
		if err := agentesService.ObserveAgentIdentity(nombre, req.Email, req.Usuario, fuente, req.ObservedAt); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		invalidateAgentStatusCaches()
	case "retirar":
		if err := agentesService.ApplyStateAction(nombre, "retirar"); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		invalidateAgentStatusCaches()
	case "rehabilitar":
		if err := agentesService.ApplyStateAction(nombre, "rehabilitar"); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		invalidateAgentStatusCaches()
		apiWriteJSON(w, http.StatusOK, launchAgentResetReanimation(nombre))
		return
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
		invalidateAgentStatusCaches()
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
		invalidateAgentStatusCaches()
		apiWriteJSON(w, http.StatusOK, apiAgenteFusionResponse{
			OK:           true,
			Origen:       nombre,
			Destino:      req.Destino,
			RutaRespaldo: rutaRespaldo,
			Resultado:    resultado,
		})
		return
	case "reset-reanimacion":
		apiWriteJSON(w, http.StatusOK, launchAgentResetReanimation(nombre))
		return
	default:
		http.NotFound(w, r)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"ok": true, "agente": nombre})
}

func apiHandlerAgentesReanimaciones(w http.ResponseWriter, r *http.Request) {
	includeFuture := parseBoolDebug(r.URL.Query().Get("all"), false)
	activeOnly := parseBoolDebug(r.URL.Query().Get("activos"), false)
	rows, err := agentesService.BuildReanimationSchedule(includeFuture, activeOnly)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiAgenteReanimationsResponse{Rows: rows})
}

func apiHandlerAgentesPresupuesto(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	activosOnly := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("activos")), "true")
	agenteFiltro := strings.TrimSpace(r.URL.Query().Get("agente"))
	if agenteFiltro != "" {
		agente, err := agentesService.GetAgent(agenteFiltro)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				apiWriteJSON(w, http.StatusOK, apiAgentesPresupuestoResponse{
					Generado: time.Now().UTC().Format(time.RFC3339),
					Activos:  activosOnly,
					Agentes:  []*db.Agente{},
				})
				return
			}
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		agentes := []*db.Agente{agente}
		apiWriteJSON(w, http.StatusOK, apiAgentesPresupuestoResponse{
			Generado: time.Now().UTC().Format(time.RFC3339),
			Activos:  activosOnly,
			Agentes:  agentes,
		})
		return
	}
	agentes, err := agentesService.ListAgents()
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	if activosOnly {
		nombresVisibles, err := nombresSesionesVisibles()
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		filtrados := make([]*db.Agente, 0, len(agentes))
		for _, agente := range agentes {
			if agenteCuentaComoConectado(agente) || (agente != nil && nombresVisibles[strings.ToLower(strings.TrimSpace(agente.Nombre))] && !strings.EqualFold(strings.TrimSpace(agente.EstadoCuota), "enfriamiento") && !strings.EqualFold(strings.TrimSpace(agente.EstadoCuota), "agotado")) {
				filtrados = append(filtrados, agente)
			}
		}
		agentes = filtrados
	}
	apiWriteJSON(w, http.StatusOK, apiAgentesPresupuestoResponse{
		Generado: time.Now().UTC().Format(time.RFC3339),
		Activos:  activosOnly,
		Agentes:  agentes,
	})
}

func apiHandlerAgentesPresupuestoRefrescar(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req struct {
		Agente string `json:"agente"`
	}
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	refrescados, err := runAPITimeboxed(apiAgentBudgetRefreshTimeout, func() (int, error) {
		return apiAgentBudgetRefreshExecutor(req.Agente)
	}, errStatusFetchTimeout)
	if err != nil {
		if errors.Is(err, errStatusFetchTimeout) {
			apiError(w, http.StatusServiceUnavailable, fmt.Errorf("presupuesto temporalmente degradado"))
			return
		}
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{
		"ok":           true,
		"agente":       strings.TrimSpace(req.Agente),
		"refrescados":  refrescados,
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	})
}

func apiHandlerAgentesCuentas(w http.ResponseWriter, r *http.Request) {
	activosOnly := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("activos")), "true")
	agentes, err := db.ListarAgentesCuentasLigero()
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	items := make([]apiAgenteCuentaItem, 0, len(agentes))
	for _, agente := range agentes {
		if agente == nil {
			continue
		}
		if activosOnly && !agente.Activo {
			continue
		}
		items = append(items, apiAgenteCuentaItem{
			Nombre:            agente.Nombre,
			Rol:               agente.Rol,
			Activo:            agente.Activo,
			Habilitado:        agente.Habilitado,
			CuentaID:          strings.TrimSpace(agente.CuentaID),
			CuentaEmail:       strings.TrimSpace(agente.CuentaEmail),
			CuentaUsuario:     strings.TrimSpace(agente.CuentaUsuario),
			CuentaFuente:      strings.TrimSpace(agente.CuentaFuente),
			CuentaObservadaAt: agente.CuentaObservadaAt,
		})
	}
	apiWriteJSON(w, http.StatusOK, apiAgentesCuentasResponse{
		Generado: time.Now().UTC().Format(time.RFC3339),
		Activos:  activosOnly,
		Agentes:  items,
	})
}

func apiHandlerAgentesRankingCuentas(w http.ResponseWriter, r *http.Request) {
	activosOnly := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("activos")), "true")
	agentes, err := agentesService.ListAgents()
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	items := construirRankingCuentasAgentes(agentes, activosOnly)
	apiWriteJSON(w, http.StatusOK, apiAgentesRankingCuentasResponse{
		Generado: time.Now().UTC().Format(time.RFC3339),
		Activos:  activosOnly,
		Cuentas:  items,
	})
}

func construirRankingCuentasAgentes(agentes []*db.Agente, activosOnly bool) []apiCuentaPresupuestoItem {
	ranking := map[string]*apiCuentaPresupuestoItem{}
	for _, agente := range agentes {
		if agente == nil {
			continue
		}
		if activosOnly && !agente.Activo {
			continue
		}
		key, item := cuentaPresupuestoDesdeAgente(agente)
		if key == "" {
			continue
		}
		actual := ranking[key]
		if actual == nil {
			clone := item
			clone.Agentes = append(clone.Agentes, strings.TrimSpace(agente.Nombre))
			ranking[key] = &clone
			continue
		}
		actual.Agentes = append(actual.Agentes, strings.TrimSpace(agente.Nombre))
		sort.Strings(actual.Agentes)
		if cuentaPresupuestoMejor(item, *actual) {
			names := actual.Agentes
			updated := item
			updated.Agentes = names
			ranking[key] = &updated
		}
	}
	out := make([]apiCuentaPresupuestoItem, 0, len(ranking))
	for _, item := range ranking {
		if item == nil {
			continue
		}
		sort.Strings(item.Agentes)
		out = append(out, *item)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return cuentaPresupuestoMejor(out[i], out[j])
	})
	return out
}

func cuentaPresupuestoDesdeAgente(agente *db.Agente) (string, apiCuentaPresupuestoItem) {
	item := apiCuentaPresupuestoItem{}
	if agente == nil {
		return "", item
	}
	cuentaID := strings.TrimSpace(agente.CuentaID)
	email := strings.TrimSpace(agente.CuentaEmail)
	usuario := strings.TrimSpace(agente.CuentaUsuario)
	key := strings.TrimSpace(db.CuentaClaveAgente(agente))
	if key == "" {
		return "", item
	}
	item = apiCuentaPresupuestoItem{
		CuentaClave:           key,
		CuentaID:              cuentaID,
		CuentaEmail:           email,
		CuentaUsuario:         usuario,
		CuentaFuente:          strings.TrimSpace(agente.CuentaFuente),
		CuentaObservadaAt:     agente.CuentaObservadaAt,
		CuotaRestantePct:      agente.CuotaRestantePct,
		PresupuestoVentana:    strings.TrimSpace(agente.PresupuestoVentana),
		PresupuestoResetAt:    agente.PresupuestoResetAt,
		PresupuestoStale:      agente.PresupuestoStale,
		RemainingSeconds:      agente.RemainingSeconds,
		RemainingMessages:     agente.RemainingMessages,
		RemainingTokens:       agente.RemainingTokens,
		RemainingCredits:      agente.RemainingCredits,
		ObservedUsageTokens:   agente.ObservedUsageTokens,
		ObservedUsageCostUSD:  agente.ObservedUsageCostUSD,
		ObservedUsageMessages: agente.ObservedUsageMessages,
		ObservedUsageTurns:    agente.ObservedUsageTurns,
		ObservedUsageAt:       agente.ObservedUsageUpdatedAt,
		ObservedSessionPath:   strings.TrimSpace(agente.ObservedSessionPath),
		PresupuestoFuente:     strings.TrimSpace(agente.PresupuestoFuente),
		PresupuestoCheckedAt:  agente.PresupuestoCheckedAt,
	}
	switch {
	case agente.RemainingTokens != nil:
		item.Criterio = "remaining_tokens"
	case agente.RemainingCredits != nil:
		item.Criterio = "remaining_credits"
	case agente.RemainingMessages != nil:
		item.Criterio = "remaining_messages"
	case agente.RemainingSeconds != nil:
		item.Criterio = "remaining_seconds"
	case agente.CuotaRestantePct != nil:
		item.Criterio = "cuota_pct"
	case agente.ObservedUsageTokens != nil || agente.ObservedUsageCostUSD != nil:
		item.Criterio = "observed_usage"
	default:
		item.Criterio = "sin_datos"
	}
	return key, item
}

func cuentaPresupuestoMejor(a, b apiCuentaPresupuestoItem) bool {
	rankA, valueA := cuentaPresupuestoOrden(a)
	rankB, valueB := cuentaPresupuestoOrden(b)
	if rankA != rankB {
		return rankA > rankB
	}
	if valueA != valueB {
		return valueA > valueB
	}
	timeA := cuentaPresupuestoObservedAt(a)
	timeB := cuentaPresupuestoObservedAt(b)
	if !timeA.Equal(timeB) {
		return timeA.After(timeB)
	}
	return strings.ToLower(strings.TrimSpace(a.CuentaClave)) < strings.ToLower(strings.TrimSpace(b.CuentaClave))
}

func cuentaPresupuestoOrden(item apiCuentaPresupuestoItem) (int, float64) {
	switch item.Criterio {
	case "remaining_tokens":
		if item.RemainingTokens != nil {
			return 5, float64(*item.RemainingTokens)
		}
	case "remaining_credits":
		if item.RemainingCredits != nil {
			return 4, *item.RemainingCredits
		}
	case "remaining_messages":
		if item.RemainingMessages != nil {
			return 3, float64(*item.RemainingMessages)
		}
	case "remaining_seconds":
		if item.RemainingSeconds != nil {
			return 2, float64(*item.RemainingSeconds)
		}
	case "cuota_pct":
		if item.CuotaRestantePct != nil {
			return 1, float64(*item.CuotaRestantePct)
		}
	case "observed_usage":
		if item.ObservedUsageTokens != nil {
			return 0, float64(*item.ObservedUsageTokens)
		}
		if item.ObservedUsageCostUSD != nil {
			return 0, *item.ObservedUsageCostUSD
		}
	}
	return 0, 0
}

func cuentaPresupuestoObservedAt(item apiCuentaPresupuestoItem) time.Time {
	if item.PresupuestoCheckedAt != nil && !item.PresupuestoCheckedAt.IsZero() {
		return item.PresupuestoCheckedAt.UTC()
	}
	if item.CuentaObservadaAt != nil && !item.CuentaObservadaAt.IsZero() {
		return item.CuentaObservadaAt.UTC()
	}
	return time.Time{}
}

func apiHandlerConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if clave := strings.TrimSpace(r.URL.Query().Get("clave")); clave != "" {
			valor, err := runAPITimeboxed(apiConfigTimeout, func() (string, error) {
				return apiConfigGetFn(clave)
			}, errStatusFetchTimeout)
			if err != nil {
				if errors.Is(err, errStatusFetchTimeout) {
					apiError(w, http.StatusServiceUnavailable, fmt.Errorf("config temporalmente degradado"))
					return
				}
				apiError(w, http.StatusNotFound, err)
				return
			}
			apiWriteJSON(w, http.StatusOK, map[string]any{"clave": clave, "valor": valor})
			return
		}
		config, err := runAPITimeboxed(apiConfigTimeout, apiConfigListFn, errStatusFetchTimeout)
		if err != nil {
			if errors.Is(err, errStatusFetchTimeout) {
				apiError(w, http.StatusServiceUnavailable, fmt.Errorf("config temporalmente degradado"))
				return
			}
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
		if _, err := runAPITimeboxed(apiConfigTimeout, func() (struct{}, error) {
			return struct{}{}, apiConfigSetFn(req.Clave, req.Valor)
		}, errStatusFetchTimeout); err != nil {
			if errors.Is(err, errStatusFetchTimeout) {
				apiError(w, http.StatusServiceUnavailable, fmt.Errorf("config temporalmente degradado"))
				return
			}
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{"ok": true, "clave": req.Clave, "valor": req.Valor})
	default:
		apiMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func apiHandlerNotificaciones(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	eventos := []openClawNormalizedEvent{}
	if items, err := runAPITimeboxed(250*time.Millisecond, func() ([]openClawNormalizedEvent, error) {
		return buildOpenClawNormalizedEvents(10)
	}, errStatusFetchTimeout); err == nil {
		eventos = items
	}
	estadoNotifs := notificaciones.EstadoNotificaciones{}
	if estado, err := runAPITimeboxed(100*time.Millisecond, func() (notificaciones.EstadoNotificaciones, error) {
		return notificaciones.DescribirConfiguracion(), nil
	}, errStatusFetchTimeout); err == nil {
		estadoNotifs = estado
	}
	entregas := notificaciones.OutboxSummary{}
	if outbox, err := runAPITimeboxed(100*time.Millisecond, func() (notificaciones.OutboxSummary, error) {
		return notificaciones.DescribirOutbox(10), nil
	}, errStatusFetchTimeout); err == nil {
		entregas = outbox
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{
		"notificaciones":       estadoNotifs,
		"entregas":             entregas,
		"eventos_normalizados": eventos,
	})
}

func apiHandlerOpenClawOperator(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		supervisor := resolveOpenClawOperatorSupervisor(r.URL.Query().Get("supervisor"))
		apiStatus := resolveOpenClawAPIStatus()
		status := buildOpenClawBaseStatusFromAPIStatus(apiStatus)
		revision := buildOpenClawReviewSnapshotSafeWithStatus(supervisor, apiStatus)
		mailboxPendiente, mailboxKnown := openClawPendingMailboxFromReviewSnapshot(revision)
		var panelRows []agentesapp.Row
		if rows, err := runAPITimeboxed(200*time.Millisecond, func() ([]agentesapp.Row, error) {
			return fetchAgentPanelRowsCached(150 * time.Millisecond)
		}, errStatusFetchTimeout); err == nil {
			panelRows = rows
		}
		statusResumen := buildOpenClawOperatorStatusBase(status, panelRows)
		if summary, err := runAPITimeboxed(200*time.Millisecond, func() (apiOpenClawStatusLite, error) {
			return buildOpenClawOperatorStatusWithRowsAndMailbox(status, panelRows, mailboxPendiente, mailboxKnown)
		}, errStatusFetchTimeout); err == nil {
			statusResumen = summary
		}
		eventos := openClawEventsFromReviewSnapshot(revision)
		threads := supervisorThreadsFromReviewSnapshot(revision)
		pipeline := supervisorPipelineFromReviewSnapshot(revision)
		worktreeDrift := openClawWorktreeDriftFromReviewSnapshot(revision)
		subagents := supervisorSubagentsFromReviewSnapshot(revision)
		if len(subagents) == 0 {
			if snapshot, err := runAPITimeboxed(150*time.Millisecond, func() (map[string]any, error) {
				return buildSupervisorSubagentsSnapshot(supervisor, "", "", 100)
			}, errStatusFetchTimeout); err == nil && snapshot != nil {
				subagents = snapshot
			}
		}
		if len(pipeline) == 0 {
			if snapshot, err := runAPITimeboxed(150*time.Millisecond, func() (map[string]any, error) {
				return buildSupervisorPipelineSnapshot(supervisor, "", 20)
			}, errStatusFetchTimeout); err == nil && snapshot != nil {
				pipeline = snapshot
			}
		}
		subagentFollowups := buildOpenClawSubagentFollowupsCompact(subagents)
		if len(subagentFollowups) == 0 {
			subagentFollowups = buildOpenClawSubagentFollowupsDirect(supervisor)
		}
		pipelineFollowup := buildOpenClawPipelineFollowupCompact(pipeline)
		if len(pipelineFollowup) == 0 && len(subagentFollowups) > 0 {
			pipelineFollowup = buildOpenClawPipelineFollowupFromSubagents(subagentFollowups)
		}
		if observed, ok := threads["observed_agent_sessions"].([]*supervisorObservedAgentSessionSummary); ok {
			threads["observed_agent_sessions"] = alignSupervisorObservedSessionsWithStatus(observed, status.AgentesActivos)
		}
		reviewCompact := buildOpenClawReviewCompact(revision)
		queueSummary := buildOpenClawQueueSummaryFromReviewSnapshot(revision)
		sessionCandidates := alignOpenClawSessionCandidatesWithStatus(buildOpenClawSessionCandidates(threads), status.AgentesActivos)
		operationalInfo := buildOpenClawOperationalInfoWithStatus(apiStatus)
		estadoNotifs := notificaciones.EstadoNotificaciones{}
		if estado, err := runAPITimeboxed(100*time.Millisecond, func() (notificaciones.EstadoNotificaciones, error) {
			return notificaciones.DescribirConfiguracion(), nil
		}, errStatusFetchTimeout); err == nil {
			estadoNotifs = estado
		}
		entregas := notificaciones.OutboxSummary{}
		if outbox, err := runAPITimeboxed(100*time.Millisecond, func() (notificaciones.OutboxSummary, error) {
			return notificaciones.DescribirOutbox(10), nil
		}, errStatusFetchTimeout); err == nil {
			entregas = outbox
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{
			"status":                     statusResumen,
			"server_operational":         operationalInfo,
			"server_operational_summary": buildOpenClawOperationalSummary(operationalInfo),
			"agentesActivos":             statusResumen.AgentesActivos,
			"agentesTrabajando":          statusResumen.AgentesTrabajando,
			"agentesAuthManual":          statusResumen.AgentesAuthManual,
			"agentesQuotaBlocked":        statusResumen.AgentesQuotaBlocked,
			"enCuota":                    statusResumen.EnCuota,
			"mailboxPendiente":           statusResumen.MailboxPendiente,
			"retenidasPorCuota":          statusResumen.RetenidasPorCuota,
			"tareasActivas":              statusResumen.TareasActivas,
			"tareasReservadas":           statusResumen.TareasReservadas,
			"propuestasAbiertas":         statusResumen.PropuestasAbiertas,
			"review":                     reviewCompact,
			"next_action":                revision["next_action"],
			"action_queue":               revision["action_queue"],
			"next_safe_action":           revision["next_safe_action"],
			"safe_action_queue":          revision["safe_action_queue"],
			"queue_summary":              queueSummary,
			"capacity_summary":           statusResumen.CapacitySummary,
			"saturated_agents":           statusResumen.AgentesSaturados,
			"notificaciones":             estadoNotifs,
			"entregas":                   entregas,
			"eventos_normalizados":       eventos,
			"thread_sessions":            threads,
			"subagentes":                 subagents["subagents"],
			"subagent_store":             subagents["store"],
			"subagent_profiles":          subagents["tool_profiles"],
			"session_candidates":         sessionCandidates,
			"worktree_drift":             worktreeDrift,
			"pipeline_state":             pipeline,
			"pipeline_followup":          pipelineFollowup,
			"subagentes_followup":        subagentFollowups,
		})
	case http.MethodPost:
		var req struct {
			Supervisor   string `json:"supervisor"`
			Mode         string `json:"mode"`
			Action       string `json:"action"`
			Target       string `json:"target"`
			Assignee     string `json:"assignee"`
			Agente       string `json:"agente"`
			MaxItems     int    `json:"max_items"`
			BatchKind    string `json:"batch_kind"`
			Proyecto     string `json:"proyecto"`
			Name         string `json:"name"`
			Description  string `json:"description"`
			Prompt       string `json:"prompt"`
			SubagentType string `json:"subagent_type"`
			Model        string `json:"model"`
			GateID       int64  `json:"gate_id"`
			Estado       string `json:"estado"`
			Reviewer     string `json:"reviewer_agente"`
			FindingsJSON string `json:"findings_json"`
		}
		if err := apiDecodeJSON(r, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		mode := strings.TrimSpace(strings.ToLower(req.Mode))
		supervisor := resolveOpenClawOperatorSupervisor(req.Supervisor)
		if shouldPrimeOpenClawOperatorStatus(mode) {
			primeOpenClawOperatorStatusSnapshot()
		}
		var (
			result any
			err    error
		)
		switch mode {
		case "action":
			if strings.TrimSpace(req.Action) == "" || strings.TrimSpace(req.Target) == "" {
				apiError(w, http.StatusBadRequest, fmt.Errorf("action y target obligatorios"))
				return
			}
			result, err = applySupervisorRecommendedAction(supervisor, req.Action, req.Target, req.Assignee)
		case "batch":
			result, err = applySupervisorRecommendedActionsBatch(supervisor, req.MaxItems, req.BatchKind)
		case "next":
			result, err = applySupervisorNextAction(supervisor)
		case "agent_action":
			result, err = applyOpenClawAgentAction(req.Agente, req.Action)
		case "review_gate_resolve":
			result, err = resolveOpenClawReviewGate(req.GateID, apiReviewGateResolveRequest{
				Estado:         req.Estado,
				ReviewerAgente: req.Reviewer,
				FindingsJSON:   req.FindingsJSON,
			})
		case "subagent_store_refresh":
			result, err = refreshSupervisorSubagentsFromStore(supervisor, req.Proyecto)
		case "subagent_launch":
			if strings.TrimSpace(req.Description) == "" || strings.TrimSpace(req.Prompt) == "" {
				apiError(w, http.StatusBadRequest, fmt.Errorf("description y prompt obligatorios"))
				return
			}
			result, err = launchClaudeSubagentExternal(supervisorSubagentLaunchRequest{
				Supervisor:   supervisor,
				Proyecto:     req.Proyecto,
				Name:         req.Name,
				Description:  req.Description,
				Prompt:       req.Prompt,
				SubagentType: req.SubagentType,
				Model:        req.Model,
			})
		default:
			apiError(w, http.StatusBadRequest, fmt.Errorf("mode inválido: usa action, batch, next, agent_action, review_gate_resolve, subagent_store_refresh o subagent_launch"))
			return
		}
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, result)
	default:
		apiMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func buildOpenClawBaseStatus() *estadoResumen {
	return buildOpenClawBaseStatusFromAPIStatus(resolveOpenClawAPIStatus())
}

func resolveOpenClawAPIStatus() apiStatusResponse {
	status, err := fetchStatusForAPIAllowDirectFallback(250*time.Millisecond, statusFastTimeout)
	if err == nil {
		return status
	}
	if cached, ok := readStatusSnapshotAny(); ok {
		reconcileStatusSnapshotWithFreshPanel(&cached)
		ensureStatusRefreshAsync()
		return cached
	}
	return degradedAPIStatusResponse()
}

func buildOpenClawBaseStatusFromAPIStatus(status apiStatusResponse) *estadoResumen {
	generado := strings.TrimSpace(status.Generado)
	if generado == "" {
		generado = time.Now().UTC().Format(time.RFC3339)
	}
	tareasActivas := normalizarTareasLiteVisibles(status.TareasActivas)
	agentesActivos := status.AgentesActivos
	agentesTrabajando := status.AgentesTrabajando
	tareasEnProgreso := filtrarOpenClawTareasPorEstado(status.TareasEnProgreso, db.TareaEnProgreso)
	if len(tareasEnProgreso) == 0 {
		tareasEnProgreso = filtrarOpenClawTareasPorEstado(tareasActivas, db.TareaEnProgreso)
	}
	tareasReservadas := filtrarOpenClawTareasPorEstado(status.TareasReservadas, db.TareaAsignada)
	if len(tareasReservadas) == 0 {
		tareasReservadas = filtrarOpenClawTareasPorEstado(tareasActivas, db.TareaAsignada)
	}
	if len(status.Agentes) > 0 && (len(agentesActivos) == 0 || len(agentesTrabajando) == 0) {
		if activos, trabajando, _ := agentesVisiblesLigero(status.Agentes, tareasEnProgreso); len(activos) > 0 || len(trabajando) > 0 {
			activos, trabajando, _ = normalizarAgentesVisiblesStatus(activos, trabajando, nil)
			if len(agentesActivos) == 0 {
				agentesActivos = activos
			}
			if len(agentesTrabajando) == 0 {
				agentesTrabajando = trabajando
			}
		}
	}
	return &estadoResumen{
		Generado:            generado,
		Agentes:             status.Agentes,
		TareasPorEstado:     status.TareasPorEstado,
		AgentesActivos:      agentesActivos,
		AgentesTrabajando:   agentesTrabajando,
		AgentesSaturados:    status.AgentesSaturados,
		AgentesAtascados:    status.AgentesAtascados,
		AgentesAuthManual:   status.AgentesAuthManual,
		AgentesQuotaBlocked: status.AgentesQuotaBlocked,
		PropuestasAbiertas:  status.PropuestasResumen,
		TareasActivas:       tareasActivas,
		TareasEnProgreso:    tareasEnProgreso,
		TareasReservadas:    tareasReservadas,
		PoolsLocales:        status.PoolsLocales,
		DeudaDispatch:       status.DeudaDispatch,
		Autonomia:           status.Autonomia,
		WorkersConectados:   status.WorkersConectados,
		WorkersTrabajando:   status.WorkersTrabajando,
		SupervisoresActivos: status.SupervisoresActivos,
	}
}

func buildOpenClawReviewSnapshotSafe(supervisor string) map[string]any {
	if snapshot, err := runAPITimeboxed(350*time.Millisecond, func() (map[string]any, error) {
		return buildSupervisorReviewSnapshot(supervisor)
	}, errStatusFetchTimeout); err == nil && snapshot != nil {
		return snapshot
	}
	return map[string]any{}
}

func buildOpenClawReviewSnapshotSafeWithStatus(supervisor string, status apiStatusResponse) map[string]any {
	if snapshot, err := runAPITimeboxed(350*time.Millisecond, func() (map[string]any, error) {
		return buildSupervisorReviewSnapshotWithStatus(supervisor, status)
	}, errStatusFetchTimeout); err == nil && snapshot != nil {
		return snapshot
	}
	return map[string]any{}
}

func resolveOpenClawOperatorSupervisor(supervisor string) string {
	if strings.TrimSpace(supervisor) == "" {
		return "OpenClaw"
	}
	return resolveSupervisorName(supervisor)
}

func shouldPrimeOpenClawOperatorStatus(mode string) bool {
	switch strings.TrimSpace(strings.ToLower(mode)) {
	case "action", "batch", "next", "review_gate_resolve":
		return true
	default:
		return false
	}
}

func primeOpenClawOperatorStatusSnapshot() {
	status, err := runAPITimeboxed(200*time.Millisecond, statusService.FetchStatus, errStatusFetchTimeout)
	if err != nil {
		return
	}
	storeStatusSnapshot(status, statusNowFunc().UTC())
}

func applyOpenClawAgentAction(agent, action string) (map[string]any, error) {
	agent = strings.TrimSpace(agent)
	action = strings.TrimSpace(action)
	if agent == "" || !webAccionEstadoAgenteValida(action) {
		return nil, fmt.Errorf("accion de agente invalida")
	}
	if strings.EqualFold(action, "reset-reanimacion") {
		resultado, err := (dbAutomationService{}).resetReanimacionResultadoConPermisoManual(agent, true)
		if err != nil {
			return nil, err
		}
		invalidateAgentStatusCaches()
		return map[string]any{
			"ok":     true,
			"agente": agent,
			"accion": action,
			"reset":  resultado,
		}, nil
	}
	if err := agentesService.ApplyStateAction(agent, action); err != nil {
		return nil, err
	}
	invalidateAgentStatusCaches()
	return map[string]any{
		"ok":     true,
		"agente": agent,
		"accion": action,
	}, nil
}

func resolveOpenClawReviewGate(id int64, req apiReviewGateResolveRequest) (map[string]any, error) {
	if id <= 0 || strings.TrimSpace(req.Estado) == "" {
		return nil, fmt.Errorf("review gate invalido")
	}
	gate, err := reviewService.Resolve(reviewapp.ResolveGateInput{
		ID:             id,
		Estado:         strings.TrimSpace(req.Estado),
		ReviewerAgente: strings.TrimSpace(req.ReviewerAgente),
		FindingsJSON:   strings.TrimSpace(req.FindingsJSON),
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"gate": gate}, nil
}

func buildOpenClawReviewCompact(review map[string]any) map[string]any {
	if len(review) == 0 {
		return map[string]any{}
	}
	compact := map[string]any{
		"supervisor": review["supervisor"],
	}
	for _, key := range []string{"review_gates", "signals", "merges", "module_conflicts"} {
		if value, ok := review[key]; ok {
			compact[key] = value
		}
	}
	return compact
}

var openClawPendingMailboxFetcher = buildOpenClawPendingMailbox

type apiOpenClawAgentLite struct {
	Nombre             string                 `json:"nombre"`
	Rol                string                 `json:"rol,omitempty"`
	EstadoCuota        string                 `json:"estado_cuota,omitempty"`
	ReanimarAt         *time.Time             `json:"reanimar_at,omitempty"`
	CuentaID           string                 `json:"cuenta_id,omitempty"`
	CuentaEmail        string                 `json:"cuenta_email,omitempty"`
	CuentaUsuario      string                 `json:"cuenta_usuario,omitempty"`
	PresupuestoVentana string                 `json:"presupuesto_ventana,omitempty"`
	CuotaRestantePct   *int                   `json:"cuota_restante_pct,omitempty"`
	CargaActiva        int                    `json:"carga_activa,omitempty"`
	CargaReservada     int                    `json:"carga_reservada,omitempty"`
	OperationalState   string                 `json:"operational_state,omitempty"`
	OperationalDetail  string                 `json:"operational_detail,omitempty"`
	CurrentTask        *agentesapp.TaskFocus  `json:"current_task,omitempty"`
	DominantOrder      *agentesapp.OrderFocus `json:"dominant_order,omitempty"`
}

type apiOpenClawStatusLite struct {
	Generado            string                     `json:"generado,omitempty"`
	TareasPorEstado     map[string]int             `json:"tareasPorEstado,omitempty"`
	DeudaDispatch       deudaDispatchResumen       `json:"deudaDispatch,omitempty"`
	AgentesActivos      []apiOpenClawAgentLite     `json:"agentesActivos,omitempty"`
	AgentesTrabajando   []apiOpenClawAgentLite     `json:"agentesTrabajando,omitempty"`
	AgentesAuthManual   []apiOpenClawAgentLite     `json:"agentesAuthManual,omitempty"`
	AgentesQuotaBlocked []apiOpenClawAgentLite     `json:"agentesQuotaBlocked,omitempty"`
	EnCuota             []apiOpenClawAgentLite     `json:"enCuota,omitempty"`
	MailboxPendiente    []apiOpenClawMailboxLite   `json:"mailboxPendiente"`
	RetenidasPorCuota   []tareaLite                `json:"retenidasPorCuota,omitempty"`
	TareasActivas       []tareaLite                `json:"tareasActivas,omitempty"`
	TareasReservadas    []tareaLite                `json:"tareasReservadas,omitempty"`
	PropuestasAbiertas  []propuestaLite            `json:"propuestasAbiertas,omitempty"`
	CapacitySummary     apiOpenClawCapacitySummary `json:"capacity_summary,omitempty"`
	AgentesSaturados    []apiOpenClawAgentLite     `json:"agentesSaturados,omitempty"`
}

type apiOpenClawMailboxLite struct {
	Agente               string     `json:"agente"`
	Count                int        `json:"count"`
	Kinds                []string   `json:"kinds,omitempty"`
	KindsCSV             string     `json:"kinds_csv,omitempty"`
	SupervisorActions    []string   `json:"supervisor_actions,omitempty"`
	SupervisorActionsCSV string     `json:"supervisor_actions_csv,omitempty"`
	Contexts             []string   `json:"contexts,omitempty"`
	ContextsCSV          string     `json:"contexts_csv,omitempty"`
	OldestCreatedAt      *time.Time `json:"oldest_created_at,omitempty"`
	OldestAgeMin         int        `json:"oldest_age_min,omitempty"`
}

type apiOpenClawQueueSummary struct {
	Total      int            `json:"total"`
	Safe       int            `json:"safe"`
	Manual     int            `json:"manual"`
	SafeByKind map[string]int `json:"safe_by_kind,omitempty"`
}

type apiOpenClawSessionCandidate struct {
	Agente              string `json:"agente"`
	Activo              bool   `json:"activo"`
	Herramienta         string `json:"herramienta,omitempty"`
	Host                string `json:"host,omitempty"`
	ExternalSessionID   string `json:"external_session_id,omitempty"`
	ObservedSessionPath string `json:"observed_session_path,omitempty"`
	UsageSummary        string `json:"usage_summary,omitempty"`
}

type apiOpenClawCapacitySummary struct {
	WorkersConectados  int `json:"workers_conectados"`
	WorkersOciosos     int `json:"workers_ociosos"`
	WorkersDisponibles int `json:"workers_disponibles"`
	WorkersSaturados   int `json:"workers_saturados"`
	CapacidadLibre     int `json:"capacidad_libre"`
	BacklogLibre       int `json:"backlog_libre"`
}

func buildOpenClawOperatorStatus(status *estadoResumen) (apiOpenClawStatusLite, error) {
	return buildOpenClawOperatorStatusWithRows(status, nil)
}

func buildOpenClawOperatorStatusBase(status *estadoResumen, rows []agentesapp.Row) apiOpenClawStatusLite {
	if status == nil {
		return apiOpenClawStatusLite{MailboxPendiente: []apiOpenClawMailboxLite{}}
	}
	return apiOpenClawStatusLite{
		Generado:            status.Generado,
		TareasPorEstado:     status.TareasPorEstado,
		DeudaDispatch:       status.DeudaDispatch,
		AgentesActivos:      compactOpenClawAgents(status.AgentesActivos, status.TareasActivas, rows),
		AgentesTrabajando:   compactOpenClawAgents(status.AgentesTrabajando, status.TareasActivas, rows),
		AgentesAuthManual:   compactOpenClawAgents(status.AgentesAuthManual, status.TareasActivas, rows),
		AgentesQuotaBlocked: compactOpenClawAgents(status.AgentesQuotaBlocked, status.TareasActivas, rows),
		EnCuota:             compactOpenClawAgents(agentesBloqueadosPorCuotaVisibles(status), status.TareasActivas, rows),
		MailboxPendiente:    []apiOpenClawMailboxLite{},
		RetenidasPorCuota:   tareasRetenidasPorCuota(status.TareasActivas, status.Agentes, status.AgentesQuotaBlocked),
		TareasActivas:       filtrarOpenClawTareasPorEstado(status.TareasActivas, db.TareaEnProgreso),
		TareasReservadas:    filtrarOpenClawTareasPorEstado(status.TareasActivas, db.TareaAsignada),
		PropuestasAbiertas:  status.PropuestasAbiertas,
		CapacitySummary:     buildOpenClawCapacitySummary(status.AgentesActivos, status.AgentesTrabajando, status.TareasActivas, status.TareasPorEstado),
		AgentesSaturados:    buildOpenClawSaturatedAgents(status.AgentesActivos, status.TareasActivas, rows),
	}
}

func buildOpenClawOperatorStatusWithRows(status *estadoResumen, rows []agentesapp.Row) (apiOpenClawStatusLite, error) {
	return buildOpenClawOperatorStatusWithRowsAndMailbox(status, rows, nil, false)
}

func buildOpenClawOperatorStatusWithRowsAndMailbox(status *estadoResumen, rows []agentesapp.Row, mailbox []apiOpenClawMailboxLite, mailboxKnown bool) (apiOpenClawStatusLite, error) {
	resumen := buildOpenClawOperatorStatusBase(status, rows)
	if status == nil {
		return resumen, nil
	}
	if mailboxKnown {
		resumen.MailboxPendiente = normalizeOpenClawMailboxLiteSlice(mailbox)
		return resumen, nil
	}
	mailboxPendiente, err := openClawPendingMailboxFetcher(status.Agentes)
	if err != nil {
		return resumen, err
	}
	resumen.MailboxPendiente = normalizeOpenClawMailboxLiteSlice(mailboxPendiente)
	return resumen, nil
}

func openClawPendingMailboxFromReviewSnapshot(review map[string]any) ([]apiOpenClawMailboxLite, bool) {
	if len(review) == 0 {
		return nil, false
	}
	items, ok := review["mailbox_pending"].([]apiOpenClawMailboxLite)
	if !ok {
		return nil, false
	}
	return normalizeOpenClawMailboxLiteSlice(items), true
}

func normalizeOpenClawMailboxLiteSlice(items []apiOpenClawMailboxLite) []apiOpenClawMailboxLite {
	if items == nil {
		return []apiOpenClawMailboxLite{}
	}
	return items
}

func buildOpenClawSessionCandidates(snapshot map[string]any) []apiOpenClawSessionCandidate {
	items, _ := snapshot["observed_agent_sessions"].([]*supervisorObservedAgentSessionSummary)
	if len(items) == 0 {
		return nil
	}
	out := make([]apiOpenClawSessionCandidate, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, apiOpenClawSessionCandidate{
			Agente:              strings.TrimSpace(item.Agente),
			Activo:              item.Activo,
			Herramienta:         strings.TrimSpace(item.Herramienta),
			Host:                strings.TrimSpace(item.Host),
			ExternalSessionID:   strings.TrimSpace(item.ExternalSessionID),
			ObservedSessionPath: strings.TrimSpace(item.ObservedSessionPath),
			UsageSummary:        strings.TrimSpace(item.UsageSummary),
		})
	}
	return out
}

func alignSupervisorObservedSessionsWithStatus(items []*supervisorObservedAgentSessionSummary, activos []*db.Agente) []*supervisorObservedAgentSessionSummary {
	if len(items) == 0 {
		return items
	}
	activosSet := make(map[string]struct{}, len(activos))
	for _, agente := range activos {
		if agente == nil {
			continue
		}
		nombre := strings.ToLower(strings.TrimSpace(agente.Nombre))
		if nombre == "" {
			continue
		}
		activosSet[nombre] = struct{}{}
	}
	out := make([]*supervisorObservedAgentSessionSummary, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		copyItem := *item
		_, ok := activosSet[strings.ToLower(strings.TrimSpace(copyItem.Agente))]
		copyItem.Activo = ok
		out = append(out, &copyItem)
	}
	return out
}

func alignOpenClawSessionCandidatesWithStatus(candidates []apiOpenClawSessionCandidate, activos []*db.Agente) []apiOpenClawSessionCandidate {
	if len(candidates) == 0 {
		return candidates
	}
	activosSet := make(map[string]struct{}, len(activos))
	for _, agente := range activos {
		if agente == nil {
			continue
		}
		nombre := strings.ToLower(strings.TrimSpace(agente.Nombre))
		if nombre == "" {
			continue
		}
		activosSet[nombre] = struct{}{}
	}
	out := make([]apiOpenClawSessionCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		item := candidate
		_, ok := activosSet[strings.ToLower(strings.TrimSpace(item.Agente))]
		item.Activo = ok
		out = append(out, item)
	}
	return out
}

func buildOpenClawQueueSummary(review map[string]any) apiOpenClawQueueSummary {
	var allActions []supervisorRecommendedAction
	if items, ok := review["action_queue"].([]supervisorRecommendedAction); ok {
		allActions = items
	}
	var safeActions []supervisorRecommendedAction
	if items, ok := review["safe_action_queue"].([]supervisorRecommendedAction); ok {
		safeActions = items
	}
	return buildOpenClawQueueSummaryFromActions(allActions, safeActions)
}

func buildOpenClawQueueSummaryFromReviewSnapshot(review map[string]any) apiOpenClawQueueSummary {
	if review == nil {
		return apiOpenClawQueueSummary{}
	}
	if summary, ok := review["queue_summary"].(apiOpenClawQueueSummary); ok {
		return summary
	}
	return buildOpenClawQueueSummary(review)
}

func openClawEventsFromReviewSnapshot(review map[string]any) []openClawNormalizedEvent {
	if review == nil {
		return nil
	}
	if events, ok := review["normalized_events"].([]openClawNormalizedEvent); ok {
		return events
	}
	return nil
}

func supervisorThreadsFromReviewSnapshot(review map[string]any) map[string]any {
	if review == nil {
		return map[string]any{}
	}
	if threads, ok := review["thread_sessions"].(map[string]any); ok && threads != nil {
		return threads
	}
	return map[string]any{}
}

func supervisorSubagentsFromReviewSnapshot(review map[string]any) map[string]any {
	if snapshot, ok := review["subagents"].(map[string]any); ok {
		return snapshot
	}
	return map[string]any{}
}

func supervisorPipelineFromReviewSnapshot(review map[string]any) map[string]any {
	if review == nil {
		return map[string]any{}
	}
	if pipeline, ok := review["pipeline_state"].(map[string]any); ok && pipeline != nil {
		return pipeline
	}
	return map[string]any{}
}

func buildOpenClawSubagentFollowupsCompact(snapshot map[string]any) []map[string]any {
	items, _ := snapshot["subagents"].([]*db.SupervisorSubagent)
	return buildOpenClawSubagentFollowupsFromItems(items)
}

func buildOpenClawSubagentFollowupsDirect(supervisor string) []map[string]any {
	items, err := db.ListarSupervisorSubagents(db.FiltroSupervisorSubagents{
		Supervisor: resolveSupervisorName(supervisor),
		Status:     "completed",
		Limit:      100,
	})
	if err != nil {
		return nil
	}
	return buildOpenClawSubagentFollowupsFromItems(items)
}

func buildOpenClawSubagentFollowupsFromItems(items []*db.SupervisorSubagent) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		metadata := metadataSupervisorSubagente(item)
		if len(metadata) == 0 || !strings.EqualFold(strings.TrimSpace(stringSupervisorSubagente(metadata["source"])), "pipeline_local_parallel") {
			continue
		}
		compact := map[string]any{
			"id":            item.ID,
			"thread_id":     strings.TrimSpace(item.ThreadID),
			"subagent_name": strings.TrimSpace(item.SubagentName),
			"status":        strings.TrimSpace(item.Status),
			"source":        "pipeline_local_parallel",
		}
		if idx := int64SupervisorSubagente(metadata["slice_index"]); idx > 0 {
			compact["slice_index"] = idx
		}
		if total := int64SupervisorSubagente(metadata["slice_total"]); total > 0 {
			compact["slice_total"] = total
		}
		if taskID := int64SupervisorSubagente(metadata["task_id"]); taskID > 0 {
			compact["task_id"] = taskID
		}
		if taskTitle := strings.TrimSpace(stringSupervisorSubagente(metadata["task_title"])); taskTitle != "" {
			compact["task_title"] = taskTitle
		}
		if phase := strings.TrimSpace(stringSupervisorSubagente(metadata["pipeline_parent_followup_phase"])); phase != "" {
			compact["followup_phase"] = phase
		}
		if action := strings.TrimSpace(stringSupervisorSubagente(metadata["pipeline_parent_followup_action"])); action != "" {
			compact["followup_action"] = action
		}
		if mergeID := int64SupervisorSubagente(metadata["pipeline_parent_followup_git_merge_id"]); mergeID > 0 {
			compact["git_merge_id"] = mergeID
		}
		if dispatched := sidecarPipelineFollowupYaDespachado(metadata); dispatched {
			compact["followup_dispatched"] = true
		}
		out = append(out, compact)
	}
	return out
}

func buildOpenClawPipelineFollowupCompact(snapshot map[string]any) map[string]any {
	items, _ := snapshot["pipelines"].([]*db.SupervisorPipelineState)
	return buildOpenClawPipelineFollowupFromPipelineItems(items)
}

func buildOpenClawPipelineFollowupFromPipelineItems(items []*db.SupervisorPipelineState) map[string]any {
	for _, item := range items {
		if item == nil || strings.TrimSpace(item.MetadataJSON) == "" {
			continue
		}
		var metadata map[string]any
		if err := json.Unmarshal([]byte(strings.TrimSpace(item.MetadataJSON)), &metadata); err != nil || len(metadata) == 0 {
			continue
		}
		followup, _ := metadata["latest_parallel_sidecar_followup"].(map[string]any)
		if len(followup) == 0 {
			continue
		}
		compact := map[string]any{
			"pipeline_name": strings.TrimSpace(item.PipelineName),
			"proyecto":      strings.TrimSpace(item.ProyectoSlug),
		}
		for _, key := range []string{"phase", "action", "task_id", "task_title", "git_merge_id", "slice_index", "slice_total", "subagent_id", "thread_id", "updated_at"} {
			if value, ok := followup[key]; ok {
				compact[key] = value
			}
		}
		return compact
	}
	return map[string]any{}
}

func buildOpenClawPipelineFollowupFromSubagents(items []map[string]any) map[string]any {
	for _, item := range items {
		if len(item) == 0 {
			continue
		}
		compact := map[string]any{}
		for _, key := range []string{"task_id", "task_title", "git_merge_id", "slice_index", "slice_total", "thread_id"} {
			if value, ok := item[key]; ok {
				compact[key] = value
			}
		}
		if phase, ok := item["followup_phase"]; ok {
			compact["phase"] = phase
		}
		if action, ok := item["followup_action"]; ok {
			compact["action"] = action
		}
		return compact
	}
	return map[string]any{}
}

func openClawWorktreeDriftFromReviewSnapshot(review map[string]any) []apiOpenClawWorktreeDrift {
	if review == nil {
		return nil
	}
	if drift, ok := review["worktree_drift"].([]apiOpenClawWorktreeDrift); ok {
		return drift
	}
	return nil
}

func buildOpenClawQueueSummaryFromActions(allActions, safeActions []supervisorRecommendedAction) apiOpenClawQueueSummary {
	total := len(allActions)
	safe := len(safeActions)
	manual := total - safe
	if manual < 0 {
		manual = 0
	}
	safeByKind := make(map[string]int)
	for _, item := range safeActions {
		kind := normalizeSupervisorBatchKind(item.Kind)
		if kind == "" {
			continue
		}
		safeByKind[kind]++
	}
	return apiOpenClawQueueSummary{
		Total:      total,
		Safe:       safe,
		Manual:     manual,
		SafeByKind: safeByKind,
	}
}

func buildOpenClawCapacitySummary(agentesActivos, agentesTrabajando []*db.Agente, tareas []tareaLite, tareasPorEstado map[string]int) apiOpenClawCapacitySummary {
	workersActivos := visibleNonSupervisorAgents(agentesActivos)
	workersTrabajando := visibleNonSupervisorAgents(agentesTrabajando)
	idle := idleSupervisorWorkers(workersActivos, workersTrabajando)
	saturated := buildOpenClawSaturatedAgents(workersActivos, tareas, nil)
	connected := len(workersActivos)
	idleCount := len(idle)
	saturatedCount := len(saturated)
	available := connected - saturatedCount
	if available < 0 {
		available = 0
	}
	backlogLibre := 0
	if tareasPorEstado != nil {
		backlogLibre = tareasPorEstado[string(db.TareaLibre)]
	}
	return apiOpenClawCapacitySummary{
		WorkersConectados:  connected,
		WorkersOciosos:     idleCount,
		WorkersDisponibles: available,
		WorkersSaturados:   saturatedCount,
		CapacidadLibre:     idleCount,
		BacklogLibre:       backlogLibre,
	}
}

func buildOpenClawSaturatedAgents(items []*db.Agente, tareas []tareaLite, rows []agentesapp.Row) []apiOpenClawAgentLite {
	agents := compactOpenClawAgents(items, tareas, rows)
	out := make([]apiOpenClawAgentLite, 0, len(agents))
	for _, agent := range agents {
		if agent.CargaReservada > 0 || agent.CargaActiva >= 3 {
			out = append(out, agent)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CargaReservada != out[j].CargaReservada {
			return out[i].CargaReservada > out[j].CargaReservada
		}
		if out[i].CargaActiva != out[j].CargaActiva {
			return out[i].CargaActiva > out[j].CargaActiva
		}
		return strings.TrimSpace(out[i].Nombre) < strings.TrimSpace(out[j].Nombre)
	})
	return out
}

func filtrarOpenClawTareasPorEstado(items []tareaLite, estados ...db.EstadoTarea) []tareaLite {
	if len(items) == 0 || len(estados) == 0 {
		return nil
	}
	permitidos := make(map[db.EstadoTarea]struct{}, len(estados))
	for _, estado := range estados {
		permitidos[estado] = struct{}{}
	}
	out := make([]tareaLite, 0, len(items))
	for _, item := range items {
		if !tareaLiteValida(item) {
			continue
		}
		if _, ok := permitidos[item.Estado]; ok {
			out = append(out, item)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func buildOpenClawPendingMailbox(agentes []*db.Agente) ([]apiOpenClawMailboxLite, error) {
	estado := "pendiente"
	items, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{Estado: &estado})
	if err != nil {
		return nil, err
	}
	out := buildOpenClawMailboxLiteFromItems(items, nil)
	sortOpenClawMailboxLiteByCount(out)
	return out, nil
}

func compactOpenClawAgents(items []*db.Agente, tareas []tareaLite, rows []agentesapp.Row) []apiOpenClawAgentLite {
	cargaActiva, cargaReservada := buildOpenClawAgentLoad(tareas)
	rowByAgent := map[string]agentesapp.Row{}
	for _, row := range rows {
		if row.Agente == nil {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(row.Agente.Nombre))
		if name == "" {
			continue
		}
		rowByAgent[name] = row
	}
	out := make([]apiOpenClawAgentLite, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		nombre := strings.TrimSpace(item.Nombre)
		row := rowByAgent[strings.ToLower(nombre)]
		out = append(out, apiOpenClawAgentLite{
			Nombre:             item.Nombre,
			Rol:                item.Rol,
			EstadoCuota:        item.EstadoCuota,
			ReanimarAt:         item.ReanimarAt,
			CuentaID:           item.CuentaID,
			CuentaEmail:        item.CuentaEmail,
			CuentaUsuario:      item.CuentaUsuario,
			PresupuestoVentana: item.PresupuestoVentana,
			CuotaRestantePct:   item.CuotaRestantePct,
			CargaActiva:        cargaActiva[nombre],
			CargaReservada:     cargaReservada[nombre],
			OperationalState:   strings.TrimSpace(row.EstadoOperativo),
			OperationalDetail:  strings.TrimSpace(row.DetalleOperativo),
			CurrentTask:        cloneOpenClawTaskFocus(row.CurrentTask),
			DominantOrder:      cloneOpenClawOrderFocus(row.DominantOrder),
		})
	}
	return out
}

func cloneOpenClawTaskFocus(item *agentesapp.TaskFocus) *agentesapp.TaskFocus {
	if item == nil {
		return nil
	}
	clone := *item
	return &clone
}

func cloneOpenClawOrderFocus(item *agentesapp.OrderFocus) *agentesapp.OrderFocus {
	if item == nil {
		return nil
	}
	clone := *item
	return &clone
}

func buildOpenClawAgentLoad(tareas []tareaLite) (map[string]int, map[string]int) {
	activa := make(map[string]int)
	reservada := make(map[string]int)
	for _, tarea := range tareas {
		nombre := strings.TrimSpace(tarea.Agente)
		if nombre == "" {
			continue
		}
		switch tarea.Estado {
		case db.TareaEnProgreso, db.TareaBloqueada:
			activa[nombre]++
		case db.TareaAsignada:
			reservada[nombre]++
		}
	}
	return activa, reservada
}

func apiHandlerOpenClawThreads(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		supervisor := strings.TrimSpace(r.URL.Query().Get("supervisor"))
		sessionID := strings.TrimSpace(r.URL.Query().Get("session_id"))
		threads, err := buildSupervisorThreadsSnapshot(supervisor, sessionID, 100)
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, threads)
	case http.MethodPost:
		var req struct {
			Supervisor string `json:"supervisor"`
			Proyecto   string `json:"proyecto"`
			SessionID  string `json:"session_id"`
			ThreadID   string `json:"thread_id"`
			Kind       string `json:"kind"`
			Mode       string `json:"mode"`
			Status     string `json:"status"`
			Source     string `json:"source"`
			TurnID     string `json:"turn_id"`
		}
		if err := apiDecodeJSON(r, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		item, err := registrarSupervisorThreadLigero(req.Supervisor, req.Proyecto, req.SessionID, req.ThreadID, req.Kind, req.Mode, req.Status, req.Source, req.TurnID)
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{"ok": true, "thread": item})
	default:
		apiMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func apiHandlerOpenClawPipeline(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		supervisor := strings.TrimSpace(r.URL.Query().Get("supervisor"))
		proyecto := strings.TrimSpace(r.URL.Query().Get("proyecto"))
		pipeline, err := buildSupervisorPipelineSnapshot(supervisor, proyecto, 20)
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, pipeline)
	case http.MethodPost:
		var req struct {
			Supervisor     string `json:"supervisor"`
			Proyecto       string `json:"proyecto"`
			PipelineName   string `json:"pipeline_name"`
			CurrentPhase   string `json:"current_phase"`
			Status         string `json:"status"`
			CurrentTaskID  *int64 `json:"current_task_id"`
			CurrentGateID  *int64 `json:"current_gate_id"`
			CurrentMergeID *int64 `json:"current_merge_id"`
			ArtifactsJSON  string `json:"artifacts_json"`
			MetadataJSON   string `json:"metadata_json"`
		}
		if err := apiDecodeJSON(r, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		item, err := db.UpsertSupervisorPipelineState(db.UpsertSupervisorPipelineStateInput{
			Supervisor:     resolveSupervisorName(req.Supervisor),
			ProyectoSlug:   strings.TrimSpace(req.Proyecto),
			PipelineName:   strings.TrimSpace(req.PipelineName),
			CurrentPhase:   strings.TrimSpace(req.CurrentPhase),
			Status:         strings.TrimSpace(req.Status),
			CurrentTaskID:  req.CurrentTaskID,
			CurrentGateID:  req.CurrentGateID,
			CurrentMergeID: req.CurrentMergeID,
			ArtifactsJSON:  strings.TrimSpace(req.ArtifactsJSON),
			MetadataJSON:   strings.TrimSpace(req.MetadataJSON),
		})
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{"ok": true, "pipeline": item})
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

func apiHandlerGobernanzaCatalogo(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	tipoAgente := strings.TrimSpace(r.URL.Query().Get("tipo_agente"))
	if tipoAgente == "" {
		tipoAgente = strings.TrimSpace(r.URL.Query().Get("rol"))
	}
	proyecto := strings.TrimSpace(r.URL.Query().Get("proyecto"))
	agente := strings.TrimSpace(r.URL.Query().Get("agente"))
	catalogo, err := gobernanzaService.ResolveCatalogForContext(tipoAgente, proyecto, agente)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiGovernanceCatalogResponse{Catalogo: catalogo})
}

func apiHandlerGobernanzaOverrides(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		tipoAgente := strings.TrimSpace(r.URL.Query().Get("tipo_agente"))
		if tipoAgente == "" {
			tipoAgente = strings.TrimSpace(r.URL.Query().Get("rol"))
		}
		agente := strings.TrimSpace(r.URL.Query().Get("agente"))
		overrides, err := gobernanzaService.ListOverrides(
			strings.TrimSpace(r.URL.Query().Get("scope_tipo")),
			strings.TrimSpace(r.URL.Query().Get("scope_ref")),
			tipoAgente,
			agente,
			strings.TrimSpace(r.URL.Query().Get("entidad")),
		)
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, apiGovernanceOverridesResponse{Overrides: overrides})
	case http.MethodPost:
		var req apiGovernanceOverrideSaveRequest
		if err := apiDecodeJSON(r, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		id, err := gobernanzaService.SaveOverride(gobernanzaapp.SaveOverrideInput{
			Actor:      strings.TrimSpace(req.Actor),
			TipoAgente: strings.TrimSpace(req.TipoAgente),
			ScopeTipo:  strings.TrimSpace(req.ScopeTipo),
			ScopeRef:   strings.TrimSpace(req.ScopeRef),
			Entidad:    strings.TrimSpace(req.Entidad),
			EntidadID:  req.EntidadID,
			Accion:     strings.TrimSpace(req.Accion),
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
	agenteFiltro := strings.TrimSpace(r.URL.Query().Get("agente"))
	accionFiltro := strings.TrimSpace(r.URL.Query().Get("accion"))
	entidadFiltro := strings.TrimSpace(r.URL.Query().Get("entidad"))
	entidadIDFiltro := strings.TrimSpace(r.URL.Query().Get("entidad_id"))
	desdeFiltro, err := apiParseRFC3339QueryTime(r.URL.Query().Get("desde"))
	if err != nil {
		apiError(w, http.StatusBadRequest, fmt.Errorf("desde inválido"))
		return
	}

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
	filtro := db.FiltroAuditoria{Limite: limit}
	if agenteFiltro != "" {
		filtro.Agente = &agenteFiltro
	}
	if accionFiltro != "" {
		filtro.Accion = &accionFiltro
	}
	if entidadFiltro != "" {
		filtro.Entidad = &entidadFiltro
	}
	if filtrarEntidadID {
		filtro.EntidadID = &entidadID
	}
	if desdeFiltro != nil {
		filtro.Desde = desdeFiltro
	}
	logs, err := runAPITimeboxed(apiAuditTimeout, func() ([]*db.LogAuditoria, error) {
		return apiListAuditFn(filtro)
	}, errStatusFetchTimeout)
	if err != nil {
		if errors.Is(err, errStatusFetchTimeout) {
			apiWriteJSON(w, http.StatusOK, apiAuditResponse{Audit: []db.AuditEntry{}})
			return
		}
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	entries := make([]db.AuditEntry, 0, len(logs))
	for _, entry := range logs {
		if entry == nil {
			continue
		}
		entries = append(entries, db.AuditEntry{
			Agente:    entry.Agente,
			Accion:    entry.Accion,
			Entidad:   entry.Entidad,
			EntidadID: entry.EntidadID,
			Detalle:   entry.Detalle,
			CreatedAt: entry.CreatedAt,
		})
	}
	apiWriteJSON(w, http.StatusOK, apiAuditResponse{Audit: entries})
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

func apiHandlerPersistenciaVerificar(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	informe, err := db.VerificarPersistenciaActual()
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiPersistenciaVerificacionResponse{Informe: informe})
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
	lite := parseBoolDebug(r.URL.Query().Get("lite"), false)
	proyectos, err := runAPITimeboxed(apiProjectListTimeout, func() ([]*db.Proyecto, error) {
		filtro := db.FiltroProyectos{Activo: activoPtr}
		if lite {
			return apiListProjectsLiteFn(filtro)
		}
		return apiListProjectsFn(filtro, "")
	}, errStatusFetchTimeout)
	if err != nil {
		if errors.Is(err, errStatusFetchTimeout) {
			apiError(w, http.StatusServiceUnavailable, fmt.Errorf("proyectos temporalmente degradado; reintenta"))
			return
		}
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
		proyecto, err := apiGetProyectoConRutaEfectivaTimeboxed(ref, "")
		if err != nil {
			apiError(w, http.StatusNotFound, err)
			return
		}
		if proyecto == nil {
			apiError(w, http.StatusNotFound, fmt.Errorf("proyecto no encontrado: %s", ref))
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{"proyecto": proyecto})
	case len(parts) == 1 && (r.Method == http.MethodPost || r.Method == http.MethodPut):
		apiHandlerProyectoActualizar(w, r, ref)
	case len(parts) == 2 && parts[1] == "overview" && r.Method == http.MethodGet:
		apiHandlerProyectoOverview(w, r, ref)
	case len(parts) == 2 && parts[1] == "control" && r.Method == http.MethodGet:
		apiHandlerProyectoControl(w, r, ref)
	case len(parts) == 2 && parts[1] == "cockpit" && r.Method == http.MethodGet:
		apiHandlerProyectoCockpit(w, r, ref)
	case len(parts) == 2 && parts[1] == "fusionar" && r.Method == http.MethodPost:
		apiHandlerProyectoFusionar(w, r, ref)
	case len(parts) == 2 && parts[1] == "decisiones" && r.Method == http.MethodPost:
		apiHandlerProyectoDecisionNueva(w, r, ref)
	case len(parts) == 2 && parts[1] == "documentacion" && r.Method == http.MethodPost:
		apiHandlerProyectoDocumentoNuevo(w, r, ref)
	case len(parts) == 2 && parts[1] == "operacion" && r.Method == http.MethodGet:
		apiHandlerProyectoOperacion(w, r, ref)
	case len(parts) == 2 && parts[1] == "operacion" && r.Method == http.MethodPost:
		apiHandlerProyectoOperacionGuardar(w, r, ref)
	case len(parts) == 2 && parts[1] == "autonomia" && r.Method == http.MethodGet:
		apiHandlerProyectoAutonomia(w, r, ref)
	case len(parts) == 2 && parts[1] == "autonomia" && r.Method == http.MethodPost:
		apiHandlerProyectoAutonomiaGuardar(w, r, ref)
	case len(parts) == 3 && parts[1] == "autonomia" && parts[2] == "microciclo" && r.Method == http.MethodPost:
		apiHandlerProyectoMicrociclo(w, r, ref)
	case len(parts) == 3 && parts[1] == "autonomia" && parts[2] == "ciclos" && r.Method == http.MethodGet:
		apiHandlerProyectoAutonomiaCiclos(w, r, ref)
	case len(parts) == 2 && parts[1] == "fabricar-app" && r.Method == http.MethodPost:
		apiHandlerProyectoFabricarApp(w, r, ref)
	case len(parts) == 3 && parts[1] == "fabricar-app" && parts[2] == "preview" && r.Method == http.MethodPost:
		apiHandlerProyectoFabricarAppPreview(w, r, ref)
	case len(parts) == 2 && parts[1] == "idiomas" && r.Method == http.MethodPost:
		apiHandlerProyectoIdiomas(w, r, ref)
	case len(parts) == 2 && parts[1] == "contexto-compartido" && r.Method == http.MethodGet:
		apiHandlerProyectoSharedContextList(w, r, ref)
	case len(parts) == 2 && parts[1] == "contexto-compartido" && r.Method == http.MethodPost:
		apiHandlerProyectoSharedContextCreate(w, r, ref)
	default:
		http.NotFound(w, r)
	}
}

func apiHandlerProyectoActualizar(w http.ResponseWriter, r *http.Request, ref string) {
	proyecto, err := db.GetProyecto(strings.TrimSpace(ref))
	if err != nil {
		apiError(w, http.StatusNotFound, err)
		return
	}
	var req apiProyectoActualizarRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	actualizado := *proyecto
	if v := strings.TrimSpace(req.Slug); v != "" {
		actualizado.Slug = v
	}
	if v := strings.TrimSpace(req.Nombre); v != "" {
		actualizado.Nombre = v
	}
	if v := strings.TrimSpace(req.RutaAbs); v != "" {
		actualizado.RutaAbs = v
	}
	if v := strings.TrimSpace(req.OrigenRepo); v != "" {
		actualizado.OrigenRepo = v
	}
	if req.RemoteURL != "" {
		actualizado.RemoteURL = strings.TrimSpace(req.RemoteURL)
	}
	if req.BranchBase != "" {
		actualizado.BranchBase = strings.TrimSpace(req.BranchBase)
	}
	if v := strings.TrimSpace(req.Tipo); v != "" {
		actualizado.Tipo = db.TipoProyecto(v)
	}
	if req.ParentID != nil {
		actualizado.ParentID = req.ParentID
	}
	if req.Activo != nil {
		actualizado.Activo = *req.Activo
	}
	if err := db.UpdateProyecto(&actualizado); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	recargado, err := db.GetProyecto(fmt.Sprintf("%d", actualizado.ID))
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"proyecto": recargado})
}

func apiHandlerProyectoFusionar(w http.ResponseWriter, r *http.Request, ref string) {
	var req apiProyectoFusionRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	resultado, err := db.FusionarProyectos(strings.TrimSpace(req.Origen), strings.TrimSpace(ref), db.FusionProyectosOptions{
		ArchivarOrigen: req.ArchivarOrigen,
	})
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiProyectoFusionResponse{Resultado: resultado})
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

func apiHandlerProyectoControl(w http.ResponseWriter, r *http.Request, ref string) {
	sinceRaw := strings.TrimSpace(r.URL.Query().Get("desde"))
	since, err := parseStatsSince(sinceRaw)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	report, err := buildProjectControlReport(strings.TrimSpace(ref), since)
	if err != nil {
		apiError(w, http.StatusNotFound, err)
		return
	}
	if report == nil {
		apiError(w, http.StatusNotFound, fmt.Errorf("proyecto no encontrado: %s", strings.TrimSpace(ref)))
		return
	}
	apiWriteJSON(w, http.StatusOK, apiProyectoControlResponse{Control: report})
}

func apiHandlerAgenteActividad(w http.ResponseWriter, r *http.Request, agent string) {
	sinceRaw := strings.TrimSpace(r.URL.Query().Get("desde"))
	since, err := parseStatsSince(sinceRaw)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	project := strings.TrimSpace(r.URL.Query().Get("proyecto"))
	report, err := buildAgentActivityReportLocal(strings.TrimSpace(agent), project, since, 200, 400)
	if err != nil {
		apiError(w, http.StatusNotFound, err)
		return
	}
	compact := *report
	compact.Detail = nil
	compact.Audit = nil
	compact.Transcript = nil
	apiWriteJSON(w, http.StatusOK, apiAgenteActividadResponse{Activity: &compact})
}

func apiHandlerProyectoCockpit(w http.ResponseWriter, r *http.Request, ref string) {
	cockpit, err := buildProyectoCockpit(strings.TrimSpace(ref))
	if err != nil {
		apiError(w, http.StatusNotFound, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiProyectoCockpitResponse{Cockpit: cockpit})
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
		Nombre:           nombre,
		Descripcion:      descripcion,
		ObjetivoNegocio:  strings.TrimSpace(req.ObjetivoNegocio),
		UsuariosObjetivo: strings.TrimSpace(req.UsuariosObjetivo),
		Restricciones:    strings.TrimSpace(req.Restricciones),
		Tipo:             strings.TrimSpace(req.Tipo),
		Frontend:         req.Frontend,
		API:              req.API,
		Auth:             req.Auth,
		Database:         req.Database,
		Docker:           req.Docker,
		I18n:             req.I18n,
		Idiomas:          req.Idiomas,

		PlatWeb:      req.PlatWeb,
		PlatDesktop:  req.PlatDesktop,
		PlatMobile:   req.PlatMobile,
		PlatCLI:      req.PlatCLI,
		PlatEmbedded: req.PlatEmbedded,

		SOLinux:   req.SOLinux,
		SOWindows: req.SOWindows,
		SOmacOS:   req.SOmacOS,
		SOAndroid: req.SOAndroid,
		SOiOS:     req.SOiOS,

		ComplianceRGPD:          req.ComplianceRGPD,
		ComplianceENS:           req.ComplianceENS,
		ComplianceLSSI:          req.ComplianceLSSI,
		ComplianceWCAG:          req.ComplianceWCAG,
		ComplianceFacturaElec:   req.ComplianceFacturaElec,
		ComplianceReutilizacion: req.ComplianceReutilizacion,

		CI:         req.CI,
		Kubernetes: req.Kubernetes,
		Terraform:  req.Terraform,
		Monitoring: req.Monitoring,

		Arquitectura:          req.Arquitectura,
		APIStyle:              req.APIStyle,
		FrontendStack:         req.FrontendStack,
		DatabaseEngine:        req.DatabaseEngine,
		AuthMode:              req.AuthMode,
		IdentityProvider:      req.IdentityProvider,
		TestingLevel:          req.TestingLevel,
		ObservabilityLevel:    req.ObservabilityLevel,
		DeploymentTarget:      req.DeploymentTarget,
		ArtifactType:          req.ArtifactType,
		BackgroundJobs:        req.BackgroundJobs,
		Notifications:         req.Notifications,
		MultiTenant:           req.MultiTenant,
		RBAC:                  req.RBAC,
		ThemeSupport:          req.ThemeSupport,
		BrandingProfiles:      req.BrandingProfiles,
		OfflineMode:           req.OfflineMode,
		ImportExport:          req.ImportExport,
		Webhooks:              req.Webhooks,
		FileUploads:           req.FileUploads,
		Reporting:             req.Reporting,
		ServicioResidente:     req.ServicioResidente,
		Cache:                 req.Cache,
		Queue:                 req.Queue,
		Scheduler:             req.Scheduler,
		ObjectStorage:         req.ObjectStorage,
		Search:                req.Search,
		RateLimiting:          req.RateLimiting,
		FeatureFlags:          req.FeatureFlags,
		AuditTrail:            req.AuditTrail,
		Backups:               req.Backups,
		DisasterRecovery:      req.DisasterRecovery,
		IntegracionesExternas: req.Integraciones,
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

func apiHandlerProyectoFabricarAppPreview(w http.ResponseWriter, r *http.Request, ref string) {
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
		descripcion = "Preview de backlog de " + nombre + "."
	}
	plan, err := newProjectAppFactory().Generate(fabricaapp.AppSpec{
		Nombre:           nombre,
		Descripcion:      descripcion,
		ObjetivoNegocio:  strings.TrimSpace(req.ObjetivoNegocio),
		UsuariosObjetivo: strings.TrimSpace(req.UsuariosObjetivo),
		Restricciones:    strings.TrimSpace(req.Restricciones),
		Tipo:             strings.TrimSpace(req.Tipo),
		Frontend:         req.Frontend,
		API:              req.API,
		Auth:             req.Auth,
		Database:         req.Database,
		Docker:           req.Docker,
		I18n:             req.I18n,
		Idiomas:          req.Idiomas,

		PlatWeb:      req.PlatWeb,
		PlatDesktop:  req.PlatDesktop,
		PlatMobile:   req.PlatMobile,
		PlatCLI:      req.PlatCLI,
		PlatEmbedded: req.PlatEmbedded,

		SOLinux:   req.SOLinux,
		SOWindows: req.SOWindows,
		SOmacOS:   req.SOmacOS,
		SOAndroid: req.SOAndroid,
		SOiOS:     req.SOiOS,

		ComplianceRGPD:          req.ComplianceRGPD,
		ComplianceENS:           req.ComplianceENS,
		ComplianceLSSI:          req.ComplianceLSSI,
		ComplianceWCAG:          req.ComplianceWCAG,
		ComplianceFacturaElec:   req.ComplianceFacturaElec,
		ComplianceReutilizacion: req.ComplianceReutilizacion,

		CI:         req.CI,
		Kubernetes: req.Kubernetes,
		Terraform:  req.Terraform,
		Monitoring: req.Monitoring,

		Arquitectura:          req.Arquitectura,
		APIStyle:              req.APIStyle,
		FrontendStack:         req.FrontendStack,
		DatabaseEngine:        req.DatabaseEngine,
		AuthMode:              req.AuthMode,
		IdentityProvider:      req.IdentityProvider,
		TestingLevel:          req.TestingLevel,
		ObservabilityLevel:    req.ObservabilityLevel,
		DeploymentTarget:      req.DeploymentTarget,
		ArtifactType:          req.ArtifactType,
		BackgroundJobs:        req.BackgroundJobs,
		Notifications:         req.Notifications,
		MultiTenant:           req.MultiTenant,
		RBAC:                  req.RBAC,
		ThemeSupport:          req.ThemeSupport,
		BrandingProfiles:      req.BrandingProfiles,
		OfflineMode:           req.OfflineMode,
		ImportExport:          req.ImportExport,
		Webhooks:              req.Webhooks,
		FileUploads:           req.FileUploads,
		Reporting:             req.Reporting,
		ServicioResidente:     req.ServicioResidente,
		Cache:                 req.Cache,
		Queue:                 req.Queue,
		Scheduler:             req.Scheduler,
		ObjectStorage:         req.ObjectStorage,
		Search:                req.Search,
		RateLimiting:          req.RateLimiting,
		FeatureFlags:          req.FeatureFlags,
		AuditTrail:            req.AuditTrail,
		Backups:               req.Backups,
		DisasterRecovery:      req.DisasterRecovery,
		IntegracionesExternas: req.Integraciones,
	})
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiProyectoFabricarAppPreviewResponse{
		OK:    true,
		Tasks: plan.Tasks,
	})
}

func apiHandlerProyectoIdiomas(w http.ResponseWriter, r *http.Request, ref string) {
	proyecto, err := db.GetProyecto(strings.TrimSpace(ref))
	if err != nil {
		apiError(w, http.StatusNotFound, err)
		return
	}
	var req apiProyectoIdiomasRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	req.Idiomas = splitCSV(strings.Join(req.Idiomas, ","))
	if len(req.Idiomas) == 0 {
		apiError(w, http.StatusBadRequest, fmt.Errorf("idiomas obligatorios"))
		return
	}
	actor := strings.TrimSpace(req.Por)
	if actor == "" {
		actor = "alberto"
	}
	plan, err := newProjectAppFactory().GenerateLanguageExpansion(strings.TrimSpace(proyecto.Nombre), req.Idiomas)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	result, err := fabricaapp.Materialize(fabricaapp.DBStore{}, proyecto.ID, actor, plan)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	if ruta := strings.TrimSpace(proyecto.RutaAbs); ruta != "" {
		if info, statErr := os.Stat(ruta); statErr == nil && info.IsDir() {
			if _, err := i18n.ExpandProjectSkeletonLanguages(ruta, req.Idiomas); err != nil {
				apiError(w, http.StatusInternalServerError, err)
				return
			}
		}
	}
	apiWriteJSON(w, http.StatusCreated, apiProyectoIdiomasResponse{
		OK:      true,
		Slug:    proyecto.Slug,
		Idiomas: req.Idiomas,
		Created: result.Created,
		Backlog: result.Backlog,
	})
}

func apiHandlerProyectoSharedContextList(w http.ResponseWriter, r *http.Request, ref string) {
	proyecto, err := db.GetProyecto(strings.TrimSpace(ref))
	if err != nil {
		apiError(w, http.StatusNotFound, err)
		return
	}
	if proyecto == nil {
		apiError(w, http.StatusNotFound, fmt.Errorf("proyecto no encontrado"))
		return
	}
	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	filtro := db.FiltroSharedContextItems{
		ProyectoID: &proyecto.ID,
		Activos:    true,
		Limit:      limit,
	}
	if agente := strings.TrimSpace(r.URL.Query().Get("agente")); agente != "" {
		filtro.Agente = &agente
	}
	if tipo := strings.TrimSpace(r.URL.Query().Get("tipo")); tipo != "" {
		filtro.Tipo = &tipo
	}
	items, err := db.ListSharedContextItems(filtro)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	_, summary := db.BuildSharedContextSummary(strings.TrimSpace(r.URL.Query().Get("agente")), proyecto)
	apiWriteJSON(w, http.StatusOK, apiProyectoSharedContextResponse{
		Items:   items,
		Summary: summary,
	})
}

func apiHandlerProyectoSharedContextCreate(w http.ResponseWriter, r *http.Request, ref string) {
	proyecto, err := db.GetProyecto(strings.TrimSpace(ref))
	if err != nil {
		apiError(w, http.StatusNotFound, err)
		return
	}
	if proyecto == nil {
		apiError(w, http.StatusNotFound, fmt.Errorf("proyecto no encontrado"))
		return
	}
	var req apiProyectoSharedContextCreateRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	titulo := strings.TrimSpace(req.Titulo)
	if titulo == "" {
		apiError(w, http.StatusBadRequest, fmt.Errorf("titulo obligatorio"))
		return
	}
	item := &db.SharedContextItem{
		ProyectoID:  &proyecto.ID,
		Agente:      strings.TrimSpace(req.Agente),
		Tipo:        strings.TrimSpace(req.Tipo),
		Titulo:      titulo,
		Detalle:     strings.TrimSpace(req.Detalle),
		PayloadJSON: strings.TrimSpace(req.PayloadJSON),
		Peso:        req.Peso,
		Origen:      strings.TrimSpace(req.Origen),
	}
	if item.Tipo == "" {
		item.Tipo = "decision"
	}
	if item.Peso <= 0 {
		item.Peso = 1
	}
	if item.Origen == "" {
		item.Origen = "humano"
	}
	if raw := strings.TrimSpace(req.ExpiresAt); raw != "" {
		ts, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			apiError(w, http.StatusBadRequest, fmt.Errorf("expires_at inválido: %w", err))
			return
		}
		item.ExpiresAt = &ts
	}
	id, err := db.CreateSharedContextItem(item)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusCreated, apiProyectoSharedContextMutationResponse{
		OK: true,
		ID: id,
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

func apiHandlerProyectoAutonomia(w http.ResponseWriter, r *http.Request, ref string) {
	policy, err := supervisionService.GetProjectPolicy(strings.TrimSpace(ref))
	if err != nil {
		apiError(w, http.StatusNotFound, err)
		return
	}
	limit := 10
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	cycles, err := supervisionService.ListCycles(strings.TrimSpace(ref), nil, limit)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiProyectoAutonomiaResponse{Policy: policy, Cycles: cycles})
}

func apiHandlerProyectoAutonomiaGuardar(w http.ResponseWriter, r *http.Request, ref string) {
	var req apiProyectoAutonomiaSaveRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	policy, err := supervisionService.UpsertProjectPolicy(strings.TrimSpace(ref), supervisionapp.PolicyInput{
		Enabled:              req.Enabled,
		ObjetivoGeneral:      strings.TrimSpace(req.ObjetivoGeneral),
		DefinitionOfDoneJSON: strings.TrimSpace(req.DefinitionOfDoneJSON),
		MaxWorkers:           req.MaxWorkers,
		SupervisorAgente:     strings.TrimSpace(req.SupervisorAgente),
		ReviewerAgente:       strings.TrimSpace(req.ReviewerAgente),
		ReserveReviewer:      req.ReserveReviewer,
		ReserveSupervisor:    req.ReserveSupervisor,
		ReviewRequired:       req.ReviewRequired,
		AutoCreateTasks:      req.AutoCreateTasks,
		AutoCloseProject:     req.AutoCloseProject,
		EstadoAutonomia:      supervisionapp.EstadoAutonomiaProyecto(strings.TrimSpace(req.EstadoAutonomia)),
	})
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	if err := bootstrapPersistentAutonomyProject(strings.TrimSpace(ref), policy, "api_autonomia_guardar"); err != nil {
		apiError(w, repoAPIStatusForError(err), err)
		return
	}
	policy, err = supervisionService.GetProjectPolicy(strings.TrimSpace(ref))
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiProyectoAutonomiaResponse{Policy: policy})
}

func apiHandlerProyectoAutonomiaCiclos(w http.ResponseWriter, r *http.Request, ref string) {
	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	var kindPtr *string
	if kind := strings.TrimSpace(r.URL.Query().Get("kind")); kind != "" {
		kindPtr = &kind
	}
	cycles, err := supervisionService.ListCycles(strings.TrimSpace(ref), kindPtr, limit)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"cycles": cycles})
}

func apiHandlerProyectoMicrociclo(w http.ResponseWriter, r *http.Request, ref string) {
	var req apiProyectoMicrocicloRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	resultado, err := activarMicrocicloProyecto(strings.TrimSpace(ref), proyectoMicrocicloRequest{
		Agente:               strings.TrimSpace(req.Agente),
		ObjetivoGeneral:      strings.TrimSpace(req.ObjetivoGeneral),
		DefinitionOfDoneJSON: strings.TrimSpace(req.DefinitionOfDoneJSON),
		Titulo:               strings.TrimSpace(req.Titulo),
		Descripcion:          strings.TrimSpace(req.Descripcion),
		Modulo:               strings.TrimSpace(req.Modulo),
		Notas:                strings.TrimSpace(req.Notas),
		LimpiarPruebas:       req.LimpiarPruebas,
	})
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiProyectoMicrocicloResponse{Resultado: resultado})
}

func apiHandlerReviewGates(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		limit := 50
		if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
			if n, err := strconv.Atoi(raw); err == nil && n > 0 {
				limit = n
			}
		}
		gates, err := reviewService.List(reviewapp.ListInput{
			ProyectoRef:    strings.TrimSpace(r.URL.Query().Get("proyecto")),
			Estado:         strings.TrimSpace(r.URL.Query().Get("estado")),
			ReviewerAgente: strings.TrimSpace(r.URL.Query().Get("reviewer_agente")),
			Limit:          limit,
		})
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{"gates": gates})
	case http.MethodPost:
		var req apiReviewGateCreateRequest
		if err := apiDecodeJSON(r, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		gate, err := reviewService.Create(reviewapp.CreateGateInput{
			ProyectoRef:     strings.TrimSpace(req.Proyecto),
			TareaID:         req.TareaID,
			WorktreeID:      req.WorktreeID,
			RequestedBy:     strings.TrimSpace(req.RequestedBy),
			ReviewerAgente:  strings.TrimSpace(req.ReviewerAgente),
			SeverityMax:     strings.TrimSpace(req.SeverityMax),
			InitialFindings: strings.TrimSpace(req.FindingsJSON),
		})
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		apiWriteJSON(w, http.StatusCreated, map[string]any{"gate": gate})
	default:
		http.NotFound(w, r)
	}
}

func apiRouterReviewGates(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/review-gates/"), "/")
	if path == "" {
		http.NotFound(w, r)
		return
	}
	parts := strings.Split(path, "/")
	id, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil || id <= 0 {
		apiError(w, http.StatusBadRequest, fmt.Errorf("id de gate inválido"))
		return
	}
	switch {
	case len(parts) == 1 && r.Method == http.MethodPost:
		var req apiReviewGateUpdateRequest
		if err := apiDecodeJSON(r, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		gate, err := reviewService.Update(reviewapp.UpdateGateInput{
			ID:             id,
			Estado:         strings.TrimSpace(req.Estado),
			ReviewerAgente: req.ReviewerAgente,
			SeverityMax:    req.SeverityMax,
			FindingsJSON:   req.FindingsJSON,
		})
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{"gate": gate})
	case len(parts) == 2 && parts[1] == "resolver" && r.Method == http.MethodPost:
		var req apiReviewGateResolveRequest
		if err := apiDecodeJSON(r, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		gate, err := reviewService.Resolve(reviewapp.ResolveGateInput{
			ID:             id,
			Estado:         strings.TrimSpace(req.Estado),
			ReviewerAgente: strings.TrimSpace(req.ReviewerAgente),
			FindingsJSON:   strings.TrimSpace(req.FindingsJSON),
		})
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{"gate": gate})
	default:
		http.NotFound(w, r)
	}
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
	if path == "local-compartido" {
		if !apiRequireMethod(w, r, http.MethodPost) {
			return
		}
		apiHandlerPoolLocalCompartidoSave(w, r)
		return
	}
	if strings.HasSuffix(path, "/local") {
		slug := strings.Trim(strings.TrimSuffix(path, "/local"), "/")
		if slug == "" {
			http.NotFound(w, r)
			return
		}
		if !apiRequireMethod(w, r, http.MethodGet) {
			return
		}
		apiHandlerPoolLocalCompartidoGet(w, r, slug)
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

func apiHandlerPoolLocalCompartidoGet(w http.ResponseWriter, r *http.Request, slug string) {
	detalle, err := capacidadService.DescribirPoolLocalCompartido(strings.TrimSpace(slug))
	if err != nil {
		apiError(w, http.StatusNotFound, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiPoolLocalCompartidoResponse{PoolLocal: detalle})
}

func apiHandlerPoolLocalCompartidoSave(w http.ResponseWriter, r *http.Request) {
	var req apiPoolLocalCompartidoSaveRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	detalle, err := capacidadService.AsegurarPoolLocalCompartido(capacidadapp.EntradaAsegurarPoolLocalCompartido{
		PoolSlug:               strings.TrimSpace(req.PoolSlug),
		Proveedor:              strings.TrimSpace(req.Proveedor),
		Runtime:                strings.TrimSpace(req.Runtime),
		ModeloPreferente:       strings.TrimSpace(req.ModeloPreferente),
		SlotsMaximos:           req.SlotsMaximos,
		ConectorCanonico:       strings.TrimSpace(req.ConectorCanonico),
		ConectorCompatibilidad: strings.TrimSpace(req.ConectorCompatibilidad),
		ExperimentalCompat:     req.ExperimentalCompat,
		Perfiles:               req.Perfiles,
	})
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusCreated, apiPoolLocalCompartidoResponse{PoolLocal: detalle})
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

func apiHandlerModeloPipelineLocal(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	pipeline, err := capacidadService.ConstruirPipelineLocalDeterminista(strings.TrimSpace(r.URL.Query().Get("proyecto")))
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiPipelineLocalDeterministaResponse{Pipeline: pipeline})
}

func apiHandlerModeloPipelineLocalPaso(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	paso, err := capacidadService.CalcularSiguientePasoPipelineLocalDeterminista(strings.TrimSpace(r.URL.Query().Get("proyecto")))
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiPasoPipelineLocalDeterministaResponse{Paso: paso})
}

func apiHandlerModeloPipelineLocalEjecutar(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	resultado, err := capacidadService.EjecutarSiguientePasoPipelineLocalDeterminista(strings.TrimSpace(r.URL.Query().Get("proyecto")))
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiEjecutarPasoPipelineLocalDeterministaResponse{Resultado: resultado})
}

func apiHandlerModeloPipelineLocalDespachar(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	resultado, err := capacidadService.EjecutarYDespacharSiguientePasoPipelineLocalDeterminista(strings.TrimSpace(r.URL.Query().Get("proyecto")))
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiEjecutarPasoPipelineLocalDeterministaResponse{Resultado: resultado})
}

func apiHandlerModeloRuntimeActivos(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	items, err := capacidadService.ListarModelosRuntimeActivos()
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiModelosRuntimeActivosResponse{Modelos: items})
}

func apiHandlerModeloRuntimeDescargar(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	descargados, err := capacidadService.DescargarModelosRuntimeActivos()
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	if descargados == nil {
		descargados = []string{}
	}
	apiWriteJSON(w, http.StatusOK, apiModelosRuntimeDescargarResponse{OK: true, Descargados: descargados})
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

func apiHandlerTareasLimpiarFrente(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req apiTareaLimpiarFrenteRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	resultado, err := tareasService.CleanActiveFront(tareasapp.CleanActiveFrontInput{
		Agente:   req.Agente,
		Proyecto: req.Proyecto,
		KeepIDs:  req.KeepIDs,
	})
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiTareaLimpiarFrenteResponse{OK: true, Resultado: resultado})
}

func apiHandlerTareasListar(w http.ResponseWriter, r *http.Request) {
	filtro := db.FiltroTareas{}
	resumen := parseBoolDebug(r.URL.Query().Get("resumen"), false)
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil || limit < 0 {
			apiError(w, http.StatusBadRequest, fmt.Errorf("limit invalido"))
			return
		}
		if limit > 500 {
			limit = 500
		}
		filtro.Limit = limit
	}
	if agente := strings.TrimSpace(r.URL.Query().Get("agente")); agente != "" {
		filtro.Agente = &agente
	}
	if modulo := strings.TrimSpace(r.URL.Query().Get("modulo")); modulo != "" {
		filtro.Modulo = &modulo
	}
	estados := parseEstadoTareaQuery(r.URL.Query()["estado"])
	if len(estados) == 1 {
		estado := estados[0]
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
	var tareas []*db.Tarea
	var err error
	tareas, err = runAPITimeboxed(apiTaskListTimeout, func() ([]*db.Tarea, error) {
		if len(estados) > 1 {
			return listarTareasPorEstadosOR(filtro, estados)
		}
		return apiListTasksFn(filtro)
	}, errStatusFetchTimeout)
	if err != nil {
		if errors.Is(err, errStatusFetchTimeout) {
			apiError(w, http.StatusServiceUnavailable, fmt.Errorf("tareas temporalmente degradado; reintenta"))
			return
		}
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	if resumen {
		tareas = compactTaskListItems(tareas)
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"tareas": tareas})
}

func parseEstadoTareaQuery(raw []string) []db.EstadoTarea {
	out := make([]db.EstadoTarea, 0, len(raw))
	seen := map[string]struct{}{}
	for _, item := range raw {
		for _, part := range strings.Split(item, ",") {
			estado := strings.TrimSpace(part)
			if estado == "" {
				continue
			}
			if _, ok := seen[estado]; ok {
				continue
			}
			seen[estado] = struct{}{}
			out = append(out, db.EstadoTarea(estado))
		}
	}
	return out
}

func listarTareasPorEstadosOR(base db.FiltroTareas, estados []db.EstadoTarea) ([]*db.Tarea, error) {
	itemsByID := map[int64]*db.Tarea{}
	for _, estado := range estados {
		filtro := base
		estadoCopy := estado
		filtro.Estado = &estadoCopy
		items, err := apiListTasksFn(filtro)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if item == nil {
				continue
			}
			itemsByID[item.ID] = item
		}
	}
	out := make([]*db.Tarea, 0, len(itemsByID))
	for _, item := range itemsByID {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func compactTaskListItems(tasks []*db.Tarea) []*db.Tarea {
	if len(tasks) == 0 {
		return tasks
	}
	out := make([]*db.Tarea, 0, len(tasks))
	for _, tarea := range tasks {
		if tarea == nil {
			continue
		}
		copia := *tarea
		copia.Descripcion = ""
		copia.Notas = ""
		copia.Dependencias = nil
		copia.CommitCierre = ""
		out = append(out, &copia)
	}
	return out
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
	if len(parts) == 1 && parts[0] == "limpiar-frente" {
		apiHandlerTareasLimpiarFrente(w, r)
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
		apiWriteJSON(w, http.StatusOK, apiTareaResponse{
			Tarea:     tarea,
			Fork:      parseRepoFunctionForkSpecFromNotes(strings.TrimSpace(tarea.Notas)),
			FinishApp: strings.Contains(strings.ToLower(strings.TrimSpace(tarea.Notas)), "autonomia:finish_app"),
		})
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
	if tareaAccionDebeDespertarWarm(req.Accion) {
		wakeControlPlaneWarm()
	}
	tarea, err := tareasService.Get(id)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"ok": true, "tarea": tarea})
}

func tareaAccionDebeDespertarWarm(accion string) bool {
	switch strings.ToLower(strings.TrimSpace(accion)) {
	case "tomar", "iniciar", "completar", "bloquear", "desbloquear", "backlog", "cancelar", "reasignar":
		return true
	default:
		return false
	}
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
		merge, err := svc.GetRequest(id)
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
	if len(parts) == 2 && parts[1] == "a2ui" {
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
		limit := 8
		if limitStr := strings.TrimSpace(r.URL.Query().Get("limit")); limitStr != "" {
			parsed, err := strconv.Atoi(limitStr)
			if err != nil || parsed <= 0 {
				apiError(w, http.StatusBadRequest, fmt.Errorf("limit inválido"))
				return
			}
			limit = parsed
		}
		apiWriteJSON(w, http.StatusOK, apiRuntimeA2UIResponse{
			Runtime: runtime,
			A2UI:    apiRuntimeA2UIToMessages(runtimeA2UIToWeb(runtime, limit)),
		})
		return
	}
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
	runtimeDetail, err := runtimesService.DescribeRuntime(id)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	samples, err := runtimesService.ListRuntimeSamples(id, 20)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{
		"runtime": runtime,
		"handle":  runtimeDetail.Handle,
		"worker":  runtimeDetail.Worker,
		"samples": samples,
	})
}

func apiRuntimeA2UIToMessages(rows []webRuntimeA2UIRow) []apiRuntimeA2UIMessage {
	out := make([]apiRuntimeA2UIMessage, 0, len(rows))
	for _, row := range rows {
		if row.Creado == "" && row.FromAgente == "" && row.Component == "" && row.Error == "" {
			continue
		}
		msg := apiRuntimeA2UIMessage{
			Creado:     row.Creado,
			FromAgente: row.FromAgente,
			Estado:     row.Estado,
			Component:  row.Component,
			Error:      row.Error,
		}
		if row.DataTable != nil {
			msg.DataTable = &apiRuntimeA2UITable{
				Title:   row.DataTable.Title,
				Columns: append([]string(nil), row.DataTable.Columns...),
				Rows:    cloneStringMatrix(row.DataTable.Rows),
			}
		}
		if row.Chart != nil {
			chartRows := make([]apiRuntimeA2UIChartRow, 0, len(row.Chart.Rows))
			for _, src := range row.Chart.Rows {
				chartRows = append(chartRows, apiRuntimeA2UIChartRow{
					Label:  src.Label,
					Values: append([]string(nil), src.Values...),
				})
			}
			msg.Chart = &apiRuntimeA2UIChart{
				Title:     row.Chart.Title,
				ChartType: row.Chart.ChartType,
				Series:    append([]string(nil), row.Chart.Series...),
				Rows:      chartRows,
			}
		}
		if row.Approval != nil {
			copy := *row.Approval
			msg.Approval = &copy
		}
		if row.Markdown != nil {
			copy := *row.Markdown
			msg.Markdown = &copy
		}
		out = append(out, msg)
	}
	return out
}

func cloneStringMatrix(src [][]string) [][]string {
	if len(src) == 0 {
		return nil
	}
	out := make([][]string, len(src))
	for i := range src {
		out[i] = append([]string(nil), src[i]...)
	}
	return out
}

func apiNombreAgenteCanonico(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if cached, ok := apiCanonicalAgentCacheGet(raw); ok {
		return cached
	}
	agente, err := apiGetAgentFn(raw)
	if err == nil && agente != nil && strings.TrimSpace(agente.Nombre) != "" {
		canonico := strings.TrimSpace(agente.Nombre)
		apiCanonicalAgentCachePut(raw, canonico)
		apiCanonicalAgentCachePut(canonico, canonico)
		return canonico
	}
	apiCanonicalAgentCachePut(raw, raw)
	return raw
}

func apiHandlerRuntimeHandles(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	agente := apiNombreAgenteCanonico(r.URL.Query().Get("agente"))
	var filtro *string
	if agente != "" {
		filtro = &agente
	}
	handles, err := runtimesService.ListRuntimeHandlesCompact(filtro)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"handles": handles})
}

type apiRuntimeHandlesPurgeRequest struct {
	Agente   string   `json:"agente"`
	Proyecto string   `json:"proyecto"`
	Estados  []string `json:"estados"`
	Actor    string   `json:"actor"`
}

type apiRuntimeHandleResidualCloseRequest struct {
	Agente   string `json:"agente"`
	Proyecto string `json:"proyecto"`
	Actor    string `json:"actor"`
}

func apiHandlerRuntimeHandlesPurgar(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req apiRuntimeHandlesPurgeRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	resultado, err := runtimesService.PurgeInactiveRuntimeHandles(runtimesapp.RuntimeHandlePurgeRequest{
		Agente:   apiNombreAgenteCanonico(req.Agente),
		Proyecto: strings.TrimSpace(req.Proyecto),
		Estados:  req.Estados,
		Actor:    strings.TrimSpace(req.Actor),
	})
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiRuntimeHandlesPurgeResponse{
		OK:         true,
		Deleted:    resultado.Deleted,
		DeletedIDs: resultado.DeletedIDs,
		Estados:    resultado.Estados,
	})
}

func apiHandlerRuntimeHandlesCerrar(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req apiRuntimeHandleResidualCloseRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	resultado, err := runtimesService.CloseResidualRuntimeHandles(runtimesapp.RuntimeHandleResidualCloseRequest{
		Agente:   apiNombreAgenteCanonico(req.Agente),
		Proyecto: strings.TrimSpace(req.Proyecto),
		Actor:    strings.TrimSpace(req.Actor),
	})
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiRuntimeHandleResidualCloseResponse{
		OK:     true,
		Closed: resultado.Closed,
	})
}

func apiHandlerRuntimesCerrar(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req apiRuntimeResidualCloseRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	resultado, err := runtimesService.CloseResidualRuntimes(runtimesapp.RuntimeResidualCloseRequest{
		Agente:   apiNombreAgenteCanonico(req.Agente),
		Proyecto: strings.TrimSpace(req.Proyecto),
		Actor:    strings.TrimSpace(req.Actor),
	})
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiRuntimeResidualCloseResponse{
		OK:        true,
		Closed:    resultado.Closed,
		ClosedIDs: resultado.ClosedIDs,
	})
}

func apiHandlerRuntimeEvents(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	filter := db.FiltroRuntimeEvents{}
	if agente := strings.TrimSpace(r.URL.Query().Get("agente")); agente != "" {
		agente = apiNombreAgenteCanonico(agente)
		filter.Agente = &agente
	}
	if proyecto := strings.TrimSpace(r.URL.Query().Get("proyecto")); proyecto != "" {
		p, err := runtimesService.GetProject(proyecto)
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		filter.ProyectoID = &p.ID
	}
	if runtimeID := strings.TrimSpace(r.URL.Query().Get("runtime_id")); runtimeID != "" {
		id, err := strconv.ParseInt(runtimeID, 10, 64)
		if err != nil || id <= 0 {
			apiError(w, http.StatusBadRequest, fmt.Errorf("runtime_id inválido"))
			return
		}
		filter.RuntimeID = &id
	}
	if kind := strings.TrimSpace(r.URL.Query().Get("kind")); kind != "" {
		filter.Kind = &kind
	}
	if limitStr := strings.TrimSpace(r.URL.Query().Get("limit")); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			apiError(w, http.StatusBadRequest, fmt.Errorf("limit inválido"))
			return
		}
		filter.Limit = limit
	}
	items, err := runtimesService.ListRuntimeEvents(filter)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"events": compactarRuntimeEventsAPI(items)})
}

func compactarRuntimeEventsAPI(items []*db.RuntimeEvent) []*db.RuntimeEvent {
	if len(items) == 0 {
		return items
	}
	out := make([]*db.RuntimeEvent, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		cloned := *item
		cloned.Message = truncarTextoAPI(cloned.Message, 512)
		if len(cloned.Payload) > 0 {
			payload := make(map[string]any, len(cloned.Payload))
			for k, v := range cloned.Payload {
				payload[k] = compactarValorRuntimeEventAPI(v)
			}
			cloned.Payload = payload
			if raw, err := json.Marshal(payload); err == nil {
				cloned.PayloadJSON = truncarTextoAPI(string(raw), 1536)
			} else {
				cloned.PayloadJSON = truncarTextoAPI(cloned.PayloadJSON, 1536)
			}
		} else {
			cloned.PayloadJSON = truncarTextoAPI(cloned.PayloadJSON, 1536)
		}
		out = append(out, &cloned)
	}
	return out
}

func compactarValorRuntimeEventAPI(v any) any {
	switch raw := v.(type) {
	case string:
		return truncarTextoAPI(raw, 512)
	case []any:
		out := make([]any, 0, len(raw))
		for _, item := range raw {
			out = append(out, compactarValorRuntimeEventAPI(item))
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(raw))
		for k, item := range raw {
			out[k] = compactarValorRuntimeEventAPI(item)
		}
		return out
	default:
		return v
	}
}

func truncarTextoAPI(raw string, limit int) string {
	raw = db.SanitizarTextoObservabilidadRuntime(raw)
	if limit <= 0 || len(raw) <= limit {
		return raw
	}
	if limit <= 1 {
		return raw[:limit]
	}
	return raw[:limit-1] + "…"
}

func apiHandlerRuntimeTranscript(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	filter := db.FiltroRuntimeTranscript{}
	desdeFiltro, err := apiParseRFC3339QueryTime(r.URL.Query().Get("desde"))
	if err != nil {
		apiError(w, http.StatusBadRequest, fmt.Errorf("desde inválido"))
		return
	}
	if desdeFiltro != nil {
		filter.Desde = desdeFiltro
	}
	if agente := strings.TrimSpace(r.URL.Query().Get("agente")); agente != "" {
		agente = apiNombreAgenteCanonico(agente)
		filter.Agente = &agente
	}
	if stream := strings.TrimSpace(r.URL.Query().Get("stream")); stream != "" {
		filter.Stream = &stream
	}
	if class := strings.TrimSpace(r.URL.Query().Get("classification")); class != "" {
		filter.Classification = &class
	}
	if proyecto := strings.TrimSpace(r.URL.Query().Get("proyecto")); proyecto != "" {
		p, err := runtimesService.GetProject(proyecto)
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		filter.ProyectoID = &p.ID
	}
	if runtimeID := strings.TrimSpace(r.URL.Query().Get("runtime_id")); runtimeID != "" {
		id, err := strconv.ParseInt(runtimeID, 10, 64)
		if err != nil || id <= 0 {
			apiError(w, http.StatusBadRequest, fmt.Errorf("runtime_id inválido"))
			return
		}
		filter.RuntimeID = &id
	}
	if handleID := strings.TrimSpace(r.URL.Query().Get("handle_id")); handleID != "" {
		id, err := strconv.ParseInt(handleID, 10, 64)
		if err != nil || id <= 0 {
			apiError(w, http.StatusBadRequest, fmt.Errorf("handle_id inválido"))
			return
		}
		filter.HandleID = &id
	}
	if q := strings.TrimSpace(r.URL.Query().Get("q")); q != "" {
		filter.Query = &q
	}
	if pending := strings.TrimSpace(r.URL.Query().Get("signals_pending")); pending != "" {
		switch strings.ToLower(pending) {
		case "1", "true", "yes", "si", "sí", "on":
			filter.SoloSenalesPend = true
		}
	}
	if limitStr := strings.TrimSpace(r.URL.Query().Get("limit")); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			apiError(w, http.StatusBadRequest, fmt.Errorf("limit inválido"))
			return
		}
		filter.Limit = limit
	}
	items, err := apiListRuntimeTranscriptFn(filter)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"transcript": items})
}

func apiParseRFC3339QueryTime(raw string) (*time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, nil
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if parsed, err := time.Parse(layout, value); err == nil {
			utc := parsed.UTC()
			return &utc, nil
		}
	}
	return nil, fmt.Errorf("invalid timestamp")
}

func apiHandlerRuntimeOrders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		filter := db.FiltroRuntimeOrders{}
		if agente := strings.TrimSpace(r.URL.Query().Get("agente")); agente != "" {
			agente = apiNombreAgenteCanonico(agente)
			filter.Agente = &agente
		}
		if estado := strings.TrimSpace(r.URL.Query().Get("estado")); estado != "" {
			filter.Estado = &estado
		}
		if proyecto := strings.TrimSpace(r.URL.Query().Get("proyecto")); proyecto != "" {
			proyectoID, err := apiResolveProjectIDTimeboxed(proyecto)
			if err != nil {
				if errors.Is(err, errStatusFetchTimeout) {
					apiError(w, http.StatusServiceUnavailable, fmt.Errorf("runtime orders temporalmente degradado"))
					return
				}
				apiError(w, http.StatusBadRequest, err)
				return
			}
			filter.ProyectoID = proyectoID
		}
		if limitStr := strings.TrimSpace(r.URL.Query().Get("limit")); limitStr != "" {
			limit, err := strconv.Atoi(limitStr)
			if err != nil || limit <= 0 {
				apiError(w, http.StatusBadRequest, fmt.Errorf("limit inválido"))
				return
			}
			filter.Limit = limit
		} else {
			filter.Limit = 50
		}
		orders, err := runAPITimeboxed(apiListRuntimeOrdersTimeout, func() ([]*db.RuntimeOrder, error) {
			return apiListRuntimeOrdersFn(filter)
		}, errStatusFetchTimeout)
		if err != nil {
			if errors.Is(err, errStatusFetchTimeout) {
				apiError(w, http.StatusServiceUnavailable, fmt.Errorf("runtime orders temporalmente degradado"))
				return
			}
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
		proyectoID, err := apiResolveProjectIDTimeboxed(req.Proyecto)
		if err != nil {
			if errors.Is(err, errStatusFetchTimeout) {
				apiError(w, http.StatusServiceUnavailable, fmt.Errorf("runtime orders temporalmente degradado"))
				return
			}
			apiError(w, http.StatusBadRequest, err)
			return
		}
		id, err := runtimesService.EnqueueRuntimeOrder(&db.RuntimeOrder{
			Agente:      apiNombreAgenteCanonico(req.Agente),
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

func apiRouterRuntimeOrders(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/runtime-orders/"), "/")
	if path == "" || strings.Contains(path, "/") {
		http.NotFound(w, r)
		return
	}
	id, err := strconv.ParseInt(path, 10, 64)
	if err != nil || id <= 0 {
		apiError(w, http.StatusBadRequest, fmt.Errorf("id inválido"))
		return
	}
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	order, err := runtimesService.GetRuntimeOrder(id)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	if order == nil {
		apiError(w, http.StatusNotFound, fmt.Errorf("runtime order no encontrada"))
		return
	}
	apiWriteJSON(w, http.StatusOK, apiRuntimeOrderResponse{Order: order})
}

func apiHandlerRuntimeOrdersPurgar(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req apiRuntimeOrdersPurgeRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	resultado, err := runtimesService.PurgeTerminalRuntimeOrders(runtimesapp.RuntimeOrderPurgeRequest{
		Agente:           apiNombreAgenteCanonico(req.Agente),
		Proyecto:         strings.TrimSpace(req.Proyecto),
		Estados:          req.Estados,
		Tipos:            req.Tipos,
		OlderThanMinutes: req.OlderThanMinutes,
		Actor:            strings.TrimSpace(req.Actor),
	})
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiRuntimeOrdersPurgeResponse{
		OK:         true,
		Deleted:    resultado.Deleted,
		DeletedIDs: resultado.DeletedIDs,
		Estados:    resultado.Estados,
		Tipos:      resultado.Tipos,
	})
}

func apiHandlerRuntimeOrdersCancelar(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req struct {
		ID     int64  `json:"id"`
		Actor  string `json:"actor"`
		Motivo string `json:"motivo"`
	}
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	if req.ID <= 0 {
		apiError(w, http.StatusBadRequest, fmt.Errorf("id inválido"))
		return
	}
	if err := runtimesService.CancelRuntimeOrder(req.ID, strings.TrimSpace(req.Actor), strings.TrimSpace(req.Motivo)); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, map[string]any{"ok": true, "id": req.ID})
}

func apiHandlerRuntimeMailbox(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		filter := db.FiltroRuntimeMailbox{Limit: apiRuntimeMailboxDefaultLimit}
		if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
			limit, err := strconv.Atoi(raw)
			if err != nil || limit <= 0 {
				apiError(w, http.StatusBadRequest, fmt.Errorf("limit inválido"))
				return
			}
			filter.Limit = limit
		}
		if toAgente := strings.TrimSpace(r.URL.Query().Get("to_agente")); toAgente != "" {
			toAgente = apiNombreAgenteCanonico(toAgente)
			filter.ToAgente = &toAgente
		}
		if fromAgente := strings.TrimSpace(r.URL.Query().Get("from_agente")); fromAgente != "" {
			fromAgente = apiNombreAgenteCanonico(fromAgente)
			filter.FromAgente = &fromAgente
		}
		if estado := strings.TrimSpace(r.URL.Query().Get("estado")); estado != "" {
			filter.Estado = &estado
		}
		if proyecto := strings.TrimSpace(r.URL.Query().Get("proyecto")); proyecto != "" {
			proyectoID, err := apiResolveProjectIDTimeboxed(proyecto)
			if err != nil {
				if errors.Is(err, errStatusFetchTimeout) {
					apiError(w, http.StatusServiceUnavailable, fmt.Errorf("runtime mailbox temporalmente degradado"))
					return
				}
				apiError(w, http.StatusBadRequest, err)
				return
			}
			filter.ProyectoID = proyectoID
		}
		mailbox, err := runAPITimeboxed(apiListRuntimeMailboxTimeout, func() ([]*db.RuntimeMailboxMessage, error) {
			return apiListRuntimeMailboxFn(filter)
		}, errStatusFetchTimeout)
		if err != nil {
			if errors.Is(err, errStatusFetchTimeout) {
				apiError(w, http.StatusServiceUnavailable, fmt.Errorf("runtime mailbox temporalmente degradado"))
				return
			}
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
			FromAgente:     apiNombreAgenteCanonico(req.FromAgente),
			ToAgente:       apiNombreAgenteCanonico(req.ToAgente),
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
	if len(parts) == 1 && parts[0] != "" {
		id, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil || id <= 0 {
			apiError(w, http.StatusBadRequest, fmt.Errorf("id inválido"))
			return
		}
		if !apiRequireMethod(w, r, http.MethodGet) {
			return
		}
		msg, err := runtimesService.GetRuntimeMailbox(id)
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		if msg == nil {
			apiError(w, http.StatusNotFound, fmt.Errorf("runtime mailbox no existe"))
			return
		}
		apiWriteJSON(w, http.StatusOK, apiRuntimeMailboxItemResponse{Message: msg})
		return
	}
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

func apiHandlerRuntimeWake(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req apiRuntimeWakeRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	resp := apiRuntimeWakeResponse{}
	if req.Orders {
		resp.Orders = wakeControlPlaneRuntimeOrders()
	}
	if req.Mailbox {
		resp.Mailbox = wakeControlPlaneRuntimeMailbox()
	}
	if req.Warm {
		resp.Warm = wakeControlPlaneWarm()
	}
	apiWriteJSON(w, http.StatusOK, resp)
}

func apiHandlerRuntimeProcessOrders(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	count, err := runAPITimeboxed(apiRuntimeProcessOrdersTimeout, func() (int, error) {
		return apiRuntimeProcessOrdersExecutor()
	}, errStatusFetchTimeout)
	if err != nil {
		if errors.Is(err, errStatusFetchTimeout) {
			apiError(w, http.StatusServiceUnavailable, fmt.Errorf("runtime orders temporalmente degradado"))
			return
		}
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiRuntimeProcessOrdersResponse{
		OK:    true,
		Count: count,
	})
}

func apiHandlerRuntimeProcessMailbox(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req apiRuntimeProcessMailboxRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	estado := "pendiente"
	filter := db.FiltroRuntimeMailbox{Estado: &estado}
	if toAgente := strings.TrimSpace(req.ToAgente); toAgente != "" {
		toAgente = apiNombreAgenteCanonico(toAgente)
		filter.ToAgente = &toAgente
	}
	if proyecto := strings.TrimSpace(req.Proyecto); proyecto != "" {
		p, err := runAPITimeboxed(apiRuntimeProcessMailboxTimeout, func() (*db.Proyecto, error) {
			return apiRuntimeProcessMailboxProjectFn(proyecto)
		}, errStatusFetchTimeout)
		if err != nil {
			if errors.Is(err, errStatusFetchTimeout) {
				apiError(w, http.StatusServiceUnavailable, fmt.Errorf("runtime mailbox temporalmente degradado"))
				return
			}
			apiError(w, http.StatusBadRequest, err)
			return
		}
		filter.ProyectoID = &p.ID
	}
	resetRuntimeMailboxReevaluationGate()
	count, err := runAPITimeboxed(apiRuntimeProcessMailboxTimeout, func() (int, error) {
		return apiRuntimeProcessMailboxExecutor(filter)
	}, errStatusFetchTimeout)
	if err != nil {
		if errors.Is(err, errStatusFetchTimeout) {
			apiError(w, http.StatusServiceUnavailable, fmt.Errorf("runtime mailbox temporalmente degradado"))
			return
		}
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiRuntimeProcessMailboxResponse{
		OK:    true,
		Count: count,
	})
}

func apiHandlerRuntimeProcessAutonomia(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req apiRuntimeProcessAutonomiaRequest
	if payload, err := io.ReadAll(r.Body); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	} else if raw := strings.TrimSpace(string(payload)); raw != "" {
		if err := json.Unmarshal(payload, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
	}
	resetAutonomiaActiveSessionsObservationGate()
	resetAutonomiaIdleAutoassignGate()
	resetAutonomiaDegradedTaskGate()
	if req.Wait {
		count, err := runtimeProcessAutonomiaBatch()
		if err != nil {
			db.Audit("server", "runtime_process_autonomia_error", "runtime", 0, err.Error())
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, apiRuntimeProcessAutonomiaResponse{
			OK:    true,
			Count: count,
		})
		return
	}
	started := runtimeProcessAutonomiaEnCurso.CompareAndSwap(false, true)
	if started {
		go func() {
			defer runtimeProcessAutonomiaEnCurso.Store(false)
			if _, err := runtimeProcessAutonomiaBatch(); err != nil {
				db.Audit("server", "runtime_process_autonomia_error", "runtime", 0, err.Error())
			}
		}()
	}
	apiWriteJSON(w, http.StatusOK, apiRuntimeProcessAutonomiaResponse{
		OK:       true,
		Count:    0,
		Accepted: started,
		Running:  runtimeProcessAutonomiaEnCurso.Load(),
	})
}

func apiHandlerRuntimeProcessTranscript(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req apiRuntimeProcessTranscriptRequest
	if payload, err := io.ReadAll(r.Body); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	} else if raw := strings.TrimSpace(string(payload)); raw != "" {
		if err := json.Unmarshal(payload, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
	}
	if req.HandleID > 0 {
		count, err := db.IngestarRuntimeTranscriptHandle(req.HandleID)
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, apiRuntimeProcessAutonomiaResponse{OK: true, Count: count})
		return
	}
	if agente := strings.TrimSpace(req.Agente); agente != "" {
		agente = apiNombreAgenteCanonico(agente)
		var (
			handle     *db.RuntimeHandle
			err        error
			proyectoID *int64
		)
		if proyecto := strings.TrimSpace(req.Proyecto); proyecto != "" {
			p, err := apiRuntimeProcessMailboxProjectFn(proyecto)
			if err != nil {
				apiError(w, http.StatusBadRequest, err)
				return
			}
			proyectoID = &p.ID
			handle, err = db.GetRuntimeHandleCanonicoRecienteAgenteProyecto(agente, proyectoID)
		} else {
			handle, err = db.GetRuntimeHandleCanonicoRecienteAgente(agente)
		}
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		if handle == nil || handle.ID <= 0 {
			apiWriteJSON(w, http.StatusOK, apiRuntimeProcessAutonomiaResponse{OK: true, Count: 0})
			return
		}
		count, err := db.IngestarRuntimeTranscriptHandle(handle.ID)
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, apiRuntimeProcessAutonomiaResponse{OK: true, Count: count})
		return
	}
	if req.Wait {
		count, err := runtimeProcessTranscriptBatch()
		if err != nil {
			db.Audit("server", "runtime_process_transcript_error", "runtime", 0, err.Error())
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, apiRuntimeProcessAutonomiaResponse{
			OK:    true,
			Count: count,
		})
		return
	}
	started := runtimeProcessTranscriptEnCurso.CompareAndSwap(false, true)
	if started {
		go func() {
			defer runtimeProcessTranscriptEnCurso.Store(false)
			if _, err := runtimeProcessTranscriptBatch(); err != nil {
				db.Audit("server", "runtime_process_transcript_error", "runtime", 0, err.Error())
			}
		}()
	}
	apiWriteJSON(w, http.StatusOK, apiRuntimeProcessAutonomiaResponse{
		OK:       true,
		Count:    0,
		Accepted: started,
		Running:  runtimeProcessTranscriptEnCurso.Load(),
	})
}

func apiHandlerRuntimePurgeTranscriptNoise(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req apiRuntimePurgeTranscriptNoiseRequest
	if payload, err := io.ReadAll(r.Body); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	} else if raw := strings.TrimSpace(string(payload)); raw != "" {
		if err := json.Unmarshal(payload, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
	}
	total := 0
	maxBatches := 20
	purgeFn := apiRuntimePurgeTranscriptNoiseExecutor
	if req.All {
		purgeFn = apiRuntimePurgeTranscriptNoiseAllExecutor
		maxBatches = 1
	}
	for i := 0; i < maxBatches; i++ {
		count, err := purgeFn()
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		total += count
		if !req.All || count == 0 {
			break
		}
	}
	apiWriteJSON(w, http.StatusOK, apiRuntimeProcessAutonomiaResponse{
		OK:    true,
		Count: total,
	})
}

func apiHandlerRuntimeProcessDegradados(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req apiRuntimeProcessAutonomiaRequest
	if payload, err := io.ReadAll(r.Body); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	} else if raw := strings.TrimSpace(string(payload)); raw != "" {
		if err := json.Unmarshal(payload, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
	}
	resetAutonomiaDegradedTaskGate()
	if req.Wait {
		started := runtimeProcessDegradadosEnCurso.CompareAndSwap(false, true)
		if !started {
			apiWriteJSON(w, http.StatusOK, apiRuntimeProcessAutonomiaResponse{
				OK:       true,
				Count:    0,
				Accepted: false,
				Running:  true,
			})
			return
		}
		type result struct {
			summary runtimeProcessDegradadosSummary
			err     error
		}
		done := make(chan result, 1)
		go func() {
			defer runtimeProcessDegradadosEnCurso.Store(false)
			summary, err := runtimeProcessDegradadosBatchDetailed()
			if err != nil {
				db.Audit("server", "runtime_process_degradados_error", "runtime", 0, err.Error())
			}
			done <- result{summary: summary, err: err}
		}()
		select {
		case out := <-done:
			if out.err != nil {
				apiError(w, http.StatusInternalServerError, out.err)
				return
			}
			apiWriteJSON(w, http.StatusOK, apiRuntimeProcessAutonomiaResponse{
				OK:                        true,
				Count:                     out.summary.Count,
				GhostAssignmentsCompacted: out.summary.GhostAssignmentsCompacted,
				ReactivatedWithoutRuntime: out.summary.ReactivatedWithoutRuntime,
				IdleAutoassigned:          out.summary.IdleAutoassigned,
			})
		case <-time.After(runtimeProcessDegradadosWaitTimeout):
			apiWriteJSON(w, http.StatusOK, apiRuntimeProcessAutonomiaResponse{
				OK:       true,
				Count:    0,
				Accepted: true,
				Running:  true,
			})
		}
		return
	}
	started := runtimeProcessDegradadosEnCurso.CompareAndSwap(false, true)
	if started {
		go func() {
			defer runtimeProcessDegradadosEnCurso.Store(false)
			if _, err := runtimeProcessDegradadosBatch(); err != nil {
				db.Audit("server", "runtime_process_degradados_error", "runtime", 0, err.Error())
			}
		}()
	}
	apiWriteJSON(w, http.StatusOK, apiRuntimeProcessAutonomiaResponse{
		OK:       true,
		Count:    0,
		Accepted: started,
		Running:  runtimeProcessDegradadosEnCurso.Load(),
	})
}

func apiHandlerRuntimeProcessHygiene(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req apiRuntimeProcessAutonomiaRequest
	if payload, err := io.ReadAll(r.Body); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	} else if raw := strings.TrimSpace(string(payload)); raw != "" {
		if err := json.Unmarshal(payload, &req); err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
	}
	if req.Wait {
		count, err := runtimeProcessHygieneBatch()
		if err != nil {
			db.Audit("server", "runtime_process_hygiene_error", "runtime", 0, err.Error())
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		apiWriteJSON(w, http.StatusOK, apiRuntimeProcessAutonomiaResponse{
			OK:    true,
			Count: count,
		})
		return
	}
	go func() {
		if _, err := runtimeProcessHygieneBatch(); err != nil {
			db.Audit("server", "runtime_process_hygiene_error", "runtime", 0, err.Error())
		}
	}()
	apiWriteJSON(w, http.StatusOK, apiRuntimeProcessAutonomiaResponse{
		OK:       true,
		Count:    0,
		Accepted: true,
		Running:  true,
	})
}

func apiHandlerRuntimeProcessReanimaciones(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req struct {
		Wait bool `json:"wait"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Wait {
		resp := runtimeProcessReanimationsBatchFn()
		if resp.Errors > 0 {
			db.Audit("server", "runtime_process_reanimaciones_wait_errors", "runtime", 0,
				fmt.Sprintf("candidates=%d reactivated=%d cooldown_sustained=%d capacity_blocked=%d errors=%d", resp.Candidates, resp.Reactivated, resp.CooldownSustained, resp.CapacityBlocked, resp.Errors))
		}
		apiWriteJSON(w, http.StatusOK, resp)
		return
	}
	started := runtimeProcessReanimacionesEnCurso.CompareAndSwap(false, true)
	if started {
		go func() {
			defer runtimeProcessReanimacionesEnCurso.Store(false)
			resp := runtimeProcessReanimationsBatchFn()
			if resp.Errors > 0 {
				db.Audit("server", "runtime_process_reanimaciones_background_errors", "runtime", 0,
					fmt.Sprintf("candidates=%d reactivated=%d cooldown_sustained=%d capacity_blocked=%d errors=%d", resp.Candidates, resp.Reactivated, resp.CooldownSustained, resp.CapacityBlocked, resp.Errors))
			}
		}()
	}
	apiWriteJSON(w, http.StatusOK, apiRuntimeProcessReanimationsResponse{
		OK:       true,
		Count:    0,
		Accepted: started,
		Running:  runtimeProcessReanimacionesEnCurso.Load(),
	})
}

func ejecutarRuntimeProcessReanimationsBatch() apiRuntimeProcessReanimationsResponse {
	resp := apiRuntimeProcessReanimationsResponse{OK: true}
	reanimar, err := (dbAutomationService{}).CheckReanimaciones()
	if err != nil {
		db.Audit("server", "runtime_process_reanimaciones_error", "runtime", 0, err.Error())
		resp.Errors++
		return resp
	}
	resp.Candidates = len(reanimar)
	for _, agente := range reanimar {
		if agente == nil || strings.TrimSpace(agente.Nombre) == "" {
			continue
		}
		resultado, err := (dbAutomationService{}).ResetReanimacionResultado(agente.Nombre)
		if err != nil {
			resp.Errors++
			db.Audit("server", "runtime_process_reanimaciones_reset_error", "agente", 0, fmt.Sprintf("agente=%s err=%v", strings.TrimSpace(agente.Nombre), err))
			continue
		}
		switch resultado {
		case resetReanimacionResultadoCooldownSostenido:
			resp.CooldownSustained++
			db.Audit("server", "runtime_process_reanimaciones_cooldown_sustained", "agente", 0, fmt.Sprintf("agente=%s motivo=%s", strings.TrimSpace(agente.Nombre), strings.TrimSpace(agente.MotivoPausa)))
		case resetReanimacionResultadoSinCapacidad:
			resp.CapacityBlocked++
			db.Audit("server", "runtime_process_reanimaciones_capacity_blocked", "agente", 0, fmt.Sprintf("agente=%s motivo=%s", strings.TrimSpace(agente.Nombre), strings.TrimSpace(agente.MotivoPausa)))
		case resetReanimacionResultadoReactivado:
			resp.Reactivated++
		}
	}
	resp.Count = resp.Reactivated
	db.Audit("server", "runtime_process_reanimaciones", "runtime", 0,
		fmt.Sprintf("candidates=%d reactivated=%d cooldown_sustained=%d capacity_blocked=%d errors=%d", resp.Candidates, resp.Reactivated, resp.CooldownSustained, resp.CapacityBlocked, resp.Errors))
	return resp
}

func launchAgentResetReanimation(nombre string) apiAgenteResetReanimacionResponse {
	nombre = strings.TrimSpace(nombre)
	if nombre == "" {
		return apiAgenteResetReanimacionResponse{OK: true}
	}
	flag := agentResetReanimationFlag(nombre)
	started := flag.CompareAndSwap(false, true)
	if started {
		func(agent string, running *atomic.Bool) {
			defer running.Store(false)
			resultado, err := (dbAutomationService{}).resetReanimacionResultadoConPermisoManual(agent, true)
			if err != nil {
				db.Audit("server", "agente_reset_reanimacion_error", "agente", 0,
					fmt.Sprintf("agente=%s err=%v", strings.TrimSpace(agent), err))
				return
			}
			db.Audit("server", "agente_reset_reanimacion", "agente", 0,
				fmt.Sprintf("agente=%s resultado=%s", strings.TrimSpace(agent), string(resultado)))
		}(nombre, flag)
	}
	return apiAgenteResetReanimacionResponse{
		OK:       true,
		Agente:   nombre,
		Accepted: started,
		Running:  started,
	}
}

func agentResetReanimationFlag(nombre string) *atomic.Bool {
	key := strings.ToLower(strings.TrimSpace(nombre))
	if key == "" {
		return &atomic.Bool{}
	}
	if existing, ok := apiAgentResetReanimacionEnCurso.Load(key); ok {
		if flag, ok := existing.(*atomic.Bool); ok && flag != nil {
			return flag
		}
	}
	flag := &atomic.Bool{}
	actual, _ := apiAgentResetReanimacionEnCurso.LoadOrStore(key, flag)
	if resolved, ok := actual.(*atomic.Bool); ok && resolved != nil {
		return resolved
	}
	return flag
}

func apiHandlerRuntimeMailboxClear(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req apiRuntimeMailboxClearRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	filter := db.FiltroRuntimeMailbox{}
	if toAgente := strings.TrimSpace(req.ToAgente); toAgente != "" {
		toAgente = apiNombreAgenteCanonico(toAgente)
		filter.ToAgente = &toAgente
	}
	if fromAgente := strings.TrimSpace(req.FromAgente); fromAgente != "" {
		fromAgente = apiNombreAgenteCanonico(fromAgente)
		filter.FromAgente = &fromAgente
	}
	if proyecto := strings.TrimSpace(req.Proyecto); proyecto != "" {
		p, err := runtimesService.GetProject(proyecto)
		if err != nil {
			apiError(w, http.StatusBadRequest, err)
			return
		}
		filter.ProyectoID = &p.ID
	}
	estados := normalizarSliceFlags(req.Estados)
	if len(estados) == 0 {
		estados = []string{"pendiente"}
	}
	kinds := normalizarSliceFlags(req.Kinds)
	ids := make([]int64, 0)
	for _, estado := range estados {
		estadoLocal := estado
		filter.Estado = &estadoLocal
		items, err := runtimesService.ListRuntimeMailbox(filter)
		if err != nil {
			apiError(w, http.StatusInternalServerError, err)
			return
		}
		for _, item := range items {
			if item == nil || item.ID <= 0 {
				continue
			}
			if len(kinds) > 0 {
				match := false
				for _, kind := range kinds {
					if strings.EqualFold(strings.TrimSpace(item.Kind), kind) {
						match = true
						break
					}
				}
				if !match {
					continue
				}
			}
			if err := runtimesService.MarkRuntimeMailboxConsumed(item.ID); err != nil {
				apiError(w, http.StatusInternalServerError, err)
				return
			}
			ids = append(ids, item.ID)
		}
	}
	apiWriteJSON(w, http.StatusOK, apiRuntimeMailboxClearResponse{
		OK:         true,
		Cleared:    len(ids),
		ClearedIDs: ids,
	})
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
	id, err := runtimesService.CreateLiveAgentHandoff(
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
		ArranqueLimpio:    req.ArranqueLimpio,
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
	if req.LimpiarContinuidad {
		empty := ""
		upd.ExternalSessionID = &empty
		upd.ResumePayloadJSON = &empty
		upd.ResumenContinuidad = &empty
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
		result, err := sesionesAPIService.RegisterBudget(req.SesionID, req.Agente, &sesionesapp.PresupuestoSesion{
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

	out, err := runAPIAgentPrepareLimited(apiAgentPrepareTimeout, func() (*agentesapp.PrepareOutput, error) {
		return apiAgentPrepareBuilder(agentesapp.PrepareInput{
			Agente:       agenteNombre,
			Proyecto:     proyectoRef,
			Conector:     conectorRef,
			Modelo:       modelo,
			Razonamiento: razonamiento,
			Perfil:       perfilTarea,
		})
	})
	if err != nil {
		if errors.Is(err, errStatusFetchTimeout) {
			apiError(w, http.StatusServiceUnavailable, fmt.Errorf("prepare temporalmente degradado"))
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			apiError(w, http.StatusNotFound, err)
			return
		}
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
		Agente              string  `json:"agente"`
		Proyecto            string  `json:"proyecto"`
		Host                string  `json:"host"`
		PID                 int64   `json:"pid"`
		CuotaPct            int     `json:"cuota_pct"`
		Finalizado          bool    `json:"finalizado"`
		Motivo              string  `json:"motivo"`
		AckStartOrderID     int64   `json:"ack_start_order_id"`
		AckBootstrapOrderID int64   `json:"ack_bootstrap_order_id"`
		AckMailboxIDs       []int64 `json:"ack_mailbox_ids"`
	}
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	if req.Agente == "" || req.Proyecto == "" {
		apiError(w, http.StatusBadRequest, fmt.Errorf("debes indicar agente y proyecto"))
		return
	}
	tickInput := agentesapp.TickInput{
		Agente:              req.Agente,
		Proyecto:            req.Proyecto,
		Host:                req.Host,
		PID:                 req.PID,
		CuotaPct:            req.CuotaPct,
		Finalizado:          req.Finalizado,
		Motivo:              req.Motivo,
		AckStartOrderID:     req.AckStartOrderID,
		AckBootstrapOrderID: req.AckBootstrapOrderID,
		AckMailboxIDs:       req.AckMailboxIDs,
	}
	out, err := runAPIAgentTickLimited(apiAgentTickTimeout, func() (*agentesapp.TickOutput, error) {
		return apiAgentTickProcessor(tickInput)
	})
	if err != nil {
		if errors.Is(err, errStatusFetchTimeout) {
			apiError(w, http.StatusServiceUnavailable, fmt.Errorf("tick temporalmente degradado"))
			return
		}
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
