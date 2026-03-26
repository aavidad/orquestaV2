package agentesapp

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"orquesta/db"
)

type Store interface {
	RegisterAgent(nombre, rol string) error
	RegisterAgentAuto(proveedor, rol string) (string, error)
	RetireAgent(nombre string) error
	RehabilitateAgent(nombre string) error
	ResetReanimation(nombre string) error
	DeleteAgent(nombre string) error
	MergeAgents(origen, destino string) (*db.FusionAgentesResultado, error)
	Audit(agente, accion, entidad string, entidadID int64, detalle string)
	GetAgent(nombre string) (*db.Agente, error)
	GetProject(ref string) (*db.Proyecto, error)
	GetConnector(ref string) (*db.Conector, error)
	ListAgents() ([]*db.Agente, error)
	ListAssignments(filtro db.FiltroAsignaciones) ([]*db.Asignacion, error)
	ListInspectionSessions(filtro db.FiltroSesionesInspeccion) ([]*db.Sesion, error)
	GetLastSession(agente string, proyectoID *int64) (*db.Sesion, error)
	GetActiveSession(agente string, proyectoID *int64) (*db.Sesion, error)
	SaveActiveSession(agente string, proyectoID *int64, upd db.SesionUpdate) error
	ListRuntimes(filtro db.FiltroRuntimes) ([]*db.RuntimeInstance, error)
	ListRuntimeHandles(agente *string) ([]*db.RuntimeHandle, error)
	ListRuntimeOrders(filtro db.FiltroRuntimeOrders) ([]*db.RuntimeOrder, error)
	ListRuntimeMailbox(filtro db.FiltroRuntimeMailbox) ([]*db.RuntimeMailboxMessage, error)
	ListRuntimeCheckpoints(filtro db.FiltroRuntimeCheckpoints) ([]*db.RuntimeCheckpoint, error)
	ListTasks(filtro db.FiltroTareas) ([]*db.Tarea, error)
	ListProjectPendingVotes(agente string, proyectoID int64) ([]*db.Propuesta, error)
	ListProjectOpenProposals(proyectoID int64) ([]*db.Propuesta, error)
	ListRules(rol string) ([]*db.Regla, error)
	ListSkills(rol string) ([]*db.Skill, error)
	ListWorkflows(rol string) ([]*db.Workflow, error)
	ListMemoryEntities(filtro db.FiltroEntidadesMemoria) ([]*db.EntidadMemoria, error)
	ConfigGet(clave string) (string, error)
	PauseAgent(nombre string, minutos int, motivo string) error
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

type Row struct {
	Agente         *db.Agente
	Asignacion     *db.Asignacion
	Sesion         *db.Sesion
	Runtime        *db.RuntimeInstance
	Handle         *db.RuntimeHandle
	OrdersOpen     int
	OrdersFailed   int
	MailboxPending int
	MailboxTotal   int
	Checkpoints    int
	LastCheckpoint *db.RuntimeCheckpoint
	OpenTasks      int
}

type Detail struct {
	Row          Row
	Asignaciones []*db.Asignacion
	Sesiones     []*db.Sesion
	Runtimes     []*db.RuntimeInstance
	Handles      []*db.RuntimeHandle
	Orders       []*db.RuntimeOrder
	Mailbox      []*db.RuntimeMailboxMessage
	Checkpoints  []*db.RuntimeCheckpoint
}

func (s *Service) RegisterAgent(nombre, rol string) error {
	return s.store.RegisterAgent(strings.TrimSpace(nombre), strings.TrimSpace(rol))
}

func (s *Service) RegisterAgentAuto(proveedor, rol string) (string, error) {
	return s.store.RegisterAgentAuto(strings.TrimSpace(proveedor), strings.TrimSpace(rol))
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
		return s.store.RetireAgent(nombre)
	case "rehabilitar":
		return s.store.RehabilitateAgent(nombre)
	case "reset-reanimacion":
		return s.store.ResetReanimation(nombre)
	case "eliminar":
		if err := s.store.DeleteAgent(nombre); err != nil {
			return err
		}
		s.store.Audit("alberto", "purgar_agente", "agente", 0, nombre)
		return nil
	default:
		return fmt.Errorf("acción de agente no soportada: %s", accion)
	}
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
	handles, err := s.store.ListRuntimeHandles(nil)
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
	for _, handle := range handles {
		if handle == nil {
			continue
		}
		actual := handlePorAgente[handle.Agente]
		if actual == nil || runtimeHandleMoment(handle).After(runtimeHandleMoment(actual)) {
			handlePorAgente[handle.Agente] = handle
		}
	}

	type orderCounters struct {
		Open   int
		Failed int
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
	for _, tarea := range tareas {
		if tarea == nil || tarea.Agente == nil {
			continue
		}
		switch tarea.Estado {
		case db.EstadoCompletada, db.EstadoCancelada:
			continue
		}
		openTasksPorAgente[*tarea.Agente]++
	}

	rows := make([]Row, 0, len(agentes))
	for _, agente := range agentes {
		if agente == nil {
			continue
		}
		orderStats := ordersPorAgente[agente.Nombre]
		mailboxStats := mailboxPorAgente[agente.Nombre]
		rows = append(rows, Row{
			Agente:         agente,
			Asignacion:     asignacionPorAgente[agente.Nombre],
			Sesion:         sesionPorAgente[agente.Nombre],
			Runtime:        runtimePorAgente[agente.Nombre],
			Handle:         handlePorAgente[agente.Nombre],
			OrdersOpen:     orderStats.Open,
			OrdersFailed:   orderStats.Failed,
			MailboxPending: mailboxStats.Pending,
			MailboxTotal:   mailboxStats.Total,
			Checkpoints:    checkpointTotalPorAgente[agente.Nombre],
			LastCheckpoint: lastCheckpointPorAgente[agente.Nombre],
			OpenTasks:      openTasksPorAgente[agente.Nombre],
		})
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
	mailbox, err := s.listMailboxForAgent(nombre)
	if err != nil {
		return nil, err
	}
	checkpoints, err := s.store.ListRuntimeCheckpoints(db.FiltroRuntimeCheckpoints{Agente: &nombre, Limit: 20})
	if err != nil {
		return nil, err
	}

	return &Detail{
		Row:          row,
		Asignaciones: asignaciones,
		Sesiones:     sesiones,
		Runtimes:     runtimes,
		Handles:      handles,
		Orders:       orders,
		Mailbox:      mailbox,
		Checkpoints:  checkpoints,
	}, nil
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

func (Repository) GetConnector(ref string) (*db.Conector, error) {
	return db.GetConector(ref)
}

func (Repository) ListAgents() ([]*db.Agente, error) {
	return db.ListarAgentes()
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

func (Repository) ListRuntimeOrders(filtro db.FiltroRuntimeOrders) ([]*db.RuntimeOrder, error) {
	return db.ListarRuntimeOrders(filtro)
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
