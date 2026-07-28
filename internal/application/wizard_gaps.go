package application

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"

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

type WizardGapsRequestOutcomeKind string

const (
	WizardGapsRequestOutcomeIntakeMutation WizardGapsRequestOutcomeKind = "intake_mutation"
	WizardGapsRequestOutcomeNoOp           WizardGapsRequestOutcomeKind = "wizard_gaps_noop"
)

// WizardGapsRequestOutcome names the immutable receipt that owns the public
// request. The Intake receipt still identifies the returned historical state
// snapshot; a no-op outcome has its own receipt because it did not mutate it.
type WizardGapsRequestOutcome struct {
	Kind       WizardGapsRequestOutcomeKind
	ReceiptRef string
}

type ApplyWizardGapsResult struct {
	Record     IntakeRecord
	Evaluation gaps.Result
	// EvaluationReplayExact is false while Facts and PackRefs remain
	// request-scoped. The Intake receipt replays Record exactly; callers must
	// not present the re-evaluated advisory Result as Intake-receipt evidence.
	EvaluationReplayExact bool
	Changed               bool
	// RequestRefReserved is true when either an Intake mutation receipt or an
	// immutable no-op outcome owns the request identity.
	RequestRefReserved bool
	RequestOutcome     WizardGapsRequestOutcome
	InputDurability    WizardGapsInputDurability
	EvaluatorIdentity  intake.DerivationIdentity
}

// ValidateApplyWizardGapsResult keeps public adapters from projecting a
// request as durable without exposing the exact receipt that owns it.
func ValidateApplyWizardGapsResult(result ApplyWizardGapsResult) error {
	if !result.RequestRefReserved || result.EvaluationReplayExact ||
		result.RequestOutcome.ReceiptRef == "" {
		return errors.New("application.wizard_gaps_request_outcome_invalid")
	}
	switch result.RequestOutcome.Kind {
	case WizardGapsRequestOutcomeIntakeMutation:
		if result.RequestOutcome.ReceiptRef != result.Record.Receipt.Ref ||
			!validWizardGapsIntakeReceiptRef(result.RequestOutcome.ReceiptRef) {
			return errors.New("application.wizard_gaps_request_outcome_invalid")
		}
	case WizardGapsRequestOutcomeNoOp:
		if !validWizardGapsNoOpOutcomeRef(result.RequestOutcome.ReceiptRef) ||
			result.RequestOutcome.ReceiptRef == result.Record.Receipt.Ref {
			return errors.New("application.wizard_gaps_request_outcome_invalid")
		}
	default:
		return errors.New("application.wizard_gaps_request_outcome_invalid")
	}
	return nil
}

func validWizardGapsIntakeReceiptRef(value string) bool {
	const prefix = "intake-receipt:"
	return strings.HasPrefix(value, prefix) &&
		validWizardGapsCanonicalHash(strings.TrimPrefix(value, prefix))
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
	outcomes   WizardGapsOutcomeStore
	evaluators wizardGapsEvaluatorResolver
}

func NewWizardGapsService(
	intakes *IntakeService,
	outcomes WizardGapsOutcomeStore,
) (*WizardGapsService, error) {
	registry := gaps.BuiltInEvaluatorRegistry()
	if registry.Empty() {
		return nil, errors.New("application.wizard_gaps_evaluators_required")
	}
	return newWizardGapsServiceWithEvaluatorResolver(
		intakes,
		outcomes,
		builtInWizardGapsEvaluatorResolver{registry: registry},
	)
}

func newWizardGapsServiceWithEvaluatorResolver(
	intakes *IntakeService,
	outcomes WizardGapsOutcomeStore,
	evaluators wizardGapsEvaluatorResolver,
) (*WizardGapsService, error) {
	if intakes == nil {
		return nil, errors.New("application.intake_service_required")
	}
	if evaluators == nil {
		return nil, errors.New("application.wizard_gaps_evaluators_required")
	}
	if outcomes == nil {
		return nil, errors.New("application.wizard_gaps_outcome_store_required")
	}
	return &WizardGapsService{
		intakes: intakes, outcomes: outcomes, evaluators: evaluators,
	}, nil
}

