package commands

import (
	"context"
	"encoding/json"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

func handleAdmitMailbox(ctx context.Context, api applicationAPI, bound handlerContext, payload json.RawMessage) (json.RawMessage, error) {
	var input struct {
		GoalRef                string   `json:"goal_ref"`
		ExpectedPlanGeneration uint64   `json:"expected_plan_generation"`
		Kind                   string   `json:"kind"`
		ParentWorkItemRef      string   `json:"parent_work_item_ref"`
		ChildWorkItemRef       string   `json:"child_work_item_ref"`
		RecipientPrincipalRef  string   `json:"recipient_principal_ref"`
		RecipientExecutionRef  string   `json:"recipient_execution_ref"`
		Summary                string   `json:"summary"`
		ArtifactRefs           []string `json:"artifact_refs"`
	}
	if err := decodePayload(payload, &input); err != nil {
		return nil, err
	}
	goalRef, err := goal.NewGoalRef(input.GoalRef)
	if err != nil {
		return nil, err
	}
	parent, err := goal.NewWorkItemRef(input.ParentWorkItemRef)
	if err != nil {
		return nil, err
	}
	child, err := goal.NewWorkItemRef(input.ChildWorkItemRef)
	if err != nil {
		return nil, err
	}
	recipientPrincipal, err := identity.NewPrincipalRef(input.RecipientPrincipalRef)
	if err != nil {
		return nil, err
	}
	recipientExecution, err := goal.NewExecutionRef(input.RecipientExecutionRef)
	if err != nil {
		return nil, err
	}
	artifacts := make([]goal.ArtifactRef, 0, len(input.ArtifactRefs))
	for _, raw := range input.ArtifactRefs {
		ref, parseErr := goal.NewArtifactRef(raw)
		if parseErr != nil {
			return nil, parseErr
		}
		artifacts = append(artifacts, ref)
	}
	result, err := api.AdmitMailbox(ctx, bound.access, application.AdmitMailboxRequest{RequestRef: bound.requestRef, GoalRef: goalRef, ExpectedPlanGeneration: goal.PlanGeneration(input.ExpectedPlanGeneration), Kind: application.MailboxKind(input.Kind), ParentWorkItemRef: parent, ChildWorkItemRef: child, SourceExecutionRef: bound.executionRef, RecipientPrincipalRef: recipientPrincipal, RecipientExecutionRef: recipientExecution, Summary: input.Summary, ArtifactRefs: artifacts})
	return marshalApplication(struct {
		Receipt mailboxAdmissionView `json:"receipt"`
	}{projectMailboxAdmission(result.Record)}, err)
}

type mailboxAddressInput struct {
	GoalRef              string `json:"goal_ref"`
	MessageRef           string `json:"message_ref"`
	RecipientWorkItemRef string `json:"recipient_work_item_ref"`
}
type mailboxClaimInput struct {
	GoalRef              string `json:"goal_ref"`
	MessageRef           string `json:"message_ref"`
	RecipientWorkItemRef string `json:"recipient_work_item_ref"`
	ClaimToken           string `json:"claim_token"`
	Fence                uint64 `json:"fence"`
}

func mailboxAddress(input mailboxAddressInput, execution goal.ExecutionRef) (goal.GoalRef, application.MailboxMessageRef, goal.WorkItemRef, error) {
	goalRef, err := goal.NewGoalRef(input.GoalRef)
	if err != nil {
		return goal.GoalRef{}, application.MailboxMessageRef{}, goal.WorkItemRef{}, err
	}
	messageRef, err := application.NewMailboxMessageRef(input.MessageRef)
	if err != nil {
		return goal.GoalRef{}, application.MailboxMessageRef{}, goal.WorkItemRef{}, err
	}
	workRef, err := goal.NewWorkItemRef(input.RecipientWorkItemRef)
	return goalRef, messageRef, workRef, err
}

func handleClaimMailbox(ctx context.Context, api applicationAPI, bound handlerContext, payload json.RawMessage) (json.RawMessage, error) {
	var input mailboxAddressInput
	if err := decodePayload(payload, &input); err != nil {
		return nil, err
	}
	goalRef, messageRef, workRef, err := mailboxAddress(input, bound.executionRef)
	if err != nil {
		return nil, err
	}
	result, err := api.ClaimMailbox(ctx, bound.access, application.ClaimMailboxRequest{RequestRef: bound.requestRef, GoalRef: goalRef, MessageRef: messageRef, RecipientWorkItemRef: workRef, RecipientExecutionRef: bound.executionRef})
	return marshalApplication(struct {
		Receipt mailboxClaimView `json:"receipt"`
	}{projectMailboxClaim(result.Claim)}, err)
}
func handleMarkMailboxDelivered(ctx context.Context, api applicationAPI, bound handlerContext, payload json.RawMessage) (json.RawMessage, error) {
	var input mailboxClaimInput
	if err := decodePayload(payload, &input); err != nil {
		return nil, err
	}
	goalRef, messageRef, workRef, err := mailboxAddress(mailboxAddressInput{input.GoalRef, input.MessageRef, input.RecipientWorkItemRef}, bound.executionRef)
	if err != nil {
		return nil, err
	}
	result, err := api.MarkMailboxDelivered(ctx, bound.access, application.MarkMailboxDeliveredRequest{RequestRef: bound.requestRef, GoalRef: goalRef, MessageRef: messageRef, RecipientWorkItemRef: workRef, RecipientExecutionRef: bound.executionRef, ClaimToken: input.ClaimToken, Fence: input.Fence})
	return marshalApplication(struct {
		Receipt mailboxMutationReceiptView `json:"receipt"`
	}{projectMailboxMutationReceipt(result.Record, input.ClaimToken, input.Fence, false)}, err)
}
func handleConsumeMailbox(ctx context.Context, api applicationAPI, bound handlerContext, payload json.RawMessage) (json.RawMessage, error) {
	var input mailboxClaimInput
	if err := decodePayload(payload, &input); err != nil {
		return nil, err
	}
	goalRef, messageRef, workRef, err := mailboxAddress(mailboxAddressInput{input.GoalRef, input.MessageRef, input.RecipientWorkItemRef}, bound.executionRef)
	if err != nil {
		return nil, err
	}
	result, err := api.ConsumeMailbox(ctx, bound.access, application.ConsumeMailboxRequest{RequestRef: bound.requestRef, GoalRef: goalRef, MessageRef: messageRef, RecipientWorkItemRef: workRef, RecipientExecutionRef: bound.executionRef, ClaimToken: input.ClaimToken, Fence: input.Fence})
	return marshalApplication(struct {
		Receipt mailboxMutationReceiptView `json:"receipt"`
	}{projectMailboxMutationReceipt(result.Record, input.ClaimToken, input.Fence, true)}, err)
}
func handleGetMailbox(ctx context.Context, api applicationAPI, bound handlerContext, payload json.RawMessage) (json.RawMessage, error) {
	var input mailboxAddressInput
	if err := decodePayload(payload, &input); err != nil {
		return nil, err
	}
	goalRef, messageRef, workRef, err := mailboxAddress(input, bound.executionRef)
	if err != nil {
		return nil, err
	}
	record, err := api.GetMailbox(ctx, bound.access, application.GetMailboxRequest{GoalRef: goalRef, MessageRef: messageRef, RecipientWorkItemRef: workRef, RecipientExecutionRef: bound.executionRef})
	return marshalApplication(projectMailbox(record), err)
}
func handleListMailbox(ctx context.Context, api applicationAPI, bound handlerContext, payload json.RawMessage) (json.RawMessage, error) {
	var input struct {
		GoalRef              string `json:"goal_ref"`
		RecipientWorkItemRef string `json:"recipient_work_item_ref"`
		Limit                int    `json:"limit"`
	}
	if err := decodePayload(payload, &input); err != nil {
		return nil, err
	}
	goalRef, err := goal.NewGoalRef(input.GoalRef)
	if err != nil {
		return nil, err
	}
	workRef, err := goal.NewWorkItemRef(input.RecipientWorkItemRef)
	if err != nil {
		return nil, err
	}
	values, err := api.ListMailbox(ctx, bound.access, application.ListMailboxRequest{GoalRef: goalRef, RecipientWorkItemRef: workRef, RecipientExecutionRef: bound.executionRef, Limit: input.Limit})
	messages := make([]mailboxView, 0, len(values))
	for _, value := range values {
		messages = append(messages, projectMailbox(value))
	}
	return marshalApplication(struct {
		Messages []mailboxView `json:"messages"`
	}{Messages: messages}, err)
}

func mailboxResolution(payload json.RawMessage, bound handlerContext) (application.ResolveMailboxRequest, error) {
	var input struct {
		GoalRef                string `json:"goal_ref"`
		MessageRef             string `json:"message_ref"`
		RecipientWorkItemRef   string `json:"recipient_work_item_ref"`
		ClaimToken             string `json:"claim_token"`
		Fence                  uint64 `json:"fence"`
		ExpectedGoalRevision   uint64 `json:"expected_goal_revision"`
		ExpectedPlanGeneration uint64 `json:"expected_plan_generation"`
		EffectOrReworkRef      string `json:"effect_or_rework_ref"`
	}
	if err := decodePayload(payload, &input); err != nil {
		return application.ResolveMailboxRequest{}, err
	}
	goalRef, messageRef, workRef, err := mailboxAddress(mailboxAddressInput{input.GoalRef, input.MessageRef, input.RecipientWorkItemRef}, bound.executionRef)
	if err != nil {
		return application.ResolveMailboxRequest{}, err
	}
	return application.ResolveMailboxRequest{RequestRef: bound.requestRef, GoalRef: goalRef, MessageRef: messageRef, RecipientWorkItemRef: workRef, RecipientExecutionRef: bound.executionRef, ClaimToken: input.ClaimToken, Fence: input.Fence, ExpectedGoalRevision: goal.Revision(input.ExpectedGoalRevision), ExpectedPlanGeneration: goal.PlanGeneration(input.ExpectedPlanGeneration), EffectOrReworkRef: input.EffectOrReworkRef}, nil
}
func handleAcknowledgeMailbox(ctx context.Context, api applicationAPI, bound handlerContext, payload json.RawMessage) (json.RawMessage, error) {
	request, err := mailboxResolution(payload, bound)
	if err != nil {
		return nil, err
	}
	result, err := api.AcknowledgeMailbox(ctx, bound.access, application.AcknowledgeMailboxRequest(request))
	return marshalApplication(struct {
		AcknowledgementRef string `json:"acknowledgement_ref"`
		MessageRef         string `json:"message_ref"`
		Outcome            string `json:"outcome"`
	}{result.Acknowledgement.Ref, result.Acknowledgement.MessageRef.String(), string(result.Acknowledgement.Outcome)}, err)
}
func handleBlockMailbox(ctx context.Context, api applicationAPI, bound handlerContext, payload json.RawMessage) (json.RawMessage, error) {
	request, err := mailboxResolution(payload, bound)
	if err != nil {
		return nil, err
	}
	result, err := api.BlockMailbox(ctx, bound.access, application.BlockMailboxRequest(request))
	return marshalApplication(struct {
		AcknowledgementRef string `json:"acknowledgement_ref"`
		MessageRef         string `json:"message_ref"`
		Outcome            string `json:"outcome"`
	}{result.Acknowledgement.Ref, result.Acknowledgement.MessageRef.String(), string(result.Acknowledgement.Outcome)}, err)
}
