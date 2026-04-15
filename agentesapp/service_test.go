package agentesapp

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orquesta/db"
)

func timePtr(v time.Time) *time.Time { return &v }

type fakeStore struct {
	agents            []*db.Agente
	project           *db.Proyecto
	connector         *db.Conector
	assignments       []*db.Asignacion
	sessions          []*db.Sesion
	runtimes          []*db.RuntimeInstance
	handles           []*db.RuntimeHandle
	canonicalHandles  []*db.RuntimeHandle
	hotHandles        map[string]*db.RuntimeHandle
	transcript        []*db.RuntimeTranscriptEntry
	orders            []*db.RuntimeOrder
	mailbox           []*db.RuntimeMailboxMessage
	checkpoints       []*db.RuntimeCheckpoint
	tasks             []*db.Tarea
	proposals         []*db.Propuesta
	latestBudget      *db.PresupuestoSesion
	rules             []*db.Regla
	skills            []*db.Skill
	workflows         []*db.Workflow
	memory            []*db.EntidadMemoria
	config            map[string]string
	lastConnectorRef  string
	lastMailboxFilter []db.FiltroRuntimeMailbox
	lastTranscript    db.FiltroRuntimeTranscript
	pausedAgent       string
	pausedMinutes     int
	pausedReason      string
	retiredAgent      string
	observedIdentity  struct {
		nombre string
		email  string
		user   string
		source string
		when   *time.Time
	}
	enqueuedOrders []*db.RuntimeOrder
}

func (f *fakeStore) RegisterAgent(nombre, rol string) error { return nil }
func (f *fakeStore) RegisterAgentAuto(proveedor, rol string) (string, error) {
	return "Codex1", nil
}
func (f *fakeStore) ObserveAgentIdentity(nombre, email, usuario, fuente string, observedAt *time.Time) error {
	f.observedIdentity.nombre = nombre
	f.observedIdentity.email = email
	f.observedIdentity.user = usuario
	f.observedIdentity.source = fuente
	f.observedIdentity.when = observedAt
	return nil
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

func (f *fakeStore) GetPool(slug string) (*db.PoolCapacidad, error) {
	return nil, nil
}

func (f *fakeStore) GetConnector(ref string) (*db.Conector, error) {
	f.lastConnectorRef = ref
	return f.connector, nil
}

func (f *fakeStore) ListAgents() ([]*db.Agente, error) { return f.agents, nil }

func (f *fakeStore) CheckReanimations() ([]*db.Agente, error) {
	now := time.Now().UTC()
	var out []*db.Agente
	for _, item := range f.agents {
		if item == nil || item.ReanimarAt == nil {
			continue
		}
		if !item.ReanimarAt.After(now) {
			out = append(out, item)
		}
	}
	return out, nil
}

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

func (f *fakeStore) ListCanonicalRuntimeHandles(agent *string) ([]*db.RuntimeHandle, error) {
	source := f.canonicalHandles
	if source == nil {
		source = f.handles
	}
	if agent == nil {
		return source, nil
	}
	var out []*db.RuntimeHandle
	for _, item := range source {
		if item != nil && item.Agente == *agent {
			out = append(out, item)
		}
	}
	return out, nil
}

func (f *fakeStore) ListRecentOperationalRuntimeHandles() (map[string]*db.RuntimeHandle, error) {
	if f.hotHandles == nil {
		return nil, nil
	}
	out := make(map[string]*db.RuntimeHandle, len(f.hotHandles))
	for key, handle := range f.hotHandles {
		if handle == nil {
			continue
		}
		cp := *handle
		out[key] = &cp
	}
	return out, nil
}

func (f *fakeStore) ListRuntimeTranscript(filter db.FiltroRuntimeTranscript) ([]*db.RuntimeTranscriptEntry, error) {
	f.lastTranscript = filter
	if filter.Agente == nil {
		return f.transcript, nil
	}
	var out []*db.RuntimeTranscriptEntry
	for _, item := range f.transcript {
		if item != nil && item.Agente == *filter.Agente {
			out = append(out, item)
		}
	}
	return out, nil
}

func TestBuildAgentEntityIncluyeCuentaID(t *testing.T) {
	entity := buildAgentEntity(Row{
		Agente: &db.Agente{
			Nombre:        "Codex7",
			CuentaID:      "acc-codex-7",
			CuentaEmail:   "shared@example.com",
			CuentaUsuario: "Codex7",
		},
	}, nil)
	if entity == nil {
		t.Fatal("entity nil")
	}
	if entity.AccountID != "acc-codex-7" || entity.AccountEmail != "shared@example.com" || entity.AccountUser != "Codex7" {
		t.Fatalf("cuenta inesperada: %+v", entity)
	}
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

func (f *fakeStore) GetLatestAgentBudget(agente string) (*db.PresupuestoSesion, *db.Sesion, error) {
	if f.latestBudget == nil {
		return nil, nil, sql.ErrNoRows
	}
	var sesion *db.Sesion
	if len(f.sessions) > 0 {
		sesion = f.sessions[0]
	}
	return f.latestBudget, sesion, nil
}

func (f *fakeStore) ResolveGovernanceCatalog(rol string, proyectoID *int64) (*db.GovernanceCatalog, error) {
	return &db.GovernanceCatalog{
		TipoAgente: rol,
		ProyectoID: proyectoID,
		Reglas:     f.rules,
		Skills:     f.skills,
		Workflows:  f.workflows,
		Hash:       "fake",
	}, nil
}

func (f *fakeStore) ResolveGovernanceCatalogForContext(rol string, proyectoID *int64, agente string) (*db.GovernanceCatalog, error) {
	return f.ResolveGovernanceCatalog(rol, proyectoID)
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

	svc := NewService(store, nil)
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

	svc := NewService(store, nil)
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

func TestApplyStateActionRetirarPrefiereHandlesCanonicosCalientes(t *testing.T) {
	proyectoID := int64(42)
	store := &fakeStore{
		handles: []*db.RuntimeHandle{
			{Agente: "Codex1", ProyectoID: &proyectoID, Estado: "cerrado"},
		},
		canonicalHandles: []*db.RuntimeHandle{
			{Agente: "Codex1", ProyectoID: &proyectoID, Estado: "activo"},
		},
	}

	svc := NewService(store, nil)
	if err := svc.ApplyStateAction("Codex1", "retirar"); err != nil {
		t.Fatalf("ApplyStateAction retirar: %v", err)
	}

	if len(store.enqueuedOrders) != 1 {
		t.Fatalf("esperaba una orden de pause desde handles canonicos, got=%d", len(store.enqueuedOrders))
	}
	order := store.enqueuedOrders[0]
	if order.Tipo != "pause" || order.Agente != "Codex1" {
		t.Fatalf("orden encolada inesperada: %+v", order)
	}
}

func TestBuildPanelRowsAggregatesOperationalState(t *testing.T) {
	now := time.Now().UTC()
	older := now.Add(-2 * time.Hour)
	newer := now.Add(-30 * time.Minute)
	lastSeen := now.Add(-10 * time.Minute)
	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "Zulu", Rol: "programador", Habilitado: true},
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
			{ID: 72, Agente: ptr("Zulu"), Estado: db.EstadoBloqueada},
		},
	}

	rows, err := NewService(store, nil).BuildPanelRows()
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
	if got.EstadoOperativo != "mailbox_atascada" {
		t.Fatalf("estado operativo=%q", got.EstadoOperativo)
	}
	if got.DetalleOperativo == "" {
		t.Fatalf("detalle operativo vacio")
	}
	if rows[1].BlockedTasks != 1 {
		t.Fatalf("blocked tasks=%d, want 1", rows[1].BlockedTasks)
	}
	if rows[1].EstadoOperativo != "bloqueado" {
		t.Fatalf("estado operativo zulu=%q", rows[1].EstadoOperativo)
	}
}

