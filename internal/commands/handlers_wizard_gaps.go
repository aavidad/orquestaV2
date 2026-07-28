package commands

import (
	"context"
	"encoding/json"

	"orquesta/internal/application"
	"orquesta/internal/intake"
	"orquesta/internal/wizard/catalog"
	"orquesta/internal/wizard/gaps"
)

type applyWizardGapsInput struct {
	IntakeRef        string               `json:"intake_ref"`
	ExpectedRevision intake.Revision      `json:"expected_revision"`
	Origin           intake.Origin        `json:"origin"`
	Facts            wizardGapsFactsInput `json:"facts"`
	PackRefs         []string             `json:"pack_refs"`
}

type wizardGapsFactsInput struct {
	Surface                gaps.Surface       `json:"surface"`
	SharingIntent          gaps.SharingIntent `json:"sharing_intent"`
	CorporateIdentity      gaps.Declaration   `json:"corporate_identity"`
	TargetUsers            gaps.Declaration   `json:"target_users"`
	IntegrationAuth        gaps.Declaration   `json:"integration_auth"`
	IntegrationCriticality gaps.Declaration   `json:"integration_criticality"`
}

type wizardGapsResultView struct {
	Intake                intakeMutationView           `json:"intake"`
	Evaluation            wizardGapsEvaluationView     `json:"evaluation"`
	EvaluationReplayExact bool                         `json:"evaluation_replay_exact"`
	RequestRefReserved    bool                         `json:"request_ref_reserved"`
	RequestOutcome        wizardGapsRequestOutcomeView `json:"request_outcome"`
	InputDurability       wizardGapsDurabilityView     `json:"input_durability"`
	EvaluatorIdentity     intake.DerivationIdentity    `json:"evaluator_identity"`
}

type wizardGapsRequestOutcomeView struct {
	Kind       application.WizardGapsRequestOutcomeKind `json:"kind"`
	ReceiptRef string                                   `json:"receipt_ref"`
}

type wizardGapsDurabilityView struct {
	Selections string `json:"selections"`
	Facts      string `json:"facts"`
	PackRefs   string `json:"pack_refs"`
}

type wizardGapsEvaluationView struct {
	SchemaVersion    string                          `json:"schema_version"`
	Issues           []wizardGapsIssueView           `json:"issues"`
	Questions        []wizardGapsQuestionView        `json:"questions"`
	DefaultProposals []wizardGapsDefaultProposalView `json:"default_proposals"`
	PackRefs         []string                        `json:"pack_refs"`
}

type wizardGapsIssueView struct {
	Ref       gaps.IssueRef       `json:"ref"`
	Kind      gaps.IssueKind      `json:"kind"`
	RuleRef   gaps.RuleRef        `json:"rule_ref"`
	Dimension gaps.DimensionRef   `json:"dimension"`
	Field     gaps.SlotKey        `json:"field"`
	DetailKey gaps.MessageKey     `json:"detail_key"`
	DependsOn []gaps.DimensionRef `json:"depends_on"`
}

type wizardGapsQuestionView struct {
	Ref          gaps.QuestionRef       `json:"ref"`
	Dimension    gaps.DimensionRef      `json:"dimension"`
	PackRef      string                 `json:"pack_ref"`
	Slot         gaps.SlotKey           `json:"slot"`
	DecisionKind gaps.DecisionKind      `json:"decision_kind"`
	DerivedFrom  []gaps.IssueRef        `json:"derived_from"`
	DependsOn    []gaps.QuestionRef     `json:"depends_on"`
	PromptKey    gaps.MessageKey        `json:"prompt_key"`
	WhyKey       gaps.MessageKey        `json:"why_key"`
	HelpKey      gaps.MessageKey        `json:"help_key"`
	ExampleKey   gaps.MessageKey        `json:"example_key"`
	Options      []wizardGapsOptionView `json:"options"`
}

type wizardGapsOptionView struct {
	Ref          gaps.OptionRef  `json:"ref"`
	Kind         gaps.OptionKind `json:"kind"`
	LabelKey     gaps.MessageKey `json:"label_key"`
	HelpKey      gaps.MessageKey `json:"help_key"`
	ExampleKey   gaps.MessageKey `json:"example_key"`
	RationaleKey gaps.MessageKey `json:"rationale_key"`
	Recommended  bool            `json:"recommended"`
}

type wizardGapsDefaultProposalView struct {
	Dimension         gaps.DimensionRef `json:"dimension"`
	DecisionKind      gaps.DecisionKind `json:"decision_kind"`
	Option            gaps.OptionRef    `json:"option"`
	LabelKey          gaps.MessageKey   `json:"label_key"`
	HelpKey           gaps.MessageKey   `json:"help_key"`
	ExampleKey        gaps.MessageKey   `json:"example_key"`
	RationaleKey      gaps.MessageKey   `json:"rationale_key"`
	ImplicitlyApplied bool              `json:"implicitly_applied"`
}

