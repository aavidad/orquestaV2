package agentesapp

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"orquesta/db"
	"orquesta/runtimeagente"
)

type Store interface {
	RegisterAgent(nombre, rol string) error
	RegisterAgentAuto(proveedor, rol string) (string, error)
	ObserveAgentIdentity(nombre, email, usuario, fuente string, observedAt *time.Time) error
	RetireAgent(nombre string) error
	RehabilitateAgent(nombre string) error
	ResetReanimation(nombre string) error
	DeleteAgent(nombre string) error
	MergeAgents(origen, destino string) (*db.FusionAgentesResultado, error)
	Audit(agente, accion, entidad string, entidadID int64, detalle string)
	GetAgent(nombre string) (*db.Agente, error)
	GetProject(ref string) (*db.Proyecto, error)
	GetPool(slug string) (*db.PoolCapacidad, error)
	GetConnector(ref string) (*db.Conector, error)
	ListAgents() ([]*db.Agente, error)
	CheckReanimations() ([]*db.Agente, error)
	ListAssignments(filtro db.FiltroAsignaciones) ([]*db.Asignacion, error)
	ListInspectionSessions(filtro db.FiltroSesionesInspeccion) ([]*db.Sesion, error)
	GetLastSession(agente string, proyectoID *int64) (*db.Sesion, error)
	GetActiveSession(agente string, proyectoID *int64) (*db.Sesion, error)
	SaveActiveSession(agente string, proyectoID *int64, upd db.SesionUpdate) error
	ListRuntimes(filtro db.FiltroRuntimes) ([]*db.RuntimeInstance, error)
	ListRuntimeHandles(agente *string) ([]*db.RuntimeHandle, error)
	ListPassiveRuntimeHandles(agente *string) ([]*db.RuntimeHandle, error)
	ListCanonicalRuntimeHandles(agente *string) ([]*db.RuntimeHandle, error)
	ListRecentOperationalRuntimeHandles() (map[string]*db.RuntimeHandle, error)
	ListRuntimeTranscript(filtro db.FiltroRuntimeTranscript) ([]*db.RuntimeTranscriptEntry, error)
	ListRuntimeOrders(filtro db.FiltroRuntimeOrders) ([]*db.RuntimeOrder, error)
	EnqueueRuntimeOrder(order *db.RuntimeOrder) (int64, error)
	ListRuntimeMailbox(filtro db.FiltroRuntimeMailbox) ([]*db.RuntimeMailboxMessage, error)
	RuntimeMailboxCoveredByBootstrapPending(mailboxID int64, handle *db.RuntimeHandle, runtime *db.RuntimeInstance) (bool, int64, int64, error)
	ListRuntimeCheckpoints(filtro db.FiltroRuntimeCheckpoints) ([]*db.RuntimeCheckpoint, error)
	ListTasks(filtro db.FiltroTareas) ([]*db.Tarea, error)
	ListProjectPendingVotes(agente string, proyectoID int64) ([]*db.Propuesta, error)
	ListProjectOpenProposals(proyectoID int64) ([]*db.Propuesta, error)
	GetLatestAgentBudget(agente string) (*db.PresupuestoSesion, *db.Sesion, error)
	ResolveGovernanceCatalog(rol string, proyectoID *int64) (*db.GovernanceCatalog, error)
	ResolveGovernanceCatalogForContext(rol string, proyectoID *int64, agente string) (*db.GovernanceCatalog, error)
	ListRules(rol string) ([]*db.Regla, error)
	ListSkills(rol string) ([]*db.Skill, error)
	ListWorkflows(rol string) ([]*db.Workflow, error)
	ListMemoryEntities(filtro db.FiltroEntidadesMemoria) ([]*db.EntidadMemoria, error)
	SharedAccountAllowsActivation(nombre string) (bool, string, error)
	ConfigGet(clave string) (string, error)
	PauseAgent(nombre string, minutos int, motivo string) error
}

type ModelPolicyProvider interface {
	ResolveModelPolicy(input db.ResolverPoliticaInput) (*db.ResolucionModelo, error)
}

type Service struct {
	store                    Store
	modelPolicyProvider      ModelPolicyProvider
	cacheMu                  sync.Mutex
	prepareAgentCache        map[string]cachedPrepareAgent
	prepareAgentFlight       map[string]*prepareAgentFlight
	prepareProjectCache      map[string]cachedPrepareProject
	prepareProjectFlight     map[string]*prepareProjectFlight
	tickProjectFlight        map[string]*prepareProjectFlight
	prepareSessionCache      map[string]cachedPrepareSession
	prepareSessionFlight     map[string]*prepareSessionFlight
	prepareConnectorCache    map[string]cachedPrepareConnector
	prepareConnectorFlight   map[string]*prepareConnectorFlight
	prepareGovCache          map[string]cachedPrepareGovernance
	prepareGovFlight         map[string]*prepareGovernanceFlight
	prepareModelPolicyCache  map[string]cachedPrepareModelPolicy
	prepareModelPolicyFlight map[string]*prepareModelPolicyFlight
	prepareOutputCache       map[string]cachedPrepareOutput
	prepareOutputFlight      map[string]*prepareOutputFlight
	tickOutputCache          map[string]cachedTickOutput
	tickOutputFlight         map[string]*tickOutputFlight
	compactDetailCache       map[string]cachedCompactDetail
	compactDetailFlight      map[string]*compactDetailFlight
	reanimCache              map[string]cachedReanimationSchedule
	reanimFlight             map[string]*reanimationScheduleFlight
}

const defaultWorkerOutputStaleSeconds = 20 * 60

var compactDetailCacheTTL = 2 * time.Second
var compactDetailStaleWhileRevalidateTTL = 15 * time.Second
var activeReanimationScheduleCacheTTL = 2 * time.Second
var activeReanimationScheduleStaleWhileRevalidateTTL = 15 * time.Second

// Prepare usa datos operativos que cambian poco frente al coste de caer a
// SQLite bajo carga. Mantener una cache algo más larga evita ráfagas de
// prepare degradadas cuando warm/autonomía pisan la única conexión activa.
var prepareEntityCacheTTL = 45 * time.Second
var prepareSessionCacheTTL = 45 * time.Second
var prepareGovernanceCacheTTL = 45 * time.Second
var prepareModelPolicyCacheTTL = 45 * time.Second
var prepareOutputCacheTTL = 20 * time.Second
var tickOutputCacheTTL = 1500 * time.Millisecond

func agentDetailDebugf(format string, args ...any) {
	if !agentDetailDebugEnabled() {
		return
	}
	log.Printf("orquesta[agentes-detail] "+format, args...)
}

func agentDetailDebugEnabled() bool {
	for _, key := range []string{"ORQUESTA_DEBUG_AGENT_DETAIL", "ORQUESTA_DEBUG"} {
		value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
		switch value {
		case "1", "true", "yes", "on", "si", "sí":
			return true
		}
	}
	return false
}

func NewService(store Store, modelPolicyProvider ModelPolicyProvider) *Service {
	return &Service{
		store:                    store,
		modelPolicyProvider:      modelPolicyProvider,
		prepareAgentCache:        map[string]cachedPrepareAgent{},
		prepareAgentFlight:       map[string]*prepareAgentFlight{},
		prepareProjectCache:      map[string]cachedPrepareProject{},
		prepareProjectFlight:     map[string]*prepareProjectFlight{},
		tickProjectFlight:        map[string]*prepareProjectFlight{},
		prepareSessionCache:      map[string]cachedPrepareSession{},
		prepareSessionFlight:     map[string]*prepareSessionFlight{},
		prepareConnectorCache:    map[string]cachedPrepareConnector{},
		prepareConnectorFlight:   map[string]*prepareConnectorFlight{},
		prepareGovCache:          map[string]cachedPrepareGovernance{},
		prepareGovFlight:         map[string]*prepareGovernanceFlight{},
		prepareModelPolicyCache:  map[string]cachedPrepareModelPolicy{},
		prepareModelPolicyFlight: map[string]*prepareModelPolicyFlight{},
		prepareOutputCache:       map[string]cachedPrepareOutput{},
		prepareOutputFlight:      map[string]*prepareOutputFlight{},
		tickOutputCache:          map[string]cachedTickOutput{},
		tickOutputFlight:         map[string]*tickOutputFlight{},
		compactDetailCache:       map[string]cachedCompactDetail{},
		compactDetailFlight:      map[string]*compactDetailFlight{},
		reanimCache:              map[string]cachedReanimationSchedule{},
		reanimFlight:             map[string]*reanimationScheduleFlight{},
	}
}

type cachedCompactDetail struct {
	detail  *Detail
	expires time.Time
}

type cachedPrepareAgent struct {
	agent   *db.Agente
	expires time.Time
}

type cachedPrepareProject struct {
	project *db.Proyecto
	expires time.Time
}

type cachedPrepareGovernance struct {
	catalog *db.GovernanceCatalog
	expires time.Time
}

type cachedPrepareSession struct {
	session *db.Sesion
	found   bool
	expires time.Time
}

type cachedPrepareConnector struct {
	connector *db.Conector
	expires   time.Time
}

type cachedPrepareModelPolicy struct {
	resolution *db.ResolucionModelo
	expires    time.Time
}

type cachedPrepareOutput struct {
	output  *PrepareOutput
	expires time.Time
}

type cachedTickOutput struct {
	output  *TickOutput
	expires time.Time
}

type prepareAgentFlight struct {
	done  chan struct{}
	agent *db.Agente
	err   error
}

type prepareProjectFlight struct {
	done    chan struct{}
	project *db.Proyecto
	err     error
}

type prepareGovernanceFlight struct {
	done    chan struct{}
	catalog *db.GovernanceCatalog
	err     error
}

type prepareSessionFlight struct {
	done    chan struct{}
	session *db.Sesion
	found   bool
	err     error
}

type prepareConnectorFlight struct {
	done      chan struct{}
	connector *db.Conector
	err       error
}

type prepareModelPolicyFlight struct {
	done       chan struct{}
	resolution *db.ResolucionModelo
	err        error
}

type prepareOutputFlight struct {
	done   chan struct{}
	output *PrepareOutput
	err    error
}

type tickOutputFlight struct {
	done   chan struct{}
	output *TickOutput
	err    error
}

type compactDetailFlight struct {
	done   chan struct{}
	detail *Detail
	err    error
}

type cachedReanimationSchedule struct {
	rows    []ReanimationCandidate
	expires time.Time
}

type reanimationScheduleFlight struct {
	done chan struct{}
	rows []ReanimationCandidate
	err  error
}

type Row struct {
	Agente                      *db.Agente
	Asignacion                  *db.Asignacion
	Sesion                      *db.Sesion
	Runtime                     *db.RuntimeInstance
	Handle                      *db.RuntimeHandle
	WorkerState                 string
	WorkerAlive                 bool
	WorkerReadyAt               *time.Time
	WorkerHeartbeat             *time.Time
	WorkerUpdatedAt             *time.Time
	WorkerLastOutput            *time.Time
	WorkerLastProgress          *time.Time
	WorkerExitError             string
	WorkerSessionRef            string
	WorkerRuntimeRef            string
	WorkerDriver                string
	WorkerTransport             string
	WorkerTMUXSession           string
	WorkerTMUXWindow            string
	WorkerTMUXPaneID            string
	WorkerCanSendInput          bool
	WorkerExternalSessionID     string
	WorkerMailboxDeliveryMode   string
	EstadoOperativo             string
	DetalleOperativo            string
	OrdersOpen                  int
	OrdersFailed                int
	ControlOrdersOpen           int
	LastControlOrderType        string
	LastControlOrderMoment      *time.Time
	MailboxPending              int
	MailboxActionablePending    int
	MailboxContinuityPending    int
	MailboxTotal                int
	LastAutonomyAction          string
	LastAutonomySource          string
	LastAutonomyMoment          *time.Time
	LastAutonomyState           string
	LastAutonomyReason          string
	LastAutonomyVerificationKey string
	LastAutonomyDispatchState   string
	LastAutonomyDeliveryState   string
	LastAutonomyReceiptSource   string
	Checkpoints                 int
	LastCheckpoint              *db.RuntimeCheckpoint
	OpenTasks                   int
	BlockedTasks                int
}

type Detail struct {
	Row                     Row                          `json:"row"`
	Entity                  *AgentEntity                 `json:"entity,omitempty"`
	Asignaciones            []*db.Asignacion             `json:"asignaciones,omitempty"`
	Sesiones                []*db.Sesion                 `json:"sesiones,omitempty"`
	Runtimes                []*db.RuntimeInstance        `json:"runtimes,omitempty"`
	Handles                 []*db.RuntimeHandle          `json:"handles,omitempty"`
	Transcript              []*db.RuntimeTranscriptEntry `json:"transcript,omitempty"`
	Orders                  []*db.RuntimeOrder           `json:"orders,omitempty"`
	Mailbox                 []*db.RuntimeMailboxMessage  `json:"mailbox,omitempty"`
	MailboxTotalCount       int                          `json:"mailbox_total_count"`
	MailboxPendingVisible   int                          `json:"mailbox_pending_visible"`
	MailboxCoveredBootstrap int                          `json:"mailbox_covered_bootstrap"`
	Checkpoints             []*db.RuntimeCheckpoint      `json:"checkpoints,omitempty"`
}

type ReanimationCandidate struct {
	Name              string      `json:"name"`
	Role              string      `json:"role,omitempty"`
	Enabled           bool        `json:"enabled"`
	ActiveNow         bool        `json:"active_now"`
	AssignmentProject string      `json:"assignment_project,omitempty"`
	EstadoCuota       string      `json:"estado_cuota,omitempty"`
	MotivoPausa       string      `json:"motivo_pausa,omitempty"`
	ReanimarAt        *time.Time  `json:"reanimar_at,omitempty"`
	Due               bool        `json:"due"`
	RuntimeState      string      `json:"runtime_state,omitempty"`
	HandleState       string      `json:"handle_state,omitempty"`
	WorkerState       string      `json:"worker_state,omitempty"`
	OperationalState  string      `json:"operational_state,omitempty"`
	OperationalDetail string      `json:"operational_detail,omitempty"`
	MailboxPending    int         `json:"mailbox_pending"`
	OpenTasks         int         `json:"open_tasks"`
	BlockedTasks      int         `json:"blocked_tasks"`
	Leases            []WorkLease `json:"leases,omitempty"`
}

type AgentEntity struct {
	Name                        string      `json:"name"`
	Role                        string      `json:"role,omitempty"`
	Enabled                     bool        `json:"enabled"`
	ActiveNow                   bool        `json:"active_now"`
	AccountID                   string      `json:"account_id,omitempty"`
	AccountEmail                string      `json:"account_email,omitempty"`
	AccountUser                 string      `json:"account_user,omitempty"`
	AccountKey                  string      `json:"account_key,omitempty"`
	AccountAvailable            bool        `json:"account_available"`
	AccountOccupiedBy           string      `json:"account_occupied_by,omitempty"`
	AssignmentProject           string      `json:"assignment_project,omitempty"`
	RuntimeAdapter              string      `json:"runtime_adapter,omitempty"`
	Transport                   string      `json:"transport,omitempty"`
	RuntimeState                string      `json:"runtime_state,omitempty"`
	HandleState                 string      `json:"handle_state,omitempty"`
	WorkerState                 string      `json:"worker_state,omitempty"`
	WorkerAlive                 bool        `json:"worker_alive"`
	WorkerDriver                string      `json:"worker_driver,omitempty"`
	WorkerTransport             string      `json:"worker_transport,omitempty"`
	WorkerSessionRef            string      `json:"worker_session_ref,omitempty"`
	WorkerRuntimeRef            string      `json:"worker_runtime_ref,omitempty"`
	ExternalSessionID           string      `json:"external_session_id,omitempty"`
	MailboxDeliveryMode         string      `json:"mailbox_delivery_mode,omitempty"`
	OperationalState            string      `json:"operational_state,omitempty"`
	OperationalDetail           string      `json:"operational_detail,omitempty"`
	LastAutonomyAction          string      `json:"last_autonomy_action,omitempty"`
	LastAutonomySource          string      `json:"last_autonomy_source,omitempty"`
	LastAutonomyMoment          *time.Time  `json:"last_autonomy_moment,omitempty"`
	LastAutonomyState           string      `json:"last_autonomy_state,omitempty"`
	LastAutonomyReason          string      `json:"last_autonomy_reason,omitempty"`
	LastAutonomyVerificationKey string      `json:"last_autonomy_verification_key,omitempty"`
	LastAutonomyDispatchState   string      `json:"last_autonomy_dispatch_state,omitempty"`
	LastAutonomyDeliveryState   string      `json:"last_autonomy_delivery_state,omitempty"`
	LastAutonomyReceiptSource   string      `json:"last_autonomy_receipt_source,omitempty"`
	Leases                      []WorkLease `json:"leases,omitempty"`
}

type WorkLease struct {
	TaskID    int64          `json:"task_id"`
	Title     string         `json:"title,omitempty"`
	State     db.EstadoTarea `json:"state"`
	ProjectID *int64         `json:"project_id,omitempty"`
	Module    string         `json:"module,omitempty"`
}

type InvestigationMatch struct {
	Transcript    *db.RuntimeTranscriptEntry `json:"transcript"`
	Runtime       *db.RuntimeInstance        `json:"runtime,omitempty"`
	TraceDir      string                     `json:"trace_dir,omitempty"`
	TraceManifest string                     `json:"trace_manifest,omitempty"`
	LogPath       string                     `json:"log_path,omitempty"`
	WorkingDir    string                     `json:"working_dir,omitempty"`
}

type InvestigationAgentResult struct {
	Agente         *db.Agente            `json:"agente"`
	Runtime        *db.RuntimeInstance   `json:"runtime,omitempty"`
	LastCheckpoint *db.RuntimeCheckpoint `json:"last_checkpoint,omitempty"`
	OpenTasks      int                   `json:"open_tasks"`
	Matches        []*InvestigationMatch `json:"matches"`
	handles        []*db.RuntimeHandle
	runtimes       []*db.RuntimeInstance
}

type InvestigationReport struct {
	Query        string                      `json:"query"`
	Proyecto     *db.Proyecto                `json:"proyecto,omitempty"`
	Results      []*InvestigationAgentResult `json:"results"`
	TotalMatches int                         `json:"total_matches"`
}

func (s *Service) RegisterAgent(nombre, rol string) error {
	return s.store.RegisterAgent(strings.TrimSpace(nombre), strings.TrimSpace(rol))
}

func (s *Service) RegisterAgentAuto(proveedor, rol string) (string, error) {
	return s.store.RegisterAgentAuto(strings.TrimSpace(proveedor), strings.TrimSpace(rol))
}

func (s *Service) ObserveAgentIdentity(nombre, email, usuario, fuente string, observedAt *time.Time) error {
	return s.store.ObserveAgentIdentity(strings.TrimSpace(nombre), strings.TrimSpace(email), strings.TrimSpace(usuario), strings.TrimSpace(fuente), observedAt)
}

func (s *Service) GetAgent(nombre string) (*db.Agente, error) {
	return s.store.GetAgent(strings.TrimSpace(nombre))
}

func (s *Service) ListAgents() ([]*db.Agente, error) {
	return s.store.ListAgents()
}

func (s *Service) ApplyStateAction(nombre, accion string) error {
	nombre = strings.TrimSpace(nombre)
	switch strings.TrimSpace(accion) {
	case "retirar":
		return s.retireAgent(nombre)
	case "rehabilitar":
		return s.store.RehabilitateAgent(nombre)
	case "reset-reanimacion":
		return s.store.ResetReanimation(nombre)
	default:
		return fmt.Errorf("acción de agente no soportada: %s", accion)
	}
}