func TestBuildPanelRowsMarcaRetiradoAunqueTengaRuntimeStale(t *testing.T) {
	now := time.Now().UTC()
	lastSeen := now.Add(-30 * time.Second)
	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "Codex4", Rol: "programador", Activo: true, Habilitado: false, EstadoSesion: "pensando"},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 21, Agente: "Codex4", LogicalState: "running", CreatedAt: now.Add(-10 * time.Minute), UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 31, Agente: "Codex4", Estado: "activo", LastSeenAt: &lastSeen, CreatedAt: now.Add(-10 * time.Minute), UpdatedAt: now},
		},
	}

	rows, err := NewService(store, nil).BuildPanelRows()
	if err != nil {
		t.Fatalf("BuildPanelRows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(rows))
	}
	if rows[0].EstadoOperativo != "retirado" {
		t.Fatalf("estado operativo=%q, want retirado", rows[0].EstadoOperativo)
	}
}

func TestBuildPanelRowsMarcaMailboxAtascadaConFallosAunqueHayaActividadReciente(t *testing.T) {
	now := time.Now().UTC()
	lastSeen := now.Add(-1 * time.Minute)
	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "Codex4", Rol: "programador", Activo: true, Habilitado: true},
		},
		assignments: []*db.Asignacion{
			{Agente: "Codex4", Estado: db.AsignacionActiva, ProyectoSlug: "orquestador"},
		},
		sessions: []*db.Sesion{
			{ID: 10, Agente: "Codex4", Activa: true, Herramienta: "codex-cli"},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 21, Agente: "Codex4", LogicalState: "esperando_io", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 31, Agente: "Codex4", Estado: "activo", LastSeenAt: &lastSeen, UpdatedAt: now},
		},
		orders: []*db.RuntimeOrder{
			{ID: 40, Agente: "Codex4", Estado: "fallida"},
			{ID: 41, Agente: "Codex4", Estado: "fallida", ErrorText: "broken pipe"},
		},
		mailbox: []*db.RuntimeMailboxMessage{
			{ID: 50, FromAgente: "server", ToAgente: "Codex4", Estado: "pendiente"},
		},
		tasks: []*db.Tarea{
			{ID: 70, Agente: ptr("Codex4"), Estado: db.EstadoEnProgreso},
		},
	}

	rows, err := NewService(store, nil).BuildPanelRows()
	if err != nil {
		t.Fatalf("BuildPanelRows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(rows))
	}
	if rows[0].EstadoOperativo != "mailbox_atascada" {
		t.Fatalf("estado operativo=%q", rows[0].EstadoOperativo)
	}
	if rows[0].DetalleOperativo != "mailbox pendiente con fallos de control" {
		t.Fatalf("detalle operativo=%q", rows[0].DetalleOperativo)
	}
}

func TestBuildPanelRowsNoMarcaMailboxAtascadaSiWorkerFreshYaEstaCorriendo(t *testing.T) {
	now := time.Now().UTC()
	lastSeen := now.Add(-1 * time.Minute)
	tmp := t.TempDir()
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")
	statusRaw, _ := json.Marshal(map[string]any{
		"state":      "running",
		"updated_at": now.Format(time.RFC3339Nano),
		"alive":      true,
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":        true,
		"heartbeat_at": now.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})

	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "Codex8", Rol: "programador", Activo: true, Habilitado: true},
		},
		assignments: []*db.Asignacion{
			{Agente: "Codex8", Estado: db.AsignacionActiva, ProyectoSlug: "orquestador"},
		},
		sessions: []*db.Sesion{
			{ID: 10, Agente: "Codex8", Activa: true, Herramienta: "codex-cli"},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 21, Agente: "Codex8", LogicalState: "esperando_io", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 31, Agente: "Codex8", Estado: "activo", LastSeenAt: &lastSeen, UpdatedAt: now, MetadataJSON: string(metaJSON)},
		},
		orders: []*db.RuntimeOrder{
			{ID: 40, Agente: "Codex8", Estado: "fallida"},
			{ID: 41, Agente: "Codex8", Estado: "fallida", ErrorText: "broken pipe"},
		},
		mailbox: []*db.RuntimeMailboxMessage{
			{ID: 50, FromAgente: "server", ToAgente: "Codex8", Estado: "pendiente"},
		},
		tasks: []*db.Tarea{
			{ID: 70, Agente: ptr("Codex8"), Estado: db.EstadoEnProgreso},
		},
	}

	rows, err := NewService(store, nil).BuildPanelRows()
	if err != nil {
		t.Fatalf("BuildPanelRows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(rows))
	}
	if rows[0].EstadoOperativo == "mailbox_atascada" {
		t.Fatalf("estado operativo no deberia quedar atascado con worker fresco: %+v", rows[0])
	}
}

func TestBuildPanelRowsMarcaAtascadoSiMailboxPendienteYWorkerStaleSinTareasActivas(t *testing.T) {
	now := time.Now().UTC()
	lastSeen := now.Add(-1 * time.Minute)
	stale := now.Add(-14 * time.Hour)
	tmp := t.TempDir()
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")
	statusRaw, _ := json.Marshal(map[string]any{
		"state":            "running",
		"updated_at":       now.Format(time.RFC3339Nano),
		"alive":            true,
		"last_output_at":   stale.Format(time.RFC3339Nano),
		"last_progress_at": stale.Format(time.RFC3339Nano),
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":            true,
		"heartbeat_at":     now.Format(time.RFC3339Nano),
		"last_output_at":   stale.Format(time.RFC3339Nano),
		"last_progress_at": stale.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
		"driver":                "tmux_cli_session",
	})
	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "Codex2", Rol: "programador", Activo: true, Habilitado: true},
		},
		assignments: []*db.Asignacion{
			{Agente: "Codex2", Estado: db.AsignacionActiva, ProyectoSlug: "orquestador"},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 21, Agente: "Codex2", LogicalState: "esperando_io", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 31, Agente: "Codex2", Estado: "activo", LastSeenAt: &lastSeen, UpdatedAt: now, MetadataJSON: string(metaJSON)},
		},
		mailbox: []*db.RuntimeMailboxMessage{
			{ID: 50, FromAgente: "server", ToAgente: "Codex2", Estado: "pendiente"},
		},
		tasks: []*db.Tarea{
			{ID: 70, Agente: ptr("Codex2"), Estado: db.EstadoBloqueada},
		},
	}

	rows, err := NewService(store, nil).BuildPanelRows()
	if err != nil {
		t.Fatalf("BuildPanelRows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(rows))
	}
	if rows[0].EstadoOperativo != "atascado" {
		t.Fatalf("estado operativo=%q", rows[0].EstadoOperativo)
	}
}

