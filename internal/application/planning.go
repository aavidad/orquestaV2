package application

import (
	"context"
	"errors"
	"hash"
	"strconv"
	"strings"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
)

// PlanSpec uses request-local keys; application generates durable refs.
type PlanSpec struct {
	Phases    []PhaseSpec
	WorkItems []WorkItemSpec
}

// PhaseSpec carries transport-neutral refs to externally owned definitions.
type PhaseSpec struct {
	Ref           string
	Key           string
	TemplateRef   string
	InputRefs     []string
	CriterionRefs []string
}

type WorkItemSpec struct {
	Key                 string
	Objective           string
	Phase               string
	Role                string
	Parent              string
	HandoffRequired     bool
	Dependencies        []string
	WriteSet            []string
	SkillRefs           []string
	ToolRefs            []string
	CapabilityRefs      []string
	OutputContract      goal.OutputContractKind
	BudgetDemand        governance.BudgetDemand
	SecurityCriticality governance.SecurityCriticality
	ReasoningEffort     governance.ReasoningEffort
}

func (orchestrator *Orchestrator) compilePlan(
	ctx context.Context,
	request SubmitRequest,
	goalRef goal.GoalRef,
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	at time.Time,
) (goal.Plan, error) {
	spec := request.Plan
	if spec == nil {
		spec = &PlanSpec{
			Phases: []PhaseSpec{{
				Ref: "phase-instance:" + goalRef.String() + ":default",
				Key: goal.DefaultPhaseKey().String(), TemplateRef: "phase-template:default",
			}},
			WorkItems: []WorkItemSpec{{
				Key: "work:default", Objective: normalizedObjective(request.Statement, request.NormalizedObjective),
				Phase: goal.DefaultPhaseKey().String(), Role: goal.DefaultRoleKey().String(),
				OutputContract: goal.OutputContractEvidenceBundle,
			}},
		}
	}
	if len(spec.Phases) == 0 || len(spec.WorkItems) == 0 {
		return goal.Plan{}, errors.New("application.plan_required")
	}

	phases := make([]goal.PhaseInstance, 0, len(spec.Phases))
	for _, phaseSpec := range spec.Phases {
		phase, err := compilePhaseSpec(phaseSpec)
		if err != nil {
			return goal.Plan{}, err
		}
		phases = append(phases, phase)
	}

	refs, err := allocateWorkItemRefs(ctx, orchestrator.ids, spec.WorkItems, nil)
	if err != nil {
		return goal.Plan{}, err
	}

	resolver := workItemRefResolver{
		requestLocal:  refs,
		parentUnknown: "application.plan_parent_unknown",
	}
	scope := workItemCompileScope{
		goalRef: goalRef, actorRef: actorRef, projectRef: projectRef, createdAt: at,
		defaultDemand: orchestrator.budgetPolicy.DefaultWorkItemDemand,
		goalLimit:     orchestrator.budgetPolicy.GoalEnvelopeTemplate.Limit,
	}
	items := make([]goal.WorkItem, 0, len(spec.WorkItems))
	for _, itemSpec := range spec.WorkItems {
		item, err := compileWorkItemSpec(itemSpec, refs[itemSpec.Key], scope, resolver)
		if err != nil {
			return goal.Plan{}, err
		}
		items = append(items, item)
	}
	if err := validatePlanFanout(items, orchestrator.maxChildrenPerParent); err != nil {
		return goal.Plan{}, err
	}
	return goal.NewPlan(goal.PlanInput{Generation: 1, Phases: phases, WorkItems: items})
}

