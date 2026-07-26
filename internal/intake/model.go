// Package intake contains pure, deterministic rules for one versioned intake
// state. Chat and form are mutation origins, never state identities.
package intake

const StateSchema = "orquesta.intake.state.v1"

// MaxAnswerTextRunes is the single domain contract shared by every intake
// producer. Transport request-byte limits remain adapter configuration.
const MaxAnswerTextRunes = 4096

type Ref string
type IssueRef string
type QuestionRef string
type OptionRef string
type Revision uint64
type MessageKey string

// DerivationIdentity identifies the exact deterministic implementation that
// compiled a mutation. Zero means a historical/operator-authored mutation.
// It is generic Intake causality: no product, Wizard or provider name is
// interpreted by this package.
type DerivationIdentity struct {
	Schema         string `json:"schema"`
	Version        string `json:"version"`
	SemanticDigest string `json:"semantic_digest"`
}

func (value DerivationIdentity) IsZero() bool {
	return value.Schema == "" && value.Version == "" && value.SemanticDigest == ""
}

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
	// AcceptsText marks an option whose selection must carry a non-empty
	// AnswerText. It is explicit domain metadata, never inferred from refs or
	// human-facing copy.
	AcceptsText bool `json:"accepts_text,omitempty"`
}

// Question is valid only when DerivedFrom refers to one or more recorded gaps
// or contradictions.
type Question struct {
	Ref         QuestionRef   `json:"ref"`
	DerivedFrom []IssueRef    `json:"derived_from"`
	DependsOn   []QuestionRef `json:"depends_on,omitempty"`
	PromptKey   MessageKey    `json:"prompt_key"`
	WhyKey      MessageKey    `json:"why_key"`
	Options     []Option      `json:"options"`
}

type Choice struct {
	QuestionRef QuestionRef `json:"question_ref"`
	OptionRef   OptionRef   `json:"option_ref"`
	AnswerText  string      `json:"answer_text,omitempty"`
}

// Change is one compare-and-swap mutation against the shared intake state.
// It cannot create a state; NewState is the sole origin-neutral constructor.
type Change struct {
	StateRef         Ref        `json:"state_ref"`
	ExpectedRevision Revision   `json:"expected_revision"`
	Origin           Origin     `json:"origin"`
	Issues           []Issue    `json:"issues,omitempty"`
	Questions        []Question `json:"questions,omitempty"`
	// QuestionRevisions replaces the active payload of an existing derived
	// question, or restores its latest tombstoned projection, without
	// rewriting immutable versions. Only the derivation that created the
	// latest version may revise it.
	QuestionRevisions []Question `json:"question_revisions,omitempty"`
	// QuestionRetirements appends a tombstone for active derived questions
	// that their exact creating derivation no longer emits.
	QuestionRetirements []QuestionRef      `json:"question_retirements,omitempty"`
	Choices             []Choice           `json:"choices,omitempty"`
	Derivation          DerivationIdentity `json:"derivation,omitempty,omitzero"`
}

// QuestionVersion is the append-only provenance of one question projection.
// ReplacesRevision is zero for the first projection and points at the exact
// active version for a later reconciliation.
type QuestionVersion struct {
	Question         Question `json:"question"`
	Revision         Revision `json:"revision"`
	ReplacesRevision Revision `json:"replaces_revision,omitempty"`
	Retired          bool     `json:"retired,omitempty"`
}

// Decision stores the actual choice and the recommendation as simultaneous
// facts. They remain distinct when the user chooses an alternative.
type Decision struct {
	QuestionRef             QuestionRef `json:"question_ref"`
	Choice                  OptionRef   `json:"choice"`
	AnswerText              string      `json:"answer_text,omitempty"`
	Recommendation          OptionRef   `json:"recommendation"`
	RecommendationRationale MessageKey  `json:"recommendation_rationale_key"`
	Origin                  Origin      `json:"origin"`
	Revision                Revision    `json:"revision"`
}

// Mutation records which view contributed to the single revision sequence.
// It is evidence about the transition, not a second channel lifecycle.
type Mutation struct {
	Origin           Origin             `json:"origin"`
	Revision         Revision           `json:"revision"`
	IssuesAdded      int                `json:"issues_added"`
	QuestionsAdded   int                `json:"questions_added"`
	QuestionsRevised int                `json:"questions_revised,omitempty"`
	QuestionsRetired int                `json:"questions_retired,omitempty"`
	ChoicesRecorded  int                `json:"choices_recorded"`
	QuestionRound    uint32             `json:"question_round"`
	Derivation       DerivationIdentity `json:"derivation,omitempty,omitzero"`
}

// AcceptRecommendationsRequest identifies one exact shared-state revision and
// question round. BuildAcceptRecommendationsChange compiles it to the regular
// Change contract so durable CAS, fingerprints and replay need no side path.
type AcceptRecommendationsRequest struct {
	StateRef         Ref      `json:"state_ref"`
	ExpectedRevision Revision `json:"expected_revision"`
	Origin           Origin   `json:"origin"`
	QuestionRound    uint32   `json:"question_round"`
}

type ContextKind string

const (
	ContextClarification ContextKind = "clarification"
	ContextHelp          ContextKind = "help"
)

// ContextRequest is a read against an exact intake revision. An empty
// QuestionRefs list means every question; a non-empty list preserves caller
// order. It never becomes a Mutation.
type ContextRequest struct {
	StateRef         Ref           `json:"state_ref"`
	ExpectedRevision Revision      `json:"expected_revision"`
	Origin           Origin        `json:"origin"`
	Kind             ContextKind   `json:"kind"`
	QuestionRefs     []QuestionRef `json:"question_refs,omitempty"`
}

type QuestionContext struct {
	Question         Question          `json:"question"`
	CurrentDecision  *Decision         `json:"current_decision,omitempty"`
	ReopenedDecision *ReopenedDecision `json:"reopened_decision,omitempty"`
}

// Context is a deterministic re-emission of already recorded intake facts,
// including any durable free-text answer. It grants no mutation authority.
type Context struct {
	StateRef  Ref               `json:"state_ref"`
	Revision  Revision          `json:"revision"`
	Origin    Origin            `json:"origin"`
	Kind      ContextKind       `json:"kind"`
	Issues    []Issue           `json:"issues,omitempty"`
	Questions []QuestionContext `json:"questions,omitempty"`
}

type DecisionChange struct {
	QuestionRef QuestionRef `json:"question_ref"`
	Revision    Revision    `json:"revision"`
}

// ReopenedDecision is a derived view, not another state collection. Previous
// remains in Decisions for audit while CurrentDecision reports it as inactive.
type ReopenedDecision struct {
	QuestionRef      QuestionRef      `json:"question_ref"`
	PreviousDecision Decision         `json:"previous_decision"`
	InvalidatedBy    []DecisionChange `json:"invalidated_by"`
}