func TestBuildPanelRowsNoCuentaMailboxHuerfanaComoAtascado(t *testing.T) {
	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "CodexZombie", Rol: "programador", Activo: true, Habilitado: true},
		},
		mailbox: []*db.RuntimeMailboxMessage{
			{ID: 50, FromAgente: "server", ToAgente: "CodexZombie", Estado: "pendiente"},
		},
	}

	rows, err := NewService(store, nil).BuildPanelRows()
	if err != nil {
		t.Fatalf("BuildPanelRows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(rows))
	}
	if rows[0].EstadoOperativo != "bloqueado" {
		t.Fatalf("estado operativo=%q", rows[0].EstadoOperativo)
	}
	if rows[0].DetalleOperativo != "mailbox pendiente sin runtime activo" {
		t.Fatalf("detalle operativo=%q", rows[0].DetalleOperativo)
	}
}

func TestBuildPanelRowsMarcaMailboxHuerfanaConTareaActivaComoBloqueadoPorRuntime(t *testing.T) {
	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "CodexZombie", Rol: "programador", Activo: true, Habilitado: true},
		},
		mailbox: []*db.RuntimeMailboxMessage{
			{ID: 50, FromAgente: "server", ToAgente: "CodexZombie", Estado: "pendiente"},
		},
		tasks: []*db.Tarea{
			{ID: 70, Agente: ptr("CodexZombie"), Estado: db.EstadoEnProgreso},
		},
	}

	rows, err := NewService(store, nil).BuildPanelRows()
	if err != nil {
		t.Fatalf("BuildPanelRows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(rows))
	}
	if rows[0].EstadoOperativo != "bloqueado_por_runtime" {
		t.Fatalf("estado operativo=%q", rows[0].EstadoOperativo)
	}
	if rows[0].DetalleOperativo != "mailbox pendiente sin runtime activo" {
		t.Fatalf("detalle operativo=%q", rows[0].DetalleOperativo)
	}
}

func TestBuildPanelRowsUsaWorkerStatusEHeartbeatEstructurados(t *testing.T) {
	now := time.Now().UTC()
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "manifest.json")
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")
	manifestRaw, _ := json.Marshal(map[string]any{
		"version":      1,
		"agent":        "Codex7",
		"driver":       "tmux_cli_session",
		"transport":    "tmux",
		"tmux_session": "orq-codex7-090000",
		"tmux_pane_id": "%3",
		"started_at":   now.Format(time.RFC3339Nano),
	})
	statusRaw, _ := json.Marshal(map[string]any{
		"state":      "running",
		"updated_at": now.Format(time.RFC3339Nano),
		"alive":      true,
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":        true,
		"heartbeat_at": now.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(manifestPath, append(manifestRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})

	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "Codex7", Rol: "programador", Activo: true, Habilitado: true},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 21, Agente: "Codex7", LogicalState: "degradado", UpdatedAt: now.Add(-20 * time.Minute)},
		},
		handles: []*db.RuntimeHandle{
			{ID: 31, Agente: "Codex7", Estado: "fallido", UpdatedAt: now.Add(-20 * time.Minute), MetadataJSON: string(metaJSON)},
		},
		tasks: []*db.Tarea{
			{ID: 70, Agente: ptr("Codex7"), Estado: db.EstadoEnProgreso},
		},
	}

	rows, err := NewService(store, nil).BuildPanelRows()
	if err != nil {
		t.Fatalf("BuildPanelRows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(rows))
	}
	if rows[0].WorkerState != "running" || !rows[0].WorkerAlive {
		t.Fatalf("worker snapshot inesperado: %+v", rows[0])
	}
	if rows[0].WorkerDriver != "tmux_cli_session" || rows[0].WorkerTransport != "tmux" {
		t.Fatalf("driver/transport worker inesperado: %+v", rows[0])
	}
	if rows[0].WorkerSessionRef != "orq-codex7-090000/%3" {
		t.Fatalf("worker session ref inesperado: %+v", rows[0])
	}
	if rows[0].EstadoOperativo != "bloqueado_por_runtime" {
		t.Fatalf("estado operativo=%q", rows[0].EstadoOperativo)
	}
}

func TestBuildPanelRowsMarcaLegacyCLIComoBloqueadoHastaMigrarATMUX(t *testing.T) {
	now := time.Now().UTC()
	tmp := t.TempDir()
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")
	statusRaw, _ := json.Marshal(map[string]any{
		"state":      "running",
		"updated_at": now.Format(time.RFC3339Nano),
		"alive":      true,
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":        true,
		"heartbeat_at": now.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                "process_pty_cli",
		"rendered_command":      "/tmp/codex-perfiles/bin/codex-perfil Codex7 --model gpt-5.4",
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})

	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "Codex7", Rol: "programador", Activo: true, Habilitado: true},
		},
		sessions: []*db.Sesion{
			{Agente: "Codex7", Herramienta: "codex-cli", Estado: "activa", Inicio: now},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 21, Agente: "Codex7", LogicalState: "esperando_io", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 31, Agente: "Codex7", Estado: "activo", UpdatedAt: now, MetadataJSON: string(metaJSON)},
		},
		tasks: []*db.Tarea{
			{ID: 70, Agente: ptr("Codex7"), Estado: db.EstadoEnProgreso},
		},
	}

	rows, err := NewService(store, nil).BuildPanelRows()
	if err != nil {
		t.Fatalf("BuildPanelRows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(rows))
	}
	if rows[0].EstadoOperativo != "bloqueado_por_runtime" {
		t.Fatalf("estado operativo=%q", rows[0].EstadoOperativo)
	}
	if !strings.Contains(rows[0].DetalleOperativo, "legacy") {
		t.Fatalf("detalle operativo inesperado: %q", rows[0].DetalleOperativo)
	}
}

func TestBuildPanelRowsNoMarcaLegacySiWorkerEstructuradoNoDeclaraDriver(t *testing.T) {
	now := time.Now().UTC()
	tmp := t.TempDir()
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")
	statusRaw, _ := json.Marshal(map[string]any{
		"state":      "running",
		"updated_at": now.Format(time.RFC3339Nano),
		"alive":      true,
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":        true,
		"heartbeat_at": now.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})

	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "Codex7", Rol: "programador", Activo: true, Habilitado: true},
		},
		sessions: []*db.Sesion{
			{Agente: "Codex7", Herramienta: "codex-cli", Estado: "activa", Inicio: now},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 21, Agente: "Codex7", LogicalState: "esperando_io", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 31, Agente: "Codex7", Estado: "activo", UpdatedAt: now, MetadataJSON: string(metaJSON)},
		},
		tasks: []*db.Tarea{
			{ID: 70, Agente: ptr("Codex7"), Estado: db.EstadoEnProgreso},
		},
	}

	rows, err := NewService(store, nil).BuildPanelRows()
	if err != nil {
		t.Fatalf("BuildPanelRows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(rows))
	}
	if rows[0].EstadoOperativo == "bloqueado_por_runtime" && strings.Contains(rows[0].DetalleOperativo, "legacy") {
		t.Fatalf("un worker estructurado sin driver explicito no deberia caer a legacy por defecto: %+v", rows[0])
	}
}

