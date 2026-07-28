package application

import (
	"context"
	"strings"
	"testing"

	"orquesta/internal/intake"
	"orquesta/internal/wizard/catalog"
	"orquesta/internal/wizard/gaps"
)

func TestWizardGapsInputReceiptPersistsCanonicalContextAndSelections(
	t *testing.T,
) {
	system, service := newWizardGapsTestSystem(t)
	packs := catalog.DomainPackRefs()
	request := wizardGapsRequest(
		t, system, "request:wizard-gaps-input-canonical", 1,
	)
	request.Facts = gaps.Facts{
		Surface:                gaps.SurfaceHumanUI,
		SharingIntent:          gaps.SharingShared,
		CorporateIdentity:      gaps.DeclarationDeclared,
		TargetUsers:            gaps.DeclarationMissing,
		IntegrationAuth:        gaps.DeclarationDeclared,
		IntegrationCriticality: gaps.DeclarationMissing,
	}
	request.PackRefs = []catalog.PackRef{
		packs[1], catalog.FoundationPackRef(), packs[0], packs[1],
	}
	first, err := service.ApplyWizardGaps(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	record := mustWizardGapsInputRecord(t, system, request.RequestRef)
	if record.Receipt.Facts != request.Facts ||
		len(record.Receipt.PackRefs) != 2 ||
		record.Receipt.PackRefs[0] != packs[0] ||
		record.Receipt.PackRefs[1] != packs[1] ||
		record.Receipt.FactsDigest != first.InputDurability.FactsDigest ||
		record.Receipt.PackRefsDigest != first.InputDurability.PackRefsDigest ||
		record.Receipt.SelectionsDigest !=
			first.InputDurability.SelectionsDigest ||
		record.Receipt.SourceIntakeReceiptRef !=
			first.InputDurability.SourceIntakeReceiptRef {
		t.Fatalf("receipt=%+v durability=%+v", record.Receipt, first.InputDurability)
	}
	if err := ValidateWizardGapsInputEvaluation(record); err != nil {
		t.Fatalf("canonical input validation: %v", err)
	}

	question := mustWizardDimensionQuestion(t, first.Evaluation, gaps.DimensionU1)
	choice, found := question.RecommendedOption()
	if !found {
		t.Fatal("U1 recommendation missing")
	}
	answered, err := system.service.ApplyIntake(
		context.Background(),
		ApplyIntakeRequest{
			RequestRef: "request:wizard-gaps-input-answer",
			ActorRef:   system.actor,
			ProjectRef: system.project,
			Change: intake.Change{
				StateRef:         "intake:shared",
				ExpectedRevision: first.Record.State.Revision(),
				Origin:           intake.OriginForm,
				Choices: []intake.Choice{{
					QuestionRef: intake.QuestionRef(question.Ref()),
					OptionRef:   intake.OptionRef(choice.Ref()),
				}},
			},
			AuthorizationReceipt: system.authorizationFor(
				t,
				IntakeOperationApply,
				"request:wizard-gaps-input-answer",
			),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	selectedRequest := wizardGapsRequest(
		t,
		system,
		"request:wizard-gaps-input-selected",
		answered.Record.State.Revision(),
	)
	selectedRequest.Facts = request.Facts
	selectedRequest.PackRefs = request.PackRefs
	if _, err = service.ApplyWizardGaps(
		context.Background(), selectedRequest,
	); err != nil {
		t.Fatal(err)
	}
	selected := mustWizardGapsInputRecord(
		t, system, selectedRequest.RequestRef,
	)
	if len(selected.Receipt.Selections) == 0 ||
		selected.Receipt.Selections[0].Dimension != string(gaps.DimensionU1) ||
		selected.Receipt.Selections[0].Option != string(choice.Ref()) {
		t.Fatalf("selections=%+v", selected.Receipt.Selections)
	}
	if err := ValidateWizardGapsInputEvaluation(selected); err != nil {
		t.Fatalf("selected input validation: %v", err)
	}
}

func TestWizardGapsInputValidationRejectsCoherentContextTampering(
	t *testing.T,
) {
	tests := map[string]func(*testing.T, *WizardGapsInputRecord){
		"facts": func(t *testing.T, record *WizardGapsInputRecord) {
			record.Receipt.Facts.Surface = gaps.Surface("foreign")
			var err error
			record.Receipt.FactsDigest, err = wizardGapsFactsDigest(
				record.Receipt.Facts,
			)
			if err != nil {
				t.Fatal(err)
			}
			rebindWizardGapsInputRequest(record)
		},
		"pack refs": func(t *testing.T, record *WizardGapsInputRecord) {
			unknown, err := catalog.NewPackRef("pack:unknown")
			if err != nil {
				t.Fatal(err)
			}
			record.Receipt.PackRefs = []catalog.PackRef{unknown}
			record.Receipt.PackRefsDigest, err = wizardGapsPackRefsDigest(
				record.Receipt.PackRefs,
			)
			if err != nil {
				t.Fatal(err)
			}
			rebindWizardGapsInputRequest(record)
		},
	}
	for name, tamper := range tests {
		t.Run(name, func(t *testing.T) {
			system, service := newWizardGapsTestSystem(t)
			request := wizardGapsRequest(
				t, system, "request:wizard-gaps-input-tamper-"+name, 1,
			)
			if _, err := service.ApplyWizardGaps(
				context.Background(), request,
			); err != nil {
				t.Fatal(err)
			}
			record := mustWizardGapsInputRecord(
				t, system, request.RequestRef,
			)
			tamper(t, &record)
			if err := ValidateWizardGapsInputRecord(record); !IsStateError(
				err, StateConflict,
			) {
				t.Fatalf("tampered input err=%v receipt=%+v", err, record.Receipt)
			}
		})
	}
}

func TestWizardGapsInputEvaluationRejectsCoherentSelectionTampering(
	t *testing.T,
) {
	system, service := newWizardGapsTestSystem(t)
	request := wizardGapsRequest(
		t, system, "request:wizard-gaps-input-selection-tamper", 1,
	)
	if _, err := service.ApplyWizardGaps(
		context.Background(), request,
	); err != nil {
		t.Fatal(err)
	}
	record := mustWizardGapsInputRecord(t, system, request.RequestRef)
	selections, digest, err := canonicalWizardGapsSelections(
		[]gaps.Selection{{
			Dimension: gaps.DimensionU1,
			Option:    "intake-option:wizard.u1.team",
		}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	record.Receipt.Selections = selections
	record.Receipt.SelectionsDigest = digest
	record.Receipt.Ref = wizardGapsInputReceiptRef(record.Receipt)
	if err := ValidateWizardGapsInputRecord(record); err != nil {
		t.Fatalf("coherent receipt should pass structural validation: %v", err)
	}
	if err := ValidateWizardGapsInputEvaluation(record); !IsStateError(
		err, StateConflict,
	) {
		t.Fatalf("tampered selections err=%v", err)
	}
}

func TestWizardGapsInputEvaluationRejectsCoherentMutationReceiptTampering(
	t *testing.T,
) {
	system, service := newWizardGapsTestSystem(t)
	request := wizardGapsRequest(
		t, system, "request:wizard-gaps-input-mutation-tamper", 1,
	)
	if _, err := service.ApplyWizardGaps(
		context.Background(), request,
	); err != nil {
		t.Fatal(err)
	}
	record := mustWizardGapsInputRecord(t, system, request.RequestRef)
	altered, err := buildIntakeReceipt(
		IntakeOperationApply,
		record.Receipt.RequestRef,
		strings.Repeat("a", 64),
		record.Receipt.ActorRef,
		record.Receipt.ProjectRef,
		record.OutcomeRecord.State,
		record.Receipt.ExpectedRevision,
		record.Receipt.AuthorizationReceiptRef,
	)
	if err != nil {
		t.Fatal(err)
	}
	record.OutcomeRecord.Receipt = altered
	record.Receipt.OutcomeReceiptRef = altered.Ref
	record.Receipt.Ref = wizardGapsInputReceiptRef(record.Receipt)
	if err := ValidateWizardGapsInputRecord(record); err != nil {
		t.Fatalf("coherent receipt should pass structural validation: %v", err)
	}
	if err := ValidateWizardGapsInputEvaluation(record); !IsStateError(
		err, StateConflict,
	) {
		t.Fatalf("tampered mutation receipt err=%v", err)
	}
}

func TestWizardGapsInputEvaluationRejectsCoherentNoOpOutcomeTampering(
	t *testing.T,
) {
	system, service := newWizardGapsTestSystem(t)
	firstRequest := wizardGapsRequest(
		t, system, "request:wizard-gaps-input-noop-base", 1,
	)
	first, err := service.ApplyWizardGaps(
		context.Background(), firstRequest,
	)
	if err != nil {
		t.Fatal(err)
	}
	request := wizardGapsRequest(
		t,
		system,
		"request:wizard-gaps-input-noop-tamper",
		first.Record.State.Revision(),
	)
	if _, err = service.ApplyWizardGaps(
		context.Background(), request,
	); err != nil {
		t.Fatal(err)
	}
	record := mustWizardGapsInputRecord(t, system, request.RequestRef)
	record.Receipt.OutcomeReceiptRef =
		"wizard-gaps-outcome:" + strings.Repeat("b", 64)
	record.Receipt.Ref = wizardGapsInputReceiptRef(record.Receipt)
	if err := ValidateWizardGapsInputRecord(record); err != nil {
		t.Fatalf("coherent receipt should pass structural validation: %v", err)
	}
	if err := ValidateWizardGapsInputEvaluation(record); !IsStateError(
		err, StateConflict,
	) {
		t.Fatalf("tampered no-op outcome err=%v", err)
	}
}

func mustWizardGapsInputRecord(
	t *testing.T,
	system intakeTestSystem,
	requestRef string,
) WizardGapsInputRecord {
	t.Helper()
	record, found := system.store.wizardGapsInputs[intakeRequestKey(
		system.actor, system.project, requestRef,
	)]
	if !found {
		t.Fatalf("wizard gaps input %q missing", requestRef)
	}
	return record
}

func rebindWizardGapsInputRequest(record *WizardGapsInputRecord) {
	receipt := &record.Receipt
	receipt.RequestFingerprint = wizardGapsRequestFingerprint(
		receipt.ActorRef,
		receipt.ProjectRef,
		receipt.StateRef,
		receipt.ExpectedRevision,
		receipt.Origin,
		receipt.FactsDigest,
		receipt.PackRefsDigest,
		receipt.EvaluatorIdentity,
		receipt.AuthorizationReceiptRef,
	)
	receipt.Ref = wizardGapsInputReceiptRef(*receipt)
}
