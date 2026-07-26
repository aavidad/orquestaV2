package commands

import (
	"errors"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/intake"
)

type wizardDossierInput struct {
	IntakeRef              string                       `json:"intake_ref"`
	ExpectedRevision       intake.Revision              `json:"expected_revision"`
	SourceIntakeReceiptRef string                       `json:"source_intake_receipt_ref"`
	CatalogVersion         string                       `json:"catalog_version"`
	TemplateRef            string                       `json:"template_ref"`
	StagePlanInput         wizardStagePlanInput         `json:"stage_plan_input"`
	ProjectionSpec         wizardDossierProjectionInput `json:"projection_spec"`
}

type wizardStagePlanInput struct {
	Objective      string                  `json:"objective"`
	InputRefs      []string                `json:"input_refs"`
	SkillRefs      []string                `json:"skill_refs"`
	ToolRefs       []string                `json:"tool_refs"`
	CapabilityRefs []string                `json:"capability_refs"`
	Tests          wizardStageTestInputs   `json:"tests"`
	Budgets        wizardStageBudgetInputs `json:"budgets"`
}

type wizardStageTestInputs struct {
	Unit          wizardStageTestBindingInput `json:"unit"`
	Contract      wizardStageTestBindingInput `json:"contract"`
	Integration   wizardStageTestBindingInput `json:"integration"`
	Security      wizardStageTestBindingInput `json:"security"`
	Review        wizardStageTestBindingInput `json:"review"`
	Postcondition wizardStageTestBindingInput `json:"postcondition"`
}

type wizardStageTestBindingInput struct {
	ToolRef          string   `json:"tool_ref"`
	Arguments        []string `json:"arguments"`
	WorkingDirectory string   `json:"working_directory"`
}

type wizardStageBudgetInputs struct {
	Routine wizardResourceInput `json:"routine"`
	Focused wizardResourceInput `json:"focused"`
	Deep    wizardResourceInput `json:"deep"`
}

type wizardResourceInput struct {
	Tokens       int64  `json:"tokens"`
	MoneyMicros  int64  `json:"money_micros"`
	Currency     string `json:"currency"`
	ActiveTimeNS int64  `json:"active_time_ns"`
	ProcessSlots int64  `json:"process_slots"`
	DiskBytes    int64  `json:"disk_bytes"`
}

type wizardDossierProjectionInput struct {
	Statement string `json:"statement"`
	Objective string `json:"objective"`

	ProductScope     wizardProductScopeInput     `json:"product_scope"`
	UsersRoles       wizardUsersRolesInput       `json:"users_roles"`
	Architecture     wizardArchitectureInput     `json:"architecture"`
	Data             wizardDataInput             `json:"data"`
	Integrations     wizardIntegrationsInput     `json:"integrations"`
	SecurityPrivacy  wizardSecurityPrivacyInput  `json:"security_privacy"`
	UIUX             wizardUIUXInput             `json:"ui_ux"`
	I18NL10N         wizardI18NL10NInput         `json:"i18n_l10n"`
	DeployOperations wizardDeployOperationsInput `json:"deploy_operations"`

	Decisions               []wizardDecisionInput `json:"decisions"`
	Risks                   []wizardRiskInput     `json:"risks"`
	OpenIssues              []wizardItemInput     `json:"open_issues"`
	DeferredDecisions       []wizardItemInput     `json:"deferred_decisions"`
	IncludeDecisionsDiagram bool                  `json:"include_decisions_diagram"`
}

type wizardItemInput struct {
	Key     string `json:"key"`
	Summary string `json:"summary"`
}

type wizardStepInput struct {
	Order   uint32 `json:"order"`
	Key     string `json:"key"`
	Summary string `json:"summary"`
}

type wizardRoleInput struct {
	Key                 string            `json:"key"`
	Summary             string            `json:"summary"`
	PermissionSummaries []wizardItemInput `json:"permission_summaries"`
}

type wizardProductScopeInput struct {
	InScope    []wizardItemInput `json:"in_scope"`
	OutOfScope []wizardItemInput `json:"out_of_scope"`
}

type wizardUsersRolesInput struct {
	Users    []wizardItemInput `json:"users"`
	Roles    []wizardRoleInput `json:"roles"`
	MainFlow []wizardStepInput `json:"main_flow"`
}

type wizardArchitectureInput struct {
	Rationale       string            `json:"rationale"`
	Domain          string            `json:"domain"`
	Application     string            `json:"application"`
	Ports           []wizardItemInput `json:"ports"`
	Adapters        []wizardItemInput `json:"adapters"`
	CompositionRoot string            `json:"composition_root"`
	CoreLimits      []wizardItemInput `json:"core_limits"`
}