func TestBuildPanelRowsPrefiereSnapshotCalienteOperativo(t *testing.T) {
	now := time.Now().UTC()
	store := &fakeStore{
		agents: []*db.Agente{{Nombre: "Codex7", Rol: "programador"}},
		runtimes: []*db.RuntimeInstance{{
			ID:           7,
			Agente:       "Codex7",
			Connector:    "codex-cli",
			LogicalState: "running",
			UpdatedAt:    now,
			CreatedAt:    now.Add(-5 * time.Minute),
			LastEventAt:  &now,
		}},
		handles: []*db.RuntimeHandle{{
			ID:           71,
			Agente:       "Codex7",
			Estado:       "activo",
			HandleKind:   "process",
			Transporte:   "cli",
			MetadataJSON: `{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex7 --model gpt-5.4"}`,
			UpdatedAt:    now,
			CreatedAt:    now.Add(-5 * time.Minute),
			LastSeenAt:   timePtr(now),
		}},
		hotHandles: map[string]*db.RuntimeHandle{
			"codex7#1": {
				ID:           72,
				Agente:       "Codex7",
				Estado:       "activo",
				HandleKind:   "process",
				Transporte:   "tmux",
				MetadataJSON: `{"driver":"tmux_cli_session","tmux_session":"orq-codex7-1"}`,
				UpdatedAt:    now.Add(10 * time.Second),
				CreatedAt:    now,
				LastSeenAt:   timePtr(now.Add(10 * time.Second)),
			},
		},
	}

	rows, err := NewService(store, nil).BuildPanelRows()
	if err != nil {
		t.Fatalf("BuildPanelRows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows=%d", len(rows))
	}
	if rows[0].Handle == nil || rows[0].Handle.ID != 72 {
		t.Fatalf("deberia preferir el handle del snapshot caliente, got=%+v", rows[0].Handle)
	}
}

func TestBuildPanelRowsUsaCanonicosCalientesAntesDelBarridoCompleto(t *testing.T) {
	now := time.Now().UTC()
	store := &fakeStore{
		agents: []*db.Agente{{Nombre: "CodexCanon", Rol: "programador"}},
		handles: []*db.RuntimeHandle{{
			ID:         81,
			Agente:     "CodexCanon",
			Estado:     "activo",
			HandleKind: "process",
			Transporte: "cli",
			UpdatedAt:  now.Add(-2 * time.Hour),
			CreatedAt:  now.Add(-3 * time.Hour),
			LastSeenAt: timePtr(now.Add(-2 * time.Hour)),
		}},
		canonicalHandles: []*db.RuntimeHandle{{
			ID:           82,
			Agente:       "CodexCanon",
			Estado:       "activo",
			HandleKind:   "session",
			Transporte:   "tmux",
			MetadataJSON: `{"driver":"tmux_cli_session","tmux_session":"orq-codexcanon-1"}`,
			UpdatedAt:    now,
			CreatedAt:    now.Add(-5 * time.Minute),
			LastSeenAt:   timePtr(now),
		}},
	}

	rows, err := NewService(store, nil).BuildPanelRows()
	if err != nil {
		t.Fatalf("BuildPanelRows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows=%d", len(rows))
	}
	if rows[0].Handle == nil || rows[0].Handle.ID != 82 {
		t.Fatalf("deberia usar el handle canónico caliente antes del barrido completo, got=%+v", rows[0].Handle)
	}
}

func TestBuildPanelRowsMarcaSaturadoSiTieneActivasYBloqueadasConActividad(t *testing.T) {
	now := time.Now().UTC()
	tmp := t.TempDir()
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")
	statusRaw, _ := json.Marshal(map[string]any{
		"state":      "running",
		"updated_at": now.Format(time.RFC3339Nano),
		"alive":      true,
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":        true,
		"heartbeat_at": now.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})

	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "Codex8", Rol: "programador", Activo: true, Habilitado: true},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 21, Agente: "Codex8", LogicalState: "esperando_io", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 31, Agente: "Codex8", Estado: "activo", UpdatedAt: now, MetadataJSON: string(metaJSON)},
		},
		tasks: []*db.Tarea{
			{ID: 70, Agente: ptr("Codex8"), Estado: db.EstadoEnProgreso},
			{ID: 71, Agente: ptr("Codex8"), Estado: db.EstadoBloqueada},
		},
	}

	rows, err := NewService(store, nil).BuildPanelRows()
	if err != nil {
		t.Fatalf("BuildPanelRows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(rows))
	}
	if rows[0].EstadoOperativo != "saturado" {
		t.Fatalf("estado operativo=%q", rows[0].EstadoOperativo)
	}
	if !strings.Contains(rows[0].DetalleOperativo, "activas") || !strings.Contains(rows[0].DetalleOperativo, "bloqueadas") {
		t.Fatalf("detalle operativo=%q", rows[0].DetalleOperativo)
	}
}

func TestBuildPanelRowsMarcaAtascadoSiWorkerFreshSinSalidaMasAllaDelUmbral(t *testing.T) {
	now := time.Now().UTC()
	lastOutput := now.Add(-25 * time.Minute)
	tmp := t.TempDir()
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")
	statusRaw, _ := json.Marshal(map[string]any{
		"state":          "running",
		"updated_at":     now.Format(time.RFC3339Nano),
		"alive":          true,
		"last_output_at": lastOutput.Format(time.RFC3339Nano),
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":          true,
		"heartbeat_at":   now.Format(time.RFC3339Nano),
		"last_output_at": lastOutput.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})

	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "Codex11", Rol: "programador", Activo: true, Habilitado: true},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 21, Agente: "Codex11", LogicalState: "esperando_io", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 31, Agente: "Codex11", Estado: "activo", UpdatedAt: now, MetadataJSON: string(metaJSON)},
		},
		tasks: []*db.Tarea{
			{ID: 70, Agente: ptr("Codex11"), Estado: db.EstadoEnProgreso},
		},
	}

	rows, err := NewService(store, nil).BuildPanelRows()
	if err != nil {
		t.Fatalf("BuildPanelRows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(rows))
	}
	if rows[0].EstadoOperativo != "atascado" {
		t.Fatalf("estado operativo=%q", rows[0].EstadoOperativo)
	}
	if !strings.Contains(rows[0].DetalleOperativo, "sin salida") {
		t.Fatalf("detalle operativo=%q", rows[0].DetalleOperativo)
	}
}

func TestBuildPanelRowsMarcaAtascadoSiWorkerReadyLlevaDemasiadoTiempoSinProgreso(t *testing.T) {
	now := time.Now().UTC()
	readyAt := now.Add(-25 * time.Minute)
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "manifest.json")
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")
	manifestRaw, _ := json.Marshal(map[string]any{
		"driver":       "tmux_cli_session",
		"transport":    "tmux",
		"tmux_session": "orq-codex14-120000",
		"tmux_pane_id": "%3",
	})
	statusRaw, _ := json.Marshal(map[string]any{
		"state":      "ready",
		"updated_at": now.Format(time.RFC3339Nano),
		"alive":      true,
		"ready_at":   readyAt.Format(time.RFC3339Nano),
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":        true,
		"heartbeat_at": now.Format(time.RFC3339Nano),
		"ready_at":     readyAt.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(manifestPath, append(manifestRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"worker_manifest_path":  manifestPath,
		"driver":                "tmux_cli_session",
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})

	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "Codex14", Rol: "programador", Activo: true, Habilitado: true},
		},
		sessions: []*db.Sesion{
			{Agente: "Codex14", Herramienta: "codex-cli", Estado: "activa", Inicio: now},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 21, Agente: "Codex14", LogicalState: "esperando_io", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 31, Agente: "Codex14", Estado: "activo", UpdatedAt: now, MetadataJSON: string(metaJSON)},
		},
		tasks: []*db.Tarea{
			{ID: 70, Agente: ptr("Codex14"), Estado: db.EstadoEnProgreso},
		},
	}

	rows, err := NewService(store, nil).BuildPanelRows()
	if err != nil {
		t.Fatalf("BuildPanelRows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(rows))
	}
	if rows[0].EstadoOperativo != "atascado" {
		t.Fatalf("estado operativo=%q", rows[0].EstadoOperativo)
	}
	if !strings.Contains(rows[0].DetalleOperativo, "sin progreso desde listo") {
		t.Fatalf("detalle operativo=%q", rows[0].DetalleOperativo)
	}
}

