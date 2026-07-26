package commands

import (
	"context"
	"encoding/json"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/wizard/stages"
)

func (api *fakeApplication) PrepareWizardDossier(
	_ context.Context,
	_ application.Access,
	request application.PrepareWizardDossierRequest,
) (application.WizardDossierResult, error) {
	api.called("PrepareWizardDossier")
	catalog, err := stages.BuiltInVersion(request.CatalogVersion)
	if err != nil {
		return application.WizardDossierResult{}, err
	}
	template, found := catalog.Template(request.TemplateRef)
	if !found {
		return application.WizardDossierResult{},
			application.ErrWizardStagePlanInvalid
	}
	compilation, err := application.CompileWizardStagePlan(
		template,
		request.StagePlanInput,
	)
	if err != nil {
		return application.WizardDossierResult{}, err
	}
	record, err := commandDossierRecord(
		application.PrepareIntakeDossierRequest{
			RequestRef: request.RequestRef,
			ActorRef:   request.ActorRef, ProjectRef: request.ProjectRef,
			StateRef:               request.StateRef,
			ExpectedRevision:       request.ExpectedRevision,
			SourceIntakeReceiptRef: request.SourceIntakeReceiptRef,
			Plan:                   compilation.Plan, Input: commandDossierInput(),
		},
	)
	if err != nil {
		return application.WizardDossierResult{}, err
	}
	return application.WizardDossierResult{
		Record: record, CatalogVersion: request.CatalogVersion,
		CatalogDigest:       request.CatalogDigest,
		TemplateDigest:      request.TemplateDigest,
		StagePlanProjection: compilation.Projection,
		Created:             true,
	}, nil
}

type captureWizardDossierApplication struct {
	*fakeApplication
	requests []application.PrepareWizardDossierRequest
}

func (api *captureWizardDossierApplication) PrepareWizardDossier(
	ctx context.Context,
	access application.Access,
	request application.PrepareWizardDossierRequest,
) (application.WizardDossierResult, error) {
	api.requests = append(api.requests, request)
	return api.fakeApplication.PrepareWizardDossier(ctx, access, request)
}

