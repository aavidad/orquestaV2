package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

const AutoprogrammingBridgeResultSchemaVersionV0 = "autoprogramming_bridge_result.v0"

const autoprogrammingBridgeOperationalTaskSourceRefV0 = "operational_director.task_source:autoprogramming"

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
	result.WaitAgentRefs = autoprogrammingBridgeWaitAgentRefsV0(work.Work.Tasks)
	continueRequest, err := autoprogrammingBridgeContinueRequestWithPlanStateV0(ctx, request, result, ports)
	if err != nil {
		return AutoprogrammingBridgeResultV0{}, err
	}
	result.Continue = continueRequest
	return result, nil
}

func autoprogrammingBridgeExistingRunResultV0(
	ctx context.Context,
	request AutoprogrammingBridgeRequestV0,
	result AutoprogrammingBridgeResultV0,
	existing orquestacoreworkflow.OrchestrationRunV0,
	expected orquestacoreworkflow.OrchestrationRunV0,
	ports orquestaappdirectorservice.StartAppDirectorPortsV0,
) (AutoprogrammingBridgeResultV0, error) {
	if err := autoprogrammingBridgeValidateExistingRunV0(existing, expected); err != nil {
		return AutoprogrammingBridgeResultV0{}, err
	}
	expectedTaskRefs := compactStringsV0(expected.Tasks)
	storedTasks, err := ports.DirectorTaskStore.LoadWorkflowTasksV0(ctx, expected.RunID, expectedTaskRefs)
	if err != nil {
		return AutoprogrammingBridgeResultV0{}, err
	}
	if err := autoprogrammingBridgeValidateStoredTasksV0(storedTasks, result.Work.Tasks); err != nil {
		return AutoprogrammingBridgeResultV0{}, err
	}
	result.Run = existing
	result.Tasks = append([]orquestacoreworkflow.WorkflowTaskV0(nil), storedTasks...)
	result.WaitAgentRefs = autoprogrammingBridgeWaitAgentRefsV0(result.Work.Tasks)
	continueRequest, err := autoprogrammingBridgeContinueRequestWithPlanStateV0(ctx, request, result, ports)
	if err != nil {
		return AutoprogrammingBridgeResultV0{}, err
	}
	result.Continue = continueRequest
	return result, nil
}

func autoprogrammingBridgeValidateExistingRunV0(
	existing orquestacoreworkflow.OrchestrationRunV0,
	expected orquestacoreworkflow.OrchestrationRunV0,
) error {
	if strings.TrimSpace(existing.RunID) != strings.TrimSpace(expected.RunID) ||
		strings.TrimSpace(existing.ProjectRef) != strings.TrimSpace(expected.ProjectRef) ||
		strings.TrimSpace(existing.AppSpecRef) != strings.TrimSpace(expected.AppSpecRef) {
		return fmt.Errorf("autoprogramming run existente incompatible: %s", expected.RunID)
	}
	for _, taskRef := range compactStringsV0(expected.Tasks) {
		if !stringInSetV0(existing.Tasks, taskRef) {
			return fmt.Errorf("autoprogramming run existente sin task esperada: %s", taskRef)
		}
	}
	for _, contractRef := range compactStringsV0(expected.FunctionContracts) {
		if !stringInSetV0(existing.FunctionContracts, contractRef) {
			return fmt.Errorf("autoprogramming run existente sin contrato esperado: %s", contractRef)
		}
	}
	return nil
}

func autoprogrammingBridgeValidateStoredTasksV0(
	stored []orquestacoreworkflow.WorkflowTaskV0,
	expected []orquestacoreworkflow.WorkflowTaskV0,
) error {
	expectedByRef := map[string]orquestacoreworkflow.WorkflowTaskV0{}
	for _, task := range expected {
		expectedByRef[strings.TrimSpace(task.TaskID)] = task
	}
	seen := map[string]bool{}
	for _, task := range stored {
		taskRef := strings.TrimSpace(task.TaskID)
		expectedTask, ok := expectedByRef[strings.TrimSpace(task.TaskID)]
		if !ok || strings.TrimSpace(task.RunID) != strings.TrimSpace(expectedTask.RunID) {
			return fmt.Errorf("autoprogramming workflow task existente incompatible: %s", task.TaskID)
		}
		if err := orquestacoreworkflow.ValidateWorkflowTaskV0(task); err != nil {
			return fmt.Errorf("autoprogramming workflow task existente invalida: %s", task.TaskID)
		}
		seen[taskRef] = true
	}
	for taskRef := range expectedByRef {
		if !seen[taskRef] {
			return fmt.Errorf("autoprogramming workflow task existente ausente: %s", taskRef)
		}
	}
	return nil
}

func normalizeAutoprogrammingBridgeRequestV0(
	request AutoprogrammingBridgeRequestV0,
) AutoprogrammingBridgeRequestV0 {
	request.OccurredAt = strings.TrimSpace(request.OccurredAt)
	request.CorrelationID = strings.TrimSpace(request.CorrelationID)
	request.RequestedBy = strings.TrimSpace(request.RequestedBy)
	if request.CorrelationID == "" {
		request.CorrelationID = "corr-" + strings.TrimSpace(request.Request.RequestRef)
	}
	if request.RequestedBy == "" {
		request.RequestedBy = "orquesta-app-codex-stack-autoprogramming"
	}
	return request
}

