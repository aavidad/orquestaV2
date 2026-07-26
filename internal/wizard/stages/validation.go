package stages

import (
	"sort"
	"strings"
)

var requiredTemplateValues = []string{
	"template:build_app",
	"template:change_app",
	"template:deploy",
	"template:domain_production",
	"template:research",
	"template:self_change",
}

func validRef(prefix, value string) (string, error) {
	if len(value) <= len(prefix)+1 || !strings.HasPrefix(value, prefix+":") {
		return "", domainError(ErrorInvalidRef, prefix+"_ref")
	}
	if !validMachineValue(value[len(prefix)+1:]) {
		return "", domainError(ErrorInvalidRef, prefix+"_ref")
	}
	return value, nil
}

func validMachineValue(value string) bool {
	if value == "" || len(value) > 160 {
		return false
	}
	runes := []rune(value)
	if !machineAlphaNumeric(runes[0]) ||
		!machineAlphaNumeric(runes[len(runes)-1]) {
		return false
	}
	previousDot := false
	for _, char := range runes {
		switch {
		case machineAlphaNumeric(char):
		case char == '_', char == '-', char == '.':
		default:
			return false
		}
		if char == '.' && previousDot {
			return false
		}
		previousDot = char == '.'
	}
	return true
}

func machineAlphaNumeric(char rune) bool {
	return char >= 'a' && char <= 'z' || char >= '0' && char <= '9'
}

func validateStage(value Stage, field string) error {
	if value.ref.value == "" {
		return domainError(ErrorInvalidRef, field+".ref")
	}
	if value.sequence <= 0 {
		return domainError(ErrorInvalidArgument, field+".sequence")
	}
	if !validPhase(value.phase) {
		return domainError(ErrorInvalidArgument, field+".phase")
	}
	if hasDuplicateRef(value.dependsOn, func(ref StageRef) string { return ref.value }) {
		return domainError(ErrorDuplicateRef, field+".depends_on")
	}
	for _, dependency := range value.dependsOn {
		if dependency.value == "" {
			return domainError(ErrorInvalidRef, field+".depends_on")
		}
		if dependency == value.ref {
			return domainError(ErrorDependencyCycle, field+".depends_on")
		}
	}
	return nil
}

