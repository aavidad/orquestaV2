package sqlite

import (
	"context"
	"database/sql"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/council"
	"orquesta/internal/goal"
)

func sqliteTestNoError(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func seedSQLiteV17Committed(
	t *testing.T,
	attestor *sqliteTestAttestor,
) (*sqliteV15System, goal.GoalRef) {
	t.Helper()
	system := newSQLiteV15System(t, 4)
	system.orchestrator = newSQLiteV16OrchestratorWithAttestor(t, system, attestor)
	result, err := system.orchestrator.Submit(context.Background(), system.access, application.SubmitRequest{
		RequestRef: "request:v17-test-attestor", Statement: "persist exact test attestation evidence", Confirm: true,
		Plan: &application.PlanSpec{
			Phases: []application.PhaseSpec{{
				Ref: "phase-instance:v17-test-attestor", Key: "phase:v17-test-attestor",
				TemplateRef: "phase-template:v17-test-attestor",
			}},
			WorkItems: []application.WorkItemSpec{{
				Key: "writer", Objective: "produce a tested isolated change",
				Phase: "phase:v17-test-attestor", Role: "role:writer",
				WriteSet:       []string{"internal/v17"},
				CouncilPolicy:  council.PolicyAuto,
				RequiredTests:  sqliteRequiredTestSpecs("required-test:v17-exact"),
				OutputContract: goal.OutputContractEvidenceBundle,
			}},
		},
	})
	sqliteTestNoError(t, err)
	processSQLiteV16Actions(t, system,
		application.ActionPrepareWorkspace,
		application.ActionLaunchAgent,
		application.ActionObserveAgent,
		application.ActionCommitChange,
	)
	record, err := system.repository.GetGoal(context.Background(), result.Record.Goal.Ref())
	if err != nil || len(record.Executions) != 1 ||
		record.Executions[0].State != application.ExecutionAwaitingAttestation ||
		len(record.ChangeSets) != 1 || len(record.Goal.WorkItems()[0].RequiredTests()) != 1 {
		t.Fatalf("V17 committed fixture incomplete: record=%+v err=%v", record, err)
	}
	return system, record.Goal.Ref()
}

func assertSQLiteV17Pass(t *testing.T, repository *Repository, goalRef goal.GoalRef) {
	t.Helper()
	record, err := repository.GetGoal(context.Background(), goalRef)
	if err != nil {
		t.Fatalf("read V17 pass: %s", sqliteTestErrorChain(err))
	}
	authors, reviewers := 0, 0
	for _, execution := range record.Executions {
		switch {
		case execution.Purpose == application.ExecutionPurposeAuthor &&
			execution.State == application.ExecutionAwaitingIntegration:
			authors++
		case (execution.Purpose == application.ExecutionPurposePrimaryReview ||
			execution.Purpose == application.ExecutionPurposeAdversarialReview) &&
			(execution.State == application.ExecutionQueued || execution.State == application.ExecutionRunning):
			reviewers++
		}
	}
	if len(record.Executions) != 3 || authors != 1 || reviewers != 2 ||
		len(record.Artifacts) != 3 || len(record.Attestations) != 2 {
		t.Fatalf("V17 pass frontier executions=%+v artifacts=%d attestations=%+v",
			record.Executions, len(record.Artifacts), record.Attestations)
	}
	typed := 0
	for _, attestation := range record.Attestations {
		if attestation.Kind != application.AttestationKindRequiredTests {
			continue
		}
		typed++
		if attestation.Verdict != application.AttestationVerdictPassed || len(attestation.Tests) != 1 ||
			attestation.Tests[0].ExitCode != 0 || attestation.SubjectDigest == "" ||
			attestation.ManifestArtifactRef.String() == "" || attestation.ReportArtifactRef.String() == "" ||
			attestation.EffectReceiptRef == "" {
			t.Fatalf("typed V17 attestation incomplete: %+v", attestation)
		}
	}
	if typed != 1 {
		t.Fatalf("typed V17 attestations=%d want=1", typed)
	}
	assertSQLiteV17TerminalCounts(t, repository.db, 1, 1, 1)
}

func assertSQLiteV17TerminalCounts(
	t *testing.T,
	database *sql.DB,
	attestations, outcomes, terminalReceipts int,
) {
	t.Helper()
	queries := []struct {
		name  string
		query string
		want  int
	}{
		{"attestations", `SELECT COUNT(*) FROM attestations WHERE kind='required_tests'`, attestations},
		{"outcomes", `SELECT COUNT(*) FROM attestation_test_outcomes`, outcomes},
		{"receipts", `SELECT COUNT(*) FROM effect_receipts WHERE status IN ('attested_passed','attested_failed')`, terminalReceipts},
		{"consumptions", `SELECT COUNT(*) FROM action_consumption_receipts WHERE kind='attest_test'`, terminalReceipts},
	}
	for _, query := range queries {
		var got int
		if err := database.QueryRow(query.query).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != query.want {
			t.Fatalf("V17 terminal %s=%d want=%d", query.name, got, query.want)
		}
	}
}

func assertSQLiteV17UnknownAppliedCounts(t *testing.T, database *sql.DB) {
	t.Helper()
	for name, query := range map[string]string{
		"attestations": `SELECT COUNT(*) FROM attestations WHERE kind='required_tests'`,
		"outcomes":     `SELECT COUNT(*) FROM attestation_test_outcomes`,
		"receipts":     `SELECT COUNT(*) FROM effect_receipts WHERE status IN ('attested_passed','attested_failed')`,
	} {
		var got int
		if err := database.QueryRow(query).Scan(&got); err != nil || got != 0 {
			t.Fatalf("unknown-applied %s=%d want=0 err=%v", name, got, err)
		}
	}
	var consumptions int
	if err := database.QueryRow(`SELECT COUNT(*) FROM action_consumption_receipts
WHERE kind='attest_test' AND outcome='quarantined'
 AND error_code='application.effect_unknown_applied'`).Scan(&consumptions); err != nil || consumptions != 1 {
		t.Fatalf("unknown-applied consumptions=%d want=1 err=%v", consumptions, err)
	}
}
