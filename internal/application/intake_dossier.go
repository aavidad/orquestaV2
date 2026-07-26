package application

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/intake"
)

const IntakeDossierSchema = "orquesta.intake.dossier.v1"

var ErrIntakeDossierInvalid = errors.New("application.intake_dossier_invalid")

type IntakeDossierRef string
type IntakeDossierSectionRef string
type IntakeDossierDiagramRef string
type IntakeRiskRef string

type IntakeDossierSectionKind string

const (
	IntakeDossierSectionProductScope      IntakeDossierSectionKind = "product_scope"
	IntakeDossierSectionUsersRoles        IntakeDossierSectionKind = "users_roles"
	IntakeDossierSectionArchitecture      IntakeDossierSectionKind = "architecture"
	IntakeDossierSectionData              IntakeDossierSectionKind = "data"
	IntakeDossierSectionIntegrations      IntakeDossierSectionKind = "integrations"
	IntakeDossierSectionSecurityPrivacy   IntakeDossierSectionKind = "security_privacy"
	IntakeDossierSectionUIUX              IntakeDossierSectionKind = "ui_ux"
	IntakeDossierSectionI18NL10N          IntakeDossierSectionKind = "i18n_l10n"
	IntakeDossierSectionDeployOperations  IntakeDossierSectionKind = "deploy_operations"
	IntakeDossierSectionOrchestrationPlan IntakeDossierSectionKind = "orchestration_plan"
	IntakeDossierSectionRisksOpenIssues   IntakeDossierSectionKind = "risks_open_issues"
)

type IntakeDossierDiagramPurpose string

const (
	IntakeDossierDiagramArchitecture     IntakeDossierDiagramPurpose = "architecture"
	IntakeDossierDiagramUserFlow         IntakeDossierDiagramPurpose = "user_flow"
	IntakeDossierDiagramDataIntegrations IntakeDossierDiagramPurpose = "data_integrations"
	IntakeDossierDiagramI18N             IntakeDossierDiagramPurpose = "i18n"
	IntakeDossierDiagramDeployment       IntakeDossierDiagramPurpose = "deployment"
	IntakeDossierDiagramDecisions        IntakeDossierDiagramPurpose = "decisions"
)

type IntakeDossierDiagramKind string

const (
	IntakeDossierDiagramMermaid          IntakeDossierDiagramKind = "mermaid"
	IntakeDossierDiagramBlueprint        IntakeDossierDiagramKind = "diagram_blueprint"
	IntakeDossierDiagramInfographicBrief IntakeDossierDiagramKind = "infographic_brief"
)

type IntakeDossierSection struct {
	Ref      IntakeDossierSectionRef  `json:"ref"`
	Kind     IntakeDossierSectionKind `json:"kind"`
	TitleKey intake.MessageKey        `json:"title_key"`
	Markdown string                   `json:"markdown"`
}

type IntakeDossierDiagram struct {
	Ref        IntakeDossierDiagramRef     `json:"ref"`
	Purpose    IntakeDossierDiagramPurpose `json:"purpose"`
	Kind       IntakeDossierDiagramKind    `json:"kind"`
	Source     string                      `json:"source"`
	AltTextKey intake.MessageKey           `json:"alt_text_key"`
}

type IntakeDossierInput struct {
	Statement string                 `json:"statement"`
	Objective string                 `json:"objective"`
	Sections  []IntakeDossierSection `json:"sections"`
	Diagrams  []IntakeDossierDiagram `json:"diagrams"`
	RiskRefs  []IntakeRiskRef        `json:"risk_refs,omitempty"`
}

