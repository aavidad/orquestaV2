package application

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"orquesta/internal/intake"
)

// IntakeDossierProjectionItem is plain, structured dossier content. Summary is
// escaped by the projector; it is never interpreted as caller-supplied
// Markdown.
type IntakeDossierProjectionItem struct {
	Key     string
	Summary string
}

type IntakeDossierProjectionStep struct {
	Order   uint32
	Key     string
	Summary string
}

type IntakeDossierProjectionRole struct {
	Key                 string
	Summary             string
	PermissionSummaries []IntakeDossierProjectionItem
}

type IntakeDossierProductScopeProjection struct {
	InScope    []IntakeDossierProjectionItem
	OutOfScope []IntakeDossierProjectionItem
}

type IntakeDossierUsersRolesProjection struct {
	Users    []IntakeDossierProjectionItem
	Roles    []IntakeDossierProjectionRole
	MainFlow []IntakeDossierProjectionStep
}

type IntakeDossierArchitectureProjection struct {
	Rationale       string
	Domain          string
	Application     string
	Ports           []IntakeDossierProjectionItem
	Adapters        []IntakeDossierProjectionItem
	CompositionRoot string
	CoreLimits      []IntakeDossierProjectionItem
}

type IntakeDossierDataProjection struct {
	Entities     []IntakeDossierProjectionItem
	Lifecycles   []IntakeDossierProjectionItem
	ImportExport []IntakeDossierProjectionItem
	Backups      []IntakeDossierProjectionItem
	Versioning   []IntakeDossierProjectionItem
}

type IntakeDossierIntegrationProjection struct {
	Key            string
	Summary        string
	Authentication string
	DataFlow       string
}

type IntakeDossierIntegrationsProjection struct {
	Connectors []IntakeDossierIntegrationProjection
}

type IntakeDossierSecurityPrivacyProjection struct {
	DataClasses         []IntakeDossierProjectionItem
	Authentication      []IntakeDossierProjectionItem
	Authorization       []IntakeDossierProjectionItem
	Audit               []IntakeDossierProjectionItem
	Retention           []IntakeDossierProjectionItem
	PrivacyRequirements []IntakeDossierProjectionItem
}

type IntakeDossierUIUXProjection struct {
	Views         []IntakeDossierProjectionItem
	Navigation    []IntakeDossierProjectionItem
	EmptyStates   []IntakeDossierProjectionItem
	ErrorStates   []IntakeDossierProjectionItem
	Accessibility []IntakeDossierProjectionItem
	Delivery      []IntakeDossierProjectionItem
}

type IntakeDossierLocaleProjection struct {
	Tag     string
	Summary string
}

type IntakeDossierI18NL10NProjection struct {
	Locales         []IntakeDossierLocaleProjection
	FallbackLocale  string
	Catalog         string
	Formats         []IntakeDossierProjectionItem
	VisibleSurfaces []IntakeDossierProjectionItem
	VisibleTextRule string
}

type IntakeDossierDeployOperationsProjection struct {
	Environments    []IntakeDossierProjectionItem
	DeploymentUnits []IntakeDossierProjectionItem
	Network         []IntakeDossierProjectionItem
	Observability   []IntakeDossierProjectionItem
	Backups         []IntakeDossierProjectionItem
	Updates         []IntakeDossierProjectionItem
	Rollback        []IntakeDossierProjectionItem
}

// IntakeDossierDecisionProjection describes how one durable intake.Decision
// should be presented. The projector derives the decision itself from its
// IntakeRecord; callers only supply human summaries keyed by QuestionRef.
type IntakeDossierDecisionProjection struct {
	QuestionRef            intake.QuestionRef
	ChoiceSummary          string
	RecommendationSummary  string
	DeviationJustification string
}

type intakeDossierBoundDecisionProjection struct {
	Decision intake.Decision
	View     IntakeDossierDecisionProjection
}

