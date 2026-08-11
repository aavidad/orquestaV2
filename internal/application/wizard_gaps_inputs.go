package application

import (
	"encoding/json"
	"errors"
	"sort"
	"strconv"

	"orquesta/internal/goal"
	"orquesta/internal/intake"
	"orquesta/internal/wizard/catalog"
	"orquesta/internal/wizard/gaps"
)

const wizardGapsInputReceiptSchema = "orquesta.wizard.gaps.input-receipt.v1"

// WizardGapsInputReceipt is the immutable, content-addressed account of the
// exact inputs accepted by one public Wizard gaps request. ResultSnapshot is
// an attached causal binding and is not part of Ref; the next SQLite cut must
// persist it atomically before EvaluationReplayExact can become true.
type WizardGapsInputReceipt struct {
	Ref                     string
	RequestRef              string
	RequestFingerprint      string
	ActorRef                goal.ActorRef
	ProjectRef              goal.ProjectRef
	StateRef                intake.Ref
	ExpectedRevision        intake.Revision
	Origin                  intake.Origin
	SourceIntakeReceiptRef  string
	OutcomeKind             WizardGapsRequestOutcomeKind
	OutcomeReceiptRef       string
	Facts                   gaps.Facts
	FactsDigest             string
	PackRefs                []catalog.PackRef
	PackRefsDigest          string
	Selections              []WizardGapsSelectionInput
	SelectionsDigest        string
	EvaluatorIdentity       intake.DerivationIdentity
	AuthorizationReceiptRef string
	ResultSnapshot          WizardGapsResultSnapshot
}

type WizardGapsInputReplayRequest struct {
	RequestRef              string
	RequestFingerprint      string
	ActorRef                goal.ActorRef
	ProjectRef              goal.ProjectRef
	StateRef                intake.Ref
	ExpectedRevision        intake.Revision
	EvaluatorIdentity       intake.DerivationIdentity
	AuthorizationReceiptRef string
}

// WizardGapsInputRecord couples the input receipt to both immutable Intake
// snapshots needed to verify it: the source evaluated and the request outcome.
type WizardGapsInputRecord struct {
	Receipt       WizardGapsInputReceipt
	SourceRecord  IntakeRecord
	OutcomeRecord IntakeRecord
}

type WizardGapsMutationReservation struct {
	Input  WizardGapsInputReceipt
	Intake IntakeApplyState
}

func canonicalWizardGapsRequest(
	request ApplyWizardGapsRequest,
	evaluator wizardGapsEvaluator,
) (WizardGapsInputReplayRequest, gaps.Facts, []catalog.PackRef, string, string, error) {
	facts, packRefs, err := gaps.CanonicalContext(
		request.Facts,
		request.PackRefs,
	)
	if err != nil {
		return WizardGapsInputReplayRequest{}, gaps.Facts{}, nil, "", "", err
	}
	factsDigest, err := wizardGapsFactsDigest(facts)
	if err != nil {
		return WizardGapsInputReplayRequest{}, gaps.Facts{}, nil, "", "", err
	}
	packRefsDigest, err := wizardGapsPackRefsDigest(packRefs)
	if err != nil {
		return WizardGapsInputReplayRequest{}, gaps.Facts{}, nil, "", "", err
	}
	fingerprint := wizardGapsRequestFingerprint(
		request.ActorRef,
		request.ProjectRef,
		request.StateRef,
		request.ExpectedRevision,
		request.Origin,
		factsDigest,
		packRefsDigest,
		evaluator.Identity(),
		request.AuthorizationReceipt.Ref(),
	)
	return WizardGapsInputReplayRequest{
		RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		ActorRef: request.ActorRef, ProjectRef: request.ProjectRef,
		StateRef: request.StateRef, ExpectedRevision: request.ExpectedRevision,
		EvaluatorIdentity:       evaluator.Identity(),
		AuthorizationReceiptRef: request.AuthorizationReceipt.Ref(),
	}, facts, packRefs, factsDigest, packRefsDigest, nil
}

