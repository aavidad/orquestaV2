package orquestaappcodexstack

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaexternalworkrun "orquesta/modulos/orquesta-external-work-run"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

type CodexStackExternalWorkGoalFirstExecutorV0 struct {
	Legacy                  orquestamcp.MCPTransportExternalWorkRunExecutorV0
	Ports                   orquestaappdirectorservice.StartAppDirectorPortsV0
	Config                  orquestaexternalworkrun.StartExternalWorkRunConfigV0
	AppChangeStore          orquestaappchange.AppChangeRecordStorePortV0
	AllowLegacyDirectorLoop bool
}

var _ orquestamcp.MCPTransportExternalWorkRunExecutorV0 = CodexStackExternalWorkGoalFirstExecutorV0{}

func NewCodexStackExternalWorkGoalFirstExecutorV0(
	legacy orquestamcp.MCPTransportExternalWorkRunExecutorV0,
	ports orquestaappdirectorservice.StartAppDirectorPortsV0,
	config orquestaexternalworkrun.StartExternalWorkRunConfigV0,
	appChangeStore orquestaappchange.AppChangeRecordStorePortV0,
	allowLegacyDirectorLoop bool,
) CodexStackExternalWorkGoalFirstExecutorV0 {
	return CodexStackExternalWorkGoalFirstExecutorV0{
		Legacy:                  legacy,
		Ports:                   ports,
		Config:                  config,
		AppChangeStore:          appChangeStore,
		AllowLegacyDirectorLoop: allowLegacyDirectorLoop,
	}
}

func (executor CodexStackExternalWorkGoalFirstExecutorV0) Execute(
	ctx context.Context,
	input orquestamcp.MCPExternalWorkRunToolInputV0,
) (orquestamcp.MCPExternalWorkRunToolResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if issues := externalWorkGoalFirstValidateInputV0(input); len(issues) > 0 {
		return orquestamcp.MCPExternalWorkRunToolResultV0{
			Estado:        orquestamcp.MCPExternalWorkRunEstadoErrorV0,
			RequestID:     strings.TrimSpace(input.RequestID),
			CorrelationID: firstExternalWorkGoalFirstValueV0(input.CorrelationID, input.RequestID),
			Errores:       issues,
		}, nil
	}
	if externalWorkLegacyDirectorLoopRequestedV0(input) {
		if !executor.AllowLegacyDirectorLoop {
			return externalWorkLegacyDirectorLoopOptInRequiredResultV0(input), nil
		}
		return executor.Legacy.Execute(ctx, input)
	}
	if !externalWorkGoalFirstBackendAvailableV0(executor.Ports) {
		if executor.AllowLegacyDirectorLoop {
			return externalWorkLegacyDirectorModeRequiredResultV0(input), nil
		}
		return externalWorkGoalFirstBackendRequiredResultV0(input), nil
	}
	request := orquestaexternalworkrun.PrepareStartExternalWorkRunRequestV0(
		externalWorkGoalFirstRequestFromMCPV0(input),
		executor.Config,
	)
	spec, issues := orquestaexternalworkrun.BuildExternalWorkGoalWorkSpecV0(request, executor.Config)
	if len(issues) > 0 {
		return externalWorkGoalFirstInputErrorResultV0(request, issues), nil
	}
	spec.DirectorKind = orquestagoal.GoalDirectorKindCodexGoalV0
	spec = orquestagoal.NormalizeGoalWorkSpecV0(spec)
	if issues := orquestagoal.ValidateGoalWorkSpecV0(spec); len(issues) > 0 {
		return externalWorkGoalFirstGoalSpecErrorResultV0(request, issues), nil
	}
	run := externalWorkGoalFirstRunV0(request, spec)
	if validationIssues := orquestacoreworkflow.ValidateOrchestrationRunV0(run); len(validationIssues) > 0 {
		return orquestamcp.MCPExternalWorkRunToolResultV0{}, fmt.Errorf("external-work goal-first run invalido: %s", validationIssues[0].Error())
	}
	if result, handled, err := executor.externalWorkGoalFirstExistingRunResultV0(ctx, input, request, spec, run); handled || err != nil {
		return result, err
	}
	if err := executor.Ports.RunStore.SaveRunV0(ctx, run); err != nil {
		return orquestamcp.MCPExternalWorkRunToolResultV0{}, err
	}
	if err := externalWorkGoalFirstSaveAppChangeRecordV0(ctx, executor.AppChangeStore, request); err != nil {
		return orquestamcp.MCPExternalWorkRunToolResultV0{}, err
	}
	receipt, err := executor.Ports.GoalLauncher.LaunchGoalWorkV0(ctx, spec)
	if err != nil {
		return externalWorkGoalFirstIssueResultV0(request, "external_work_goal_launch_failed", "goal_launcher"), nil
	}
	state, err := externalWorkGoalFirstNewGoalStateV0(run.RunID, spec, receipt)
	if err != nil {
		return orquestamcp.MCPExternalWorkRunToolResultV0{}, err
	}
	if err := executor.Ports.GoalStateStore.SaveGoalWorkStateV0(ctx, state); err != nil {
		return externalWorkGoalFirstIssueResultV0(request, "external_work_goal_state_save_failed", "goal_state"), nil
	}
	return externalWorkGoalFirstResultV0(request, state), nil
}

