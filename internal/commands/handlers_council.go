package commands

import (
	"context"
	"encoding/json"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func handleOpenCouncilRound(
	ctx context.Context,
	api applicationAPI,
	bound handlerContext,
	payload json.RawMessage,
) (json.RawMessage, error) {
	var input struct {
		GoalRef              string `json:"goal_ref"`
		ChangeRef            string `json:"change_ref"`
		ExpectedGoalRevision uint64 `json:"expected_goal_revision"`
		ExpectedItemRevision uint64 `json:"expected_item_revision"`
		LeaseToken           string `json:"lease_token"`
		LeaseFence           uint64 `json:"lease_fence"`
	}
	if err := decodePayload(payload, &input); err != nil {
		return nil, err
	}
	goalRef, err := goal.NewGoalRef(input.GoalRef)
	if err != nil {
		return nil, err
	}
	changeRef, err := ports.NewChangeSetRef(input.ChangeRef)
	if err != nil {
		return nil, err
	}
	result, err := api.OpenCouncilRound(ctx, bound.access, application.OpenCouncilRoundRequest{
		RequestRef:           bound.requestRef,
		GoalRef:              goalRef,
		ChangeRef:            changeRef,
		ExpectedGoalRevision: goal.Revision(input.ExpectedGoalRevision),
		ExpectedItemRevision: goal.Revision(input.ExpectedItemRevision),
		LeaseToken:           input.LeaseToken,
		LeaseFence:           input.LeaseFence,
	})
	return marshalApplication(struct {
		Round councilRoundView `json:"round"`
	}{Round: projectCouncilRound(result.Round)}, err)
}

func handleSkipCouncil(
	ctx context.Context,
	api applicationAPI,
	bound handlerContext,
	payload json.RawMessage,
) (json.RawMessage, error) {
	var input struct {
		GoalRef              string `json:"goal_ref"`
		ChangeRef            string `json:"change_ref"`
		ExpectedGoalRevision uint64 `json:"expected_goal_revision"`
		ExpectedItemRevision uint64 `json:"expected_item_revision"`
		Reason               string `json:"reason"`
	}
	if err := decodePayload(payload, &input); err != nil {
		return nil, err
	}
	goalRef, err := goal.NewGoalRef(input.GoalRef)
	if err != nil {
		return nil, err
	}
	changeRef, err := ports.NewChangeSetRef(input.ChangeRef)
	if err != nil {
		return nil, err
	}
	result, err := api.SkipCouncil(ctx, bound.access, application.SkipCouncilRequest{
		RequestRef:           bound.requestRef,
		GoalRef:              goalRef,
		ChangeRef:            changeRef,
		ExpectedGoalRevision: goal.Revision(input.ExpectedGoalRevision),
		ExpectedItemRevision: goal.Revision(input.ExpectedItemRevision),
		Reason:               input.Reason,
	})
	return marshalApplication(struct {
		Skip councilSkipView `json:"skip"`
	}{Skip: projectCouncilSkip(result.Skip)}, err)
}