// IntakeDossier is a content-addressed application proposal. Its only builder
// accepts the durable IntakeRecord and concrete PlanSpec, so callers cannot
// substitute unverified state or plan digests.
type IntakeDossier struct {
	ref                    IntakeDossierRef
	actorRef               goal.ActorRef
	projectRef             goal.ProjectRef
	stateRef               intake.Ref
	stateRevision          intake.Revision
	stateDigest            string
	sourceIntakeReceiptRef string
	statement              string
	objective              string
	sections               []IntakeDossierSection
	diagrams               []IntakeDossierDiagram
	decisions              []intake.Decision
	riskRefs               []IntakeRiskRef
	plan                   PlanSpec
	planDigest             string
	digest                 string
}

func BuildIntakeDossier(
	record IntakeRecord,
	plan PlanSpec,
	input IntakeDossierInput,
) (IntakeDossier, error) {
	if err := validateStoredIntakeRecord(
		record.ActorRef, record.ProjectRef, record.State.Ref(), record,
	); err != nil {
		return IntakeDossier{}, invalidIntakeDossier("record", err)
	}
	stateDigest, err := IntakeStateDigest(record.State)
	if err != nil || stateDigest != record.Receipt.StateDigest {
		return IntakeDossier{}, invalidIntakeDossier("state_digest", err)
	}
	decisions, err := currentIntakeDossierDecisions(record.State)
	if err != nil {
		return IntakeDossier{}, err
	}
	if err := validateIntakeDossierInput(input); err != nil {
		return IntakeDossier{}, err
	}
	if err := validateIntakeDossierPlan(plan); err != nil {
		return IntakeDossier{}, err
	}

	dossier := IntakeDossier{
		actorRef: record.ActorRef, projectRef: record.ProjectRef,
		stateRef: record.State.Ref(), stateRevision: record.State.Revision(),
		stateDigest: stateDigest, sourceIntakeReceiptRef: record.Receipt.Ref,
		statement: input.Statement, objective: input.Objective,
		sections:  cloneIntakeDossierSections(input.Sections),
		diagrams:  cloneIntakeDossierDiagrams(input.Diagrams),
		decisions: append([]intake.Decision(nil), decisions...),
		riskRefs:  append([]IntakeRiskRef(nil), input.RiskRefs...),
		plan:      cloneIntakeDossierPlan(plan),
		planDigest: IntakeDossierPlanDigest(
			plan,
		),
	}
	dossier.digest = hashIntakeDossier(dossier)
	dossier.ref = IntakeDossierRef("intake-dossier:" + dossier.digest)
	return dossier, nil
}

func IntakeDossierPlanDigest(plan PlanSpec) string {
	digest := fingerprintDigest("orquesta.intake.dossier.plan.v1")
	writePlanFingerprint(digest, &plan)
	return fingerprintHex(digest)
}

func (dossier IntakeDossier) Schema() string                 { return IntakeDossierSchema }
func (dossier IntakeDossier) Ref() IntakeDossierRef          { return dossier.ref }
func (dossier IntakeDossier) ActorRef() goal.ActorRef        { return dossier.actorRef }
func (dossier IntakeDossier) ProjectRef() goal.ProjectRef    { return dossier.projectRef }
func (dossier IntakeDossier) StateRef() intake.Ref           { return dossier.stateRef }
func (dossier IntakeDossier) StateRevision() intake.Revision { return dossier.stateRevision }
func (dossier IntakeDossier) StateDigest() string            { return dossier.stateDigest }
func (dossier IntakeDossier) SourceIntakeReceiptRef() string { return dossier.sourceIntakeReceiptRef }
func (dossier IntakeDossier) Statement() string              { return dossier.statement }
func (dossier IntakeDossier) Objective() string              { return dossier.objective }
func (dossier IntakeDossier) PlanDigest() string             { return dossier.planDigest }
func (dossier IntakeDossier) Digest() string                 { return dossier.digest }
func (dossier IntakeDossier) Sections() []IntakeDossierSection {
	return cloneIntakeDossierSections(dossier.sections)
}
func (dossier IntakeDossier) Diagrams() []IntakeDossierDiagram {
	return cloneIntakeDossierDiagrams(dossier.diagrams)
}
func (dossier IntakeDossier) Decisions() []intake.Decision {
	return append([]intake.Decision(nil), dossier.decisions...)
}
func (dossier IntakeDossier) RiskRefs() []IntakeRiskRef {
	return append([]IntakeRiskRef(nil), dossier.riskRefs...)
}
func (dossier IntakeDossier) Plan() PlanSpec { return cloneIntakeDossierPlan(dossier.plan) }

