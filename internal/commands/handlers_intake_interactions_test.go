package commands

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/intake"
)

func (api *fakeApplication) AcceptIntakeRecommendations(
	_ context.Context,
	_ application.Access,
	request application.AcceptIntakeRecommendationsRequest,
) (application.IntakeResult, error) {
	api.called("AcceptIntakeRecommendations")
	state, err := intake.NewState(
		request.StateRef,
		intake.Policy{MaxQuestionRounds: 6},
	)
	return application.IntakeResult{
		Record: application.IntakeRecord{
			ActorRef:   request.ActorRef,
			ProjectRef: request.ProjectRef,
			State:      state,
			Receipt: application.IntakeReceipt{
				Ref: "intake-receipt:recommendations",
			},
		},
		Changed: true,
	}, err
}

func (api *fakeApplication) GetIntakeContext(
	_ context.Context,
	_ application.Access,
	request application.GetIntakeContextRequest,
) (intake.Context, error) {
	api.called("GetIntakeContext")
	questionRef := intake.QuestionRef("intake-question:audience")
	recommendation := intake.OptionRef("intake-option:audience-team")
	return intake.Context{
		StateRef: request.Context.StateRef,
		Revision: request.Context.ExpectedRevision,
		Origin:   request.Context.Origin,
		Kind:     request.Context.Kind,
		Issues: []intake.Issue{{
			Ref: "intake-issue:audience", Kind: intake.IssueGap,
			Field: "audience", DetailKey: "intake.issue.audience.missing",
		}},
		Questions: []intake.QuestionContext{{
			Question: intake.Question{
				Ref:         questionRef,
				DerivedFrom: []intake.IssueRef{"intake-issue:audience"},
				PromptKey:   "intake.question.audience.prompt",
				WhyKey:      "intake.question.audience.why",
				Options: []intake.Option{{
					Ref:          recommendation,
					LabelKey:     "intake.option.audience.team.label",
					RationaleKey: "intake.option.audience.team.rationale",
					Recommended:  true,
				}},
			},
			ReopenedDecision: &intake.ReopenedDecision{
				QuestionRef: questionRef,
				PreviousDecision: intake.Decision{
					QuestionRef:             questionRef,
					Choice:                  recommendation,
					Recommendation:          recommendation,
					RecommendationRationale: "intake.option.audience.team.rationale",
					Origin:                  intake.OriginChat,
					Revision:                request.Context.ExpectedRevision - 1,
				},
				InvalidatedBy: []intake.DecisionChange{{
					QuestionRef: "intake-question:root",
					Revision:    request.Context.ExpectedRevision,
				}},
			},
		}},
	}, nil
}

type captureIntakeInteractionsApplication struct {
	*fakeApplication
	accepts  []application.AcceptIntakeRecommendationsRequest
	contexts []application.GetIntakeContextRequest
}

func (api *captureIntakeInteractionsApplication) AcceptIntakeRecommendations(
	ctx context.Context,
	access application.Access,
	request application.AcceptIntakeRecommendationsRequest,
) (application.IntakeResult, error) {
	api.accepts = append(api.accepts, request)
	return api.fakeApplication.AcceptIntakeRecommendations(ctx, access, request)
}

func (api *captureIntakeInteractionsApplication) GetIntakeContext(
	ctx context.Context,
	access application.Access,
	request application.GetIntakeContextRequest,
) (intake.Context, error) {
	api.contexts = append(api.contexts, request)
	return api.fakeApplication.GetIntakeContext(ctx, access, request)
}

func TestIntakeInteractionCommandsRouteBindAuthorityAndProjectStableOutput(t *testing.T) {
	api := &captureIntakeInteractionsApplication{
		fakeApplication: newFakeApplication(),
	}
	dispatcher, err := newDispatcher(
		api,
		newMemoryAudit(),
		APILimits{
			MaxRequestBytes:         1 << 20,
			MaxListLimit:            100,
			IntakeMaxQuestionRounds: 6,
		},
		exactTestExecutionAuthority(t),
	)
	if err != nil {
		t.Fatal(err)
	}

	accepted := invoke(
		t,
		dispatcher,
		"orquesta.intakes.recommendations.accept",
		"request:intake-recommendations",
		map[string]any{
			"intake_ref":        "intake:test",
			"expected_revision": 2,
			"origin":            "chat",
			"question_round":    1,
		},
		false,
	)
	contextResult := invoke(
		t,
		dispatcher,
		"orquesta.intakes.context.get",
		"request:intake-context",
		map[string]any{
			"intake_ref":        "intake:test",
			"expected_revision": 2,
			"origin":            "form",
			"kind":              "help",
		},
		false,
	)
	if accepted.Failure != nil || contextResult.Failure != nil {
		t.Fatalf("accepted=%+v context=%+v", accepted, contextResult)
	}
	if len(api.accepts) != 1 || len(api.contexts) != 1 {
		t.Fatalf("requests=%d/%d", len(api.accepts), len(api.contexts))
	}
	if got := api.accepts[0]; got.RequestRef != "request:intake-recommendations" ||
		got.ActorRef.String() != "actor:test" ||
		got.ProjectRef.String() != "project:test" ||
		got.StateRef != "intake:test" ||
		got.ExpectedRevision != 2 ||
		got.Origin != intake.OriginChat ||
		got.QuestionRound != 1 {
		t.Fatalf("accept request=%+v", got)
	}
	if got := api.contexts[0]; got.ActorRef.String() != "actor:test" ||
		got.ProjectRef.String() != "project:test" ||
		got.Context.StateRef != "intake:test" ||
		got.Context.ExpectedRevision != 2 ||
		got.Context.Origin != intake.OriginForm ||
		got.Context.Kind != intake.ContextHelp ||
		got.Context.QuestionRefs == nil ||
		len(got.Context.QuestionRefs) != 0 {
		t.Fatalf("context request=%+v", got)
	}

	var mutation struct {
		Intake intakeMutationView `json:"intake"`
	}
	if err := json.Unmarshal(accepted.Data, &mutation); err != nil {
		t.Fatal(err)
	}
	if mutation.Intake.IntakeRef != "intake:test" ||
		mutation.Intake.ProjectRef != "project:test" ||
		mutation.Intake.Revision != 1 ||
		mutation.Intake.ReceiptRef != "intake-receipt:recommendations" {
		t.Fatalf("mutation=%s", accepted.Data)
	}
	var projected struct {
		Context struct {
			Issues    []json.RawMessage `json:"issues"`
			Questions []struct {
				Question struct {
					DerivedFrom []string          `json:"derived_from"`
					DependsOn   []string          `json:"depends_on"`
					Options     []json.RawMessage `json:"options"`
				} `json:"question"`
				ReopenedDecision struct {
					InvalidatedBy []json.RawMessage `json:"invalidated_by"`
				} `json:"reopened_decision"`
			} `json:"questions"`
		} `json:"context"`
	}
	if err := json.Unmarshal(contextResult.Data, &projected); err != nil {
		t.Fatal(err)
	}
	if projected.Context.Issues == nil ||
		len(projected.Context.Questions) != 1 ||
		projected.Context.Questions[0].Question.DerivedFrom == nil ||
		projected.Context.Questions[0].Question.DependsOn == nil ||
		projected.Context.Questions[0].Question.Options == nil ||
		projected.Context.Questions[0].ReopenedDecision.InvalidatedBy == nil {
		t.Fatalf("unstable context arrays=%s", contextResult.Data)
	}
	if api.calls["ApplyIntake"] != 0 ||
		api.calls["CreateIntake"] != 0 ||
		api.calls["AcceptIntakeRecommendations"] != 1 ||
		api.calls["GetIntakeContext"] != 1 {
		t.Fatalf("unexpected mutations/calls=%v", api.calls)
	}
}

