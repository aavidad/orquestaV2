package commands

import (
	"encoding/base64"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

type goalView struct {
	GoalRef        string `json:"goal_ref"`
	ProjectRef     string `json:"project_ref"`
	State          string `json:"state"`
	Revision       uint64 `json:"revision"`
	PlanGeneration uint64 `json:"plan_generation"`
	SpecHash       string `json:"spec_hash"`
	WorkItemCount  int    `json:"work_item_count"`
}

type goalReceiptView struct {
	GoalRef    string `json:"goal_ref"`
	ProjectRef string `json:"project_ref"`
	SpecHash   string `json:"spec_hash"`
}

func projectGoalReceipt(value goal.Goal) goalReceiptView {
	return goalReceiptView{GoalRef: value.Ref().String(), ProjectRef: value.Project().String(), SpecHash: value.SpecHash()}
}

type goalSummaryView struct {
	GoalRef           string `json:"goal_ref"`
	ProjectRef        string `json:"project_ref"`
	Statement         string `json:"statement"`
	State             string `json:"state"`
	Revision          uint64 `json:"revision"`
	AppSpecGeneration uint64 `json:"app_spec_generation"`
	SpecHash          string `json:"spec_hash"`
	ArtifactCount     int    `json:"artifact_count"`
}

func projectGoal(value goal.Goal) goalView {
	return goalView{
		GoalRef: value.Ref().String(), ProjectRef: value.Project().String(), State: string(value.State()),
		Revision: uint64(value.Revision()), PlanGeneration: uint64(value.PlanGeneration()),
		SpecHash: value.SpecHash(), WorkItemCount: value.WorkItemCount(),
	}
}

func projectGoalSummary(value application.GoalSummary) goalSummaryView {
	return goalSummaryView{
		GoalRef: value.Ref.String(), ProjectRef: value.ProjectRef.String(), Statement: value.Statement,
		State: string(value.State), Revision: uint64(value.Revision),
		AppSpecGeneration: uint64(value.AppSpecGeneration), SpecHash: value.SpecHash, ArtifactCount: value.ArtifactCount,
	}
}

type membershipAuditView struct {
	AuditRef         string `json:"audit_ref"`
	Action           string `json:"action"`
	ActorRef         string `json:"actor_ref"`
	TargetRef        string `json:"target_ref"`
	ProjectRef       string `json:"project_ref"`
	Role             string `json:"role"`
	PreviousRevision uint64 `json:"previous_revision"`
	Revision         uint64 `json:"revision"`
	OccurredAt       string `json:"occurred_at"`
}

func projectMembershipAudit(value identity.MembershipAuditReceipt) membershipAuditView {
	return membershipAuditView{
		AuditRef: value.Ref(), Action: string(value.Action()), ActorRef: value.ActorRef().String(),
		TargetRef: value.TargetRef().String(), ProjectRef: value.ProjectRef().String(), Role: string(value.Role()),
		PreviousRevision: uint64(value.PreviousRevision()), Revision: uint64(value.Revision()), OccurredAt: utc(value.OccurredAt()),
	}
}

type leaseView struct {
	GoalRef      string `json:"goal_ref"`
	PrincipalRef string `json:"principal_ref"`
	Token        string `json:"token"`
	Fence        uint64 `json:"fence"`
	LeaseUntil   string `json:"lease_until"`
}

func projectLease(value application.DirectorLeaseRecord) leaseView {
	return leaseView{
		GoalRef: value.GoalRef.String(), PrincipalRef: value.PrincipalRef.String(), Token: value.Token,
		Fence: value.Fence, LeaseUntil: utc(value.LeaseUntil),
	}
}

type mailboxView struct {
	MessageRef   string `json:"message_ref"`
	GoalRef      string `json:"goal_ref"`
	State        string `json:"state"`
	AttemptCount int    `json:"attempt_count"`
}

func projectMailbox(value application.MailboxRecord) mailboxView {
	return mailboxView{
		MessageRef: value.Envelope.Ref.String(), GoalRef: value.Envelope.GoalRef.String(),
		State: string(value.State), AttemptCount: len(value.Attempts),
	}
}

type mailboxAdmissionView struct {
	AdmissionRef string `json:"admission_ref"`
	MessageRef   string `json:"message_ref"`
	GoalRef      string `json:"goal_ref"`
}

func projectMailboxAdmission(value application.MailboxRecord) mailboxAdmissionView {
	return mailboxAdmissionView{
		AdmissionRef: value.Admission.Ref, MessageRef: value.Admission.MessageRef.String(), GoalRef: value.Envelope.GoalRef.String(),
	}
}

type mailboxClaimView struct {
	MessageRef            string `json:"message_ref"`
	RecipientPrincipalRef string `json:"recipient_principal_ref"`
	RecipientWorkItemRef  string `json:"recipient_work_item_ref"`
	RecipientExecutionRef string `json:"recipient_execution_ref"`
	ClaimToken            string `json:"claim_token"`
	Fence                 uint64 `json:"fence"`
}

func projectMailboxClaim(value application.MailboxClaim) mailboxClaimView {
	return mailboxClaimView{
		MessageRef: value.Attempt.MessageRef.String(), RecipientPrincipalRef: value.Attempt.Recipient.PrincipalRef.String(),
		RecipientWorkItemRef: value.Attempt.Recipient.WorkItemRef.String(), RecipientExecutionRef: value.Attempt.Recipient.ExecutionRef.String(),
		ClaimToken: value.Attempt.ClaimToken, Fence: value.Attempt.Fence,
	}
}

type mailboxMutationReceiptView struct {
	MessageRef string `json:"message_ref"`
	ReceiptRef string `json:"receipt_ref"`
}

func projectMailboxMutationReceipt(value application.MailboxRecord, claimToken string, fence uint64, consumed bool) mailboxMutationReceiptView {
	result := mailboxMutationReceiptView{MessageRef: value.Envelope.Ref.String()}
	for _, attempt := range value.Attempts {
		if attempt.ClaimToken != claimToken || attempt.Fence != fence {
			continue
		}
		result.ReceiptRef = attempt.DeliveryRef
		if consumed {
			result.ReceiptRef = attempt.ConsumptionRef
		}
		break
	}
	return result
}

type changeView struct {
	ChangeRef    string `json:"change_ref"`
	GoalRef      string `json:"goal_ref"`
	WorkItemRef  string `json:"work_item_ref"`
	ExecutionRef string `json:"execution_ref"`
	Status       string `json:"status"`
	TargetOID    string `json:"target_oid"`
}

func projectChange(value application.PendingChange) changeView {
	return changeView{
		ChangeRef: value.ChangeSet.Ref.String(), GoalRef: value.ChangeSet.GoalRef.String(),
		WorkItemRef: value.ChangeSet.WorkItemRef.String(), ExecutionRef: value.ChangeSet.ExecutionRef.String(),
		Status: string(value.Observation.Status), TargetOID: value.Observation.TargetOID,
	}
}

type councilRoundView struct {
	RoundRef      string `json:"round_ref"`
	GoalRef       string `json:"goal_ref"`
	WorkItemRef   string `json:"work_item_ref"`
	ChangeRef     string `json:"change_ref"`
	SubjectDigest string `json:"subject_digest"`
	Opener        string `json:"opener"`
	DirectorFence uint64 `json:"director_fence"`
	OpenedAt      string `json:"opened_at"`
}

func projectCouncilRound(value application.CouncilRoundRecord) councilRoundView {
	return councilRoundView{
		RoundRef: value.Ref, GoalRef: value.GoalRef.String(), WorkItemRef: value.WorkItemRef.String(),
		ChangeRef: value.ChangeSetRef, SubjectDigest: string(value.SubjectDigest),
		Opener: string(value.Opener), DirectorFence: value.DirectorFence, OpenedAt: utc(value.OpenedAt),
	}
}

type councilSkipView struct {
	SkipRef       string `json:"skip_ref"`
	GoalRef       string `json:"goal_ref"`
	WorkItemRef   string `json:"work_item_ref"`
	ChangeRef     string `json:"change_ref"`
	SubjectDigest string `json:"subject_digest"`
	PrincipalRef  string `json:"principal_ref"`
	Reason        string `json:"reason"`
	RecordedAt    string `json:"recorded_at"`
}

func projectCouncilSkip(value application.CouncilSkipRecord) councilSkipView {
	return councilSkipView{
		SkipRef: value.Ref, GoalRef: value.Subject.GoalRef, WorkItemRef: value.Subject.WorkItemRef,
		ChangeRef: value.Subject.ChangeSetRef, SubjectDigest: string(value.SubjectDigest),
		PrincipalRef: value.Skip.PrincipalRef, Reason: value.Skip.Reason, RecordedAt: utc(value.RecordedAt),
	}
}

type artifactView struct {
	ArtifactRef   string `json:"artifact_ref"`
	Digest        string `json:"digest"`
	MediaType     string `json:"media_type"`
	Size          int64  `json:"size"`
	ContentBase64 string `json:"content_base64"`
}

func projectArtifact(value ports.ArtifactContent) artifactView {
	return artifactView{
		ArtifactRef: value.Ref.String(), Digest: value.Digest, MediaType: value.MediaType,
		Size: value.Size, ContentBase64: base64.StdEncoding.EncodeToString(value.Content),
	}
}

func utc(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }
