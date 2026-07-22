package goal

import (
	"math"
	"path"
	"strings"

	"orquesta/internal/governance"
)

// PlanGeneration is the optimistic version of the plan embedded in a Goal.
// Zero means that no plan has been applied yet.
type PlanGeneration uint64

// PhaseKey identifies immutable phase metadata inside one plan generation.
type PhaseKey struct{ value string }

// RoleKey identifies the role requested from an execution. It is policy-free
// metadata: providers and directors decide how to satisfy it outside domain.
type RoleKey struct{ value string }

// WriteScope is a clean repository-relative path. Globs are intentionally not
// part of domain; adapters may expand user syntax before proposing a plan.
type WriteScope struct{ value string }

func NewPhaseKey(value string) (PhaseKey, error) {
	value, err := validPlanKey("phase_key", value)
	return PhaseKey{value: value}, err
}

func NewRoleKey(value string) (RoleKey, error) {
	value, err := validPlanKey("role_key", value)
	return RoleKey{value: value}, err
}

func NewWriteScope(value string) (WriteScope, error) {
	value, err := validWriteScope(value)
	return WriteScope{value: value}, err
}

func (key PhaseKey) String() string     { return key.value }
func (key RoleKey) String() string      { return key.value }
func (scope WriteScope) String() string { return scope.value }

func defaultPhaseKey() PhaseKey { return PhaseKey{value: "phase:default"} }
func defaultRoleKey() RoleKey   { return RoleKey{value: "role:worker"} }

// DefaultPhaseKey and DefaultRoleKey make the minimal single-item plan
// explicit without introducing product policy.
func DefaultPhaseKey() PhaseKey { return defaultPhaseKey() }
func DefaultRoleKey() RoleKey   { return defaultRoleKey() }

func validPhaseKey(key PhaseKey) bool { return key.value != "" }
func validRoleKey(key RoleKey) bool   { return key.value != "" }
func validWriteScopeValue(scope WriteScope) bool {
	return scope.value != ""
}

// PhaseInstanceInput identifies one immutable use of a phase template. Inputs
// and criteria are opaque references; their schemas and evaluation live behind
// ports rather than in Goal lifecycle.
type PhaseInstanceInput struct {
	Ref           PhaseRef
	Key           PhaseKey
	TemplateRef   PhaseTemplateRef
	InputRefs     []InputRef
	CriterionRefs []CriterionRef
}

// PhaseInstance is immutable plan metadata. It intentionally has no state or
// lifecycle; phase progress is derived from its WorkItems and evidence.
type PhaseInstance struct {
	ref           PhaseRef
	key           PhaseKey
	templateRef   PhaseTemplateRef
	inputRefs     []InputRef
	criterionRefs []CriterionRef
}

// NewPhaseInstance preserves the minimal V04 constructor. Its deterministic
// refs make the default explicit without adding phase policy.
func NewPhaseInstance(key PhaseKey) (PhaseInstance, error) {
	if !validPhaseKey(key) {
		return PhaseInstance{}, domainError(ErrorInvalidPlan, "phase_key")
	}
	return NewPhaseInstanceWithMetadata(PhaseInstanceInput{
		Ref:         PhaseRef{value: "phase-instance:" + key.value},
		Key:         key,
		TemplateRef: PhaseTemplateRef{value: "phase-template:" + key.value},
	})
}

func NewPhaseInstanceWithMetadata(input PhaseInstanceInput) (PhaseInstance, error) {
	if !validPhaseRef(input.Ref) {
		return PhaseInstance{}, domainError(ErrorInvalidRef, "phase_ref")
	}
	if !validPhaseKey(input.Key) {
		return PhaseInstance{}, domainError(ErrorInvalidPlan, "phase_key")
	}
	if !validPhaseTemplateRef(input.TemplateRef) {
		return PhaseInstance{}, domainError(ErrorInvalidRef, "phase_template_ref")
	}
	if err := validateUniqueRefs(input.InputRefs, validInputRef, "input_ref"); err != nil {
		return PhaseInstance{}, err
	}
	if err := validateUniqueRefs(input.CriterionRefs, validCriterionRef, "criterion_ref"); err != nil {
		return PhaseInstance{}, err
	}
	return PhaseInstance{
		ref: input.Ref, key: input.Key, templateRef: input.TemplateRef,
		inputRefs: cloneRefs(input.InputRefs), criterionRefs: cloneRefs(input.CriterionRefs),
	}, nil
}

