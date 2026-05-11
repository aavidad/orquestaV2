package orquestacapacity

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

const CapacityDecisionSchemaVersionV0 = "capacity_decision.v0"

type CapacityDecisionErrorCodeV0 string

const (
	ErrCapacityDecisionJSONInvalidoV0      CapacityDecisionErrorCodeV0 = "capacity_decision_json_invalido"
	ErrCapacityDecisionSchemaNoSoportadoV0 CapacityDecisionErrorCodeV0 = "capacity_decision_schema_no_soportado"
	ErrCapacityDecisionInvalidaV0          CapacityDecisionErrorCodeV0 = "capacity_decision_invalida"
	ErrPerfilTareaNoSoportadoV0            CapacityDecisionErrorCodeV0 = "perfil_tarea_no_soportado"
	ErrEvidenciaInsuficienteParaXHighV0    CapacityDecisionErrorCodeV0 = "evidencia_insuficiente_para_xhigh"
	ErrCuotaNoDisponibleV0                 CapacityDecisionErrorCodeV0 = "cuota_no_disponible"
	ErrCuotaObsoletaV0                     CapacityDecisionErrorCodeV0 = "cuota_obsoleta"
	ErrPoolNoDisponibleV0                  CapacityDecisionErrorCodeV0 = "pool_no_disponible"
	ErrModeloNoHabilitadoV0                CapacityDecisionErrorCodeV0 = "modelo_no_habilitado"
	ErrProveedorRestringidoV0              CapacityDecisionErrorCodeV0 = "proveedor_restringido"
	ErrLocalSinScoreSuficienteV0           CapacityDecisionErrorCodeV0 = "local_sin_score_suficiente"
	ErrConcurrenciaHomeAgotadaV0           CapacityDecisionErrorCodeV0 = "concurrencia_home_agotada"
	ErrHandoffRequeridoV0                  CapacityDecisionErrorCodeV0 = "handoff_requerido"
	ErrReferenciaNoOpacaV0                 CapacityDecisionErrorCodeV0 = "referencia_no_opaca"
	ErrSecretoDetectadoV0                  CapacityDecisionErrorCodeV0 = "secreto_detectado"
	ErrRutaHomeRealDetectadaV0             CapacityDecisionErrorCodeV0 = "ruta_home_real_detectada"
)

type CapacityDecisionV0 struct {
	SchemaVersion string                     `json:"schema_version"`
	Request       CapacityDecisionRequestV0  `json:"request"`
	Response      CapacityDecisionResponseV0 `json:"response"`
}

type CapacityDecisionRequestV0 struct {
	TaskRef          string             `json:"task_ref"`
	PerfilTarea      string             `json:"perfil_tarea"`
	Fase             string             `json:"fase,omitempty"`
	Riesgo           string             `json:"riesgo"`
	CalidadRequerida string             `json:"calidad_requerida"`
	ContextoEstimado ContextoEstimadoV0 `json:"contexto_estimado"`
	Restricciones    RestriccionesV0    `json:"restricciones"`
	Evidencia        EvidenceBundleV0   `json:"evidencia"`
}

type CapacityDecisionResponseV0 struct {
	DecisionID       string                  `json:"decision_id"`
	NivelCapacidad   string                  `json:"nivel_capacidad"`
	ReasoningEffort  string                  `json:"reasoning_effort"`
	Pool             PoolCapacidadV0         `json:"pool"`
	Modelo           ModelCapacityRefV0      `json:"modelo"`
	Home             *AgentHomeSummaryV0     `json:"home,omitempty"`
	Cuota            QuotaSnapshotV0         `json:"cuota"`
	PoliticaEscalado string                  `json:"politica_escalado"`
	Handoff          string                  `json:"handoff"`
	Motivos          []string                `json:"motivos"`
	Alternativas     []CapacityAlternativeV0 `json:"alternativas"`
	Degradacion      DegradacionV0           `json:"degradacion"`
}

