package commands

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

func TestV22GoalAndMailboxPublicSchemasExposeCausalEvidenceWithoutPrivateFields(t *testing.T) {
	goalDefinition := v22Definition(t, "orquesta.goals.get")
	var goalSchema struct {
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	}
	if err := json.Unmarshal(goalDefinition.OutputSchema, &goalSchema); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{
		"goal", "execution_count", "artifact_count", "work_items", "executions",
		"attestations", "reviews", "controls", "integration_receipts", "mailbox_receipts",
	} {
		if _, ok := goalSchema.Properties[field]; !ok || !containsV22(goalSchema.Required, field) {
			t.Fatalf("goals.get missing required public field %q", field)
		}
	}
	var goalObject struct {
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	}
	if err := json.Unmarshal(goalSchema.Properties["goal"], &goalObject); err != nil {
		t.Fatal(err)
	}
	if _, ok := goalObject.Properties["app_spec_generation"]; !ok ||
		!containsV22(goalObject.Required, "app_spec_generation") {
		t.Fatal("goals.get goal lacks required app_spec_generation")
	}
	getMailbox := v22Definition(t, "orquesta.mailbox.get")
	listMailbox := v22Definition(t, "orquesta.mailbox.list")
	var listSchema struct {
		Properties map[string]struct {
			Items json.RawMessage `json:"items"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(listMailbox.OutputSchema, &listSchema); err != nil {
		t.Fatal(err)
	}
	messages, ok := listSchema.Properties["messages"]
	if !ok || !v22JSONSemanticallyEqual(getMailbox.OutputSchema, messages.Items) {
		t.Fatal("mailbox.get and mailbox.list item schemas drift")
	}

	for _, definition := range []Definition{goalDefinition, getMailbox, listMailbox} {
		body := strings.ToLower(string(definition.OutputSchema))
		for _, forbidden := range []string{
			"external_ref", "repository_ref", "execution_workspace_ref", "provider_ref",
			"model_ref", "agent_ref", "idempotency_key", "effect_intent_ref",
			"request_fingerprint", "claim_token",
		} {
			if strings.Contains(body, forbidden) {
				t.Fatalf("%s exposes forbidden field %q", definition.ID, forbidden)
			}
		}
	}
}

func TestV22PublicProjectionCarriesRefsAttemptsStatesCodesAndDoesNotLeakPrivateValues(t *testing.T) {
	workItem := v22WorkItem(t)
	execution := application.ExecutionRecord{
		Ref: v22ExecutionRef(t, "execution:public"), WorkItemRef: workItem.Ref(),
		AttemptNo: 2, MaxExecutionAttempts: 3, ReplacesExecutionRef: v22ExecutionRef(t, "execution:old"),
		PlanGeneration: 4, AppSpecGeneration: 5, State: application.ExecutionStopped,
		Purpose: application.ExecutionPurposeWork, FailureCode: "agent.stopped",
		RecipientMailboxRetired: true,
		ExternalRef:             "secret-external", ProviderRef: "secret-provider", ModelRef: "secret-model",
		AgentRef: "secret-agent", IdempotencyKey: "secret-idempotency",
	}
	testRef, err := goal.NewRequiredTestRef("required-test:public")
	if err != nil {
		t.Fatal(err)
	}
	changeRef, err := ports.NewChangeSetRef("change:public")
	if err != nil {
		t.Fatal(err)
	}
	attestationRef, err := goal.NewAttestationRef("attestation:public")
	if err != nil {
		t.Fatal(err)
	}
	attestation := application.AttestationRecord{
		Ref: attestationRef, Kind: application.AttestationKindRequiredTests,
		Verdict: application.AttestationVerdictPassed, WorkItemRef: workItem.Ref(),
		ExecutionRef: execution.Ref, ExecutionAttempt: 2, PlanGeneration: 4,
		WorkItemGeneration: 3, AppSpecGeneration: 5, ChangeSetRef: changeRef,
		Tests:       []ports.RequiredTestOutcome{{RequiredTestRef: testRef, ExitCode: 0, OutputDigest: "sha256:test-output"}},
		AttestorRef: "secret-attestor", ReceiptRef: "secret-attestation-receipt",
	}
	review := application.ReviewRecord{
		Ref: "review:public", WorkItemRef: workItem.Ref(), ChangeSetRef: changeRef,
		SubjectDigest: "sha256:subject", Role: "primary", Verdict: "approved",
		ReviewerExecutionRef: execution.Ref, ReviewerExecutionAttempt: 2,
		ExternalRef: "secret-review-external", AgentRef: "secret-review-agent",
	}
	control := application.ControlRecord{
		Ref: "control:public", Operation: application.ControlStop, Target: application.ControlTargetExecution,
		Status: application.ControlConfirmed, Mode: "force", GoalRevision: 6, PlanGeneration: 4,
		AppSpecGeneration: 5, WorkItemRef: workItem.Ref(), WorkItemRevision: 3,
		ExecutionRef: execution.Ref, ExecutionAttempt: 2, ReceiptRef: "control-receipt:public",
		Reason: "secret-control-reason", RequestFingerprint: "secret-control-fingerprint",
	}
	integration := application.IntegrationReceipt{
		Ref: "integration:public", ChangeRef: changeRef, Status: "integrated",
		TargetBeforeOID: "oid-before", TargetAfterOID: "oid-after", TreeOID: "oid-tree",
		SourceOID: "secret-source-oid", TargetRef: "secret-target-ref", AdapterRef: "secret-adapter",
	}

	projected := goalRecordView{
		Goal: projectGoal(goal.Goal{}), ExecutionCount: 1, ArtifactCount: 0,
		WorkItems:    []workItemView{projectWorkItem(workItem)},
		Executions:   []executionView{projectExecution(execution)},
		Attestations: []attestationView{projectAttestation(attestation)},
		Reviews:      []reviewView{projectReview(review)}, Controls: []controlView{projectControl(control)},
		IntegrationReceipts: []integrationReceiptView{projectIntegrationReceipt(integration)},
		MailboxReceipts:     []mailboxReceiptView{},
	}
	encoded, err := json.Marshal(projected)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := validatePayload(v22Definition(t, "orquesta.goals.get").OutputSchema, encoded); err != nil {
		t.Fatalf("projected goals.get output violates schema: %v data=%s", err, encoded)
	}
	for _, want := range []string{
		"execution:public", "execution:old", "agent.stopped", "required-test:public",
		"review:public", "control:public", "integration:public", "work-item:child",
	} {
		if !strings.Contains(string(encoded), want) {
			t.Fatalf("public output lacks %q: %s", want, encoded)
		}
	}
	for _, secret := range []string{
		"secret-external", "secret-provider", "secret-model", "secret-agent", "secret-idempotency",
		"secret-attestor", "secret-attestation-receipt", "secret-review-external",
		"secret-review-agent", "secret-control-reason", "secret-control-fingerprint",
		"secret-source-oid", "secret-target-ref", "secret-adapter",
	} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("public output leaked %q: %s", secret, encoded)
		}
	}
}

func TestV22MailboxProjectionExposesCompactHandoffAndNoAdmissionSecrets(t *testing.T) {
	messageRef, err := application.NewMailboxMessageRef("mailbox-message:public")
	if err != nil {
		t.Fatal(err)
	}
	artifactRef, err := goal.NewArtifactRef("artifact:public")
	if err != nil {
		t.Fatal(err)
	}
	record := application.MailboxRecord{
		Envelope: application.MailboxEnvelope{
			Ref: messageRef, GoalRef: v22GoalRef(t, "goal:public"), TargetPlanGeneration: 7,
			Kind:              application.MailboxKindChildDelivery,
			ParentWorkItemRef: v22WorkItemRef(t, "work-item:parent"),
			ChildWorkItemRef:  v22WorkItemRef(t, "work-item:child"),
			Source: application.MailboxEndpoint{
				PrincipalRef: v22PrincipalRef(t, "principal:child"),
				WorkItemRef:  v22WorkItemRef(t, "work-item:child"),
				ExecutionRef: v22ExecutionRef(t, "execution:child"),
			},
			Recipient: application.MailboxEndpoint{
				PrincipalRef: v22PrincipalRef(t, "principal:parent"),
				WorkItemRef:  v22WorkItemRef(t, "work-item:parent"),
				ExecutionRef: v22ExecutionRef(t, "execution:parent"),
			},
			Summary: "child result ready", ArtifactRefs: []goal.ArtifactRef{artifactRef},
			RequestFingerprint: "secret-mailbox-fingerprint", ContentHash: "secret-content-hash",
		},
		Admission: application.MailboxAdmissionReceipt{
			Ref: "admission:public", PrincipalRef: v22PrincipalRef(t, "principal:child"),
		},
		State: application.MailboxStateAcknowledged,
		Attempts: []application.MailboxDeliveryAttempt{{
			ClaimToken: "secret-claim-token", DeliveryRef: "secret-delivery-receipt",
			ConsumptionRef: "consumption:public",
		}},
		Acknowledgement: &application.MailboxAcknowledgement{
			Ref: "acknowledgement:public", Outcome: application.MailboxOutcomeAcknowledged,
		},
	}
	projected := projectMailbox(record)
	encoded, err := json.Marshal(projected)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := validatePayload(v22Definition(t, "orquesta.mailbox.get").OutputSchema, encoded); err != nil {
		t.Fatalf("projected mailbox output violates schema: %v data=%s", err, encoded)
	}
	for _, want := range []string{
		"work-item:parent", "work-item:child", "execution:child", "execution:parent",
		"child result ready", "artifact:public",
	} {
		if !strings.Contains(string(encoded), want) {
			t.Fatalf("mailbox output lacks %q: %s", want, encoded)
		}
	}
	for _, secret := range []string{"secret-mailbox-fingerprint", "secret-content-hash", "secret-claim-token", "secret-delivery-receipt"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("mailbox output leaked %q: %s", secret, encoded)
		}
	}
	goalEncoded, err := json.Marshal(projectGoalRecord(application.GoalRecord{Mailboxes: []application.MailboxRecord{record}}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := validatePayload(v22Definition(t, "orquesta.goals.get").OutputSchema, goalEncoded); err != nil {
		t.Fatalf("mailbox receipt projection violates goals.get schema: %v data=%s", err, goalEncoded)
	}
	receipt := projectMailboxReceipt(record)
	if receipt.SourcePrincipalRef != "principal:child" || receipt.SourceExecutionRef != "execution:child" ||
		receipt.RecipientPrincipalRef != "principal:parent" || receipt.RecipientExecutionRef != "execution:parent" ||
		receipt.AdmissionRef != "admission:public" || receipt.ConsumptionRef != "consumption:public" ||
		receipt.AcknowledgementRef != "acknowledgement:public" || receipt.Outcome != "acknowledged" {
		t.Fatalf("goals.get mailbox receipt lost causal refs: %+v", receipt)
	}
	if strings.Contains(string(goalEncoded), "secret-") {
		t.Fatalf("goals.get mailbox receipts leaked private values: %s", goalEncoded)
	}
}

func TestV22EmptyPublicCollectionsEncodeAsArrays(t *testing.T) {
	encoded, err := json.Marshal(projectGoalRecord(application.GoalRecord{}))
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"work_items", "executions", "attestations", "reviews", "controls", "integration_receipts", "mailbox_receipts"} {
		if value, ok := decoded[field]; !ok || reflect.TypeOf(value).Kind() != reflect.Slice {
			t.Fatalf("%s=%#v, want array", field, value)
		}
	}
}

func v22Definition(t *testing.T, id string) Definition {
	t.Helper()
	for _, definition := range CanonicalDefinitions() {
		if definition.ID == id {
			return definition
		}
	}
	t.Fatalf("missing definition %s", id)
	return Definition{}
}

func v22WorkItem(t *testing.T) goal.WorkItem {
	t.Helper()
	item, err := goal.NewWorkItem(goal.NewWorkItemInput{
		Ref: v22WorkItemRef(t, "work-item:child"), Goal: v22GoalRef(t, "goal:public"),
		Actor: v22ActorRef(t, "actor:public"), Project: v22ProjectRef(t, "project:public"),
		Objective: "return child result", CreatedAt: time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC),
		Parent: v22WorkItemRef(t, "work-item:parent"), HandoffRequired: true,
		Dependencies: []goal.WorkItemRef{v22WorkItemRef(t, "work-item:dependency")},
	})
	if err != nil {
		t.Fatal(err)
	}
	return item
}

func v22GoalRef(t *testing.T, value string) goal.GoalRef {
	t.Helper()
	ref, err := goal.NewGoalRef(value)
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

func v22WorkItemRef(t *testing.T, value string) goal.WorkItemRef {
	t.Helper()
	ref, err := goal.NewWorkItemRef(value)
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

func v22ExecutionRef(t *testing.T, value string) goal.ExecutionRef {
	t.Helper()
	ref, err := goal.NewExecutionRef(value)
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

func v22ActorRef(t *testing.T, value string) goal.ActorRef {
	t.Helper()
	ref, err := goal.NewActorRef(value)
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

func v22ProjectRef(t *testing.T, value string) goal.ProjectRef {
	t.Helper()
	ref, err := goal.NewProjectRef(value)
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

func v22PrincipalRef(t *testing.T, value string) identity.PrincipalRef {
	t.Helper()
	ref, err := identity.NewPrincipalRef(value)
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

func containsV22(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func v22JSONSemanticallyEqual(left, right json.RawMessage) bool {
	var leftValue, rightValue any
	return json.Unmarshal(left, &leftValue) == nil &&
		json.Unmarshal(right, &rightValue) == nil &&
		reflect.DeepEqual(leftValue, rightValue)
}
