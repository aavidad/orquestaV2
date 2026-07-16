package application

import (
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"strconv"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

// mailboxEnvelopeInputBytes accounts for every caller-controlled field that
// becomes immutable envelope content. The fixed eight-byte prefix mirrors the
// canonical length-delimited hash encoding and keeps refs in the same budget
// as the summary. Generated IDs and hashes are bounded by their own contracts.
func mailboxEnvelopeInputBytes(
	sourcePrincipalRef identity.PrincipalRef,
	projectRef goal.ProjectRef,
	request AdmitMailboxRequest,
) int64 {
	fields := []string{
		request.RequestRef, projectRef.String(), request.GoalRef.String(),
		strconv.FormatUint(uint64(request.ExpectedPlanGeneration), 10), string(request.Kind),
		request.ParentWorkItemRef.String(), request.ChildWorkItemRef.String(),
		sourcePrincipalRef.String(), request.SourceExecutionRef.String(), request.RecipientPrincipalRef.String(),
		request.RecipientExecutionRef.String(), request.Summary,
	}
	var size int64
	for _, field := range fields {
		size += 8 + int64(len(field))
	}
	for _, artifactRef := range request.ArtifactRefs {
		size += 8 + int64(len(artifactRef.String()))
	}
	return size
}

func admitMailboxFingerprint(
	principalRef identity.PrincipalRef,
	projectRef goal.ProjectRef,
	request AdmitMailboxRequest,
) string {
	return mailboxAdmissionFingerprint(
		principalRef, projectRef, request.GoalRef, request.ExpectedPlanGeneration,
		request.Kind, request.ParentWorkItemRef, request.ChildWorkItemRef,
		request.SourceExecutionRef, request.RecipientPrincipalRef,
		request.RecipientExecutionRef, request.Summary, request.ArtifactRefs,
	)
}

// MailboxAdmissionFingerprint recomputes the canonical admission fingerprint
// from immutable persisted envelope fields. Recovery adapters use it to detect
// a request fingerprint that was altered together with its durable row.
func MailboxAdmissionFingerprint(envelope MailboxEnvelope) string {
	return mailboxAdmissionFingerprint(
		envelope.Source.PrincipalRef, envelope.ProjectRef, envelope.GoalRef,
		envelope.TargetPlanGeneration, envelope.Kind, envelope.ParentWorkItemRef,
		envelope.ChildWorkItemRef, envelope.Source.ExecutionRef,
		envelope.Recipient.PrincipalRef, envelope.Recipient.ExecutionRef,
		envelope.Summary, envelope.ArtifactRefs,
	)
}

func mailboxAdmissionFingerprint(
	principalRef identity.PrincipalRef,
	projectRef goal.ProjectRef,
	goalRef goal.GoalRef,
	planGeneration goal.PlanGeneration,
	kind MailboxKind,
	parentWorkItemRef goal.WorkItemRef,
	childWorkItemRef goal.WorkItemRef,
	sourceExecutionRef goal.ExecutionRef,
	recipientPrincipalRef identity.PrincipalRef,
	recipientExecutionRef goal.ExecutionRef,
	summary string,
	artifactRefs []goal.ArtifactRef,
) string {
	digest := sha256.New()
	writeMailboxFingerprintFields(digest,
		"orquesta.mailbox.admit.v1", principalRef.String(), projectRef.String(),
		goalRef.String(), strconv.FormatUint(uint64(planGeneration), 10), string(kind),
		parentWorkItemRef.String(), childWorkItemRef.String(), sourceExecutionRef.String(),
		recipientPrincipalRef.String(), recipientExecutionRef.String(), summary,
	)
	for _, ref := range artifactRefs {
		writeFingerprintField(digest, ref.String())
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func mailboxMutationFingerprint(
	kind MailboxMutationKind,
	principalRef identity.PrincipalRef,
	projectRef goal.ProjectRef,
	goalRef goal.GoalRef,
	messageRef MailboxMessageRef,
	workItemRef goal.WorkItemRef,
	executionRef goal.ExecutionRef,
	claimToken string,
	fence uint64,
	extra string,
) string {
	return MailboxMutationFingerprint(
		kind, principalRef, projectRef, goalRef, messageRef, workItemRef,
		executionRef, claimToken, fence, extra,
	)
}

// MailboxMutationFingerprint is the canonical digest for a persisted mailbox
// claim, delivery, consumption or resolution request.
func MailboxMutationFingerprint(
	kind MailboxMutationKind,
	principalRef identity.PrincipalRef,
	projectRef goal.ProjectRef,
	goalRef goal.GoalRef,
	messageRef MailboxMessageRef,
	workItemRef goal.WorkItemRef,
	executionRef goal.ExecutionRef,
	claimToken string,
	fence uint64,
	extra string,
) string {
	digest := sha256.New()
	writeMailboxFingerprintFields(digest,
		"orquesta.mailbox.mutation.v2", string(kind), principalRef.String(), projectRef.String(),
		goalRef.String(), messageRef.String(), workItemRef.String(), executionRef.String(),
		claimToken, strconv.FormatUint(fence, 10), extra,
	)
	return hex.EncodeToString(digest.Sum(nil))
}

// MailboxEnvelopeContentHash is the canonical provider-neutral digest for one
// immutable mailbox envelope. Persistence and recovery adapters use this same
// pure helper instead of duplicating the encoding contract.
func MailboxEnvelopeContentHash(envelope MailboxEnvelope) string {
	digest := sha256.New()
	writeMailboxFingerprintFields(digest,
		"orquesta.mailbox.envelope.v1", envelope.Ref.String(), envelope.ProjectRef.String(),
		envelope.GoalRef.String(), strconv.FormatUint(uint64(envelope.TargetPlanGeneration), 10),
		string(envelope.Kind), envelope.ParentWorkItemRef.String(), envelope.ChildWorkItemRef.String(),
		envelope.Source.PrincipalRef.String(), envelope.Source.WorkItemRef.String(),
		envelope.Source.ExecutionRef.String(), envelope.Recipient.PrincipalRef.String(),
		envelope.Recipient.WorkItemRef.String(), envelope.Recipient.ExecutionRef.String(), envelope.Summary,
	)
	for _, ref := range envelope.ArtifactRefs {
		writeFingerprintField(digest, ref.String())
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func mailboxEnvelopeContentHash(envelope MailboxEnvelope) string {
	return MailboxEnvelopeContentHash(envelope)
}

func writeMailboxFingerprintFields(digest hash.Hash, fields ...string) {
	for _, field := range fields {
		writeFingerprintField(digest, field)
	}
}

func mailboxAuthorizationRequestRef(kind MailboxMutationKind, requestRef, fingerprint string) string {
	digest := sha256.New()
	writeMailboxFingerprintFields(digest, "orquesta.mailbox.authorization.v1", string(kind), requestRef, fingerprint)
	return "authorization-request:mailbox:" + hex.EncodeToString(digest.Sum(nil))
}