func (s *Service) retireAgent(nombre string) error {
	if err := s.enqueueRetirementPauseIfNeeded(nombre); err != nil {
		return err
	}
	return s.store.RetireAgent(nombre)
}

func (s *Service) listCanonicalRuntimeHandlesWithFallback(nombre string) ([]*db.RuntimeHandle, error) {
	nombre = strings.TrimSpace(nombre)
	if nombre == "" {
		return nil, nil
	}
	handles, err := s.store.ListCanonicalRuntimeHandles(&nombre)
	if err != nil {
		return nil, err
	}
	if len(handles) > 0 {
		return handles, nil
	}
	return s.store.ListRuntimeHandles(&nombre)
}

func (s *Service) enqueueRetirementPauseIfNeeded(nombre string) error {
	handles, err := s.listCanonicalRuntimeHandlesWithFallback(nombre)
	if err != nil {
		return err
	}

	var (
		needsPause bool
		proyectoID *int64
	)
	for _, handle := range handles {
		if handle == nil {
			continue
		}
		if strings.TrimSpace(handle.Estado) != "activo" {
			continue
		}
		needsPause = true
		if proyectoID == nil && handle.ProyectoID != nil {
			id := *handle.ProyectoID
			proyectoID = &id
		}
	}
	if !needsPause {
		return nil
	}

	orders, err := s.store.ListRuntimeOrders(db.FiltroRuntimeOrders{Agente: &nombre})
	if err != nil {
		return err
	}
	for _, order := range orders {
		if order == nil {
			continue
		}
		if order.Tipo != "pause" && order.Tipo != "stop" {
			continue
		}
		switch strings.TrimSpace(order.Estado) {
		case "pendiente", "tomada", "ejecutando":
			return nil
		}
	}

	payloadJSON, err := json.Marshal(map[string]any{
		"accion": "pause",
		"motivo": "retiro_agente",
		"por":    "server",
	})
	if err != nil {
		return err
	}
	_, err = s.store.EnqueueRuntimeOrder(&db.RuntimeOrder{
		Agente:      nombre,
		ProyectoID:  proyectoID,
		Tipo:        "pause",
		PayloadJSON: string(payloadJSON),
	})
	return err
}

func (s *Service) DeleteAgent(nombre, actor string) error {
	nombre = strings.TrimSpace(nombre)
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "orquesta"
	}
	if err := s.store.DeleteAgent(nombre); err != nil {
		return err
	}
	s.store.Audit(actor, "purgar_agente", "agente", 0, nombre)
	return nil
}

func (s *Service) MergeAgents(origen, destino string) (*db.FusionAgentesResultado, error) {
	return s.store.MergeAgents(strings.TrimSpace(origen), strings.TrimSpace(destino))
}

func (s *Service) PauseTemporarily(nombre string, minutos int, motivo, accion, entidad, detalle string) error {
	nombre = strings.TrimSpace(nombre)
	motivo = strings.TrimSpace(motivo)
	if nombre == "" || minutos <= 0 || motivo == "" {
		return fmt.Errorf("debes indicar agente, minutos positivos y motivo")
	}
	if err := s.store.PauseAgent(nombre, minutos, motivo); err != nil {
		return err
	}
	if strings.TrimSpace(detalle) == "" {
		detalle = fmt.Sprintf("Bloqueado %d min por: %s", minutos, motivo)
	}
	s.store.Audit(nombre, valueOrFallback(strings.TrimSpace(accion), "pausa_externa"), valueOrFallback(strings.TrimSpace(entidad), "agente"), 0, strings.TrimSpace(detalle))
	return nil
}

func (s *Service) BuildPanelRows() ([]Row, error) {
	agentes, err := s.ListAgents()
	if err != nil {
		return nil, err
	}
	asignaciones, err := s.store.ListAssignments(db.FiltroAsignaciones{})
	if err != nil {
		return nil, err
	}
	activa := true
	sesiones, err := s.store.ListInspectionSessions(db.FiltroSesionesInspeccion{Activa: &activa})
	if err != nil {
		return nil, err
	}
	runtimes, err := s.store.ListRuntimes(db.FiltroRuntimes{})
	if err != nil {
		return nil, err
	}
	orders, err := s.store.ListRuntimeOrders(db.FiltroRuntimeOrders{})
	if err != nil {
		return nil, err
	}
	mailbox, err := s.store.ListRuntimeMailbox(db.FiltroRuntimeMailbox{})
	if err != nil {
		return nil, err
	}
	checkpoints, err := s.store.ListRuntimeCheckpoints(db.FiltroRuntimeCheckpoints{})
	if err != nil {
		return nil, err
	}
	tareas, err := s.store.ListTasks(db.FiltroTareas{})
	if err != nil {
		return nil, err
	}

	asignacionPorAgente := map[string]*db.Asignacion{}
	for _, asignacion := range asignaciones {
		if asignacion == nil || asignacion.Estado != db.AsignacionActiva {
			continue
		}
		if _, ok := asignacionPorAgente[asignacion.Agente]; !ok {
			asignacionPorAgente[asignacion.Agente] = asignacion
		}
	}

	sesionPorAgente := map[string]*db.Sesion{}
	for _, sesion := range sesiones {
		if sesion == nil {
			continue
		}
		if _, ok := sesionPorAgente[sesion.Agente]; !ok {
			sesionPorAgente[sesion.Agente] = sesion
		}
	}

	runtimePorAgente := map[string]*db.RuntimeInstance{}
	for _, runtime := range runtimes {
		if runtime == nil {
			continue
		}
		actual := runtimePorAgente[runtime.Agente]
		if actual == nil || runtimeMoment(runtime).After(runtimeMoment(actual)) {
			runtimePorAgente[runtime.Agente] = runtime
		}
	}

	handlePorAgente := map[string]*db.RuntimeHandle{}
	if hotHandles, err := s.store.ListRecentOperationalRuntimeHandles(); err == nil {
		for _, handle := range hotHandles {
			if handle == nil {
				continue
			}
			actual := handlePorAgente[handle.Agente]
			if actual == nil || runtimeHandleMoment(handle).After(runtimeHandleMoment(actual)) {
				handlePorAgente[handle.Agente] = handle
			}
		}
	}
	handlesCanonicos, err := s.store.ListCanonicalRuntimeHandles(nil)
	if err != nil {
		return nil, err
	}
	for _, handle := range handlesCanonicos {
		if handle == nil {
			continue
		}
		if _, ok := handlePorAgente[handle.Agente]; ok {
			continue
		}
		handlePorAgente[handle.Agente] = handle
	}
	handlesHistoricos, err := s.store.ListRuntimeHandles(nil)
	if err != nil {
		return nil, err
	}
	for _, handle := range handlesHistoricos {
		if handle == nil {
			continue
		}
		actual := handlePorAgente[handle.Agente]
		if actual == nil {
			handlePorAgente[handle.Agente] = handle
			continue
		}
		if runtimeHandleMoment(handle).After(runtimeHandleMoment(actual)) && !runtimeHandleSostieneOperacion(actual) {
			handlePorAgente[handle.Agente] = handle
		}
	}

	ordersPorAgente := map[string][]*db.RuntimeOrder{}
	for _, order := range orders {
		if order == nil {
			continue
		}
		ordersPorAgente[order.Agente] = append(ordersPorAgente[order.Agente], order)
	}

	type mailboxCounters struct {
		Pending           int
		ActionablePending int
		ContinuityPending int
		Total             int
		LastAction        string
		LastSource        string
		LastMoment        *time.Time
	}
	mailboxPorAgente := map[string]mailboxCounters{}
	checkpointTotalPorAgente := map[string]int{}
	lastCheckpointPorAgente := map[string]*db.RuntimeCheckpoint{}
	for _, checkpoint := range checkpoints {
		if checkpoint == nil {
			continue
		}
		checkpointTotalPorAgente[checkpoint.Agente]++
		if _, ok := lastCheckpointPorAgente[checkpoint.Agente]; !ok {
			lastCheckpointPorAgente[checkpoint.Agente] = checkpoint
		}
	}

	openTasksPorAgente := map[string]int{}
	blockedTasksPorAgente := map[string]int{}
	taskByID := map[int64]*db.Tarea{}
	for _, tarea := range tareas {
		if tarea == nil || tarea.Agente == nil {
			continue
		}
		taskByID[tarea.ID] = tarea
		switch tarea.Estado {
		case db.EstadoCompletada, db.EstadoCancelada:
			continue
		case db.EstadoBloqueada:
			blockedTasksPorAgente[*tarea.Agente]++
			continue
		}
		openTasksPorAgente[*tarea.Agente]++
	}
	for _, msg := range mailbox {
		if msg == nil {
			continue
		}
		for _, agente := range []string{msg.ToAgente, msg.FromAgente} {
			agente = strings.TrimSpace(agente)
			if agente == "" {
				continue
			}
			stats := mailboxPorAgente[agente]
			stats.Total++
			if !strings.EqualFold(strings.TrimSpace(msg.Estado), "pendiente") {
				if action, source, moment := autonomyMailboxActionSummary(msg, agente); strings.TrimSpace(action) != "" {
					if mailboxCountersShouldReplace(stats.LastMoment, moment) {
						stats.LastAction = action
						stats.LastSource = source
						stats.LastMoment = moment
					}
				}
				mailboxPorAgente[agente] = stats
				continue
			}
			stats.Pending++
			if strings.EqualFold(strings.TrimSpace(msg.ToAgente), agente) {
				if runtimeMailboxCuentaComoTrabajoCanonico(msg, taskByID, agente) {
					stats.ActionablePending++
				}
				if runtimeMailboxEsContinuidadPendiente(msg, taskByID, agente) {
					stats.ContinuityPending++
				}
			}
			if action, source, moment := autonomyMailboxActionSummary(msg, agente); strings.TrimSpace(action) != "" {
				if mailboxCountersShouldReplace(stats.LastMoment, moment) {
					stats.LastAction = action
					stats.LastSource = source
					stats.LastMoment = moment
				}
			}
			mailboxPorAgente[agente] = stats
		}
	}

	now := time.Now().UTC()
	staleThreshold := workerOutputStaleThreshold(s.store)
	rows := make([]Row, 0, len(agentes))
	for _, agente := range agentes {
		if agente == nil {
			continue
		}
		mailboxStats := mailboxPorAgente[agente.Nombre]
		row := Row{
			Agente:                   agente,
			Asignacion:               asignacionPorAgente[agente.Nombre],
			Sesion:                   sesionPorAgente[agente.Nombre],
			Runtime:                  runtimePorAgente[agente.Nombre],
			Handle:                   handlePorAgente[agente.Nombre],
			MailboxPending:           mailboxStats.Pending,
			MailboxActionablePending: mailboxStats.ActionablePending,
			MailboxContinuityPending: mailboxStats.ContinuityPending,
			MailboxTotal:             mailboxStats.Total,
			LastAutonomyAction:       mailboxStats.LastAction,
			LastAutonomySource:       mailboxStats.LastSource,
			LastAutonomyMoment:       mailboxStats.LastMoment,
			Checkpoints:              checkpointTotalPorAgente[agente.Nombre],
			LastCheckpoint:           lastCheckpointPorAgente[agente.Nombre],
			OpenTasks:                openTasksPorAgente[agente.Nombre],
			BlockedTasks:             blockedTasksPorAgente[agente.Nombre],
		}
		row.OrdersOpen, row.OrdersFailed, row.ControlOrdersOpen, row.LastControlOrderType, row.LastControlOrderMoment =
			summarizeOrdersForRow(row, ordersPorAgente[agente.Nombre])
		if action, source, moment, state, reason, verificationKey, dispatchState, deliveryState, receiptSource := summarizeAutonomyOrdersForRow(ordersPorAgente[agente.Nombre]); strings.TrimSpace(action) != "" {
			if mailboxCountersShouldReplace(row.LastAutonomyMoment, moment) {
				row.LastAutonomyAction = action
				row.LastAutonomySource = source
				row.LastAutonomyMoment = moment
				row.LastAutonomyState = state
				row.LastAutonomyReason = reason
				row.LastAutonomyVerificationKey = verificationKey
				row.LastAutonomyDispatchState = dispatchState
				row.LastAutonomyDeliveryState = deliveryState
				row.LastAutonomyReceiptSource = receiptSource
			}
		}
		if structured := loadStructuredWorkerSnapshot(row.Runtime, row.Handle); structured != nil {
			if view := structured.View(now, time.Minute); view != nil {
				row.WorkerState = strings.TrimSpace(view.State)
				row.WorkerAlive = view.Alive
				row.WorkerReadyAt = view.ReadyAt
				row.WorkerHeartbeat = view.HeartbeatAt
				row.WorkerUpdatedAt = view.UpdatedAt
				row.WorkerLastOutput = view.LastOutputAt
				row.WorkerLastProgress = view.LastProgressAt
				row.WorkerExitError = strings.TrimSpace(view.ExitError)
				row.WorkerSessionRef = strings.TrimSpace(view.SessionRef)
				row.WorkerRuntimeRef = strings.TrimSpace(view.RuntimeRef)
				row.WorkerDriver = strings.TrimSpace(view.Driver)
				row.WorkerTransport = strings.TrimSpace(view.Transport)
				row.WorkerTMUXSession = strings.TrimSpace(view.TmuxSession)
				row.WorkerTMUXWindow = strings.TrimSpace(view.TmuxWindow)
				row.WorkerTMUXPaneID = strings.TrimSpace(view.TmuxPaneID)
				row.WorkerCanSendInput = view.CanSendInput
				row.WorkerExternalSessionID = strings.TrimSpace(view.ExternalSessionID)
				row.WorkerMailboxDeliveryMode = strings.TrimSpace(view.MailboxDeliveryMode)
				applyDurableWorkQueueToTickRow(&row, view)
			}
		}
		applyResumePayloadAutonomyToRow(&row)
		row.EstadoOperativo, row.DetalleOperativo = deriveOperationalState(row, now, staleThreshold)
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].Agente.Nombre) < strings.ToLower(rows[j].Agente.Nombre)
	})
	return rows, nil
}

func (s *Service) BuildDetail(nombre string) (*Detail, error) {
	return s.buildDetail(nombre, false)
}

func (s *Service) BuildDetailCompact(nombre string) (*Detail, error) {
	nombre = strings.TrimSpace(nombre)
	if nombre == "" {
		return s.buildDetail(nombre, true)
	}
	return s.getOrBuildCompactDetail(nombre, func() (*Detail, error) {
		return s.buildDetail(nombre, true)
	})
}

func (s *Service) InvalidateCompactDetailCache(names ...string) {
	if s == nil {
		return
	}
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	if len(names) == 0 {
		s.compactDetailCache = map[string]cachedCompactDetail{}
		return
	}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		delete(s.compactDetailCache, name)
	}
}

func (s *Service) buildDetail(nombre string, compact bool) (*Detail, error) {
	nombre = strings.TrimSpace(nombre)
	if compact {
		row, asignaciones, tareas, err := s.buildOperationalRowContextForAgentCompact(nombre, time.Now().UTC())
		if err != nil {
			return nil, err
		}
		return &Detail{
			Row:                     row,
			Entity:                  buildAgentEntity(s.store, row, tareas),
			Asignaciones:            asignaciones,
			MailboxTotalCount:       row.MailboxTotal,
			MailboxPendingVisible:   row.MailboxPending,
			MailboxCoveredBootstrap: 0,
		}, nil
	}
	mailbox, err := s.listMailboxForAgent(nombre)
	if err != nil {
		return nil, err
	}
	row, err := s.buildRowForAgentWithMailbox(nombre, mailbox)
	if err != nil {
		return nil, err
	}

	asignaciones, err := s.store.ListAssignments(db.FiltroAsignaciones{Agente: &nombre})
	if err != nil {
		return nil, err
	}
	tareas, err := s.store.ListTasks(db.FiltroTareas{Agente: &nombre})
	if err != nil {
		return nil, err
	}
	mailboxPendingVisible, mailboxCoveredBootstrap, err := s.summarizeMailboxOverview(mailbox, row.Handle, row.Runtime)
	if err != nil {
		return nil, err
	}

	detail := &Detail{
		Row:                     row,
		Entity:                  buildAgentEntity(s.store, row, tareas),
		Asignaciones:            asignaciones,
		MailboxTotalCount:       len(mailbox),
		MailboxPendingVisible:   mailboxPendingVisible,
		MailboxCoveredBootstrap: mailboxCoveredBootstrap,
	}

	sesiones, err := s.store.ListInspectionSessions(db.FiltroSesionesInspeccion{Agente: &nombre})
	if err != nil {
		return nil, err
	}
	runtimes, err := s.store.ListRuntimes(db.FiltroRuntimes{Agente: &nombre})
	if err != nil {
		return nil, err
	}
	handles, err := s.store.ListRuntimeHandles(&nombre)
	if err != nil {
		return nil, err
	}
	orders, err := s.store.ListRuntimeOrders(db.FiltroRuntimeOrders{Agente: &nombre})
	if err != nil {
		return nil, err
	}
	transcript, err := s.store.ListRuntimeTranscript(db.FiltroRuntimeTranscript{Agente: &nombre, Limit: 80})
	if err != nil {
		return nil, err
	}
	checkpoints, err := s.store.ListRuntimeCheckpoints(db.FiltroRuntimeCheckpoints{Agente: &nombre, Limit: 20})
	if err != nil {
		return nil, err
	}

	detail.Sesiones = sesiones
	detail.Runtimes = runtimes
	detail.Handles = handles
	detail.Transcript = transcript
	detail.Orders = orders
	detail.Mailbox = mailbox
	detail.Checkpoints = checkpoints
	return detail, nil
}

func (s *Service) buildRowForAgent(nombre string) (Row, error) {
	mailbox, err := s.listMailboxForAgent(strings.TrimSpace(nombre))
	if err != nil {
		return Row{}, err
	}
	return s.buildRowForAgentWithMailbox(nombre, mailbox)
}

