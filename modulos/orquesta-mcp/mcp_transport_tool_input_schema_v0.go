package orquestamcp

import (
	"reflect"
	"strings"

	orquestacontext "orquesta/modulos/orquesta-context"
	channel "orquesta/modulos/orquesta-operator-director-channel"
	operator "orquesta/modulos/orquesta-operator-mcp"
)

type MCPTransportToolInputFieldV0 struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Required bool     `json:"required,omitempty"`
	Enum     []string `json:"enum,omitempty"`
}

func MCPTransportToolInputFieldsV0(name string) ([]MCPTransportToolInputFieldV0, bool) {
	dto, ok := mcpTransportToolInputDTOByNameV0(strings.TrimSpace(name))
	if !ok {
		return nil, false
	}
	fields := mcpTransportToolInputFieldsFromDTOV0(dto)
	required := mcpTransportToolRequiredFieldsV0(name)
	enums := mcpTransportToolEnumsV0(name)
	for idx := range fields {
		fields[idx].Required = required[fields[idx].Name]
		fields[idx].Enum = enums[fields[idx].Name]
	}
	return fields, true
}

func mcpTransportToolInputDTOByNameV0(name string) (any, bool) {
	switch name {
	case MCPNuevaAppToolNameV0:
		return MCPNuevaAppToolInputV0{}, true
	case MCPNuevaAppWizardToolNameV0:
		return MCPNuevaAppWizardToolInputV0{}, true
	case MCPArrancarDirectorAppToolNameV0:
		return MCPArrancarDirectorAppToolInputV0{}, true
	case MCPObserveAppDirectorGoalToolNameV0:
		return MCPObserveAppDirectorGoalToolInputV0{}, true
	case MCPRequestAppChangeToolNameV0:
		return MCPRequestAppChangeToolInputV0{}, true
	case MCPDirectorAgentDecisionToolNameV0:
		return MCPDirectorAgentDecisionToolInputV0{}, true
	case MCPDirectorSupervisorBriefingToolNameV0:
		return MCPDirectorSupervisorBriefingToolInputV0{}, true
	case MCPDirectorStatsToolNameV0:
		return MCPDirectorStatsToolInputV0{}, true
	case MCPPrepararOrquestacionAppToolNameV0:
		return MCPPrepararOrquestacionAppToolInputV0{}, true
	case MCPEjecutarOrquestacionAppToolNameV0:
		return MCPEjecutarOrquestacionAppToolInputV0{}, true
	case MCPAutoprogrammingValidateRequestToolNameV0:
		return MCPAutoprogrammingValidateRequestToolInputV0{}, true
	case MCPHumanDirectorWorkReviewPlanToolNameV0:
		return MCPHumanDirectorWorkReviewPlanToolInputV0{}, true
	case MCPAutoprogrammingSelfImprovementToolNameV0:
		return MCPAutoprogrammingSelfImprovementToolInputV0{}, true
	case MCPAutoprogrammingPrepareRunToolNameV0:
		return MCPAutoprogrammingPrepareRunToolInputV0{}, true
	case MCPAutoprogrammingObserveGoalToolNameV0:
		return MCPAutoprogrammingObserveGoalToolInputV0{}, true
	case MCPAutoprogrammingObserveActiveGoalsToolNameV0:
		return MCPAutoprogrammingObserveActiveGoalsToolInputV0{}, true
	case MCPAutoprogrammingStatusToolNameV0:
		return mcpAutoprogrammingStatusTransportInputV0{}, true
	case MCPAutoprogrammingSuperviseToolNameV0:
		return mcpAutoprogrammingSuperviseTransportInputV0{}, true
	case MCPBootstrapToolNameV0:
		return MCPBootstrapToolInputV0{}, true
	case MCPCoreWorkflowCommandToolNameV0:
		return MCPCoreWorkflowCommandToolInputV0{}, true
	case MCPRunControlToolNameV0:
		return MCPRunControlToolInputV0{}, true
	case MCPRuntimeModelsToolNameV0:
		return MCPRuntimeModelsToolInputV0{}, true
	case MCPRunQueuePriorityToolNameV0:
		return MCPRunQueuePriorityToolInputV0{}, true
	case MCPRunSupervisorToolNameV0:
		return MCPRunSupervisorToolInputV0{}, true
	case MCPWorkspaceTimelineToolNameV0:
		return MCPWorkspaceTimelineToolInputV0{}, true
	case MCPServerShutdownToolNameV0:
		return MCPServerShutdownToolInputV0{}, true
	case MCPDomainWorkToolNameV0:
		return MCPDomainWorkToolInputV0{}, true
	case MCPExternalWorkDryRunToolNameV0:
		return MCPExternalWorkDryRunToolInputV0{}, true
	case MCPExternalWorkRunToolNameV0:
		return MCPExternalWorkRunToolInputV0{}, true
	case MCPCodebaseQueryToolNameV0:
		return MCPCodebaseQueryToolInputV0{}, true
	case MCPCodebaseStatusToolNameV0:
		return MCPCodebaseStatusToolInputV0{}, true
	case MCPAppVCSToolNameV0:
		return MCPAppVCSToolInputV0{}, true
	case channel.OperatorDirectorMessageToolNameV0:
		return channel.OperatorMessageV0{}, true
	case operator.OperatorMCPStatusToolNameV0:
		return operator.OperatorStatusQueryV0{}, true
	case operator.OperatorMCPBurstToolNameV0:
		return operator.OperatorSupervisedBurstRequestV0{}, true
	case operator.OperatorMCPOutboxToolNameV0:
		return operator.OperatorPendingOutboxQueryV0{}, true
	case operator.OperatorMCPDirectedQueryToolV0:
		return operator.OperatorDirectedQueryV0{}, true
	case MCPOperatorFriendlyStatusToolNameV0, MCPOperatorFriendlyTasksToolNameV0,
		MCPOperatorFriendlyProjectsToolNameV0, MCPOperatorFriendlyAgentsToolNameV0,
		MCPOperatorFriendlyCommandToolNameV0:
		return MCPOperatorFriendlyQueryV0{}, true
	default:
		return nil, false
	}
}

