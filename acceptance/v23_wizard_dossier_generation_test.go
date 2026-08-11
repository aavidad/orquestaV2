package acceptance_test

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"orquesta/internal/adapters/state/sqlite"
	"orquesta/internal/application"
	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/intake"
)

func TestV23DossierGenerationContractUsesCanonicalProjection(t *testing.T) {
	ctx := context.Background()
	repository, err := sqlite.Open(ctx, sqlite.Options{
		Path:        filepath.Join(t.TempDir(), "state", "v23-dossier.db"),
		BusyTimeout: time.Second, MaxOpenConnections: 4,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repository.Close() })
	service, err := application.NewIntakeService(repository)
	if err != nil {
		t.Fatal(err)
	}
	principal, project := v06Principal(t), v06Project(t)
	actor := principal.ActorRef
	at := time.Date(2026, 8, 11, 8, 0, 0, 0, time.UTC)
	if err := repository.ProvisionLocalAccess(ctx, principal, v06Hierarchy(t),
		identity.RoleProjectOwner, at); err != nil {
		t.Fatal(err)
	}
	created, err := service.CreateIntake(ctx, application.CreateIntakeRequest{
		RequestRef: "request:v23-dossier-create", ActorRef: actor,
		ProjectRef: project, StateRef: "intake:v23-dossier",
		Policy: intake.Policy{MaxQuestionRounds: 2},
		AuthorizationReceipt: v23DossierAuthorization(t, repository, principal, project,
			application.IntakeOperationCreate, "request:v23-dossier-create"),
	})
	if err != nil {
		t.Fatal(err)
	}
	questions, err := service.ApplyIntake(ctx, application.ApplyIntakeRequest{
		RequestRef: "request:v23-dossier-questions", ActorRef: actor,
		ProjectRef: project, Change: intake.Change{
			StateRef: created.Record.State.Ref(), ExpectedRevision: 1,
			Origin: intake.OriginChat, Issues: []intake.Issue{v23AudienceGap()},
			Questions: []intake.Question{v23AudienceQuestion(true, false)},
		}, AuthorizationReceipt: v23DossierAuthorization(t, repository, principal, project,
			application.IntakeOperationApply, "request:v23-dossier-questions"),
	})
	if err != nil {
		t.Fatal(err)
	}
	answered, err := service.ApplyIntake(ctx, application.ApplyIntakeRequest{
		RequestRef: "request:v23-dossier-answer", ActorRef: actor,
		ProjectRef: project, Change: intake.Change{
			StateRef: questions.Record.State.Ref(), ExpectedRevision: 2,
			Origin: intake.OriginForm, Choices: []intake.Choice{{
				QuestionRef: "intake-question:audience",
				OptionRef:   "intake-option:audience-personal",
			}},
		}, AuthorizationReceipt: v23DossierAuthorization(t, repository, principal, project,
			application.IntakeOperationApply, "request:v23-dossier-answer"),
	})
	if err != nil {
		t.Fatal(err)
	}

	before, _ := application.IntakeStateDigest(answered.Record.State)
	first, err := application.NewIntakeDossierProjection(answered.Record, v23DossierSpec())
	if err != nil {
		t.Fatal(err)
	}
	second, err := application.NewIntakeDossierProjection(answered.Record, v23DossierSpec())
	if err != nil {
		t.Fatal(err)
	}
	input := first.Input()
	if len(input.Sections) != 11 || len(input.Diagrams) != 6 ||
		len(first.Decisions()) != 1 || len(input.RiskRefs) != 1 ||
		len(first.Plan().Phases) != 1 || len(first.Plan().WorkItems) != 1 {
		t.Fatalf("incomplete canonical dossier: sections=%d diagrams=%d decisions=%d risks=%d",
			len(input.Sections), len(input.Diagrams), len(first.Decisions()), len(input.RiskRefs))
	}
	for _, section := range input.Sections {
		if strings.TrimSpace(section.Markdown) == "" {
			t.Fatalf("empty section %q", section.Kind)
		}
	}
	for _, diagram := range input.Diagrams {
		if diagram.Kind != application.IntakeDossierDiagramMermaid ||
			strings.TrimSpace(diagram.Source) == "" {
			t.Fatalf("invalid diagram: %+v", diagram)
		}
	}
	if !reflect.DeepEqual(first.Input(), second.Input()) ||
		!reflect.DeepEqual(first.Plan(), second.Plan()) {
		t.Fatal("equivalent replay changed the canonical projection")
	}
	firstDossier, err := application.BuildIntakeDossier(answered.Record, first.Plan(), first.Input())
	if err != nil {
		t.Fatal(err)
	}
	secondDossier, err := application.BuildIntakeDossier(answered.Record, second.Plan(), second.Input())
	if err != nil {
		t.Fatal(err)
	}
	after, _ := application.IntakeStateDigest(answered.Record.State)
	if firstDossier.Ref() != secondDossier.Ref() || firstDossier.Digest() != secondDossier.Digest() ||
		before != after || answered.Record.State.Revision() != 3 {
		t.Fatalf("projection was not stable/read-only: refs=%q/%q digest=%q/%q state=%q/%q",
			firstDossier.Ref(), secondDossier.Ref(), firstDossier.Digest(), secondDossier.Digest(), before, after)
	}
}

