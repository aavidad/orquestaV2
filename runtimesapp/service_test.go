package runtimesapp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orquesta/db"
)

type fakeStore struct {
	projectRef               string
	projectResponse          *db.Proyecto
	agentName                string
	agentResponse            *db.Agente
	sharedAccountAllowSet    bool
	sharedAccountAgent       string
	sharedAccountAllowed     bool
	sharedAccountOccupiedBy  string
	runtimesFilter           db.FiltroRuntimes
	runtimesResponse         []*db.RuntimeInstance
	treeFilter               db.FiltroRuntimes
	treeResponse             []*db.RuntimeTreeNode
	runtimeID                int64
	runtimeResponse          *db.RuntimeInstance
	primaryRuntimeAgent      string
	primaryRuntimeResp       *db.RuntimeInstance
	primaryProjRuntimeAgent  string
	primaryProjRuntimeID     *int64
	primaryProjRuntimeResp   *db.RuntimeInstance
	handleID                 int64
	handleByIDResp           *db.RuntimeHandle
	handlesFilter            *string
	handlesResponse          []*db.RuntimeHandle
	canonicalHandlesFilter   *string
	canonicalHandlesResponse []*db.RuntimeHandle
	syncHandleInput          *db.RuntimeHandle
	syncHandleSource         string
	syncHandleResp           *db.RuntimeHandle
	purgeFilter              db.FiltroPurgadoRuntimeHandles
	purgeResponse            *db.PurgaRuntimeHandlesResultado
	purgeOrdersFilter        db.FiltroPurgadoRuntimeOrders
	purgeOrdersResponse      *db.PurgaRuntimeOrdersResultado
	handleAgent              string
	handleResponse           *db.RuntimeHandle
	handleProjAgent          string
	handleProjID             *int64
	handleProjResp           *db.RuntimeHandle
	operationalHandleAgent   string
	operationalHandleResp    *db.RuntimeHandle
	operationalProjAgent     string
	operationalProjID        *int64
	operationalProjResp      *db.RuntimeHandle
	eventsFilter             db.FiltroRuntimeEvents
	eventsResp               []*db.RuntimeEvent
	transcriptFilter         db.FiltroRuntimeTranscript
	transcriptResp           []*db.RuntimeTranscriptEntry
	samplesID                int64
	samplesLimit             int
	samplesResponse          []*db.RuntimeTelemetrySample
	createOrder              *db.RuntimeOrder
	createOrderID            int64
	orderID                  int64
	ordersFilter             db.FiltroRuntimeOrders
	ordersResponse           []*db.RuntimeOrder
	handoffOrigen            string
	handoffDestino           string
	handoffTareaID           *int64
	handoffMotivo            string
	handoffResumen           string
	handoffExternalSessionID string
	handoffID                int64
	createMailbox            *db.RuntimeMailboxMessage
	createMailboxID          int64
	mailboxID                int64
	mailboxFilter            db.FiltroRuntimeMailbox
	mailboxResponse          []*db.RuntimeMailboxMessage
	deliveredID              int64
	consumedID               int64
	createCheck              *db.RuntimeCheckpoint
	createCheckID            int64
	checkFilter              db.FiltroRuntimeCheckpoints
	checkResponse            []*db.RuntimeCheckpoint
	checkpointID             int64
	checkpointResp           *db.RuntimeCheckpoint
	latestAgent              string
	latestProjectID          *int64
	latestResp               *db.RuntimeCheckpoint
	memoryFilter             db.FiltroEntidadesMemoria
	memoryResponse           []*db.EntidadMemoria
	memoryUpsert             *db.EntidadMemoria
	memoryUpsertID           int64
	memoryName               string
	memoryProjectID          *int64
	memoryEntityResp         *db.EntidadMemoria
	auditAgent               string
	auditAction              string
	auditEntity              string
	auditEntityID            int64
	auditDetail              string
}

func (f *fakeStore) GetProject(ref string) (*db.Proyecto, error) {
	f.projectRef = ref
	return f.projectResponse, nil
}
func (f *fakeStore) GetAgent(nombre string) (*db.Agente, error) {
	f.agentName = nombre
	return f.agentResponse, nil
}
func (f *fakeStore) SharedAccountActivationAllowed(agente string) (bool, string, error) {
	f.sharedAccountAgent = agente
	if !f.sharedAccountAllowSet {
		return true, "", nil
	}
	if f.sharedAccountAllowed {
		return true, "", nil
	}
	return false, f.sharedAccountOccupiedBy, nil
}
func (f *fakeStore) ListRuntimes(filter db.FiltroRuntimes) ([]*db.RuntimeInstance, error) {
	f.runtimesFilter = filter
	return f.runtimesResponse, nil
}
func (f *fakeStore) BuildRuntimeTree(filter db.FiltroRuntimes) ([]*db.RuntimeTreeNode, error) {
	f.treeFilter = filter
	return f.treeResponse, nil
}
func (f *fakeStore) GetRuntime(id int64) (*db.RuntimeInstance, error) {
	f.runtimeID = id
	return f.runtimeResponse, nil
}
func (f *fakeStore) GetPrimaryRuntime(agente string) (*db.RuntimeInstance, error) {
	f.primaryRuntimeAgent = agente
	return f.primaryRuntimeResp, nil
}
func (f *fakeStore) GetPrimaryRuntimeForProject(agente string, proyectoID *int64) (*db.RuntimeInstance, error) {
	f.primaryProjRuntimeAgent = agente
	f.primaryProjRuntimeID = proyectoID
	return f.primaryProjRuntimeResp, nil
}
func (f *fakeStore) GetRuntimeHandle(id int64) (*db.RuntimeHandle, error) {
	f.handleID = id
	return f.handleByIDResp, nil
}
func (f *fakeStore) ListRuntimeHandles(filter *string) ([]*db.RuntimeHandle, error) {
	f.handlesFilter = filter
	return f.handlesResponse, nil
}
func (f *fakeStore) ListCanonicalRuntimeHandles(filter *string) ([]*db.RuntimeHandle, error) {
	f.canonicalHandlesFilter = filter
	return f.canonicalHandlesResponse, nil
}
func (f *fakeStore) SyncSupervisedRuntimeHandle(handle *db.RuntimeHandle, source string) (*db.RuntimeHandle, error) {
	f.syncHandleInput = handle
	f.syncHandleSource = source
	if f.syncHandleResp != nil {
		return f.syncHandleResp, nil
	}
	return handle, nil
}
func (f *fakeStore) PurgeInactiveRuntimeHandles(filter db.FiltroPurgadoRuntimeHandles) (*db.PurgaRuntimeHandlesResultado, error) {
	f.purgeFilter = filter
	return f.purgeResponse, nil
}
func (f *fakeStore) PurgeTerminalRuntimeOrders(filter db.FiltroPurgadoRuntimeOrders) (*db.PurgaRuntimeOrdersResultado, error) {
	f.purgeOrdersFilter = filter
	return f.purgeOrdersResponse, nil
}
func (f *fakeStore) GetActiveRuntimeHandle(agente string) (*db.RuntimeHandle, error) {
	f.handleAgent = agente
	return f.handleResponse, nil
}
func (f *fakeStore) GetActiveRuntimeHandleForProject(agente string, proyectoID *int64) (*db.RuntimeHandle, error) {
	f.handleProjAgent = agente
	f.handleProjID = proyectoID
	return f.handleProjResp, nil
}
func (f *fakeStore) GetOperationalRuntimeHandle(agente string) (*db.RuntimeHandle, error) {
	f.operationalHandleAgent = agente
	return f.operationalHandleResp, nil
}
func (f *fakeStore) GetOperationalRuntimeHandleForProject(agente string, proyectoID *int64) (*db.RuntimeHandle, error) {
	f.operationalProjAgent = agente
	f.operationalProjID = proyectoID
	return f.operationalProjResp, nil
}
func (f *fakeStore) ListRuntimeEvents(filter db.FiltroRuntimeEvents) ([]*db.RuntimeEvent, error) {
	f.eventsFilter = filter
	return f.eventsResp, nil
}
func (f *fakeStore) ListRuntimeTranscript(filter db.FiltroRuntimeTranscript) ([]*db.RuntimeTranscriptEntry, error) {
	f.transcriptFilter = filter
	return f.transcriptResp, nil
}
func (f *fakeStore) ListRuntimeSamples(runtimeID int64, limit int) ([]*db.RuntimeTelemetrySample, error) {
	f.samplesID = runtimeID
	f.samplesLimit = limit
	return f.samplesResponse, nil
}
func (f *fakeStore) CreateRuntimeOrder(order *db.RuntimeOrder) (int64, error) {
	f.createOrder = order
	return f.createOrderID, nil
}
func (f *fakeStore) GetRuntimeOrder(id int64) (*db.RuntimeOrder, error) {
	f.orderID = id
	if len(f.ordersResponse) > 0 {
		return f.ordersResponse[0], nil
	}
	return nil, nil
}
func (f *fakeStore) ListRuntimeOrders(filter db.FiltroRuntimeOrders) ([]*db.RuntimeOrder, error) {
	f.ordersFilter = filter
	return f.ordersResponse, nil
}
func (f *fakeStore) CreateLiveAgentHandoff(origen, destino string, tareaID *int64, motivo, resumenContinuidad, externalSessionID string) (int64, error) {
	f.handoffOrigen = origen
	f.handoffDestino = destino
	f.handoffTareaID = tareaID
	f.handoffMotivo = motivo
	f.handoffResumen = resumenContinuidad
	f.handoffExternalSessionID = externalSessionID
	return f.handoffID, nil
}
func (f *fakeStore) CreateRuntimeMailbox(msg *db.RuntimeMailboxMessage) (int64, error) {
	f.createMailbox = msg
	return f.createMailboxID, nil
}
func (f *fakeStore) GetRuntimeMailbox(id int64) (*db.RuntimeMailboxMessage, error) {
	f.mailboxID = id
	if len(f.mailboxResponse) > 0 {
		return f.mailboxResponse[0], nil
	}
	return nil, nil
}
func (f *fakeStore) ListRuntimeMailbox(filter db.FiltroRuntimeMailbox) ([]*db.RuntimeMailboxMessage, error) {
	f.mailboxFilter = filter
	return f.mailboxResponse, nil
}
func (f *fakeStore) MarkRuntimeMailboxDelivered(id int64) error {
	f.deliveredID = id
	return nil
}
func (f *fakeStore) MarkRuntimeMailboxConsumed(id int64) error {
	f.consumedID = id
	return nil
}
func (f *fakeStore) CreateRuntimeCheckpoint(checkpoint *db.RuntimeCheckpoint) (int64, error) {
	f.createCheck = checkpoint
	return f.createCheckID, nil
}
func (f *fakeStore) ListRuntimeCheckpoints(filter db.FiltroRuntimeCheckpoints) ([]*db.RuntimeCheckpoint, error) {
	f.checkFilter = filter
	return f.checkResponse, nil
}
func (f *fakeStore) GetRuntimeCheckpoint(id int64) (*db.RuntimeCheckpoint, error) {
	f.checkpointID = id
	return f.checkpointResp, nil
}
func (f *fakeStore) LatestRuntimeCheckpoint(agente string, proyectoID *int64) (*db.RuntimeCheckpoint, error) {
	f.latestAgent = agente
	f.latestProjectID = proyectoID
	return f.latestResp, nil
}
func (f *fakeStore) ListMemoryEntities(filter db.FiltroEntidadesMemoria) ([]*db.EntidadMemoria, error) {
	f.memoryFilter = filter
	return f.memoryResponse, nil
}
func (f *fakeStore) UpsertMemoryEntity(entidad *db.EntidadMemoria) (int64, error) {
	f.memoryUpsert = entidad
	return f.memoryUpsertID, nil
}
func (f *fakeStore) GetMemoryEntity(nombre string, proyectoID *int64) (*db.EntidadMemoria, error) {
	f.memoryName = nombre
	f.memoryProjectID = proyectoID
	return f.memoryEntityResp, nil
}
func (f *fakeStore) Audit(agente, accion, entidad string, entidadID int64, detalle string) {
	f.auditAgent = agente
	f.auditAction = accion
	f.auditEntity = entidad
	f.auditEntityID = entidadID
	f.auditDetail = detalle
}

