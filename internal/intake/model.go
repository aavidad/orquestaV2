// Package intake contains pure, deterministic rules for one versioned intake
// state. Chat and form are mutation origins, never state identities.
package intake

const StateSchema = "orquesta.intake.state.v1"

type Ref string
type IssueRef string
type QuestionRef string
type OptionRef string
type Revision uint64
type MessageKey string

type Origin string

const (
	OriginChat Origin = "chat"
	OriginForm Origin = "form"
)

type IssueKind string

const (
	IssueGap           IssueKind = "gap"
	IssueContradiction IssueKind = "contradiction"
)

// Policy is supplied by composition. This package intentionally has no
// default round count; the canonical configuration is owned elsewhere.
type Policy struct {
	MaxQuestionRounds uint32 `json:"max_question_rounds"`
}

// Issue is a missing or contradictory fact from which questions can be
// derived. DetailKey is catalog-owned human copy; Field is a machine key.
type Issue struct {
	Ref       IssueRef   `json:"ref"`
	Kind      IssueKind  `json:"kind"`
	Field     string     `json:"field"`
	DetailKey MessageKey `json:"detail_key"`
}

// Option keeps all human-facing copy behind catalog keys. Exactly one option
// per question must set Recommended.
type Option struct {
	Ref          OptionRef  `json:"ref"`
	LabelKey     MessageKey `json:"label_key"`
	RationaleKey MessageKey `json:"rationale_key"`
	Recommended  bool       `json:"recommended"`
}

// Question is valid only when DerivedFrom refers to one or more recorded gaps
// or contradictions.
type Question struct {
	Ref         QuestionRef `json:"ref"`
	DerivedFrom []IssueRef  `json:"derived_from"`
	PromptKey   MessageKey  `json:"prompt_key"`
	WhyKey      MessageKey  `json:"why_key"`
	Options     []Option    `json:"options"`
}

type Choice struct {
	QuestionRef QuestionRef `json:"question_ref"`
	OptionRef   OptionRef   `json:"option_ref"`
}

// Change is one compare-and-swap mutation against the shared intake state.
// It cannot create a state; NewState is the sole origin-neutral constructor.
type Change struct {
	StateRef         Ref        `json:"state_ref"`
	ExpectedRevision Revision   `json:"expected_revision"`
	Origin           Origin     `json:"origin"`
	Issues           []Issue    `json:"issues,omitempty"`
	Questions        []Question `json:"questions,omitempty"`
	Choices          []Choice   `json:"choices,omitempty"`
}

// Decision stores the actual choice and the recommendation as simultaneous
// facts. They remain distinct when the user chooses an alternative.
type Decision struct {
	QuestionRef             QuestionRef `json:"question_ref"`
	Choice                  OptionRef   `json:"choice"`
	Recommendation          OptionRef   `json:"recommendation"`
	RecommendationRationale MessageKey  `json:"recommendation_rationale_key"`
	Origin                  Origin      `json:"origin"`
	Revision                Revision    `json:"revision"`
}

// Mutation records which view contributed to the single revision sequence.
// It is evidence about the transition, not a second channel lifecycle.
type Mutation struct {
	Origin          Origin   `json:"origin"`
	Revision        Revision `json:"revision"`
	IssuesAdded     int      `json:"issues_added"`
	QuestionsAdded  int      `json:"questions_added"`
	ChoicesRecorded int      `json:"choices_recorded"`
	QuestionRound   uint32   `json:"question_round"`
}
