package runtimesapp

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orquesta/db"
	"orquesta/microprogramacionapp"
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
	reconcileOrdersCalls     int
	runtimesFilter           db.FiltroRuntimes
	runtimesResponse         []*db.RuntimeInstance
	treeFilter               db.FiltroRuntimes
	treeResponse             []*db.RuntimeTreeNode
	runtimeID                int64
	runtimeResponse          *db.RuntimeInstance
	runtimeHandleSessionID   int64
	runtimeHandleSessionResp *db.RuntimeHandle
	primaryRuntimeAgent      string
	primaryRuntimeResp       *db.RuntimeInstance
	primaryProjRuntimeAgent  string
	primaryProjRuntimeID     *int64
	primaryProjRuntimeResp   *db.RuntimeInstance
	primaryProjRuntimeErr    error
	closedRuntimeID          int64
	closedRuntimeIDs         []int64
	closeHandlesAgent        string
	closeHandlesProjectID    *int64
	handleID                 int64
	handleByIDResp           *db.RuntimeHandle
	handlesFilter            *string
	handlesResponse          []*db.RuntimeHandle
	passiveHandlesFilter     *string
	passiveHandlesResponse   []*db.RuntimeHandle
	canonicalHandlesFilter   *string
	canonicalHandlesResponse []*db.RuntimeHandle
	syncHandleInput          *db.RuntimeHandle
	syncHandleCalls          int
	syncHandleSource         string
	syncHandleResp           *db.RuntimeHandle
	syncHandleRespSet        bool
	purgeFilter              db.FiltroPurgadoRuntimeHandles
	purgeResponse            *db.PurgaRuntimeHandlesResultado
	purgeOrdersFilter        db.FiltroPurgadoRuntimeOrders
	purgeOrdersResponse      *db.PurgaRuntimeOrdersResultado
	handleAgent              string
	handleResponse           *db.RuntimeHandle
	handleProjAgent          string
	handleProjID             *int64
	handleProjResp           *db.RuntimeHandle
	handleProjErr            error
	operationalHandleAgent   string
	operationalHandleResp    *db.RuntimeHandle
	operationalHandleErr     error
	operationalProjAgent     string
	operationalProjID        *int64
	operationalProjResp      *db.RuntimeHandle
	operationalProjErr       error
	eventsFilter             db.FiltroRuntimeEvents
	eventsResp               []*db.RuntimeEvent
	transcriptFilter         db.FiltroRuntimeTranscript
	transcriptResp           []*db.RuntimeTranscriptEntry
	transcriptRegistrado     *db.RuntimeTranscriptEntry
	transcriptRegistradoID   int64
	samplesID                int64
	samplesLimit             int
	samplesResponse          []*db.RuntimeTelemetrySample
	createOrder              *db.RuntimeOrder
	createOrderID            int64
	orderID                  int64
	ordersFilter             db.FiltroRuntimeOrders
	ordersResponse           []*db.RuntimeOrder
	markOrderStateID         int64
	markOrderStateEstado     string
	markOrderStateResultado  string
	markOrderStateError      string
	markedOrderStates        []int64
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

type fakeRegistradorEntregaGit struct {
	id        int64
	entrada   microprogramacionapp.EntradaRegistrarEntregaGit
	resultado *microprogramacionapp.ResultadoRegistrarEntregaGit
	err       error
}

type fakeRegistradorEntregaGitPremium struct {
	entrada   EntradaRegistrarEntregaGitPremium
	resultado *ResultadoRegistrarEntregaGitPremium
	err       error
}

type fakeMaterializadorEntrega struct {
	id        int64
	raiz      string
	respuesta string
	evidencia string
	resultado *microprogramacionapp.ResultadoMaterializarEntrega
	err       error
}

type fakeResolvedorWorktree struct {
	proyectoSlug string
	agente       string
	resultado    *db.Worktree
	err          error
}

type fakeAseguradorWorktree struct {
	proyectoSlug string
	agente       string
	resultado    *db.Worktree
	err          error
}

type fakeTaskCompleter struct {
	id     int64
	agente string
	commit string
	err    error
}

func (f *fakeStore) GetProject(ref string) (*db.Proyecto, error) {
	f.projectRef = ref
	return f.projectResponse, nil
}

func (f *fakeRegistradorEntregaGit) RegistrarEntregaGit(id int64, entrada microprogramacionapp.EntradaRegistrarEntregaGit) (*microprogramacionapp.ResultadoRegistrarEntregaGit, error) {
	f.id = id
	f.entrada = entrada
	if f.err != nil {
		return nil, f.err
	}
	if f.resultado != nil {
		return f.resultado, nil
	}
	return &microprogramacionapp.ResultadoRegistrarEntregaGit{
		EspecificacionID:   id,
		ArchivoObjetivo:    "microprogramacionapp/service.go",
		WriteSetPermitido:  []string{"microprogramacionapp/service.go"},
		WorktreeID:         1,
		RutaWorktree:       "/tmp/worktree",
		SourceBranch:       "orq/test",
		TargetBranch:       "main",
		HeadCommit:         "abc123",
		ArchivosEntregados: []string{"microprogramacionapp/service.go"},
		GitMergeID:         55,
	}, nil
}

func (f *fakeRegistradorEntregaGitPremium) RegistrarEntregaGitPremium(entrada EntradaRegistrarEntregaGitPremium) (*ResultadoRegistrarEntregaGitPremium, error) {
	f.entrada = entrada
	if f.err != nil {
		return nil, f.err
	}
	if f.resultado != nil {
		return f.resultado, nil
	}
	return &ResultadoRegistrarEntregaGitPremium{
		WorktreeID:         81,
		RutaWorktree:       "/tmp/orq-premium",
		SourceBranch:       "orq/orquestador/codex1",
		TargetBranch:       "main",
		HeadCommit:         "abc123",
		ArchivosEntregados: []string{"cmd/controlplane_support.go"},
		GitMergeID:         77,
	}, nil
}

func (f *fakeMaterializadorEntrega) MaterializarEntregaDesdeRespuesta(id int64, raizProyecto, respuesta, evidencia string) (*microprogramacionapp.ResultadoMaterializarEntrega, error) {
	f.id = id
	f.raiz = raizProyecto
	f.respuesta = respuesta
	f.evidencia = evidencia
	if f.err != nil {
		return nil, f.err
	}
	if f.resultado != nil {
		return f.resultado, nil
	}
	return &microprogramacionapp.ResultadoMaterializarEntrega{
		EspecificacionID:       id,
		ArchivoObjetivo:        "microprogramacionapp/extractor_entrega.go",
		ArchivosMaterializados: []string{"microprogramacionapp/extractor_entrega.go"},
		WriteSetPermitido:      []string{"microprogramacionapp/extractor_entrega.go"},
	}, nil
}

func (f *fakeResolvedorWorktree) ResolveActiveWorktree(proyectoSlug, agente string) (*db.Worktree, error) {
	f.proyectoSlug = proyectoSlug
	f.agente = agente
	if f.err != nil {
		return nil, f.err
	}
	return f.resultado, nil
}

func (f *fakeAseguradorWorktree) EnsureActiveWorktree(proyectoSlug, agente string) (*db.Worktree, error) {
	f.proyectoSlug = proyectoSlug
	f.agente = agente
	if f.err != nil {
		return nil, f.err
	}
	return f.resultado, nil
}

