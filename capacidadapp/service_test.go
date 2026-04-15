package capacidadapp

import (
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
	if manager.created[0].ReviewerAgente != "Claude1" {
		t.Fatalf("reviewer inesperado: %+v", manager.created[0])
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
