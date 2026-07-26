package bootstrap

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	commandcore "orquesta/internal/commands"
	"orquesta/internal/identity"
)

func TestV23DossierCommandsPersistReplayAndCanonicalReadAcrossRestart(t *testing.T) {
	root := t.TempDir()
	configPath := writeTestConfig(t, root)
	first := buildV23IntakeRuntime(t, configPath)
	principal, hierarchy, err := localIdentityComposition(first.config)
	if err != nil {
		t.Fatal(err)
	}
	projectRef := hierarchy.ProjectRef().String()

	created := dispatchV23IntakeCommand(
		t, first, principal, projectRef,
		"orquesta.intakes.create", "request:v23-dossier-intake-create",
		map[string]any{"intake_ref": "intake:v23-dossier-e2e", "max_question_rounds": 2},
	)
	if result := decodeV23IntakeMutation(t, created); result.Revision != 1 {
		t.Fatalf("created intake=%+v", result)
	}
	question := dispatchV23IntakeCommand(
		t, first, principal, projectRef,
		"orquesta.intakes.apply", "request:v23-dossier-intake-question",
		v23DossierQuestionPayload(1),
	)
	if result := decodeV23IntakeMutation(t, question); result.Revision != 2 {
		t.Fatalf("question intake=%+v", result)
	}
	answered := dispatchV23IntakeCommand(
		t, first, principal, projectRef,
		"orquesta.intakes.apply", "request:v23-dossier-intake-answer",
		v23DossierAnswerPayload(2),
	)
	answerView := decodeV23IntakeMutation(t, answered)
	if answerView.Revision != 3 {
		t.Fatalf("answered intake=%+v", answerView)
	}

	preparePayload := v23DossierPreparePayload(answerView.ReceiptRef)
	spoofedPayload := cloneV23Map(t, preparePayload)
	spoofedPayload["actor_ref"] = "actor:spoofed"
	spoofed := dispatchV23IntakeCommand(
		t, first, principal, projectRef,
		"orquesta.intakes.dossier.prepare", "request:v23-dossier-spoof",
		spoofedPayload,
	)
	if spoofed.Failure == nil || spoofed.Failure.Code != commandcore.CodeInvalidRequest ||
		spoofed.AuditRef != "" || len(spoofed.Data) != 0 {
		t.Fatalf("dossier payload authority was admitted: %+v", spoofed)
	}

	prepared := dispatchV23IntakeCommand(
		t, first, principal, projectRef,
		"orquesta.intakes.dossier.prepare", "request:v23-dossier-prepare",
		preparePayload,
	)
	preparedView := decodeV23Dossier(t, prepared)
	assertV23PreparedDossier(
		t, preparedView, principal.ActorRef.String(), projectRef,
		"request:v23-dossier-prepare", answerView.ReceiptRef,
	)
	replayed := dispatchV23IntakeCommand(
		t, first, principal, projectRef,
		"orquesta.intakes.dossier.prepare", "request:v23-dossier-prepare",
		preparePayload,
	)
	if replayed.Failure != nil || replayed.AuditRef != prepared.AuditRef ||
		!bytes.Equal(replayed.Data, prepared.Data) {
		t.Fatalf("dossier replay=%+v prepared=%+v", replayed, prepared)
	}

	divergentPayload := cloneV23Map(t, preparePayload)
	divergentPayload["objective"] = "Objetivo divergente"
	divergent := dispatchV23IntakeCommand(
		t, first, principal, projectRef,
		"orquesta.intakes.dossier.prepare", "request:v23-dossier-prepare",
		divergentPayload,
	)
	if divergent.Failure == nil || divergent.Failure.Code != commandcore.CodeConflict ||
		len(divergent.Data) != 0 {
		t.Fatalf("divergent dossier replay=%+v", divergent)
	}

	canonical := getV23Dossier(
		t, first, principal, projectRef, "request:v23-dossier-get-first",
		preparedView.DossierRef,
	)
	if !equalV23DossierIdentity(canonical, preparedView) ||
		canonical.GenerationReceipt.RequestRef != "request:v23-dossier-prepare" {
		t.Fatalf("canonical dossier=%+v prepared=%+v", canonical, preparedView)
	}
	shutdownRuntime(t, first)

	second := buildV23IntakeRuntime(t, configPath)
	t.Cleanup(func() { shutdownRuntime(t, second) })
	restartedPrincipal, restartedHierarchy, err := localIdentityComposition(second.config)
	if err != nil {
		t.Fatal(err)
	}
	restartedProjectRef := restartedHierarchy.ProjectRef().String()
	if restartedPrincipal != principal || restartedProjectRef != projectRef {
		t.Fatalf("restart authority changed: principal=%+v project=%s",
			restartedPrincipal, restartedProjectRef)
	}
	restartedReplay := dispatchV23IntakeCommand(
		t, second, principal, projectRef,
		"orquesta.intakes.dossier.prepare", "request:v23-dossier-prepare",
		preparePayload,
	)
	if restartedReplay.Failure != nil ||
		restartedReplay.AuditRef != prepared.AuditRef ||
		!bytes.Equal(restartedReplay.Data, prepared.Data) {
		t.Fatalf("restarted replay=%+v prepared=%+v", restartedReplay, prepared)
	}
	secondGeneration := dispatchV23IntakeCommand(
		t, second, principal, projectRef,
		"orquesta.intakes.dossier.prepare", "request:v23-dossier-prepare-second",
		preparePayload,
	)
	secondView := decodeV23Dossier(t, secondGeneration)
	if !equalV23DossierIdentity(secondView, preparedView) ||
		secondView.GenerationReceipt.RequestRef != "request:v23-dossier-prepare-second" ||
		secondView.GenerationReceipt.ReceiptRef == preparedView.GenerationReceipt.ReceiptRef {
		t.Fatalf("second generation=%+v first=%+v", secondView, preparedView)
	}
	restartedCanonical := getV23Dossier(
		t, second, principal, projectRef, "request:v23-dossier-get-restart",
		preparedView.DossierRef,
	)
	if !equalV23DossierIdentity(restartedCanonical, preparedView) ||
		restartedCanonical.GenerationReceipt.RequestRef != "request:v23-dossier-prepare" ||
		restartedCanonical.GenerationReceipt.ReceiptRef != preparedView.GenerationReceipt.ReceiptRef {
		t.Fatalf("restarted canonical=%+v first=%+v", restartedCanonical, preparedView)
	}

	confirmSpoofed := dispatchV23IntakeCommand(
		t, second, principal, projectRef,
		"orquesta.intakes.dossier.confirm", "request:v23-dossier-confirm-spoof",
		map[string]any{
			"dossier_ref": preparedView.DossierRef,
			"confirm":     true,
			"statement":   "contenido sustituido",
		},
	)
	if confirmSpoofed.Failure == nil ||
		confirmSpoofed.Failure.Code != commandcore.CodeInvalidRequest ||
		confirmSpoofed.AuditRef != "" || len(confirmSpoofed.Data) != 0 {
		t.Fatalf("confirmation payload substitution was admitted: %+v", confirmSpoofed)
	}
	notConfirmed := dispatchV23IntakeCommand(
		t, second, principal, projectRef,
		"orquesta.intakes.dossier.confirm", "request:v23-dossier-confirm-false",
		map[string]any{
			"dossier_ref": preparedView.DossierRef,
			"confirm":     false,
		},
	)
	if notConfirmed.Failure == nil ||
		notConfirmed.Failure.Code != commandcore.CodeInvalidRequest ||
		len(notConfirmed.Data) != 0 {
		t.Fatalf("confirm=false=%+v", notConfirmed)
	}

	confirmation := dispatchV23IntakeCommand(
		t, second, principal, projectRef,
		"orquesta.intakes.dossier.confirm", "request:v23-dossier-confirm",
		map[string]any{
			"dossier_ref": preparedView.DossierRef,
			"confirm":     true,
		},
	)
	confirmationView := decodeV23DossierConfirmation(t, confirmation)
	if confirmationView.Confirmation.RequestRef != "request:v23-dossier-confirm" ||
		confirmationView.Confirmation.ActorRef != principal.ActorRef.String() ||
		confirmationView.Confirmation.ProjectRef != projectRef ||
		confirmationView.Confirmation.DossierRef != preparedView.DossierRef ||
		confirmationView.Confirmation.IntakeRef != preparedView.IntakeRef ||
		confirmationView.Confirmation.IntakeRevision != preparedView.IntakeRevision ||
		confirmationView.Confirmation.IntakeDigest != preparedView.IntakeDigest ||
		confirmationView.Confirmation.SourceIntakeReceiptRef !=
			preparedView.SourceIntakeReceiptRef ||
		confirmationView.Confirmation.PlanDigest != preparedView.PlanDigest ||
		confirmationView.Confirmation.GoalRef == "" ||
		confirmationView.Confirmation.AppSpecRef == "" ||
		confirmationView.Confirmation.SpecHash == "" ||
		confirmationView.Confirmation.ConfirmedAt == "" ||
		confirmationView.Goal.GoalRef != confirmationView.Confirmation.GoalRef ||
		confirmationView.Goal.ProjectRef != projectRef ||
		confirmationView.Goal.SpecHash != confirmationView.Confirmation.SpecHash ||
		!strings.HasPrefix(
			confirmationView.Confirmation.ReceiptRef,
			"intake-dossier-confirmation-receipt:",
		) {
		t.Fatalf("confirmation=%+v dossier=%+v", confirmationView, preparedView)
	}
	replayedConfirmation := dispatchV23IntakeCommand(
		t, second, principal, projectRef,
		"orquesta.intakes.dossier.confirm", "request:v23-dossier-confirm",
		map[string]any{
			"dossier_ref": preparedView.DossierRef,
			"confirm":     true,
		},
	)
	if replayedConfirmation.Failure != nil ||
		replayedConfirmation.AuditRef != confirmation.AuditRef ||
		!bytes.Equal(replayedConfirmation.Data, confirmation.Data) {
		t.Fatalf(
			"confirmation replay=%+v first=%+v",
			replayedConfirmation, confirmation,
		)
	}
}

