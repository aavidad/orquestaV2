package bootstrap

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestV23WizardDossierPublicReplaySurvivesSQLiteRestart(t *testing.T) {
	root := t.TempDir()
	configPath := writeTestConfig(t, root)
	first := buildV23IntakeRuntime(t, configPath)
	principal, hierarchy, err := localIdentityComposition(first.config)
	if err != nil {
		t.Fatal(err)
	}
	projectRef := hierarchy.ProjectRef().String()

	dispatchV23IntakeCommand(
		t, first, principal, projectRef,
		"orquesta.intakes.create", "request:wizard-e2e-create",
		map[string]any{
			"intake_ref":          "intake:v23-dossier-e2e",
			"max_question_rounds": 2,
		},
	)
	dispatchV23IntakeCommand(
		t, first, principal, projectRef,
		"orquesta.intakes.apply", "request:wizard-e2e-question",
		v23DossierQuestionPayload(1),
	)
	answered := dispatchV23IntakeCommand(
		t, first, principal, projectRef,
		"orquesta.intakes.apply", "request:wizard-e2e-answer",
		v23DossierAnswerPayload(2),
	)
	answer := decodeV23IntakeMutation(t, answered)
	payload := v23WizardDossierPreparePayload(answer.ReceiptRef)
	prepared := dispatchV23IntakeCommand(
		t, first, principal, projectRef,
		"orquesta.intakes.wizard.dossier.prepare",
		"request:wizard-e2e-prepare",
		payload,
	)
	if prepared.Failure != nil || prepared.AuditRef == "" {
		t.Fatalf("prepared=%+v failure=%+v", prepared, prepared.Failure)
	}
	assertV23WizardDossierOutput(t, prepared.Data)
	replayed := dispatchV23IntakeCommand(
		t, first, principal, projectRef,
		"orquesta.intakes.wizard.dossier.prepare",
		"request:wizard-e2e-prepare",
		payload,
	)
	if replayed.Failure != nil || replayed.AuditRef != prepared.AuditRef ||
		!bytes.Equal(replayed.Data, prepared.Data) {
		t.Fatalf("replayed=%+v prepared=%+v", replayed, prepared)
	}
	shutdownRuntime(t, first)

	second := buildV23IntakeRuntime(t, configPath)
	t.Cleanup(func() { shutdownRuntime(t, second) })
	restarted := dispatchV23IntakeCommand(
		t, second, principal, projectRef,
		"orquesta.intakes.wizard.dossier.prepare",
		"request:wizard-e2e-prepare",
		payload,
	)
	if restarted.Failure != nil ||
		restarted.AuditRef != prepared.AuditRef ||
		!bytes.Equal(restarted.Data, prepared.Data) {
		t.Fatalf("restarted=%+v prepared=%+v", restarted, prepared)
	}
}

func assertV23WizardDossierOutput(t *testing.T, encoded json.RawMessage) {
	t.Helper()
	var output struct {
		Dossier struct {
			DossierRef string `json:"dossier_ref"`
			PlanDigest string `json:"plan_digest"`
		} `json:"dossier"`
		Identity struct {
			CatalogVersion string `json:"catalog_version"`
			CatalogDigest  string `json:"catalog_digest"`
			TemplateRef    string `json:"template_ref"`
			TemplateDigest string `json:"template_digest"`
		} `json:"identity"`
		Preview struct {
			TemplateRef           string   `json:"template_ref"`
			RoadmapCapabilityRefs []string `json:"roadmap_capability_refs"`
			Stages                []any    `json:"stages"`
			Units                 []any    `json:"units"`
			EffectsAuthorized     bool     `json:"effects_authorized"`
			EffectsExecuted       bool     `json:"effects_executed"`
		} `json:"preview"`
	}
	if err := json.Unmarshal(encoded, &output); err != nil {
		t.Fatal(err)
	}
	if output.Dossier.DossierRef == "" ||
		output.Dossier.PlanDigest == "" ||
		output.Identity.CatalogVersion != "orquesta.wizard.stages.v1" ||
		output.Identity.CatalogDigest == "" ||
		output.Identity.TemplateRef != "template:build_app" ||
		output.Identity.TemplateDigest == "" ||
		output.Preview.TemplateRef != output.Identity.TemplateRef ||
		output.Preview.RoadmapCapabilityRefs == nil ||
		len(output.Preview.Stages) == 0 ||
		len(output.Preview.Units) == 0 ||
		output.Preview.EffectsAuthorized ||
		output.Preview.EffectsExecuted {
		t.Fatalf("wizard output=%+v", output)
	}
}

func v23WizardDossierPreparePayload(
	sourceReceiptRef string,
) map[string]any {
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
		"intake_ref":                "intake:v23-dossier-e2e",
		"expected_revision":         3,
		"source_intake_receipt_ref": sourceReceiptRef,
		"catalog_version":           "orquesta.wizard.stages.v1",
		"template_ref":              "template:build_app",
		"stage_plan_input": map[string]any{
			"objective":       "Crear una aplicación de agenda",
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
			"statement": "Quiero una agenda compartida",
			"objective": "Crear una aplicación de agenda",
			"product_scope": map[string]any{
				"in_scope":     items("agenda"),
				"out_of_scope": items("billing"),
			},
			"users_roles": map[string]any{
				"users": items("team"),
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
				"question_ref":            "intake-question:audience",
				"choice_summary":          "Team",
				"recommendation_summary":  "Team",
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
