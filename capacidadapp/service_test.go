package capacidadapp

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"orquesta/db"
	"orquesta/reviewapp"
)

func ptrInt64(v int64) *int64    { return &v }
func ptrString(v string) *string { return &v }

type fakeStore struct{}

type fakePoolLocalProvider struct {
	telemetria *TelemetriaPoolLocal
}

type fakeRuntimeModelManager struct {
	items      []ModeloRuntimeActivo
	downloaded []string
	listErr    error
	downErr    error
}

type fakeTaskProvider struct {
	tareas []*db.Tarea
}

type fakeTaskActionProvider struct {
	tareas map[int64]*db.Tarea
}

type fakePhaseControlProvider struct {
	fases  []*db.FaseProyecto
	nextID int64
}

type fakeReviewGateProvider struct {
	items []*reviewapp.Gate
}

type fakeReviewGateManager struct {
	created []reviewapp.CreateGateInput
}

type fakeAgentResolver struct {
	agente string
}

type fakePipelineDispatcher struct {
	resultado *ResultadoDespachoPipeline
	entrada   SolicitudDespachoPipeline
}

func (f fakePoolLocalProvider) DescribirPoolLocalCompartido(slug string) (*TelemetriaPoolLocal, error) {
	if f.telemetria == nil || f.telemetria.PoolSlug != slug {
		return nil, nil
	}
	copia := *f.telemetria
	return &copia, nil
}

func (f *fakeRuntimeModelManager) ListarModelosActivos() ([]ModeloRuntimeActivo, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return append([]ModeloRuntimeActivo(nil), f.items...), nil
}

func (f *fakeRuntimeModelManager) DescargarModelo(modelo string) error {
	if f.downErr != nil {
		return f.downErr
	}
	f.downloaded = append(f.downloaded, modelo)
	return nil
}

func (f fakeTaskProvider) ListProjectTasks(proyecto string) ([]*db.Tarea, error) {
	return f.tareas, nil
}

func (f *fakeTaskActionProvider) GetTask(id int64) (*db.Tarea, error) {
	return f.tareas[id], nil
}

func (f *fakeTaskActionProvider) TakeTask(id int64, agente string) error {
	item := f.tareas[id]
	if item == nil {
		return nil
	}
	item.Estado = db.TareaAsignada
	item.Agente = &agente
	return nil
}

func (f *fakeTaskActionProvider) ReassignTask(id int64, agente string) error {
	item := f.tareas[id]
	if item == nil {
		return nil
	}
	item.Agente = &agente
	return nil
}

func (f *fakeTaskActionProvider) StartTask(id int64, agente string) error {
	item := f.tareas[id]
	if item == nil {
		return nil
	}
	item.Estado = db.TareaEnProgreso
	item.Agente = &agente
	return nil
}

func (f *fakePhaseControlProvider) ListProjectPhases(proyecto string) ([]*db.FaseProyecto, error) {
	return f.fases, nil
}

func (f *fakePhaseControlProvider) RegisterProjectPhase(fase *db.FaseProyecto) (int64, error) {
	f.nextID++
	copia := *fase
	copia.ID = f.nextID
	f.fases = append(f.fases, &copia)
	return copia.ID, nil
}

func (f *fakePhaseControlProvider) GetProjectPhase(id int64) (*db.FaseProyecto, error) {
	for _, item := range f.fases {
		if item != nil && item.ID == id {
			return item, nil
		}
	}
	return nil, nil
}

func (f *fakePhaseControlProvider) UpdateProjectPhase(fase *db.FaseProyecto) error {
	for i, item := range f.fases {
		if item != nil && item.ID == fase.ID {
			copia := *fase
			f.fases[i] = &copia
			return nil
		}
	}
	return nil
}

func (f fakeReviewGateProvider) List(input reviewapp.ListInput) ([]*reviewapp.Gate, error) {
	return f.items, nil
}

func (f *fakeReviewGateManager) Create(input reviewapp.CreateGateInput) (*reviewapp.Gate, error) {
	f.created = append(f.created, input)
	id := int64(len(f.created))
	return &reviewapp.Gate{
		ID:             id,
		ProyectoSlug:   input.ProyectoRef,
		TareaID:        input.TareaID,
		ReviewerAgente: input.ReviewerAgente,
		Estado:         reviewapp.GateStatePending,
	}, nil
}

func (f fakeAgentResolver) ResolverAgentePipeline(entrada EntradaResolverAgentePipeline) (string, error) {
	return f.agente, nil
}

func (f *fakePipelineDispatcher) DespacharPipeline(entrada SolicitudDespachoPipeline) (*ResultadoDespachoPipeline, error) {
	f.entrada = entrada
	if f.resultado != nil {
		return f.resultado, nil
	}
	return &ResultadoDespachoPipeline{Estado: "encolado"}, nil
}

type fakePhaseProvider struct {
	fase string
}

func (f fakePhaseProvider) GetActivePhase(proyecto string) (string, error) {
	return f.fase, nil
}

type fakeStorePoolLocal struct {
	pool            *db.PoolCapacidad
	modelos         []*db.PoolModelo
	politicas       []*db.PoliticaModelo
	savePoolCalls   []*db.PoolCapacidad
	saveModelCalls  []string
	savePolicyCalls []*db.PoliticaModelo
}

func (fakeStore) ListPoolsSummary(activo *bool) ([]*db.PoolCapacidadResumen, error) {
	return []*db.PoolCapacidadResumen{{
		Pool:                &db.PoolCapacidad{ID: 1, Slug: "codex", CapacidadTotal: 4, CapacidadReservada: 1},
		SesionesActivas:     2,
		CapacidadDisponible: 1,
	}}, nil
}

func (fakeStore) GetPool(slug string) (*db.PoolCapacidad, error) {
	return &db.PoolCapacidad{ID: 1, Slug: slug, CapacidadTotal: 4, CapacidadReservada: 1}, nil
}

func (fakeStore) ListPoolModels(slug string) ([]*db.PoolModelo, error) {
	return []*db.PoolModelo{{PoolID: 1, ModelSlug: "gpt-5.4"}}, nil
}

func (fakeStore) SavePool(pool *db.PoolCapacidad) (int64, error) { return 1, nil }
func (fakeStore) SavePoolModel(poolSlug string, modelo *db.PoolModelo) (int64, error) {
	return 2, nil
}
func (fakeStore) SeedInitialModels() error { return nil }
func (fakeStore) ListModelPolicies(scopeTipo, scopeRef string, activa *bool) ([]*db.PoliticaModelo, error) {
	return []*db.PoliticaModelo{{ScopeTipo: "global", PerfilTarea: "*"}}, nil
}
func (fakeStore) SaveModelPolicy(policy *db.PoliticaModelo) (int64, error) { return 3, nil }
func (fakeStore) SeedInitialModelPolicies() error                          { return nil }
func (fakeStore) ResolveModelPolicy(input db.ResolverPoliticaInput) (*db.ResolucionModelo, error) {
	return &db.ResolucionModelo{PerfilTarea: "implementacion", PoolSlug: "codex", ModelSlug: "gpt-5.4"}, nil
}

func (f *fakeStorePoolLocal) ListPoolsSummary(activo *bool) ([]*db.PoolCapacidadResumen, error) {
	if f.pool == nil {
		return nil, nil
	}
	return []*db.PoolCapacidadResumen{{
		Pool:                f.pool,
		SesionesActivas:     0,
		CapacidadDisponible: f.pool.CapacidadTotal - f.pool.CapacidadReservada,
	}}, nil
}

func (f *fakeStorePoolLocal) GetPool(slug string) (*db.PoolCapacidad, error) {
	if f.pool == nil || f.pool.Slug != slug {
		return nil, nil
	}
	return f.pool, nil
}

func (f *fakeStorePoolLocal) ListPoolModels(slug string) ([]*db.PoolModelo, error) {
	if f.pool == nil || f.pool.Slug != slug {
		return nil, nil
	}
	return f.modelos, nil
}

func (f *fakeStorePoolLocal) SavePool(pool *db.PoolCapacidad) (int64, error) {
	copia := *pool
	f.savePoolCalls = append(f.savePoolCalls, &copia)
	f.pool = &copia
	f.pool.ID = 7
	return 7, nil
}

func (f *fakeStorePoolLocal) SavePoolModel(poolSlug string, modelo *db.PoolModelo) (int64, error) {
	f.saveModelCalls = append(f.saveModelCalls, poolSlug)
	copia := *modelo
	f.modelos = []*db.PoolModelo{&copia}
	return 8, nil
}

func (f *fakeStorePoolLocal) SeedInitialModels() error { return nil }

func (f *fakeStorePoolLocal) ListModelPolicies(scopeTipo, scopeRef string, activa *bool) ([]*db.PoliticaModelo, error) {
	var out []*db.PoliticaModelo
	for _, item := range f.politicas {
		if item == nil {
			continue
		}
		if item.ScopeTipo == scopeTipo && item.ScopeRef == scopeRef {
			out = append(out, item)
		}
	}
	return out, nil
}

func (f *fakeStorePoolLocal) SaveModelPolicy(policy *db.PoliticaModelo) (int64, error) {
	copia := *policy
	f.savePolicyCalls = append(f.savePolicyCalls, &copia)
	f.politicas = append(f.politicas, &copia)
	return int64(len(f.savePolicyCalls)), nil
}

func (f *fakeStorePoolLocal) SeedInitialModelPolicies() error { return nil }

func (f *fakeStorePoolLocal) ResolveModelPolicy(input db.ResolverPoliticaInput) (*db.ResolucionModelo, error) {
	return nil, nil
}

func TestGetPoolDetail(t *testing.T) {
	service := NewService(fakeStore{})
	detail, err := service.GetPoolDetail("codex")
	if err != nil {
		t.Fatalf("GetPoolDetail: %v", err)
	}
	if detail.Pool.Slug != "codex" || detail.SesionesActivas != 2 || detail.CapacidadDisponible != 1 {
		t.Fatalf("detalle inesperado: %+v", detail)
	}
	if len(detail.Modelos) != 1 || detail.Modelos[0].ModelSlug != "gpt-5.4" {
		t.Fatalf("modelos inesperados: %+v", detail.Modelos)
	}
}

func TestAsegurarPoolLocalCompartidoProvisionaGemmaConPerfilesCanonicos(t *testing.T) {
	store := &fakeStorePoolLocal{}
	service := NewService(store)

	pool, err := service.AsegurarPoolLocalCompartido(EntradaAsegurarPoolLocalCompartido{
		PoolSlug:         "ollama-gemma4",
		ModeloPreferente: "gemma4:26b",
		SlotsMaximos:     1,
	})
	if err != nil {
		t.Fatalf("AsegurarPoolLocalCompartido: %v", err)
	}
	if pool == nil {
		t.Fatalf("pool local nil")
	}
	if pool.PoolSlug != "ollama-gemma4" || pool.ModeloPreferente != "gemma4:26b" || pool.SlotsMaximos != 1 {
		t.Fatalf("pool local inesperado: %+v", pool)
	}
	if pool.ConectorCanonico != "ollama_pool_local" || pool.ConectorCompatibilidad != "ollama-cli" {
		t.Fatalf("conectores inesperados: %+v", pool)
	}
	if len(store.savePoolCalls) != 1 {
		t.Fatalf("se esperaba un save pool, got=%d", len(store.savePoolCalls))
	}
	if got := store.savePoolCalls[0].Runtime; got != "ollama" {
		t.Fatalf("runtime de pool inesperado: %s", got)
	}
	if len(store.savePolicyCalls) != 3 {
		t.Fatalf("se esperaban 3 politicas, got=%d", len(store.savePolicyCalls))
	}
	perfiles := []string{
		store.savePolicyCalls[0].PerfilTarea,
		store.savePolicyCalls[1].PerfilTarea,
		store.savePolicyCalls[2].PerfilTarea,
	}
	want := []string{"analisis", "implementacion", "revision"}
	for i := range want {
		if perfiles[i] != want[i] {
			t.Fatalf("perfiles inesperados: got=%v want=%v", perfiles, want)
		}
	}
}

