package agentesapp

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"orquesta/db"
)

type fakeStore struct {
	agents            []*db.Agente
	project           *db.Proyecto
	connector         *db.Conector
	assignments       []*db.Asignacion
	sessions          []*db.Sesion
	runtimes          []*db.RuntimeInstance
	handles           []*db.RuntimeHandle
	orders            []*db.RuntimeOrder
	mailbox           []*db.RuntimeMailboxMessage
	checkpoints       []*db.RuntimeCheckpoint
	tasks             []*db.Tarea
	proposals         []*db.Propuesta
	rules             []*db.Regla
	skills            []*db.Skill
	workflows         []*db.Workflow
	memory            []*db.EntidadMemoria
	config            map[string]string
	lastMailboxFilter []db.FiltroRuntimeMailbox
	pausedAgent       string
	pausedMinutes     int
	pausedReason      string
	retiredAgent      string
	enqueuedOrders    []*db.RuntimeOrder
}

func (f *fakeStore) RegisterAgent(nombre, rol string) error { return nil }
func (f *fakeStore) RegisterAgentAuto(proveedor, rol string) (string, error) {
	return "Codex1", nil
}
func (f *fakeStore) RetireAgent(nombre string) error {
	f.retiredAgent = nombre
	return nil
}
func (f *fakeStore) RehabilitateAgent(nombre string) error { return nil }
func (f *fakeStore) ResetReanimation(nombre string) error  { return nil }
func (f *fakeStore) DeleteAgent(nombre string) error       { return nil }
func (f *fakeStore) MergeAgents(origen, destino string) (*db.FusionAgentesResultado, error) {
	return &db.FusionAgentesResultado{Origen: origen, Destino: destino, Actualizadas: map[string]int64{}}, nil
}
func (f *fakeStore) Audit(agente, accion, entidad string, entidadID int64, detalle string) {}

func (f *fakeStore) GetAgent(nombre string) (*db.Agente, error) {
	for _, item := range f.agents {
		if item != nil && item.Nombre == nombre {
			return item, nil
		}
	}
	return nil, nil
}

func (f *fakeStore) GetProject(ref string) (*db.Proyecto, error) { return f.project, nil }

func (f *fakeStore) GetConnector(ref string) (*db.Conector, error) { return f.connector, nil }

func (f *fakeStore) ListAgents() ([]*db.Agente, error) { return f.agents, nil }

func (f *fakeStore) ListAssignments(filter db.FiltroAsignaciones) ([]*db.Asignacion, error) {
	if filter.Agente == nil {
		return f.assignments, nil
	}
	var out []*db.Asignacion
	for _, item := range f.assignments {
		if item != nil && item.Agente == *filter.Agente {
			out = append(out, item)
		}
	}
	return out, nil
}

func (f *fakeStore) ListInspectionSessions(filter db.FiltroSesionesInspeccion) ([]*db.Sesion, error) {
	if filter.Agente == nil {
		return f.sessions, nil
	}
	var out []*db.Sesion
	for _, item := range f.sessions {
		if item != nil && item.Agente == *filter.Agente {
			out = append(out, item)
		}
	}
	return out, nil
}

func (f *fakeStore) GetLastSession(agente string, proyectoID *int64) (*db.Sesion, error) {
	if len(f.sessions) == 0 {
		return nil, sql.ErrNoRows
	}
	return f.sessions[0], nil
}

func (f *fakeStore) GetActiveSession(agente string, proyectoID *int64) (*db.Sesion, error) {
	if len(f.sessions) == 0 {
		return nil, sql.ErrNoRows
	}
	return f.sessions[0], nil
}

func (f *fakeStore) SaveActiveSession(agente string, proyectoID *int64, upd db.SesionUpdate) error {
	return nil
}

func (f *fakeStore) ListRuntimes(filter db.FiltroRuntimes) ([]*db.RuntimeInstance, error) {
	if filter.Agente == nil {
		return f.runtimes, nil
	}
	var out []*db.RuntimeInstance
	for _, item := range f.runtimes {
		if item != nil && item.Agente == *filter.Agente {
			out = append(out, item)
		}
	}
	return out, nil
}

func (f *fakeStore) ListRuntimeHandles(agent *string) ([]*db.RuntimeHandle, error) {
	if agent == nil {
		return f.handles, nil
	}
	var out []*db.RuntimeHandle
	for _, item := range f.handles {
		if item != nil && item.Agente == *agent {
			out = append(out, item)
		}
	}
	return out, nil
}

func (f *fakeStore) ListRuntimeOrders(filter db.FiltroRuntimeOrders) ([]*db.RuntimeOrder, error) {
	if filter.Agente == nil {
		return f.orders, nil
	}
	var out []*db.RuntimeOrder
	for _, item := range f.orders {
		if item != nil && item.Agente == *filter.Agente {
			out = append(out, item)
		}
	}
	return out, nil
}

func (f *fakeStore) EnqueueRuntimeOrder(order *db.RuntimeOrder) (int64, error) {
	f.enqueuedOrders = append(f.enqueuedOrders, order)
	return int64(len(f.enqueuedOrders)), nil
}

