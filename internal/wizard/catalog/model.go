// Package catalog defines the pure, versioned question taxonomy used by the
// Wizard. It classifies no prose and owns no intake, Goal, provider or UI
// lifecycle. Domain packs are selected explicitly by opaque reference.
package catalog

type CatalogVersion struct{ value string }
type PackRef struct{ value string }
type TaxonomyRef struct{ value string }
type TermRef struct{ value string }
type QuestionRef struct{ value string }
type OptionRef struct{ value string }
type DefaultRef struct{ value string }
type SlotKey struct{ value string }
type MessageKey struct{ value string }

func NewCatalogVersion(value string) (CatalogVersion, error) {
	if !validMachineKey(value) {
		return CatalogVersion{}, domainError(ErrorInvalidArgument, "catalog_version")
	}
	return CatalogVersion{value: value}, nil
}

func NewPackRef(value string) (PackRef, error) {
	value, err := validRef("pack", value)
	return PackRef{value: value}, err
}

func NewTaxonomyRef(value string) (TaxonomyRef, error) {
	value, err := validRef("taxonomy", value)
	return TaxonomyRef{value: value}, err
}

func NewTermRef(value string) (TermRef, error) {
	value, err := validRef("term", value)
	return TermRef{value: value}, err
}

func NewQuestionRef(value string) (QuestionRef, error) {
	value, err := validRef("question", value)
	return QuestionRef{value: value}, err
}

func NewOptionRef(value string) (OptionRef, error) {
	value, err := validRef("option", value)
	return OptionRef{value: value}, err
}

func NewDefaultRef(value string) (DefaultRef, error) {
	value, err := validRef("default", value)
	return DefaultRef{value: value}, err
}

func NewSlotKey(value string) (SlotKey, error) {
	if !validMachineKey(value) {
		return SlotKey{}, domainError(ErrorInvalidArgument, "slot_key")
	}
	return SlotKey{value: value}, nil
}

func NewMessageKey(value string) (MessageKey, error) {
	if !validMachineKey(value) || !containsDot(value) {
		return MessageKey{}, domainError(ErrorInvalidMessageKey, "message_key")
	}
	return MessageKey{value: value}, nil
}

func (value CatalogVersion) String() string { return value.value }
func (value PackRef) String() string        { return value.value }
func (value TaxonomyRef) String() string    { return value.value }
func (value TermRef) String() string        { return value.value }
func (value QuestionRef) String() string    { return value.value }
func (value OptionRef) String() string      { return value.value }
func (value DefaultRef) String() string     { return value.value }
func (value SlotKey) String() string        { return value.value }
func (value MessageKey) String() string     { return value.value }

// RoadmapCapabilityRef is traceability metadata only. A reference means this
// partial catalog is relevant to a roadmap capability; it never means the
// capability is implemented, exercised or accredited.
type RoadmapCapabilityRef struct{ value string }

func NewRoadmapCapabilityRef(value string) (RoadmapCapabilityRef, error) {
	if !validRoadmapCapabilityRef(value) {
		return RoadmapCapabilityRef{}, domainError(
			ErrorInvalidRef,
			"roadmap_capability_ref",
		)
	}
	return RoadmapCapabilityRef{value: value}, nil
}

func (value RoadmapCapabilityRef) String() string { return value.value }

type AnswerKind string

const (
	AnswerChoice   AnswerKind = "choice"
	AnswerFreeText AnswerKind = "free_text"
)

type DecisionKind string

const (
	DecisionProduct          DecisionKind = "product"
	DecisionTechnicalDefault DecisionKind = "technical_default"
)

type TermInput struct {
	Ref        TermRef
	LabelKey   MessageKey
	HelpKey    MessageKey
	ExampleKey MessageKey
}

type Term struct {
	ref        TermRef
	labelKey   MessageKey
	helpKey    MessageKey
	exampleKey MessageKey
}

func NewTerm(input TermInput) (Term, error) {
	if input.Ref.value == "" {
		return Term{}, domainError(ErrorInvalidRef, "term.ref")
	}
	if err := validatePresentationKeys(
		"term", input.LabelKey, input.HelpKey, input.ExampleKey,
	); err != nil {
		return Term{}, err
	}
	return Term{
		ref: input.Ref, labelKey: input.LabelKey,
		helpKey: input.HelpKey, exampleKey: input.ExampleKey,
	}, nil
}

