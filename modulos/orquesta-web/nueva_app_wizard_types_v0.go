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
