package capacidadapp

import (
	"strings"

	"orquesta/db"
	"orquesta/reviewapp"
)

type Store interface {
	ListPoolsSummary(activo *bool) ([]*db.PoolCapacidadResumen, error)
	GetPool(slug string) (*db.PoolCapacidad, error)
	ListPoolModels(slug string) ([]*db.PoolModelo, error)
	SavePool(pool *db.PoolCapacidad) (int64, error)
	SavePoolModel(poolSlug string, modelo *db.PoolModelo) (int64, error)
	SeedInitialModels() error
	ListModelPolicies(scopeTipo, scopeRef string, activa *bool) ([]*db.PoliticaModelo, error)
	SaveModelPolicy(policy *db.PoliticaModelo) (int64, error)
	SeedInitialModelPolicies() error
	ResolveModelPolicy(input db.ResolverPoliticaInput) (*db.ResolucionModelo, error)
}

type PhaseProvider interface {
	GetActivePhase(proyecto string) (string, error)
}

type PoolLocalProvider interface {
	DescribirPoolLocalCompartido(slug string) (*TelemetriaPoolLocal, error)
}

type TaskProvider interface {
	ListProjectTasks(proyecto string) ([]*db.Tarea, error)
}

type TaskActionProvider interface {
	GetTask(id int64) (*db.Tarea, error)
	TakeTask(id int64, agente string) error
	StartTask(id int64, agente string) error
}

type ReviewGateManager interface {
	Create(input reviewapp.CreateGateInput) (*reviewapp.Gate, error)
}

type PhaseControlProvider interface {
	ListProjectPhases(proyecto string) ([]*db.FaseProyecto, error)
	RegisterProjectPhase(fase *db.FaseProyecto) (int64, error)
	GetProjectPhase(id int64) (*db.FaseProyecto, error)
	UpdateProjectPhase(fase *db.FaseProyecto) error
}

type EntradaResolverAgentePipeline struct {
	ProyectoSlug     string
	Fase             string
	AccionTarea      string
	PerfilTarea      string
	Carril           string
	ObjetivoModelo   string
	ModeloFallback   string
	RequiereWorktree bool
	UsaMicroprograma bool
}

type ResolvedorAgentePipeline interface {
	ResolverAgentePipeline(entrada EntradaResolverAgentePipeline) (string, error)
}

type SolicitudDespachoPipeline struct {
	ProyectoSlug string
	Despacho     *DespachoPipelineLocal
}

type ResultadoDespachoPipeline struct {
	Estado         string `json:"estado"`
	Motivo         string `json:"motivo,omitempty"`
	StartOrderID   *int64 `json:"start_order_id,omitempty"`
	RuntimeOrderID *int64 `json:"runtime_order_id,omitempty"`
}

type DespachadorPipeline interface {
	DespacharPipeline(entrada SolicitudDespachoPipeline) (*ResultadoDespachoPipeline, error)
}

type Service struct {
	store                Store
	phaseProvider        PhaseProvider
	poolLocalProvider    PoolLocalProvider
	reviewGateProvider   ReviewGateProvider
	reviewGateManager    ReviewGateManager
	runtimeModelManager  GestorRuntimeModelos
	taskProvider         TaskProvider
	taskActionProvider   TaskActionProvider
	phaseControlProvider PhaseControlProvider
	agentResolver        ResolvedorAgentePipeline
	pipelineDispatcher   DespachadorPipeline
}

type PoolDetail struct {
	Pool                *db.PoolCapacidad
	SesionesActivas     int
	CapacidadDisponible int
	Modelos             []*db.PoolModelo
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) SetPhaseProvider(provider PhaseProvider) {
	s.phaseProvider = provider
}

func (s *Service) SetPoolLocalProvider(provider PoolLocalProvider) {
	s.poolLocalProvider = provider
}

func (s *Service) SetReviewGateManager(manager ReviewGateManager) {
	s.reviewGateManager = manager
}

func (s *Service) SetTaskProvider(provider TaskProvider) {
	s.taskProvider = provider
}

