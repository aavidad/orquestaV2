package sesionesapp

import (
	"testing"
)

type stubStore struct {
	startedAgente  string
	finishedAgente string
	project        *Proyecto
	connector      *Conector
	inspection     []*Sesion
	active         *Sesion
	last           *Sesion
	byID           *Sesion
	lastBudgetID   int64
	lastBudget     *PresupuestoSesion
	evalBudget     *EvaluacionPresupuesto
	registered     *PresupuestoSesion
	registeredID   int64
	startedContext *SesionInicio
	modelPolicy    *ModelPolicyResolution
	capacityPool   *CapacityPool
	assignedPool   struct {
		sessionID int64
		poolID    int64
	}
	saved struct {
		agente     string
		proyectoID *int64
		upd        SesionUpdate
	}
}

func (s *stubStore) RegisterCodex() (string, error) { return "codex9", nil }
func (s *stubStore) StartSession(agente string) (int64, error) {
	s.startedAgente = agente
	return 17, nil
}
func (s *stubStore) FinishSession(agente string) error {
	s.finishedAgente = agente
	return nil
}
func (s *stubStore) GetProject(ref string) (*Proyecto, error) { return s.project, nil }
func (s *stubStore) GetConnector(ref string) (*Conector, error) {
	return s.connector, nil
}
func (s *stubStore) ActivateAssignment(agente string, proyectoID int64, nota string) error {
	return nil
}
func (s *stubStore) StartSessionContext(in SesionInicio) (*Sesion, error) {
	s.startedContext = &in
	return &Sesion{ID: 77, Agente: in.Agente, ProyectoID: in.ProyectoID, Herramienta: in.Herramienta, CWD: in.CWD}, nil
}
func (s *stubStore) GetLastSession(agente string, proyectoID *int64) (*Sesion, error) {
	return s.last, nil
}
func (s *stubStore) GetSessionByID(id int64) (*Sesion, error) {
	if s.byID != nil {
		return s.byID, nil
	}
	return &Sesion{ID: id, Agente: "Codex2"}, nil
}
func (s *stubStore) ListInspectionSessions(filtro FiltroInspeccion) ([]*Sesion, error) {
	return s.inspection, nil
}
func (s *stubStore) GetInspectionSessionByID(id int64) (*Sesion, error) {
	return &Sesion{ID: id, Agente: "Codex2"}, nil
}
func (s *stubStore) SaveActiveSession(agente string, proyectoID *int64, upd SesionUpdate) error {
	s.saved.agente = agente
	s.saved.proyectoID = proyectoID
	s.saved.upd = upd
	return nil
}
func (s *stubStore) GetActiveSession(agente string, proyectoID *int64) (*Sesion, error) {
	return s.active, nil
}
func (s *stubStore) GetLastSessionWithFilter(agente string, proyectoID *int64, cwd string) (*Sesion, error) {
	return s.last, nil
}
func (s *stubStore) RegisterSessionBudget(p *PresupuestoSesion) (int64, error) {
	s.registered = p
	return s.registeredID, nil
}
func (s *stubStore) GetLatestSessionBudget(sesionID int64) (*PresupuestoSesion, error) {
	s.lastBudgetID = sesionID
	return s.lastBudget, nil
}
func (s *stubStore) EvaluateSessionBudget(p *PresupuestoSesion) (*EvaluacionPresupuesto, error) {
	return s.evalBudget, nil
}
func (s *stubStore) ListAgents() ([]*Agente, error) {
	return []*Agente{{Nombre: "codex9", Rol: "programador"}, {Nombre: "Codex2", Rol: "programador"}}, nil
}
func (s *stubStore) ListPendingProposals(agente string) ([]*Propuesta, error) {
	return []*Propuesta{{Codigo: "OP-080", Titulo: "Servidor unico"}}, nil
}
func (s *stubStore) ResolveGovernanceCatalog(rol string, proyectoID *int64) (*GovernanceCatalog, error) {
	reglas, _ := s.ListRules(rol)
	skills, _ := s.ListSkills(rol)
	workflows, _ := s.ListWorkflows(rol)
	return &GovernanceCatalog{
		Reglas:    reglas,
		Skills:    skills,
		Workflows: workflows,
	}, nil
}
func (s *stubStore) ResolveGovernanceCatalogForContext(rol string, proyectoID *int64, agente string) (*GovernanceCatalog, error) {
	return s.ResolveGovernanceCatalog(rol, proyectoID)
}
func (s *stubStore) ResolveGovernanceWorkflowForContext(rol string, proyectoID *int64, agente, nombre string) (*Workflow, error) {
	return s.GetWorkflow(rol, nombre)
}
func (s *stubStore) ListRules(rol string) ([]*Regla, error) {
	return []*Regla{{Titulo: "No romper tests"}}, nil
}
func (s *stubStore) ListSkills(rol string) ([]*Skill, error) {
	return []*Skill{{Nombre: "rg"}}, nil
}
func (s *stubStore) GetWorkflow(rol, nombre string) (*Workflow, error) {
	return &Workflow{Pasos: `["1. votar","2. tomar tarea"]`}, nil
}
func (s *stubStore) ListWorkflows(rol string) ([]*Workflow, error) {
	return []*Workflow{{Nombre: "inicio-sesion"}, {Nombre: "fin-sesion"}}, nil
}
func (s *stubStore) ResolveModelPolicy(input ModelPolicyInput) (*ModelPolicyResolution, error) {
	return s.modelPolicy, nil
}
func (s *stubStore) GetCapacityPool(slug string) (*CapacityPool, error) {
	return s.capacityPool, nil
}
func (s *stubStore) AssignSessionPool(sessionID, poolID int64) error {
	s.assignedPool.sessionID = sessionID
	s.assignedPool.poolID = poolID
	return nil
}

