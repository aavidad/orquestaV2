package commands

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/intake"
	"orquesta/internal/wizard/catalog"
	"orquesta/internal/wizard/gaps"
)

func (api *fakeApplication) ApplyWizardGaps(
	_ context.Context,
	_ application.Access,
	request application.ApplyWizardGapsRequest,
) (application.ApplyWizardGapsResult, error) {
	api.called("ApplyWizardGaps")
	const intakeReceiptRef = "intake-receipt:" +
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	state, err := intake.NewState(
		request.StateRef,
		intake.Policy{MaxQuestionRounds: 2},
	)
	if err != nil {
		return application.ApplyWizardGapsResult{}, err
	}
	evaluation, err := gaps.EvaluateV1(gaps.Input{
		Facts: request.Facts, PackRefs: request.PackRefs,
	})
	if err != nil {
		return application.ApplyWizardGapsResult{}, err
	}
	return application.ApplyWizardGapsResult{
		Record: application.IntakeRecord{
			ActorRef: request.ActorRef, ProjectRef: request.ProjectRef,
			State: state, Receipt: application.IntakeReceipt{
				Ref: intakeReceiptRef,
			},
		},
		Evaluation:            evaluation,
		Changed:               true,
		EvaluationReplayExact: false,
		RequestRefReserved:    true,
		RequestOutcome: application.WizardGapsRequestOutcome{
			Kind:       application.WizardGapsRequestOutcomeIntakeMutation,
			ReceiptRef: intakeReceiptRef,
		},
		InputDurability: application.WizardGapsInputDurability{
			ReceiptRef: "wizard-gaps-input:" +
				"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
			SourceIntakeReceiptRef: intakeReceiptRef,
			Selections:             "wizard_gaps_input_receipt",
			SelectionsDigest:       "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Facts:                  "wizard_gaps_input_receipt",
			FactsDigest:            "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			PackRefs:               "wizard_gaps_input_receipt",
			PackRefsDigest:         "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		},
		EvaluatorIdentity: request.EvaluatorIdentity,
	}, nil
}

type captureWizardGapsApplication struct {
	*fakeApplication
	requests []application.ApplyWizardGapsRequest
}

func (api *captureWizardGapsApplication) ApplyWizardGaps(
	ctx context.Context,
	access application.Access,
	request application.ApplyWizardGapsRequest,
) (application.ApplyWizardGapsResult, error) {
	api.requests = append(api.requests, request)
	return api.fakeApplication.ApplyWizardGaps(ctx, access, request)
}

func TestWizardGapsCommandBindsAuthorityAndProjectsCompleteEvaluation(t *testing.T) {
	api := &captureWizardGapsApplication{
		fakeApplication: newFakeApplication(),
	}
	dispatcher, err := newDispatcher(
		api,
		newMemoryAudit(),
		APILimits{
			MaxRequestBytes: 64 << 10, MaxListLimit: 10,
			IntakeMaxQuestionRounds: 6,
		},
		exactTestExecutionAuthority(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	packRef := catalog.DomainPackRefs()[0].String()
	payload := canonicalWizardGapsPayload()
	payload["pack_refs"] = []any{packRef}
	result := invoke(
		t,
		dispatcher,
		"orquesta.intakes.wizard.gaps.apply",
		"request:wizard-gaps-public",
		payload,
		false,
	)
	if result.Failure != nil {
		t.Fatalf("result=%+v", result)
	}
	if len(api.requests) != 1 {
		t.Fatalf("requests=%d", len(api.requests))
	}
	request := api.requests[0]
	wantIdentity := gaps.EvaluatorV1Identity()
	if request.RequestRef != "request:wizard-gaps-public" ||
		request.ActorRef.String() != "actor:test" ||
		request.ProjectRef.String() != "project:test" ||
		request.StateRef != "intake:test" ||
		request.ExpectedRevision != 1 ||
		request.Origin != intake.OriginForm ||
		request.Facts.Surface != gaps.SurfaceServerService ||
		request.Facts.SharingIntent != gaps.SharingShared ||
		request.EvaluatorIdentity != wantIdentity ||
		len(request.PackRefs) != 1 ||
		request.PackRefs[0].String() != packRef {
		t.Fatalf("request=%+v", request)
	}
	var output wizardGapsResultView
	if err := json.Unmarshal(result.Data, &output); err != nil {
		t.Fatal(err)
	}
	if output.Intake.IntakeRef != "intake:test" ||
		output.Evaluation.SchemaVersion != gaps.SchemaVersion ||
		len(output.Evaluation.Issues) == 0 ||
		len(output.Evaluation.Questions) == 0 ||
		len(output.Evaluation.Questions[0].Options) == 0 ||
		len(output.Evaluation.PackRefs) != 1 ||
		output.Evaluation.PackRefs[0] != packRef ||
		output.InputDurability.ReceiptRef == "" ||
		output.InputDurability.SourceIntakeReceiptRef == "" ||
		output.InputDurability.Selections != "wizard_gaps_input_receipt" ||
		output.InputDurability.Facts != "wizard_gaps_input_receipt" ||
		output.InputDurability.PackRefs != "wizard_gaps_input_receipt" ||
		output.EvaluationReplayExact ||
		!output.RequestRefReserved ||
		output.RequestOutcome.Kind !=
			application.WizardGapsRequestOutcomeIntakeMutation ||
		output.RequestOutcome.ReceiptRef != output.Intake.ReceiptRef ||
		output.EvaluatorIdentity != wantIdentity {
		t.Fatalf("output=%+v", output)
	}
}

type invalidWizardGapsOutcomeApplication struct {
	*fakeApplication
}

func (api *invalidWizardGapsOutcomeApplication) ApplyWizardGaps(
	ctx context.Context,
	access application.Access,
	request application.ApplyWizardGapsRequest,
) (application.ApplyWizardGapsResult, error) {
	result, err := api.fakeApplication.ApplyWizardGaps(ctx, access, request)
	result.RequestOutcome = application.WizardGapsRequestOutcome{}
	return result, err
}

func TestWizardGapsCommandRejectsMissingApplicationRequestOutcome(t *testing.T) {
	api := &invalidWizardGapsOutcomeApplication{
		fakeApplication: newFakeApplication(),
	}
	dispatcher, err := newDispatcher(
		api,
		newMemoryAudit(),
		APILimits{
			MaxRequestBytes: 64 << 10, MaxListLimit: 10,
			IntakeMaxQuestionRounds: 6,
		},
		exactTestExecutionAuthority(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	result := invoke(
		t,
		dispatcher,
		"orquesta.intakes.wizard.gaps.apply",
		"request:wizard-gaps-missing-outcome",
		canonicalWizardGapsPayload(),
		false,
	)
	if result.Failure == nil || result.Failure.Code != CodeInternal {
		t.Fatalf("result=%+v", result)
	}
}

func TestWizardGapsCommandRejectsUnknownPackAndSpoofedAuthorityBeforeUseCase(
	t *testing.T,
) {
	api := &captureWizardGapsApplication{
		fakeApplication: newFakeApplication(),
	}
	dispatcher, err := newDispatcher(
		api,
		newMemoryAudit(),
		APILimits{
			MaxRequestBytes: 64 << 10, MaxListLimit: 10,
			IntakeMaxQuestionRounds: 6,
		},
		exactTestExecutionAuthority(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	unknownPack := canonicalWizardGapsPayload()
	unknownPack["pack_refs"] = []any{"not-a-pack"}
	result := invoke(
		t,
		dispatcher,
		"orquesta.intakes.wizard.gaps.apply",
		"request:wizard-gaps-unknown-pack",
		unknownPack,
		false,
	)
	if result.Failure == nil || result.Failure.Code != CodeInvalidRequest {
		t.Fatalf("unknown pack=%+v", result)
	}
	spoofed := canonicalWizardGapsPayload()
	spoofed["actor_ref"] = "actor:spoofed"
	result = invoke(
		t,
		dispatcher,
		"orquesta.intakes.wizard.gaps.apply",
		"request:wizard-gaps-spoofed",
		spoofed,
		false,
	)
	if result.Failure == nil || result.Failure.Code != CodeInvalidRequest ||
		len(api.requests) != 0 {
		t.Fatalf("spoofed=%+v requests=%d", result, len(api.requests))
	}
}

func TestWizardGapsCommandHasCanonicalHTTPMCPAndCLIBindings(t *testing.T) {
	var found *Definition
	for _, definition := range CanonicalDefinitions() {
		if definition.ID == "orquesta.intakes.wizard.gaps.apply" {
			candidate := definition
			found = &candidate
			break
		}
	}
	if found == nil ||
		found.Handler != "ApplyWizardGaps" ||
		found.Permission != "goals.create" ||
		found.HTTP.Method != "POST" ||
		found.HTTP.Path !=
			"/api/v1/commands/orquesta.intakes.wizard.gaps.apply" ||
		found.MCP.Tool != "orquesta.intakes.wizard.gaps.apply" ||
		!reflect.DeepEqual(
			found.CLI.Path,
			[]string{"intakes", "wizard", "gaps", "apply"},
		) {
		t.Fatalf("definition=%+v", found)
	}
}

func canonicalWizardGapsPayload() map[string]any {
	return map[string]any{
		"intake_ref":        "intake:test",
		"expected_revision": 1,
		"origin":            "form",
		"facts": map[string]any{
			"surface":                 "server_service",
			"sharing_intent":          "shared",
			"corporate_identity":      "declared",
			"target_users":            "declared",
			"integration_auth":        "declared",
			"integration_criticality": "declared",
		},
		"pack_refs": []any{},
	}
}