// compilePlanExtension preserves the existing plan prefix and resolves local/existing refs.
func (orchestrator *Orchestrator) compilePlanExtension(
	ctx context.Context,
	aggregate goal.Goal,
	spec PlanSpec,
	policy effectPolicySnapshot,
	at time.Time,
) (goal.Plan, error) {
	if len(spec.WorkItems) == 0 {
		return goal.Plan{}, errors.New("application.director_plan_work_items_required")
	}

	phases := aggregate.Phases()
	phaseKeys := make(map[string]struct{}, len(phases)+len(spec.Phases))
	for _, phase := range phases {
		phaseKeys[phase.Key().String()] = struct{}{}
	}
	for _, phaseSpec := range spec.Phases {
		if _, duplicate := phaseKeys[phaseSpec.Key]; duplicate {
			return goal.Plan{}, errors.New("application.plan_phase_duplicate")
		}
		phase, err := compilePhaseSpec(phaseSpec)
		if err != nil {
			return goal.Plan{}, err
		}
		phases = append(phases, phase)
		phaseKeys[phase.Key().String()] = struct{}{}
	}

	existingRefs := indexWorkItemRefs(aggregate.WorkItems())
	newRefs, err := allocateWorkItemRefs(ctx, orchestrator.ids, spec.WorkItems, existingRefs)
	if err != nil {
		return goal.Plan{}, err
	}

	resolver := workItemRefResolver{
		requestLocal: newRefs,
		existing:     existingRefs,
		// Keep the existing Director error contract for an unknown parent.
		parentUnknown: "application.plan_dependency_unknown",
	}
	scope := workItemCompileScope{
		goalRef: aggregate.Ref(), actorRef: aggregate.Actor(), projectRef: aggregate.Project(), createdAt: at,
		defaultDemand: policy.DefaultDemand,
		goalLimit:     policy.GoalLimit,
	}
	items := aggregate.WorkItems()
	for _, itemSpec := range spec.WorkItems {
		if _, exists := phaseKeys[itemSpec.Phase]; !exists {
			return goal.Plan{}, errors.New("application.plan_phase_unknown")
		}
		item, err := compileWorkItemSpec(itemSpec, newRefs[itemSpec.Key], scope, resolver)
		if err != nil {
			return goal.Plan{}, err
		}
		items = append(items, item)
	}
	if err := validatePlanFanout(items, orchestrator.maxChildrenPerParent); err != nil {
		return goal.Plan{}, err
	}

	generation := aggregate.PlanGeneration() + 1
	if generation == 0 {
		return goal.Plan{}, errors.New("application.plan_generation_overflow")
	}
	return goal.NewPlan(goal.PlanInput{Generation: generation, Phases: phases, WorkItems: items})
}

func validatePlanFanout(items []goal.WorkItem, maximum int) error {
	children := make(map[goal.WorkItemRef]int)
	for _, item := range items {
		parent, found := item.Parent()
		if !found {
			continue
		}
		children[parent]++
		if children[parent] > maximum {
			return errors.New("application.plan_fanout_exceeded")
		}
	}
	return nil
}

func compilePhaseSpec(spec PhaseSpec) (goal.PhaseInstance, error) {
	ref, err := goal.NewPhaseRef(spec.Ref)
	if err != nil {
		return goal.PhaseInstance{}, err
	}
	key, err := goal.NewPhaseKey(spec.Key)
	if err != nil {
		return goal.PhaseInstance{}, err
	}
	templateRef, err := goal.NewPhaseTemplateRef(spec.TemplateRef)
	if err != nil {
		return goal.PhaseInstance{}, err
	}
	inputRefs, err := parsePlanRefs(spec.InputRefs, goal.NewInputRef)
	if err != nil {
		return goal.PhaseInstance{}, err
	}
	criterionRefs, err := parsePlanRefs(spec.CriterionRefs, goal.NewCriterionRef)
	if err != nil {
		return goal.PhaseInstance{}, err
	}
	return goal.NewPhaseInstanceWithMetadata(goal.PhaseInstanceInput{
		Ref: ref, Key: key, TemplateRef: templateRef,
		InputRefs: inputRefs, CriterionRefs: criterionRefs,
	})
}

