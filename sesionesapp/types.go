package sesionesapp

import (
	"time"
)

type Agente struct {
	Nombre                    string
	Rol                       string
	Activo                    bool // en sesión ahora mismo
	Habilitado                bool // false = retirado por Alberto
	SinCuotaProveedor         bool
	EstadoSesion              string // disponible | programando | esperando | votando
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

type SesionInicio struct {
	Agente             string
	ConectorID         *int64
	ProyectoID         *int64
	CWD                string
	Herramienta        string
	ExternalSessionID  string
	ResumePayloadJSON  string
	ResumenContinuidad string
	Branch             string
	Host               string
	PID                *int64
}

type SesionUpdate struct {
	CWD                *string
	Herramienta        *string
	ExternalSessionID  *string
	ResumePayloadJSON  *string
	ResumenContinuidad *string
	Branch             *string
	Host               *string
	PID                *int64
	Heartbeat          bool
	Estado             *string
}
