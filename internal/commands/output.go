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
	GoalRef           string `json:"goal_ref"`
	ProjectRef        string `json:"project_ref"`
	State             string `json:"state"`
	Revision          uint64 `json:"revision"`
	PlanGeneration    uint64 `json:"plan_generation"`
	AppSpecGeneration uint64 `json:"app_spec_generation"`
	SpecHash          string `json:"spec_hash"`
	WorkItemCount     int    `json:"work_item_count"`
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
		AppSpecGeneration: uint64(value.AppSpec().Generation()), SpecHash: value.SpecHash(), WorkItemCount: value.WorkItemCount(),
	}
}

type workItemView struct {
	WorkItemRef       string   `json:"work_item_ref"`
	State             string   `json:"state"`
	Revision          uint64   `json:"revision"`
	ParentWorkItemRef string   `json:"parent_work_item_ref"`
	DependencyRefs    []string `json:"dependency_refs"`
	HandoffRequired   bool     `json:"handoff_required"`
	ExecutionRef      string   `json:"execution_ref"`
	ArtifactRefs      []string `json:"artifact_refs"`
	AttestationRefs   []string `json:"attestation_refs"`
	Paused            bool     `json:"paused"`
	CancelRequested   bool     `json:"cancel_requested"`
	InterruptCode     string   `json:"interrupt_code"`
}

func projectWorkItem(value goal.WorkItem) workItemView {
	parentRef := ""
	if ref, ok := value.Parent(); ok {
		parentRef = ref.String()
	}
	executionRef := ""
	if ref, ok := value.Execution(); ok {
		executionRef = ref.String()
	}
	interruptCode := ""
	if cause, ok := value.InterruptCause(); ok {
		interruptCode = string(cause)
	}
	dependencies := value.Dependencies()
	dependencyRefs := make([]string, 0, len(dependencies))
	for _, ref := range dependencies {
		dependencyRefs = append(dependencyRefs, ref.String())
	}
	artifacts := value.Artifacts()
	artifactRefs := make([]string, 0, len(artifacts))
	for _, ref := range artifacts {
		artifactRefs = append(artifactRefs, ref.String())
	}
	attestations := value.Attestations()
	attestationRefs := make([]string, 0, len(attestations))
	for _, ref := range attestations {
		attestationRefs = append(attestationRefs, ref.String())
	}
	return workItemView{
		WorkItemRef: value.Ref().String(), State: string(value.State()), Revision: uint64(value.Revision()),
		ParentWorkItemRef: parentRef, DependencyRefs: dependencyRefs, HandoffRequired: value.HandoffRequired(),
		ExecutionRef: executionRef, ArtifactRefs: artifactRefs, AttestationRefs: attestationRefs,
		Paused: value.Paused(), CancelRequested: value.CancelRequested(), InterruptCode: interruptCode,
	}
}

type executionView struct {
	ExecutionRef            string `json:"execution_ref"`
	WorkItemRef             string `json:"work_item_ref"`
	AttemptNo               uint64 `json:"attempt_no"`
	MaxAttempts             uint64 `json:"max_attempts"`
	ReplacesExecutionRef    string `json:"replaces_execution_ref"`
	PlanGeneration          uint64 `json:"plan_generation"`
	AppSpecGeneration       uint64 `json:"app_spec_generation"`
	State                   string `json:"state"`
	Purpose                 string `json:"purpose"`
	FailureCode             string `json:"failure_code"`
	RecipientMailboxRetired bool   `json:"recipient_mailbox_retired"`
}

func projectExecution(value application.ExecutionRecord) executionView {
	return executionView{
		ExecutionRef: value.Ref.String(), WorkItemRef: value.WorkItemRef.String(),
		AttemptNo: value.AttemptNo, MaxAttempts: value.MaxExecutionAttempts,
		ReplacesExecutionRef: value.ReplacesExecutionRef.String(),
		PlanGeneration:       uint64(value.PlanGeneration), AppSpecGeneration: uint64(value.AppSpecGeneration),
		State: string(value.State), Purpose: string(value.Purpose), FailureCode: value.FailureCode,
		RecipientMailboxRetired: value.RecipientMailboxRetired,
	}
}