func allocateWorkItemRefs(
	ctx context.Context,
	ids IDGenerator,
	specs []WorkItemSpec,
	existing map[string]goal.WorkItemRef,
) (map[string]goal.WorkItemRef, error) {
	refs := make(map[string]goal.WorkItemRef, len(specs))
	for _, spec := range specs {
		if strings.TrimSpace(spec.Key) == "" || strings.TrimSpace(spec.Key) != spec.Key {
			return nil, errors.New("application.plan_item_key_invalid")
		}
		if _, duplicate := refs[spec.Key]; duplicate {
			return nil, errors.New("application.plan_item_key_duplicate")
		}
		if _, collision := existing[spec.Key]; collision {
			return nil, errors.New("application.plan_item_key_ref_collision")
		}
		ref, err := newWorkItemRef(ctx, ids)
		if err != nil {
			return nil, err
		}
		refs[spec.Key] = ref
	}
	return refs, nil
}

type workItemCompileScope struct {
	goalRef       goal.GoalRef
	actorRef      goal.ActorRef
	projectRef    goal.ProjectRef
	createdAt     time.Time
	defaultDemand governance.ResourceVector
	goalLimit     governance.ResourceVector
}

type workItemRefResolver struct {
	requestLocal  map[string]goal.WorkItemRef
	existing      map[string]goal.WorkItemRef
	parentUnknown string
}

func (resolver workItemRefResolver) dependency(value string) (goal.WorkItemRef, error) {
	return resolver.resolve(value, "application.plan_dependency_unknown")
}

func (resolver workItemRefResolver) parent(value string) (goal.WorkItemRef, error) {
	return resolver.resolve(value, resolver.parentUnknown)
}

func (resolver workItemRefResolver) resolve(value, unknown string) (goal.WorkItemRef, error) {
	if ref, exists := resolver.requestLocal[value]; exists {
		return ref, nil
	}
	if ref, exists := resolver.existing[value]; exists {
		return ref, nil
	}
	return goal.WorkItemRef{}, errors.New(unknown)
}

func compileWorkItemSpec(
	spec WorkItemSpec,
	ref goal.WorkItemRef,
	scope workItemCompileScope,
	resolver workItemRefResolver,
) (goal.WorkItem, error) {
	phaseKey, err := goal.NewPhaseKey(spec.Phase)
	if err != nil {
		return goal.WorkItem{}, err
	}
	roleKey, err := goal.NewRoleKey(spec.Role)
	if err != nil {
		return goal.WorkItem{}, err
	}
	contract, err := goal.NewOutputContract(spec.OutputContract)
	if err != nil {
		return goal.WorkItem{}, err
	}
	dependencies := make([]goal.WorkItemRef, 0, len(spec.Dependencies))
	for _, value := range spec.Dependencies {
		dependency, err := resolver.dependency(value)
		if err != nil {
			return goal.WorkItem{}, err
		}
		dependencies = append(dependencies, dependency)
	}
	var parent goal.WorkItemRef
	if spec.Parent != "" {
		parent, err = resolver.parent(spec.Parent)
		if err != nil {
			return goal.WorkItem{}, err
		}
	}
	writeSet := make([]goal.WriteScope, 0, len(spec.WriteSet))
	for _, raw := range spec.WriteSet {
		writeScope, err := goal.NewWriteScope(raw)
		if err != nil {
			return goal.WorkItem{}, err
		}
		writeSet = append(writeSet, writeScope)
	}
	skillRefs, err := parsePlanRefs(spec.SkillRefs, goal.NewSkillRef)
	if err != nil {
		return goal.WorkItem{}, err
	}
	toolRefs, err := parsePlanRefs(spec.ToolRefs, goal.NewToolRef)
	if err != nil {
		return goal.WorkItem{}, err
	}
	capabilityRefs, err := parsePlanRefs(spec.CapabilityRefs, goal.NewCapabilityRef)
	if err != nil {
		return goal.WorkItem{}, err
	}
	demand := effectiveDemand(spec.BudgetDemand, scope.defaultDemand)
	fits, fitErr := governance.Fits(scope.goalLimit, demand.Resources)
	if fitErr != nil || !fits || demand.Resources.ProcessSlots == 0 {
		return goal.WorkItem{}, errors.New("application.work_item_budget_demand_invalid")
	}
	return goal.NewWorkItem(goal.NewWorkItemInput{
		Ref: ref, Goal: scope.goalRef, Actor: scope.actorRef,
		Project: scope.projectRef, Objective: spec.Objective, CreatedAt: scope.createdAt,
		Phase: phaseKey, Role: roleKey, Parent: parent, HandoffRequired: spec.HandoffRequired,
		Dependencies: dependencies,
		WriteSet:     writeSet, SkillRefs: skillRefs, ToolRefs: toolRefs,
		CapabilityRefs: capabilityRefs, OutputContract: contract,
		BudgetDemand: demand, SecurityCriticality: spec.SecurityCriticality,
		ReasoningEffort: spec.ReasoningEffort,
	})
}