func TestBuildPanelRowsMarcaBloqueadoSiWorkerRequiereAutenticacionManual(t *testing.T) {
	now := time.Now().UTC()
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "manifest.json")
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")
	manifestRaw, _ := json.Marshal(map[string]any{
		"driver":       "tmux_cli_session",
		"transport":    "tmux",
		"tmux_session": "orq-codexauth-120000",
		"tmux_pane_id": "%3",
	})
	statusRaw, _ := json.Marshal(map[string]any{
		"state":      "blocked_auth",
		"updated_at": now.Format(time.RFC3339Nano),
		"alive":      true,
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":        true,
		"heartbeat_at": now.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(manifestPath, append(manifestRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
		"driver":                "tmux_cli_session",
	})

	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "CodexAuth", Rol: "programador", Activo: true, Habilitado: true},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 51, Agente: "CodexAuth", LogicalState: "esperando_io", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 61, Agente: "CodexAuth", Estado: "activo", UpdatedAt: now, MetadataJSON: string(metaJSON)},
		},
		tasks: []*db.Tarea{
			{ID: 71, Agente: ptr("CodexAuth"), Estado: db.EstadoEnProgreso},
		},
	}

	rows, err := NewService(store, nil).BuildPanelRows()
	if err != nil {
		t.Fatalf("BuildPanelRows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(rows))
	}
	if rows[0].EstadoOperativo != "bloqueado_por_runtime" {
		t.Fatalf("estado operativo=%q", rows[0].EstadoOperativo)
	}
	if rows[0].DetalleOperativo != "worker requiere autenticacion manual" {
		t.Fatalf("detalle operativo=%q", rows[0].DetalleOperativo)
	}
}

func TestBuildPanelRowsMarcaBloqueadoPorCuotaSiWorkerGolpeaUsageLimit(t *testing.T) {
	now := time.Now().UTC()
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "manifest.json")
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")
	manifestRaw, _ := json.Marshal(map[string]any{
		"driver":       "tmux_cli_session",
		"transport":    "tmux",
		"tmux_session": "orq-codexquota-120000",
		"tmux_pane_id": "%4",
	})
	statusRaw, _ := json.Marshal(map[string]any{
		"state":      "blocked_quota",
		"updated_at": now.Format(time.RFC3339Nano),
		"alive":      true,
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":        true,
		"heartbeat_at": now.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(manifestPath, append(manifestRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
		"driver":                "tmux_cli_session",
	})

	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "CodexQuota", Rol: "programador", Activo: true, Habilitado: true},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 52, Agente: "CodexQuota", LogicalState: "esperando_io", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 62, Agente: "CodexQuota", Estado: "activo", UpdatedAt: now, MetadataJSON: string(metaJSON)},
		},
		tasks: []*db.Tarea{
			{ID: 72, Agente: ptr("CodexQuota"), Estado: db.EstadoEnProgreso},
		},
	}

	rows, err := NewService(store, nil).BuildPanelRows()
	if err != nil {
		t.Fatalf("BuildPanelRows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(rows))
	}
	if rows[0].EstadoOperativo != "bloqueado_por_cuota" {
		t.Fatalf("estado operativo=%q", rows[0].EstadoOperativo)
	}
	if rows[0].DetalleOperativo != "worker bloqueado por cuota" {
		t.Fatalf("detalle operativo=%q", rows[0].DetalleOperativo)
	}
}

func TestBuildPanelRowsNoMarcaCuotaParaAgenteLocalSinProveedor(t *testing.T) {
	now := time.Now().UTC()
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "manifest.json")
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")
	manifestRaw, _ := json.Marshal(map[string]any{
		"driver":       "tmux_cli_session",
		"transport":    "tmux",
		"tmux_session": "orq-gemma1-120000",
		"tmux_pane_id": "%7",
	})
	statusRaw, _ := json.Marshal(map[string]any{
		"state":      "blocked_quota",
		"updated_at": now.Format(time.RFC3339Nano),
		"alive":      true,
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":        true,
		"heartbeat_at": now.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(manifestPath, append(manifestRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
		"driver":                "tmux_cli_session",
	})

	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "Gemma1", Rol: "programador", Activo: true, Habilitado: true, SinCuotaProveedor: true, EstadoCuota: "enfriamiento"},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 53, Agente: "Gemma1", LogicalState: "esperando_io", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 63, Agente: "Gemma1", Estado: "activo", UpdatedAt: now, MetadataJSON: string(metaJSON)},
		},
		tasks: []*db.Tarea{
			{ID: 73, Agente: ptr("Gemma1"), Estado: db.EstadoEnProgreso},
		},
	}

	rows, err := NewService(store, nil).BuildPanelRows()
	if err != nil {
		t.Fatalf("BuildPanelRows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(rows))
	}
	if rows[0].EstadoOperativo == "bloqueado_por_cuota" {
		t.Fatalf("un agente local sin cuota no deberia quedar bloqueado por cuota: %+v", rows[0])
	}
}

func TestBuildPanelRowsNoMarcaAtascadoSiHayProgresoRecienteAunqueNoHayaSalida(t *testing.T) {
	now := time.Now().UTC()
	lastOutput := now.Add(-25 * time.Minute)
	lastProgress := now.Add(-2 * time.Minute)
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "manifest.json")
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")
	manifestRaw, _ := json.Marshal(map[string]any{
		"driver":       "tmux_cli_session",
		"transport":    "tmux",
		"tmux_session": "orq-codex12-120000",
		"tmux_pane_id": "%1",
	})
	statusRaw, _ := json.Marshal(map[string]any{
		"state":            "running",
		"updated_at":       now.Format(time.RFC3339Nano),
		"alive":            true,
		"last_output_at":   lastOutput.Format(time.RFC3339Nano),
		"last_progress_at": lastProgress.Format(time.RFC3339Nano),
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":            true,
		"heartbeat_at":     now.Format(time.RFC3339Nano),
		"last_output_at":   lastOutput.Format(time.RFC3339Nano),
		"last_progress_at": lastProgress.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(manifestPath, append(manifestRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"worker_manifest_path":  manifestPath,
		"driver":                "tmux_cli_session",
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})

	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "Codex12", Rol: "programador", Activo: true, Habilitado: true},
		},
		sessions: []*db.Sesion{
			{Agente: "Codex12", Herramienta: "codex-cli", Estado: "activa", Inicio: now},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 21, Agente: "Codex12", LogicalState: "esperando_io", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 31, Agente: "Codex12", Estado: "activo", UpdatedAt: now, MetadataJSON: string(metaJSON)},
		},
		tasks: []*db.Tarea{
			{ID: 70, Agente: ptr("Codex12"), Estado: db.EstadoEnProgreso},
		},
	}

	rows, err := NewService(store, nil).BuildPanelRows()
	if err != nil {
		t.Fatalf("BuildPanelRows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(rows))
	}
	if rows[0].EstadoOperativo != "trabajando" {
		t.Fatalf("estado operativo=%q", rows[0].EstadoOperativo)
	}
}

