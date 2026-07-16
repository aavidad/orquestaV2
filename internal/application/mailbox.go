package application

import (
	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

type AdmitMailboxRequest struct {
	RequestRef             string
	GoalRef                goal.GoalRef
	ExpectedPlanGeneration goal.PlanGeneration
	Kind                   MailboxKind
	ParentWorkItemRef      goal.WorkItemRef
	ChildWorkItemRef       goal.WorkItemRef
	SourceExecutionRef     goal.ExecutionRef
	RecipientPrincipalRef  identity.PrincipalRef
	RecipientExecutionRef  goal.ExecutionRef
	Summary                string
	ArtifactRefs           []goal.ArtifactRef
}

type ClaimMailboxRequest struct {
	RequestRef            string
	GoalRef               goal.GoalRef
	MessageRef            MailboxMessageRef
	RecipientWorkItemRef  goal.WorkItemRef
	RecipientExecutionRef goal.ExecutionRef
}

type MarkMailboxDeliveredRequest struct {
	RequestRef            string
	GoalRef               goal.GoalRef
	MessageRef            MailboxMessageRef
	RecipientWorkItemRef  goal.WorkItemRef
	RecipientExecutionRef goal.ExecutionRef
	ClaimToken            string
	Fence                 uint64
}

type ConsumeMailboxRequest struct {
	RequestRef            string
	GoalRef               goal.GoalRef
	MessageRef            MailboxMessageRef
	RecipientWorkItemRef  goal.WorkItemRef
	RecipientExecutionRef goal.ExecutionRef
	ClaimToken            string
	Fence                 uint64
}

type ResolveMailboxRequest struct {
	RequestRef             string
	GoalRef                goal.GoalRef
	MessageRef             MailboxMessageRef
	RecipientWorkItemRef   goal.WorkItemRef
	RecipientExecutionRef  goal.ExecutionRef
	ClaimToken             string
	Fence                  uint64
	ExpectedGoalRevision   goal.Revision
	ExpectedPlanGeneration goal.PlanGeneration
	EffectOrReworkRef      string
}

type AcknowledgeMailboxRequest ResolveMailboxRequest

type BlockMailboxRequest ResolveMailboxRequest

type GetMailboxRequest struct {
	GoalRef               goal.GoalRef
	MessageRef            MailboxMessageRef
	RecipientWorkItemRef  goal.WorkItemRef
	RecipientExecutionRef goal.ExecutionRef
}

type ListMailboxRequest struct {
	GoalRef               goal.GoalRef
	RecipientWorkItemRef  goal.WorkItemRef
	RecipientExecutionRef goal.ExecutionRef
	Limit                 int
}

type MailboxAdmissionResult struct {
	Record  MailboxRecord
	Created bool
}

type MailboxClaimResult struct {
	Claim   MailboxClaim
	Claimed bool
}

type MailboxMutationResult struct {
	Record  MailboxRecord
	Changed bool
}

type MailboxResolutionResult struct {
	Acknowledgement MailboxAcknowledgement
	Created         bool
}

type ListMailboxResult []MailboxRecord
