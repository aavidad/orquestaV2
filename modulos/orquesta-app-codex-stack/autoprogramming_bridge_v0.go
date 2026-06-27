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

const AutoprogrammingBridgeResultSchemaVersionV0 = "autoprogramming_bridge_result.v0"

type AutoprogrammingBridgeRequestV0 struct {
	Request              orquestaautoprogramming.AutoprogrammingRequestV0 `json:"request"`
	OccurredAt           string                                           `json:"occurred_at,omitempty"`
	CorrelationID        string                                           `json:"correlation_id,omitempty"`
	RequestedBy          string                                           `json:"requested_by,omitempty"`
	MaxBursts            int                                              `json:"max_bursts,omitempty"`
	MaxStepsPerBurst     int                                              `json:"max_steps_per_burst,omitempty"`
	MaxDispatchesPerWait int                                              `json:"max_dispatches_per_wait,omitempty"`
	MaxCommands          int                                              `json:"max_commands,omitempty"`
	MaxOutboxPerCycle    int                                              `json:"max_outbox_per_cycle,omitempty"`
}

type AutoprogrammingBridgeResultV0 struct {
	SchemaVersion string                                                    `json:"schema_version"`
	Accepted      bool                                                      `json:"accepted"`
	Work          orquestaautoprogramming.AutoprogrammingProgrammableWorkV0 `json:"work"`
	Run           orquestacoreworkflow.OrchestrationRunV0                   `json:"run,omitempty"`
	Tasks         []orquestacoreworkflow.WorkflowTaskV0                     `json:"tasks,omitempty"`
	WaitAgentRefs []string                                                  `json:"wait_agent_refs,omitempty"`
	Continue      orquestaappdirectorservice.ContinueAppDirectorRequestV0   `json:"continue,omitempty"`
	GoalState     orquestagoal.GoalWorkStateV0                              `json:"goal_state,omitempty"`
	GoalReceipt   *orquestagoal.GoalLaunchReceiptV0                         `json:"goal_receipt,omitempty"`
	GoalStates    []orquestagoal.GoalWorkStateV0                            `json:"goal_states,omitempty"`
	GoalReceipts  []orquestagoal.GoalLaunchReceiptV0                        `json:"goal_receipts,omitempty"`
	Issues        []orquestaautoprogramming.AutoprogrammingRequestIssueV0   `json:"issues,omitempty"`
}

func PrepareAutoprogrammingRunV0(
	ctx context.Context,
	request AutoprogrammingBridgeRequestV0,
	ports orquestaappdirectorservice.StartAppDirectorPortsV0,
) (AutoprogrammingBridgeResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	request = normalizeAutoprogrammingBridgeRequestV0(request)
	if err := validateAutoprogrammingBridgePortsV0(ports); err != nil {
		return AutoprogrammingBridgeResultV0{}, err
	}
	work := orquestaautoprogramming.BuildAutoprogrammingProgrammableWorkV0(request.Request)
	return prepareAutoprogrammingRunWithWorkV0(ctx, request, ports, work)
}

func prepareAutoprogrammingRunWithWorkV0(
	ctx context.Context,
	request AutoprogrammingBridgeRequestV0,
	ports orquestaappdirectorservice.StartAppDirectorPortsV0,
	work orquestaautoprogramming.AutoprogrammingProgrammableWorkResultV0,
) (AutoprogrammingBridgeResultV0, error) {
	if work.Accepted {
		tasks, issue := autoprogrammingBridgeOperationalTasksV0(work.Work.Tasks)
		if issue.Code != "" {
			work.Accepted = false
			work.Issues = append(work.Issues, issue)
		}
		work.Work.Tasks = tasks
	}
	result := AutoprogrammingBridgeResultV0{
		SchemaVersion: AutoprogrammingBridgeResultSchemaVersionV0,
		Accepted:      work.Accepted,
		Work:          work.Work,
		Issues:        append([]orquestaautoprogramming.AutoprogrammingRequestIssueV0(nil), work.Issues...),
	}
	if !work.Accepted {
		return result, nil
	}
	if issue := autoprogrammingBridgeGoalFirstLaunchIssueV0(work.Work, ports); issue.Code != "" {
		result.Accepted = false
		result.Issues = append(result.Issues, issue)
		return result, nil
	}
	if autoprogrammingBridgeShouldStartGoalFirstV0(work.Work, ports) {
		return autoprogrammingBridgeStartGoalFirstV0(ctx, request, result, ports)
	}
	if !autoprogrammingBridgeShouldMaterializeLegacyLoopV0(work.Work) {
		return result, nil
	}
	run := autoprogrammingBridgeRunV0(request, work.Work)
	if issues := orquestacoreworkflow.ValidateOrchestrationRunV0(run); len(issues) > 0 {
		return AutoprogrammingBridgeResultV0{}, fmt.Errorf("autoprogramming run invalido: %s", issues[0].Error())
	}
	existing, err := ports.RunStore.LoadRunV0(ctx, run.RunID)
	if err != nil && !orquestacionnucleoapp.IsRunNotFoundErrorV0(err) {
		return AutoprogrammingBridgeResultV0{}, err
	}
	if err == nil {
		return autoprogrammingBridgeExistingRunResultV0(ctx, request, result, existing, run, ports)
	}
	for _, task := range work.Work.Tasks {
		if err := ports.DirectorTaskStore.SaveWorkflowTaskV0(ctx, task); err != nil {
			return AutoprogrammingBridgeResultV0{}, err
		}
	}
	if err := ports.RunStore.SaveRunV0(ctx, run); err != nil {
		return AutoprogrammingBridgeResultV0{}, err
	}
	result.Run = run
	result.Tasks = append([]orquestacoreworkflow.WorkflowTaskV0(nil), work.Work.Tasks...)
	result.WaitAgentRefs = autoprogrammingBridgeWaitAgentRefsForRunV0(work.Work.Tasks, run)
	continueRequest, err := autoprogrammingBridgeContinueRequestWithPlanStateV0(ctx, request, result, ports)
	if err != nil {
		return AutoprogrammingBridgeResultV0{}, err
	}
	result.Continue = continueRequest
	return result, nil
}

