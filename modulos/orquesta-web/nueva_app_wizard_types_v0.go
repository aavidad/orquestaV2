package orquestaweb

import orquestafactory "orquesta/modulos/orquesta-factory"

const (
	WizardTopicUsoV0     = "uso"
	WizardTopicDatosV0   = "datos"
	WizardTopicEntregaV0 = "entrega"

	WizardImportanceAltaV0  = "alta"
	WizardImportanceMediaV0 = "media"
)

type WizardQuestionV0 struct {
	QuestionRef     string           `json:"question_ref"`
	Field           string           `json:"field"`
	TopicGroup      string           `json:"topic_group"`
	Importance      string           `json:"importance"`
	PromptKey       string           `json:"prompt_key"`
	WhyKey          string           `json:"why_key"`
	HelpKey         string           `json:"help_key"`
	RequiresFacts   []WizardFactV0   `json:"requires_facts,omitempty"`
	ExcludedByFacts []WizardFactV0   `json:"excluded_by_facts,omitempty"`
	Options         []WizardOptionV0 `json:"options"`
}

type WizardOptionV0 struct {
	Value           string         `json:"value"`
	LabelKey        string         `json:"label_key"`
	HelpKey         string         `json:"help_key"`
	ExampleKey      string         `json:"example_key,omitempty"`
	Recommended     bool           `json:"recommended"`
	RationaleKey    string         `json:"rationale_key,omitempty"`
	RequiresFacts   []WizardFactV0 `json:"requires_facts,omitempty"`
	ExcludedByFacts []WizardFactV0 `json:"excluded_by_facts,omitempty"`
}

type WizardAnswerV0 struct {
	QuestionRef        string `json:"question_ref"`
	UserChoice         string `json:"user_choice"`
	FreeText           bool   `json:"free_text,omitempty"`
	ComprehensionQuery string `json:"comprehension_query,omitempty"`
	Justification      string `json:"justification,omitempty"`
}

type WizardTurnResultV0 struct {
	SchemaVersion       string                            `json:"schema_version"`
	SessionRef          string                            `json:"session_ref"`
	Turn                int                               `json:"turn"`
	Questions           []WizardQuestionV0                `json:"questions"`
	Decisions           []WebNuevaAppIntakeDecisionV0     `json:"decisions,omitempty"`
	ResolvedByExclusion []WizardDecisionV0                `json:"resolved_by_exclusion,omitempty"`
	Contrasts           []WizardContrastV0                `json:"contrasts,omitempty"`
	EngineeringDefaults []WizardDefaultV0                 `json:"engineering_defaults"`
	Dossier             WizardDossierV0                   `json:"dossier"`
	GlossaryExpanded    bool                              `json:"glossary_expanded"`
	GlossaryResponse    *WizardGlossaryResponseV0         `json:"glossary_response,omitempty"`
	SpecComplete        bool                              `json:"spec_complete"`
	SpecPreview         *orquestafactory.AppSpecRequestV0 `json:"spec_preview,omitempty"`
	LaunchReady         bool                              `json:"launch_ready"`
}

type WizardGlossaryResponseV0 struct {
	Query      string `json:"query"`
	TermKey    string `json:"term_key"`
	HelpKey    string `json:"help_key"`
	ExampleKey string `json:"example_key,omitempty"`
}

type WizardContrastV0 struct {
	QuestionRef   string `json:"question_ref"`
	UserChoice    string `json:"user_choice"`
	Recommended   string `json:"recommended"`
	RationaleKey  string `json:"rationale_key"`
	Justification string `json:"justification,omitempty"`
}

type WizardDefaultV0 struct {
	Area       string `json:"area"`
	Value      string `json:"value"`
	WhyKey     string `json:"why_key"`
	HelpKey    string `json:"help_key"`
	ExampleKey string `json:"example_key,omitempty"`
}

type WizardFactV0 struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type WizardDecisionV0 = WebNuevaAppIntakeDecisionV0

type WizardDossierV0 struct {
	SchemaVersion          string                         `json:"schema_version"`
	DossierRef             string                         `json:"dossier_ref"`
	Ready                  bool                           `json:"ready"`
	Summary                string                         `json:"summary"`
	Markdown               string                         `json:"markdown"`
	Sections               []WizardDossierSectionV0       `json:"sections"`
	Diagrams               []WizardDossierDiagramV0       `json:"diagrams"`
	DecisionRefs           []string                       `json:"decision_refs"`
	RiskRefs               []string                       `json:"risk_refs"`
	Architecture           WizardDossierArchitectureV0    `json:"architecture"`
	I18N                   WizardDossierI18NV0            `json:"i18n"`
	Connectors             []WizardDossierConnectorV0     `json:"connectors"`
	Documentation          []WizardDossierDocumentationV0 `json:"documentation"`
	Infographics           []WizardDossierInfographicV0   `json:"infographics"`
	Decisions              []WizardDossierDecisionV0      `json:"decisions"`
	Alternatives           []WizardDossierAlternativeV0   `json:"alternatives"`
	AcceptanceBeforeLaunch []string                       `json:"acceptance_before_launch"`
}

type WizardDossierSectionV0 struct {
	SectionRef string `json:"section_ref"`
	Title      string `json:"title"`
	Markdown   string `json:"markdown"`
}

type WizardDossierDiagramV0 struct {
	DiagramRef string `json:"diagram_ref"`
	Kind       string `json:"kind"`
	Source     string `json:"source"`
	AltText    string `json:"alt_text"`
}

type WizardDossierArchitectureV0 struct {
	Style      string   `json:"style"`
	Layers     []string `json:"layers"`
	Ports      []string `json:"ports"`
	Adapters   []string `json:"adapters"`
	Boundaries []string `json:"boundaries"`
}

type WizardDossierI18NV0 struct {
	Enabled       bool     `json:"enabled"`
	DefaultLocale string   `json:"default_locale"`
	Locales       []string `json:"locales"`
	Plan          []string `json:"plan"`
}

type WizardDossierConnectorV0 struct {
	Tipo          string   `json:"tipo"`
	Nombre        string   `json:"nombre"`
	Proposito     string   `json:"proposito"`
	Direccion     string   `json:"direccion,omitempty"`
	Auth          string   `json:"auth,omitempty"`
	Criticidad    string   `json:"criticidad,omitempty"`
	Requerido     bool     `json:"requerido"`
	PortRef       string   `json:"port_ref"`
	AdapterRef    string   `json:"adapter_ref"`
	Restricciones []string `json:"restricciones,omitempty"`
}

type WizardDossierDocumentationV0 struct {
	ArtifactRef string   `json:"artifact_ref"`
	Title       string   `json:"title"`
	Purpose     string   `json:"purpose"`
	Sections    []string `json:"sections"`
}

type WizardDossierInfographicV0 struct {
	ArtifactRef string   `json:"artifact_ref"`
	Title       string   `json:"title"`
	Format      string   `json:"format"`
	Nodes       []string `json:"nodes"`
	Edges       []string `json:"edges"`
}

type WizardDossierDecisionV0 struct {
	Area      string `json:"area"`
	Decision  string `json:"decision"`
	Rationale string `json:"rationale"`
	Source    string `json:"source"`
}

type WizardDossierAlternativeV0 struct {
	Area      string `json:"area"`
	Option    string `json:"option"`
	Tradeoff  string `json:"tradeoff"`
	WhenToUse string `json:"when_to_use"`
}
