package commands

import (
	"context"
	"encoding/json"
	"errors"

	"orquesta/internal/application"
	"orquesta/internal/governance"
	"orquesta/internal/intake"
	"orquesta/internal/wizard/stages"
)

type wizardDossierView struct {
	Dossier  intakeDossierView          `json:"dossier"`
	Identity wizardCatalogIdentityView  `json:"identity"`
	Preview  wizardStagePlanPreviewView `json:"preview"`
}

type wizardCatalogIdentityView struct {
	CatalogVersion string `json:"catalog_version"`
	CatalogDigest  string `json:"catalog_digest"`
	TemplateRef    string `json:"template_ref"`
	TemplateDigest string `json:"template_digest"`
}

type wizardStagePlanPreviewView struct {
	TemplateRef           string                `json:"template_ref"`
	RoadmapCapabilityRefs []string              `json:"roadmap_capability_refs"`
	Stages                []wizardStageView     `json:"stages"`
	Units                 []wizardStageUnitView `json:"units"`
	EffectsAuthorized     bool                  `json:"effects_authorized"`
	EffectsExecuted       bool                  `json:"effects_executed"`
}

type wizardStageView struct {
	Ref       string   `json:"ref"`
	Sequence  int      `json:"sequence"`
	Phase     string   `json:"phase"`
	DependsOn []string `json:"depends_on"`
	PlanKey   string   `json:"plan_key"`
}

type wizardStageUnitView struct {
	Ref                 string                        `json:"ref"`
	StageRef            string                        `json:"stage_ref"`
	DependsOn           []string                      `json:"depends_on"`
	PlanDependencies    []string                      `json:"plan_dependencies"`
	Role                string                        `json:"role"`
	WriteSet            []string                      `json:"write_set"`
	RequiredTests       []wizardStageRequiredTestView `json:"required_tests"`
	AcceptanceCriteria  []string                      `json:"acceptance_criteria"`
	Effects             []wizardStageEffectView       `json:"effects"`
	Policy              wizardStagePolicyView         `json:"policy"`
	BudgetDemand        wizardBudgetDemandView        `json:"budget_demand"`
	CouncilPolicy       string                        `json:"council_policy"`
	SecurityCriticality string                        `json:"security_criticality"`
	ReasoningEffort     string                        `json:"reasoning_effort"`
}

type wizardStageRequiredTestView struct {
	Ref           string   `json:"ref"`
	Kind          string   `json:"kind"`
	CriterionRefs []string `json:"criterion_refs"`
}

type wizardStageEffectView struct {
	Ref              string `json:"ref"`
	Kind             string `json:"kind"`
	ApprovalRequired bool   `json:"approval_required"`
}

type wizardStagePolicyView struct {
	Effort   string                  `json:"effort"`
	Security wizardStageSecurityView `json:"security"`
}

type wizardStageSecurityView struct {
	Risk            string `json:"risk"`
	Approval        string `json:"approval"`
	LeastPrivilege  bool   `json:"least_privilege"`
	SensitiveInputs bool   `json:"sensitive_inputs"`
}

type wizardBudgetDemandView struct {
	Ref       string             `json:"ref"`
	Resources wizardResourceView `json:"resources"`
}

type wizardResourceView struct {
	Tokens       int64  `json:"tokens"`
	MoneyMicros  int64  `json:"money_micros"`
	Currency     string `json:"currency"`
	ActiveTimeNS int64  `json:"active_time_ns"`
	ProcessSlots int64  `json:"process_slots"`
	DiskBytes    int64  `json:"disk_bytes"`
}

func handlePrepareWizardDossier(
	ctx context.Context,
	api applicationAPI,
	bound handlerContext,
	payload json.RawMessage,
) (json.RawMessage, error) {
	var input wizardDossierInput
	if err := decodePayload(payload, &input); err != nil {
		return nil, err
	}
	version, err := stages.NewCatalogVersion(input.CatalogVersion)
	if err != nil {
		return nil, commandError{code: CodeInvalidRequest, cause: err}
	}
	catalog, err := stages.BuiltInVersion(version)
	if err != nil {
		return nil, commandError{code: CodeInvalidRequest, cause: err}
	}
	templateRef, err := stages.NewTemplateRef(input.TemplateRef)
	if err != nil {
		return nil, commandError{code: CodeInvalidRequest, cause: err}
	}
	template, found := catalog.Template(templateRef)
	if !found {
		return nil, commandError{
			code:  CodeInvalidRequest,
			cause: errors.New("commands.wizard_template_unknown"),
		}
	}
	stagePlanInput, err := applicationWizardStagePlanInput(
		input.StagePlanInput,
	)
	if err != nil {
		return nil, err
	}
	result, err := api.PrepareWizardDossier(
		ctx,
		bound.access,
		application.PrepareWizardDossierRequest{
			RequestRef:             bound.requestRef,
			ActorRef:               bound.principal.ActorRef,
			ProjectRef:             bound.projectRef,
			StateRef:               intake.Ref(input.IntakeRef),
			ExpectedRevision:       input.ExpectedRevision,
			SourceIntakeReceiptRef: input.SourceIntakeReceiptRef,
			CatalogVersion:         version,
			CatalogDigest:          catalog.Digest(),
			TemplateRef:            templateRef,
			TemplateDigest:         template.Digest(),
			StagePlanInput:         stagePlanInput,
			ProjectionSpec: applicationWizardProjectionSpec(
				input.ProjectionSpec,
			),
		},
	)
	return marshalApplication(
		wizardDossierView{
			Dossier: projectIntakeDossier(result.Record),
			Identity: wizardCatalogIdentityView{
				CatalogVersion: result.CatalogVersion.String(),
				CatalogDigest:  result.CatalogDigest.String(),
				TemplateRef:    result.StagePlanProjection.TemplateRef.String(),
				TemplateDigest: result.TemplateDigest.String(),
			},
			Preview: projectWizardStagePlanPreview(
				result.StagePlanProjection,
			),
		},
		normalizeWizardDossierError(err),
	)
}