func TestDescribirPoolLocalCompartidoLeeMetadataYPoliticas(t *testing.T) {
	store := &fakeStorePoolLocal{
		pool: &db.PoolCapacidad{
			ID:             9,
			Slug:           "ollama-gemma4",
			Proveedor:      "Ollama",
			Runtime:        "ollama",
			CapacidadTotal: 1,
			MetadataJSON:   `{"slots_maximos":1,"conector_canonico":"ollama_pool_local","conector_compatibilidad":"ollama-cli","experimental_compat":true,"modelo_preferente":"gemma4:26b"}`,
		},
		modelos: []*db.PoolModelo{
			{PoolID: 9, ModelSlug: "gemma4:26b", Activo: true},
		},
		politicas: []*db.PoliticaModelo{
			{ScopeTipo: "perfil", ScopeRef: "implementacion", PerfilTarea: "implementacion", PoolSlug: "ollama-gemma4", ModelSlug: "gemma4:26b", ReasoningEffort: "high", Prioridad: 10},
			{ScopeTipo: "perfil", ScopeRef: "revision", PerfilTarea: "revision", PoolSlug: "ollama-gemma4", ModelSlug: "gemma4:26b", ReasoningEffort: "high", Prioridad: 10},
			{ScopeTipo: "perfil", ScopeRef: "analisis", PerfilTarea: "analisis", PoolSlug: "ollama-gemma4", ModelSlug: "gemma4:26b", ReasoningEffort: "high", Prioridad: 10},
		},
	}
	service := NewService(store)
	service.SetPoolLocalProvider(fakePoolLocalProvider{telemetria: &TelemetriaPoolLocal{
		PoolSlug:               "ollama-gemma4",
		SlotsMaximos:           1,
		SlotsActivos:           1,
		SesionesLogicasActivas: 1,
		SesionesReady:          1,
	}})

	pool, err := service.DescribirPoolLocalCompartido("ollama-gemma4")
	if err != nil {
		t.Fatalf("DescribirPoolLocalCompartido: %v", err)
	}
	if pool == nil {
		t.Fatalf("pool local nil")
	}
	if pool.SlotsMaximos != 1 || pool.ModeloPreferente != "gemma4:26b" {
		t.Fatalf("metadata inesperada: %+v", pool)
	}
	if len(pool.Perfiles) != 3 {
		t.Fatalf("perfiles inesperados: %+v", pool.Perfiles)
	}
	if pool.Telemetria == nil || pool.Telemetria.SlotsActivos != 1 || pool.Telemetria.SesionesReady != 1 {
		t.Fatalf("telemetria inesperada: %+v", pool.Telemetria)
	}
}

func TestConstruirPipelineLocalDeterministaDefineFasesYRevisionEscalonada(t *testing.T) {
	service := NewService(fakeStore{})

	pipeline, err := service.ConstruirPipelineLocalDeterminista("orquestador")
	if err != nil {
		t.Fatalf("ConstruirPipelineLocalDeterminista: %v", err)
	}
	if pipeline == nil {
		t.Fatalf("pipeline nil")
	}
	if pipeline.ProyectoSlug != "orquestador" {
		t.Fatalf("proyecto inesperado: %q", pipeline.ProyectoSlug)
	}
	if len(pipeline.Fases) != 5 {
		t.Fatalf("numero de fases inesperado: %d", len(pipeline.Fases))
	}
	if pipeline.Fases[0].Fase != "especificacion" || pipeline.Fases[0].ObjetivoModelo != "qwen3.5:27b-q4_K_M" {
		t.Fatalf("fase especificacion inesperada: %+v", pipeline.Fases[0])
	}
	if pipeline.Fases[0].Carril != "premium_worktree" || !pipeline.Fases[0].RequiereWorktree || pipeline.Fases[0].UsaMicroprograma {
		t.Fatalf("carril de especificacion inesperado: %+v", pipeline.Fases[0])
	}
	if pipeline.Fases[1].Fase != "implementacion" || pipeline.Fases[1].ObjetivoModelo != "qwen2.5-coder:32b" {
		t.Fatalf("fase implementacion inesperada: %+v", pipeline.Fases[1])
	}
	if pipeline.Fases[1].Carril != "premium_worktree" || pipeline.Fases[1].EntregaCanonica != "git_worktree" {
		t.Fatalf("carril de implementacion inesperado: %+v", pipeline.Fases[1])
	}
	if pipeline.Fases[2].Carril != "revision_diff" || pipeline.Fases[2].EntregaCanonica != "hallazgos_estructurados" {
		t.Fatalf("carril de revision inesperado: %+v", pipeline.Fases[2])
	}
	if pipeline.Fases[4].Fase != "integracion" || pipeline.Fases[4].RequiereModelo {
		t.Fatalf("fase integracion inesperada: %+v", pipeline.Fases[4])
	}
	if pipeline.Fases[4].Carril != "determinista_app" || pipeline.Fases[4].RequiereWorktree || pipeline.Fases[4].UsaMicroprograma {
		t.Fatalf("carril de integracion inesperado: %+v", pipeline.Fases[4])
	}
	if pipeline.Revision == nil {
		t.Fatalf("revision nil")
	}
	if len(pipeline.Revision.Revisores) != 3 {
		t.Fatalf("numero de revisores inesperado: %d", len(pipeline.Revision.Revisores))
	}
	if pipeline.Revision.Revisores[0].ObjetivoModelo != "deepseek-coder-v2" {
		t.Fatalf("revisor base inesperado: %+v", pipeline.Revision.Revisores[0])
	}
	if pipeline.Revision.Revisores[0].Carril != "revision_diff" || !pipeline.Revision.Revisores[0].RequiereWorktree {
		t.Fatalf("carril de revisor base inesperado: %+v", pipeline.Revision.Revisores[0])
	}
	if !pipeline.Revision.Revisores[2].Premium || pipeline.Revision.Revisores[2].ObjetivoModelo != "gpt-5.4" {
		t.Fatalf("revisor premium inesperado: %+v", pipeline.Revision.Revisores[2])
	}
	if pipeline.Revision.Revisores[2].Carril != "revision_diff" || pipeline.Revision.Revisores[2].EntregaCanonica != "hallazgos_estructurados" {
		t.Fatalf("carril de revisor premium inesperado: %+v", pipeline.Revision.Revisores[2])
	}
	if len(pipeline.Revision.AbrirSegundaOpinionCuando) == 0 {
		t.Fatalf("faltan gates de segunda opinion")
	}
}

func TestCalcularSiguientePasoPipelineLocalDeterministaArrancaPrimeraFaseSiNoHayActiva(t *testing.T) {
	service := NewService(fakeStore{})
	service.SetPhaseProvider(fakePhaseProvider{fase: ""})
	service.SetTaskProvider(fakeTaskProvider{tareas: []*db.Tarea{
		{ID: 9, Titulo: "Implementar adaptador", Estado: db.TareaLibre, Prioridad: db.PrioridadAlta},
	}})

	paso, err := service.CalcularSiguientePasoPipelineLocalDeterminista("orquestador")
	if err != nil {
		t.Fatalf("CalcularSiguientePasoPipelineLocalDeterminista: %v", err)
	}
	if paso == nil {
		t.Fatalf("paso nil")
	}
	if paso.Accion != "arrancar_fase" || paso.FaseObjetivo != "especificacion" {
		t.Fatalf("paso inesperado: %+v", paso)
	}
	if paso.TareaObjetivo == nil || paso.TareaObjetivo.ID != 9 || paso.AccionTarea != "especificar" {
		t.Fatalf("tarea objetivo inesperada: %+v", paso)
	}
}

func TestCalcularSiguientePasoPipelineLocalDeterministaAbreCorreccionPorGateBloqueante(t *testing.T) {
	service := NewService(fakeStore{})
	service.SetPhaseProvider(fakePhaseProvider{fase: "revision"})
	service.SetReviewGateProvider(fakeReviewGateProvider{items: []*reviewapp.Gate{
		{ID: 7, Estado: reviewapp.GateStateChangesAsked, ProyectoSlug: "orquestador"},
	}})

	paso, err := service.CalcularSiguientePasoPipelineLocalDeterminista("orquestador")
	if err != nil {
		t.Fatalf("CalcularSiguientePasoPipelineLocalDeterminista: %v", err)
	}
	if paso == nil || paso.Accion != "abrir_correccion" || paso.FaseObjetivo != "correccion" {
		t.Fatalf("paso inesperado: %+v", paso)
	}
	if paso.GateBloqueante == nil || paso.GateBloqueante.ID != 7 {
		t.Fatalf("gate bloqueante inesperada: %+v", paso.GateBloqueante)
	}
}

func TestCalcularSiguientePasoPipelineLocalDeterministaProponeIntegrarEnFaseIntegracion(t *testing.T) {
	service := NewService(fakeStore{})
	service.SetPhaseProvider(fakePhaseProvider{fase: "integracion"})

	paso, err := service.CalcularSiguientePasoPipelineLocalDeterminista("orquestador")
	if err != nil {
		t.Fatalf("CalcularSiguientePasoPipelineLocalDeterminista: %v", err)
	}
	if paso == nil || paso.Accion != "integrar" || paso.FaseObjetivo != "integracion" {
		t.Fatalf("paso inesperado: %+v", paso)
	}
}

func TestCalcularSiguientePasoPipelineLocalDeterministaSeleccionaTareaEnProgresoParaImplementacion(t *testing.T) {
	service := NewService(fakeStore{})
	service.SetPhaseProvider(fakePhaseProvider{fase: "implementacion"})
	service.SetTaskProvider(fakeTaskProvider{tareas: []*db.Tarea{
		{ID: 1, Titulo: "Libre", Estado: db.TareaLibre, Prioridad: db.PrioridadAlta},
		{ID: 2, Titulo: "En progreso", Estado: db.TareaEnProgreso, Prioridad: db.PrioridadMedia},
	}})

	paso, err := service.CalcularSiguientePasoPipelineLocalDeterminista("orquestador")
	if err != nil {
		t.Fatalf("CalcularSiguientePasoPipelineLocalDeterminista: %v", err)
	}
	if paso.TareaObjetivo == nil || paso.TareaObjetivo.ID != 2 || paso.AccionTarea != "implementar" {
		t.Fatalf("seleccion de tarea inesperada: %+v", paso)
	}
}

