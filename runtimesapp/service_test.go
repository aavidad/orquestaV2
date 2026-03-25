package runtimesapp

import (
	"testing"

	"orquesta/db"
)

type fakeStore struct {
	projectRef       string
	projectResponse  *db.Proyecto
	treeFilter       db.FiltroRuntimes
	treeResponse     []*db.RuntimeTreeNode
	runtimeID        int64
	runtimeResponse  *db.RuntimeInstance
	samplesID        int64
	samplesLimit     int
	samplesResponse  []*db.RuntimeTelemetrySample
	ordersFilter     db.FiltroRuntimeOrders
	ordersResponse   []*db.RuntimeOrder
	mailboxFilter    db.FiltroRuntimeMailbox
	mailboxResponse  []*db.RuntimeMailboxMessage
	checkFilter      db.FiltroRuntimeCheckpoints
	checkResponse    []*db.RuntimeCheckpoint
	checkpointID     int64
	checkpointResp   *db.RuntimeCheckpoint
	memoryFilter     db.FiltroEntidadesMemoria
	memoryResponse   []*db.EntidadMemoria
}

func (f *fakeStore) GetProject(ref string) (*db.Proyecto, error) {
	f.projectRef = ref
	return f.projectResponse, nil
}
func (f *fakeStore) BuildRuntimeTree(filter db.FiltroRuntimes) ([]*db.RuntimeTreeNode, error) {
	f.treeFilter = filter
	return f.treeResponse, nil
}
func (f *fakeStore) GetRuntime(id int64) (*db.RuntimeInstance, error) {
	f.runtimeID = id
	return f.runtimeResponse, nil
}
func (f *fakeStore) ListRuntimeSamples(runtimeID int64, limit int) ([]*db.RuntimeTelemetrySample, error) {
	f.samplesID = runtimeID
	f.samplesLimit = limit
	return f.samplesResponse, nil
}
func (f *fakeStore) ListRuntimeOrders(filter db.FiltroRuntimeOrders) ([]*db.RuntimeOrder, error) {
	f.ordersFilter = filter
	return f.ordersResponse, nil
}
func (f *fakeStore) ListRuntimeMailbox(filter db.FiltroRuntimeMailbox) ([]*db.RuntimeMailboxMessage, error) {
	f.mailboxFilter = filter
	return f.mailboxResponse, nil
}
func (f *fakeStore) ListRuntimeCheckpoints(filter db.FiltroRuntimeCheckpoints) ([]*db.RuntimeCheckpoint, error) {
	f.checkFilter = filter
	return f.checkResponse, nil
}
func (f *fakeStore) GetRuntimeCheckpoint(id int64) (*db.RuntimeCheckpoint, error) {
	f.checkpointID = id
	return f.checkpointResp, nil
}
func (f *fakeStore) ListMemoryEntities(filter db.FiltroEntidadesMemoria) ([]*db.EntidadMemoria, error) {
	f.memoryFilter = filter
	return f.memoryResponse, nil
}

func TestServiceDelegatesRuntimeQueries(t *testing.T) {
	agent := "Codex2"
	projectID := int64(7)
	store := &fakeStore{
		projectResponse: &db.Proyecto{Slug: "orquestador"},
		treeResponse:    []*db.RuntimeTreeNode{{Runtime: &db.RuntimeInstance{ID: 10}}},
		runtimeResponse: &db.RuntimeInstance{ID: 10},
		samplesResponse: []*db.RuntimeTelemetrySample{{ID: 20}},
		ordersResponse:  []*db.RuntimeOrder{{ID: 30}},
		mailboxResponse: []*db.RuntimeMailboxMessage{{ID: 40}},
		checkResponse:   []*db.RuntimeCheckpoint{{ID: 50}},
		checkpointResp:  &db.RuntimeCheckpoint{ID: 50},
		memoryResponse:  []*db.EntidadMemoria{{ID: 60}},
	}
	service := NewService(store)

	if _, err := service.GetProject("orquestador"); err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if store.projectRef != "orquestador" {
		t.Fatalf("projectRef=%q", store.projectRef)
	}
	if _, err := service.BuildRuntimeTree(db.FiltroRuntimes{Agente: &agent}); err != nil {
		t.Fatalf("BuildRuntimeTree: %v", err)
	}
	if store.treeFilter.Agente == nil || *store.treeFilter.Agente != "Codex2" {
		t.Fatalf("treeFilter=%+v", store.treeFilter)
	}
	if _, err := service.GetRuntime(10); err != nil {
		t.Fatalf("GetRuntime: %v", err)
	}
	if store.runtimeID != 10 {
		t.Fatalf("runtimeID=%d", store.runtimeID)
	}
	if _, err := service.ListRuntimeSamples(10, 25); err != nil {
		t.Fatalf("ListRuntimeSamples: %v", err)
	}
	if store.samplesID != 10 || store.samplesLimit != 25 {
		t.Fatalf("samples runtime=%d limit=%d", store.samplesID, store.samplesLimit)
	}
	if _, err := service.ListRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agent, ProyectoID: &projectID}); err != nil {
		t.Fatalf("ListRuntimeOrders: %v", err)
	}
	if store.ordersFilter.Agente == nil || *store.ordersFilter.Agente != "Codex2" {
		t.Fatalf("ordersFilter=%+v", store.ordersFilter)
	}
	if _, err := service.ListRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agent, ProyectoID: &projectID}); err != nil {
		t.Fatalf("ListRuntimeMailbox: %v", err)
	}
	if store.mailboxFilter.ToAgente == nil || *store.mailboxFilter.ToAgente != "Codex2" {
		t.Fatalf("mailboxFilter=%+v", store.mailboxFilter)
	}
	if _, err := service.ListRuntimeCheckpoints(db.FiltroRuntimeCheckpoints{Agente: &agent, Limit: 5}); err != nil {
		t.Fatalf("ListRuntimeCheckpoints: %v", err)
	}
	if store.checkFilter.Agente == nil || *store.checkFilter.Agente != "Codex2" || store.checkFilter.Limit != 5 {
		t.Fatalf("checkFilter=%+v", store.checkFilter)
	}
	if _, err := service.GetRuntimeCheckpoint(50); err != nil {
		t.Fatalf("GetRuntimeCheckpoint: %v", err)
	}
	if store.checkpointID != 50 {
		t.Fatalf("checkpointID=%d", store.checkpointID)
	}
	if _, err := service.ListMemoryEntities(db.FiltroEntidadesMemoria{ProyectoID: &projectID}); err != nil {
		t.Fatalf("ListMemoryEntities: %v", err)
	}
	if store.memoryFilter.ProyectoID == nil || *store.memoryFilter.ProyectoID != 7 {
		t.Fatalf("memoryFilter=%+v", store.memoryFilter)
	}
}