func (service *WizardGapsService) ApplyWizardGaps(
	ctx context.Context,
	request ApplyWizardGapsRequest,
) (ApplyWizardGapsResult, error) {
	if service == nil || service.intakes == nil || service.outcomes == nil ||
		service.evaluators == nil {
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
	replayRequest, err := wizardGapsNoOpReplayRequest(
		request, evaluator.Identity(),
	)
	if err != nil {
		return ApplyWizardGapsResult{}, err
	}
	if replayed, found, replayErr := service.outcomes.ReplayWizardGapsNoOp(
		ctx, replayRequest,
	); replayErr != nil {
		return ApplyWizardGapsResult{}, replayErr
	} else if found {
		return service.replayWizardGapsNoOp(
			replayRequest, request, evaluator, replayed,
		)
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
	evaluated, err := evaluateWizardGapsDetailed(
		base,
		request.Facts,
		request.PackRefs,
		evaluator,
	)
	if err != nil {
		return ApplyWizardGapsResult{}, err
	}
	change, err := wizardGapsChangeWithReconciliation(
		base,
		request.Origin,
		evaluator.Identity(),
		evaluated.result,
		evaluated.reconcilable,
	)
	if err != nil {
		return ApplyWizardGapsResult{}, err
	}
	result := ApplyWizardGapsResult{
		Record: current, Evaluation: evaluated.result,
		InputDurability:   wizardGapsDurability(),
		EvaluatorIdentity: evaluator.Identity(),
	}
	if len(change.Issues) == 0 && len(change.Questions) == 0 &&
		len(change.QuestionRevisions) == 0 &&
		len(change.QuestionRetirements) == 0 {
		if current.State.Revision() != request.ExpectedRevision {
			return ApplyWizardGapsResult{}, &intake.DomainError{
				Code:  intake.ErrorRevisionConflict,
				Field: "request.expected_revision",
			}
		}
		outcome, err := buildWizardGapsNoOpOutcome(
			replayRequest, current, evaluated.result,
		)
		if err != nil {
			return ApplyWizardGapsResult{}, err
		}
		reserved, _, err := service.outcomes.ReserveWizardGapsNoOp(
			ctx,
			WizardGapsNoOpReservation{
				Outcome:              outcome,
				AuthorizationReceipt: request.AuthorizationReceipt,
			},
		)
		if err != nil {
			if !IsStateError(err, StateConflict) {
				return ApplyWizardGapsResult{}, err
			}
			reservationErr := err
			var found bool
			reserved, found, err = service.outcomes.ReplayWizardGapsNoOp(
				ctx, replayRequest,
			)
			if err != nil {
				return ApplyWizardGapsResult{}, err
			}
			if !found {
				return ApplyWizardGapsResult{}, reservationErr
			}
		}
		if err := validateWizardGapsNoOpOutcome(
			replayRequest, reserved, evaluated.result,
		); err != nil {
			return ApplyWizardGapsResult{}, err
		}
		result.Record = reserved.Record
		result.RequestRefReserved = true
		result.RequestOutcome = WizardGapsRequestOutcome{
			Kind:       WizardGapsRequestOutcomeNoOp,
			ReceiptRef: reserved.Ref,
		}
		if err := ValidateApplyWizardGapsResult(result); err != nil {
			return ApplyWizardGapsResult{}, err
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
	result.RequestOutcome = WizardGapsRequestOutcome{
		Kind:       WizardGapsRequestOutcomeIntakeMutation,
		ReceiptRef: applied.Record.Receipt.Ref,
	}
	if err := ValidateApplyWizardGapsResult(result); err != nil {
		return ApplyWizardGapsResult{}, err
	}
	return result, nil
}

func (service *WizardGapsService) replayWizardGapsNoOp(
	replayRequest WizardGapsNoOpReplayRequest,
	request ApplyWizardGapsRequest,
	evaluator wizardGapsEvaluator,
	outcome WizardGapsNoOpOutcome,
) (ApplyWizardGapsResult, error) {
	evaluated, err := evaluateWizardGapsDetailed(
		outcome.Record.State,
		request.Facts,
		request.PackRefs,
		evaluator,
	)
	if err != nil {
		return ApplyWizardGapsResult{}, err
	}
	if err := validateWizardGapsNoOpOutcome(
		replayRequest, outcome, evaluated.result,
	); err != nil {
		return ApplyWizardGapsResult{}, err
	}
	change, err := wizardGapsChangeWithReconciliation(
		outcome.Record.State,
		request.Origin,
		evaluator.Identity(),
		evaluated.result,
		evaluated.reconcilable,
	)
	if err != nil {
		return ApplyWizardGapsResult{}, err
	}
	if len(change.Issues) != 0 || len(change.Questions) != 0 ||
		len(change.QuestionRevisions) != 0 ||
		len(change.QuestionRetirements) != 0 {
		return ApplyWizardGapsResult{}, &StateError{Code: StateConflict}
	}
	result := ApplyWizardGapsResult{
		Record: outcome.Record, Evaluation: evaluated.result,
		RequestRefReserved: true, InputDurability: wizardGapsDurability(),
		EvaluatorIdentity: evaluator.Identity(),
		RequestOutcome: WizardGapsRequestOutcome{
			Kind:       WizardGapsRequestOutcomeNoOp,
			ReceiptRef: outcome.Ref,
		},
	}
	if err := ValidateApplyWizardGapsResult(result); err != nil {
		return ApplyWizardGapsResult{}, err
	}
	return result, nil
}

type detailedWizardGapsEvaluation struct {
	result       gaps.Result
	reconcilable map[intake.QuestionRef]struct{}
}

func evaluateWizardGaps(
	state intake.State,
	facts gaps.Facts,
	packRefs []catalog.PackRef,
	evaluator wizardGapsEvaluator,
) (gaps.Result, error) {
	evaluated, err := evaluateWizardGapsDetailed(
		state,
		facts,
		packRefs,
		evaluator,
	)
	return evaluated.result, err
}

func evaluateWizardGapsDetailed(
	state intake.State,
	facts gaps.Facts,
	packRefs []catalog.PackRef,
	evaluator wizardGapsEvaluator,
) (detailedWizardGapsEvaluation, error) {
	preflight, err := preflightWizardGapsQuestions(state, evaluator)
	if err != nil {
		return detailedWizardGapsEvaluation{}, err
	}
	selections, remaining := wizardGapsDimensionSelections(state, preflight.dimensions)
	preliminary, err := evaluator.Evaluate(gaps.Input{
		Facts: facts, Selections: selections, PackRefs: packRefs,
	})
	if err != nil {
		return detailedWizardGapsEvaluation{}, err
	}
	reconcilable, err := wizardGapsCounterfactualReconciliation(
		state,
		facts,
		selections,
		packRefs,
		evaluator,
		preliminary,
	)
	if err != nil {
		return detailedWizardGapsEvaluation{}, err
	}
	preflight.reopened = reconcilable
	if err = preflightWizardGapsDimensionPayloads(
		preflight,
		evaluator,
		facts,
		selections,
		packRefs,
	); err != nil {
		return detailedWizardGapsEvaluation{}, err
	}
	if err = preflightWizardGapsSupplementalQuestions(
		preflight,
		evaluator,
		preliminary,
	); err != nil {
		return detailedWizardGapsEvaluation{}, err
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
	result, err := evaluator.Evaluate(gaps.Input{
		Facts: facts, Selections: selections,
		QuestionSelections: questionSelections,
		PackRefs:           packRefs,
	})
	if err != nil {
		return detailedWizardGapsEvaluation{}, err
	}
	return detailedWizardGapsEvaluation{
		result: result, reconcilable: reconcilable,
	}, nil
}

func wizardGapsChange(
	state intake.State,
	origin intake.Origin,
	derivation intake.DerivationIdentity,
	evaluation gaps.Result,
) (intake.Change, error) {
	return wizardGapsChangeWithReconciliation(
		state,
		origin,
		derivation,
		evaluation,
		wizardGapsReconcilableQuestions(state),
	)
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

func wizardGapsChangeWithReconciliation(
	state intake.State,
	origin intake.Origin,
	derivation intake.DerivationIdentity,
	evaluation gaps.Result,
	reconcilableQuestions map[intake.QuestionRef]struct{},
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
	_, questionDerivations, err := wizardGapsArtifactDerivations(state)
	if err != nil {
		return intake.Change{}, err
	}
	latestVersions := make(map[intake.QuestionRef]intake.QuestionVersion)
	for _, version := range state.QuestionVersions() {
		latestVersions[version.Question.Ref] = version
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
	revisions := make([]intake.Question, 0)
	expectedQuestions := make(map[intake.QuestionRef]struct{})
	for _, question := range evaluation.Questions() {
		projected := question.IntakeQuestion()
		expectedQuestions[projected.Ref] = struct{}{}
		if existing, found := existingQuestions[projected.Ref]; found {
			if !wizardGapsQuestionPayloadEqual(existing, projected) {
				if _, causallyAffected := reconcilableQuestions[projected.Ref]; !causallyAffected {
					return intake.Change{}, wizardGapsProjectionConflict(
						"question",
						string(projected.Ref),
					)
				}
				revisions = append(revisions, projected)
			}
			continue
		}
		if latest, found := latestVersions[projected.Ref]; found {
			if !latest.Retired ||
				questionDerivations[projected.Ref] != derivation {
				return intake.Change{}, wizardGapsProjectionConflict(
					"question",
					string(projected.Ref),
				)
			}
			if _, causallyAffected := reconcilableQuestions[projected.Ref]; !causallyAffected {
				return intake.Change{}, wizardGapsProjectionConflict(
					"question",
					string(projected.Ref),
				)
			}
			revisions = append(revisions, projected)
			continue
		}
		questions = append(questions, projected)
	}
	retirements := make([]intake.QuestionRef, 0)
	for _, question := range state.Questions() {
		if _, stillEmitted := expectedQuestions[question.Ref]; stillEmitted {
			continue
		}
		if questionDerivations[question.Ref] != derivation {
			continue
		}
		if _, currentlyAnswered := state.CurrentDecision(question.Ref); currentlyAnswered {
			continue
		}
		if _, causallyAffected := reconcilableQuestions[question.Ref]; !causallyAffected {
			continue
		}
		retirements = append(retirements, question.Ref)
	}
	return intake.Change{
		StateRef: state.Ref(), ExpectedRevision: state.Revision(),
		Origin: origin, Issues: issues, Questions: questions,
		QuestionRevisions:   revisions,
		QuestionRetirements: retirements,
		Derivation:          derivation,
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