func TestIntakeInteractionSchemasRejectAuthoritySpoofingBeforeApplication(t *testing.T) {
	api := &captureIntakeInteractionsApplication{
		fakeApplication: newFakeApplication(),
	}
	audit := newMemoryAudit()
	dispatcher, err := newDispatcher(
		api,
		audit,
		APILimits{
			MaxRequestBytes:         1 << 20,
			MaxListLimit:            100,
			IntakeMaxQuestionRounds: 6,
		},
		exactTestExecutionAuthority(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		commandID string
		payload   map[string]any
	}{
		{
			commandID: "orquesta.intakes.recommendations.accept",
			payload: map[string]any{
				"intake_ref":        "intake:test",
				"expected_revision": 2,
				"origin":            "chat",
				"question_round":    1,
				"actor_ref":         "actor:spoof",
			},
		},
		{
			commandID: "orquesta.intakes.context.get",
			payload: map[string]any{
				"intake_ref":        "intake:test",
				"expected_revision": 2,
				"origin":            "form",
				"kind":              "clarification",
				"project_ref":       "project:spoof",
			},
		},
		{
			commandID: "orquesta.intakes.context.get",
			payload: map[string]any{
				"intake_ref":          "intake:test",
				"expected_revision":   2,
				"origin":              "form",
				"kind":                "help",
				"request_ref":         "request:spoof",
				"request_fingerprint": "sha256:spoof",
			},
		},
	}
	for index, test := range tests {
		result := invoke(
			t,
			dispatcher,
			test.commandID,
			"request:spoof:"+string(rune('a'+index)),
			test.payload,
			false,
		)
		if result.Failure == nil || result.Failure.Code != CodeInvalidRequest {
			t.Errorf("%s=%+v", test.commandID, result)
		}
	}
	if len(api.accepts) != 0 ||
		len(api.contexts) != 0 ||
		audit.admits != 0 {
		t.Fatalf(
			"writers=%d readers=%d admissions=%d",
			len(api.accepts),
			len(api.contexts),
			audit.admits,
		)
	}
}

func TestProjectIntakeContextDefensivelyCopiesArrays(t *testing.T) {
	source := intake.Context{
		StateRef: "intake:test",
		Revision: 2,
		Origin:   intake.OriginChat,
		Kind:     intake.ContextHelp,
		Issues: []intake.Issue{{
			Ref:       "intake-issue:root",
			Kind:      intake.IssueGap,
			Field:     "root",
			DetailKey: "intake.issue.root",
		}},
		Questions: []intake.QuestionContext{{
			Question: intake.Question{
				Ref:         "intake-question:root",
				DerivedFrom: []intake.IssueRef{"intake-issue:root"},
				DependsOn:   []intake.QuestionRef{},
				PromptKey:   "intake.question.root.prompt",
				WhyKey:      "intake.question.root.why",
				Options: []intake.Option{{
					Ref:          "intake-option:root",
					LabelKey:     "intake.option.root.label",
					RationaleKey: "intake.option.root.rationale",
					Recommended:  true,
				}},
			},
		}},
	}
	view := projectIntakeContext(source)
	view.Issues[0].Field = "changed"
	view.Questions[0].Question.DerivedFrom[0] = "intake-issue:changed"
	view.Questions[0].Question.Options[0].Ref = "intake-option:changed"
	if source.Issues[0].Field != "root" ||
		!reflect.DeepEqual(
			source.Questions[0].Question.DerivedFrom,
			[]intake.IssueRef{"intake-issue:root"},
		) ||
		source.Questions[0].Question.Options[0].Ref != "intake-option:root" {
		t.Fatalf("projection mutated source=%+v", source)
	}
}