type IntakeDossierRiskProjection struct {
	Ref        IntakeRiskRef
	Summary    string
	Mitigation string
	StatusKey  intake.MessageKey
}

// IntakeDossierProjectionSpec contains no Markdown, diagram source, provider
// input or persistence handle. Plan is copied and decisions are derived from
// the supplied immutable IntakeRecord, without another store or authority.
type IntakeDossierProjectionSpec struct {
	Statement string
	Objective string

	ProductScope     IntakeDossierProductScopeProjection
	UsersRoles       IntakeDossierUsersRolesProjection
	Architecture     IntakeDossierArchitectureProjection
	Data             IntakeDossierDataProjection
	Integrations     IntakeDossierIntegrationsProjection
	SecurityPrivacy  IntakeDossierSecurityPrivacyProjection
	UIUX             IntakeDossierUIUXProjection
	I18NL10N         IntakeDossierI18NL10NProjection
	DeployOperations IntakeDossierDeployOperationsProjection

	Decisions         []IntakeDossierDecisionProjection
	Risks             []IntakeDossierRiskProjection
	OpenIssues        []IntakeDossierProjectionItem
	DeferredDecisions []IntakeDossierProjectionItem
	Plan              PlanSpec

	IncludeDecisionsDiagram bool

	boundDecisions []intakeDossierBoundDecisionProjection
}

// IntakeDossierProjection is an immutable, request-local compilation. Accessors
// return defensive copies, so concurrent reads cannot mutate future results.
type IntakeDossierProjection struct {
	input         IntakeDossierInput
	plan          PlanSpec
	decisions     []intake.Decision
	decisionViews []IntakeDossierDecisionProjection
}

func NewIntakeDossierProjection(
	record IntakeRecord,
	spec IntakeDossierProjectionSpec,
) (IntakeDossierProjection, error) {
	decisions, err := intakeDossierProjectionRecordDecisions(record)
	if err != nil {
		return IntakeDossierProjection{}, err
	}
	canonical := canonicalIntakeDossierProjectionSpec(spec)
	bound, err := bindIntakeDossierProjectionDecisions(
		canonical.Decisions, decisions,
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
		input: input, plan: cloneIntakeDossierPlan(canonical.Plan),
		decisions:     append([]intake.Decision(nil), decisions...),
		decisionViews: intakeDossierProjectionDecisionViews(bound),
	}, nil
}

// ProjectIntakeDossierInput is the one-shot projection API. Callers that also
// need the exact plan should retain NewIntakeDossierProjection's result and
// use Plan, ensuring display and BuildIntakeDossier receive the same plan.
func ProjectIntakeDossierInput(
	record IntakeRecord,
	spec IntakeDossierProjectionSpec,
) (IntakeDossierInput, error) {
	projection, err := NewIntakeDossierProjection(record, spec)
	if err != nil {
		return IntakeDossierInput{}, err
	}
	return projection.Input(), nil
}

func (projection IntakeDossierProjection) Input() IntakeDossierInput {
	return IntakeDossierInput{
		Statement: projection.input.Statement,
		Objective: projection.input.Objective,
		Sections:  cloneIntakeDossierSections(projection.input.Sections),
		Diagrams:  cloneIntakeDossierDiagrams(projection.input.Diagrams),
		RiskRefs:  append([]IntakeRiskRef(nil), projection.input.RiskRefs...),
	}
}

func (projection IntakeDossierProjection) Plan() PlanSpec {
	return cloneIntakeDossierPlan(projection.plan)
}

func (projection IntakeDossierProjection) Decisions() []intake.Decision {
	return append([]intake.Decision(nil), projection.decisions...)
}

func (projection IntakeDossierProjection) DecisionViews() []IntakeDossierDecisionProjection {
	return cloneIntakeDossierProjectionDecisions(projection.decisionViews)
}

