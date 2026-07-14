package application

import (
	"context"
	"errors"
	"hash"
	"strconv"
	"strings"
	"time"

	"orquesta/internal/goal"
)

// PlanSpec is an application input contract. Keys are local to the request;
// durable opaque WorkItem refs are generated inside the application.
type PlanSpec struct {
	Phases    []PhaseSpec
	WorkItems []WorkItemSpec
}

// PhaseSpec is transport-neutral immutable metadata. References point to
// definitions owned outside the Goal aggregate; application only validates
// and carries them into the authoritative plan.
type PhaseSpec struct {
	Ref           string
	Key           string
	TemplateRef   string
	InputRefs     []string
	CriterionRefs []string
}

type WorkItemSpec struct {
	Key            string
	Objective      string
	Phase          string
	Role           string
	Parent         string
	Dependencies   []string
	WriteSet       []string
	SkillRefs      []string
	ToolRefs       []string
	CapabilityRefs []string
	OutputContract goal.OutputContractKind
}

func (orchestrator *Orchestrator) compilePlan(
	ctx context.Context,
	request SubmitRequest,
	goalRef goal.GoalRef,
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
		ref, err := goal.NewPhaseRef(phaseSpec.Ref)
		if err != nil {
			return goal.Plan{}, err
		}
		key, err := goal.NewPhaseKey(phaseSpec.Key)
		if err != nil {
			return goal.Plan{}, err
		}
		templateRef, err := goal.NewPhaseTemplateRef(phaseSpec.TemplateRef)
		if err != nil {
			return goal.Plan{}, err
		}
		inputRefs, err := parsePlanRefs(phaseSpec.InputRefs, goal.NewInputRef)
		if err != nil {
			return goal.Plan{}, err
		}
		criterionRefs, err := parsePlanRefs(phaseSpec.CriterionRefs, goal.NewCriterionRef)
		if err != nil {
			return goal.Plan{}, err
		}
		phase, err := goal.NewPhaseInstanceWithMetadata(goal.PhaseInstanceInput{
			Ref: ref, Key: key, TemplateRef: templateRef,
			InputRefs: inputRefs, CriterionRefs: criterionRefs,
		})
		if err != nil {
			return goal.Plan{}, err
		}
		phases = append(phases, phase)
	}

	refs := make(map[string]goal.WorkItemRef, len(spec.WorkItems))
	for _, itemSpec := range spec.WorkItems {
		if strings.TrimSpace(itemSpec.Key) == "" || strings.TrimSpace(itemSpec.Key) != itemSpec.Key {
			return goal.Plan{}, errors.New("application.plan_item_key_invalid")
		}
		if _, duplicate := refs[itemSpec.Key]; duplicate {
			return goal.Plan{}, errors.New("application.plan_item_key_duplicate")
		}
		ref, err := newWorkItemRef(ctx, orchestrator.ids)
		if err != nil {
			return goal.Plan{}, err
		}
		refs[itemSpec.Key] = ref
	}

	items := make([]goal.WorkItem, 0, len(spec.WorkItems))
	for _, itemSpec := range spec.WorkItems {
		phaseKey, err := goal.NewPhaseKey(itemSpec.Phase)
		if err != nil {
			return goal.Plan{}, err
		}
		roleKey, err := goal.NewRoleKey(itemSpec.Role)
		if err != nil {
			return goal.Plan{}, err
		}
		contract, err := goal.NewOutputContract(itemSpec.OutputContract)
		if err != nil {
			return goal.Plan{}, err
		}
		dependencies := make([]goal.WorkItemRef, 0, len(itemSpec.Dependencies))
		for _, key := range itemSpec.Dependencies {
			ref, ok := refs[key]
			if !ok {
				return goal.Plan{}, errors.New("application.plan_dependency_unknown")
			}
			dependencies = append(dependencies, ref)
		}
		var parent goal.WorkItemRef
		if itemSpec.Parent != "" {
			var ok bool
			parent, ok = refs[itemSpec.Parent]
			if !ok {
				return goal.Plan{}, errors.New("application.plan_parent_unknown")
			}
		}
		writeSet := make([]goal.WriteScope, 0, len(itemSpec.WriteSet))
		for _, raw := range itemSpec.WriteSet {
			scope, err := goal.NewWriteScope(raw)
			if err != nil {
				return goal.Plan{}, err
			}
			writeSet = append(writeSet, scope)
		}
		skillRefs, err := parsePlanRefs(itemSpec.SkillRefs, goal.NewSkillRef)
		if err != nil {
			return goal.Plan{}, err
		}
		toolRefs, err := parsePlanRefs(itemSpec.ToolRefs, goal.NewToolRef)
		if err != nil {
			return goal.Plan{}, err
		}
		capabilityRefs, err := parsePlanRefs(itemSpec.CapabilityRefs, goal.NewCapabilityRef)
		if err != nil {
			return goal.Plan{}, err
		}
		item, err := goal.NewWorkItem(goal.NewWorkItemInput{
			Ref: refs[itemSpec.Key], Goal: goalRef, Actor: request.ActorRef,
			Project: request.ProjectRef, Objective: itemSpec.Objective, CreatedAt: at,
			Phase: phaseKey, Role: roleKey, Parent: parent, Dependencies: dependencies,
			WriteSet: writeSet, SkillRefs: skillRefs, ToolRefs: toolRefs,
			CapabilityRefs: capabilityRefs, OutputContract: contract,
		})
		if err != nil {
			return goal.Plan{}, err
		}
		items = append(items, item)
	}
	return goal.NewPlan(goal.PlanInput{Generation: 1, Phases: phases, WorkItems: items})
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
	at time.Time,
) ([]ExecutionRecord, []ActionRecord, []EventRecord, error) {
	scheduled := make(map[goal.WorkItemRef]struct{}, len(existing))
	for _, execution := range existing {
		scheduled[execution.WorkItemRef] = struct{}{}
	}
	var executions []ExecutionRecord
	var actions []ActionRecord
	var events []EventRecord
	for _, item := range aggregate.ReadyWorkItems() {
		if _, exists := scheduled[item.Ref()]; exists {
			continue
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
		executions = append(executions, execution)
		actions = append(actions, ActionRecord{
			Ref: "action:launch:" + executionRef.String(), Kind: ActionLaunchAgent,
			GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), ExecutionRef: executionRef,
			PlanGeneration: aggregate.PlanGeneration(), WorkItemGeneration: item.Revision(),
			AvailableAt: at,
		})
		events = append(events, EventRecord{
			Ref: "event:execution-queued:" + executionRef.String(), Kind: "execution.queued",
			GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), ExecutionRef: executionRef,
			OccurredAt: at,
		})
		scheduled[item.Ref()] = struct{}{}
	}
	return executions, actions, events, nil
}

func writePlanFingerprint(digest hash.Hash, spec *PlanSpec) {
	writeFingerprintField(digest, "orquesta.plan.v1")
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
		writeFingerprintField(digest, string(item.OutputContract))
		writeFingerprintStrings(digest, "dependencies", item.Dependencies)
		writeFingerprintStrings(digest, "write_set", item.WriteSet)
		writeFingerprintStrings(digest, "skills", item.SkillRefs)
		writeFingerprintStrings(digest, "tools", item.ToolRefs)
		writeFingerprintStrings(digest, "capabilities", item.CapabilityRefs)
	}
}

func writeFingerprintStrings(digest hash.Hash, label string, values []string) {
	writeFingerprintField(digest, label)
	writeFingerprintField(digest, strconv.Itoa(len(values)))
	for _, value := range values {
		writeFingerprintField(digest, value)
	}
}
