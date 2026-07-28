package application

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/intake"
	"orquesta/internal/wizard/gaps"
)

// WizardGapsNoOpOutcome is an immutable request reservation for an evaluation
// that did not need an Intake mutation. It binds the exact request and
// evaluator to the historical Intake receipt used by that evaluation without
// pretending that request-scoped facts or pack refs became Intake state.
type WizardGapsNoOpOutcome struct {
	Ref                     string
	RequestRef              string
	RequestFingerprint      string
	ActorRef                goal.ActorRef
	ProjectRef              goal.ProjectRef
	StateRef                intake.Ref
	ExpectedRevision        intake.Revision
	SourceIntakeReceiptRef  string
	EvaluatorIdentity       intake.DerivationIdentity
	EvaluationDigest        string
	AuthorizationReceiptRef string
	Record                  IntakeRecord
}

type WizardGapsNoOpReplayRequest struct {
	RequestRef              string
	RequestFingerprint      string
	ActorRef                goal.ActorRef
	ProjectRef              goal.ProjectRef
	StateRef                intake.Ref
	ExpectedRevision        intake.Revision
	EvaluatorIdentity       intake.DerivationIdentity
	AuthorizationReceiptRef string
}

type WizardGapsNoOpReservation struct {
	Outcome              WizardGapsNoOpOutcome
	AuthorizationReceipt identity.AuthorizationReceipt
}

// WizardGapsOutcomeStore owns only immutable no-op request reservations.
// IntakeStore remains the sole authority for Intake mutations and revisions.
type WizardGapsOutcomeStore interface {
	ReplayWizardGapsNoOp(
		context.Context,
		WizardGapsNoOpReplayRequest,
	) (WizardGapsNoOpOutcome, bool, error)
	ReserveWizardGapsNoOp(
		context.Context,
		WizardGapsNoOpReservation,
	) (WizardGapsNoOpOutcome, bool, error)
}

func wizardGapsNoOpReplayRequest(
	request ApplyWizardGapsRequest,
	evaluatorIdentity intake.DerivationIdentity,
) (WizardGapsNoOpReplayRequest, error) {
	factsJSON, err := json.Marshal(request.Facts)
	if err != nil {
		return WizardGapsNoOpReplayRequest{}, errors.New(
			"application.wizard_gaps_facts_invalid",
		)
	}
	packRefs := make([]string, len(request.PackRefs))
	for index, ref := range request.PackRefs {
		packRefs[index] = ref.String()
	}
	packRefsJSON, err := json.Marshal(packRefs)
	if err != nil {
		return WizardGapsNoOpReplayRequest{}, errors.New(
			"application.wizard_gaps_pack_refs_invalid",
		)
	}
	fingerprint := fingerprintFields(
		"orquesta.wizard.gaps.noop.request.v1",
		request.ActorRef.String(),
		request.ProjectRef.String(),
		string(request.StateRef),
		strconv.FormatUint(uint64(request.ExpectedRevision), 10),
		string(request.Origin),
		string(factsJSON),
		string(packRefsJSON),
		evaluatorIdentity.Schema,
		evaluatorIdentity.Version,
		evaluatorIdentity.SemanticDigest,
		request.AuthorizationReceipt.Ref(),
	)
	return WizardGapsNoOpReplayRequest{
		RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		ActorRef: request.ActorRef, ProjectRef: request.ProjectRef,
		StateRef: request.StateRef, ExpectedRevision: request.ExpectedRevision,
		EvaluatorIdentity:       evaluatorIdentity,
		AuthorizationReceiptRef: request.AuthorizationReceipt.Ref(),
	}, nil
}

func buildWizardGapsNoOpOutcome(
	request WizardGapsNoOpReplayRequest,
	record IntakeRecord,
	evaluation gaps.Result,
) (WizardGapsNoOpOutcome, error) {
	evaluationDigest, err := wizardGapsEvaluationDigest(evaluation)
	if err != nil {
		return WizardGapsNoOpOutcome{}, err
	}
	outcome := WizardGapsNoOpOutcome{
		RequestRef: request.RequestRef, RequestFingerprint: request.RequestFingerprint,
		ActorRef: request.ActorRef, ProjectRef: request.ProjectRef,
		StateRef: request.StateRef, ExpectedRevision: request.ExpectedRevision,
		SourceIntakeReceiptRef:  record.Receipt.Ref,
		EvaluatorIdentity:       request.EvaluatorIdentity,
		EvaluationDigest:        evaluationDigest,
		AuthorizationReceiptRef: request.AuthorizationReceiptRef,
		Record:                  record,
	}
	outcome.Ref = wizardGapsNoOpOutcomeRef(outcome)
	return outcome, nil
}