func validateIntakeDossierProjectionSpec(spec IntakeDossierProjectionSpec) error {
	requiredTexts := []struct {
		field string
		value string
	}{
		{"statement", spec.Statement},
		{"objective", spec.Objective},
		{"architecture.rationale", spec.Architecture.Rationale},
		{"architecture.domain", spec.Architecture.Domain},
		{"architecture.application", spec.Architecture.Application},
		{"architecture.composition_root", spec.Architecture.CompositionRoot},
		{"i18n_l10n.fallback_locale", spec.I18NL10N.FallbackLocale},
		{"i18n_l10n.catalog", spec.I18NL10N.Catalog},
		{"i18n_l10n.visible_text_rule", spec.I18NL10N.VisibleTextRule},
	}
	for _, required := range requiredTexts {
		if !validIntakeDossierProjectionText(required.value) {
			return invalidIntakeDossier("projection."+required.field, nil)
		}
	}

	itemSets := []struct {
		field  string
		values []IntakeDossierProjectionItem
	}{
		{"product_scope.in_scope", spec.ProductScope.InScope},
		{"product_scope.out_of_scope", spec.ProductScope.OutOfScope},
		{"users_roles.users", spec.UsersRoles.Users},
		{"architecture.ports", spec.Architecture.Ports},
		{"architecture.adapters", spec.Architecture.Adapters},
		{"architecture.core_limits", spec.Architecture.CoreLimits},
		{"data.entities", spec.Data.Entities},
		{"data.lifecycles", spec.Data.Lifecycles},
		{"data.import_export", spec.Data.ImportExport},
		{"data.backups", spec.Data.Backups},
		{"security_privacy.data_classes", spec.SecurityPrivacy.DataClasses},
		{"security_privacy.authentication", spec.SecurityPrivacy.Authentication},
		{"security_privacy.authorization", spec.SecurityPrivacy.Authorization},
		{"security_privacy.audit", spec.SecurityPrivacy.Audit},
		{"security_privacy.retention", spec.SecurityPrivacy.Retention},
		{"security_privacy.privacy_requirements", spec.SecurityPrivacy.PrivacyRequirements},
		{"ui_ux.views", spec.UIUX.Views},
		{"ui_ux.navigation", spec.UIUX.Navigation},
		{"ui_ux.empty_states", spec.UIUX.EmptyStates},
		{"ui_ux.error_states", spec.UIUX.ErrorStates},
		{"ui_ux.accessibility", spec.UIUX.Accessibility},
		{"ui_ux.delivery", spec.UIUX.Delivery},
		{"i18n_l10n.formats", spec.I18NL10N.Formats},
		{"i18n_l10n.visible_surfaces", spec.I18NL10N.VisibleSurfaces},
		{"deploy_operations.environments", spec.DeployOperations.Environments},
		{"deploy_operations.deployment_units", spec.DeployOperations.DeploymentUnits},
		{"deploy_operations.network", spec.DeployOperations.Network},
		{"deploy_operations.observability", spec.DeployOperations.Observability},
		{"deploy_operations.backups", spec.DeployOperations.Backups},
		{"deploy_operations.updates", spec.DeployOperations.Updates},
		{"deploy_operations.rollback", spec.DeployOperations.Rollback},
	}
	for _, set := range itemSets {
		if err := validateIntakeDossierProjectionItems(set.values, true); err != nil {
			return invalidIntakeDossier("projection."+set.field, err)
		}
	}
	optionalItemSets := []struct {
		field  string
		values []IntakeDossierProjectionItem
	}{
		{"data.versioning", spec.Data.Versioning},
		{"risks_open_issues.open_issues", spec.OpenIssues},
		{"risks_open_issues.deferred_decisions", spec.DeferredDecisions},
	}
	for _, set := range optionalItemSets {
		if err := validateIntakeDossierProjectionItems(set.values, false); err != nil {
			return invalidIntakeDossier("projection."+set.field, err)
		}
	}
	if err := validateIntakeDossierProjectionRoles(spec.UsersRoles.Roles); err != nil {
		return invalidIntakeDossier("projection.users_roles.roles", err)
	}
	if err := validateIntakeDossierProjectionSteps(spec.UsersRoles.MainFlow); err != nil {
		return invalidIntakeDossier("projection.users_roles.main_flow", err)
	}
	if err := validateIntakeDossierProjectionIntegrations(spec.Integrations.Connectors); err != nil {
		return invalidIntakeDossier("projection.integrations.connectors", err)
	}
	if err := validateIntakeDossierProjectionLocales(spec.I18NL10N); err != nil {
		return invalidIntakeDossier("projection.i18n_l10n.locales", err)
	}
	if err := validateIntakeDossierProjectionDecisions(
		spec.Decisions, spec.boundDecisions,
	); err != nil {
		return invalidIntakeDossier("projection.decisions", err)
	}
	if err := validateIntakeDossierProjectionRisks(spec.Risks); err != nil {
		return invalidIntakeDossier("projection.risks", err)
	}
	if err := validateIntakeDossierPlan(spec.Plan); err != nil {
		return invalidIntakeDossier("projection.plan", err)
	}
	var criteria, requiredTests int
	for _, phase := range spec.Plan.Phases {
		criteria += len(phase.CriterionRefs)
	}
	for _, item := range spec.Plan.WorkItems {
		requiredTests += len(item.RequiredTests)
	}
	if criteria == 0 || requiredTests == 0 {
		return invalidIntakeDossier("projection.plan.minimum_evidence", nil)
	}
	return nil
}