func (s *Service) SetTaskActionProvider(provider TaskActionProvider) {
	s.taskActionProvider = provider
}

func (s *Service) IntentarAutoasignarTareaPipelineLocal(proyectoSlug, agente string) (*TareaPipelineLocal, error) {
	if s == nil || s.taskActionProvider == nil {
		return nil, nil
	}
	proyectoSlug = strings.TrimSpace(proyectoSlug)
	agente = strings.TrimSpace(agente)
	if proyectoSlug == "" || agente == "" {
		return nil, nil
	}
	paso, err := s.CalcularSiguientePasoPipelineLocalDeterminista(proyectoSlug)
	if err != nil || paso == nil || paso.TareaObjetivo == nil {
		return nil, err
	}
	switch strings.ToLower(strings.TrimSpace(paso.AccionTarea)) {
	case "especificar", "implementar", "corregir", "revisar":
	default:
		return nil, nil
	}
	actual, err := s.taskActionProvider.GetTask(paso.TareaObjetivo.ID)
	if err != nil {
		return nil, err
	}
	if actual == nil {
		return nil, nil
	}
	cambioReal := false
	switch actual.Estado {
	case db.TareaLibre, db.TareaBacklog:
		if err := s.taskActionProvider.TakeTask(actual.ID, agente); err != nil {
			return nil, err
		}
		cambioReal = true
	}
	actual, err = s.taskActionProvider.GetTask(actual.ID)
	if err != nil {
		return nil, err
	}
	if actual == nil {
		return nil, nil
	}
	if !cambioReal {
		return nil, nil
	}
	actual, err = s.taskActionProvider.GetTask(actual.ID)
	if err != nil {
		return nil, err
	}
	return tareaPipelineLocalDesdeDB(actual), nil
}

func (s *Service) SetPhaseControlProvider(provider PhaseControlProvider) {
	s.phaseControlProvider = provider
}

func (s *Service) SetAgentResolver(resolver ResolvedorAgentePipeline) {
	s.agentResolver = resolver
}

func (s *Service) SetPipelineDispatcher(dispatcher DespachadorPipeline) {
	s.pipelineDispatcher = dispatcher
}

func (s *Service) DespacharPipelineLocal(proyectoSlug string, despacho *DespachoPipelineLocal) (*ResultadoDespachoPipeline, error) {
	if despacho == nil {
		return &ResultadoDespachoPipeline{
			Estado: "sin_despacho",
			Motivo: "paso sin despacho asociado",
		}, nil
	}
	if s == nil || s.pipelineDispatcher == nil {
		return &ResultadoDespachoPipeline{
			Estado: "sin_despachador",
			Motivo: "despachador de pipeline no configurado",
		}, nil
	}
	return s.pipelineDispatcher.DespacharPipeline(SolicitudDespachoPipeline{
		ProyectoSlug: strings.TrimSpace(proyectoSlug),
		Despacho:     despacho,
	})
}

func (s *Service) ListPoolsSummary(activo *bool) ([]*db.PoolCapacidadResumen, error) {
	return s.store.ListPoolsSummary(activo)
}

func (s *Service) GetPoolDetail(slug string) (*PoolDetail, error) {
	pool, err := s.store.GetPool(slug)
	if err != nil {
		return nil, err
	}
	modelos, err := s.store.ListPoolModels(slug)
	if err != nil {
		return nil, err
	}
	resumen, err := s.store.ListPoolsSummary(nil)
	if err != nil {
		return nil, err
	}
	detail := &PoolDetail{
		Pool:                pool,
		SesionesActivas:     0,
		CapacidadDisponible: pool.CapacidadTotal - pool.CapacidadReservada,
		Modelos:             modelos,
	}
	for _, item := range resumen {
		if item.Pool.ID == pool.ID {
			detail.SesionesActivas = item.SesionesActivas
			detail.CapacidadDisponible = item.CapacidadDisponible
			break
		}
	}
	return detail, nil
}

func (s *Service) ListPoolModels(slug string) ([]*db.PoolModelo, error) {
	return s.store.ListPoolModels(slug)
}