func (phase PhaseInstance) Ref() PhaseRef                 { return phase.ref }
func (phase PhaseInstance) Key() PhaseKey                 { return phase.key }
func (phase PhaseInstance) TemplateRef() PhaseTemplateRef { return phase.templateRef }
func (phase PhaseInstance) InputRefs() []InputRef         { return cloneRefs(phase.inputRefs) }
func (phase PhaseInstance) CriterionRefs() []CriterionRef { return cloneRefs(phase.criterionRefs) }

type OutputContractKind string

const (
	OutputContractEvidenceBundle OutputContractKind = "evidence_bundle"
	OutputContractArtifact       OutputContractKind = "artifact"
	OutputContractAttestation    OutputContractKind = "attestation"
)

// OutputContract is the smallest typed evidence contract useful to domain.
// Detailed schemas, validators and provider formats belong behind ports.
type OutputContract struct {
	kind OutputContractKind
}

func NewOutputContract(kind OutputContractKind) (OutputContract, error) {
	if !validOutputContractKind(kind) {
		return OutputContract{}, domainError(ErrorInvalidPlan, "output_contract")
	}
	return OutputContract{kind: kind}, nil
}

func EvidenceBundleOutputContract() OutputContract {
	return OutputContract{kind: OutputContractEvidenceBundle}
}

func (contract OutputContract) Kind() OutputContractKind { return contract.kind }

// PlanInput is one immutable proposal of pending WorkItems. It contains no
// lifecycle choice and introduces no second execution-unit model.
type PlanInput struct {
	Generation PlanGeneration
	Phases     []PhaseInstance
	WorkItems  []WorkItem
}

// Plan is an immutable proposal, not an aggregate or lifecycle authority.
// Goal.ApplyPlan is the only operation that can make it authoritative.
type Plan struct {
	generation PlanGeneration
	phases     []PhaseInstance
	items      []WorkItem
}

func NewPlan(input PlanInput) (Plan, error) {
	plan := Plan{
		generation: input.Generation,
		phases:     clonePhases(input.Phases),
		items:      cloneWorkItems(input.WorkItems),
	}
	if err := validatePlan(plan); err != nil {
		return Plan{}, err
	}
	return plan, nil
}

func (plan Plan) Generation() PlanGeneration { return plan.generation }
func (plan Plan) Phases() []PhaseInstance    { return clonePhases(plan.phases) }
func (plan Plan) WorkItems() []WorkItem      { return cloneWorkItems(plan.items) }

func validatePlan(plan Plan) error {
	return validatePlanShape(plan, false, false)
}

func validateRestoredPlan(plan Plan) error {
	return validatePlanShape(plan, false, true)
}

func validatePlanShape(plan Plan, requirePendingItems, allowMissingRequiredTests bool) error {
	byRef, err := validatePlanMembers(plan, requirePendingItems, allowMissingRequiredTests)
	if err != nil {
		return err
	}
	return validatePlanDependencies(plan.items, byRef)
}