func validateIntakeDossierProjectionItems(
	values []IntakeDossierProjectionItem,
	required bool,
) error {
	if required && len(values) == 0 {
		return fmt.Errorf("required")
	}
	seen := make(map[string]struct{}, len(values))
	for index, value := range values {
		if !validIntakeDossierProjectionKey(value.Key) ||
			!validIntakeDossierProjectionText(value.Summary) {
			return fmt.Errorf("item[%d]", index)
		}
		if _, duplicate := seen[value.Key]; duplicate {
			return fmt.Errorf("duplicate_key")
		}
		seen[value.Key] = struct{}{}
	}
	return nil
}

func validateIntakeDossierProjectionRoles(
	values []IntakeDossierProjectionRole,
) error {
	if len(values) == 0 {
		return fmt.Errorf("required")
	}
	seen := make(map[string]struct{}, len(values))
	for index, value := range values {
		if !validIntakeDossierProjectionKey(value.Key) ||
			!validIntakeDossierProjectionText(value.Summary) {
			return fmt.Errorf("role[%d]", index)
		}
		if _, duplicate := seen[value.Key]; duplicate {
			return fmt.Errorf("duplicate_key")
		}
		if err := validateIntakeDossierProjectionItems(
			value.PermissionSummaries, true,
		); err != nil {
			return fmt.Errorf("role[%d].permissions", index)
		}
		seen[value.Key] = struct{}{}
	}
	return nil
}

func validateIntakeDossierProjectionSteps(
	values []IntakeDossierProjectionStep,
) error {
	if len(values) < 2 {
		return fmt.Errorf("at_least_two_required")
	}
	seenKeys := make(map[string]struct{}, len(values))
	seenOrders := make(map[uint32]struct{}, len(values))
	for index, value := range values {
		if value.Order == 0 || !validIntakeDossierProjectionKey(value.Key) ||
			!validIntakeDossierProjectionText(value.Summary) {
			return fmt.Errorf("step[%d]", index)
		}
		if _, duplicate := seenKeys[value.Key]; duplicate {
			return fmt.Errorf("duplicate_key")
		}
		if _, duplicate := seenOrders[value.Order]; duplicate {
			return fmt.Errorf("duplicate_order")
		}
		seenKeys[value.Key] = struct{}{}
		seenOrders[value.Order] = struct{}{}
	}
	return nil
}