func (value Term) Ref() TermRef           { return value.ref }
func (value Term) LabelKey() MessageKey   { return value.labelKey }
func (value Term) HelpKey() MessageKey    { return value.helpKey }
func (value Term) ExampleKey() MessageKey { return value.exampleKey }

type TaxonomyInput struct {
	Ref        TaxonomyRef
	LabelKey   MessageKey
	HelpKey    MessageKey
	ExampleKey MessageKey
	Terms      []Term
}

type Taxonomy struct {
	ref        TaxonomyRef
	labelKey   MessageKey
	helpKey    MessageKey
	exampleKey MessageKey
	terms      []Term
}

func NewTaxonomy(input TaxonomyInput) (Taxonomy, error) {
	if input.Ref.value == "" {
		return Taxonomy{}, domainError(ErrorInvalidRef, "taxonomy.ref")
	}
	if err := validatePresentationKeys(
		"taxonomy", input.LabelKey, input.HelpKey, input.ExampleKey,
	); err != nil {
		return Taxonomy{}, err
	}
	terms := cloneTerms(input.Terms)
	if len(terms) == 0 {
		return Taxonomy{}, domainError(ErrorInvalidArgument, "taxonomy.terms")
	}
	if err := uniqueTerms(terms, "taxonomy.terms"); err != nil {
		return Taxonomy{}, err
	}
	sortTerms(terms)
	return Taxonomy{
		ref: input.Ref, labelKey: input.LabelKey, helpKey: input.HelpKey,
		exampleKey: input.ExampleKey, terms: terms,
	}, nil
}

func (value Taxonomy) Ref() TaxonomyRef       { return value.ref }
func (value Taxonomy) LabelKey() MessageKey   { return value.labelKey }
func (value Taxonomy) HelpKey() MessageKey    { return value.helpKey }
func (value Taxonomy) ExampleKey() MessageKey { return value.exampleKey }
func (value Taxonomy) Terms() []Term          { return cloneTerms(value.terms) }

type OptionInput struct {
	Ref          OptionRef
	TermRef      TermRef
	LabelKey     MessageKey
	HelpKey      MessageKey
	ExampleKey   MessageKey
	RationaleKey MessageKey
	Recommended  bool
}

type Option struct {
	ref          OptionRef
	termRef      TermRef
	labelKey     MessageKey
	helpKey      MessageKey
	exampleKey   MessageKey
	rationaleKey MessageKey
	recommended  bool
}

func NewOption(input OptionInput) (Option, error) {
	if input.Ref.value == "" {
		return Option{}, domainError(ErrorInvalidRef, "option.ref")
	}
	if input.TermRef.value == "" {
		return Option{}, domainError(ErrorInvalidRef, "option.term_ref")
	}
	if err := validatePresentationKeys(
		"option", input.LabelKey, input.HelpKey, input.ExampleKey,
	); err != nil {
		return Option{}, err
	}
	if input.RationaleKey.value == "" {
		return Option{}, domainError(ErrorInvalidMessageKey, "option.rationale_key")
	}
	return Option{
		ref: input.Ref, termRef: input.TermRef, labelKey: input.LabelKey,
		helpKey: input.HelpKey, exampleKey: input.ExampleKey,
		rationaleKey: input.RationaleKey, recommended: input.Recommended,
	}, nil
}

func (value Option) Ref() OptionRef           { return value.ref }
func (value Option) TermRef() TermRef         { return value.termRef }
func (value Option) LabelKey() MessageKey     { return value.labelKey }
func (value Option) HelpKey() MessageKey      { return value.helpKey }
func (value Option) ExampleKey() MessageKey   { return value.exampleKey }
func (value Option) RationaleKey() MessageKey { return value.rationaleKey }
func (value Option) Recommended() bool        { return value.recommended }

type QuestionInput struct {
	Ref          QuestionRef
	Slot         SlotKey
	AnswerKind   AnswerKind
	DecisionKind DecisionKind
	TaxonomyRef  TaxonomyRef
	PromptKey    MessageKey
	WhyKey       MessageKey
	HelpKey      MessageKey
	ExampleKey   MessageKey
	Options      []Option
}

