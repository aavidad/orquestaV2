package sqlite

import (
	"errors"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/review"
)

func validateGoalRecordReviews(record application.GoalRecord, items map[goal.WorkItemRef]goal.WorkItem,
	executions map[goal.ExecutionRef]application.ExecutionRecord,
) error {
	bySubject := make(map[string][]review.Assessment)
	subjects := make(map[string]review.Subject)
	seenRoles := make(map[string]struct{})
	participantRoles := make(map[string]map[review.Role]struct{})
	for _, participant := range executions {
		role, reviewer := sqliteReviewerRole(participant.Purpose)
		if !reviewer {
			continue
		}
		item, found := items[participant.WorkItemRef]
		if !found || application.ValidatePersistedReviewerBinding(record, item, participant) != nil {
			return errors.New("sqlite.goal_record_review_participant_invalid")
		}
		if participantRoles[participant.ReviewSubjectDigest] == nil {
			participantRoles[participant.ReviewSubjectDigest] = make(map[review.Role]struct{}, 2)
		}
		participantRoles[participant.ReviewSubjectDigest][role] = struct{}{}
	}
	for _, roles := range participantRoles {
		if len(roles) != 2 {
			return errors.New("sqlite.goal_record_review_pair_invalid")
		}
	}
	if err := validateGoalRecordReviewCleanupControls(record, items, executions); err != nil {
		return err
	}
	for _, fact := range record.Reviews {
		item, itemFound := items[fact.WorkItemRef]
		reviewer, executionFound := executions[fact.ReviewerExecutionRef]
		change, changeFound := reviewChange(record, fact)
		author, authorFound := executions[change.ExecutionRef]
		artifact, artifactFound := reviewArtifact(record, fact.AssessmentArtifactRef)
		intent, intentFound := reviewLaunchIntent(record, reviewer)
		authority, authorityFound := reviewAuthority(record, fact.WorkItemRef)
		role, roleOK := sqliteReviewerRole(reviewer.Purpose)
		if !itemFound || !executionFound || !changeFound || !authorFound || !artifactFound || !roleOK ||
			!intentFound || !authorityFound ||
			fact.GoalRef != record.Goal.Ref() || fact.Role != role || fact.SubjectDigest != reviewer.ReviewSubjectDigest ||
			fact.Ref != "review:"+reviewer.Ref.String() ||
			fact.ReviewerExecutionAttempt != reviewer.AttemptNo || fact.LaunchReceiptRef != reviewer.LaunchReceiptRef ||
			fact.AgentRef != reviewer.AgentRef || fact.ExternalRef != reviewer.ExternalRef ||
			fact.PrincipalRef != intent.ProposedBy ||
			fact.PrincipalRef != authority.PrincipalRef || intent.Permission != authority.Permission ||
			intent.Authority.Ref() != authority.AuthorizationReceipt.Ref() || reviewer.State != application.ExecutionSucceeded ||
			!fact.RecordedAt.Equal(reviewer.FinishedAt) || !fact.RecordedAt.Equal(artifact.CreatedAt) ||
			artifact.Kind != application.ArtifactKindReviewAssessment || artifact.ExecutionRef != reviewer.Ref ||
			artifact.GoalRef != fact.GoalRef || artifact.WorkItemRef != fact.WorkItemRef ||
			artifact.ExecutionAttempt != reviewer.AttemptNo || artifact.PlanGeneration != reviewer.PlanGeneration ||
			artifact.AppSpecGeneration != reviewer.AppSpecGeneration || artifact.SpecHash != reviewer.SpecHash ||
			artifact.Stored.Digest != fact.AssessmentDigest || artifact.Stored.Ref.String() != fact.AssessmentArtifactRef {
			return errors.New("sqlite.goal_record_review_scope_invalid")
		}
		subject, err := persistedReviewSubject(record, item, author, change)
		if err != nil || subject.Digest() != fact.SubjectDigest ||
			artifact.WorkItemGeneration != goal.Revision(subject.WorkItemGeneration) {
			return errors.New("sqlite.goal_record_review_subject_invalid")
		}
		assessment, err := review.NewAssessment(review.Assessment{
			SubjectDigest: fact.SubjectDigest, Role: fact.Role, Verdict: fact.Verdict,
			ReviewerExecutionRef:     fact.ReviewerExecutionRef.String(),
			ReviewerExecutionAttempt: fact.ReviewerExecutionAttempt, LaunchReceiptRef: fact.LaunchReceiptRef,
			ReviewerExternalRef:   fact.ExternalRef,
			AssessmentArtifactRef: fact.AssessmentArtifactRef, AssessmentDigest: fact.AssessmentDigest,
			RecordedAt: fact.RecordedAt,
		})
		key := fact.SubjectDigest + "\x00" + string(fact.Role)
		if err != nil {
			return errors.New("sqlite.goal_record_review_invalid")
		}
		if _, duplicate := seenRoles[key]; duplicate {
			return errors.New("sqlite.goal_record_review_duplicate")
		}
		seenRoles[key] = struct{}{}
		subjects[fact.SubjectDigest] = subject
		bySubject[fact.SubjectDigest] = append(bySubject[fact.SubjectDigest], assessment)
	}
	for digest, assessments := range bySubject {
		if _, err := review.EvaluateGate(subjects[digest], assessments); err != nil {
			return errors.New("sqlite.goal_record_review_gate_invalid")
		}
	}
	return nil
}