func TestWizardDossierCommandResolvesIdentityBindsAuthorityAndExposesSafePreview(
	t *testing.T,
) {
	api := &captureWizardDossierApplication{
		fakeApplication: newFakeApplication(),
	}
	dispatcher, err := newDispatcher(
		api,
		newMemoryAudit(),
		APILimits{
			MaxRequestBytes: 1 << 20, MaxListLimit: 100,
			IntakeMaxQuestionRounds: 6,
		},
		exactTestExecutionAuthority(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	payload := canonicalWizardDossierPreparePayload()
	spoofed := cloneWizardPayload(t, payload)
	spoofed["actor_ref"] = "actor:spoof"
	if result := invoke(
		t, dispatcher, "orquesta.intakes.wizard.dossier.prepare",
		"request:wizard-dossier-spoof", spoofed, false,
	); result.Failure == nil || result.Failure.Code != CodeInvalidRequest ||
		len(api.requests) != 0 {
		t.Fatalf("spoof admitted=%+v requests=%d", result, len(api.requests))
	}

	result := invoke(
		t, dispatcher, "orquesta.intakes.wizard.dossier.prepare",
		"request:wizard-dossier", payload, false,
	)
	if result.Failure != nil || len(api.requests) != 1 {
		t.Fatalf("result=%+v requests=%d", result, len(api.requests))
	}
	request := api.requests[0]
	catalog := stages.BuiltIn()
	templateRef, _ := stages.NewTemplateRef("template:build_app")
	template, _ := catalog.Template(templateRef)
	if request.RequestRef != "request:wizard-dossier" ||
		request.ActorRef.String() != "actor:test" ||
		request.ProjectRef.String() != "project:test" ||
		request.CatalogVersion != catalog.Version() ||
		request.CatalogDigest != catalog.Digest() ||
		request.TemplateRef != templateRef ||
		request.TemplateDigest != template.Digest() ||
		len(request.ProjectionSpec.Plan.Phases) != 0 ||
		len(request.ProjectionSpec.Plan.WorkItems) != 0 {
		t.Fatalf("request authority/identity=%+v", request)
	}
	var view wizardDossierView
	if err := json.Unmarshal(result.Data, &view); err != nil {
		t.Fatal(err)
	}
	if view.Identity.CatalogDigest != catalog.Digest().String() ||
		view.Identity.TemplateDigest != template.Digest().String() ||
		view.Preview.EffectsAuthorized || view.Preview.EffectsExecuted ||
		len(view.Preview.Stages) == 0 || len(view.Preview.Units) == 0 {
		t.Fatalf("view=%+v", view)
	}
	for _, unit := range view.Preview.Units {
		if unit.DependsOn == nil || unit.PlanDependencies == nil ||
			unit.WriteSet == nil || unit.RequiredTests == nil ||
			unit.AcceptanceCriteria == nil || unit.Effects == nil {
			t.Fatalf("nullable preview unit=%+v", unit)
		}
	}
}

func TestWizardDossierCommandRejectsCallerPlanAndDigestSpoofBeforeAdmission(
	t *testing.T,
) {
	dispatcher, api, audit := testDispatcher(t)
	cases := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"plan", func(payload map[string]any) {
			payload["plan"] = map[string]any{}
		}},
		{"projection_plan", func(payload map[string]any) {
			payload["projection_spec"].(map[string]any)["plan"] =
				map[string]any{}
		}},
		{"catalog_digest", func(payload map[string]any) {
			payload["catalog_digest"] = "spoof"
		}},
		{"template_digest", func(payload map[string]any) {
			payload["template_digest"] = "spoof"
		}},
	}
	for _, test := range cases {
		payload := canonicalWizardDossierPreparePayload()
		test.mutate(payload)
		result := invoke(
			t, dispatcher, "orquesta.intakes.wizard.dossier.prepare",
			"request:wizard-dossier-forbidden:"+test.name,
			payload,
			false,
		)
		if result.Failure == nil ||
			result.Failure.Code != CodeInvalidRequest {
			t.Errorf("%s result=%+v", test.name, result)
		}
	}
	if api.calls["PrepareWizardDossier"] != 0 || audit.admits != 0 {
		t.Fatalf(
			"forbidden input reached authority calls=%d admits=%d",
			api.calls["PrepareWizardDossier"],
			audit.admits,
		)
	}
}

func TestWizardDossierCommandRejectsUnknownCatalogAndTemplateWithoutUseCase(
	t *testing.T,
) {
	dispatcher, api, _ := testDispatcher(t)
	cases := []struct {
		name  string
		field string
		value string
	}{
		{
			name: "unknown catalog", field: "catalog_version",
			value: "orquesta.wizard.stages.unknown",
		},
		{
			name: "unknown template", field: "template_ref",
			value: "template:unknown",
		},
	}
	for _, test := range cases {
		payload := canonicalWizardDossierPreparePayload()
		payload[test.field] = test.value
		result := invoke(
			t, dispatcher, "orquesta.intakes.wizard.dossier.prepare",
			"request:wizard-dossier-identity:"+test.field,
			payload, false,
		)
		if result.Failure == nil ||
			result.Failure.Code != CodeInvalidRequest {
			t.Errorf("%s result=%+v", test.name, result)
		}
	}
	if api.calls["PrepareWizardDossier"] != 0 {
		t.Fatalf(
			"unknown identity reached application calls=%d",
			api.calls["PrepareWizardDossier"],
		)
	}
}