func autoprogrammingBridgeShouldStartGoalFirstV0(
	work orquestaautoprogramming.AutoprogrammingProgrammableWorkV0,
	ports orquestaappdirectorservice.StartAppDirectorPortsV0,
) bool {
	return autoprogrammingBridgeGoalFirstBackendAvailableV0(ports) &&
		strings.TrimSpace(work.GoalMigration.Status) == orquestaautoprogramming.AutoprogrammingGoalMigrationGoalReadyV0 &&
		len(work.GoalSpecs) > 0
}

func autoprogrammingBridgeGoalFirstBackendAvailableV0(
	ports orquestaappdirectorservice.StartAppDirectorPortsV0,
) bool {
	return ports.GoalLauncher != nil &&
		ports.GoalObserver != nil &&
		ports.GoalClosureValidator != nil &&
		ports.GoalStateStore != nil
}

func autoprogrammingBridgeGoalFirstLaunchIssueV0(
	work orquestaautoprogramming.AutoprogrammingProgrammableWorkV0,
	ports orquestaappdirectorservice.StartAppDirectorPortsV0,
) orquestaautoprogramming.AutoprogrammingRequestIssueV0 {
	if strings.TrimSpace(work.GoalMigration.Status) != orquestaautoprogramming.AutoprogrammingGoalMigrationGoalReadyV0 {
		return orquestaautoprogramming.AutoprogrammingRequestIssueV0{}
	}
	if autoprogrammingBridgeGoalFirstBackendConfiguredV0(ports) &&
		!autoprogrammingBridgeGoalFirstBackendAvailableV0(ports) {
		return orquestaautoprogramming.AutoprogrammingRequestIssueV0{
			Code:    "autoprogramming_goal_backend_incomplete",
			Field:   autoprogrammingBridgeMissingGoalPortFieldV0(ports),
			Message: "backend goal-first incompleto; requiere launcher, observer, closure validator y state store",
		}
	}
	if !autoprogrammingBridgeGoalFirstBackendAvailableV0(ports) {
		return orquestaautoprogramming.AutoprogrammingRequestIssueV0{}
	}
	switch len(work.GoalSpecs) {
	case 0:
		return orquestaautoprogramming.AutoprogrammingRequestIssueV0{
			Code:    "autoprogramming_goal_specs_missing",
			Field:   "goal_specs",
			Message: "goal_specs requerido para lanzamiento goal-first",
		}
	default:
		return orquestaautoprogramming.AutoprogrammingRequestIssueV0{}
	}
}

func autoprogrammingBridgeGoalFirstBackendConfiguredV0(
	ports orquestaappdirectorservice.StartAppDirectorPortsV0,
) bool {
	return ports.GoalLauncher != nil || ports.GoalObserver != nil
}

func autoprogrammingBridgeMissingGoalPortFieldV0(
	ports orquestaappdirectorservice.StartAppDirectorPortsV0,
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

func autoprogrammingBridgeShouldMaterializeLegacyLoopV0(
	work orquestaautoprogramming.AutoprogrammingProgrammableWorkV0,
) bool {
	status := strings.TrimSpace(work.GoalMigration.Status)
	action := strings.TrimSpace(work.GoalMigration.RecommendedAction)
	switch status {
	case orquestaautoprogramming.AutoprogrammingGoalMigrationGoalReadyV0,
		orquestaautoprogramming.AutoprogrammingGoalMigrationCoveredByGoalFirstV0,
		orquestaautoprogramming.AutoprogrammingGoalMigrationBlockedByGoalCapabilityV0:
		return false
	case orquestaautoprogramming.AutoprogrammingGoalMigrationLegacyCompatibleV0,
		orquestaautoprogramming.AutoprogrammingGoalMigrationLegacyLoopRequiredV0:
		return true
	default:
		return action == "" || action == orquestaautoprogramming.AutoprogrammingGoalMigrationActionKeepLegacyLoopV0
	}
}