func TestServiceDelegatesRuntimeQueries(t *testing.T) {
	agent := "Codex2"
	projectID := int64(7)
	store := &fakeStore{
		projectResponse:     &db.Proyecto{Slug: "orquestador"},
		agentResponse:       &db.Agente{Nombre: "Codex2", Rol: "programador"},
		runtimesResponse:    []*db.RuntimeInstance{{ID: 5}},
		treeResponse:        []*db.RuntimeTreeNode{{Runtime: &db.RuntimeInstance{ID: 10}}},
		runtimeResponse:     &db.RuntimeInstance{ID: 10},
		handleByIDResp:      &db.RuntimeHandle{ID: 16},
		handlesResponse:     []*db.RuntimeHandle{{ID: 15}},
		syncHandleResp:      &db.RuntimeHandle{ID: 15, Estado: "activo"},
		purgeResponse:       &db.PurgaRuntimeHandlesResultado{Deleted: 2, Estados: []string{"cerrado", "fallido"}},
		purgeOrdersResponse: &db.PurgaRuntimeOrdersResultado{Deleted: 3, Estados: []string{"completada", "fallida"}, Tipos: []string{"send_instruction"}},
		handleResponse:      &db.RuntimeHandle{ID: 16},
		handleProjResp:      &db.RuntimeHandle{ID: 17},
		eventsResp:          []*db.RuntimeEvent{{ID: 17, Kind: "auto_guidance_sent"}},
		transcriptResp:      []*db.RuntimeTranscriptEntry{{ID: 18}},
		samplesResponse:     []*db.RuntimeTelemetrySample{{ID: 20}},
		createOrderID:       25,
		ordersResponse:      []*db.RuntimeOrder{{ID: 30}},
		createMailboxID:     35,
		mailboxResponse:     []*db.RuntimeMailboxMessage{{ID: 40}},
		createCheckID:       45,
		checkResponse:       []*db.RuntimeCheckpoint{{ID: 50}},
		checkpointResp:      &db.RuntimeCheckpoint{ID: 50},
		latestResp:          &db.RuntimeCheckpoint{ID: 55},
		memoryResponse:      []*db.EntidadMemoria{{ID: 60}},
		memoryUpsertID:      61,
		memoryEntityResp:    &db.EntidadMemoria{ID: 62, Nombre: "decision"},
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
	if _, err := service.ListRuntimes(db.FiltroRuntimes{Agente: &agent}); err != nil {
		t.Fatalf("ListRuntimes: %v", err)
	}
	if store.runtimesFilter.Agente == nil || *store.runtimesFilter.Agente != "Codex2" {
		t.Fatalf("runtimesFilter=%+v", store.runtimesFilter)
	}
	if _, err := service.GetRuntime(10); err != nil {
		t.Fatalf("GetRuntime: %v", err)
	}
	if store.runtimeID != 10 {
		t.Fatalf("runtimeID=%d", store.runtimeID)
	}
	if _, err := service.GetRuntimeHandle(16); err != nil {
		t.Fatalf("GetRuntimeHandle: %v", err)
	}
	if store.handleID != 16 {
		t.Fatalf("handleID=%d", store.handleID)
	}
	if _, err := service.ListRuntimeHandles(&agent); err != nil {
		t.Fatalf("ListRuntimeHandles: %v", err)
	}
	if store.handlesFilter == nil || *store.handlesFilter != "Codex2" {
		t.Fatalf("handlesFilter=%v", store.handlesFilter)
	}
	if store.syncHandleInput == nil || store.syncHandleInput.ID != 15 {
		t.Fatalf("syncHandleInput=%+v", store.syncHandleInput)
	}
	if store.syncHandleSource != "runtimesapp_list_runtime_handles" {
		t.Fatalf("syncHandleSource=%q", store.syncHandleSource)
	}
	if _, err := service.GetActiveRuntimeHandle(agent); err != nil {
		t.Fatalf("GetActiveRuntimeHandle: %v", err)
	}
	if _, err := service.PurgeInactiveRuntimeHandles(RuntimeHandlePurgeRequest{Agente: agent, Estados: []string{"cerrado"}}); err != nil {
		t.Fatalf("PurgeInactiveRuntimeHandles: %v", err)
	}
	if store.purgeFilter.Agente == nil || *store.purgeFilter.Agente != "Codex2" {
		t.Fatalf("purgeFilter=%+v", store.purgeFilter)
	}
	if _, err := service.PurgeTerminalRuntimeOrders(RuntimeOrderPurgeRequest{Agente: agent, Estados: []string{"completada", "fallida"}, Tipos: []string{"send_instruction"}}); err != nil {
		t.Fatalf("PurgeTerminalRuntimeOrders: %v", err)
	}
	if store.purgeOrdersFilter.Agente == nil || *store.purgeOrdersFilter.Agente != "Codex2" || len(store.purgeOrdersFilter.Tipos) != 1 || store.purgeOrdersFilter.Tipos[0] != "send_instruction" {
		t.Fatalf("purgeOrdersFilter=%+v", store.purgeOrdersFilter)
	}
	if store.handleAgent != "Codex2" {
		t.Fatalf("handleAgent=%q", store.handleAgent)
	}
	if _, err := service.GetActiveRuntimeHandleForProject(agent, &projectID); err != nil {
		t.Fatalf("GetActiveRuntimeHandleForProject: %v", err)
	}
	if store.handleProjAgent != "Codex2" || store.handleProjID == nil || *store.handleProjID != 7 {
		t.Fatalf("handleProj agent=%q project=%v", store.handleProjAgent, store.handleProjID)
	}
	if _, err := service.ListRuntimeEvents(db.FiltroRuntimeEvents{Agente: &agent, ProyectoID: &projectID, Limit: 5}); err != nil {
		t.Fatalf("ListRuntimeEvents: %v", err)
	}
	if store.eventsFilter.Agente == nil || *store.eventsFilter.Agente != "Codex2" || store.eventsFilter.ProyectoID == nil || *store.eventsFilter.ProyectoID != 7 || store.eventsFilter.Limit != 5 {
		t.Fatalf("eventsFilter=%+v", store.eventsFilter)
	}
	if _, err := service.ListRuntimeTranscript(db.FiltroRuntimeTranscript{Agente: &agent, Limit: 10}); err != nil {
		t.Fatalf("ListRuntimeTranscript: %v", err)
	}
	if store.transcriptFilter.Agente == nil || *store.transcriptFilter.Agente != "Codex2" {
		t.Fatalf("transcriptFilter=%+v", store.transcriptFilter)
	}
	if _, err := service.ListRuntimeSamples(10, 25); err != nil {
		t.Fatalf("ListRuntimeSamples: %v", err)
	}
	if store.samplesID != 10 || store.samplesLimit != 25 {
		t.Fatalf("samples runtime=%d limit=%d", store.samplesID, store.samplesLimit)
	}
	if _, err := service.CreateRuntimeOrder(&db.RuntimeOrder{Agente: agent}); err != nil {
		t.Fatalf("CreateRuntimeOrder: %v", err)
	}
	if store.createOrder == nil || store.createOrder.Agente != "Codex2" {
		t.Fatalf("createOrder=%+v", store.createOrder)
	}
	if _, err := service.ListRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agent, ProyectoID: &projectID}); err != nil {
		t.Fatalf("ListRuntimeOrders: %v", err)
	}
	if store.ordersFilter.Agente == nil || *store.ordersFilter.Agente != "Codex2" {
		t.Fatalf("ordersFilter=%+v", store.ordersFilter)
	}
	if _, err := service.CreateRuntimeMailbox(&db.RuntimeMailboxMessage{ToAgente: agent}); err != nil {
		t.Fatalf("CreateRuntimeMailbox: %v", err)
	}
	if store.createMailbox == nil || store.createMailbox.ToAgente != "Codex2" {
		t.Fatalf("createMailbox=%+v", store.createMailbox)
	}
	if _, err := service.ListRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agent, ProyectoID: &projectID}); err != nil {
		t.Fatalf("ListRuntimeMailbox: %v", err)
	}
	if store.mailboxFilter.ToAgente == nil || *store.mailboxFilter.ToAgente != "Codex2" {
		t.Fatalf("mailboxFilter=%+v", store.mailboxFilter)
	}
	if err := service.MarkRuntimeMailboxDelivered(40); err != nil {
		t.Fatalf("MarkRuntimeMailboxDelivered: %v", err)
	}
	if store.deliveredID != 40 {
		t.Fatalf("deliveredID=%d", store.deliveredID)
	}
	if err := service.MarkRuntimeMailboxConsumed(41); err != nil {
		t.Fatalf("MarkRuntimeMailboxConsumed: %v", err)
	}
	if store.consumedID != 41 {
		t.Fatalf("consumedID=%d", store.consumedID)
	}
	if _, err := service.CreateRuntimeCheckpoint(&db.RuntimeCheckpoint{Agente: agent}); err != nil {
		t.Fatalf("CreateRuntimeCheckpoint: %v", err)
	}
	if store.createCheck == nil || store.createCheck.Agente != "Codex2" {
		t.Fatalf("createCheck=%+v", store.createCheck)
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
	if _, err := service.LatestRuntimeCheckpoint(agent, &projectID); err != nil {
		t.Fatalf("LatestRuntimeCheckpoint: %v", err)
	}
	if store.latestAgent != "Codex2" || store.latestProjectID == nil || *store.latestProjectID != 7 {
		t.Fatalf("latest agent=%q project=%v", store.latestAgent, store.latestProjectID)
	}
	if _, err := service.ListMemoryEntities(db.FiltroEntidadesMemoria{ProyectoID: &projectID}); err != nil {
		t.Fatalf("ListMemoryEntities: %v", err)
	}
	if store.memoryFilter.ProyectoID == nil || *store.memoryFilter.ProyectoID != 7 {
		t.Fatalf("memoryFilter=%+v", store.memoryFilter)
	}
	if _, err := service.UpsertMemoryEntity(&db.EntidadMemoria{Nombre: "decision", ProyectoID: &projectID}); err != nil {
		t.Fatalf("UpsertMemoryEntity: %v", err)
	}
	if store.memoryUpsert == nil || store.memoryUpsert.Nombre != "decision" {
		t.Fatalf("memoryUpsert=%+v", store.memoryUpsert)
	}
	if _, err := service.GetMemoryEntity("decision", &projectID); err != nil {
		t.Fatalf("GetMemoryEntity: %v", err)
	}
	if store.memoryName != "decision" || store.memoryProjectID == nil || *store.memoryProjectID != 7 {
		t.Fatalf("memory entity name=%q project=%v", store.memoryName, store.memoryProjectID)
	}
}

