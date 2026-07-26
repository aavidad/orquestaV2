package application

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/wizard/stages"
)

func TestPrepareWizardDossierCompilesBuiltInPlanAndPersistsOneDossier(
	t *testing.T,
) {
	t.Parallel()

	system := newIntakeDossierTestSystem(t)
	request := wizardDossierTestRequest(
		t,
		system,
		"request:wizard-dossier-first",
		"template:deploy",
	)
	template, found := stages.BuiltIn().Template(request.TemplateRef)
	if !found {
		t.Fatal("built-in template missing")
	}
	wantCompilation, err := CompileWizardStagePlan(
		template,
		mustBindWizardCatalogIdentity(t, request),
	)
	if err != nil {
		t.Fatal(err)
	}

	result, err := system.service.PrepareWizardDossier(
		context.Background(),
		request,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Created {
		t.Fatal("first preparation was not created")
	}
	if result.Record.Dossier.PlanDigest() !=
		IntakeDossierPlanDigest(wantCompilation.Plan) {
		t.Fatal("dossier plan digest differs from built-in compilation")
	}
	if !reflect.DeepEqual(
		result.StagePlanProjection,
		wantCompilation.Projection,
	) {
		t.Fatal("stage preview differs from built-in compilation")
	}
	if system.dossiers.creates.Load() != 1 ||
		system.intakes.gets.Load() != 2 {
		t.Fatalf(
			"creates=%d intake_gets=%d",
			system.dossiers.creates.Load(),
			system.intakes.gets.Load(),
		)
	}
	if !wizardProjectionHasEffect(
		result.StagePlanProjection,
		stages.EffectMutateExternal,
	) {
		t.Fatal("deploy effect absent from preview")
	}
	if result.Record.Dossier.StateRevision() != request.ExpectedRevision ||
		result.Record.Dossier.SourceIntakeReceiptRef() !=
			request.SourceIntakeReceiptRef ||
		result.CatalogVersion != request.CatalogVersion ||
		result.CatalogDigest != request.CatalogDigest ||
		result.TemplateDigest != request.TemplateDigest {
		t.Fatal("dossier lost exact durable intake binding")
	}
	plan := result.Record.Dossier.Plan()
	for _, binding := range wizardCatalogBindingValues(request) {
		for index, phase := range plan.Phases {
			if !containsString(phase.InputRefs, binding) {
				t.Fatalf("phase[%d] lacks durable catalog binding %q", index, binding)
			}
		}
		if !strings.Contains(
			wizardDossierSectionMarkdown(
				result.Record.Dossier,
				IntakeDossierSectionOrchestrationPlan,
			),
			binding,
		) {
			t.Fatalf("orchestration projection lacks durable binding %q", binding)
		}
	}
}

func TestPrepareWizardDossierReplaysExactHistoricalProjectionAfterIntakeAdvance(
	t *testing.T,
) {
	t.Parallel()

	system := newIntakeDossierTestSystem(t)
	request := wizardDossierTestRequest(
		t,
		system,
		"request:wizard-dossier-replay",
		"template:build_app",
	)
	first, err := system.service.PrepareWizardDossier(
		context.Background(),
		request,
	)
	if err != nil {
		t.Fatal(err)
	}
	wantDossier := SnapshotIntakeDossier(first.Record.Dossier)
	wantReceipt := first.Record.Receipt
	wantStageProjection := cloneWizardStagePlanProjection(
		first.StagePlanProjection,
	)

	// Mutating returned data cannot affect persisted dossier or later preview.
	first.Record.Dossier.sections[0].Markdown = "caller mutation"
	first.StagePlanProjection.Units[0].Effects[0].Kind =
		stages.EffectMutateExternal
	first.StagePlanProjection.Stages[0].DependsOn = append(
		first.StagePlanProjection.Stages[0].DependsOn,
		first.StagePlanProjection.Stages[0].Ref,
	)

	system.advanceIntake(t)
	beforeGets := system.intakes.gets.Load()
	beforeCreates := system.dossiers.creates.Load()
	replayed, err := system.service.PrepareWizardDossier(
		context.Background(),
		request,
	)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Created ||
		replayed.Record.Receipt != wantReceipt ||
		!reflect.DeepEqual(
			SnapshotIntakeDossier(replayed.Record.Dossier),
			wantDossier,
		) ||
		!reflect.DeepEqual(
			replayed.StagePlanProjection,
			wantStageProjection,
		) {
		t.Fatalf("non-exact replay: %+v", replayed)
	}
	if got := system.intakes.gets.Load(); got != beforeGets+1 {
		t.Fatalf("historical replay intake reads=%d want=%d", got, beforeGets+1)
	}
	if got := system.dossiers.creates.Load(); got != beforeCreates {
		t.Fatalf("historical replay wrote dossier: creates=%d want=%d", got, beforeCreates)
	}
}

func TestPrepareWizardDossierRejectsDivergenceAndStaleFirstCreation(
	t *testing.T,
) {
	t.Parallel()

	system := newIntakeDossierTestSystem(t)
	request := wizardDossierTestRequest(
		t,
		system,
		"request:wizard-dossier-divergent",
		"template:research",
	)
	if _, err := system.service.PrepareWizardDossier(
		context.Background(),
		request,
	); err != nil {
		t.Fatal(err)
	}
	system.advanceIntake(t)
	beforeCreates := system.dossiers.creates.Load()

	divergent := request
	divergent.ProjectionSpec = cloneIntakeDossierProjectionSpec(
		request.ProjectionSpec,
	)
	divergent.ProjectionSpec.Architecture.Rationale += " changed"
	if _, err := system.service.PrepareWizardDossier(
		context.Background(),
		divergent,
	); !IsStateError(err, StateConflict) {
		t.Fatalf("divergent replay error=%v", err)
	}

	stale := request
	stale.RequestRef = "request:wizard-dossier-stale-first"
	stale.AuthorizationReceipt = intakeDossierAuthorization(
		t,
		stale.ActorRef,
		stale.ProjectRef,
		stale.RequestRef,
	)
	if _, err := system.service.PrepareWizardDossier(
		context.Background(),
		stale,
	); !IsStateError(err, StateConflict) {
		t.Fatalf("stale first creation error=%v", err)
	}
	if got := system.dossiers.creates.Load(); got != beforeCreates {
		t.Fatalf("rejected requests wrote dossiers: creates=%d want=%d", got, beforeCreates)
	}
}

func TestPrepareWizardDossierRejectsUnknownTemplateCallerPlanAndBadAuthority(
	t *testing.T,
) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*testing.T, *PrepareWizardDossierRequest)
		wantErr error
	}{
		{
			name: "unknown template",
			mutate: func(t *testing.T, request *PrepareWizardDossierRequest) {
				t.Helper()
				ref, err := stages.NewTemplateRef("template:unknown")
				if err != nil {
					t.Fatal(err)
				}
				request.TemplateRef = ref
			},
			wantErr: ErrWizardStagePlanInvalid,
		},
		{
			name: "caller plan injection",
			mutate: func(_ *testing.T, request *PrepareWizardDossierRequest) {
				request.ProjectionSpec.Plan = PlanSpec{
					Phases: []PhaseSpec{{Key: "caller"}},
				}
			},
			wantErr: ErrIntakeDossierInvalid,
		},
		{
			name: "objective divergence",
			mutate: func(_ *testing.T, request *PrepareWizardDossierRequest) {
				request.StagePlanInput.Objective = "another objective"
			},
			wantErr: ErrWizardStagePlanInvalid,
		},
		{
			name: "missing authority",
			mutate: func(_ *testing.T, request *PrepareWizardDossierRequest) {
				request.AuthorizationReceipt = identity.AuthorizationReceipt{}
			},
			wantErr: ErrForbidden,
		},
		{
			name: "missing catalog version",
			mutate: func(_ *testing.T, request *PrepareWizardDossierRequest) {
				request.CatalogVersion = stages.CatalogVersion{}
			},
			wantErr: ErrWizardStagePlanInvalid,
		},
		{
			name: "missing catalog digest",
			mutate: func(_ *testing.T, request *PrepareWizardDossierRequest) {
				request.CatalogDigest = stages.Digest{}
			},
			wantErr: ErrWizardStagePlanInvalid,
		},
		{
			name: "unknown catalog version",
			mutate: func(t *testing.T, request *PrepareWizardDossierRequest) {
				t.Helper()
				version, err := stages.NewCatalogVersion(
					"orquesta.wizard.stages.v999",
				)
				if err != nil {
					t.Fatal(err)
				}
				request.CatalogVersion = version
			},
			wantErr: ErrWizardStagePlanInvalid,
		},
		{
			name: "catalog digest mismatch",
			mutate: func(t *testing.T, request *PrepareWizardDossierRequest) {
				t.Helper()
				request.CatalogDigest = wizardDifferentDigest(
					t,
					request.CatalogDigest,
				)
			},
			wantErr: ErrWizardStagePlanInvalid,
		},
		{
			name: "template digest mismatch",
			mutate: func(t *testing.T, request *PrepareWizardDossierRequest) {
				t.Helper()
				request.TemplateDigest = wizardDifferentDigest(
					t,
					request.TemplateDigest,
				)
			},
			wantErr: ErrWizardStagePlanInvalid,
		},
		{
			name: "reserved catalog binding injection",
			mutate: func(t *testing.T, request *PrepareWizardDossierRequest) {
				t.Helper()
				ref, err := goal.NewInputRef(
					"input:wizard-catalog:forged:sha256:" +
						strings.Repeat("a", 64),
				)
				if err != nil {
					t.Fatal(err)
				}
				request.StagePlanInput.InputRefs = append(
					request.StagePlanInput.InputRefs,
					ref,
				)
			},
			wantErr: ErrWizardStagePlanInvalid,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			system := newIntakeDossierTestSystem(t)
			request := wizardDossierTestRequest(
				t,
				system,
				"request:wizard-dossier-negative-"+test.name,
				"template:self_change",
			)
			test.mutate(t, &request)
			result, err := system.service.PrepareWizardDossier(
				context.Background(),
				request,
			)
			if !errors.Is(err, test.wantErr) ||
				!reflect.DeepEqual(result, WizardDossierResult{}) {
				t.Fatalf("result=%+v error=%v want=%v", result, err, test.wantErr)
			}
			if system.intakes.gets.Load() != 0 ||
				system.dossiers.creates.Load() != 0 {
				t.Fatal("rejected request reached state")
			}
		})
	}
}

