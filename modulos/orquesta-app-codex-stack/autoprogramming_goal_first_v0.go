package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func autoprogrammingBridgeStartGoalFirstV0(
	ctx context.Context,
	request AutoprogrammingBridgeRequestV0,
	result AutoprogrammingBridgeResultV0,
	ports orquestaappdirectorservice.StartAppDirectorPortsV0,
) (AutoprogrammingBridgeResultV0, error) {
	run := autoprogrammingBridgeGoalRunV0(request, result.Work)
	if issues := orquestacoreworkflow.ValidateOrchestrationRunV0(run); len(issues) > 0 {
		return AutoprogrammingBridgeResultV0{}, fmt.Errorf("autoprogramming goal-first run invalido: %s", issues[0].Error())
	}
	spec := autoprogrammingBridgeGoalSpecForRunV0(result.Work, run)
	if issues := orquestagoal.ValidateGoalWorkSpecV0(spec); len(issues) > 0 {
		return AutoprogrammingBridgeResultV0{}, fmt.Errorf("autoprogramming goal spec invalido: %s", issues[0].Code)
	}
	result.Work.GoalSpecs = []orquestagoal.GoalWorkSpecV0{spec}

	existing, err := ports.RunStore.LoadRunV0(ctx, run.RunID)
	if err != nil && !orquestacionnucleoapp.IsRunNotFoundErrorV0(err) {
		return AutoprogrammingBridgeResultV0{}, err
	}
	if err == nil {
		if err := autoprogrammingBridgeValidateExistingGoalRunV0(existing, run); err != nil {
			return AutoprogrammingBridgeResultV0{}, err
		}
		if state, loaded, err := autoprogrammingBridgeLoadExistingGoalStateV0(ctx, ports, run.RunID); err != nil {
			return AutoprogrammingBridgeResultV0{}, err
		} else if loaded {
			result.Run = existing
			result.GoalState = state
			receipt := state.LaunchReceipt
			result.GoalReceipt = &receipt
			return result, nil
		}
		result.Run = existing
	} else {
		if err := ports.RunStore.SaveRunV0(ctx, run); err != nil {
			return AutoprogrammingBridgeResultV0{}, err
		}
		result.Run = run
	}

	receipt, err := ports.GoalLauncher.LaunchGoalWorkV0(ctx, spec)
	if err != nil {
		return AutoprogrammingBridgeResultV0{}, err
	}
	state, err := autoprogrammingBridgeNewGoalStateV0(run.RunID, spec, receipt, result.Work)
	if err != nil {
		return AutoprogrammingBridgeResultV0{}, err
	}
	if err := ports.GoalStateStore.SaveGoalWorkStateV0(ctx, state); err != nil {
		return AutoprogrammingBridgeResultV0{}, err
	}
	result.GoalState = state
	receipt = state.LaunchReceipt
	result.GoalReceipt = &receipt
	return result, nil
}

func autoprogrammingBridgeGoalRunV0(
	request AutoprogrammingBridgeRequestV0,
	work orquestaautoprogramming.AutoprogrammingProgrammableWorkV0,
) orquestacoreworkflow.OrchestrationRunV0 {
	run := autoprogrammingBridgeRunV0(request, work)
	run.Tasks = nil
	run.FunctionContracts = nil
	return run
}

func autoprogrammingBridgeValidateExistingGoalRunV0(
	existing orquestacoreworkflow.OrchestrationRunV0,
	expected orquestacoreworkflow.OrchestrationRunV0,
) error {
	if err := autoprogrammingBridgeValidateExistingRunV0(existing, expected); err != nil {
		return err
	}
	if len(compactStringsV0(existing.Tasks)) > 0 || len(compactStringsV0(existing.FunctionContracts)) > 0 {
		return fmt.Errorf("autoprogramming goal-first run existente contiene loop legacy: %s", expected.RunID)
	}
	return nil
}

func autoprogrammingBridgeGoalSpecForRunV0(
	work orquestaautoprogramming.AutoprogrammingProgrammableWorkV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) orquestagoal.GoalWorkSpecV0 {
	spec := orquestagoal.NormalizeGoalWorkSpecV0(work.GoalSpecs[0])
	if strings.TrimSpace(spec.RunRef) == "" {
		spec.RunRef = strings.TrimSpace(run.RunID)
	}
	if strings.TrimSpace(spec.RequestRef) == "" {
		spec.RequestRef = strings.TrimSpace(work.RequestRef)
	}
	if strings.TrimSpace(spec.ProjectRef) == "" {
		spec.ProjectRef = strings.TrimSpace(work.ProjectRef)
	}
	return orquestagoal.NormalizeGoalWorkSpecV0(spec)
}

func autoprogrammingBridgeLoadExistingGoalStateV0(
	ctx context.Context,
	ports orquestaappdirectorservice.StartAppDirectorPortsV0,
	runRef string,
) (orquestagoal.GoalWorkStateV0, bool, error) {
	state, err := ports.GoalStateStore.LoadGoalWorkStateV0(ctx, runRef)
	if err != nil {
		return orquestagoal.GoalWorkStateV0{}, false, nil
	}
	state, err = orquestagoal.NewGoalWorkStateV0(state)
	if err != nil {
		return orquestagoal.GoalWorkStateV0{}, false, err
	}
	return state, true, nil
}

func autoprogrammingBridgeNewGoalStateV0(
	runRef string,
	spec orquestagoal.GoalWorkSpecV0,
	receipt orquestagoal.GoalLaunchReceiptV0,
	work orquestaautoprogramming.AutoprogrammingProgrammableWorkV0,
) (orquestagoal.GoalWorkStateV0, error) {
	receipt = orquestagoal.GoalLaunchReceiptV0{
		SchemaVersion:   firstNonEmptyAutoprogrammingStackV0(receipt.SchemaVersion, orquestagoal.GoalWorkLaunchReceiptSchemaV0),
		Status:          firstNonEmptyAutoprogrammingStackV0(receipt.Status, orquestagoal.GoalStatusRunningV0),
		GoalRef:         firstNonEmptyAutoprogrammingStackV0(receipt.GoalRef, spec.GoalRef),
		ExternalGoalRef: firstNonEmptyAutoprogrammingStackV0(receipt.ExternalGoalRef, spec.GoalRef),
		EvidenceRefs:    compactStringsV0(receipt.EvidenceRefs),
		Issues:          append([]orquestagoal.GoalWorkIssueV0(nil), receipt.Issues...),
	}
	state := orquestagoal.GoalWorkStateV0{
		SchemaVersion:   orquestagoal.GoalWorkStateSchemaV0,
		RunRef:          strings.TrimSpace(runRef),
		GoalRef:         strings.TrimSpace(receipt.GoalRef),
		ExternalGoalRef: strings.TrimSpace(receipt.ExternalGoalRef),
		Status:          strings.TrimSpace(receipt.Status),
		Spec:            orquestagoal.NormalizeGoalWorkSpecV0(spec),
		LaunchReceipt:   receipt,
		EvidenceRefs: compactStringsV0(append(
			append([]string{
				"evidence-ref-autoprogramming-goal-first-state-v0",
				"evidence-ref-autoprogramming-goal-migration-" + strings.TrimSpace(work.GoalMigration.Status),
			}, spec.EvidenceRefs...),
			receipt.EvidenceRefs...,
		)),
	}
	return orquestagoal.NewGoalWorkStateV0(state)
}

func autoprogrammingBridgeResultIsGoalFirstV0(result AutoprogrammingBridgeResultV0) bool {
	return strings.TrimSpace(result.GoalState.GoalRef) != "" || result.GoalReceipt != nil
}