type requiredTestOutcomeView struct {
	RequiredTestRef string `json:"required_test_ref"`
	ExitCode        int    `json:"exit_code"`
	OutputDigest    string `json:"output_digest"`
}

type attestationView struct {
	AttestationRef    string                    `json:"attestation_ref"`
	Kind              string                    `json:"kind"`
	Verdict           string                    `json:"verdict"`
	WorkItemRef       string                    `json:"work_item_ref"`
	ExecutionRef      string                    `json:"execution_ref"`
	ExecutionAttempt  uint64                    `json:"execution_attempt"`
	PlanGeneration    uint64                    `json:"plan_generation"`
	WorkItemRevision  uint64                    `json:"work_item_revision"`
	AppSpecGeneration uint64                    `json:"app_spec_generation"`
	ChangeRef         string                    `json:"change_ref"`
	Tests             []requiredTestOutcomeView `json:"tests"`
}

func projectAttestation(value application.AttestationRecord) attestationView {
	tests := make([]requiredTestOutcomeView, 0, len(value.Tests))
	for _, outcome := range value.Tests {
		tests = append(tests, requiredTestOutcomeView{
			RequiredTestRef: outcome.RequiredTestRef.String(), ExitCode: outcome.ExitCode, OutputDigest: outcome.OutputDigest,
		})
	}
	return attestationView{
		AttestationRef: value.Ref.String(), Kind: string(value.Kind), Verdict: string(value.Verdict),
		WorkItemRef: value.WorkItemRef.String(), ExecutionRef: value.ExecutionRef.String(),
		ExecutionAttempt: value.ExecutionAttempt, PlanGeneration: uint64(value.PlanGeneration),
		WorkItemRevision: uint64(value.WorkItemGeneration), AppSpecGeneration: uint64(value.AppSpecGeneration),
		ChangeRef: value.ChangeSetRef.String(), Tests: tests,
	}
}

type reviewView struct {
	ReviewRef                string `json:"review_ref"`
	WorkItemRef              string `json:"work_item_ref"`
	ChangeRef                string `json:"change_ref"`
	SubjectDigest            string `json:"subject_digest"`
	Role                     string `json:"role"`
	Verdict                  string `json:"verdict"`
	ReviewerExecutionRef     string `json:"reviewer_execution_ref"`
	ReviewerExecutionAttempt uint64 `json:"reviewer_execution_attempt"`
}

func projectReview(value application.ReviewRecord) reviewView {
	return reviewView{
		ReviewRef: value.Ref, WorkItemRef: value.WorkItemRef.String(), ChangeRef: value.ChangeSetRef.String(),
		SubjectDigest: value.SubjectDigest, Role: string(value.Role), Verdict: string(value.Verdict),
		ReviewerExecutionRef:     value.ReviewerExecutionRef.String(),
		ReviewerExecutionAttempt: value.ReviewerExecutionAttempt,
	}
}

type controlView struct {
	ControlRef             string `json:"control_ref"`
	Operation              string `json:"operation"`
	Target                 string `json:"target"`
	Status                 string `json:"status"`
	Mode                   string `json:"mode"`
	GoalRevision           uint64 `json:"goal_revision"`
	PlanGeneration         uint64 `json:"plan_generation"`
	AppSpecGeneration      uint64 `json:"app_spec_generation"`
	WorkItemRef            string `json:"work_item_ref"`
	WorkItemRevision       uint64 `json:"work_item_revision"`
	ExecutionRef           string `json:"execution_ref"`
	ExecutionAttempt       uint64 `json:"execution_attempt"`
	ReceiptRef             string `json:"receipt_ref"`
	SupersedesControlRef   string `json:"supersedes_control_ref"`
	SupersededByControlRef string `json:"superseded_by_control_ref"`
}

