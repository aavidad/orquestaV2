package sqlite

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/intake"
)

func TestIntakeSQLiteDerivationIdentitySurvivesRestartAndReplay(t *testing.T) {
	ctx := context.Background()
	system := newSQLiteIntakeTestSystem(t)
	created := system.create(t, "request:intake:derived-create")
	identity, err := intake.NewDerivationIdentity(
		"orquesta.test.deriver",
		"v1",
		strings.Repeat("f", 64),
	)
	sqliteTestNoError(t, err)
	change := sqliteIntakeAudienceChange(
		created.Record.State,
		system.stateRef,
		intake.OriginChat,
		"intake-option:audience-personal",
	)
	change.Derivation = identity
	request := application.ApplyIntakeRequest{
		RequestRef: "request:intake:derived-apply",
		ActorRef:   system.principal.ActorRef, ProjectRef: system.project,
		Change: change,
		AuthorizationReceipt: system.authorizeIntake(
			t,
			application.IntakeOperationApply,
			"request:intake:derived-apply",
		),
	}
	first, err := system.service.ApplyIntake(ctx, request)
	sqliteTestNoError(t, err)
	if !first.Changed {
		t.Fatal("derived mutation was not persisted")
	}

	sqliteTestNoError(t, system.repository.Close())
	system.repository = openSQLiteIntakeTestRepository(t, system.path)
	t.Cleanup(func() { _ = system.repository.Close() })
	system.service, err = application.NewIntakeService(system.repository)
	sqliteTestNoError(t, err)

	current, err := system.service.GetIntake(ctx, application.GetIntakeRequest{
		ActorRef: system.principal.ActorRef, ProjectRef: system.project,
		StateRef: system.stateRef,
	})
	sqliteTestNoError(t, err)
	history := current.State.History()
	if len(history) != 1 ||
		history[0].Derivation != identity ||
		!reflect.DeepEqual(
			application.SnapshotIntake(current.State),
			application.SnapshotIntake(first.Record.State),
		) {
		t.Fatalf("restart history=%+v", history)
	}
	replayed, err := system.service.ApplyIntake(ctx, request)
	sqliteTestNoError(t, err)
	if replayed.Changed || replayed.Record.Receipt != first.Record.Receipt {
		t.Fatalf(
			"first=%+v replayed=%+v",
			first.Record.Receipt,
			replayed.Record.Receipt,
		)
	}
}