func (f *fakeStore) ListRuntimeMailbox(filter db.FiltroRuntimeMailbox) ([]*db.RuntimeMailboxMessage, error) {
	f.lastMailboxFilter = append(f.lastMailboxFilter, filter)
	var out []*db.RuntimeMailboxMessage
	for _, item := range f.mailbox {
		if item == nil {
			continue
		}
		if filter.ToAgente != nil && item.ToAgente != *filter.ToAgente {
			continue
		}
		if filter.FromAgente != nil && item.FromAgente != *filter.FromAgente {
			continue
		}
		out = append(out, item)
	}
	return out, nil
}

func (f *fakeStore) ListRuntimeCheckpoints(filter db.FiltroRuntimeCheckpoints) ([]*db.RuntimeCheckpoint, error) {
	if filter.Agente == nil {
		return f.checkpoints, nil
	}
	var out []*db.RuntimeCheckpoint
	for _, item := range f.checkpoints {
		if item != nil && item.Agente == *filter.Agente {
			out = append(out, item)
		}
	}
	return out, nil
}

func (f *fakeStore) ListTasks(filter db.FiltroTareas) ([]*db.Tarea, error) { return f.tasks, nil }

func (f *fakeStore) ListProjectPendingVotes(agente string, proyectoID int64) ([]*db.Propuesta, error) {
	return f.proposals, nil
}

func (f *fakeStore) ListProjectOpenProposals(proyectoID int64) ([]*db.Propuesta, error) {
	return f.proposals, nil
}

func (f *fakeStore) ListRules(rol string) ([]*db.Regla, error) { return f.rules, nil }

func (f *fakeStore) ListSkills(rol string) ([]*db.Skill, error) { return f.skills, nil }

func (f *fakeStore) ListWorkflows(rol string) ([]*db.Workflow, error) { return f.workflows, nil }

func (f *fakeStore) ListMemoryEntities(filtro db.FiltroEntidadesMemoria) ([]*db.EntidadMemoria, error) {
	return f.memory, nil
}

func (f *fakeStore) ConfigGet(clave string) (string, error) {
	if f.config == nil {
		return "", nil
	}
	return f.config[clave], nil
}

func (f *fakeStore) PauseAgent(nombre string, minutos int, motivo string) error {
	f.pausedAgent = nombre
	f.pausedMinutes = minutos
	f.pausedReason = motivo
	return nil
}

func TestApplyStateActionRetirarEncolaPauseSiHayHandleActivo(t *testing.T) {
	proyectoID := int64(42)
	store := &fakeStore{
		handles: []*db.RuntimeHandle{
			{Agente: "Codex1", ProyectoID: &proyectoID, Estado: "activo"},
		},
	}

	svc := NewService(store)
	if err := svc.ApplyStateAction("Codex1", "retirar"); err != nil {
		t.Fatalf("ApplyStateAction retirar: %v", err)
	}

	if store.retiredAgent != "Codex1" {
		t.Fatalf("agente retirado inesperado: %s", store.retiredAgent)
	}
	if len(store.enqueuedOrders) != 1 {
		t.Fatalf("esperaba una orden de pause, got=%d", len(store.enqueuedOrders))
	}
	order := store.enqueuedOrders[0]
	if order.Tipo != "pause" || order.Agente != "Codex1" {
		t.Fatalf("orden encolada inesperada: %+v", order)
	}
	if order.ProyectoID == nil || *order.ProyectoID != proyectoID {
		t.Fatalf("proyecto de pause inesperado: %+v", order)
	}
	if !strings.Contains(order.PayloadJSON, `"motivo":"retiro_agente"`) {
		t.Fatalf("payload de pause inesperado: %s", order.PayloadJSON)
	}
}

func TestApplyStateActionRetirarNoDuplicaPauseSiYaExisteOrdenAbierta(t *testing.T) {
	store := &fakeStore{
		handles: []*db.RuntimeHandle{
			{Agente: "Codex1", Estado: "activo"},
		},
		orders: []*db.RuntimeOrder{
			{Agente: "Codex1", Tipo: "pause", Estado: "pendiente"},
		},
	}

	svc := NewService(store)
	if err := svc.ApplyStateAction("Codex1", "retirar"); err != nil {
		t.Fatalf("ApplyStateAction retirar: %v", err)
	}

	if store.retiredAgent != "Codex1" {
		t.Fatalf("agente retirado inesperado: %s", store.retiredAgent)
	}
	if len(store.enqueuedOrders) != 0 {
		t.Fatalf("no deberia duplicar pause: %+v", store.enqueuedOrders)
	}
}

