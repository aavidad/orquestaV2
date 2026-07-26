package commands

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/intake"
)

func (api *fakeApplication) CreateIntake(
	_ context.Context,
	_ application.Access,
	request application.CreateIntakeRequest,
) (application.IntakeResult, error) {
	if _, err := application.IntakeAuthorizationRequestRef(
		application.IntakeOperationCreate, request.RequestRef,
	); err != nil {
		return application.IntakeResult{}, err
	}
	api.called("CreateIntake")
	state, err := intake.NewState(request.StateRef, request.Policy)
	return application.IntakeResult{Record: application.IntakeRecord{
		ActorRef: request.ActorRef, ProjectRef: request.ProjectRef, State: state,
		Receipt: application.IntakeReceipt{Ref: "intake-receipt:create"},
	}, Changed: true}, err
}

func (api *fakeApplication) GetIntake(
	_ context.Context,
	_ application.Access,
	request application.GetIntakeRequest,
) (application.IntakeRecord, error) {
	api.called("GetIntake")
	state, err := intake.NewState(request.StateRef, intake.Policy{MaxQuestionRounds: 2})
	return application.IntakeRecord{
		ActorRef: request.ActorRef, ProjectRef: request.ProjectRef, State: state,
	}, err
}

func (api *fakeApplication) ApplyIntake(
	_ context.Context,
	_ application.Access,
	request application.ApplyIntakeRequest,
) (application.IntakeResult, error) {
	if _, err := application.IntakeAuthorizationRequestRef(
		application.IntakeOperationApply, request.RequestRef,
	); err != nil {
		return application.IntakeResult{}, err
	}
	api.called("ApplyIntake")
	state, err := intake.NewState(request.Change.StateRef, intake.Policy{MaxQuestionRounds: 2})
	if err == nil {
		state, err = intake.Apply(state, request.Change)
	}
	return application.IntakeResult{Record: application.IntakeRecord{
		ActorRef: request.ActorRef, ProjectRef: request.ProjectRef, State: state,
		Receipt: application.IntakeReceipt{Ref: "intake-receipt:apply"},
	}, Changed: true}, err
}

type captureIntakeApplication struct {
	*fakeApplication
	creates []application.CreateIntakeRequest
	gets    []application.GetIntakeRequest
	applies []application.ApplyIntakeRequest
}

func (api *captureIntakeApplication) CreateIntake(
	ctx context.Context,
	access application.Access,
	request application.CreateIntakeRequest,
) (application.IntakeResult, error) {
	api.creates = append(api.creates, request)
	return api.fakeApplication.CreateIntake(ctx, access, request)
}

func (api *captureIntakeApplication) GetIntake(
	ctx context.Context,
	access application.Access,
	request application.GetIntakeRequest,
) (application.IntakeRecord, error) {
	api.gets = append(api.gets, request)
	return api.fakeApplication.GetIntake(ctx, access, request)
}

func (api *captureIntakeApplication) ApplyIntake(
	ctx context.Context,
	access application.Access,
	request application.ApplyIntakeRequest,
) (application.IntakeResult, error) {
	api.applies = append(api.applies, request)
	return api.fakeApplication.ApplyIntake(ctx, access, request)
}

func TestIntakeCommandsBindAuthorityOutsidePayloadAndProjectPublicState(t *testing.T) {
	api := &captureIntakeApplication{fakeApplication: newFakeApplication()}
	audit := newMemoryAudit()
	dispatcher, err := newDispatcher(
		api, audit, APILimits{MaxRequestBytes: 1 << 20, MaxListLimit: 100},
		exactTestExecutionAuthority(t),
	)
	if err != nil {
		t.Fatal(err)
	}

	create := invoke(t, dispatcher, "orquesta.intakes.create", "request:intake-create", map[string]any{
		"intake_ref": "intake:test", "max_question_rounds": 2,
	}, false)
	get := invoke(t, dispatcher, "orquesta.intakes.get", "request:intake-get", map[string]any{
		"intake_ref": "intake:test",
	}, false)
	apply := invoke(t, dispatcher, "orquesta.intakes.apply", "request:intake-apply", canonicalIntakeApplyPayload(), false)
	for name, result := range map[string]Result{"create": create, "get": get, "apply": apply} {
		if result.Failure != nil {
			t.Fatalf("%s=%+v", name, result)
		}
	}
	if len(api.creates) != 1 || len(api.gets) != 1 || len(api.applies) != 1 {
		t.Fatalf("requests=%d/%d/%d", len(api.creates), len(api.gets), len(api.applies))
	}
	if api.creates[0].RequestRef != "request:intake-create" ||
		api.creates[0].ActorRef.String() != "actor:test" ||
		api.creates[0].ProjectRef.String() != "project:test" ||
		api.applies[0].RequestRef != "request:intake-apply" ||
		api.applies[0].ActorRef.String() != "actor:test" ||
		api.applies[0].ProjectRef.String() != "project:test" ||
		api.gets[0].ActorRef.String() != "actor:test" ||
		api.gets[0].ProjectRef.String() != "project:test" {
		t.Fatalf("authority leaked from payload: create=%+v get=%+v apply=%+v",
			api.creates[0], api.gets[0], api.applies[0])
	}
	var projected struct {
		Intake struct {
			StateSchema string            `json:"state_schema"`
			IntakeRef   string            `json:"intake_ref"`
			ActorRef    string            `json:"actor_ref"`
			ProjectRef  string            `json:"project_ref"`
			Issues      []json.RawMessage `json:"issues"`
			Questions   []json.RawMessage `json:"questions"`
			Decisions   []json.RawMessage `json:"decisions"`
			History     []json.RawMessage `json:"history"`
		} `json:"intake"`
	}
	if err := json.Unmarshal(get.Data, &projected); err != nil {
		t.Fatal(err)
	}
	if projected.Intake.StateSchema != intake.StateSchema ||
		projected.Intake.IntakeRef != "intake:test" ||
		projected.Intake.ActorRef != "actor:test" ||
		projected.Intake.ProjectRef != "project:test" ||
		projected.Intake.Issues == nil || projected.Intake.Questions == nil ||
		projected.Intake.Decisions == nil || projected.Intake.History == nil {
		t.Fatalf("public intake=%s", get.Data)
	}

	spoof := invoke(t, dispatcher, "orquesta.intakes.create", "request:intake-spoof", map[string]any{
		"intake_ref": "intake:test", "max_question_rounds": 2,
		"actor_ref": "actor:spoof", "project_ref": "project:spoof",
	}, false)
	if spoof.Failure == nil || spoof.Failure.Code != CodeInvalidRequest ||
		len(api.creates) != 1 || audit.admits != 3 {
		t.Fatalf("spoof=%+v creates=%d admits=%d", spoof, len(api.creates), audit.admits)
	}
}

