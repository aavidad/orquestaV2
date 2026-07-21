package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

// MergeObservationStatusClean is the only preview outcome that may lead to an
// applied target update. Conflict and stale remain durable pending evidence.
const MergeObservationStatusClean ports.MergeStatus = "clean"

func newExecutionWorkspaceRef(ctx context.Context, ids IDGenerator) (ports.ExecutionWorkspaceRef, error) {
	value, err := ids.NewID(ctx, "execution-workspace")
	if err != nil {
		return ports.ExecutionWorkspaceRef{}, err
	}
	return ports.NewExecutionWorkspaceRef(value)
}

// WorkspaceBinding is an immutable causal fact. Its physical path belongs only
// to the local adapter that resolves its opaque WorkspaceRef.
type WorkspaceBinding struct {
	Ref               ports.ExecutionWorkspaceRef
	PrincipalRef      identity.PrincipalRef
	ActorRef          goal.ActorRef
	ProjectRef        goal.ProjectRef
	RepositoryRef     identity.RepositoryRef
	GoalRef           goal.GoalRef
	WorkItemRef       goal.WorkItemRef
	ExecutionRef      goal.ExecutionRef
	ExecutionAttempt  uint64
	PlanGeneration    goal.PlanGeneration
	AppSpecGeneration goal.AppSpecGeneration
	SpecHash          string
	WriteSet          []string
	WriteSetDigest    string
	TargetRef         string
	BaseOID           string
	ObjectFormat      ports.GitObjectFormat
	AdapterRef        string
	EffectIntentRef   string
	EffectAttemptRef  string
	EffectFence       uint64
	ReceiptRef        string
	PreparedAt        time.Time
}

func (binding WorkspaceBinding) Digest() string {
	return workspaceFactDigest("workspace_binding", []string{
		binding.Ref.String(), binding.PrincipalRef.String(), binding.ActorRef.String(), binding.ProjectRef.String(), binding.RepositoryRef.String(),
		binding.GoalRef.String(), binding.WorkItemRef.String(), binding.ExecutionRef.String(), decimal(binding.ExecutionAttempt), decimal(uint64(binding.PlanGeneration)), decimal(uint64(binding.AppSpecGeneration)), binding.SpecHash,
		binding.WriteSetDigest, binding.TargetRef, binding.BaseOID, string(binding.ObjectFormat), binding.AdapterRef, binding.EffectIntentRef, binding.EffectAttemptRef, decimal(binding.EffectFence), binding.ReceiptRef, binding.PreparedAt.UTC().Format(time.RFC3339Nano),
	}, binding.WriteSet)
}

func ValidateWorkspaceBinding(binding WorkspaceBinding) error {
	preparedAt := binding.PreparedAt
	request := ports.WorkspacePrepareRequest{
		WorkspaceRef: binding.Ref, PrincipalRef: binding.PrincipalRef, ActorRef: binding.ActorRef,
		ProjectRef: binding.ProjectRef, RepositoryRef: binding.RepositoryRef, GoalRef: binding.GoalRef,
		WorkItemRef: binding.WorkItemRef, ExecutionRef: binding.ExecutionRef, ExecutionAttempt: binding.ExecutionAttempt,
		PlanGeneration: binding.PlanGeneration, AppSpecGeneration: binding.AppSpecGeneration, AppSpecHash: binding.SpecHash,
		WriteSet: binding.WriteSet, WriteSetDigest: binding.WriteSetDigest, TargetRef: binding.TargetRef,
		IntentRef: binding.EffectIntentRef, AttemptRef: binding.EffectAttemptRef,
		ActionFence: binding.EffectFence, IdempotencyKey: binding.EffectIntentRef, PreparedAt: preparedAt,
	}
	if err := ports.ValidateWorkspacePrepareRequest(request); err != nil {
		return errors.New("application.workspace_binding_invalid")
	}
	prepared := ports.WorkspacePrepared{WorkspaceRef: binding.Ref, RepositoryRef: binding.RepositoryRef, ExecutionRef: binding.ExecutionRef, TargetRef: binding.TargetRef, BaseOID: binding.BaseOID, ObjectFormat: binding.ObjectFormat, WriteSetDigest: binding.WriteSetDigest, AdapterRef: binding.AdapterRef, ReceiptRef: binding.ReceiptRef, PreparedAt: binding.PreparedAt}
	if err := ports.ValidateWorkspacePrepared(request, prepared); err != nil || !validApplicationRef(binding.ReceiptRef) {
		return errors.New("application.workspace_binding_receipt_invalid")
	}
	return nil
}