func validateUnit(value Unit, field string) error {
	if value.ref.value == "" {
		return domainError(ErrorInvalidRef, field+".ref")
	}
	if value.stageRef.value == "" {
		return domainError(ErrorInvalidRef, field+".stage_ref")
	}
	if value.role.value == "" {
		return domainError(ErrorInvalidRef, field+".role")
	}
	if hasDuplicateRef(value.dependsOn, func(ref UnitRef) string { return ref.value }) {
		return domainError(ErrorDuplicateRef, field+".depends_on")
	}
	for _, dependency := range value.dependsOn {
		if dependency.value == "" {
			return domainError(ErrorInvalidRef, field+".depends_on")
		}
		if dependency == value.ref {
			return domainError(ErrorDependencyCycle, field+".depends_on")
		}
	}
	if len(value.writeSet) == 0 {
		return domainError(ErrorInvalidArgument, field+".write_set")
	}
	seenScopes := make(map[string]struct{}, len(value.writeSet))
	for _, scope := range value.writeSet {
		if !validWritePath(scope.Path) {
			return domainError(ErrorInvalidArgument, field+".write_set")
		}
		if _, exists := seenScopes[scope.Path]; exists {
			return domainError(ErrorDuplicateRef, field+".write_set")
		}
		seenScopes[scope.Path] = struct{}{}
	}
	if len(value.requiredTests) == 0 {
		return domainError(ErrorInvalidArgument, field+".required_tests")
	}
	seenTests := make(map[string]struct{}, len(value.requiredTests))
	for _, test := range value.requiredTests {
		if test.Ref.value == "" {
			return domainError(ErrorInvalidRef, field+".required_tests.ref")
		}
		if !validTestKind(test.Kind) {
			return domainError(ErrorInvalidArgument, field+".required_tests.kind")
		}
		if len(test.CriterionRefs) == 0 {
			return domainError(ErrorInvalidArgument, field+".required_tests.criteria")
		}
		if hasDuplicateRef(test.CriterionRefs, func(ref CriterionRef) string { return ref.value }) {
			return domainError(ErrorDuplicateRef, field+".required_tests.criteria")
		}
		if _, exists := seenTests[test.Ref.value]; exists {
			return domainError(ErrorDuplicateRef, field+".required_tests")
		}
		seenTests[test.Ref.value] = struct{}{}
	}
	if len(value.acceptanceCriteria) == 0 {
		return domainError(ErrorInvalidArgument, field+".acceptance_criteria")
	}
	criteria := make(map[string]struct{}, len(value.acceptanceCriteria))
	for _, criterion := range value.acceptanceCriteria {
		if criterion.Ref.value == "" {
			return domainError(ErrorInvalidRef, field+".acceptance_criteria.ref")
		}
		if _, exists := criteria[criterion.Ref.value]; exists {
			return domainError(ErrorDuplicateRef, field+".acceptance_criteria")
		}
		criteria[criterion.Ref.value] = struct{}{}
	}
	for _, test := range value.requiredTests {
		for _, criterionRef := range test.CriterionRefs {
			if _, exists := criteria[criterionRef.value]; !exists {
				return domainError(ErrorReferenceNotFound, field+".required_tests.criteria")
			}
		}
	}
	if len(value.effects) == 0 {
		return domainError(ErrorInvalidArgument, field+".effects")
	}
	seenEffects := make(map[string]struct{}, len(value.effects))
	for _, effect := range value.effects {
		if effect.Ref.value == "" {
			return domainError(ErrorInvalidRef, field+".effects.ref")
		}
		if !validEffectKind(effect.Kind) {
			return domainError(ErrorInvalidArgument, field+".effects.kind")
		}
		if effectNeedsApproval(effect.Kind) && !effect.ApprovalRequired {
			return domainError(ErrorInvalidArgument, field+".effects.approval_required")
		}
		if _, exists := seenEffects[effect.Ref.value]; exists {
			return domainError(ErrorDuplicateRef, field+".effects")
		}
		seenEffects[effect.Ref.value] = struct{}{}
	}
	if !validExecutionPolicy(value.policy) {
		return domainError(ErrorInvalidArgument, field+".policy")
	}
	if value.policy.Security.Approval == ApprovalNone {
		for _, effect := range value.effects {
			if effect.ApprovalRequired {
				return domainError(ErrorInvalidArgument, field+".policy.security.approval")
			}
		}
	}
	for _, effect := range value.effects {
		if !approvalSupportsEffect(value.policy.Security, effect.Kind) {
			return domainError(ErrorInvalidArgument, field+".policy.security")
		}
	}
	return nil
}

