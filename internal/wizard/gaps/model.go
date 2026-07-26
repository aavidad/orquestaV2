// Package gaps detects Wizard gaps and semantic contradictions from typed,
// explicit facts. It owns no state, lifecycle, provider, persistence or UI.
package gaps

import (
	"orquesta/internal/intake"
	"orquesta/internal/wizard/catalog"
)

const SchemaVersion = "orquesta.wizard.gaps.v1"

type DimensionRef string
type RuleRef string
type SlotKey string
type MessageKey string
type QuestionRef string
type OptionRef string
type IssueRef string

const (
	DimensionU1  DimensionRef = "U1"
	DimensionU2  DimensionRef = "U2"
	DimensionU3  DimensionRef = "U3"
	DimensionU4  DimensionRef = "U4"
	DimensionU5  DimensionRef = "U5"
	DimensionU6  DimensionRef = "U6"
	DimensionU7  DimensionRef = "U7"
	DimensionU8  DimensionRef = "U8"
	DimensionU9  DimensionRef = "U9"
	DimensionU10 DimensionRef = "U10"
	DimensionU11 DimensionRef = "U11"
	DimensionU12 DimensionRef = "U12"

	DimensionT1 DimensionRef = "T1"
	DimensionT2 DimensionRef = "T2"
	DimensionT3 DimensionRef = "T3"
	DimensionT4 DimensionRef = "T4"
	DimensionT5 DimensionRef = "T5"
	DimensionT6 DimensionRef = "T6"
	DimensionT7 DimensionRef = "T7"
	DimensionT8 DimensionRef = "T8"
)

const (
	RuleR1 RuleRef = "R1"
	RuleR2 RuleRef = "R2"
	RuleR3 RuleRef = "R3"
	RuleR4 RuleRef = "R4"
	RuleR5 RuleRef = "R5"
	RuleR6 RuleRef = "R6"
	RuleR7 RuleRef = "R7"
	RuleR8 RuleRef = "R8"
)

type Layer string

const (
	LayerUniversal Layer = "universal"
	LayerTechnical Layer = "technical"
)

type DecisionKind string

const (
	DecisionProduct          DecisionKind = "product"
	DecisionTechnical        DecisionKind = "technical"
	DecisionTechnicalDefault DecisionKind = "technical_default"
)

type IssueKind string

const (
	IssueGap           IssueKind = "gap"
	IssueContradiction IssueKind = "contradiction"
)

type OptionKind string

const (
	OptionPreset   OptionKind = "preset"
	OptionFreeText OptionKind = "free_text"
)

// Facts contains only typed facts supplied by a caller. This package never
// classifies objective prose or derives these values from keywords.
type Facts struct {
	Surface                Surface
	SharingIntent          SharingIntent
	CorporateIdentity      Declaration
	TargetUsers            Declaration
	IntegrationAuth        Declaration
	IntegrationCriticality Declaration
}

type Surface string

const (
	SurfaceUnspecified   Surface = ""
	SurfaceHumanUI       Surface = "human_ui"
	SurfaceNativeMobile  Surface = "native_mobile"
	SurfaceNativeDesktop Surface = "native_desktop"
	SurfaceServerService Surface = "server_service"
	SurfaceKernelModule  Surface = "kernel_module"
)

type SharingIntent string

const (
	SharingUnspecified SharingIntent = ""
	SharingAmbiguous   SharingIntent = "ambiguous"
	SharingPersonal    SharingIntent = "personal"
	SharingShared      SharingIntent = "shared"
)

type Declaration string

const (
	DeclarationUnspecified Declaration = ""
	DeclarationMissing     Declaration = "missing"
	DeclarationDeclared    Declaration = "declared"
)

// Selection is explicit and dimension-scoped. FreeText is required only for
// a real free-text option; it is never parsed or classified by this package.
type Selection struct {
	Dimension DimensionRef
	Option    OptionRef
	FreeText  string
}

// QuestionSelection answers a rule- or explicit-pack question for one pure
// evaluation. Keeping it separate prevents an opaque pack ref being mistaken
// for a U/T dimension. A caller projects it to the shared intake Choice.
type QuestionSelection struct {
	Question QuestionRef
	Option   OptionRef
	FreeText string
}

type Input struct {
	Facts              Facts
	Selections         []Selection
	QuestionSelections []QuestionSelection
	PackRefs           []catalog.PackRef
}

type Option struct {
	ref          OptionRef
	kind         OptionKind
	labelKey     MessageKey
	helpKey      MessageKey
	exampleKey   MessageKey
	rationaleKey MessageKey
	recommended  bool
}

func (value Option) Ref() OptionRef           { return value.ref }
func (value Option) Kind() OptionKind         { return value.kind }
func (value Option) LabelKey() MessageKey     { return value.labelKey }
func (value Option) HelpKey() MessageKey      { return value.helpKey }
func (value Option) ExampleKey() MessageKey   { return value.exampleKey }
func (value Option) RationaleKey() MessageKey { return value.rationaleKey }
func (value Option) Recommended() bool        { return value.recommended }

type Issue struct {
	ref       IssueRef
	kind      IssueKind
	ruleRef   RuleRef
	dimension DimensionRef
	field     SlotKey
	detailKey MessageKey
	dependsOn []DimensionRef
}

func (value Issue) Ref() IssueRef           { return value.ref }
func (value Issue) Kind() IssueKind         { return value.kind }
func (value Issue) RuleRef() RuleRef        { return value.ruleRef }
func (value Issue) Dimension() DimensionRef { return value.dimension }
func (value Issue) Field() SlotKey          { return value.field }
func (value Issue) DetailKey() MessageKey   { return value.detailKey }
func (value Issue) DependsOn() []DimensionRef {
	return append([]DimensionRef(nil), value.dependsOn...)
}