type PoolCapacidadV0 struct {
	PoolID            string               `json:"pool_id"`
	ProviderKind      string               `json:"provider_kind"`
	RuntimeKind       string               `json:"runtime_kind"`
	PlanKind          string               `json:"plan_kind"`
	Paid              bool                 `json:"paid"`
	Active            bool                 `json:"active"`
	TotalSlots        int                  `json:"total_slots"`
	ReservedSlots     int                  `json:"reserved_slots"`
	AllowsChildAgents bool                 `json:"allows_child_agents"`
	AllowsMultiModel  bool                 `json:"allows_multi_model"`
	AllowsOvercost    bool                 `json:"allows_overcost"`
	HandoffPolicy     string               `json:"handoff_policy"`
	TelemetrySource   string               `json:"telemetry_source"`
	CredentialModes   []string             `json:"credential_modes"`
	QuotaScope        string               `json:"quota_scope"`
	Modelos           []ModelCapacityRefV0 `json:"modelos"`
}

type ModelCapacityRefV0 struct {
	ModelRef         string                `json:"model_ref"`
	Enabled          bool                  `json:"enabled"`
	Priority         int                   `json:"priority"`
	RelativeCost     float64               `json:"relative_cost"`
	Locality         string                `json:"locality"`
	KnownLimits      *KnownLimitsV0        `json:"known_limits,omitempty"`
	ExecutionMode    string                `json:"execution_mode"`
	EnablementSource string                `json:"enablement_source"`
	SupportedEfforts []string              `json:"supported_efforts,omitempty"`
	Score            *ModelEvidenceScoreV0 `json:"score,omitempty"`
}

type AgentHomeSummaryV0 struct {
	LogicalAgentRef string `json:"logical_agent_ref"`
	HomeRef         string `json:"home_ref"`
	PoolID          string `json:"pool_id"`
	RuntimeRef      string `json:"runtime_ref"`
	ProviderRef     string `json:"provider_ref"`
	AccountRef      string `json:"account_ref"`
	CredentialRef   string `json:"credential_ref,omitempty"`
}

type QuotaSnapshotV0 struct {
	Source            string   `json:"source"`
	Freshness         string   `json:"freshness"`
	CheckedAt         string   `json:"checked_at,omitempty"`
	Scope             string   `json:"scope"`
	WindowKind        string   `json:"window_kind"`
	ResetAt           string   `json:"reset_at,omitempty"`
	RemainingSeconds  *int     `json:"remaining_seconds,omitempty"`
	RemainingMessages *int     `json:"remaining_messages,omitempty"`
	RemainingTokens   *int     `json:"remaining_tokens,omitempty"`
	RemainingCredits  *float64 `json:"remaining_credits,omitempty"`
	Confidence        *float64 `json:"confidence,omitempty"`
	RawRef            string   `json:"raw_ref,omitempty"`
}

type ModelEvidenceScoreV0 struct {
	Materia        string  `json:"materia"`
	ScoreBase      float64 `json:"score_base"`
	ScoreObservado float64 `json:"score_observado"`
	ScoreTotal     float64 `json:"score_total"`
	Confianza      float64 `json:"confianza"`
	Muestras       int     `json:"muestras"`
	Benchmarks     int     `json:"benchmarks"`
	LastObservedAt string  `json:"last_observed_at,omitempty"`
}

type KnownLimitsV0 struct {
	Messages *int     `json:"messages,omitempty"`
	Tokens   *int     `json:"tokens,omitempty"`
	Seconds  *int     `json:"seconds,omitempty"`
	Credits  *float64 `json:"credits,omitempty"`
	Window   string   `json:"window,omitempty"`
}

type CapacityAlternativeV0 struct {
	Priority     int     `json:"priority"`
	PoolID       string  `json:"pool_id"`
	ModelRef     string  `json:"model_ref"`
	RelativeCost float64 `json:"relative_cost"`
	Availability string  `json:"availability"`
	Reason       string  `json:"reason"`
}

