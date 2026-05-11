package orquestadirector

import (
	"encoding/json"
	"strings"

	orquestacontext "orquesta/modulos/orquesta-context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func BuildLaunchContextBundleV0(input LaunchContextBundleInputV0) (LaunchContextBundleResultV0, error) {
	input = normalizeLaunchContextBundleInputV0(input)
	payload, err := launchContextBundlePayloadV0(input.Message)
	if err != nil {
		result := LaunchContextBundleResultV0{Issues: []LaunchContextBundleIssueV0{launchContextIssueV0(ErrDirectorContextBundleOutboxV0, "message", "outbox invalido")}}
		return result, launchContextBundleErrorV0(ErrDirectorContextBundleOutboxV0, "message", result.Issues)
	}

	request := orquestacontext.ContextBundleRequestV0{
		SchemaVersion:   orquestacontext.ContextBundleRequestSchemaVersionV0,
		BundleRef:       "context-bundle-" + payload.AgentRequestID,
		WorkOrderRef:    payload.AgentRequestID,
		TargetModule:    input.TargetModule,
		Phase:           payload.PhaseID,
		TaskKind:        firstNonEmptyLaunchContextV0(input.TaskKind, payload.Role),
		Objective:       firstNonEmptyLaunchContextV0(input.Objective, payload.Summary),
		CapacityLevel:   input.CapacityLevel,
		ReadSet:         input.ReadSet,
		WriteSet:        input.WriteSet,
		ContractRefs:    append([]string{payload.TaskRef}, input.ContractRefs...),
		CrossModuleRefs: input.CrossModuleRefs,
		EvidenceRefs:    payload.EvidenceRefs,
		MaxEntries:      input.MaxEntries,
		MaxTotalBytes:   input.MaxTotalBytes,
	}
	bundle := orquestacontext.BuildContextBundleV0(request)
	result := LaunchContextBundleResultV0{Bundle: bundle}
	if len(bundle.Issues) > 0 {
		result.Issues = launchContextIssuesFromBundleV0(bundle.Issues)
		return result, launchContextBundleErrorV0(ErrDirectorContextBundleInvalidoV0, "context_bundle", result.Issues)
	}
	return result, nil
}

func launchContextBundlePayloadV0(
	message orquestacoreworkflow.OutboxMessageV0,
) (orquestaruntime.LaunchRuntimeAgentRequestV0, error) {
	if err := orquestacoreworkflow.ValidateOutboxMessageV0(message); err != nil {
		return orquestaruntime.LaunchRuntimeAgentRequestV0{}, err
	}
	if message.MessageType != orquestacoreworkflow.OutboxMessageLaunchRuntimeAgentV0 ||
		message.TargetPort != orquestacoreworkflow.OutboxTargetAgentLauncherV0 {
		return orquestaruntime.LaunchRuntimeAgentRequestV0{}, launchContextBundleErrorV0(ErrDirectorContextBundleOutboxV0, "message_type", nil)
	}
	var payload orquestaruntime.LaunchRuntimeAgentRequestV0
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		return orquestaruntime.LaunchRuntimeAgentRequestV0{}, err
	}
	if issues := orquestaruntime.ValidateLaunchRuntimeAgentRequestV0(payload); len(issues) > 0 {
		return orquestaruntime.LaunchRuntimeAgentRequestV0{}, launchContextBundleErrorV0(ErrDirectorContextBundleOutboxV0, "payload", nil)
	}
	return payload, nil
}

func normalizeLaunchContextBundleInputV0(input LaunchContextBundleInputV0) LaunchContextBundleInputV0 {
	input.TargetModule = strings.TrimSpace(input.TargetModule)
	input.TaskKind = strings.TrimSpace(input.TaskKind)
	input.Objective = strings.TrimSpace(input.Objective)
	input.CapacityLevel = strings.TrimSpace(input.CapacityLevel)
	input.ReadSet = compactLaunchContextStringsV0(input.ReadSet)
	input.WriteSet = compactLaunchContextStringsV0(input.WriteSet)
	input.ContractRefs = compactLaunchContextStringsV0(input.ContractRefs)
	input.CrossModuleRefs = compactLaunchContextStringsV0(input.CrossModuleRefs)
	return input
}