type Question struct {
	ref          QuestionRef
	slot         SlotKey
	answerKind   AnswerKind
	decisionKind DecisionKind
	taxonomyRef  TaxonomyRef
	promptKey    MessageKey
	whyKey       MessageKey
	helpKey      MessageKey
	exampleKey   MessageKey
	options      []Option
}

func NewQuestion(input QuestionInput) (Question, error) {
	if input.Ref.value == "" {
		return Question{}, domainError(ErrorInvalidRef, "question.ref")
	}
	if input.Slot.value == "" {
		return Question{}, domainError(ErrorInvalidArgument, "question.slot")
	}
	if input.DecisionKind != DecisionProduct {
		return Question{}, domainError(ErrorInvalidArgument, "question.decision_kind")
	}
	if err := validatePresentationKeys(
		"question", input.PromptKey, input.WhyKey, input.HelpKey, input.ExampleKey,
	); err != nil {
		return Question{}, err
	}
	options := cloneOptions(input.Options)
	switch input.AnswerKind {
	case AnswerChoice:
		if input.TaxonomyRef.value == "" {
			return Question{}, domainError(ErrorInvalidRef, "question.taxonomy_ref")
		}
		if len(options) < 2 {
			return Question{}, domainError(ErrorInvalidArgument, "question.options")
		}
		if err := uniqueOptions(options, "question.options"); err != nil {
			return Question{}, err
		}
		recommended := 0
		for _, option := range options {
			if option.recommended {
				recommended++
			}
		}
		if recommended != 1 {
			return Question{}, domainError(ErrorRecommendationCount, "question.options")
		}
		sortOptions(options)
	case AnswerFreeText:
		if input.TaxonomyRef.value != "" {
			return Question{}, domainError(ErrorInvalidArgument, "question.taxonomy_ref")
		}
		if len(options) != 0 {
			return Question{}, domainError(ErrorInvalidArgument, "question.options")
		}
	default:
		return Question{}, domainError(ErrorInvalidArgument, "question.answer_kind")
	}
	return Question{
		ref: input.Ref, slot: input.Slot, answerKind: input.AnswerKind,
		decisionKind: input.DecisionKind, taxonomyRef: input.TaxonomyRef,
		promptKey: input.PromptKey, whyKey: input.WhyKey,
		helpKey: input.HelpKey, exampleKey: input.ExampleKey, options: options,
	}, nil
}

func (value Question) Ref() QuestionRef           { return value.ref }
func (value Question) Slot() SlotKey              { return value.slot }
func (value Question) AnswerKind() AnswerKind     { return value.answerKind }
func (value Question) DecisionKind() DecisionKind { return value.decisionKind }
func (value Question) TaxonomyRef() TaxonomyRef   { return value.taxonomyRef }
func (value Question) PromptKey() MessageKey      { return value.promptKey }
func (value Question) WhyKey() MessageKey         { return value.whyKey }
func (value Question) HelpKey() MessageKey        { return value.helpKey }
func (value Question) ExampleKey() MessageKey     { return value.exampleKey }
func (value Question) Options() []Option          { return cloneOptions(value.options) }

type TechnicalDefaultInput struct {
	Ref          DefaultRef
	Slot         SlotKey
	Value        TermRef
	LabelKey     MessageKey
	HelpKey      MessageKey
	ExampleKey   MessageKey
	RationaleKey MessageKey
}

// TechnicalDefault is disclosed metadata, never a product question or silent
// mutation. Application policy decides if and when a default is applied.
type TechnicalDefault struct {
	ref          DefaultRef
	slot         SlotKey
	value        TermRef
	labelKey     MessageKey
	helpKey      MessageKey
	exampleKey   MessageKey
	rationaleKey MessageKey
}

func NewTechnicalDefault(input TechnicalDefaultInput) (TechnicalDefault, error) {
	if input.Ref.value == "" {
		return TechnicalDefault{}, domainError(ErrorInvalidRef, "default.ref")
	}
	if input.Slot.value == "" || input.Value.value == "" {
		return TechnicalDefault{}, domainError(ErrorInvalidArgument, "default.value")
	}
	if err := validatePresentationKeys(
		"default", input.LabelKey, input.HelpKey, input.ExampleKey,
	); err != nil {
		return TechnicalDefault{}, err
	}
	if input.RationaleKey.value == "" {
		return TechnicalDefault{}, domainError(ErrorInvalidMessageKey, "default.rationale_key")
	}
	return TechnicalDefault{
		ref: input.Ref, slot: input.Slot, value: input.Value,
		labelKey: input.LabelKey, helpKey: input.HelpKey,
		exampleKey: input.ExampleKey, rationaleKey: input.RationaleKey,
	}, nil
}