func projectControl(value application.ControlRecord) controlView {
	return controlView{
		ControlRef: value.Ref, Operation: string(value.Operation), Target: string(value.Target),
		Status: string(value.Status), Mode: string(value.Mode), GoalRevision: uint64(value.GoalRevision),
		PlanGeneration: uint64(value.PlanGeneration), AppSpecGeneration: uint64(value.AppSpecGeneration),
		WorkItemRef: value.WorkItemRef.String(), WorkItemRevision: uint64(value.WorkItemRevision),
		ExecutionRef: value.ExecutionRef.String(), ExecutionAttempt: value.ExecutionAttempt,
		ReceiptRef: value.ReceiptRef, SupersedesControlRef: value.SupersedesControlRef,
		SupersededByControlRef: value.SupersededByControlRef,
	}
}

type integrationReceiptView struct {
	IntegrationRef  string `json:"integration_ref"`
	ChangeRef       string `json:"change_ref"`
	Status          string `json:"status"`
	TargetBeforeOID string `json:"target_before_oid"`
	TargetAfterOID  string `json:"target_after_oid"`
	TreeOID         string `json:"tree_oid"`
	ConflictDigest  string `json:"conflict_digest"`
}

func projectIntegrationReceipt(value application.IntegrationReceipt) integrationReceiptView {
	return integrationReceiptView{
		IntegrationRef: value.Ref, ChangeRef: value.ChangeRef.String(), Status: string(value.Status),
		TargetBeforeOID: value.TargetBeforeOID, TargetAfterOID: value.TargetAfterOID,
		TreeOID: value.TreeOID, ConflictDigest: value.ConflictDigest,
	}
}

type goalRecordView struct {
	Goal                goalView                 `json:"goal"`
	ExecutionCount      int                      `json:"execution_count"`
	ArtifactCount       int                      `json:"artifact_count"`
	WorkItems           []workItemView           `json:"work_items"`
	Executions          []executionView          `json:"executions"`
	Attestations        []attestationView        `json:"attestations"`
	Reviews             []reviewView             `json:"reviews"`
	Controls            []controlView            `json:"controls"`
	IntegrationReceipts []integrationReceiptView `json:"integration_receipts"`
	MailboxReceipts     []mailboxReceiptView     `json:"mailbox_receipts"`
}

