package application

import (
	"context"
	"errors"
	"strings"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/intake"
	"orquesta/internal/wizard/stages"
)

// PrepareWizardDossierRequest binds one canonical Wizard template and its
// editorial projection to an exact durable intake revision. ProjectionSpec
// must not contain a Plan: application compiles and injects the selected
// built-in template so callers cannot substitute executable work.
type PrepareWizardDossierRequest struct {
	RequestRef             string
	ActorRef               goal.ActorRef
	ProjectRef             goal.ProjectRef
	StateRef               intake.Ref
	ExpectedRevision       intake.Revision
	SourceIntakeReceiptRef string
	CatalogVersion         stages.CatalogVersion
	CatalogDigest          stages.Digest
	TemplateRef            stages.TemplateRef
	TemplateDigest         stages.Digest
	StagePlanInput         WizardStagePlanInput
	ProjectionSpec         IntakeDossierProjectionSpec
	AuthorizationReceipt   identity.AuthorizationReceipt
}

// WizardDossierResult keeps effects and other template-only semantics visible
// for preview. StagePlanProjection is descriptive data; this use case neither
// authorizes nor performs any projected effect.
type WizardDossierResult struct {
	Record              IntakeDossierRecord
	CatalogVersion      stages.CatalogVersion
	CatalogDigest       stages.Digest
	TemplateDigest      stages.Digest
	StagePlanProjection WizardStagePlanProjection
	Created             bool
}

type wizardStageCatalogResolver func(
	stages.CatalogVersion,
) (stages.Catalog, error)

// PrepareWizardDossier composes the pure stage compiler and dossier projector,
// then delegates persistence and idempotency to PrepareIntakeDossier. It adds
// no receipt, store, transaction, lifecycle or Goal transition.
func (service *IntakeDossierService) PrepareWizardDossier(
	ctx context.Context,
	request PrepareWizardDossierRequest,
) (WizardDossierResult, error) {
	return service.prepareWizardDossier(
		ctx,
		request,
		stages.BuiltInVersion,
	)
}