func (s *Service) buildRowForAgentWithMailbox(nombre string, mailbox []*db.RuntimeMailboxMessage) (Row, error) {
	nombre = strings.TrimSpace(nombre)
	if nombre == "" {
		return Row{}, fmt.Errorf("agente obligatorio")
	}
	agente, err := s.store.GetAgent(nombre)
	if err != nil {
		return Row{}, err
	}
	row := Row{Agente: agente}

	asignaciones, err := s.store.ListAssignments(db.FiltroAsignaciones{Agente: &nombre})
	if err != nil {
		return Row{}, err
	}
	row.Asignacion = latestAssignmentForAgent(asignaciones)

	sesion, err := s.currentOrLastSessionForAgent(nombre)
	if err != nil {
		return Row{}, err
	}
	row.Sesion = sesion

	runtimes, err := s.store.ListRuntimes(db.FiltroRuntimes{Agente: &nombre})
	if err != nil {
		return Row{}, err
	}
	row.Runtime = latestRuntimeForAgent(runtimes)

	if hotHandles, err := s.store.ListRecentOperationalRuntimeHandles(); err == nil {
		if handle := hotHandles[nombre]; handle != nil {
			row.Handle = handle
		}
	}
	if row.Handle == nil {
		if handles, err := s.store.ListCanonicalRuntimeHandles(&nombre); err != nil {
			return Row{}, err
		} else {
			row.Handle = latestHandleForAgent(handles)
		}
	}
	if row.Handle == nil {
		if handles, err := s.store.ListRuntimeHandles(&nombre); err != nil {
			return Row{}, err
		} else {
			row.Handle = latestHandleForAgent(handles)
		}
	}

	orders, err := s.store.ListRuntimeOrders(db.FiltroRuntimeOrders{Agente: &nombre})
	if err != nil {
		return Row{}, err
	}
	row.OrdersOpen, row.OrdersFailed, row.ControlOrdersOpen, row.LastControlOrderType, row.LastControlOrderMoment =
		summarizeOrdersForRow(row, orders)
	if action, source, moment, state, reason, verificationKey, dispatchState, deliveryState, receiptSource := summarizeAutonomyOrdersForRow(orders); strings.TrimSpace(action) != "" {
		if mailboxCountersShouldReplace(row.LastAutonomyMoment, moment) {
			row.LastAutonomyAction = action
			row.LastAutonomySource = source
			row.LastAutonomyMoment = moment
			row.LastAutonomyState = state
			row.LastAutonomyReason = reason
			row.LastAutonomyVerificationKey = verificationKey
			row.LastAutonomyDispatchState = dispatchState
			row.LastAutonomyDeliveryState = deliveryState
			row.LastAutonomyReceiptSource = receiptSource
		}
	}

	tareas, err := s.store.ListTasks(db.FiltroTareas{Agente: &nombre})
	if err != nil {
		return Row{}, err
	}
	taskByID := map[int64]*db.Tarea{}
	for _, tarea := range tareas {
		if tarea == nil || tarea.Agente == nil {
			continue
		}
		taskByID[tarea.ID] = tarea
		switch tarea.Estado {
		case db.EstadoCompletada, db.EstadoCancelada:
			continue
		case db.EstadoBloqueada:
			row.BlockedTasks++
		default:
			row.OpenTasks++
		}
	}

	applyMailboxStatsToRow(&row, mailbox, taskByID, nombre)

	checkpoints, err := s.store.ListRuntimeCheckpoints(db.FiltroRuntimeCheckpoints{Agente: &nombre, Limit: 5})
	if err != nil {
		return Row{}, err
	}
	row.Checkpoints = len(checkpoints)
	if len(checkpoints) > 0 {
		row.LastCheckpoint = checkpoints[0]
	}

	now := time.Now().UTC()
	if structured := loadStructuredWorkerSnapshot(row.Runtime, row.Handle); structured != nil {
		if view := structured.View(now, time.Minute); view != nil {
			row.WorkerState = strings.TrimSpace(view.State)
			row.WorkerAlive = view.Alive
			row.WorkerReadyAt = view.ReadyAt
			row.WorkerHeartbeat = view.HeartbeatAt
			row.WorkerUpdatedAt = view.UpdatedAt
			row.WorkerLastOutput = view.LastOutputAt
			row.WorkerLastProgress = view.LastProgressAt
			row.WorkerExitError = strings.TrimSpace(view.ExitError)
			row.WorkerSessionRef = strings.TrimSpace(view.SessionRef)
			row.WorkerRuntimeRef = strings.TrimSpace(view.RuntimeRef)
			row.WorkerDriver = strings.TrimSpace(view.Driver)
			row.WorkerTransport = strings.TrimSpace(view.Transport)
			row.WorkerTMUXSession = strings.TrimSpace(view.TmuxSession)
			row.WorkerTMUXWindow = strings.TrimSpace(view.TmuxWindow)
			row.WorkerTMUXPaneID = strings.TrimSpace(view.TmuxPaneID)
			row.WorkerCanSendInput = view.CanSendInput
			row.WorkerExternalSessionID = strings.TrimSpace(view.ExternalSessionID)
			row.WorkerMailboxDeliveryMode = strings.TrimSpace(view.MailboxDeliveryMode)
			applyDurableWorkQueueToTickRow(&row, view)
		}
	}
	applyResumePayloadAutonomyToRow(&row)
	row.EstadoOperativo, row.DetalleOperativo = deriveOperationalState(row, now, workerOutputStaleThreshold(s.store))
	return row, nil
}

func (s *Service) buildOperationalRowForAgent(nombre string, now time.Time) (Row, error) {
	row, _, _, err := s.buildOperationalRowContextForAgent(nombre, now)
	return row, err
}

func (s *Service) buildOperationalRowContextForAgentCompact(nombre string, now time.Time) (Row, []*db.Asignacion, []*db.Tarea, error) {
	return s.buildOperationalRowContextForAgentCompactWithSession(nombre, now, nil)
}

func (s *Service) buildOperationalRowContextForAgentCompactWithSession(nombre string, now time.Time, sessionHint *db.Sesion) (Row, []*db.Asignacion, []*db.Tarea, error) {
	nombre = strings.TrimSpace(nombre)
	if nombre == "" {
		return Row{}, nil, nil, fmt.Errorf("agente obligatorio")
	}
	start := time.Now()
	agentDetailDebugf("compact start agente=%s", nombre)

	stepStart := time.Now()
	agente, err := s.store.GetAgent(nombre)
	if err != nil {
		return Row{}, nil, nil, err
	}
	if agente == nil {
		return Row{}, nil, nil, fmt.Errorf("agente no encontrado: %s", nombre)
	}
	s.cachePrepareAgent(agente)
	agentDetailDebugf("compact step=get_agent agente=%s duration=%s", nombre, time.Since(stepStart).Round(time.Millisecond))

	row := Row{Agente: agente}

	stepStart = time.Now()
	asignaciones, err := s.store.ListAssignments(db.FiltroAsignaciones{Agente: &nombre})
	if err != nil {
		return Row{}, nil, nil, err
	}
	for _, asignacion := range asignaciones {
		if asignacion != nil && asignacion.Estado == db.AsignacionActiva {
			row.Asignacion = asignacion
			break
		}
	}
	agentDetailDebugf("compact step=list_assignments agente=%s count=%d duration=%s", nombre, len(asignaciones), time.Since(stepStart).Round(time.Millisecond))

	stepStart = time.Now()
	if sessionHint != nil {
		row.Sesion = sessionHint
	} else {
		sesion, err := s.store.GetActiveSession(nombre, nil)
		switch {
		case err == nil:
			row.Sesion = sesion
		case err != sql.ErrNoRows:
			return Row{}, nil, nil, err
		}
	}
	agentDetailDebugf("compact step=get_active_session agente=%s duration=%s", nombre, time.Since(stepStart).Round(time.Millisecond))

	stepStart = time.Now()
	runtimes, err := s.store.ListRuntimes(db.FiltroRuntimes{Agente: &nombre})
	if err != nil {
		return Row{}, nil, nil, err
	}
	row.Runtime = latestRuntimeForAgent(runtimes)
	agentDetailDebugf("compact step=list_runtimes agente=%s count=%d duration=%s", nombre, len(runtimes), time.Since(stepStart).Round(time.Millisecond))

	stepStart = time.Now()
	handles, err := s.store.ListCanonicalRuntimeHandles(&nombre)
	if err != nil {
		return Row{}, nil, nil, err
	}
	row.Handle = latestHandleForAgent(handles)
	if row.Handle == nil {
		handles, err = s.store.ListPassiveRuntimeHandles(&nombre)
		if err != nil {
			return Row{}, nil, nil, err
		}
		row.Handle = latestHandleForAgent(handles)
	}
	agentDetailDebugf("compact step=list_handles agente=%s count=%d duration=%s", nombre, len(handles), time.Since(stepStart).Round(time.Millisecond))
	if structured := loadStructuredWorkerSnapshot(row.Runtime, row.Handle); structured != nil {
		if view := structured.View(now, time.Minute); view != nil {
			row.WorkerState = strings.TrimSpace(view.State)
			row.WorkerAlive = view.Alive
			row.WorkerReadyAt = view.ReadyAt
			row.WorkerHeartbeat = view.HeartbeatAt
			row.WorkerUpdatedAt = view.UpdatedAt
			row.WorkerLastOutput = view.LastOutputAt
			row.WorkerLastProgress = view.LastProgressAt
			row.WorkerExitError = strings.TrimSpace(view.ExitError)
			row.WorkerSessionRef = strings.TrimSpace(view.SessionRef)
			row.WorkerRuntimeRef = strings.TrimSpace(view.RuntimeRef)
			row.WorkerDriver = strings.TrimSpace(view.Driver)
			row.WorkerTransport = strings.TrimSpace(view.Transport)
			row.WorkerTMUXSession = strings.TrimSpace(view.TmuxSession)
			row.WorkerTMUXWindow = strings.TrimSpace(view.TmuxWindow)
			row.WorkerTMUXPaneID = strings.TrimSpace(view.TmuxPaneID)
			row.WorkerCanSendInput = view.CanSendInput
			row.WorkerExternalSessionID = strings.TrimSpace(view.ExternalSessionID)
			row.WorkerMailboxDeliveryMode = strings.TrimSpace(view.MailboxDeliveryMode)
			applyDurableWorkQueueToTickRow(&row, view)
			if row.Asignacion != nil && row.Asignacion.ProyectoID > 0 {
				if tarea, ok, err := s.durableWorkQueueTaskForAgent(nombre, row.Asignacion.ProyectoID, view); err != nil {
					return Row{}, nil, nil, err
				} else if ok {
					tareas := []*db.Tarea{tarea}
					switch tarea.Estado {
					case db.EstadoBloqueada:
						row.BlockedTasks++
					case db.EstadoCompletada, db.EstadoCancelada, db.EstadoBacklog:
					default:
						row.OpenTasks++
					}
					stepStart = time.Now()
					mailbox, err := s.store.ListRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &nombre})
					if err != nil {
						return Row{}, nil, nil, err
					}
					taskByID := map[int64]*db.Tarea{tarea.ID: tarea}
					applyMailboxStatsToRow(&row, mailbox, taskByID, nombre)
					if err := s.applyAutonomyOrdersToRow(&row, nombre); err != nil {
						return Row{}, nil, nil, err
					}
					agentDetailDebugf("compact step=list_mailbox_to agente=%s count=%d duration=%s", nombre, len(mailbox), time.Since(stepStart).Round(time.Millisecond))
					applyResumePayloadAutonomyToRow(&row)
					row.EstadoOperativo, row.DetalleOperativo = deriveOperationalState(row, now, workerOutputStaleThreshold(s.store))
					agentDetailDebugf("compact done agente=%s duration=%s", nombre, time.Since(start).Round(time.Millisecond))
					return row, asignaciones, tareas, nil
				}
			}
		}
	}

	stepStart = time.Now()
	tareas, err := s.store.ListTasks(db.FiltroTareas{Agente: &nombre})
	if err != nil {
		return Row{}, nil, nil, err
	}
	for _, tarea := range tareas {
		if tarea == nil {
			continue
		}
		switch tarea.Estado {
		case db.EstadoCompletada, db.EstadoCancelada, db.EstadoBacklog:
			continue
		case db.EstadoBloqueada:
			row.BlockedTasks++
		default:
			row.OpenTasks++
		}
	}
	agentDetailDebugf("compact step=list_tasks agente=%s count=%d duration=%s", nombre, len(tareas), time.Since(stepStart).Round(time.Millisecond))

	stepStart = time.Now()
	mailbox, err := s.store.ListRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &nombre})
	if err != nil {
		return Row{}, nil, nil, err
	}
	taskByID := map[int64]*db.Tarea{}
	for _, tarea := range tareas {
		if tarea == nil {
			continue
		}
		taskByID[tarea.ID] = tarea
	}
	applyMailboxStatsToRow(&row, mailbox, taskByID, nombre)
	if err := s.applyAutonomyOrdersToRow(&row, nombre); err != nil {
		return Row{}, nil, nil, err
	}
	agentDetailDebugf("compact step=list_mailbox_to agente=%s count=%d duration=%s", nombre, len(mailbox), time.Since(stepStart).Round(time.Millisecond))

	applyResumePayloadAutonomyToRow(&row)
	row.EstadoOperativo, row.DetalleOperativo = deriveOperationalState(row, now, workerOutputStaleThreshold(s.store))
	agentDetailDebugf("compact done agente=%s duration=%s", nombre, time.Since(start).Round(time.Millisecond))
	return row, asignaciones, tareas, nil
}

func (s *Service) applyAutonomyOrdersToRow(row *Row, agente string) error {
	if s == nil || s.store == nil || row == nil {
		return nil
	}
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return nil
	}
	orders, err := s.store.ListRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente})
	if err != nil {
		return err
	}
	if action, source, moment, state, reason, verificationKey, dispatchState, deliveryState, receiptSource := summarizeAutonomyOrdersForRow(orders); strings.TrimSpace(action) != "" {
		if mailboxCountersShouldReplace(row.LastAutonomyMoment, moment) {
			row.LastAutonomyAction = action
			row.LastAutonomySource = source
			row.LastAutonomyMoment = moment
			row.LastAutonomyState = state
			row.LastAutonomyReason = reason
			row.LastAutonomyVerificationKey = verificationKey
			row.LastAutonomyDispatchState = dispatchState
			row.LastAutonomyDeliveryState = deliveryState
			row.LastAutonomyReceiptSource = receiptSource
		}
	}
	return nil
}

func applyMailboxStatsToRow(row *Row, mailbox []*db.RuntimeMailboxMessage, taskByID map[int64]*db.Tarea, nombre string) {
	if row == nil {
		return
	}
	nombre = strings.TrimSpace(nombre)
	for _, msg := range mailbox {
		if msg == nil {
			continue
		}
		row.MailboxTotal++
		if !strings.EqualFold(strings.TrimSpace(msg.Estado), "pendiente") {
			continue
		}
		row.MailboxPending++
		if !strings.EqualFold(strings.TrimSpace(msg.ToAgente), nombre) {
			continue
		}
		if runtimeMailboxCuentaComoTrabajoCanonico(msg, taskByID, nombre) {
			row.MailboxActionablePending++
		}
		if runtimeMailboxEsContinuidadPendiente(msg, taskByID, nombre) {
			row.MailboxContinuityPending++
		}
		if action, source, moment := autonomyMailboxActionSummary(msg, nombre); strings.TrimSpace(action) != "" {
			if mailboxCountersShouldReplace(row.LastAutonomyMoment, moment) {
				row.LastAutonomyAction = action
				row.LastAutonomySource = source
				row.LastAutonomyMoment = moment
			}
		}
	}
}

func (s *Service) buildOperationalRowContextForAgent(nombre string, now time.Time) (Row, []*db.Asignacion, []*db.Tarea, error) {
	nombre = strings.TrimSpace(nombre)
	if nombre == "" {
		return Row{}, nil, nil, fmt.Errorf("agente obligatorio")
	}

	agente, err := s.store.GetAgent(nombre)
	if err != nil {
		return Row{}, nil, nil, err
	}
	if agente == nil {
		return Row{}, nil, nil, fmt.Errorf("agente no encontrado: %s", nombre)
	}
	s.cachePrepareAgent(agente)

	row := Row{Agente: agente}

	asignaciones, err := s.store.ListAssignments(db.FiltroAsignaciones{Agente: &nombre})
	if err != nil {
		return Row{}, nil, nil, err
	}
	for _, asignacion := range asignaciones {
		if asignacion != nil && asignacion.Estado == db.AsignacionActiva {
			row.Asignacion = asignacion
			break
		}
	}

	sesion, err := s.currentOrLastSessionForAgent(nombre)
	if err != nil {
		return Row{}, nil, nil, err
	}
	row.Sesion = sesion

	runtimes, err := s.store.ListRuntimes(db.FiltroRuntimes{Agente: &nombre})
	if err != nil {
		return Row{}, nil, nil, err
	}
	row.Runtime = latestRuntimeForAgent(runtimes)

	handles, err := s.store.ListCanonicalRuntimeHandles(&nombre)
	if err != nil {
		return Row{}, nil, nil, err
	}
	row.Handle = latestHandleForAgent(handles)
	if row.Handle == nil {
		handles, err = s.store.ListPassiveRuntimeHandles(&nombre)
		if err != nil {
			return Row{}, nil, nil, err
		}
		row.Handle = latestHandleForAgent(handles)
	}
	if structured := loadStructuredWorkerSnapshot(row.Runtime, row.Handle); structured != nil {
		if view := structured.View(now, time.Minute); view != nil {
			row.WorkerState = strings.TrimSpace(view.State)
			row.WorkerAlive = view.Alive
			row.WorkerReadyAt = view.ReadyAt
			row.WorkerHeartbeat = view.HeartbeatAt
			row.WorkerUpdatedAt = view.UpdatedAt
			row.WorkerLastOutput = view.LastOutputAt
			row.WorkerLastProgress = view.LastProgressAt
			row.WorkerExitError = strings.TrimSpace(view.ExitError)
			row.WorkerSessionRef = strings.TrimSpace(view.SessionRef)
			row.WorkerRuntimeRef = strings.TrimSpace(view.RuntimeRef)
			row.WorkerDriver = strings.TrimSpace(view.Driver)
			row.WorkerTransport = strings.TrimSpace(view.Transport)
			row.WorkerTMUXSession = strings.TrimSpace(view.TmuxSession)
			row.WorkerTMUXWindow = strings.TrimSpace(view.TmuxWindow)
			row.WorkerTMUXPaneID = strings.TrimSpace(view.TmuxPaneID)
			row.WorkerCanSendInput = view.CanSendInput
			row.WorkerExternalSessionID = strings.TrimSpace(view.ExternalSessionID)
			row.WorkerMailboxDeliveryMode = strings.TrimSpace(view.MailboxDeliveryMode)
		}
	}

	tareas, err := s.store.ListTasks(db.FiltroTareas{Agente: &nombre})
	if err != nil {
		return Row{}, nil, nil, err
	}
	for _, tarea := range tareas {
		if tarea == nil {
			continue
		}
		switch tarea.Estado {
		case db.EstadoCompletada, db.EstadoCancelada, db.EstadoBacklog:
			continue
		case db.EstadoBloqueada:
			row.BlockedTasks++
		default:
			row.OpenTasks++
		}
	}

	row.EstadoOperativo, row.DetalleOperativo = deriveOperationalState(row, now, workerOutputStaleThreshold(s.store))
	return row, asignaciones, tareas, nil
}

func (s *Service) cachePrepareAgent(agent *db.Agente) {
	if s == nil || agent == nil || strings.TrimSpace(agent.Nombre) == "" {
		return
	}
	clone := *agent
	key := strings.ToLower(strings.TrimSpace(agent.Nombre))
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.prepareAgentCache[key] = cachedPrepareAgent{
		agent:   &clone,
		expires: time.Now().Add(prepareEntityCacheTTL),
	}
}

func (s *Service) cachedPrepareAgent(nombre string) *db.Agente {
	if s == nil {
		return nil
	}
	key := strings.ToLower(strings.TrimSpace(nombre))
	if key == "" {
		return nil
	}
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	item, ok := s.prepareAgentCache[key]
	if !ok || item.agent == nil || time.Now().After(item.expires) {
		if ok {
			delete(s.prepareAgentCache, key)
		}
		return nil
	}
	clone := *item.agent
	return &clone
}

func (s *Service) cachePrepareProject(project *db.Proyecto, refs ...string) {
	if s == nil || project == nil {
		return
	}
	clone := *project
	keys := map[string]struct{}{}
	for _, ref := range refs {
		ref = strings.ToLower(strings.TrimSpace(ref))
		if ref != "" {
			keys[ref] = struct{}{}
		}
	}
	if project.ID > 0 {
		keys[strconv.FormatInt(project.ID, 10)] = struct{}{}
	}
	if slug := strings.ToLower(strings.TrimSpace(project.Slug)); slug != "" {
		keys[slug] = struct{}{}
	}
	if path := strings.ToLower(strings.TrimSpace(project.RutaAbs)); path != "" {
		keys[path] = struct{}{}
	}
	if len(keys) == 0 {
		return
	}
	expires := time.Now().Add(prepareEntityCacheTTL)
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	for key := range keys {
		s.prepareProjectCache[key] = cachedPrepareProject{
			project: &clone,
			expires: expires,
		}
	}
}

func (s *Service) cachedPrepareProject(ref string) *db.Proyecto {
	if s == nil {
		return nil
	}
	key := strings.ToLower(strings.TrimSpace(ref))
	if key == "" {
		return nil
	}
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	item, ok := s.prepareProjectCache[key]
	if !ok || item.project == nil || time.Now().After(item.expires) {
		if ok {
			delete(s.prepareProjectCache, key)
		}
		return nil
	}
	clone := *item.project
	return &clone
}

