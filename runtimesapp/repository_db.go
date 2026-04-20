package runtimesapp

import "orquesta/db"

type Repository struct{}

func (Repository) GetProject(ref string) (*db.Proyecto, error) {
	return db.GetProyectoConRutaEfectiva(ref, "")
}

func (Repository) GetAgent(nombre string) (*db.Agente, error) {
	return db.GetAgente(nombre)
}

func (Repository) SharedAccountActivationAllowed(agente string) (bool, string, error) {
	return db.CuentaCompartidaPermiteActivacionAgente(agente)
}

func (Repository) ReconcileStaleRuntimeHandles() (int, error) {
	return db.ReconciliarRuntimeHandlesStale()
}

func (Repository) ReconcileStaleRuntimeOrders() (int, error) {
	return db.ReconciliarRuntimeOrdersStale()
}

func (Repository) ListRuntimes(filtro db.FiltroRuntimes) ([]*db.RuntimeInstance, error) {
	return db.ListarRuntimes(filtro)
}

func (Repository) BuildRuntimeTree(filtro db.FiltroRuntimes) ([]*db.RuntimeTreeNode, error) {
	return db.ConstruirArbolRuntimes(filtro)
}

func (Repository) GetRuntime(id int64) (*db.RuntimeInstance, error) {
	return db.GetRuntime(id)
}

func (Repository) GetRuntimeBySessionID(sessionID int64) (*db.RuntimeInstance, error) {
	return db.GetRuntimeBySesionID(sessionID)
}

func (Repository) GetRuntimeHandleBySessionID(sessionID int64) (*db.RuntimeHandle, error) {
	return db.GetRuntimeHandleBySesionID(sessionID)
}

func (Repository) GetPrimaryRuntime(agente string) (*db.RuntimeInstance, error) {
	return db.GetRuntimePrincipalAgente(agente)
}

func (Repository) GetPrimaryRuntimeForProject(agente string, proyectoID *int64) (*db.RuntimeInstance, error) {
	return db.GetRuntimePrincipalAgenteProyecto(agente, proyectoID)
}

func (Repository) CloseRuntime(id int64) error {
	return db.MarcarRuntimeCerrado(id)
}

func (Repository) CloseRuntimeHandles(agente string, proyectoID *int64) error {
	return db.MarcarRuntimeHandlesCerrados(agente, proyectoID)
}

func (Repository) GetRuntimeHandle(id int64) (*db.RuntimeHandle, error) {
	return db.GetRuntimeHandle(id)
}

func (Repository) ListRuntimeHandles(filtro *string) ([]*db.RuntimeHandle, error) {
	return db.ListarRuntimeHandles(filtro)
}

func (Repository) ListPassiveRuntimeHandles(filtro *string) ([]*db.RuntimeHandle, error) {
	return db.ListarRuntimeHandlesPasivos(filtro)
}

func (Repository) ListCanonicalRuntimeHandles(filtro *string) ([]*db.RuntimeHandle, error) {
	return db.ListarRuntimeHandlesCanonicosRecientes(filtro)
}

func (Repository) SyncSupervisedRuntimeHandle(handle *db.RuntimeHandle, source string) (*db.RuntimeHandle, error) {
	handle, _, _, err := db.SincronizarRuntimeHandleSupervisado(handle, nil, source)
	return handle, err
}

func (Repository) PurgeInactiveRuntimeHandles(filtro db.FiltroPurgadoRuntimeHandles) (*db.PurgaRuntimeHandlesResultado, error) {
	return db.PurgarRuntimeHandlesInactivos(filtro)
}

func (Repository) PurgeTerminalRuntimeOrders(filtro db.FiltroPurgadoRuntimeOrders) (*db.PurgaRuntimeOrdersResultado, error) {
	return db.PurgarRuntimeOrdersTerminales(filtro)
}

func (Repository) GetActiveRuntimeHandle(agente string) (*db.RuntimeHandle, error) {
	return db.GetRuntimeHandleCanonicoRecienteAgente(agente)
}

func (Repository) GetActiveRuntimeHandleForProject(agente string, proyectoID *int64) (*db.RuntimeHandle, error) {
	return db.GetRuntimeHandleCanonicoRecienteAgenteProyecto(agente, proyectoID)
}

func (Repository) GetOperationalRuntimeHandle(agente string) (*db.RuntimeHandle, error) {
	return db.GetRuntimeHandleOperativoRecienteAgente(agente)
}

