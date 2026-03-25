/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package runtimesapp

import "orquesta/db"

type Store interface {
	GetProject(ref string) (*db.Proyecto, error)
	BuildRuntimeTree(filtro db.FiltroRuntimes) ([]*db.RuntimeTreeNode, error)
	GetRuntime(id int64) (*db.RuntimeInstance, error)
	ListRuntimeSamples(runtimeID int64, limit int) ([]*db.RuntimeTelemetrySample, error)
	ListRuntimeOrders(filtro db.FiltroRuntimeOrders) ([]*db.RuntimeOrder, error)
	ListRuntimeMailbox(filtro db.FiltroRuntimeMailbox) ([]*db.RuntimeMailboxMessage, error)
	ListRuntimeCheckpoints(filtro db.FiltroRuntimeCheckpoints) ([]*db.RuntimeCheckpoint, error)
	GetRuntimeCheckpoint(id int64) (*db.RuntimeCheckpoint, error)
	ListMemoryEntities(filtro db.FiltroEntidadesMemoria) ([]*db.EntidadMemoria, error)
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

func (s *Service) BuildRuntimeTree(filtro db.FiltroRuntimes) ([]*db.RuntimeTreeNode, error) {
	return s.store.BuildRuntimeTree(filtro)
}

func (s *Service) GetRuntime(id int64) (*db.RuntimeInstance, error) {
	return s.store.GetRuntime(id)
}

func (s *Service) ListRuntimeSamples(runtimeID int64, limit int) ([]*db.RuntimeTelemetrySample, error) {
	return s.store.ListRuntimeSamples(runtimeID, limit)
}

func (s *Service) ListRuntimeOrders(filtro db.FiltroRuntimeOrders) ([]*db.RuntimeOrder, error) {
	return s.store.ListRuntimeOrders(filtro)
}

func (s *Service) ListRuntimeMailbox(filtro db.FiltroRuntimeMailbox) ([]*db.RuntimeMailboxMessage, error) {
	return s.store.ListRuntimeMailbox(filtro)
}

func (s *Service) ListRuntimeCheckpoints(filtro db.FiltroRuntimeCheckpoints) ([]*db.RuntimeCheckpoint, error) {
	return s.store.ListRuntimeCheckpoints(filtro)
}

func (s *Service) GetRuntimeCheckpoint(id int64) (*db.RuntimeCheckpoint, error) {
	return s.store.GetRuntimeCheckpoint(id)
}

func (s *Service) ListMemoryEntities(filtro db.FiltroEntidadesMemoria) ([]*db.EntidadMemoria, error) {
	return s.store.ListMemoryEntities(filtro)
}

type Repository struct{}

func (Repository) GetProject(ref string) (*db.Proyecto, error) {
	return db.GetProyecto(ref)
}

func (Repository) BuildRuntimeTree(filtro db.FiltroRuntimes) ([]*db.RuntimeTreeNode, error) {
	return db.ConstruirArbolRuntimes(filtro)
}

func (Repository) GetRuntime(id int64) (*db.RuntimeInstance, error) {
	return db.GetRuntime(id)
}

func (Repository) ListRuntimeSamples(runtimeID int64, limit int) ([]*db.RuntimeTelemetrySample, error) {
	return db.ListarMuestrasRuntime(runtimeID, limit)
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

func (Repository) GetRuntimeCheckpoint(id int64) (*db.RuntimeCheckpoint, error) {
	return db.GetRuntimeCheckpoint(id)
}

func (Repository) ListMemoryEntities(filtro db.FiltroEntidadesMemoria) ([]*db.EntidadMemoria, error) {
	return db.ListarEntidadesMemoria(filtro)
}