func (f *fakeTaskCompleter) Complete(id int64, agente, commit string) error {
	f.id = id
	f.agente = agente
	f.commit = commit
	return f.err
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

func (f *fakeStore) ReconcileStaleRuntimeHandles() (int, error) {
	return 0, nil
}

func (f *fakeStore) ReconcileStaleRuntimeOrders() (int, error) {
	f.reconcileOrdersCalls++
	return 0, nil
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

func (f *fakeStore) GetRuntimeBySessionID(sessionID int64) (*db.RuntimeInstance, error) {
	return f.runtimeResponse, nil
}
func (f *fakeStore) GetRuntimeHandleBySessionID(sessionID int64) (*db.RuntimeHandle, error) {
	f.runtimeHandleSessionID = sessionID
	return f.runtimeHandleSessionResp, nil
}
func (f *fakeStore) GetPrimaryRuntime(agente string) (*db.RuntimeInstance, error) {
	f.primaryRuntimeAgent = agente
	return f.primaryRuntimeResp, nil
}
func (f *fakeStore) GetPrimaryRuntimeForProject(agente string, proyectoID *int64) (*db.RuntimeInstance, error) {
	f.primaryProjRuntimeAgent = agente
	f.primaryProjRuntimeID = proyectoID
	if f.primaryProjRuntimeErr != nil {
		return nil, f.primaryProjRuntimeErr
	}
	return f.primaryProjRuntimeResp, nil
}
func (f *fakeStore) CloseRuntime(id int64) error {
	f.closedRuntimeID = id
	f.closedRuntimeIDs = append(f.closedRuntimeIDs, id)
	return nil
}
func (f *fakeStore) CloseRuntimeHandles(agente string, proyectoID *int64) error {
	f.closeHandlesAgent = agente
	f.closeHandlesProjectID = proyectoID
	return nil
}
func (f *fakeStore) GetRuntimeHandle(id int64) (*db.RuntimeHandle, error) {
	f.handleID = id
	return f.handleByIDResp, nil
}
func (f *fakeStore) ListRuntimeHandles(filter *string) ([]*db.RuntimeHandle, error) {
	f.handlesFilter = filter
	return f.handlesResponse, nil
}
func (f *fakeStore) ListPassiveRuntimeHandles(filter *string) ([]*db.RuntimeHandle, error) {
	f.passiveHandlesFilter = filter
	if f.passiveHandlesResponse != nil {
		return f.passiveHandlesResponse, nil
	}
	return f.handlesResponse, nil
}
func (f *fakeStore) ListCanonicalRuntimeHandles(filter *string) ([]*db.RuntimeHandle, error) {
	f.canonicalHandlesFilter = filter
	return f.canonicalHandlesResponse, nil
}
func (f *fakeStore) SyncSupervisedRuntimeHandle(handle *db.RuntimeHandle, source string) (*db.RuntimeHandle, error) {
	f.syncHandleInput = handle
	f.syncHandleCalls++
	f.syncHandleSource = source
	if f.syncHandleRespSet {
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
	if f.handleProjErr != nil {
		return nil, f.handleProjErr
	}
	return f.handleProjResp, nil
}
func (f *fakeStore) GetOperationalRuntimeHandle(agente string) (*db.RuntimeHandle, error) {
	f.operationalHandleAgent = agente
	if f.operationalHandleErr != nil {
		return nil, f.operationalHandleErr
	}
	return f.operationalHandleResp, nil
}
func (f *fakeStore) GetOperationalRuntimeHandleForProject(agente string, proyectoID *int64) (*db.RuntimeHandle, error) {
	f.operationalProjAgent = agente
	f.operationalProjID = proyectoID
	if f.operationalProjErr != nil {
		return nil, f.operationalProjErr
	}
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
func (f *fakeStore) RegisterRuntimeTranscript(entry *db.RuntimeTranscriptEntry) (int64, error) {
	f.transcriptRegistrado = entry
	return f.transcriptRegistradoID, nil
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
	if len(f.ordersResponse) == 0 {
		return nil, nil
	}
	out := make([]*db.RuntimeOrder, 0, len(f.ordersResponse))
	for _, order := range f.ordersResponse {
		if order == nil {
			continue
		}
		if filter.Agente != nil && strings.TrimSpace(*filter.Agente) != "" && !strings.EqualFold(strings.TrimSpace(order.Agente), strings.TrimSpace(*filter.Agente)) {
			continue
		}
		if filter.Estado != nil && strings.TrimSpace(*filter.Estado) != "" && !strings.EqualFold(strings.TrimSpace(order.Estado), strings.TrimSpace(*filter.Estado)) {
			continue
		}
		if filter.ProyectoID != nil {
			if order.ProyectoID == nil || *order.ProyectoID != *filter.ProyectoID {
				continue
			}
		}
		out = append(out, order)
	}
	return out, nil
}
func (f *fakeStore) MarkRuntimeOrderState(id int64, estado, resultadoJSON, errorText string) error {
	f.markOrderStateID = id
	f.markOrderStateEstado = estado
	f.markOrderStateResultado = resultadoJSON
	f.markOrderStateError = errorText
	f.markedOrderStates = append(f.markedOrderStates, id)
	return nil
}

func TestCancelRuntimeOrderMarcaCanceladaYAudita(t *testing.T) {
	store := &fakeStore{
		ordersResponse: []*db.RuntimeOrder{{
			ID:            87,
			Agente:        "Codex1",
			Tipo:          "send_instruction",
			Estado:        "pendiente",
			ResultadoJSON: `{}`,
		}},
	}
	svc := NewService(store)

	if err := svc.CancelRuntimeOrder(87, "Codex1", "limpieza de prueba"); err != nil {
		t.Fatalf("CancelRuntimeOrder: %v", err)
	}
	if store.markOrderStateID != 87 || store.markOrderStateEstado != "cancelada" {
		t.Fatalf("orden no cancelada: id=%d estado=%q", store.markOrderStateID, store.markOrderStateEstado)
	}
	if !strings.Contains(store.markOrderStateResultado, `"cancelled_by":"Codex1"`) || !strings.Contains(store.markOrderStateResultado, `"reason":"limpieza de prueba"`) {
		t.Fatalf("resultado cancelado sin trazabilidad: %s", store.markOrderStateResultado)
	}
	if store.auditAction != "cancelar_runtime_order" || store.auditEntityID != 87 {
		t.Fatalf("audit inesperada: action=%q entity_id=%d", store.auditAction, store.auditEntityID)
	}
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

func TestListRuntimeHandlesSincronizaSoloSubsetOperativo(t *testing.T) {
	agent := "Codex2"
	store := &fakeStore{
		handlesResponse: []*db.RuntimeHandle{
			{ID: 11, Agente: agent, Estado: "activo"},
			{ID: 12, Agente: agent, Estado: "ready"},
			{ID: 13, Agente: agent, Estado: "cerrado"},
		},
	}
	service := NewService(store)

	if _, err := service.ListRuntimeHandles(&agent); err != nil {
		t.Fatalf("ListRuntimeHandles: %v", err)
	}
	if store.syncHandleCalls != 2 {
		t.Fatalf("syncHandleCalls=%d", store.syncHandleCalls)
	}
	if store.syncHandleInput == nil || store.syncHandleInput.ID != 12 {
		t.Fatalf("ultimo handle sincronizado=%+v", store.syncHandleInput)
	}
}

func TestListRuntimeHandlesCompactPrefiereCanonicosSinSincronizar(t *testing.T) {
	agent := "Codex2"
	store := &fakeStore{
		passiveHandlesResponse: []*db.RuntimeHandle{{ID: 21, Agente: agent, Estado: "activo"}},
	}
	service := NewService(store)

	handles, err := service.ListRuntimeHandlesCompact(&agent)
	if err != nil {
		t.Fatalf("ListRuntimeHandlesCompact: %v", err)
	}
	if len(handles) != 1 || handles[0] == nil || handles[0].ID != 21 {
		t.Fatalf("handles=%+v", handles)
	}
	if store.passiveHandlesFilter == nil || *store.passiveHandlesFilter != agent {
		t.Fatalf("passiveHandlesFilter=%v", store.passiveHandlesFilter)
	}
	if store.syncHandleCalls != 0 {
		t.Fatalf("syncHandleCalls=%d", store.syncHandleCalls)
	}
}

func TestListRuntimeHandlesDescartaHandleSincronizadoANil(t *testing.T) {
	agent := "Codex2"
	store := &fakeStore{
		handlesResponse:   []*db.RuntimeHandle{{ID: 31, Agente: agent, Estado: "activo"}},
		syncHandleRespSet: true,
		syncHandleResp:    nil,
	}
	service := NewService(store)

	handles, err := service.ListRuntimeHandles(&agent)
	if err != nil {
		t.Fatalf("ListRuntimeHandles: %v", err)
	}
	if len(handles) != 0 {
		t.Fatalf("len(handles)=%d", len(handles))
	}
	if store.syncHandleCalls != 1 || store.syncHandleInput == nil || store.syncHandleInput.ID != 31 {
		t.Fatalf("sync inesperado calls=%d input=%+v", store.syncHandleCalls, store.syncHandleInput)
	}
}

func TestRuntimeHandleShouldSupersedeStartNoDevuelveTrueConWorkerEstructuradoStarting(t *testing.T) {
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "manifest.json")
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")

	write := func(path string, payload any) {
		t.Helper()
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	now := time.Now().UTC()
	write(manifestPath, map[string]any{
		"driver":       "tmux_cli_session",
		"transport":    "tmux",
		"tmux_session": "orq-codex1-101500",
		"tmux_pane_id": "%9",
	})
	write(statusPath, map[string]any{
		"state":      "starting",
		"alive":      true,
		"updated_at": now.Format(time.RFC3339),
	})
	write(heartbeatPath, map[string]any{
		"alive":            true,
		"heartbeat_at":     now.Format(time.RFC3339),
		"last_progress_at": now.Format(time.RFC3339),
	})

	meta, err := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_session":          "orq-codex1-101500",
		"tmux_pane_id":          "%9",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}

	handle := &db.RuntimeHandle{
		Estado:       "activo",
		MetadataJSON: string(meta),
	}
	runtime := &db.RuntimeInstance{
		LogicalState: "disponible",
		ProcessState: "running",
	}

	if got := runtimeHandleShouldSupersedeStart(handle, runtime); got {
		t.Fatalf("runtimeHandleShouldSupersedeStart() = true, want false with structured worker starting")
	}
}

func TestRuntimeHandleShouldSupersedeStartDevuelveTrueConWorkerCanonicoBlockedRuntime(t *testing.T) {
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "manifest.json")
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")

	write := func(path string, payload any) {
		t.Helper()
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	now := time.Now().UTC()
	write(manifestPath, map[string]any{
		"driver":       "tmux_cli_session",
		"transport":    "tmux",
		"tmux_session": "orq-codex1-101500",
		"tmux_pane_id": "%10",
	})
	write(statusPath, map[string]any{
		"state":      "blocked_runtime",
		"alive":      true,
		"updated_at": now.Format(time.RFC3339),
	})
	write(heartbeatPath, map[string]any{
		"alive":            true,
		"heartbeat_at":     now.Format(time.RFC3339),
		"last_progress_at": now.Format(time.RFC3339),
	})

	meta, err := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_session":          "orq-codex1-101500",
		"tmux_pane_id":          "%10",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}

	handle := &db.RuntimeHandle{
		Estado:       "activo",
		MetadataJSON: string(meta),
	}
	runtime := &db.RuntimeInstance{
		LogicalState: "disponible",
		ProcessState: "running",
	}

	if got := runtimeHandleShouldSupersedeStart(handle, runtime); !got {
		t.Fatalf("runtimeHandleShouldSupersedeStart() = false, want true with canonical blocked_runtime")
	}
}

func TestRuntimeHandleCanonicoParaControlDevuelveFalseConWorkerCanonicoBlockedRuntime(t *testing.T) {
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "manifest.json")
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")

	write := func(path string, payload any) {
		t.Helper()
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	now := time.Now().UTC()
	write(manifestPath, map[string]any{
		"driver":       "tmux_cli_session",
		"transport":    "tmux",
		"tmux_session": "orq-codex1-101501",
		"tmux_pane_id": "%11",
	})
	write(statusPath, map[string]any{
		"state":      "blocked_runtime",
		"alive":      true,
		"updated_at": now.Format(time.RFC3339),
	})
	write(heartbeatPath, map[string]any{
		"alive":            true,
		"heartbeat_at":     now.Format(time.RFC3339),
		"last_progress_at": now.Format(time.RFC3339),
	})

	meta, err := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_session":          "orq-codex1-101501",
		"tmux_pane_id":          "%11",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}

	handle := &db.RuntimeHandle{
		Estado:       "activo",
		MetadataJSON: string(meta),
	}

	if got := runtimeHandleCanonicoParaControl(handle); got {
		t.Fatalf("runtimeHandleCanonicoParaControl() = true, want false with canonical blocked_runtime")
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

func TestMarcarRuntimeOrderDispatchNotificadoPersisteContratoCanonico(t *testing.T) {
	store := &fakeStore{
		orderID:        41,
		ordersResponse: nil,
	}
	store.orderID = 41
	store.ordersResponse = []*db.RuntimeOrder{{ID: 41, Agente: "Gemma1", Tipo: "send_instruction", ResultadoJSON: `{"delivery_state":"queued"}`}}
	service := NewService(store)

	if err := service.MarcarRuntimeOrderDispatchNotificado(41, "worker notificado"); err != nil {
		t.Fatalf("MarcarRuntimeOrderDispatchNotificado: %v", err)
	}
	if store.markOrderStateID != 41 || store.markOrderStateEstado != "pendiente" {
		t.Fatalf("marca inesperada: id=%d estado=%q", store.markOrderStateID, store.markOrderStateEstado)
	}
	for _, token := range []string{
		`"dispatch_state":"notified"`,
		`"delivery_state":"notified"`,
		`"deferred_reason":"worker notificado"`,
	} {
		if !strings.Contains(store.markOrderStateResultado, token) {
			t.Fatalf("resultado sin token %q: %s", token, store.markOrderStateResultado)
		}
	}
}

func TestMarcarRuntimeOrderDispatchFallidoPersisteContratoCanonico(t *testing.T) {
	store := &fakeStore{}
	store.orderID = 42
	store.ordersResponse = []*db.RuntimeOrder{{ID: 42, Agente: "Gemma1", Tipo: "send_instruction", ResultadoJSON: `{"delivery_state":"queued"}`}}
	service := NewService(store)

	if err := service.MarcarRuntimeOrderDispatchFallido(42, "pane no listo"); err != nil {
		t.Fatalf("MarcarRuntimeOrderDispatchFallido: %v", err)
	}
	if store.markOrderStateID != 42 || store.markOrderStateEstado != "fallida" {
		t.Fatalf("marca inesperada: id=%d estado=%q", store.markOrderStateID, store.markOrderStateEstado)
	}
	for _, token := range []string{
		`"dispatch_state":"failed"`,
		`"delivery_state":"failed"`,
		`"last_reason":"pane no listo"`,
	} {
		if !strings.Contains(store.markOrderStateResultado, token) {
			t.Fatalf("resultado sin token %q: %s", token, store.markOrderStateResultado)
		}
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
		FormatoSalida:     "ficheros+evidencia",
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

func TestEnqueueAgentNudgeEncolaOrdenCanonica(t *testing.T) {
	runtimeID := int64(21)
	projectID := int64(9)
	store := &fakeStore{
		projectResponse:     &db.Proyecto{ID: projectID, Slug: "orquestador", Nombre: "Orquestador", RutaAbs: "/tmp/orquestador"},
		operationalProjResp: &db.RuntimeHandle{ID: 14, Agente: "Codex1", Estado: "activo", RuntimeID: &runtimeID},
		runtimeResponse:     &db.RuntimeInstance{ID: runtimeID, Agente: "Codex1", ProyectoID: &projectID},
		handleByIDResp:      &db.RuntimeHandle{ID: 14, Agente: "Codex1", Estado: "activo", RuntimeID: &runtimeID},
		createOrderID:       77,
	}
	service := NewService(store)

	resultado, err := service.EnqueueAgentNudge(AgentNudgeRequest{
		Agente:      "Codex1",
		Proyecto:    "orquestador",
		Accion:      "implementar",
		Motivo:      "seguir tarea",
		Instruction: "TAREA: #31 corregir parser",
		Actor:       "orquesta",
		Metadata: map[string]any{
			"source": "pipeline_local",
		},
	})
	if err != nil {
		t.Fatalf("EnqueueAgentNudge: %v", err)
	}
	if resultado == nil || resultado.RuntimeOrderID <= 0 {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
	if store.createOrder == nil || store.createOrder.Tipo != "nudge" {
		t.Fatalf("orden encolada inesperada: %+v", store.createOrder)
	}
	if !strings.Contains(store.createOrder.PayloadJSON, `"accion":"implementar"`) || !strings.Contains(store.createOrder.PayloadJSON, `"source":"pipeline_local"`) {
		t.Fatalf("payload nudge inesperado: %s", store.createOrder.PayloadJSON)
	}
	if store.createOrder.HandleID == nil || *store.createOrder.HandleID != 14 {
		t.Fatalf("handle no propagado: %+v", store.createOrder)
	}
}

func TestDispatchMicroprogramacionInstructionIncluyeWorktreeActivaEnPayload(t *testing.T) {
	projectID := int64(10)
	store := &fakeStore{
		createOrderID:   89,
		projectResponse: &db.Proyecto{ID: projectID, Slug: "orquestador", RutaAbs: "/tmp/orquestador"},
	}
	resolvedor := &fakeResolvedorWorktree{
		resultado: &db.Worktree{
			ID:           71,
			ProyectoSlug: "orquestador",
			RutaAbs:      "/tmp/orquestador/.orquesta-worktrees/orq-codex1",
			Branch:       "orq/orquestador/codex1",
			BaseRef:      "main",
		},
	}
	service := NewService(store)
	service.SetResolvedorWorktreeActiva(resolvedor)

	_, err := service.DispatchMicroprogramacionInstruction(MicroprogramacionDispatchRequest{
		AgenteDestino:     "Codex1",
		ProyectoID:        &projectID,
		Mensaje:           "MICROTAREA CERRADA\nARCHIVO: microprogramacionapp/service.go",
		EspecificacionID:  32,
		ArchivoObjetivo:   "microprogramacionapp/service.go",
		SimboloObjetivo:   "RegistrarEntregaGit",
		WriteSet:          []string{"microprogramacionapp/service.go"},
		TestsObligatorios: []string{"go test ./microprogramacionapp -count=1"},
		FormatoSalida:     "git_worktree+evidencia",
	})
	if err != nil {
		t.Fatalf("DispatchMicroprogramacionInstruction: %v", err)
	}
	if resolvedor.proyectoSlug != "orquestador" || resolvedor.agente != "Codex1" {
		t.Fatalf("resolvedor worktree inesperado: proyecto=%q agente=%q", resolvedor.proyectoSlug, resolvedor.agente)
	}
	if !strings.Contains(store.createOrder.PayloadJSON, `"worktree_id":71`) {
		t.Fatalf("payload sin worktree_id: %s", store.createOrder.PayloadJSON)
	}
	if !strings.Contains(store.createOrder.PayloadJSON, `"ruta_worktree":"/tmp/orquestador/.orquesta-worktrees/orq-codex1"`) {
		t.Fatalf("payload sin ruta_worktree: %s", store.createOrder.PayloadJSON)
	}
	if !strings.Contains(store.createOrder.PayloadJSON, `"branch_worktree":"orq/orquestador/codex1"`) {
		t.Fatalf("payload sin branch_worktree: %s", store.createOrder.PayloadJSON)
	}
	if !strings.Contains(store.createOrder.PayloadJSON, `"base_ref_worktree":"main"`) {
		t.Fatalf("payload sin base_ref_worktree: %s", store.createOrder.PayloadJSON)
	}
}

func TestDispatchMicroprogramacionInstructionAseguraWorktreeGitSiNoExisteActiva(t *testing.T) {
	projectID := int64(11)
	store := &fakeStore{
		createOrderID:   90,
		projectResponse: &db.Proyecto{ID: projectID, Slug: "orquestador", RutaAbs: "/tmp/orquestador"},
	}
	resolvedor := &fakeResolvedorWorktree{err: fmt.Errorf("no existe worktree activa")}
	asegurador := &fakeAseguradorWorktree{
		resultado: &db.Worktree{
			ID:           72,
			ProyectoSlug: "orquestador",
			RutaAbs:      "/tmp/orquestador/.orquesta-worktrees/orq-codex1",
			Branch:       "orq/orquestador/codex1",
			BaseRef:      "HEAD",
		},
	}
	service := NewService(store)
	service.SetResolvedorWorktreeActiva(resolvedor)
	service.SetAseguradorWorktreeActiva(asegurador)

	_, err := service.DispatchMicroprogramacionInstruction(MicroprogramacionDispatchRequest{
		AgenteDestino:     "Codex1",
		ProyectoID:        &projectID,
		Mensaje:           "MICROTAREA CERRADA\nARCHIVO: cabeceras/merge_headers.go",
		EspecificacionID:  33,
		ArchivoObjetivo:   "cabeceras/merge_headers.go",
		SimboloObjetivo:   "FusionarCabecerasCanonicas",
		WriteSet:          []string{"cabeceras/merge_headers.go"},
		TestsObligatorios: []string{"go test ./cabeceras -run TestFusionarCabecerasCanonicas -count=1"},
		FormatoSalida:     "git_worktree+evidencia",
	})
	if err != nil {
		t.Fatalf("DispatchMicroprogramacionInstruction: %v", err)
	}
	if resolvedor.proyectoSlug != "orquestador" || asegurador.proyectoSlug != "orquestador" {
		t.Fatalf("proyecto inesperado en resolvedor/asegurador: %q / %q", resolvedor.proyectoSlug, asegurador.proyectoSlug)
	}
	if asegurador.agente != "Codex1" {
		t.Fatalf("asegurador sin agente esperado: %q", asegurador.agente)
	}
	if !strings.Contains(store.createOrder.PayloadJSON, `"worktree_id":72`) || !strings.Contains(store.createOrder.PayloadJSON, `"branch_worktree":"orq/orquestador/codex1"`) {
		t.Fatalf("payload sin worktree asegurada: %s", store.createOrder.PayloadJSON)
	}
}

func TestDispatchMicroprogramacionInstructionOllamaGitSeDegradaAFicheros(t *testing.T) {
	projectID := int64(12)
	proyectoDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(proyectoDir, "cabeceras"), 0o755); err != nil {
		t.Fatalf("mkdir cabeceras: %v", err)
	}
	if err := os.WriteFile(filepath.Join(proyectoDir, "cabeceras", "merge_headers.go"), []byte("package cabeceras\n\nfunc FusionarCabecerasCanonicas(izquierda, derecha []string) []string {\n\treturn nil\n}\n"), 0o644); err != nil {
		t.Fatalf("write merge_headers.go: %v", err)
	}
	store := &fakeStore{
		createOrderID:   94,
		projectResponse: &db.Proyecto{ID: projectID, Slug: "proyecto", RutaAbs: proyectoDir},
	}
	service := NewService(store)

	resultado, err := service.DispatchMicroprogramacionInstruction(MicroprogramacionDispatchRequest{
		AgenteDestino:     "Gemma1",
		ProyectoID:        &projectID,
		Mensaje:           "MICROTAREA CERRADA\nARCHIVO: cabeceras/merge_headers.go",
		EspecificacionID:  47,
		ArchivoObjetivo:   "cabeceras/merge_headers.go",
		SimboloObjetivo:   "FusionarCabecerasCanonicas",
		WriteSet:          []string{"cabeceras/merge_headers.go"},
		TestsObligatorios: []string{"go test ./cabeceras -run TestFusionarCabecerasCanonicas -count=1"},
		FormatoSalida:     "git_worktree+evidencia",
	})
	if err != nil {
		t.Fatalf("DispatchMicroprogramacionInstruction: %v", err)
	}
	if resultado == nil || resultado.RuntimeOrderID != 94 {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
	if store.createOrder == nil {
		t.Fatalf("runtime order no creada")
	}
	if !strings.Contains(store.createOrder.PayloadJSON, `"formato_salida":"ficheros+evidencia"`) {
		t.Fatalf("payload sin degradacion a ficheros+evidencia: %s", store.createOrder.PayloadJSON)
	}
	if !strings.Contains(store.createOrder.PayloadJSON, "FICHEROS: devuelve uno o varios bloques") {
		t.Fatalf("payload degradado sin contrato de ficheros: %s", store.createOrder.PayloadJSON)
	}
	if strings.Contains(store.createOrder.PayloadJSON, "ENTREGA_GIT:") {
		t.Fatalf("payload degradado no deberia seguir pidiendo entrega git: %s", store.createOrder.PayloadJSON)
	}
	if strings.Contains(store.createOrder.PayloadJSON, "git_worktree+evidencia") {
		t.Fatalf("payload degradado no deberia seguir mencionando git_worktree: %s", store.createOrder.PayloadJSON)
	}
}

func TestDispatchMicroprogramacionInstructionOllamaPatchSeDegradaAFicheros(t *testing.T) {
	projectID := int64(120)
	proyectoDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(proyectoDir, "capacidadapp"), 0o755); err != nil {
		t.Fatalf("mkdir capacidadapp: %v", err)
	}
	if err := os.WriteFile(filepath.Join(proyectoDir, "capacidadapp", "pool_local_compartido.go"), []byte("package capacidadapp\n"), 0o644); err != nil {
		t.Fatalf("write pool_local_compartido.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(proyectoDir, "capacidadapp", "service_test.go"), []byte("package capacidadapp\n"), 0o644); err != nil {
		t.Fatalf("write service_test.go: %v", err)
	}
	store := &fakeStore{
		createOrderID:   121,
		projectResponse: &db.Proyecto{ID: projectID, Slug: "proyecto", RutaAbs: proyectoDir},
	}
	service := NewService(store)

	resultado, err := service.DispatchMicroprogramacionInstruction(MicroprogramacionDispatchRequest{
		AgenteDestino:     "Gemma1",
		ProyectoID:        &projectID,
		Mensaje:           "MICROTAREA CERRADA\nARCHIVO: capacidadapp/pool_local_compartido.go\nSALIDA: patch+evidencia",
		EspecificacionID:  49,
		ArchivoObjetivo:   "capacidadapp/pool_local_compartido.go",
		SimboloObjetivo:   "DescribirPoolLocalCompartido",
		WriteSet:          []string{"capacidadapp/pool_local_compartido.go", "capacidadapp/service_test.go"},
		TestsObligatorios: []string{"go test ./capacidadapp -run TestDescribirPoolLocalCompartidoLeeMetadataYPoliticas -count=1"},
		FormatoSalida:     "patch+evidencia",
	})
	if err != nil {
		t.Fatalf("DispatchMicroprogramacionInstruction: %v", err)
	}
	if resultado == nil || resultado.RuntimeOrderID != 121 {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
	if !strings.Contains(store.createOrder.PayloadJSON, `"formato_salida":"ficheros+evidencia"`) {
		t.Fatalf("payload sin degradacion a ficheros+evidencia: %s", store.createOrder.PayloadJSON)
	}
	if strings.Contains(store.createOrder.PayloadJSON, "PATCH_UNIFICADO:") {
		t.Fatalf("payload degradado no deberia seguir pidiendo patch unificado: %s", store.createOrder.PayloadJSON)
	}
	if !strings.Contains(store.createOrder.PayloadJSON, "FICHEROS: devuelve uno o varios bloques") {
		t.Fatalf("payload degradado sin contrato de ficheros: %s", store.createOrder.PayloadJSON)
	}
}

func TestDispatchMicroprogramacionInstructionOllamaPoolLocalConservaGit(t *testing.T) {
	projectID := int64(13)
	proyectoDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(proyectoDir, "pkg", "hola"), 0o755); err != nil {
		t.Fatalf("mkdir pkg/hola: %v", err)
	}
	if err := os.WriteFile(filepath.Join(proyectoDir, "pkg", "hola", "hola.go"), []byte("package hola\n\nfunc Hola() string { return \"hola\" }\n"), 0o644); err != nil {
		t.Fatalf("write hola.go: %v", err)
	}
	store := &fakeStore{
		createOrderID:          95,
		projectResponse:        &db.Proyecto{ID: projectID, Slug: "proyecto", RutaAbs: proyectoDir},
		primaryProjRuntimeID:   &projectID,
		primaryProjRuntimeResp: &db.RuntimeInstance{Agente: "Gemma1", ProyectoID: &projectID, Connector: "ollama_pool_local"},
	}
	service := NewService(store)

	resultado, err := service.DispatchMicroprogramacionInstruction(MicroprogramacionDispatchRequest{
		AgenteDestino:     "Gemma1",
		ProyectoID:        &projectID,
		Mensaje:           "MICROTAREA CERRADA\nARCHIVO: pkg/hola/hola.go\nSALIDA: git_worktree+evidencia\nENTREGA_GIT: usa tu worktree",
		EspecificacionID:  48,
		ArchivoObjetivo:   "pkg/hola/hola.go",
		SimboloObjetivo:   "Hola",
		WriteSet:          []string{"pkg/hola/hola.go"},
		TestsObligatorios: []string{"go test ./pkg/hola -count=1"},
		FormatoSalida:     "git_worktree+evidencia",
	})
	if err != nil {
		t.Fatalf("DispatchMicroprogramacionInstruction: %v", err)
	}
	if resultado == nil || resultado.RuntimeOrderID != 95 {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
	if !strings.Contains(store.createOrder.PayloadJSON, `"formato_salida":"git_worktree+evidencia"`) {
		t.Fatalf("payload de pool local no deberia degradarse: %s", store.createOrder.PayloadJSON)
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
		FormatoSalida:     "ficheros+evidencia",
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
	if !strings.Contains(store.createOrder.PayloadJSON, "FICHEROS: devuelve uno o varios bloques") {
		t.Fatalf("payload sin contrato de ficheros inline: %s", store.createOrder.PayloadJSON)
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
		FormatoSalida:     "ficheros+evidencia",
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
	if !strings.Contains(store.createOrder.PayloadJSON, "FICHEROS: devuelve uno o varios bloques") {
		t.Fatalf("payload sin contrato de ficheros inline: %s", store.createOrder.PayloadJSON)
	}
	if !strings.Contains(store.createOrder.PayloadJSON, "identidad/nuevo.go") || !strings.Contains(store.createOrder.PayloadJSON, "identidad/nuevo_test.go") {
		t.Fatalf("payload sin write_set esperado: %s", store.createOrder.PayloadJSON)
	}
}

func TestDispatchMicroprogramacionInstructionOllamaDetectaFormatoFileBlocks(t *testing.T) {
	projectID := int64(13)
	proyectoDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(proyectoDir, "cabeceras"), 0o755); err != nil {
		t.Fatalf("mkdir cabeceras: %v", err)
	}
	if err := os.WriteFile(filepath.Join(proyectoDir, "cabeceras", "merge_headers.go"), []byte("package cabeceras\n\nfunc FusionarCabecerasCanonicas(izquierda, derecha []string) []string {\n\treturn nil\n}\n"), 0o644); err != nil {
		t.Fatalf("write merge_headers.go: %v", err)
	}
	store := &fakeStore{
		createOrderID:   94,
		projectResponse: &db.Proyecto{ID: projectID, Slug: "proyecto", RutaAbs: proyectoDir},
	}
	service := NewService(store)

	_, err := service.DispatchMicroprogramacionInstruction(MicroprogramacionDispatchRequest{
		AgenteDestino:     "Gemma1",
		ProyectoID:        &projectID,
		Mensaje:           "MICROTAREA CERRADA\nARCHIVO: cabeceras/merge_headers.go",
		EspecificacionID:  47,
		ArchivoObjetivo:   "cabeceras/merge_headers.go",
		SimboloObjetivo:   "FusionarCabecerasCanonicas",
		WriteSet:          []string{"cabeceras/merge_headers.go"},
		TestsObligatorios: []string{"go test ./cabeceras -run TestFusionarCabecerasCanonicas -count=1"},
		FormatoSalida:     "Responde solo con bloques // FILE: ruta seguidos del contenido completo del fichero.",
	})
	if err != nil {
		t.Fatalf("DispatchMicroprogramacionInstruction: %v", err)
	}
	if store.createOrder == nil {
		t.Fatalf("runtime order no creada")
	}
	if !strings.Contains(store.createOrder.PayloadJSON, "FICHEROS: devuelve uno o varios bloques") {
		t.Fatalf("payload sin contrato de ficheros inline: %s", store.createOrder.PayloadJSON)
	}
	if strings.Contains(store.createOrder.PayloadJSON, "EVIDENCIA:") {
		t.Fatalf("payload no deberia pedir evidencia para bloques FILE: %s", store.createOrder.PayloadJSON)
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

func TestResolverContextoEntregaMicroprogramacionLeePayloadCanonico(t *testing.T) {
	projectID := int64(11)
	agent := "Gemma1"
	store := &fakeStore{
		ordersResponse: []*db.RuntimeOrder{
			{
				ID:          101,
				Agente:      agent,
				ProyectoID:  &projectID,
				Tipo:        "send_instruction",
				PayloadJSON: `{"source":"microprogramacion","microprogramacion":{"especificacion_id":44,"archivo_objetivo":"microprogramacionapp/service.go","simbolo_objetivo":"ResolveControlHandle","write_set":["microprogramacionapp/service.go","microprogramacionapp/service_test.go"]}}`,
			},
		},
	}
	service := NewService(store)

	ctx, err := service.ResolverContextoEntregaMicroprogramacion(agent, &projectID)
	if err != nil {
		t.Fatalf("ResolverContextoEntregaMicroprogramacion: %v", err)
	}
	if ctx == nil || ctx.EspecificacionID != 44 || ctx.RuntimeOrderID != 101 {
		t.Fatalf("contexto inesperado: %+v", ctx)
	}
	if len(ctx.WriteSet) != 2 || ctx.ArchivoObjetivo != "microprogramacionapp/service.go" {
		t.Fatalf("contexto incompleto: %+v", ctx)
	}
}

func TestResolverContextoEntregaMicroprogramacionParaRespuestaPrefiereWriteSetCoincidente(t *testing.T) {
	projectID := int64(11)
	agent := "Gemma1"
	store := &fakeStore{
		ordersResponse: []*db.RuntimeOrder{
			{
				ID:          101,
				Agente:      agent,
				ProyectoID:  &projectID,
				Tipo:        "send_instruction",
				PayloadJSON: `{"source":"microprogramacion","microprogramacion":{"especificacion_id":1,"archivo_objetivo":"tmp/ollama_smoke_eval/normalizar_identificador.go","simbolo_objetivo":"NormalizarIdentificadorTecnico","write_set":["tmp/ollama_smoke_eval/normalizar_identificador.go"]}}`,
			},
			{
				ID:          102,
				Agente:      agent,
				ProyectoID:  &projectID,
				Tipo:        "send_instruction",
				PayloadJSON: `{"source":"microprogramacion","microprogramacion":{"especificacion_id":14,"archivo_objetivo":"cabeceras/merge_headers.go","simbolo_objetivo":"FusionarCabecerasCanonicas","write_set":["cabeceras/merge_headers.go"]}}`,
			},
		},
	}
	service := NewService(store)

	ctx, err := service.ResolverContextoEntregaMicroprogramacionParaRespuesta(agent, &projectID, `// FILE: cabeceras/merge_headers.go
package cabeceras
`)
	if err != nil {
		t.Fatalf("ResolverContextoEntregaMicroprogramacionParaRespuesta: %v", err)
	}
	if ctx == nil || ctx.RuntimeOrderID != 102 || ctx.EspecificacionID != 14 {
		t.Fatalf("contexto inesperado: %+v", ctx)
	}
}

func TestCompletarEntregaMicroprogramacionCierraOrdenYConsumeMailbox(t *testing.T) {
	projectID := int64(11)
	store := &fakeStore{
		orderID:        201,
		createOrderID:  0,
		handleByIDResp: nil,
	}
	store.ordersResponse = []*db.RuntimeOrder{
		{
			ID:            201,
			Agente:        "Gemma1",
			ProyectoID:    &projectID,
			Tipo:          "send_instruction",
			PayloadJSON:   `{"source":"microprogramacion","mailbox_id":77,"microprogramacion":{"especificacion_id":44}}`,
			ResultadoJSON: `{"delivery_state":"queued"}`,
		},
	}
	service := NewService(store)

	err := service.CompletarEntregaMicroprogramacion(201, "ollama_pool_local_materializada")
	if err != nil {
		t.Fatalf("CompletarEntregaMicroprogramacion: %v", err)
	}
	if store.consumedID != 77 {
		t.Fatalf("mailbox consumida inesperada: %d", store.consumedID)
	}
	if store.markOrderStateID != 201 || store.markOrderStateEstado != "completada" {
		t.Fatalf("mark order inesperado: id=%d estado=%q", store.markOrderStateID, store.markOrderStateEstado)
	}
	if !strings.Contains(store.markOrderStateResultado, `"receipt_source":"ollama_pool_local_materializada"`) {
		t.Fatalf("resultado sin receipt_source esperado: %s", store.markOrderStateResultado)
	}
}

func TestCompletarEntregasMicroprogramacionParaRespuestaCierraReintentosCoincidentes(t *testing.T) {
	projectID := int64(11)
	store := &fakeStore{
		orderID:        202,
		createOrderID:  0,
		handleByIDResp: nil,
	}
	store.ordersResponse = []*db.RuntimeOrder{
		{
			ID:            201,
			Agente:        "Gemma1",
			ProyectoID:    &projectID,
			Tipo:          "send_instruction",
			PayloadJSON:   `{"source":"microprogramacion","mailbox_id":77,"microprogramacion":{"especificacion_id":44,"archivo_objetivo":"cabeceras/merge_headers.go","simbolo_objetivo":"FusionarCabecerasCanonicas","write_set":["cabeceras/merge_headers.go"]}}`,
			ResultadoJSON: `{"delivery_state":"queued"}`,
		},
		{
			ID:            202,
			Agente:        "Gemma1",
			ProyectoID:    &projectID,
			Tipo:          "send_instruction",
			PayloadJSON:   `{"source":"microprogramacion","mailbox_id":78,"microprogramacion":{"especificacion_id":44,"archivo_objetivo":"cabeceras/merge_headers.go","simbolo_objetivo":"FusionarCabecerasCanonicas","write_set":["cabeceras/merge_headers.go"]}}`,
			ResultadoJSON: `{"delivery_state":"queued"}`,
		},
	}
	service := NewService(store)

	n, err := service.CompletarEntregasMicroprogramacionParaRespuesta("Gemma1", &projectID, `// FILE: cabeceras/merge_headers.go
package cabeceras
`, "ollama_pool_local_materializada")
	if err != nil {
		t.Fatalf("CompletarEntregasMicroprogramacionParaRespuesta: %v", err)
	}
	if n != 2 {
		t.Fatalf("se esperaban 2 ordenes completadas, got=%d", n)
	}
	if len(store.markedOrderStates) != 2 {
		t.Fatalf("marcas de orden inesperadas: %+v", store.markedOrderStates)
	}
}

func TestMaterializarEntregaMicroprogramacionActivaDesdeRespuestaUsaMaterializadorYCompletaOrdenes(t *testing.T) {
	projectID := int64(11)
	store := &fakeStore{
		projectResponse: &db.Proyecto{ID: projectID, Slug: "orquestador", RutaAbs: "/tmp/orquestador"},
		ordersResponse: []*db.RuntimeOrder{
			{
				ID:            211,
				Agente:        "Gemma1",
				ProyectoID:    &projectID,
				Tipo:          "send_instruction",
				PayloadJSON:   `{"source":"microprogramacion","mailbox_id":77,"microprogramacion":{"especificacion_id":44,"archivo_objetivo":"microprogramacionapp/extractor_entrega.go","simbolo_objetivo":"recortarSeccionEvidencia","write_set":["microprogramacionapp/extractor_entrega.go","microprogramacionapp/extractor_entrega_test.go"],"formato_salida":"ficheros+evidencia"}}`,
				ResultadoJSON: `{"delivery_state":"queued"}`,
			},
		},
	}
	materializador := &fakeMaterializadorEntrega{}
	service := NewService(store)
	service.SetMaterializadorEntregaMicroprogramacion(materializador)

	resultado, err := service.MaterializarEntregaMicroprogramacionActivaDesdeRespuesta(
		"Gemma1",
		&projectID,
		"orquestador",
		"// FILE: microprogramacionapp/extractor_entrega.go\npackage microprogramacionapp\n",
		"runtime_transcript_id=23",
		"ollama_pool_local_materializada",
	)
	if err != nil {
		t.Fatalf("MaterializarEntregaMicroprogramacionActivaDesdeRespuesta: %v", err)
	}
	if resultado == nil {
		t.Fatalf("resultado nulo")
	}
	if materializador.id != 44 || materializador.raiz != "/tmp/orquestador" {
		t.Fatalf("materializador inesperado: id=%d raiz=%q", materializador.id, materializador.raiz)
	}
	if store.consumedID != 77 || store.markOrderStateID != 211 || store.markOrderStateEstado != "completada" {
		t.Fatalf("orden no completada correctamente: consumed=%d id=%d estado=%q", store.consumedID, store.markOrderStateID, store.markOrderStateEstado)
	}
	if !strings.Contains(store.markOrderStateResultado, `"receipt_source":"ollama_pool_local_materializada"`) {
		t.Fatalf("resultado sin receipt esperado: %s", store.markOrderStateResultado)
	}
}

func TestMaterializarEntregaMicroprogramacionActivaDesdeRespuestaIgnoraFormatoGit(t *testing.T) {
	projectID := int64(11)
	store := &fakeStore{
		projectResponse: &db.Proyecto{ID: projectID, Slug: "orquestador", RutaAbs: "/tmp/orquestador"},
		ordersResponse: []*db.RuntimeOrder{
			{
				ID:          212,
				Agente:      "Gemma1",
				ProyectoID:  &projectID,
				Tipo:        "send_instruction",
				PayloadJSON: `{"source":"microprogramacion","microprogramacion":{"especificacion_id":45,"archivo_objetivo":"microprogramacionapp/service.go","simbolo_objetivo":"RegistrarEntregaGit","write_set":["microprogramacionapp/service.go"],"formato_salida":"git_worktree+evidencia"}}`,
			},
		},
	}
	materializador := &fakeMaterializadorEntrega{}
	service := NewService(store)
	service.SetMaterializadorEntregaMicroprogramacion(materializador)

	resultado, err := service.MaterializarEntregaMicroprogramacionActivaDesdeRespuesta(
		"Gemma1",
		&projectID,
		"orquestador",
		"// FILE: microprogramacionapp/service.go\npackage microprogramacionapp\n",
		"runtime_transcript_id=24",
		"ollama_pool_local_materializada",
	)
	if err != nil {
		t.Fatalf("MaterializarEntregaMicroprogramacionActivaDesdeRespuesta: %v", err)
	}
	if resultado != nil {
		t.Fatalf("no deberia materializar formato git: %+v", resultado)
	}
	if materializador.id != 0 || store.markOrderStateID != 0 {
		t.Fatalf("no deberia tocar materializador ni orden: materializador=%d order=%d", materializador.id, store.markOrderStateID)
	}
}

func TestReencolarCorreccionEntregaMicroprogramacionSupersedeYCreaNuevaOrden(t *testing.T) {
	projectID := int64(11)
	store := &fakeStore{
		createOrderID: 901,
		ordersResponse: []*db.RuntimeOrder{
			{
				ID:          450,
				Agente:      "Gemma1",
				ProyectoID:  &projectID,
				Tipo:        "send_instruction",
				Estado:      "pendiente",
				PayloadJSON: `{"to_agente":"Gemma1","texto":"MICROTAREA_BASE","source":"microprogramacion","mailbox_id":77,"microprogramacion":{"especificacion_id":44,"archivo_objetivo":"microprogramacionapp/extractor_entrega.go","simbolo_objetivo":"recortarSeccionEvidencia","write_set":["microprogramacionapp/extractor_entrega.go","microprogramacionapp/extractor_entrega_test.go"],"formato_salida":"ficheros+evidencia"}}`,
			},
		},
		orderID: 450,
	}
	service := NewService(store)

	orderID, err := service.ReencolarCorreccionEntregaMicroprogramacion(450, "go invalido: expected declaration")
	if err != nil {
		t.Fatalf("ReencolarCorreccionEntregaMicroprogramacion: %v", err)
	}
	if orderID != 901 {
		t.Fatalf("orderID inesperado: got=%d want=901", orderID)
	}
	if store.markOrderStateID != 450 || store.markOrderStateEstado != "cancelada" {
		t.Fatalf("orden previa no supersedida: id=%d estado=%q", store.markOrderStateID, store.markOrderStateEstado)
	}
	if store.consumedID != 77 {
		t.Fatalf("mailbox previa no consumida: %d", store.consumedID)
	}
	if store.createOrder == nil || strings.TrimSpace(store.createOrder.Agente) != "Gemma1" {
		t.Fatalf("orden nueva no creada correctamente: %+v", store.createOrder)
	}
	if !strings.Contains(store.createOrder.PayloadJSON, "CORRECCION_OBLIGATORIA") {
		t.Fatalf("payload sin correccion obligatoria: %s", store.createOrder.PayloadJSON)
	}
	if !strings.Contains(store.createOrder.PayloadJSON, "go invalido: expected declaration") {
		t.Fatalf("payload sin error exacto: %s", store.createOrder.PayloadJSON)
	}
	if !strings.Contains(store.createOrder.PayloadJSON, `"correccion_intento":1`) {
		t.Fatalf("payload sin correccion_intento: %s", store.createOrder.PayloadJSON)
	}
	if strings.Contains(store.createOrder.PayloadJSON, `"mailbox_id":77`) {
		t.Fatalf("la correccion no deberia reutilizar mailbox consumida: %s", store.createOrder.PayloadJSON)
	}
}

func TestRegistrarEntregaGitMicroprogramacionActivaUsaRegistradorYCompletaOrden(t *testing.T) {
	projectID := int64(11)
	store := &fakeStore{
		ordersResponse: []*db.RuntimeOrder{
			{
				ID:            301,
				Agente:        "Gemma1",
				ProyectoID:    &projectID,
				Tipo:          "send_instruction",
				PayloadJSON:   `{"source":"microprogramacion","mailbox_id":77,"microprogramacion":{"especificacion_id":44,"archivo_objetivo":"microprogramacionapp/service.go","simbolo_objetivo":"RegistrarEntregaGit","write_set":["microprogramacionapp/service.go"],"formato_salida":"git_worktree+evidencia"}}`,
				ResultadoJSON: `{"delivery_state":"queued"}`,
			},
		},
	}
	registrador := &fakeRegistradorEntregaGit{}
	service := NewService(store)
	service.SetRegistradorEntregaGit(registrador)

	resultado, err := service.RegistrarEntregaGitMicroprogramacionActiva("Gemma1", &projectID, "orquestador", "diff listo", "OpenClaw")
	if err != nil {
		t.Fatalf("RegistrarEntregaGitMicroprogramacionActiva: %v", err)
	}
	if resultado == nil || resultado.Contexto == nil || resultado.Contexto.RuntimeOrderID != 301 {
		t.Fatalf("resultado/contexto inesperado: %+v", resultado)
	}
	if registrador.id != 44 {
		t.Fatalf("especificacion enviada al registrador inesperada: %d", registrador.id)
	}
	if registrador.entrada.ProyectoSlug != "orquestador" || registrador.entrada.SolicitadoPor != "OpenClaw" {
		t.Fatalf("entrada al registrador inesperada: %+v", registrador.entrada)
	}
	if store.consumedID != 77 || store.markOrderStateID != 301 || store.markOrderStateEstado != "completada" {
		t.Fatalf("orden no completada correctamente: consumed=%d id=%d estado=%q", store.consumedID, store.markOrderStateID, store.markOrderStateEstado)
	}
	if !strings.Contains(store.markOrderStateResultado, `"receipt_source":"git_worktree"`) {
		t.Fatalf("resultado sin receipt git: %s", store.markOrderStateResultado)
	}
}

func TestSupersederMicroprogramacionAbiertaEquivalenteCancelaOrdenVieja(t *testing.T) {
	projectID := int64(11)
	store := &fakeStore{
		ordersResponse: []*db.RuntimeOrder{
			{
				ID:          305,
				Agente:      "Gemma1",
				ProyectoID:  &projectID,
				Tipo:        "send_instruction",
				Estado:      "ejecutando",
				PayloadJSON: `{"source":"microprogramacion","mailbox_id":88,"microprogramacion":{"especificacion_id":44,"archivo_objetivo":"microprogramacionapp/extractor_entrega.go","simbolo_objetivo":"recortarSeccionEvidencia","write_set":["microprogramacionapp/extractor_entrega.go","microprogramacionapp/extractor_entrega_test.go"],"formato_salida":"ficheros+evidencia"}}`,
			},
		},
	}
	service := NewService(store)

	err := service.supersederMicroprogramacionAbiertaEquivalente("Gemma1", &projectID, map[string]any{
		"especificacion_id": 45,
		"archivo_objetivo":  "microprogramacionapp/extractor_entrega.go",
		"simbolo_objetivo":  "recortarSeccionEvidencia",
		"write_set":         []string{"microprogramacionapp/extractor_entrega.go", "microprogramacionapp/extractor_entrega_test.go"},
	})
	if err != nil {
		t.Fatalf("supersederMicroprogramacionAbiertaEquivalente: %v", err)
	}
	if store.markOrderStateID != 305 || store.markOrderStateEstado != "cancelada" {
		t.Fatalf("orden vieja no supersedida: id=%d estado=%q", store.markOrderStateID, store.markOrderStateEstado)
	}
	if !strings.Contains(store.markOrderStateResultado, `"delivery_state":"superseded"`) {
		t.Fatalf("resultado de supersede inesperado: %s", store.markOrderStateResultado)
	}
}

func TestRegistrarEntregaGitMicroprogramacionActivaIgnoraFormatoNoGit(t *testing.T) {
	projectID := int64(11)
	store := &fakeStore{
		ordersResponse: []*db.RuntimeOrder{
			{
				ID:          302,
				Agente:      "Gemma1",
				ProyectoID:  &projectID,
				Tipo:        "send_instruction",
				PayloadJSON: `{"source":"microprogramacion","microprogramacion":{"especificacion_id":45,"archivo_objetivo":"microprogramacionapp/service.go","simbolo_objetivo":"MaterializarEntrega","write_set":["microprogramacionapp/service.go"],"formato_salida":"patch+evidencia"}}`,
			},
		},
	}
	registrador := &fakeRegistradorEntregaGit{}
	service := NewService(store)
	service.SetRegistradorEntregaGit(registrador)

	resultado, err := service.RegistrarEntregaGitMicroprogramacionActiva("Gemma1", &projectID, "orquestador", "diff listo", "OpenClaw")
	if err != nil {
		t.Fatalf("RegistrarEntregaGitMicroprogramacionActiva: %v", err)
	}
	if resultado != nil {
		t.Fatalf("no deberia registrar entrega git para formato no git: %+v", resultado)
	}
	if registrador.id != 0 || store.markOrderStateID != 0 {
		t.Fatalf("no deberia tocar registrador ni orden: registrador=%d order=%d", registrador.id, store.markOrderStateID)
	}
}

func TestIntentarRegistrarEntregaGitMicroprogramacionActivaToleraSinCambios(t *testing.T) {
	projectID := int64(11)
	store := &fakeStore{
		ordersResponse: []*db.RuntimeOrder{
			{
				ID:          303,
				Agente:      "Gemma1",
				ProyectoID:  &projectID,
				Tipo:        "send_instruction",
				PayloadJSON: `{"source":"microprogramacion","microprogramacion":{"especificacion_id":46,"archivo_objetivo":"microprogramacionapp/service.go","simbolo_objetivo":"RegistrarEntregaGit","write_set":["microprogramacionapp/service.go"],"formato_salida":"git_worktree+evidencia"}}`,
			},
		},
	}
	registrador := &fakeRegistradorEntregaGit{err: fmt.Errorf("la entrega git no contiene archivos modificados")}
	service := NewService(store)
	service.SetRegistradorEntregaGit(registrador)

	resultado, err := service.IntentarRegistrarEntregaGitMicroprogramacionActiva("Gemma1", &projectID, "orquestador", "sin cambios", "OpenClaw")
	if err != nil {
		t.Fatalf("IntentarRegistrarEntregaGitMicroprogramacionActiva: %v", err)
	}
	if resultado != nil || store.markOrderStateID != 0 {
		t.Fatalf("no deberia completar orden sin cambios: %+v order=%d", resultado, store.markOrderStateID)
	}
}

func TestRegistrarEntregaGitMicroprogramacionActivaPreferentePropagaWorktree(t *testing.T) {
	store := &fakeStore{
		ordersResponse: []*db.RuntimeOrder{{
			ID:          303,
			Agente:      "ClaudeSub1",
			Tipo:        "send_instruction",
			Estado:      "pendiente",
			PayloadJSON: `{"source":"microprogramacion","microprogramacion":{"especificacion_id":49,"archivo_objetivo":"microprogramacionapp/service.go","simbolo_objetivo":"RegistrarEntregaGit","write_set":["microprogramacionapp/service.go"],"formato_salida":"git_worktree+evidencia"}}`,
		}},
	}
	registrador := &fakeRegistradorEntregaGit{}
	service := NewService(store)
	service.SetRegistradorEntregaGit(registrador)

	worktreeID := int64(77)
	resultado, err := service.RegistrarEntregaGitMicroprogramacionActivaPreferente(
		"ClaudeSub1",
		nil,
		"orquestador",
		"thread_id=sub-1",
		"OpenClaw",
		microprogramacionapp.EntradaRegistrarEntregaGit{
			PreferenciaWorktreeID:   &worktreeID,
			PreferenciaRutaWorktree: "/tmp/orquesta-worktree-sub1",
			PreferenciaBranch:       "orq/orquestador/claude-sub1",
			PreferenciaBaseRef:      "main",
		},
	)
	if err != nil {
		t.Fatalf("RegistrarEntregaGitMicroprogramacionActivaPreferente: %v", err)
	}
	if resultado == nil || resultado.Entrega == nil {
		t.Fatalf("resultado vacío: %+v", resultado)
	}
	if registrador.entrada.PreferenciaWorktreeID == nil || *registrador.entrada.PreferenciaWorktreeID != worktreeID {
		t.Fatalf("preferencia worktree no propagada: %+v", registrador.entrada)
	}
	if registrador.entrada.PreferenciaRutaWorktree != "/tmp/orquesta-worktree-sub1" {
		t.Fatalf("ruta worktree no propagada: %+v", registrador.entrada)
	}
}

func TestRegistrarEntregaGitMicroprogramacionActivaUsaWorktreeDelPayload(t *testing.T) {
	store := &fakeStore{
		ordersResponse: []*db.RuntimeOrder{{
			ID:          304,
			Agente:      "Codex1",
			Tipo:        "send_instruction",
			Estado:      "pendiente",
			PayloadJSON: `{"source":"microprogramacion","microprogramacion":{"especificacion_id":50,"archivo_objetivo":"microprogramacionapp/service.go","simbolo_objetivo":"RegistrarEntregaGit","write_set":["microprogramacionapp/service.go"],"formato_salida":"git_worktree+evidencia","worktree_id":81,"ruta_worktree":"/tmp/orq-wt","branch_worktree":"orq/orquestador/codex1","base_ref_worktree":"main"}}`,
		}},
	}
	registrador := &fakeRegistradorEntregaGit{}
	service := NewService(store)
	service.SetRegistradorEntregaGit(registrador)

	_, err := service.RegistrarEntregaGitMicroprogramacionActiva("Codex1", nil, "orquestador", "diff listo", "OpenClaw")
	if err != nil {
		t.Fatalf("RegistrarEntregaGitMicroprogramacionActiva: %v", err)
	}
	if registrador.entrada.PreferenciaWorktreeID == nil || *registrador.entrada.PreferenciaWorktreeID != 81 {
		t.Fatalf("worktree payload no propagada: %+v", registrador.entrada)
	}
	if registrador.entrada.PreferenciaRutaWorktree != "/tmp/orq-wt" || registrador.entrada.PreferenciaBranch != "orq/orquestador/codex1" || registrador.entrada.PreferenciaBaseRef != "main" {
		t.Fatalf("preferencias de payload inesperadas: %+v", registrador.entrada)
	}
}

func TestRegistrarEntregaGitMicroprogramacionActivaFallbackPremiumUsaRegistradorYCompletaOrden(t *testing.T) {
	projectID := int64(11)
	store := &fakeStore{
		ordersResponse: []*db.RuntimeOrder{{
			ID:            405,
			Agente:        "Codex1",
			ProyectoID:    &projectID,
			Tipo:          "send_instruction",
			Estado:        "ejecutando",
			PayloadJSON:   `{"source":"pipeline_local","mailbox_id":91,"carril":"premium_worktree","tarea_objetivo_id":530,"write_set":["cmd/controlplane_support.go","db/controlplane_entities.go"],"worktree_id":81,"ruta_worktree":"/tmp/orq-premium","branch_worktree":"orq/orquestador/codex1","base_ref_worktree":"main"}`,
			ResultadoJSON: `{"delivery_state":"queued"}`,
		}},
	}
	registrador := &fakeRegistradorEntregaGitPremium{}
	completer := &fakeTaskCompleter{}
	service := NewService(store)
	service.SetRegistradorEntregaGitPremium(registrador)
	service.SetTaskCompleter(completer)

	resultado, err := service.RegistrarEntregaGitMicroprogramacionActiva("Codex1", &projectID, "orquestador", "diff listo", "OpenClaw")
	if err != nil {
		t.Fatalf("RegistrarEntregaGitMicroprogramacionActiva fallback premium: %v", err)
	}
	if resultado == nil || resultado.ContextoPremium == nil || resultado.ContextoPremium.RuntimeOrderID != 405 {
		t.Fatalf("resultado/contexto premium inesperado: %+v", resultado)
	}
	if resultado.EntregaPremium == nil || resultado.EntregaPremium.GitMergeID != 77 {
		t.Fatalf("entrega premium inesperada: %+v", resultado)
	}
	if registrador.entrada.Carril != "premium_worktree" || registrador.entrada.TareaObjetivoID != 530 {
		t.Fatalf("payload premium no propagado: %+v", registrador.entrada)
	}
	if registrador.entrada.PreferenciaWorktreeID == nil || *registrador.entrada.PreferenciaWorktreeID != 81 {
		t.Fatalf("worktree premium no propagada: %+v", registrador.entrada)
	}
	if len(registrador.entrada.WriteSet) != 2 || registrador.entrada.WriteSet[0] != "cmd/controlplane_support.go" {
		t.Fatalf("write_set premium no propagado: %+v", registrador.entrada)
	}
	if completer.id != 530 || completer.agente != "Codex1" || completer.commit != "abc123" {
		t.Fatalf("tarea premium no completada via app: %+v", completer)
	}
	if store.consumedID != 91 || store.markOrderStateID != 405 || store.markOrderStateEstado != "completada" {
		t.Fatalf("orden premium no completada correctamente: consumed=%d id=%d estado=%q", store.consumedID, store.markOrderStateID, store.markOrderStateEstado)
	}
	if !strings.Contains(store.markOrderStateResultado, `"receipt_source":"git_worktree"`) {
		t.Fatalf("resultado premium sin receipt git: %s", store.markOrderStateResultado)
	}
}

func TestRegistrarEntregaGitMicroprogramacionActivaFallbackPremiumUsaMailboxKindPipelineLocal(t *testing.T) {
	projectID := int64(11)
	store := &fakeStore{
		ordersResponse: []*db.RuntimeOrder{{
			ID:            405,
			Agente:        "Codex1",
			ProyectoID:    &projectID,
			Tipo:          "send_instruction",
			Estado:        "ejecutando",
			PayloadJSON:   `{"kind":"pipeline_local","mailbox_kind":"pipeline_local","mailbox_id":91,"carril":"premium_worktree","tarea_objetivo_id":530,"write_set":["cmd/controlplane_support.go","db/controlplane_entities.go"],"worktree_id":81,"ruta_worktree":"/tmp/orq-premium","branch_worktree":"orq/orquestador/codex1","base_ref_worktree":"main"}`,
			ResultadoJSON: `{"delivery_state":"queued"}`,
		}},
	}
	registrador := &fakeRegistradorEntregaGitPremium{}
	service := NewService(store)
	service.SetRegistradorEntregaGitPremium(registrador)

	resultado, err := service.RegistrarEntregaGitMicroprogramacionActiva("Codex1", &projectID, "orquestador", "diff listo", "OpenClaw")
	if err != nil {
		t.Fatalf("RegistrarEntregaGitMicroprogramacionActiva fallback premium mailbox kind: %v", err)
	}
	if resultado == nil || resultado.ContextoPremium == nil || resultado.ContextoPremium.RuntimeOrderID != 405 {
		t.Fatalf("resultado/contexto premium inesperado: %+v", resultado)
	}
	if resultado.EntregaPremium == nil || resultado.EntregaPremium.GitMergeID != 77 {
		t.Fatalf("entrega premium inesperada: %+v", resultado)
	}
	if registrador.entrada.Carril != "premium_worktree" || registrador.entrada.TareaObjetivoID != 530 {
		t.Fatalf("payload premium no propagado: %+v", registrador.entrada)
	}
	if store.consumedID != 91 || store.markOrderStateID != 405 || store.markOrderStateEstado != "completada" {
		t.Fatalf("orden premium no completada correctamente: consumed=%d id=%d estado=%q", store.consumedID, store.markOrderStateID, store.markOrderStateEstado)
	}
	if !strings.Contains(store.markOrderStateResultado, `"receipt_source":"git_worktree"`) {
		t.Fatalf("resultado premium sin receipt git: %s", store.markOrderStateResultado)
	}
}

func TestResolverContextoEntregaGitPremiumDesdeBootstrapLease(t *testing.T) {
	projectID := int64(11)
	store := &fakeStore{
		ordersResponse: []*db.RuntimeOrder{{
			ID:            406,
			Agente:        "Codex1",
			ProyectoID:    &projectID,
			Tipo:          "start",
			Estado:        "completada",
			ResultadoJSON: `{"lease_state":"delivered","mailbox_ids":[91],"sesion_id":12}`,
		}},
		mailboxResponse: []*db.RuntimeMailboxMessage{{
			ID:          91,
			ToAgente:    "Codex1",
			ProyectoID:  &projectID,
			Kind:        "pipeline_local",
			PayloadJSON: `{"source":"pipeline_local","carril":"premium_worktree","tarea_objetivo_id":530,"write_set":["cmd/controlplane_support.go","db/controlplane_entities.go"],"worktree_id":81,"ruta_worktree":"/tmp/orq-premium","branch_worktree":"orq/orquestador/codex1","base_ref_worktree":"main"}`,
		}},
	}
	service := NewService(store)

	ctx, err := service.ResolverContextoEntregaGitPremium("Codex1", &projectID)
	if err != nil {
		t.Fatalf("ResolverContextoEntregaGitPremium: %v", err)
	}
	if ctx == nil {
		t.Fatal("deberia resolver contexto premium desde bootstrap lease")
	}
	if ctx.RuntimeOrderID != 406 || ctx.Carril != "premium_worktree" || ctx.TareaObjetivoID != 530 {
		t.Fatalf("contexto premium inesperado: %+v", ctx)
	}
	if ctx.WorktreeID == nil || *ctx.WorktreeID != 81 {
		t.Fatalf("worktree premium inesperada: %+v", ctx)
	}
	if len(ctx.WriteSet) != 2 || ctx.WriteSet[1] != "db/controlplane_entities.go" {
		t.Fatalf("write_set premium inesperado: %+v", ctx)
	}
}

func TestResolverContextoEntregaGitPremiumDesdeMailboxKindPipelineLocal(t *testing.T) {
	projectID := int64(11)
	store := &fakeStore{
		ordersResponse: []*db.RuntimeOrder{{
			ID:            406,
			Agente:        "Codex1",
			ProyectoID:    &projectID,
			Tipo:          "send_instruction",
			Estado:        "completada",
			PayloadJSON:   `{"kind":"pipeline_local","mailbox_kind":"pipeline_local","carril":"premium_worktree","tarea_objetivo_id":530,"write_set":["cmd/controlplane_support.go","db/controlplane_entities.go"],"worktree_id":81,"ruta_worktree":"/tmp/orq-premium","branch_worktree":"orq/orquestador/codex1","base_ref_worktree":"main"}`,
			ResultadoJSON: `{"delivery_state":"notified"}`,
		}},
	}
	service := NewService(store)

	ctx, err := service.ResolverContextoEntregaGitPremium("Codex1", &projectID)
	if err != nil {
		t.Fatalf("ResolverContextoEntregaGitPremium mailbox kind: %v", err)
	}
	if ctx == nil {
		t.Fatal("deberia resolver contexto premium desde mailbox_kind pipeline_local")
	}
	if ctx.RuntimeOrderID != 406 || ctx.Carril != "premium_worktree" || ctx.TareaObjetivoID != 530 {
		t.Fatalf("contexto premium inesperado: %+v", ctx)
	}
	if ctx.WorktreeID == nil || *ctx.WorktreeID != 81 {
		t.Fatalf("worktree premium inesperada: %+v", ctx)
	}
}

func TestResolverContextoEntregaGitPremiumDesdeBootstrapLeaseCaseInsensitive(t *testing.T) {
	projectID := int64(11)
	store := &fakeStore{
		ordersResponse: []*db.RuntimeOrder{{
			ID:            406,
			Agente:        "Codex1",
			ProyectoID:    &projectID,
			Tipo:          "start",
			Estado:        "completada",
			ResultadoJSON: `{"lease_state":"Delivered","mailbox_ids":[91],"sesion_id":12}`,
		}},
		mailboxResponse: []*db.RuntimeMailboxMessage{{
			ID:          91,
			ToAgente:    "Codex1",
			ProyectoID:  &projectID,
			Kind:        "pipeline_local",
			PayloadJSON: `{"source":"pipeline_local","carril":"premium_worktree","tarea_objetivo_id":530,"write_set":["cmd/controlplane_support.go","db/controlplane_entities.go"],"worktree_id":81,"ruta_worktree":"/tmp/orq-premium","branch_worktree":"orq/orquestador/codex1","base_ref_worktree":"main"}`,
		}},
	}
	service := NewService(store)

	ctx, err := service.ResolverContextoEntregaGitPremium("Codex1", &projectID)
	if err != nil {
		t.Fatalf("ResolverContextoEntregaGitPremium: %v", err)
	}
	if ctx == nil {
		t.Fatal("deberia resolver contexto premium desde bootstrap lease con lease_state case-insensitive")
	}
	if ctx.RuntimeOrderID != 406 || ctx.Carril != "premium_worktree" || ctx.TareaObjetivoID != 530 {
		t.Fatalf("contexto premium inesperado: %+v", ctx)
	}
}

func TestRegistrarEntregaGitMicroprogramacionActivaFallbackPremiumBootstrapLeaseUsaRegistradorYCompletaOrden(t *testing.T) {
	projectID := int64(11)
	store := &fakeStore{
		ordersResponse: []*db.RuntimeOrder{{
			ID:            406,
			Agente:        "Codex1",
			ProyectoID:    &projectID,
			Tipo:          "start",
			Estado:        "completada",
			ResultadoJSON: `{"lease_state":"delivered","mailbox_ids":[91],"sesion_id":12,"delivery_state":"delivered"}`,
		}},
		mailboxResponse: []*db.RuntimeMailboxMessage{{
			ID:          91,
			ToAgente:    "Codex1",
			ProyectoID:  &projectID,
			Kind:        "pipeline_local",
			PayloadJSON: `{"source":"pipeline_local","carril":"premium_worktree","tarea_objetivo_id":530,"write_set":["cmd/controlplane_support.go","db/controlplane_entities.go"],"worktree_id":81,"ruta_worktree":"/tmp/orq-premium","branch_worktree":"orq/orquestador/codex1","base_ref_worktree":"main"}`,
		}},
	}
	registrador := &fakeRegistradorEntregaGitPremium{}
	completer := &fakeTaskCompleter{}
	service := NewService(store)
	service.SetRegistradorEntregaGitPremium(registrador)
	service.SetTaskCompleter(completer)

	resultado, err := service.RegistrarEntregaGitMicroprogramacionActiva("Codex1", &projectID, "orquestador", "diff listo", "OpenClaw")
	if err != nil {
		t.Fatalf("RegistrarEntregaGitMicroprogramacionActiva fallback premium bootstrap: %v", err)
	}
	if resultado == nil || resultado.ContextoPremium == nil || resultado.ContextoPremium.RuntimeOrderID != 406 {
		t.Fatalf("resultado/contexto premium inesperado: %+v", resultado)
	}
	if resultado.EntregaPremium == nil || resultado.EntregaPremium.GitMergeID != 77 {
		t.Fatalf("entrega premium inesperada: %+v", resultado)
	}
	if registrador.entrada.PreferenciaWorktreeID == nil || *registrador.entrada.PreferenciaWorktreeID != 81 {
		t.Fatalf("worktree premium no propagada: %+v", registrador.entrada)
	}
	if len(registrador.entrada.WriteSet) != 2 || registrador.entrada.WriteSet[0] != "cmd/controlplane_support.go" {
		t.Fatalf("write_set premium bootstrap no propagado: %+v", registrador.entrada)
	}
	if completer.id != 530 || completer.agente != "Codex1" || completer.commit != "abc123" {
		t.Fatalf("tarea premium bootstrap no completada via app: %+v", completer)
	}
	if store.consumedID != 91 || store.markOrderStateID != 406 || store.markOrderStateEstado != "completada" {
		t.Fatalf("bootstrap premium no completado correctamente: consumed=%d id=%d estado=%q", store.consumedID, store.markOrderStateID, store.markOrderStateEstado)
	}
	if !strings.Contains(store.markOrderStateResultado, `"receipt_source":"git_worktree"`) {
		t.Fatalf("resultado premium bootstrap sin receipt git: %s", store.markOrderStateResultado)
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

func TestRegistrarSalidaObservadaUsaHandleYRuntimeActivos(t *testing.T) {
	projectID := int64(7)
	store := &fakeStore{
		operationalProjResp:    &db.RuntimeHandle{ID: 18, Agente: "Gemma1", ProyectoID: &projectID, RuntimeID: apuntarInt64(31), Estado: "activo"},
		handleByIDResp:         &db.RuntimeHandle{ID: 18, Agente: "Gemma1", ProyectoID: &projectID, RuntimeID: apuntarInt64(31), Estado: "activo"},
		runtimeResponse:        &db.RuntimeInstance{ID: 31, Agente: "Gemma1", ProyectoID: &projectID},
		transcriptRegistradoID: 44,
	}
	service := NewService(store)
	id, err := service.RegistrarSalidaObservada("Gemma1", &projectID, "PATCH_UNIFICADO:\n--- a/a.go\n+++ b/a.go")
	if err != nil {
		t.Fatalf("RegistrarSalidaObservada: %v", err)
	}
	if id != 44 {
		t.Fatalf("id transcript inesperado: %d", id)
	}
	if store.transcriptRegistrado == nil {
		t.Fatal("faltaba transcript registrado")
	}
	if store.transcriptRegistrado.RuntimeID != 31 || store.transcriptRegistrado.HandleID == nil || *store.transcriptRegistrado.HandleID != 18 {
		t.Fatalf("transcript registrado inesperado: %+v", store.transcriptRegistrado)
	}
	if store.transcriptRegistrado.Stream != "pty_out" || !strings.Contains(store.transcriptRegistrado.Text, "PATCH_UNIFICADO") {
		t.Fatalf("transcript registrado inesperado: %+v", store.transcriptRegistrado)
	}
}

func TestRegistrarSalidaObservadaInvocaHookTrasRegistrarTranscript(t *testing.T) {
	projectID := int64(7)
	store := &fakeStore{
		operationalProjResp:    &db.RuntimeHandle{ID: 18, Agente: "Gemma1", ProyectoID: &projectID, RuntimeID: apuntarInt64(31), Estado: "activo"},
		handleByIDResp:         &db.RuntimeHandle{ID: 18, Agente: "Gemma1", ProyectoID: &projectID, RuntimeID: apuntarInt64(31), Estado: "activo"},
		runtimeResponse:        &db.RuntimeInstance{ID: 31, Agente: "Gemma1", ProyectoID: &projectID},
		transcriptRegistradoID: 45,
	}
	service := NewService(store)
	var hookID int64
	var hookEntry *db.RuntimeTranscriptEntry
	service.SetAfterRegisterRuntimeTranscriptHook(func(entry *db.RuntimeTranscriptEntry, transcriptID int64) {
		hookID = transcriptID
		hookEntry = entry
	})
	if _, err := service.RegistrarSalidaObservada("Gemma1", &projectID, "salida útil"); err != nil {
		t.Fatalf("RegistrarSalidaObservada: %v", err)
	}
	if hookID != 45 {
		t.Fatalf("hook transcript id inesperado: %d", hookID)
	}
	if hookEntry == nil || hookEntry.RuntimeID != 31 || hookEntry.HandleID == nil || *hookEntry.HandleID != 18 {
		t.Fatalf("hook transcript entry inesperada: %+v", hookEntry)
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

func TestEnqueueAgentControlStartToleraSinHandleNiRuntimePrevios(t *testing.T) {
	projectID := int64(9)
	store := &fakeStore{
		projectResponse:      &db.Proyecto{ID: projectID, Slug: "orquestador"},
		agentResponse:        &db.Agente{Nombre: "GemmaProgramador4", Rol: "programador"},
		operationalProjErr:   sql.ErrNoRows,
		handleProjErr:        sql.ErrNoRows,
		operationalHandleErr: sql.ErrNoRows,
		createOrderID:        88,
	}
	service := NewService(store)

	orderID, accion, err := service.EnqueueAgentControl(AgentControlRequest{
		Agente:       "GemmaProgramador4",
		Proyecto:     "orquestador",
		Accion:       "arrancar",
		Conector:     "ollama_pool_local",
		Modelo:       "gemma4:26b",
		Razonamiento: "high",
		Perfil:       "implementacion",
		Motivo:       "arranque inicial limpio",
		Por:          "test",
	})
	if err != nil {
		t.Fatalf("EnqueueAgentControl start limpio: %v", err)
	}
	if orderID != 88 || accion != "start" {
		t.Fatalf("orderID=%d accion=%s", orderID, accion)
	}
	if store.createOrder == nil {
		t.Fatalf("createOrder nil")
	}
	if store.createOrder.HandleID != nil || store.createOrder.RuntimeID != nil {
		t.Fatalf("no deberia reutilizar handle/runtime inexistentes: %+v", store.createOrder)
	}
}

func TestEnqueueAgentControlStartReusesExistingLiveStartOrder(t *testing.T) {
	projectID := int64(9)
	store := &fakeStore{
		projectResponse: &db.Proyecto{ID: projectID, Slug: "orquestador"},
		agentResponse:   &db.Agente{Nombre: "Codex2", Rol: "programador"},
		ordersResponse: []*db.RuntimeOrder{
			{
				ID:          91,
				Agente:      "Codex2",
				ProyectoID:  &projectID,
				Tipo:        "start",
				Estado:      "ejecutando",
				PayloadJSON: `{"accion":"start","proyecto":"orquestador"}`,
			},
			{
				ID:          88,
				Agente:      "Codex2",
				ProyectoID:  &projectID,
				Tipo:        "start",
				Estado:      "pendiente",
				PayloadJSON: `{"accion":"start","proyecto":"orquestador"}`,
			},
		},
	}
	service := NewService(store)

	orderID, accion, err := service.EnqueueAgentControl(AgentControlRequest{
		Agente:   "Codex2",
		Proyecto: "orquestador",
		Accion:   "arrancar",
		Motivo:   "retry start",
		Por:      "test",
	})
	if err != nil {
		t.Fatalf("EnqueueAgentControl start reuse: %v", err)
	}
	if orderID != 91 || accion != "start" {
		t.Fatalf("orderID=%d accion=%s", orderID, accion)
	}
	if store.createOrder != nil {
		t.Fatalf("no deberia crear orden nueva si ya existe start viva: %+v", store.createOrder)
	}
	if store.sharedAccountAgent != "" {
		t.Fatalf("no deberia reevaluar cuota si ya existe start viva: %q", store.sharedAccountAgent)
	}
	if store.auditAction != "control_agente_start_reuse" || store.auditEntityID != 91 {
		t.Fatalf("audit reuse inesperada: action=%q entityID=%d", store.auditAction, store.auditEntityID)
	}
	if len(store.markedOrderStates) != 1 || store.markedOrderStates[0] != 88 || store.markOrderStateEstado != "cancelada" {
		t.Fatalf("deberia cancelar starts duplicadas mas viejas: ids=%v estado=%q", store.markedOrderStates, store.markOrderStateEstado)
	}
}

func TestEnqueueAgentControlStartNoReusesPendingStartOrder(t *testing.T) {
	projectID := int64(9)
	store := &fakeStore{
		projectResponse: &db.Proyecto{ID: projectID, Slug: "orquestador"},
		agentResponse:   &db.Agente{Nombre: "Codex3", Rol: "programador"},
		ordersResponse: []*db.RuntimeOrder{
			{
				ID:          91,
				Agente:      "Codex3",
				ProyectoID:  &projectID,
				Tipo:        "start",
				Estado:      "pendiente",
				PayloadJSON: `{"accion":"start","proyecto":"orquestador"}`,
			},
		},
		createOrderID: 92,
	}
	service := NewService(store)

	orderID, accion, err := service.EnqueueAgentControl(AgentControlRequest{
		Agente:   "Codex3",
		Proyecto: "orquestador",
		Accion:   "arrancar",
		Motivo:   "nuevo intento tras start zombi",
		Por:      "test",
	})
	if err != nil {
		t.Fatalf("EnqueueAgentControl pending start: %v", err)
	}
	if orderID != 92 || accion != "start" {
		t.Fatalf("orderID=%d accion=%s", orderID, accion)
	}
	if store.createOrder == nil || store.createOrder.Tipo != "start" {
		t.Fatalf("deberia crear nueva start y no reutilizar la pendiente: %+v", store.createOrder)
	}
	if len(store.markedOrderStates) != 1 || store.markedOrderStates[0] != 91 || store.markOrderStateEstado != "cancelada" {
		t.Fatalf("deberia cancelar la start pendiente previa: ids=%v estado=%q", store.markedOrderStates, store.markOrderStateEstado)
	}
	if store.auditAction != "control_agente_start" || store.auditEntityID != 92 {
		t.Fatalf("audit final inesperada: action=%q entityID=%d", store.auditAction, store.auditEntityID)
	}
}

func TestEnqueueAgentControlStartReconcilesStaleOrdersBeforeReuse(t *testing.T) {
	projectID := int64(9)
	store := &fakeStore{
		projectResponse: &db.Proyecto{ID: projectID, Slug: "orquestador"},
		agentResponse:   &db.Agente{Nombre: "Codex2", Rol: "programador"},
		ordersResponse: []*db.RuntimeOrder{
			{
				ID:          91,
				Agente:      "Codex2",
				ProyectoID:  &projectID,
				Tipo:        "start",
				Estado:      "ejecutando",
				PayloadJSON: `{"accion":"start","proyecto":"orquestador"}`,
			},
		},
	}
	service := NewService(store)

	orderID, accion, err := service.EnqueueAgentControl(AgentControlRequest{
		Agente:   "Codex2",
		Proyecto: "orquestador",
		Accion:   "arrancar",
		Por:      "test",
	})
	if err != nil {
		t.Fatalf("EnqueueAgentControl start reconcile: %v", err)
	}
	if orderID != 91 || accion != "start" {
		t.Fatalf("orderID=%d accion=%s", orderID, accion)
	}
	if store.reconcileOrdersCalls != 1 {
		t.Fatalf("deberia reconciliar stale antes del reuse, got=%d", store.reconcileOrdersCalls)
	}
}

func TestEnqueueAgentControlStartNoReusesLiveOrderWithDifferentExecutionProfile(t *testing.T) {
	projectID := int64(9)
	store := &fakeStore{
		projectResponse: &db.Proyecto{ID: projectID, Slug: "orquestador"},
		agentResponse:   &db.Agente{Nombre: "Codex4", Rol: "programador"},
		ordersResponse: []*db.RuntimeOrder{
			{
				ID:          91,
				Agente:      "Codex4",
				ProyectoID:  &projectID,
				Tipo:        "start",
				Estado:      "ejecutando",
				PayloadJSON: `{"accion":"start","proyecto":"orquestador","modelo":"gpt-5.4","perfil":"spec"}`,
			},
		},
		createOrderID: 92,
	}
	service := NewService(store)

	orderID, accion, err := service.EnqueueAgentControl(AgentControlRequest{
		Agente:   "Codex4",
		Proyecto: "orquestador",
		Accion:   "arrancar",
		Modelo:   "gpt-5.4",
		Perfil:   "implementacion",
		Por:      "test",
	})
	if err != nil {
		t.Fatalf("EnqueueAgentControl start different profile: %v", err)
	}
	if orderID != 92 || accion != "start" {
		t.Fatalf("orderID=%d accion=%s", orderID, accion)
	}
	if store.createOrder == nil || store.createOrder.Tipo != "start" {
		t.Fatalf("deberia crear nueva start: %+v", store.createOrder)
	}
	if len(store.markedOrderStates) != 1 || store.markedOrderStates[0] != 91 || store.markOrderStateEstado != "cancelada" {
		t.Fatalf("deberia superseder la start viva no equivalente: ids=%v estado=%q", store.markedOrderStates, store.markOrderStateEstado)
	}
}

func TestEnqueueAgentControlStartSupersedesPausedHandleConStopPendiente(t *testing.T) {
	projectID := int64(9)
	runtimeID := int64(12)
	store := &fakeStore{
		projectResponse: &db.Proyecto{ID: projectID, Slug: "orquestador"},
		agentResponse:   &db.Agente{Nombre: "Codex8", Rol: "programador"},
		handleProjResp:  &db.RuntimeHandle{ID: 10, RuntimeID: &runtimeID, Estado: "pausado"},
		ordersResponse:  []*db.RuntimeOrder{{ID: 91, Agente: "Codex8", Tipo: "stop", Estado: "pendiente", ProyectoID: &projectID}},
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

func TestEnqueueAgentControlStartConProyectoExplicitoNoUsaFallbackGlobalDeOtroProyecto(t *testing.T) {
	projectID := int64(9)
	oldRuntimeID := int64(41)
	oldHandleRuntimeID := int64(40)
	ptr := func(v int64) *int64 { return &v }
	store := &fakeStore{
		projectResponse:       &db.Proyecto{ID: projectID, Slug: "orquestador"},
		agentResponse:         &db.Agente{Nombre: "Codex2", Rol: "programador"},
		operationalHandleResp: &db.RuntimeHandle{ID: 11, RuntimeID: &oldHandleRuntimeID, Estado: "activo", ProyectoID: ptr(77)},
		handleResponse:        &db.RuntimeHandle{ID: 12, RuntimeID: &oldHandleRuntimeID, Estado: "activo", ProyectoID: ptr(77)},
		primaryRuntimeResp:    &db.RuntimeInstance{ID: oldRuntimeID, LogicalState: "esperando_io", ProcessState: "running", ProyectoID: ptr(77)},
		createOrderID:         101,
	}
	service := NewService(store)

	orderID, accion, err := service.EnqueueAgentControl(AgentControlRequest{
		Agente:   "Codex2",
		Proyecto: "orquestador",
		Accion:   "arrancar",
		Motivo:   "test_project_boundary",
		Por:      "test",
	})
	if err != nil {
		t.Fatalf("EnqueueAgentControl start project boundary: %v", err)
	}
	if orderID != 101 || accion != "start" {
		t.Fatalf("orderID=%d accion=%s", orderID, accion)
	}
	if store.operationalHandleAgent != "" || store.handleAgent != "" {
		t.Fatalf("no deberia consultar fallback global: operational=%q active=%q", store.operationalHandleAgent, store.handleAgent)
	}
	if store.createOrder == nil || store.createOrder.Tipo != "start" {
		t.Fatalf("createOrder inesperado: %+v", store.createOrder)
	}
	if store.createOrder.HandleID != nil || store.createOrder.RuntimeID != nil {
		t.Fatalf("start limpio no deberia contaminarse con ids de otro proyecto: %+v", store.createOrder)
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

func TestEnqueueAgentControlResumeDegradaAStartParaCodexTMUXNoInteractivo(t *testing.T) {
	projectID := int64(9)
	runtimeID := int64(12)
	store := &fakeStore{
		projectResponse:        &db.Proyecto{ID: projectID, Slug: "orquestador"},
		agentResponse:          &db.Agente{Nombre: "Codex8", Rol: "programador"},
		handleProjResp:         &db.RuntimeHandle{ID: 10, RuntimeID: &runtimeID, Estado: "activo", Transporte: "tmux", HandleKind: "session", MetadataJSON: `{"driver":"tmux_cli_session","mailbox_delivery_mode":"bootstrap_only","can_send_input":false}`, CapabilitiesJSON: `{"mailbox_delivery_mode":"bootstrap_only","can_send_input":false}`},
		primaryProjRuntimeResp: &db.RuntimeInstance{ID: runtimeID, LogicalState: "esperando_io", ProcessState: "running"},
		createOrderID:          96,
	}
	service := NewService(store)

	orderID, accion, err := service.EnqueueAgentControl(AgentControlRequest{
		Agente:   "Codex8",
		Proyecto: "orquestador",
		Accion:   "reanudar",
		Motivo:   "microciclo",
		Por:      "test",
	})
	if err != nil {
		t.Fatalf("EnqueueAgentControl resume->start codex tmux: %v", err)
	}
	if orderID != 96 || accion != "start" {
		t.Fatalf("orderID=%d accion=%s", orderID, accion)
	}
	if store.createOrder == nil || store.createOrder.Tipo != "start" {
		t.Fatalf("createOrder inesperado: %+v", store.createOrder)
	}
	if store.createOrder.HandleID != nil || store.createOrder.RuntimeID != nil {
		t.Fatalf("resume degradado a start no deberia atarse al handle viejo: %+v", store.createOrder)
	}
}

func TestEnqueueAgentControlResumeRecuperaHandleCanonicoDegradadoConSesionExterna(t *testing.T) {
	projectID := int64(9)
	runtimeID := int64(12)
	handleID := int64(10)
	store := &fakeStore{
		projectResponse: &db.Proyecto{ID: projectID, Slug: "orquestador"},
		agentResponse:   &db.Agente{Nombre: "Codex8", Rol: "programador"},
		canonicalHandlesResponse: []*db.RuntimeHandle{{
			ID:           handleID,
			Agente:       "Codex8",
			ProyectoID:   &projectID,
			RuntimeID:    &runtimeID,
			Transporte:   "api",
			HandleKind:   "session",
			HandleRef:    "sess-remote-codex8",
			Estado:       "fallido",
			MetadataJSON: `{"external_session_id":"sess-remote-codex8"}`,
		}},
		primaryProjRuntimeResp: &db.RuntimeInstance{ID: runtimeID, LogicalState: "degradado", ProcessState: "remote_status_error"},
		createOrderID:          97,
	}
	service := NewService(store)

	orderID, accion, err := service.EnqueueAgentControl(AgentControlRequest{
		Agente:   "Codex8",
		Proyecto: "orquestador",
		Accion:   "reanudar",
		Motivo:   "remote_runtime_degraded",
		Por:      "test",
	})
	if err != nil {
		t.Fatalf("EnqueueAgentControl resume degraded canonical: %v", err)
	}
	if orderID != 97 || accion != "resume" {
		t.Fatalf("orderID=%d accion=%s", orderID, accion)
	}
	if store.canonicalHandlesFilter == nil || *store.canonicalHandlesFilter != "Codex8" {
		t.Fatalf("deberia consultar handles canónicos del agente: %+v", store.canonicalHandlesFilter)
	}
	if store.createOrder == nil || store.createOrder.Tipo != "resume" {
		t.Fatalf("createOrder inesperado: %+v", store.createOrder)
	}
	if store.createOrder.HandleID == nil || *store.createOrder.HandleID != handleID {
		t.Fatalf("resume remoto deberia conservar el handle canónico degradado: %+v", store.createOrder)
	}
	if store.createOrder.RuntimeID == nil || *store.createOrder.RuntimeID != runtimeID {
		t.Fatalf("resume remoto deberia apuntar al runtime canónico: %+v", store.createOrder)
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

func TestCloseResidualRuntimeHandlesDelegatesAndAudits(t *testing.T) {
	projectID := int64(9)
	store := &fakeStore{
		projectResponse: &db.Proyecto{ID: projectID, Slug: "orquestador"},
		agentResponse:   &db.Agente{Nombre: "Codex5", Rol: "programador"},
	}
	service := NewService(store)

	resultado, err := service.CloseResidualRuntimeHandles(RuntimeHandleResidualCloseRequest{
		Agente:   "Codex5",
		Proyecto: "orquestador",
		Actor:    "Codex1",
	})
	if err != nil {
		t.Fatalf("CloseResidualRuntimeHandles: %v", err)
	}
	if resultado == nil || !resultado.Closed {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
	if store.closeHandlesAgent != "Codex5" {
		t.Fatalf("closeHandlesAgent=%q", store.closeHandlesAgent)
	}
	if store.closeHandlesProjectID == nil || *store.closeHandlesProjectID != projectID {
		t.Fatalf("closeHandlesProjectID=%v", store.closeHandlesProjectID)
	}
	if store.auditAction != "cerrar_runtime_handles_residuales" || store.auditAgent != "Codex1" {
		t.Fatalf("audit inesperado agente=%q accion=%q detalle=%q", store.auditAgent, store.auditAction, store.auditDetail)
	}
}

func TestCloseResidualRuntimesDelegatesAndAudits(t *testing.T) {
	projectID := int64(9)
	store := &fakeStore{
		projectResponse: &db.Proyecto{ID: projectID, Slug: "orquestador"},
		agentResponse:   &db.Agente{Nombre: "Codex5", Rol: "programador"},
		runtimesResponse: []*db.RuntimeInstance{
			{ID: 21, Agente: "Codex5", ProyectoID: &projectID, LogicalState: "activo"},
			{ID: 22, Agente: "Codex5", ProyectoID: &projectID, LogicalState: "esperando_io"},
			{ID: 23, Agente: "Codex5", ProyectoID: &projectID, LogicalState: "cerrado"},
		},
	}
	service := NewService(store)

	resultado, err := service.CloseResidualRuntimes(RuntimeResidualCloseRequest{
		Agente:   "Codex5",
		Proyecto: "orquestador",
		Actor:    "Codex1",
	})
	if err != nil {
		t.Fatalf("CloseResidualRuntimes: %v", err)
	}
	if resultado == nil || resultado.Closed != 2 {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
	if len(store.closedRuntimeIDs) != 2 || store.closedRuntimeIDs[0] != 21 || store.closedRuntimeIDs[1] != 22 {
		t.Fatalf("closedRuntimeIDs=%v", store.closedRuntimeIDs)
	}
	if store.runtimesFilter.Agente == nil || *store.runtimesFilter.Agente != "Codex5" {
		t.Fatalf("runtimesFilter agente=%v", store.runtimesFilter.Agente)
	}
	if store.runtimesFilter.ProyectoID == nil || *store.runtimesFilter.ProyectoID != projectID {
		t.Fatalf("runtimesFilter proyecto=%v", store.runtimesFilter.ProyectoID)
	}
	if store.runtimesFilter.Activos == nil || !*store.runtimesFilter.Activos {
		t.Fatalf("runtimesFilter activos=%v", store.runtimesFilter.Activos)
	}
	if store.auditAction != "cerrar_runtimes_residuales" || store.auditAgent != "Codex1" || !strings.Contains(store.auditDetail, "closed=2") {
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

func apuntarInt64(v int64) *int64 {
	return &v
}