func (executor CodexStackExternalWorkGoalFirstExecutorV0) externalWorkGoalFirstExistingRunResultV0(
	ctx context.Context,
	input orquestamcp.MCPExternalWorkRunToolInputV0,
	request orquestaexternalworkrun.StartExternalWorkRunRequestV0,
	spec orquestagoal.GoalWorkSpecV0,
	expected orquestacoreworkflow.OrchestrationRunV0,
) (orquestamcp.MCPExternalWorkRunToolResultV0, bool, error) {
	existing, err := executor.Ports.RunStore.LoadRunV0(ctx, expected.RunID)
	if err != nil {
		if orquestacionnucleoapp.IsRunNotFoundErrorV0(err) {
			return orquestamcp.MCPExternalWorkRunToolResultV0{}, false, nil
		}
		return orquestamcp.MCPExternalWorkRunToolResultV0{}, true, err
	}
	if externalWorkGoalFirstRunLooksLegacyV0(existing) {
		if !executor.AllowLegacyDirectorLoop {
			return externalWorkGoalFirstExistingLegacyRunResultV0(request), true, nil
		}
		if !externalWorkLegacyDirectorLoopRequestedV0(input) {
			return externalWorkLegacyDirectorModeRequiredResultV0(input), true, nil
		}
		result, err := executor.Legacy.Execute(ctx, input)
		return result, true, err
	}
	if strings.TrimSpace(existing.ProjectRef) != strings.TrimSpace(expected.ProjectRef) ||
		strings.TrimSpace(existing.AppSpecRef) != strings.TrimSpace(expected.AppSpecRef) {
		return externalWorkGoalFirstIssueResultV0(request, "external_work_goal_existing_run_conflict", "run_ref"), true, nil
	}
	state, err := executor.Ports.GoalStateStore.LoadGoalWorkStateV0(ctx, expected.RunID)
	if err != nil {
		return externalWorkGoalFirstIssueResultV0(request, "external_work_goal_state_unavailable_for_existing_run", "goal_state"), true, nil
	}
	state, err = orquestagoal.NewGoalWorkStateV0(state)
	if err != nil {
		return externalWorkGoalFirstIssueResultV0(request, "external_work_goal_state_invalid_for_existing_run", "goal_state"), true, nil
	}
	if !reflect.DeepEqual(orquestagoal.NormalizeGoalWorkSpecV0(state.Spec), orquestagoal.NormalizeGoalWorkSpecV0(spec)) {
		return externalWorkGoalFirstIssueResultV0(request, "external_work_goal_state_spec_mismatch", "goal_state.spec"), true, nil
	}
	return externalWorkGoalFirstResultV0(request, state), true, nil
}

func externalWorkGoalFirstValidateInputV0(
	input orquestamcp.MCPExternalWorkRunToolInputV0,
) []orquestamcp.MCPExternalWorkRunIssueV0 {
	if externalWorkGoalFirstInputHasAppChangeV0(input.ExternalWorkRunRequest.AppChangeRequest) &&
		externalWorkGoalFirstInputHasAppChangeV0(input.AppChangeRequest) {
		return []orquestamcp.MCPExternalWorkRunIssueV0{{
			Code:  orquestamcp.MCPExternalWorkRunInputAmbiguousV0,
			Field: "app_change_request",
		}}
	}
	return nil
}