func indexWorkItemRefs(items []goal.WorkItem) map[string]goal.WorkItemRef {
	refs := make(map[string]goal.WorkItemRef, len(items))
	for _, item := range items {
		refs[item.Ref().String()] = item.Ref()
	}
	return refs
}

func parsePlanRefs[T any](values []string, parse func(string) (T, error)) ([]T, error) {
	refs := make([]T, 0, len(values))
	for _, value := range values {
		ref, err := parse(value)
		if err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	return refs, nil
}

func (orchestrator *Orchestrator) scheduleReady(
	ctx context.Context,
	aggregate goal.Goal,
	existing []ExecutionRecord,
	authorities []WorkItemAuthority,
	policy effectPolicySnapshot,
	at time.Time,
) ([]ExecutionRecord, []ActionRecord, []EventRecord, error) {
	scheduled := make(map[goal.WorkItemRef]struct{}, len(existing))
	for _, execution := range existing {
		scheduled[execution.WorkItemRef] = struct{}{}
	}
	var executions []ExecutionRecord
	var actions []ActionRecord
	var events []EventRecord
	legacyGovernance := len(authorities) == 0
	for _, item := range aggregate.ReadyWorkItems() {
		if _, exists := scheduled[item.Ref()]; exists {
			continue
		}
		authority, found := workItemAuthorityFor(authorities, item.Ref())
		if !found && legacyGovernance {
			continue
		}
		if !found || validateWorkItemAuthority(aggregate, authority) != nil {
			return nil, nil, nil, errors.New("application.work_item_authority_missing")
		}
		executionRef, err := newExecutionRef(ctx, orchestrator.ids)
		if err != nil {
			return nil, nil, nil, err
		}
		execution := ExecutionRecord{
			Ref: executionRef, GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(),
			AttemptNo: 1, MaxExecutionAttempts: orchestrator.maxExecutionAttempts,
			PlanGeneration: aggregate.PlanGeneration(), AppSpecGeneration: aggregate.AppSpec().Generation(),
			SpecHash: aggregate.SpecHash(),
			State:    ExecutionQueued, ArtifactMediaType: agentArtifactMediaType,
			IdempotencyKey: "execution:" + executionRef.String(),
			MaxOutputBytes: orchestrator.maxOutputBytes,
			CreatedAt:      at,
		}
		action, err := orchestrator.launchAction(policy, aggregate, item, execution, authority, at, at)
		if err != nil {
			return nil, nil, nil, err
		}
		executions = append(executions, execution)
		actions = append(actions, action)
		events = append(events, EventRecord{
			Ref: "event:execution-queued:" + executionRef.String(), Kind: "execution.queued",
			GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), ExecutionRef: executionRef,
			OccurredAt: at,
		})
		scheduled[item.Ref()] = struct{}{}
	}
	return executions, actions, events, nil
}

