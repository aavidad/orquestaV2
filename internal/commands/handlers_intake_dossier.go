package commands

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/intake"
)

type prepareIntakeDossierInput struct {
	IntakeRef              string                             `json:"intake_ref"`
	ExpectedRevision       intake.Revision                    `json:"expected_revision"`
	SourceIntakeReceiptRef string                             `json:"source_intake_receipt_ref"`
	Statement              string                             `json:"statement"`
	Objective              string                             `json:"objective"`
	Sections               []application.IntakeDossierSection `json:"sections"`
	Diagrams               []application.IntakeDossierDiagram `json:"diagrams"`
	RiskRefs               []application.IntakeRiskRef        `json:"risk_refs"`
	Plan                   planInput                          `json:"plan"`
}

type intakeDossierView struct {
	Schema                 string                             `json:"schema"`
	DossierRef             string                             `json:"dossier_ref"`
	ActorRef               string                             `json:"actor_ref"`
	ProjectRef             string                             `json:"project_ref"`
	IntakeRef              string                             `json:"intake_ref"`
	IntakeRevision         uint64                             `json:"intake_revision"`
	IntakeDigest           string                             `json:"intake_digest"`
	SourceIntakeReceiptRef string                             `json:"source_intake_receipt_ref"`
	Statement              string                             `json:"statement"`
	Objective              string                             `json:"objective"`
	Sections               []application.IntakeDossierSection `json:"sections"`
	Diagrams               []application.IntakeDossierDiagram `json:"diagrams"`
	Decisions              []intake.Decision                  `json:"decisions"`
	RiskRefs               []application.IntakeRiskRef        `json:"risk_refs"`
	Plan                   planInput                          `json:"plan"`
	PlanDigest             string                             `json:"plan_digest"`
	Digest                 string                             `json:"digest"`
	GenerationReceipt      intakeDossierGenerationReceiptView `json:"generation_receipt"`
}

type intakeDossierGenerationReceiptView struct {
	ReceiptRef              string `json:"receipt_ref"`
	RequestRef              string `json:"request_ref"`
	RequestFingerprint      string `json:"request_fingerprint"`
	ActorRef                string `json:"actor_ref"`
	ProjectRef              string `json:"project_ref"`
	IntakeRef               string `json:"intake_ref"`
	IntakeRevision          uint64 `json:"intake_revision"`
	IntakeDigest            string `json:"intake_digest"`
	SourceIntakeReceiptRef  string `json:"source_intake_receipt_ref"`
	DossierRef              string `json:"dossier_ref"`
	DossierDigest           string `json:"dossier_digest"`
	PlanDigest              string `json:"plan_digest"`
	AuthorizationReceiptRef string `json:"authorization_receipt_ref"`
}

type intakeDossierConfirmationView struct {
	ReceiptRef              string    `json:"receipt_ref"`
	RequestRef              string    `json:"request_ref"`
	RequestFingerprint      string    `json:"request_fingerprint"`
	PrincipalRef            string    `json:"principal_ref"`
	ActorRef                string    `json:"actor_ref"`
	ProjectRef              string    `json:"project_ref"`
	IntakeRef               string    `json:"intake_ref"`
	IntakeRevision          uint64    `json:"intake_revision"`
	IntakeDigest            string    `json:"intake_digest"`
	SourceIntakeReceiptRef  string    `json:"source_intake_receipt_ref"`
	DossierRef              string    `json:"dossier_ref"`
	DossierDigest           string    `json:"dossier_digest"`
	PlanDigest              string    `json:"plan_digest"`
	GoalRef                 string    `json:"goal_ref"`
	AppSpecRef              string    `json:"app_spec_ref"`
	SpecHash                string    `json:"spec_hash"`
	AuthorizationReceiptRef string    `json:"authorization_receipt_ref"`
	ConfirmedAt             time.Time `json:"confirmed_at"`
}