func TestIntakeMutationReplayKeepsStablePublicReceipt(t *testing.T) {
	dispatcher, api, _ := testDispatcher(t)
	payload := canonicalIntakeApplyPayload()
	first := invoke(t, dispatcher, "orquesta.intakes.apply", "request:intake-replay", payload, false)
	second := invoke(t, dispatcher, "orquesta.intakes.apply", "request:intake-replay", payload, false)
	if first.Failure != nil || second.Failure != nil ||
		string(first.Data) != string(second.Data) || api.calls["ApplyIntake"] != 2 {
		t.Fatalf("first=%+v second=%+v calls=%d", first, second, api.calls["ApplyIntake"])
	}
	var output struct {
		Intake struct {
			ReceiptRef string `json:"receipt_ref"`
		} `json:"intake"`
	}
	if err := json.Unmarshal(first.Data, &output); err != nil ||
		output.Intake.ReceiptRef != "intake-receipt:apply" {
		t.Fatalf("mutation receipt=%s err=%v", first.Data, err)
	}
}

func TestIntakeErrorsPreservePublicConflictInvalidAndNotFoundClasses(t *testing.T) {
	tests := []struct {
		err  error
		code string
	}{
		{&intake.DomainError{Code: intake.ErrorRevisionConflict, Field: "change.expected_revision"}, CodeConflict},
		{&intake.DomainError{Code: intake.ErrorInvalidRef, Field: "state_ref"}, CodeInvalidRequest},
		{&application.StateError{Code: application.StateNotFound}, CodeNotFound},
	}
	for _, test := range tests {
		failure := applicationFailure(normalizeIntakeError(test.err))
		if failure == nil || failure.Code != test.code {
			t.Errorf("error=%v failure=%+v want=%s", test.err, failure, test.code)
		}
	}
}

func TestIntakeMutationsRejectOversizedRequestRefBeforeApplicationWriter(t *testing.T) {
	dispatcher, api, audit := testDispatcher(t)
	requestRef := strings.Repeat("r", 513)
	for _, test := range []struct {
		commandID string
		payload   map[string]any
		call      string
	}{
		{
			commandID: "orquesta.intakes.create",
			payload:   map[string]any{"intake_ref": "intake:test", "max_question_rounds": 2},
			call:      "CreateIntake",
		},
		{
			commandID: "orquesta.intakes.apply",
			payload:   canonicalIntakeApplyPayload(),
			call:      "ApplyIntake",
		},
	} {
		result := invoke(t, dispatcher, test.commandID, requestRef, test.payload, false)
		if result.Failure == nil || result.Failure.Code != CodeInvalidRequest ||
			api.calls[test.call] != 0 {
			t.Errorf("%s result=%+v writer calls=%d", test.commandID, result, api.calls[test.call])
		}
	}
	if audit.admits != 2 {
		t.Fatalf("audit admissions=%d", audit.admits)
	}
}

func canonicalIntakeApplyPayload() map[string]any {
	return map[string]any{
		"intake_ref": "intake:test", "expected_revision": 1, "origin": "chat",
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
					"ref": "intake-option:audience-team", "label_key": "intake.option.audience.team.label",
					"rationale_key": "intake.option.audience.team.rationale", "recommended": true,
				},
				map[string]any{
					"ref": "intake-option:audience-personal", "label_key": "intake.option.audience.personal.label",
					"rationale_key": "intake.option.audience.personal.rationale", "recommended": false,
				},
			},
		}},
		"choices": []any{},
	}
}
