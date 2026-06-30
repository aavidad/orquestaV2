package main

import (
	"fmt"
	"strings"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	operator "orquesta/modulos/orquesta-operator-mcp"
)

const (
	codexToolbeltStatusServerLiveV0       = "transporte_http_si_servidor_vivo"
	codexToolbeltStatusToolRegisteredV0   = "tool_registrado"
	codexToolbeltStatusConnectorOptInV0   = "puede_devolver_mcp_transport_tool_unbound_si_falta_puerto"
	codexToolbeltStatusTransportMissingV0 = "transport_missing_from_registry"
	codexToolbeltStatusResourceV0         = "resource_registrado_con_subtools_operador"
)

type codexToolbeltHTTPEntryV0 struct {
	Method string
	Path   string
	Status string
}

type codexToolbeltMCPEntryV0 struct {
	Name   string
	Status string
}

func codexToolbeltHTTPHintV0() string {
	return "Toolbelt HTTP: " + strings.Join(formatCodexToolbeltHTTPEntriesV0(codexToolbeltHTTPEntriesV0()), ", ") + "."
}

func codexToolbeltMCPHintV0() string {
	return "Toolbelt MCP equivalente: " + strings.Join(formatCodexToolbeltMCPEntriesV0(codexToolbeltMCPEntriesV0()), ", ") + "."
}

func codexToolbeltHTTPEntriesV0() []codexToolbeltHTTPEntryV0 {
	return []codexToolbeltHTTPEntryV0{
		{Method: "POST", Path: orquestamcp.MCPAutoprogrammingStatusHTTPPathV0, Status: codexToolbeltStatusServerLiveV0},
		{Method: "POST", Path: orquestamcp.MCPAutoprogrammingSuperviseHTTPPathV0, Status: codexToolbeltStatusServerLiveV0},
		{Method: "POST", Path: orquestamcp.MCPAutoprogrammingPrepareRunHTTPPathV0, Status: codexToolbeltStatusServerLiveV0},
		{Method: "POST", Path: orquestamcp.MCPAutoprogrammingObserveGoalHTTPPathV0, Status: codexToolbeltStatusServerLiveV0},
		{Method: "POST", Path: orquestamcp.MCPAutoprogrammingSelfImprovementHTTPPathV0, Status: codexToolbeltStatusServerLiveV0},
		{Method: "POST", Path: orquestamcp.MCPHumanDirectorWorkReviewPlanHTTPPathV0, Status: codexToolbeltStatusServerLiveV0},
		{Method: "POST", Path: orquestamcp.MCPRunSupervisorHTTPPathV0, Status: codexToolbeltStatusServerLiveV0},
		{Method: "POST", Path: orquestamcp.MCPRunControlHTTPPathV0, Status: codexToolbeltStatusServerLiveV0},
		{Method: "POST", Path: orquestamcp.MCPRunQueuePriorityHTTPPathV0, Status: codexToolbeltStatusServerLiveV0},
		{Method: "GET", Path: orquestamcp.MCPQueueGlobalStatusHTTPPathV0, Status: codexToolbeltStatusServerLiveV0},
		{Method: "POST", Path: orquestamcp.MCPDirectorStatsHTTPPathV0, Status: codexToolbeltStatusServerLiveV0},
		{Method: "POST", Path: orquestamcp.MCPDomainWorkHTTPPathV0, Status: codexToolbeltStatusServerLiveV0},
		{Method: "GET", Path: orquestamcp.MCPDomainWorkStatusHTTPPathV0, Status: codexToolbeltStatusServerLiveV0},
		{Method: "POST", Path: orquestamcp.MCPExternalWorkDryRunHTTPPathV0, Status: codexToolbeltStatusServerLiveV0},
		{Method: "POST", Path: orquestamcp.MCPExternalWorkRunHTTPPathV0, Status: codexToolbeltStatusServerLiveV0},
		{Method: "POST", Path: orquestamcp.MCPCodebaseQueryHTTPPathV0, Status: codexToolbeltStatusServerLiveV0},
		{Method: "POST", Path: orquestamcp.MCPCodebaseStatusHTTPPathV0, Status: codexToolbeltStatusServerLiveV0},
	}
}