func validateWizardGapsNoOpOutcome(
	request WizardGapsNoOpReplayRequest,
	outcome WizardGapsNoOpOutcome,
	evaluation gaps.Result,
) error {
	evaluationDigest, err := wizardGapsEvaluationDigest(evaluation)
	if err != nil {
		return err
	}
	expected, err := buildWizardGapsNoOpOutcome(request, outcome.Record, evaluation)
	if err != nil {
		return err
	}
	if outcome.Ref != expected.Ref ||
		outcome.RequestRef != request.RequestRef ||
		outcome.RequestFingerprint != request.RequestFingerprint ||
		outcome.ActorRef != request.ActorRef ||
		outcome.ProjectRef != request.ProjectRef ||
		outcome.StateRef != request.StateRef ||
		outcome.ExpectedRevision != request.ExpectedRevision ||
		outcome.EvaluatorIdentity != request.EvaluatorIdentity ||
		outcome.EvaluationDigest != evaluationDigest ||
		outcome.AuthorizationReceiptRef != request.AuthorizationReceiptRef ||
		outcome.SourceIntakeReceiptRef != outcome.Record.Receipt.Ref ||
		outcome.Record.State.Revision() != request.ExpectedRevision {
		return &StateError{Code: StateConflict}
	}
	return validateStoredIntakeRecord(
		request.ActorRef, request.ProjectRef, request.StateRef, outcome.Record,
	)
}

// ValidateWizardGapsNoOpOutcomeRecord is the adapter boundary validator for an
// immutable no-op outcome and its historical Intake receipt.
func ValidateWizardGapsNoOpOutcomeRecord(outcome WizardGapsNoOpOutcome) error {
	if !validIntakeRequestRef(outcome.RequestRef) ||
		!validWizardGapsCanonicalHash(outcome.RequestFingerprint) ||
		!validWizardGapsCanonicalHash(outcome.EvaluationDigest) ||
		outcome.ExpectedRevision == 0 ||
		outcome.Ref != wizardGapsNoOpOutcomeRef(outcome) ||
		outcome.SourceIntakeReceiptRef != outcome.Record.Receipt.Ref ||
		outcome.Record.State.Revision() != outcome.ExpectedRevision ||
		outcome.Record.Receipt.Revision != outcome.ExpectedRevision ||
		outcome.AuthorizationReceiptRef == "" {
		return &StateError{Code: StateConflict}
	}
	identity, err := intake.NewDerivationIdentity(
		outcome.EvaluatorIdentity.Schema,
		outcome.EvaluatorIdentity.Version,
		outcome.EvaluatorIdentity.SemanticDigest,
	)
	if err != nil || identity != outcome.EvaluatorIdentity {
		return &StateError{Code: StateConflict, Cause: err}
	}
	return validateStoredIntakeRecord(
		outcome.ActorRef, outcome.ProjectRef, outcome.StateRef, outcome.Record,
	)
}

func wizardGapsNoOpOutcomeRef(outcome WizardGapsNoOpOutcome) string {
	return "wizard-gaps-outcome:" + fingerprintFields(
		"orquesta.wizard.gaps.noop.outcome.v1",
		outcome.RequestRef,
		outcome.RequestFingerprint,
		outcome.ActorRef.String(),
		outcome.ProjectRef.String(),
		string(outcome.StateRef),
		strconv.FormatUint(uint64(outcome.ExpectedRevision), 10),
		outcome.SourceIntakeReceiptRef,
		outcome.EvaluatorIdentity.Schema,
		outcome.EvaluatorIdentity.Version,
		outcome.EvaluatorIdentity.SemanticDigest,
		outcome.EvaluationDigest,
		outcome.AuthorizationReceiptRef,
	)
}

func validWizardGapsNoOpOutcomeRef(value string) bool {
	const prefix = "wizard-gaps-outcome:"
	return strings.HasPrefix(value, prefix) &&
		validWizardGapsCanonicalHash(strings.TrimPrefix(value, prefix))
}

func validWizardGapsCanonicalHash(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, char := range value {
		if !strings.ContainsRune("0123456789abcdef", char) {
			return false
		}
	}
	return true
}

type wizardGapsEvaluationDigestView struct {
	SchemaVersion string                         `json:"schema_version"`
	Issues        []wizardGapsIssueDigestView    `json:"issues"`
	Questions     []wizardGapsQuestionDigestView `json:"questions"`
	Defaults      []wizardGapsDefaultDigestView  `json:"default_proposals"`
	PackRefs      []string                       `json:"pack_refs"`
}

type wizardGapsIssueDigestView struct {
	Ref       gaps.IssueRef       `json:"ref"`
	Kind      gaps.IssueKind      `json:"kind"`
	RuleRef   gaps.RuleRef        `json:"rule_ref"`
	Dimension gaps.DimensionRef   `json:"dimension"`
	Field     gaps.SlotKey        `json:"field"`
	DetailKey gaps.MessageKey     `json:"detail_key"`
	DependsOn []gaps.DimensionRef `json:"depends_on"`
}