func (value TechnicalDefault) Ref() DefaultRef            { return value.ref }
func (value TechnicalDefault) Slot() SlotKey              { return value.slot }
func (value TechnicalDefault) Value() TermRef             { return value.value }
func (value TechnicalDefault) DecisionKind() DecisionKind { return DecisionTechnicalDefault }
func (value TechnicalDefault) LabelKey() MessageKey       { return value.labelKey }
func (value TechnicalDefault) HelpKey() MessageKey        { return value.helpKey }
func (value TechnicalDefault) ExampleKey() MessageKey     { return value.exampleKey }
func (value TechnicalDefault) RationaleKey() MessageKey   { return value.rationaleKey }

type PackInput struct {
	Ref                   PackRef
	LabelKey              MessageKey
	HelpKey               MessageKey
	ExampleKey            MessageKey
	RoadmapCapabilityRefs []RoadmapCapabilityRef
	Taxonomies            []Taxonomy
	Questions             []Question
	Defaults              []TechnicalDefault
}

type Pack struct {
	ref                   PackRef
	labelKey              MessageKey
	helpKey               MessageKey
	exampleKey            MessageKey
	roadmapCapabilityRefs []RoadmapCapabilityRef
	taxonomies            []Taxonomy
	questions             []Question
	defaults              []TechnicalDefault
}

func NewPack(input PackInput) (Pack, error) {
	if input.Ref.value == "" {
		return Pack{}, domainError(ErrorInvalidRef, "pack.ref")
	}
	if err := validatePresentationKeys(
		"pack", input.LabelKey, input.HelpKey, input.ExampleKey,
	); err != nil {
		return Pack{}, err
	}
	roadmapCapabilityRefs, err := normalizeRoadmapCapabilityRefs(
		input.RoadmapCapabilityRefs,
	)
	if err != nil {
		return Pack{}, err
	}
	taxonomies := cloneTaxonomies(input.Taxonomies)
	questions := cloneQuestions(input.Questions)
	defaults := cloneDefaults(input.Defaults)
	if len(taxonomies)+len(questions)+len(defaults) == 0 {
		return Pack{}, domainError(ErrorInvalidArgument, "pack.content")
	}
	if err := uniqueTaxonomies(taxonomies, "pack.taxonomies"); err != nil {
		return Pack{}, err
	}
	if err := uniqueQuestions(questions, "pack.questions"); err != nil {
		return Pack{}, err
	}
	if err := uniqueDefaults(defaults, "pack.defaults"); err != nil {
		return Pack{}, err
	}
	sortTaxonomies(taxonomies)
	sortQuestions(questions)
	sortDefaults(defaults)
	return Pack{
		ref: input.Ref, labelKey: input.LabelKey, helpKey: input.HelpKey,
		exampleKey:            input.ExampleKey,
		roadmapCapabilityRefs: roadmapCapabilityRefs,
		taxonomies:            taxonomies, questions: questions, defaults: defaults,
	}, nil
}

func (value Pack) Ref() PackRef           { return value.ref }
func (value Pack) LabelKey() MessageKey   { return value.labelKey }
func (value Pack) HelpKey() MessageKey    { return value.helpKey }
func (value Pack) ExampleKey() MessageKey { return value.exampleKey }

// RoadmapCapabilityRefs returns related roadmap IDs for audit/navigation. It
// is not a completion, coverage or accreditation claim.
func (value Pack) RoadmapCapabilityRefs() []RoadmapCapabilityRef {
	return cloneRoadmapCapabilityRefs(value.roadmapCapabilityRefs)
}

func (value Pack) Taxonomies() []Taxonomy       { return cloneTaxonomies(value.taxonomies) }
func (value Pack) Questions() []Question        { return cloneQuestions(value.questions) }
func (value Pack) Defaults() []TechnicalDefault { return cloneDefaults(value.defaults) }