func validateIntakeDossierProjectionIntegrations(
	values []IntakeDossierIntegrationProjection,
) error {
	if len(values) == 0 {
		return fmt.Errorf("required")
	}
	seen := make(map[string]struct{}, len(values))
	for index, value := range values {
		if !validIntakeDossierProjectionKey(value.Key) ||
			!validIntakeDossierProjectionText(value.Summary) ||
			!validIntakeDossierProjectionText(value.Authentication) ||
			!validIntakeDossierProjectionText(value.DataFlow) {
			return fmt.Errorf("connector[%d]", index)
		}
		if _, duplicate := seen[value.Key]; duplicate {
			return fmt.Errorf("duplicate_key")
		}
		seen[value.Key] = struct{}{}
	}
	return nil
}

func validateIntakeDossierProjectionLocales(
	value IntakeDossierI18NL10NProjection,
) error {
	if len(value.Locales) == 0 {
		return fmt.Errorf("required")
	}
	seen := make(map[string]struct{}, len(value.Locales))
	fallbackFound := false
	for index, locale := range value.Locales {
		if !validIntakeDossierProjectionLocale(locale.Tag) ||
			!validIntakeDossierProjectionText(locale.Summary) {
			return fmt.Errorf("locale[%d]", index)
		}
		key := strings.ToLower(locale.Tag)
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("duplicate_tag")
		}
		seen[key] = struct{}{}
		fallbackFound = fallbackFound ||
			strings.EqualFold(locale.Tag, value.FallbackLocale)
	}
	if !fallbackFound {
		return fmt.Errorf("fallback_not_declared")
	}
	return nil
}

func validateIntakeDossierProjectionDecisions(
	values []IntakeDossierDecisionProjection,
	bound []intakeDossierBoundDecisionProjection,
) error {
	if len(values) == 0 || len(values) != len(bound) {
		return fmt.Errorf("required")
	}
	viewsByQuestion := make(
		map[intake.QuestionRef]IntakeDossierDecisionProjection,
		len(values),
	)
	for index, value := range values {
		if !validIntakeDossierRef(string(value.QuestionRef), "intake-question:") ||
			!validIntakeDossierProjectionText(value.ChoiceSummary) ||
			!validIntakeDossierProjectionText(value.RecommendationSummary) ||
			(value.DeviationJustification != "" &&
				!validIntakeDossierProjectionText(value.DeviationJustification)) {
			return fmt.Errorf("decision[%d]", index)
		}
		if _, duplicate := viewsByQuestion[value.QuestionRef]; duplicate {
			return fmt.Errorf("duplicate_question")
		}
		viewsByQuestion[value.QuestionRef] = value
	}
	for index, value := range bound {
		view, found := viewsByQuestion[value.Decision.QuestionRef]
		if !found || view != value.View {
			return fmt.Errorf("decision_binding[%d]", index)
		}
	}
	return nil
}

func intakeDossierProjectionRecordDecisions(
	record IntakeRecord,
) ([]intake.Decision, error) {
	if err := validateStoredIntakeRecord(
		record.ActorRef, record.ProjectRef, record.State.Ref(), record,
	); err != nil {
		return nil, invalidIntakeDossier("projection.record", err)
	}
	stateDigest, err := IntakeStateDigest(record.State)
	if err != nil || stateDigest != record.Receipt.StateDigest {
		return nil, invalidIntakeDossier("projection.record.state_digest", err)
	}
	decisions, err := currentIntakeDossierDecisions(record.State)
	if err != nil {
		return nil, err
	}
	return decisions, nil
}