func v23DossierSpec() application.IntakeDossierProjectionSpec {
	items := func(key string) []application.IntakeDossierProjectionItem {
		return []application.IntakeDossierProjectionItem{{Key: key, Summary: "Contenido verificable"}}
	}
	return application.IntakeDossierProjectionSpec{
		Statement: "Coordinar calendarios", Objective: "Crear una agenda segura",
		ProductScope: application.IntakeDossierProductScopeProjection{InScope: items("calendar"), OutOfScope: items("billing")},
		UsersRoles: application.IntakeDossierUsersRolesProjection{
			Users: items("member"), Roles: []application.IntakeDossierProjectionRole{{
				Key: "member", Summary: "Miembro", PermissionSummaries: items("events_read"),
			}}, MainFlow: []application.IntakeDossierProjectionStep{
				{Order: 1, Key: "open", Summary: "Abrir agenda"},
				{Order: 2, Key: "create", Summary: "Crear evento"},
			},
		},
		Architecture: application.IntakeDossierArchitectureProjection{
			Rationale: "Aislar efectos", Domain: "Calendarios", Application: "Casos de uso",
			Ports: items("store"), Adapters: items("sqlite"), CompositionRoot: "Bootstrap único", CoreLimits: items("no_provider"),
		},
		Data: application.IntakeDossierDataProjection{Entities: items("event"), Lifecycles: items("event_lifecycle"), ImportExport: items("json"), Backups: items("daily")},
		Integrations: application.IntakeDossierIntegrationsProjection{Connectors: []application.IntakeDossierIntegrationProjection{{
			Key: "email", Summary: "Avisos", Authentication: "Credencial referenciada", DataFlow: "Datos mínimos",
		}}},
		SecurityPrivacy: application.IntakeDossierSecurityPrivacyProjection{
			DataClasses: items("personal"), Authentication: items("session"), Authorization: items("rbac"),
			Audit: items("changes"), Retention: items("events"), PrivacyRequirements: items("erasure"),
		},
		UIUX: application.IntakeDossierUIUXProjection{
			Views: items("agenda"), Navigation: items("primary"), EmptyStates: items("calendar"),
			ErrorStates: items("conflict"), Accessibility: items("keyboard"), Delivery: items("web"),
		},
		I18NL10N: application.IntakeDossierI18NL10NProjection{
			Locales:        []application.IntakeDossierLocaleProjection{{Tag: "es-ES", Summary: "Español"}},
			FallbackLocale: "es-ES", Catalog: "Catálogo único", Formats: items("datetime"),
			VisibleSurfaces: items("web"), VisibleTextRule: "Todo texto usa catálogo",
		},
		DeployOperations: application.IntakeDossierDeployOperationsProjection{
			Environments: items("production"), DeploymentUnits: items("app"), Network: items("https"),
			Observability: items("logs"), Backups: items("database"), Updates: items("release"), Rollback: items("application"),
		},
		Decisions: []application.IntakeDossierDecisionProjection{{
			QuestionRef: "intake-question:audience", ChoiceSummary: "Uso personal",
			RecommendationSummary: "Uso de equipo", DeviationJustification: "Primera entrega personal",
		}},
		Risks: []application.IntakeDossierRiskProjection{{
			Ref: "intake-risk:provider", Summary: "Cambio externo", Mitigation: "Puerto sustituible", StatusKey: "intake.risk.provider.open",
		}}, IncludeDecisionsDiagram: true,
		Plan: application.PlanSpec{
			Phases: []application.PhaseSpec{{Ref: "phase-instance:v23-dossier", Key: "phase.build", TemplateRef: "phase-template:build", CriterionRefs: []string{"criterion:tests"}}},
			WorkItems: []application.WorkItemSpec{{
				Key: "work:build", Objective: "Construir agenda", Phase: "phase.build", Role: goal.DefaultRoleKey().String(),
				CouncilPolicy: council.PolicyRequired, RequiredTests: []application.RequiredTestSpec{{
					Ref: "required-test:v23-dossier", ToolRef: "tool:go-test", Arguments: []string{"go", "test", "./..."}, WorkingDirectory: ".",
				}}, OutputContract: goal.OutputContractEvidenceBundle,
			}},
		},
	}
}

func v23DossierAuthorization(t *testing.T, repository *sqlite.Repository,
	principal identity.Principal, project goal.ProjectRef, operation application.IntakeOperation,
	mutationRef string) identity.AuthorizationReceipt {
	t.Helper()
	at := time.Date(2026, 8, 11, 8, 0, 0, 0, time.UTC)
	requestRef, err := application.IntakeAuthorizationRequestRef(operation, mutationRef)
	if err != nil {
		t.Fatal(err)
	}
	request, err := identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{
		RequestRef: requestRef, Principal: principal, ProjectRef: project,
		Permission: identity.PermissionGoalsCreate, ResourceRef: project.String(), RequestedAt: at,
	})
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := repository.Authorize(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	return receipt
}