type wizardGapsQuestionDigestView struct {
	Ref          gaps.QuestionRef             `json:"ref"`
	Dimension    gaps.DimensionRef            `json:"dimension"`
	PackRef      string                       `json:"pack_ref"`
	Slot         gaps.SlotKey                 `json:"slot"`
	DecisionKind gaps.DecisionKind            `json:"decision_kind"`
	DerivedFrom  []gaps.IssueRef              `json:"derived_from"`
	DependsOn    []gaps.QuestionRef           `json:"depends_on"`
	PromptKey    gaps.MessageKey              `json:"prompt_key"`
	WhyKey       gaps.MessageKey              `json:"why_key"`
	HelpKey      gaps.MessageKey              `json:"help_key"`
	ExampleKey   gaps.MessageKey              `json:"example_key"`
	Options      []wizardGapsOptionDigestView `json:"options"`
}

type wizardGapsOptionDigestView struct {
	Ref          gaps.OptionRef  `json:"ref"`
	Kind         gaps.OptionKind `json:"kind"`
	LabelKey     gaps.MessageKey `json:"label_key"`
	HelpKey      gaps.MessageKey `json:"help_key"`
	ExampleKey   gaps.MessageKey `json:"example_key"`
	RationaleKey gaps.MessageKey `json:"rationale_key"`
	Recommended  bool            `json:"recommended"`
}

type wizardGapsDefaultDigestView struct {
	Dimension         gaps.DimensionRef `json:"dimension"`
	DecisionKind      gaps.DecisionKind `json:"decision_kind"`
	Option            gaps.OptionRef    `json:"option"`
	LabelKey          gaps.MessageKey   `json:"label_key"`
	HelpKey           gaps.MessageKey   `json:"help_key"`
	ExampleKey        gaps.MessageKey   `json:"example_key"`
	RationaleKey      gaps.MessageKey   `json:"rationale_key"`
	ImplicitlyApplied bool              `json:"implicitly_applied"`
}

func wizardGapsEvaluationDigest(result gaps.Result) (string, error) {
	view := wizardGapsEvaluationDigestView{
		SchemaVersion: result.SchemaVersion(),
		Issues:        make([]wizardGapsIssueDigestView, 0, len(result.Issues())),
		Questions:     make([]wizardGapsQuestionDigestView, 0, len(result.Questions())),
		Defaults:      make([]wizardGapsDefaultDigestView, 0, len(result.DefaultProposals())),
		PackRefs:      make([]string, 0, len(result.PackRefs())),
	}
	for _, issue := range result.Issues() {
		view.Issues = append(view.Issues, wizardGapsIssueDigestView{
			Ref: issue.Ref(), Kind: issue.Kind(), RuleRef: issue.RuleRef(),
			Dimension: issue.Dimension(), Field: issue.Field(),
			DetailKey: issue.DetailKey(), DependsOn: issue.DependsOn(),
		})
	}
	for _, question := range result.Questions() {
		options := make([]wizardGapsOptionDigestView, 0, len(question.Options()))
		for _, option := range question.Options() {
			options = append(options, wizardGapsOptionDigestView{
				Ref: option.Ref(), Kind: option.Kind(), LabelKey: option.LabelKey(),
				HelpKey: option.HelpKey(), ExampleKey: option.ExampleKey(),
				RationaleKey: option.RationaleKey(), Recommended: option.Recommended(),
			})
		}
		view.Questions = append(view.Questions, wizardGapsQuestionDigestView{
			Ref: question.Ref(), Dimension: question.Dimension(),
			PackRef: question.PackRef().String(), Slot: question.Slot(),
			DecisionKind: question.DecisionKind(),
			DerivedFrom:  question.DerivedFrom(), DependsOn: question.DependsOn(),
			PromptKey: question.PromptKey(), WhyKey: question.WhyKey(),
			HelpKey: question.HelpKey(), ExampleKey: question.ExampleKey(),
			Options: options,
		})
	}
	for _, proposal := range result.DefaultProposals() {
		view.Defaults = append(view.Defaults, wizardGapsDefaultDigestView{
			Dimension: proposal.Dimension(), DecisionKind: proposal.DecisionKind(),
			Option: proposal.Option(), LabelKey: proposal.LabelKey(),
			HelpKey: proposal.HelpKey(), ExampleKey: proposal.ExampleKey(),
			RationaleKey:      proposal.RationaleKey(),
			ImplicitlyApplied: proposal.ImplicitlyApplied(),
		})
	}
	for _, ref := range result.PackRefs() {
		view.PackRefs = append(view.PackRefs, ref.String())
	}
	encoded, err := json.Marshal(view)
	if err != nil {
		return "", errors.New("application.wizard_gaps_evaluation_invalid")
	}
	return fingerprintFields(
		"orquesta.wizard.gaps.evaluation.v1",
		string(encoded),
	), nil
}
