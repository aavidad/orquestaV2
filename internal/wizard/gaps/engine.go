package gaps

import (
	"strings"

	"orquesta/internal/wizard/catalog"
)

type selectionIndex struct {
	resolved  map[DimensionRef]Selection
	conflicts map[DimensionRef]bool
}

type questionSelectionIndex struct {
	resolved  map[QuestionRef]QuestionSelection
	conflicts map[QuestionRef]bool
}

type evaluation struct {
	dimensions         []DimensionDescriptor
	byDimension        map[DimensionRef]DimensionDescriptor
	selections         selectionIndex
	questionSelections questionSelectionIndex
	facts              Facts
	explicitPackRefs   []catalog.PackRef
	issues             []Issue
	issueRefs          map[IssueRef]struct{}
	causes             map[DimensionRef][]IssueRef
	dependencies       map[DimensionRef][]DimensionRef
	open               map[DimensionRef]bool
	recommendation     map[DimensionRef]OptionRef
	extraQuestions     []Question
	packQuestions      []Question
}

// Evaluate is pure and deterministic. Input order, repeated equal selections
// and repeated pack refs do not change output.
func Evaluate(input Input) (Result, error) {
	return EvaluateV1(input)
}

// EvaluateV1 is retained as the historical V1 implementation. Current aliases
// may advance only by adding a new evaluator; V1 replay keeps calling this one.
func EvaluateV1(input Input) (Result, error) {
	inventory := BuiltInV1()
	dimensions := inventory.Dimensions()
	byDimension := dimensionsByRef(dimensions)
	if err := validateFacts(input.Facts); err != nil {
		return Result{}, err
	}
	packRefs, err := validatePackRefs(input.PackRefs)
	if err != nil {
		return Result{}, err
	}
	selections, err := indexSelections(input.Selections, byDimension)
	if err != nil {
		return Result{}, err
	}
	questionSelections, err := indexQuestionSelections(
		input.QuestionSelections,
		supplementalQuestionCatalog(packRefs),
	)
	if err != nil {
		return Result{}, err
	}
	value := evaluation{
		dimensions:         dimensions,
		byDimension:        byDimension,
		selections:         selections,
		questionSelections: questionSelections,
		facts:              input.Facts,
		explicitPackRefs:   packRefs,
		issueRefs:          make(map[IssueRef]struct{}),
		causes:             make(map[DimensionRef][]IssueRef),
		dependencies:       make(map[DimensionRef][]DimensionRef),
		open:               make(map[DimensionRef]bool),
		recommendation:     make(map[DimensionRef]OptionRef),
	}
	value.detectDimensionIssues()
	if err := value.detectCrossRuleIssues(); err != nil {
		return Result{}, err
	}
	questions := value.buildQuestions()
	defaults := value.buildDefaultProposals()
	return Result{
		schemaVersion: SchemaVersion,
		issues:        cloneIssues(value.issues),
		questions:     cloneQuestions(questions),
		defaults:      append([]DefaultProposal(nil), defaults...),
		packRefs:      append([]catalog.PackRef(nil), packRefs...),
	}, nil
}

func (value *evaluation) detectDimensionIssues() {
	for _, dimension := range value.dimensions {
		if value.selections.conflicts[dimension.ref] {
			value.addDimensionIssue(dimension, IssueContradiction)
			continue
		}
		if _, selected := value.selections.resolved[dimension.ref]; selected {
			continue
		}
		if dimension.layer == LayerTechnical && !value.technicalActive(dimension.ref) {
			continue
		}
		value.addDimensionIssue(dimension, IssueGap)
	}
}

func (value *evaluation) addDimensionIssue(
	dimension DimensionDescriptor,
	kind IssueKind,
) {
	ref := dimensionIssueRef(dimension.ref)
	detailSuffix := ".gap"
	if kind == IssueContradiction {
		detailSuffix = ".contradiction"
	}
	value.addIssue(Issue{
		ref: ref, kind: kind, dimension: dimension.ref, field: dimension.slot,
		detailKey: MessageKey(
			"wizard.gaps.dimension." + lowerRef(dimension.ref) + ".issue" + detailSuffix,
		),
		dependsOn: append([]DimensionRef(nil), dimension.dependsOn...),
	})
	value.openDimension(dimension.ref, ref)
}

