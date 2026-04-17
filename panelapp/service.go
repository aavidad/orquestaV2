package panelapp

import "time"

type EstadoPropuesta string

const (
	PropuestaAbierta EstadoPropuesta = "abierta"
)

type EstadoTarea string

const (
	EstadoEnProgreso EstadoTarea = "en_progreso"
	EstadoCompletada EstadoTarea = "completada"
)

type PrioridadTarea string

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

type Propuesta struct {
	ID           int64
	Codigo       string
	Titulo       string
	Descripcion  string
	ProyectoID   *int64
	Tipo         string
	Estado       EstadoPropuesta
	PropuestoPor string
	Distribuidor string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	CerradaAt    *time.Time
}

type Tarea struct {
	ID               int64
	Titulo           string
	Descripcion      string
	ProyectoID       *int64
	Modulo           string
	Estado           EstadoTarea
	Agente           *string
	PropuestaID      *int64
	Prioridad        PrioridadTarea
	Dependencias     []int64
	ContratoDefinido bool
	BlueprintKey     string
	CreadoPor        string
	CommitCierre     string
	Notas            string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	CompletadaAt     *time.Time
}

type FiltroTareas struct {
	Estado      *EstadoTarea
	Agente      *string
	ProyectoID  *int64
	Modulo      *string
	PropuestaID *int64
	Libre       bool
}

type Store interface {
	ListAgents() ([]*Agente, error)
	CountTasksByState() (map[string]int, error)
	ListProposals(estado *EstadoPropuesta) ([]*Propuesta, error)
	CountVotes(propuestaID int64) (int, int, int, int, error)
	ListTasks(filtro FiltroTareas) ([]*Tarea, error)
}

type Service struct {
	store Store
}

type ProposalSummary struct {
	Codigo     string
	Titulo     string
	Acuerdo    int
	Desacuerdo int
	Pendiente  int
}

type Summary struct {
	Agents      []*Agente
	TaskCounts  map[string]int
	TotalTasks  int
	DoneTasks   int
	PercentDone int
	OpenProps   []ProposalSummary
	ActiveTasks []*Tarea
	GeneratedAt time.Time
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) BuildSummary() (*Summary, error) {
	agents, err := s.store.ListAgents()
	if err != nil {
		return nil, err
	}
	counts, err := s.store.CountTasksByState()
	if err != nil {
		return nil, err
	}
	totalTasks := 0
	doneTasks := 0
	for estado, count := range counts {
		totalTasks += count
		if estado == string(EstadoCompletada) {
			doneTasks = count
		}
	}
	percentDone := 0
	if totalTasks > 0 {
		percentDone = doneTasks * 100 / totalTasks
	}

	openState := PropuestaAbierta
	proposals, err := s.store.ListProposals(&openState)
	if err != nil {
		return nil, err
	}
	openProps := make([]ProposalSummary, 0, len(proposals))
	for _, proposal := range proposals {
		acuerdo, desacuerdo, _, pendiente, err := s.store.CountVotes(proposal.ID)
		if err != nil {
			return nil, err
		}
		openProps = append(openProps, ProposalSummary{
			Codigo:     proposal.Codigo,
			Titulo:     proposal.Titulo,
			Acuerdo:    acuerdo,
			Desacuerdo: desacuerdo,
			Pendiente:  pendiente,
		})
	}

	activeState := EstadoEnProgreso
	activeTasks, err := s.store.ListTasks(FiltroTareas{Estado: &activeState})
	if err != nil {
		return nil, err
	}

	return &Summary{
		Agents:      agents,
		TaskCounts:  counts,
		TotalTasks:  totalTasks,
		DoneTasks:   doneTasks,
		PercentDone: percentDone,
		OpenProps:   openProps,
		ActiveTasks: activeTasks,
		GeneratedAt: time.Now(),
	}, nil
}