func (s *Service) cachedPrepareGovernance(rol string, proyectoID int64, agente string) *db.GovernanceCatalog {
	if s == nil {
		return nil
	}
	key := prepareGovernanceCacheKey(rol, proyectoID, agente)
	if key == "" {
		return nil
	}
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	item, ok := s.prepareGovCache[key]
	if !ok || item.catalog == nil || time.Now().After(item.expires) {
		if ok {
			delete(s.prepareGovCache, key)
		}
		return nil
	}
	return clonePrepareGovernanceCatalog(item.catalog)
}

func (s *Service) cachedPrepareSession(agente string, proyectoID int64) (*db.Sesion, bool, bool) {
	if s == nil {
		return nil, false, false
	}
	key := prepareSessionCacheKey(agente, proyectoID)
	if key == "" {
		return nil, false, false
	}
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	item, ok := s.prepareSessionCache[key]
	if !ok || time.Now().After(item.expires) {
		if ok {
			delete(s.prepareSessionCache, key)
		}
		return nil, false, false
	}
	return clonePrepareSession(item.session), item.found, true
}

func (s *Service) cachePrepareSession(agente string, proyectoID int64, session *db.Sesion, found bool) {
	if s == nil {
		return
	}
	key := prepareSessionCacheKey(agente, proyectoID)
	if key == "" {
		return
	}
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.prepareSessionCache[key] = cachedPrepareSession{
		session: clonePrepareSession(session),
		found:   found,
		expires: time.Now().Add(prepareSessionCacheTTL),
	}
}

func (s *Service) cachedPrepareConnector(ref string) *db.Conector {
	if s == nil {
		return nil
	}
	key := strings.ToLower(strings.TrimSpace(ref))
	if key == "" {
		return nil
	}
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	item, ok := s.prepareConnectorCache[key]
	if !ok || item.connector == nil || time.Now().After(item.expires) {
		if ok {
			delete(s.prepareConnectorCache, key)
		}
		return nil
	}
	return clonePrepareConnector(item.connector)
}

func (s *Service) cachedPrepareModelPolicy(key string) *db.ResolucionModelo {
	if s == nil {
		return nil
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return nil
	}
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	item, ok := s.prepareModelPolicyCache[key]
	if !ok || item.resolution == nil || time.Now().After(item.expires) {
		if ok {
			delete(s.prepareModelPolicyCache, key)
		}
		return nil
	}
	return clonePrepareModelResolution(item.resolution)
}

func (s *Service) cachePrepareModelPolicy(key string, resolution *db.ResolucionModelo) {
	if s == nil || strings.TrimSpace(key) == "" || resolution == nil {
		return
	}
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.prepareModelPolicyCache[strings.TrimSpace(key)] = cachedPrepareModelPolicy{
		resolution: clonePrepareModelResolution(resolution),
		expires:    time.Now().Add(prepareModelPolicyCacheTTL),
	}
}

func (s *Service) cachePrepareConnector(connector *db.Conector, refs ...string) {
	if s == nil || connector == nil {
		return
	}
	keys := map[string]struct{}{}
	for _, ref := range refs {
		ref = strings.ToLower(strings.TrimSpace(ref))
		if ref != "" {
			keys[ref] = struct{}{}
		}
	}
	if connector.ID > 0 {
		keys[strconv.FormatInt(connector.ID, 10)] = struct{}{}
	}
	if slug := strings.ToLower(strings.TrimSpace(connector.Slug)); slug != "" {
		keys[slug] = struct{}{}
	}
	if len(keys) == 0 {
		return
	}
	expires := time.Now().Add(prepareEntityCacheTTL)
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	for key := range keys {
		s.prepareConnectorCache[key] = cachedPrepareConnector{
			connector: clonePrepareConnector(connector),
			expires:   expires,
		}
	}
}

func (s *Service) cachePrepareGovernance(rol string, proyectoID int64, agente string, catalog *db.GovernanceCatalog) {
	if s == nil || catalog == nil {
		return
	}
	key := prepareGovernanceCacheKey(rol, proyectoID, agente)
	if key == "" {
		return
	}
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.prepareGovCache[key] = cachedPrepareGovernance{
		catalog: clonePrepareGovernanceCatalog(catalog),
		expires: time.Now().Add(prepareGovernanceCacheTTL),
	}
}

func (s *Service) beginPrepareAgentFlight(nombre string) *prepareAgentFlight {
	if s == nil {
		return nil
	}
	key := strings.ToLower(strings.TrimSpace(nombre))
	if key == "" {
		return nil
	}
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	if item, ok := s.prepareAgentCache[key]; ok && item.agent != nil && time.Now().Before(item.expires) {
		return nil
	}
	if _, ok := s.prepareAgentFlight[key]; ok {
		return nil
	}
	flight := &prepareAgentFlight{done: make(chan struct{})}
	s.prepareAgentFlight[key] = flight
	return flight
}

func (s *Service) waitPrepareAgentFlight(nombre string) (*db.Agente, error) {
	if s == nil {
		return nil, nil
	}
	key := strings.ToLower(strings.TrimSpace(nombre))
	if key == "" {
		return nil, nil
	}
	for {
		s.cacheMu.Lock()
		flight, ok := s.prepareAgentFlight[key]
		s.cacheMu.Unlock()
		if !ok {
			return s.cachedPrepareAgent(nombre), nil
		}
		<-flight.done
		if flight.err != nil {
			return nil, flight.err
		}
		if flight.agent != nil {
			return clonePrepareAgent(flight.agent), nil
		}
		return s.cachedPrepareAgent(nombre), nil
	}
}

func (s *Service) finishPrepareAgentFlight(nombre string, flight *prepareAgentFlight) {
	if s == nil || flight == nil {
		return
	}
	key := strings.ToLower(strings.TrimSpace(nombre))
	if key == "" {
		return
	}
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	delete(s.prepareAgentFlight, key)
	close(flight.done)
}

func (s *Service) beginPrepareProjectFlight(ref string) *prepareProjectFlight {
	if s == nil {
		return nil
	}
	key := strings.ToLower(strings.TrimSpace(ref))
	if key == "" {
		return nil
	}
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	if item, ok := s.prepareProjectCache[key]; ok && item.project != nil && time.Now().Before(item.expires) {
		return nil
	}
	if _, ok := s.prepareProjectFlight[key]; ok {
		return nil
	}
	flight := &prepareProjectFlight{done: make(chan struct{})}
	s.prepareProjectFlight[key] = flight
	return flight
}

func (s *Service) beginTickProjectFlight(ref string) *prepareProjectFlight {
	if s == nil {
		return nil
	}
	key := strings.ToLower(strings.TrimSpace(ref))
	if key == "" {
		return nil
	}
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	if item, ok := s.prepareProjectCache[key]; ok && item.project != nil && time.Now().Before(item.expires) {
		return nil
	}
	if _, ok := s.tickProjectFlight[key]; ok {
		return nil
	}
	flight := &prepareProjectFlight{done: make(chan struct{})}
	s.tickProjectFlight[key] = flight
	return flight
}

func (s *Service) waitPrepareProjectFlight(ref string) (*db.Proyecto, error) {
	if s == nil {
		return nil, nil
	}
	key := strings.ToLower(strings.TrimSpace(ref))
	if key == "" {
		return nil, nil
	}
	for {
		s.cacheMu.Lock()
		flight, ok := s.prepareProjectFlight[key]
		s.cacheMu.Unlock()
		if !ok {
			return s.cachedPrepareProject(ref), nil
		}
		<-flight.done
		if flight.err != nil {
			return nil, flight.err
		}
		if flight.project != nil {
			return clonePrepareProject(flight.project), nil
		}
		return s.cachedPrepareProject(ref), nil
	}
}

func (s *Service) waitTickProjectFlight(ref string) (*db.Proyecto, error) {
	if s == nil {
		return nil, nil
	}
	key := strings.ToLower(strings.TrimSpace(ref))
	if key == "" {
		return nil, nil
	}
	for {
		s.cacheMu.Lock()
		flight, ok := s.tickProjectFlight[key]
		s.cacheMu.Unlock()
		if !ok {
			return s.cachedPrepareProject(ref), nil
		}
		<-flight.done
		if flight.err != nil {
			return nil, flight.err
		}
		if flight.project != nil {
			return clonePrepareProject(flight.project), nil
		}
		return s.cachedPrepareProject(ref), nil
	}
}

func (s *Service) finishPrepareProjectFlight(ref string, flight *prepareProjectFlight) {
	if s == nil || flight == nil {
		return
	}
	key := strings.ToLower(strings.TrimSpace(ref))
	if key == "" {
		return
	}
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	delete(s.prepareProjectFlight, key)
	close(flight.done)
}

func (s *Service) finishTickProjectFlight(ref string, flight *prepareProjectFlight) {
	if s == nil || flight == nil {
		return
	}
	key := strings.ToLower(strings.TrimSpace(ref))
	if key == "" {
		return
	}
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	delete(s.tickProjectFlight, key)
	close(flight.done)
}

func (s *Service) beginPrepareGovernanceFlight(rol string, proyectoID int64, agente string) *prepareGovernanceFlight {
	if s == nil {
		return nil
	}
	key := prepareGovernanceCacheKey(rol, proyectoID, agente)
	if key == "" {
		return nil
	}
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	if item, ok := s.prepareGovCache[key]; ok && item.catalog != nil && time.Now().Before(item.expires) {
		return nil
	}
	if _, ok := s.prepareGovFlight[key]; ok {
		return nil
	}
	flight := &prepareGovernanceFlight{done: make(chan struct{})}
	s.prepareGovFlight[key] = flight
	return flight
}

func (s *Service) waitPrepareGovernanceFlight(rol string, proyectoID int64, agente string) (*db.GovernanceCatalog, error) {
	if s == nil {
		return nil, nil
	}
	key := prepareGovernanceCacheKey(rol, proyectoID, agente)
	if key == "" {
		return nil, nil
	}
	for {
		s.cacheMu.Lock()
		flight, ok := s.prepareGovFlight[key]
		s.cacheMu.Unlock()
		if !ok {
			return s.cachedPrepareGovernance(rol, proyectoID, agente), nil
		}
		<-flight.done
		if flight.err != nil {
			return nil, flight.err
		}
		if flight.catalog != nil {
			return clonePrepareGovernanceCatalog(flight.catalog), nil
		}
		return s.cachedPrepareGovernance(rol, proyectoID, agente), nil
	}
}

func (s *Service) finishPrepareGovernanceFlight(rol string, proyectoID int64, agente string, flight *prepareGovernanceFlight) {
	if s == nil || flight == nil {
		return
	}
	key := prepareGovernanceCacheKey(rol, proyectoID, agente)
	if key == "" {
		return
	}
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	delete(s.prepareGovFlight, key)
	close(flight.done)
}

func (s *Service) beginPrepareSessionFlight(agente string, proyectoID int64) *prepareSessionFlight {
	if s == nil {
		return nil
	}
	key := prepareSessionCacheKey(agente, proyectoID)
	if key == "" {
		return nil
	}
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	if item, ok := s.prepareSessionCache[key]; ok && time.Now().Before(item.expires) {
		return nil
	}
	if _, ok := s.prepareSessionFlight[key]; ok {
		return nil
	}
	flight := &prepareSessionFlight{done: make(chan struct{})}
	s.prepareSessionFlight[key] = flight
	return flight
}

func (s *Service) waitPrepareSessionFlight(agente string, proyectoID int64) (*db.Sesion, bool, error) {
	if s == nil {
		return nil, false, nil
	}
	key := prepareSessionCacheKey(agente, proyectoID)
	if key == "" {
		return nil, false, nil
	}
	for {
		s.cacheMu.Lock()
		flight, ok := s.prepareSessionFlight[key]
		s.cacheMu.Unlock()
		if !ok {
			session, found, cached := s.cachedPrepareSession(agente, proyectoID)
			if !cached {
				return nil, false, nil
			}
			return session, found, nil
		}
		<-flight.done
		if flight.err != nil {
			return nil, false, flight.err
		}
		return clonePrepareSession(flight.session), flight.found, nil
	}
}

func (s *Service) finishPrepareSessionFlight(agente string, proyectoID int64, flight *prepareSessionFlight) {
	if s == nil || flight == nil {
		return
	}
	key := prepareSessionCacheKey(agente, proyectoID)
	if key == "" {
		return
	}
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	delete(s.prepareSessionFlight, key)
	close(flight.done)
}

func (s *Service) beginPrepareConnectorFlight(ref string) *prepareConnectorFlight {
	if s == nil {
		return nil
	}
	key := strings.ToLower(strings.TrimSpace(ref))
	if key == "" {
		return nil
	}
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	if item, ok := s.prepareConnectorCache[key]; ok && item.connector != nil && time.Now().Before(item.expires) {
		return nil
	}
	if _, ok := s.prepareConnectorFlight[key]; ok {
		return nil
	}
	flight := &prepareConnectorFlight{done: make(chan struct{})}
	s.prepareConnectorFlight[key] = flight
	return flight
}

func (s *Service) waitPrepareConnectorFlight(ref string) (*db.Conector, error) {
	if s == nil {
		return nil, nil
	}
	key := strings.ToLower(strings.TrimSpace(ref))
	if key == "" {
		return nil, nil
	}
	for {
		s.cacheMu.Lock()
		flight, ok := s.prepareConnectorFlight[key]
		s.cacheMu.Unlock()
		if !ok {
			return s.cachedPrepareConnector(ref), nil
		}
		<-flight.done
		if flight.err != nil {
			return nil, flight.err
		}
		return clonePrepareConnector(flight.connector), nil
	}
}

func (s *Service) finishPrepareConnectorFlight(ref string, flight *prepareConnectorFlight) {
	if s == nil || flight == nil {
		return
	}
	key := strings.ToLower(strings.TrimSpace(ref))
	if key == "" {
		return
	}
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	delete(s.prepareConnectorFlight, key)
	close(flight.done)
}

func (s *Service) beginPrepareModelPolicyFlight(key string) *prepareModelPolicyFlight {
	if s == nil {
		return nil
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return nil
	}
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	if item, ok := s.prepareModelPolicyCache[key]; ok && item.resolution != nil && time.Now().Before(item.expires) {
		return nil
	}
	if _, ok := s.prepareModelPolicyFlight[key]; ok {
		return nil
	}
	flight := &prepareModelPolicyFlight{done: make(chan struct{})}
	s.prepareModelPolicyFlight[key] = flight
	return flight
}

func (s *Service) waitPrepareModelPolicyFlight(key string) (*db.ResolucionModelo, error) {
	if s == nil {
		return nil, nil
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, nil
	}
	for {
		s.cacheMu.Lock()
		flight, ok := s.prepareModelPolicyFlight[key]
		s.cacheMu.Unlock()
		if !ok {
			return s.cachedPrepareModelPolicy(key), nil
		}
		<-flight.done
		if flight.err != nil {
			return nil, flight.err
		}
		return clonePrepareModelResolution(flight.resolution), nil
	}
}

