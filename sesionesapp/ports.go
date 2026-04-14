package sesionesapp

import "time"

type FiltroInspeccion struct {
	Agente     *string
	ProyectoID *int64
	Activa     *bool
	Estado     *string
	Limit      int
}

type Proyecto struct {
	ID                int64
	Slug              string
	Nombre            string
	RutaAbs           string
	Descripcion       string
	Estado            string
	Tipo              string
	RutaContextoExtra string
	MaxContextRetries int
	CreatedAt         time.Time
}

type Conector struct {
	ID                     int64
	Slug                   string
	Nombre                 string
	RutaAbs                string
	RutaInstalacion        string
	Tipo                   string
	Estado                 string
	Version                string
	RepositorioOriginalURL string
	Observaciones          string
	MetadataJSON           string
	CreatedAt              time.Time
}

type ModelPolicyInput struct {
	AgentName   *string
	TaskID      *int64
	ProjectSlug string
	Phase       string
	TaskProfile string
}

type ModelPolicyResolution struct {
	TaskProfile     string
	PoolSlug        string
	ModelSlug       string
	ReasoningEffort string
}

type CapacityPool struct {
	ID           int64
	Runtime      string
	MetadataJSON string
}

type Propuesta struct {
	ID              int64
	Codigo          string
	Tipo            string
	Titulo          string
	Descripcion     string
	Estado          string
	ProyectoSlug    string
	PropuestoPor    string
	DistribuidorPor string
	VotosRequeridos int
	CreatedAt       time.Time
	UpdatedAt       time.Time
	ResolucionVotos string
	CerradoPor      string
	CerradoAt       *time.Time
}

type Regla struct {
	ID          int64
	TipoAgente  string
	Categoria   string
	Titulo      string
	Descripcion string
	Activa      bool
	CreatedAt   time.Time
	Obligatoria bool
}

type Skill struct {
	ID                 int64
	TipoAgente         string
	Nombre             string
	Descripcion        string
	Params             string
	CuandoUsar         string
	Escenario          string
	Prioridad          int
	AliasesJSON        string
	HerramientasJSON   string
	Origen             string
	NivelRiesgo        string
	RequiereAprobacion bool
	Activa             bool
	CreatedAt          time.Time
}

type Workflow struct {
	ID          int64
	TipoAgente  string
	Nombre      string
	Descripcion string
	Pasos       string
	Activo      bool
	CreatedAt   time.Time
}

type GovernanceCatalog struct {
	TipoAgente       string      `json:"tipo_agente"`
	Rol              string      `json:"rol"`
	CatalogoChecksum string      `json:"catalogo_checksum"`
	Reglas           []*Regla    `json:"reglas,omitempty"`
	Skills           []*Skill    `json:"skills,omitempty"`
	Workflows        []*Workflow `json:"workflows,omitempty"`
	ResolucionActual string      `json:"-"`
}

type PresupuestoSesion struct {
	ID                int64
	SesionID          int64
	PoolID            *int64
	ModelSlug         string
	WindowKind        string
	WindowStartedAt   *time.Time
	ResetAt           *time.Time
	RemainingSeconds  *int64
	RemainingMessages *int64
	RemainingTokens   *int64
	RemainingCredits  *float64
	BudgetSource      string
	RawSnapshotJSON   string
	CheckedAt         time.Time
	CreatedAt         time.Time
}

type EvaluacionPresupuesto struct {
	Estado           string
	DebeHandoff      bool
	Motivo           string
	ThresholdSeconds int64
	ThresholdRatio   float64
	RemainingRatio   *float64
	Valido           bool
}

type Store interface {
	RegisterCodex() (string, error)
	StartSession(agente string) (int64, error)
	FinishSession(agente string) error
	GetProject(ref string) (*Proyecto, error)
	GetConnector(ref string) (*Conector, error)
	ActivateAssignment(agente string, proyectoID int64, nota string) error
	StartSessionContext(in SesionInicio) (*Sesion, error)
	GetLastSession(agente string, proyectoID *int64) (*Sesion, error)
	GetSessionByID(id int64) (*Sesion, error)
	ListInspectionSessions(filtro FiltroInspeccion) ([]*Sesion, error)
	GetInspectionSessionByID(id int64) (*Sesion, error)
	SaveActiveSession(agente string, proyectoID *int64, upd SesionUpdate) error
	GetActiveSession(agente string, proyectoID *int64) (*Sesion, error)
	GetLastSessionWithFilter(agente string, proyectoID *int64, cwd string) (*Sesion, error)
	RegisterSessionBudget(p *PresupuestoSesion) (int64, error)
	GetLatestSessionBudget(sesionID int64) (*PresupuestoSesion, error)
	EvaluateSessionBudget(p *PresupuestoSesion) (*EvaluacionPresupuesto, error)
	ListAgents() ([]*Agente, error)
	ListPendingProposals(agente string) ([]*Propuesta, error)
	ResolveGovernanceCatalog(rol string, proyectoID *int64) (*GovernanceCatalog, error)
	ResolveGovernanceCatalogForContext(rol string, proyectoID *int64, agente string) (*GovernanceCatalog, error)
	ResolveGovernanceWorkflowForContext(rol string, proyectoID *int64, agente, nombre string) (*Workflow, error)
	ListRules(rol string) ([]*Regla, error)
	ListSkills(rol string) ([]*Skill, error)
	GetWorkflow(rol, nombre string) (*Workflow, error)
	ListWorkflows(rol string) ([]*Workflow, error)
	ResolveModelPolicy(input ModelPolicyInput) (*ModelPolicyResolution, error)
	GetCapacityPool(slug string) (*CapacityPool, error)
	AssignSessionPool(sessionID, poolID int64) error
}