func autoprogrammingBridgeOperationalTasksV0(
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) ([]orquestacoreworkflow.WorkflowTaskV0, orquestaautoprogramming.AutoprogrammingRequestIssueV0) {
	out := make([]orquestacoreworkflow.WorkflowTaskV0, 0, len(tasks))
	for _, task := range tasks {
		task.ContextRefs = compactStringsV0(append(task.ContextRefs, autoprogrammingBridgeOperationalTaskSourceRefV0))
		repaired, issue := orquestaautoprogramming.EnsureAutoprogrammingWorkflowTaskAcceptedByCoreV0(task)
		if issue.Code != "" {
			return out, issue
		}
		task = repaired
		out = append(out, task)
	}
	return out, orquestaautoprogramming.AutoprogrammingRequestIssueV0{}
}

func validateAutoprogrammingBridgePortsV0(
	ports orquestaappdirectorservice.StartAppDirectorPortsV0,
) error {
	if ports.RunStore == nil {
		return fmt.Errorf("ports.run_store requerido")
	}
	if ports.DirectorTaskStore == nil {
		return fmt.Errorf("ports.director_task_store requerido")
	}
	return nil
}

func autoprogrammingBridgeRunV0(
	request AutoprogrammingBridgeRequestV0,
	work orquestaautoprogramming.AutoprogrammingProgrammableWorkV0,
) orquestacoreworkflow.OrchestrationRunV0 {
	phases := orquestacoreworkflow.OrchestrationPhaseCatalogV0()
	for index := range phases {
		if phases[index].ID != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
			continue
		}
		phases[index].Status = orquestacoreworkflow.OrchestrationPhaseStatusActiveV0
		phases[index].OpenedAt = request.OccurredAt
	}
	taskRefs := make([]string, 0, len(work.Tasks))
	for _, task := range work.Tasks {
		taskRefs = append(taskRefs, task.TaskID)
	}
	return orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion:     orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:             work.RequestRef,
		ProjectRef:        work.ProjectRef,
		AppSpecRef:        "app-spec-ref-autoprogramming-" + autoprogrammingBridgeHashRefV0(work.RequestRef),
		Status:            orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:      orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Phases:            phases,
		Tasks:             compactStringsV0(taskRefs),
		FunctionContracts: autoprogrammingBridgeFunctionRefsV0(work.Tasks),
	}
}

func autoprogrammingBridgeContinueRequestV0(
	request AutoprogrammingBridgeRequestV0,
	result AutoprogrammingBridgeResultV0,
) orquestaappdirectorservice.ContinueAppDirectorRequestV0 {
	return orquestaappdirectorservice.ContinueAppDirectorRequestV0{
		RunRef:               result.Run.RunID,
		OccurredAt:           request.OccurredAt,
		CorrelationID:        request.CorrelationID,
		RequestedBy:          request.RequestedBy,
		MaxBursts:            request.MaxBursts,
		MaxStepsPerBurst:     request.MaxStepsPerBurst,
		MaxDispatchesPerWait: request.MaxDispatchesPerWait,
		WaitAgentRefs:        append([]string(nil), result.WaitAgentRefs...),
		MaxCommands:          request.MaxCommands,
		MaxOutboxPerCycle:    request.MaxOutboxPerCycle,
	}
}

func autoprogrammingBridgeContinueRequestWithPlanStateV0(
	ctx context.Context,
	request AutoprogrammingBridgeRequestV0,
	result AutoprogrammingBridgeResultV0,
	ports orquestaappdirectorservice.StartAppDirectorPortsV0,
) (orquestaappdirectorservice.ContinueAppDirectorRequestV0, error) {
	continueRequest := autoprogrammingBridgeContinueRequestV0(request, result)
	stateRequest, err := orquestaappdirectorservice.EnsureOperationalDirectorPlanStateFromWorkflowTasksV0(
		ctx,
		orquestaappdirectorservice.ContinueAppDirectorRequestV0{
			RunRef:            result.Run.RunID,
			OccurredAt:        request.OccurredAt,
			CorrelationID:     request.CorrelationID,
			RequestedBy:       request.RequestedBy,
			MaxBursts:         request.MaxBursts,
			MaxStepsPerBurst:  request.MaxStepsPerBurst,
			MaxCommands:       request.MaxCommands,
			MaxOutboxPerCycle: request.MaxOutboxPerCycle,
		},
		ports,
	)
	if err != nil {
		return orquestaappdirectorservice.ContinueAppDirectorRequestV0{}, err
	}
	if strings.TrimSpace(stateRequest.OperationalDirectorPlanRef) != "" {
		continueRequest.OperationalDirectorPlanRef = stateRequest.OperationalDirectorPlanRef
	}
	return continueRequest, nil
}

func autoprogrammingBridgeWaitAgentRefsV0(
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) []string {
	refs := make([]string, 0, len(tasks))
	for _, task := range tasks {
		refs = append(refs, orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(task.TaskID))
	}
	return compactStringsV0(refs)
}

func autoprogrammingBridgeFunctionRefsV0(
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) []string {
	refs := make([]string, 0)
	for _, task := range tasks {
		for _, ref := range task.FunctionContractRefs {
			if strings.TrimSpace(ref.ContractRef) != "" {
				refs = append(refs, ref.ContractRef)
				continue
			}
			refs = append(refs, ref.FunctionName)
		}
	}
	return compactStringsV0(refs)
}

func autoprogrammingBridgeHashRefV0(value string) string {
	return codexStackDeterministicDigestV0(value)
}
