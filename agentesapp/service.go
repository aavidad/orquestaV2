package agentesapp

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
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
	ListCanonicalRuntimeHandles(agente *string) ([]*db.RuntimeHandle, error)
	ListRecentOperationalRuntimeHandles() (map[string]*db.RuntimeHandle, error)
	ListRuntimeTranscript(filtro db.FiltroRuntimeTranscript) ([]*db.RuntimeTranscriptEntry, error)
	ListRuntimeOrders(filtro db.FiltroRuntimeOrders) ([]*db.RuntimeOrder, error)
	EnqueueRuntimeOrder(order *db.RuntimeOrder) (int64, error)
	ListRuntimeMailbox(filtro db.FiltroRuntimeMailbox) ([]*db.RuntimeMailboxMessage, error)
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
	ConfigGet(clave string) (string, error)
	PauseAgent(nombre string, minutos int, motivo string) error
}

type ModelPolicyProvider interface {
	ResolveModelPolicy(input db.ResolverPoliticaInput) (*db.ResolucionModelo, error)
}

type Service struct {
	store               Store
	modelPolicyProvider ModelPolicyProvider
}

const defaultWorkerOutputStaleSeconds = 20 * 60

func NewService(store Store, modelPolicyProvider ModelPolicyProvider) *Service {
	return &Service{store: store, modelPolicyProvider: modelPolicyProvider}
}

type Row struct {
	Agente                    *db.Agente
	Asignacion                *db.Asignacion
	Sesion                    *db.Sesion
	Runtime                   *db.RuntimeInstance
	Handle                    *db.RuntimeHandle
	WorkerState               string
	WorkerAlive               bool
	WorkerReadyAt             *time.Time
	WorkerHeartbeat           *time.Time
	WorkerUpdatedAt           *time.Time
	WorkerLastOutput          *time.Time
	WorkerLastProgress        *time.Time
	WorkerExitError           string
	WorkerSessionRef          string
	WorkerRuntimeRef          string
	WorkerDriver              string
	WorkerTransport           string
	WorkerTMUXSession         string
	WorkerTMUXWindow          string
	WorkerTMUXPaneID          string
	WorkerCanSendInput        bool
	WorkerExternalSessionID   string
	WorkerMailboxDeliveryMode string
	EstadoOperativo           string
	DetalleOperativo          string
	OrdersOpen                int
	OrdersFailed              int
	ControlOrdersOpen         int
	LastControlOrderType      string
	LastControlOrderMoment    *time.Time
	MailboxPending            int
	MailboxTotal              int
	Checkpoints               int
	LastCheckpoint            *db.RuntimeCheckpoint
	OpenTasks                 int
	BlockedTasks              int
}