func projectGoalRecord(value application.GoalRecord) goalRecordView {
	items := value.Goal.WorkItems()
	workItems := make([]workItemView, 0, len(items))
	for _, item := range items {
		workItems = append(workItems, projectWorkItem(item))
	}
	executions := make([]executionView, 0, len(value.Executions))
	for _, execution := range value.Executions {
		executions = append(executions, projectExecution(execution))
	}
	attestations := make([]attestationView, 0, len(value.Attestations))
	for _, attestation := range value.Attestations {
		attestations = append(attestations, projectAttestation(attestation))
	}
	reviews := make([]reviewView, 0, len(value.Reviews))
	for _, review := range value.Reviews {
		reviews = append(reviews, projectReview(review))
	}
	controls := make([]controlView, 0, len(value.Controls))
	for _, control := range value.Controls {
		controls = append(controls, projectControl(control))
	}
	integrationReceipts := make([]integrationReceiptView, 0, len(value.IntegrationReceipts))
	for _, receipt := range value.IntegrationReceipts {
		integrationReceipts = append(integrationReceipts, projectIntegrationReceipt(receipt))
	}
	mailboxReceipts := make([]mailboxReceiptView, 0, len(value.Mailboxes))
	for _, mailbox := range value.Mailboxes {
		mailboxReceipts = append(mailboxReceipts, projectMailboxReceipt(mailbox))
	}
	return goalRecordView{
		Goal: projectGoal(value.Goal), ExecutionCount: len(value.Executions), ArtifactCount: len(value.Artifacts),
		WorkItems: workItems, Executions: executions, Attestations: attestations, Reviews: reviews,
		Controls: controls, IntegrationReceipts: integrationReceipts, MailboxReceipts: mailboxReceipts,
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
	MessageRef            string   `json:"message_ref"`
	GoalRef               string   `json:"goal_ref"`
	TargetPlanGeneration  uint64   `json:"target_plan_generation"`
	Kind                  string   `json:"kind"`
	ParentWorkItemRef     string   `json:"parent_work_item_ref"`
	ChildWorkItemRef      string   `json:"child_work_item_ref"`
	SourceWorkItemRef     string   `json:"source_work_item_ref"`
	SourceExecutionRef    string   `json:"source_execution_ref"`
	RecipientWorkItemRef  string   `json:"recipient_work_item_ref"`
	RecipientExecutionRef string   `json:"recipient_execution_ref"`
	Summary               string   `json:"summary"`
	ArtifactRefs          []string `json:"artifact_refs"`
	State                 string   `json:"state"`
	AttemptCount          int      `json:"attempt_count"`
}

type mailboxReceiptView struct {
	MessageRef            string `json:"message_ref"`
	State                 string `json:"state"`
	SourcePrincipalRef    string `json:"source_principal_ref"`
	SourceExecutionRef    string `json:"source_execution_ref"`
	RecipientPrincipalRef string `json:"recipient_principal_ref"`
	RecipientExecutionRef string `json:"recipient_execution_ref"`
	AdmissionRef          string `json:"admission_ref"`
	ConsumptionRef        string `json:"consumption_ref"`
	AcknowledgementRef    string `json:"acknowledgement_ref"`
	Outcome               string `json:"outcome"`
}

func projectMailboxReceipt(value application.MailboxRecord) mailboxReceiptView {
	result := mailboxReceiptView{
		MessageRef: value.Envelope.Ref.String(), State: string(value.State),
		SourcePrincipalRef:    value.Admission.PrincipalRef.String(),
		SourceExecutionRef:    value.Envelope.Source.ExecutionRef.String(),
		RecipientPrincipalRef: value.Envelope.Recipient.PrincipalRef.String(),
		RecipientExecutionRef: value.Envelope.Recipient.ExecutionRef.String(),
		AdmissionRef:          value.Admission.Ref,
	}
	if len(value.Attempts) != 0 {
		result.ConsumptionRef = value.Attempts[len(value.Attempts)-1].ConsumptionRef
	}
	if value.Acknowledgement != nil {
		result.AcknowledgementRef, result.Outcome = value.Acknowledgement.Ref, string(value.Acknowledgement.Outcome)
	}
	return result
}

func projectMailbox(value application.MailboxRecord) mailboxView {
	artifactRefs := make([]string, 0, len(value.Envelope.ArtifactRefs))
	for _, ref := range value.Envelope.ArtifactRefs {
		artifactRefs = append(artifactRefs, ref.String())
	}
	return mailboxView{
		MessageRef: value.Envelope.Ref.String(), GoalRef: value.Envelope.GoalRef.String(),
		TargetPlanGeneration: uint64(value.Envelope.TargetPlanGeneration), Kind: string(value.Envelope.Kind),
		ParentWorkItemRef: value.Envelope.ParentWorkItemRef.String(), ChildWorkItemRef: value.Envelope.ChildWorkItemRef.String(),
		SourceWorkItemRef: value.Envelope.Source.WorkItemRef.String(), SourceExecutionRef: value.Envelope.Source.ExecutionRef.String(),
		RecipientWorkItemRef:  value.Envelope.Recipient.WorkItemRef.String(),
		RecipientExecutionRef: value.Envelope.Recipient.ExecutionRef.String(),
		Summary:               value.Envelope.Summary, ArtifactRefs: artifactRefs,
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