func TestCalcularSiguientePasoPipelineLocalDeterministaPriorizaTrabajoAbiertoSobreHistoricoCompletado(t *testing.T) {
	service := NewService(fakeStore{})
	service.SetPhaseProvider(fakePhaseProvider{fase: "implementacion"})
	service.SetTaskProvider(fakeTaskProvider{tareas: []*db.Tarea{
		{ID: 12, Titulo: "Historico completado", Estado: db.TareaCompletada, Prioridad: db.PrioridadAlta},
		{ID: 13, Titulo: "Siguiente frente libre", Estado: db.TareaLibre, Prioridad: db.PrioridadAlta},
	}})

	paso, err := service.CalcularSiguientePasoPipelineLocalDeterminista("orquestador")
	if err != nil {
		t.Fatalf("CalcularSiguientePasoPipelineLocalDeterminista: %v", err)
	}
	if paso == nil || paso.TareaObjetivo == nil {
		t.Fatalf("paso inesperado: %+v", paso)
	}
	if paso.TareaObjetivo.ID != 13 || paso.Accion != "ejecutar_fase" || paso.AccionTarea != "implementar" {
		t.Fatalf("deberia priorizar el frente abierto antes que historico completado: %+v", paso)
	}
}

func TestCalcularSiguientePasoPipelineLocalDeterministaPriorizaMicrocicloActivoSobreTrabajoAjeno(t *testing.T) {
	service := NewService(fakeStore{})
	service.SetPhaseProvider(fakePhaseProvider{fase: "implementacion"})
	service.SetTaskProvider(fakeTaskProvider{tareas: []*db.Tarea{
		{ID: 40, Titulo: "Trabajo ajeno", Estado: db.TareaEnProgreso, Prioridad: db.PrioridadAlta},
		{ID: 41, Titulo: "Frente premium activo", Estado: db.TareaEnProgreso, Prioridad: db.PrioridadAlta, Notas: "autonomia:microrefactor_loop"},
	}})

	paso, err := service.CalcularSiguientePasoPipelineLocalDeterminista("orquestador")
	if err != nil {
		t.Fatalf("CalcularSiguientePasoPipelineLocalDeterminista: %v", err)
	}
	if paso == nil || paso.TareaObjetivo == nil {
		t.Fatalf("paso inesperado: %+v", paso)
	}
	if paso.TareaObjetivo.ID != 41 || paso.AccionTarea != "implementar" {
		t.Fatalf("deberia priorizar el frente microciclo activo sobre trabajo ajeno: %+v", paso)
	}
}

func TestCalcularSiguientePasoPipelineLocalDeterministaNoPriorizaMicrocicloLibreSobreFrenteMayorActivo(t *testing.T) {
	service := NewService(fakeStore{})
	service.SetPhaseProvider(fakePhaseProvider{fase: "implementacion"})
	service.SetTaskProvider(fakeTaskProvider{tareas: []*db.Tarea{
		{ID: 40, Titulo: "Frente mayor activo", Estado: db.TareaEnProgreso, Prioridad: db.PrioridadAlta, Notas: "autonomia:premium_frontier"},
		{ID: 41, Titulo: "Microciclo libre viejo", Estado: db.TareaLibre, Prioridad: db.PrioridadAlta, Notas: "autonomia:microrefactor_loop"},
	}})

	paso, err := service.CalcularSiguientePasoPipelineLocalDeterminista("orquestador")
	if err != nil {
		t.Fatalf("CalcularSiguientePasoPipelineLocalDeterminista: %v", err)
	}
	if paso == nil || paso.TareaObjetivo == nil {
		t.Fatalf("paso inesperado: %+v", paso)
	}
	if paso.TareaObjetivo.ID != 40 || paso.AccionTarea != "implementar" {
		t.Fatalf("un microciclo libre no deberia secuestrar el carril frente a un frente mayor activo: %+v", paso)
	}
}

func TestCalcularSiguientePasoPipelineLocalDeterministaPriorizaFrenteAcotadoSobreSemillaPremiumActiva(t *testing.T) {
	service := NewService(fakeStore{})
	service.SetPhaseProvider(fakePhaseProvider{fase: "implementacion"})
	service.SetTaskProvider(fakeTaskProvider{tareas: []*db.Tarea{
		{ID: 40, Titulo: "Autonomía premium: abrir siguiente frente mayor útil", Estado: db.TareaEnProgreso, Prioridad: db.PrioridadAlta, Notas: "autonomia:premium_frontier"},
		{
			ID:          41,
			Titulo:      "Cerrar reconcile de mailbox premium",
			Estado:      db.TareaLibre,
			Prioridad:   db.PrioridadAlta,
			Descripcion: "Cerrar el siguiente frente premium util.\nWRITE_SET: cmd/controlplane_support.go, cmd/controlplane_support_test.go\nTests minimos: go test ./cmd -run 'TestProcesarRuntimeMailboxSessionResumeBatch.*'",
		},
	}})

	paso, err := service.CalcularSiguientePasoPipelineLocalDeterminista("orquestador")
	if err != nil {
		t.Fatalf("CalcularSiguientePasoPipelineLocalDeterminista: %v", err)
	}
	if paso == nil || paso.TareaObjetivo == nil {
		t.Fatalf("paso inesperado: %+v", paso)
	}
	if paso.TareaObjetivo.ID != 41 || paso.AccionTarea != "implementar" {
		t.Fatalf("el frente premium acotado deberia ganar sobre la semilla premium activa: %+v", paso)
	}
}

func TestCalcularSiguientePasoPipelineLocalDeterministaPriorizaLibreSobreAsignadaAjena(t *testing.T) {
	service := NewService(fakeStore{})
	service.SetPhaseProvider(fakePhaseProvider{fase: "implementacion"})
	service.SetTaskProvider(fakeTaskProvider{tareas: []*db.Tarea{
		{ID: 50, Titulo: "Reservada por otro agente", Estado: db.TareaAsignada, Agente: ptrString("Claude2"), Prioridad: db.PrioridadAlta, Notas: "autonomia:microrefactor_loop"},
		{ID: 51, Titulo: "Frente libre util", Estado: db.TareaLibre, Prioridad: db.PrioridadAlta, Notas: "autonomia:microrefactor_loop"},
	}})

	paso, err := service.CalcularSiguientePasoPipelineLocalDeterminista("orquestador")
	if err != nil {
		t.Fatalf("CalcularSiguientePasoPipelineLocalDeterminista: %v", err)
	}
	if paso == nil || paso.TareaObjetivo == nil {
		t.Fatalf("paso inesperado: %+v", paso)
	}
	if paso.TareaObjetivo.ID != 51 || paso.AccionTarea != "implementar" {
		t.Fatalf("deberia priorizar tarea libre sobre asignada ajena: %+v", paso)
	}
}

func TestCalcularSiguientePasoPipelineLocalDeterministaSeleccionaTareaCompletadaParaIntegracion(t *testing.T) {
	service := NewService(fakeStore{})
	service.SetPhaseProvider(fakePhaseProvider{fase: "integracion"})
	service.SetTaskProvider(fakeTaskProvider{tareas: []*db.Tarea{
		{ID: 3, Titulo: "Completada", Estado: db.TareaCompletada, Prioridad: db.PrioridadAlta},
		{ID: 4, Titulo: "Bloqueada", Estado: db.TareaBloqueada, Prioridad: db.PrioridadAlta},
	}})

	paso, err := service.CalcularSiguientePasoPipelineLocalDeterminista("orquestador")
	if err != nil {
		t.Fatalf("CalcularSiguientePasoPipelineLocalDeterminista: %v", err)
	}
	if paso.TareaObjetivo == nil || paso.TareaObjetivo.ID != 3 || paso.AccionTarea != "integrar" {
		t.Fatalf("seleccion de tarea inesperada: %+v", paso)
	}
}

func TestCalcularSiguientePasoPipelineLocalDeterministaNoSigueRevisionHistoricaSiHayTrabajoAbierto(t *testing.T) {
	service := NewService(fakeStore{})
	service.SetPhaseProvider(fakePhaseProvider{fase: "revision"})
	service.SetTaskProvider(fakeTaskProvider{tareas: []*db.Tarea{
		{ID: 5, Titulo: "En progreso", Estado: db.TareaEnProgreso, Prioridad: db.PrioridadAlta},
		{ID: 6, Titulo: "Completada", Estado: db.TareaCompletada, Prioridad: db.PrioridadMedia},
	}})

	paso, err := service.CalcularSiguientePasoPipelineLocalDeterminista("orquestador")
	if err != nil {
		t.Fatalf("CalcularSiguientePasoPipelineLocalDeterminista: %v", err)
	}
	if paso == nil || paso.TareaObjetivo == nil {
		t.Fatalf("paso inesperado: %+v", paso)
	}
	if paso.Accion != "reconciliar_fase" || paso.FaseObjetivo != "implementacion" || paso.TareaObjetivo.ID != 5 || paso.AccionTarea != "implementar" {
		t.Fatalf("si hay trabajo abierto el pipeline no deberia seguir revision historica: %+v", paso)
	}
}

func TestCalcularSiguientePasoPipelineLocalDeterministaReconciliarFaseDesconocidaLlevaEspecificacion(t *testing.T) {
	service := NewService(fakeStore{})
	service.SetPhaseProvider(fakePhaseProvider{fase: "fase_inexistente"})
	service.SetTaskProvider(fakeTaskProvider{tareas: []*db.Tarea{
		{
			ID:          101,
			Titulo:      "Implementar scheduler",
			Estado:      db.TareaLibre,
			Prioridad:   db.PrioridadAlta,
			Descripcion: "Write-set exclusivo: capacidadapp/pipeline_local.go. Tests minimos: go test ./capacidadapp/...",
		},
	}})

	paso, err := service.CalcularSiguientePasoPipelineLocalDeterminista("orquestador")
	if err != nil {
		t.Fatalf("CalcularSiguientePasoPipelineLocalDeterminista: %v", err)
	}
	if paso == nil || paso.Accion != "reconciliar_fase" {
		t.Fatalf("esperaba reconciliar_fase, got: %+v", paso)
	}
	if paso.Especificacion == nil {
		t.Fatalf("reconciliar_fase a fase desconocida debe llevar especificacion")
	}
	if paso.Especificacion.Encabezado != "Implementar scheduler" {
		t.Fatalf("encabezado inesperado: %q", paso.Especificacion.Encabezado)
	}
}

func TestCalcularSiguientePasoPipelineLocalDeterministaReconciliarVuelveImplementacionConEspecificacion(t *testing.T) {
	service := NewService(fakeStore{})
	service.SetPhaseProvider(fakePhaseProvider{fase: "revision"})
	service.SetTaskProvider(fakeTaskProvider{tareas: []*db.Tarea{
		{
			ID:          102,
			Titulo:      "Cerrar mailbox",
			Estado:      db.TareaEnProgreso,
			Prioridad:   db.PrioridadAlta,
			Descripcion: "Write-set exclusivo: cmd/controlplane_support.go. Tests minimos: go test ./cmd -run TestX",
		},
	}})

	paso, err := service.CalcularSiguientePasoPipelineLocalDeterminista("orquestador")
	if err != nil {
		t.Fatalf("CalcularSiguientePasoPipelineLocalDeterminista: %v", err)
	}
	if paso == nil || paso.Accion != "reconciliar_fase" || paso.FaseObjetivo != "implementacion" {
		t.Fatalf("esperaba reconciliar_fase a implementacion, got: %+v", paso)
	}
	if paso.Especificacion == nil {
		t.Fatalf("reconciliar_fase a implementacion debe llevar especificacion")
	}
	if paso.Especificacion.Encabezado != "Cerrar mailbox" {
		t.Fatalf("encabezado inesperado: %q", paso.Especificacion.Encabezado)
	}
	if len(paso.Especificacion.WriteSet) != 1 || paso.Especificacion.WriteSet[0] != "cmd/controlplane_support.go" {
		t.Fatalf("write_set inesperado: %+v", paso.Especificacion.WriteSet)
	}
}