func validatePlanMembers(plan Plan, requirePendingItems, allowMissingRequiredTests bool) (map[WorkItemRef]WorkItem, error) {
	switch {
	case plan.generation == 0:
		return nil, domainError(ErrorInvalidPlan, "plan_generation")
	case len(plan.phases) == 0:
		return nil, domainError(ErrorInvalidPlan, "phases")
	case len(plan.items) == 0:
		return nil, domainError(ErrorWorkItemsRequired, "work_items")
	}
	phases := make(map[PhaseKey]struct{}, len(plan.phases))
	phaseRefs := make(map[PhaseRef]struct{}, len(plan.phases))
	for _, phase := range plan.phases {
		if !validPhaseRef(phase.ref) || !validPhaseTemplateRef(phase.templateRef) || !validPhaseKey(phase.key) {
			return nil, domainError(ErrorInvalidPlan, "phase_key")
		}
		if err := validateUniqueRefs(phase.inputRefs, validInputRef, "input_ref"); err != nil {
			return nil, err
		}
		if err := validateUniqueRefs(phase.criterionRefs, validCriterionRef, "criterion_ref"); err != nil {
			return nil, err
		}
		if _, duplicate := phaseRefs[phase.ref]; duplicate {
			return nil, domainError(ErrorInvalidPlan, "duplicate_phase_ref")
		}
		if _, duplicate := phases[phase.key]; duplicate {
			return nil, domainError(ErrorInvalidPlan, "duplicate_phase")
		}
		phaseRefs[phase.ref] = struct{}{}
		phases[phase.key] = struct{}{}
	}
	byRef := make(map[WorkItemRef]WorkItem, len(plan.items))
	demandRefs := make(map[string]struct{}, len(plan.items))
	for _, item := range plan.items {
		if !validWorkItemRef(item.ref) {
			return nil, domainError(ErrorInvalidRef, "work_item_ref")
		}
		if _, duplicate := byRef[item.ref]; duplicate {
			return nil, domainError(ErrorDuplicateWorkItem, "work_item_ref")
		}
		if _, exists := phases[item.phase]; !exists {
			return nil, domainError(ErrorInvalidPlan, "work_item_phase")
		}
		if !validRoleKey(item.role) {
			return nil, domainError(ErrorInvalidPlan, "work_item_role")
		}
		if !validOutputContractKind(item.outputContract.kind) {
			return nil, domainError(ErrorInvalidPlan, "output_contract")
		}
		if requirePendingItems && (item.state != WorkItemStatePending || item.revision != 1 ||
			!item.startedAt.IsZero() || !item.finishedAt.IsZero() ||
			validExecutionRef(item.execution) || len(item.artifacts) != 0 || len(item.attestations) != 0 ||
			item.paused || item.cancelRequested || item.controlSequence != 0 || item.interruptCause != "" ||
			!item.interruptedAt.IsZero() || validWorkItemRef(item.reworkOf)) {
			return nil, domainError(ErrorInvalidPlan, "work_item_snapshot")
		}
		if err := validateWorkItemPlanMetadataForRestore(item, allowMissingRequiredTests); err != nil {
			return nil, err
		}
		if _, duplicate := demandRefs[item.budgetDemand.Ref]; duplicate {
			return nil, domainError(ErrorInvalidPlan, "duplicate_budget_demand_ref")
		}
		demandRefs[item.budgetDemand.Ref] = struct{}{}
		if !requirePendingItems && item.state != WorkItemStatePending {
			if err := validateRestoredWorkItem(item); err != nil {
				return nil, err
			}
		}
		byRef[item.ref] = item
	}
	return byRef, nil
}