func (service *IntakeDossierService) prepareWizardDossier(
	ctx context.Context,
	request PrepareWizardDossierRequest,
	resolveCatalog wizardStageCatalogResolver,
) (WizardDossierResult, error) {
	if service == nil || service.intakes == nil || service.dossiers == nil {
		return WizardDossierResult{}, errors.New("application.unavailable")
	}
	if resolveCatalog == nil {
		return WizardDossierResult{}, errors.New("application.unavailable")
	}
	if err := validatePrepareWizardDossierRequest(request); err != nil {
		return WizardDossierResult{}, err
	}
	authorizationRequestRef, err := IntakeDossierAuthorizationRequestRef(
		request.RequestRef,
	)
	if err != nil {
		return WizardDossierResult{}, err
	}
	if err := validateIntakeAuthorization(
		request.AuthorizationReceipt,
		request.ActorRef,
		request.ProjectRef,
		authorizationRequestRef,
	); err != nil {
		return WizardDossierResult{}, err
	}

	catalog, err := resolveCatalog(request.CatalogVersion)
	if err != nil {
		return WizardDossierResult{},
			invalidWizardStagePlan("catalog_version", err)
	}
	if catalog.Version() != request.CatalogVersion ||
		catalog.Digest() != request.CatalogDigest {
		return WizardDossierResult{},
			invalidWizardStagePlan("catalog_digest", nil)
	}
	template, found := catalog.Template(request.TemplateRef)
	if !found {
		return WizardDossierResult{},
			invalidWizardStagePlan("template_ref", nil)
	}
	if template.Digest() != request.TemplateDigest {
		return WizardDossierResult{},
			invalidWizardStagePlan("template_digest", nil)
	}
	stagePlanInput, err := bindWizardCatalogIdentity(
		request.StagePlanInput,
		request.CatalogVersion,
		request.CatalogDigest,
		request.TemplateRef,
		request.TemplateDigest,
	)
	if err != nil {
		return WizardDossierResult{}, err
	}
	compilation, err := CompileWizardStagePlan(
		template,
		stagePlanInput,
	)
	if err != nil {
		return WizardDossierResult{}, err
	}

	current, err := service.intakes.GetIntake(
		ctx,
		request.ActorRef,
		request.ProjectRef,
		request.StateRef,
	)
	if err != nil {
		return WizardDossierResult{}, err
	}
	if err := validateStoredIntakeRecord(
		request.ActorRef,
		request.ProjectRef,
		request.StateRef,
		current,
	); err != nil {
		return WizardDossierResult{}, err
	}
	source, err := intakeStateAtRevision(
		current.State,
		request.ExpectedRevision,
	)
	if err != nil {
		return WizardDossierResult{}, err
	}

	spec := cloneIntakeDossierProjectionSpec(request.ProjectionSpec)
	spec.Plan = cloneIntakeDossierPlan(compilation.Plan)
	projection, err := projectWizardDossierAtRevision(
		current,
		source,
		spec,
	)
	if err != nil {
		return WizardDossierResult{}, err
	}
	prepared, err := service.PrepareIntakeDossier(
		ctx,
		PrepareIntakeDossierRequest{
			RequestRef: request.RequestRef,
			ActorRef:   request.ActorRef, ProjectRef: request.ProjectRef,
			StateRef:               request.StateRef,
			ExpectedRevision:       request.ExpectedRevision,
			SourceIntakeReceiptRef: request.SourceIntakeReceiptRef,
			Plan:                   projection.Plan(),
			Input:                  projection.Input(),
			AuthorizationReceipt:   request.AuthorizationReceipt,
		},
	)
	if err != nil {
		return WizardDossierResult{}, err
	}
	return WizardDossierResult{
		Record:         prepared.Record,
		CatalogVersion: request.CatalogVersion,
		CatalogDigest:  request.CatalogDigest,
		TemplateDigest: request.TemplateDigest,
		StagePlanProjection: cloneWizardStagePlanProjection(
			compilation.Projection,
		),
		Created: prepared.Created,
	}, nil
}

func validatePrepareWizardDossierRequest(
	request PrepareWizardDossierRequest,
) error {
	if err := validateIntakeRequestScope(
		request.RequestRef,
		request.ActorRef,
		request.ProjectRef,
	); err != nil {
		return err
	}
	if !validIntakeStateRef(request.StateRef) {
		return &intake.DomainError{
			Code: intake.ErrorInvalidRef, Field: "state_ref",
		}
	}
	if request.ExpectedRevision == 0 {
		return invalidIntakeDossier("expected_revision", nil)
	}
	if !validIntakeDossierRef(
		request.SourceIntakeReceiptRef,
		"intake-receipt:",
	) {
		return invalidIntakeDossier("source_intake_receipt_ref", nil)
	}
	if request.CatalogVersion.String() == "" ||
		request.CatalogDigest.String() == "" ||
		request.TemplateRef.String() == "" ||
		request.TemplateDigest.String() == "" {
		return invalidWizardStagePlan("catalog_identity", nil)
	}
	if len(request.ProjectionSpec.Plan.Phases) != 0 ||
		len(request.ProjectionSpec.Plan.WorkItems) != 0 {
		return invalidIntakeDossier("projection.plan_forbidden", nil)
	}
	if request.StagePlanInput.Objective != request.ProjectionSpec.Objective {
		return invalidWizardStagePlan("objective_binding", nil)
	}
	return nil
}