type v23DossierView struct {
	Schema                 string `json:"schema"`
	DossierRef             string `json:"dossier_ref"`
	ActorRef               string `json:"actor_ref"`
	ProjectRef             string `json:"project_ref"`
	IntakeRef              string `json:"intake_ref"`
	IntakeRevision         uint64 `json:"intake_revision"`
	IntakeDigest           string `json:"intake_digest"`
	SourceIntakeReceiptRef string `json:"source_intake_receipt_ref"`
	PlanDigest             string `json:"plan_digest"`
	Digest                 string `json:"digest"`
	GenerationReceipt      struct {
		ReceiptRef             string `json:"receipt_ref"`
		RequestRef             string `json:"request_ref"`
		ActorRef               string `json:"actor_ref"`
		ProjectRef             string `json:"project_ref"`
		DossierRef             string `json:"dossier_ref"`
		SourceIntakeReceiptRef string `json:"source_intake_receipt_ref"`
	} `json:"generation_receipt"`
}

type v23DossierConfirmationOutput struct {
	Goal struct {
		GoalRef    string `json:"goal_ref"`
		ProjectRef string `json:"project_ref"`
		SpecHash   string `json:"spec_hash"`
	} `json:"goal"`
	Confirmation struct {
		ReceiptRef             string `json:"receipt_ref"`
		RequestRef             string `json:"request_ref"`
		ActorRef               string `json:"actor_ref"`
		ProjectRef             string `json:"project_ref"`
		IntakeRef              string `json:"intake_ref"`
		IntakeRevision         uint64 `json:"intake_revision"`
		IntakeDigest           string `json:"intake_digest"`
		SourceIntakeReceiptRef string `json:"source_intake_receipt_ref"`
		DossierRef             string `json:"dossier_ref"`
		PlanDigest             string `json:"plan_digest"`
		GoalRef                string `json:"goal_ref"`
		AppSpecRef             string `json:"app_spec_ref"`
		SpecHash               string `json:"spec_hash"`
		ConfirmedAt            string `json:"confirmed_at"`
	} `json:"confirmation"`
}

