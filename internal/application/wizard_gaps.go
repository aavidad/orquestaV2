package application

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/intake"
	"orquesta/internal/wizard/catalog"
	"orquesta/internal/wizard/gaps"
)

// WizardGapsInputDurability makes the current persistence frontier explicit.
// Durable selections are reconstructed from Intake decisions. Facts and pack
// refs remain typed request context until Intake gains corresponding typed
// decisions; this service never creates a parallel store for them.
type WizardGapsInputDurability struct {
	Selections string
	Facts      string
	PackRefs   string
}

func wizardGapsDurability() WizardGapsInputDurability {
	return WizardGapsInputDurability{
		Selections: "intake_decisions",
		Facts:      "request_scoped",
		PackRefs:   "request_scoped",
	}
}

type ApplyWizardGapsRequest struct {
	RequestRef           string
	ActorRef             goal.ActorRef
	ProjectRef           goal.ProjectRef
	StateRef             intake.Ref
	ExpectedRevision     intake.Revision
	Origin               intake.Origin
	Facts                gaps.Facts
	PackRefs             []catalog.PackRef
	EvaluatorIdentity    intake.DerivationIdentity
	AuthorizationReceipt identity.AuthorizationReceipt
}

type ApplyWizardGapsResult struct {
	Record     IntakeRecord
	Evaluation gaps.Result
	// EvaluationReplayExact is false while Facts and PackRefs remain
	// request-scoped. The Intake receipt replays Record exactly; callers must
	// not present the re-evaluated advisory Result as receipt-bound evidence.
	EvaluationReplayExact bool
	Changed               bool
	// RequestRefReserved is true only when IntakeService found or wrote the
	// regular mutation receipt. A no-op writes nothing, does not reserve its
	// request_ref and is deliberately not advertised as an exact replay.
	RequestRefReserved bool
	InputDurability    WizardGapsInputDurability
	EvaluatorIdentity  intake.DerivationIdentity
}

type wizardGapsEvaluator interface {
	Identity() intake.DerivationIdentity
	Dimensions() []gaps.DimensionDescriptor
	Rules() []gaps.RuleDescriptor
	Evaluate(gaps.Input) (gaps.Result, error)
}

type wizardGapsEvaluatorResolver interface {
	Resolve(intake.DerivationIdentity) (wizardGapsEvaluator, error)
}

type builtInWizardGapsEvaluatorResolver struct {
	registry gaps.EvaluatorRegistry
}

func (resolver builtInWizardGapsEvaluatorResolver) Resolve(
	identity intake.DerivationIdentity,
) (wizardGapsEvaluator, error) {
	return resolver.registry.Resolve(identity)
}

// WizardGapsService evaluates the pure gaps engine and compiles only missing
// findings into the existing Intake CAS mutation. IntakeService remains the
// sole writer and its receipt remains the sole mutation receipt.
type WizardGapsService struct {
	intakes    *IntakeService
	evaluators wizardGapsEvaluatorResolver
}

func NewWizardGapsService(intakes *IntakeService) (*WizardGapsService, error) {
	registry := gaps.BuiltInEvaluatorRegistry()
	if registry.Empty() {
		return nil, errors.New("application.wizard_gaps_evaluators_required")
	}
	return newWizardGapsServiceWithEvaluatorResolver(
		intakes,
		builtInWizardGapsEvaluatorResolver{registry: registry},
	)
}

func newWizardGapsServiceWithEvaluatorResolver(
	intakes *IntakeService,
	evaluators wizardGapsEvaluatorResolver,
) (*WizardGapsService, error) {
	if intakes == nil {
		return nil, errors.New("application.intake_service_required")
	}
	if evaluators == nil {
		return nil, errors.New("application.wizard_gaps_evaluators_required")
	}
	return &WizardGapsService{intakes: intakes, evaluators: evaluators}, nil
}