func (value *evaluation) technicalActive(ref DimensionRef) bool {
	switch ref {
	case DimensionT1:
		return value.audienceShared()
	case DimensionT2:
		return value.facts.CorporateIdentity == DeclarationDeclared
	case DimensionT3:
		return value.deploymentIs("own_server", "cloud_container") ||
			value.facts.Surface == SurfaceKernelModule
	case DimensionT4:
		return value.selectionIs(
			DimensionU4, "persisted_user_data", "external_source",
		)
	case DimensionT5:
		return value.integrationSelected() || len(value.explicitPackRefs) > 0
	case DimensionT6:
		return value.deploymentIs("own_server", "cloud_container") ||
			value.facts.Surface == SurfaceKernelModule
	case DimensionT7:
		return value.integrationSelected() ||
			value.selectionIs(DimensionU11, "realtime", "high_concurrency")
	case DimensionT8:
		return value.selectionIs(
			DimensionU5, "minimized_personal", "sensitive", "regulated",
		)
	default:
		return false
	}
}

func (value *evaluation) buildQuestions() []Question {
	result := make([]Question, 0, len(value.open)+len(value.extraQuestions)+len(value.packQuestions))
	for _, dimension := range value.dimensions {
		if !value.open[dimension.ref] {
			continue
		}
		options := recommendedOptions(
			dimension.options,
			value.recommendedOption(dimension),
		)
		dimensionDependencies := value.orderedDependencies(
			value.dependencies[dimension.ref],
		)
		dependsOn := make([]QuestionRef, len(dimensionDependencies))
		for index, dependency := range dimensionDependencies {
			dependsOn[index] = questionRef(dependency)
		}
		result = append(result, Question{
			ref: questionRef(dimension.ref), dimension: dimension.ref,
			slot: dimension.slot, decisionKind: dimension.decisionKind,
			derivedFrom: append([]IssueRef(nil), value.causes[dimension.ref]...),
			dependsOn:   dependsOn,
			promptKey:   dimension.promptKey,
			whyKey:      dimension.whyKey,
			helpKey:     dimension.helpKey,
			exampleKey:  dimension.exampleKey,
			options:     options,
		})
	}
	result = append(result, cloneQuestions(value.packQuestions)...)
	result = append(result, cloneQuestions(value.extraQuestions)...)
	return result
}

func (value *evaluation) orderedDependencies(
	dependencies []DimensionRef,
) []DimensionRef {
	selected := make(map[DimensionRef]struct{}, len(dependencies))
	for _, dependency := range dependencies {
		selected[dependency] = struct{}{}
	}
	result := make([]DimensionRef, 0, len(selected))
	for _, dimension := range value.dimensions {
		if _, found := selected[dimension.ref]; found {
			result = append(result, dimension.ref)
		}
	}
	return result
}

func (value *evaluation) buildDefaultProposals() []DefaultProposal {
	result := make([]DefaultProposal, 0, 8)
	for _, dimension := range value.dimensions {
		if dimension.layer != LayerTechnical || value.open[dimension.ref] {
			continue
		}
		if _, selected := value.selections.resolved[dimension.ref]; selected {
			continue
		}
		if value.selections.conflicts[dimension.ref] ||
			value.technicalActive(dimension.ref) {
			continue
		}
		recommended := value.recommendedOption(dimension)
		option, found := findOption(dimension.options, recommended)
		if !found {
			panic("wizard gaps built-in technical default missing")
		}
		prefix := MessageKey("wizard.gaps.default." + lowerRef(dimension.ref))
		result = append(result, DefaultProposal{
			dimension: dimension.ref, decisionKind: DecisionTechnicalDefault,
			option:   recommended,
			labelKey: prefix + ".label", helpKey: prefix + ".help",
			exampleKey: prefix + ".example", rationaleKey: option.rationaleKey,
		})
	}
	return result
}

