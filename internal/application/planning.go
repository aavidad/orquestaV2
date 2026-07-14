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
	Phases    []string
	WorkItems []WorkItemSpec
}

type WorkItemSpec struct {
	Key            string
	Objective      string
	Phase          string
	Role           string
	Dependencies   []string
	WriteSet       []string
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
			Phases: []string{goal.DefaultPhaseKey().String()},
			WorkItems: []WorkItemSpec{{
				Key: "work:default", Objective: request.Statement,
				Phase: goal.DefaultPhaseKey().String(), Role: goal.DefaultRoleKey().String(),
				OutputContract: goal.OutputContractEvidenceBundle,
			}},
		}
	}
	if len(spec.Phases) == 0 || len(spec.WorkItems) == 0 {
		return goal.Plan{}, errors.New("application.plan_required")
	}

	phases := make([]goal.PhaseInstance, 0, len(spec.Phases))
	for _, raw := range spec.Phases {
		key, err := goal.NewPhaseKey(raw)
		if err != nil {
			return goal.Plan{}, err
		}
		phase, err := goal.NewPhaseInstance(key)
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
		writeSet := make([]goal.WriteScope, 0, len(itemSpec.WriteSet))
		for _, raw := range itemSpec.WriteSet {
			scope, err := goal.NewWriteScope(raw)
			if err != nil {
				return goal.Plan{}, err
			}
			writeSet = append(writeSet, scope)
		}
		item, err := goal.NewWorkItem(goal.NewWorkItemInput{
			Ref: refs[itemSpec.Key], Goal: goalRef, Actor: request.ActorRef,
			Project: request.ProjectRef, Objective: itemSpec.Objective, CreatedAt: at,
			Phase: phaseKey, Role: roleKey, Dependencies: dependencies,
			WriteSet: writeSet, OutputContract: contract,
		})
		if err != nil {
			return goal.Plan{}, err
		}
		items = append(items, item)
	}
	return goal.NewPlan(goal.PlanInput{Generation: 1, Phases: phases, WorkItems: items})
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
			State: ExecutionQueued, ArtifactMediaType: agentArtifactMediaType,
			IdempotencyKey: "execution:" + executionRef.String(),
			MaxOutputBytes: orchestrator.maxOutputBytes, MaxAttempts: orchestrator.maxActionAttempts,
			CreatedAt: at,
		}
		executions = append(executions, execution)
		actions = append(actions, ActionRecord{
			Ref: "action:launch:" + executionRef.String(), Kind: ActionLaunchAgent,
			GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), ExecutionRef: executionRef,
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
		writeFingerprintField(digest, phase)
	}
	writeFingerprintField(digest, "work_items")
	writeFingerprintField(digest, strconv.Itoa(len(spec.WorkItems)))
	for _, item := range spec.WorkItems {
		writeFingerprintField(digest, "work_item")
		writeFingerprintField(digest, item.Key)
		writeFingerprintField(digest, item.Objective)
		writeFingerprintField(digest, item.Phase)
		writeFingerprintField(digest, item.Role)
		writeFingerprintField(digest, string(item.OutputContract))
		writeFingerprintField(digest, "dependencies")
		writeFingerprintField(digest, strconv.Itoa(len(item.Dependencies)))
		for _, dependency := range item.Dependencies {
			writeFingerprintField(digest, dependency)
		}
		writeFingerprintField(digest, "write_set")
		writeFingerprintField(digest, strconv.Itoa(len(item.WriteSet)))
		for _, scope := range item.WriteSet {
			writeFingerprintField(digest, scope)
		}
	}
}
