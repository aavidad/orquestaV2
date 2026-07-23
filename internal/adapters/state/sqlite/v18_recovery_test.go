package sqlite

import (
	"context"
	"strings"
	"testing"

	"orquesta/internal/application"
)

func TestReviewRecoveryRejectsMissingDuplicateAndCrossProjectFacts(t *testing.T) {
	tests := []struct {
		name    string
		trigger string
		mutate  string
	}{
		{
			name: "active reviewer subject", trigger: "executions_identity_immutable",
			mutate: `UPDATE executions SET review_subject_digest='sha256:` + strings.Repeat("1", 64) + `'
WHERE purpose='primary_review'`,
		},
		{
			name: "candidate tree", trigger: "change_sets_immutable_update",
			mutate: `UPDATE change_sets SET tree_oid='` + strings.Repeat("2", 40) + `'`,
		},
		{
			name: "passed test subject", trigger: "attestations_immutable_update",
			mutate: `UPDATE attestations SET subject_digest='` + strings.Repeat("3", 64) + `'
WHERE kind='required_tests'`,
		},
		{
			name: "reviewer launch target", trigger: "effect_intents_immutable_update",
			mutate: `UPDATE effect_intents SET target_digest='` + strings.Repeat("4", 64) + `'
WHERE execution_ref IN (SELECT ref FROM executions WHERE purpose='adversarial_review')`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			attestor := &sqliteTestAttestor{}
			system, _ := seedSQLiteV17Committed(t, attestor)
			processSQLiteV16Actions(t, system, application.ActionAttestTest)
			if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
				t.Fatalf("valid active review frontier: %v", err)
			}
			rewriteRecoveryTrigger(t, system.repository.db, test.trigger, func() {
				mustV10Exec(t, system.repository.db, test.mutate)
			})
			if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err == nil {
				t.Fatal("tampered active review frontier passed recovery")
			}
		})
	}
}

func TestReviewIntegrationRecoveryRejectsGateOrTargetDrift(t *testing.T) {
	for _, test := range []struct {
		name    string
		trigger string
		mutate  string
	}{
		{
			name: "gate digest", trigger: "outbox_integration_admission_immutable",
			mutate: `UPDATE outbox SET review_gate_digest='sha256:` + strings.Repeat("5", 64) + `'
WHERE kind='integrate_change' AND completed_at IS NULL`,
		},
		{
			name: "effect target", trigger: "effect_intents_immutable_update",
			mutate: `UPDATE effect_intents SET target_digest='` + strings.Repeat("6", 64) + `'
WHERE action_kind='integrate_change'`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			system := seedSQLiteV18Integration(t, false)
			if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
				t.Fatalf("valid integration review gate: %s", sqliteTestErrorChain(err))
			}
			rewriteRecoveryTrigger(t, system.repository.db, test.trigger, func() {
				mustV10Exec(t, system.repository.db, test.mutate)
			})
			if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err == nil {
				t.Fatal("tampered integration review gate passed recovery")
			}
		})
	}
}