func handlePrepareIntakeDossier(
	ctx context.Context,
	api applicationAPI,
	bound handlerContext,
	payload json.RawMessage,
) (json.RawMessage, error) {
	var input prepareIntakeDossierInput
	if err := decodePayload(payload, &input); err != nil {
		return nil, err
	}
	result, err := api.PrepareIntakeDossier(
		ctx,
		bound.access,
		application.PrepareIntakeDossierRequest{
			RequestRef:             bound.requestRef,
			ActorRef:               bound.principal.ActorRef,
			ProjectRef:             bound.projectRef,
			StateRef:               intake.Ref(input.IntakeRef),
			ExpectedRevision:       input.ExpectedRevision,
			SourceIntakeReceiptRef: input.SourceIntakeReceiptRef,
			Plan:                   *applicationPlan(&input.Plan),
			Input: application.IntakeDossierInput{
				Statement: input.Statement,
				Objective: input.Objective,
				Sections:  input.Sections,
				Diagrams:  input.Diagrams,
				RiskRefs:  input.RiskRefs,
			},
		},
	)
	return marshalApplication(
		struct {
			Dossier intakeDossierView `json:"dossier"`
		}{Dossier: projectIntakeDossier(result.Record)},
		normalizeIntakeDossierError(err),
	)
}

func handleGetIntakeDossier(
	ctx context.Context,
	api applicationAPI,
	bound handlerContext,
	payload json.RawMessage,
) (json.RawMessage, error) {
	var input struct {
		DossierRef string `json:"dossier_ref"`
	}
	if err := decodePayload(payload, &input); err != nil {
		return nil, err
	}
	record, err := api.GetIntakeDossier(
		ctx,
		bound.access,
		application.GetIntakeDossierRequest{
			ActorRef:   bound.principal.ActorRef,
			ProjectRef: bound.projectRef,
			DossierRef: application.IntakeDossierRef(input.DossierRef),
		},
	)
	return marshalApplication(
		struct {
			Dossier intakeDossierView `json:"dossier"`
		}{Dossier: projectIntakeDossier(record)},
		normalizeIntakeDossierError(err),
	)
}

func handleConfirmIntakeDossier(
	ctx context.Context,
	api applicationAPI,
	bound handlerContext,
	payload json.RawMessage,
) (json.RawMessage, error) {
	var input struct {
		DossierRef string `json:"dossier_ref"`
		Confirm    bool   `json:"confirm"`
	}
	if err := decodePayload(payload, &input); err != nil {
		return nil, err
	}
	result, err := api.ConfirmIntakeDossier(
		ctx,
		bound.access,
		application.ConfirmIntakeDossierRequest{
			RequestRef: bound.requestRef,
			DossierRef: application.IntakeDossierRef(input.DossierRef),
			Confirm:    input.Confirm,
		},
	)
	return marshalApplication(
		struct {
			Goal         goalReceiptView               `json:"goal"`
			Confirmation intakeDossierConfirmationView `json:"confirmation"`
		}{
			Goal:         projectGoalReceipt(result.Record.Goal),
			Confirmation: projectIntakeDossierConfirmation(result.Confirmation),
		},
		err,
	)
}

func projectIntakeDossier(record application.IntakeDossierRecord) intakeDossierView {
	dossier := record.Dossier
	receipt := record.Receipt
	return intakeDossierView{
		Schema:                 dossier.Schema(),
		DossierRef:             string(dossier.Ref()),
		ActorRef:               record.ActorRef.String(),
		ProjectRef:             record.ProjectRef.String(),
		IntakeRef:              string(dossier.StateRef()),
		IntakeRevision:         uint64(dossier.StateRevision()),
		IntakeDigest:           dossier.StateDigest(),
		SourceIntakeReceiptRef: dossier.SourceIntakeReceiptRef(),
		Statement:              dossier.Statement(),
		Objective:              dossier.Objective(),
		Sections:               nonNil(dossier.Sections()),
		Diagrams:               nonNil(dossier.Diagrams()),
		Decisions:              nonNil(dossier.Decisions()),
		RiskRefs:               nonNil(dossier.RiskRefs()),
		Plan:                   projectIntakeDossierPlan(dossier.Plan()),
		PlanDigest:             dossier.PlanDigest(),
		Digest:                 dossier.Digest(),
		GenerationReceipt: intakeDossierGenerationReceiptView{
			ReceiptRef:              receipt.Ref,
			RequestRef:              receipt.RequestRef,
			RequestFingerprint:      receipt.RequestFingerprint,
			ActorRef:                receipt.ActorRef.String(),
			ProjectRef:              receipt.ProjectRef.String(),
			IntakeRef:               string(receipt.StateRef),
			IntakeRevision:          uint64(receipt.StateRevision),
			IntakeDigest:            receipt.StateDigest,
			SourceIntakeReceiptRef:  receipt.SourceIntakeReceiptRef,
			DossierRef:              string(receipt.DossierRef),
			DossierDigest:           receipt.DossierDigest,
			PlanDigest:              receipt.PlanDigest,
			AuthorizationReceiptRef: receipt.AuthorizationReceiptRef,
		},
	}
}

