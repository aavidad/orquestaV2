package orquestafactory

type AppSpecV0 struct {
	SchemaVersion    string              `json:"schema_version"`
	SpecID           string              `json:"spec_id"`
	RequestID        string              `json:"request_id"`
	CreatedAt        string              `json:"created_at"`
	Locale           string              `json:"locale"`
	RequestKind      string              `json:"request_kind"`
	ExecutionMode    string              `json:"execution_mode"`
	App              AppInfoV0           `json:"app"`
	Scope            ScopeV0             `json:"scope"`
	Architecture     ArchitectureV0      `json:"architecture"`
	I18N             I18NSpecV0          `json:"i18n"`
	Data             DataSpecV0          `json:"data"`
	Connectors       ConnectorsSpecV0    `json:"connectors"`
	Platforms        []string            `json:"platforms"`
	Deploy           DeploySpecV0        `json:"deploy"`
	Quality          QualitySpecV0       `json:"quality"`
	Docs             DocsSpecV0          `json:"docs"`
	AgentPreferences AgentPreferencesV0  `json:"agent_preferences"`
	DefaultsApplied  []DefaultAppliedV0  `json:"defaults_applied"`
	Validation       ValidationSummaryV0 `json:"validation"`
}

type AppInfoV0 struct {
	Nombre           string   `json:"nombre"`
	Slug             string   `json:"slug"`
	Objetivo         string   `json:"objetivo"`
	Descripcion      string   `json:"descripcion,omitempty"`
	TipoApp          string   `json:"tipo_app"`
	UsuariosObjetivo []string `json:"usuarios_objetivo"`
}

type ScopeV0 struct {
	Objetivos         []string `json:"objetivos"`
	FueraDeAlcance    []string `json:"fuera_de_alcance"`
	Supuestos         []string `json:"supuestos"`
	PreguntasAbiertas []string `json:"preguntas_abiertas"`
}

type ArchitectureV0 struct {
	Patron             string             `json:"patron"`
	ModulosIniciales   []ModuleBoundaryV0 `json:"modulos_iniciales"`
	Fronteras          []string           `json:"fronteras"`
	ContratosEsperados []string           `json:"contratos_esperados"`
}

type ModuleBoundaryV0 struct {
	Nombre          string   `json:"nombre"`
	Responsabilidad string   `json:"responsabilidad"`
	Puertos         []string `json:"puertos,omitempty"`
}

type I18NSpecV0 struct {
	Enabled       bool     `json:"enabled"`
	DefaultLocale string   `json:"default_locale"`
	Locales       []string `json:"locales"`
	Justificacion string   `json:"justificacion,omitempty"`
}

type DataSpecV0 struct {
	PersistenceRequired bool     `json:"persistence_required"`
	Needs               []string `json:"needs"`
	Sensitivity         string   `json:"sensitivity,omitempty"`
	Connector           string   `json:"connector,omitempty"`
	Retention           string   `json:"retention,omitempty"`
}

type ConnectorsSpecV0 struct {
	Required []ConnectorSpecV0 `json:"required"`
	Optional []ConnectorSpecV0 `json:"optional"`
}

type ConnectorSpecV0 struct {
	Nombre    string `json:"nombre"`
	Proposito string `json:"proposito"`
	Contrato  string `json:"contrato,omitempty"`
}

type DeploySpecV0 struct {
	Target       string   `json:"target"`
	Restrictions []string `json:"restrictions"`
}

type QualitySpecV0 struct {
	Tests         string   `json:"tests"`
	Accessibility string   `json:"accessibility"`
	Security      []string `json:"security"`
	Compliance    []string `json:"compliance"`
	Observability bool     `json:"observability"`
}

type DocsSpecV0 struct {
	User        bool     `json:"user"`
	Development bool     `json:"development"`
	Systems     bool     `json:"systems"`
	Locales     []string `json:"locales"`
}

type AgentPreferencesV0 struct {
	HumanReview bool     `json:"human_review"`
	Autonomy    string   `json:"autonomy"`
	Notes       []string `json:"notes,omitempty"`
}

type DefaultAppliedV0 struct {
	Campo  string `json:"campo"`
	Valor  any    `json:"valor"`
	Motivo string `json:"motivo"`
}

type ValidationSummaryV0 struct {
	Estado   string            `json:"estado"`
	Warnings []ValidationIssue `json:"warnings"`
	Errores  []ValidationIssue `json:"errores"`
}