func TestPrepareWizardDossierKeepsV1PreviewWhenAdditiveV2ChangesOnlyRoadmap(
	t *testing.T,
) {
	t.Parallel()

	system := newIntakeDossierTestSystem(t)
	v1, err := stages.BuiltInVersion(stages.VersionV1())
	if err != nil {
		t.Fatal(err)
	}
	v2 := wizardRoadmapOnlyV2Catalog(t, v1)
	resolve := func(version stages.CatalogVersion) (stages.Catalog, error) {
		switch version {
		case v1.Version():
			return v1, nil
		case v2.Version():
			return v2, nil
		default:
			return stages.Catalog{}, stages.DomainError{
				Code:  stages.ErrorCatalogVersionUnknown,
				Field: "catalog.version",
			}
		}
	}
	request := wizardDossierTestRequest(
		t,
		system,
		"request:wizard-dossier-versioned",
		"template:research",
	)
	v1Template, _ := v1.Template(request.TemplateRef)
	v2Template, _ := v2.Template(request.TemplateRef)
	v1Raw, err := CompileWizardStagePlan(v1Template, request.StagePlanInput)
	if err != nil {
		t.Fatal(err)
	}
	v2Raw, err := CompileWizardStagePlan(v2Template, request.StagePlanInput)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(v1Raw.Plan, v2Raw.Plan) ||
		reflect.DeepEqual(v1Raw.Projection, v2Raw.Projection) {
		t.Fatal("fixture is not a preview-only catalog change")
	}

	first, err := system.service.prepareWizardDossier(
		context.Background(),
		request,
		resolve,
	)
	if err != nil {
		t.Fatal(err)
	}
	wantReceipt := first.Record.Receipt
	wantDossier := SnapshotIntakeDossier(first.Record.Dossier)
	wantPreview := cloneWizardStagePlanProjection(first.StagePlanProjection)
	system.advanceIntake(t)

	replayed, err := system.service.prepareWizardDossier(
		context.Background(),
		request,
		resolve,
	)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Created ||
		replayed.CatalogVersion != v1.Version() ||
		replayed.Record.Receipt != wantReceipt ||
		!reflect.DeepEqual(
			SnapshotIntakeDossier(replayed.Record.Dossier),
			wantDossier,
		) ||
		!reflect.DeepEqual(replayed.StagePlanProjection, wantPreview) {
		t.Fatal("v1 replay mixed the additive v2 preview")
	}

	v2Request := request
	v2Request.CatalogVersion = v2.Version()
	v2Request.CatalogDigest = v2.Digest()
	v2Request.TemplateDigest = v2Template.Digest()
	beforeCreates := system.dossiers.creates.Load()
	if _, err := system.service.prepareWizardDossier(
		context.Background(),
		v2Request,
		resolve,
	); !IsStateError(err, StateConflict) {
		t.Fatalf("same request accepted v2 preview-only drift: %v", err)
	}
	if system.dossiers.creates.Load() != beforeCreates {
		t.Fatal("preview-only divergent replay wrote another dossier")
	}
}