func buildWizardGapsInputReceipt(
	request WizardGapsInputReplayRequest,
	origin intake.Origin,
	source IntakeRecord,
	outcome WizardGapsRequestOutcome,
	facts gaps.Facts,
	factsDigest string,
	packRefs []catalog.PackRef,
	packRefsDigest string,
	selections []WizardGapsSelectionInput,
	selectionsDigest string,
	evaluation gaps.Result,
) (WizardGapsInputReceipt, error) {
	receipt := WizardGapsInputReceipt{
		RequestRef: request.RequestRef, RequestFingerprint: request.RequestFingerprint,
		ActorRef: request.ActorRef, ProjectRef: request.ProjectRef,
		StateRef: request.StateRef, ExpectedRevision: request.ExpectedRevision,
		Origin: origin, SourceIntakeReceiptRef: source.Receipt.Ref,
		OutcomeKind: outcome.Kind, OutcomeReceiptRef: outcome.ReceiptRef,
		Facts: facts, FactsDigest: factsDigest,
		PackRefs:                append([]catalog.PackRef(nil), packRefs...),
		PackRefsDigest:          packRefsDigest,
		Selections:              cloneWizardGapsSelections(selections),
		SelectionsDigest:        selectionsDigest,
		EvaluatorIdentity:       request.EvaluatorIdentity,
		AuthorizationReceiptRef: request.AuthorizationReceiptRef,
	}
	if err := validateWizardGapsInputReceiptShape(receipt); err != nil {
		return WizardGapsInputReceipt{}, err
	}
	recomputedSelectionsDigest, err := wizardGapsSelectionsDigest(
		receipt.Selections,
	)
	if err != nil || recomputedSelectionsDigest != receipt.SelectionsDigest {
		return WizardGapsInputReceipt{}, errors.Join(
			errors.New("application.wizard_gaps_selections_digest_mismatch"),
			err,
		)
	}
	receipt.Ref = wizardGapsInputReceiptRef(receipt)
	receipt.ResultSnapshot, err = buildWizardGapsResultSnapshot(
		receipt.Ref,
		evaluation,
	)
	if err != nil {
		return WizardGapsInputReceipt{}, err
	}
	return receipt, nil
}

func cloneWizardGapsSelections(
	values []WizardGapsSelectionInput,
) []WizardGapsSelectionInput {
	result := make([]WizardGapsSelectionInput, len(values))
	copy(result, values)
	return result
}

func validateWizardGapsInputRecord(
	request WizardGapsInputReplayRequest,
	record WizardGapsInputRecord,
) error {
	receipt := record.Receipt
	if receipt.RequestRef != request.RequestRef ||
		receipt.RequestFingerprint != request.RequestFingerprint ||
		receipt.ActorRef != request.ActorRef ||
		receipt.ProjectRef != request.ProjectRef ||
		receipt.StateRef != request.StateRef ||
		receipt.ExpectedRevision != request.ExpectedRevision ||
		receipt.EvaluatorIdentity != request.EvaluatorIdentity ||
		receipt.AuthorizationReceiptRef != request.AuthorizationReceiptRef {
		return &StateError{Code: StateConflict}
	}
	return ValidateWizardGapsInputRecord(record)
}