// ChangeSet preserves the one immutable commit that an execution produced.
type ChangeSet struct {
	Ref               ports.ChangeSetRef
	WorkspaceRef      ports.ExecutionWorkspaceRef
	PrincipalRef      identity.PrincipalRef
	ActorRef          goal.ActorRef
	ProjectRef        goal.ProjectRef
	RepositoryRef     identity.RepositoryRef
	GoalRef           goal.GoalRef
	WorkItemRef       goal.WorkItemRef
	ExecutionRef      goal.ExecutionRef
	ExecutionAttempt  uint64
	PlanGeneration    goal.PlanGeneration
	AppSpecGeneration goal.AppSpecGeneration
	SpecHash          string
	BaseOID           string
	ParentOID         string
	HeadOID           string
	TreeOID           string
	ObjectFormat      ports.GitObjectFormat
	DiffDigest        string
	ChangedPaths      []string
	WriteSet          []string
	WriteSetDigest    string
	ParentChangeRef   ports.ChangeSetRef
	EffectIntentRef   string
	EffectAttemptRef  string
	EffectFence       uint64
	IdempotencyKey    string
	AdapterRef        string
	ReceiptRef        string
	CommittedAt       time.Time
}

func (change ChangeSet) Digest() string {
	return workspaceFactDigest("change_set", []string{
		change.Ref.String(), change.WorkspaceRef.String(), change.PrincipalRef.String(), change.ActorRef.String(), change.ProjectRef.String(), change.RepositoryRef.String(), change.GoalRef.String(), change.WorkItemRef.String(), change.ExecutionRef.String(), decimal(change.ExecutionAttempt), decimal(uint64(change.PlanGeneration)), decimal(uint64(change.AppSpecGeneration)), change.SpecHash,
		change.BaseOID, change.ParentOID, change.HeadOID, change.TreeOID, string(change.ObjectFormat), change.DiffDigest, change.WriteSetDigest, change.ParentChangeRef.String(), change.EffectIntentRef, change.EffectAttemptRef, decimal(change.EffectFence), change.IdempotencyKey, change.AdapterRef, change.ReceiptRef, change.CommittedAt.UTC().Format(time.RFC3339Nano),
	}, append(append([]string(nil), change.WriteSet...), change.ChangedPaths...))
}

func ValidateChangeSet(change ChangeSet) error {
	request := ports.CommitRequest{
		ChangeSetRef: change.Ref, WorkspaceRef: change.WorkspaceRef, PrincipalRef: change.PrincipalRef, ActorRef: change.ActorRef,
		ProjectRef: change.ProjectRef, RepositoryRef: change.RepositoryRef, GoalRef: change.GoalRef, WorkItemRef: change.WorkItemRef,
		ExecutionRef: change.ExecutionRef, ExecutionAttempt: change.ExecutionAttempt, PlanGeneration: change.PlanGeneration, AppSpecGeneration: change.AppSpecGeneration,
		AppSpecHash: change.SpecHash, BaseOID: change.BaseOID, ObjectFormat: change.ObjectFormat, WriteSet: change.WriteSet,
		WriteSetDigest: change.WriteSetDigest, ParentChangeRef: change.ParentChangeRef, IntentRef: change.EffectIntentRef, AttemptRef: change.EffectAttemptRef,
		ActionFence: change.EffectFence, IdempotencyKey: change.IdempotencyKey, CommittedAt: change.CommittedAt,
	}
	result := ports.CommitResult{
		ChangeSetRef: change.Ref, WorkspaceRef: change.WorkspaceRef, RepositoryRef: change.RepositoryRef, ExecutionRef: change.ExecutionRef,
		BaseOID: change.BaseOID, ParentOID: change.ParentOID, HeadOID: change.HeadOID, TreeOID: change.TreeOID, ObjectFormat: change.ObjectFormat,
		DiffDigest: change.DiffDigest, ChangedPaths: change.ChangedPaths, WriteSetDigest: change.WriteSetDigest, ParentChangeRef: change.ParentChangeRef,
		AdapterRef: change.AdapterRef, ReceiptRef: change.ReceiptRef, CommittedAt: change.CommittedAt,
	}
	if err := ports.ValidateCommitResult(request, result); err != nil {
		return errors.New("application.change_set_invalid")
	}
	return nil
}

type MergeObservation struct {
	Ref              string
	ChangeRef        ports.ChangeSetRef
	RepositoryRef    identity.RepositoryRef
	SourceOID        string
	TargetRef        string
	TargetOID        string
	ObjectFormat     ports.GitObjectFormat
	Status           ports.MergeStatus
	CandidateTreeOID string
	ConflictDigest   string
	AdapterRef       string
	ObservedAt       time.Time
}

