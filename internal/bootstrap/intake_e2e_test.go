package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"

	commandcore "orquesta/internal/commands"
	"orquesta/internal/identity"
	"orquesta/internal/intake"
)

func TestV23IntakeDispatcherPersistsCASAndReplayAcrossRestart(t *testing.T) {
	root := t.TempDir()
	configPath := writeTestConfig(t, root)
	first := buildV23IntakeRuntime(t, configPath)
	principal, hierarchy, err := localIdentityComposition(first.config)
	if err != nil {
		t.Fatal(err)
	}
	projectRef := hierarchy.ProjectRef().String()

	spoofed := dispatchV23IntakeCommand(t, first, principal, projectRef,
		"orquesta.intakes.create", "request:v23-intake-spoof", map[string]any{
			"intake_ref": "intake:v23-e2e", "max_question_rounds": 2,
			"actor_ref": "actor:spoofed", "project_ref": "project:spoofed",
		})
	if spoofed.Failure == nil || spoofed.Failure.Code != commandcore.CodeInvalidRequest ||
		spoofed.AuditRef != "" {
		t.Fatalf("payload authority was admitted: %+v", spoofed)
	}

	createPayload := map[string]any{
		"intake_ref": "intake:v23-e2e", "max_question_rounds": 2,
	}
	created := dispatchV23IntakeCommand(t, first, principal, projectRef,
		"orquesta.intakes.create", "request:v23-intake-create", createPayload)
	createdView := decodeV23IntakeMutation(t, created)
	if createdView.IntakeRef != "intake:v23-e2e" ||
		createdView.ProjectRef != projectRef ||
		createdView.Revision != 1 ||
		createdView.ReceiptRef == "" {
		t.Fatalf("create projection=%+v", createdView)
	}
	createReplay := dispatchV23IntakeCommand(t, first, principal, projectRef,
		"orquesta.intakes.create", "request:v23-intake-create", createPayload)
	assertV23IntakeReplay(t, created, createReplay, createdView.ReceiptRef)

	invalidGet := dispatchV23IntakeCommand(t, first, principal, projectRef,
		"orquesta.intakes.get", "request:v23-intake-invalid-get", map[string]any{
			"intake_ref": "form:private-copy",
		})
	if invalidGet.Failure == nil ||
		invalidGet.Failure.Code != commandcore.CodeInvalidRequest ||
		invalidGet.AuditRef == "" ||
		len(invalidGet.Data) != 0 {
		t.Fatalf("invalid public intake ref=%+v", invalidGet)
	}

	applyPayload := v23IntakeApplyPayload(1)
	applied := dispatchV23IntakeCommand(t, first, principal, projectRef,
		"orquesta.intakes.apply", "request:v23-intake-apply", applyPayload)
	appliedView := decodeV23IntakeMutation(t, applied)
	if appliedView.IntakeRef != createdView.IntakeRef ||
		appliedView.ProjectRef != projectRef ||
		appliedView.Revision != 2 ||
		appliedView.ReceiptRef == "" ||
		appliedView.ReceiptRef == createdView.ReceiptRef {
		t.Fatalf("apply projection=%+v create=%+v", appliedView, createdView)
	}
	applyReplay := dispatchV23IntakeCommand(t, first, principal, projectRef,
		"orquesta.intakes.apply", "request:v23-intake-apply", applyPayload)
	assertV23IntakeReplay(t, applied, applyReplay, appliedView.ReceiptRef)

	stale := dispatchV23IntakeCommand(t, first, principal, projectRef,
		"orquesta.intakes.apply", "request:v23-intake-stale", v23IntakeApplyPayload(1))
	if stale.Failure == nil || stale.Failure.Code != commandcore.CodeConflict ||
		stale.AuditRef == "" || len(stale.Data) != 0 {
		t.Fatalf("stale CAS=%+v", stale)
	}
	current := getV23Intake(t, first, principal, projectRef, "request:v23-intake-get-first")
	assertV23IntakeState(t, current, principal, projectRef, appliedView.ReceiptRef)
	shutdownRuntime(t, first)

	second := buildV23IntakeRuntime(t, configPath)
	t.Cleanup(func() { shutdownRuntime(t, second) })
	restartedPrincipal, restartedHierarchy, err := localIdentityComposition(second.config)
	if err != nil {
		t.Fatal(err)
	}
	if restartedPrincipal != principal ||
		restartedHierarchy.ProjectRef().String() != projectRef {
		t.Fatalf("restart authority changed: principal=%+v project=%s",
			restartedPrincipal, restartedHierarchy.ProjectRef())
	}
	restartedReplay := dispatchV23IntakeCommand(t, second, principal, projectRef,
		"orquesta.intakes.apply", "request:v23-intake-apply", applyPayload)
	assertV23IntakeReplay(t, applied, restartedReplay, appliedView.ReceiptRef)
	restarted := getV23Intake(
		t, second, principal, projectRef, "request:v23-intake-get-restart",
	)
	assertV23IntakeState(t, restarted, principal, projectRef, appliedView.ReceiptRef)
}

