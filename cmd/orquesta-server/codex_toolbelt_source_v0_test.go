package main

import (
	"strings"
	"testing"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	operator "orquesta/modulos/orquesta-operator-mcp"
)

func TestCodexToolbeltSourceV0DerivaHTTPDeContratosMCPGateway(t *testing.T) {
	paths := codexToolbeltHTTPPathsForTestV0()
	for _, want := range []string{
		orquestamcp.MCPAutoprogrammingStatusHTTPPathV0,
		orquestamcp.MCPAutoprogrammingSuperviseHTTPPathV0,
		orquestamcp.MCPAutoprogrammingPrepareRunHTTPPathV0,
		orquestamcp.MCPAutoprogrammingObserveGoalHTTPPathV0,
		orquestamcp.MCPAutoprogrammingSelfImprovementHTTPPathV0,
		orquestamcp.MCPHumanDirectorWorkReviewPlanHTTPPathV0,
		orquestamcp.MCPRunSupervisorHTTPPathV0,
		orquestamcp.MCPRunControlHTTPPathV0,
		orquestamcp.MCPRunQueuePriorityHTTPPathV0,
		orquestamcp.MCPQueueGlobalStatusHTTPPathV0,
		orquestamcp.MCPDirectorStatsHTTPPathV0,
		orquestamcp.MCPDomainWorkHTTPPathV0,
		orquestamcp.MCPDomainWorkStatusHTTPPathV0,
		orquestamcp.MCPExternalWorkDryRunHTTPPathV0,
		orquestamcp.MCPExternalWorkRunHTTPPathV0,
		orquestamcp.MCPCodebaseQueryHTTPPathV0,
		orquestamcp.MCPCodebaseStatusHTTPPathV0,
	} {
		if !paths[want] {
			t.Fatalf("toolbelt HTTP no contiene ruta registrada %q", want)
		}
	}
}

func TestCodexToolbeltSourceV0DerivaMCPDeRegistroYPublicaDirectedQuery(t *testing.T) {
	registered := codexRegisteredMCPToolNamesV0()
	for _, entry := range codexToolbeltMCPEntriesV0() {
		if entry.Name == orquestamcp.MCPOperatorOperationsResourceNameV0 {
			continue
		}
		if _, ok := registered[entry.Name]; !ok {
			t.Fatalf("toolbelt MCP no registrado: %+v", entry)
		}
		if entry.Status == codexToolbeltStatusTransportMissingV0 {
			t.Fatalf("toolbelt MCP stale: %+v", entry)
		}
	}
	if !codexToolbeltMCPNameForTestV0(operator.OperatorMCPDirectedQueryToolV0) {
		t.Fatalf("toolbelt MCP no publica %s", operator.OperatorMCPDirectedQueryToolV0)
	}
}

func TestCodexToolbeltSourceV0NoFiltraDatosSensiblesEnHints(t *testing.T) {
	joined := codexToolbeltHTTPHintV0() + "\n" + codexToolbeltMCPHintV0()
	for _, forbidden := range []string{"HOME", "token", "secret", "prompt completo", "transcript", "/home/"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("toolbelt hint contiene dato prohibido %q: %s", forbidden, joined)
		}
	}
	for _, want := range []string{
		"transporte_http_si_servidor_vivo",
		"puede_devolver_mcp_transport_tool_unbound_si_falta_puerto",
		"resource_registrado_con_subtools_operador",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("toolbelt hint no distingue estado %q: %s", want, joined)
		}
	}
}

func codexToolbeltHTTPPathsForTestV0() map[string]bool {
	out := map[string]bool{}
	for _, entry := range codexToolbeltHTTPEntriesV0() {
		out[entry.Path] = true
	}
	return out
}

func codexToolbeltMCPNameForTestV0(name string) bool {
	for _, entry := range codexToolbeltMCPEntriesV0() {
		if entry.Name == name {
			return true
		}
	}
	return false
}
