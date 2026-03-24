package runtimectl

import (
	"testing"

	"orquesta/db"
)

type fakeStore struct {
	projectID  *int64
	lastHandle *db.RuntimeHandle
	lastOrder  *db.RuntimeOrder
}

func (f *fakeStore) ResolveProyectoIDBySlug(slug string) (*int64, error) {
	return f.projectID, nil
}

func (f *fakeStore) SaveRuntimeHandle(h *db.RuntimeHandle) (int64, error) {
	f.lastHandle = h
	return 1, nil
}

func (f *fakeStore) ListRuntimeHandles(agente, estado string) ([]*db.RuntimeHandle, error) {
	return nil, nil
}

func (f *fakeStore) CreateRuntimeOrder(o *db.RuntimeOrder) (int64, error) {
	f.lastOrder = o
	return 2, nil
}

func (f *fakeStore) ListRuntimeOrders(agente, estado string) ([]*db.RuntimeOrder, error) {
	return nil, nil
}

func (f *fakeStore) ExecuteNextRuntimeOrder(agente string) (*db.RuntimeOrder, error) {
	return nil, nil
}

func TestRegisterHandleResuelveProyecto(t *testing.T) {
	projectID := int64(7)
	store := &fakeStore{projectID: &projectID}
	svc := NewService(store)

	if _, err := svc.RegisterHandle(RegisterHandleInput{
		Agente:       "Codex2",
		ProyectoSlug: "orquestador",
		Transporte:   "local",
		HandleKind:   "pty",
		HandleRef:    "/tmp/tty",
		Estado:       "activo",
		MetadataJSON: "{}",
	}); err != nil {
		t.Fatalf("RegisterHandle: %v", err)
	}
	if store.lastHandle == nil || store.lastHandle.ProyectoID == nil || *store.lastHandle.ProyectoID != projectID {
		t.Fatalf("proyecto no resuelto en handle: %+v", store.lastHandle)
	}
}

func TestEnqueueInstructionConstruyeRuntimeOrder(t *testing.T) {
	store := &fakeStore{}
	svc := NewService(store)

	if _, err := svc.EnqueueInstruction("Codex2", "", "hola"); err != nil {
		t.Fatalf("EnqueueInstruction: %v", err)
	}
	if store.lastOrder == nil {
		t.Fatalf("no se creo runtime order")
	}
	if store.lastOrder.Tipo != "enviar_instruccion" {
		t.Fatalf("tipo inesperado: %+v", store.lastOrder)
	}
	if store.lastOrder.PayloadJSON == "{}" {
		t.Fatalf("payload vacio: %+v", store.lastOrder)
	}
}
