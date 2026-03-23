package runtimectl

import (
	"encoding/json"

	"orquesta/db"
)

type Store interface {
	ResolveProyectoIDBySlug(slug string) (*int64, error)
	SaveRuntimeHandle(h *db.RuntimeHandle) (int64, error)
	ListRuntimeHandles(agente, estado string) ([]*db.RuntimeHandle, error)
	CreateRuntimeOrder(o *db.RuntimeOrder) (int64, error)
	ListRuntimeOrders(agente, estado string) ([]*db.RuntimeOrder, error)
	ExecuteNextRuntimeOrder(agente string) (*db.RuntimeOrder, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

type RegisterHandleInput struct {
	Agente       string
	SesionID     *int64
	ProyectoSlug string
	Transporte   string
	HandleKind   string
	HandleRef    string
	Estado       string
	MetadataJSON string
}

func (s *Service) RegisterHandle(input RegisterHandleInput) (int64, error) {
	proyectoID, err := s.store.ResolveProyectoIDBySlug(input.ProyectoSlug)
	if err != nil {
		return 0, err
	}
	return s.store.SaveRuntimeHandle(&db.RuntimeHandle{
		Agente:       input.Agente,
		SesionID:     input.SesionID,
		ProyectoID:   proyectoID,
		Transporte:   input.Transporte,
		HandleKind:   input.HandleKind,
		HandleRef:    input.HandleRef,
		Estado:       input.Estado,
		MetadataJSON: input.MetadataJSON,
	})
}

func (s *Service) ListHandles(agente, estado string) ([]*db.RuntimeHandle, error) {
	return s.store.ListRuntimeHandles(agente, estado)
}

func (s *Service) EnqueueInstruction(agente, proyectoSlug, mensaje string) (int64, error) {
	payload, _ := json.Marshal(map[string]any{"mensaje": mensaje})
	return s.enqueueOrder(agente, proyectoSlug, "enviar_instruccion", string(payload))
}

func (s *Service) EnqueuePause(agente, proyectoSlug string) (int64, error) {
	return s.enqueueOrder(agente, proyectoSlug, "pausar", "{}")
}

func (s *Service) EnqueueContinue(agente, proyectoSlug string) (int64, error) {
	return s.enqueueOrder(agente, proyectoSlug, "continuar", "{}")
}

func (s *Service) EnqueueHandoff(origen, destino, proyectoSlug string) (int64, error) {
	payload, _ := json.Marshal(map[string]any{"destino": destino})
	return s.enqueueOrder(origen, proyectoSlug, "handoff", string(payload))
}

func (s *Service) ListOrders(agente, estado string) ([]*db.RuntimeOrder, error) {
	return s.store.ListRuntimeOrders(agente, estado)
}

func (s *Service) ExecuteNext(agente string) (*db.RuntimeOrder, error) {
	return s.store.ExecuteNextRuntimeOrder(agente)
}

func (s *Service) enqueueOrder(agente, proyectoSlug, tipo, payloadJSON string) (int64, error) {
	proyectoID, err := s.store.ResolveProyectoIDBySlug(proyectoSlug)
	if err != nil {
		return 0, err
	}
	return s.store.CreateRuntimeOrder(&db.RuntimeOrder{
		Agente:      agente,
		ProyectoID:  proyectoID,
		Tipo:        tipo,
		PayloadJSON: payloadJSON,
	})
}