func handleApplyWizardGaps(
	ctx context.Context,
	api applicationAPI,
	bound handlerContext,
	payload json.RawMessage,
) (json.RawMessage, error) {
	var input applyWizardGapsInput
	if err := decodePayload(payload, &input); err != nil {
		return nil, err
	}
	packRefs, err := wizardCatalogPackRefs(input.PackRefs)
	if err != nil {
		return nil, commandError{code: CodeInvalidRequest, cause: err}
	}
	result, err := api.ApplyWizardGaps(
		ctx,
		bound.access,
		application.ApplyWizardGapsRequest{
			RequestRef:       bound.requestRef,
			ActorRef:         bound.principal.ActorRef,
			ProjectRef:       bound.projectRef,
			StateRef:         intake.Ref(input.IntakeRef),
			ExpectedRevision: input.ExpectedRevision,
			Origin:           input.Origin,
			Facts: gaps.Facts{
				Surface:                input.Facts.Surface,
				SharingIntent:          input.Facts.SharingIntent,
				CorporateIdentity:      input.Facts.CorporateIdentity,
				TargetUsers:            input.Facts.TargetUsers,
				IntegrationAuth:        input.Facts.IntegrationAuth,
				IntegrationCriticality: input.Facts.IntegrationCriticality,
			},
			PackRefs:          packRefs,
			EvaluatorIdentity: gaps.EvaluatorV1Identity(),
		},
	)
	if err != nil {
		return marshalApplication(wizardGapsResultView{}, normalizeWizardGapsError(err))
	}
	projected, err := projectWizardGapsResult(result)
	return marshalApplication(
		projected,
		normalizeWizardGapsError(err),
	)
}

func wizardCatalogPackRefs(values []string) ([]catalog.PackRef, error) {
	result := make([]catalog.PackRef, 0, len(values))
	for _, value := range values {
		ref, err := catalog.NewPackRef(value)
		if err != nil {
			return nil, err
		}
		result = append(result, ref)
	}
	return result, nil
}

func projectWizardGapsResult(
	source application.ApplyWizardGapsResult,
) (wizardGapsResultView, error) {
	if err := application.ValidateApplyWizardGapsResult(source); err != nil {
		return wizardGapsResultView{}, err
	}
	return wizardGapsResultView{
		Intake:                projectIntakeMutation(source.Record),
		Evaluation:            projectWizardGapsEvaluation(source.Evaluation),
		EvaluationReplayExact: source.EvaluationReplayExact,
		RequestRefReserved:    source.RequestRefReserved,
		RequestOutcome: wizardGapsRequestOutcomeView{
			Kind:       source.RequestOutcome.Kind,
			ReceiptRef: source.RequestOutcome.ReceiptRef,
		},
		InputDurability: wizardGapsDurabilityView{
			Selections: source.InputDurability.Selections,
			Facts:      source.InputDurability.Facts,
			PackRefs:   source.InputDurability.PackRefs,
		},
		EvaluatorIdentity: source.EvaluatorIdentity,
	}, nil
}

func projectWizardGapsEvaluation(source gaps.Result) wizardGapsEvaluationView {
	issues := make([]wizardGapsIssueView, 0, len(source.Issues()))
	for _, issue := range source.Issues() {
		issues = append(issues, wizardGapsIssueView{
			Ref: issue.Ref(), Kind: issue.Kind(), RuleRef: issue.RuleRef(),
			Dimension: issue.Dimension(), Field: issue.Field(),
			DetailKey: issue.DetailKey(), DependsOn: nonNil(issue.DependsOn()),
		})
	}
	questions := make([]wizardGapsQuestionView, 0, len(source.Questions()))
	for _, question := range source.Questions() {
		options := make([]wizardGapsOptionView, 0, len(question.Options()))
		for _, option := range question.Options() {
			options = append(options, wizardGapsOptionView{
				Ref: option.Ref(), Kind: option.Kind(), LabelKey: option.LabelKey(),
				HelpKey: option.HelpKey(), ExampleKey: option.ExampleKey(),
				RationaleKey: option.RationaleKey(), Recommended: option.Recommended(),
			})
		}
		questions = append(questions, wizardGapsQuestionView{
			Ref: question.Ref(), Dimension: question.Dimension(),
			PackRef: question.PackRef().String(), Slot: question.Slot(),
			DecisionKind: question.DecisionKind(),
			DerivedFrom:  nonNil(question.DerivedFrom()),
			DependsOn:    nonNil(question.DependsOn()),
			PromptKey:    question.PromptKey(), WhyKey: question.WhyKey(),
			HelpKey: question.HelpKey(), ExampleKey: question.ExampleKey(),
			Options: nonNil(options),
		})
	}
	defaults := make(
		[]wizardGapsDefaultProposalView,
		0,
		len(source.DefaultProposals()),
	)
	for _, proposal := range source.DefaultProposals() {
		defaults = append(defaults, wizardGapsDefaultProposalView{
			Dimension: proposal.Dimension(), DecisionKind: proposal.DecisionKind(),
			Option: proposal.Option(), LabelKey: proposal.LabelKey(),
			HelpKey: proposal.HelpKey(), ExampleKey: proposal.ExampleKey(),
			RationaleKey:      proposal.RationaleKey(),
			ImplicitlyApplied: proposal.ImplicitlyApplied(),
		})
	}
	packRefs := make([]string, 0, len(source.PackRefs()))
	for _, ref := range source.PackRefs() {
		packRefs = append(packRefs, ref.String())
	}
	return wizardGapsEvaluationView{
		SchemaVersion:    source.SchemaVersion(),
		Issues:           nonNil(issues),
		Questions:        nonNil(questions),
		DefaultProposals: nonNil(defaults),
		PackRefs:         nonNil(packRefs),
	}
}

func normalizeWizardGapsError(err error) error {
	if err == nil {
		return nil
	}
	if gaps.ErrorCodeOf(err) != "" || catalog.ErrorCodeOf(err) != "" {
		return commandError{code: CodeInvalidRequest, cause: err}
	}
	return normalizeIntakeError(err)
}