func projectIntakeDossierPlan(plan application.PlanSpec) planInput {
	projected := planInput{
		Phases:    make([]phaseInput, 0, len(plan.Phases)),
		WorkItems: make([]workItemInput, 0, len(plan.WorkItems)),
	}
	for _, phase := range plan.Phases {
		projected.Phases = append(projected.Phases, phaseInput{
			Ref:           phase.Ref,
			Key:           phase.Key,
			TemplateRef:   phase.TemplateRef,
			InputRefs:     nonNil(phase.InputRefs),
			CriterionRefs: nonNil(phase.CriterionRefs),
		})
	}
	for _, item := range plan.WorkItems {
		tests := make([]requiredTestInput, 0, len(item.RequiredTests))
		for _, test := range item.RequiredTests {
			tests = append(tests, requiredTestInput{
				Ref:              test.Ref,
				ToolRef:          test.ToolRef,
				Arguments:        nonNil(test.Arguments),
				WorkingDirectory: test.WorkingDirectory,
			})
		}
		projected.WorkItems = append(projected.WorkItems, workItemInput{
			Key:                 item.Key,
			Objective:           item.Objective,
			Phase:               item.Phase,
			Role:                item.Role,
			Parent:              item.Parent,
			HandoffRequired:     item.HandoffRequired,
			Dependencies:        nonNil(item.Dependencies),
			WriteSet:            nonNil(item.WriteSet),
			RequiredTests:       tests,
			SkillRefs:           nonNil(item.SkillRefs),
			ToolRefs:            nonNil(item.ToolRefs),
			CapabilityRefs:      nonNil(item.CapabilityRefs),
			OutputContract:      string(item.OutputContract),
			SecurityCriticality: string(item.SecurityCriticality),
			ReasoningEffort:     string(item.ReasoningEffort),
			CouncilPolicy:       string(item.CouncilPolicy),
		})
	}
	return projected
}

func projectIntakeDossierConfirmation(
	confirmation application.IntakeDossierConfirmation,
) intakeDossierConfirmationView {
	return intakeDossierConfirmationView{
		ReceiptRef:              confirmation.Ref,
		RequestRef:              confirmation.RequestRef,
		RequestFingerprint:      confirmation.RequestFingerprint,
		PrincipalRef:            confirmation.PrincipalRef.String(),
		ActorRef:                confirmation.ActorRef.String(),
		ProjectRef:              confirmation.ProjectRef.String(),
		IntakeRef:               string(confirmation.StateRef),
		IntakeRevision:          uint64(confirmation.StateRevision),
		IntakeDigest:            confirmation.StateDigest,
		SourceIntakeReceiptRef:  confirmation.SourceIntakeReceiptRef,
		DossierRef:              string(confirmation.DossierRef),
		DossierDigest:           confirmation.DossierDigest,
		PlanDigest:              confirmation.PlanDigest,
		GoalRef:                 confirmation.GoalRef.String(),
		AppSpecRef:              confirmation.AppSpecRef.String(),
		SpecHash:                confirmation.SpecHash,
		AuthorizationReceiptRef: confirmation.AuthorizationReceiptRef,
		ConfirmedAt:             confirmation.ConfirmedAt,
	}
}

func normalizeIntakeDossierError(err error) error {
	if errors.Is(err, application.ErrIntakeDossierInvalid) {
		return commandError{code: CodeInvalidRequest, cause: err}
	}
	return normalizeIntakeError(err)
}