func (value *evaluation) recommendedOption(
	dimension DimensionDescriptor,
) OptionRef {
	if selected, overridden := value.recommendation[dimension.ref]; overridden {
		return selected
	}
	switch dimension.ref {
	case DimensionU3:
		if value.selectionIs(DimensionU1, "personal") {
			return optionRef(DimensionU3, "no_login")
		}
	case DimensionU6:
		if value.selectionIs(DimensionU1, "personal") {
			return optionRef(DimensionU6, "none")
		}
	case DimensionU10:
		return value.compatibleDeploymentRecommendation()
	case DimensionT3:
		if value.facts.Surface == SurfaceKernelModule {
			return optionRef(DimensionT3, "kernel_trace")
		}
	case DimensionT4:
		if value.audienceShared() ||
			value.deploymentIs("own_server", "cloud_container") {
			return optionRef(DimensionT4, "postgres_restore")
		}
	case DimensionT6:
		if value.facts.Surface == SurfaceKernelModule {
			return optionRef(DimensionT6, "kernel_build_ci")
		}
	}
	return dimension.defaultRefOrRecommended()
}

func (value DimensionDescriptor) defaultRefOrRecommended() OptionRef {
	if value.defaultRef != "" {
		return value.defaultRef
	}
	for _, option := range value.options {
		if option.recommended {
			return option.ref
		}
	}
	panic("wizard gaps built-in recommendation missing")
}

func (value *evaluation) selectionIs(
	dimension DimensionRef,
	options ...string,
) bool {
	selection, found := value.selections.resolved[dimension]
	if !found {
		return false
	}
	for _, option := range options {
		if selection.Option == optionRef(dimension, option) {
			return true
		}
	}
	return false
}

func (value *evaluation) audienceShared() bool {
	return value.selectionIs(DimensionU1, "team", "public") ||
		(value.facts.SharingIntent == SharingShared &&
			!value.selectionIs(DimensionU1, "personal"))
}

func (value *evaluation) deploymentIs(options ...string) bool {
	return value.selectionIs(DimensionU10, options...)
}

func (value *evaluation) integrationSelected() bool {
	return value.selectionIs(DimensionU7, "external_services", "public_api", "custom")
}

func (value *evaluation) deploymentCompatible(option OptionRef) bool {
	if option == optionRef(DimensionU10, "custom") {
		return true
	}
	switch value.facts.Surface {
	case SurfaceNativeMobile:
		return option == optionRef(DimensionU10, "distribution_store") ||
			option == optionRef(DimensionU10, "local_device")
	case SurfaceNativeDesktop, SurfaceKernelModule:
		return option == optionRef(DimensionU10, "local_device")
	case SurfaceServerService:
		return option == optionRef(DimensionU10, "own_server") ||
			option == optionRef(DimensionU10, "cloud_container")
	default:
		return true
	}
}

func (value *evaluation) compatibleDeploymentRecommendation() OptionRef {
	switch value.facts.Surface {
	case SurfaceNativeMobile:
		return optionRef(DimensionU10, "distribution_store")
	case SurfaceNativeDesktop, SurfaceKernelModule:
		return optionRef(DimensionU10, "local_device")
	case SurfaceServerService:
		return optionRef(DimensionU10, "cloud_container")
	}
	if value.selectionIs(DimensionU1, "personal") ||
		value.facts.SharingIntent == SharingPersonal {
		return optionRef(DimensionU10, "local_device")
	}
	return optionRef(DimensionU10, "cloud_container")
}

func ruleQuestion(
	ref RuleRef,
	slot SlotKey,
	kind DecisionKind,
	recommended string,
	values ...string,
) Question {
	prefix := "wizard.gaps.rule." + lowerRuleRef(ref)
	options := make([]Option, 0, len(values)+1)
	for _, value := range values {
		optionPrefix := prefix + ".option." + value
		options = append(options, Option{
			ref:          OptionRef("intake-option:wizard." + lowerRuleRef(ref) + "." + value),
			kind:         OptionPreset,
			labelKey:     MessageKey(optionPrefix + ".label"),
			helpKey:      MessageKey(optionPrefix + ".help"),
			exampleKey:   MessageKey(optionPrefix + ".example"),
			rationaleKey: MessageKey(optionPrefix + ".rationale"),
			recommended:  value == recommended,
		})
	}
	customPrefix := prefix + ".option.custom"
	options = append(options, Option{
		ref:          OptionRef("intake-option:wizard." + lowerRuleRef(ref) + ".custom"),
		kind:         OptionFreeText,
		labelKey:     MessageKey(customPrefix + ".label"),
		helpKey:      MessageKey(customPrefix + ".help"),
		exampleKey:   MessageKey(customPrefix + ".example"),
		rationaleKey: MessageKey(customPrefix + ".rationale"),
	})
	return Question{
		ref:          QuestionRef("intake-question:wizard." + lowerRuleRef(ref)),
		slot:         slot,
		decisionKind: kind,
		promptKey:    MessageKey(prefix + ".prompt"),
		whyKey:       MessageKey(prefix + ".why"),
		helpKey:      MessageKey(prefix + ".help"),
		exampleKey:   MessageKey(prefix + ".example"),
		options:      options,
	}
}