type DegradacionV0 struct {
	Action         string `json:"action"`
	TargetLevel    string `json:"target_level,omitempty"`
	TargetPoolID   string `json:"target_pool_id,omitempty"`
	TargetModelRef string `json:"target_model_ref,omitempty"`
	Reason         string `json:"reason"`
}

type ContextoEstimadoV0 struct {
	Status            string `json:"status"`
	EstimatedTokens   *int   `json:"estimated_tokens,omitempty"`
	EstimatedMessages *int   `json:"estimated_messages,omitempty"`
	EstimatedSeconds  *int   `json:"estimated_seconds,omitempty"`
}

type RestriccionesV0 struct {
	LocalOnly                 bool     `json:"local_only"`
	RemoteAllowed             bool     `json:"remote_allowed"`
	PaidAllowed               bool     `json:"paid_allowed"`
	MaxCosteRelativo          *float64 `json:"max_coste_relativo,omitempty"`
	RequiereHandoffPreventivo bool     `json:"requiere_handoff_preventivo"`
	RequierePrivacidadAlta    bool     `json:"requiere_privacidad_alta"`
}

type EvidenceBundleV0 struct {
	QualitySignals   []EvidenceSignalV0 `json:"quality_signals"`
	BenchmarkSignals []EvidenceSignalV0 `json:"benchmark_signals"`
	TelemetrySignals []EvidenceSignalV0 `json:"telemetry_signals"`
	QuotaSignals     []EvidenceSignalV0 `json:"quota_signals"`
	FailureSignals   []EvidenceSignalV0 `json:"failure_signals"`
	HumanSignals     []EvidenceSignalV0 `json:"human_signals"`
}

type EvidenceSignalV0 struct {
	SignalRef string `json:"signal_ref"`
	Kind      string `json:"kind"`
	Source    string `json:"source"`
	Freshness string `json:"freshness"`
	Weight    string `json:"weight"`
	Summary   string `json:"summary,omitempty"`
}

type CapacityDecisionIssueV0 struct {
	Code  CapacityDecisionErrorCodeV0 `json:"code"`
	Field string                      `json:"field,omitempty"`
}

type CapacityDecisionValidationErrorV0 struct {
	Issues []CapacityDecisionIssueV0 `json:"issues"`
}

func (err CapacityDecisionValidationErrorV0) Error() string {
	if len(err.Issues) == 0 {
		return string(ErrCapacityDecisionInvalidaV0)
	}
	return string(err.Issues[0].Code)
}

func DecodeCapacityDecisionV0(data []byte) (CapacityDecisionV0, error) {
	var decision CapacityDecisionV0
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decision); err != nil {
		return CapacityDecisionV0{}, CapacityDecisionValidationErrorV0{
			Issues: []CapacityDecisionIssueV0{{Code: ErrCapacityDecisionJSONInvalidoV0}},
		}
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return CapacityDecisionV0{}, CapacityDecisionValidationErrorV0{
			Issues: []CapacityDecisionIssueV0{{Code: ErrCapacityDecisionJSONInvalidoV0}},
		}
	}
	if issues := ValidateCapacityDecisionV0(decision); len(issues) > 0 {
		return CapacityDecisionV0{}, CapacityDecisionValidationErrorV0{Issues: issues}
	}
	return decision, nil
}

func ValidateCapacityDecisionV0(decision CapacityDecisionV0) []CapacityDecisionIssueV0 {
	v := capacityDecisionValidatorV0{}
	v.validate(decision)
	return v.issues
}

func (decision CapacityDecisionV0) Validate() []CapacityDecisionIssueV0 {
	return ValidateCapacityDecisionV0(decision)
}

func (decision CapacityDecisionV0) Valid() bool {
	return len(ValidateCapacityDecisionV0(decision)) == 0
}
