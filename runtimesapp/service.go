package runtimesapp

import "orquesta/db"

type Store interface {
	GetProject(ref string) (*db.Proyecto, error)
	ListRuntimes(filtro db.FiltroRuntimes) ([]*db.RuntimeInstance, error)
	BuildRuntimeTree(filtro db.FiltroRuntimes) ([]*db.RuntimeTreeNode, error)
	GetRuntime(id int64) (*db.RuntimeInstance, error)
	ListRuntimeHandles(filtro *string) ([]*db.RuntimeHandle, error)
	GetActiveRuntimeHandle(agente string) (*db.RuntimeHandle, error)
	GetActiveRuntimeHandleForProject(agente string, proyectoID *int64) (*db.RuntimeHandle, error)
	ListRuntimeSamples(runtimeID int64, limit int) ([]*db.RuntimeTelemetrySample, error)
	CreateRuntimeOrder(order *db.RuntimeOrder) (int64, error)
	ListRuntimeOrders(filtro db.FiltroRuntimeOrders) ([]*db.RuntimeOrder, error)
	CreateRuntimeMailbox(msg *db.RuntimeMailboxMessage) (int64, error)
	ListRuntimeMailbox(filtro db.FiltroRuntimeMailbox) ([]*db.RuntimeMailboxMessage, error)
	MarkRuntimeMailboxDelivered(id int64) error
	MarkRuntimeMailboxConsumed(id int64) error
	CreateRuntimeCheckpoint(checkpoint *db.RuntimeCheckpoint) (int64, error)
	ListRuntimeCheckpoints(filtro db.FiltroRuntimeCheckpoints) ([]*db.RuntimeCheckpoint, error)
	GetRuntimeCheckpoint(id int64) (*db.RuntimeCheckpoint, error)
	LatestRuntimeCheckpoint(agente string, proyectoID *int64) (*db.RuntimeCheckpoint, error)
	ListMemoryEntities(filtro db.FiltroEntidadesMemoria) ([]*db.EntidadMemoria, error)
	UpsertMemoryEntity(entidad *db.EntidadMemoria) (int64, error)
	GetMemoryEntity(nombre string, proyectoID *int64) (*db.EntidadMemoria, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) GetProject(ref string) (*db.Proyecto, error) {
	return s.store.GetProject(ref)
}

func (s *Service) ListRuntimes(filtro db.FiltroRuntimes) ([]*db.RuntimeInstance, error) {
	return s.store.ListRuntimes(filtro)
}

func (s *Service) BuildRuntimeTree(filtro db.FiltroRuntimes) ([]*db.RuntimeTreeNode, error) {
	return s.store.BuildRuntimeTree(filtro)
}

func (s *Service) GetRuntime(id int64) (*db.RuntimeInstance, error) {
	return s.store.GetRuntime(id)
}

func (s *Service) ListRuntimeHandles(filtro *string) ([]*db.RuntimeHandle, error) {
	return s.store.ListRuntimeHandles(filtro)
}

func (s *Service) GetActiveRuntimeHandle(agente string) (*db.RuntimeHandle, error) {
	return s.store.GetActiveRuntimeHandle(agente)
}

func (s *Service) GetActiveRuntimeHandleForProject(agente string, proyectoID *int64) (*db.RuntimeHandle, error) {
	return s.store.GetActiveRuntimeHandleForProject(agente, proyectoID)
}

func (s *Service) ListRuntimeSamples(runtimeID int64, limit int) ([]*db.RuntimeTelemetrySample, error) {
	return s.store.ListRuntimeSamples(runtimeID, limit)
}

func (s *Service) CreateRuntimeOrder(order *db.RuntimeOrder) (int64, error) {
	return s.store.CreateRuntimeOrder(order)
}

func (s *Service) ListRuntimeOrders(filtro db.FiltroRuntimeOrders) ([]*db.RuntimeOrder, error) {
	return s.store.ListRuntimeOrders(filtro)
}

func (s *Service) CreateRuntimeMailbox(msg *db.RuntimeMailboxMessage) (int64, error) {
	return s.store.CreateRuntimeMailbox(msg)
}

func (s *Service) ListRuntimeMailbox(filtro db.FiltroRuntimeMailbox) ([]*db.RuntimeMailboxMessage, error) {
	return s.store.ListRuntimeMailbox(filtro)
}

func (s *Service) MarkRuntimeMailboxDelivered(id int64) error {
	return s.store.MarkRuntimeMailboxDelivered(id)
}

func (s *Service) MarkRuntimeMailboxConsumed(id int64) error {
	return s.store.MarkRuntimeMailboxConsumed(id)
}

func (s *Service) CreateRuntimeCheckpoint(checkpoint *db.RuntimeCheckpoint) (int64, error) {
	return s.store.CreateRuntimeCheckpoint(checkpoint)
}

func (s *Service) ListRuntimeCheckpoints(filtro db.FiltroRuntimeCheckpoints) ([]*db.RuntimeCheckpoint, error) {
	return s.store.ListRuntimeCheckpoints(filtro)
}

func (s *Service) GetRuntimeCheckpoint(id int64) (*db.RuntimeCheckpoint, error) {
	return s.store.GetRuntimeCheckpoint(id)
}

func (s *Service) LatestRuntimeCheckpoint(agente string, proyectoID *int64) (*db.RuntimeCheckpoint, error) {
	return s.store.LatestRuntimeCheckpoint(agente, proyectoID)
}

func (s *Service) ListMemoryEntities(filtro db.FiltroEntidadesMemoria) ([]*db.EntidadMemoria, error) {
	return s.store.ListMemoryEntities(filtro)
}

func (s *Service) UpsertMemoryEntity(entidad *db.EntidadMemoria) (int64, error) {
	return s.store.UpsertMemoryEntity(entidad)
}

func (s *Service) GetMemoryEntity(nombre string, proyectoID *int64) (*db.EntidadMemoria, error) {
	return s.store.GetMemoryEntity(nombre, proyectoID)
}

type Repository struct{}

func (Repository) GetProject(ref string) (*db.Proyecto, error) {
	return db.GetProyecto(ref)
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

func (Repository) ListRuntimeHandles(filtro *string) ([]*db.RuntimeHandle, error) {
	return db.ListarRuntimeHandles(filtro)
}

func (Repository) GetActiveRuntimeHandle(agente string) (*db.RuntimeHandle, error) {
	return db.GetRuntimeHandleActivoAgente(agente)
}

func (Repository) GetActiveRuntimeHandleForProject(agente string, proyectoID *int64) (*db.RuntimeHandle, error) {
	return db.GetRuntimeHandleActivoAgenteProyecto(agente, proyectoID)
}

func (Repository) ListRuntimeSamples(runtimeID int64, limit int) ([]*db.RuntimeTelemetrySample, error) {
	return db.ListarMuestrasRuntime(runtimeID, limit)
}

func (Repository) CreateRuntimeOrder(order *db.RuntimeOrder) (int64, error) {
	return db.EncolarRuntimeOrder(order)
}

func (Repository) ListRuntimeOrders(filtro db.FiltroRuntimeOrders) ([]*db.RuntimeOrder, error) {
	return db.ListarRuntimeOrders(filtro)
}

func (Repository) CreateRuntimeMailbox(msg *db.RuntimeMailboxMessage) (int64, error) {
	return db.EnviarRuntimeMailbox(msg)
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
