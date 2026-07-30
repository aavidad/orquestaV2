package sqlite

import (
	"context"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

func TestAgentQuotaAppendCurrentIsolationAndRestart(t *testing.T) {
	ctx := context.Background()
	repository, path := openTestRepository(t)
	placement, _ := ports.NewAgentPlacementRef("placement:q3-one")
	if record, found, err := repository.CurrentAgentQuotaObservation(ctx, placement); err != nil || found || record != (application.AgentQuotaObservationRecord{}) {
		t.Fatalf("lectura vacía: registro=%+v encontrado=%v error=%v", record, found, err)
	}
	at := time.Now()
	first := agentQuotaTestRecord(t, placement, "quota:q3-one", "quota-key:q3-shared", 0, at)
	saved, created, err := repository.AppendAgentQuotaObservation(ctx, first)
	sqliteTestNoError(t, err)
	canonical := first
	canonical.ObservedAt, canonical.ExpiresAt = first.ObservedAt.Round(0).UTC(), first.ExpiresAt.Round(0).UTC()
	canonical.ResetAt, canonical.RetryAt = first.ResetAt.Round(0).UTC(), first.RetryAt.Round(0).UTC()
	if !created || saved != canonical {
		t.Fatalf("registro inicial: creado=%v registro=%+v", created, saved)
	}
	replayed, created, err := repository.AppendAgentQuotaObservation(ctx, first)
	if err != nil || created || replayed != canonical {
		t.Fatalf("repetición monotónica: creado=%v registro=%+v error=%v", created, replayed, err)
	}
	var rows int
	sqliteTestNoError(t, repository.db.QueryRow(`SELECT COUNT(*) FROM agent_quota_observations WHERE placement_ref=?`, placement.String()).Scan(&rows))
	if rows != 1 {
		t.Fatalf("la repetición creó %d filas", rows)
	}
	requireConflict := func(record application.AgentQuotaObservationRecord, reason string) {
		t.Helper()
		if _, _, err := repository.AppendAgentQuotaObservation(ctx, record); !application.IsStateError(err, application.StateConflict) ||
			!recoveryErrorContains(err, "sqlite.agent_quota_observation_conflict") {
			t.Fatalf("%s aceptado: %v", reason, err)
		}
	}
	divergent := first
	divergent.Status = application.AgentQuotaExhausted
	requireConflict(divergent, "contenido divergente con la misma clave")
	requireConflict(agentQuotaTestRecord(t, placement, "quota:q3-stale", "quota-key:q3-stale", 0, at), "revisión obsoleta")
	requireConflict(agentQuotaTestRecord(t, placement, "quota:q3-gap", "quota-key:q3-gap", 2, at), "salto de revisión")
	second := agentQuotaTestRecord(t, placement, "quota:q3-two", "quota-key:q3-two", 1, at.Add(time.Minute))
	second, created, err = repository.AppendAgentQuotaObservation(ctx, second)
	if err != nil || !created {
		t.Fatalf("segunda revisión: creada=%v error=%v", created, err)
	}
	otherPlacement, _ := ports.NewAgentPlacementRef("placement:q3-other")
	other := agentQuotaTestRecord(t, otherPlacement, "quota:q3-other", first.IdempotencyKey, 0, at)
	other, created, err = repository.AppendAgentQuotaObservation(ctx, other)
	if err != nil || !created {
		t.Fatalf("colocación aislada: creada=%v error=%v", created, err)
	}
	current, found, err := repository.CurrentAgentQuotaObservation(ctx, placement)
	if err != nil || !found || current != second {
		t.Fatalf("cuota actual: encontrada=%v registro=%+v error=%v", found, current, err)
	}
	otherCurrent, found, err := repository.CurrentAgentQuotaObservation(ctx, otherPlacement)
	if err != nil || !found || otherCurrent != other {
		t.Fatalf("cuota aislada: encontrada=%v registro=%+v error=%v", found, otherCurrent, err)
	}
	sqliteTestNoError(t, repository.Close())
	reopened := openFastTestRepository(t, Options{Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4})
	current, found, err = reopened.CurrentAgentQuotaObservation(ctx, placement)
	if err != nil || !found || current != second {
		t.Fatalf("reinicio: encontrada=%v registro=%+v error=%v", found, current, err)
	}
	if _, _, err := reopened.AppendAgentQuotaObservation(ctx, application.AgentQuotaObservationRecord{}); !application.IsStateError(err, application.StateInvalid) {
		t.Fatalf("registro sin materializar aceptado: %v", err)
	}
	if _, _, err := reopened.CurrentAgentQuotaObservation(ctx, ports.AgentPlacementRef{}); !application.IsStateError(err, application.StateInvalid) {
		t.Fatalf("colocación vacía aceptada: %v", err)
	}
}

func agentQuotaTestRecord(t *testing.T, placement ports.AgentPlacementRef, ref, key string, expected uint64, at time.Time) application.AgentQuotaObservationRecord {
	t.Helper()
	evidence, err := goal.NewArtifactRef("artifact:q3-evidence")
	sqliteTestNoError(t, err)
	record, err := application.MaterializeAgentQuotaObservation(application.AgentQuotaObservationSubmission{
		AgentQuotaObservation: application.AgentQuotaObservation{
			PlacementRef: placement, WindowRef: application.AgentQuotaWindowRef("quota-window:q3"),
			Status: application.AgentQuotaAvailable, Quality: governance.UsageQualityMeasured,
			ObservedAt: at, ExpiresAt: at.Add(time.Hour), ResetAt: at.Add(45 * time.Minute),
			RetryAt: at.Add(30 * time.Minute), EvidenceRef: evidence,
		},
		Ref: ref, IdempotencyKey: key,
	}, expected)
	sqliteTestNoError(t, err)
	return record
}