func TestBuildPanelRowsMarcaAtascadoSiElProgresoEstaStaleAunqueHayaSalidaReciente(t *testing.T) {
	now := time.Now().UTC()
	lastOutput := now.Add(-2 * time.Minute)
	lastProgress := now.Add(-25 * time.Minute)
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "manifest.json")
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")
	manifestRaw, _ := json.Marshal(map[string]any{
		"driver":       "tmux_cli_session",
		"transport":    "tmux",
		"tmux_session": "orq-codex13-120000",
		"tmux_pane_id": "%2",
	})
	statusRaw, _ := json.Marshal(map[string]any{
		"state":            "running",
		"updated_at":       now.Format(time.RFC3339Nano),
		"alive":            true,
		"last_output_at":   lastOutput.Format(time.RFC3339Nano),
		"last_progress_at": lastProgress.Format(time.RFC3339Nano),
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":            true,
		"heartbeat_at":     now.Format(time.RFC3339Nano),
		"last_output_at":   lastOutput.Format(time.RFC3339Nano),
		"last_progress_at": lastProgress.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(manifestPath, append(manifestRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"worker_manifest_path":  manifestPath,
		"driver":                "tmux_cli_session",
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})

	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "Codex13", Rol: "programador", Activo: true, Habilitado: true},
		},
		sessions: []*db.Sesion{
			{Agente: "Codex13", Herramienta: "codex-cli", Estado: "activa", Inicio: now},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 21, Agente: "Codex13", LogicalState: "esperando_io", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 31, Agente: "Codex13", Estado: "activo", UpdatedAt: now, MetadataJSON: string(metaJSON)},
		},
		tasks: []*db.Tarea{
			{ID: 70, Agente: ptr("Codex13"), Estado: db.EstadoEnProgreso},
		},
	}

	rows, err := NewService(store, nil).BuildPanelRows()
	if err != nil {
		t.Fatalf("BuildPanelRows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(rows))
	}
	if rows[0].EstadoOperativo != "atascado" {
		t.Fatalf("estado operativo=%q", rows[0].EstadoOperativo)
	}
	if !strings.Contains(rows[0].DetalleOperativo, "sin progreso") {
		t.Fatalf("detalle operativo=%q", rows[0].DetalleOperativo)
	}
}

func TestBuildPanelRowsMarcaCheckpointPausaPorCuotaComoBloqueadoPorCuota(t *testing.T) {
	now := time.Now().UTC()
	tmp := t.TempDir()
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")
	statusRaw, _ := json.Marshal(map[string]any{
		"state":      "running",
		"updated_at": now.Format(time.RFC3339Nano),
		"alive":      true,
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":        true,
		"heartbeat_at": now.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})

	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "Codex7", Rol: "programador", Activo: true, Habilitado: true, EstadoCuota: "activo"},
		},
		sessions: []*db.Sesion{
			{Agente: "Codex7", Herramienta: "codex-cli", Estado: "activa", Inicio: now},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 21, Agente: "Codex7", LogicalState: "esperando_io", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 31, Agente: "Codex7", Estado: "activo", UpdatedAt: now, MetadataJSON: string(metaJSON)},
		},
		checkpoints: []*db.RuntimeCheckpoint{
			{
				ID:             90,
				Agente:         "Codex7",
				CheckpointKind: "pause",
				Resumen:        "Cuota agotada (enfriamiento). Reanimación programada a las 19:28",
				Source:         "runtime_order:84053:pause",
				CreatedAt:      now,
			},
		},
		tasks: []*db.Tarea{
			{ID: 70, Agente: ptr("Codex7"), Estado: db.EstadoEnProgreso},
		},
	}

	rows, err := NewService(store, nil).BuildPanelRows()
	if err != nil {
		t.Fatalf("BuildPanelRows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(rows))
	}
	if rows[0].EstadoOperativo != "bloqueado_por_cuota" {
		t.Fatalf("estado operativo=%q", rows[0].EstadoOperativo)
	}
	if !strings.Contains(strings.ToLower(rows[0].DetalleOperativo), "enfriamiento") {
		t.Fatalf("detalle operativo=%q", rows[0].DetalleOperativo)
	}
}

func TestBuildPanelRowsMarcaOrdenControlPendienteComoBloqueadoPorRuntime(t *testing.T) {
	now := time.Now().UTC()
	tmp := t.TempDir()
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")
	statusRaw, _ := json.Marshal(map[string]any{
		"state":      "running",
		"updated_at": now.Add(-2 * time.Minute).Format(time.RFC3339Nano),
		"alive":      true,
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":        true,
		"heartbeat_at": now.Add(-2 * time.Minute).Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})

	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "Codex3", Rol: "programador", Activo: true, Habilitado: true, EstadoCuota: "activo"},
		},
		sessions: []*db.Sesion{
			{Agente: "Codex3", Herramienta: "codex-cli", Estado: "activa", Inicio: now},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 21, Agente: "Codex3", LogicalState: "esperando_io", UpdatedAt: now.Add(-2 * time.Minute)},
		},
		handles: []*db.RuntimeHandle{
			{ID: 31, Agente: "Codex3", Estado: "activo", UpdatedAt: now.Add(-2 * time.Minute), MetadataJSON: string(metaJSON)},
		},
		orders: []*db.RuntimeOrder{
			{ID: 80, Agente: "Codex3", Tipo: "stop", Estado: "pendiente", CreatedAt: now.Add(-30 * time.Second), UpdatedAt: now.Add(-30 * time.Second), AvailableAt: now.Add(-30 * time.Second)},
		},
		tasks: []*db.Tarea{
			{ID: 70, Agente: ptr("Codex3"), Estado: db.EstadoEnProgreso},
		},
	}

	rows, err := NewService(store, nil).BuildPanelRows()
	if err != nil {
		t.Fatalf("BuildPanelRows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(rows))
	}
	if rows[0].EstadoOperativo != "bloqueado_por_runtime" {
		t.Fatalf("estado operativo=%q", rows[0].EstadoOperativo)
	}
	if !strings.Contains(rows[0].DetalleOperativo, "stop") {
		t.Fatalf("detalle operativo=%q", rows[0].DetalleOperativo)
	}
}

func TestBuildPanelRowsNoBloqueaSiWorkerYaSeRecuperoTrasOrdenControlPendiente(t *testing.T) {
	now := time.Now().UTC()
	tmp := t.TempDir()
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")
	statusRaw, _ := json.Marshal(map[string]any{
		"state":      "running",
		"updated_at": now.Format(time.RFC3339Nano),
		"alive":      true,
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":        true,
		"heartbeat_at": now.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})

	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "Codex3", Rol: "programador", Activo: true, Habilitado: true, EstadoCuota: "activo"},
		},
		sessions: []*db.Sesion{
			{Agente: "Codex3", Herramienta: "codex-cli", Estado: "activa", Inicio: now},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 21, Agente: "Codex3", LogicalState: "esperando_io", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 31, Agente: "Codex3", Estado: "activo", UpdatedAt: now, MetadataJSON: string(metaJSON)},
		},
		orders: []*db.RuntimeOrder{
			{ID: 80, Agente: "Codex3", Tipo: "stop", Estado: "pendiente", CreatedAt: now.Add(-2 * time.Minute), UpdatedAt: now.Add(-2 * time.Minute), AvailableAt: now.Add(-2 * time.Minute)},
		},
		tasks: []*db.Tarea{
			{ID: 70, Agente: ptr("Codex3"), Estado: db.EstadoEnProgreso},
		},
	}

	rows, err := NewService(store, nil).BuildPanelRows()
	if err != nil {
		t.Fatalf("BuildPanelRows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(rows))
	}
	if rows[0].EstadoOperativo == "bloqueado_por_runtime" && strings.Contains(rows[0].DetalleOperativo, "stop") {
		t.Fatalf("el worker ya se recuperó tras la orden de control y no deberia seguir bloqueado: %+v", rows[0])
	}
}

