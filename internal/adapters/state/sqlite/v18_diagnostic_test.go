package sqlite

import (
	"context"
	"strings"
	"testing"

	"orquesta/internal/application"
)

func TestSQLiteMalformedReviewPersistsRedactedDiagnosticAndRecoversCleanup(t *testing.T) {
	attestor := &sqliteTestAttestor{}
	system, goalRef := seedSQLiteV17Committed(t, attestor)
	processSQLiteV16Actions(t, system,
		application.ActionAttestTest, application.ActionLaunchAgent, application.ActionLaunchAgent,
	)
	rewriteRecoveryTrigger(t, system.repository.db, "executions_identity_immutable", func() {
		mustV10Exec(t, system.repository.db, `UPDATE executions SET max_execution_attempts=1
WHERE purpose IN ('primary_review','adversarial_review')`)
	})
	const secret = `{"token":"sqlite-secret-must-not-persist"}`
	system.external.mu.Lock()
	system.external.reviewContent = []byte(secret)
	system.external.mu.Unlock()
	processSQLiteV16Actions(t, system, application.ActionObserveAgent)

	record, err := system.repository.GetGoal(context.Background(), goalRef)
	sqliteTestNoError(t, err)
	diagnostics := 0
	for _, artifact := range record.Artifacts {
		if artifact.Kind != application.ArtifactKindReviewDiagnostic {
			continue
		}
		diagnostics++
		content, readErr := system.external.Get(context.Background(), artifact.Stored.Ref, artifact.Stored.Size)
		sqliteTestNoError(t, readErr)
		if strings.Contains(string(content.Content), "sqlite-secret-must-not-persist") ||
			!strings.Contains(string(content.Content), "review.assessment_invalid") {
			t.Fatalf("unsafe diagnostic content=%s", content.Content)
		}
	}
	var persistedDiagnostics int
	sqliteTestNoError(t, system.repository.db.QueryRow(`SELECT COUNT(*) FROM artifact_occurrences
WHERE kind='review_diagnostic'`).Scan(&persistedDiagnostics))
	if diagnostics != 1 || persistedDiagnostics != 1 {
		t.Fatalf("diagnostics aggregate/sqlite=%d/%d", diagnostics, persistedDiagnostics)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("pending review cleanup recovery: %v", err)
	}
	processSQLiteV16Actions(t, system, application.ActionStopAgent)
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("completed review cleanup recovery: %s", sqliteTestErrorChain(err))
	}
	mustV10Exec(t, system.repository.db, `UPDATE executions SET failure_code='review.foreign_failure'
WHERE purpose='author' AND state='failed'`)
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err == nil ||
		!strings.Contains(sqliteTestErrorChain(err), "sqlite.goal_record_artifact_scope_invalid") {
		t.Fatalf("foreign author failure accepted staged V18 output: %s", sqliteTestErrorChain(err))
	}
}

func TestSQLiteReviewRetryHistoryPreservesRejectedCandidateEvidence(t *testing.T) {
	attestor := &sqliteTestAttestor{}
	system, goalRef := seedSQLiteV17Committed(t, attestor)
	processSQLiteV16Actions(t, system,
		application.ActionAttestTest, application.ActionLaunchAgent, application.ActionLaunchAgent,
	)
	rewriteRecoveryTrigger(t, system.repository.db, "executions_identity_immutable", func() {
		mustV10Exec(t, system.repository.db, `UPDATE executions SET max_execution_attempts=2
WHERE purpose IN ('primary_review','adversarial_review')`)
	})
	system.external.mu.Lock()
	system.external.reviewContent = []byte(`{"malformed":"attempt-one"}`)
	system.external.mu.Unlock()
	processSQLiteV16Actions(t, system, application.ActionObserveAgent)
	system.external.mu.Lock()
	system.external.reviewContent = nil
	system.external.mu.Unlock()
	processSQLiteV16Actions(t, system, application.ActionObserveAgent, application.ActionLaunchAgent)
	system.external.mu.Lock()
	system.external.reviewContent = []byte(`{"malformed":"attempt-two"}`)
	system.external.mu.Unlock()
	processSQLiteV16Actions(t, system, application.ActionObserveAgent)

	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("retry history rejected exact V18 candidate: %s", sqliteTestErrorChain(err))
	}
	record, err := system.repository.GetGoal(context.Background(), goalRef)
	sqliteTestNoError(t, err)
	reviewers, diagnostics, approvals := 0, 0, 0
	for _, execution := range record.Executions {
		if execution.Purpose == application.ExecutionPurposePrimaryReview ||
			execution.Purpose == application.ExecutionPurposeAdversarialReview {
			reviewers++
		}
	}
	for _, artifact := range record.Artifacts {
		if artifact.Kind == application.ArtifactKindReviewDiagnostic {
			diagnostics++
		}
	}
	for _, fact := range record.Reviews {
		if fact.Verdict == "approve" {
			approvals++
		}
	}
	if reviewers != 3 || diagnostics != 2 || approvals != 1 {
		t.Fatalf("retry history reviewers=%d diagnostics=%d approvals=%d", reviewers, diagnostics, approvals)
	}
}