func canonicalWizardDossierPreparePayload() map[string]any {
	item := func(key string) map[string]any {
		return map[string]any{"key": key, "summary": "Resumen " + key}
	}
	items := func(key string) []any { return []any{item(key)} }
	resource := func(multiplier int64) map[string]any {
		return map[string]any{
			"tokens": 100 * multiplier, "money_micros": 0,
			"currency": "", "active_time_ns": 1000 * multiplier,
			"process_slots": 1, "disk_bytes": 1024 * multiplier,
		}
	}
	testBinding := func(kind string) map[string]any {
		return map[string]any{
			"tool_ref":  "tool:test-" + kind,
			"arguments": []any{"run", kind}, "working_directory": ".",
		}
	}
	return map[string]any{
		"intake_ref":                "intake:command-dossier",
		"expected_revision":         2,
		"source_intake_receipt_ref": "intake-receipt:command-dossier",
		"catalog_version":           "orquesta.wizard.stages.v1",
		"template_ref":              "template:build_app",
		"stage_plan_input": map[string]any{
			"objective":       "Deliver a verified application",
			"input_refs":      []any{"input:requirements"},
			"skill_refs":      []any{"skill:implementation"},
			"tool_refs":       []any{"tool:test-unit"},
			"capability_refs": []any{"capability:wizard-dossier"},
			"tests": map[string]any{
				"unit":          testBinding("unit"),
				"contract":      testBinding("contract"),
				"integration":   testBinding("integration"),
				"security":      testBinding("security"),
				"review":        testBinding("review"),
				"postcondition": testBinding("postcondition"),
			},
			"budgets": map[string]any{
				"routine": resource(1), "focused": resource(2),
				"deep": resource(3),
			},
		},
		"projection_spec": map[string]any{
			"statement": "Build a durable application",
			"objective": "Deliver a verified application",
			"product_scope": map[string]any{
				"in_scope":     items("agenda"),
				"out_of_scope": items("billing"),
			},
			"users_roles": map[string]any{
				"users": items("member"),
				"roles": []any{map[string]any{
					"key": "owner", "summary": "Owner",
					"permission_summaries": items("manage"),
				}},
				"main_flow": []any{
					map[string]any{
						"order": 1, "key": "open", "summary": "Open",
					},
					map[string]any{
						"order": 2, "key": "save", "summary": "Save",
					},
				},
			},
			"architecture": map[string]any{
				"rationale": "Hexagonal", "domain": "Domain",
				"application": "Application",
				"ports":       items("state"), "adapters": items("sqlite"),
				"composition_root": "cmd/orquesta",
				"core_limits":      items("no_transport"),
			},
			"data": map[string]any{
				"entities": items("event"), "lifecycles": items("event"),
				"import_export": items("json"), "backups": items("snapshot"),
				"versioning": []any{},
			},
			"integrations": map[string]any{
				"connectors": []any{map[string]any{
					"key": "calendar", "summary": "Calendar",
					"authentication": "OAuth", "data_flow": "Events",
				}},
			},
			"security_privacy": map[string]any{
				"data_classes":   items("personal"),
				"authentication": items("oidc"),
				"authorization":  items("rbac"), "audit": items("events"),
				"retention":            items("policy"),
				"privacy_requirements": items("minimize"),
			},
			"ui_ux": map[string]any{
				"views": items("calendar"), "navigation": items("main"),
				"empty_states": items("none"), "error_states": items("retry"),
				"accessibility": items("wcag"), "delivery": items("web"),
			},
			"i18n_l10n": map[string]any{
				"locales": []any{map[string]any{
					"tag": "es-ES", "summary": "Español",
				}},
				"fallback_locale": "es-ES", "catalog": "orquesta",
				"formats":           items("dates"),
				"visible_surfaces":  items("web"),
				"visible_text_rule": "Catalog keys only",
			},
			"deploy_operations": map[string]any{
				"environments":     items("production"),
				"deployment_units": items("binary"),
				"network":          items("https"),
				"observability":    items("metrics"), "backups": items("daily"),
				"updates": items("atomic"), "rollback": items("previous"),
			},
			"decisions": []any{map[string]any{
				"question_ref":   "intake-question:scope",
				"choice_summary": "CLI", "recommendation_summary": "Web",
				"deviation_justification": "",
			}},
			"risks": []any{map[string]any{
				"ref":     "intake-risk:calendar-provider",
				"summary": "Provider outage", "mitigation": "Retry",
				"status_key": "intake.risk.open",
			}}, "open_issues": []any{},
			"deferred_decisions":        []any{},
			"include_decisions_diagram": true,
		},
	}
}

func cloneWizardPayload(
	t *testing.T,
	source map[string]any,
) map[string]any {
	t.Helper()
	encoded, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(encoded, &result); err != nil {
		t.Fatal(err)
	}
	return result
}
