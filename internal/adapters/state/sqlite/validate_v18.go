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
	for _, control := range record.Controls {
		if application.IsReviewCleanupControl(control) &&
			application.ValidatePersistedReviewCleanupControl(record, control) != nil {
			return errors.New("sqlite.goal_record_review_cleanup_authority_invalid")
		}
	}
	for _, fact := range record.Reviews {
		item, found := items[fact.WorkItemRef]
		subject, assessment, err := application.ValidatePersistedReviewRecord(record, item, fact)
		if !found || err != nil {
			return errors.New("sqlite.goal_record_review_scope_invalid")
		}
		key := fact.SubjectDigest + "\x00" + string(fact.Role)
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