func (s *Service) finishPrepareModelPolicyFlight(key string, flight *prepareModelPolicyFlight) {
	if s == nil || flight == nil {
		return
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	delete(s.prepareModelPolicyFlight, key)
	close(flight.done)
}

func prepareGovernanceCacheKey(rol string, proyectoID int64, agente string) string {
	rol = strings.ToLower(strings.TrimSpace(rol))
	agente = strings.ToLower(strings.TrimSpace(agente))
	if rol == "" || proyectoID <= 0 {
		return ""
	}
	return rol + "|" + strconv.FormatInt(proyectoID, 10) + "|" + agente
}

func prepareSessionCacheKey(agente string, proyectoID int64) string {
	agente = strings.ToLower(strings.TrimSpace(agente))
	if agente == "" || proyectoID <= 0 {
		return ""
	}
	return agente + "|" + strconv.FormatInt(proyectoID, 10)
}

func latestAssignmentForAgent(items []*db.Asignacion) *db.Asignacion {
	var best *db.Asignacion
	for _, item := range items {
		if item == nil {
			continue
		}
		if best == nil {
			best = item
			continue
		}
		bestActive := best.Estado == db.AsignacionActiva
		itemActive := item.Estado == db.AsignacionActiva
		if itemActive != bestActive {
			if itemActive {
				best = item
			}
			continue
		}
		if item.UpdatedAt.After(best.UpdatedAt) || (item.UpdatedAt.Equal(best.UpdatedAt) && item.ID > best.ID) {
			best = item
		}
	}
	return best
}

func latestSessionForAgent(items []*db.Sesion) *db.Sesion {
	var best *db.Sesion
	for _, item := range items {
		if item == nil {
			continue
		}
		if best == nil || latestSessionCandidate(item, best) {
			best = item
		}
	}
	return best
}

func latestSessionCandidate(candidate, current *db.Sesion) bool {
	if current == nil {
		return true
	}
	if candidate == nil {
		return false
	}
	if !candidate.Inicio.Equal(current.Inicio) {
		return candidate.Inicio.After(current.Inicio)
	}
	return candidate.ID > current.ID
}

func latestRuntimeForAgent(items []*db.RuntimeInstance) *db.RuntimeInstance {
	var best *db.RuntimeInstance
	for _, item := range items {
		if item == nil {
			continue
		}
		if best == nil || runtimeMoment(item).After(runtimeMoment(best)) {
			best = item
		}
	}
	return best
}

func latestHandleForAgent(items []*db.RuntimeHandle) *db.RuntimeHandle {
	var best *db.RuntimeHandle
	for _, item := range items {
		if item == nil {
			continue
		}
		if best == nil || runtimeHandleMoment(item).After(runtimeHandleMoment(best)) {
			best = item
		}
	}
	return best
}

func latestHandleForProject(items []*db.RuntimeHandle, proyectoID *int64) *db.RuntimeHandle {
	if proyectoID == nil || *proyectoID <= 0 {
		return latestHandleForAgent(items)
	}
	var best *db.RuntimeHandle
	for _, item := range items {
		if item == nil || item.ProyectoID == nil || *item.ProyectoID != *proyectoID {
			continue
		}
		if best == nil || runtimeHandleMoment(item).After(runtimeHandleMoment(best)) {
			best = item
		}
	}
	return best
}

func (s *Service) summarizeMailboxOverview(items []*db.RuntimeMailboxMessage, handle *db.RuntimeHandle, runtime *db.RuntimeInstance) (pendingVisible int, coveredBootstrap int, err error) {
	for _, item := range items {
		if item == nil {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(item.Estado), "pendiente") {
			continue
		}
		covered, _, _, coverErr := s.store.RuntimeMailboxCoveredByBootstrapPending(item.ID, handle, runtime)
		if coverErr != nil {
			return 0, 0, coverErr
		}
		if covered {
			coveredBootstrap++
			continue
		}
		pendingVisible++
	}
	return pendingVisible, coveredBootstrap, nil
}

func (s *Service) BuildReanimationSchedule(includeFuture, activeOnly bool) ([]ReanimationCandidate, error) {
	if activeOnly {
		key := fmt.Sprintf("active:%t:future:%t", activeOnly, includeFuture)
		return s.getOrBuildActiveReanimationSchedule(key, func() ([]ReanimationCandidate, error) {
			return s.buildReanimationSchedule(includeFuture, activeOnly)
		})
	}
	return s.buildReanimationSchedule(includeFuture, activeOnly)
}

func (s *Service) buildReanimationSchedule(includeFuture, activeOnly bool) ([]ReanimationCandidate, error) {
	agentes, err := s.ListAgents()
	if err != nil {
		return nil, err
	}
	asignaciones, err := s.store.ListAssignments(db.FiltroAsignaciones{})
	if err != nil {
		return nil, err
	}
	dueAgents, err := s.store.CheckReanimations()
	if err != nil {
		return nil, err
	}
	tareas, err := s.store.ListTasks(db.FiltroTareas{})
	if err != nil {
		return nil, err
	}

	asignacionPorAgente := map[string]*db.Asignacion{}
	for _, asignacion := range asignaciones {
		if asignacion == nil || asignacion.Estado != db.AsignacionActiva {
			continue
		}
		if _, ok := asignacionPorAgente[asignacion.Agente]; !ok {
			asignacionPorAgente[asignacion.Agente] = asignacion
		}
	}
	openTasksPorAgente := map[string]int{}
	blockedTasksPorAgente := map[string]int{}
	leasesByAgent := map[string][]WorkLease{}
	for _, tarea := range tareas {
		if tarea == nil || tarea.Agente == nil {
			continue
		}
		switch tarea.Estado {
		case db.EstadoCompletada, db.EstadoCancelada:
			continue
		case db.EstadoBloqueada:
			blockedTasksPorAgente[*tarea.Agente]++
		default:
			openTasksPorAgente[*tarea.Agente]++
		}
		switch tarea.Estado {
		case db.EstadoAsignada, db.EstadoEnProgreso, db.EstadoBloqueada:
			agente := strings.TrimSpace(*tarea.Agente)
			if agente == "" {
				continue
			}
			leasesByAgent[agente] = append(leasesByAgent[agente], WorkLease{
				TaskID:    tarea.ID,
				Title:     strings.TrimSpace(tarea.Titulo),
				State:     tarea.Estado,
				ProjectID: tarea.ProyectoID,
				Module:    strings.TrimSpace(tarea.Modulo),
			})
		}
	}
	for agente, leases := range leasesByAgent {
		sort.Slice(leases, func(i, j int) bool { return leases[i].TaskID < leases[j].TaskID })
		leasesByAgent[agente] = leases
	}

	now := time.Now().UTC()
	staleThreshold := workerOutputStaleThreshold(s.store)
	agenteByName := map[string]*db.Agente{}
	candidateNames := map[string]struct{}{}
	for _, agente := range agentes {
		if agente == nil {
			continue
		}
		name := strings.TrimSpace(agente.Nombre)
		if name == "" {
			continue
		}
		agenteByName[name] = agente
		if agente.ReanimarAt == nil {
			continue
		}
		if activeOnly && !agente.Habilitado {
			continue
		}
		due := !agente.ReanimarAt.After(now)
		if !includeFuture && !due {
			continue
		}
		if activeOnly && !reanimationCandidateHasCanonicalWork(ReanimationCandidate{
			OpenTasks:    openTasksPorAgente[name],
			BlockedTasks: blockedTasksPorAgente[name],
			Leases:       leasesByAgent[name],
		}) {
			continue
		}
		candidateNames[name] = struct{}{}
	}
	for _, agente := range dueAgents {
		if agente == nil {
			continue
		}
		name := strings.TrimSpace(agente.Nombre)
		if name == "" {
			continue
		}
		if activeOnly && !agente.Habilitado {
			continue
		}
		if activeOnly && !reanimationCandidateHasCanonicalWork(ReanimationCandidate{
			OpenTasks:    openTasksPorAgente[name],
			BlockedTasks: blockedTasksPorAgente[name],
			Leases:       leasesByAgent[name],
		}) {
			continue
		}
		candidateNames[name] = struct{}{}
		if _, ok := agenteByName[name]; !ok {
			agenteByName[name] = agente
		}
	}

	rowByAgent := map[string]Row{}
	for name := range candidateNames {
		agente := agenteByName[name]
		if agente == nil {
			continue
		}
		var row Row
		if activeOnly {
			row = buildLightOperationalRow(agente, asignacionPorAgente[name], openTasksPorAgente[name], blockedTasksPorAgente[name], now, staleThreshold)
		} else {
			var err error
			row, err = s.buildReanimationRowForAgent(agente, asignacionPorAgente[name], openTasksPorAgente[name], blockedTasksPorAgente[name], now, staleThreshold)
			if err != nil {
				return nil, err
			}
		}
		rowByAgent[name] = row
	}

	out := make([]ReanimationCandidate, 0, len(rowByAgent)+len(dueAgents))
	candidates := map[string]ReanimationCandidate{}
	for _, row := range rowByAgent {
		if row.Agente == nil || row.Agente.ReanimarAt == nil {
			continue
		}
		if activeOnly && !row.Agente.Habilitado {
			continue
		}
		due := !row.Agente.ReanimarAt.After(now)
		if !includeFuture && !due {
			continue
		}
		name := strings.TrimSpace(row.Agente.Nombre)
		candidates[name] = ReanimationCandidate{
			Name:              strings.TrimSpace(row.Agente.Nombre),
			Role:              strings.TrimSpace(row.Agente.Rol),
			Enabled:           row.Agente.Habilitado,
			ActiveNow:         row.Agente.Activo,
			EstadoCuota:       strings.TrimSpace(row.Agente.EstadoCuota),
			MotivoPausa:       strings.TrimSpace(row.Agente.MotivoPausa),
			ReanimarAt:        row.Agente.ReanimarAt,
			Due:               due,
			RuntimeState:      strings.TrimSpace(row.runtimeState()),
			HandleState:       strings.TrimSpace(row.handleState()),
			WorkerState:       strings.TrimSpace(row.WorkerState),
			OperationalState:  strings.TrimSpace(row.EstadoOperativo),
			OperationalDetail: strings.TrimSpace(row.DetalleOperativo),
			MailboxPending:    row.MailboxPending,
			OpenTasks:         row.OpenTasks,
			BlockedTasks:      row.BlockedTasks,
			Leases:            leasesByAgent[name],
		}
		if row.Asignacion != nil {
			item := candidates[name]
			item.AssignmentProject = strings.TrimSpace(row.Asignacion.ProyectoSlug)
			candidates[name] = item
		}
	}
	for _, agente := range dueAgents {
		if agente == nil {
			continue
		}
		name := strings.TrimSpace(agente.Nombre)
		if name == "" {
			continue
		}
		if activeOnly && !agente.Habilitado {
			continue
		}
		row, ok := rowByAgent[name]
		item := candidates[name]
		if !ok {
			if activeOnly {
				row = buildLightOperationalRow(agente, asignacionPorAgente[name], openTasksPorAgente[name], blockedTasksPorAgente[name], now, staleThreshold)
			} else {
				row, err = s.buildReanimationRowForAgent(agente, asignacionPorAgente[name], openTasksPorAgente[name], blockedTasksPorAgente[name], now, staleThreshold)
				if err != nil {
					return nil, err
				}
			}
		}
		item.Name = name
		item.Role = firstNonEmpty(strings.TrimSpace(agente.Rol), item.Role)
		item.Enabled = agente.Habilitado
		item.ActiveNow = agente.Activo
		item.EstadoCuota = firstNonEmpty(strings.TrimSpace(agente.EstadoCuota), item.EstadoCuota)
		item.MotivoPausa = firstNonEmpty(strings.TrimSpace(agente.MotivoPausa), item.MotivoPausa)
		item.ReanimarAt = agente.ReanimarAt
		item.Due = true
		if item.AssignmentProject == "" && row.Asignacion != nil {
			item.AssignmentProject = strings.TrimSpace(row.Asignacion.ProyectoSlug)
		}
		if item.RuntimeState == "" {
			item.RuntimeState = strings.TrimSpace(row.runtimeState())
		}
		if item.HandleState == "" {
			item.HandleState = strings.TrimSpace(row.handleState())
		}
		if item.WorkerState == "" {
			item.WorkerState = strings.TrimSpace(row.WorkerState)
		}
		if item.OperationalState == "" {
			item.OperationalState = strings.TrimSpace(row.EstadoOperativo)
		}
		if item.OperationalDetail == "" {
			item.OperationalDetail = strings.TrimSpace(row.DetalleOperativo)
		}
		if item.MailboxPending == 0 {
			item.MailboxPending = row.MailboxPending
		}
		if item.OpenTasks == 0 {
			item.OpenTasks = row.OpenTasks
		}
		if item.BlockedTasks == 0 {
			item.BlockedTasks = row.BlockedTasks
		}
		if len(item.Leases) == 0 {
			item.Leases = leasesByAgent[name]
		}
		candidates[name] = item
	}
	for _, item := range candidates {
		if activeOnly && !reanimationCandidateHasCanonicalWork(item) {
			continue
		}
		out = append(out, item)
	}

	sort.Slice(out, func(i, j int) bool {
		leftDue, rightDue := out[i].Due, out[j].Due
		if leftDue != rightDue {
			return leftDue
		}
		leftAt, rightAt := out[i].ReanimarAt, out[j].ReanimarAt
		switch {
		case leftAt == nil && rightAt == nil:
			return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
		case leftAt == nil:
			return false
		case rightAt == nil:
			return true
		case !leftAt.Equal(*rightAt):
			return leftAt.Before(*rightAt)
		default:
			return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
		}
	})

	return out, nil
}

func (s *Service) buildActiveReanimationSchedule(agentes []*db.Agente, includeFuture bool) ([]ReanimationCandidate, error) {
	started := time.Now()
	dueAgents, err := s.store.CheckReanimations()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	staleThreshold := workerOutputStaleThreshold(s.store)
	agenteByName := map[string]*db.Agente{}
	candidateNames := map[string]struct{}{}
	for _, agente := range agentes {
		if agente == nil || !agente.Habilitado {
			continue
		}
		name := strings.TrimSpace(agente.Nombre)
		if name == "" {
			continue
		}
		agenteByName[name] = agente
		if agente.ReanimarAt == nil {
			continue
		}
		due := !agente.ReanimarAt.After(now)
		if !includeFuture && !due {
			continue
		}
		candidateNames[name] = struct{}{}
	}
	for _, agente := range dueAgents {
		if agente == nil || !agente.Habilitado {
			continue
		}
		name := strings.TrimSpace(agente.Nombre)
		if name == "" {
			continue
		}
		candidateNames[name] = struct{}{}
		if _, ok := agenteByName[name]; !ok {
			agenteByName[name] = agente
		}
	}

	out := make([]ReanimationCandidate, 0, len(candidateNames))
	for name := range candidateNames {
		agente := agenteByName[name]
		if agente == nil || agente.ReanimarAt == nil {
			continue
		}

		asignaciones, err := s.store.ListAssignments(db.FiltroAsignaciones{Agente: &name})
		if err != nil {
			return nil, err
		}
		asignacion := latestAssignmentForAgent(asignaciones)

		tareas, err := s.store.ListTasks(db.FiltroTareas{Agente: &name})
		if err != nil {
			return nil, err
		}
		openTasks := 0
		blockedTasks := 0
		leases := make([]WorkLease, 0, len(tareas))
		for _, tarea := range tareas {
			if tarea == nil || tarea.Agente == nil {
				continue
			}
			switch tarea.Estado {
			case db.EstadoCompletada, db.EstadoCancelada:
				continue
			case db.EstadoBloqueada:
				blockedTasks++
			default:
				openTasks++
			}
			switch tarea.Estado {
			case db.EstadoAsignada, db.EstadoEnProgreso, db.EstadoBloqueada:
				leases = append(leases, WorkLease{
					TaskID:    tarea.ID,
					Title:     strings.TrimSpace(tarea.Titulo),
					State:     tarea.Estado,
					ProjectID: tarea.ProyectoID,
					Module:    strings.TrimSpace(tarea.Modulo),
				})
			}
		}
		sort.Slice(leases, func(i, j int) bool { return leases[i].TaskID < leases[j].TaskID })

		item := ReanimationCandidate{
			Name:         name,
			Role:         strings.TrimSpace(agente.Rol),
			Enabled:      agente.Habilitado,
			ActiveNow:    agente.Activo,
			EstadoCuota:  strings.TrimSpace(agente.EstadoCuota),
			MotivoPausa:  strings.TrimSpace(agente.MotivoPausa),
			ReanimarAt:   agente.ReanimarAt,
			Due:          !agente.ReanimarAt.After(now),
			OpenTasks:    openTasks,
			BlockedTasks: blockedTasks,
			Leases:       leases,
		}
		if asignacion != nil {
			item.AssignmentProject = strings.TrimSpace(asignacion.ProyectoSlug)
		}
		if !reanimationCandidateHasCanonicalWork(item) {
			continue
		}

		row := buildLightOperationalRow(agente, asignacion, openTasks, blockedTasks, now, staleThreshold)
		item.RuntimeState = strings.TrimSpace(row.runtimeState())
		item.HandleState = strings.TrimSpace(row.handleState())
		item.WorkerState = strings.TrimSpace(row.WorkerState)
		item.OperationalState = strings.TrimSpace(row.EstadoOperativo)
		item.OperationalDetail = strings.TrimSpace(row.DetalleOperativo)
		out = append(out, item)
	}

	sort.Slice(out, func(i, j int) bool {
		leftDue, rightDue := out[i].Due, out[j].Due
		if leftDue != rightDue {
			return leftDue
		}
		leftAt, rightAt := out[i].ReanimarAt, out[j].ReanimarAt
		switch {
		case leftAt == nil && rightAt == nil:
			return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
		case leftAt == nil:
			return false
		case rightAt == nil:
			return true
		case !leftAt.Equal(*rightAt):
			return leftAt.Before(*rightAt)
		default:
			return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
		}
	})

	if elapsed := time.Since(started); elapsed > time.Second {
		log.Printf("orquesta[reanimaciones] active_only include_future=%t agentes=%d due=%d candidates=%d out=%d elapsed=%s",
			includeFuture, len(agentes), len(dueAgents), len(candidateNames), len(out), elapsed.Round(time.Millisecond))
	}

	return out, nil
}

func (s *Service) buildReanimationRowForAgent(agente *db.Agente, asignacion *db.Asignacion, openTasks, blockedTasks int, now time.Time, staleThreshold time.Duration) (Row, error) {
	if agente == nil {
		return Row{}, nil
	}
	nombre := strings.TrimSpace(agente.Nombre)
	row := Row{
		Agente:       agente,
		Asignacion:   asignacion,
		OpenTasks:    openTasks,
		BlockedTasks: blockedTasks,
	}

	sesion, err := s.currentOrLastSessionForAgent(nombre)
	if err != nil {
		return Row{}, err
	}
	row.Sesion = sesion

	runtimes, err := s.store.ListRuntimes(db.FiltroRuntimes{Agente: &nombre})
	if err != nil {
		return Row{}, err
	}
	row.Runtime = latestRuntimeForAgent(runtimes)

	if hotHandles, err := s.store.ListRecentOperationalRuntimeHandles(); err == nil {
		if handle := hotHandles[nombre]; handle != nil {
			row.Handle = handle
		}
	}
	if row.Handle == nil {
		handles, err := s.store.ListCanonicalRuntimeHandles(&nombre)
		if err != nil {
			return Row{}, err
		}
		row.Handle = latestHandleForAgent(handles)
	}
	if row.Handle == nil {
		handles, err := s.store.ListRuntimeHandles(&nombre)
		if err != nil {
			return Row{}, err
		}
		row.Handle = latestHandleForAgent(handles)
	}

	row.EstadoOperativo, row.DetalleOperativo = deriveOperationalState(row, now, staleThreshold)
	return row, nil
}

func buildLightOperationalRow(agente *db.Agente, asignacion *db.Asignacion, openTasks, blockedTasks int, now time.Time, staleThreshold time.Duration) Row {
	row := Row{
		Agente:       agente,
		Asignacion:   asignacion,
		OpenTasks:    openTasks,
		BlockedTasks: blockedTasks,
	}
	row.EstadoOperativo, row.DetalleOperativo = deriveOperationalState(row, now, staleThreshold)
	return row
}

func cloneCompactDetail(detail *Detail) *Detail {
	if detail == nil {
		return nil
	}
	clone := *detail
	if len(detail.Asignaciones) > 0 {
		clone.Asignaciones = append([]*db.Asignacion(nil), detail.Asignaciones...)
	}
	return &clone
}

func cloneReanimationCandidates(rows []ReanimationCandidate) []ReanimationCandidate {
	if len(rows) == 0 {
		return nil
	}
	out := make([]ReanimationCandidate, len(rows))
	copy(out, rows)
	for i := range out {
		if len(out[i].Leases) > 0 {
			out[i].Leases = append([]WorkLease(nil), out[i].Leases...)
		}
	}
	return out
}

func (s *Service) getOrBuildCompactDetail(nombre string, fn func() (*Detail, error)) (*Detail, error) {
	now := time.Now().UTC()
	s.cacheMu.Lock()
	cached, cachedOK := s.compactDetailCache[nombre]
	if cachedOK && cached.detail != nil && now.Before(cached.expires) {
		detail := cloneCompactDetail(cached.detail)
		s.cacheMu.Unlock()
		return detail, nil
	}
	if flight, ok := s.compactDetailFlight[nombre]; ok {
		if cachedOK && cached.detail != nil && staleCacheStillUsable(now, cached.expires, compactDetailStaleWhileRevalidateTTL) {
			detail := cloneCompactDetail(cached.detail)
			s.cacheMu.Unlock()
			return detail, nil
		}
		done := flight.done
		s.cacheMu.Unlock()
		<-done
		return cloneCompactDetail(flight.detail), flight.err
	}
	if cachedOK && cached.detail != nil && staleCacheStillUsable(now, cached.expires, compactDetailStaleWhileRevalidateTTL) {
		flight := &compactDetailFlight{done: make(chan struct{})}
		s.compactDetailFlight[nombre] = flight
		detail := cloneCompactDetail(cached.detail)
		s.cacheMu.Unlock()
		go s.refreshCompactDetail(nombre, flight, fn)
		return detail, nil
	}
	flight := &compactDetailFlight{done: make(chan struct{})}
	s.compactDetailFlight[nombre] = flight
	s.cacheMu.Unlock()
	return s.runCompactDetailRefresh(nombre, flight, fn)
}

func (s *Service) getOrBuildActiveReanimationSchedule(key string, fn func() ([]ReanimationCandidate, error)) ([]ReanimationCandidate, error) {
	now := time.Now().UTC()
	s.cacheMu.Lock()
	cached, cachedOK := s.reanimCache[key]
	if cachedOK && now.Before(cached.expires) {
		rows := cloneReanimationCandidates(cached.rows)
		s.cacheMu.Unlock()
		return rows, nil
	}
	if flight, ok := s.reanimFlight[key]; ok {
		if cachedOK && staleCacheStillUsable(now, cached.expires, activeReanimationScheduleStaleWhileRevalidateTTL) {
			rows := cloneReanimationCandidates(cached.rows)
			s.cacheMu.Unlock()
			return rows, nil
		}
		done := flight.done
		s.cacheMu.Unlock()
		<-done
		return cloneReanimationCandidates(flight.rows), flight.err
	}
	if cachedOK && staleCacheStillUsable(now, cached.expires, activeReanimationScheduleStaleWhileRevalidateTTL) {
		flight := &reanimationScheduleFlight{done: make(chan struct{})}
		s.reanimFlight[key] = flight
		rows := cloneReanimationCandidates(cached.rows)
		s.cacheMu.Unlock()
		go s.refreshActiveReanimationSchedule(key, flight, fn)
		return rows, nil
	}
	flight := &reanimationScheduleFlight{done: make(chan struct{})}
	s.reanimFlight[key] = flight
	s.cacheMu.Unlock()
	return s.runActiveReanimationScheduleRefresh(key, flight, fn)
}

func staleCacheStillUsable(now, expires time.Time, grace time.Duration) bool {
	if grace <= 0 || expires.IsZero() {
		return false
	}
	return now.Before(expires.Add(grace))
}

func (s *Service) refreshCompactDetail(nombre string, flight *compactDetailFlight, fn func() (*Detail, error)) {
	_, _ = s.runCompactDetailRefresh(nombre, flight, fn)
}

func (s *Service) runCompactDetailRefresh(nombre string, flight *compactDetailFlight, fn func() (*Detail, error)) (*Detail, error) {
	detail, err := fn()

	s.cacheMu.Lock()
	if err == nil && detail != nil {
		s.compactDetailCache[nombre] = cachedCompactDetail{
			detail:  cloneCompactDetail(detail),
			expires: time.Now().UTC().Add(compactDetailCacheTTL),
		}
	}
	flight.detail = cloneCompactDetail(detail)
	flight.err = err
	delete(s.compactDetailFlight, nombre)
	close(flight.done)
	s.cacheMu.Unlock()
	return cloneCompactDetail(detail), err
}

func (s *Service) refreshActiveReanimationSchedule(key string, flight *reanimationScheduleFlight, fn func() ([]ReanimationCandidate, error)) {
	_, _ = s.runActiveReanimationScheduleRefresh(key, flight, fn)
}

func (s *Service) runActiveReanimationScheduleRefresh(key string, flight *reanimationScheduleFlight, fn func() ([]ReanimationCandidate, error)) ([]ReanimationCandidate, error) {
	rows, err := fn()

	s.cacheMu.Lock()
	if err == nil {
		s.reanimCache[key] = cachedReanimationSchedule{
			rows:    cloneReanimationCandidates(rows),
			expires: time.Now().UTC().Add(activeReanimationScheduleCacheTTL),
		}
	}
	flight.rows = cloneReanimationCandidates(rows)
	flight.err = err
	delete(s.reanimFlight, key)
	close(flight.done)
	s.cacheMu.Unlock()
	return cloneReanimationCandidates(rows), err
}

func (s *Service) currentOrLastSessionForAgent(nombre string) (*db.Sesion, error) {
	nombre = strings.TrimSpace(nombre)
	if nombre == "" {
		return nil, nil
	}
	sesion, err := s.store.GetActiveSession(nombre, nil)
	switch {
	case err == nil:
		if sesion != nil {
			return sesion, nil
		}
	case err != sql.ErrNoRows:
		return nil, err
	}
	sesion, err = s.store.GetLastSession(nombre, nil)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return sesion, err
}

func reanimationCandidateHasCanonicalWork(item ReanimationCandidate) bool {
	if item.OpenTasks > 0 || item.BlockedTasks > 0 || len(item.Leases) > 0 {
		return true
	}
	return false
}

func (s *Service) countActionableMailboxPendingForReanimation(items []*db.RuntimeMailboxMessage, handle *db.RuntimeHandle, runtime *db.RuntimeInstance, taskByID map[int64]*db.Tarea, agente string) (int, error) {
	count := 0
	for _, item := range items {
		if item == nil || !strings.EqualFold(strings.TrimSpace(item.Estado), "pendiente") {
			continue
		}
		covered := false
		if coveredValue, _, _, err := s.store.RuntimeMailboxCoveredByBootstrapPending(item.ID, handle, runtime); err == nil {
			covered = coveredValue
		}
		if covered || !runtimeMailboxCuentaComoTrabajoCanonico(item, taskByID, agente) {
			continue
		}
		count++
	}
	return count, nil
}

func runtimeMailboxCuentaComoTrabajoCanonico(msg *db.RuntimeMailboxMessage, taskByID map[int64]*db.Tarea, agente string) bool {
	if msg == nil {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(msg.Kind), "autonomia") {
		return true
	}
	payload := runtimeMailboxPayloadMap(msg.PayloadJSON)
	accion := strings.ToLower(strings.TrimSpace(stringFromRuntimeMailboxPayload(payload, "accion")))
	switch accion {
	case "pedir_intervencion":
		return false
	case "continuar_trabajo":
		tareaID := int64FromRuntimeMailboxPayload(payload, "tarea_id")
		if tareaID <= 0 {
			return false
		}
		tarea := taskByID[tareaID]
		if tarea == nil || tarea.Agente == nil || !strings.EqualFold(strings.TrimSpace(*tarea.Agente), strings.TrimSpace(agente)) {
			return false
		}
		switch tarea.Estado {
		case db.EstadoAsignada, db.EstadoEnProgreso, db.EstadoBloqueada:
			return true
		default:
			return false
		}
	default:
		return true
	}
}

func runtimeMailboxEsContinuidadPendiente(msg *db.RuntimeMailboxMessage, taskByID map[int64]*db.Tarea, agente string) bool {
	if msg == nil || !strings.EqualFold(strings.TrimSpace(msg.Kind), "autonomia") {
		return false
	}
	payload := runtimeMailboxPayloadMap(msg.PayloadJSON)
	if !strings.EqualFold(strings.TrimSpace(stringFromRuntimeMailboxPayload(payload, "accion")), "continuar_trabajo") {
		return false
	}
	tareaID := int64FromRuntimeMailboxPayload(payload, "tarea_id")
	if tareaID <= 0 {
		return false
	}
	tarea := taskByID[tareaID]
	if tarea == nil || tarea.Agente == nil || !strings.EqualFold(strings.TrimSpace(*tarea.Agente), strings.TrimSpace(agente)) {
		return false
	}
	switch tarea.Estado {
	case db.EstadoAsignada, db.EstadoEnProgreso:
		return true
	default:
		return false
	}
}

func runtimeMailboxPayloadMap(raw string) map[string]any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil || out == nil {
		return map[string]any{}
	}
	return out
}