func (s *Service) SavePool(pool *db.PoolCapacidad) (int64, error) {
	return s.store.SavePool(pool)
}

func (s *Service) SavePoolModel(poolSlug string, modelo *db.PoolModelo) (int64, error) {
	return s.store.SavePoolModel(poolSlug, modelo)
}

func (s *Service) SeedInitialModels() error {
	return s.store.SeedInitialModels()
}

func (s *Service) ListModelPolicies(scopeTipo, scopeRef string, activa *bool) ([]*db.PoliticaModelo, error) {
	return s.store.ListModelPolicies(scopeTipo, scopeRef, activa)
}

func (s *Service) SaveModelPolicy(policy *db.PoliticaModelo) (int64, error) {
	return s.store.SaveModelPolicy(policy)
}

func (s *Service) SeedInitialModelPolicies() error {
	return s.store.SeedInitialModelPolicies()
}

func (s *Service) ResolveModelPolicy(input db.ResolverPoliticaInput) (*db.ResolucionModelo, error) {
	if input.ProyectoSlug != "" && input.Fase == "" && s.phaseProvider != nil {
		fase, err := s.phaseProvider.GetActivePhase(input.ProyectoSlug)
		if err == nil && fase != "" {
			input.Fase = fase
		}
	}
	return s.store.ResolveModelPolicy(input)
}

type Repository struct{}

func (Repository) ListPoolsSummary(activo *bool) ([]*db.PoolCapacidadResumen, error) {
	return db.ListarPoolsResumen(activo)
}

func (Repository) GetPool(slug string) (*db.PoolCapacidad, error) {
	return db.GetPool(slug)
}

func (Repository) ListPoolModels(slug string) ([]*db.PoolModelo, error) {
	return db.ListarModelosPool(slug)
}

func (Repository) SavePool(pool *db.PoolCapacidad) (int64, error) {
	return db.GuardarPool(pool)
}

func (Repository) SavePoolModel(poolSlug string, modelo *db.PoolModelo) (int64, error) {
	return db.GuardarPoolModelo(poolSlug, modelo)
}

func (Repository) SeedInitialModels() error {
	return db.SeedModelosIniciales()
}

func (Repository) ListModelPolicies(scopeTipo, scopeRef string, activa *bool) ([]*db.PoliticaModelo, error) {
	return db.ListarPoliticasModelo(scopeTipo, scopeRef, activa)
}

func (Repository) SaveModelPolicy(policy *db.PoliticaModelo) (int64, error) {
	return db.GuardarPoliticaModelo(policy)
}

func (Repository) SeedInitialModelPolicies() error {
	return db.SeedPoliticasModeloIniciales()
}

func (Repository) ResolveModelPolicy(input db.ResolverPoliticaInput) (*db.ResolucionModelo, error) {
	return db.ResolverPoliticaModelo(input)
}

func (Repository) ListProjectTasks(proyecto string) ([]*db.Tarea, error) {
	proyecto = strings.TrimSpace(proyecto)
	if proyecto == "" {
		return nil, nil
	}
	item, err := db.GetProyecto(proyecto)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, nil
	}
	return db.ListarTareas(db.FiltroTareas{ProyectoID: &item.ID})
}

func (Repository) GetTask(id int64) (*db.Tarea, error) {
	return db.GetTarea(id)
}

func (Repository) TakeTask(id int64, agente string) error {
	return db.TomarTarea(id, agente)
}

func (Repository) StartTask(id int64, agente string) error {
	return db.IniciarTarea(id, agente)
}

func (Repository) ListProjectPhases(proyecto string) ([]*db.FaseProyecto, error) {
	return db.ListarFasesProyecto(strings.TrimSpace(proyecto))
}

func (Repository) RegisterProjectPhase(fase *db.FaseProyecto) (int64, error) {
	return db.RegistrarFaseProyecto(fase)
}

func (Repository) GetProjectPhase(id int64) (*db.FaseProyecto, error) {
	return db.GetFaseProyecto(id)
}

func (Repository) UpdateProjectPhase(fase *db.FaseProyecto) error {
	return db.ActualizarFaseProyecto(fase)
}
