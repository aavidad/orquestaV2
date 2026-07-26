package sqlite

import (
	"context"
	"database/sql"
	"reflect"
	"sync"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
	"orquesta/internal/intake"
)

func TestIntakeDossierSQLiteRestartAndHistoricalReplay(t *testing.T) {
	ctx := context.Background()
	system := newSQLiteIntakeDossierTestSystem(t)
	request, first := system.prepare(t, "request:intake-dossier:restart")
	_ = system.apply(
		t, system.current.State, "request:intake-dossier:later-intake",
		intake.OriginChat, intake.OptionRef("intake-option:audience-personal"),
	)

	sqliteTestNoError(t, system.repository.Close())
	system.repository = openSQLiteIntakeTestRepository(t, system.path)
	service, err := application.NewIntakeDossierService(system.repository, system.repository)
	sqliteTestNoError(t, err)
	system.dossiers = service

	replayed, err := system.dossiers.PrepareIntakeDossier(ctx, request)
	sqliteTestNoError(t, err)
	if replayed.Created || replayed.Record.Receipt != first.Record.Receipt ||
		!reflect.DeepEqual(
			application.SnapshotIntakeDossier(replayed.Record.Dossier),
			application.SnapshotIntakeDossier(first.Record.Dossier),
		) {
		t.Fatalf("historical replay mismatch: created=%v receipt=%+v", replayed.Created, replayed.Record.Receipt)
	}
	got, err := system.dossiers.GetIntakeDossier(ctx, application.GetIntakeDossierRequest{
		ActorRef: system.principal.ActorRef, ProjectRef: system.project,
		DossierRef: first.Record.Dossier.Ref(),
	})
	sqliteTestNoError(t, err)
	if got.Receipt != first.Record.Receipt ||
		!reflect.DeepEqual(
			application.SnapshotIntakeDossier(got.Dossier),
			application.SnapshotIntakeDossier(first.Record.Dossier),
		) {
		t.Fatal("restart get changed immutable dossier")
	}
	assertIntakeDossierCounts(t, system.repository, 1, 1)
}

func TestIntakeDossierSQLiteDivergentReplayAndScopeIsolation(t *testing.T) {
	ctx := context.Background()
	system := newSQLiteIntakeDossierTestSystem(t)
	request, result := system.prepare(t, "request:intake-dossier:scope")

	request.Input.Objective += " divergente"
	if _, err := system.dossiers.PrepareIntakeDossier(ctx, request); !application.IsStateError(
		err, application.StateConflict,
	) {
		t.Fatalf("divergent replay = %v", err)
	}
	otherProject := mustRef(t, "project:intake-dossier-other", goal.NewProjectRef)
	otherPrincipal := testPrincipal(
		t, "principal:intake-dossier-other", "actor:intake-dossier-other",
		identity.PrincipalKindHuman,
	)
	provisionTestAccess(
		t, system.repository, otherPrincipal, otherProject, identity.RoleProjectOwner, system.now,
	)
	if _, err := system.repository.GetIntakeDossier(
		ctx, system.principal.ActorRef, otherProject, result.Record.Dossier.Ref(),
	); !application.IsStateError(err, application.StateNotFound) {
		t.Fatalf("cross-project get = %v", err)
	}
	if _, err := system.repository.GetIntakeDossier(
		ctx, otherPrincipal.ActorRef, system.project, result.Record.Dossier.Ref(),
	); !application.IsStateError(err, application.StateNotFound) {
		t.Fatalf("cross-actor get = %v", err)
	}
	assertIntakeDossierCounts(t, system.repository, 1, 1)
}

func TestIntakeDossierSQLiteReusesContentAndRecordsDistinctRequests(t *testing.T) {
	system := newSQLiteIntakeDossierTestSystem(t)
	_, first := system.prepare(t, "request:intake-dossier:content-a")
	_, second := system.prepare(t, "request:intake-dossier:content-b")

	if !second.Created || first.Record.Dossier.Ref() != second.Record.Dossier.Ref() ||
		first.Record.Receipt.Ref == second.Record.Receipt.Ref {
		t.Fatalf(
			"shared content invalid: created=%v dossier=%q/%q receipt=%q/%q",
			second.Created, first.Record.Dossier.Ref(), second.Record.Dossier.Ref(),
			first.Record.Receipt.Ref, second.Record.Receipt.Ref,
		)
	}
	got, err := system.repository.GetIntakeDossier(
		context.Background(), system.principal.ActorRef, system.project,
		first.Record.Dossier.Ref(),
	)
	sqliteTestNoError(t, err)
	if got.Receipt != first.Record.Receipt ||
		!reflect.DeepEqual(
			application.SnapshotIntakeDossier(got.Dossier),
			application.SnapshotIntakeDossier(first.Record.Dossier),
		) {
		t.Fatalf(
			"Get changed creator receipt or content: receipt=%q want=%q",
			got.Receipt.Ref, first.Record.Receipt.Ref,
		)
	}
	assertIntakeDossierCounts(t, system.repository, 1, 2)
}