// ValidateWizardGapsInputRecord validates adapter hydration without trusting
// stored digests, refs or duplicated scope fields.
func ValidateWizardGapsInputRecord(record WizardGapsInputRecord) error {
	receipt := record.Receipt
	if err := validateWizardGapsInputReceiptShape(receipt); err != nil {
		return wizardGapsInputConflict("shape", err)
	}
	canonicalFacts, canonicalPackRefs, err := gaps.CanonicalContext(
		receipt.Facts,
		receipt.PackRefs,
	)
	if err != nil || canonicalFacts != receipt.Facts ||
		!equalWizardGapsPackRefs(canonicalPackRefs, receipt.PackRefs) {
		return wizardGapsInputConflict("canonical-context", err)
	}
	factsDigest, err := wizardGapsFactsDigest(receipt.Facts)
	if err != nil || factsDigest != receipt.FactsDigest {
		return wizardGapsInputConflict("facts-digest", err)
	}
	packRefsDigest, err := wizardGapsPackRefsDigest(receipt.PackRefs)
	if err != nil || packRefsDigest != receipt.PackRefsDigest {
		return wizardGapsInputConflict("pack-refs-digest", err)
	}
	if receipt.RequestFingerprint != wizardGapsRequestFingerprint(
		receipt.ActorRef,
		receipt.ProjectRef,
		receipt.StateRef,
		receipt.ExpectedRevision,
		receipt.Origin,
		receipt.FactsDigest,
		receipt.PackRefsDigest,
		receipt.EvaluatorIdentity,
		receipt.AuthorizationReceiptRef,
	) {
		return wizardGapsInputConflict("request-fingerprint", nil)
	}
	selectionsDigest, err := wizardGapsSelectionsDigest(receipt.Selections)
	if err != nil || selectionsDigest != receipt.SelectionsDigest {
		return wizardGapsInputConflict("selections-digest", err)
	}
	if receipt.Ref != wizardGapsInputReceiptRef(receipt) ||
		record.SourceRecord.Receipt.Ref != receipt.SourceIntakeReceiptRef ||
		record.SourceRecord.State.Revision() != receipt.ExpectedRevision ||
		record.SourceRecord.Receipt.Revision != receipt.ExpectedRevision {
		return wizardGapsInputConflict("source-binding", nil)
	}
	if err := validateStoredIntakeRecord(
		receipt.ActorRef, receipt.ProjectRef, receipt.StateRef,
		record.SourceRecord,
	); err != nil {
		return wizardGapsInputConflict("source-record", err)
	}
	if err := validateStoredIntakeRecord(
		receipt.ActorRef, receipt.ProjectRef, receipt.StateRef,
		record.OutcomeRecord,
	); err != nil {
		return wizardGapsInputConflict("outcome-record", err)
	}
	switch receipt.OutcomeKind {
	case WizardGapsRequestOutcomeIntakeMutation:
		if record.OutcomeRecord.Receipt.Ref != receipt.OutcomeReceiptRef ||
			record.OutcomeRecord.Receipt.RequestRef != receipt.RequestRef ||
			record.OutcomeRecord.State.Revision() != receipt.ExpectedRevision+1 {
			return wizardGapsInputConflict("mutation-outcome", nil)
		}
	case WizardGapsRequestOutcomeNoOp:
		if !validWizardGapsNoOpOutcomeRef(receipt.OutcomeReceiptRef) ||
			record.OutcomeRecord.Receipt != record.SourceRecord.Receipt ||
			!reflectIntakeStateEqual(
				record.OutcomeRecord.State,
				record.SourceRecord.State,
			) {
			return wizardGapsInputConflict("noop-outcome", nil)
		}
	default:
		return wizardGapsInputConflict("outcome-kind", nil)
	}
	return nil
}