type Detail struct {
	Row          Row                          `json:"row"`
	Entity       *AgentEntity                 `json:"entity,omitempty"`
	Asignaciones []*db.Asignacion             `json:"asignaciones,omitempty"`
	Sesiones     []*db.Sesion                 `json:"sesiones,omitempty"`
	Runtimes     []*db.RuntimeInstance        `json:"runtimes,omitempty"`
	Handles      []*db.RuntimeHandle          `json:"handles,omitempty"`
	Transcript   []*db.RuntimeTranscriptEntry `json:"transcript,omitempty"`
	Orders       []*db.RuntimeOrder           `json:"orders,omitempty"`
	Mailbox      []*db.RuntimeMailboxMessage  `json:"mailbox,omitempty"`
	Checkpoints  []*db.RuntimeCheckpoint      `json:"checkpoints,omitempty"`
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
	Name                string      `json:"name"`
	Role                string      `json:"role,omitempty"`
	Enabled             bool        `json:"enabled"`
	ActiveNow           bool        `json:"active_now"`
	AccountID           string      `json:"account_id,omitempty"`
	AccountEmail        string      `json:"account_email,omitempty"`
	AccountUser         string      `json:"account_user,omitempty"`
	AssignmentProject   string      `json:"assignment_project,omitempty"`
	RuntimeAdapter      string      `json:"runtime_adapter,omitempty"`
	Transport           string      `json:"transport,omitempty"`
	RuntimeState        string      `json:"runtime_state,omitempty"`
	HandleState         string      `json:"handle_state,omitempty"`
	WorkerState         string      `json:"worker_state,omitempty"`
	WorkerAlive         bool        `json:"worker_alive"`
	WorkerDriver        string      `json:"worker_driver,omitempty"`
	WorkerTransport     string      `json:"worker_transport,omitempty"`
	WorkerSessionRef    string      `json:"worker_session_ref,omitempty"`
	WorkerRuntimeRef    string      `json:"worker_runtime_ref,omitempty"`
	ExternalSessionID   string      `json:"external_session_id,omitempty"`
	MailboxDeliveryMode string      `json:"mailbox_delivery_mode,omitempty"`
	OperationalState    string      `json:"operational_state,omitempty"`
	OperationalDetail   string      `json:"operational_detail,omitempty"`
	Leases              []WorkLease `json:"leases,omitempty"`
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

	type orderCounters struct {
		Open              int
		Failed            int
		ControlOpen       int
		LastControlType   string
		LastControlMoment *time.Time
	}
	ordersPorAgente := map[string]orderCounters{}
	for _, order := range orders {
		if order == nil {
			continue
		}
		stats := ordersPorAgente[order.Agente]
		switch strings.TrimSpace(order.Estado) {
		case "pendiente", "tomada", "ejecutando":
			stats.Open++
			if runtimeOrderEsControl(strings.TrimSpace(order.Tipo)) {
				stats.ControlOpen++
				moment := runtimeOrderMoment(order)
				if stats.LastControlMoment == nil || moment.After(*stats.LastControlMoment) {
					ts := moment
					stats.LastControlMoment = &ts
					stats.LastControlType = strings.TrimSpace(order.Tipo)
				}
			}
		case "fallida":
			stats.Failed++
		}
		if strings.TrimSpace(order.ErrorText) != "" {
			stats.Failed++
		}
		ordersPorAgente[order.Agente] = stats
	}

	type mailboxCounters struct {
		Pending int
		Total   int
	}
	mailboxPorAgente := map[string]mailboxCounters{}
	for _, msg := range mailbox {
		if msg == nil {
			continue
		}
		for _, agente := range []string{msg.ToAgente, msg.FromAgente} {
			if strings.TrimSpace(agente) == "" {
				continue
			}
			stats := mailboxPorAgente[agente]
			stats.Total++
			if strings.TrimSpace(msg.Estado) == "pendiente" {
				stats.Pending++
			}
			mailboxPorAgente[agente] = stats
		}
	}

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
	for _, tarea := range tareas {
		if tarea == nil || tarea.Agente == nil {
			continue
		}
		switch tarea.Estado {
		case db.EstadoCompletada, db.EstadoCancelada:
			continue
		case db.EstadoBloqueada:
			blockedTasksPorAgente[*tarea.Agente]++
			continue
		}
		openTasksPorAgente[*tarea.Agente]++
	}

	now := time.Now().UTC()
	staleThreshold := workerOutputStaleThreshold(s.store)
	rows := make([]Row, 0, len(agentes))
	for _, agente := range agentes {
		if agente == nil {
			continue
		}
		orderStats := ordersPorAgente[agente.Nombre]
		mailboxStats := mailboxPorAgente[agente.Nombre]
		row := Row{
			Agente:                 agente,
			Asignacion:             asignacionPorAgente[agente.Nombre],
			Sesion:                 sesionPorAgente[agente.Nombre],
			Runtime:                runtimePorAgente[agente.Nombre],
			Handle:                 handlePorAgente[agente.Nombre],
			OrdersOpen:             orderStats.Open,
			OrdersFailed:           orderStats.Failed,
			ControlOrdersOpen:      orderStats.ControlOpen,
			LastControlOrderType:   strings.TrimSpace(orderStats.LastControlType),
			LastControlOrderMoment: orderStats.LastControlMoment,
			MailboxPending:         mailboxStats.Pending,
			MailboxTotal:           mailboxStats.Total,
			Checkpoints:            checkpointTotalPorAgente[agente.Nombre],
			LastCheckpoint:         lastCheckpointPorAgente[agente.Nombre],
			OpenTasks:              openTasksPorAgente[agente.Nombre],
			BlockedTasks:           blockedTasksPorAgente[agente.Nombre],
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
		row.EstadoOperativo, row.DetalleOperativo = deriveOperationalState(row, now, staleThreshold)
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].Agente.Nombre) < strings.ToLower(rows[j].Agente.Nombre)
	})
	return rows, nil
}

