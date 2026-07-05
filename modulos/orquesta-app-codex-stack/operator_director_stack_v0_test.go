package orquestaappcodexstack

import (
	"context"
	"strings"
	"testing"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	channel "orquesta/modulos/orquesta-operator-director-channel"
)

func TestBuildStackOperatorDirectorMessageV0CableaServicioRealConQueueYControlSeguro(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	executor := orquestamcp.MCPTransportOperatorDirectorMessageExecutorV0{
		Service: stack.MCPTransportBindings.OperatorDirectorMessage,
	}

	queue := executor.Execute(context.Background(), channel.OperatorMessageV0{
		RequestRef: "request-ref-stack-operator-director-queue-001",
		TargetRef:  "director-ref-stack-operator-director-queue-001",
		Body:       "cola de runs",
	})
	if queue.Estado != orquestamcp.MCPOperatorDirectorMessageEstadoOKV0 ||
		queue.Response == nil ||
		queue.Response.Status != "accepted" ||
		queue.Response.Summary != "operator_director_queue_ready" ||
		!strings.Contains(queue.Response.ResponseText, `"action":"rank"`) {
		t.Fatalf("queue no funcional: %+v", queue)
	}

	status := executor.Execute(context.Background(), channel.OperatorMessageV0{
		RequestRef: "request-ref-stack-operator-director-status-001",
		TargetRef:  "director-ref-stack-operator-director-status-001",
		Intent:     "status",
		Body:       "estado",
	})
	if status.Estado != orquestamcp.MCPOperatorDirectorMessageEstadoOKV0 ||
		status.Response == nil ||
		status.Response.Status != "accepted" ||
		status.Response.Summary != "operator_director_status_ready" {
		t.Fatalf("status no funcional: %+v", status)
	}

	observe := executor.Execute(context.Background(), channel.OperatorMessageV0{
		RequestRef: "request-ref-stack-operator-director-observe-001",
		TargetRef:  "director-ref-stack-operator-director-observe-001",
		Intent:     "observe_run",
		Body:       "observa",
	})
	if observe.Estado != orquestamcp.MCPOperatorDirectorMessageEstadoOKV0 ||
		observe.Response == nil ||
		observe.Response.Status != "blocked" ||
		observe.Response.Summary != "operator_director_observe_run_ref_required" {
		t.Fatalf("observe sin run_ref debe bloquear: %+v", observe)
	}

	for _, item := range []struct {
		name    string
		body    string
		summary string
	}{
		{"launch", "launch nueva app", "operator_director_launch_requires_structured_payload"},
		{"handoff", "handoff goal", "operator_director_handoff_requires_structured_payload"},
	} {
		result := executor.Execute(context.Background(), channel.OperatorMessageV0{
			RequestRef: "request-ref-stack-operator-director-" + item.name + "-001",
			TargetRef:  "director-ref-stack-operator-director-" + item.name + "-001",
			Body:       item.body,
		})
		if result.Estado != orquestamcp.MCPOperatorDirectorMessageEstadoOKV0 ||
			result.Response == nil ||
			result.Response.Status != "blocked" ||
			result.Response.Summary != item.summary {
			t.Fatalf("%s debe bloquear seguro: %+v", item.name, result)
		}
	}

	control := executor.Execute(context.Background(), channel.OperatorMessageV0{
		RequestRef: "request-ref-stack-operator-director-control-001",
		TargetRef:  "run-ref-stack-operator-director-control-001",
		Intent:     "instruction",
		Body:       "stop run-ref-stack-operator-director-control-001",
	})
	if control.Estado != orquestamcp.MCPOperatorDirectorMessageEstadoOKV0 ||
		control.Response == nil ||
		control.Response.Status != "blocked" ||
		control.Response.Summary != "operator_director_control_confirmation_required" {
		t.Fatalf("control sin confirmacion debe bloquear: %+v", control)
	}
}