func externalWorkLegacyDirectorLoopRequestedV0(
	input orquestamcp.MCPExternalWorkRunToolInputV0,
) bool {
	return strings.TrimSpace(input.DirectorExecutionMode) == orquestamcp.MCPExternalWorkRunDirectorExecutionModeLegacyLoopV0
}

func externalWorkGoalFirstRequestFromMCPV0(
	input orquestamcp.MCPExternalWorkRunToolInputV0,
) orquestaexternalworkrun.StartExternalWorkRunRequestV0 {
	request := input.ExternalWorkRunRequest
	if request.AppChangeRequest.ChangeRef == "" && input.AppChangeRequest.ChangeRef != "" {
		request.AppChangeRequest = input.AppChangeRequest
	}
	if strings.TrimSpace(request.RequestID) == "" {
		request.RequestID = strings.TrimSpace(input.RequestID)
	}
	if strings.TrimSpace(request.CorrelationID) == "" {
		request.CorrelationID = strings.TrimSpace(input.CorrelationID)
	}
	return request
}

func externalWorkGoalFirstInputHasAppChangeV0(
	request orquestaappchange.AppChangeRequestV0,
) bool {
	return strings.TrimSpace(request.RunRef) != "" ||
		strings.TrimSpace(request.AppRef) != "" ||
		strings.TrimSpace(request.ChangeRef) != "" ||
		strings.TrimSpace(request.UserIntent) != "" ||
		request.ExternalWork != nil
}

func externalWorkGoalFirstBackendAvailableV0(
	ports orquestaappdirectorservice.StartAppDirectorPortsV0,
) bool {
	return ports.GoalLauncher != nil &&
		ports.GoalObserver != nil &&
		ports.GoalClosureValidator != nil &&
		ports.GoalStateStore != nil
}

func externalWorkGoalFirstBackendRequiredResultV0(
	input orquestamcp.MCPExternalWorkRunToolInputV0,
) orquestamcp.MCPExternalWorkRunToolResultV0 {
	return orquestamcp.MCPExternalWorkRunToolResultV0{
		Estado:                orquestamcp.MCPExternalWorkRunEstadoErrorV0,
		RoutePolicy:           orquestamcp.MCPExternalWorkRunRoutePolicyGoalFirstV0,
		DirectorExecutionMode: orquestamcp.MCPExternalWorkRunDirectorExecutionModeGoalFirstV0,
		RequestID:             strings.TrimSpace(firstExternalWorkGoalFirstValueV0(input.RequestID, input.ExternalWorkRunRequest.RequestID)),
		CorrelationID: strings.TrimSpace(firstExternalWorkGoalFirstValueV0(
			input.CorrelationID,
			input.ExternalWorkRunRequest.CorrelationID,
			input.RequestID,
		)),
		NextActions: []string{
			orquestamcp.MCPExternalWorkRunNextActionConfigureGoalBackendV0,
			orquestamcp.MCPExternalWorkRunNextActionDoNotFallbackLegacyV0,
		},
		Errores: []orquestamcp.MCPExternalWorkRunIssueV0{{
			Code:  orquestamcp.MCPExternalWorkRunGoalBackendRequiredV0,
			Field: "codex_goal_backend",
		}},
	}
}