func (service *WizardGapsService) ApplyWizardGaps(
	ctx context.Context,
	request ApplyWizardGapsRequest,
) (ApplyWizardGapsResult, error) {
	if service == nil || service.intakes == nil || service.evaluators == nil {
		return ApplyWizardGapsResult{}, errors.New("application.unavailable")
	}
	if err := validateIntakeRequestScope(
		request.RequestRef,
		request.ActorRef,
		request.ProjectRef,
	); err != nil {
		return ApplyWizardGapsResult{}, err
	}
	authorizationRequestRef, err := IntakeAuthorizationRequestRef(
		IntakeOperationApply,
		request.RequestRef,
	)
	if err != nil {
		return ApplyWizardGapsResult{}, err
	}
	if err = validateIntakeAuthorization(
		request.AuthorizationReceipt,
		request.ActorRef,
		request.ProjectRef,
		authorizationRequestRef,
	); err != nil {
		return ApplyWizardGapsResult{}, err
	}
	if request.Origin != intake.OriginChat && request.Origin != intake.OriginForm {
		return ApplyWizardGapsResult{}, &intake.DomainError{
			Code:  intake.ErrorInvalidOrigin,
			Field: "request.origin",
		}
	}
	evaluator, err := service.evaluators.Resolve(request.EvaluatorIdentity)
	if err != nil {
		return ApplyWizardGapsResult{}, err
	}

	current, err := service.intakes.GetIntake(ctx, GetIntakeRequest{
		ActorRef: request.ActorRef, ProjectRef: request.ProjectRef,
		StateRef: request.StateRef,
	})
	if err != nil {
		return ApplyWizardGapsResult{}, err
	}
	base, err := intakeStateAtRevision(current.State, request.ExpectedRevision)
	if err != nil {
		return ApplyWizardGapsResult{}, err
	}
	evaluation, err := evaluateWizardGaps(
		base,
		request.Facts,
		request.PackRefs,
		evaluator,
	)
	if err != nil {
		return ApplyWizardGapsResult{}, err
	}
	change, err := wizardGapsChange(
		base,
		request.Origin,
		evaluator.Identity(),
		evaluation,
	)
	if err != nil {
		return ApplyWizardGapsResult{}, err
	}
	result := ApplyWizardGapsResult{
		Record: current, Evaluation: evaluation,
		InputDurability:   wizardGapsDurability(),
		EvaluatorIdentity: evaluator.Identity(),
	}
	if len(change.Issues) == 0 && len(change.Questions) == 0 {
		if current.State.Revision() != request.ExpectedRevision {
			return ApplyWizardGapsResult{}, &intake.DomainError{
				Code:  intake.ErrorRevisionConflict,
				Field: "request.expected_revision",
			}
		}
		return result, nil
	}

	applied, err := service.intakes.ApplyIntake(ctx, ApplyIntakeRequest{
		RequestRef: request.RequestRef, ActorRef: request.ActorRef,
		ProjectRef: request.ProjectRef, Change: change,
		AuthorizationReceipt: request.AuthorizationReceipt,
	})
	if err != nil {
		return ApplyWizardGapsResult{}, err
	}
	result.Record = applied.Record
	result.Changed = applied.Changed
	result.RequestRefReserved = true
	return result, nil
}

func evaluateWizardGaps(
	state intake.State,
	facts gaps.Facts,
	packRefs []catalog.PackRef,
	evaluator wizardGapsEvaluator,
) (gaps.Result, error) {
	preflight, err := preflightWizardGapsQuestions(state, evaluator)
	if err != nil {
		return gaps.Result{}, err
	}
	selections, remaining := wizardGapsDimensionSelections(state, preflight.dimensions)
	if err = preflightWizardGapsDimensionPayloads(
		preflight,
		evaluator,
		facts,
		selections,
		packRefs,
	); err != nil {
		return gaps.Result{}, err
	}
	preliminary, err := evaluator.Evaluate(gaps.Input{
		Facts: facts, Selections: selections, PackRefs: packRefs,
	})
	if err != nil {
		return gaps.Result{}, err
	}
	if err = preflightWizardGapsSupplementalQuestions(
		preflight,
		evaluator,
		preliminary,
	); err != nil {
		return gaps.Result{}, err
	}
	knownQuestions := make(map[gaps.QuestionRef]struct{})
	for _, question := range preliminary.Questions() {
		knownQuestions[question.Ref()] = struct{}{}
	}
	questionSelections := make([]gaps.QuestionSelection, 0, len(remaining))
	for _, decision := range remaining {
		ref := gaps.QuestionRef(decision.QuestionRef)
		if _, found := knownQuestions[ref]; !found {
			continue
		}
		questionSelections = append(questionSelections, gaps.QuestionSelection{
			Question: ref,
			Option:   gaps.OptionRef(decision.Choice),
			FreeText: decision.AnswerText,
		})
	}
	return evaluator.Evaluate(gaps.Input{
		Facts: facts, Selections: selections,
		QuestionSelections: questionSelections,
		PackRefs:           packRefs,
	})
}

