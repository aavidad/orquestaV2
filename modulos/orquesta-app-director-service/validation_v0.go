package orquestaappdirectorservice

import "strings"

func validateStartAppDirectorRequestV0(
	request StartAppDirectorRequestV0,
) error {
	if strings.TrimSpace(request.AppSpecRequest.SchemaVersion) == "" {
		return AppDirectorServiceIssueV0{Field: "app_spec_request.schema_version"}
	}
	if strings.TrimSpace(request.AppSpecRequest.RequestID) == "" {
		return AppDirectorServiceIssueV0{Field: "app_spec_request.request_id"}
	}
	if continueHasOperationalDirectorPlanV0(request.OperationalDirectorPlan) {
		if strings.TrimSpace(request.OperationalDirectorPlan.RunRef) != "" &&
			strings.TrimSpace(request.RunRef) != "" &&
			strings.TrimSpace(request.OperationalDirectorPlan.RunRef) != strings.TrimSpace(request.RunRef) {
			return AppDirectorServiceIssueV0{Field: "operational_director_plan.run_ref"}
		}
		if len(startAppDirectorOperationalFunctionContractRefsV0(request)) == 0 {
			return AppDirectorServiceIssueV0{Field: "operational_director_function_contract_refs"}
		}
		if !startAppDirectorOperationalTargetPhaseAllowedV0(startAppDirectorOperationalTargetPhaseIDV0(request)) {
			return AppDirectorServiceIssueV0{Field: "operational_director_target_phase_id"}
		}
	}
	return nil
}

func validateStartAppDirectorPortsV0(
	ports StartAppDirectorPortsV0,
) error {
	if ports.RunStore == nil {
		return AppDirectorServiceIssueV0{Field: "ports.run_store"}
	}
	if ports.EventSink == nil {
		return AppDirectorServiceIssueV0{Field: "ports.event_sink"}
	}
	if ports.OutboxLedger == nil {
		return AppDirectorServiceIssueV0{Field: "ports.outbox_ledger"}
	}
	if len(ports.Dispatchers) == 0 && len(ports.BatchDispatchers) == 0 {
		return AppDirectorServiceIssueV0{Field: "ports.dispatchers"}
	}
	if appDirectorGoalPortsConfiguredForLaunchV0(ports) && !appDirectorGoalPortsReadyV0(ports) {
		return AppDirectorServiceIssueV0{Field: appDirectorMissingGoalPortFieldV0(ports)}
	}
	return nil
}

func appDirectorGoalPortsConfiguredForLaunchV0(
	ports StartAppDirectorPortsV0,
) bool {
	return ports.GoalLauncher != nil || ports.GoalObserver != nil
}

func appDirectorGoalPortsReadyV0(
	ports StartAppDirectorPortsV0,
) bool {
	return ports.GoalLauncher != nil &&
		ports.GoalObserver != nil &&
		ports.GoalClosureValidator != nil &&
		ports.GoalStateStore != nil
}

func appDirectorMissingGoalPortFieldV0(
	ports StartAppDirectorPortsV0,
) string {
	switch {
	case ports.GoalLauncher == nil:
		return "ports.goal_launcher"
	case ports.GoalObserver == nil:
		return "ports.goal_observer"
	case ports.GoalClosureValidator == nil:
		return "ports.goal_closure_validator"
	case ports.GoalStateStore == nil:
		return "ports.goal_state_store"
	default:
		return "ports.goal_bundle"
	}
}