func validatePlanDependencies(items []WorkItem, byRef map[WorkItemRef]WorkItem) error {
	indegree := make(map[WorkItemRef]int, len(items))
	dependents := make(map[WorkItemRef][]WorkItemRef, len(items))
	for _, item := range items {
		for _, dependency := range item.dependencies {
			if _, exists := byRef[dependency]; !exists {
				return domainError(ErrorInvalidPlan, "dependency_ref")
			}
			indegree[item.ref]++
			dependents[dependency] = append(dependents[dependency], item.ref)
		}
		if validWorkItemRef(item.parent) {
			if _, exists := byRef[item.parent]; !exists {
				return domainError(ErrorInvalidPlan, "parent_ref")
			}
			indegree[item.ref]++
			dependents[item.parent] = append(dependents[item.parent], item.ref)
		}
		if validWorkItemRef(item.reworkOf) {
			source, exists := byRef[item.reworkOf]
			if !exists || source.state != WorkItemStateSuperseded || item.reworkOf == item.ref ||
				reworkSourceTouchesHandoffIn(item.reworkOf, byRef) {
				return domainError(ErrorInvalidPlan, "rework_of")
			}
			// A superseded source waits for every successor's logical outcome.
			indegree[item.reworkOf]++
			dependents[item.ref] = append(dependents[item.ref], item.reworkOf)
		}
	}

	queue := make([]WorkItemRef, 0, len(items))
	for _, item := range items {
		if indegree[item.ref] == 0 {
			queue = append(queue, item.ref)
		}
	}
	visited := 0
	for len(queue) > 0 {
		ref := queue[0]
		queue = queue[1:]
		visited++
		for _, dependent := range dependents[ref] {
			indegree[dependent]--
			if indegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}
	if visited != len(items) {
		return domainError(ErrorInvalidPlan, "dependency_or_parent_cycle")
	}
	for _, item := range items {
		if item.state == WorkItemStateSuperseded && !hasReworkSuccessorIn(item.ref, byRef) {
			return domainError(ErrorInvalidPlan, "superseded_without_successor")
		}
		if item.state != WorkItemStatePending && item.state != WorkItemStateSkipped &&
			item.state != WorkItemStateCanceled && item.state != WorkItemStateSuperseded &&
			!dependenciesSucceededIn(item, byRef) {
			return domainError(ErrorInvalidPlan, "active_dependency_state")
		}
		if item.state == WorkItemStatePending && hasFailedDependencyIn(item, byRef) {
			return domainError(ErrorInvalidPlan, "pending_failed_dependency")
		}
	}
	for leftIndex, left := range items {
		if left.state != WorkItemStateRunning {
			continue
		}
		for _, right := range items[leftIndex+1:] {
			if right.state == WorkItemStateRunning && writeSetsOverlap(left.writeSet, right.writeSet) {
				return domainError(ErrorInvalidPlan, "running_write_set_conflict")
			}
		}
	}
	return nil
}

func validateWorkItemPlanMetadata(item WorkItem) error {
	return validateWorkItemPlanMetadataForRestore(item, false)
}

func validateWorkItemPlanMetadataForRestore(item WorkItem, allowMissingRequiredTests bool) error {
	if err := governance.ValidateBudgetDemand(item.budgetDemand); err != nil {
		return domainError(ErrorInvalidPlan, "budget_demand")
	}
	if err := governance.ValidateSecurityCriticality(item.securityCriticality); err != nil {
		return domainError(ErrorInvalidPlan, "security_criticality")
	}
	if err := governance.ValidateReasoningEffort(item.reasoningEffort); err != nil {
		return domainError(ErrorInvalidPlan, "reasoning_effort")
	}
	if item.handoffRequired && !validWorkItemRef(item.parent) {
		return domainError(ErrorInvalidPlan, "handoff_parent")
	}
	if validWorkItemRef(item.parent) && item.parent == item.ref {
		return domainError(ErrorInvalidPlan, "self_parent")
	}
	seenDependencies := make(map[WorkItemRef]struct{}, len(item.dependencies))
	for _, ref := range item.dependencies {
		if !validWorkItemRef(ref) {
			return domainError(ErrorInvalidRef, "dependency_ref")
		}
		if ref == item.ref {
			return domainError(ErrorInvalidPlan, "self_dependency")
		}
		if _, duplicate := seenDependencies[ref]; duplicate {
			return domainError(ErrorInvalidPlan, "duplicate_dependency")
		}
		seenDependencies[ref] = struct{}{}
	}
	seenScopes := make(map[WriteScope]struct{}, len(item.writeSet))
	for _, scope := range item.writeSet {
		if !validWriteScopeValue(scope) {
			return domainError(ErrorInvalidPlan, "write_scope")
		}
		if _, duplicate := seenScopes[scope]; duplicate {
			return domainError(ErrorInvalidPlan, "duplicate_write_scope")
		}
		seenScopes[scope] = struct{}{}
	}
	if err := validateRequiredTests(item.requiredTests); err != nil {
		return err
	}
	if !allowMissingRequiredTests && len(item.writeSet) > 0 && len(item.requiredTests) == 0 {
		return domainError(ErrorInvalidPlan, "required_tests")
	}
	if err := validateUniqueRefs(item.skillRefs, validSkillRef, "skill_ref"); err != nil {
		return err
	}
	if err := validateUniqueRefs(item.toolRefs, validToolRef, "tool_ref"); err != nil {
		return err
	}
	if err := validateUniqueRefs(item.capabilityRefs, validCapabilityRef, "capability_ref"); err != nil {
		return err
	}
	return nil
}

func validPlanKey(field, value string) (string, error) {
	if value == "" || strings.TrimSpace(value) != value {
		return "", domainError(ErrorInvalidPlan, field)
	}
	for index, character := range value {
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') ||
			(index > 0 && strings.ContainsRune("._:-", character)) {
			continue
		}
		return "", domainError(ErrorInvalidPlan, field)
	}
	return value, nil
}

func validWriteScope(value string) (string, error) {
	if value == "" || strings.TrimSpace(value) != value || path.IsAbs(value) ||
		strings.Contains(value, "\\") || strings.ContainsAny(value, "*?[]") {
		return "", domainError(ErrorInvalidPlan, "write_scope")
	}
	cleaned := path.Clean(value)
	if cleaned == "." || cleaned != value || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", domainError(ErrorInvalidPlan, "write_scope")
	}
	return value, nil
}