func validateGoalRecordReviewCleanupControls(record application.GoalRecord,
	items map[goal.WorkItemRef]goal.WorkItem, executions map[goal.ExecutionRef]application.ExecutionRecord,
) error {
	for _, control := range record.Controls {
		if !application.IsReviewCleanupControl(control) {
			continue
		}
		item, itemFound := items[control.WorkItemRef]
		reviewer, reviewerFound := executions[control.ExecutionRef]
		author, authorFound := application.ExecutionRecord{}, false
		if itemFound {
			bound, boundFound := item.Execution()
			if boundFound {
				author, authorFound = executions[bound]
			}
		}
		intent, intentFound := reviewLaunchIntent(record, author)
		authority, authorityFound := reviewAuthority(record, control.WorkItemRef)
		_, reviewerRoleOK := sqliteReviewerRole(reviewer.Purpose)
		if !itemFound || !reviewerFound || !authorFound || !intentFound || !authorityFound ||
			control.GoalRef != record.Goal.Ref() || reviewer.GoalRef != control.GoalRef ||
			reviewer.WorkItemRef != control.WorkItemRef || reviewer.AttemptNo != control.ExecutionAttempt ||
			!reviewerRoleOK ||
			author.Purpose != application.ExecutionPurposeAuthor || intent.ProposedBy != control.PrincipalRef ||
			intent.Authority.Ref() != control.AuthorizationReceipt.Ref() ||
			authority.PrincipalRef != control.PrincipalRef ||
			authority.AuthorizationReceipt.Ref() != control.AuthorizationReceipt.Ref() {
			return errors.New("sqlite.goal_record_review_cleanup_authority_invalid")
		}
	}
	return nil
}

func reviewLaunchIntent(record application.GoalRecord, execution application.ExecutionRecord) (application.EffectIntent, bool) {
	for _, intent := range record.EffectIntents {
		if intent.Ref == execution.EffectIntentRef && intent.ActionKind == application.ActionLaunchAgent &&
			intent.Subject.ProjectRef == record.Goal.Project() && intent.Subject.GoalRef == record.Goal.Ref() &&
			intent.Subject.WorkItemRef == execution.WorkItemRef && intent.Subject.ExecutionRef == execution.Ref &&
			intent.Subject.PlanGeneration == execution.PlanGeneration &&
			intent.Subject.AppSpecGeneration == execution.AppSpecGeneration && intent.Subject.SpecHash == execution.SpecHash {
			return intent, true
		}
	}
	return application.EffectIntent{}, false
}

func reviewAuthority(record application.GoalRecord, ref goal.WorkItemRef) (application.WorkItemAuthority, bool) {
	for _, authority := range record.WorkItemAuthorities {
		if authority.WorkItemRef == ref {
			return authority, true
		}
	}
	return application.WorkItemAuthority{}, false
}