func TestIntakeDossierSQLiteConcurrentReplayCreatesOneReceipt(t *testing.T) {
	system := newSQLiteIntakeDossierTestSystem(t)
	request := system.prepareRequest(t, "request:intake-dossier:race")
	type outcome struct {
		result application.IntakeDossierResult
		err    error
	}
	start := make(chan struct{})
	outcomes := make(chan outcome, 2)
	var wait sync.WaitGroup
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			result, err := system.dossiers.PrepareIntakeDossier(context.Background(), request)
			outcomes <- outcome{result: result, err: err}
		}()
	}
	close(start)
	wait.Wait()
	close(outcomes)
	var created, replayed int
	for outcome := range outcomes {
		if outcome.err != nil {
			t.Fatal(outcome.err)
		}
		if outcome.result.Created {
			created++
		} else {
			replayed++
		}
	}
	if created != 1 || replayed != 1 {
		t.Fatalf("concurrent dossier created=%d replayed=%d", created, replayed)
	}
	assertIntakeDossierCounts(t, system.repository, 1, 1)
}

func TestIntakeDossierSQLiteRejectsStaleSourceAndAuthorizationWithoutRows(t *testing.T) {
	t.Run("source", func(t *testing.T) {
		system := newSQLiteIntakeDossierTestSystem(t)
		state := system.createState(t, "request:intake-dossier:stale-source")
		_ = system.apply(
			t, system.current.State, "request:intake-dossier:advance",
			intake.OriginChat, intake.OptionRef("intake-option:audience-personal"),
		)
		_, created, err := system.repository.CreateIntakeDossier(context.Background(), state)
		if !application.IsStateError(err, application.StateConflict) || created {
			t.Fatalf("stale source = created:%v err:%v", created, err)
		}
		assertIntakeDossierCounts(t, system.repository, 0, 0)
	})
	t.Run("authorization", func(t *testing.T) {
		system := newSQLiteIntakeDossierTestSystem(t)
		state := system.createState(t, "request:intake-dossier:stale-auth")
		_, err := system.repository.db.Exec(`
UPDATE project_memberships
SET revision=revision+1, status='revoked', revoked_by_ref=?, revoked_at=?
WHERE principal_ref=? AND project_ref=?`,
			system.principal.Ref.String(), system.now.Add(time.Minute).UnixNano(),
			system.principal.Ref.String(), system.project.String(),
		)
		sqliteTestNoError(t, err)
		_, created, err := system.repository.CreateIntakeDossier(context.Background(), state)
		if !application.IsStateError(err, application.StateConflict) || created {
			t.Fatalf("stale authorization = created:%v err:%v", created, err)
		}
		assertIntakeDossierCounts(t, system.repository, 0, 0)
	})
}

func TestIntakeDossierSQLiteRowsAreImmutable(t *testing.T) {
	system := newSQLiteIntakeDossierTestSystem(t)
	_, result := system.prepare(t, "request:intake-dossier:immutable")
	if _, err := system.repository.db.Exec(
		`UPDATE intake_dossiers SET plan_digest=? WHERE ref=?`,
		result.Record.Dossier.StateDigest(), result.Record.Dossier.Ref(),
	); err == nil {
		t.Fatal("mutable dossier accepted")
	}
	if _, err := system.repository.db.Exec(
		`DELETE FROM intake_dossier_generation_receipts WHERE ref=?`,
		result.Record.Receipt.Ref,
	); err == nil {
		t.Fatal("mutable dossier receipt accepted")
	}
	assertIntakeDossierCounts(t, system.repository, 1, 1)
}