func wizardDossierTestRequest(
	t *testing.T,
	system intakeDossierTestSystem,
	requestRef string,
	templateValue string,
) PrepareWizardDossierRequest {
	t.Helper()

	templateRef, err := stages.NewTemplateRef(templateValue)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := stages.BuiltInVersion(stages.VersionV1())
	if err != nil {
		t.Fatal(err)
	}
	template, found := catalog.Template(templateRef)
	if !found {
		t.Fatalf("template %q missing", templateValue)
	}
	stageInput := wizardStagePlanTestInput(t)
	spec := validIntakeDossierProjectionSpec()
	spec.Objective = stageInput.Objective
	spec.Plan = PlanSpec{}
	return PrepareWizardDossierRequest{
		RequestRef:             requestRef,
		ActorRef:               system.record.ActorRef,
		ProjectRef:             system.record.ProjectRef,
		StateRef:               system.record.State.Ref(),
		ExpectedRevision:       system.record.State.Revision(),
		SourceIntakeReceiptRef: system.record.Receipt.Ref,
		CatalogVersion:         catalog.Version(),
		CatalogDigest:          catalog.Digest(),
		TemplateRef:            templateRef,
		TemplateDigest:         template.Digest(),
		StagePlanInput:         stageInput,
		ProjectionSpec:         spec,
		AuthorizationReceipt: intakeDossierAuthorization(
			t,
			system.record.ActorRef,
			system.record.ProjectRef,
			requestRef,
		),
	}
}