func persistedReviewSubject(record application.GoalRecord, item goal.WorkItem, author application.ExecutionRecord,
	change application.ChangeSet,
) (review.Subject, error) {
	var binding application.WorkspaceBinding
	bindingFound := false
	for _, candidate := range record.WorkspaceBindings {
		if candidate.ExecutionRef == author.Ref && candidate.Ref == change.WorkspaceRef {
			binding, bindingFound = candidate, true
		}
	}
	var pass application.AttestationRecord
	passFound := false
	for _, candidate := range record.Attestations {
		if candidate.Kind == application.AttestationKindRequiredTests &&
			candidate.Verdict == application.AttestationVerdictPassed && candidate.ExecutionRef == author.Ref &&
			candidate.ChangeSetRef == change.Ref {
			pass, passFound = candidate, true
		}
	}
	if !bindingFound || !passFound || author.Purpose != application.ExecutionPurposeAuthor {
		return review.Subject{}, errors.New("sqlite.review_subject_missing")
	}
	return review.NewSubject(review.Subject{
		GoalRef: author.GoalRef.String(), WorkItemRef: author.WorkItemRef.String(),
		AuthorExecutionRef: author.Ref.String(), AuthorExecutionAttempt: author.AttemptNo,
		PlanGeneration: uint64(author.PlanGeneration), WorkItemGeneration: uint64(pass.WorkItemGeneration),
		AppSpecGeneration: uint64(author.AppSpecGeneration), SpecHash: author.SpecHash,
		AuthorLaunchReceiptRef: author.LaunchReceiptRef, AuthorExternalRef: author.ExternalRef,
		WorkspaceBindingDigest: binding.Digest(),
		ChangeSetRef:           change.Ref.String(), ChangeSetDigest: change.Digest(), TreeOID: change.TreeOID,
		DiffDigest: change.DiffDigest, WriteSetDigest: change.WriteSetDigest,
		RequiredTestsDigest: pass.RequiredTestsDigest, TestAttestationRef: pass.Ref.String(),
		TestSubjectDigest: pass.SubjectDigest, TestPolicyDigest: pass.PolicyDigest,
	})
}

func v18FailedReviewPreservesCandidate(record application.GoalRecord, item goal.WorkItem,
	author application.ExecutionRecord, change application.ChangeSet,
) bool {
	subject, err := persistedReviewSubject(record, item, author, change)
	if err != nil {
		return false
	}
	digest := subject.Digest()
	latest := make(map[review.Role]application.ExecutionRecord, 2)
	for _, participant := range record.Executions {
		role, reviewer := sqliteReviewerRole(participant.Purpose)
		if !reviewer || participant.ReviewSubjectDigest != digest {
			continue
		}
		if previous, found := latest[role]; !found || participant.AttemptNo > previous.AttemptNo {
			latest[role] = participant
		}
	}
	unavailable := false
	for _, role := range []review.Role{review.RolePrimary, review.RoleAdversarial} {
		participant, found := latest[role]
		if !found {
			return false
		}
		switch participant.State {
		case application.ExecutionSucceeded:
		case application.ExecutionFailed, application.ExecutionStopped:
			unavailable = true
		default:
			return false
		}
	}
	if author.FailureCode == "review.unavailable" {
		return unavailable
	}
	if author.FailureCode != string(goal.ReplanCauseReviewChangesRequested) {
		return false
	}
	assessments := make([]review.Assessment, 0, 2)
	for _, fact := range record.Reviews {
		if fact.SubjectDigest != digest || fact.ChangeSetRef != change.Ref {
			continue
		}
		assessment, assessmentErr := review.NewAssessment(review.Assessment{
			SubjectDigest: fact.SubjectDigest, Role: fact.Role, Verdict: fact.Verdict,
			ReviewerExecutionRef:     fact.ReviewerExecutionRef.String(),
			ReviewerExecutionAttempt: fact.ReviewerExecutionAttempt,
			LaunchReceiptRef:         fact.LaunchReceiptRef, ReviewerExternalRef: fact.ExternalRef,
			AssessmentArtifactRef: fact.AssessmentArtifactRef, AssessmentDigest: fact.AssessmentDigest,
			RecordedAt: fact.RecordedAt,
		})
		if assessmentErr != nil {
			return false
		}
		assessments = append(assessments, assessment)
	}
	gate, err := review.EvaluateGate(subject, assessments)
	return err == nil && gate.Status == review.GateChangesRequested
}

func reviewChange(record application.GoalRecord, fact application.ReviewRecord) (application.ChangeSet, bool) {
	for _, change := range record.ChangeSets {
		if change.Ref == fact.ChangeSetRef && change.WorkItemRef == fact.WorkItemRef {
			return change, true
		}
	}
	return application.ChangeSet{}, false
}

func reviewArtifact(record application.GoalRecord, ref string) (application.ArtifactRecord, bool) {
	for _, artifact := range record.Artifacts {
		if artifact.Stored.Ref.String() == ref {
			return artifact, true
		}
	}
	return application.ArtifactRecord{}, false
}

func sqliteReviewerRole(purpose application.ExecutionPurpose) (review.Role, bool) {
	switch purpose {
	case application.ExecutionPurposePrimaryReview:
		return review.RolePrimary, true
	case application.ExecutionPurposeAdversarialReview:
		return review.RoleAdversarial, true
	default:
		return "", false
	}
}