func stringFromRuntimeMailboxPayload(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	value, ok := payload[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func int64FromRuntimeMailboxPayload(payload map[string]any, key string) int64 {
	if payload == nil {
		return 0
	}
	value, ok := payload[key]
	if !ok {
		return 0
	}
	switch v := value.(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	case json.Number:
		out, _ := v.Int64()
		return out
	case string:
		out, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		return out
	default:
		return 0
	}
}

func buildAgentEntity(store Store, row Row, tareas []*db.Tarea) *AgentEntity {
	if row.Agente == nil {
		return nil
	}
	accountAvailable := true
	accountOccupiedBy := ""
	if store != nil {
		if ok, ocupadoPor, err := store.SharedAccountAllowsActivation(strings.TrimSpace(row.Agente.Nombre)); err == nil {
			accountAvailable = ok
			accountOccupiedBy = strings.TrimSpace(ocupadoPor)
		}
	}
	entity := &AgentEntity{
		Name:                        strings.TrimSpace(row.Agente.Nombre),
		Role:                        strings.TrimSpace(row.Agente.Rol),
		Enabled:                     row.Agente.Habilitado,
		ActiveNow:                   row.Agente.Activo,
		AccountID:                   strings.TrimSpace(row.Agente.CuentaID),
		AccountEmail:                strings.TrimSpace(row.Agente.CuentaEmail),
		AccountUser:                 strings.TrimSpace(row.Agente.CuentaUsuario),
		AccountKey:                  strings.TrimSpace(db.CuentaClaveAgente(row.Agente)),
		AccountAvailable:            accountAvailable,
		AccountOccupiedBy:           accountOccupiedBy,
		RuntimeAdapter:              strings.TrimSpace(runtimeAdapterName(row)),
		Transport:                   strings.TrimSpace(handleTransportName(row.Handle)),
		RuntimeState:                strings.TrimSpace(row.runtimeState()),
		HandleState:                 strings.TrimSpace(row.handleState()),
		WorkerState:                 strings.TrimSpace(row.WorkerState),
		WorkerAlive:                 row.WorkerAlive,
		WorkerDriver:                strings.TrimSpace(row.WorkerDriver),
		WorkerTransport:             strings.TrimSpace(row.WorkerTransport),
		WorkerSessionRef:            strings.TrimSpace(row.WorkerSessionRef),
		WorkerRuntimeRef:            strings.TrimSpace(row.WorkerRuntimeRef),
		ExternalSessionID:           strings.TrimSpace(row.WorkerExternalSessionID),
		MailboxDeliveryMode:         strings.TrimSpace(row.WorkerMailboxDeliveryMode),
		OperationalState:            strings.TrimSpace(row.EstadoOperativo),
		OperationalDetail:           strings.TrimSpace(row.DetalleOperativo),
		LastAutonomyAction:          strings.TrimSpace(row.LastAutonomyAction),
		LastAutonomySource:          strings.TrimSpace(row.LastAutonomySource),
		LastAutonomyMoment:          row.LastAutonomyMoment,
		LastAutonomyState:           strings.TrimSpace(row.LastAutonomyState),
		LastAutonomyReason:          strings.TrimSpace(row.LastAutonomyReason),
		LastAutonomyVerificationKey: strings.TrimSpace(row.LastAutonomyVerificationKey),
		LastAutonomyDispatchState:   strings.TrimSpace(row.LastAutonomyDispatchState),
		LastAutonomyDeliveryState:   strings.TrimSpace(row.LastAutonomyDeliveryState),
		LastAutonomyReceiptSource:   strings.TrimSpace(row.LastAutonomyReceiptSource),
		Leases:                      buildWorkLeases(tareas),
	}
	if row.Asignacion != nil {
		entity.AssignmentProject = strings.TrimSpace(row.Asignacion.ProyectoSlug)
	}
	return entity
}

func buildWorkLeases(tareas []*db.Tarea) []WorkLease {
	out := make([]WorkLease, 0, len(tareas))
	for _, tarea := range tareas {
		if tarea == nil {
			continue
		}
		switch tarea.Estado {
		case db.EstadoAsignada, db.EstadoEnProgreso, db.EstadoBloqueada:
			out = append(out, WorkLease{
				TaskID:    tarea.ID,
				Title:     strings.TrimSpace(tarea.Titulo),
				State:     tarea.Estado,
				ProjectID: tarea.ProyectoID,
				Module:    strings.TrimSpace(tarea.Modulo),
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TaskID < out[j].TaskID })
	return out
}

func runtimeAdapterName(row Row) string {
	if row.Runtime != nil && strings.TrimSpace(row.Runtime.Connector) != "" {
		return strings.TrimSpace(row.Runtime.Connector)
	}
	if row.Sesion != nil && strings.TrimSpace(row.Sesion.Herramienta) != "" {
		return strings.TrimSpace(row.Sesion.Herramienta)
	}
	return ""
}

func handleTransportName(handle *db.RuntimeHandle) string {
	if handle == nil {
		return ""
	}
	return strings.TrimSpace(handle.Transporte)
}

func (s *Service) Investigate(query, projectRef string, limit int) (*InvestigationReport, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("consulta obligatoria")
	}
	if limit <= 0 {
		limit = 25
	}

	var (
		project   *db.Proyecto
		projectID *int64
		err       error
	)
	projectRef = strings.TrimSpace(projectRef)
	if projectRef != "" {
		project, err = s.store.GetProject(projectRef)
		if err != nil {
			return nil, err
		}
		if project != nil {
			projectID = &project.ID
		}
	}

	transcript, err := s.store.ListRuntimeTranscript(db.FiltroRuntimeTranscript{
		ProyectoID: projectID,
		Query:      &query,
		Limit:      limit,
	})
	if err != nil {
		return nil, err
	}

	report := &InvestigationReport{
		Query:        query,
		Proyecto:     project,
		Results:      []*InvestigationAgentResult{},
		TotalMatches: len(transcript),
	}
	if len(transcript) == 0 {
		return report, nil
	}

	byAgent := map[string]*InvestigationAgentResult{}
	agentOrder := make([]string, 0, len(transcript))
	for _, item := range transcript {
		if item == nil {
			continue
		}
		agente := strings.TrimSpace(item.Agente)
		if agente == "" {
			continue
		}
		result := byAgent[agente]
		if result == nil {
			result, err = s.buildInvestigationAgentResult(agente, projectID)
			if err != nil {
				return nil, err
			}
			byAgent[agente] = result
			agentOrder = append(agentOrder, agente)
		}
		match := &InvestigationMatch{Transcript: item}
		match.Runtime = investigationRuntimeForMatch(result, item.RuntimeID)
		if item.HandleID != nil {
			traceDir, traceManifest, logPath, workingDir := investigationTraceInfo(result, *item.HandleID)
			match.TraceDir = traceDir
			match.TraceManifest = traceManifest
			match.LogPath = logPath
			match.WorkingDir = firstNonEmptyAgent(strings.TrimSpace(workingDir), runtimeWorkingDir(match.Runtime))
		} else {
			match.WorkingDir = runtimeWorkingDir(match.Runtime)
		}
		result.Matches = append(result.Matches, match)
	}

	for _, agente := range agentOrder {
		if item := byAgent[agente]; item != nil {
			report.Results = append(report.Results, item)
		}
	}
	sort.Slice(report.Results, func(i, j int) bool {
		left := latestInvestigationMoment(report.Results[i])
		right := latestInvestigationMoment(report.Results[j])
		if !left.Equal(right) {
			return left.After(right)
		}
		leftName := ""
		rightName := ""
		if report.Results[i].Agente != nil {
			leftName = strings.ToLower(strings.TrimSpace(report.Results[i].Agente.Nombre))
		}
		if report.Results[j].Agente != nil {
			rightName = strings.ToLower(strings.TrimSpace(report.Results[j].Agente.Nombre))
		}
		return leftName < rightName
	})
	return report, nil
}

func (s *Service) buildInvestigationAgentResult(nombre string, projectID *int64) (*InvestigationAgentResult, error) {
	agente, err := s.store.GetAgent(nombre)
	if err != nil {
		return nil, err
	}
	if agente == nil {
		agente = &db.Agente{Nombre: nombre}
	}
	runtimes, err := s.store.ListRuntimes(db.FiltroRuntimes{Agente: &nombre, ProyectoID: projectID})
	if err != nil {
		return nil, err
	}
	handles, err := s.store.ListRuntimeHandles(&nombre)
	if err != nil {
		return nil, err
	}
	tasks, err := s.store.ListTasks(db.FiltroTareas{Agente: &nombre, ProyectoID: projectID})
	if err != nil {
		return nil, err
	}
	if projectID != nil && len(tasks) == 0 {
		tasks, err = s.store.ListTasks(db.FiltroTareas{Agente: &nombre})
		if err != nil {
			return nil, err
		}
	}
	checkpoints, err := s.store.ListRuntimeCheckpoints(db.FiltroRuntimeCheckpoints{Agente: &nombre, ProyectoID: projectID, Limit: 5})
	if err != nil {
		return nil, err
	}
	return &InvestigationAgentResult{
		Agente:         agente,
		Runtime:        latestRuntimeForInvestigation(runtimes),
		LastCheckpoint: latestCheckpointForInvestigation(checkpoints),
		OpenTasks:      openTasksForInvestigation(tasks),
		Matches:        []*InvestigationMatch{},
		handles:        handles,
		runtimes:       runtimes,
	}, nil
}

func latestInvestigationMoment(result *InvestigationAgentResult) time.Time {
	if result == nil {
		return time.Time{}
	}
	var latest time.Time
	for _, item := range result.Matches {
		if item == nil || item.Transcript == nil {
			continue
		}
		if item.Transcript.CreatedAt.After(latest) {
			latest = item.Transcript.CreatedAt
		}
	}
	return latest
}

func latestRuntimeForInvestigation(items []*db.RuntimeInstance) *db.RuntimeInstance {
	var picked *db.RuntimeInstance
	for _, item := range items {
		if item == nil {
			continue
		}
		if picked == nil || runtimeMoment(item).After(runtimeMoment(picked)) {
			picked = item
		}
	}
	return picked
}

func latestCheckpointForInvestigation(items []*db.RuntimeCheckpoint) *db.RuntimeCheckpoint {
	var picked *db.RuntimeCheckpoint
	for _, item := range items {
		if item == nil {
			continue
		}
		if picked == nil || item.CreatedAt.After(picked.CreatedAt) {
			picked = item
		}
	}
	return picked
}

func openTasksForInvestigation(items []*db.Tarea) int {
	total := 0
	for _, item := range items {
		if item == nil {
			continue
		}
		switch item.Estado {
		case db.TareaCompletada, db.TareaCancelada, db.TareaBacklog:
			continue
		default:
			total++
		}
	}
	return total
}

func investigationTraceInfo(result *InvestigationAgentResult, handleID int64) (string, string, string, string) {
	if result == nil || handleID <= 0 {
		return "", "", "", ""
	}
	for _, handle := range result.handles {
		if handle == nil || handle.ID != handleID {
			continue
		}
		meta := map[string]any{}
		if strings.TrimSpace(handle.MetadataJSON) != "" {
			_ = json.Unmarshal([]byte(handle.MetadataJSON), &meta)
		}
		return stringMapValueAgent(meta, "trace_dir"),
			stringMapValueAgent(meta, "trace_manifest"),
			stringMapValueAgent(meta, "log_path"),
			stringMapValueAgent(meta, "working_dir")
	}
	return "", "", "", ""
}

func investigationRuntimeForMatch(result *InvestigationAgentResult, runtimeID int64) *db.RuntimeInstance {
	if result == nil || runtimeID <= 0 {
		return result.Runtime
	}
	for _, item := range result.runtimes {
		if item != nil && item.ID == runtimeID {
			return item
		}
	}
	return result.Runtime
}

func runtimeWorkingDir(runtime *db.RuntimeInstance) string {
	if runtime == nil {
		return ""
	}
	return strings.TrimSpace(runtime.CWD)
}

func firstNonEmptyAgent(items ...string) string {
	for _, item := range items {
		if strings.TrimSpace(item) != "" {
			return strings.TrimSpace(item)
		}
	}
	return ""
}

func stringMapValueAgent(raw map[string]any, key string) string {
	if raw == nil {
		return ""
	}
	value, ok := raw[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func (s *Service) listMailboxForAgent(nombre string) ([]*db.RuntimeMailboxMessage, error) {
	toAgente := nombre
	inbox, err := s.store.ListRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &toAgente})
	if err != nil {
		return nil, err
	}
	fromAgente := nombre
	outbox, err := s.store.ListRuntimeMailbox(db.FiltroRuntimeMailbox{FromAgente: &fromAgente})
	if err != nil {
		return nil, err
	}
	merged := make([]*db.RuntimeMailboxMessage, 0, len(inbox)+len(outbox))
	seen := map[int64]struct{}{}
	for _, set := range [][]*db.RuntimeMailboxMessage{inbox, outbox} {
		for _, msg := range set {
			if msg == nil {
				continue
			}
			if _, ok := seen[msg.ID]; ok {
				continue
			}
			seen[msg.ID] = struct{}{}
			merged = append(merged, msg)
		}
	}
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].ID > merged[j].ID
	})
	return merged, nil
}

func runtimeMoment(runtime *db.RuntimeInstance) time.Time {
	if runtime == nil {
		return time.Time{}
	}
	for _, value := range []*time.Time{runtime.UltimaActividadAt, runtime.LastHeartbeatAt, runtime.LastEventAt} {
		if value != nil && !value.IsZero() {
			return *value
		}
	}
	if !runtime.UpdatedAt.IsZero() {
		return runtime.UpdatedAt
	}
	return runtime.CreatedAt
}

func runtimeHandleMoment(handle *db.RuntimeHandle) time.Time {
	if handle == nil {
		return time.Time{}
	}
	if handle.LastSeenAt != nil && !handle.LastSeenAt.IsZero() {
		return *handle.LastSeenAt
	}
	if !handle.UpdatedAt.IsZero() {
		return handle.UpdatedAt
	}
	return handle.CreatedAt
}

func runtimeHandleSostieneOperacion(handle *db.RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(handle.Estado)) {
	case "activo", "pausado":
		return true
	default:
		return false
	}
}