func buildV23IntakeRuntime(t *testing.T, configPath string) *Runtime {
	t.Helper()
	runtime, err := Build(context.Background(), Options{
		ConfigPath: configPath, Version: "v23-intake-e2e",
		AgentFactory: countingFactory(new(atomic.Int64)),
	})
	if err != nil {
		t.Fatalf("build V23 intake runtime: %v", err)
	}
	return runtime
}

func dispatchV23IntakeCommand(
	t *testing.T,
	runtime *Runtime,
	principal identity.Principal,
	projectRef, commandID, requestRef string,
	payload any,
) commandcore.Result {
	t.Helper()
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return runtime.dispatcher.Dispatch(context.Background(), commandcore.Invocation{
		CommandID: commandID, CommandVersion: "1", RequestRef: requestRef,
		ProjectRef: projectRef, Principal: principal, Payload: encoded,
	})
}

type v23IntakeMutationView struct {
	IntakeRef  string `json:"intake_ref"`
	ProjectRef string `json:"project_ref"`
	Revision   uint64 `json:"revision"`
	ReceiptRef string `json:"receipt_ref"`
}

func decodeV23IntakeMutation(
	t *testing.T,
	result commandcore.Result,
) v23IntakeMutationView {
	t.Helper()
	if result.Failure != nil || result.AuditRef == "" {
		t.Fatalf("intake mutation=%+v", result)
	}
	var output struct {
		Intake v23IntakeMutationView `json:"intake"`
	}
	if err := json.Unmarshal(result.Data, &output); err != nil {
		t.Fatalf("decode intake mutation %s: %v", result.Data, err)
	}
	return output.Intake
}

func assertV23IntakeReplay(
	t *testing.T,
	original, replay commandcore.Result,
	receiptRef string,
) {
	t.Helper()
	if replay.Failure != nil ||
		replay.AuditRef != original.AuditRef ||
		!bytes.Equal(replay.Data, original.Data) ||
		decodeV23IntakeMutation(t, replay).ReceiptRef != receiptRef {
		t.Fatalf("replay=%+v original=%+v", replay, original)
	}
}

type v23IntakeStateView struct {
	StateSchema       string            `json:"state_schema"`
	IntakeRef         string            `json:"intake_ref"`
	ActorRef          string            `json:"actor_ref"`
	ProjectRef        string            `json:"project_ref"`
	Revision          uint64            `json:"revision"`
	MaxQuestionRounds uint32            `json:"max_question_rounds"`
	QuestionRounds    uint32            `json:"question_rounds"`
	Issues            []intake.Issue    `json:"issues"`
	Questions         []intake.Question `json:"questions"`
}

func getV23Intake(
	t *testing.T,
	runtime *Runtime,
	principal identity.Principal,
	projectRef, requestRef string,
) v23IntakeStateView {
	t.Helper()
	result := dispatchV23IntakeCommand(t, runtime, principal, projectRef,
		"orquesta.intakes.get", requestRef, map[string]any{
			"intake_ref": "intake:v23-e2e",
		})
	if result.Failure != nil || result.AuditRef == "" {
		t.Fatalf("get intake=%+v", result)
	}
	var output struct {
		Intake v23IntakeStateView `json:"intake"`
	}
	if err := json.Unmarshal(result.Data, &output); err != nil {
		t.Fatalf("decode intake state %s: %v", result.Data, err)
	}
	return output.Intake
}

func assertV23IntakeState(
	t *testing.T,
	state v23IntakeStateView,
	principal identity.Principal,
	projectRef, expectedReceipt string,
) {
	t.Helper()
	if state.StateSchema != intake.StateSchema ||
		state.IntakeRef != "intake:v23-e2e" ||
		state.ActorRef != principal.ActorRef.String() ||
		state.ProjectRef != projectRef ||
		state.Revision != 2 ||
		state.MaxQuestionRounds != 2 ||
		state.QuestionRounds != 1 ||
		len(state.Issues) != 1 ||
		len(state.Questions) != 1 {
		t.Fatalf("persisted intake=%+v expected_receipt=%s", state, expectedReceipt)
	}
}

func v23IntakeApplyPayload(expectedRevision uint64) map[string]any {
	return map[string]any{
		"intake_ref": "intake:v23-e2e", "expected_revision": expectedRevision,
		"origin": "chat",
		"issues": []any{map[string]any{
			"ref": "intake-issue:audience", "kind": "gap",
			"field": "audience", "detail_key": "intake.issue.audience.missing",
		}},
		"questions": []any{map[string]any{
			"ref":          "intake-question:audience",
			"derived_from": []any{"intake-issue:audience"},
			"prompt_key":   "intake.question.audience.prompt",
			"why_key":      "intake.question.audience.why",
			"options": []any{
				map[string]any{
					"ref":           "intake-option:audience-team",
					"label_key":     "intake.option.audience.team.label",
					"rationale_key": "intake.option.audience.team.rationale",
					"recommended":   true,
				},
				map[string]any{
					"ref":           "intake-option:audience-personal",
					"label_key":     "intake.option.audience.personal.label",
					"rationale_key": "intake.option.audience.personal.rationale",
					"recommended":   false,
				},
			},
		}},
		"choices": []any{},
	}
}
