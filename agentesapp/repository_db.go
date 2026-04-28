package agentesapp

import (
	"time"

	"orquesta/db"
)

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

func (Repository) GetAgentPrepareLite(nombre string) (*db.Agente, error) {
	return db.GetAgentePrepareLite(nombre)
}

func (Repository) GetProject(ref string) (*db.Proyecto, error) {
	return db.GetProyectoConRutaEfectiva(ref, "")
}

func (Repository) GetProjectPrepareLite(ref string) (*db.Proyecto, error) {
	return db.GetProyectoPrepareLite(ref)
}

func (Repository) GetPool(slug string) (*db.PoolCapacidad, error) {
	return db.GetPool(slug)
}

func (Repository) GetConnector(ref string) (*db.Conector, error) {
	return db.GetConector(ref)
}

func (Repository) GetConnectorPrepareLite(ref string) (*db.Conector, error) {
	return db.GetConectorPrepareLite(ref)
}

func (Repository) ListAgents() ([]*db.Agente, error) {
	return db.ListarAgentes()
}

func (Repository) ListAgentsLight() ([]*db.Agente, error) {
	return db.ListarAgentesEstadoLigero()
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

func (Repository) GetLastSessionPrepareLite(agente string, proyectoID *int64) (*db.Sesion, error) {
	return db.ObtenerUltimaSesionPrepareLite(agente, proyectoID)
}

func (Repository) GetActiveSession(agente string, proyectoID *int64) (*db.Sesion, error) {
	return db.GetSesionActiva(agente, proyectoID)
}

func (Repository) SaveActiveSession(agente string, proyectoID *int64, upd db.SesionUpdate) error {
	return db.GuardarSesionActiva(agente, proyectoID, upd)
}

func (Repository) GetRuntimeBySessionID(sessionID int64) (*db.RuntimeInstance, error) {
	return db.GetRuntimeBySesionID(sessionID)
}

func (Repository) GetRuntimeHandleBySessionID(sessionID int64) (*db.RuntimeHandle, error) {
	return db.GetRuntimeHandleBySesionID(sessionID)
}

func (Repository) ListRuntimes(filtro db.FiltroRuntimes) ([]*db.RuntimeInstance, error) {
	return db.ListarRuntimes(filtro)
}

func (Repository) GetRuntimePrincipalForTick(agente string, proyectoID *int64) (*db.RuntimeInstance, error) {
	return db.GetRuntimePrincipalAgenteProyecto(agente, proyectoID)
}

func (Repository) ListRuntimeHandles(agente *string) ([]*db.RuntimeHandle, error) {
	return db.ListarRuntimeHandles(agente)
}

func (Repository) ListPassiveRuntimeHandles(agente *string) ([]*db.RuntimeHandle, error) {
	return db.ListarRuntimeHandlesPasivos(agente)
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

func (Repository) SummarizeRuntimeMailboxForPanel() ([]*db.RuntimeMailboxPanelSummary, error) {
	return db.ResumirRuntimeMailboxPorAgente()
}

func (Repository) ListLatestAutonomyMailboxForPanel() ([]*db.RuntimeMailboxMessage, error) {
	return db.ListarUltimaAutonomiaMailboxPorAgente()
}

func (Repository) RuntimeMailboxCoveredByBootstrapPending(mailboxID int64, handle *db.RuntimeHandle, runtime *db.RuntimeInstance) (bool, int64, int64, error) {
	return db.RuntimeMailboxCubiertoPorBootstrapPendiente(mailboxID, handle, runtime)
}

func (Repository) ListRuntimeCheckpoints(filtro db.FiltroRuntimeCheckpoints) ([]*db.RuntimeCheckpoint, error) {
	return db.ListarRuntimeCheckpoints(filtro)
}

func (Repository) SummarizeRuntimeCheckpointsForPanel() ([]*db.RuntimeCheckpointPanelSummary, error) {
	return db.ResumirRuntimeCheckpointsPorAgente()
}

func (Repository) ListTasks(filtro db.FiltroTareas) ([]*db.Tarea, error) {
	return db.ListarTareas(filtro)
}

func (Repository) ListTasksForTick(agente string) ([]*db.Tarea, error) {
	return db.ListarTareasNoTerminalesAgente(agente)
}

func (Repository) GetTaskForTick(id int64) (*db.Tarea, error) {
	return db.GetTarea(id)
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

func (Repository) SharedAccountAllowsActivation(nombre string) (bool, string, error) {
	return db.CuentaCompartidaPermiteActivacionAgente(nombre)
}

func (Repository) ConfigGet(clave string) (string, error) {
	return db.ConfigGet(clave)
}

func (Repository) PauseAgent(nombre string, minutos int, motivo string) error {
	return db.PausarAgente(nombre, minutos, motivo)
}