func validateTemplate(value Template) error {
	if value.ref.value == "" {
		return domainError(ErrorInvalidRef, "template.ref")
	}
	if len(value.roadmapCapabilityRefs) == 0 {
		return domainError(ErrorInvalidArgument, "template.roadmap_capability_refs")
	}
	if hasDuplicateRef(
		value.roadmapCapabilityRefs,
		func(ref RoadmapCapabilityRef) string { return ref.value },
	) {
		return domainError(ErrorDuplicateRef, "template.roadmap_capability_refs")
	}
	for _, ref := range value.roadmapCapabilityRefs {
		if ref.value == "" {
			return domainError(ErrorInvalidRef, "template.roadmap_capability_refs")
		}
	}
	if len(value.stages) == 0 {
		return domainError(ErrorInvalidArgument, "template.stages")
	}
	if len(value.units) == 0 {
		return domainError(ErrorInvalidArgument, "template.units")
	}

	stagesByRef := make(map[string]Stage, len(value.stages))
	sequences := make(map[int]struct{}, len(value.stages))
	for _, stage := range value.stages {
		if err := validateStage(stage, "template.stages"); err != nil {
			return err
		}
		if _, exists := stagesByRef[stage.ref.value]; exists {
			return domainError(ErrorDuplicateRef, "template.stages")
		}
		if _, exists := sequences[stage.sequence]; exists {
			return domainError(ErrorDuplicateRef, "template.stages.sequence")
		}
		stagesByRef[stage.ref.value] = stage
		sequences[stage.sequence] = struct{}{}
	}
	for _, stage := range value.stages {
		for _, dependency := range stage.dependsOn {
			_, exists := stagesByRef[dependency.value]
			if !exists {
				return domainError(ErrorReferenceNotFound, "template.stages.depends_on")
			}
		}
	}
	if hasCycle(
		stageKeys(value.stages),
		func(ref string) []string { return stageDependencyKeys(stagesByRef[ref]) },
	) {
		return domainError(ErrorDependencyCycle, "template.stages")
	}
	for _, stage := range value.stages {
		for _, dependency := range stage.dependsOn {
			if stagesByRef[dependency.value].sequence >= stage.sequence {
				return domainError(ErrorDependencyOrder, "template.stages.depends_on")
			}
		}
	}

	unitsByRef := make(map[string]Unit, len(value.units))
	stageUnitCount := make(map[string]int, len(value.stages))
	testRefs := make(map[string]struct{})
	criterionRefs := make(map[string]struct{})
	effectRefs := make(map[string]struct{})
	for _, unit := range value.units {
		if err := validateUnit(unit, "template.units"); err != nil {
			return err
		}
		if _, exists := unitsByRef[unit.ref.value]; exists {
			return domainError(ErrorDuplicateRef, "template.units")
		}
		if _, exists := stagesByRef[unit.stageRef.value]; !exists {
			return domainError(ErrorReferenceNotFound, "template.units.stage_ref")
		}
		unitsByRef[unit.ref.value] = unit
		stageUnitCount[unit.stageRef.value]++
		for _, test := range unit.requiredTests {
			if _, exists := testRefs[test.Ref.value]; exists {
				return domainError(ErrorDuplicateRef, "template.units.required_tests")
			}
			testRefs[test.Ref.value] = struct{}{}
		}
		for _, criterion := range unit.acceptanceCriteria {
			if _, exists := criterionRefs[criterion.Ref.value]; exists {
				return domainError(ErrorDuplicateRef, "template.units.acceptance_criteria")
			}
			criterionRefs[criterion.Ref.value] = struct{}{}
		}
		for _, effect := range unit.effects {
			if _, exists := effectRefs[effect.Ref.value]; exists {
				return domainError(ErrorDuplicateRef, "template.units.effects")
			}
			effectRefs[effect.Ref.value] = struct{}{}
		}
	}
	for ref := range stagesByRef {
		if stageUnitCount[ref] == 0 {
			return domainError(ErrorInvalidArgument, "template.stages.units")
		}
	}
	for _, unit := range value.units {
		currentStage := stagesByRef[unit.stageRef.value]
		for _, dependencyRef := range unit.dependsOn {
			dependency, exists := unitsByRef[dependencyRef.value]
			if !exists {
				return domainError(ErrorReferenceNotFound, "template.units.depends_on")
			}
			dependencyStage := stagesByRef[dependency.stageRef.value]
			if dependencyStage.ref != currentStage.ref &&
				!dependsTransitively(
					currentStage.ref.value,
					dependencyStage.ref.value,
					func(ref string) []string {
						return stageDependencyKeys(stagesByRef[ref])
					},
				) {
				return domainError(ErrorDependencyOrder, "template.units.depends_on")
			}
		}
	}
	if hasCycle(
		unitKeys(value.units),
		func(ref string) []string { return unitDependencyKeys(unitsByRef[ref]) },
	) {
		return domainError(ErrorDependencyCycle, "template.units")
	}
	if err := validateConcurrentWriteSets(value, stagesByRef, unitsByRef); err != nil {
		return err
	}
	return nil
}

func validateConcurrentWriteSets(
	template Template,
	stages map[string]Stage,
	units map[string]Unit,
) error {
	for leftIndex := range template.units {
		for rightIndex := leftIndex + 1; rightIndex < len(template.units); rightIndex++ {
			left := template.units[leftIndex]
			right := template.units[rightIndex]
			if causallyOrdered(left, right, stages, units) {
				continue
			}
			for _, leftScope := range left.writeSet {
				for _, rightScope := range right.writeSet {
					if pathsOverlap(leftScope.Path, rightScope.Path) {
						return domainError(ErrorWriteSetConflict, "template.units.write_set")
					}
				}
			}
		}
	}
	return nil
}

func causallyOrdered(
	left Unit,
	right Unit,
	stages map[string]Stage,
	units map[string]Unit,
) bool {
	dependencies := func(ref string) []string {
		return unitDependencyKeys(units[ref])
	}
	if dependsTransitively(left.ref.value, right.ref.value, dependencies) ||
		dependsTransitively(right.ref.value, left.ref.value, dependencies) {
		return true
	}
	stageDependencies := func(ref string) []string {
		return stageDependencyKeys(stages[ref])
	}
	return dependsTransitively(
		left.stageRef.value,
		right.stageRef.value,
		stageDependencies,
	) || dependsTransitively(
		right.stageRef.value,
		left.stageRef.value,
		stageDependencies,
	)
}