func TestCreateRuntimeOrderInvocaHookTrasEncolar(t *testing.T) {
	store := &fakeStore{createOrderID: 73}
	service := NewService(store)
	var (
		gotOrderID int64
		gotAgent   string
	)
	service.SetAfterEnqueueRuntimeOrderHook(func(order *db.RuntimeOrder, orderID int64) {
		gotOrderID = orderID
		if order != nil {
			gotAgent = order.Agente
		}
	})

	orderID, err := service.CreateRuntimeOrder(&db.RuntimeOrder{Agente: "Codex2", Tipo: "start"})
	if err != nil {
		t.Fatalf("CreateRuntimeOrder: %v", err)
	}
	if orderID != 73 {
		t.Fatalf("orderID inesperado: %d", orderID)
	}
	if gotOrderID != 73 || gotAgent != "Codex2" {
		t.Fatalf("hook inesperado: orderID=%d agent=%q", gotOrderID, gotAgent)
	}
}

func TestSendRuntimeMailboxInvocaHookTrasCrear(t *testing.T) {
	store := &fakeStore{createMailboxID: 35}
	service := NewService(store)
	var (
		gotMailboxID int64
		gotAgent     string
	)
	service.SetAfterCreateRuntimeMailboxHook(func(msg *db.RuntimeMailboxMessage, mailboxID int64) {
		gotMailboxID = mailboxID
		if msg != nil {
			gotAgent = msg.ToAgente
		}
	})

	mailboxID, err := service.SendRuntimeMailbox(&db.RuntimeMailboxMessage{FromAgente: "server", ToAgente: "OllamaLocal", Kind: "instruction"})
	if err != nil {
		t.Fatalf("SendRuntimeMailbox: %v", err)
	}
	if mailboxID != 35 {
		t.Fatalf("mailboxID inesperado: %d", mailboxID)
	}
	if gotMailboxID != 35 || gotAgent != "OllamaLocal" {
		t.Fatalf("hook inesperado: mailboxID=%d agent=%q", gotMailboxID, gotAgent)
	}
}