func (orchestrator *Orchestrator) scheduleHistoricalReady(
	ctx context.Context, record GoalRecord, aggregate goal.Goal, existing []ExecutionRecord, at time.Time,
) ([]ExecutionRecord, []ActionRecord, []EventRecord, error) {
	if legacyGovernanceRecord(record) {
		return nil, nil, nil, nil
	}
	policy, err := historicalEffectPolicy(record)
	if err != nil {
		return nil, nil, nil, err
	}
	return orchestrator.scheduleReady(ctx, aggregate, existing, record.WorkItemAuthorities, policy, at)
}

func writePlanFingerprint(digest hash.Hash, spec *PlanSpec) {
	version := "orquesta.plan.v1"
	if planDeclaresGovernance(spec) {
		version = "orquesta.plan.v2"
	}
	writeFingerprintField(digest, version)
	if spec == nil {
		writeFingerprintField(digest, "mode:default")
		return
	}
	writeFingerprintField(digest, "mode:explicit")
	writeFingerprintField(digest, "phases")
	writeFingerprintField(digest, strconv.Itoa(len(spec.Phases)))
	for _, phase := range spec.Phases {
		writeFingerprintField(digest, "phase")
		writeFingerprintField(digest, phase.Ref)
		writeFingerprintField(digest, phase.Key)
		writeFingerprintField(digest, phase.TemplateRef)
		writeFingerprintStrings(digest, "inputs", phase.InputRefs)
		writeFingerprintStrings(digest, "criteria", phase.CriterionRefs)
	}
	writeFingerprintField(digest, "work_items")
	writeFingerprintField(digest, strconv.Itoa(len(spec.WorkItems)))
	for _, item := range spec.WorkItems {
		writeFingerprintField(digest, "work_item")
		writeFingerprintField(digest, item.Key)
		writeFingerprintField(digest, item.Objective)
		writeFingerprintField(digest, item.Phase)
		writeFingerprintField(digest, item.Role)
		writeFingerprintField(digest, item.Parent)
		writeFingerprintField(digest, strconv.FormatBool(item.HandoffRequired))
		writeFingerprintField(digest, string(item.OutputContract))
		writeFingerprintStrings(digest, "dependencies", item.Dependencies)
		writeFingerprintStrings(digest, "write_set", item.WriteSet)
		writeFingerprintStrings(digest, "skills", item.SkillRefs)
		writeFingerprintStrings(digest, "tools", item.ToolRefs)
		writeFingerprintStrings(digest, "capabilities", item.CapabilityRefs)
		if version == "orquesta.plan.v2" {
			writeFingerprintField(digest, "governance")
			writeFingerprintField(digest, item.BudgetDemand.Ref)
			writeFingerprintField(digest, strconv.FormatInt(item.BudgetDemand.Resources.Tokens, 10))
			writeFingerprintField(digest, strconv.FormatInt(item.BudgetDemand.Resources.MoneyMicros, 10))
			writeFingerprintField(digest, string(item.BudgetDemand.Resources.Currency))
			writeFingerprintField(digest, strconv.FormatInt(item.BudgetDemand.Resources.ActiveTimeNS, 10))
			writeFingerprintField(digest, strconv.FormatInt(item.BudgetDemand.Resources.ProcessSlots, 10))
			writeFingerprintField(digest, strconv.FormatInt(item.BudgetDemand.Resources.DiskBytes, 10))
			writeFingerprintField(digest, string(item.SecurityCriticality))
			writeFingerprintField(digest, string(item.ReasoningEffort))
		}
	}
}

func planDeclaresGovernance(spec *PlanSpec) bool {
	if spec == nil {
		return false
	}
	for _, item := range spec.WorkItems {
		if item.BudgetDemand != (governance.BudgetDemand{}) || item.SecurityCriticality != "" ||
			item.ReasoningEffort != "" {
			return true
		}
	}
	return false
}

func writeFingerprintStrings(digest hash.Hash, label string, values []string) {
	writeFingerprintField(digest, label)
	writeFingerprintField(digest, strconv.Itoa(len(values)))
	for _, value := range values {
		writeFingerprintField(digest, value)
	}
}