func hasCycle(keys []string, dependencies func(string) []string) bool {
	const (
		unseen = iota
		visiting
		visited
	)
	state := make(map[string]int, len(keys))
	var visit func(string) bool
	visit = func(key string) bool {
		switch state[key] {
		case visiting:
			return true
		case visited:
			return false
		}
		state[key] = visiting
		for _, dependency := range dependencies(key) {
			if visit(dependency) {
				return true
			}
		}
		state[key] = visited
		return false
	}
	for _, key := range keys {
		if visit(key) {
			return true
		}
	}
	return false
}

func dependsTransitively(
	from string,
	target string,
	dependencies func(string) []string,
) bool {
	if from == target {
		return false
	}
	visited := make(map[string]struct{})
	stack := append([]string(nil), dependencies(from)...)
	for len(stack) > 0 {
		index := len(stack) - 1
		current := stack[index]
		stack = stack[:index]
		if current == target {
			return true
		}
		if _, seen := visited[current]; seen {
			continue
		}
		visited[current] = struct{}{}
		stack = append(stack, dependencies(current)...)
	}
	return false
}

func validWritePath(value string) bool {
	if value == "" || len(value) > 240 || strings.HasPrefix(value, "/") ||
		strings.HasSuffix(value, "/") || strings.Contains(value, "\\") {
		return false
	}
	for _, part := range strings.Split(value, "/") {
		if part == "" || part == "." || part == ".." || !validMachineValue(part) {
			return false
		}
	}
	return true
}

func pathsOverlap(left, right string) bool {
	return left == right ||
		strings.HasPrefix(left, right+"/") ||
		strings.HasPrefix(right, left+"/")
}

func validPhase(value PhaseKind) bool {
	switch value {
	case PhaseDiscover, PhaseDesign, PhaseProduce, PhaseVerify, PhaseRelease:
		return true
	default:
		return false
	}
}

func validTestKind(value TestKind) bool {
	switch value {
	case TestUnit, TestContract, TestIntegration, TestSecurity, TestReview,
		TestPostcondition:
		return true
	default:
		return false
	}
}

func validEffectKind(value EffectKind) bool {
	switch value {
	case EffectReadContext, EffectWriteArtifact, EffectMutateWorkspace,
		EffectRequestExternal, EffectMutateExternal, EffectPublish,
		EffectMutateSelf:
		return true
	default:
		return false
	}
}

func effectNeedsApproval(value EffectKind) bool {
	switch value {
	case EffectMutateExternal, EffectPublish, EffectMutateSelf:
		return true
	default:
		return false
	}
}

func approvalSupportsEffect(policy SecurityPolicy, effect EffectKind) bool {
	switch effect {
	case EffectMutateExternal:
		return policy.Approval == ApprovalBeforeMutation &&
			(policy.Risk == RiskHigh || policy.Risk == RiskCritical)
	case EffectPublish:
		return policy.Approval == ApprovalBeforePublish &&
			(policy.Risk == RiskHigh || policy.Risk == RiskCritical)
	case EffectMutateSelf:
		return policy.Approval == ApprovalIndependentReview &&
			policy.Risk == RiskCritical
	default:
		return true
	}
}

func validExecutionPolicy(value ExecutionPolicy) bool {
	switch value.Effort {
	case EffortRoutine, EffortFocused, EffortDeep:
	default:
		return false
	}
	switch value.Security.Risk {
	case RiskLow, RiskModerate, RiskHigh, RiskCritical:
	default:
		return false
	}
	switch value.Security.Approval {
	case ApprovalNone, ApprovalBeforeMutation, ApprovalBeforePublish,
		ApprovalIndependentReview:
	default:
		return false
	}
	return value.Security.LeastPrivilege
}

func hasDuplicateRef[T any](values []T, ref func(T) string) bool {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		key := ref(value)
		if _, exists := seen[key]; exists {
			return true
		}
		seen[key] = struct{}{}
	}
	return false
}

func stageKeys(values []Stage) []string {
	out := make([]string, len(values))
	for index, value := range values {
		out[index] = value.ref.value
	}
	return out
}

func unitKeys(values []Unit) []string {
	out := make([]string, len(values))
	for index, value := range values {
		out[index] = value.ref.value
	}
	return out
}

func stageDependencyKeys(value Stage) []string {
	out := make([]string, len(value.dependsOn))
	for index, ref := range value.dependsOn {
		out[index] = ref.value
	}
	return out
}

func unitDependencyKeys(value Unit) []string {
	out := make([]string, len(value.dependsOn))
	for index, ref := range value.dependsOn {
		out[index] = ref.value
	}
	return out
}