type wizardDataInput struct {
	Entities     []wizardItemInput `json:"entities"`
	Lifecycles   []wizardItemInput `json:"lifecycles"`
	ImportExport []wizardItemInput `json:"import_export"`
	Backups      []wizardItemInput `json:"backups"`
	Versioning   []wizardItemInput `json:"versioning"`
}

type wizardIntegrationInput struct {
	Key            string `json:"key"`
	Summary        string `json:"summary"`
	Authentication string `json:"authentication"`
	DataFlow       string `json:"data_flow"`
}

type wizardIntegrationsInput struct {
	Connectors []wizardIntegrationInput `json:"connectors"`
}

type wizardSecurityPrivacyInput struct {
	DataClasses         []wizardItemInput `json:"data_classes"`
	Authentication      []wizardItemInput `json:"authentication"`
	Authorization       []wizardItemInput `json:"authorization"`
	Audit               []wizardItemInput `json:"audit"`
	Retention           []wizardItemInput `json:"retention"`
	PrivacyRequirements []wizardItemInput `json:"privacy_requirements"`
}

type wizardUIUXInput struct {
	Views         []wizardItemInput `json:"views"`
	Navigation    []wizardItemInput `json:"navigation"`
	EmptyStates   []wizardItemInput `json:"empty_states"`
	ErrorStates   []wizardItemInput `json:"error_states"`
	Accessibility []wizardItemInput `json:"accessibility"`
	Delivery      []wizardItemInput `json:"delivery"`
}

type wizardLocaleInput struct {
	Tag     string `json:"tag"`
	Summary string `json:"summary"`
}

type wizardI18NL10NInput struct {
	Locales         []wizardLocaleInput `json:"locales"`
	FallbackLocale  string              `json:"fallback_locale"`
	Catalog         string              `json:"catalog"`
	Formats         []wizardItemInput   `json:"formats"`
	VisibleSurfaces []wizardItemInput   `json:"visible_surfaces"`
	VisibleTextRule string              `json:"visible_text_rule"`
}

type wizardDeployOperationsInput struct {
	Environments    []wizardItemInput `json:"environments"`
	DeploymentUnits []wizardItemInput `json:"deployment_units"`
	Network         []wizardItemInput `json:"network"`
	Observability   []wizardItemInput `json:"observability"`
	Backups         []wizardItemInput `json:"backups"`
	Updates         []wizardItemInput `json:"updates"`
	Rollback        []wizardItemInput `json:"rollback"`
}

type wizardDecisionInput struct {
	QuestionRef            intake.QuestionRef `json:"question_ref"`
	ChoiceSummary          string             `json:"choice_summary"`
	RecommendationSummary  string             `json:"recommendation_summary"`
	DeviationJustification string             `json:"deviation_justification"`
}

type wizardRiskInput struct {
	Ref        application.IntakeRiskRef `json:"ref"`
	Summary    string                    `json:"summary"`
	Mitigation string                    `json:"mitigation"`
	StatusKey  intake.MessageKey         `json:"status_key"`
}

func applicationWizardStagePlanInput(
	input wizardStagePlanInput,
) (application.WizardStagePlanInput, error) {
	inputRefs, err := newWizardRefs(input.InputRefs, goal.NewInputRef)
	if err != nil {
		return application.WizardStagePlanInput{}, err
	}
	skillRefs, err := newWizardRefs(input.SkillRefs, goal.NewSkillRef)
	if err != nil {
		return application.WizardStagePlanInput{}, err
	}
	toolRefs, err := newWizardRefs(input.ToolRefs, goal.NewToolRef)
	if err != nil {
		return application.WizardStagePlanInput{}, err
	}
	capabilityRefs, err := newWizardRefs(
		input.CapabilityRefs,
		goal.NewCapabilityRef,
	)
	if err != nil {
		return application.WizardStagePlanInput{}, err
	}
	tests, err := applicationWizardTestInputs(input.Tests)
	if err != nil {
		return application.WizardStagePlanInput{}, err
	}
	return application.WizardStagePlanInput{
		Objective:      input.Objective,
		InputRefs:      inputRefs,
		SkillRefs:      skillRefs,
		ToolRefs:       toolRefs,
		CapabilityRefs: capabilityRefs,
		Tests:          tests,
		Budgets: application.WizardStageEffortBudgets{
			Routine: applicationWizardResources(input.Budgets.Routine),
			Focused: applicationWizardResources(input.Budgets.Focused),
			Deep:    applicationWizardResources(input.Budgets.Deep),
		},
	}, nil
}