func TestDispatchMicroprogramacionInstructionEncolaSendInstructionCanonica(t *testing.T) {
	projectID := int64(9)
	store := &fakeStore{createOrderID: 88}
	service := NewService(store)

	resultado, err := service.DispatchMicroprogramacionInstruction(MicroprogramacionDispatchRequest{
		AgenteDestino:     "Codex1",
		ProyectoID:        &projectID,
		Mensaje:           "MICROTAREA CERRADA\nARCHIVO: tmp/x.go",
		EspecificacionID:  31,
		ArchivoObjetivo:   "tmp/x.go",
		SimboloObjetivo:   "Normalizar",
		WriteSet:          []string{"tmp/x.go"},
		TestsObligatorios: []string{"go test ./tmp -count=1"},
		FormatoSalida:     "patch+evidencia",
	})
	if err != nil {
		t.Fatalf("DispatchMicroprogramacionInstruction: %v", err)
	}
	if resultado == nil || resultado.RuntimeOrderID != 88 {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
	if store.createOrder == nil || store.createOrder.Tipo != "send_instruction" || store.createOrder.Agente != "Codex1" {
		t.Fatalf("runtime order inesperada: %+v", store.createOrder)
	}
	if !strings.Contains(store.createOrder.PayloadJSON, `"source":"microprogramacion"`) || !strings.Contains(store.createOrder.PayloadJSON, `"especificacion_id":31`) {
		t.Fatalf("payload sin trazabilidad microprogramada: %s", store.createOrder.PayloadJSON)
	}
}

func TestDispatchMicroprogramacionInstructionOllamaIncluyeContextoInline(t *testing.T) {
	projectID := int64(9)
	proyectoDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(proyectoDir, "identidad"), 0o755); err != nil {
		t.Fatalf("mkdir identidad: %v", err)
	}
	if err := os.WriteFile(filepath.Join(proyectoDir, "identidad", "normalizar.go"), []byte("package identidad\n\nfunc NormalizarIdentificador(s string) string { return s }\n"), 0o644); err != nil {
		t.Fatalf("write normalizar.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(proyectoDir, "identidad", "normalizar_test.go"), []byte("package identidad\n\nfunc TestNormalizarIdentificador() {}\n"), 0o644); err != nil {
		t.Fatalf("write normalizar_test.go: %v", err)
	}
	store := &fakeStore{
		createOrderID:   91,
		projectResponse: &db.Proyecto{ID: projectID, Slug: "proyecto", RutaAbs: proyectoDir},
	}
	service := NewService(store)

	resultado, err := service.DispatchMicroprogramacionInstruction(MicroprogramacionDispatchRequest{
		AgenteDestino:     "Gemma1",
		ProyectoID:        &projectID,
		Mensaje:           "MICROTAREA CERRADA\nARCHIVO: identidad/normalizar.go",
		EspecificacionID:  44,
		ArchivoObjetivo:   "identidad/normalizar.go",
		SimboloObjetivo:   "NormalizarIdentificador",
		WriteSet:          []string{"identidad/normalizar.go"},
		TestsObligatorios: []string{"go test ./identidad -run TestNormalizarIdentificador"},
		FormatoSalida:     "patch+evidencia",
	})
	if err != nil {
		t.Fatalf("DispatchMicroprogramacionInstruction: %v", err)
	}
	if resultado == nil || resultado.RuntimeOrderID != 91 {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
	if store.projectRef != "9" {
		t.Fatalf("se esperaba lookup por proyecto ID, got=%q", store.projectRef)
	}
	if store.createOrder == nil {
		t.Fatalf("runtime order no creada")
	}
	if !strings.Contains(store.createOrder.PayloadJSON, "PROTOCOLO_MICROPROGRAMACION_INLINE") {
		t.Fatalf("payload sin protocolo inline: %s", store.createOrder.PayloadJSON)
	}
	if !strings.Contains(store.createOrder.PayloadJSON, "identidad/normalizar.go") || !strings.Contains(store.createOrder.PayloadJSON, "func NormalizarIdentificador") {
		t.Fatalf("payload sin archivo objetivo inline: %s", store.createOrder.PayloadJSON)
	}
	if !strings.Contains(store.createOrder.PayloadJSON, "identidad/normalizar_test.go") || !strings.Contains(store.createOrder.PayloadJSON, "TestNormalizarIdentificador") {
		t.Fatalf("payload sin test inline: %s", store.createOrder.PayloadJSON)
	}
}

func TestDispatchMicroprogramacionInstructionOllamaToleraWriteSetConArchivosNuevos(t *testing.T) {
	projectID := int64(12)
	proyectoDir := t.TempDir()
	store := &fakeStore{
		createOrderID:   93,
		projectResponse: &db.Proyecto{ID: projectID, Slug: "proyecto", RutaAbs: proyectoDir},
	}
	service := NewService(store)

	resultado, err := service.DispatchMicroprogramacionInstruction(MicroprogramacionDispatchRequest{
		AgenteDestino:     "Gemma1",
		ProyectoID:        &projectID,
		Mensaje:           "MICROTAREA CERRADA\nARCHIVO: identidad/nuevo.go",
		EspecificacionID:  46,
		ArchivoObjetivo:   "identidad/nuevo.go",
		SimboloObjetivo:   "NuevoSlice",
		WriteSet:          []string{"identidad/nuevo.go", "identidad/nuevo_test.go"},
		TestsObligatorios: []string{"go test ./identidad -run TestNuevoSlice"},
		FormatoSalida:     "patch+evidencia",
	})
	if err != nil {
		t.Fatalf("DispatchMicroprogramacionInstruction: %v", err)
	}
	if resultado == nil || resultado.RuntimeOrderID != 93 {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
	if store.createOrder == nil {
		t.Fatalf("runtime order no creada")
	}
	if !strings.Contains(store.createOrder.PayloadJSON, "ARCHIVO_NO_EXISTE_TODAVIA") {
		t.Fatalf("payload sin placeholder de archivo nuevo: %s", store.createOrder.PayloadJSON)
	}
	if !strings.Contains(store.createOrder.PayloadJSON, "identidad/nuevo.go") || !strings.Contains(store.createOrder.PayloadJSON, "identidad/nuevo_test.go") {
		t.Fatalf("payload sin write_set esperado: %s", store.createOrder.PayloadJSON)
	}
}

func TestDispatchMicroprogramacionInstructionCodexNoFuerzaContextoInline(t *testing.T) {
	projectID := int64(11)
	store := &fakeStore{createOrderID: 92}
	service := NewService(store)

	_, err := service.DispatchMicroprogramacionInstruction(MicroprogramacionDispatchRequest{
		AgenteDestino:     "Codex1",
		ProyectoID:        &projectID,
		Mensaje:           "MICROTAREA CERRADA\nARCHIVO: runtimeagente/driver.go",
		EspecificacionID:  45,
		ArchivoObjetivo:   "runtimeagente/driver.go",
		SimboloObjetivo:   "BuildSpec",
		WriteSet:          []string{"runtimeagente/driver.go"},
		TestsObligatorios: []string{"go test ./runtimeagente -run TestBuildSpec"},
		FormatoSalida:     "patch+evidencia",
	})
	if err != nil {
		t.Fatalf("DispatchMicroprogramacionInstruction: %v", err)
	}
	if store.projectRef != "" {
		t.Fatalf("Codex no deberia resolver proyecto para contexto inline: %q", store.projectRef)
	}
	if store.createOrder == nil {
		t.Fatalf("runtime order no creada")
	}
	if strings.Contains(store.createOrder.PayloadJSON, "PROTOCOLO_MICROPROGRAMACION_INLINE") {
		t.Fatalf("Codex no deberia recibir contexto inline forzado: %s", store.createOrder.PayloadJSON)
	}
}

func TestCreateLiveAgentHandoffDelegates(t *testing.T) {
	tareaID := int64(41)
	store := &fakeStore{handoffID: 88}
	service := NewService(store)

	id, err := service.CreateLiveAgentHandoff(" Codex1 ", " Codex2 ", &tareaID, " relevo ", " continuar ", " ext-1 ")
	if err != nil {
		t.Fatalf("CreateLiveAgentHandoff: %v", err)
	}
	if id != 88 {
		t.Fatalf("id inesperado: %d", id)
	}
	if store.handoffOrigen != "Codex1" || store.handoffDestino != "Codex2" {
		t.Fatalf("handoff origen/destino inesperados: %q -> %q", store.handoffOrigen, store.handoffDestino)
	}
	if store.handoffTareaID == nil || *store.handoffTareaID != tareaID {
		t.Fatalf("handoffTareaID inesperado: %v", store.handoffTareaID)
	}
	if store.handoffMotivo != "relevo" || store.handoffResumen != "continuar" || store.handoffExternalSessionID != "ext-1" {
		t.Fatalf("handoff payload inesperado: motivo=%q resumen=%q ext=%q", store.handoffMotivo, store.handoffResumen, store.handoffExternalSessionID)
	}
}

func TestResolveControlHandleOmiteLegacyTMUXPreferred(t *testing.T) {
	store := &fakeStore{
		operationalHandleResp: &db.RuntimeHandle{
			ID:           10,
			Agente:       "CodexLegacy",
			Transporte:   "cli",
			HandleKind:   "process",
			Estado:       "activo",
			MetadataJSON: `{"driver":"process_pty_cli","rendered_command":"codex-perfil CodexLegacy --model gpt-5.4"}`,
		},
		handleByIDResp: &db.RuntimeHandle{
			ID:           10,
			Agente:       "CodexLegacy",
			Transporte:   "cli",
			HandleKind:   "process",
			Estado:       "activo",
			MetadataJSON: `{"driver":"process_pty_cli","rendered_command":"codex-perfil CodexLegacy --model gpt-5.4"}`,
		},
	}
	service := NewService(store)

	handle, err := service.ResolveControlHandle("CodexLegacy", nil)
	if err != nil {
		t.Fatalf("ResolveControlHandle: %v", err)
	}
	if handle != nil {
		t.Fatalf("debería omitir el handle legacy tmux-preferred, obtuvo %+v", handle)
	}
}

func TestResolveControlHandleAceptaWorkerTMUXFresh(t *testing.T) {
	metaJSON := mustStructuredWorkerMetadataJSONRuntimesTest(t, time.Now().UTC(), "running", true)
	store := &fakeStore{
		operationalHandleResp: &db.RuntimeHandle{
			ID:           11,
			Agente:       "CodexFresh",
			Transporte:   "tmux",
			HandleKind:   "session",
			Estado:       "activo",
			MetadataJSON: metaJSON,
		},
		handleByIDResp: &db.RuntimeHandle{
			ID:           11,
			Agente:       "CodexFresh",
			Transporte:   "tmux",
			HandleKind:   "session",
			Estado:       "activo",
			MetadataJSON: metaJSON,
		},
	}
	service := NewService(store)

	handle, err := service.ResolveControlHandle("CodexFresh", nil)
	if err != nil {
		t.Fatalf("ResolveControlHandle: %v", err)
	}
	if handle == nil || handle.ID != 11 {
		t.Fatalf("debería aceptar el worker tmux fresco: %+v", handle)
	}
}

func TestResolveTranscriptDeliveryHandleUsaHandleDelTranscript(t *testing.T) {
	store := &fakeStore{
		handleByIDResp: &db.RuntimeHandle{ID: 33, Agente: "Codex8", Estado: "activo"},
	}
	service := NewService(store)

	handle, err := service.ResolveTranscriptDeliveryHandle(&db.RuntimeTranscriptEntry{
		Agente:    "Codex8",
		RuntimeID: 90,
		HandleID:  ptrInt64RuntimeServiceTest(33),
	})
	if err != nil {
		t.Fatalf("ResolveTranscriptDeliveryHandle: %v", err)
	}
	if handle == nil || handle.ID != 33 {
		t.Fatalf("debería devolver el handle del transcript: %+v", handle)
	}
}

func TestPreferredRuntimeHandlePrefiereOperativoReciente(t *testing.T) {
	projectID := int64(7)
	store := &fakeStore{
		handleProjResp:        &db.RuntimeHandle{ID: 17},
		operationalProjResp:   &db.RuntimeHandle{ID: 99},
		handleResponse:        &db.RuntimeHandle{ID: 18},
		operationalHandleResp: &db.RuntimeHandle{ID: 98},
	}
	service := NewService(store)

	handle, err := service.preferredRuntimeHandle(&db.RuntimeInstance{
		Agente:     "Codex2",
		ProyectoID: &projectID,
	})
	if err != nil {
		t.Fatalf("preferredRuntimeHandle: %v", err)
	}
	if handle == nil || handle.ID != 99 {
		t.Fatalf("deberia preferir el handle operativo reciente del proyecto: %+v", handle)
	}
	if store.operationalProjAgent != "Codex2" {
		t.Fatalf("deberia consultar primero el handle operativo del proyecto")
	}
}

func TestDescribeRuntimeExponeWorkerEstructuradoDelHandleCanonico(t *testing.T) {
	tmp := t.TempDir()
	now := time.Now().UTC()
	manifestPath := filepath.Join(tmp, "manifest.json")
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")

	writeJSON := func(path string, payload any) {
		t.Helper()
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	writeJSON(manifestPath, map[string]any{
		"version":      1,
		"agent":        "Codex7",
		"driver":       "tmux_cli_session",
		"transport":    "tmux",
		"tmux_session": "orq-codex7-101500",
		"tmux_pane_id": "%9",
		"child_pid":    4567,
	})
	writeJSON(statusPath, map[string]any{
		"state":          "running",
		"updated_at":     now.Format(time.RFC3339Nano),
		"alive":          true,
		"last_output_at": now.Format(time.RFC3339Nano),
	})
	writeJSON(heartbeatPath, map[string]any{
		"alive":        true,
		"heartbeat_at": now.Format(time.RFC3339Nano),
	})
	metaJSON, _ := json.Marshal(map[string]any{
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})

	sesionID := int64(33)
	runtimeID := int64(44)
	store := &fakeStore{
		runtimeResponse: &db.RuntimeInstance{ID: runtimeID, Agente: "Codex7", SesionID: &sesionID},
		handlesResponse: []*db.RuntimeHandle{
			{ID: 90, Agente: "Codex7", Estado: "cerrado"},
			{ID: 91, Agente: "Codex7", SesionID: &sesionID, RuntimeID: &runtimeID, Estado: "activo", MetadataJSON: string(metaJSON)},
		},
	}
	service := NewService(store)

	desc, err := service.DescribeRuntime(runtimeID)
	if err != nil {
		t.Fatalf("DescribeRuntime: %v", err)
	}
	if desc == nil || desc.Runtime == nil || desc.Runtime.ID != runtimeID {
		t.Fatalf("runtime inesperado: %+v", desc)
	}
	if desc.Handle == nil || desc.Handle.ID != 91 {
		t.Fatalf("handle inesperado: %+v", desc.Handle)
	}
	if desc.Worker == nil {
		t.Fatal("worker estructurado ausente")
	}
	if got := desc.Worker.Driver; got != "tmux_cli_session" {
		t.Fatalf("driver inesperado: %q", got)
	}
	if got := desc.Worker.RuntimeRef; got != "orq-codex7-101500/%9" {
		t.Fatalf("runtime ref inesperado: %q", got)
	}
	if !desc.Worker.Alive {
		t.Fatalf("worker deberia salir vivo: %+v", desc.Worker)
	}
	if store.handlesFilter == nil || *store.handlesFilter != "Codex7" {
		t.Fatalf("handlesFilter=%v", store.handlesFilter)
	}
	if store.syncHandleInput == nil || store.syncHandleInput.ID != 91 {
		t.Fatalf("syncHandleInput=%+v", store.syncHandleInput)
	}
	if store.syncHandleSource != "runtimesapp_describe_runtime" {
		t.Fatalf("syncHandleSource=%q", store.syncHandleSource)
	}
}

func TestDescribeRuntimePrefiereHandleTMUXSobreLegacyConMismoRuntimeYSesion(t *testing.T) {
	tmp := t.TempDir()
	now := time.Now().UTC()
	manifestPath := filepath.Join(tmp, "manifest.json")
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")

	writeJSON := func(path string, payload any) {
		t.Helper()
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	writeJSON(manifestPath, map[string]any{
		"version":      1,
		"agent":        "Codex3",
		"driver":       "tmux_cli_session",
		"transport":    "tmux",
		"tmux_session": "orq-codex3-153000",
		"tmux_pane_id": "%40",
		"child_pid":    777,
	})
	writeJSON(statusPath, map[string]any{
		"state":          "running",
		"updated_at":     now.Format(time.RFC3339Nano),
		"alive":          true,
		"last_output_at": now.Format(time.RFC3339Nano),
	})
	writeJSON(heartbeatPath, map[string]any{
		"alive":        true,
		"heartbeat_at": now.Format(time.RFC3339Nano),
	})

	tmuxMetaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	legacyMetaJSON, _ := json.Marshal(map[string]any{
		"driver":           "process_pty_cli",
		"rendered_command": "codex-perfil Codex3 --model gpt-5.4",
	})

	sesionID := int64(44)
	runtimeID := int64(55)
	store := &fakeStore{
		runtimeResponse: &db.RuntimeInstance{ID: runtimeID, Agente: "Codex3", SesionID: &sesionID},
		handlesResponse: []*db.RuntimeHandle{
			{ID: 100, Agente: "Codex3", SesionID: &sesionID, RuntimeID: &runtimeID, Estado: "activo", MetadataJSON: string(legacyMetaJSON)},
			{ID: 101, Agente: "Codex3", SesionID: &sesionID, RuntimeID: &runtimeID, Estado: "activo", MetadataJSON: string(tmuxMetaJSON)},
		},
	}

	desc, err := NewService(store).DescribeRuntime(runtimeID)
	if err != nil {
		t.Fatalf("DescribeRuntime: %v", err)
	}
	if desc == nil || desc.Handle == nil {
		t.Fatalf("runtime description inesperada: %+v", desc)
	}
	if desc.Handle.ID != 101 {
		t.Fatalf("deberia priorizar el handle tmux sobre legacy, got=%d", desc.Handle.ID)
	}
	if desc.Worker == nil || desc.Worker.Driver != "tmux_cli_session" {
		t.Fatalf("worker inesperado: %+v", desc.Worker)
	}
}

func TestDescribeRuntimeUsaHandleActivoCanonicoAntesDeBarrerHandles(t *testing.T) {
	now := time.Now().UTC()
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "manifest.json")
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")
	writeJSON := func(path string, payload map[string]any) {
		t.Helper()
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	writeJSON(manifestPath, map[string]any{
		"version":      1,
		"agent":        "Codex2",
		"driver":       "tmux_cli_session",
		"transport":    "tmux",
		"tmux_session": "orq-codex2-1",
		"tmux_pane_id": "%11",
	})
	writeJSON(statusPath, map[string]any{
		"state":          "running",
		"updated_at":     now.Format(time.RFC3339Nano),
		"alive":          true,
		"last_output_at": now.Format(time.RFC3339Nano),
	})
	writeJSON(heartbeatPath, map[string]any{
		"alive":        true,
		"heartbeat_at": now.Format(time.RFC3339Nano),
	})
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})

	sesionID := int64(24)
	runtimeID := int64(25)
	projectID := int64(26)
	active := &db.RuntimeHandle{
		ID:           77,
		Agente:       "Codex2",
		ProyectoID:   &projectID,
		SesionID:     &sesionID,
		RuntimeID:    &runtimeID,
		Estado:       "activo",
		MetadataJSON: string(metaJSON),
	}
	store := &fakeStore{
		runtimeResponse: &db.RuntimeInstance{ID: runtimeID, Agente: "Codex2", ProyectoID: &projectID, SesionID: &sesionID},
		handleProjResp:  active,
	}

	desc, err := NewService(store).DescribeRuntime(runtimeID)
	if err != nil {
		t.Fatalf("DescribeRuntime: %v", err)
	}
	if desc == nil || desc.Handle == nil || desc.Handle.ID != 77 {
		t.Fatalf("descripcion inesperada: %+v", desc)
	}
	if store.handlesFilter != nil {
		t.Fatalf("no deberia barrer todos los handles si el handle activo canónico ya coincide")
	}
	if desc.Worker == nil || desc.Worker.Driver != "tmux_cli_session" {
		t.Fatalf("worker inesperado: %+v", desc.Worker)
	}
}

func TestDescribeRuntimeUsaHandlesCanonicosCalientesAntesDelBarridoCompleto(t *testing.T) {
	now := time.Now().UTC()
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "manifest.json")
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")
	writeJSON := func(path string, payload map[string]any) {
		t.Helper()
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	writeJSON(manifestPath, map[string]any{
		"version":      1,
		"agent":        "Codex3",
		"driver":       "tmux_cli_session",
		"transport":    "tmux",
		"tmux_session": "orq-codex3-1",
		"tmux_pane_id": "%12",
	})
	writeJSON(statusPath, map[string]any{
		"state":          "running",
		"updated_at":     now.Format(time.RFC3339Nano),
		"alive":          true,
		"last_output_at": now.Format(time.RFC3339Nano),
	})
	writeJSON(heartbeatPath, map[string]any{
		"alive":        true,
		"heartbeat_at": now.Format(time.RFC3339Nano),
	})
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})

	runtimeID := int64(35)
	projectID := int64(36)
	canonical := &db.RuntimeHandle{
		ID:           88,
		Agente:       "Codex3",
		ProyectoID:   &projectID,
		RuntimeID:    &runtimeID,
		Estado:       "activo",
		MetadataJSON: string(metaJSON),
	}
	store := &fakeStore{
		runtimeResponse:          &db.RuntimeInstance{ID: runtimeID, Agente: "Codex3", ProyectoID: &projectID},
		canonicalHandlesResponse: []*db.RuntimeHandle{canonical},
	}

	desc, err := NewService(store).DescribeRuntime(runtimeID)
	if err != nil {
		t.Fatalf("DescribeRuntime: %v", err)
	}
	if desc == nil || desc.Handle == nil || desc.Handle.ID != 88 {
		t.Fatalf("descripcion inesperada: %+v", desc)
	}
	if store.canonicalHandlesFilter == nil || *store.canonicalHandlesFilter != "Codex3" {
		t.Fatalf("deberia consultar handles canonicos calientes: %+v", store.canonicalHandlesFilter)
	}
	if store.handlesFilter != nil {
		t.Fatalf("no deberia caer al barrido completo si el snapshot canonico ya resuelve el runtime")
	}
	if desc.Worker == nil || desc.Worker.Driver != "tmux_cli_session" {
		t.Fatalf("worker inesperado: %+v", desc.Worker)
	}
}

func TestEnqueueAgentControlCreatesRuntimeOrder(t *testing.T) {
	projectID := int64(9)
	runtimeID := int64(12)
	store := &fakeStore{
		projectResponse: &db.Proyecto{ID: projectID, Slug: "orquestador"},
		agentResponse:   &db.Agente{Nombre: "Codex2", Rol: "programador"},
		handleProjResp:  &db.RuntimeHandle{ID: 10, RuntimeID: &runtimeID},
		createOrderID:   77,
	}
	service := NewService(store)

	orderID, accion, err := service.EnqueueAgentControl(AgentControlRequest{
		Agente:       "Codex2",
		Proyecto:     "orquestador",
		Accion:       "reanudar",
		Conector:     "codex-cli",
		Modelo:       "gpt-5.4",
		Razonamiento: "high",
		Perfil:       "implementacion",
		Motivo:       "prueba",
		Por:          "test",
	})
	if err != nil {
		t.Fatalf("EnqueueAgentControl: %v", err)
	}
	if orderID != 77 || accion != "resume" {
		t.Fatalf("orderID=%d accion=%s", orderID, accion)
	}
	if store.agentName != "Codex2" {
		t.Fatalf("agentName=%q", store.agentName)
	}
	if store.createOrder == nil {
		t.Fatalf("createOrder nil")
	}
	if store.createOrder.Tipo != "resume" || store.createOrder.HandleID == nil || *store.createOrder.HandleID != 10 {
		t.Fatalf("createOrder=%+v", store.createOrder)
	}
	if store.createOrder.RuntimeID == nil || *store.createOrder.RuntimeID != runtimeID {
		t.Fatalf("runtimeID=%+v", store.createOrder.RuntimeID)
	}
	if store.auditAction != "control_agente_resume" || store.auditEntity != "runtime_order" || store.auditEntityID != 77 {
		t.Fatalf("audit=%s %s %d", store.auditAction, store.auditEntity, store.auditEntityID)
	}
}

func TestEnqueueAgentControlRechazaStartSiCuentaCompartidaEstaOcupada(t *testing.T) {
	projectID := int64(9)
	store := &fakeStore{
		projectResponse:         &db.Proyecto{ID: projectID, Slug: "orquestador"},
		agentResponse:           &db.Agente{Nombre: "Codex11", Rol: "programador"},
		sharedAccountAllowSet:   true,
		sharedAccountAllowed:    false,
		sharedAccountOccupiedBy: "Codex10",
	}
	service := NewService(store)

	_, _, err := service.EnqueueAgentControl(AgentControlRequest{
		Agente:   "Codex11",
		Proyecto: "orquestador",
		Accion:   "arrancar",
		Motivo:   "prueba cuenta compartida",
		Por:      "test",
	})
	if err == nil || !strings.Contains(err.Error(), "Codex10") {
		t.Fatalf("deberia rechazar start por cuenta compartida ocupada, got=%v", err)
	}
	if store.createOrder != nil {
		t.Fatalf("no deberia encolar runtime order si la cuenta compartida ya está ocupada: %+v", store.createOrder)
	}
}

func TestEnqueueAgentControlStartSupersedesPausedHandleConStopPendiente(t *testing.T) {
	projectID := int64(9)
	runtimeID := int64(12)
	store := &fakeStore{
		projectResponse: &db.Proyecto{ID: projectID, Slug: "orquestador"},
		agentResponse:   &db.Agente{Nombre: "Codex8", Rol: "programador"},
		handleProjResp:  &db.RuntimeHandle{ID: 10, RuntimeID: &runtimeID, Estado: "pausado"},
		ordersResponse:  []*db.RuntimeOrder{{ID: 91, Agente: "Codex8", Tipo: "stop", Estado: "pendiente"}},
		createOrderID:   92,
	}
	service := NewService(store)

	orderID, accion, err := service.EnqueueAgentControl(AgentControlRequest{
		Agente:   "Codex8",
		Proyecto: "orquestador",
		Accion:   "arrancar",
		Motivo:   "rearmado tmux",
		Por:      "test",
	})
	if err != nil {
		t.Fatalf("EnqueueAgentControl start supersede: %v", err)
	}
	if orderID != 92 || accion != "start" {
		t.Fatalf("orderID=%d accion=%s", orderID, accion)
	}
	if store.ordersFilter.Agente == nil || *store.ordersFilter.Agente != "Codex8" {
		t.Fatalf("ordersFilter agente inesperado: %+v", store.ordersFilter)
	}
	if store.ordersFilter.Estado == nil || *store.ordersFilter.Estado != "pendiente" {
		t.Fatalf("ordersFilter estado inesperado: %+v", store.ordersFilter)
	}
	if store.createOrder == nil || store.createOrder.Tipo != "start" {
		t.Fatalf("createOrder inesperado: %+v", store.createOrder)
	}
	if store.createOrder.HandleID != nil || store.createOrder.RuntimeID != nil {
		t.Fatalf("start supersede no deberia atarse al handle viejo: %+v", store.createOrder)
	}
}

func TestEnqueueAgentControlResumeUsaRuntimePrincipalCanonicoSobreHandleViejo(t *testing.T) {
	projectID := int64(9)
	legacyRuntimeID := int64(12)
	canonicalRuntimeID := int64(99)
	store := &fakeStore{
		projectResponse:        &db.Proyecto{ID: projectID, Slug: "orquestador"},
		agentResponse:          &db.Agente{Nombre: "Codex8", Rol: "programador"},
		handleProjResp:         &db.RuntimeHandle{ID: 10, RuntimeID: &legacyRuntimeID, Estado: "activo"},
		primaryProjRuntimeResp: &db.RuntimeInstance{ID: canonicalRuntimeID, LogicalState: "esperando_io", ProcessState: "running"},
		createOrderID:          95,
	}
	service := NewService(store)

	orderID, accion, err := service.EnqueueAgentControl(AgentControlRequest{
		Agente:   "Codex8",
		Proyecto: "orquestador",
		Accion:   "reanudar",
		Motivo:   "resume canonico",
		Por:      "test",
	})
	if err != nil {
		t.Fatalf("EnqueueAgentControl resume canonical runtime: %v", err)
	}
	if orderID != 95 || accion != "resume" {
		t.Fatalf("orderID=%d accion=%s", orderID, accion)
	}
	if store.primaryProjRuntimeAgent != "Codex8" || store.primaryProjRuntimeID == nil || *store.primaryProjRuntimeID != projectID {
		t.Fatalf("deberia consultar el runtime principal del proyecto: agente=%q proyecto=%v", store.primaryProjRuntimeAgent, store.primaryProjRuntimeID)
	}
	if store.createOrder == nil || store.createOrder.Tipo != "resume" {
		t.Fatalf("createOrder inesperado: %+v", store.createOrder)
	}
	if store.createOrder.RuntimeID == nil || *store.createOrder.RuntimeID != canonicalRuntimeID {
		t.Fatalf("resume deberia apuntar al runtime canónico: %+v", store.createOrder)
	}
	if store.createOrder.HandleID == nil || *store.createOrder.HandleID != 10 {
		t.Fatalf("resume deberia conservar el handle activo: %+v", store.createOrder)
	}
}

func TestEnqueueAgentControlStartSupersedesActiveHandleConRuntimeDegradado(t *testing.T) {
	projectID := int64(9)
	runtimeID := int64(12)
	store := &fakeStore{
		projectResponse:        &db.Proyecto{ID: projectID, Slug: "orquestador"},
		agentResponse:          &db.Agente{Nombre: "Codex8", Rol: "programador"},
		handleProjResp:         &db.RuntimeHandle{ID: 10, RuntimeID: &runtimeID, Estado: "activo"},
		primaryProjRuntimeResp: &db.RuntimeInstance{ID: runtimeID, LogicalState: "degradado", ProcessState: "remote_status_error"},
		createOrderID:          93,
	}
	service := NewService(store)

	orderID, accion, err := service.EnqueueAgentControl(AgentControlRequest{
		Agente:   "Codex8",
		Proyecto: "orquestador",
		Accion:   "arrancar",
		Motivo:   "recovery",
		Por:      "test",
	})
	if err != nil {
		t.Fatalf("EnqueueAgentControl start degraded supersede: %v", err)
	}
	if orderID != 93 || accion != "start" {
		t.Fatalf("orderID=%d accion=%s", orderID, accion)
	}
	if store.primaryProjRuntimeAgent != "Codex8" || store.primaryProjRuntimeID == nil || *store.primaryProjRuntimeID != projectID {
		t.Fatalf("deberia consultar primero el runtime principal del proyecto: agente=%q proyecto=%v", store.primaryProjRuntimeAgent, store.primaryProjRuntimeID)
	}
	if store.runtimeID != 0 {
		t.Fatalf("no deberia caer al runtime pegado al handle si ya existe runtime principal: got=%d", store.runtimeID)
	}
	if store.createOrder == nil || store.createOrder.Tipo != "start" {
		t.Fatalf("createOrder inesperado: %+v", store.createOrder)
	}
	if store.createOrder.HandleID != nil || store.createOrder.RuntimeID != nil {
		t.Fatalf("start supersede por runtime degradado no deberia atarse al handle viejo: %+v", store.createOrder)
	}
}

func TestEnqueueAgentControlStartSupersedesHandleActivoUsandoRuntimeCanonicoMasReciente(t *testing.T) {
	projectID := int64(9)
	legacyRuntimeID := int64(12)
	canonicalRuntimeID := int64(99)
	store := &fakeStore{
		projectResponse:        &db.Proyecto{ID: projectID, Slug: "orquestador"},
		agentResponse:          &db.Agente{Nombre: "Codex8", Rol: "programador"},
		handleProjResp:         &db.RuntimeHandle{ID: 10, RuntimeID: &legacyRuntimeID, Estado: "activo"},
		primaryProjRuntimeResp: &db.RuntimeInstance{ID: canonicalRuntimeID, LogicalState: "degradado", ProcessState: "remote_status_error"},
		createOrderID:          94,
	}
	service := NewService(store)

	orderID, accion, err := service.EnqueueAgentControl(AgentControlRequest{
		Agente:   "Codex8",
		Proyecto: "orquestador",
		Accion:   "arrancar",
		Motivo:   "recovery",
		Por:      "test",
	})
	if err != nil {
		t.Fatalf("EnqueueAgentControl canonical supersede: %v", err)
	}
	if orderID != 94 || accion != "start" {
		t.Fatalf("orderID=%d accion=%s", orderID, accion)
	}
	if store.primaryProjRuntimeAgent != "Codex8" || store.primaryProjRuntimeID == nil || *store.primaryProjRuntimeID != projectID {
		t.Fatalf("runtime principal canónico no consultado: agente=%q proyecto=%v", store.primaryProjRuntimeAgent, store.primaryProjRuntimeID)
	}
	if store.runtimeID != 0 {
		t.Fatalf("no deberia usar el runtimeID viejo del handle: got=%d", store.runtimeID)
	}
	if store.createOrder == nil || store.createOrder.Tipo != "start" {
		t.Fatalf("createOrder inesperado: %+v", store.createOrder)
	}
	if store.createOrder.HandleID != nil || store.createOrder.RuntimeID != nil {
		t.Fatalf("el start supersede no deberia quedar atado al handle legacy: %+v", store.createOrder)
	}
}

func TestEnqueueAgentControlStartNoSupersedesActiveHandleSano(t *testing.T) {
	projectID := int64(9)
	runtimeID := int64(12)
	store := &fakeStore{
		projectResponse: &db.Proyecto{ID: projectID, Slug: "orquestador"},
		agentResponse:   &db.Agente{Nombre: "Codex8", Rol: "programador"},
		handleProjResp:  &db.RuntimeHandle{ID: 10, RuntimeID: &runtimeID, Estado: "activo"},
		runtimeResponse: &db.RuntimeInstance{ID: runtimeID, LogicalState: "esperando_io", ProcessState: "running"},
	}
	service := NewService(store)

	_, _, err := service.EnqueueAgentControl(AgentControlRequest{
		Agente:   "Codex8",
		Proyecto: "orquestador",
		Accion:   "arrancar",
		Motivo:   "recovery",
		Por:      "test",
	})
	if err == nil || !strings.Contains(err.Error(), "ya tiene un runtime handle activo") {
		t.Fatalf("deberia rechazar supersede sobre handle sano, got=%v", err)
	}
}

func TestEnqueueAgentControlStartSupersedesTMUXSinSesionAunqueSnapshotDigaRunning(t *testing.T) {
	tmp := t.TempDir()
	statusPath := filepath.Join(tmp, "worker-status.json")
	heartbeatPath := filepath.Join(tmp, "worker-heartbeat.json")
	fakeTmux := filepath.Join(tmp, "fake-tmux-missing-session")
	now := time.Now().UTC()
	if err := os.WriteFile(fakeTmux, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("write fake tmux: %v", err)
	}
	statusRaw, _ := json.Marshal(map[string]any{
		"state":      "running",
		"alive":      true,
		"updated_at": now.Format(time.RFC3339Nano),
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
	projectID := int64(9)
	runtimeID := int64(12)
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"tmux_command":          fakeTmux,
		"tmux_session":          "orq-gemma1-missing-session",
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	handle := &db.RuntimeHandle{
		ID:           10,
		RuntimeID:    &runtimeID,
		Estado:       "activo",
		Transporte:   "tmux",
		HandleKind:   "session",
		HandleRef:    "orq-gemma1-missing-session",
		MetadataJSON: string(metaJSON),
	}
	store := &fakeStore{
		projectResponse: &db.Proyecto{ID: projectID, Slug: "orquestador"},
		agentResponse:   &db.Agente{Nombre: "Gemma1", Rol: "programador"},
		handleProjResp:  handle,
		handleByIDResp:  handle,
		runtimeResponse: &db.RuntimeInstance{ID: runtimeID, LogicalState: "esperando_io", ProcessState: "running"},
		createOrderID:   96,
	}
	service := NewService(store)

	orderID, accion, err := service.EnqueueAgentControl(AgentControlRequest{
		Agente:   "Gemma1",
		Proyecto: "orquestador",
		Accion:   "arrancar",
		Motivo:   "recovery",
		Por:      "test",
	})
	if err != nil {
		t.Fatalf("EnqueueAgentControl tmux missing session supersede: %v", err)
	}
	if orderID != 96 || accion != "start" {
		t.Fatalf("orderID=%d accion=%s", orderID, accion)
	}
	if store.createOrder == nil {
		t.Fatalf("faltaba createOrder")
	}
	if store.createOrder.HandleID != nil || store.createOrder.RuntimeID != nil {
		t.Fatalf("start no deberia quedar atado a handle tmux sin sesion: %+v", store.createOrder)
	}
}

func TestEnqueueAgentControlStartSupersedeHandleTrasSyncDegradado(t *testing.T) {
	projectID := int64(9)
	runtimeID := int64(12)
	store := &fakeStore{
		projectResponse: &db.Proyecto{ID: projectID, Slug: "orquestador"},
		agentResponse:   &db.Agente{Nombre: "Codex8", Rol: "programador"},
		handleProjResp:  &db.RuntimeHandle{ID: 10, RuntimeID: &runtimeID, Estado: "activo"},
		syncHandleResp:  &db.RuntimeHandle{ID: 10, RuntimeID: &runtimeID, Estado: "fallido"},
		runtimeResponse: &db.RuntimeInstance{ID: runtimeID, LogicalState: "degradado", ProcessState: "remote_status_error"},
		createOrderID:   95,
	}
	service := NewService(store)

	orderID, accion, err := service.EnqueueAgentControl(AgentControlRequest{
		Agente:   "Codex8",
		Proyecto: "orquestador",
		Accion:   "arrancar",
		Motivo:   "recovery",
		Por:      "test",
	})
	if err != nil {
		t.Fatalf("EnqueueAgentControl sync supersede: %v", err)
	}
	if orderID != 95 || accion != "start" {
		t.Fatalf("orderID=%d accion=%s", orderID, accion)
	}
	if store.syncHandleInput == nil || store.syncHandleInput.ID != 10 {
		t.Fatalf("faltaba sync del handle previo al start: %+v", store.syncHandleInput)
	}
	if store.createOrder == nil || store.createOrder.HandleID != nil || store.createOrder.RuntimeID != nil {
		t.Fatalf("el start supersede no deberia quedar atado a handle degradado: %+v", store.createOrder)
	}
}

func TestSelectBestRuntimeHandleDescartaLegacyProcessPTYSiExisteWorkerNoLegacy(t *testing.T) {
	runtimeID := int64(12)
	runtime := &db.RuntimeInstance{ID: runtimeID, Agente: "Codex3"}
	legacy := &db.RuntimeHandle{
		ID:           10,
		Agente:       "Codex3",
		RuntimeID:    &runtimeID,
		Estado:       "activo",
		HandleKind:   "process",
		MetadataJSON: `{"driver":"process_pty_cli"}`,
	}
	tmux := &db.RuntimeHandle{
		ID:           11,
		Agente:       "Codex3",
		Estado:       "activo",
		HandleKind:   "session",
		MetadataJSON: `{"driver":"tmux_cli_session"}`,
	}

	best := selectBestRuntimeHandle(runtime, []*db.RuntimeHandle{legacy, tmux})
	if best == nil || best.ID != tmux.ID {
		t.Fatalf("deberia preferir worker no legacy sobre process_pty_cli: %+v", best)
	}
}

func TestSelectBestRuntimeHandleAceptaLegacySiNoHayAlternativa(t *testing.T) {
	runtimeID := int64(12)
	runtime := &db.RuntimeInstance{ID: runtimeID, Agente: "Codex3"}
	legacy := &db.RuntimeHandle{
		ID:           10,
		Agente:       "Codex3",
		RuntimeID:    &runtimeID,
		Estado:       "activo",
		HandleKind:   "process",
		MetadataJSON: `{"driver":"process_pty_cli"}`,
	}

	best := selectBestRuntimeHandle(runtime, []*db.RuntimeHandle{legacy})
	if best == nil || best.ID != legacy.ID {
		t.Fatalf("sin alternativa no deberia descartar el legacy: %+v", best)
	}
}

func TestPurgeInactiveRuntimeHandlesDelegatesAndAudits(t *testing.T) {
	projectID := int64(9)
	store := &fakeStore{
		projectResponse: &db.Proyecto{ID: projectID, Slug: "orquestador"},
		agentResponse:   &db.Agente{Nombre: "Codex5", Rol: "programador"},
		purgeResponse:   &db.PurgaRuntimeHandlesResultado{Deleted: 2, DeletedIDs: []int64{48, 39}, Estados: []string{"cerrado", "fallido"}},
	}
	service := NewService(store)

	resultado, err := service.PurgeInactiveRuntimeHandles(RuntimeHandlePurgeRequest{
		Agente:   "Codex5",
		Proyecto: "orquestador",
		Estados:  []string{"cerrado", "fallido"},
		Actor:    "Codex1",
	})
	if err != nil {
		t.Fatalf("PurgeInactiveRuntimeHandles: %v", err)
	}
	if resultado == nil || resultado.Deleted != 2 {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
	if store.purgeFilter.Agente == nil || *store.purgeFilter.Agente != "Codex5" {
		t.Fatalf("purgeFilter agente=%v", store.purgeFilter.Agente)
	}
	if store.purgeFilter.ProyectoID == nil || *store.purgeFilter.ProyectoID != projectID {
		t.Fatalf("purgeFilter proyecto=%v", store.purgeFilter.ProyectoID)
	}
	if strings.Join(store.purgeFilter.Estados, ",") != "cerrado,fallido" {
		t.Fatalf("purgeFilter estados=%v", store.purgeFilter.Estados)
	}
	if store.auditAction != "purgar_runtime_handles" || store.auditAgent != "Codex1" {
		t.Fatalf("audit inesperado agente=%q accion=%q detalle=%q", store.auditAgent, store.auditAction, store.auditDetail)
	}
}

func ptrInt64RuntimeServiceTest(v int64) *int64 {
	return &v
}

func mustStructuredWorkerMetadataJSONRuntimesTest(t *testing.T, heartbeatAt time.Time, state string, alive bool) string {
	t.Helper()
	tmp := t.TempDir()
	heartbeatAt = heartbeatAt.UTC()
	workingDir := filepath.Join(tmp, "repo")
	runDir := filepath.Join(workingDir, ".orquesta-runtime", "codexfresh", "20260407-010203-000000001")
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")
	manifestPath := filepath.Join(tmp, "manifest.json")
	runtimeManifestPath := filepath.Join(runDir, "runtime.json")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir runDir: %v", err)
	}
	statusRaw, err := json.Marshal(map[string]any{
		"state":      state,
		"updated_at": heartbeatAt.Format(time.RFC3339Nano),
		"alive":      alive,
		"child_pid":  os.Getpid(),
	})
	if err != nil {
		t.Fatalf("marshal status: %v", err)
	}
	heartbeatRaw, err := json.Marshal(map[string]any{
		"alive":        alive,
		"heartbeat_at": heartbeatAt.Format(time.RFC3339Nano),
		"child_pid":    os.Getpid(),
	})
	if err != nil {
		t.Fatalf("marshal heartbeat: %v", err)
	}
	manifestRaw, err := json.Marshal(map[string]any{
		"version":               1,
		"agent":                 "CodexFresh",
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_session":          "orq-codexfresh-1",
		"tmux_pane_id":          "%1",
		"status_path":           statusPath,
		"heartbeat_path":        heartbeatPath,
		"started_at":            heartbeatAt.Add(-time.Minute).Format(time.RFC3339Nano),
		"working_dir":           workingDir,
		"child_pid":             os.Getpid(),
		"runtime_manifest_path": runtimeManifestPath,
	})
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, statusRaw, 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, heartbeatRaw, 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	if err := os.WriteFile(manifestPath, manifestRaw, 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	runtimeManifestRaw, err := json.Marshal(map[string]any{
		"agente":                "CodexFresh",
		"proyecto":              "orquestador",
		"working_dir":           workingDir,
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"pid":                   os.Getpid(),
		"supervisor_ref":        runDir,
		"tmux_session":          "orq-codexfresh-1",
		"tmux_pane_id":          "%1",
		"status_path":           statusPath,
		"heartbeat_path":        heartbeatPath,
		"mailbox_delivery_mode": "session_resume",
		"created_at":            heartbeatAt.Add(-time.Minute).Format(time.RFC3339Nano),
	})
	if err != nil {
		t.Fatalf("marshal runtime manifest: %v", err)
	}
	if err := os.WriteFile(runtimeManifestPath, append(runtimeManifestRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write runtime manifest: %v", err)
	}
	metaRaw, err := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"working_dir":           workingDir,
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	return string(metaRaw)
}

func TestPurgeTerminalRuntimeOrdersAplicaCutoffYAudita(t *testing.T) {
	projectID := int64(9)
	store := &fakeStore{
		projectResponse:     &db.Proyecto{ID: projectID, Slug: "orquestador"},
		agentResponse:       &db.Agente{Nombre: "Codex5", Rol: "programador"},
		purgeOrdersResponse: &db.PurgaRuntimeOrdersResultado{Deleted: 3, DeletedIDs: []int64{91, 88, 77}, Estados: []string{"completada", "fallida"}, Tipos: []string{"send_instruction"}},
	}
	service := NewService(store)
	beforeCall := time.Now().UTC()

	resultado, err := service.PurgeTerminalRuntimeOrders(RuntimeOrderPurgeRequest{
		Agente:           "Codex5",
		Proyecto:         "orquestador",
		Estados:          []string{"completada", "fallida"},
		Tipos:            []string{"send_instruction"},
		OlderThanMinutes: 60,
		Actor:            "Codex1",
	})
	if err != nil {
		t.Fatalf("PurgeTerminalRuntimeOrders: %v", err)
	}
	if resultado == nil || resultado.Deleted != 3 {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
	if store.purgeOrdersFilter.Agente == nil || *store.purgeOrdersFilter.Agente != "Codex5" {
		t.Fatalf("purgeOrdersFilter agente=%v", store.purgeOrdersFilter.Agente)
	}
	if store.purgeOrdersFilter.ProyectoID == nil || *store.purgeOrdersFilter.ProyectoID != projectID {
		t.Fatalf("purgeOrdersFilter proyecto=%v", store.purgeOrdersFilter.ProyectoID)
	}
	if store.purgeOrdersFilter.CreatedBefore == nil {
		t.Fatalf("purgeOrdersFilter.CreatedBefore deberia estar informado")
	}
	if got := store.purgeOrdersFilter.CreatedBefore.UTC(); got.After(beforeCall.Add(-59*time.Minute)) || got.Before(beforeCall.Add(-61*time.Minute)) {
		t.Fatalf("cutoff inesperado: %s", got.Format(time.RFC3339))
	}
	if store.auditAction != "purgar_runtime_orders" || store.auditAgent != "Codex1" || !strings.Contains(store.auditDetail, "older_than_minutes=60") {
		t.Fatalf("audit inesperado agente=%q accion=%q detalle=%q", store.auditAgent, store.auditAction, store.auditDetail)
	}
}

func TestServiceReadRuntimeTracePorAgenteProyecto(t *testing.T) {
	tmp := t.TempDir()
	traceDir := filepath.Join(tmp, ".orquesta-runtime", "codex2", "20260329-120000-000000001")
	if err := os.MkdirAll(traceDir, 0o700); err != nil {
		t.Fatalf("mkdir trace dir: %v", err)
	}
	logPath := filepath.Join(traceDir, "pty.log")
	manifestPath := filepath.Join(traceDir, "runtime.json")
	if err := os.WriteFile(logPath, []byte("inicio\navance\nfin\n"), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	if err := os.WriteFile(manifestPath, []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	projectID := int64(7)
	metaJSON, _ := json.Marshal(map[string]any{
		"trace_dir":      traceDir,
		"trace_manifest": manifestPath,
		"log_path":       logPath,
		"working_dir":    "/repo",
		"driver":         "process_pty_cli",
	})
	store := &fakeStore{
		projectResponse: &db.Proyecto{ID: projectID, Slug: "orquestador"},
		operationalProjResp: &db.RuntimeHandle{
			ID:           17,
			Agente:       "Codex2",
			ProyectoID:   &projectID,
			MetadataJSON: string(metaJSON),
		},
	}

	trace, err := NewService(store).ReadRuntimeTrace(RuntimeTraceRequest{
		Agente:   "Codex2",
		Proyecto: "orquestador",
		MaxBytes: 10,
	})
	if err != nil {
		t.Fatalf("ReadRuntimeTrace: %v", err)
	}
	if store.projectRef != "orquestador" {
		t.Fatalf("projectRef=%q", store.projectRef)
	}
	if store.operationalProjAgent != "Codex2" || store.operationalProjID == nil || *store.operationalProjID != projectID {
		t.Fatalf("operationalProj agent=%q project=%v", store.operationalProjAgent, store.operationalProjID)
	}
	if !trace.Available {
		t.Fatalf("trace no disponible: %+v", trace)
	}
	if trace.HandleID != 17 || trace.Agente != "Codex2" || trace.Proyecto != "orquestador" {
		t.Fatalf("trace basica inesperada: %+v", trace)
	}
	if trace.TraceManifest != manifestPath || trace.LogPath != logPath {
		t.Fatalf("paths de trace inesperados: %+v", trace)
	}
	if !trace.Truncated || trace.TotalBytes == 0 || trace.BytesRead == 0 {
		t.Fatalf("trace truncada inesperada: %+v", trace)
	}
	if !strings.Contains(trace.RawTail, "ance\nfin\n") {
		t.Fatalf("tail inesperado: %q", trace.RawTail)
	}
}
