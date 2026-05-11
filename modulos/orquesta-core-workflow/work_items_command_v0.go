package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

const (
	maxCreateMicrotaskCommandPayloadBytesV0 = 4608
	maxMicrotaskCreatedPayloadBytesV0       = 1024
)

type CreateMicrotaskCommandPayloadV0 struct {
	Task WorkflowTaskV0 `json:"task"`
}

type MicrotaskCreatedPayloadV0 struct {
	TaskID               string   `json:"task_id"`
	PhaseID              string   `json:"phase_id"`
	FunctionContractRefs []string `json:"function_contract_refs,omitempty"`
}

func NewCreateMicrotaskCommandV0(meta OrchestrationCommandMetaV0, payload CreateMicrotaskCommandPayloadV0) (OrchestrationCommandV0, error) {
	return newOrchestrationCommandV0(meta, OrchestrationCommandCreateMicrotaskV0, payload)
}

func NewMicrotaskCreatedEventV0(meta OrchestrationEventMetaV0, payload MicrotaskCreatedPayloadV0) (OrchestrationEventV0, error) {
	return newOrchestrationEventV0(meta, OrchestrationEventMicrotaskCreatedV0, payload)
}

func decodeCreateMicrotaskCommandPayloadV0(raw json.RawMessage) (CreateMicrotaskCommandPayloadV0, error) {
	if len(raw) == 0 || len(raw) > maxCreateMicrotaskCommandPayloadBytesV0 {
		return CreateMicrotaskCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	var payload CreateMicrotaskCommandPayloadV0
	if err := json.Unmarshal(raw, &payload); err != nil {
		return CreateMicrotaskCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	task, err := NewWorkflowTaskV0(payload.Task)
	if err != nil {
		return CreateMicrotaskCommandPayloadV0{}, commandPayloadErrorV0(err)
	}
	if err := validateMicrotaskProjectionForCommandV0(task); err != nil {
		return CreateMicrotaskCommandPayloadV0{}, err
	}
	if err := validateCreateMicrotaskFunctionRefsRequiredV0(task); err != nil {
		return CreateMicrotaskCommandPayloadV0{}, err
	}
	payload.Task = task
	return payload, nil
}

func microtaskCreatedPayloadFromTaskV0(task WorkflowTaskV0) MicrotaskCreatedPayloadV0 {
	return MicrotaskCreatedPayloadV0{
		TaskID:               strings.TrimSpace(task.TaskID),
		PhaseID:              strings.TrimSpace(string(task.PhaseID)),
		FunctionContractRefs: compactFunctionContractRefsV0(task.FunctionContractRefs),
	}
}

func validateMicrotaskCreatedPayloadV0(event OrchestrationEventV0) error {
	var payload MicrotaskCreatedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return err
	}
	return validateMicrotaskCreatedPayloadDataV0(payload)
}

func validateMicrotaskProjectionForCommandV0(task WorkflowTaskV0) error {
	if err := validateMicrotaskCreatedPayloadDataV0(microtaskCreatedPayloadFromTaskV0(task)); err != nil {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload.task.function_contract_refs")
	}
	return nil
}

func validateMicrotaskCreatedPayloadDataV0(payload MicrotaskCreatedPayloadV0) error {
	if err := requirePayloadFieldsV0(map[string]string{
		"task_id":  payload.TaskID,
		"phase_id": payload.PhaseID,
	}); err != nil {
		return err
	}
	if err := ValidateOrchestrationPhaseIDV0(OrchestrationPhaseIDV0(payload.PhaseID)); err != nil {
		return err
	}
	if err := validateMicrotaskCreatedFunctionRefsV0(payload.FunctionContractRefs); err != nil {
		return err
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxMicrotaskCreatedPayloadBytesV0 {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func validateMicrotaskCreatedFunctionRefsV0(refs []string) error {
	if len(refs) > maxWorkflowTaskCollectionV0 {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload.function_contract_refs")
	}
	for _, ref := range refs {
		compact := strings.TrimSpace(ref)
		if compact == "" || len(compact) > maxWorkflowTaskStringV0 {
			return eventErrorV0(ErrPayloadInvalidoV0, "payload.function_contract_refs")
		}
		if workflowTaskStringHasForbiddenDetailV0(compact) {
			return eventErrorV0(ErrDetalleProhibidoV0, "payload.function_contract_refs")
		}
	}
	return nil
}

func handleCreateMicrotaskCommandV0(current OrchestrationRunV0, command OrchestrationCommandV0) (OrchestrationCommandResultV0, error) {
	payload, err := decodeCreateMicrotaskCommandPayloadV0(command.Payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	if err := ensureActiveRunForCommandV0(current, command); err != nil {
		return emptyCommandResultV0(), err
	}
	if strings.TrimSpace(payload.Task.RunID) != strings.TrimSpace(command.RunID) {
		return emptyCommandResultV0(), commandErrorV0(ErrTransicionInvalidaV0, "payload.task.run_id")
	}
	if err := ensureCreateMicrotaskCommandPlanningCurrentV0(current); err != nil {
		return emptyCommandResultV0(), err
	}
	if err := ensureCreateMicrotaskCommandContractsReadyV0(current, payload.Task); err != nil {
		return emptyCommandResultV0(), err
	}
	if err := validateWorkflowTaskDependencyRefsExistV0(current, payload.Task); err != nil {
		return emptyCommandResultV0(), err
	}
	if microtaskAlreadyReflectedV0(current, payload.Task.TaskID) {
		if err := ensureCommandEffectMatchesV0(current, command, OrchestrationEventMicrotaskCreatedV0, payload.Task.TaskID, microtaskCreatedPayloadFromTaskV0(payload.Task)); err != nil {
			return emptyCommandResultV0(), err
		}
		return idempotentCommandResultV0(), nil
	}
	event, err := NewMicrotaskCreatedEventV0(
		commandEventMetaV0(current, command, OrchestrationEventMicrotaskCreatedV0),
		microtaskCreatedPayloadFromTaskV0(payload.Task),
	)
	return eventCommandResultV0(event, err)
}

func microtaskAlreadyReflectedV0(current OrchestrationRunV0, taskID string) bool {
	taskID = strings.TrimSpace(taskID)
	for _, existing := range current.Tasks {
		if strings.TrimSpace(existing) == taskID {
			return true
		}
	}
	return false
}

func compactFunctionContractRefsV0(refs []WorkflowFunctionContractRefV0) []string {
	compact := make([]string, 0, len(refs))
	for _, ref := range normalizeWorkflowFunctionContractRefsV0(refs) {
		if strings.TrimSpace(ref.ContractRef) != "" {
			compact = appendUniqueCompactRefV0(compact, ref.ContractRef)
			continue
		}
		compact = appendUniqueCompactRefV0(compact, ref.FunctionName)
	}
	return compact
}