func (observation MergeObservation) Digest() string {
	return workspaceFactDigest("merge_observation", []string{
		observation.Ref, observation.ChangeRef.String(), observation.RepositoryRef.String(), observation.SourceOID, observation.TargetRef, observation.TargetOID, string(observation.ObjectFormat), string(observation.Status), observation.CandidateTreeOID, observation.ConflictDigest, observation.AdapterRef, observation.ObservedAt.UTC().Format(time.RFC3339Nano),
	}, nil)
}

func ValidateMergeObservation(observation MergeObservation) error {
	switch observation.Status {
	case MergeObservationStatusClean, ports.MergeStatusConflicted, ports.MergeStatusStale:
	default:
		return errors.New("application.merge_observation_status_invalid")
	}
	request := ports.IntegrationPreviewRequest{ChangeSetRef: observation.ChangeRef, RepositoryRef: observation.RepositoryRef, SourceOID: observation.SourceOID, TargetRef: observation.TargetRef, TargetOID: observation.TargetOID, ObjectFormat: observation.ObjectFormat, IdempotencyKey: "preview:" + observation.Digest(), RequestedAt: observation.ObservedAt}
	preview := ports.IntegrationPreview{ChangeSetRef: observation.ChangeRef, RepositoryRef: observation.RepositoryRef, SourceOID: observation.SourceOID, TargetRef: observation.TargetRef, TargetOID: observation.TargetOID, ObjectFormat: observation.ObjectFormat, Status: observation.Status, CandidateTreeOID: observation.CandidateTreeOID, ConflictDigest: observation.ConflictDigest, AdapterRef: observation.AdapterRef, ObservedAt: observation.ObservedAt}
	if err := ports.ValidateIntegrationPreview(request, preview); err != nil || !validApplicationRef(observation.Ref) {
		return errors.New("application.merge_observation_invalid")
	}
	return nil
}

// IntegrationReceipt is the durable evidence of a local integration attempt.
// A non-clean result deliberately records no target mutation.
type IntegrationReceipt struct {
	Ref              string
	ChangeRef        ports.ChangeSetRef
	RepositoryRef    identity.RepositoryRef
	SourceOID        string
	TargetRef        string
	TargetBeforeOID  string
	TargetAfterOID   string
	TreeOID          string
	ObjectFormat     ports.GitObjectFormat
	Status           ports.IntegrationStatus
	MarkerRef        string
	ConflictDigest   string
	EffectIntentRef  string
	EffectAttemptRef string
	EffectFence      uint64
	EffectReceiptRef string
	AdapterRef       string
	RecordedAt       time.Time
}

func (receipt IntegrationReceipt) Digest() string {
	return workspaceFactDigest("integration_receipt", []string{
		receipt.Ref, receipt.ChangeRef.String(), receipt.RepositoryRef.String(), receipt.SourceOID, receipt.TargetRef, receipt.TargetBeforeOID, receipt.TargetAfterOID, receipt.TreeOID, string(receipt.ObjectFormat), string(receipt.Status), receipt.MarkerRef, receipt.ConflictDigest, receipt.EffectIntentRef, receipt.EffectAttemptRef, decimal(receipt.EffectFence), receipt.EffectReceiptRef, receipt.AdapterRef, receipt.RecordedAt.UTC().Format(time.RFC3339Nano),
	}, nil)
}

func ValidateIntegrationReceipt(receipt IntegrationReceipt) error {
	result := ports.IntegrationResult{ChangeSetRef: receipt.ChangeRef, RepositoryRef: receipt.RepositoryRef, SourceOID: receipt.SourceOID, TargetRef: receipt.TargetRef, TargetBeforeOID: receipt.TargetBeforeOID, TargetAfterOID: receipt.TargetAfterOID, TreeOID: receipt.TreeOID, ObjectFormat: receipt.ObjectFormat, Status: receipt.Status, MarkerRef: receipt.MarkerRef, ConflictDigest: receipt.ConflictDigest, AdapterRef: receipt.AdapterRef, ReceiptRef: receipt.EffectReceiptRef, RecordedAt: receipt.RecordedAt}
	if err := ports.ValidateIntegrationResultFields(result); err != nil ||
		!validApplicationRef(receipt.Ref) || !validApplicationRef(receipt.EffectIntentRef) ||
		!validApplicationRef(receipt.EffectAttemptRef) || receipt.EffectFence == 0 ||
		!validApplicationRef(receipt.EffectReceiptRef) {
		return errors.New("application.integration_receipt_invalid")
	}
	return nil
}

func workspaceFactDigest(kind string, fields, lists []string) string {
	digest := sha256.New()
	digest.Write([]byte("orquesta." + kind + ".v1\x00"))
	for _, field := range fields {
		digest.Write([]byte(field))
		digest.Write([]byte{0})
	}
	for _, value := range lists {
		digest.Write([]byte(value))
		digest.Write([]byte{0})
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func decimal(value uint64) string {
	return strconv.FormatUint(value, 10)
}