func projectWizardStagePlanPreview(
	source application.WizardStagePlanProjection,
) wizardStagePlanPreviewView {
	roadmapRefs := make([]string, len(source.RoadmapCapabilityRefs))
	for index, ref := range source.RoadmapCapabilityRefs {
		roadmapRefs[index] = ref.String()
	}
	stageViews := make([]wizardStageView, len(source.Stages))
	for index, stage := range source.Stages {
		dependencies := make([]string, len(stage.DependsOn))
		for dependencyIndex, dependency := range stage.DependsOn {
			dependencies[dependencyIndex] = dependency.String()
		}
		stageViews[index] = wizardStageView{
			Ref: stage.Ref.String(), Sequence: stage.Sequence,
			Phase: string(stage.Phase), DependsOn: dependencies,
			PlanKey: stage.PlanKey,
		}
	}
	unitViews := make([]wizardStageUnitView, len(source.Units))
	for index, unit := range source.Units {
		unitViews[index] = projectWizardStageUnit(unit)
	}
	return wizardStagePlanPreviewView{
		TemplateRef:           source.TemplateRef.String(),
		RoadmapCapabilityRefs: roadmapRefs,
		Stages:                stageViews, Units: unitViews,
		EffectsAuthorized: false, EffectsExecuted: false,
	}
}

func projectWizardStageUnit(
	source application.WizardStageUnitProjection,
) wizardStageUnitView {
	dependencies := make([]string, len(source.DependsOn))
	for index, dependency := range source.DependsOn {
		dependencies[index] = dependency.String()
	}
	writeSet := make([]string, len(source.WriteSet))
	for index, scope := range source.WriteSet {
		writeSet[index] = scope.Path
	}
	tests := make([]wizardStageRequiredTestView, len(source.RequiredTests))
	for index, test := range source.RequiredTests {
		criterionRefs := make([]string, len(test.CriterionRefs))
		for criterionIndex, ref := range test.CriterionRefs {
			criterionRefs[criterionIndex] = ref.String()
		}
		tests[index] = wizardStageRequiredTestView{
			Ref: test.Ref.String(), Kind: string(test.Kind),
			CriterionRefs: criterionRefs,
		}
	}
	criteria := make([]string, len(source.AcceptanceCriteria))
	for index, criterion := range source.AcceptanceCriteria {
		criteria[index] = criterion.Ref.String()
	}
	effects := make([]wizardStageEffectView, len(source.Effects))
	for index, effect := range source.Effects {
		effects[index] = wizardStageEffectView{
			Ref: effect.Ref.String(), Kind: string(effect.Kind),
			ApprovalRequired: effect.ApprovalRequired,
		}
	}
	return wizardStageUnitView{
		Ref: source.Ref.String(), StageRef: source.StageRef.String(),
		DependsOn: dependencies,
		PlanDependencies: nonNil(append(
			[]string(nil),
			source.PlanDependencies...,
		)),
		Role: source.Role.String(), WriteSet: writeSet,
		RequiredTests: tests, AcceptanceCriteria: criteria,
		Effects: effects,
		Policy: wizardStagePolicyView{
			Effort: string(source.Policy.Effort),
			Security: wizardStageSecurityView{
				Risk:            string(source.Policy.Security.Risk),
				Approval:        string(source.Policy.Security.Approval),
				LeastPrivilege:  source.Policy.Security.LeastPrivilege,
				SensitiveInputs: source.Policy.Security.SensitiveInputs,
			},
		},
		BudgetDemand:        projectWizardBudgetDemand(source.BudgetDemand),
		CouncilPolicy:       string(source.CouncilPolicy),
		SecurityCriticality: string(source.SecurityCriticality),
		ReasoningEffort:     string(source.ReasoningEffort),
	}
}

func projectWizardBudgetDemand(
	source governance.BudgetDemand,
) wizardBudgetDemandView {
	return wizardBudgetDemandView{
		Ref: source.Ref,
		Resources: wizardResourceView{
			Tokens:       source.Resources.Tokens,
			MoneyMicros:  source.Resources.MoneyMicros,
			Currency:     string(source.Resources.Currency),
			ActiveTimeNS: source.Resources.ActiveTimeNS,
			ProcessSlots: source.Resources.ProcessSlots,
			DiskBytes:    source.Resources.DiskBytes,
		},
	}
}

func normalizeWizardDossierError(err error) error {
	if errors.Is(err, application.ErrWizardStagePlanInvalid) {
		return commandError{code: CodeInvalidRequest, cause: err}
	}
	return normalizeIntakeDossierError(err)
}