func bindIntakeDossierProjectionDecisions(
	views []IntakeDossierDecisionProjection,
	decisions []intake.Decision,
) ([]intakeDossierBoundDecisionProjection, error) {
	if len(views) != len(decisions) {
		return nil, invalidIntakeDossier(
			"projection.decisions.record_binding", nil,
		)
	}
	byQuestion := make(map[intake.QuestionRef]IntakeDossierDecisionProjection, len(views))
	for _, view := range views {
		if _, duplicate := byQuestion[view.QuestionRef]; duplicate {
			return nil, invalidIntakeDossier(
				"projection.decisions.record_binding", nil,
			)
		}
		byQuestion[view.QuestionRef] = view
	}
	bound := make([]intakeDossierBoundDecisionProjection, len(decisions))
	for index, decision := range decisions {
		view, found := byQuestion[decision.QuestionRef]
		if !found {
			return nil, invalidIntakeDossier(
				"projection.decisions.record_binding", nil,
			)
		}
		bound[index] = intakeDossierBoundDecisionProjection{
			Decision: decision, View: view,
		}
	}
	return bound, nil
}

func intakeDossierProjectionDecisionViews(
	values []intakeDossierBoundDecisionProjection,
) []IntakeDossierDecisionProjection {
	views := make([]IntakeDossierDecisionProjection, len(values))
	for index, value := range values {
		views[index] = value.View
	}
	return views
}

func validateIntakeDossierProjectionRisks(
	values []IntakeDossierRiskProjection,
) error {
	if len(values) == 0 {
		return fmt.Errorf("required")
	}
	seen := make(map[IntakeRiskRef]struct{}, len(values))
	for index, value := range values {
		if !validIntakeDossierRef(string(value.Ref), "intake-risk:") ||
			!validIntakeDossierProjectionText(value.Summary) ||
			!validIntakeDossierProjectionText(value.Mitigation) ||
			!validIntakeDossierMessageKey(value.StatusKey) {
			return fmt.Errorf("risk[%d]", index)
		}
		if _, duplicate := seen[value.Ref]; duplicate {
			return fmt.Errorf("duplicate_ref")
		}
		seen[value.Ref] = struct{}{}
	}
	return nil
}

func validIntakeDossierProjectionKey(value string) bool {
	if value == "" || strings.TrimSpace(value) != value ||
		strings.HasPrefix(value, ".") || strings.HasSuffix(value, ".") ||
		strings.Contains(value, "..") {
		return false
	}
	for _, char := range value {
		if (char < 'a' || char > 'z') && (char < '0' || char > '9') &&
			char != '.' && char != '_' && char != '-' {
			return false
		}
	}
	return true
}

func validIntakeDossierProjectionText(value string) bool {
	if strings.TrimSpace(value) == "" || strings.TrimSpace(value) != value {
		return false
	}
	for _, char := range value {
		if unicode.IsControl(char) && char != '\n' && char != '\r' && char != '\t' {
			return false
		}
	}
	return true
}

func validIntakeDossierProjectionLocale(value string) bool {
	if value == "" || strings.TrimSpace(value) != value {
		return false
	}
	parts := strings.Split(value, "-")
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, char := range part {
			if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') &&
				(char < '0' || char > '9') {
				return false
			}
		}
	}
	return true
}