func TestBuildPanelRowsAggregatesOperationalState(t *testing.T) {
	now := time.Now().UTC()
	older := now.Add(-2 * time.Hour)
	newer := now.Add(-30 * time.Minute)
	lastSeen := now.Add(-10 * time.Minute)
	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "Zulu", Rol: "programador"},
			{Nombre: "alpha", Rol: "programador", Activo: true, Habilitado: true},
		},
		assignments: []*db.Asignacion{
			{Agente: "alpha", Estado: db.AsignacionActiva, ProyectoSlug: "orquestador"},
			{Agente: "Zulu", Estado: db.AsignacionPausada, ProyectoSlug: "otro"},
		},
		sessions: []*db.Sesion{
			{ID: 10, Agente: "alpha", Activa: true, Herramienta: "codex-cli"},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 20, Agente: "alpha", CreatedAt: older, UpdatedAt: older},
			{ID: 21, Agente: "alpha", CreatedAt: older, UpdatedAt: newer},
		},
		handles: []*db.RuntimeHandle{
			{ID: 30, Agente: "alpha", CreatedAt: older, UpdatedAt: older},
			{ID: 31, Agente: "alpha", LastSeenAt: &lastSeen, CreatedAt: older, UpdatedAt: older},
		},
		orders: []*db.RuntimeOrder{
			{ID: 40, Agente: "alpha", Estado: "pendiente"},
			{ID: 41, Agente: "alpha", Estado: "fallida"},
		},
		mailbox: []*db.RuntimeMailboxMessage{
			{ID: 50, FromAgente: "supervisor", ToAgente: "alpha", Estado: "pendiente"},
			{ID: 51, FromAgente: "alpha", ToAgente: "Zulu", Estado: "consumida"},
		},
		checkpoints: []*db.RuntimeCheckpoint{
			{ID: 60, Agente: "alpha", Resumen: "ultimo"},
		},
		tasks: []*db.Tarea{
			{ID: 70, Agente: ptr("alpha"), Estado: db.EstadoEnProgreso},
			{ID: 71, Agente: ptr("alpha"), Estado: db.EstadoCompletada},
		},
	}

	rows, err := NewService(store).BuildPanelRows()
	if err != nil {
		t.Fatalf("BuildPanelRows: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows=%d, want 2", len(rows))
	}
	if rows[0].Agente == nil || rows[0].Agente.Nombre != "alpha" {
		t.Fatalf("rows[0]=%+v, want alpha sorted first", rows[0].Agente)
	}
	got := rows[0]
	if got.Asignacion == nil || got.Asignacion.ProyectoSlug != "orquestador" {
		t.Fatalf("assignment=%+v", got.Asignacion)
	}
	if got.Sesion == nil || got.Sesion.ID != 10 {
		t.Fatalf("session=%+v", got.Sesion)
	}
	if got.Runtime == nil || got.Runtime.ID != 21 {
		t.Fatalf("runtime=%+v", got.Runtime)
	}
	if got.Handle == nil || got.Handle.ID != 31 {
		t.Fatalf("handle=%+v", got.Handle)
	}
	if got.OrdersOpen != 1 || got.OrdersFailed != 1 {
		t.Fatalf("orders open=%d failed=%d", got.OrdersOpen, got.OrdersFailed)
	}
	if got.MailboxPending != 1 || got.MailboxTotal != 2 {
		t.Fatalf("mailbox pending=%d total=%d", got.MailboxPending, got.MailboxTotal)
	}
	if got.Checkpoints != 1 || got.LastCheckpoint == nil || got.LastCheckpoint.ID != 60 {
		t.Fatalf("checkpoints total=%d last=%+v", got.Checkpoints, got.LastCheckpoint)
	}
	if got.OpenTasks != 1 {
		t.Fatalf("open tasks=%d, want 1", got.OpenTasks)
	}
}

func TestBuildDetailMergesMailboxWithoutDuplicates(t *testing.T) {
	store := &fakeStore{
		agents: []*db.Agente{{Nombre: "Codex2", Rol: "programador"}},
		mailbox: []*db.RuntimeMailboxMessage{
			{ID: 100, FromAgente: "Supervisor", ToAgente: "Codex2"},
			{ID: 101, FromAgente: "Codex2", ToAgente: "Codex1"},
		},
	}

	detail, err := NewService(store).BuildDetail("Codex2")
	if err != nil {
		t.Fatalf("BuildDetail: %v", err)
	}
	if detail.Row.Agente == nil || detail.Row.Agente.Nombre != "Codex2" {
		t.Fatalf("row agent=%+v", detail.Row.Agente)
	}
	if len(detail.Mailbox) != 2 {
		t.Fatalf("mailbox len=%d, want 2", len(detail.Mailbox))
	}
	if len(store.lastMailboxFilter) != 3 {
		t.Fatalf("mailbox filters=%d, want 3", len(store.lastMailboxFilter))
	}
	if store.lastMailboxFilter[1].ToAgente == nil || *store.lastMailboxFilter[1].ToAgente != "Codex2" {
		t.Fatalf("detail inbox filter=%+v", store.lastMailboxFilter[1])
	}
	if store.lastMailboxFilter[2].FromAgente == nil || *store.lastMailboxFilter[2].FromAgente != "Codex2" {
		t.Fatalf("detail outbox filter=%+v", store.lastMailboxFilter[2])
	}
}

func ptr(value string) *string {
	return &value
}
