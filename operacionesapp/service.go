package operacionesapp

import "time"

type Agente struct {
	Nombre                    string
	Rol                       string
	Activo                    bool
	Habilitado                bool
	SinCuotaProveedor         bool
	EstadoSesion              string
	UltimaSesion              *time.Time
	ConsumoDiaSegundos        int
	ConsumoSemanalSegundos    int
	LimiteDiaSegundos         int
	LimiteSemanalSegundos     int
	LastUsageResetAt          *time.Time
	EstadoCuota               string
	ReanimarAt                *time.Time
	MotivoPausa               string
	CuotaRestantePct          *int
	PresupuestoEstado         string
	PresupuestoFuente         string
	PresupuestoCheckedAt      *time.Time
	PresupuestoStale          bool
	PresupuestoVentana        string
	PresupuestoResetAt        *time.Time
	PresupuestoSesionPct      *int
	PresupuestoSesionResetAt  *time.Time
	PresupuestoDiarioPct      *int
	PresupuestoDiarioResetAt  *time.Time
	PresupuestoSemanalPct     *int
	PresupuestoSemanalResetAt *time.Time
	RemainingSeconds          *int64
	RemainingMessages         *int64
	RemainingTokens           *int64
	RemainingCredits          *float64
	ObservedUsageTokens       *int64
	ObservedUsageCostUSD      *float64
	ObservedUsageMessages     *int
	ObservedUsageTurns        *int
	ObservedUsageUpdatedAt    *time.Time
	ObservedSessionPath       string
	CuentaID                  string
	CuentaUsuario             string
	CuentaEmail               string
	CuentaFuente              string
	CuentaObservadaAt         *time.Time
}

type Conector struct {
	ID           int64
	Slug         string
	Nombre       string
	Transporte   string
	Comando      string
	ArgsJSON     string
	EnvJSON      string
	MetadataJSON string
	Activo       bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type EstadoAsignacion string

type Asignacion struct {
	ID             int64
	Agente         string
	ProyectoID     int64
	ProyectoSlug   string
	ProyectoNombre string
	Estado         EstadoAsignacion
	Nota           string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	CerradaAt      *time.Time
}

type SesionActiva struct {
	ID                 int64
	Agente             string
	Inicio             time.Time
	Fin                *time.Time
	Activa             bool
	ConectorID         *int64
	ConectorSlug       string
	ProyectoID         *int64
	ProyectoSlug       string
	Estado             string
	Cwd                string
	Herramienta        string
	ExternalSessionID  string
	ResumePayloadJSON  string
	ResumenContinuidad string
	Branch             string
	HeartbeatAt        *time.Time
	Host               string
	PID                *int64
}

type Proyecto struct {
	ID        int64
	Slug      string
	Nombre    string
	RutaAbs   string
	Tipo      string
	ParentID  *int64
	Activo    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type FiltroSesionesInspeccion struct {
	Agente     *string
	ProyectoID *int64
	Activa     *bool
	Estado     *string
	Limit      int
}

type Sesion struct {
	ID                 int64
	Agente             string
	ConectorID         *int64
	ConectorSlug       string
	ConectorNombre     string
	ProyectoID         *int64
	ProyectoSlug       string
	ProyectoNombre     string
	Inicio             time.Time
	Fin                *time.Time
	Activa             bool
	Estado             string
	CWD                string
	Herramienta        string
	ExternalSessionID  string
	ResumePayloadJSON  string
	ResumenContinuidad string
	Branch             string
	HeartbeatAt        *time.Time
	Host               string
	PID                *int64
}

type AuditEntry struct {
	Agente    string
	Accion    string
	Entidad   string
	EntidadID int64
	Detalle   string
	CreatedAt time.Time
}

type Store interface {
	ListAgents() ([]*Agente, error)
	ListConnectors() ([]*Conector, error)
	ListAssignments(estado, agente string) ([]*Asignacion, error)
	ListActiveSessions() ([]*SesionActiva, error)
	GetProject(ref string) (*Proyecto, error)
	ListInspectionSessions(filtro FiltroSesionesInspeccion) ([]*Sesion, error)
	GetInspectionSession(id int64) (*Sesion, error)
	AuditLog(limit int) ([]AuditEntry, error)
	RegisterAgent(nombre, rol string) error
	RetireAgent(nombre string) error
	RehabilitateAgent(nombre string) error
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) ListAgents() ([]*Agente, error) {
	return s.store.ListAgents()
}

func (s *Service) ListConnectors() ([]*Conector, error) {
	return s.store.ListConnectors()
}

func (s *Service) ListAssignments(estado, agente string) ([]*Asignacion, error) {
	return s.store.ListAssignments(estado, agente)
}

func (s *Service) ListActiveSessions() ([]*SesionActiva, error) {
	return s.store.ListActiveSessions()
}

type ListInspectionSessionsInput struct {
	Agente      string
	ProyectoRef string
	Estado      string
	Activa      *bool
}

func (s *Service) ListInspectionSessions(input ListInspectionSessionsInput) ([]*Sesion, error) {
	filtro := FiltroSesionesInspeccion{}
	if input.Agente != "" {
		filtro.Agente = &input.Agente
	}
	if input.ProyectoRef != "" {
		proyecto, err := s.store.GetProject(input.ProyectoRef)
		if err != nil {
			return nil, err
		}
		filtro.ProyectoID = &proyecto.ID
	}
	if input.Estado != "" {
		filtro.Estado = &input.Estado
	}
	if input.Activa != nil {
		filtro.Activa = input.Activa
	}
	return s.store.ListInspectionSessions(filtro)
}

func (s *Service) GetInspectionSession(id int64) (*Sesion, error) {
	return s.store.GetInspectionSession(id)
}

func (s *Service) AuditLog(limit int) ([]AuditEntry, error) {
	return s.store.AuditLog(limit)
}

func (s *Service) RegisterAgent(nombre, rol string) error {
	return s.store.RegisterAgent(nombre, rol)
}

func (s *Service) RetireAgent(nombre string) error {
	return s.store.RetireAgent(nombre)
}

func (s *Service) RehabilitateAgent(nombre string) error {
	return s.store.RehabilitateAgent(nombre)
}