func TestStartConstruyeBriefing(t *testing.T) {
	store := &stubStore{}
	service := NewService(store)

	result, err := service.Start("", true)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if result.Agente != "codex9" || result.SesionID != 17 || result.Rol != "programador" {
		t.Fatalf("resultado inesperado: %+v", result)
	}
	if len(result.PropuestasPendientes) != 1 || len(result.Reglas) != 1 || len(result.Skills) != 1 || len(result.WorkflowPasos) != 2 {
		t.Fatalf("briefing incompleto: %+v", result)
	}
}

func TestFinishDevuelveChecklistYCierraSesion(t *testing.T) {
	store := &stubStore{}
	service := NewService(store)

	result, err := service.Finish("Codex2")
	if err != nil {
		t.Fatalf("Finish: %v", err)
	}
	if result.Agente != "Codex2" || len(result.WorkflowPasos) != 2 || store.finishedAgente != "Codex2" {
		t.Fatalf("resultado inesperado: %+v store=%+v", result, store)
	}
}

func TestBuildBriefing(t *testing.T) {
	store := &stubStore{}
	service := NewService(store)

	result, err := service.BuildBriefing("Codex2")
	if err != nil {
		t.Fatalf("BuildBriefing: %v", err)
	}
	if result.Agent == nil || result.Agent.Nombre != "Codex2" {
		t.Fatalf("agente inesperado: %+v", result.Agent)
	}
	if len(result.PropuestasPendientes) != 1 || len(result.Reglas) != 1 || len(result.Skills) != 1 || len(result.Workflows) != 2 {
		t.Fatalf("briefing inesperado: %+v", result)
	}
}

