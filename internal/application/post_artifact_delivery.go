package application

import (
	"context"
	"errors"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

type PostArtifactMailboxAdmissionRequest struct {
	Session                ports.ExecutionSessionEnsureRequest
	RequestRef             string
	GoalRef                goal.GoalRef
	ExpectedPlanGeneration goal.PlanGeneration
	ParentWorkItemRef      goal.WorkItemRef
	ChildWorkItemRef       goal.WorkItemRef
	RecipientPrincipalRef  identity.PrincipalRef
	RecipientExecutionRef  goal.ExecutionRef
	Summary                string
	ArtifactRefs           []goal.ArtifactRef
}

type PostArtifactMailboxAdmissionReceipt struct {
	GoalRef      goal.GoalRef
	MessageRef   MailboxMessageRef
	AdmissionRef string
}

type PostArtifactMailboxAdmitter interface {
	AdmitPostArtifactMailbox(context.Context, PostArtifactMailboxAdmissionRequest) (PostArtifactMailboxAdmissionReceipt, error)
}

func postArtifactMailboxAction(aggregate goal.Goal, item goal.WorkItem, execution ExecutionRecord, at time.Time) (*ActionRecord, error) {
	if !item.HandoffRequired() {
		return nil, nil
	}
	if _, ok := item.Parent(); !ok || item.State() != goal.WorkItemStateSucceeded ||
		execution.State != ExecutionSucceeded {
		return nil, errors.New("application.post_artifact_delivery_invalid")
	}
	action := &ActionRecord{
		Ref: "action:admit-mailbox:" + execution.Ref.String(), Kind: ActionAdmitMailbox,
		GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref,
		PlanGeneration: execution.PlanGeneration, WorkItemGeneration: item.Revision(), AvailableAt: at.UTC(),
	}
	return action, nil
}

func (orchestrator *Orchestrator) processPostArtifactMailboxAdmission(
	ctx context.Context,
	claim ActionClaim,
) error {
	if orchestrator.postArtifactMailbox == nil || orchestrator.executionSessions == nil {
		return orchestrator.requeuePostArtifactMailbox(ctx, claim, ExecutionRecord{}, "application.post_artifact_mailbox_unavailable")
	}
	record, err := orchestrator.state.GetGoal(ctx, claim.Action.GoalRef)
	if err != nil {
		return err
	}
	child, childFound := record.Goal.WorkItem(claim.Action.WorkItemRef)
	execution, executionFound := executionForAction(record, claim.Action)
	parentRef, hasParent := child.Parent()
	parent, parentFound := record.Goal.WorkItem(parentRef)
	if !childFound || !executionFound || !hasParent || !parentFound ||
		claim.Action.Kind != ActionAdmitMailbox || claim.Action.Ref != "action:admit-mailbox:"+execution.Ref.String() ||
		child.State() != goal.WorkItemStateSucceeded || !child.HandoffRequired() || execution.State != ExecutionSucceeded ||
		parent.State() != goal.WorkItemStateRunning {
		return orchestrator.quarantine(ctx, claim, "application.post_artifact_mailbox_invalid")
	}
	parentExecutionRef, hasParentExecution := parent.Execution()
	parentExecution, found := executionByRef(record.Executions, parentExecutionRef)
	if !hasParentExecution || !found || parentExecution.State != ExecutionRunning {
		return orchestrator.requeuePostArtifactMailbox(ctx, claim, execution, "application.post_artifact_parent_unavailable")
	}
	sourceRequest := ExecutionSessionRequest(record.Goal, execution)
	sourceSession, err := orchestrator.executionSessions.Ensure(ctx, sourceRequest)
	if err != nil {
		return orchestrator.requeuePostArtifactMailbox(ctx, claim, execution, "application.execution_session_unavailable")
	}
	method := sourceSession.Authority.ServicePrincipal.Method
	sourceAuthority, err := DeriveExecutionSessionAuthority(sourceRequest, method)
	if err != nil || sourceAuthority != sourceSession.Authority {
		return orchestrator.quarantine(ctx, claim, "application.execution_session_invalid")
	}
	parentAuthority, err := DeriveExecutionSessionAuthority(
		ExecutionSessionRequest(record.Goal, parentExecution), method,
	)
	if err != nil {
		return orchestrator.quarantine(ctx, claim, "application.execution_session_invalid")
	}
	request := PostArtifactMailboxAdmissionRequest{
		Session: sourceRequest, RequestRef: "request:post-artifact-mailbox:" + execution.Ref.String(),
		GoalRef: record.Goal.Ref(), ExpectedPlanGeneration: record.Goal.PlanGeneration(),
		ParentWorkItemRef: parent.Ref(), ChildWorkItemRef: child.Ref(),
		RecipientPrincipalRef: parentAuthority.ServicePrincipal.Ref,
		RecipientExecutionRef: parentExecution.Ref, Summary: child.Objective(), ArtifactRefs: child.Artifacts(),
	}
	receipt, err := orchestrator.postArtifactMailbox.AdmitPostArtifactMailbox(ctx, request)
	if err != nil {
		return orchestrator.requeuePostArtifactMailbox(ctx, claim, execution, "application.post_artifact_mailbox_unavailable")
	}
	if receipt.GoalRef != request.GoalRef || receipt.MessageRef.String() == "" || receipt.AdmissionRef == "" {
		return orchestrator.quarantine(ctx, claim, "application.post_artifact_mailbox_receipt_invalid")
	}
	return orchestrator.state.RecordPostArtifactMailboxAdmitted(ctx, PostArtifactMailboxAdmittedState{
		Claim: claim, MessageRef: receipt.MessageRef, AdmissionRef: receipt.AdmissionRef, OperationAt: orchestrator.clock.Now().UTC(),
	})
}

func (orchestrator *Orchestrator) requeuePostArtifactMailbox(ctx context.Context, claim ActionClaim, execution ExecutionRecord, code string) error {
	if execution.Ref.String() == "" {
		record, err := orchestrator.state.GetGoal(ctx, claim.Action.GoalRef)
		if err != nil {
			return err
		}
		execution, _ = executionForAction(record, claim.Action)
	}
	now := orchestrator.clock.Now().UTC()
	return orchestrator.state.RequeueAction(ctx, ActionRequeuedState{
		Claim: claim, Execution: execution, ErrorCode: code, AvailableAt: now.Add(orchestrator.observationDelay), OperationAt: now,
	})
}
