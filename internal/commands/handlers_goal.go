package commands

import (
	"context"
	"encoding/json"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

func handleSubmit(ctx context.Context, api applicationAPI, bound handlerContext, payload json.RawMessage) (json.RawMessage, error) {
	var input struct {
		Statement           string     `json:"statement"`
		NormalizedObjective string     `json:"normalized_objective,omitempty"`
		Confirm             bool       `json:"confirm"`
		Plan                *planInput `json:"plan,omitempty"`
	}
	if err := decodePayload(payload, &input); err != nil {
		return nil, err
	}
	result, err := api.Submit(ctx, bound.access, application.SubmitRequest{RequestRef: bound.requestRef, Statement: input.Statement, NormalizedObjective: input.NormalizedObjective, Confirm: input.Confirm, Plan: applicationPlan(input.Plan)})
	return marshalApplication(struct {
		Goal goalReceiptView `json:"goal"`
	}{projectGoalReceipt(result.Record.Goal)}, err)
}

func handleAmend(ctx context.Context, api applicationAPI, bound handlerContext, payload json.RawMessage) (json.RawMessage, error) {
	var input struct {
		SourceGoalRef          string `json:"source_goal_ref"`
		ExpectedSourceRevision uint64 `json:"expected_source_revision"`
		ExpectedSourceSpecHash string `json:"expected_source_spec_hash"`
		Statement              string `json:"statement"`
		NormalizedObjective    string `json:"normalized_objective,omitempty"`
		Reason                 string `json:"reason"`
		Confirm                bool   `json:"confirm"`
	}
	if err := decodePayload(payload, &input); err != nil {
		return nil, err
	}
	ref, err := goal.NewGoalRef(input.SourceGoalRef)
	if err != nil {
		return nil, err
	}
	result, err := api.Amend(ctx, bound.access, application.AmendRequest{RequestRef: bound.requestRef, SourceGoalRef: ref, ExpectedSourceRevision: goal.Revision(input.ExpectedSourceRevision), ExpectedSourceSpecHash: input.ExpectedSourceSpecHash, Statement: input.Statement, NormalizedObjective: input.NormalizedObjective, Reason: input.Reason, Confirm: input.Confirm})
	return marshalApplication(struct {
		Goal goalReceiptView `json:"goal"`
	}{projectGoalReceipt(result.Record.Goal)}, err)
}

func handleGetGoal(ctx context.Context, api applicationAPI, bound handlerContext, payload json.RawMessage) (json.RawMessage, error) {
	var input struct {
		GoalRef string `json:"goal_ref"`
	}
	if err := decodePayload(payload, &input); err != nil {
		return nil, err
	}
	ref, err := goal.NewGoalRef(input.GoalRef)
	if err != nil {
		return nil, err
	}
	record, err := api.GetGoal(ctx, bound.access, ref)
	return marshalApplication(projectGoalRecord(record), err)
}

func handleListGoals(ctx context.Context, api applicationAPI, bound handlerContext, payload json.RawMessage) (json.RawMessage, error) {
	var input struct {
		Limit int `json:"limit"`
	}
	if err := decodePayload(payload, &input); err != nil {
		return nil, err
	}
	values, err := api.ListGoals(ctx, bound.access, input.Limit)
	projected := make([]goalSummaryView, 0, len(values))
	for _, value := range values {
		projected = append(projected, projectGoalSummary(value))
	}
	return marshalApplication(struct {
		Goals []goalSummaryView `json:"goals"`
	}{Goals: projected}, err)
}

func handleGetArtifact(ctx context.Context, api applicationAPI, bound handlerContext, payload json.RawMessage) (json.RawMessage, error) {
	var input struct {
		GoalRef     string `json:"goal_ref"`
		ArtifactRef string `json:"artifact_ref"`
	}
	if err := decodePayload(payload, &input); err != nil {
		return nil, err
	}
	goalRef, err := goal.NewGoalRef(input.GoalRef)
	if err != nil {
		return nil, err
	}
	artifactRef, err := goal.NewArtifactRef(input.ArtifactRef)
	if err != nil {
		return nil, err
	}
	value, err := api.GetArtifact(ctx, bound.access, goalRef, artifactRef)
	return marshalApplication(projectArtifact(value), err)
}

func handleStatus(ctx context.Context, api applicationAPI, bound handlerContext, payload json.RawMessage) (json.RawMessage, error) {
	var input struct{}
	if err := decodePayload(payload, &input); err != nil {
		return nil, err
	}
	value, err := api.Status(ctx, bound.access)
	return marshalApplication(struct {
		Goals              int64 `json:"goals"`
		RunningGoals       int64 `json:"running_goals"`
		PendingActions     int64 `json:"pending_actions"`
		QuarantinedActions int64 `json:"quarantined_actions"`
	}{value.Goals, value.RunningGoals, value.PendingActions, value.QuarantinedActions}, err)
}

func handleGrantMembership(ctx context.Context, api applicationAPI, bound handlerContext, payload json.RawMessage) (json.RawMessage, error) {
	var input struct {
		TargetPrincipalRef string `json:"target_principal_ref"`
		TargetActorRef     string `json:"target_actor_ref"`
		TargetKind         string `json:"target_kind"`
		TargetMethod       string `json:"target_method"`
		Role               string `json:"role"`
		ExpectedRevision   uint64 `json:"expected_revision"`
	}
	if err := decodePayload(payload, &input); err != nil {
		return nil, err
	}
	targetRef, err := identity.NewPrincipalRef(input.TargetPrincipalRef)
	if err != nil {
		return nil, err
	}
	actorRef, err := goal.NewActorRef(input.TargetActorRef)
	if err != nil {
		return nil, err
	}
	target, err := identity.NewPrincipal(targetRef, actorRef, identity.PrincipalKind(input.TargetKind), input.TargetMethod)
	if err != nil {
		return nil, err
	}
	request, err := identity.NewMembershipGrantRequest(identity.MembershipGrantRequestInput{RequestRef: bound.requestRef, Actor: bound.principal, TargetRef: targetRef, ProjectRef: bound.projectRef, Role: identity.Role(input.Role), ExpectedRevision: identity.MembershipRevision(input.ExpectedRevision), RequestedAt: bound.requestedAt})
	if err != nil {
		return nil, err
	}
	_, audit, _, err := api.GrantMembership(ctx, bound.access, request, target)
	return marshalApplication(struct {
		Receipt membershipAuditView `json:"receipt"`
	}{projectMembershipAudit(audit)}, err)
}

func handleRevokeMembership(ctx context.Context, api applicationAPI, bound handlerContext, payload json.RawMessage) (json.RawMessage, error) {
	var input struct {
		TargetPrincipalRef string `json:"target_principal_ref"`
		ExpectedRevision   uint64 `json:"expected_revision"`
	}
	if err := decodePayload(payload, &input); err != nil {
		return nil, err
	}
	targetRef, err := identity.NewPrincipalRef(input.TargetPrincipalRef)
	if err != nil {
		return nil, err
	}
	request, err := identity.NewMembershipRevokeRequest(identity.MembershipRevokeRequestInput{RequestRef: bound.requestRef, Actor: bound.principal, TargetRef: targetRef, ProjectRef: bound.projectRef, ExpectedRevision: identity.MembershipRevision(input.ExpectedRevision), RequestedAt: bound.requestedAt})
	if err != nil {
		return nil, err
	}
	_, audit, _, err := api.RevokeMembership(ctx, bound.access, request)
	return marshalApplication(struct {
		Receipt membershipAuditView `json:"receipt"`
	}{projectMembershipAudit(audit)}, err)
}