func decodeV23Dossier(t *testing.T, result commandcore.Result) v23DossierView {
	t.Helper()
	if result.Failure != nil || result.AuditRef == "" {
		t.Fatalf("dossier command=%+v", result)
	}
	var output struct {
		Dossier v23DossierView `json:"dossier"`
	}
	if err := json.Unmarshal(result.Data, &output); err != nil {
		t.Fatalf("decode dossier %s: %v", result.Data, err)
	}
	return output.Dossier
}

func decodeV23DossierConfirmation(
	t *testing.T,
	result commandcore.Result,
) v23DossierConfirmationOutput {
	t.Helper()
	if result.Failure != nil || result.AuditRef == "" {
		t.Fatalf("dossier confirmation command=%+v", result)
	}
	var output v23DossierConfirmationOutput
	if err := json.Unmarshal(result.Data, &output); err != nil {
		t.Fatalf("decode dossier confirmation %s: %v", result.Data, err)
	}
	return output
}

func getV23Dossier(
	t *testing.T,
	runtime *Runtime,
	principal identity.Principal,
	projectRef, requestRef, dossierRef string,
) v23DossierView {
	t.Helper()
	result := dispatchV23IntakeCommand(
		t, runtime, principal, projectRef,
		"orquesta.intakes.dossier.get", requestRef,
		map[string]any{"dossier_ref": dossierRef},
	)
	return decodeV23Dossier(t, result)
}

