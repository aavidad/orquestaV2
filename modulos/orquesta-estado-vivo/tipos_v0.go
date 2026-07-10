package orquestaestadovivo

const (
	ProyeccionCicloVidaSchemaV0 = "orquesta_proyeccion_ciclo_vida.v0"
	VeredictoCausalSchemaV0     = "orquesta_veredicto_causal.v0"
)

type FaseCicloVidaV0 string

const (
	FaseSolicitadoV0       FaseCicloVidaV0 = "solicitado"
	FaseLanzadoV0          FaseCicloVidaV0 = "lanzado"
	FaseProcesoVivoV0      FaseCicloVidaV0 = "proceso_vivo"
	FaseEntregadoParcialV0 FaseCicloVidaV0 = "entregado_parcial"
	FaseBloqueadoV0        FaseCicloVidaV0 = "bloqueado"
	FaseTerminalAceptadoV0 FaseCicloVidaV0 = "terminal_aceptado"
	FaseTerminalReworkV0   FaseCicloVidaV0 = "terminal_rework"
	FaseHuerfanoV0         FaseCicloVidaV0 = "huerfano"
	FaseConflictoV0        FaseCicloVidaV0 = "conflicto"
	FaseDesconocidoV0      FaseCicloVidaV0 = "desconocido"
)

type EvidenciaEstadoV0 struct {
	RunRef                      string
	GoalRef                     string
	ExternalGoalRef             string
	Fuente                      string
	Estado                      string
	Scope                       ScopeEvidenciaEstadoV0
	RuntimeIdentityRef          string
	RuntimeGenerationRef        string
	RuntimeIdentityMismatch     bool
	RuntimeObservationAttempted bool
	RuntimeObservado            bool
	ProcesoVivo                 bool
	Terminal                    bool
	Aceptado                    bool
	ObservadoEn                 string
	EvidenceRefs                []string
}

type ScopeEvidenciaEstadoV0 string

const (
	ScopeGoalExecutionV0  ScopeEvidenciaEstadoV0 = "goal_execution"
	ScopeGoalV0           ScopeEvidenciaEstadoV0 = ScopeGoalExecutionV0
	ScopeBackendServiceV0 ScopeEvidenciaEstadoV0 = "backend_service"
)

type ClaseVeredictoCausalV0 string

const (
	VeredictoRunningConfirmedV0      ClaseVeredictoCausalV0 = "running_confirmed"
	VeredictoTerminalByArtifactV0    ClaseVeredictoCausalV0 = "terminal_by_artifact"
	VeredictoProcessDeadStateStaleV0 ClaseVeredictoCausalV0 = "process_dead_state_stale"
	VeredictoDivergentNeedsRepairV0  ClaseVeredictoCausalV0 = "divergent_needs_repair"
	VeredictoIndeterminateV0         ClaseVeredictoCausalV0 = "indeterminate"
)

type VeredictoCausalV0 struct {
	SchemaVersion         string                 `json:"schema_version"`
	RunRef                string                 `json:"run_ref,omitempty"`
	GoalRef               string                 `json:"goal_ref,omitempty"`
	ExternalGoalRef       string                 `json:"external_goal_ref,omitempty"`
	Clase                 ClaseVeredictoCausalV0 `json:"class"`
	ReasonCode            string                 `json:"reason_code"`
	EstadoPersistido      string                 `json:"persisted_state,omitempty"`
	RuntimeIdentityRef    string                 `json:"runtime_identity_ref,omitempty"`
	RuntimeGenerationRef  string                 `json:"runtime_generation_ref,omitempty"`
	RuntimeIdentityRefs   []string               `json:"runtime_identity_refs,omitempty"`
	RuntimeGenerationRefs []string               `json:"runtime_generation_refs,omitempty"`
	RuntimeObservado      bool                   `json:"runtime_observed"`
	ProcesoVivo           bool                   `json:"process_live"`
	ResultadoTerminal     bool                   `json:"terminal_result"`
	ResultadoAceptado     bool                   `json:"result_accepted"`
	PublicarRunning       bool                   `json:"publish_running"`
	RequiereReparacion    bool                   `json:"needs_repair"`
	EvidenceRefs          []string               `json:"evidence_refs,omitempty"`
	Conflictos            []ConflictoEstadoV0    `json:"conflicts,omitempty"`
}

type ConflictoEstadoV0 struct {
	RunRef  string
	Codigo  string
	Fuentes []string
}

type NodoCicloVidaV0 struct {
	RunRef          string
	GoalRef         string
	ExternalGoalRef string
	Fase            FaseCicloVidaV0
	Veredicto       VeredictoCausalV0 `json:"veredicto_causal"`
	Evidencias      []EvidenciaEstadoV0
	Conflictos      []ConflictoEstadoV0
}

type ProyeccionCicloVidaV0 struct {
	SchemaVersion string
	GeneradaEn    string
	Nodos         []NodoCicloVidaV0
}