func validOutputContractKind(kind OutputContractKind) bool {
	return kind == OutputContractEvidenceBundle ||
		kind == OutputContractArtifact ||
		kind == OutputContractAttestation
}

func nextPlanGeneration(current PlanGeneration) (PlanGeneration, error) {
	if uint64(current) == math.MaxUint64 {
		return 0, domainError(ErrorInvalidPlan, "plan_generation")
	}
	return current + 1, nil
}

func clonePhases(phases []PhaseInstance) []PhaseInstance {
	cloned := make([]PhaseInstance, len(phases))
	for index, phase := range phases {
		cloned[index] = phase.clone()
	}
	return cloned
}

func cloneWorkItems(items []WorkItem) []WorkItem {
	cloned := make([]WorkItem, len(items))
	for index, item := range items {
		cloned[index] = item.clone()
	}
	return cloned
}

func writeSetsOverlap(first, second []WriteScope) bool {
	for _, left := range first {
		for _, right := range second {
			if writeScopesOverlap(left, right) {
				return true
			}
		}
	}
	return false
}

func writeScopesOverlap(first, second WriteScope) bool {
	left := scopeSegments(first)
	right := scopeSegments(second)
	shortest := len(left)
	if len(right) < shortest {
		shortest = len(right)
	}
	for index := 0; index < shortest; index++ {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func scopeSegments(scope WriteScope) []string {
	return strings.Split(scope.value, "/")
}

func (phase PhaseInstance) clone() PhaseInstance {
	phase.inputRefs = cloneRefs(phase.inputRefs)
	phase.criterionRefs = cloneRefs(phase.criterionRefs)
	return phase
}

func equalPhaseInstances(left, right PhaseInstance) bool {
	return left.ref == right.ref && left.key == right.key && left.templateRef == right.templateRef &&
		refsEqual(left.inputRefs, right.inputRefs) && refsEqual(left.criterionRefs, right.criterionRefs)
}

func validateUniqueRefs[T comparable](refs []T, valid func(T) bool, field string) error {
	seen := make(map[T]struct{}, len(refs))
	for _, ref := range refs {
		if !valid(ref) {
			return domainError(ErrorInvalidRef, field)
		}
		if _, duplicate := seen[ref]; duplicate {
			return domainError(ErrorInvalidPlan, "duplicate_"+field)
		}
		seen[ref] = struct{}{}
	}
	return nil
}

func cloneRefs[T any](refs []T) []T {
	return append([]T(nil), refs...)
}

func refsEqual[T comparable](left, right []T) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func dependenciesSucceededIn(item WorkItem, items map[WorkItemRef]WorkItem) bool {
	for _, dependency := range item.dependencies {
		outcome, resolved := logicalWorkItemOutcomeIn(items, dependency, make(map[WorkItemRef]bool))
		if !resolved || outcome != WorkItemLogicalSucceeded {
			return false
		}
	}
	return true
}

func hasFailedDependencyIn(item WorkItem, items map[WorkItemRef]WorkItem) bool {
	for _, dependency := range item.dependencies {
		outcome, resolved := logicalWorkItemOutcomeIn(items, dependency, make(map[WorkItemRef]bool))
		if resolved && outcome == WorkItemLogicalFailed {
			return true
		}
	}
	return false
}

func hasReworkSuccessorIn(source WorkItemRef, items map[WorkItemRef]WorkItem) bool {
	for _, item := range items {
		if item.reworkOf == source {
			return true
		}
	}
	return false
}

func reworkSourceTouchesHandoffIn(source WorkItemRef, items map[WorkItemRef]WorkItem) bool {
	if items[source].handoffRequired {
		return true
	}
	for _, item := range items {
		if item.parent == source && item.handoffRequired {
			return true
		}
	}
	return false
}