func packQuestion(packRef catalog.PackRef, source catalog.Question) Question {
	sourceOptions := source.Options()
	options := make([]Option, 0, len(sourceOptions)+1)
	for _, item := range sourceOptions {
		options = append(options, Option{
			ref: OptionRef(
				"intake-option:wizard.catalog." +
					strings.TrimPrefix(item.Ref().String(), "option:"),
			),
			kind:         OptionPreset,
			labelKey:     MessageKey(item.LabelKey().String()),
			helpKey:      MessageKey(item.HelpKey().String()),
			exampleKey:   MessageKey(item.ExampleKey().String()),
			rationaleKey: MessageKey(item.RationaleKey().String()),
			recommended:  item.Recommended(),
		})
	}
	customPrefix := "wizard.gaps.pack.option.custom"
	options = append(options, Option{
		ref: OptionRef(
			"intake-option:wizard.catalog." +
				strings.TrimPrefix(source.Ref().String(), "question:") + ".custom",
		),
		kind:         OptionFreeText,
		labelKey:     MessageKey(customPrefix + ".label"),
		helpKey:      MessageKey(customPrefix + ".help"),
		exampleKey:   MessageKey(customPrefix + ".example"),
		rationaleKey: MessageKey(customPrefix + ".rationale"),
	})
	return Question{
		ref: QuestionRef(
			"intake-question:wizard.catalog." +
				strings.TrimPrefix(source.Ref().String(), "question:"),
		),
		packRef: packRef, slot: SlotKey(source.Slot().String()),
		decisionKind: DecisionProduct,
		promptKey:    MessageKey(source.PromptKey().String()),
		whyKey:       MessageKey(source.WhyKey().String()),
		helpKey:      MessageKey(source.HelpKey().String()),
		exampleKey:   MessageKey(source.ExampleKey().String()),
		options:      options,
	}
}

func recommendedOptions(values []Option, recommended OptionRef) []Option {
	result := append([]Option(nil), values...)
	found := false
	for index := range result {
		result[index].recommended = result[index].ref == recommended
		found = found || result[index].recommended
	}
	if !found {
		panic("wizard gaps recommended option absent")
	}
	return result
}

func findOption(values []Option, ref OptionRef) (Option, bool) {
	for _, value := range values {
		if value.ref == ref {
			return value, true
		}
	}
	return Option{}, false
}

func uniqueDimensionsExcluding(
	values []DimensionRef,
	excluded DimensionRef,
) []DimensionRef {
	seen := make(map[DimensionRef]struct{}, len(values))
	result := make([]DimensionRef, 0, len(values))
	for _, value := range values {
		if value == "" || value == excluded {
			continue
		}
		if _, duplicate := seen[value]; duplicate {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func cloneIssues(values []Issue) []Issue {
	out := append([]Issue(nil), values...)
	for index := range out {
		out[index].dependsOn = append([]DimensionRef(nil), out[index].dependsOn...)
	}
	return out
}

func cloneQuestions(values []Question) []Question {
	out := append([]Question(nil), values...)
	for index := range out {
		out[index].derivedFrom = append([]IssueRef(nil), out[index].derivedFrom...)
		out[index].dependsOn = append([]QuestionRef(nil), out[index].dependsOn...)
		out[index].options = append([]Option(nil), out[index].options...)
	}
	return out
}
