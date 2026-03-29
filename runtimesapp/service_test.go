package runtimesapp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/db"
)

type fakeStore struct {
	projectRef       string
	projectResponse  *db.Proyecto
	agentName        string
	agentResponse    *db.Agente
	runtimesFilter   db.FiltroRuntimes
	runtimesResponse []*db.RuntimeInstance
	treeFilter       db.FiltroRuntimes
	treeResponse     []*db.RuntimeTreeNode
	runtimeID        int64
	runtimeResponse  *db.RuntimeInstance
	handleID         int64
	handleByIDResp   *db.RuntimeHandle
	handlesFilter    *string
	handlesResponse  []*db.RuntimeHandle
	purgeFilter      db.FiltroPurgadoRuntimeHandles
	purgeResponse    *db.PurgaRuntimeHandlesResultado
	handleAgent      string
	handleResponse   *db.RuntimeHandle
	handleProjAgent  string
	handleProjID     *int64
	handleProjResp   *db.RuntimeHandle
	transcriptFilter db.FiltroRuntimeTranscript
	transcriptResp   []*db.RuntimeTranscriptEntry
	samplesID        int64
	samplesLimit     int
	samplesResponse  []*db.RuntimeTelemetrySample
	createOrder      *db.RuntimeOrder
	createOrderID    int64
	ordersFilter     db.FiltroRuntimeOrders
	ordersResponse   []*db.RuntimeOrder
	createMailbox    *db.RuntimeMailboxMessage
	createMailboxID  int64
	mailboxFilter    db.FiltroRuntimeMailbox
	mailboxResponse  []*db.RuntimeMailboxMessage
	deliveredID      int64
	consumedID       int64
	createCheck      *db.RuntimeCheckpoint
	createCheckID    int64
	checkFilter      db.FiltroRuntimeCheckpoints
	checkResponse    []*db.RuntimeCheckpoint
	checkpointID     int64
	checkpointResp   *db.RuntimeCheckpoint
	latestAgent      string
	latestProjectID  *int64
	latestResp       *db.RuntimeCheckpoint
	memoryFilter     db.FiltroEntidadesMemoria
	memoryResponse   []*db.EntidadMemoria
	memoryUpsert     *db.EntidadMemoria
	memoryUpsertID   int64
	memoryName       string
	memoryProjectID  *int64
	memoryEntityResp *db.EntidadMemoria
	auditAgent       string
	auditAction      string
	auditEntity      string
	auditEntityID    int64
	auditDetail      string
}

func (f *fakeStore) GetProject(ref string) (*db.Proyecto, error) {
	f.projectRef = ref
	return f.projectResponse, nil
}
func (f *fakeStore) GetAgent(nombre string) (*db.Agente, error) {
	f.agentName = nombre
	return f.agentResponse, nil
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
func (f *fakeStore) GetRuntimeHandle(id int64) (*db.RuntimeHandle, error) {
	f.handleID = id
	return f.handleByIDResp, nil
}
func (f *fakeStore) ListRuntimeHandles(filter *string) ([]*db.RuntimeHandle, error) {
	f.handlesFilter = filter
	return f.handlesResponse, nil
}
func (f *fakeStore) PurgeInactiveRuntimeHandles(filter db.FiltroPurgadoRuntimeHandles) (*db.PurgaRuntimeHandlesResultado, error) {
	f.purgeFilter = filter
	return f.purgeResponse, nil
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
func (f *fakeStore) ListRuntimeOrders(filter db.FiltroRuntimeOrders) ([]*db.RuntimeOrder, error) {
	f.ordersFilter = filter
	return f.ordersResponse, nil
}
func (f *fakeStore) CreateRuntimeMailbox(msg *db.RuntimeMailboxMessage) (int64, error) {
	f.createMailbox = msg
	return f.createMailboxID, nil
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
		projectResponse:  &db.Proyecto{Slug: "orquestador"},
		agentResponse:    &db.Agente{Nombre: "Codex2", Rol: "programador"},
		runtimesResponse: []*db.RuntimeInstance{{ID: 5}},
		treeResponse:     []*db.RuntimeTreeNode{{Runtime: &db.RuntimeInstance{ID: 10}}},
		runtimeResponse:  &db.RuntimeInstance{ID: 10},
		handleByIDResp:   &db.RuntimeHandle{ID: 16},
		handlesResponse:  []*db.RuntimeHandle{{ID: 15}},
		handleResponse:   &db.RuntimeHandle{ID: 16},
		handleProjResp:   &db.RuntimeHandle{ID: 17},
		transcriptResp:   []*db.RuntimeTranscriptEntry{{ID: 18}},
		samplesResponse:  []*db.RuntimeTelemetrySample{{ID: 20}},
		createOrderID:    25,
		ordersResponse:   []*db.RuntimeOrder{{ID: 30}},
		createMailboxID:  35,
		mailboxResponse:  []*db.RuntimeMailboxMessage{{ID: 40}},
		createCheckID:    45,
		checkResponse:    []*db.RuntimeCheckpoint{{ID: 50}},
		checkpointResp:   &db.RuntimeCheckpoint{ID: 50},
		latestResp:       &db.RuntimeCheckpoint{ID: 55},
		memoryResponse:   []*db.EntidadMemoria{{ID: 60}},
		memoryUpsertID:   61,
		memoryEntityResp: &db.EntidadMemoria{ID: 62, Nombre: "decision"},
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
	if _, err := service.GetActiveRuntimeHandle(agent); err != nil {
		t.Fatalf("GetActiveRuntimeHandle: %v", err)
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
		handleProjResp: &db.RuntimeHandle{
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
	if store.handleProjAgent != "Codex2" || store.handleProjID == nil || *store.handleProjID != projectID {
		t.Fatalf("handleProj agent=%q project=%v", store.handleProjAgent, store.handleProjID)
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