func codexToolbeltMCPEntriesV0() []codexToolbeltMCPEntryV0 {
	registered := codexRegisteredMCPToolNamesV0()
	names := []string{
		orquestamcp.MCPAutoprogrammingStatusToolNameV0,
		orquestamcp.MCPAutoprogrammingSuperviseToolNameV0,
		orquestamcp.MCPAutoprogrammingPrepareRunToolNameV0,
		orquestamcp.MCPAutoprogrammingObserveGoalToolNameV0,
		orquestamcp.MCPAutoprogrammingSelfImprovementToolNameV0,
		orquestamcp.MCPHumanDirectorWorkReviewPlanToolNameV0,
		orquestamcp.MCPRunSupervisorToolNameV0,
		orquestamcp.MCPRunControlToolNameV0,
		orquestamcp.MCPRunQueuePriorityToolNameV0,
		orquestamcp.MCPDirectorStatsToolNameV0,
		orquestamcp.MCPDomainWorkToolNameV0,
		orquestamcp.MCPExternalWorkDryRunToolNameV0,
		orquestamcp.MCPExternalWorkRunToolNameV0,
		orquestamcp.MCPCodebaseQueryToolNameV0,
		orquestamcp.MCPCodebaseStatusToolNameV0,
		operator.OperatorMCPDirectedQueryToolV0,
	}
	out := make([]codexToolbeltMCPEntryV0, 0, len(names)+1)
	out = append(out, codexToolbeltMCPEntryV0{
		Name:   orquestamcp.MCPOperatorOperationsResourceNameV0,
		Status: codexToolbeltStatusResourceV0,
	})
	for _, name := range names {
		status := codexToolbeltStatusForMCPToolV0(name, registered)
		out = append(out, codexToolbeltMCPEntryV0{Name: name, Status: status})
	}
	return out
}

func codexRegisteredMCPToolNamesV0() map[string]struct{} {
	out := map[string]struct{}{}
	for _, tool := range orquestamcp.MCPTransportToolsV0(orquestamcp.MCPTransportBindingsV0{}) {
		out[strings.TrimSpace(tool.Name)] = struct{}{}
	}
	return out
}

func codexToolbeltStatusForMCPToolV0(name string, registered map[string]struct{}) string {
	if _, ok := registered[strings.TrimSpace(name)]; !ok {
		return codexToolbeltStatusTransportMissingV0
	}
	if strings.HasPrefix(name, "orquesta.operator.") ||
		name == orquestamcp.MCPAutoprogrammingPrepareRunToolNameV0 ||
		name == orquestamcp.MCPAutoprogrammingObserveGoalToolNameV0 ||
		name == orquestamcp.MCPAutoprogrammingStatusToolNameV0 ||
		name == orquestamcp.MCPAutoprogrammingSuperviseToolNameV0 ||
		name == orquestamcp.MCPDomainWorkToolNameV0 ||
		name == orquestamcp.MCPExternalWorkRunToolNameV0 ||
		name == orquestamcp.MCPCodebaseQueryToolNameV0 ||
		name == orquestamcp.MCPCodebaseStatusToolNameV0 {
		return codexToolbeltStatusConnectorOptInV0
	}
	return codexToolbeltStatusToolRegisteredV0
}

func formatCodexToolbeltHTTPEntriesV0(entries []codexToolbeltHTTPEntryV0) []string {
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		out = append(out, strings.TrimSpace(entry.Method)+" "+strings.TrimSpace(entry.Path)+" ["+strings.TrimSpace(entry.Status)+"]")
	}
	return out
}

func formatCodexToolbeltMCPEntriesV0(entries []codexToolbeltMCPEntryV0) []string {
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		out = append(out, fmt.Sprintf("%s [%s]", strings.TrimSpace(entry.Name), strings.TrimSpace(entry.Status)))
	}
	return out
}