func canonicalIntakeDossierProjectionSpec(
	spec IntakeDossierProjectionSpec,
) IntakeDossierProjectionSpec {
	cloned := cloneIntakeDossierProjectionSpec(spec)
	sortProjectionItems(cloned.ProductScope.InScope)
	sortProjectionItems(cloned.ProductScope.OutOfScope)
	sortProjectionItems(cloned.UsersRoles.Users)
	sort.Slice(cloned.UsersRoles.Roles, func(left, right int) bool {
		return cloned.UsersRoles.Roles[left].Key < cloned.UsersRoles.Roles[right].Key
	})
	for index := range cloned.UsersRoles.Roles {
		sortProjectionItems(cloned.UsersRoles.Roles[index].PermissionSummaries)
	}
	sort.Slice(cloned.UsersRoles.MainFlow, func(left, right int) bool {
		if cloned.UsersRoles.MainFlow[left].Order != cloned.UsersRoles.MainFlow[right].Order {
			return cloned.UsersRoles.MainFlow[left].Order < cloned.UsersRoles.MainFlow[right].Order
		}
		return cloned.UsersRoles.MainFlow[left].Key < cloned.UsersRoles.MainFlow[right].Key
	})
	for _, values := range [][]IntakeDossierProjectionItem{
		cloned.Architecture.Ports, cloned.Architecture.Adapters,
		cloned.Architecture.CoreLimits, cloned.Data.Entities,
		cloned.Data.Lifecycles, cloned.Data.ImportExport, cloned.Data.Backups,
		cloned.Data.Versioning, cloned.SecurityPrivacy.DataClasses,
		cloned.SecurityPrivacy.Authentication, cloned.SecurityPrivacy.Authorization,
		cloned.SecurityPrivacy.Audit, cloned.SecurityPrivacy.Retention,
		cloned.SecurityPrivacy.PrivacyRequirements, cloned.UIUX.Views,
		cloned.UIUX.Navigation, cloned.UIUX.EmptyStates, cloned.UIUX.ErrorStates,
		cloned.UIUX.Accessibility, cloned.UIUX.Delivery, cloned.I18NL10N.Formats,
		cloned.I18NL10N.VisibleSurfaces, cloned.DeployOperations.Environments,
		cloned.DeployOperations.DeploymentUnits, cloned.DeployOperations.Network,
		cloned.DeployOperations.Observability, cloned.DeployOperations.Backups,
		cloned.DeployOperations.Updates, cloned.DeployOperations.Rollback,
		cloned.OpenIssues, cloned.DeferredDecisions,
	} {
		sortProjectionItems(values)
	}
	sort.Slice(cloned.Integrations.Connectors, func(left, right int) bool {
		return cloned.Integrations.Connectors[left].Key <
			cloned.Integrations.Connectors[right].Key
	})
	sort.Slice(cloned.I18NL10N.Locales, func(left, right int) bool {
		return strings.ToLower(cloned.I18NL10N.Locales[left].Tag) <
			strings.ToLower(cloned.I18NL10N.Locales[right].Tag)
	})
	sort.Slice(cloned.Decisions, func(left, right int) bool {
		return cloned.Decisions[left].QuestionRef <
			cloned.Decisions[right].QuestionRef
	})
	sort.Slice(cloned.Risks, func(left, right int) bool {
		return cloned.Risks[left].Ref < cloned.Risks[right].Ref
	})
	return cloned
}

func sortProjectionItems(values []IntakeDossierProjectionItem) {
	sort.Slice(values, func(left, right int) bool {
		return values[left].Key < values[right].Key
	})
}