func TestCalcularSiguientePasoPipelineLocalDeterministaArranqueLlevaEspecificacion(t *testing.T) {
	service := NewService(fakeStore{})
	service.SetPhaseProvider(fakePhaseProvider{fase: ""})
	service.SetTaskProvider(fakeTaskProvider{tareas: []*db.Tarea{
		{
			ID:          95,
			Titulo:      "Especificar pipeline entrega",
			Estado:      db.TareaLibre,
			Prioridad:   db.PrioridadAlta,
			Descripcion: "Write-set exclusivo: capacidadapp/pipeline_local.go. Tests minimos: go test ./capacidadapp/...",
		},
	}})

	paso, err := service.CalcularSiguientePasoPipelineLocalDeterminista("orquestador")
	if err != nil {
		t.Fatalf("CalcularSiguientePasoPipelineLocalDeterminista: %v", err)
	}
	if paso == nil || paso.Accion != "arrancar_fase" || paso.FaseObjetivo != "especificacion" {
		t.Fatalf("paso inesperado: %+v", paso)
	}
	if paso.Especificacion == nil {
		t.Fatalf("especificacion nil en arranque de fase: el paso debe llevar el contrato")
	}
	if paso.Especificacion.Encabezado != "Especificar pipeline entrega" {
		t.Fatalf("encabezado inesperado: %q", paso.Especificacion.Encabezado)
	}
	if paso.Especificacion.SalidaEsperada != "git_worktree" {
		t.Fatalf("salida_esperada inesperada: %q", paso.Especificacion.SalidaEsperada)
	}
	if len(paso.Especificacion.WriteSet) != 1 || paso.Especificacion.WriteSet[0] != "capacidadapp/pipeline_local.go" {
		t.Fatalf("write_set inesperado: %+v", paso.Especificacion.WriteSet)
	}
}

func TestCalcularSiguientePasoPipelineLocalDeterministaAvanzaAImplementacionCuandoEspecificacionCompleta(t *testing.T) {
	service := NewService(fakeStore{})
	service.SetPhaseProvider(fakePhaseProvider{fase: "especificacion"})
	service.SetTaskProvider(fakeTaskProvider{tareas: []*db.Tarea{
		{ID: 80, Titulo: "Especificación cerrada", Estado: db.TareaCompletada, Prioridad: db.PrioridadAlta},
	}})

	paso, err := service.CalcularSiguientePasoPipelineLocalDeterminista("orquestador")
	if err != nil {
		t.Fatalf("CalcularSiguientePasoPipelineLocalDeterminista: %v", err)
	}
	if paso == nil || paso.Accion != "avanzar_fase" || paso.FaseObjetivo != "implementacion" {
		t.Fatalf("esperaba avanzar_fase a implementacion, got: %+v", paso)
	}
	if paso.AccionTarea != "implementar" {
		t.Fatalf("accion_tarea inesperada: %q", paso.AccionTarea)
	}
	if paso.FaseActual != "especificacion" {
		t.Fatalf("fase_actual inesperada: %q", paso.FaseActual)
	}
}

func TestCalcularSiguientePasoPipelineLocalDeterministaAvanzaARevisionCuandoImplementacionCompleta(t *testing.T) {
	service := NewService(fakeStore{})
	service.SetPhaseProvider(fakePhaseProvider{fase: "implementacion"})
	service.SetTaskProvider(fakeTaskProvider{tareas: []*db.Tarea{
		{ID: 12, Titulo: "Implementado", Estado: db.TareaCompletada, Prioridad: db.PrioridadAlta},
	}})

	paso, err := service.CalcularSiguientePasoPipelineLocalDeterminista("orquestador")
	if err != nil {
		t.Fatalf("CalcularSiguientePasoPipelineLocalDeterminista: %v", err)
	}
	if paso == nil || paso.Accion != "avanzar_fase" || paso.FaseObjetivo != "revision" {
		t.Fatalf("paso inesperado: %+v", paso)
	}
}

func TestCalcularSiguientePasoPipelineLocalDeterministaAvanzaAIntegracionConGateAprobada(t *testing.T) {
	service := NewService(fakeStore{})
	service.SetPhaseProvider(fakePhaseProvider{fase: "revision"})
	service.SetTaskProvider(fakeTaskProvider{tareas: []*db.Tarea{
		{ID: 21, Titulo: "Revisado", Estado: db.TareaCompletada, Prioridad: db.PrioridadAlta},
	}})
	service.SetReviewGateProvider(fakeReviewGateProvider{items: []*reviewapp.Gate{
		{ID: 9, ProyectoSlug: "orquestador", TareaID: ptrInt64(21), Estado: reviewapp.GateStateApproved},
	}})

	paso, err := service.CalcularSiguientePasoPipelineLocalDeterminista("orquestador")
	if err != nil {
		t.Fatalf("CalcularSiguientePasoPipelineLocalDeterminista: %v", err)
	}
	if paso == nil || paso.Accion != "avanzar_fase" || paso.FaseObjetivo != "integracion" {
		t.Fatalf("paso inesperado: %+v", paso)
	}
}

func TestCalcularSiguientePasoPipelineLocalDeterministaVuelveAImplementacionSiRevisionTieneTrabajoAbierto(t *testing.T) {
	service := NewService(fakeStore{})
	service.SetPhaseProvider(fakePhaseProvider{fase: "revision"})
	service.SetTaskProvider(fakeTaskProvider{tareas: []*db.Tarea{
		{ID: 30, Titulo: "Siguiente frente libre", Estado: db.TareaLibre, Prioridad: db.PrioridadAlta},
		{ID: 31, Titulo: "Revision historica", Estado: db.TareaCompletada, Prioridad: db.PrioridadMedia},
	}})

	paso, err := service.CalcularSiguientePasoPipelineLocalDeterminista("orquestador")
	if err != nil {
		t.Fatalf("CalcularSiguientePasoPipelineLocalDeterminista: %v", err)
	}
	if paso == nil {
		t.Fatalf("paso nil")
	}
	if paso.Accion != "reconciliar_fase" || paso.FaseObjetivo != "implementacion" {
		t.Fatalf("deberia volver a implementacion si hay trabajo abierto: %+v", paso)
	}
	if paso.TareaObjetivo == nil || paso.TareaObjetivo.ID != 30 || paso.AccionTarea != "implementar" {
		t.Fatalf("tarea objetivo inesperada al volver a implementacion: %+v", paso)
	}
}

func TestEjecutarSiguientePasoPipelineLocalDeterministaActivaFaseYArrancaTarea(t *testing.T) {
	service := NewService(fakeStore{})
	service.SetPhaseProvider(fakePhaseProvider{fase: ""})
	service.SetTaskProvider(fakeTaskProvider{tareas: []*db.Tarea{
		{ID: 11, Titulo: "Preparar backlog", Estado: db.TareaLibre, Prioridad: db.PrioridadAlta},
	}})
	service.SetTaskActionProvider(&fakeTaskActionProvider{
		tareas: map[int64]*db.Tarea{
			11: {ID: 11, Titulo: "Preparar backlog", Estado: db.TareaLibre, Prioridad: db.PrioridadAlta},
		},
	})
	service.SetPhaseControlProvider(&fakePhaseControlProvider{})
	service.SetAgentResolver(fakeAgentResolver{agente: "Codex1"})

	resultado, err := service.EjecutarSiguientePasoPipelineLocalDeterminista("orquestador")
	if err != nil {
		t.Fatalf("EjecutarSiguientePasoPipelineLocalDeterminista: %v", err)
	}
	if resultado == nil || resultado.Paso == nil {
		t.Fatalf("resultado nil")
	}
	if resultado.FaseActivada == nil || *resultado.FaseActivada != "especificacion" {
		t.Fatalf("fase activada inesperada: %+v", resultado)
	}
	if resultado.TareaActualizada == nil || resultado.TareaActualizada.ID != 11 || resultado.TareaActualizada.Estado != string(db.TareaEnProgreso) {
		t.Fatalf("tarea actualizada inesperada: %+v", resultado)
	}
	if resultado.Despacho == nil {
		t.Fatalf("despacho nil")
	}
	if resultado.Despacho.Carril != "premium_worktree" || resultado.Despacho.EntregaCanonica != "git_worktree" {
		t.Fatalf("despacho inesperado: %+v", resultado.Despacho)
	}
	if resultado.Despacho.AccionTarea != "especificar" || resultado.Despacho.AgenteSugerido != "Codex1" {
		t.Fatalf("agente sugerido inesperado: %+v", resultado.Despacho)
	}
	if resultado.Despacho.TareaObjetivoID != 11 || resultado.Despacho.TareaObjetivo != "Preparar backlog" {
		t.Fatalf("tarea de despacho inesperada: %+v", resultado.Despacho)
	}
	if resultado.Despacho.AgenteTarea != "Codex1" {
		t.Fatalf("agente tarea inesperado: %+v", resultado.Despacho)
	}
}

