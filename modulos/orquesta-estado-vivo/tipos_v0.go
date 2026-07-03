package orquestaestadovivo

const (
	ProyeccionCicloVidaSchemaV0 = "orquesta_proyeccion_ciclo_vida.v0"
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
	RunRef          string
	GoalRef         string
	ExternalGoalRef string
	Fuente          string
	Estado          string
	ProcesoVivo     bool
	Terminal        bool
	Aceptado        bool
	ObservadoEn     string
	EvidenceRefs    []string
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
	Evidencias      []EvidenciaEstadoV0
	Conflictos      []ConflictoEstadoV0
}

type ProyeccionCicloVidaV0 struct {
	SchemaVersion string
	GeneradaEn    string
	Nodos         []NodoCicloVidaV0
}