func currentIntakeDossierDecisions(state intake.State) ([]intake.Decision, error) {
	if len(state.History()) == 0 {
		return nil, invalidIntakeDossier("state_not_ready", nil)
	}
	derivedIssues := make(map[intake.IssueRef]struct{}, len(state.Issues()))
	decisions := make([]intake.Decision, 0, len(state.Questions()))
	for _, question := range state.Questions() {
		for _, issueRef := range question.DerivedFrom {
			derivedIssues[issueRef] = struct{}{}
		}
		decision, found := state.CurrentDecision(question.Ref)
		if !found {
			return nil, invalidIntakeDossier("unanswered_question", nil)
		}
		decisions = append(decisions, decision)
	}
	for _, issue := range state.Issues() {
		if _, found := derivedIssues[issue.Ref]; !found {
			return nil, invalidIntakeDossier("unresolved_issue", nil)
		}
	}
	return decisions, nil
}

func validateIntakeDossierInput(input IntakeDossierInput) error {
	if strings.TrimSpace(input.Statement) == "" {
		return invalidIntakeDossier("statement", nil)
	}
	if strings.TrimSpace(input.Objective) == "" {
		return invalidIntakeDossier("objective", nil)
	}
	if err := validateIntakeDossierSections(input.Sections); err != nil {
		return err
	}
	if err := validateIntakeDossierDiagrams(input.Diagrams); err != nil {
		return err
	}
	seenRisks := make(map[IntakeRiskRef]struct{}, len(input.RiskRefs))
	for index, ref := range input.RiskRefs {
		if !validIntakeDossierRef(string(ref), "intake-risk:") {
			return invalidIntakeDossier(indexedDossierField("risk_refs", index), nil)
		}
		if _, duplicate := seenRisks[ref]; duplicate {
			return invalidIntakeDossier("risk_refs.duplicate", nil)
		}
		seenRisks[ref] = struct{}{}
	}
	return nil
}

func validateIntakeDossierSections(sections []IntakeDossierSection) error {
	required := map[IntakeDossierSectionKind]struct{}{
		IntakeDossierSectionProductScope: {}, IntakeDossierSectionUsersRoles: {},
		IntakeDossierSectionArchitecture: {}, IntakeDossierSectionData: {},
		IntakeDossierSectionIntegrations: {}, IntakeDossierSectionSecurityPrivacy: {},
		IntakeDossierSectionUIUX: {}, IntakeDossierSectionI18NL10N: {},
		IntakeDossierSectionDeployOperations: {}, IntakeDossierSectionOrchestrationPlan: {},
		IntakeDossierSectionRisksOpenIssues: {},
	}
	seenRefs := make(map[IntakeDossierSectionRef]struct{}, len(sections))
	seenKinds := make(map[IntakeDossierSectionKind]struct{}, len(sections))
	for index, section := range sections {
		if !validIntakeDossierRef(string(section.Ref), "intake-dossier-section:") {
			return invalidIntakeDossier(indexedDossierField("sections.ref", index), nil)
		}
		if _, duplicate := seenRefs[section.Ref]; duplicate {
			return invalidIntakeDossier("sections.ref.duplicate", nil)
		}
		if _, valid := required[section.Kind]; !valid {
			return invalidIntakeDossier(indexedDossierField("sections.kind", index), nil)
		}
		if _, duplicate := seenKinds[section.Kind]; duplicate {
			return invalidIntakeDossier("sections.kind.duplicate", nil)
		}
		if !validIntakeDossierMessageKey(section.TitleKey) {
			return invalidIntakeDossier(indexedDossierField("sections.title_key", index), nil)
		}
		if strings.TrimSpace(section.Markdown) == "" {
			return invalidIntakeDossier(indexedDossierField("sections.markdown", index), nil)
		}
		seenRefs[section.Ref] = struct{}{}
		seenKinds[section.Kind] = struct{}{}
	}
	if len(seenKinds) != len(required) {
		return invalidIntakeDossier("sections.required", nil)
	}
	return nil
}

