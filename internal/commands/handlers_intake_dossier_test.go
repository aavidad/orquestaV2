package commands

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/intake"
)

type commandDossierIntakeStore struct {
	current application.IntakeRecord
}

func (store *commandDossierIntakeStore) ReplayIntake(
	context.Context,
	application.IntakeReplayRequest,
) (application.IntakeRecord, bool, error) {
	return application.IntakeRecord{}, false, nil
}

func (store *commandDossierIntakeStore) CreateIntake(
	_ context.Context,
	state application.IntakeCreateState,
) (application.IntakeRecord, bool, error) {
	store.current = application.IntakeRecord{
		ActorRef: state.ActorRef, ProjectRef: state.ProjectRef,
		State: state.State, Receipt: state.Receipt,
	}
	return store.current, true, nil
}

func (store *commandDossierIntakeStore) GetIntake(
	context.Context,
	goal.ActorRef,
	goal.ProjectRef,
	intake.Ref,
) (application.IntakeRecord, error) {
	return store.current, nil
}

func (store *commandDossierIntakeStore) ApplyIntake(
	_ context.Context,
	state application.IntakeApplyState,
) (application.IntakeRecord, bool, error) {
	store.current = application.IntakeRecord{
		ActorRef: state.ActorRef, ProjectRef: state.ProjectRef,
		State: state.State, Receipt: state.Receipt,
	}
	return store.current, true, nil
}

func (api *fakeApplication) PrepareIntakeDossier(
	_ context.Context,
	_ application.Access,
	request application.PrepareIntakeDossierRequest,
) (application.IntakeDossierResult, error) {
	api.called("PrepareIntakeDossier")
	record, err := commandDossierRecord(request)
	return application.IntakeDossierResult{Record: record, Created: true}, err
}

func (api *fakeApplication) GetIntakeDossier(
	_ context.Context,
	_ application.Access,
	request application.GetIntakeDossierRequest,
) (application.IntakeDossierRecord, error) {
	api.called("GetIntakeDossier")
	plan, input := commandDossierPlan(), commandDossierInput()
	return commandDossierRecord(application.PrepareIntakeDossierRequest{
		RequestRef: "request:command-dossier-get",
		ActorRef:   request.ActorRef, ProjectRef: request.ProjectRef,
		StateRef: "intake:command-dossier", ExpectedRevision: 2,
		SourceIntakeReceiptRef: "intake-receipt:command-dossier",
		Plan:                   plan, Input: input,
	})
}

func (api *fakeApplication) ConfirmIntakeDossier(
	_ context.Context,
	_ application.Access,
	request application.ConfirmIntakeDossierRequest,
) (application.ConfirmIntakeDossierResult, error) {
	api.called("ConfirmIntakeDossier")
	if !request.Confirm {
		return application.ConfirmIntakeDossierResult{},
			errors.New("application.confirmation_required")
	}
	principalRef, _ := identity.NewPrincipalRef("principal:test")
	actorRef, _ := goal.NewActorRef("actor:test")
	projectRef, _ := goal.NewProjectRef("project:test")
	goalRef, _ := goal.NewGoalRef("goal:command-dossier-confirmed")
	appSpecRef, _ := goal.NewAppSpecRef("app-spec:command-dossier-confirmed")
	return application.ConfirmIntakeDossierResult{
		Confirmation: application.IntakeDossierConfirmation{
			Ref: "intake-dossier-confirmation-receipt:" +
				strings.Repeat("a", 64),
			RequestRef:             request.RequestRef,
			RequestFingerprint:     strings.Repeat("b", 64),
			PrincipalRef:           principalRef,
			ActorRef:               actorRef,
			ProjectRef:             projectRef,
			StateRef:               "intake:command-dossier",
			StateRevision:          2,
			StateDigest:            strings.Repeat("c", 64),
			SourceIntakeReceiptRef: "intake-receipt:command-dossier",
			DossierRef:             request.DossierRef,
			DossierDigest:          strings.Repeat("d", 64),
			PlanDigest:             strings.Repeat("e", 64),
			GoalRef:                goalRef,
			AppSpecRef:             appSpecRef,
			SpecHash:               strings.Repeat("f", 64),
			AuthorizationReceiptRef: "authorization-receipt:" +
				request.RequestRef,
			ConfirmedAt: time.Date(2026, 7, 26, 18, 0, 0, 0, time.UTC),
		},
		Created: true,
	}, nil
}