func newWizardRefs[T any](
	values []string,
	constructor func(string) (T, error),
) ([]T, error) {
	result := make([]T, len(values))
	for index, value := range values {
		ref, err := constructor(value)
		if err != nil {
			return nil, commandError{
				code:  CodeInvalidRequest,
				cause: errors.New("commands.wizard_ref_invalid"),
			}
		}
		result[index] = ref
	}
	return result, nil
}

func applicationWizardTestInputs(
	input wizardStageTestInputs,
) (application.WizardStageTestBindings, error) {
	values := []wizardStageTestBindingInput{
		input.Unit, input.Contract, input.Integration,
		input.Security, input.Review, input.Postcondition,
	}
	converted := make([]application.WizardStageTestBinding, len(values))
	for index, value := range values {
		toolRef, err := goal.NewToolRef(value.ToolRef)
		if err != nil {
			return application.WizardStageTestBindings{}, commandError{
				code:  CodeInvalidRequest,
				cause: errors.New("commands.wizard_test_tool_ref_invalid"),
			}
		}
		converted[index] = application.WizardStageTestBinding{
			ToolRef: toolRef,
			Arguments: append(
				[]string(nil),
				value.Arguments...,
			),
			WorkingDirectory: value.WorkingDirectory,
		}
	}
	return application.WizardStageTestBindings{
		Unit: converted[0], Contract: converted[1],
		Integration: converted[2], Security: converted[3],
		Review: converted[4], Postcondition: converted[5],
	}, nil
}

func applicationWizardResources(
	input wizardResourceInput,
) governance.ResourceVector {
	return governance.ResourceVector{
		Tokens: input.Tokens, MoneyMicros: input.MoneyMicros,
		Currency:     governance.Currency(input.Currency),
		ActiveTimeNS: input.ActiveTimeNS, ProcessSlots: input.ProcessSlots,
		DiskBytes: input.DiskBytes,
	}
}

func applicationWizardProjectionSpec(
	input wizardDossierProjectionInput,
) application.IntakeDossierProjectionSpec {
	return application.IntakeDossierProjectionSpec{
		Statement: input.Statement,
		Objective: input.Objective,
		ProductScope: application.IntakeDossierProductScopeProjection{
			InScope:    applicationWizardItems(input.ProductScope.InScope),
			OutOfScope: applicationWizardItems(input.ProductScope.OutOfScope),
		},
		UsersRoles: application.IntakeDossierUsersRolesProjection{
			Users:    applicationWizardItems(input.UsersRoles.Users),
			Roles:    applicationWizardRoles(input.UsersRoles.Roles),
			MainFlow: applicationWizardSteps(input.UsersRoles.MainFlow),
		},
		Architecture: application.IntakeDossierArchitectureProjection{
			Rationale:       input.Architecture.Rationale,
			Domain:          input.Architecture.Domain,
			Application:     input.Architecture.Application,
			Ports:           applicationWizardItems(input.Architecture.Ports),
			Adapters:        applicationWizardItems(input.Architecture.Adapters),
			CompositionRoot: input.Architecture.CompositionRoot,
			CoreLimits:      applicationWizardItems(input.Architecture.CoreLimits),
		},
		Data: application.IntakeDossierDataProjection{
			Entities:     applicationWizardItems(input.Data.Entities),
			Lifecycles:   applicationWizardItems(input.Data.Lifecycles),
			ImportExport: applicationWizardItems(input.Data.ImportExport),
			Backups:      applicationWizardItems(input.Data.Backups),
			Versioning:   applicationWizardItems(input.Data.Versioning),
		},
		Integrations: application.IntakeDossierIntegrationsProjection{
			Connectors: applicationWizardIntegrations(
				input.Integrations.Connectors,
			),
		},
		SecurityPrivacy: application.IntakeDossierSecurityPrivacyProjection{
			DataClasses: applicationWizardItems(
				input.SecurityPrivacy.DataClasses,
			),
			Authentication: applicationWizardItems(
				input.SecurityPrivacy.Authentication,
			),
			Authorization: applicationWizardItems(
				input.SecurityPrivacy.Authorization,
			),
			Audit: applicationWizardItems(input.SecurityPrivacy.Audit),
			Retention: applicationWizardItems(
				input.SecurityPrivacy.Retention,
			),
			PrivacyRequirements: applicationWizardItems(
				input.SecurityPrivacy.PrivacyRequirements,
			),
		},
		UIUX: application.IntakeDossierUIUXProjection{
			Views:         applicationWizardItems(input.UIUX.Views),
			Navigation:    applicationWizardItems(input.UIUX.Navigation),
			EmptyStates:   applicationWizardItems(input.UIUX.EmptyStates),
			ErrorStates:   applicationWizardItems(input.UIUX.ErrorStates),
			Accessibility: applicationWizardItems(input.UIUX.Accessibility),
			Delivery:      applicationWizardItems(input.UIUX.Delivery),
		},
		I18NL10N: applicationWizardI18N(input.I18NL10N),
		DeployOperations: application.IntakeDossierDeployOperationsProjection{
			Environments: applicationWizardItems(
				input.DeployOperations.Environments,
			),
			DeploymentUnits: applicationWizardItems(
				input.DeployOperations.DeploymentUnits,
			),
			Network: applicationWizardItems(input.DeployOperations.Network),
			Observability: applicationWizardItems(
				input.DeployOperations.Observability,
			),
			Backups:  applicationWizardItems(input.DeployOperations.Backups),
			Updates:  applicationWizardItems(input.DeployOperations.Updates),
			Rollback: applicationWizardItems(input.DeployOperations.Rollback),
		},
		Decisions:               applicationWizardDecisions(input.Decisions),
		Risks:                   applicationWizardRisks(input.Risks),
		OpenIssues:              applicationWizardItems(input.OpenIssues),
		DeferredDecisions:       applicationWizardItems(input.DeferredDecisions),
		IncludeDecisionsDiagram: input.IncludeDecisionsDiagram,
	}
}