func TestBuildPanelRowsRespetaUmbralConfiguradoDeSalidaStale(t *testing.T) {
	now := time.Now().UTC()
	lastOutput := now.Add(-2 * time.Minute)
	tmp := t.TempDir()
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")
	statusRaw, _ := json.Marshal(map[string]any{
		"state":          "running",
		"updated_at":     now.Format(time.RFC3339Nano),
		"alive":          true,
		"last_output_at": lastOutput.Format(time.RFC3339Nano),
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":          true,
		"heartbeat_at":   now.Format(time.RFC3339Nano),
		"last_output_at": lastOutput.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})

	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "Codex12", Rol: "programador", Activo: true, Habilitado: true},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 22, Agente: "Codex12", LogicalState: "esperando_io", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 32, Agente: "Codex12", Estado: "activo", UpdatedAt: now, MetadataJSON: string(metaJSON)},
		},
		tasks: []*db.Tarea{
			{ID: 72, Agente: ptr("Codex12"), Estado: db.EstadoEnProgreso},
		},
		config: map[string]string{
			"runtime_worker_output_stale_seconds": "60",
		},
	}

	rows, err := NewService(store, nil).BuildPanelRows()
	if err != nil {
		t.Fatalf("BuildPanelRows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(rows))
	}
	if rows[0].EstadoOperativo != "atascado" {
		t.Fatalf("estado operativo=%q", rows[0].EstadoOperativo)
	}
}

func TestBuildPanelRowsNoTapaRuntimePausadoConWorkerFresh(t *testing.T) {
	now := time.Now().UTC()
	tmp := t.TempDir()
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")
	statusRaw, _ := json.Marshal(map[string]any{
		"state":      "running",
		"updated_at": now.Format(time.RFC3339Nano),
		"alive":      true,
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":        true,
		"heartbeat_at": now.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})

	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "Codex7", Rol: "programador", Activo: true, Habilitado: true},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 21, Agente: "Codex7", LogicalState: "pausado", ProcessState: "stopped", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 31, Agente: "Codex7", Estado: "pausado", UpdatedAt: now, MetadataJSON: string(metaJSON)},
		},
	}

	rows, err := NewService(store, nil).BuildPanelRows()
	if err != nil {
		t.Fatalf("BuildPanelRows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(rows))
	}
	if rows[0].WorkerState != "running" || !rows[0].WorkerAlive {
		t.Fatalf("worker snapshot inesperado: %+v", rows[0])
	}
	if rows[0].EstadoOperativo != "bloqueado_por_runtime" {
		t.Fatalf("estado operativo=%q", rows[0].EstadoOperativo)
	}
	if rows[0].DetalleOperativo != "pausado" {
		t.Fatalf("detalle operativo=%q", rows[0].DetalleOperativo)
	}
}

func TestBuildPanelRowsNoMarcaDisponibleUnRuntimeRotoSinTareas(t *testing.T) {
	now := time.Now().UTC()
	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "Codex11", Rol: "programador", Activo: true, Habilitado: true},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: 41, Agente: "Codex11", LogicalState: "degradado", ProcessState: "stopped", UpdatedAt: now},
		},
		handles: []*db.RuntimeHandle{
			{ID: 51, Agente: "Codex11", Estado: "fallido", UpdatedAt: now},
		},
	}

	rows, err := NewService(store, nil).BuildPanelRows()
	if err != nil {
		t.Fatalf("BuildPanelRows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(rows))
	}
	if rows[0].EstadoOperativo != "bloqueado_por_runtime" {
		t.Fatalf("estado operativo=%q", rows[0].EstadoOperativo)
	}
	if rows[0].DetalleOperativo != "degradado" {
		t.Fatalf("detalle operativo=%q", rows[0].DetalleOperativo)
	}
}