func TestEjecutarSiguientePasoPipelineLocalDeterministaReasignaFrenteMicrocicloAlAgenteSugerido(t *testing.T) {
	service := NewService(fakeStore{})
	service.SetPhaseProvider(fakePhaseProvider{fase: "implementacion"})
	service.SetTaskProvider(fakeTaskProvider{tareas: []*db.Tarea{
		{ID: 61, Titulo: "Frente microciclo", Estado: db.TareaEnProgreso, Agente: ptrString("Claude2"), Prioridad: db.PrioridadAlta, Notas: "autonomia:microrefactor_loop"},
	}})
	service.SetTaskActionProvider(&fakeTaskActionProvider{
		tareas: map[int64]*db.Tarea{
			61: {ID: 61, Titulo: "Frente microciclo", Estado: db.TareaEnProgreso, Agente: ptrString("Claude2"), Prioridad: db.PrioridadAlta, Notas: "autonomia:microrefactor_loop"},
		},
	})
	service.SetAgentResolver(fakeAgentResolver{agente: "Codex1"})

	resultado, err := service.EjecutarSiguientePasoPipelineLocalDeterminista("orquestador")
	if err != nil {
		t.Fatalf("EjecutarSiguientePasoPipelineLocalDeterminista: %v", err)
	}
	if resultado == nil || resultado.TareaActualizada == nil || resultado.Despacho == nil {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
	if resultado.TareaActualizada.Agente != "Codex1" || resultado.TareaActualizada.Estado != string(db.TareaEnProgreso) {
		t.Fatalf("deberia reasignar el frente microciclo al agente sugerido: %+v", resultado.TareaActualizada)
	}
	if resultado.Despacho.AgenteSugerido != "Codex1" || resultado.Despacho.AgenteTarea != "Codex1" {
		t.Fatalf("despacho deberia reflejar la reasignacion al agente sugerido: %+v", resultado.Despacho)
	}
}

func TestEjecutarSiguientePasoPipelineLocalDeterministaConservaAgenteAsignadoSiNoHaySugerencia(t *testing.T) {
	service := NewService(fakeStore{})
	service.SetPhaseProvider(fakePhaseProvider{fase: "implementacion"})
	service.SetTaskProvider(fakeTaskProvider{tareas: []*db.Tarea{
		{ID: 62, Titulo: "Frente premium", Estado: db.TareaAsignada, Agente: ptrString("Gemini1"), Prioridad: db.PrioridadAlta, Notas: "autonomia:premium_frontier"},
	}})
	service.SetTaskActionProvider(&fakeTaskActionProvider{
		tareas: map[int64]*db.Tarea{
			62: {ID: 62, Titulo: "Frente premium", Estado: db.TareaAsignada, Agente: ptrString("Gemini1"), Prioridad: db.PrioridadAlta, Notas: "autonomia:premium_frontier"},
		},
	})
	service.SetAgentResolver(fakeAgentResolver{})

	resultado, err := service.EjecutarSiguientePasoPipelineLocalDeterminista("orquestador")
	if err != nil {
		t.Fatalf("EjecutarSiguientePasoPipelineLocalDeterminista: %v", err)
	}
	if resultado == nil || resultado.TareaActualizada == nil || resultado.Despacho == nil {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
	if resultado.TareaActualizada.Agente != "Gemini1" || resultado.TareaActualizada.Estado != string(db.TareaEnProgreso) {
		t.Fatalf("deberia conservar el agente asignado al arrancar la tarea: %+v", resultado.TareaActualizada)
	}
	if resultado.Despacho.AgenteTarea != "Gemini1" {
		t.Fatalf("despacho deberia reflejar el agente actual de la tarea: %+v", resultado.Despacho)
	}
}

func TestIntentarAutoasignarTareaPipelineLocalAsignaAAgenteConcreto(t *testing.T) {
	service := NewService(fakeStore{})
	service.SetPhaseProvider(fakePhaseProvider{fase: "implementacion"})
	service.SetTaskProvider(fakeTaskProvider{tareas: []*db.Tarea{
		{ID: 51, Titulo: "Implementar scheduler", Estado: db.TareaLibre, Prioridad: db.PrioridadAlta},
	}})
	service.SetTaskActionProvider(&fakeTaskActionProvider{
		tareas: map[int64]*db.Tarea{
			51: {ID: 51, Titulo: "Implementar scheduler", Estado: db.TareaLibre, Prioridad: db.PrioridadAlta},
		},
	})

	tarea, err := service.IntentarAutoasignarTareaPipelineLocal("orquestador", "Codex1")
	if err != nil {
		t.Fatalf("IntentarAutoasignarTareaPipelineLocal: %v", err)
	}
	if tarea == nil || tarea.ID != 51 || tarea.Estado != string(db.TareaEnProgreso) {
		t.Fatalf("tarea autoasignada inesperada: %+v", tarea)
	}
	if tarea.Agente != "Codex1" {
		t.Fatalf("agente autoasignado inesperado: %+v", tarea)
	}
}

func TestIntentarAutoasignarTareaPipelineLocalPromueveAsignadaDelMismoAgente(t *testing.T) {
	service := NewService(fakeStore{})
	service.SetPhaseProvider(fakePhaseProvider{fase: "implementacion"})
	service.SetTaskProvider(fakeTaskProvider{tareas: []*db.Tarea{
		{ID: 55, Titulo: "Implementar promotion runtime", Estado: db.TareaAsignada, Agente: ptrString("Codex1"), Prioridad: db.PrioridadAlta},
	}})
	service.SetTaskActionProvider(&fakeTaskActionProvider{
		tareas: map[int64]*db.Tarea{
			55: {ID: 55, Titulo: "Implementar promotion runtime", Estado: db.TareaAsignada, Agente: ptrString("Codex1"), Prioridad: db.PrioridadAlta},
		},
	})

	tarea, err := service.IntentarAutoasignarTareaPipelineLocal("orquestador", "Codex1")
	if err != nil {
		t.Fatalf("IntentarAutoasignarTareaPipelineLocal: %v", err)
	}
	if tarea == nil || tarea.ID != 55 || tarea.Estado != string(db.TareaEnProgreso) || tarea.Agente != "Codex1" {
		t.Fatalf("deberia promover la tarea ya asignada al mismo agente: %+v", tarea)
	}
}

func TestIntentarAutoasignarTareaPipelineLocalNoReasignaTrabajoAjeno(t *testing.T) {
	service := NewService(fakeStore{})
	service.SetPhaseProvider(fakePhaseProvider{fase: "implementacion"})
	service.SetTaskProvider(fakeTaskProvider{tareas: []*db.Tarea{
		{ID: 52, Titulo: "Implementar parser", Estado: db.TareaAsignada, Agente: ptrString("Claude1"), Prioridad: db.PrioridadAlta},
	}})
	service.SetTaskActionProvider(&fakeTaskActionProvider{
		tareas: map[int64]*db.Tarea{
			52: {ID: 52, Titulo: "Implementar parser", Estado: db.TareaAsignada, Agente: ptrString("Claude1"), Prioridad: db.PrioridadAlta},
		},
	})

	tarea, err := service.IntentarAutoasignarTareaPipelineLocal("orquestador", "Codex1")
	if err != nil {
		t.Fatalf("IntentarAutoasignarTareaPipelineLocal: %v", err)
	}
	if tarea != nil {
		t.Fatalf("no debería reasignar trabajo ajeno: %+v", tarea)
	}
}

func TestIntentarAutoasignarTareaPipelineLocalRecuperaFrentePremiumEnOrquesta(t *testing.T) {
	service := NewService(fakeStore{})
	service.SetPhaseProvider(fakePhaseProvider{fase: "implementacion"})
	service.SetTaskProvider(fakeTaskProvider{tareas: []*db.Tarea{
		{
			ID:          56,
			Titulo:      "Runtime mailbox/session_resume",
			Estado:      db.TareaEnProgreso,
			Agente:      ptrString("orquesta"),
			Prioridad:   db.PrioridadAlta,
			Descripcion: "Write-set exclusivo: cmd/controlplane_support.go\nTests minimos: go test ./cmd -run 'TestProcesarRuntimeMailboxSessionResumeBatch.*' -count=1",
		},
	}})
	service.SetTaskActionProvider(&fakeTaskActionProvider{
		tareas: map[int64]*db.Tarea{
			56: {
				ID:          56,
				Titulo:      "Runtime mailbox/session_resume",
				Estado:      db.TareaEnProgreso,
				Agente:      ptrString("orquesta"),
				Prioridad:   db.PrioridadAlta,
				Descripcion: "Write-set exclusivo: cmd/controlplane_support.go\nTests minimos: go test ./cmd -run 'TestProcesarRuntimeMailboxSessionResumeBatch.*' -count=1",
			},
		},
	})

	tarea, err := service.IntentarAutoasignarTareaPipelineLocal("orquestador", "Gemini1")
	if err != nil {
		t.Fatalf("IntentarAutoasignarTareaPipelineLocal: %v", err)
	}
	if tarea == nil || tarea.ID != 56 || tarea.Estado != string(db.TareaEnProgreso) || tarea.Agente != "Gemini1" {
		t.Fatalf("deberia recuperar el frente premium acotado desde orquesta: %+v", tarea)
	}
}

func TestIntentarAutoasignarTareaPipelineLocalIgnoraFrentePremiumSinContrato(t *testing.T) {
	service := NewService(fakeStore{})
	service.SetPhaseProvider(fakePhaseProvider{fase: "implementacion"})
	service.SetTaskProvider(fakeTaskProvider{tareas: []*db.Tarea{
		{ID: 53, Titulo: "Auditoria server-first CLI/API/web y transporte real", Estado: db.TareaLibre, Prioridad: db.PrioridadAlta},
	}})
	service.SetTaskActionProvider(&fakeTaskActionProvider{
		tareas: map[int64]*db.Tarea{
			53: {ID: 53, Titulo: "Auditoria server-first CLI/API/web y transporte real", Estado: db.TareaLibre, Prioridad: db.PrioridadAlta},
		},
	})

	tarea, err := service.IntentarAutoasignarTareaPipelineLocal("orquestador", "Codex1")
	if err != nil {
		t.Fatalf("IntentarAutoasignarTareaPipelineLocal: %v", err)
	}
	if tarea != nil {
		t.Fatalf("no deberia autoasignar un frente premium amplio sin write_set/tests: %+v", tarea)
	}
}

func TestIntentarAutoasignarTareaPipelineLocalPermitePremiumFrontierCanonico(t *testing.T) {
	service := NewService(fakeStore{})
	service.SetPhaseProvider(fakePhaseProvider{fase: "implementacion"})
	service.SetTaskProvider(fakeTaskProvider{tareas: []*db.Tarea{
		{ID: 54, Titulo: "Autonomía premium: abrir siguiente frente mayor útil", Estado: db.TareaLibre, Prioridad: db.PrioridadAlta, Notas: "autonomia:premium_frontier"},
	}})
	service.SetTaskActionProvider(&fakeTaskActionProvider{
		tareas: map[int64]*db.Tarea{
			54: {ID: 54, Titulo: "Autonomía premium: abrir siguiente frente mayor útil", Estado: db.TareaLibre, Prioridad: db.PrioridadAlta, Notas: "autonomia:premium_frontier"},
		},
	})

	tarea, err := service.IntentarAutoasignarTareaPipelineLocal("orquestador", "Codex1")
	if err != nil {
		t.Fatalf("IntentarAutoasignarTareaPipelineLocal: %v", err)
	}
	if tarea == nil || tarea.ID != 54 || tarea.Estado != string(db.TareaEnProgreso) || tarea.Agente != "Codex1" {
		t.Fatalf("deberia permitir el frente premium canónico acotador: %+v", tarea)
	}
}

func TestIntentarAutoasignarTareaPipelineLocalRecuperaMicrofrenteLibreSiElPasoGlobalNoAplica(t *testing.T) {
	service := NewService(fakeStore{})
	service.SetPhaseProvider(fakePhaseProvider{fase: "implementacion"})
	service.SetTaskProvider(fakeTaskProvider{tareas: []*db.Tarea{
		{ID: 70, Titulo: "Auditoria server-first CLI/API/web y transporte real", Estado: db.TareaLibre, Prioridad: db.PrioridadAlta},
		{
			ID:          71,
			Titulo:      "Runtime mailbox/session_resume",
			Estado:      db.TareaLibre,
			Prioridad:   db.PrioridadAlta,
			Descripcion: "Write-set exclusivo: cmd/controlplane_support.go\nTests minimos: go test ./cmd -run 'TestProcesarRuntimeMailboxSessionResumeBatch.*' -count=1",
			Notas:       "autonomia:microrefactor_loop",
		},
	}})
	service.SetTaskActionProvider(&fakeTaskActionProvider{
		tareas: map[int64]*db.Tarea{
			70: {ID: 70, Titulo: "Auditoria server-first CLI/API/web y transporte real", Estado: db.TareaLibre, Prioridad: db.PrioridadAlta},
			71: {
				ID:          71,
				Titulo:      "Runtime mailbox/session_resume",
				Estado:      db.TareaLibre,
				Prioridad:   db.PrioridadAlta,
				Descripcion: "Write-set exclusivo: cmd/controlplane_support.go\nTests minimos: go test ./cmd -run 'TestProcesarRuntimeMailboxSessionResumeBatch.*' -count=1",
				Notas:       "autonomia:microrefactor_loop",
			},
		},
	})

	tarea, err := service.IntentarAutoasignarTareaPipelineLocal("orquestador", "Gemini1")
	if err != nil {
		t.Fatalf("IntentarAutoasignarTareaPipelineLocal: %v", err)
	}
	if tarea == nil || tarea.ID != 71 || tarea.Estado != string(db.TareaEnProgreso) || tarea.Agente != "Gemini1" {
		t.Fatalf("deberia tomar el microfrente premium acotado libre cuando el paso global no aplica: %+v", tarea)
	}
}

func TestConstruirDespachoPipelineLocalUsaCarrilRevision(t *testing.T) {
	paso := &PasoPipelineLocalDeterminista{
		ProyectoSlug: "orquestador",
		AccionTarea:  "revisar",
		Motivo:       "hay una review gate pendiente",
		EtapaObjetivo: &EtapaPipelineLocal{
			Fase:             "revision",
			PerfilTarea:      "revision",
			ModoEjecucion:    "worker_modelo",
			Carril:           "revision_diff",
			EntregaCanonica:  "hallazgos_estructurados",
			RequiereWorktree: true,
			RequiereModelo:   true,
			ObjetivoModelo:   "deepseek-coder-v2",
			ModeloFallback:   "gemma4:26b",
		},
		TareaObjetivo: &TareaPipelineLocal{ID: 22, Titulo: "Revisar control plane"},
	}

	service := NewService(fakeStore{})
	service.SetAgentResolver(fakeAgentResolver{agente: "Claude1"})
	despacho := service.construirDespachoPipelineLocal(paso, nil)
	if despacho == nil {
		t.Fatalf("despacho nil")
	}
	if despacho.Carril != "revision_diff" || despacho.AgenteSugerido != "Claude1" {
		t.Fatalf("despacho revision inesperado: %+v", despacho)
	}
	if despacho.EntregaCanonica != "hallazgos_estructurados" || !despacho.RequiereWorktree {
		t.Fatalf("entrega revision inesperada: %+v", despacho)
	}
	if despacho.TareaObjetivoID != 22 || despacho.TareaObjetivo != "Revisar control plane" {
		t.Fatalf("tarea revision inesperada: %+v", despacho)
	}
}

func TestConstruirDespachoPipelineLocalPropagaContratoDelFrente(t *testing.T) {
	paso := &PasoPipelineLocalDeterminista{
		ProyectoSlug: "orquestador",
		AccionTarea:  "implementar",
		Motivo:       "frente activo",
		EtapaObjetivo: &EtapaPipelineLocal{
			Fase:             "implementacion",
			PerfilTarea:      "implementacion",
			ModoEjecucion:    "worker_modelo",
			Carril:           "premium_worktree",
			EntregaCanonica:  "git_worktree",
			RequiereWorktree: true,
		},
		TareaObjetivo: &TareaPipelineLocal{
			ID:           33,
			Titulo:       "Cerrar runtime mailbox",
			Descripcion:  "Write-set exclusivo: cmd/controlplane_support.go, db/controlplane_entities.go y db/controlplane_entities_test.go.",
			WriteSet:     []string{"cmd/controlplane_support.go", "db/controlplane_entities.go", "db/controlplane_entities_test.go"},
			SimbolosFoco: "procesarRuntimeMailboxSessionResumeBatchConMailbox, resolverBootstrapRuntimeLeasePendiente",
			TestsMinimos: "go test ./cmd -run 'TestProcesarRuntimeMailboxSessionResumeBatch.*'",
		},
	}

	service := NewService(fakeStore{})
	despacho := service.construirDespachoPipelineLocal(paso, nil)
	if despacho == nil {
		t.Fatalf("despacho nil")
	}
	if len(despacho.WriteSet) != 3 {
		t.Fatalf("write_set inesperado: %+v", despacho)
	}
	if despacho.WriteSet[0] != "cmd/controlplane_support.go" || despacho.WriteSet[2] != "db/controlplane_entities_test.go" {
		t.Fatalf("write_set propagado inesperado: %+v", despacho.WriteSet)
	}
	if despacho.SimbolosFoco != "procesarRuntimeMailboxSessionResumeBatchConMailbox, resolverBootstrapRuntimeLeasePendiente" {
		t.Fatalf("simbolos foco inesperados: %+v", despacho)
	}
	if despacho.TestsMinimos != "go test ./cmd -run 'TestProcesarRuntimeMailboxSessionResumeBatch.*'" {
		t.Fatalf("tests minimos inesperados: %+v", despacho)
	}
}

func TestTareaPipelineLocalDesdeDBExtraeContratoDelFrente(t *testing.T) {
	tarea := tareaPipelineLocalDesdeDB(&db.Tarea{
		ID:          77,
		Titulo:      "Cerrar mailbox premium",
		Descripcion: "Frente actual. Simbolos foco: procesarRuntimeMailboxSessionResumeBatchConMailbox y resolverBootstrapRuntimeLeasePendiente. Write-set exclusivo: cmd/controlplane_support.go, db/controlplane_entities.go y db/controlplane_entities_test.go. Trabaja solo dentro de ese write_set. Tests minimos del slice: go test ./cmd -run 'TestProcesarRuntimeMailboxSessionResumeBatch.*' -count=1.",
		Estado:      db.TareaEnProgreso,
	})
	if tarea == nil {
		t.Fatalf("tarea nil")
	}
	if len(tarea.WriteSet) != 3 {
		t.Fatalf("write_set inesperado: %+v", tarea)
	}
	if tarea.WriteSet[0] != "cmd/controlplane_support.go" || tarea.WriteSet[2] != "db/controlplane_entities_test.go" {
		t.Fatalf("write_set extraido inesperado: %+v", tarea.WriteSet)
	}
	if tarea.SimbolosFoco != "procesarRuntimeMailboxSessionResumeBatchConMailbox y resolverBootstrapRuntimeLeasePendiente" {
		t.Fatalf("simbolos foco extraidos inesperados: %+v", tarea)
	}
	if tarea.TestsMinimos != "go test ./cmd -run 'TestProcesarRuntimeMailboxSessionResumeBatch.*'" {
		t.Fatalf("tests minimos extraidos inesperados: %+v", tarea)
	}
}

func TestEjecutarYDespacharSiguientePasoPipelineLocalDeterministaEncolaResultado(t *testing.T) {
	service := NewService(fakeStore{})
	service.SetPhaseProvider(fakePhaseProvider{fase: ""})
	service.SetTaskProvider(fakeTaskProvider{tareas: []*db.Tarea{
		{ID: 31, Titulo: "Corregir parser", Estado: db.TareaLibre, Prioridad: db.PrioridadAlta},
	}})
	service.SetTaskActionProvider(&fakeTaskActionProvider{
		tareas: map[int64]*db.Tarea{
			31: {ID: 31, Titulo: "Corregir parser", Estado: db.TareaLibre, Prioridad: db.PrioridadAlta},
		},
	})
	service.SetPhaseControlProvider(&fakePhaseControlProvider{})
	service.SetAgentResolver(fakeAgentResolver{agente: "Codex1"})
	dispatcher := &fakePipelineDispatcher{
		resultado: &ResultadoDespachoPipeline{Estado: "encolado"},
	}
	service.SetPipelineDispatcher(dispatcher)

	resultado, err := service.EjecutarYDespacharSiguientePasoPipelineLocalDeterminista("orquestador")
	if err != nil {
		t.Fatalf("EjecutarYDespacharSiguientePasoPipelineLocalDeterminista: %v", err)
	}
	if resultado == nil || resultado.DispatchRuntime == nil || resultado.DispatchRuntime.Estado != "encolado" {
		t.Fatalf("dispatch runtime inesperado: %+v", resultado)
	}
	if dispatcher.entrada.Despacho == nil || dispatcher.entrada.Despacho.AgenteSugerido != "Codex1" {
		t.Fatalf("entrada de dispatcher inesperada: %+v", dispatcher.entrada)
	}
}

func TestEjecutarSiguientePasoPipelineLocalDeterministaCreaGateRevisionSiNoExiste(t *testing.T) {
	service := NewService(fakeStore{})
	service.SetPhaseProvider(fakePhaseProvider{fase: "revision"})
	service.SetTaskProvider(fakeTaskProvider{tareas: []*db.Tarea{
		{ID: 41, Titulo: "Revisar diff", Estado: db.TareaCompletada, Prioridad: db.PrioridadAlta},
	}})
	service.SetReviewGateProvider(fakeReviewGateProvider{})
	manager := &fakeReviewGateManager{}
	service.SetReviewGateManager(manager)
	service.SetAgentResolver(fakeAgentResolver{agente: "Claude1"})

	resultado, err := service.EjecutarSiguientePasoPipelineLocalDeterminista("orquestador")
	if err != nil {
		t.Fatalf("EjecutarSiguientePasoPipelineLocalDeterminista: %v", err)
	}
	if resultado == nil || resultado.Despacho == nil {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
	if len(manager.created) != 1 {
		t.Fatalf("gates creadas=%d, want=1", len(manager.created))
	}
	if manager.created[0].TareaID == nil || *manager.created[0].TareaID != 41 {
		t.Fatalf("gate creada inesperada: %+v", manager.created[0])
	}
	// Con RevisorObjetivo presente, se usa el NombreRol del revisor en lugar del AgenteSugerido genérico.
	if manager.created[0].ReviewerAgente != "revisor_base_local" {
		t.Fatalf("reviewer inesperado: %+v", manager.created[0])
	}
	if manager.created[0].SeverityMax != "medium" {
		t.Fatalf("severity inesperada para revisor base: %+v", manager.created[0])
	}
}

func TestResumenEspecificacionGateIncluyeContrato(t *testing.T) {
	spec := &EspecificacionFuncion{
		Encabezado:     "Cerrar session_resume",
		SalidaEsperada: "git_worktree",
		WriteSet:       []string{"cmd/controlplane_support.go", "db/controlplane_entities.go"},
		TestsMinimos:   "go test ./cmd -run 'TestProcesarRuntimeMailboxSessionResumeBatch.*'",
	}
	resumen := resumenEspecificacionGate(spec)
	if resumen == "" {
		t.Fatalf("resumen nil para spec completa")
	}
	if !strings.Contains(resumen, "encabezado: Cerrar session_resume") {
		t.Fatalf("encabezado ausente: %q", resumen)
	}
	if !strings.Contains(resumen, "write_set: cmd/controlplane_support.go, db/controlplane_entities.go") {
		t.Fatalf("write_set ausente: %q", resumen)
	}
	if !strings.Contains(resumen, "tests_minimos:") {
		t.Fatalf("tests_minimos ausente: %q", resumen)
	}
}

func TestResumenEspecificacionGateVacioSiNilSpec(t *testing.T) {
	if got := resumenEspecificacionGate(nil); got != "" {
		t.Fatalf("esperaba vacío para spec nil, got: %q", got)
	}
}

func TestEjecutarSiguientePasoPipelineLocalDeterministaGateLlevaResumenEspecificacion(t *testing.T) {
	service := NewService(fakeStore{})
	service.SetPhaseProvider(fakePhaseProvider{fase: "revision"})
	service.SetTaskProvider(fakeTaskProvider{tareas: []*db.Tarea{
		{
			ID:          99,
			Titulo:      "Cerrar mailbox premium",
			Estado:      db.TareaCompletada,
			Prioridad:   db.PrioridadAlta,
			Descripcion: "Write-set exclusivo: cmd/controlplane_support.go. Tests minimos: go test ./cmd -run TestX",
		},
	}})
	service.SetReviewGateProvider(fakeReviewGateProvider{})
	manager := &fakeReviewGateManager{}
	service.SetReviewGateManager(manager)
	service.SetAgentResolver(fakeAgentResolver{agente: "Claude1"})

	resultado, err := service.EjecutarSiguientePasoPipelineLocalDeterminista("orquestador")
	if err != nil {
		t.Fatalf("EjecutarSiguientePasoPipelineLocalDeterminista: %v", err)
	}
	if resultado == nil || resultado.Despacho == nil {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
	if len(manager.created) != 1 {
		t.Fatalf("gates creadas=%d, want=1", len(manager.created))
	}
	findings := manager.created[0].InitialFindings
	if findings == "" {
		t.Fatalf("gate creada sin initial_findings: el revisor necesita el contrato de la especificación")
	}
	if !strings.Contains(findings, "Cerrar mailbox premium") {
		t.Fatalf("findings no incluye encabezado de la especificacion: %q", findings)
	}
	if !strings.Contains(findings, "cmd/controlplane_support.go") {
		t.Fatalf("findings no incluye write_set: %q", findings)
	}
}

func TestEjecutarSiguientePasoPipelineLocalDeterministaCreaGatePremiumConSeveridadAlta(t *testing.T) {
	service := NewService(fakeStore{})
	service.SetPhaseProvider(fakePhaseProvider{fase: "implementacion"})
	service.SetTaskProvider(fakeTaskProvider{tareas: []*db.Tarea{
		{
			ID:          98,
			Titulo:      "Cerrar race en controlplane",
			Estado:      db.TareaCompletada,
			Prioridad:   db.PrioridadAlta,
			Descripcion: "Write-set exclusivo: cmd/controlplane_support.go, cmd/controlplane_support_test.go. Tests minimos: go test ./cmd -run TestX",
		},
	}})
	service.SetReviewGateProvider(fakeReviewGateProvider{})
	manager := &fakeReviewGateManager{}
	service.SetReviewGateManager(manager)
	service.SetAgentResolver(fakeAgentResolver{agente: "Claude1"})
	service.SetTaskActionProvider(&fakeTaskActionProvider{
		tareas: map[int64]*db.Tarea{
			98: {
				ID:          98,
				Titulo:      "Cerrar race en controlplane",
				Estado:      db.TareaCompletada,
				Prioridad:   db.PrioridadAlta,
				Descripcion: "Write-set exclusivo: cmd/controlplane_support.go, cmd/controlplane_support_test.go. Tests minimos: go test ./cmd -run TestX",
			},
		},
	})
	service.SetPhaseControlProvider(&fakePhaseControlProvider{})

	resultado, err := service.EjecutarSiguientePasoPipelineLocalDeterminista("orquestador")
	if err != nil {
		t.Fatalf("EjecutarSiguientePasoPipelineLocalDeterminista: %v", err)
	}
	if resultado == nil || resultado.Despacho == nil {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
	if resultado.Despacho.Fase != "revision" {
		t.Fatalf("fase esperada revision, got: %q", resultado.Despacho.Fase)
	}
	if len(manager.created) != 1 {
		t.Fatalf("gates creadas=%d, want=1", len(manager.created))
	}
	// zona_critica_control_plane → revisor premium → severity high
	if manager.created[0].ReviewerAgente != "segunda_opinion_premium" {
		t.Fatalf("revisor premium esperado, got: %q", manager.created[0].ReviewerAgente)
	}
	if manager.created[0].SeverityMax != "high" {
		t.Fatalf("severity alta esperada para revisor premium, got: %q", manager.created[0].SeverityMax)
	}
}

func TestSeleccionarRevisorEscalonadoPipelineEscalaAPremiumPorZonaCritica(t *testing.T) {
	service := NewService(fakeStore{})
	pipeline, err := service.ConstruirPipelineLocalDeterminista("orquestador")
	if err != nil {
		t.Fatalf("ConstruirPipelineLocalDeterminista: %v", err)
	}
	tarea := &TareaPipelineLocal{
		ID:       90,
		Titulo:   "Cerrar race condition",
		WriteSet: []string{"cmd/controlplane_support.go", "cmd/controlplane_support_test.go"},
	}
	señales := extraerSeñalesTareaPipelineLocal(tarea)
	revisor := seleccionarRevisorEscalonadoPipeline(pipeline.Revision, señales)
	if revisor == nil {
		t.Fatalf("revisor nil")
	}
	if !revisor.Premium {
		t.Fatalf("zona_critica_control_plane debería escalar a revisor premium; got=%+v", revisor)
	}
}

func TestSeleccionarRevisorEscalonadoPipelineUsaBaseParaTareaSimple(t *testing.T) {
	service := NewService(fakeStore{})
	pipeline, err := service.ConstruirPipelineLocalDeterminista("orquestador")
	if err != nil {
		t.Fatalf("ConstruirPipelineLocalDeterminista: %v", err)
	}
	tarea := &TareaPipelineLocal{
		ID:       91,
		Titulo:   "Ajustar timeout conexion",
		WriteSet: []string{"internal/net/timeout.go"},
	}
	señales := extraerSeñalesTareaPipelineLocal(tarea)
	revisor := seleccionarRevisorEscalonadoPipeline(pipeline.Revision, señales)
	if revisor == nil {
		t.Fatalf("revisor nil")
	}
	if revisor.Premium {
		t.Fatalf("tarea simple debería usar revisor base; got=%+v", revisor)
	}
}

func TestCalcularSiguientePasoPipelineLocalDeterministaEscalaRevisorPorZonaCritica(t *testing.T) {
	service := NewService(fakeStore{})
	service.SetPhaseProvider(fakePhaseProvider{fase: "implementacion"})
	service.SetTaskProvider(fakeTaskProvider{tareas: []*db.Tarea{
		{
			ID:          92,
			Titulo:      "Cerrar mailbox premium",
			Estado:      db.TareaCompletada,
			Prioridad:   db.PrioridadAlta,
			Descripcion: "Write-set exclusivo: cmd/controlplane_support.go, cmd/controlplane_support_test.go. Tests minimos: go test ./cmd -run 'TestX.*'",
		},
	}})

	paso, err := service.CalcularSiguientePasoPipelineLocalDeterminista("orquestador")
	if err != nil {
		t.Fatalf("CalcularSiguientePasoPipelineLocalDeterminista: %v", err)
	}
	if paso == nil || paso.Accion != "avanzar_fase" || paso.FaseObjetivo != "revision" {
		t.Fatalf("paso inesperado: %+v", paso)
	}
	if paso.RevisorObjetivo == nil {
		t.Fatalf("revisor_objetivo nil en avance a revision")
	}
	if !paso.RevisorObjetivo.Premium {
		t.Fatalf("zona_critica_control_plane deberia escalar a revisor premium; got=%+v", paso.RevisorObjetivo)
	}
}

func TestExtraerSeñalesTareaPipelineLocalZonaCriticaPorModulo(t *testing.T) {
	tarea := &TareaPipelineLocal{
		ID:     110,
		Titulo: "Cerrar race condition",
		Modulo: "control_plane",
	}
	señales := extraerSeñalesTareaPipelineLocal(tarea)
	found := false
	for _, s := range señales {
		if s == "zona_critica_control_plane" {
			found = true
		}
	}
	if !found {
		t.Fatalf("modulo control_plane debe disparar zona_critica_control_plane; got=%v", señales)
	}
}

func TestExtraerSeñalesTareaPipelineLocalCorreccionesRepetidas(t *testing.T) {
	tarea := &TareaPipelineLocal{
		ID:    111,
		Titulo: "Cerrar mailbox",
		Notas: "correcciones_repetidas; el agente ya fue corregido dos veces en este frente",
	}
	señales := extraerSeñalesTareaPipelineLocal(tarea)
	found := false
	for _, s := range señales {
		if s == "correcciones_repetidas" {
			found = true
		}
	}
	if !found {
		t.Fatalf("notas con correcciones_repetidas deben disparar la señal; got=%v", señales)
	}
}

func TestExtraerSeñalesTareaPipelineLocalHallazgosContradictorios(t *testing.T) {
	tarea := &TareaPipelineLocal{
		ID:          112,
		Titulo:      "Revisar scheduler",
		Descripcion: "hallazgos_contradictorios entre revisor1 y revisor2",
	}
	señales := extraerSeñalesTareaPipelineLocal(tarea)
	found := false
	for _, s := range señales {
		if s == "hallazgos_contradictorios" {
			found = true
		}
	}
	if !found {
		t.Fatalf("descripcion con hallazgos_contradictorios debe disparar la señal; got=%v", señales)
	}
}

func TestExtraerSeñalesTareaPipelineLocalCicloCorreccionEscalaSegundaOpinion(t *testing.T) {
	service := NewService(fakeStore{})
	pipeline, err := service.ConstruirPipelineLocalDeterminista("orquestador")
	if err != nil {
		t.Fatalf("ConstruirPipelineLocalDeterminista: %v", err)
	}
	tarea := &TareaPipelineLocal{
		ID:    113,
		Titulo: "Cerrar carril premium",
		Notas: "ciclo_correccion:2 — agente ya aplicó dos rondas sin convergencia",
	}
	señales := extraerSeñalesTareaPipelineLocal(tarea)
	revisor := seleccionarRevisorEscalonadoPipeline(pipeline.Revision, señales)
	if revisor == nil {
		t.Fatalf("revisor nil")
	}
	// correcciones_repetidas → segunda_opinion (no premium en primera instancia)
	if !strings.Contains(revisor.NombreRol, "segunda_opinion") {
		t.Fatalf("ciclo_correccion debe escalar a segunda_opinion; got=%q", revisor.NombreRol)
	}
}

func TestConstruirDespachoPipelineLocalIncludeEspecificacionFuncion(t *testing.T) {
	paso := &PasoPipelineLocalDeterminista{
		ProyectoSlug: "orquestador",
		AccionTarea:  "implementar",
		Motivo:       "frente activo",
		EtapaObjetivo: &EtapaPipelineLocal{
			Fase:            "implementacion",
			PerfilTarea:     "implementacion",
			ModoEjecucion:   "worker_modelo",
			Carril:          "premium_worktree",
			EntregaCanonica: "git_worktree",
			RequiereModelo:  true,
		},
		TareaObjetivo: &TareaPipelineLocal{
			ID:           44,
			Titulo:       "Cerrar session_resume premium",
			WriteSet:     []string{"cmd/controlplane_support.go", "db/controlplane_entities.go"},
			SimbolosFoco: "procesarRuntimeMailboxSessionResumeBatchConMailbox",
			TestsMinimos: "go test ./cmd -run 'TestProcesarRuntimeMailboxSessionResumeBatch.*'",
		},
	}

	service := NewService(fakeStore{})
	despacho := service.construirDespachoPipelineLocal(paso, nil)
	if despacho == nil {
		t.Fatalf("despacho nil")
	}
	spec := despacho.Especificacion
	if spec == nil {
		t.Fatalf("especificacion nil: el orquestador debe fijar contrato en el despacho")
	}
	if spec.Encabezado != "Cerrar session_resume premium" {
		t.Fatalf("encabezado inesperado: %q", spec.Encabezado)
	}
	if spec.SalidaEsperada != "git_worktree" {
		t.Fatalf("salida_esperada inesperada: %q", spec.SalidaEsperada)
	}
	if len(spec.WriteSet) != 2 || spec.WriteSet[0] != "cmd/controlplane_support.go" {
		t.Fatalf("write_set inesperado: %+v", spec.WriteSet)
	}
	if spec.SimbolosFoco != "procesarRuntimeMailboxSessionResumeBatchConMailbox" {
		t.Fatalf("simbolos_foco inesperados: %q", spec.SimbolosFoco)
	}
	if spec.TestsMinimos != "go test ./cmd -run 'TestProcesarRuntimeMailboxSessionResumeBatch.*'" {
		t.Fatalf("tests_minimos inesperados: %q", spec.TestsMinimos)
	}
}

func TestConstruirDespachoPipelineLocalNoIncludeEspecificacionSiEtapaNoRequiereModelo(t *testing.T) {
	paso := &PasoPipelineLocalDeterminista{
		ProyectoSlug: "orquestador",
		AccionTarea:  "integrar",
		Motivo:       "integracion determinista",
		EtapaObjetivo: &EtapaPipelineLocal{
			Fase:            "integracion",
			PerfilTarea:     "analisis",
			ModoEjecucion:   "determinista_app",
			Carril:          "determinista_app",
			EntregaCanonica: "merge_controlado",
			RequiereModelo:  false,
		},
		TareaObjetivo: &TareaPipelineLocal{
			ID:     45,
			Titulo: "Integrar rama feature",
		},
	}

	service := NewService(fakeStore{})
	despacho := service.construirDespachoPipelineLocal(paso, nil)
	if despacho == nil {
		t.Fatalf("despacho nil")
	}
	if despacho.Especificacion != nil {
		t.Fatalf("la etapa determinista no debe generar especificacion: %+v", despacho.Especificacion)
	}
}

func TestConstruirDespachoPipelineLocalPropagaRevisorObjetivo(t *testing.T) {
	revisor := &RevisorEscalonado{
		NombreRol:       "revisor_base_local",
		Clase:           "local",
		ObjetivoModelo:  "deepseek-coder-v2",
		EntregaCanonica: "hallazgos_estructurados",
	}
	paso := &PasoPipelineLocalDeterminista{
		ProyectoSlug: "orquestador",
		AccionTarea:  "revisar",
		Motivo:       "revision pendiente",
		EtapaObjetivo: &EtapaPipelineLocal{
			Fase:            "revision",
			PerfilTarea:     "revision",
			ModoEjecucion:   "worker_modelo",
			Carril:          "revision_diff",
			EntregaCanonica: "hallazgos_estructurados",
			RequiereModelo:  true,
		},
		TareaObjetivo:   &TareaPipelineLocal{ID: 46, Titulo: "Revisar control plane"},
		RevisorObjetivo: revisor,
	}

	service := NewService(fakeStore{})
	despacho := service.construirDespachoPipelineLocal(paso, nil)
	if despacho == nil {
		t.Fatalf("despacho nil")
	}
	if despacho.RevisorObjetivo == nil {
		t.Fatalf("revisor_objetivo nil: debe propagarse desde el paso")
	}
	if despacho.RevisorObjetivo.NombreRol != "revisor_base_local" {
		t.Fatalf("revisor_objetivo inesperado: %+v", despacho.RevisorObjetivo)
	}
}

func TestListarModelosRuntimeActivosUsaGestorConfigurado(t *testing.T) {
	manager := &fakeRuntimeModelManager{
		items: []ModeloRuntimeActivo{
			{Modelo: "gemma4:26b", Runtime: "ollama"},
			{Modelo: "qwen3.5-coder-local", Runtime: "ollama"},
		},
	}
	service := NewService(fakeStore{})
	service.SetRuntimeModelManager(manager)

	items, err := service.ListarModelosRuntimeActivos()
	if err != nil {
		t.Fatalf("ListarModelosRuntimeActivos: %v", err)
	}
	if len(items) != 2 || items[0].Modelo != "gemma4:26b" || items[1].Modelo != "qwen3.5-coder-local" {
		t.Fatalf("modelos runtime inesperados: %+v", items)
	}
}

func TestDescargarModelosRuntimeActivosDescargaTodosLosActivos(t *testing.T) {
	manager := &fakeRuntimeModelManager{
		items: []ModeloRuntimeActivo{
			{Modelo: "qwen3.5-coder-local", Runtime: "ollama"},
			{Modelo: "gemma4:26b", Runtime: "ollama"},
		},
	}
	service := NewService(fakeStore{})
	service.SetRuntimeModelManager(manager)

	descargados, err := service.DescargarModelosRuntimeActivos()
	if err != nil {
		t.Fatalf("DescargarModelosRuntimeActivos: %v", err)
	}
	if len(descargados) != 2 {
		t.Fatalf("descargados inesperados: %+v", descargados)
	}
	if len(manager.downloaded) != 2 || manager.downloaded[0] != "qwen3.5-coder-local" || manager.downloaded[1] != "gemma4:26b" {
		t.Fatalf("descargas inesperadas: %+v", manager.downloaded)
	}
}

type fakeReviewGateProviderError struct {
	err error
}

func (f fakeReviewGateProviderError) List(input reviewapp.ListInput) ([]*reviewapp.Gate, error) {
	return nil, f.err
}

func TestListarGatesPipelineLocalDevuelveNilEnSqlErrNoRows(t *testing.T) {
	service := NewService(fakeStore{})
	service.SetReviewGateProvider(fakeReviewGateProviderError{err: sql.ErrNoRows})

	items, err := service.listarGatesPipelineLocal("orquestador")
	if err != nil {
		t.Fatalf("sql.ErrNoRows debe tratarse como lista vacía, got err: %v", err)
	}
	if items != nil {
		t.Fatalf("items debe ser nil para sql.ErrNoRows, got: %+v", items)
	}
}

func TestListarGatesPipelineLocalPropagaErrorReal(t *testing.T) {
	errReal := fmt.Errorf("fallo de conexión")
	service := NewService(fakeStore{})
	service.SetReviewGateProvider(fakeReviewGateProviderError{err: errReal})

	_, err := service.listarGatesPipelineLocal("orquestador")
	if err == nil {
		t.Fatal("error real debe propagarse desde listarGatesPipelineLocal")
	}
}

func TestElegirNuevaCandidataTareaPipelineLocalSinCandidataUsaScore(t *testing.T) {
	nueva := &db.Tarea{ID: 1, Titulo: "T1", Estado: db.TareaLibre}
	got, score := elegirNuevaCandidataTareaPipelineLocal(nil, nueva, "implementacion", -1)
	if got != nueva {
		t.Fatal("sin candidata, nueva con score>=0 debe ganar")
	}
	if score != 40 {
		t.Fatalf("score esperado 40 para libre/implementacion, got %d", score)
	}
}

func TestElegirNuevaCandidataTareaPipelineLocalSinCandidataScoreNegativoNoActualiza(t *testing.T) {
	nueva := &db.Tarea{ID: 1, Estado: db.TareaBloqueada}
	got, score := elegirNuevaCandidataTareaPipelineLocal(nil, nueva, "implementacion", -1)
	if got != nil {
		t.Fatalf("tarea bloqueada con score -1 no debe convertirse en candidata, got: %+v", got)
	}
	if score != -1 {
		t.Fatalf("score debe permanecer -1, got %d", score)
	}
}

func TestElegirNuevaCandidataTareaPipelineLocalSemillaVsContratoRealNuevaGana(t *testing.T) {
	// candidata es semilla premium, nueva tiene contrato real → nueva debe ganar
	candidata := &db.Tarea{
		ID:     10,
		Titulo: "Autonomía premium: abrir siguiente frente mayor útil",
		Estado: db.TareaLibre,
	}
	nueva := &db.Tarea{
		ID:          20,
		Titulo:      "Implementar scheduler",
		Estado:      db.TareaLibre,
		Descripcion: "Write-set exclusivo: capacidadapp/pipeline_local.go. Tests minimos: go test ./capacidadapp/...",
	}
	got, _ := elegirNuevaCandidataTareaPipelineLocal(candidata, nueva, "implementacion", 40)
	if got != nueva {
		t.Fatalf("contrato real debe ganar a semilla, got ID=%d", got.ID)
	}
}

func TestElegirNuevaCandidataTareaPipelineLocalSemillaVsContratoRealCandidataSeQueda(t *testing.T) {
	// candidata tiene contrato real, nueva es semilla → candidata se mantiene
	candidata := &db.Tarea{
		ID:          10,
		Titulo:      "Implementar scheduler",
		Estado:      db.TareaLibre,
		Descripcion: "Write-set exclusivo: capacidadapp/pipeline_local.go. Tests minimos: go test ./capacidadapp/...",
	}
	nueva := &db.Tarea{
		ID:     20,
		Titulo: "Autonomía premium: abrir siguiente frente mayor útil",
		Estado: db.TareaLibre,
	}
	got, _ := elegirNuevaCandidataTareaPipelineLocal(candidata, nueva, "implementacion", 40)
	if got != candidata {
		t.Fatalf("candidata con contrato real debe mantenerse frente a semilla, got ID=%d", got.ID)
	}
}

func TestElegirNuevaCandidataTareaPipelineLocalMicrocicloGanaASinMicrociclo(t *testing.T) {
	candidata := &db.Tarea{ID: 5, Estado: db.TareaLibre}
	nueva := &db.Tarea{
		ID:     6,
		Estado: db.TareaEnProgreso,
		Notas:  "autonomia:microrefactor_loop",
	}
	got, _ := elegirNuevaCandidataTareaPipelineLocal(candidata, nueva, "implementacion", 40)
	if got != nueva {
		t.Fatalf("microciclo en progreso debe ganar a candidata sin microciclo, got ID=%d", got.ID)
	}
}

func TestElegirNuevaCandidataTareaPipelineLocalCandidataMicrocicloSeQuedaFrenteANuevaSinMicrociclo(t *testing.T) {
	candidata := &db.Tarea{
		ID:     5,
		Estado: db.TareaEnProgreso,
		Notas:  "autonomia:microrefactor_loop",
	}
	nueva := &db.Tarea{ID: 6, Estado: db.TareaLibre}
	got, _ := elegirNuevaCandidataTareaPipelineLocal(candidata, nueva, "implementacion", 50)
	if got != candidata {
		t.Fatalf("candidata microciclo debe mantenerse frente a nueva sin microciclo, got ID=%d", got.ID)
	}
}

func TestElegirNuevaCandidataTareaPipelineLocalEmpateIDMayorGana(t *testing.T) {
	candidata := &db.Tarea{ID: 3, Estado: db.TareaLibre}
	nueva := &db.Tarea{ID: 7, Estado: db.TareaLibre}
	got, _ := elegirNuevaCandidataTareaPipelineLocal(candidata, nueva, "implementacion", 40)
	if got != nueva {
		t.Fatalf("con igual score, ID mayor debe ganar, got ID=%d", got.ID)
	}
}

func TestElegirNuevaCandidataTareaPipelineLocalScoreMayorGana(t *testing.T) {
	candidata := &db.Tarea{ID: 10, Estado: db.TareaLibre}    // score 40
	nueva := &db.Tarea{ID: 5, Estado: db.TareaEnProgreso}    // score 50
	got, score := elegirNuevaCandidataTareaPipelineLocal(candidata, nueva, "implementacion", 40)
	if got != nueva {
		t.Fatalf("score mayor debe ganar, got ID=%d", got.ID)
	}
	if score != 50 {
		t.Fatalf("score esperado 50, got %d", score)
	}
}