func bindWizardCatalogIdentity(
	input WizardStagePlanInput,
	version stages.CatalogVersion,
	catalogDigest stages.Digest,
	templateRef stages.TemplateRef,
	templateDigest stages.Digest,
) (WizardStagePlanInput, error) {
	for _, ref := range input.InputRefs {
		if strings.HasPrefix(ref.String(), "input:wizard-catalog:") ||
			strings.HasPrefix(ref.String(), "input:wizard-template:") {
			return WizardStagePlanInput{},
				invalidWizardStagePlan("catalog_binding_reserved", nil)
		}
	}
	catalogBinding, err := goal.NewInputRef(
		"input:wizard-catalog:" + version.String() +
			":sha256:" + catalogDigest.String(),
	)
	if err != nil {
		return WizardStagePlanInput{},
			invalidWizardStagePlan("catalog_binding", err)
	}
	templateBinding, err := goal.NewInputRef(
		"input:wizard-template:" + templateRef.String() +
			":sha256:" + templateDigest.String(),
	)
	if err != nil {
		return WizardStagePlanInput{},
			invalidWizardStagePlan("template_binding", err)
	}
	bound := input
	bound.InputRefs = append(
		append([]goal.InputRef(nil), input.InputRefs...),
		catalogBinding,
		templateBinding,
	)
	return bound, nil
}

func projectWizardDossierAtRevision(
	current IntakeRecord,
	source intake.State,
	spec IntakeDossierProjectionSpec,
) (IntakeDossierProjection, error) {
	if source.Revision() == current.State.Revision() {
		return NewIntakeDossierProjection(current, spec)
	}

	// A current IntakeRecord carries only its latest receipt. For an exact
	// dossier replay after later mutations, rebuild the requested immutable
	// state from that validated append-only chain and run the same projector
	// pipeline without inventing a historical receipt or a second read port.
	decisions, err := currentIntakeDossierDecisions(source)
	if err != nil {
		return IntakeDossierProjection{}, err
	}
	canonical := canonicalIntakeDossierProjectionSpec(spec)
	bound, err := bindIntakeDossierProjectionDecisions(
		canonical.Decisions,
		decisions,
	)
	if err != nil {
		return IntakeDossierProjection{}, err
	}
	canonical.boundDecisions = bound
	if err := validateIntakeDossierProjectionSpec(canonical); err != nil {
		return IntakeDossierProjection{}, err
	}
	input := renderIntakeDossierProjection(canonical)
	if err := validateIntakeDossierInput(input); err != nil {
		return IntakeDossierProjection{}, err
	}
	return IntakeDossierProjection{
		input: input,
		plan:  cloneIntakeDossierPlan(canonical.Plan),
		decisions: append(
			[]intake.Decision(nil),
			decisions...,
		),
		decisionViews: intakeDossierProjectionDecisionViews(bound),
	}, nil
}

func cloneWizardStagePlanProjection(
	value WizardStagePlanProjection,
) WizardStagePlanProjection {
	cloned := value
	cloned.RoadmapCapabilityRefs = append(
		[]stages.RoadmapCapabilityRef(nil),
		value.RoadmapCapabilityRefs...,
	)
	cloned.Stages = append(
		[]WizardStageProjection(nil),
		value.Stages...,
	)
	for index := range cloned.Stages {
		cloned.Stages[index].DependsOn = append(
			[]stages.StageRef(nil),
			value.Stages[index].DependsOn...,
		)
	}
	cloned.Units = append(
		[]WizardStageUnitProjection(nil),
		value.Units...,
	)
	for index := range cloned.Units {
		source := value.Units[index]
		cloned.Units[index].DependsOn = append(
			[]stages.UnitRef(nil),
			source.DependsOn...,
		)
		cloned.Units[index].PlanDependencies = append(
			[]string(nil),
			source.PlanDependencies...,
		)
		cloned.Units[index].WriteSet = append(
			[]stages.WriteScope(nil),
			source.WriteSet...,
		)
		cloned.Units[index].RequiredTests = append(
			[]stages.RequiredTest(nil),
			source.RequiredTests...,
		)
		for testIndex := range cloned.Units[index].RequiredTests {
			cloned.Units[index].RequiredTests[testIndex].CriterionRefs = append(
				[]stages.CriterionRef(nil),
				source.RequiredTests[testIndex].CriterionRefs...,
			)
		}
		cloned.Units[index].AcceptanceCriteria = append(
			[]stages.Criterion(nil),
			source.AcceptanceCriteria...,
		)
		cloned.Units[index].Effects = append(
			[]stages.Effect(nil),
			source.Effects...,
		)
	}
	return cloned
}