func TestSaveAndContinueSession(t *testing.T) {
	projectID := int64(31)
	store := &stubStore{
		project: &Proyecto{ID: projectID, Slug: "orquestador"},
		active:  &Sesion{ID: 41, Agente: "Codex2"},
		last:    &Sesion{ID: 42, Agente: "Codex2"},
	}
	service := NewService(store)

	saved, err := service.SaveActiveSession("Codex2", "orquestador", SesionUpdate{Heartbeat: true})
	if err != nil {
		t.Fatalf("SaveActiveSession: %v", err)
	}
	if saved == nil || saved.ID != 41 {
		t.Fatalf("sesion guardada inesperada: %+v", saved)
	}
	if store.saved.agente != "Codex2" || store.saved.proyectoID == nil || *store.saved.proyectoID != projectID || !store.saved.upd.Heartbeat {
		t.Fatalf("save inesperado: %+v", store.saved)
	}

	continued, err := service.Continue("Codex2", "orquestador", "/tmp/x")
	if err != nil {
		t.Fatalf("Continue: %v", err)
	}
	if continued == nil || continued.ID != 42 {
		t.Fatalf("sesion continuada inesperada: %+v", continued)
	}
}

func TestResolveProjectIDReturnsNilWhenProjectMissing(t *testing.T) {
	store := &stubStore{}
	service := NewService(store)

	id, err := service.ResolveProjectID("desconocido")
	if err != nil {
		t.Fatalf("ResolveProjectID: %v", err)
	}
	if id != nil {
		t.Fatalf("project id inesperado: %v", *id)
	}
}

func TestGetAndRegisterBudget(t *testing.T) {
	store := &stubStore{
		active:       &Sesion{ID: 41, Agente: "Codex2"},
		byID:         &Sesion{ID: 52, Agente: "Codex2"},
		lastBudget:   &PresupuestoSesion{ID: 91, SesionID: 41},
		evalBudget:   &EvaluacionPresupuesto{Valido: true},
		registeredID: 88,
	}
	service := NewService(store)

	got, err := service.GetBudget(0, "Codex2")
	if err != nil {
		t.Fatalf("GetBudget: %v", err)
	}
	if got == nil || got.Sesion == nil || got.Sesion.ID != 41 || got.Presupuesto == nil || got.Presupuesto.ID != 91 {
		t.Fatalf("budget inesperado: %+v", got)
	}
	if store.lastBudgetID != 41 {
		t.Fatalf("lastBudgetID=%d", store.lastBudgetID)
	}

	store.lastBudget = &PresupuestoSesion{ID: 88, SesionID: 52}
	res, err := service.RegisterBudget(52, "", &PresupuestoSesion{})
	if err != nil {
		t.Fatalf("RegisterBudget: %v", err)
	}
	if res.ID != 88 || res.Sesion == nil || res.Sesion.ID != 52 {
		t.Fatalf("register budget inesperado: %+v", res)
	}
	if store.registered == nil || store.registered.SesionID != 52 {
		t.Fatalf("registered=%+v", store.registered)
	}
}

func TestStartContextBuildsFullResponse(t *testing.T) {
	projectID := int64(31)
	store := &stubStore{
		project:   &Proyecto{ID: projectID, Slug: "orquestador", RutaAbs: "/tmp/orquestador"},
		connector: &Conector{ID: 12, Slug: "codex-cli"},
		last:      &Sesion{ID: 41, Agente: "codex9"},
	}
	service := NewService(store)

	result, err := service.StartContext(StartContextInput{
		NuevoCodex: true,
		Proyecto:   "orquestador",
		Conector:   "codex-cli",
		Host:       "host1",
		PID:        99,
	})
	if err != nil {
		t.Fatalf("StartContext: %v", err)
	}
	if result == nil || result.Sesion == nil || result.Sesion.ID != 77 || result.Rol != "programador" {
		t.Fatalf("resultado inesperado: %+v", result)
	}
	if result.SesionPrevia == nil || result.SesionPrevia.ID != 41 {
		t.Fatalf("sesion previa inesperada: %+v", result.SesionPrevia)
	}
	if len(result.PropuestasPendientes) != 1 || len(result.Reglas) != 1 || len(result.Skills) != 1 || result.Workflow == nil {
		t.Fatalf("briefing incompleto: %+v", result)
	}
	if store.startedContext == nil || store.startedContext.Agente != "codex9" || store.startedContext.ProyectoID == nil || *store.startedContext.ProyectoID != projectID {
		t.Fatalf("startedContext=%+v", store.startedContext)
	}
	if store.startedContext.CWD != "/tmp/orquestador" || store.startedContext.Herramienta != "codex-cli" {
		t.Fatalf("contexto inesperado: %+v", store.startedContext)
	}
}