func externalWorkLegacyDirectorModeRequiredResultV0(
	input orquestamcp.MCPExternalWorkRunToolInputV0,
) orquestamcp.MCPExternalWorkRunToolResultV0 {
	request := externalWorkGoalFirstRequestFromMCPV0(input)
	return orquestamcp.MCPExternalWorkRunToolResultV0{
		Estado:                orquestamcp.MCPExternalWorkRunEstadoErrorV0,
		RoutePolicy:           orquestamcp.MCPExternalWorkRunRoutePolicyGoalFirstV0,
		DirectorExecutionMode: orquestamcp.MCPExternalWorkRunDirectorExecutionModeGoalFirstV0,
		RequestID:             strings.TrimSpace(firstExternalWorkGoalFirstValueV0(input.RequestID, request.RequestID)),
		CorrelationID:         strings.TrimSpace(firstExternalWorkGoalFirstValueV0(input.CorrelationID, request.CorrelationID, input.RequestID)),
		RunRef:                strings.TrimSpace(request.RunRef),
		ProjectRef:            strings.TrimSpace(request.ProjectRef),
		AppRef:                strings.TrimSpace(request.AppChangeRequest.AppRef),
		ChangeRef:             strings.TrimSpace(request.AppChangeRequest.ChangeRef),
		NextActions: []string{
			orquestamcp.MCPExternalWorkRunNextActionConfigureGoalBackendV0,
			orquestamcp.MCPExternalWorkRunNextActionDoNotFallbackLegacyV0,
			orquestamcp.MCPExternalWorkRunNextActionEnableLegacyOptInV0,
		},
		Errores: []orquestamcp.MCPExternalWorkRunIssueV0{{
			Code:  orquestamcp.MCPExternalWorkRunLegacyDirectorModeRequiredV0,
			Field: "director_execution_mode",
		}},
	}
}

func externalWorkLegacyDirectorLoopOptInRequiredResultV0(
	input orquestamcp.MCPExternalWorkRunToolInputV0,
) orquestamcp.MCPExternalWorkRunToolResultV0 {
	request := externalWorkGoalFirstRequestFromMCPV0(input)
	return orquestamcp.MCPExternalWorkRunToolResultV0{
		Estado:                orquestamcp.MCPExternalWorkRunEstadoErrorV0,
		RoutePolicy:           orquestamcp.MCPExternalWorkRunRoutePolicyGoalFirstV0,
		DirectorExecutionMode: orquestamcp.MCPExternalWorkRunDirectorExecutionModeGoalFirstV0,
		RequestID:             strings.TrimSpace(firstExternalWorkGoalFirstValueV0(input.RequestID, request.RequestID)),
		CorrelationID:         strings.TrimSpace(firstExternalWorkGoalFirstValueV0(input.CorrelationID, request.CorrelationID, input.RequestID)),
		RunRef:                strings.TrimSpace(request.RunRef),
		ProjectRef:            strings.TrimSpace(request.ProjectRef),
		AppRef:                strings.TrimSpace(request.AppChangeRequest.AppRef),
		ChangeRef:             strings.TrimSpace(request.AppChangeRequest.ChangeRef),
		NextActions: []string{
			orquestamcp.MCPExternalWorkRunNextActionConfigureGoalBackendV0,
			orquestamcp.MCPExternalWorkRunNextActionDoNotFallbackLegacyV0,
			orquestamcp.MCPExternalWorkRunNextActionEnableLegacyOptInV0,
		},
		Errores: []orquestamcp.MCPExternalWorkRunIssueV0{{
			Code:  orquestamcp.MCPExternalWorkRunLegacyDirectorLoopOptInRequiredV0,
			Field: "legacy_director_loop_opt_in",
		}},
	}
}

func externalWorkGoalFirstRunV0(
	request orquestaexternalworkrun.StartExternalWorkRunRequestV0,
	spec orquestagoal.GoalWorkSpecV0,
) orquestacoreworkflow.OrchestrationRunV0 {
	phases := orquestacoreworkflow.OrchestrationPhaseCatalogV0()
	for index := range phases {
		if phases[index].ID != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
			continue
		}
		phases[index].Status = orquestacoreworkflow.OrchestrationPhaseStatusActiveV0
		phases[index].OpenedAt = request.OccurredAt
	}
	return orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         spec.RunRef,
		ProjectRef:    spec.ProjectRef,
		AppSpecRef:    request.AppSpecRef,
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Phases:        phases,
	}
}

func externalWorkGoalFirstRunLooksLegacyV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) bool {
	return len(compactStringsV0(run.Tasks)) > 0 ||
		len(compactStringsV0(run.FunctionContracts)) > 0 ||
		len(compactStringsV0(run.DirectorQuestions)) > 0
}