func mcpTransportToolInputFieldsFromDTOV0(dto any) []MCPTransportToolInputFieldV0 {
	typ := reflect.TypeOf(dto)
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	fields := []MCPTransportToolInputFieldV0{}
	for idx := 0; idx < typ.NumField(); idx++ {
		field := typ.Field(idx)
		if field.PkgPath != "" {
			continue
		}
		if field.Anonymous {
			fields = append(fields, mcpTransportToolInputFieldsFromDTOV0(reflect.New(field.Type).Elem().Interface())...)
			continue
		}
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		if name == "" || name == "-" {
			continue
		}
		fields = append(fields, MCPTransportToolInputFieldV0{
			Name: name,
			Type: mcpTransportJSONTypeForFieldV0(field.Type),
		})
	}
	return fields
}

func mcpTransportJSONTypeForFieldV0(typ reflect.Type) string {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	switch typ.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Struct, reflect.Map, reflect.Interface:
		return "object"
	default:
		return "string"
	}
}

func mcpTransportToolRequiredFieldsV0(name string) map[string]bool {
	switch strings.TrimSpace(name) {
	case MCPObserveAppDirectorGoalToolNameV0:
		return map[string]bool{"run_ref": true}
	case MCPAutoprogrammingObserveGoalToolNameV0:
		return map[string]bool{"run_ref": true}
	case MCPAppVCSToolNameV0:
		return map[string]bool{"action": true, "app_ref": true, "repo_ref": true}
	case MCPCodebaseQueryToolNameV0:
		return map[string]bool{"repository_ref": true, "query": true}
	case MCPDirectorSupervisorBriefingToolNameV0:
		return map[string]bool{"briefing_input": true}
	case channel.OperatorDirectorMessageToolNameV0:
		return map[string]bool{"request_ref": true, "target_ref": true, "body": true}
	case operator.OperatorMCPStatusToolNameV0:
		return map[string]bool{"request_ref": true, "subject_ref": true, "status_connector_ref": true}
	case operator.OperatorMCPBurstToolNameV0:
		return map[string]bool{"request_ref": true, "run_ref": true, "burst_connector_ref": true, "supervision_ref": true, "max_steps": true}
	case operator.OperatorMCPOutboxToolNameV0:
		return map[string]bool{"request_ref": true, "subject_ref": true, "outbox_connector_ref": true, "limit": true}
	case operator.OperatorMCPDirectedQueryToolV0:
		return map[string]bool{"query_ref": true, "target_ref": true, "query_connector_ref": true, "question": true}
	default:
		return map[string]bool{}
	}
}

func mcpTransportToolEnumsV0(name string) map[string][]string {
	switch strings.TrimSpace(name) {
	case MCPArrancarDirectorAppToolNameV0:
		return map[string][]string{"director_execution_mode": []string{"goal_first", "legacy_director_loop"}}
	case MCPRunSupervisorToolNameV0, MCPAutoprogrammingSuperviseToolNameV0, MCPAutoprogrammingPrepareRunToolNameV0,
		MCPAutoprogrammingSelfImprovementToolNameV0:
		return map[string][]string{"director_execution_mode": []string{"goal_first", "legacy_director_loop"}}
	case MCPRunControlToolNameV0:
		return map[string][]string{"action": []string{"pause", "resume", "stop", "cancel"}}
	case MCPRuntimeModelsToolNameV0:
		return map[string][]string{"action": []string{"list", "status", "pull", "serve", "stop"}}
	case MCPRunQueuePriorityToolNameV0:
		return map[string][]string{"action": []string{"rank", "set_priority"}}
	case MCPDomainWorkToolNameV0:
		return map[string][]string{"action": []string{"create_job", "submit_artifact"}}
	case MCPExternalWorkRunToolNameV0:
		return map[string][]string{"director_execution_mode": []string{"goal_first", "legacy_director_loop"}}
	case MCPExternalWorkDryRunToolNameV0:
		return map[string][]string{"director_execution_mode": []string{"goal_first"}}
	case MCPAppVCSToolNameV0:
		return map[string][]string{"action": []string{"prepare_repo", "review_repo", "commit", "push"}}
	case MCPCodebaseQueryToolNameV0:
		return map[string][]string{"query_kind": []string{
			orquestacontext.CodeContextQueryKindSearchV0,
			orquestacontext.CodeContextQueryKindSymbolV0,
			orquestacontext.CodeContextQueryKindArchitectureV0,
			orquestacontext.CodeContextQueryKindRepoMapV0,
			orquestacontext.CodeContextQueryKindCallersV0,
			orquestacontext.CodeContextQueryKindImportsV0,
			orquestacontext.CodeContextQueryKindModuleExportsV0,
			orquestacontext.CodeContextQueryKindRelevantSnippetsV0,
		}}
	default:
		return map[string][]string{}
	}
}