func cloneStageRefs(values []StageRef) []StageRef {
	return append([]StageRef(nil), values...)
}

func cloneUnitRefs(values []UnitRef) []UnitRef {
	return append([]UnitRef(nil), values...)
}

func cloneCriterionRefs(values []CriterionRef) []CriterionRef {
	return append([]CriterionRef(nil), values...)
}

func cloneWriteScopes(values []WriteScope) []WriteScope {
	return append([]WriteScope(nil), values...)
}

func cloneRequiredTests(values []RequiredTest) []RequiredTest {
	out := append([]RequiredTest(nil), values...)
	for index := range out {
		out[index].CriterionRefs = cloneCriterionRefs(out[index].CriterionRefs)
	}
	return out
}

func cloneCriteria(values []Criterion) []Criterion {
	return append([]Criterion(nil), values...)
}

func cloneEffects(values []Effect) []Effect {
	return append([]Effect(nil), values...)
}

func cloneRoadmapCapabilityRefs(
	values []RoadmapCapabilityRef,
) []RoadmapCapabilityRef {
	return append([]RoadmapCapabilityRef(nil), values...)
}

func cloneStages(values []Stage) []Stage {
	out := append([]Stage(nil), values...)
	for index := range out {
		out[index].dependsOn = cloneStageRefs(out[index].dependsOn)
	}
	return out
}

func cloneUnits(values []Unit) []Unit {
	out := append([]Unit(nil), values...)
	for index := range out {
		out[index].dependsOn = cloneUnitRefs(out[index].dependsOn)
		out[index].writeSet = cloneWriteScopes(out[index].writeSet)
		out[index].requiredTests = cloneRequiredTests(out[index].requiredTests)
		out[index].acceptanceCriteria = cloneCriteria(out[index].acceptanceCriteria)
		out[index].effects = cloneEffects(out[index].effects)
	}
	return out
}

func cloneTemplates(values []Template) []Template {
	out := append([]Template(nil), values...)
	for index := range out {
		out[index].roadmapCapabilityRefs = cloneRoadmapCapabilityRefs(
			out[index].roadmapCapabilityRefs,
		)
		out[index].stages = cloneStages(out[index].stages)
		out[index].units = cloneUnits(out[index].units)
	}
	return out
}

func sortStageRefs(values []StageRef) {
	sort.Slice(values, func(left, right int) bool {
		return values[left].value < values[right].value
	})
}

func sortUnitRefs(values []UnitRef) {
	sort.Slice(values, func(left, right int) bool {
		return values[left].value < values[right].value
	})
}

func sortWriteScopes(values []WriteScope) {
	sort.Slice(values, func(left, right int) bool {
		return values[left].Path < values[right].Path
	})
}

func sortRequiredTests(values []RequiredTest) {
	for index := range values {
		sort.Slice(values[index].CriterionRefs, func(left, right int) bool {
			return values[index].CriterionRefs[left].value <
				values[index].CriterionRefs[right].value
		})
	}
	sort.Slice(values, func(left, right int) bool {
		return values[left].Ref.value < values[right].Ref.value
	})
}

func sortCriteria(values []Criterion) {
	sort.Slice(values, func(left, right int) bool {
		return values[left].Ref.value < values[right].Ref.value
	})
}

func sortEffects(values []Effect) {
	sort.Slice(values, func(left, right int) bool {
		return values[left].Ref.value < values[right].Ref.value
	})
}

func sortRoadmapCapabilityRefs(values []RoadmapCapabilityRef) {
	sort.Slice(values, func(left, right int) bool {
		return values[left].value < values[right].value
	})
}

func sortStages(values []Stage) {
	sort.Slice(values, func(left, right int) bool {
		if values[left].sequence != values[right].sequence {
			return values[left].sequence < values[right].sequence
		}
		return values[left].ref.value < values[right].ref.value
	})
}

func sortUnits(values []Unit, stages []Stage) {
	sequence := make(map[string]int, len(stages))
	for _, stage := range stages {
		sequence[stage.ref.value] = stage.sequence
	}
	sort.Slice(values, func(left, right int) bool {
		leftSequence := sequence[values[left].stageRef.value]
		rightSequence := sequence[values[right].stageRef.value]
		if leftSequence != rightSequence {
			return leftSequence < rightSequence
		}
		return values[left].ref.value < values[right].ref.value
	})
}

func sortTemplates(values []Template) {
	sort.Slice(values, func(left, right int) bool {
		return values[left].ref.value < values[right].ref.value
	})
}
