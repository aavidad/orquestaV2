package goal

import (
	"math"
	"path"
	"strings"
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

// PhaseInstance is immutable plan metadata. It intentionally has no state or
// lifecycle; phase progress is derived from its WorkItems.
type PhaseInstance struct {
	key PhaseKey
}

func NewPhaseInstance(key PhaseKey) (PhaseInstance, error) {
	if !validPhaseKey(key) {
		return PhaseInstance{}, domainError(ErrorInvalidPlan, "phase_key")
	}
	return PhaseInstance{key: key}, nil
}

func (phase PhaseInstance) Key() PhaseKey { return phase.key }

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
	return validatePlanShape(plan, true)
}

func validateRestoredPlan(plan Plan) error {
	return validatePlanShape(plan, false)
}

func validatePlanShape(plan Plan, requirePendingItems bool) error {
	if plan.generation == 0 {
		return domainError(ErrorInvalidPlan, "plan_generation")
	}
	if len(plan.phases) == 0 {
		return domainError(ErrorInvalidPlan, "phases")
	}
	if len(plan.items) == 0 {
		return domainError(ErrorWorkItemsRequired, "work_items")
	}

	phases := make(map[PhaseKey]struct{}, len(plan.phases))
	for _, phase := range plan.phases {
		if !validPhaseKey(phase.key) {
			return domainError(ErrorInvalidPlan, "phase_key")
		}
		if _, duplicate := phases[phase.key]; duplicate {
			return domainError(ErrorInvalidPlan, "duplicate_phase")
		}
		phases[phase.key] = struct{}{}
	}

	byRef := make(map[WorkItemRef]WorkItem, len(plan.items))
	for _, item := range plan.items {
		if !validWorkItemRef(item.ref) {
			return domainError(ErrorInvalidRef, "work_item_ref")
		}
		if _, duplicate := byRef[item.ref]; duplicate {
			return domainError(ErrorDuplicateWorkItem, "work_item_ref")
		}
		if _, exists := phases[item.phase]; !exists {
			return domainError(ErrorInvalidPlan, "work_item_phase")
		}
		if !validRoleKey(item.role) {
			return domainError(ErrorInvalidPlan, "work_item_role")
		}
		if !validOutputContractKind(item.outputContract.kind) {
			return domainError(ErrorInvalidPlan, "output_contract")
		}
		if requirePendingItems && (item.state != WorkItemStatePending || item.revision != 1 ||
			!item.startedAt.IsZero() || !item.finishedAt.IsZero() ||
			validExecutionRef(item.execution) || len(item.artifacts) != 0 || len(item.attestations) != 0) {
			return domainError(ErrorInvalidPlan, "work_item_snapshot")
		}
		if err := validateWorkItemPlanMetadata(item); err != nil {
			return err
		}
		byRef[item.ref] = item
	}

	indegree := make(map[WorkItemRef]int, len(plan.items))
	dependents := make(map[WorkItemRef][]WorkItemRef, len(plan.items))
	for _, item := range plan.items {
		for _, dependency := range item.dependencies {
			if _, exists := byRef[dependency]; !exists {
				return domainError(ErrorInvalidPlan, "dependency_ref")
			}
			indegree[item.ref]++
			dependents[dependency] = append(dependents[dependency], item.ref)
		}
	}

	queue := make([]WorkItemRef, 0, len(plan.items))
	for _, item := range plan.items {
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
	if visited != len(plan.items) {
		return domainError(ErrorInvalidPlan, "dependency_cycle")
	}
	return nil
}

func validateWorkItemPlanMetadata(item WorkItem) error {
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
	return append([]PhaseInstance(nil), phases...)
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
