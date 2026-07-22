package application

import (
	"errors"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func buildTestSubject(
	aggregate goal.Goal,
	item goal.WorkItem,
	execution ExecutionRecord,
	binding WorkspaceBinding,
	change ChangeSet,
	policy TestAttestationPolicy,
) (ports.TestSubject, error) {
	currentItem, currentItemFound := aggregate.WorkItem(item.Ref())
	if ValidateTestAttestationPolicy(policy) != nil || len(item.RequiredTests()) == 0 ||
		!currentItemFound || currentItem.Revision() != item.Revision() ||
		binding.GoalRef != aggregate.Ref() || binding.WorkItemRef != item.Ref() ||
		binding.ExecutionRef != execution.Ref || change.WorkspaceRef != binding.Ref ||
		change.GoalRef != aggregate.Ref() || change.WorkItemRef != item.Ref() ||
		change.ExecutionRef != execution.Ref || change.ExecutionAttempt != execution.AttemptNo ||
		change.PlanGeneration != execution.PlanGeneration || change.AppSpecGeneration != execution.AppSpecGeneration ||
		change.SpecHash != execution.SpecHash || change.RepositoryRef != binding.RepositoryRef ||
		change.WriteSetDigest != binding.WriteSetDigest {
		return ports.TestSubject{}, errors.New("application.test_subject_causality_invalid")
	}
	return ports.TestSubject{
		GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref,
		ExecutionAttempt: execution.AttemptNo, PlanGeneration: execution.PlanGeneration,
		WorkItemGeneration: item.Revision(),
		AppSpecGeneration:  execution.AppSpecGeneration, AppSpecHash: execution.SpecHash,
		WorkspaceRef: binding.Ref, WorkspaceBindingDigest: binding.Digest(),
		ChangeSetRef: change.Ref, ChangeSetDigest: change.Digest(), RepositoryRef: change.RepositoryRef,
		ObjectFormat: change.ObjectFormat, BaseOID: change.BaseOID, ParentOID: change.ParentOID,
		HeadOID: change.HeadOID, TreeOID: change.TreeOID, DiffDigest: change.DiffDigest,
		WriteSetDigest: change.WriteSetDigest, RequiredTestsDigest: item.RequiredTestsDigest(),
		PolicyRef: policy.Ref, PolicyDigest: policy.Digest,
	}, nil
}

func testSnapshotRequest(subject ports.TestSubject, change ChangeSet) ports.SnapshotVerificationRequest {
	return ports.SnapshotVerificationRequest{
		Subject: subject, SubjectDigest: ports.TestSubjectDigest(subject),
		ChangedPaths: append([]string(nil), change.ChangedPaths...),
		WriteSet:     append([]string(nil), change.WriteSet...),
	}
}