func applicationWizardItems(
	input []wizardItemInput,
) []application.IntakeDossierProjectionItem {
	result := make([]application.IntakeDossierProjectionItem, len(input))
	for index, value := range input {
		result[index] = application.IntakeDossierProjectionItem{
			Key: value.Key, Summary: value.Summary,
		}
	}
	return result
}

func applicationWizardSteps(
	input []wizardStepInput,
) []application.IntakeDossierProjectionStep {
	result := make([]application.IntakeDossierProjectionStep, len(input))
	for index, value := range input {
		result[index] = application.IntakeDossierProjectionStep{
			Order: value.Order, Key: value.Key, Summary: value.Summary,
		}
	}
	return result
}

func applicationWizardRoles(
	input []wizardRoleInput,
) []application.IntakeDossierProjectionRole {
	result := make([]application.IntakeDossierProjectionRole, len(input))
	for index, value := range input {
		result[index] = application.IntakeDossierProjectionRole{
			Key: value.Key, Summary: value.Summary,
			PermissionSummaries: applicationWizardItems(
				value.PermissionSummaries,
			),
		}
	}
	return result
}

func applicationWizardIntegrations(
	input []wizardIntegrationInput,
) []application.IntakeDossierIntegrationProjection {
	result := make(
		[]application.IntakeDossierIntegrationProjection,
		len(input),
	)
	for index, value := range input {
		result[index] = application.IntakeDossierIntegrationProjection{
			Key: value.Key, Summary: value.Summary,
			Authentication: value.Authentication, DataFlow: value.DataFlow,
		}
	}
	return result
}

func applicationWizardI18N(
	input wizardI18NL10NInput,
) application.IntakeDossierI18NL10NProjection {
	locales := make(
		[]application.IntakeDossierLocaleProjection,
		len(input.Locales),
	)
	for index, value := range input.Locales {
		locales[index] = application.IntakeDossierLocaleProjection{
			Tag: value.Tag, Summary: value.Summary,
		}
	}
	return application.IntakeDossierI18NL10NProjection{
		Locales: locales, FallbackLocale: input.FallbackLocale,
		Catalog:         input.Catalog,
		Formats:         applicationWizardItems(input.Formats),
		VisibleSurfaces: applicationWizardItems(input.VisibleSurfaces),
		VisibleTextRule: input.VisibleTextRule,
	}
}

func applicationWizardDecisions(
	input []wizardDecisionInput,
) []application.IntakeDossierDecisionProjection {
	result := make(
		[]application.IntakeDossierDecisionProjection,
		len(input),
	)
	for index, value := range input {
		result[index] = application.IntakeDossierDecisionProjection{
			QuestionRef:            value.QuestionRef,
			ChoiceSummary:          value.ChoiceSummary,
			RecommendationSummary:  value.RecommendationSummary,
			DeviationJustification: value.DeviationJustification,
		}
	}
	return result
}

func applicationWizardRisks(
	input []wizardRiskInput,
) []application.IntakeDossierRiskProjection {
	result := make([]application.IntakeDossierRiskProjection, len(input))
	for index, value := range input {
		result[index] = application.IntakeDossierRiskProjection{
			Ref: value.Ref, Summary: value.Summary,
			Mitigation: value.Mitigation, StatusKey: value.StatusKey,
		}
	}
	return result
}