func cloneIntakeDossierProjectionSpec(
	spec IntakeDossierProjectionSpec,
) IntakeDossierProjectionSpec {
	cloned := spec
	cloned.ProductScope.InScope = cloneProjectionItems(spec.ProductScope.InScope)
	cloned.ProductScope.OutOfScope = cloneProjectionItems(spec.ProductScope.OutOfScope)
	cloned.UsersRoles.Users = cloneProjectionItems(spec.UsersRoles.Users)
	cloned.UsersRoles.Roles = append([]IntakeDossierProjectionRole(nil), spec.UsersRoles.Roles...)
	for index := range cloned.UsersRoles.Roles {
		cloned.UsersRoles.Roles[index].PermissionSummaries = cloneProjectionItems(
			spec.UsersRoles.Roles[index].PermissionSummaries,
		)
	}
	cloned.UsersRoles.MainFlow = append(
		[]IntakeDossierProjectionStep(nil), spec.UsersRoles.MainFlow...,
	)
	cloned.Architecture.Ports = cloneProjectionItems(spec.Architecture.Ports)
	cloned.Architecture.Adapters = cloneProjectionItems(spec.Architecture.Adapters)
	cloned.Architecture.CoreLimits = cloneProjectionItems(spec.Architecture.CoreLimits)
	cloned.Data.Entities = cloneProjectionItems(spec.Data.Entities)
	cloned.Data.Lifecycles = cloneProjectionItems(spec.Data.Lifecycles)
	cloned.Data.ImportExport = cloneProjectionItems(spec.Data.ImportExport)
	cloned.Data.Backups = cloneProjectionItems(spec.Data.Backups)
	cloned.Data.Versioning = cloneProjectionItems(spec.Data.Versioning)
	cloned.Integrations.Connectors = append(
		[]IntakeDossierIntegrationProjection(nil), spec.Integrations.Connectors...,
	)
	cloned.SecurityPrivacy.DataClasses = cloneProjectionItems(spec.SecurityPrivacy.DataClasses)
	cloned.SecurityPrivacy.Authentication = cloneProjectionItems(spec.SecurityPrivacy.Authentication)
	cloned.SecurityPrivacy.Authorization = cloneProjectionItems(spec.SecurityPrivacy.Authorization)
	cloned.SecurityPrivacy.Audit = cloneProjectionItems(spec.SecurityPrivacy.Audit)
	cloned.SecurityPrivacy.Retention = cloneProjectionItems(spec.SecurityPrivacy.Retention)
	cloned.SecurityPrivacy.PrivacyRequirements = cloneProjectionItems(spec.SecurityPrivacy.PrivacyRequirements)
	cloned.UIUX.Views = cloneProjectionItems(spec.UIUX.Views)
	cloned.UIUX.Navigation = cloneProjectionItems(spec.UIUX.Navigation)
	cloned.UIUX.EmptyStates = cloneProjectionItems(spec.UIUX.EmptyStates)
	cloned.UIUX.ErrorStates = cloneProjectionItems(spec.UIUX.ErrorStates)
	cloned.UIUX.Accessibility = cloneProjectionItems(spec.UIUX.Accessibility)
	cloned.UIUX.Delivery = cloneProjectionItems(spec.UIUX.Delivery)
	cloned.I18NL10N.Locales = append(
		[]IntakeDossierLocaleProjection(nil), spec.I18NL10N.Locales...,
	)
	cloned.I18NL10N.Formats = cloneProjectionItems(spec.I18NL10N.Formats)
	cloned.I18NL10N.VisibleSurfaces = cloneProjectionItems(spec.I18NL10N.VisibleSurfaces)
	cloned.DeployOperations.Environments = cloneProjectionItems(spec.DeployOperations.Environments)
	cloned.DeployOperations.DeploymentUnits = cloneProjectionItems(spec.DeployOperations.DeploymentUnits)
	cloned.DeployOperations.Network = cloneProjectionItems(spec.DeployOperations.Network)
	cloned.DeployOperations.Observability = cloneProjectionItems(spec.DeployOperations.Observability)
	cloned.DeployOperations.Backups = cloneProjectionItems(spec.DeployOperations.Backups)
	cloned.DeployOperations.Updates = cloneProjectionItems(spec.DeployOperations.Updates)
	cloned.DeployOperations.Rollback = cloneProjectionItems(spec.DeployOperations.Rollback)
	cloned.Decisions = cloneIntakeDossierProjectionDecisions(spec.Decisions)
	cloned.Risks = append([]IntakeDossierRiskProjection(nil), spec.Risks...)
	cloned.OpenIssues = cloneProjectionItems(spec.OpenIssues)
	cloned.DeferredDecisions = cloneProjectionItems(spec.DeferredDecisions)
	cloned.Plan = cloneIntakeDossierPlan(spec.Plan)
	return cloned
}

func cloneProjectionItems(
	values []IntakeDossierProjectionItem,
) []IntakeDossierProjectionItem {
	return append([]IntakeDossierProjectionItem(nil), values...)
}

func cloneIntakeDossierProjectionDecisions(
	values []IntakeDossierDecisionProjection,
) []IntakeDossierDecisionProjection {
	return append([]IntakeDossierDecisionProjection(nil), values...)
}