func wizardRoadmapOnlyV2Catalog(
	t *testing.T,
	v1 stages.Catalog,
) stages.Catalog {
	t.Helper()

	templates := v1.Templates()
	targetRef, err := stages.NewTemplateRef("template:research")
	if err != nil {
		t.Fatal(err)
	}
	for index, template := range templates {
		if template.Ref() != targetRef {
			continue
		}
		trace := template.RoadmapCapabilityRefs()
		added, err := stages.NewRoadmapCapabilityRef("WIZ-25")
		if err != nil {
			t.Fatal(err)
		}
		trace = append(trace, added)
		changed, err := stages.NewTemplate(stages.TemplateInput{
			Ref:                   template.Ref(),
			RoadmapCapabilityRefs: trace,
			Stages:                template.Stages(),
			Units:                 template.Units(),
		})
		if err != nil {
			t.Fatal(err)
		}
		templates[index] = changed
	}
	version, err := stages.NewCatalogVersion("orquesta.wizard.stages.v2")
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := stages.NewCatalog(stages.CatalogInput{
		Version: version, Templates: templates,
	})
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

func wizardDifferentDigest(t *testing.T, current stages.Digest) stages.Digest {
	t.Helper()

	value := strings.Repeat("a", 64)
	if current.String() == value {
		value = strings.Repeat("b", 64)
	}
	digest, err := stages.NewDigest(value)
	if err != nil {
		t.Fatal(err)
	}
	return digest
}

func mustBindWizardCatalogIdentity(
	t *testing.T,
	request PrepareWizardDossierRequest,
) WizardStagePlanInput {
	t.Helper()
	bound, err := bindWizardCatalogIdentity(
		request.StagePlanInput,
		request.CatalogVersion,
		request.CatalogDigest,
		request.TemplateRef,
		request.TemplateDigest,
	)
	if err != nil {
		t.Fatal(err)
	}
	return bound
}

func wizardCatalogBindingValues(
	request PrepareWizardDossierRequest,
) []string {
	return []string{
		"input:wizard-catalog:" + request.CatalogVersion.String() +
			":sha256:" + request.CatalogDigest.String(),
		"input:wizard-template:" + request.TemplateRef.String() +
			":sha256:" + request.TemplateDigest.String(),
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func wizardDossierSectionMarkdown(
	dossier IntakeDossier,
	kind IntakeDossierSectionKind,
) string {
	for _, section := range dossier.Sections() {
		if section.Kind == kind {
			return section.Markdown
		}
	}
	return ""
}

func wizardProjectionHasEffect(
	projection WizardStagePlanProjection,
	kind stages.EffectKind,
) bool {
	for _, unit := range projection.Units {
		for _, effect := range unit.Effects {
			if effect.Kind == kind {
				return true
			}
		}
	}
	return false
}