func assertV23PreparedDossier(
	t *testing.T,
	view v23DossierView,
	actorRef, projectRef, requestRef, sourceReceiptRef string,
) {
	t.Helper()
	if view.Schema != "orquesta.intake.dossier.v1" ||
		view.DossierRef == "" ||
		view.ActorRef != actorRef ||
		view.ProjectRef != projectRef ||
		view.IntakeRef != "intake:v23-dossier-e2e" ||
		view.IntakeRevision != 3 ||
		view.IntakeDigest == "" ||
		view.SourceIntakeReceiptRef != sourceReceiptRef ||
		view.PlanDigest == "" ||
		view.Digest == "" ||
		view.GenerationReceipt.ReceiptRef == "" ||
		view.GenerationReceipt.RequestRef != requestRef ||
		view.GenerationReceipt.ActorRef != actorRef ||
		view.GenerationReceipt.ProjectRef != projectRef ||
		view.GenerationReceipt.DossierRef != view.DossierRef ||
		view.GenerationReceipt.SourceIntakeReceiptRef != sourceReceiptRef {
		t.Fatalf("prepared dossier=%+v", view)
	}
}

func equalV23DossierIdentity(left, right v23DossierView) bool {
	return left.Schema == right.Schema &&
		left.DossierRef == right.DossierRef &&
		left.ActorRef == right.ActorRef &&
		left.ProjectRef == right.ProjectRef &&
		left.IntakeRef == right.IntakeRef &&
		left.IntakeRevision == right.IntakeRevision &&
		left.IntakeDigest == right.IntakeDigest &&
		left.SourceIntakeReceiptRef == right.SourceIntakeReceiptRef &&
		left.PlanDigest == right.PlanDigest &&
		left.Digest == right.Digest
}

func v23DossierQuestionPayload(expectedRevision uint64) map[string]any {
	payload := v23IntakeApplyPayload(expectedRevision)
	payload["intake_ref"] = "intake:v23-dossier-e2e"
	return payload
}

func v23DossierAnswerPayload(expectedRevision uint64) map[string]any {
	return map[string]any{
		"intake_ref": "intake:v23-dossier-e2e", "expected_revision": expectedRevision,
		"origin": "form", "issues": []any{}, "questions": []any{},
		"choices": []any{map[string]any{
			"question_ref": "intake-question:audience",
			"option_ref":   "intake-option:audience-team",
		}},
	}
}

func v23DossierPreparePayload(sourceReceiptRef string) map[string]any {
	sectionKinds := []string{
		"product_scope", "users_roles", "architecture", "data", "integrations",
		"security_privacy", "ui_ux", "i18n_l10n", "deploy_operations",
		"orchestration_plan", "risks_open_issues",
	}
	sections := make([]any, 0, len(sectionKinds))
	for _, kind := range sectionKinds {
		sections = append(sections, map[string]any{
			"ref":       "intake-dossier-section:" + kind,
			"kind":      kind,
			"title_key": "intake.dossier." + kind + ".title",
			"markdown":  "Contenido " + kind,
		})
	}
	diagramPurposes := []string{
		"architecture", "user_flow", "data_integrations", "i18n", "deployment",
	}
	diagrams := make([]any, 0, len(diagramPurposes))
	for _, purpose := range diagramPurposes {
		diagrams = append(diagrams, map[string]any{
			"ref":          "intake-dossier-diagram:" + purpose,
			"purpose":      purpose,
			"kind":         "mermaid",
			"source":       "flowchart TD\n  source --> " + purpose,
			"alt_text_key": "intake.dossier.diagram." + purpose + ".alt",
		})
	}
	return map[string]any{
		"intake_ref":                "intake:v23-dossier-e2e",
		"expected_revision":         3,
		"source_intake_receipt_ref": sourceReceiptRef,
		"statement":                 "Quiero una agenda compartida",
		"objective":                 "Crear una aplicación de agenda",
		"sections":                  sections,
		"diagrams":                  diagrams,
		"risk_refs":                 []any{"intake-risk:calendar-provider"},
		"plan": map[string]any{
			"phases": []any{map[string]any{
				"ref": "phase-instance:v23-dossier", "key": "phase.v23-dossier",
				"template_ref":   "phase-template:v23-dossier",
				"input_refs":     []any{"input:v23-dossier"},
				"criterion_refs": []any{"criterion:v23-dossier-reviewed"},
			}},
			"work_items": []any{map[string]any{
				"key": "work:v23-dossier", "objective": "Crear la aplicación confirmada",
				"phase": "phase.v23-dossier", "role": "role:default",
				"dependencies": []any{}, "write_set": []any{},
				"output_contract": "evidence_bundle",
			}},
		},
	}
}

func cloneV23Map(t *testing.T, source map[string]any) map[string]any {
	t.Helper()
	encoded, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	var cloned map[string]any
	if err := json.Unmarshal(encoded, &cloned); err != nil {
		t.Fatal(err)
	}
	return cloned
}