func TestIntakeDossierMigration018RollsBackPartialSchema(t *testing.T) {
	ctx := context.Background()
	migrations, err := loadMigrations()
	sqliteTestNoError(t, err)
	database, err := sql.Open(driverName, t.TempDir()+"/migration.db")
	sqliteTestNoError(t, err)
	t.Cleanup(func() { _ = database.Close() })
	database.SetMaxOpenConns(1)
	sqliteTestNoError(
		t,
		applyRecoveryMigrationPrefix(
			ctx, database, migrations[:recoverySchemaV23Intake],
		),
	)
	_, err = database.Exec(
		`CREATE TABLE intake_dossier_generation_receipts(blocker TEXT)`,
	)
	sqliteTestNoError(t, err)
	if err := applyMigrations(ctx, database); err == nil {
		t.Fatal("conflicting migration accepted")
	}
	var version, receipt, dossiers int
	sqliteTestNoError(t, database.QueryRow(`PRAGMA user_version`).Scan(&version))
	sqliteTestNoError(t, database.QueryRow(
		`SELECT COUNT(*) FROM schema_migrations WHERE version=?`, recoverySchemaV23,
	).Scan(&receipt))
	sqliteTestNoError(t, database.QueryRow(`
SELECT COUNT(*) FROM sqlite_schema
WHERE type='table' AND name='intake_dossiers'`).Scan(&dossiers))
	if version != recoverySchemaV23Intake || receipt != 0 || dossiers != 0 {
		t.Fatalf(
			"partial migration escaped: version=%d receipt=%d dossiers=%d",
			version, receipt, dossiers,
		)
	}
}

type sqliteIntakeDossierTestSystem struct {
	*sqliteIntakeTestSystem
	dossiers *application.IntakeDossierService
	current  application.IntakeRecord
}

func newSQLiteIntakeDossierTestSystem(t *testing.T) *sqliteIntakeDossierTestSystem {
	t.Helper()
	intakes := newSQLiteIntakeTestSystem(t)
	service, err := application.NewIntakeDossierService(intakes.repository, intakes.repository)
	sqliteTestNoError(t, err)
	created := intakes.create(t, "request:intake-dossier:source-create")
	first := intakes.apply(
		t, created.Record.State, "request:intake-dossier:source-first",
		intake.OriginChat, intake.OptionRef("intake-option:audience-personal"),
	)
	second := intakes.apply(
		t, first.Record.State, "request:intake-dossier:source-second",
		intake.OriginForm, intake.OptionRef("intake-option:audience-team"),
	)
	return &sqliteIntakeDossierTestSystem{
		sqliteIntakeTestSystem: intakes, dossiers: service, current: second.Record,
	}
}

func (system *sqliteIntakeDossierTestSystem) prepare(
	t *testing.T,
	requestRef string,
) (application.PrepareIntakeDossierRequest, application.IntakeDossierResult) {
	t.Helper()
	request := system.prepareRequest(t, requestRef)
	result, err := system.dossiers.PrepareIntakeDossier(context.Background(), request)
	if err != nil {
		t.Fatalf("prepare dossier: %s", intakeTestErrorChain(err))
	}
	if !result.Created {
		t.Fatal("first dossier request was replayed")
	}
	return request, result
}

func (system *sqliteIntakeDossierTestSystem) prepareRequest(
	t *testing.T,
	requestRef string,
) application.PrepareIntakeDossierRequest {
	t.Helper()
	authorizationRequestRef, err := application.IntakeDossierAuthorizationRequestRef(requestRef)
	sqliteTestNoError(t, err)
	return application.PrepareIntakeDossierRequest{
		RequestRef: requestRef, ActorRef: system.principal.ActorRef,
		ProjectRef: system.project, StateRef: system.current.State.Ref(),
		ExpectedRevision:       system.current.State.Revision(),
		SourceIntakeReceiptRef: system.current.Receipt.Ref,
		Plan:                   sqliteIntakeDossierPlan(),
		Input:                  sqliteIntakeDossierInput(),
		AuthorizationReceipt:   system.authorize(t, authorizationRequestRef),
	}
}

func (system *sqliteIntakeDossierTestSystem) createState(
	t *testing.T,
	requestRef string,
) application.IntakeDossierCreateState {
	t.Helper()
	request := system.prepareRequest(t, requestRef)
	dossier, err := application.BuildIntakeDossier(system.current, request.Plan, request.Input)
	sqliteTestNoError(t, err)
	fingerprint := application.IntakeDossierGenerationFingerprint(
		dossier, request.AuthorizationReceipt.Ref(),
	)
	receipt, err := application.BuildIntakeDossierGenerationReceipt(
		request.RequestRef, fingerprint, dossier, request.AuthorizationReceipt.Ref(),
	)
	sqliteTestNoError(t, err)
	return application.IntakeDossierCreateState{
		RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		ActorRef: request.ActorRef, ProjectRef: request.ProjectRef,
		ExpectedRevision: request.ExpectedRevision, SourceIntakeReceiptRef: request.SourceIntakeReceiptRef,
		AuthorizationReceipt: request.AuthorizationReceipt, Dossier: dossier, Receipt: receipt,
	}
}