// ValidateWizardGapsInputEvaluation recomputes canonical selections and
// verifies that the persisted source produces the linked state outcome. It
// does not persist the result snapshot or make EvaluationReplayExact true.
func ValidateWizardGapsInputEvaluation(record WizardGapsInputRecord) error {
	if err := ValidateWizardGapsInputRecord(record); err != nil {
		return err
	}
	evaluator, err := gaps.BuiltInEvaluatorRegistry().Resolve(
		record.Receipt.EvaluatorIdentity,
	)
	if err != nil {
		return wizardGapsInputConflict("historical-evaluator", err)
	}
	evaluated, err := evaluateWizardGapsDetailed(
		record.SourceRecord.State,
		record.Receipt.Facts,
		record.Receipt.PackRefs,
		evaluator,
	)
	if err != nil ||
		evaluated.selectionsDigest != record.Receipt.SelectionsDigest ||
		!equalWizardGapsSelections(evaluated.selections, record.Receipt.Selections) {
		return wizardGapsInputConflict("historical-selections", err)
	}
	if !wizardGapsResultSnapshotEmpty(record.Receipt.ResultSnapshot) {
		if _, err := validateWizardGapsResultSnapshot(
			record.Receipt.Ref,
			record.Receipt.ResultSnapshot,
			evaluated.result,
		); err != nil {
			return wizardGapsInputConflict("result-snapshot", err)
		}
	}
	change, err := wizardGapsChangeWithReconciliation(
		record.SourceRecord.State,
		record.Receipt.Origin,
		evaluator.Identity(),
		evaluated.result,
		evaluated.reconcilable,
	)
	if err != nil {
		return wizardGapsInputConflict("historical-change", err)
	}
	empty := len(change.Issues) == 0 && len(change.Questions) == 0 &&
		len(change.QuestionRevisions) == 0 &&
		len(change.QuestionRetirements) == 0
	switch record.Receipt.OutcomeKind {
	case WizardGapsRequestOutcomeNoOp:
		if !empty {
			return wizardGapsInputConflict("historical-noop", nil)
		}
		expectedOutcome, err := buildWizardGapsNoOpOutcome(
			wizardGapsNoOpReplayRequest(
				wizardGapsInputReplayRequest(record.Receipt),
			),
			record.SourceRecord,
			evaluated.result,
		)
		if err != nil ||
			expectedOutcome.Ref != record.Receipt.OutcomeReceiptRef {
			return wizardGapsInputConflict("historical-noop-outcome", err)
		}
	case WizardGapsRequestOutcomeIntakeMutation:
		if empty {
			return wizardGapsInputConflict("historical-mutation", nil)
		}
		next, err := intake.Apply(record.SourceRecord.State, change)
		if err != nil || !reflectIntakeStateEqual(next, record.OutcomeRecord.State) {
			return wizardGapsInputConflict("historical-mutation", err)
		}
		fingerprint, err := intakeApplyRequestFingerprint(
			record.Receipt.ActorRef,
			record.Receipt.ProjectRef,
			change,
			record.Receipt.AuthorizationReceiptRef,
		)
		if err != nil {
			return wizardGapsInputConflict("historical-mutation-fingerprint", err)
		}
		expectedReceipt, err := buildIntakeReceipt(
			IntakeOperationApply,
			record.Receipt.RequestRef,
			fingerprint,
			record.Receipt.ActorRef,
			record.Receipt.ProjectRef,
			next,
			record.Receipt.ExpectedRevision,
			record.Receipt.AuthorizationReceiptRef,
		)
		if err != nil || expectedReceipt != record.OutcomeRecord.Receipt ||
			record.Receipt.OutcomeReceiptRef != expectedReceipt.Ref {
			return wizardGapsInputConflict("historical-mutation-receipt", err)
		}
	default:
		return wizardGapsInputConflict("historical-outcome-kind", nil)
	}
	return nil
}

func wizardGapsInputReplayRequest(
	receipt WizardGapsInputReceipt,
) WizardGapsInputReplayRequest {
	return WizardGapsInputReplayRequest{
		RequestRef:              receipt.RequestRef,
		RequestFingerprint:      receipt.RequestFingerprint,
		ActorRef:                receipt.ActorRef,
		ProjectRef:              receipt.ProjectRef,
		StateRef:                receipt.StateRef,
		ExpectedRevision:        receipt.ExpectedRevision,
		EvaluatorIdentity:       receipt.EvaluatorIdentity,
		AuthorizationReceiptRef: receipt.AuthorizationReceiptRef,
	}
}

