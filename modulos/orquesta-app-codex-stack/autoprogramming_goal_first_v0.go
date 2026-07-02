package orquestaappcodexstack

import (
	"context"
	"fmt"
	"reflect"
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
	specs := autoprogrammingBridgeGoalSpecsForRunsV0(result.Work)
	result.Work.GoalSpecs = specs
	for _, spec := range specs {
		var err error
		result, err = autoprogrammingBridgeStartSingleGoalFirstV0(ctx, request, result, ports, spec)
		if err != nil {
			return AutoprogrammingBridgeResultV0{}, err
		}
		if !result.Accepted {
			return result, nil
		}
	}
	return autoprogrammingBridgeFinalizeGoalFirstResultV0(result), nil
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

func autoprogrammingBridgeGoalRunForRefV0(
	request AutoprogrammingBridgeRequestV0,
	work orquestaautoprogramming.AutoprogrammingProgrammableWorkV0,
	runRef string,
) orquestacoreworkflow.OrchestrationRunV0 {
	run := autoprogrammingBridgeGoalRunV0(request, work)
	run.RunID = strings.TrimSpace(runRef)
	run.AppSpecRef = "app-spec-ref-autoprogramming-" + autoprogrammingBridgeHashRefV0(run.RunID)
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

func autoprogrammingBridgeGoalSpecsForRunsV0(
	work orquestaautoprogramming.AutoprogrammingProgrammableWorkV0,
) []orquestagoal.GoalWorkSpecV0 {
	specs := make([]orquestagoal.GoalWorkSpecV0, 0, len(work.GoalSpecs))
	for index, spec := range work.GoalSpecs {
		spec = autoprogrammingBridgeGoalSpecForRunRefV0(work, spec, autoprogrammingBridgeGoalRunRefForSpecV0(work, index))
		specs = append(specs, spec)
	}
	if specs == nil {
		return []orquestagoal.GoalWorkSpecV0{}
	}
	return specs
}

func autoprogrammingBridgeGoalSpecForRunRefV0(
	work orquestaautoprogramming.AutoprogrammingProgrammableWorkV0,
	spec orquestagoal.GoalWorkSpecV0,
	runRef string,
) orquestagoal.GoalWorkSpecV0 {
	spec = orquestagoal.NormalizeGoalWorkSpecV0(spec)
	if strings.TrimSpace(spec.RunRef) == "" {
		spec.RunRef = strings.TrimSpace(runRef)
	}
	if strings.TrimSpace(spec.RequestRef) == "" {
		spec.RequestRef = strings.TrimSpace(work.RequestRef)
	}
	if strings.TrimSpace(spec.ProjectRef) == "" {
		spec.ProjectRef = strings.TrimSpace(work.ProjectRef)
	}
	return orquestagoal.NormalizeGoalWorkSpecV0(spec)
}

func autoprogrammingBridgeGoalRunRefForSpecV0(
	work orquestaautoprogramming.AutoprogrammingProgrammableWorkV0,
	index int,
) string {
	if len(work.GoalSpecs) <= 1 {
		return strings.TrimSpace(work.RequestRef)
	}
	return fmt.Sprintf("%s-goal-%02d", strings.TrimSpace(work.RequestRef), index+1)
}

func autoprogrammingBridgeStartSingleGoalFirstV0(
	ctx context.Context,
	request AutoprogrammingBridgeRequestV0,
	result AutoprogrammingBridgeResultV0,
	ports orquestaappdirectorservice.StartAppDirectorPortsV0,
	spec orquestagoal.GoalWorkSpecV0,
) (AutoprogrammingBridgeResultV0, error) {
	if issues := orquestagoal.ValidateGoalWorkSpecV0(spec); len(issues) > 0 {
		return AutoprogrammingBridgeResultV0{}, fmt.Errorf("autoprogramming goal spec invalido: %s", issues[0].Code)
	}
	run := autoprogrammingBridgeGoalRunForRefV0(request, result.Work, spec.RunRef)
	if issues := orquestacoreworkflow.ValidateOrchestrationRunV0(run); len(issues) > 0 {
		return AutoprogrammingBridgeResultV0{}, fmt.Errorf("autoprogramming goal-first run invalido: %s", issues[0].Error())
	}

	existing, err := ports.RunStore.LoadRunV0(ctx, run.RunID)
	if err != nil && !orquestacionnucleoapp.IsRunNotFoundErrorV0(err) {
		return AutoprogrammingBridgeResultV0{}, err
	}
	if err == nil {
		if err := autoprogrammingBridgeValidateExistingGoalRunV0(existing, run); err != nil {
			return AutoprogrammingBridgeResultV0{}, err
		}
		if state, loaded, err := autoprogrammingBridgeLoadExistingGoalStateV0(ctx, ports, run.RunID); err != nil {
			if strings.TrimSpace(result.Run.RunID) == "" {
				result.Run = existing
			}
			result.Accepted = false
			result.Issues = append(result.Issues, autoprogrammingBridgeExistingGoalStateIssueV0(run.RunID))
			return result, nil
		} else if loaded {
			if !autoprogrammingBridgeGoalSpecMatchesStateV0(state.Spec, spec) {
				if strings.TrimSpace(result.Run.RunID) == "" {
					result.Run = existing
				}
				result.Accepted = false
				result.Issues = append(result.Issues, autoprogrammingBridgeExistingGoalSpecMismatchIssueV0(run.RunID))
				return result, nil
			}
			if err := autoprogrammingBridgeSaveGoalRunMarkerV0(ctx, ports, state); err != nil {
				if strings.TrimSpace(result.Run.RunID) == "" {
					result.Run = existing
				}
				result.Accepted = false
				result.Issues = append(result.Issues, autoprogrammingBridgeGoalMarkerSaveIssueV0(run.RunID))
				return result, nil
			}
			result = autoprogrammingBridgeAppendGoalRunV0(result, existing, state)
			return result, nil
		}
	} else {
		if err := ports.RunStore.SaveRunV0(ctx, run); err != nil {
			return AutoprogrammingBridgeResultV0{}, err
		}
	}

	receipt, err := ports.GoalLauncher.LaunchGoalWorkV0(ctx, spec)
	if err != nil {
		state, stateErr := autoprogrammingBridgeBlockedGoalStateV0(run.RunID, spec, receipt, err, result.Work)
		if stateErr == nil {
			stateErr = ports.GoalStateStore.SaveGoalWorkStateV0(ctx, state)
		}
		if stateErr == nil {
			stateErr = autoprogrammingBridgeSaveGoalRunMarkerV0(ctx, ports, state)
		}
		if strings.TrimSpace(result.Run.RunID) == "" {
			result.Run = run
		}
		if stateErr == nil {
			result = autoprogrammingBridgeAppendGoalRunV0(result, run, state)
		}
		result.Accepted = false
		result.Issues = append(result.Issues, autoprogrammingBridgeGoalLaunchIssueV0(
			run.RunID,
			autoprogrammingBridgeGoalLaunchFailureReasonV0(receipt, err),
			stateErr,
		))
		return result, nil
	}
	state, err := autoprogrammingBridgeNewGoalStateV0(run.RunID, spec, receipt, result.Work)
	if err != nil {
		return AutoprogrammingBridgeResultV0{}, err
	}
	if err := ports.GoalStateStore.SaveGoalWorkStateV0(ctx, state); err != nil {
		if strings.TrimSpace(result.Run.RunID) == "" {
			result.Run = run
		}
		result.Accepted = false
		result.Issues = append(result.Issues, autoprogrammingBridgeGoalStateSaveIssueV0(run.RunID))
		return result, nil
	}
	if err := autoprogrammingBridgeSaveGoalRunMarkerV0(ctx, ports, state); err != nil {
		if strings.TrimSpace(result.Run.RunID) == "" {
			result.Run = run
		}
		result.Accepted = false
		result.Issues = append(result.Issues, autoprogrammingBridgeGoalMarkerSaveIssueV0(run.RunID))
		return result, nil
	}
	return autoprogrammingBridgeAppendGoalRunV0(result, run, state), nil
}

func autoprogrammingBridgeGoalSpecMatchesStateV0(
	persisted orquestagoal.GoalWorkSpecV0,
	expected orquestagoal.GoalWorkSpecV0,
) bool {
	return reflect.DeepEqual(
		orquestagoal.NormalizeGoalWorkSpecV0(persisted),
		orquestagoal.NormalizeGoalWorkSpecV0(expected),
	)
}

func autoprogrammingBridgeAppendGoalRunV0(
	result AutoprogrammingBridgeResultV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	state orquestagoal.GoalWorkStateV0,
) AutoprogrammingBridgeResultV0 {
	if strings.TrimSpace(result.Run.RunID) == "" {
		result.Run = run
	}
	receipt := state.LaunchReceipt
	result.GoalStates = append(result.GoalStates, state)
	result.GoalReceipts = append(result.GoalReceipts, receipt)
	return result
}

func autoprogrammingBridgeFinalizeGoalFirstResultV0(
	result AutoprogrammingBridgeResultV0,
) AutoprogrammingBridgeResultV0 {
	if len(result.GoalStates) == 1 {
		result.GoalState = result.GoalStates[0]
		receipt := result.GoalState.LaunchReceipt
		result.GoalReceipt = &receipt
	}
	return result
}

func autoprogrammingBridgeLoadExistingGoalStateV0(
	ctx context.Context,
	ports orquestaappdirectorservice.StartAppDirectorPortsV0,
	runRef string,
) (orquestagoal.GoalWorkStateV0, bool, error) {
	state, err := ports.GoalStateStore.LoadGoalWorkStateV0(ctx, runRef)
	if err != nil {
		return orquestagoal.GoalWorkStateV0{}, false, err
	}
	state, err = orquestagoal.NewGoalWorkStateV0(state)
	if err != nil {
		return orquestagoal.GoalWorkStateV0{}, false, err
	}
	return state, true, nil
}

func autoprogrammingBridgeExistingGoalStateIssueV0(
	runRef string,
) orquestaautoprogramming.AutoprogrammingRequestIssueV0 {
	return orquestaautoprogramming.AutoprogrammingRequestIssueV0{
		Code:    "autoprogramming_goal_state_unavailable_for_existing_run",
		Field:   "goal_state",
		Message: "run goal-first existente sin estado goal cargable; no se relanza ni se cae al loop legacy: " + strings.TrimSpace(runRef),
	}
}

func autoprogrammingBridgeExistingGoalSpecMismatchIssueV0(
	runRef string,
) orquestaautoprogramming.AutoprogrammingRequestIssueV0 {
	return orquestaautoprogramming.AutoprogrammingRequestIssueV0{
		Code:    "autoprogramming_goal_state_spec_mismatch",
		Field:   "goal_state.spec",
		Message: "run goal-first existente apunta a un GoalWorkSpec distinto; no se reutiliza ni se relanza: " + strings.TrimSpace(runRef),
	}
}

func autoprogrammingBridgeGoalLaunchIssueV0(
	runRef string,
	reason string,
	stateErr error,
) orquestaautoprogramming.AutoprogrammingRequestIssueV0 {
	message := "launcher goal-first no pudo lanzar el goal; state bloqueado persistido y no se cae al loop legacy: " + strings.TrimSpace(runRef)
	if reason = strings.TrimSpace(reason); reason != "" {
		message = "launcher goal-first no pudo lanzar el goal (" + reason + "); state bloqueado persistido y no se cae al loop legacy: " + strings.TrimSpace(runRef)
	}
	if stateErr != nil {
		message = "launcher goal-first no pudo lanzar el goal y el state bloqueado no pudo persistirse; no se cae al loop legacy: " + strings.TrimSpace(runRef)
	}
	return orquestaautoprogramming.AutoprogrammingRequestIssueV0{
		Code:    "autoprogramming_goal_launch_failed",
		Field:   "goal_launcher",
		Message: message,
	}
}

func autoprogrammingBridgeGoalStateSaveIssueV0(
	runRef string,
) orquestaautoprogramming.AutoprogrammingRequestIssueV0 {
	return orquestaautoprogramming.AutoprogrammingRequestIssueV0{
		Code:    "autoprogramming_goal_state_save_failed",
		Field:   "goal_state",
		Message: "state goal-first no pudo persistirse; no se relanza automaticamente: " + strings.TrimSpace(runRef),
	}
}

func autoprogrammingBridgeGoalMarkerSaveIssueV0(
	runRef string,
) orquestaautoprogramming.AutoprogrammingRequestIssueV0 {
	return orquestaautoprogramming.AutoprogrammingRequestIssueV0{
		Code:    "autoprogramming_goal_marker_save_failed",
		Field:   "goal_first_run_marker",
		Message: "marker goal-first no pudo persistirse; no se declara aceptado ni se cae al loop legacy: " + strings.TrimSpace(runRef),
	}
}

func autoprogrammingBridgeNewGoalStateV0(
	runRef string,
	spec orquestagoal.GoalWorkSpecV0,
	receipt orquestagoal.GoalLaunchReceiptV0,
	work orquestaautoprogramming.AutoprogrammingProgrammableWorkV0,
) (orquestagoal.GoalWorkStateV0, error) {
	return orquestagoal.NewGoalWorkStateFromLaunchV0(orquestagoal.GoalWorkStateFromLaunchRequestV0{
		RunRef:        runRef,
		Spec:          spec,
		LaunchReceipt: receipt,
		EvidenceRefs: compactStringsV0([]string{
			"evidence-ref-autoprogramming-goal-first-state-v0",
			"evidence-ref-autoprogramming-goal-migration-" + strings.TrimSpace(work.GoalMigration.Status),
		}),
	})
}

func autoprogrammingBridgeSaveGoalRunMarkerV0(
	ctx context.Context,
	ports orquestaappdirectorservice.StartAppDirectorPortsV0,
	state orquestagoal.GoalWorkStateV0,
) error {
	if ports.GoalFirstRunMarkerStore == nil {
		return nil
	}
	marker, err := autoprogrammingBridgeGoalRunMarkerFromStateV0(state)
	if err != nil {
		return err
	}
	return ports.GoalFirstRunMarkerStore.SaveGoalWorkRunMarkerV0(ctx, marker)
}

func autoprogrammingBridgeGoalRunMarkerFromStateV0(
	state orquestagoal.GoalWorkStateV0,
) (orquestagoal.GoalWorkRunMarkerV0, error) {
	state, err := orquestagoal.NewGoalWorkStateV0(state)
	if err != nil {
		return orquestagoal.GoalWorkRunMarkerV0{}, err
	}
	spec := orquestagoal.NormalizeGoalWorkSpecV0(state.Spec)
	receipt := orquestagoal.NormalizeGoalLaunchReceiptV0(state.LaunchReceipt)
	marker := orquestagoal.GoalWorkRunMarkerV0{
		SchemaVersion:   orquestagoal.GoalWorkRunMarkerSchemaV0,
		RunRef:          strings.TrimSpace(state.RunRef),
		GoalRef:         autoprogrammingBridgeFirstGoalValueV0(receipt.GoalRef, state.GoalRef, spec.GoalRef),
		ExternalGoalRef: strings.TrimSpace(receipt.ExternalGoalRef),
		DirectorKind:    orquestagoal.GoalDirectorKindCodexGoalV0,
		Status:          strings.TrimSpace(state.Status),
		Spec:            &spec,
		LaunchReceipt:   &receipt,
		EvidenceRefs: compactStringsV0(append(
			[]string{"evidence-ref-autoprogramming-goal-first-run-marker-v0"},
			append(state.EvidenceRefs, receipt.EvidenceRefs...)...,
		)),
	}
	return orquestagoal.NewGoalWorkRunMarkerV0(marker)
}

func autoprogrammingBridgeFirstGoalValueV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func autoprogrammingBridgeBlockedGoalStateV0(
	runRef string,
	spec orquestagoal.GoalWorkSpecV0,
	receipt orquestagoal.GoalLaunchReceiptV0,
	err error,
	work orquestaautoprogramming.AutoprogrammingProgrammableWorkV0,
) (orquestagoal.GoalWorkStateV0, error) {
	reason := autoprogrammingBridgeGoalLaunchFailureReasonV0(receipt, err)
	receipt = orquestagoal.NormalizeGoalLaunchReceiptV0(receipt)
	receipt.Status = orquestagoal.GoalStatusInvalidV0
	if len(receipt.Issues) == 0 {
		receipt.Issues = []orquestagoal.GoalWorkIssueV0{{
			Code:  reason,
			Field: "goal_launcher",
		}}
	}
	receipt.EvidenceRefs = compactStringsV0(append(
		[]string{"evidence-ref-autoprogramming-goal-launch-failed"},
		receipt.EvidenceRefs...,
	))
	return orquestagoal.NewGoalWorkStateFromLaunchV0(orquestagoal.GoalWorkStateFromLaunchRequestV0{
		RunRef:        runRef,
		Spec:          spec,
		LaunchReceipt: receipt,
		EvidenceRefs: compactStringsV0([]string{
			"evidence-ref-autoprogramming-goal-first-launch-failed-state-v0",
			"evidence-ref-autoprogramming-goal-migration-" + strings.TrimSpace(work.GoalMigration.Status),
		}),
	})
}

func autoprogrammingBridgeGoalLaunchFailureReasonV0(
	receipt orquestagoal.GoalLaunchReceiptV0,
	err error,
) string {
	for _, issue := range receipt.Issues {
		if code := strings.TrimSpace(issue.Code); code != "" {
			return code
		}
	}
	message := ""
	if err != nil {
		message = strings.TrimSpace(err.Error())
	}
	if code := externalWorkGoalFirstKnownLaunchFailureReasonV0(message); code != "" {
		return code
	}
	return "goal_launcher_error"
}

func autoprogrammingBridgeResultIsGoalFirstV0(result AutoprogrammingBridgeResultV0) bool {
	return strings.TrimSpace(result.GoalState.GoalRef) != "" ||
		result.GoalReceipt != nil ||
		len(result.GoalStates) > 0 ||
		len(result.GoalReceipts) > 0
}