func validateIntakeDossierDiagrams(diagrams []IntakeDossierDiagram) error {
	required := map[IntakeDossierDiagramPurpose]struct{}{
		IntakeDossierDiagramArchitecture: {}, IntakeDossierDiagramUserFlow: {},
		IntakeDossierDiagramDataIntegrations: {}, IntakeDossierDiagramI18N: {},
		IntakeDossierDiagramDeployment: {},
	}
	seenRefs := make(map[IntakeDossierDiagramRef]struct{}, len(diagrams))
	seenPurposes := make(map[IntakeDossierDiagramPurpose]struct{}, len(diagrams))
	for index, diagram := range diagrams {
		if !validIntakeDossierRef(string(diagram.Ref), "intake-dossier-diagram:") {
			return invalidIntakeDossier(indexedDossierField("diagrams.ref", index), nil)
		}
		if _, duplicate := seenRefs[diagram.Ref]; duplicate {
			return invalidIntakeDossier("diagrams.ref.duplicate", nil)
		}
		if !validIntakeDossierDiagramPurpose(diagram.Purpose) {
			return invalidIntakeDossier(indexedDossierField("diagrams.purpose", index), nil)
		}
		if _, duplicate := seenPurposes[diagram.Purpose]; duplicate {
			return invalidIntakeDossier("diagrams.purpose.duplicate", nil)
		}
		if !validIntakeDossierDiagramKind(diagram.Kind) {
			return invalidIntakeDossier(indexedDossierField("diagrams.kind", index), nil)
		}
		if strings.TrimSpace(diagram.Source) == "" {
			return invalidIntakeDossier(indexedDossierField("diagrams.source", index), nil)
		}
		if !validIntakeDossierMessageKey(diagram.AltTextKey) {
			return invalidIntakeDossier(indexedDossierField("diagrams.alt_text_key", index), nil)
		}
		seenRefs[diagram.Ref] = struct{}{}
		seenPurposes[diagram.Purpose] = struct{}{}
	}
	for purpose := range required {
		if _, found := seenPurposes[purpose]; !found {
			return invalidIntakeDossier("diagrams.required", nil)
		}
	}
	return nil
}