type captureDossierApplication struct {
	*fakeApplication
	prepares []application.PrepareIntakeDossierRequest
	gets     []application.GetIntakeDossierRequest
	confirms []application.ConfirmIntakeDossierRequest
}

func (api *captureDossierApplication) PrepareIntakeDossier(
	ctx context.Context,
	access application.Access,
	request application.PrepareIntakeDossierRequest,
) (application.IntakeDossierResult, error) {
	api.prepares = append(api.prepares, request)
	return api.fakeApplication.PrepareIntakeDossier(ctx, access, request)
}

func (api *captureDossierApplication) GetIntakeDossier(
	ctx context.Context,
	access application.Access,
	request application.GetIntakeDossierRequest,
) (application.IntakeDossierRecord, error) {
	api.gets = append(api.gets, request)
	return api.fakeApplication.GetIntakeDossier(ctx, access, request)
}

func (api *captureDossierApplication) ConfirmIntakeDossier(
	ctx context.Context,
	access application.Access,
	request application.ConfirmIntakeDossierRequest,
) (application.ConfirmIntakeDossierResult, error) {
	api.confirms = append(api.confirms, request)
	return api.fakeApplication.ConfirmIntakeDossier(ctx, access, request)
}

func TestIntakeDossierCommandsBindAuthorityRejectSpoofAndProjectCompletePlan(t *testing.T) {
	api := &captureDossierApplication{fakeApplication: newFakeApplication()}
	audit := newMemoryAudit()
	dispatcher, err := newDispatcher(
		api, audit, APILimits{MaxRequestBytes: 1 << 20, MaxListLimit: 100},
		exactTestExecutionAuthority(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	prepare := invoke(
		t, dispatcher, "orquesta.intakes.dossier.prepare",
		"request:dossier-prepare", canonicalIntakeDossierPreparePayload(), false,
	)
	if prepare.Failure != nil {
		t.Fatalf("prepare=%+v", prepare)
	}
	get := invoke(
		t, dispatcher, "orquesta.intakes.dossier.get",
		"request:dossier-get", map[string]any{"dossier_ref": "intake-dossier:public"}, false,
	)
	if get.Failure != nil {
		t.Fatalf("get=%+v", get)
	}
	if len(api.prepares) != 1 || len(api.gets) != 1 {
		t.Fatalf("requests=%d/%d", len(api.prepares), len(api.gets))
	}
	prepared := api.prepares[0]
	if prepared.RequestRef != "request:dossier-prepare" ||
		prepared.ActorRef.String() != "actor:test" ||
		prepared.ProjectRef.String() != "project:test" ||
		prepared.AuthorizationReceipt.Ref() != "" ||
		api.gets[0].ActorRef.String() != "actor:test" ||
		api.gets[0].ProjectRef.String() != "project:test" ||
		api.gets[0].DossierRef != "intake-dossier:public" {
		t.Fatalf("authority prepare=%+v get=%+v", prepared, api.gets[0])
	}
	var output struct {
		Dossier struct {
			Schema            string            `json:"schema"`
			DossierRef        string            `json:"dossier_ref"`
			IntakeRef         string            `json:"intake_ref"`
			ActorRef          string            `json:"actor_ref"`
			ProjectRef        string            `json:"project_ref"`
			Sections          []json.RawMessage `json:"sections"`
			Diagrams          []json.RawMessage `json:"diagrams"`
			Decisions         []json.RawMessage `json:"decisions"`
			RiskRefs          []string          `json:"risk_refs"`
			Plan              planInput         `json:"plan"`
			GenerationReceipt struct {
				RequestRef string `json:"request_ref"`
			} `json:"generation_receipt"`
		} `json:"dossier"`
	}
	if err := json.Unmarshal(prepare.Data, &output); err != nil {
		t.Fatal(err)
	}
	if output.Dossier.Schema != application.IntakeDossierSchema ||
		output.Dossier.DossierRef == "" ||
		output.Dossier.IntakeRef != "intake:command-dossier" ||
		output.Dossier.ActorRef != "actor:test" ||
		output.Dossier.ProjectRef != "project:test" ||
		len(output.Dossier.Sections) != 11 || len(output.Dossier.Diagrams) != 5 ||
		len(output.Dossier.Decisions) != 1 || output.Dossier.RiskRefs == nil ||
		len(output.Dossier.Plan.Phases) != 1 || len(output.Dossier.Plan.WorkItems) != 1 ||
		output.Dossier.Plan.WorkItems[0].OutputContract != "evidence_bundle" ||
		output.Dossier.GenerationReceipt.RequestRef != "request:dossier-prepare" ||
		strings.Contains(string(prepare.Data), `"Created"`) ||
		strings.Contains(string(prepare.Data), `"Phases"`) {
		t.Fatalf("projection=%s", prepare.Data)
	}

	for _, field := range []string{
		"actor_ref", "project_ref", "request_ref", "request_fingerprint",
		"authorization_receipt_ref",
	} {
		payload := canonicalIntakeDossierPreparePayload()
		payload[field] = "spoof"
		result := invoke(
			t, dispatcher, "orquesta.intakes.dossier.prepare",
			"request:dossier-spoof:"+field, payload, false,
		)
		if result.Failure == nil || result.Failure.Code != CodeInvalidRequest {
			t.Errorf("%s=%+v", field, result)
		}
	}
	getSpoof := invoke(
		t, dispatcher, "orquesta.intakes.dossier.get", "request:dossier-get-spoof",
		map[string]any{
			"dossier_ref": "intake-dossier:public",
			"actor_ref":   "actor:spoof",
			"project_ref": "project:spoof",
			"request_ref": "request:spoof",
		},
		false,
	)
	if getSpoof.Failure == nil || getSpoof.Failure.Code != CodeInvalidRequest ||
		len(api.prepares) != 1 || len(api.gets) != 1 || audit.admits != 2 {
		t.Fatalf(
			"spoof reached application: result=%+v prepares=%d gets=%d admits=%d",
			getSpoof, len(api.prepares), len(api.gets), audit.admits,
		)
	}
	invalidPlan := canonicalIntakeDossierPreparePayload()
	plan := invalidPlan["plan"].(planInput)
	plan.WorkItems[0].OutputContract = "unknown"
	invalidPlan["plan"] = plan
	invalidOutputContract := invoke(
		t, dispatcher, "orquesta.intakes.dossier.prepare",
		"request:dossier-invalid-output-contract", invalidPlan, false,
	)
	if invalidOutputContract.Failure == nil ||
		invalidOutputContract.Failure.Code != CodeInvalidRequest ||
		len(api.prepares) != 1 || audit.admits != 2 {
		t.Fatalf(
			"invalid output contract reached writer: result=%+v prepares=%d admits=%d",
			invalidOutputContract, len(api.prepares), audit.admits,
		)
	}
}

func TestIntakeDossierInvalidInputKeepsPublicInvalidRequestClass(t *testing.T) {
	api := &invalidDossierApplication{fakeApplication: newFakeApplication()}
	dispatcher, err := newDispatcher(
		api, newMemoryAudit(), APILimits{MaxRequestBytes: 1 << 20, MaxListLimit: 100},
		exactTestExecutionAuthority(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	result := invoke(
		t, dispatcher, "orquesta.intakes.dossier.prepare",
		"request:dossier-invalid", canonicalIntakeDossierPreparePayload(), false,
	)
	if result.Failure == nil || result.Failure.Code != CodeInvalidRequest {
		t.Fatalf("result=%+v", result)
	}
}

func TestConfirmIntakeDossierCommandBindsEnvelopeAndExposesExactReceipt(t *testing.T) {
	api := &captureDossierApplication{fakeApplication: newFakeApplication()}
	audit := newMemoryAudit()
	dispatcher, err := newDispatcher(
		api, audit, APILimits{MaxRequestBytes: 1 << 20, MaxListLimit: 100},
		exactTestExecutionAuthority(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	const requestRef = "request:dossier-confirm-public"
	const dossierRef = "intake-dossier:public-confirmation"
	result := invoke(
		t, dispatcher, "orquesta.intakes.dossier.confirm", requestRef,
		map[string]any{"dossier_ref": dossierRef, "confirm": true}, false,
	)
	if result.Failure != nil {
		t.Fatalf("confirm=%+v", result)
	}
	if len(api.confirms) != 1 ||
		api.confirms[0] != (application.ConfirmIntakeDossierRequest{
			RequestRef: requestRef,
			DossierRef: dossierRef,
			Confirm:    true,
		}) {
		t.Fatalf("confirmation request=%+v", api.confirms)
	}
	var output struct {
		Goal         goalReceiptView               `json:"goal"`
		Confirmation intakeDossierConfirmationView `json:"confirmation"`
	}
	if err := json.Unmarshal(result.Data, &output); err != nil {
		t.Fatal(err)
	}
	if output.Confirmation.RequestRef != requestRef ||
		output.Confirmation.DossierRef != dossierRef ||
		output.Confirmation.ActorRef != "actor:test" ||
		output.Confirmation.ProjectRef != "project:test" ||
		output.Confirmation.IntakeRef != "intake:command-dossier" ||
		output.Confirmation.IntakeRevision != 2 ||
		output.Confirmation.GoalRef != "goal:command-dossier-confirmed" ||
		output.Confirmation.AppSpecRef != "app-spec:command-dossier-confirmed" ||
		output.Confirmation.ConfirmedAt.IsZero() ||
		strings.Contains(string(result.Data), `"Created"`) ||
		strings.Contains(string(result.Data), `"Record"`) {
		t.Fatalf("confirmation projection=%s", result.Data)
	}

	for _, field := range []string{
		"actor_ref", "project_ref", "request_ref", "request_fingerprint",
		"authorization_receipt_ref", "statement", "plan",
	} {
		payload := map[string]any{
			"dossier_ref": dossierRef,
			"confirm":     true,
			field:         "spoof",
		}
		spoofed := invoke(
			t, dispatcher, "orquesta.intakes.dossier.confirm",
			"request:dossier-confirm-spoof:"+field, payload, false,
		)
		if spoofed.Failure == nil ||
			spoofed.Failure.Code != CodeInvalidRequest {
			t.Errorf("%s=%+v", field, spoofed)
		}
	}
	if len(api.confirms) != 1 || audit.admits != 1 {
		t.Fatalf(
			"spoof reached writer: confirms=%d admits=%d",
			len(api.confirms), audit.admits,
		)
	}

	for _, payload := range []map[string]any{
		{"dossier_ref": dossierRef},
		{"confirm": true},
		{"dossier_ref": dossierRef, "confirm": "true"},
	} {
		invalid := invoke(
			t, dispatcher, "orquesta.intakes.dossier.confirm",
			"request:dossier-confirm-invalid", payload, false,
		)
		if invalid.Failure == nil ||
			invalid.Failure.Code != CodeInvalidRequest {
			t.Errorf("invalid payload=%v result=%+v", payload, invalid)
		}
	}
	if len(api.confirms) != 1 || audit.admits != 1 {
		t.Fatalf(
			"invalid shape reached writer: confirms=%d admits=%d",
			len(api.confirms), audit.admits,
		)
	}

	notConfirmed := invoke(
		t, dispatcher, "orquesta.intakes.dossier.confirm",
		"request:dossier-confirm-false",
		map[string]any{"dossier_ref": dossierRef, "confirm": false}, false,
	)
	if notConfirmed.Failure == nil ||
		notConfirmed.Failure.Code != CodeInvalidRequest ||
		len(api.confirms) != 2 ||
		api.confirms[1].Confirm {
		t.Fatalf("confirm=false=%+v requests=%+v", notConfirmed, api.confirms)
	}
}

type invalidDossierApplication struct{ *fakeApplication }

func (api *invalidDossierApplication) PrepareIntakeDossier(
	context.Context,
	application.Access,
	application.PrepareIntakeDossierRequest,
) (application.IntakeDossierResult, error) {
	api.called("PrepareIntakeDossier")
	return application.IntakeDossierResult{}, application.ErrIntakeDossierInvalid
}

func canonicalIntakeDossierPreparePayload() map[string]any {
	plan, input := commandDossierPlan(), commandDossierInput()
	projectedPlan := projectIntakeDossierPlan(plan)
	return map[string]any{
		"intake_ref": "intake:command-dossier", "expected_revision": 2,
		"source_intake_receipt_ref": "intake-receipt:command-dossier",
		"statement":                 input.Statement, "objective": input.Objective,
		"sections": input.Sections, "diagrams": input.Diagrams,
		"risk_refs": input.RiskRefs, "plan": projectedPlan,
	}
}

func commandDossierPlan() application.PlanSpec {
	return application.PlanSpec{
		Phases: []application.PhaseSpec{{
			Ref: "phase-instance:main", Key: "phase:main",
			TemplateRef: "phase-template:build",
		}},
		WorkItems: []application.WorkItemSpec{{
			Key: "work:build", Objective: "build application", Phase: "phase:main",
			Role: "role:builder", Dependencies: []string{}, WriteSet: []string{"src"},
			CouncilPolicy: council.PolicyRequired,
			RequiredTests: []application.RequiredTestSpec{{
				Ref: "required-test:build", ToolRef: "tool:go-test",
				Arguments: []string{"go", "test", "./..."}, WorkingDirectory: ".",
			}},
			OutputContract: "evidence_bundle",
		}},
	}
}

func commandDossierInput() application.IntakeDossierInput {
	sectionKinds := []application.IntakeDossierSectionKind{
		application.IntakeDossierSectionProductScope,
		application.IntakeDossierSectionUsersRoles,
		application.IntakeDossierSectionArchitecture,
		application.IntakeDossierSectionData,
		application.IntakeDossierSectionIntegrations,
		application.IntakeDossierSectionSecurityPrivacy,
		application.IntakeDossierSectionUIUX,
		application.IntakeDossierSectionI18NL10N,
		application.IntakeDossierSectionDeployOperations,
		application.IntakeDossierSectionOrchestrationPlan,
		application.IntakeDossierSectionRisksOpenIssues,
	}
	sections := make([]application.IntakeDossierSection, 0, len(sectionKinds))
	for _, kind := range sectionKinds {
		sections = append(sections, application.IntakeDossierSection{
			Ref:      application.IntakeDossierSectionRef("intake-dossier-section:" + string(kind)),
			Kind:     kind,
			TitleKey: intake.MessageKey("dossier.section." + string(kind) + ".title"),
			Markdown: "Complete " + string(kind),
		})
	}
	diagramPurposes := []application.IntakeDossierDiagramPurpose{
		application.IntakeDossierDiagramArchitecture,
		application.IntakeDossierDiagramUserFlow,
		application.IntakeDossierDiagramDataIntegrations,
		application.IntakeDossierDiagramI18N,
		application.IntakeDossierDiagramDeployment,
	}
	diagrams := make([]application.IntakeDossierDiagram, 0, len(diagramPurposes))
	for _, purpose := range diagramPurposes {
		diagrams = append(diagrams, application.IntakeDossierDiagram{
			Ref:        application.IntakeDossierDiagramRef("intake-dossier-diagram:" + string(purpose)),
			Purpose:    purpose,
			Kind:       application.IntakeDossierDiagramMermaid,
			Source:     "flowchart LR\nA --> B",
			AltTextKey: intake.MessageKey("dossier.diagram." + string(purpose) + ".alt"),
		})
	}
	return application.IntakeDossierInput{
		Statement: "Build a durable application",
		Objective: "Deliver a verified application",
		Sections:  sections, Diagrams: diagrams,
		RiskRefs: []application.IntakeRiskRef{},
	}
}

func commandDossierRecord(
	request application.PrepareIntakeDossierRequest,
) (application.IntakeDossierRecord, error) {
	store := &commandDossierIntakeStore{}
	service, err := application.NewIntakeService(store)
	if err != nil {
		return application.IntakeDossierRecord{}, err
	}
	createRef := "request:command-dossier-create"
	createAuth, err := commandDossierAuthorization(
		request.ActorRef, request.ProjectRef, createRef, application.IntakeOperationCreate,
	)
	if err != nil {
		return application.IntakeDossierRecord{}, err
	}
	if _, err = service.CreateIntake(context.Background(), application.CreateIntakeRequest{
		RequestRef: createRef, ActorRef: request.ActorRef, ProjectRef: request.ProjectRef,
		StateRef: "intake:command-dossier",
		Policy:   intake.Policy{MaxQuestionRounds: 2}, AuthorizationReceipt: createAuth,
	}); err != nil {
		return application.IntakeDossierRecord{}, err
	}
	applyRef := "request:command-dossier-apply"
	applyAuth, err := commandDossierAuthorization(
		request.ActorRef, request.ProjectRef, applyRef, application.IntakeOperationApply,
	)
	if err != nil {
		return application.IntakeDossierRecord{}, err
	}
	applied, err := service.ApplyIntake(context.Background(), application.ApplyIntakeRequest{
		RequestRef: applyRef, ActorRef: request.ActorRef, ProjectRef: request.ProjectRef,
		AuthorizationReceipt: applyAuth,
		Change: intake.Change{
			StateRef: "intake:command-dossier", ExpectedRevision: 1, Origin: intake.OriginChat,
			Issues: []intake.Issue{{
				Ref: "intake-issue:scope", Kind: intake.IssueGap,
				Field: "scope", DetailKey: "intake.issue.scope.missing",
			}},
			Questions: []intake.Question{{
				Ref: "intake-question:scope", DerivedFrom: []intake.IssueRef{"intake-issue:scope"},
				PromptKey: "intake.question.scope.prompt", WhyKey: "intake.question.scope.why",
				Options: []intake.Option{
					{
						Ref: "intake-option:scope-web", LabelKey: "intake.option.scope.web.label",
						RationaleKey: "intake.option.scope.web.rationale", Recommended: true,
					},
					{
						Ref: "intake-option:scope-cli", LabelKey: "intake.option.scope.cli.label",
						RationaleKey: "intake.option.scope.cli.rationale",
					},
				},
			}},
			Choices: []intake.Choice{{
				QuestionRef: "intake-question:scope", OptionRef: "intake-option:scope-cli",
			}},
		},
	})
	if err != nil {
		return application.IntakeDossierRecord{}, err
	}
	dossier, err := application.BuildIntakeDossier(applied.Record, request.Plan, request.Input)
	if err != nil {
		return application.IntakeDossierRecord{}, err
	}
	receipt, err := application.BuildIntakeDossierGenerationReceipt(
		request.RequestRef, strings.Repeat("a", 64), dossier,
		"authorization-receipt:command-dossier",
	)
	if err != nil {
		return application.IntakeDossierRecord{}, err
	}
	return application.IntakeDossierRecord{
		ActorRef: request.ActorRef, ProjectRef: request.ProjectRef,
		Dossier: dossier, Receipt: receipt,
	}, nil
}

func commandDossierAuthorization(
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	requestRef string,
	operation application.IntakeOperation,
) (identity.AuthorizationReceipt, error) {
	principalRef, err := identity.NewPrincipalRef("principal:command-dossier")
	if err != nil {
		return identity.AuthorizationReceipt{}, err
	}
	principal, err := identity.NewPrincipal(
		principalRef, actorRef, identity.PrincipalKindHuman, "test",
	)
	if err != nil {
		return identity.AuthorizationReceipt{}, err
	}
	authorizationRequestRef, err := application.IntakeAuthorizationRequestRef(operation, requestRef)
	if err != nil {
		return identity.AuthorizationReceipt{}, err
	}
	at := time.Date(2026, 7, 26, 16, 0, 0, 0, time.UTC)
	authorizationRequest, err := identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{
		RequestRef: authorizationRequestRef, Principal: principal,
		ProjectRef: projectRef, Permission: identity.PermissionGoalsCreate,
		ResourceRef: projectRef.String(), RequestedAt: at,
	})
	if err != nil {
		return identity.AuthorizationReceipt{}, err
	}
	decision, err := identity.NewAuthorizationDecision(identity.AuthorizationDecisionInput{
		Request: authorizationRequest, Outcome: identity.AuthorizationAllowed,
		Role: identity.RoleProjectOwner, MembershipRevision: 1,
		ReasonCode: "allowed", DecidedAt: at,
	})
	if err != nil {
		return identity.AuthorizationReceipt{}, err
	}
	return identity.NewAuthorizationReceipt(identity.AuthorizationReceiptInput{
		Ref:      "authorization-receipt:" + authorizationRequestRef,
		Decision: decision, RecordedAt: at,
	})
}