// IntakeIssue projects the pure finding to the shared intake mutation type.
func (value Issue) IntakeIssue() intake.Issue {
	return intake.Issue{
		Ref:       intake.IssueRef(value.ref),
		Kind:      intake.IssueKind(value.kind),
		Field:     string(value.field),
		DetailKey: intake.MessageKey(value.detailKey),
	}
}

type Question struct {
	ref          QuestionRef
	dimension    DimensionRef
	packRef      catalog.PackRef
	slot         SlotKey
	decisionKind DecisionKind
	derivedFrom  []IssueRef
	dependsOn    []QuestionRef
	promptKey    MessageKey
	whyKey       MessageKey
	helpKey      MessageKey
	exampleKey   MessageKey
	options      []Option
}

func (value Question) Ref() QuestionRef           { return value.ref }
func (value Question) Dimension() DimensionRef    { return value.dimension }
func (value Question) PackRef() catalog.PackRef   { return value.packRef }
func (value Question) Slot() SlotKey              { return value.slot }
func (value Question) DecisionKind() DecisionKind { return value.decisionKind }
func (value Question) DerivedFrom() []IssueRef {
	return append([]IssueRef(nil), value.derivedFrom...)
}
func (value Question) DependsOn() []QuestionRef {
	return append([]QuestionRef(nil), value.dependsOn...)
}
func (value Question) PromptKey() MessageKey  { return value.promptKey }
func (value Question) WhyKey() MessageKey     { return value.whyKey }
func (value Question) HelpKey() MessageKey    { return value.helpKey }
func (value Question) ExampleKey() MessageKey { return value.exampleKey }
func (value Question) Options() []Option      { return append([]Option(nil), value.options...) }

func (value Question) RecommendedOption() (Option, bool) {
	var found Option
	count := 0
	for _, option := range value.options {
		if option.recommended {
			found, count = option, count+1
		}
	}
	return found, count == 1
}

// IntakeQuestion preserves causal refs, recommendation and the explicit
// free-text capability. Presentation help stays on Question/Option until
// intake grows that public schema.
func (value Question) IntakeQuestion() intake.Question {
	options := make([]intake.Option, len(value.options))
	for index, option := range value.options {
		options[index] = intake.Option{
			Ref:          intake.OptionRef(option.ref),
			LabelKey:     intake.MessageKey(option.labelKey),
			RationaleKey: intake.MessageKey(option.rationaleKey),
			Recommended:  option.recommended,
			AcceptsText:  option.kind == OptionFreeText,
		}
	}
	derivedFrom := make([]intake.IssueRef, len(value.derivedFrom))
	for index, ref := range value.derivedFrom {
		derivedFrom[index] = intake.IssueRef(ref)
	}
	dependsOn := make([]intake.QuestionRef, len(value.dependsOn))
	for index, ref := range value.dependsOn {
		dependsOn[index] = intake.QuestionRef(ref)
	}
	return intake.Question{
		Ref:         intake.QuestionRef(value.ref),
		DerivedFrom: derivedFrom,
		DependsOn:   dependsOn,
		PromptKey:   intake.MessageKey(value.promptKey),
		WhyKey:      intake.MessageKey(value.whyKey),
		Options:     options,
	}
}

// DefaultProposal is disclosed and unapplied. It cannot silently become a
// product decision or mutate intake state.
type DefaultProposal struct {
	dimension    DimensionRef
	decisionKind DecisionKind
	option       OptionRef
	labelKey     MessageKey
	helpKey      MessageKey
	exampleKey   MessageKey
	rationaleKey MessageKey
}

func (value DefaultProposal) Dimension() DimensionRef    { return value.dimension }
func (value DefaultProposal) DecisionKind() DecisionKind { return value.decisionKind }
func (value DefaultProposal) Option() OptionRef          { return value.option }
func (value DefaultProposal) LabelKey() MessageKey       { return value.labelKey }
func (value DefaultProposal) HelpKey() MessageKey        { return value.helpKey }
func (value DefaultProposal) ExampleKey() MessageKey     { return value.exampleKey }
func (value DefaultProposal) RationaleKey() MessageKey   { return value.rationaleKey }
func (value DefaultProposal) ImplicitlyApplied() bool    { return false }

type Result struct {
	schemaVersion string
	issues        []Issue
	questions     []Question
	defaults      []DefaultProposal
	packRefs      []catalog.PackRef
}

func (value Result) SchemaVersion() string { return value.schemaVersion }
func (value Result) Issues() []Issue {
	out := append([]Issue(nil), value.issues...)
	for index := range out {
		out[index].dependsOn = append([]DimensionRef(nil), out[index].dependsOn...)
	}
	return out
}
func (value Result) Questions() []Question {
	out := append([]Question(nil), value.questions...)
	for index := range out {
		out[index].derivedFrom = append([]IssueRef(nil), out[index].derivedFrom...)
		out[index].dependsOn = append([]QuestionRef(nil), out[index].dependsOn...)
		out[index].options = append([]Option(nil), out[index].options...)
	}
	return out
}
func (value Result) DefaultProposals() []DefaultProposal {
	return append([]DefaultProposal(nil), value.defaults...)
}
func (value Result) PackRefs() []catalog.PackRef {
	return append([]catalog.PackRef(nil), value.packRefs...)
}