func validateIntakeDossierPlan(plan PlanSpec) error {
	if len(plan.Phases) == 0 || len(plan.WorkItems) == 0 {
		return invalidIntakeDossier("plan.required", nil)
	}
	phases := make([]goal.PhaseInstance, 0, len(plan.Phases))
	for index, phase := range plan.Phases {
		compiled, err := compilePhaseSpec(phase)
		if err != nil {
			return invalidIntakeDossier(indexedDossierField("plan.phases", index), err)
		}
		phases = append(phases, compiled)
	}

	refs := make(map[string]goal.WorkItemRef, len(plan.WorkItems))
	for index, item := range plan.WorkItems {
		if strings.TrimSpace(item.Key) == "" ||
			strings.TrimSpace(item.Key) != item.Key {
			return invalidIntakeDossier(
				indexedDossierField("plan.work_items.key", index),
				errors.New("application.plan_item_key_invalid"),
			)
		}
		if _, duplicate := refs[item.Key]; duplicate {
			return invalidIntakeDossier("plan.work_items.key.duplicate", nil)
		}
		ref, err := goal.NewWorkItemRef(
			"work-item:intake-dossier-plan-validation:" + strconv.Itoa(index+1),
		)
		if err != nil {
			return invalidIntakeDossier(
				indexedDossierField("plan.work_items.ref", index), err,
			)
		}
		refs[item.Key] = ref
	}

	goalRef, err := goal.NewGoalRef("goal:intake-dossier-plan-validation")
	if err != nil {
		return invalidIntakeDossier("plan.validation_goal_ref", err)
	}
	actorRef, err := goal.NewActorRef("actor:intake-dossier-plan-validation")
	if err != nil {
		return invalidIntakeDossier("plan.validation_actor_ref", err)
	}
	projectRef, err := goal.NewProjectRef("project:intake-dossier-plan-validation")
	if err != nil {
		return invalidIntakeDossier("plan.validation_project_ref", err)
	}
	const validationProcessSlots int64 = 1
	defaultDemand := governance.ResourceVector{ProcessSlots: validationProcessSlots}
	baseScope := workItemCompileScope{
		goalRef: goalRef, actorRef: actorRef, projectRef: projectRef,
		createdAt:     time.Unix(1, 0).UTC(),
		defaultDemand: defaultDemand,
	}
	resolver := workItemRefResolver{
		requestLocal: refs, parentUnknown: ErrPlanParentUnknown,
	}
	items := make([]goal.WorkItem, 0, len(plan.WorkItems))
	for index, item := range plan.WorkItems {
		scope := baseScope
		// Dossier validation is structural, not deployment policy. Giving the
		// declared demand an equal neutral envelope lets compileWorkItemSpec
		// apply its canonical demand/ref/metadata validation without imposing
		// one runtime composition's configurable budget ceiling.
		scope.goalLimit = effectiveDemand(item.BudgetDemand, defaultDemand).Resources
		compiled, compileErr := compileWorkItemSpec(
			item, refs[item.Key], scope, resolver,
		)
		if compileErr != nil {
			return invalidIntakeDossier(
				indexedDossierField("plan.work_items", index), compileErr,
			)
		}
		items = append(items, compiled)
	}
	if _, err := goal.NewPlan(goal.PlanInput{
		Generation: 1, Phases: phases, WorkItems: items,
	}); err != nil {
		return invalidIntakeDossier("plan", err)
	}
	return nil
}

func validIntakeDossierDiagramPurpose(purpose IntakeDossierDiagramPurpose) bool {
	switch purpose {
	case IntakeDossierDiagramArchitecture, IntakeDossierDiagramUserFlow,
		IntakeDossierDiagramDataIntegrations, IntakeDossierDiagramI18N,
		IntakeDossierDiagramDeployment, IntakeDossierDiagramDecisions:
		return true
	default:
		return false
	}
}

func validIntakeDossierDiagramKind(kind IntakeDossierDiagramKind) bool {
	switch kind {
	case IntakeDossierDiagramMermaid, IntakeDossierDiagramBlueprint,
		IntakeDossierDiagramInfographicBrief:
		return true
	default:
		return false
	}
}

func validIntakeDossierRef(value, prefix string) bool {
	if len(value) <= len(prefix) || len(value) > 512 ||
		!strings.HasPrefix(value, prefix) || strings.TrimSpace(value) != value {
		return false
	}
	for _, char := range value {
		if unicode.IsControl(char) || unicode.IsSpace(char) {
			return false
		}
	}
	return true
}