func sqliteIntakeDossierPlan() application.PlanSpec {
	return application.PlanSpec{
		Phases: []application.PhaseSpec{{
			Ref: "phase-instance:intake-dossier", Key: "phase.intake-dossier",
			TemplateRef: "phase-template:intake-dossier",
			InputRefs: []string{
				"input:intake-dossier",
			},
			CriterionRefs: []string{"criterion:intake-dossier-reviewed"},
		}},
		WorkItems: []application.WorkItemSpec{{
			Key: "work:dossier", Objective: "Crear aplicación confirmada",
			Phase: "phase.intake-dossier", Role: goal.DefaultRoleKey().String(),
			WriteSet:      []string{"internal"},
			CouncilPolicy: council.PolicyRequired,
			RequiredTests: []application.RequiredTestSpec{{
				Ref: "required-test:intake-dossier", ToolRef: "tool:go-test",
				Arguments: []string{"./..."}, WorkingDirectory: ".",
			}},
			SkillRefs: []string{"skill:hexagonal"}, ToolRefs: []string{"tool:git"},
			CapabilityRefs: []string{"WIZ-24"},
			OutputContract: goal.OutputContractEvidenceBundle,
			BudgetDemand: governance.BudgetDemand{
				Ref: "budget-demand:intake-dossier",
				Resources: governance.ResourceVector{
					Tokens: 1000, MoneyMicros: 50, Currency: governance.Currency("EUR"),
					ActiveTimeNS: int64(time.Second), ProcessSlots: 1, DiskBytes: 1024,
				},
			},
			SecurityCriticality: governance.SecurityCriticalitySensitive,
			ReasoningEffort:     governance.ReasoningEffortHigh,
		}},
	}
}

func sqliteIntakeDossierInput() application.IntakeDossierInput {
	kinds := []application.IntakeDossierSectionKind{
		application.IntakeDossierSectionProductScope,
		application.IntakeDossierSectionUsersRoles,
		application.IntakeDossierSectionArchitecture,
		application.IntakeDossierSectionData,
		application.IntakeDossierSectionIntegrations,
		application.IntakeDossierSectionSecurityPrivacy,
		application.IntakeDossierSectionUIUX,
		application.IntakeDossierSectionI18NL10N,
		application.IntakeDossierSectionDeployOperations,
		application.IntakeDossierSectionOrchestrationPlan,
		application.IntakeDossierSectionRisksOpenIssues,
	}
	sections := make([]application.IntakeDossierSection, 0, len(kinds))
	for _, kind := range kinds {
		sections = append(sections, application.IntakeDossierSection{
			Ref:  application.IntakeDossierSectionRef("intake-dossier-section:" + string(kind)),
			Kind: kind, TitleKey: intake.MessageKey("intake.dossier." + string(kind) + ".title"),
			Markdown: "Contenido " + string(kind),
		})
	}
	purposes := []application.IntakeDossierDiagramPurpose{
		application.IntakeDossierDiagramArchitecture,
		application.IntakeDossierDiagramUserFlow,
		application.IntakeDossierDiagramDataIntegrations,
		application.IntakeDossierDiagramI18N,
		application.IntakeDossierDiagramDeployment,
	}
	diagrams := make([]application.IntakeDossierDiagram, 0, len(purposes))
	for _, purpose := range purposes {
		diagrams = append(diagrams, application.IntakeDossierDiagram{
			Ref:     application.IntakeDossierDiagramRef("intake-dossier-diagram:" + string(purpose)),
			Purpose: purpose, Kind: application.IntakeDossierDiagramMermaid,
			Source: "graph TD; A-->B",
			AltTextKey: intake.MessageKey(
				"intake.dossier.diagram." + string(purpose) + ".alt",
			),
		})
	}
	return application.IntakeDossierInput{
		Statement: "Crear producto", Objective: "Producto verificable",
		Sections: sections, Diagrams: diagrams,
		RiskRefs: []application.IntakeRiskRef{"intake-risk:delivery"},
	}
}

func assertIntakeDossierCounts(
	t *testing.T,
	repository *Repository,
	dossiers int,
	receipts int,
) {
	t.Helper()
	if got := tableCount(t, repository, "intake_dossiers"); got != dossiers {
		t.Fatalf("dossier count=%d want=%d", got, dossiers)
	}
	if got := tableCount(t, repository, "intake_dossier_generation_receipts"); got != receipts {
		t.Fatalf("dossier receipt count=%d want=%d", got, receipts)
	}
	if goals := tableCount(t, repository, "goals"); goals != 0 {
		t.Fatalf("dossier persistence created %d Goals", goals)
	}
}