func TestBuildDetailMergesMailboxWithoutDuplicates(t *testing.T) {
	store := &fakeStore{
		agents: []*db.Agente{{Nombre: "Codex2", Rol: "programador"}},
		tasks: []*db.Tarea{
			{ID: 200, Titulo: "lease activa", Agente: ptr("Codex2"), Estado: db.EstadoEnProgreso, Modulo: "runtime"},
		},
		mailbox: []*db.RuntimeMailboxMessage{
			{ID: 100, FromAgente: "Supervisor", ToAgente: "Codex2"},
			{ID: 101, FromAgente: "Codex2", ToAgente: "Codex1"},
		},
	}

	detail, err := NewService(store, nil).BuildDetail("Codex2")
	if err != nil {
		t.Fatalf("BuildDetail: %v", err)
	}
	if detail.Row.Agente == nil || detail.Row.Agente.Nombre != "Codex2" {
		t.Fatalf("row agent=%+v", detail.Row.Agente)
	}
	if detail.Entity == nil || detail.Entity.Name != "Codex2" {
		t.Fatalf("entity inesperada: %+v", detail.Entity)
	}
	if len(detail.Entity.Leases) != 1 || detail.Entity.Leases[0].TaskID != 200 {
		t.Fatalf("leases inesperadas: %+v", detail.Entity.Leases)
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

func TestBuildReanimationScheduleFiltraVencidasYExponeLeases(t *testing.T) {
	now := time.Now().UTC()
	proyectoID := int64(7)
	store := &fakeStore{
		agents: []*db.Agente{
			{Nombre: "Codex1", Rol: "programador", Activo: true, Habilitado: true, EstadoCuota: "enfriamiento", MotivoPausa: "worker bloqueado por cuota", ReanimarAt: timePtr(now.Add(-2 * time.Minute))},
			{Nombre: "Claude1", Rol: "programador", Activo: true, Habilitado: true, EstadoCuota: "enfriamiento", MotivoPausa: "worker bloqueado por cuota", ReanimarAt: timePtr(now.Add(30 * time.Minute))},
		},
		assignments: []*db.Asignacion{
			{Agente: "Codex1", ProyectoSlug: "orquestador", Estado: db.AsignacionActiva, Nota: "microciclo_exclusivo"},
			{Agente: "Claude1", ProyectoSlug: "orquestador", Estado: db.AsignacionActiva, Nota: "microciclo_exclusivo"},
		},
		tasks: []*db.Tarea{
			{ID: 41, Titulo: "Frente Codex", Estado: db.EstadoBloqueada, ProyectoID: &proyectoID, Agente: ptr("Codex1"), Modulo: "cmd"},
			{ID: 42, Titulo: "Frente Claude", Estado: db.EstadoBloqueada, ProyectoID: &proyectoID, Agente: ptr("Claude1"), Modulo: "db"},
		},
	}

	rows, err := NewService(store, nil).BuildReanimationSchedule(false, false)
	if err != nil {
		t.Fatalf("BuildReanimationSchedule(false): %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows due = %d, want 1", len(rows))
	}
	if rows[0].Name != "Codex1" || !rows[0].Due {
		t.Fatalf("due row inesperada: %+v", rows[0])
	}
	if rows[0].AssignmentProject != "orquestador" {
		t.Fatalf("assignment project inesperado: %+v", rows[0])
	}
	if len(rows[0].Leases) != 1 || rows[0].Leases[0].TaskID != 41 {
		t.Fatalf("leases inesperadas: %+v", rows[0].Leases)
	}

	allRows, err := NewService(store, nil).BuildReanimationSchedule(true, false)
	if err != nil {
		t.Fatalf("BuildReanimationSchedule(true): %v", err)
	}
	if len(allRows) != 2 {
		t.Fatalf("all rows = %d, want 2", len(allRows))
	}
	if !allRows[0].Due || allRows[1].Due {
		t.Fatalf("orden due/future inesperado: %+v", allRows)
	}

	store.agents[1].Habilitado = false
	activeRows, err := NewService(store, nil).BuildReanimationSchedule(true, true)
	if err != nil {
		t.Fatalf("BuildReanimationSchedule(true,true): %v", err)
	}
	if len(activeRows) != 1 || activeRows[0].Name != "Codex1" {
		t.Fatalf("active rows inesperadas: %+v", activeRows)
	}

	store.agents[1].Habilitado = true
	store.agents = append(store.agents, &db.Agente{
		Nombre:      "CodexHist",
		Rol:         "programador",
		Activo:      false,
		Habilitado:  true,
		EstadoCuota: "enfriamiento",
		MotivoPausa: "worker bloqueado por cuota",
		ReanimarAt:  timePtr(now.Add(10 * time.Minute)),
	})
	activeRows, err = NewService(store, nil).BuildReanimationSchedule(true, true)
	if err != nil {
		t.Fatalf("BuildReanimationSchedule(true,true) con historico: %v", err)
	}
	if len(activeRows) != 2 {
		t.Fatalf("active rows con historico=%d, want 2", len(activeRows))
	}
}

func TestInvestigateAgrupaContextoYEvidencia(t *testing.T) {
	proyectoID := int64(42)
	handleID := int64(77)
	runtimeID := int64(55)
	created := time.Now().UTC()
	metaJSON, _ := json.Marshal(map[string]any{
		"trace_dir":      "/repo/.orquesta-runtime/codex2/20260329-120000-000000001",
		"trace_manifest": "/repo/.orquesta-runtime/codex2/20260329-120000-000000001/runtime.json",
		"log_path":       "/repo/.orquesta-runtime/codex2/20260329-120000-000000001/pty.log",
		"working_dir":    "/repo",
	})

	store := &fakeStore{
		project: &db.Proyecto{ID: proyectoID, Slug: "orquestador", Nombre: "Orquestador"},
		agents: []*db.Agente{
			{Nombre: "Codex2", Rol: "programador"},
		},
		runtimes: []*db.RuntimeInstance{
			{ID: runtimeID, Agente: "Codex2", ProyectoID: &proyectoID, CWD: "/repo", UpdatedAt: created},
		},
		handles: []*db.RuntimeHandle{
			{ID: handleID, Agente: "Codex2", ProyectoID: &proyectoID, MetadataJSON: string(metaJSON)},
		},
		transcript: []*db.RuntimeTranscriptEntry{
			{ID: 90, RuntimeID: runtimeID, HandleID: &handleID, Agente: "Codex2", ProyectoID: &proyectoID, Text: "He implementado el refactor del router", CreatedAt: created},
		},
		checkpoints: []*db.RuntimeCheckpoint{
			{ID: 33, Agente: "Codex2", ProyectoID: &proyectoID, Resumen: "router listo", CreatedAt: created},
		},
		tasks: []*db.Tarea{
			{ID: 12, Agente: ptr("Codex2"), Estado: db.TareaEnProgreso},
			{ID: 13, Agente: ptr("Codex2"), Estado: db.TareaCompletada},
		},
	}

	report, err := NewService(store, nil).Investigate("refactor router", "orquestador", 20)
	if err != nil {
		t.Fatalf("Investigate: %v", err)
	}
	if store.lastTranscript.Query == nil || *store.lastTranscript.Query != "refactor router" {
		t.Fatalf("query transcript inesperada: %+v", store.lastTranscript)
	}
	if store.lastTranscript.ProyectoID == nil || *store.lastTranscript.ProyectoID != proyectoID {
		t.Fatalf("proyecto transcript inesperado: %+v", store.lastTranscript)
	}
	if report.TotalMatches != 1 || len(report.Results) != 1 {
		t.Fatalf("report inesperado: %+v", report)
	}
	result := report.Results[0]
	if result.Agente == nil || result.Agente.Nombre != "Codex2" {
		t.Fatalf("agente inesperado: %+v", result.Agente)
	}
	if result.OpenTasks != 1 {
		t.Fatalf("open tasks inesperado: %d", result.OpenTasks)
	}
	if result.LastCheckpoint == nil || result.LastCheckpoint.ID != 33 {
		t.Fatalf("checkpoint inesperado: %+v", result.LastCheckpoint)
	}
	if len(result.Matches) != 1 {
		t.Fatalf("matches inesperados: %+v", result.Matches)
	}
	match := result.Matches[0]
	if match.TraceDir == "" || !strings.Contains(match.TraceDir, ".orquesta-runtime/codex2") {
		t.Fatalf("trace dir inesperado: %+v", match)
	}
	if match.TraceManifest == "" || !strings.HasSuffix(match.TraceManifest, "/runtime.json") {
		t.Fatalf("trace manifest inesperado: %+v", match)
	}
	if match.LogPath == "" || !strings.HasSuffix(match.LogPath, "/pty.log") {
		t.Fatalf("log path inesperado: %+v", match)
	}
}

func TestResolvePrepareConnectorOllamaUsaConectorPorDefectoDelAgente(t *testing.T) {
	store := &fakeStore{
		connector: &db.Conector{
			ID:           44,
			Slug:         "ollama-cli",
			Nombre:       "Ollama CLI",
			Transporte:   "cli",
			Comando:      "ollama",
			ArgsJSON:     `["run"]`,
			MetadataJSON: `{"default_model":"qwen2.5-coder:7b","default_reasoning_effort":"medium","model_positional":true}`,
			Activo:       true,
		},
	}

	conector, err := NewService(store, nil).resolvePrepareConnector("Ollama1", "", nil)
	if err != nil {
		t.Fatalf("resolvePrepareConnector: %v", err)
	}
	if store.lastConnectorRef != "ollama-cli" {
		t.Fatalf("conector por defecto inesperado: %s", store.lastConnectorRef)
	}
	if conector == nil || conector.Slug != "ollama-cli" {
		t.Fatalf("conector inesperado: %+v", conector)
	}
}

func ptr(value string) *string {
	return &value
}

func TestResolvePrepareConnectorIgnoraSesionContaminadaIncompatible(t *testing.T) {
	store := &fakeStore{
		connector: &db.Conector{
			ID:           71,
			Slug:         "claude-code",
			Nombre:       "Claude Code",
			Transporte:   "cli",
			Comando:      "claude-code",
			MetadataJSON: `{"familia":"anthropic"}`,
			Activo:       true,
		},
	}

	conector, err := NewService(store, nil).resolvePrepareConnector("Claude1", "", &db.Sesion{
		ConectorSlug: "ollama_pool_local",
		Herramienta:  "ollama_pool_local",
	})
	if err != nil {
		t.Fatalf("resolvePrepareConnector: %v", err)
	}
	if store.lastConnectorRef != "claude-code" {
		t.Fatalf("conector resuelto=%q; want claude-code", store.lastConnectorRef)
	}
	if conector == nil || conector.Slug != "claude-code" {
		t.Fatalf("conector inesperado: %+v", conector)
	}
}