func validIntakeDossierMessageKey(key intake.MessageKey) bool {
	value := string(key)
	if !strings.Contains(value, ".") || value == "" || len(value) > 256 ||
		strings.TrimSpace(value) != value || strings.HasPrefix(value, ".") ||
		strings.HasSuffix(value, ".") || strings.Contains(value, "..") {
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

func hashIntakeDossier(dossier IntakeDossier) string {
	digest := fingerprintDigest(
		IntakeDossierSchema,
		dossier.actorRef.String(), dossier.projectRef.String(),
		string(dossier.stateRef), strconv.FormatUint(uint64(dossier.stateRevision), 10),
		dossier.stateDigest, dossier.sourceIntakeReceiptRef,
		dossier.statement, dossier.objective, dossier.planDigest,
	)
	writeFingerprintField(digest, strconv.Itoa(len(dossier.sections)))
	for _, section := range dossier.sections {
		writeFingerprintField(digest, string(section.Ref))
		writeFingerprintField(digest, string(section.Kind))
		writeFingerprintField(digest, string(section.TitleKey))
		writeFingerprintField(digest, section.Markdown)
	}
	writeFingerprintField(digest, strconv.Itoa(len(dossier.diagrams)))
	for _, diagram := range dossier.diagrams {
		writeFingerprintField(digest, string(diagram.Ref))
		writeFingerprintField(digest, string(diagram.Purpose))
		writeFingerprintField(digest, string(diagram.Kind))
		writeFingerprintField(digest, diagram.Source)
		writeFingerprintField(digest, string(diagram.AltTextKey))
	}
	writeFingerprintField(digest, strconv.Itoa(len(dossier.decisions)))
	for _, decision := range dossier.decisions {
		writeFingerprintField(digest, string(decision.QuestionRef))
		writeFingerprintField(digest, string(decision.Choice))
		writeFingerprintField(digest, string(decision.Recommendation))
		writeFingerprintField(digest, string(decision.RecommendationRationale))
		writeFingerprintField(digest, string(decision.Origin))
		writeFingerprintField(digest, strconv.FormatUint(uint64(decision.Revision), 10))
	}
	writeFingerprintField(digest, strconv.Itoa(len(dossier.riskRefs)))
	for _, ref := range dossier.riskRefs {
		writeFingerprintField(digest, string(ref))
	}
	return fingerprintHex(digest)
}

func cloneIntakeDossierSections(values []IntakeDossierSection) []IntakeDossierSection {
	return append([]IntakeDossierSection(nil), values...)
}

func cloneIntakeDossierDiagrams(values []IntakeDossierDiagram) []IntakeDossierDiagram {
	return append([]IntakeDossierDiagram(nil), values...)
}

func cloneIntakeDossierPlan(plan PlanSpec) PlanSpec {
	cloned := PlanSpec{
		Phases:    append([]PhaseSpec(nil), plan.Phases...),
		WorkItems: append([]WorkItemSpec(nil), plan.WorkItems...),
	}
	for index := range cloned.Phases {
		cloned.Phases[index].InputRefs = append([]string(nil), plan.Phases[index].InputRefs...)
		cloned.Phases[index].CriterionRefs = append([]string(nil), plan.Phases[index].CriterionRefs...)
	}
	for index := range cloned.WorkItems {
		source := plan.WorkItems[index]
		cloned.WorkItems[index].Dependencies = append([]string(nil), source.Dependencies...)
		cloned.WorkItems[index].WriteSet = append([]string(nil), source.WriteSet...)
		cloned.WorkItems[index].RequiredTests = append([]RequiredTestSpec(nil), source.RequiredTests...)
		for testIndex := range cloned.WorkItems[index].RequiredTests {
			cloned.WorkItems[index].RequiredTests[testIndex].Arguments = append(
				[]string(nil), source.RequiredTests[testIndex].Arguments...,
			)
		}
		cloned.WorkItems[index].SkillRefs = append([]string(nil), source.SkillRefs...)
		cloned.WorkItems[index].ToolRefs = append([]string(nil), source.ToolRefs...)
		cloned.WorkItems[index].CapabilityRefs = append([]string(nil), source.CapabilityRefs...)
	}
	return cloned
}

func indexedDossierField(field string, index int) string {
	return field + "[" + strconv.Itoa(index) + "]"
}

func invalidIntakeDossier(field string, cause error) error {
	if cause == nil {
		return fmt.Errorf("%w: %s", ErrIntakeDossierInvalid, field)
	}
	return fmt.Errorf("%w: %s: %v", ErrIntakeDossierInvalid, field, cause)
}