func wizardGapsDimensionSelections(
	state intake.State,
	dimensions map[intake.QuestionRef]gaps.DimensionDescriptor,
) ([]gaps.Selection, []intake.Decision) {
	current := make(map[intake.QuestionRef]intake.Decision)
	for _, decision := range state.Decisions() {
		current[decision.QuestionRef] = decision
	}
	for _, reopened := range state.ReopenedDecisions() {
		delete(current, reopened.QuestionRef)
	}
	selections := make([]gaps.Selection, 0)
	remaining := make([]intake.Decision, 0)
	for _, question := range state.Questions() {
		decision, found := current[question.Ref]
		if !found {
			continue
		}
		dimension, isDimension := dimensions[question.Ref]
		if !isDimension {
			remaining = append(remaining, decision)
			continue
		}
		selections = append(selections, gaps.Selection{
			Dimension: dimension.Ref(),
			Option:    gaps.OptionRef(decision.Choice),
			FreeText:  decision.AnswerText,
		})
	}
	return selections, remaining
}

func wizardGapsChange(
	state intake.State,
	origin intake.Origin,
	derivation intake.DerivationIdentity,
	evaluation gaps.Result,
) (intake.Change, error) {
	existingIssues := make(map[intake.IssueRef]intake.Issue, len(state.Issues()))
	for _, issue := range state.Issues() {
		existingIssues[issue.Ref] = issue
	}
	existingQuestions := make(
		map[intake.QuestionRef]intake.Question,
		len(state.Questions()),
	)
	for _, question := range state.Questions() {
		existingQuestions[question.Ref] = question
	}

	issues := make([]intake.Issue, 0)
	for _, issue := range evaluation.Issues() {
		projected := issue.IntakeIssue()
		if existing, found := existingIssues[projected.Ref]; found {
			if existing != projected {
				return intake.Change{}, wizardGapsProjectionConflict(
					"issue",
					string(projected.Ref),
				)
			}
			continue
		}
		issues = append(issues, projected)
	}
	questions := make([]intake.Question, 0)
	for _, question := range evaluation.Questions() {
		projected := question.IntakeQuestion()
		if existing, found := existingQuestions[projected.Ref]; found {
			if !wizardGapsQuestionPayloadEqual(existing, projected) {
				return intake.Change{}, wizardGapsProjectionConflict(
					"question",
					string(projected.Ref),
				)
			}
			continue
		}
		questions = append(questions, projected)
	}
	return intake.Change{
		StateRef: state.Ref(), ExpectedRevision: state.Revision(),
		Origin: origin, Issues: issues, Questions: questions,
		Derivation: derivation,
	}, nil
}

func wizardGapsQuestionPayloadEqual(
	left intake.Question,
	right intake.Question,
) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftJSON, rightJSON)
}

var ErrWizardGapsProjectionConflict = errors.New(
	"application.wizard_gaps_projection_conflict",
)

type WizardGapsProjectionConflictError struct {
	Kind string
	Ref  string
}

func (err *WizardGapsProjectionConflictError) Error() string {
	if err == nil {
		return ""
	}
	return ErrWizardGapsProjectionConflict.Error() + ":" + err.Kind + ":" + err.Ref
}

func (err *WizardGapsProjectionConflictError) Unwrap() error {
	return ErrWizardGapsProjectionConflict
}

func wizardGapsProjectionConflict(kind, ref string) error {
	return &StateError{
		Code:  StateConflict,
		Cause: &WizardGapsProjectionConflictError{Kind: kind, Ref: ref},
	}
}