func deriveOperationalState(row Row, now time.Time, workerOutputStaleThreshold time.Duration) (string, string) {
	agente := row.Agente
	if agente == nil {
		return "desconocido", ""
	}
	if !agente.Habilitado {
		return "retirado", "agente retirado"
	}

	sinCuotaProveedor := db.AgenteSinCuotaProveedorEfectivo(agente)
	workerState := strings.ToLower(strings.TrimSpace(row.WorkerState))
	workerOperativoFresco := row.WorkerAlive &&
		row.workerHeartbeatRecent(now) &&
		workerState != "" &&
		workerState != "blocked_quota" &&
		workerState != "blocked_auth" &&
		workerState != "failed" &&
		workerState != "stopped"

	if !sinCuotaProveedor {
		if blocked, detalle := row.runtimeCheckpointQuotaBlocked(now); blocked {
			return "bloqueado_por_cuota", detalle
		}
	}
	if !sinCuotaProveedor && !workerOperativoFresco && estadoCuotaBloqueado(agente.EstadoCuota) {
		detalle := strings.TrimSpace(agente.MotivoPausa)
		if detalle == "" && agente.ReanimarAt != nil && !agente.ReanimarAt.IsZero() {
			detalle = "reanimacion " + agente.ReanimarAt.Local().Format("15:04")
		}
		return "bloqueado_por_cuota", detalle
	}
	if blocked, detalle := row.pendingControlOrderBlocksRuntime(now); blocked {
		return "bloqueado_por_runtime", detalle
	}

	handleEstado := strings.ToLower(strings.TrimSpace(row.handleState()))
	runtimeEstado := strings.ToLower(strings.TrimSpace(row.runtimeState()))
	recentActivity := row.hasRecentOperationalActivity(now)
	hasActiveTask := row.OpenTasks > 0
	hasBlockedTask := row.BlockedTasks > 0
	continuidadAccionablePendiente := row.MailboxActionablePending > 0 || row.MailboxContinuityPending > 0
	if !continuidadAccionablePendiente && hasActiveTask && row.DurableContinuityPending(now) {
		continuidadAccionablePendiente = true
		row.MailboxContinuityPending++
	}
	pausedRuntime := estadoRuntimePausado(runtimeEstado) || estadoHandlePausado(handleEstado)
	runtimeStale := row.runtimePrincipalStale(now)
	workerRunningFresh := false
	workerAnchored := row.hasOperationalAnchor()
	if pausedRuntime {
		detalle := firstNonEmpty(row.runtimeState(), row.handleState(), "runtime pausado")
		return "bloqueado_por_runtime", detalle
	}
	if runtimeStale && !hasActiveTask {
		return "bloqueado_por_runtime", firstNonEmpty(strings.TrimSpace(row.WorkerExitError), "runtime principal stale")
	}

	if workerState != "" {
		switch workerState {
		case "blocked_auth":
			return "bloqueado_por_runtime", "worker requiere autenticacion manual"
		case "blocked_quota":
			if sinCuotaProveedor {
				return "bloqueado_por_runtime", "worker local mal clasificado como cuota"
			}
			return "bloqueado_por_cuota", "worker bloqueado por cuota"
		case "failed", "stopped":
			if hasActiveTask || row.MailboxPending > 0 {
				return "bloqueado_por_runtime", firstNonEmpty(strings.TrimSpace(row.WorkerExitError), "worker "+workerState)
			}
		}
		if (hasActiveTask || row.MailboxPending > 0) && row.workerHeartbeatStale(now) {
			return "bloqueado_por_runtime", firstNonEmpty(strings.TrimSpace(row.WorkerExitError), "heartbeat worker retrasado")
		}
		if row.workerHeartbeatRecent(now) && row.WorkerAlive {
			workerRunningFresh = true
		}
	}
	if hasActiveTask && row.KnownLegacyCLIWorker(now) {
		return "bloqueado_por_runtime", "runtime CLI legacy sin tmux canónico"
	}
	if hasActiveTask && row.MailboxContinuityPending == 1 &&
		(row.WorkerSupportsContinuityRecovery(now) ||
			(row.workerHeartbeatRecent(now) && strings.EqualFold(strings.TrimSpace(row.handleDriver()), "tmux_cli_session"))) {
		return "trabajando", firstNonEmpty(row.activitySummary(), "continuidad pendiente útil")
	}
	if hasActiveTask &&
		workerRunningFresh &&
		strings.EqualFold(workerState, "ready") &&
		runtimeagente.NormalizeMailboxDeliveryMode(row.WorkerMailboxDeliveryMode) == runtimeagente.MailboxDeliverySessionResume &&
		row.WorkerSupportsContinuityRecovery(now) {
		if row.MailboxContinuityPending > 0 ||
			row.workerProgressRecent(now, workerOutputStaleThreshold) ||
			row.workerWarmupRecent(now) {
			return "trabajando", firstNonEmpty(row.activitySummary(), "continuidad session_resume lista")
		}
	}
	if hasActiveTask && workerRunningFresh && workerState == "starting" {
		return "arrancando", firstNonEmpty(row.activitySummary(), "worker starting")
	}
	if hasActiveTask && workerRunningFresh && row.workerProgressStale(now, workerOutputStaleThreshold) {
		return "atascado", firstNonEmpty(row.workerLastProgressSummary(), row.workerLastOutputSummary(), "worker sin progreso reciente")
	}
	if hasActiveTask && workerRunningFresh && row.workerOutputStale(now, workerOutputStaleThreshold) && !row.workerProgressRecent(now, workerOutputStaleThreshold) {
		return "atascado", firstNonEmpty(row.workerLastOutputSummary(), "worker sin salida reciente")
	}
	if !hasActiveTask && row.supervisorAutonomyLive(now) {
		return "trabajando", firstNonEmpty(row.activitySummary(), "supervision autonoma viva")
	}
	if !hasActiveTask && row.MailboxPending > 0 && row.KnownLegacyCLIWorker(now) {
		return "bloqueado_por_runtime", "runtime CLI legacy sin tmux canónico"
	}
	if !hasActiveTask && row.MailboxPending > 0 && workerRunningFresh && row.workerProgressStale(now, workerOutputStaleThreshold) {
		return "atascado", firstNonEmpty(row.workerLastProgressSummary(), row.workerLastOutputSummary(), "worker sin progreso reciente")
	}
	if !hasActiveTask && row.MailboxPending > 0 && workerRunningFresh && row.workerOutputStale(now, workerOutputStaleThreshold) && !row.workerProgressRecent(now, workerOutputStaleThreshold) {
		return "atascado", firstNonEmpty(row.workerLastOutputSummary(), "worker sin salida reciente")
	}
	if hasActiveTask && row.MailboxPending > 0 && !workerAnchored {
		return "bloqueado_por_runtime", "mailbox pendiente sin runtime activo"
	}
	if !hasActiveTask && row.MailboxPending > 0 && !workerAnchored {
		return "bloqueado", "mailbox pendiente sin runtime activo"
	}
	if !hasActiveTask && row.MailboxPending > 0 && row.OrdersFailed > 0 && !workerRunningFresh {
		return "mailbox_atascada", "mailbox pendiente con fallos de control"
	}
	if !hasActiveTask && row.MailboxPending > 0 && !recentActivity {
		return "mailbox_atascada", "pendiente sin avance reciente"
	}
	if !hasActiveTask && workerState != "" && workerRunningFresh {
		if row.KnownLegacyCLIWorker(now) {
			return "bloqueado_por_runtime", "runtime CLI legacy sin tmux canónico"
		}
		return "disponible", firstNonEmpty(row.activitySummary(), "worker "+workerState)
	}
	if !hasActiveTask && (estadoRuntimeRoto(runtimeEstado) || estadoHandleRoto(handleEstado)) {
		return "bloqueado_por_runtime", firstNonEmpty(row.runtimeState(), row.handleState(), "runtime degradado")
	}

	if hasBlockedTask && !hasActiveTask {
		if row.MailboxPending > 0 || row.OrdersOpen > 0 || estadoRuntimeRoto(runtimeEstado) || estadoHandleRoto(handleEstado) {
			return "bloqueado_por_runtime", firstNonEmpty(row.activitySummary(), "tareas bloqueadas")
		}
		return "bloqueado", "tareas bloqueadas"
	}
	if hasActiveTask && row.MailboxPending > 0 && row.OrdersFailed > 0 && !workerRunningFresh {
		return "mailbox_atascada", "mailbox pendiente con fallos de control"
	}
	if hasActiveTask && row.MailboxPending > 0 && !recentActivity {
		return "mailbox_atascada", "pendiente sin avance reciente"
	}
	if hasActiveTask && continuidadAccionablePendiente &&
		(row.DurableContinuityPending(now) ||
			workerRunningFresh ||
			row.WorkerSupportsContinuityRecovery(now) ||
			(row.workerHeartbeatRecent(now) && strings.EqualFold(strings.TrimSpace(row.handleDriver()), "tmux_cli_session"))) {
		return "trabajando", firstNonEmpty(row.activitySummary(), "continuidad accionable pendiente")
	}
	if hasActiveTask && (estadoRuntimeRoto(runtimeEstado) || estadoHandleRoto(handleEstado)) {
		return "bloqueado_por_runtime", firstNonEmpty(row.runtimeState(), row.handleState())
	}
	if hasActiveTask && hasBlockedTask && recentActivity {
		return "saturado", fmt.Sprintf("%d activas, %d bloqueadas", row.OpenTasks, row.BlockedTasks)
	}
	if hasActiveTask && workerRunningFresh && row.workerHasRecentWorkSignal(now, workerOutputStaleThreshold) {
		return "trabajando", firstNonEmpty(row.activitySummary(), "worker "+workerState)
	}
	if hasActiveTask && workerRunningFresh && row.workerWarmupRecent(now) {
		return "arrancando", firstNonEmpty(row.activitySummary(), "worker "+workerState)
	}
	if hasActiveTask && workerRunningFresh {
		return "atascado", firstNonEmpty(row.workerLastProgressSummary(), row.workerLastOutputSummary(), "worker sin señal de trabajo reciente")
	}
	if hasActiveTask && recentActivity && row.workerHasRecentWorkSignal(now, workerOutputStaleThreshold) {
		return "trabajando", row.activitySummary()
	}
	if hasActiveTask && recentActivity {
		return "arrancando", row.activitySummary()
	}
	if hasActiveTask {
		return "caido", row.activitySummary()
	}
	if recentActivity {
		return "disponible", row.activitySummary()
	}
	return "sin_tarea", row.activitySummary()
}

func workerOutputStaleThreshold(store Store) time.Duration {
	if store == nil {
		return time.Duration(defaultWorkerOutputStaleSeconds) * time.Second
	}
	raw, err := store.ConfigGet("runtime_worker_output_stale_seconds")
	if err != nil {
		return time.Duration(defaultWorkerOutputStaleSeconds) * time.Second
	}
	seconds, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || seconds <= 0 {
		return time.Duration(defaultWorkerOutputStaleSeconds) * time.Second
	}
	return time.Duration(seconds) * time.Second
}

func (r Row) hasRecentOperationalActivity(now time.Time) bool {
	cutoff := now.Add(-10 * time.Minute)
	for _, ts := range []time.Time{
		runtimeMoment(r.Runtime),
		runtimeHandleMoment(r.Handle),
		sesionMoment(r.Sesion),
		checkpointMoment(r.LastCheckpoint),
	} {
		if !ts.IsZero() && !ts.Before(cutoff) {
			return true
		}
	}
	if r.WorkerHeartbeat != nil && !r.WorkerHeartbeat.IsZero() && !r.WorkerHeartbeat.Before(now.Add(-45*time.Second)) {
		return true
	}
	if r.WorkerUpdatedAt != nil && !r.WorkerUpdatedAt.IsZero() && !r.WorkerUpdatedAt.Before(now.Add(-45*time.Second)) {
		return true
	}
	if r.WorkerLastProgress != nil && !r.WorkerLastProgress.IsZero() && !r.WorkerLastProgress.Before(cutoff) {
		return true
	}
	return false
}

func (r Row) supervisorAutonomyLive(now time.Time) bool {
	if !strings.EqualFold(strings.TrimSpace(r.LastAutonomyAction), "supervisar_proyecto") {
		return false
	}
	if !r.WorkerAlive || !r.workerHeartbeatRecent(now) {
		return false
	}
	if r.LastAutonomyMoment != nil && !r.LastAutonomyMoment.IsZero() && r.LastAutonomyMoment.Before(now.Add(-10*time.Minute)) {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(r.LastAutonomySource)) {
	case "resume_payload_mailbox", "mailbox", "work_queue", "send_instruction":
		return true
	default:
		return false
	}
}

func (r Row) runtimeState() string {
	if r.Runtime == nil {
		return ""
	}
	if value := strings.TrimSpace(r.Runtime.LogicalState); value != "" {
		return value
	}
	return strings.TrimSpace(r.Runtime.ProcessState)
}

func (r Row) hasOperationalAnchor() bool {
	if r.Runtime != nil || r.Handle != nil || r.Sesion != nil {
		return true
	}
	if strings.TrimSpace(r.WorkerState) != "" {
		return true
	}
	return r.WorkerHeartbeat != nil || r.WorkerUpdatedAt != nil || r.WorkerLastProgress != nil || r.WorkerLastOutput != nil
}

func (r Row) handleState() string {
	if r.Handle == nil {
		return ""
	}
	return strings.TrimSpace(r.Handle.Estado)
}

func (r Row) activitySummary() string {
	switch {
	case r.ControlOrdersOpen > 0 && strings.TrimSpace(r.LastControlOrderType) != "":
		return "orden " + strings.TrimSpace(r.LastControlOrderType) + " pendiente"
	case strings.TrimSpace(r.WorkerState) != "":
		return "worker " + strings.TrimSpace(r.WorkerState)
	case strings.TrimSpace(r.runtimeState()) != "":
		return "runtime " + strings.TrimSpace(r.runtimeState())
	case strings.TrimSpace(r.handleState()) != "":
		return "handle " + strings.TrimSpace(r.handleState())
	case r.Sesion != nil && strings.TrimSpace(r.Sesion.Estado) != "":
		return "sesion " + strings.TrimSpace(r.Sesion.Estado)
	case r.MailboxPending > 0:
		return fmt.Sprintf("mailbox %d", r.MailboxPending)
	case r.OrdersOpen > 0:
		return fmt.Sprintf("orders %d", r.OrdersOpen)
	default:
		return ""
	}
}

func (r Row) workerHeartbeatRecent(now time.Time) bool {
	if r.WorkerHeartbeat != nil && !r.WorkerHeartbeat.IsZero() {
		return !r.WorkerHeartbeat.Before(now.Add(-45 * time.Second))
	}
	if r.WorkerUpdatedAt != nil && !r.WorkerUpdatedAt.IsZero() {
		return !r.WorkerUpdatedAt.Before(now.Add(-45 * time.Second))
	}
	return false
}

func (r Row) workerWarmupRecent(now time.Time) bool {
	for _, ts := range []*time.Time{
		r.WorkerReadyAt,
		r.WorkerUpdatedAt,
	} {
		if ts == nil || ts.IsZero() {
			continue
		}
		return !ts.Before(now.Add(-2 * time.Minute))
	}
	return false
}

func (r Row) WorkerFresh(now time.Time) bool {
	if !r.WorkerAlive {
		return false
	}
	return r.workerHeartbeatRecent(now)
}

func (r Row) WorkerTMUXFresh(now time.Time) bool {
	if !r.WorkerFresh(now) {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(r.WorkerDriver), "tmux_cli_session")
}

func (r Row) KnownLegacyCLIWorker(now time.Time) bool {
	if !r.WorkerFresh(now) {
		return false
	}
	if !r.RequiresStructuredTMUX() {
		return false
	}
	if r.WorkerTMUXFresh(now) {
		return false
	}
	for _, driver := range []string{
		strings.TrimSpace(r.WorkerDriver),
		r.handleDriver(),
	} {
		if strings.EqualFold(driver, "process_pty_cli") {
			return true
		}
	}
	return false
}

func (r Row) WorkerSupportsContinuityRecovery(now time.Time) bool {
	if !r.WorkerFresh(now) {
		return false
	}
	if r.WorkerCanSendInput {
		return true
	}
	switch runtimeagente.NormalizeMailboxDeliveryMode(r.WorkerMailboxDeliveryMode) {
	case runtimeagente.MailboxDeliveryInteractive:
		return true
	case runtimeagente.MailboxDeliverySessionResume:
		return strings.TrimSpace(r.WorkerExternalSessionID) != "" || strings.TrimSpace(r.WorkerSessionRef) != ""
	case runtimeagente.MailboxDeliveryBootstrapOnly:
		if !r.WorkerTMUXFresh(now) || strings.TrimSpace(r.WorkerTMUXSession) == "" {
			return false
		}
		switch strings.ToLower(strings.TrimSpace(r.WorkerState)) {
		case "starting", "running", "ready", "working":
			return true
		default:
			return false
		}
	default:
		return false
	}
}

func applyResumePayloadAutonomyToRow(row *Row) {
	if row == nil || row.Sesion == nil || strings.TrimSpace(row.Sesion.ResumePayloadJSON) == "" {
		return
	}
	action, source, moment, state, reason, verificationKey := summarizeResumePayloadAutonomy(row.Sesion)
	if strings.TrimSpace(action) == "" {
		return
	}
	if !mailboxCountersShouldReplace(row.LastAutonomyMoment, moment) {
		return
	}
	row.LastAutonomyAction = action
	row.LastAutonomySource = source
	row.LastAutonomyMoment = moment
	row.LastAutonomyState = state
	row.LastAutonomyReason = reason
	row.LastAutonomyVerificationKey = verificationKey
}

func summarizeResumePayloadAutonomy(sesion *db.Sesion) (action, source string, moment *time.Time, state, reason, verificationKey string) {
	if sesion == nil {
		return "", "", nil, "", "", ""
	}
	envelope := db.ParseResumePayloadEnvelope(strings.TrimSpace(sesion.ResumePayloadJSON))
	items, ok := envelope["mailbox"].([]any)
	if !ok {
		return "", "", nil, "", "", ""
	}
	for i := len(items) - 1; i >= 0; i-- {
		msg, ok := items[i].(map[string]any)
		if !ok {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(stringFromResumePayloadAny(msg["kind"])), "autonomia") {
			continue
		}
		payload, ok := msg["payload"].(map[string]any)
		if !ok {
			continue
		}
		action = strings.TrimSpace(stringFromResumePayloadAny(payload["accion"]))
		if action == "" {
			continue
		}
		source = "resume_payload_mailbox"
		state = "embedded"
		reason = strings.TrimSpace(stringFromResumePayloadAny(payload["motivo"]))
		if reason == "" {
			reason = strings.TrimSpace(stringFromResumePayloadAny(payload["bootstrap_kind"]))
		}
		verificationKey = strings.TrimSpace(stringFromResumePayloadAny(payload["verification_key"]))
		switch {
		case sesion.HeartbeatAt != nil && !sesion.HeartbeatAt.IsZero():
			ts := sesion.HeartbeatAt.UTC()
			moment = &ts
		case !sesion.Inicio.IsZero():
			ts := sesion.Inicio.UTC()
			moment = &ts
		}
		return action, source, moment, state, reason, verificationKey
	}
	return "", "", nil, "", "", ""
}

func stringFromResumePayloadAny(v any) string {
	switch value := v.(type) {
	case string:
		return value
	case fmt.Stringer:
		return value.String()
	case nil:
		return ""
	default:
		return fmt.Sprint(value)
	}
}

func (r Row) durableWorkQueueView(now time.Time) *runtimeagente.WorkerStatusView {
	if r.Handle == nil || strings.TrimSpace(r.Handle.MetadataJSON) == "" {
		return nil
	}
	snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(strings.TrimSpace(r.Handle.MetadataJSON))
	if err != nil || snap == nil {
		return nil
	}
	view := snap.View(now, time.Minute)
	if view == nil || strings.TrimSpace(view.WorkQueueAction) == "" {
		return nil
	}
	return view
}