func TestStartContextArranqueLimpioLimpiaContinuidadYOmiteSesionPrevia(t *testing.T) {
	projectID := int64(31)
	store := &stubStore{
		project:   &Proyecto{ID: projectID, Slug: "orquestador", RutaAbs: "/tmp/orquestador"},
		connector: &Conector{ID: 12, Slug: "codex-cli"},
		last:      &Sesion{ID: 41, Agente: "Codex2", ExternalSessionID: "sess-previa", ResumePayloadJSON: `{"continuidad":true}`, ResumenContinuidad: "seguir"},
	}
	service := NewService(store)

	result, err := service.StartContext(StartContextInput{
		Agente:            "Codex2",
		Proyecto:          "orquestador",
		Conector:          "codex-cli",
		ExternalSessionID: "sess-nueva",
		ResumePayload:     `{"nueva":true}`,
		Resumen:           "no deberia persistir",
		ArranqueLimpio:    true,
	})
	if err != nil {
		t.Fatalf("StartContext arranque limpio: %v", err)
	}
	if result == nil || result.Sesion == nil {
		t.Fatalf("resultado inesperado: %+v", result)
	}
	if result.SesionPrevia != nil {
		t.Fatalf("no deberia exponer sesion previa en arranque limpio: %+v", result.SesionPrevia)
	}
	if store.startedContext == nil {
		t.Fatalf("faltan datos de inicio")
	}
	if store.startedContext.ExternalSessionID != "" || store.startedContext.ResumePayloadJSON != "" || store.startedContext.ResumenContinuidad != "" {
		t.Fatalf("continuidad no limpiada: %+v", store.startedContext)
	}
}

func TestAssignCanonicalPoolForSessionDelegatesPolicyOutsideDB(t *testing.T) {
	store := &stubStore{
		modelPolicy:  &ModelPolicyResolution{PoolSlug: "ollama-gemma4"},
		capacityPool: &CapacityPool{ID: 9, Runtime: "ollama", MetadataJSON: `{"conector_canonico":"ollama_pool_local"}`},
	}
	service := NewService(store)

	err := service.AssignCanonicalPoolForSession(&Sesion{
		ID:           77,
		Agente:       "Gemma1",
		ProyectoSlug: "orquestador",
		Herramienta:  "ollama_pool_local",
	})
	if err != nil {
		t.Fatalf("AssignCanonicalPoolForSession: %v", err)
	}
	if store.assignedPool.sessionID != 77 || store.assignedPool.poolID != 9 {
		t.Fatalf("pool asignado inesperado: %+v", store.assignedPool)
	}
}

func TestAssignCanonicalPoolForSessionSkipsNonSharedPoolRuntime(t *testing.T) {
	store := &stubStore{
		modelPolicy:  &ModelPolicyResolution{PoolSlug: "codex"},
		capacityPool: &CapacityPool{ID: 4, Runtime: "codex", MetadataJSON: `{"conector_canonico":"codex-cli"}`},
	}
	service := NewService(store)

	err := service.AssignCanonicalPoolForSession(&Sesion{
		ID:           80,
		Agente:       "Codex2",
		ProyectoSlug: "orquestador",
		Herramienta:  "ollama_pool_local",
	})
	if err != nil {
		t.Fatalf("AssignCanonicalPoolForSession: %v", err)
	}
	if store.assignedPool.sessionID != 0 || store.assignedPool.poolID != 0 {
		t.Fatalf("no deberia asignar pool: %+v", store.assignedPool)
	}
}
