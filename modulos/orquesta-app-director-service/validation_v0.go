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
	if ports.GoalLauncher != nil && ports.GoalStateStore == nil {
		return AppDirectorServiceIssueV0{Field: "ports.goal_state_store"}
	}
	return nil
}