func (r Row) DurableContinuityPending(now time.Time) bool {
	view := r.durableWorkQueueView(now)
	if view == nil {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(view.WorkQueueKind), "autonomia") {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(view.WorkQueueAction), "continuar_trabajo") {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(view.WorkQueueState)) {
	case "", "pending", "notified", "delivered", "claimed", "working", "running":
		return true
	default:
		return false
	}
}

func (r Row) EffectiveContinuityPending(now time.Time) bool {
	if r.MailboxContinuityPending <= 0 && !r.DurableContinuityPending(now) {
		return false
	}
	source := strings.ToLower(strings.TrimSpace(r.LastAutonomySource))
	state := strings.ToLower(strings.TrimSpace(r.LastAutonomyState))
	if source == "resume_payload_mailbox" && r.WorkerFresh(now) {
		if r.OpenTasks > 0 || r.supervisorAutonomyLive(now) {
			return false
		}
	}
	if source == "work_queue" && r.WorkerFresh(now) {
		switch state {
		case "delivered", "working", "running":
			if r.OpenTasks > 0 || r.supervisorAutonomyLive(now) {
				return false
			}
		}
	}
	if source == "work_queue" && r.autonomyWorkQueueAbsorbedByWorker(now) && (r.OpenTasks > 0 || r.supervisorAutonomyLive(now)) {
		return false
	}
	return true
}

func (r Row) autonomyWorkQueueAbsorbedByWorker(now time.Time) bool {
	if strings.ToLower(strings.TrimSpace(r.LastAutonomySource)) != "work_queue" {
		return false
	}
	moment := r.LastAutonomyMoment
	if moment == nil || moment.IsZero() {
		return false
	}
	for _, ts := range []*time.Time{r.WorkerLastProgress, r.WorkerLastOutput} {
		if ts == nil || ts.IsZero() {
			continue
		}
		if !ts.Before(*moment) && !ts.Before(now.Add(-10*time.Minute)) {
			return true
		}
	}
	return false
}

func (r Row) RequiresStructuredTMUX() bool {
	if strings.EqualFold(strings.TrimSpace(r.WorkerDriver), "tmux_cli_session") {
		return true
	}
	for _, value := range []string{
		func() string {
			if r.Sesion == nil {
				return ""
			}
			return strings.TrimSpace(r.Sesion.Herramienta)
		}(),
		r.handleRenderedCommand(),
		r.handleDriver(),
	} {
		v := strings.ToLower(strings.TrimSpace(value))
		switch {
		case strings.Contains(v, "codex"),
			strings.Contains(v, "claude"),
			strings.Contains(v, "gemini"):
			return true
		}
	}
	return false
}

func (r Row) handleDriver() string {
	if r.Handle == nil || strings.TrimSpace(r.Handle.MetadataJSON) == "" {
		return ""
	}
	var meta map[string]any
	if err := json.Unmarshal([]byte(r.Handle.MetadataJSON), &meta); err != nil {
		return ""
	}
	value, _ := meta["driver"].(string)
	return strings.TrimSpace(value)
}

func (r Row) handleRenderedCommand() string {
	if r.Handle == nil || strings.TrimSpace(r.Handle.MetadataJSON) == "" {
		return ""
	}
	var meta map[string]any
	if err := json.Unmarshal([]byte(r.Handle.MetadataJSON), &meta); err != nil {
		return ""
	}
	for _, key := range []string{"rendered_command", "wrapped_command"} {
		if value, _ := meta[key].(string); strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (r Row) pendingControlOrderBlocksRuntime(now time.Time) (bool, string) {
	if r.ControlOrdersOpen <= 0 {
		return false, ""
	}
	tipo := strings.ToLower(strings.TrimSpace(r.LastControlOrderType))
	switch tipo {
	case "stop", "start", "pause", "resume":
	default:
		return false, ""
	}
	moment := time.Time{}
	if r.LastControlOrderMoment != nil {
		moment = r.LastControlOrderMoment.UTC()
	}
	if moment.IsZero() {
		return true, "orden " + tipo + " pendiente"
	}
	if moment.Before(now.Add(-20 * time.Minute)) {
		return false, ""
	}
	if latest := r.latestWorkerRecoveryMoment(); !latest.IsZero() && latest.After(moment) {
		return false, ""
	}
	return true, "orden " + tipo + " pendiente"
}

func (r Row) latestWorkerRecoveryMoment() time.Time {
	var latest time.Time
	for _, ts := range []time.Time{
		runtimeMoment(r.Runtime),
		runtimeHandleMoment(r.Handle),
		func() time.Time {
			if r.WorkerHeartbeat != nil {
				return r.WorkerHeartbeat.UTC()
			}
			return time.Time{}
		}(),
		func() time.Time {
			if r.WorkerUpdatedAt != nil {
				return r.WorkerUpdatedAt.UTC()
			}
			return time.Time{}
		}(),
	} {
		if !ts.IsZero() && ts.After(latest) {
			latest = ts
		}
	}
	return latest
}

func (r Row) workerHeartbeatStale(now time.Time) bool {
	if r.WorkerHeartbeat == nil || r.WorkerHeartbeat.IsZero() {
		if r.WorkerUpdatedAt == nil || r.WorkerUpdatedAt.IsZero() {
			return false
		}
		return r.WorkerUpdatedAt.Before(now.Add(-2 * time.Minute))
	}
	return r.WorkerHeartbeat.Before(now.Add(-2 * time.Minute))
}

func (r Row) workerOutputStale(now time.Time, threshold time.Duration) bool {
	if threshold <= 0 {
		threshold = 4 * time.Hour
	}
	baseline := r.workerOutputBaseline()
	if baseline == nil || baseline.IsZero() {
		return false
	}
	return baseline.Before(now.Add(-threshold))
}

func (r Row) workerHasRecentWorkSignal(now time.Time, threshold time.Duration) bool {
	if threshold <= 0 {
		threshold = 4 * time.Hour
	}
	for _, ts := range []*time.Time{
		r.WorkerLastProgress,
		r.WorkerLastOutput,
	} {
		if ts == nil || ts.IsZero() {
			continue
		}
		if !ts.Before(now.Add(-threshold)) {
			return true
		}
	}
	return false
}

func (r Row) workerProgressRecent(now time.Time, threshold time.Duration) bool {
	if threshold <= 0 {
		threshold = 4 * time.Hour
	}
	baseline := r.workerProgressBaseline()
	if baseline == nil || baseline.IsZero() {
		return false
	}
	return !baseline.Before(now.Add(-threshold))
}

func (r Row) workerProgressStale(now time.Time, threshold time.Duration) bool {
	if threshold <= 0 {
		threshold = 4 * time.Hour
	}
	baseline := r.workerProgressBaseline()
	if baseline == nil || baseline.IsZero() {
		return false
	}
	return baseline.Before(now.Add(-threshold))
}

func (r Row) runtimeCheckpointQuotaBlocked(now time.Time) (bool, string) {
	cp := r.LastCheckpoint
	if cp == nil {
		return false, ""
	}
	moment := checkpointMoment(cp)
	if moment.IsZero() || moment.Before(now.Add(-20*time.Minute)) {
		return false, ""
	}
	kind := strings.ToLower(strings.TrimSpace(cp.CheckpointKind))
	source := strings.ToLower(strings.TrimSpace(cp.Source))
	resumen := strings.ToLower(strings.TrimSpace(cp.Resumen))
	if kind != "pause" && !strings.Contains(source, ":pause") {
		return false, ""
	}
	if !strings.Contains(resumen, "cuota") &&
		!strings.Contains(resumen, "enfriamiento") &&
		!strings.Contains(resumen, "usage limit") {
		return false, ""
	}
	detalle := strings.TrimSpace(cp.Resumen)
	if detalle == "" {
		detalle = "runtime pausado por cuota"
	}
	return true, detalle
}

func (r Row) workerLastOutputSummary() string {
	baseline := r.workerOutputBaseline()
	if baseline == nil || baseline.IsZero() {
		return ""
	}
	if r.WorkerLastOutput != nil && !r.WorkerLastOutput.IsZero() {
		return "sin salida desde " + r.WorkerLastOutput.Local().Format("15:04")
	}
	return "sin salida desde listo " + baseline.Local().Format("15:04")
}

func (r Row) workerLastProgressSummary() string {
	baseline := r.workerProgressBaseline()
	if baseline == nil || baseline.IsZero() {
		return ""
	}
	if r.WorkerLastProgress != nil && !r.WorkerLastProgress.IsZero() {
		return "sin progreso desde " + r.WorkerLastProgress.Local().Format("15:04")
	}
	return "sin progreso desde listo " + baseline.Local().Format("15:04")
}

func (r Row) workerOutputBaseline() *time.Time {
	if r.WorkerLastOutput != nil && !r.WorkerLastOutput.IsZero() {
		return r.WorkerLastOutput
	}
	if r.WorkerReadyAt != nil && !r.WorkerReadyAt.IsZero() {
		return r.WorkerReadyAt
	}
	return nil
}

func (r Row) workerProgressBaseline() *time.Time {
	if r.WorkerLastProgress != nil && !r.WorkerLastProgress.IsZero() {
		return r.WorkerLastProgress
	}
	if r.WorkerReadyAt != nil && !r.WorkerReadyAt.IsZero() {
		return r.WorkerReadyAt
	}
	return nil
}

func (r Row) runtimePrincipalStale(now time.Time) bool {
	if r.Runtime == nil {
		return false
	}
	moment := runtimeMoment(r.Runtime)
	if moment.IsZero() {
		return false
	}
	if !moment.Before(now.Add(-10 * time.Minute)) {
		return false
	}
	if strings.TrimSpace(r.WorkerState) != "" {
		return !r.WorkerAlive || r.workerHeartbeatStale(now)
	}
	return true
}

func loadStructuredWorkerSnapshot(runtime *db.RuntimeInstance, handle *db.RuntimeHandle) *runtimeagente.WorkerSnapshot {
	var metas []string
	if handle != nil && !estadoHandleRoto(handle.Estado) && !estadoHandlePausado(handle.Estado) {
		metas = append(metas, metadataFromHandle(handle))
	}
	if runtime != nil && !estadoRuntimeRoto(runtime.LogicalState) && !estadoRuntimePausado(runtime.LogicalState) {
		metas = append(metas, metadataFromRuntime(runtime))
	}
	for _, metaJSON := range metas {
		snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(metaJSON)
		if err == nil && snap != nil {
			return snap
		}
	}
	return nil
}

func metadataFromRuntime(runtime *db.RuntimeInstance) string {
	_ = runtime
	return ""
}

func metadataFromHandle(handle *db.RuntimeHandle) string {
	if handle == nil {
		return ""
	}
	return handle.MetadataJSON
}

func sesionMoment(sesion *db.Sesion) time.Time {
	if sesion == nil {
		return time.Time{}
	}
	if sesion.HeartbeatAt != nil && !sesion.HeartbeatAt.IsZero() {
		return *sesion.HeartbeatAt
	}
	if sesion.Fin != nil && !sesion.Fin.IsZero() {
		return *sesion.Fin
	}
	return sesion.Inicio
}

func checkpointMoment(cp *db.RuntimeCheckpoint) time.Time {
	if cp == nil {
		return time.Time{}
	}
	return cp.CreatedAt
}

func runtimeOrderMoment(order *db.RuntimeOrder) time.Time {
	if order == nil {
		return time.Time{}
	}
	for _, ts := range []time.Time{order.UpdatedAt, order.AvailableAt, order.CreatedAt} {
		if !ts.IsZero() {
			return ts
		}
	}
	return time.Time{}
}

func runtimeOrderEsControl(tipo string) bool {
	switch strings.ToLower(strings.TrimSpace(tipo)) {
	case "start", "stop", "pause", "resume":
		return true
	default:
		return false
	}
}

func summarizeOrdersForRow(row Row, orders []*db.RuntimeOrder) (open int, failed int, controlOpen int, lastControlType string, lastControlMoment *time.Time) {
	for _, order := range orders {
		if order == nil {
			continue
		}
		switch strings.TrimSpace(order.Estado) {
		case "pendiente", "tomada", "ejecutando":
			open++
			if runtimeOrderEsControl(strings.TrimSpace(order.Tipo)) && runtimeOrderMatchesRowProject(row, order) {
				controlOpen++
				moment := runtimeOrderMoment(order)
				if lastControlMoment == nil || moment.After(*lastControlMoment) {
					ts := moment
					lastControlMoment = &ts
					lastControlType = strings.TrimSpace(order.Tipo)
				}
			}
		case "fallida":
			failed++
		}
		if strings.TrimSpace(order.ErrorText) != "" {
			failed++
		}
	}
	return open, failed, controlOpen, lastControlType, lastControlMoment
}

func summarizeAutonomyOrdersForRow(orders []*db.RuntimeOrder) (lastAutonomyAction string, lastAutonomySource string, lastAutonomyMoment *time.Time, lastAutonomyState string, lastAutonomyReason string, lastAutonomyVerificationKey string, lastAutonomyDispatchState string, lastAutonomyDeliveryState string, lastAutonomyReceiptSource string) {
	for _, order := range orders {
		if action, source, moment, state, reason, verificationKey, dispatchState, deliveryState, receiptSource := autonomyOrderActionSummary(order); strings.TrimSpace(action) != "" {
			if mailboxCountersShouldReplace(lastAutonomyMoment, moment) {
				lastAutonomyAction = action
				lastAutonomySource = source
				lastAutonomyMoment = moment
				lastAutonomyState = state
				lastAutonomyReason = reason
				lastAutonomyVerificationKey = verificationKey
				lastAutonomyDispatchState = dispatchState
				lastAutonomyDeliveryState = deliveryState
				lastAutonomyReceiptSource = receiptSource
			}
		}
	}
	return lastAutonomyAction, lastAutonomySource, lastAutonomyMoment, lastAutonomyState, lastAutonomyReason, lastAutonomyVerificationKey, lastAutonomyDispatchState, lastAutonomyDeliveryState, lastAutonomyReceiptSource
}

func mailboxCountersShouldReplace(current *time.Time, candidate *time.Time) bool {
	if candidate == nil || candidate.IsZero() {
		return current == nil || current.IsZero()
	}
	if current == nil || current.IsZero() {
		return true
	}
	return candidate.After(*current)
}

func autonomyMailboxActionSummary(msg *db.RuntimeMailboxMessage, agente string) (string, string, *time.Time) {
	if msg == nil || !strings.EqualFold(strings.TrimSpace(msg.Kind), "autonomia") {
		return "", "", nil
	}
	if agente = strings.TrimSpace(agente); agente != "" && !strings.EqualFold(strings.TrimSpace(msg.ToAgente), agente) {
		return "", "", nil
	}
	payload := runtimeMailboxPayloadMap(msg.PayloadJSON)
	action := strings.TrimSpace(stringFromRuntimeMailboxPayload(payload, "accion"))
	if action == "" {
		return "", "", nil
	}
	ts := msg.CreatedAt.UTC()
	return action, "mailbox", &ts
}

func autonomyOrderActionSummary(order *db.RuntimeOrder) (string, string, *time.Time, string, string, string, string, string, string) {
	if order == nil {
		return "", "", nil, "", "", "", "", "", ""
	}
	payload := runtimeMailboxPayloadMap(order.PayloadJSON)
	if !strings.EqualFold(strings.TrimSpace(stringFromRuntimeMailboxPayload(payload, "kind")), "autonomia") {
		return "", "", nil, "", "", "", "", "", ""
	}
	action := strings.TrimSpace(stringFromRuntimeMailboxPayload(payload, "accion"))
	if action == "" {
		return "", "", nil, "", "", "", "", "", ""
	}
	ts := runtimeOrderMoment(order)
	source := strings.TrimSpace(order.Tipo)
	if source == "" {
		source = "runtime_order"
	}
	result := runtimeMailboxPayloadMap(order.ResultadoJSON)
	dispatchState := strings.TrimSpace(stringFromRuntimeMailboxPayload(result, "dispatch_state"))
	deliveryState := strings.TrimSpace(stringFromRuntimeMailboxPayload(result, "delivery_state"))
	receiptSource := strings.TrimSpace(stringFromRuntimeMailboxPayload(result, "receipt_source"))
	state := autonomyOrderObservedState(strings.TrimSpace(order.Estado), dispatchState, deliveryState, receiptSource)
	reason := strings.TrimSpace(stringFromRuntimeMailboxPayload(payload, "motivo"))
	if reason == "" {
		reason = strings.TrimSpace(stringFromRuntimeMailboxPayload(result, "deferred_reason"))
	}
	return action,
		source,
		&ts,
		state,
		reason,
		strings.TrimSpace(stringFromRuntimeMailboxPayload(payload, "verification_key")),
		dispatchState,
		deliveryState,
		receiptSource
}

func autonomyOrderObservedState(orderState, dispatchState, deliveryState, receiptSource string) string {
	orderState = strings.TrimSpace(orderState)
	dispatchState = strings.ToLower(strings.TrimSpace(dispatchState))
	deliveryState = strings.ToLower(strings.TrimSpace(deliveryState))
	receiptSource = strings.ToLower(strings.TrimSpace(receiptSource))
	switch {
	case dispatchState == "delivered" && deliveryState == "delivered" && autonomyReceiptSourceConfirmsWork(receiptSource):
		return "work_confirmed"
	case dispatchState == "delivered" && deliveryState == "delivered":
		return "awaiting_work"
	default:
		return orderState
	}
}

func autonomyReceiptSourceConfirmsWork(source string) bool {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "last_progress",
		"worker_activity",
		"tmux_transcript_activity",
		"transcript_patch",
		"tmux_pane_patch",
		"git_worktree":
		return true
	default:
		return false
	}
}

func runtimeOrderMatchesRowProject(row Row, order *db.RuntimeOrder) bool {
	if order == nil || !runtimeOrderEsControl(strings.TrimSpace(order.Tipo)) {
		return true
	}
	rowProjectID := row.projectID()
	if rowProjectID == nil || order.ProyectoID == nil {
		return true
	}
	return *rowProjectID == *order.ProyectoID
}

func (r Row) projectID() *int64 {
	if r.Runtime != nil && r.Runtime.ProyectoID != nil && *r.Runtime.ProyectoID > 0 {
		return r.Runtime.ProyectoID
	}
	if r.Handle != nil && r.Handle.ProyectoID != nil && *r.Handle.ProyectoID > 0 {
		return r.Handle.ProyectoID
	}
	if r.Sesion != nil && r.Sesion.ProyectoID != nil && *r.Sesion.ProyectoID > 0 {
		return r.Sesion.ProyectoID
	}
	if r.Asignacion != nil && r.Asignacion.ProyectoID > 0 {
		id := r.Asignacion.ProyectoID
		return &id
	}
	return nil
}

func estadoCuotaBloqueado(estado string) bool {
	estado = strings.ToLower(strings.TrimSpace(estado))
	return estado == "agotado" || estado == "enfriamiento"
}

func estadoRuntimeRoto(estado string) bool {
	switch strings.ToLower(strings.TrimSpace(estado)) {
	case "fallido", "degradado", "cerrado", "finalizado":
		return true
	default:
		return false
	}
}

func estadoHandleRoto(estado string) bool {
	switch strings.ToLower(strings.TrimSpace(estado)) {
	case "fallido", "cerrado":
		return true
	default:
		return false
	}
}

func estadoRuntimePausado(estado string) bool {
	switch strings.ToLower(strings.TrimSpace(estado)) {
	case "pausado", "stopped", "stop", "paused":
		return true
	default:
		return false
	}
}

func estadoHandlePausado(estado string) bool {
	return strings.EqualFold(strings.TrimSpace(estado), "pausado")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func valueOrFallback(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
