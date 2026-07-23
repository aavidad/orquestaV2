package sqlite

import (
	"context"
	"strings"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/council"
)

func TestSQLiteV19CouncilRecoveryRejectsDecisionDigestCorruption(t *testing.T) {
	system, _ := seedSQLiteV19Council(t, council.PolicyAuto, false)
	processSQLiteV16Actions(t, system,
		application.ActionLaunchAgent, application.ActionLaunchAgent, application.ActionLaunchAgent,
		application.ActionObserveAgent, application.ActionObserveAgent, application.ActionObserveAgent)
	rewriteRecoveryTrigger(t, system.repository.db, "council_decisions_immutable_update", func() {
		mustV10Exec(t, system.repository.db, `UPDATE council_decisions SET decision_digest='sha256:`+
			strings.Repeat("f", 64)+`'`)
	})
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err == nil {
		t.Fatal("corrupted Council decision passed recovery")
	}
}

func TestSQLiteV19CouncilRecoveryRejectsFakeReviewGate(t *testing.T) {
	system, _ := seedSQLiteV19Council(t, council.PolicyAuto, false)
	rewriteRecoveryTrigger(t, system.repository.db, "council_rounds_immutable_update", func() {
		mustV10Exec(t, system.repository.db, `UPDATE council_rounds SET review_gate_digest='sha256:`+
			strings.Repeat("e", 64)+`'`)
	})
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err == nil {
		t.Fatal("Council round with fake V18 gate passed recovery")
	}
}