func (Repository) GetOperationalRuntimeHandleForProject(agente string, proyectoID *int64) (*db.RuntimeHandle, error) {
	return db.GetRuntimeHandleOperativoRecienteAgenteProyecto(agente, proyectoID)
}

func (Repository) ListRuntimeEvents(filtro db.FiltroRuntimeEvents) ([]*db.RuntimeEvent, error) {
	return db.ListarRuntimeEvents(filtro)
}

func (Repository) ListRuntimeTranscript(filtro db.FiltroRuntimeTranscript) ([]*db.RuntimeTranscriptEntry, error) {
	return db.ListarRuntimeTranscript(filtro)
}

func (Repository) RegisterRuntimeTranscript(entry *db.RuntimeTranscriptEntry) (int64, error) {
	return db.RegistrarRuntimeTranscript(entry)
}

func (Repository) ListRuntimeSamples(runtimeID int64, limit int) ([]*db.RuntimeTelemetrySample, error) {
	return db.ListarMuestrasRuntime(runtimeID, limit)
}

func (Repository) CreateRuntimeOrder(order *db.RuntimeOrder) (int64, error) {
	return db.EncolarRuntimeOrder(order)
}

func (Repository) CreateLiveAgentHandoff(origen, destino string, tareaID *int64, motivo, resumenContinuidad, externalSessionID string) (int64, error) {
	return db.CrearHandoffAgenteVivo(origen, destino, tareaID, motivo, resumenContinuidad, externalSessionID)
}

func (Repository) CreateStaleAgentHandoff(origen, destino string, tareaID *int64, motivo, resumenContinuidad, externalSessionID string) (int64, error) {
	return db.CrearHandoffAgenteStale(origen, destino, tareaID, motivo, resumenContinuidad, externalSessionID)
}

func (Repository) ListRuntimeOrders(filtro db.FiltroRuntimeOrders) ([]*db.RuntimeOrder, error) {
	return db.ListarRuntimeOrders(filtro)
}

func (Repository) GetRuntimeOrder(id int64) (*db.RuntimeOrder, error) {
	return db.GetRuntimeOrder(id)
}

func (Repository) MarkRuntimeOrderState(id int64, estado, resultadoJSON, errorText string) error {
	return db.MarcarRuntimeOrderEstado(id, estado, resultadoJSON, errorText)
}

func (Repository) CreateRuntimeMailbox(msg *db.RuntimeMailboxMessage) (int64, error) {
	return db.EnviarRuntimeMailbox(msg)
}

func (Repository) GetRuntimeMailbox(id int64) (*db.RuntimeMailboxMessage, error) {
	return db.GetRuntimeMailbox(id)
}

func (Repository) ListRuntimeMailbox(filtro db.FiltroRuntimeMailbox) ([]*db.RuntimeMailboxMessage, error) {
	return db.ListarRuntimeMailbox(filtro)
}

func (Repository) MarkRuntimeMailboxDelivered(id int64) error {
	return db.MarcarRuntimeMailboxEntregado(id)
}

func (Repository) MarkRuntimeMailboxConsumed(id int64) error {
	return db.MarcarRuntimeMailboxConsumido(id)
}

func (Repository) CreateRuntimeCheckpoint(checkpoint *db.RuntimeCheckpoint) (int64, error) {
	return db.CrearRuntimeCheckpoint(checkpoint)
}

func (Repository) ListRuntimeCheckpoints(filtro db.FiltroRuntimeCheckpoints) ([]*db.RuntimeCheckpoint, error) {
	return db.ListarRuntimeCheckpoints(filtro)
}

func (Repository) GetRuntimeCheckpoint(id int64) (*db.RuntimeCheckpoint, error) {
	return db.GetRuntimeCheckpoint(id)
}

func (Repository) LatestRuntimeCheckpoint(agente string, proyectoID *int64) (*db.RuntimeCheckpoint, error) {
	return db.UltimoRuntimeCheckpoint(agente, proyectoID)
}

func (Repository) ListMemoryEntities(filtro db.FiltroEntidadesMemoria) ([]*db.EntidadMemoria, error) {
	return db.ListarEntidadesMemoria(filtro)
}

func (Repository) UpsertMemoryEntity(entidad *db.EntidadMemoria) (int64, error) {
	return db.UpsertEntidadMemoria(entidad)
}

func (Repository) GetMemoryEntity(nombre string, proyectoID *int64) (*db.EntidadMemoria, error) {
	return db.GetEntidadMemoria(nombre, proyectoID)
}

func (Repository) Audit(agente, accion, entidad string, entidadID int64, detalle string) {
	db.Audit(agente, accion, entidad, entidadID, detalle)
}