func equalWizardGapsSelections(
	left,
	right []WizardGapsSelectionInput,
) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func wizardGapsRequestFingerprint(
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	stateRef intake.Ref,
	expectedRevision intake.Revision,
	origin intake.Origin,
	factsDigest,
	packRefsDigest string,
	evaluatorIdentity intake.DerivationIdentity,
	authorizationReceiptRef string,
) string {
	return fingerprintFields(
		"orquesta.wizard.gaps.request.v2",
		actorRef.String(),
		projectRef.String(),
		string(stateRef),
		strconv.FormatUint(uint64(expectedRevision), 10),
		string(origin),
		factsDigest,
		packRefsDigest,
		evaluatorIdentity.Schema,
		evaluatorIdentity.Version,
		evaluatorIdentity.SemanticDigest,
		authorizationReceiptRef,
	)
}

func wizardGapsInputConflict(stage string, cause error) error {
	stageErr := errors.New("application.wizard_gaps_input_" + stage + "_invalid")
	if cause != nil {
		stageErr = errors.Join(stageErr, cause)
	}
	return &StateError{Code: StateConflict, Cause: stageErr}
}

func equalWizardGapsPackRefs(
	left,
	right []catalog.PackRef,
) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func validateWizardGapsInputReceiptShape(receipt WizardGapsInputReceipt) error {
	if !validIntakeRequestRef(receipt.RequestRef) ||
		!validWizardGapsCanonicalHash(receipt.RequestFingerprint) ||
		receipt.ExpectedRevision == 0 ||
		(receipt.Origin != intake.OriginChat && receipt.Origin != intake.OriginForm) ||
		receipt.SourceIntakeReceiptRef == "" ||
		receipt.OutcomeReceiptRef == "" ||
		!validWizardGapsCanonicalHash(receipt.FactsDigest) ||
		!validWizardGapsCanonicalHash(receipt.PackRefsDigest) ||
		!validWizardGapsCanonicalHash(receipt.SelectionsDigest) ||
		receipt.AuthorizationReceiptRef == "" {
		return errors.New("application.wizard_gaps_input_receipt_invalid")
	}
	if err := validateIntakeScope(receipt.ActorRef, receipt.ProjectRef); err != nil {
		return err
	}
	if !validIntakeStateRef(receipt.StateRef) {
		return errors.New("application.wizard_gaps_input_state_ref_invalid")
	}
	identity, err := intake.NewDerivationIdentity(
		receipt.EvaluatorIdentity.Schema,
		receipt.EvaluatorIdentity.Version,
		receipt.EvaluatorIdentity.SemanticDigest,
	)
	if err != nil || identity != receipt.EvaluatorIdentity {
		return errors.New("application.wizard_gaps_input_evaluator_invalid")
	}
	return nil
}

func wizardGapsInputReceiptRef(receipt WizardGapsInputReceipt) string {
	return "wizard-gaps-input:" + fingerprintFields(
		wizardGapsInputReceiptSchema,
		receipt.RequestRef,
		receipt.RequestFingerprint,
		receipt.ActorRef.String(),
		receipt.ProjectRef.String(),
		string(receipt.StateRef),
		strconv.FormatUint(uint64(receipt.ExpectedRevision), 10),
		string(receipt.Origin),
		receipt.SourceIntakeReceiptRef,
		string(receipt.OutcomeKind),
		receipt.OutcomeReceiptRef,
		receipt.FactsDigest,
		receipt.PackRefsDigest,
		receipt.SelectionsDigest,
		receipt.EvaluatorIdentity.Schema,
		receipt.EvaluatorIdentity.Version,
		receipt.EvaluatorIdentity.SemanticDigest,
		receipt.AuthorizationReceiptRef,
	)
}

type wizardGapsFactsDigestView struct {
	Surface                gaps.Surface       `json:"surface"`
	SharingIntent          gaps.SharingIntent `json:"sharing_intent"`
	CorporateIdentity      gaps.Declaration   `json:"corporate_identity"`
	TargetUsers            gaps.Declaration   `json:"target_users"`
	IntegrationAuth        gaps.Declaration   `json:"integration_auth"`
	IntegrationCriticality gaps.Declaration   `json:"integration_criticality"`
}