func externalWorkGoalFirstExistingLegacyRunResultV0(
	request orquestaexternalworkrun.StartExternalWorkRunRequestV0,
) orquestamcp.MCPExternalWorkRunToolResultV0 {
	return orquestamcp.MCPExternalWorkRunToolResultV0{
		Estado:                orquestamcp.MCPExternalWorkRunEstadoErrorV0,
		RoutePolicy:           orquestamcp.MCPExternalWorkRunRoutePolicyGoalFirstV0,
		DirectorExecutionMode: orquestamcp.MCPExternalWorkRunDirectorExecutionModeGoalFirstV0,
		RequestID:             strings.TrimSpace(request.RequestID),
		CorrelationID:         strings.TrimSpace(request.CorrelationID),
		RunRef:                strings.TrimSpace(request.RunRef),
		ProjectRef:            strings.TrimSpace(request.ProjectRef),
		AppRef:                strings.TrimSpace(request.AppChangeRequest.AppRef),
		ChangeRef:             strings.TrimSpace(request.AppChangeRequest.ChangeRef),
		NextActions: []string{
			orquestamcp.MCPExternalWorkRunNextActionEnableLegacyOptInV0,
			orquestamcp.MCPExternalWorkRunNextActionMigrateGoalFirstV0,
			orquestamcp.MCPExternalWorkRunNextActionDoNotFallbackLegacyV0,
		},
		Errores: []orquestamcp.MCPExternalWorkRunIssueV0{{
			Code:  orquestamcp.MCPExternalWorkRunExistingLegacyRunRequiresOptInV0,
			Field: "run_ref",
		}},
	}
}

func externalWorkGoalFirstSaveAppChangeRecordV0(
	ctx context.Context,
	store orquestaappchange.AppChangeRecordStorePortV0,
	request orquestaexternalworkrun.StartExternalWorkRunRequestV0,
) error {
	if store == nil {
		return nil
	}
	return store.SaveAppChangeRequestV0(ctx, orquestaappchange.AppChangeRecordV0{
		Request:     request.AppChangeRequest,
		ReceivedAt:  request.OccurredAt,
		RequestedBy: request.RequestedBy,
	})
}

func externalWorkGoalFirstNewGoalStateV0(
	runRef string,
	spec orquestagoal.GoalWorkSpecV0,
	receipt orquestagoal.GoalLaunchReceiptV0,
) (orquestagoal.GoalWorkStateV0, error) {
	return orquestagoal.NewGoalWorkStateFromLaunchV0(orquestagoal.GoalWorkStateFromLaunchRequestV0{
		RunRef:        runRef,
		Spec:          spec,
		LaunchReceipt: receipt,
		EvidenceRefs:  []string{"evidence-ref-external-work-goal-first-state-v0"},
	})
}

func externalWorkGoalFirstResultV0(
	request orquestaexternalworkrun.StartExternalWorkRunRequestV0,
	state orquestagoal.GoalWorkStateV0,
) orquestamcp.MCPExternalWorkRunToolResultV0 {
	return orquestamcp.MCPExternalWorkRunToolResultV0{
		Estado:                orquestamcp.MCPExternalWorkRunEstadoOKV0,
		RoutePolicy:           orquestamcp.MCPExternalWorkRunRoutePolicyGoalFirstV0,
		DirectorExecutionMode: orquestamcp.MCPExternalWorkRunDirectorExecutionModeGoalFirstV0,
		RequestID:             strings.TrimSpace(request.RequestID),
		CorrelationID:         strings.TrimSpace(request.CorrelationID),
		RunRef:                strings.TrimSpace(request.RunRef),
		ProjectRef:            strings.TrimSpace(request.ProjectRef),
		AppRef:                strings.TrimSpace(request.AppChangeRequest.AppRef),
		ChangeRef:             strings.TrimSpace(request.AppChangeRequest.ChangeRef),
		GoalRef:               strings.TrimSpace(state.GoalRef),
		ExternalGoalRef:       strings.TrimSpace(state.ExternalGoalRef),
		EvidenceRefs:          compactStringsV0(state.EvidenceRefs),
		NextActions:           []string{orquestamcp.MCPExternalWorkRunNextActionObserveGoalV0},
	}
}