func (s *Service) BuildDetail(nombre string) (*Detail, error) {
	nombre = strings.TrimSpace(nombre)
	agente, err := s.store.GetAgent(nombre)
	if err != nil {
		return nil, err
	}
	rows, err := s.BuildPanelRows()
	if err != nil {
		return nil, err
	}
	row := Row{Agente: agente}
	for _, item := range rows {
		if item.Agente != nil && item.Agente.Nombre == nombre {
			row = item
			break
		}
	}

	asignaciones, err := s.store.ListAssignments(db.FiltroAsignaciones{Agente: &nombre})
	if err != nil {
		return nil, err
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
	mailbox, err := s.listMailboxForAgent(nombre)
	if err != nil {
		return nil, err
	}
	checkpoints, err := s.store.ListRuntimeCheckpoints(db.FiltroRuntimeCheckpoints{Agente: &nombre, Limit: 20})
	if err != nil {
		return nil, err
	}
	tareas, err := s.store.ListTasks(db.FiltroTareas{Agente: &nombre})
	if err != nil {
		return nil, err
	}

	return &Detail{
		Row:          row,
		Entity:       buildAgentEntity(row, tareas),
		Asignaciones: asignaciones,
		Sesiones:     sesiones,
		Runtimes:     runtimes,
		Handles:      handles,
		Transcript:   transcript,
		Orders:       orders,
		Mailbox:      mailbox,
		Checkpoints:  checkpoints,
	}, nil
}

func (s *Service) BuildReanimationSchedule(includeFuture, activeOnly bool) ([]ReanimationCandidate, error) {
	rows, err := s.BuildPanelRows()
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

	rowByAgent := map[string]Row{}
	for _, row := range rows {
		if row.Agente == nil {
			continue
		}
		rowByAgent[strings.TrimSpace(row.Agente.Nombre)] = row
	}
	leasesByAgent := map[string][]WorkLease{}
	for _, tarea := range tareas {
		if tarea == nil || tarea.Agente == nil {
			continue
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
	out := make([]ReanimationCandidate, 0, len(rows)+len(dueAgents))
	candidates := map[string]ReanimationCandidate{}
	for _, row := range rows {
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
			row = Row{Agente: agente}
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

func buildAgentEntity(row Row, tareas []*db.Tarea) *AgentEntity {
	if row.Agente == nil {
		return nil
	}
	entity := &AgentEntity{
		Name:                strings.TrimSpace(row.Agente.Nombre),
		Role:                strings.TrimSpace(row.Agente.Rol),
		Enabled:             row.Agente.Habilitado,
		ActiveNow:           row.Agente.Activo,
		AccountID:           strings.TrimSpace(row.Agente.CuentaID),
		AccountEmail:        strings.TrimSpace(row.Agente.CuentaEmail),
		AccountUser:         strings.TrimSpace(row.Agente.CuentaUsuario),
		RuntimeAdapter:      strings.TrimSpace(runtimeAdapterName(row)),
		Transport:           strings.TrimSpace(handleTransportName(row.Handle)),
		RuntimeState:        strings.TrimSpace(row.runtimeState()),
		HandleState:         strings.TrimSpace(row.handleState()),
		WorkerState:         strings.TrimSpace(row.WorkerState),
		WorkerAlive:         row.WorkerAlive,
		WorkerDriver:        strings.TrimSpace(row.WorkerDriver),
		WorkerTransport:     strings.TrimSpace(row.WorkerTransport),
		WorkerSessionRef:    strings.TrimSpace(row.WorkerSessionRef),
		WorkerRuntimeRef:    strings.TrimSpace(row.WorkerRuntimeRef),
		ExternalSessionID:   strings.TrimSpace(row.WorkerExternalSessionID),
		MailboxDeliveryMode: strings.TrimSpace(row.WorkerMailboxDeliveryMode),
		OperationalState:    strings.TrimSpace(row.EstadoOperativo),
		OperationalDetail:   strings.TrimSpace(row.DetalleOperativo),
		Leases:              buildWorkLeases(tareas),
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

	if !sinCuotaProveedor && estadoCuotaBloqueado(agente.EstadoCuota) {
		detalle := strings.TrimSpace(agente.MotivoPausa)
		if detalle == "" && agente.ReanimarAt != nil && !agente.ReanimarAt.IsZero() {
			detalle = "reanimacion " + agente.ReanimarAt.Local().Format("15:04")
		}
		return "bloqueado_por_cuota", detalle
	}
	if !sinCuotaProveedor {
		if blocked, detalle := row.runtimeCheckpointQuotaBlocked(now); blocked {
			return "bloqueado_por_cuota", detalle
		}
	}
	if blocked, detalle := row.pendingControlOrderBlocksRuntime(now); blocked {
		return "bloqueado_por_runtime", detalle
	}

	handleEstado := strings.ToLower(strings.TrimSpace(row.handleState()))
	runtimeEstado := strings.ToLower(strings.TrimSpace(row.runtimeState()))
	workerState := strings.ToLower(strings.TrimSpace(row.WorkerState))
	recentActivity := row.hasRecentOperationalActivity(now)
	hasActiveTask := row.OpenTasks > 0
	hasBlockedTask := row.BlockedTasks > 0
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
	if hasActiveTask && workerRunningFresh && row.workerProgressStale(now, workerOutputStaleThreshold) {
		return "atascado", firstNonEmpty(row.workerLastProgressSummary(), row.workerLastOutputSummary(), "worker sin progreso reciente")
	}
	if hasActiveTask && workerRunningFresh && row.workerOutputStale(now, workerOutputStaleThreshold) && !row.workerProgressRecent(now, workerOutputStaleThreshold) {
		return "atascado", firstNonEmpty(row.workerLastOutputSummary(), "worker sin salida reciente")
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
	if hasActiveTask && (estadoRuntimeRoto(runtimeEstado) || estadoHandleRoto(handleEstado)) {
		return "bloqueado_por_runtime", firstNonEmpty(row.runtimeState(), row.handleState())
	}
	if hasActiveTask && hasBlockedTask && recentActivity {
		return "saturado", fmt.Sprintf("%d activas, %d bloqueadas", row.OpenTasks, row.BlockedTasks)
	}
	if hasActiveTask && workerRunningFresh {
		return "trabajando", firstNonEmpty(row.activitySummary(), "worker "+workerState)
	}
	if hasActiveTask && recentActivity {
		return "trabajando", row.activitySummary()
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
	for _, metaJSON := range []string{
		metadataFromHandle(handle),
		metadataFromRuntime(runtime),
	} {
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

type Repository struct{}

func (Repository) RegisterAgent(nombre, rol string) error {
	return db.RegistrarAgente(nombre, rol)
}

func (Repository) RegisterAgentAuto(proveedor, rol string) (string, error) {
	return db.RegistrarAgenteAuto(proveedor, rol)
}

func (Repository) ObserveAgentIdentity(nombre, email, usuario, fuente string, observedAt *time.Time) error {
	return db.UpsertAgenteIdentidadObservada(nombre, email, usuario, fuente, observedAt)
}

func (Repository) RetireAgent(nombre string) error {
	return db.RetirarAgente(nombre)
}

func (Repository) RehabilitateAgent(nombre string) error {
	return db.RehabilitarAgente(nombre)
}

func (Repository) ResetReanimation(nombre string) error {
	return db.ResetReanimacion(nombre)
}

func (Repository) DeleteAgent(nombre string) error {
	return db.EliminarAgente(nombre)
}

func (Repository) MergeAgents(origen, destino string) (*db.FusionAgentesResultado, error) {
	return db.FusionarAgentes(origen, destino)
}

func (Repository) Audit(agente, accion, entidad string, entidadID int64, detalle string) {
	db.Audit(agente, accion, entidad, entidadID, detalle)
}

func (Repository) GetAgent(nombre string) (*db.Agente, error) {
	return db.GetAgente(nombre)
}

func (Repository) GetProject(ref string) (*db.Proyecto, error) {
	return db.GetProyecto(ref)
}

func (Repository) GetPool(slug string) (*db.PoolCapacidad, error) {
	return db.GetPool(slug)
}

func (Repository) GetConnector(ref string) (*db.Conector, error) {
	return db.GetConector(ref)
}

func (Repository) ListAgents() ([]*db.Agente, error) {
	return db.ListarAgentes()
}

func (Repository) CheckReanimations() ([]*db.Agente, error) {
	return db.CheckReanimaciones()
}

func (Repository) ListAssignments(filtro db.FiltroAsignaciones) ([]*db.Asignacion, error) {
	return db.ListarAsignaciones(filtro)
}

func (Repository) ListInspectionSessions(filtro db.FiltroSesionesInspeccion) ([]*db.Sesion, error) {
	return db.ListarSesionesInspeccion(filtro)
}

func (Repository) GetLastSession(agente string, proyectoID *int64) (*db.Sesion, error) {
	return db.ObtenerUltimaSesion(agente, proyectoID)
}

func (Repository) GetActiveSession(agente string, proyectoID *int64) (*db.Sesion, error) {
	return db.GetSesionActiva(agente, proyectoID)
}

func (Repository) SaveActiveSession(agente string, proyectoID *int64, upd db.SesionUpdate) error {
	return db.GuardarSesionActiva(agente, proyectoID, upd)
}

func (Repository) ListRuntimes(filtro db.FiltroRuntimes) ([]*db.RuntimeInstance, error) {
	return db.ListarRuntimes(filtro)
}

func (Repository) ListRuntimeHandles(agente *string) ([]*db.RuntimeHandle, error) {
	return db.ListarRuntimeHandles(agente)
}

func (Repository) ListCanonicalRuntimeHandles(agente *string) ([]*db.RuntimeHandle, error) {
	return db.ListarRuntimeHandlesCanonicosRecientes(agente)
}

func (Repository) ListRecentOperationalRuntimeHandles() (map[string]*db.RuntimeHandle, error) {
	return db.ListarRuntimeHandlesActivosOperativosRecientes()
}

func (Repository) ListRuntimeTranscript(filtro db.FiltroRuntimeTranscript) ([]*db.RuntimeTranscriptEntry, error) {
	return db.ListarRuntimeTranscript(filtro)
}

func (Repository) ListRuntimeOrders(filtro db.FiltroRuntimeOrders) ([]*db.RuntimeOrder, error) {
	return db.ListarRuntimeOrders(filtro)
}

func (Repository) EnqueueRuntimeOrder(order *db.RuntimeOrder) (int64, error) {
	return db.EncolarRuntimeOrder(order)
}

func (Repository) ListRuntimeMailbox(filtro db.FiltroRuntimeMailbox) ([]*db.RuntimeMailboxMessage, error) {
	return db.ListarRuntimeMailbox(filtro)
}

func (Repository) ListRuntimeCheckpoints(filtro db.FiltroRuntimeCheckpoints) ([]*db.RuntimeCheckpoint, error) {
	return db.ListarRuntimeCheckpoints(filtro)
}

func (Repository) ListTasks(filtro db.FiltroTareas) ([]*db.Tarea, error) {
	return db.ListarTareas(filtro)
}

func (Repository) ListProjectPendingVotes(agente string, proyectoID int64) ([]*db.Propuesta, error) {
	return db.PropuestasPendientesVotoProyecto(agente, &proyectoID)
}

func (Repository) ListProjectOpenProposals(proyectoID int64) ([]*db.Propuesta, error) {
	estado := db.PropuestaAbierta
	return db.ListarPropuestas(&estado, &proyectoID)
}

func (Repository) GetLatestAgentBudget(agente string) (*db.PresupuestoSesion, *db.Sesion, error) {
	return db.UltimoPresupuestoAgente(agente)
}

func (Repository) ResolveGovernanceCatalog(rol string, proyectoID *int64) (*db.GovernanceCatalog, error) {
	return db.ResolveGovernanceCatalog(rol, proyectoID)
}

func (Repository) ResolveGovernanceCatalogForContext(rol string, proyectoID *int64, agente string) (*db.GovernanceCatalog, error) {
	return db.ResolveGovernanceCatalogForContext(rol, proyectoID, agente)
}

func (Repository) ListRules(rol string) ([]*db.Regla, error) {
	return db.GetReglasAgente(rol)
}

func (Repository) ListSkills(rol string) ([]*db.Skill, error) {
	return db.GetSkillsAgente(rol)
}

func (Repository) ListWorkflows(rol string) ([]*db.Workflow, error) {
	return db.GetWorkflowsAgente(rol)
}

func (Repository) ListMemoryEntities(filtro db.FiltroEntidadesMemoria) ([]*db.EntidadMemoria, error) {
	return db.ListarEntidadesMemoria(filtro)
}

func (Repository) ConfigGet(clave string) (string, error) {
	return db.ConfigGet(clave)
}

func (Repository) PauseAgent(nombre string, minutos int, motivo string) error {
	return db.PausarAgente(nombre, minutos, motivo)
}