func wizardGapsFactsDigest(facts gaps.Facts) (string, error) {
	encoded, err := json.Marshal(wizardGapsFactsDigestView{
		Surface: facts.Surface, SharingIntent: facts.SharingIntent,
		CorporateIdentity: facts.CorporateIdentity, TargetUsers: facts.TargetUsers,
		IntegrationAuth:        facts.IntegrationAuth,
		IntegrationCriticality: facts.IntegrationCriticality,
	})
	if err != nil {
		return "", errors.New("application.wizard_gaps_facts_invalid")
	}
	return fingerprintFields("orquesta.wizard.gaps.facts.v1", string(encoded)), nil
}

func wizardGapsPackRefsDigest(packRefs []catalog.PackRef) (string, error) {
	refs := make([]string, len(packRefs))
	for index, ref := range packRefs {
		refs[index] = ref.String()
		if refs[index] == "" {
			return "", errors.New("application.wizard_gaps_pack_refs_invalid")
		}
	}
	if !sort.StringsAreSorted(refs) {
		return "", errors.New("application.wizard_gaps_pack_refs_not_canonical")
	}
	for index := 1; index < len(refs); index++ {
		if refs[index] == refs[index-1] {
			return "", errors.New("application.wizard_gaps_pack_refs_not_canonical")
		}
	}
	encoded, err := json.Marshal(refs)
	if err != nil {
		return "", errors.New("application.wizard_gaps_pack_refs_invalid")
	}
	return fingerprintFields("orquesta.wizard.gaps.pack-refs.v1", string(encoded)), nil
}

type WizardGapsSelectionInput struct {
	Dimension string `json:"dimension,omitempty"`
	Question  string `json:"question,omitempty"`
	Option    string `json:"option"`
	FreeText  string `json:"free_text,omitempty"`
}

func canonicalWizardGapsSelections(
	selections []gaps.Selection,
	questionSelections []gaps.QuestionSelection,
) ([]WizardGapsSelectionInput, string, error) {
	values := make(
		[]WizardGapsSelectionInput,
		0,
		len(selections)+len(questionSelections),
	)
	for _, selection := range selections {
		values = append(values, WizardGapsSelectionInput{
			Dimension: string(selection.Dimension),
			Option:    string(selection.Option), FreeText: selection.FreeText,
		})
	}
	for _, selection := range questionSelections {
		values = append(values, WizardGapsSelectionInput{
			Question: string(selection.Question),
			Option:   string(selection.Option), FreeText: selection.FreeText,
		})
	}
	sort.Slice(values, func(left, right int) bool {
		a, b := values[left], values[right]
		if a.Dimension != b.Dimension {
			return a.Dimension < b.Dimension
		}
		if a.Question != b.Question {
			return a.Question < b.Question
		}
		if a.Option != b.Option {
			return a.Option < b.Option
		}
		return a.FreeText < b.FreeText
	})
	digest, err := wizardGapsSelectionsDigest(values)
	if err != nil {
		return nil, "", err
	}
	return values, digest, nil
}

func wizardGapsSelectionsDigest(
	values []WizardGapsSelectionInput,
) (string, error) {
	for index, value := range values {
		if (value.Dimension == "") == (value.Question == "") ||
			value.Option == "" {
			return "", errors.New("application.wizard_gaps_selections_invalid")
		}
		if index > 0 {
			previous := values[index-1]
			if previous.Dimension > value.Dimension ||
				(previous.Dimension == value.Dimension &&
					previous.Question > value.Question) ||
				(previous.Dimension == value.Dimension &&
					previous.Question == value.Question &&
					previous.Option > value.Option) ||
				(previous.Dimension == value.Dimension &&
					previous.Question == value.Question &&
					previous.Option == value.Option &&
					previous.FreeText >= value.FreeText) {
				return "", errors.New(
					"application.wizard_gaps_selections_not_canonical",
				)
			}
		}
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		return "", errors.New("application.wizard_gaps_selections_invalid")
	}
	return fingerprintFields("orquesta.wizard.gaps.selections.v1", string(encoded)), nil
}