func externalWorkGoalFirstInputErrorResultV0(
	request orquestaexternalworkrun.StartExternalWorkRunRequestV0,
	issues []orquestaexternalworkrun.ExternalWorkRunIssueV0,
) orquestamcp.MCPExternalWorkRunToolResultV0 {
	return orquestamcp.MCPExternalWorkRunToolResultV0{
		Estado:        orquestamcp.MCPExternalWorkRunEstadoErrorV0,
		RequestID:     strings.TrimSpace(request.RequestID),
		CorrelationID: strings.TrimSpace(request.CorrelationID),
		RunRef:        strings.TrimSpace(request.RunRef),
		ProjectRef:    strings.TrimSpace(request.ProjectRef),
		AppRef:        strings.TrimSpace(request.AppChangeRequest.AppRef),
		ChangeRef:     strings.TrimSpace(request.AppChangeRequest.ChangeRef),
		Errores:       externalWorkGoalFirstIssuesV0(issues),
	}
}

func externalWorkGoalFirstGoalSpecErrorResultV0(
	request orquestaexternalworkrun.StartExternalWorkRunRequestV0,
	issues []orquestagoal.GoalWorkIssueV0,
) orquestamcp.MCPExternalWorkRunToolResultV0 {
	mapped := make([]orquestamcp.MCPExternalWorkRunIssueV0, 0, len(issues))
	for _, issue := range issues {
		code := "external_work_goal_spec_invalid"
		if strings.TrimSpace(issue.Code) != "" {
			code += ":" + strings.TrimSpace(issue.Code)
		}
		field := "goal_spec"
		if strings.TrimSpace(issue.Field) != "" {
			field += "." + strings.TrimSpace(issue.Field)
		}
		mapped = append(mapped, orquestamcp.MCPExternalWorkRunIssueV0{Code: code, Field: field})
	}
	return orquestamcp.MCPExternalWorkRunToolResultV0{
		Estado:        orquestamcp.MCPExternalWorkRunEstadoErrorV0,
		RequestID:     strings.TrimSpace(request.RequestID),
		CorrelationID: strings.TrimSpace(request.CorrelationID),
		RunRef:        strings.TrimSpace(request.RunRef),
		ProjectRef:    strings.TrimSpace(request.ProjectRef),
		AppRef:        strings.TrimSpace(request.AppChangeRequest.AppRef),
		ChangeRef:     strings.TrimSpace(request.AppChangeRequest.ChangeRef),
		Errores:       mapped,
	}
}

func externalWorkGoalFirstIssueResultV0(
	request orquestaexternalworkrun.StartExternalWorkRunRequestV0,
	code string,
	field string,
) orquestamcp.MCPExternalWorkRunToolResultV0 {
	return orquestamcp.MCPExternalWorkRunToolResultV0{
		Estado:        orquestamcp.MCPExternalWorkRunEstadoErrorV0,
		RequestID:     strings.TrimSpace(request.RequestID),
		CorrelationID: strings.TrimSpace(request.CorrelationID),
		RunRef:        strings.TrimSpace(request.RunRef),
		ProjectRef:    strings.TrimSpace(request.ProjectRef),
		AppRef:        strings.TrimSpace(request.AppChangeRequest.AppRef),
		ChangeRef:     strings.TrimSpace(request.AppChangeRequest.ChangeRef),
		Errores: []orquestamcp.MCPExternalWorkRunIssueV0{{
			Code:  strings.TrimSpace(code),
			Field: strings.TrimSpace(field),
		}},
	}
}

func externalWorkGoalFirstIssuesV0(
	issues []orquestaexternalworkrun.ExternalWorkRunIssueV0,
) []orquestamcp.MCPExternalWorkRunIssueV0 {
	out := make([]orquestamcp.MCPExternalWorkRunIssueV0, 0, len(issues))
	for _, issue := range issues {
		out = append(out, orquestamcp.MCPExternalWorkRunIssueV0{
			Code:  strings.TrimSpace(issue.Code),
			Field: strings.TrimSpace(issue.Field),
		})
	}
	if out == nil {
		return nil
	}
	return out
}

func firstExternalWorkGoalFirstValueV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
