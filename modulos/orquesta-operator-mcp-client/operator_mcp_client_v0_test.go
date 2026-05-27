package orquestaoperatormcpclient

import (
	"context"
	"errors"
	"testing"
	"time"

	operator "orquesta/modulos/orquesta-operator-mcp"
)

func TestOperatorMCPClientConnectorV0ImplementaCuatroPuertos(t *testing.T) {
	fake := &fakeGenericMCPClientV0{}
	connector := NewOperatorMCPClientConnectorV0(OperatorMCPClientConfigV0{Client: fake})

	status, err := connector.QueryOperatorStatusV0(operator.OperatorStatusQueryV0{
		RequestRef:         "req-status",
		SubjectRef:         "run-1",
		StatusConnectorRef: "status-ref",
	})
	if err != nil || status.Status != "healthy" {
		t.Fatalf("status inesperado: status=%+v err=%v", status, err)
	}

	burst, err := connector.RequestOperatorSupervisedBurstV0(operator.OperatorSupervisedBurstRequestV0{
		RequestRef:        "req-burst",
		RunRef:            "run-1",
		BurstConnectorRef: "burst-ref",
		SupervisionRef:    "supervision-1",
		MaxSteps:          4,
	})
	if err != nil || burst.BurstRef != "burst-ref-remote" {
		t.Fatalf("burst inesperado: burst=%+v err=%v", burst, err)
	}

	outbox, err := connector.ListOperatorPendingOutboxV0(operator.OperatorPendingOutboxQueryV0{
		RequestRef:         "req-outbox",
		SubjectRef:         "run-1",
		OutboxConnectorRef: "outbox-ref",
		Limit:              5,
	})
	if err != nil || outbox.PendingCount != 1 {
		t.Fatalf("outbox inesperado: outbox=%+v err=%v", outbox, err)
	}

	query, err := connector.RaiseOperatorDirectedQueryV0(operator.OperatorDirectedQueryV0{
		QueryRef:          "query-1",
		TargetRef:         "director-1",
		QueryConnectorRef: "query-ref",
		Question:          "Falta alguna decision publica?",
	})
	if err != nil || !query.Accepted {
		t.Fatalf("query inesperada: query=%+v err=%v", query, err)
	}
}

func TestOperatorMCPClientConnectorV0UsaToolNamesYRefsConfigurables(t *testing.T) {
	fake := &fakeGenericMCPClientV0{}
	connector := NewOperatorMCPClientConnectorV0(OperatorMCPClientConfigV0{
		Client: fake,
		ToolNames: OperatorMCPClientToolNamesV0{
			Status:        "hermes.status",
			Burst:         "hermes.burst",
			Outbox:        "hermes.outbox",
			DirectedQuery: "hermes.query",
		},
		ConnectorRefs: OperatorMCPClientConnectorRefsV0{
			Status:        "hermes-status-ref",
			Burst:         "hermes-burst-ref",
			Outbox:        "hermes-outbox-ref",
			DirectedQuery: "hermes-query-ref",
		},
	})

	_, _ = connector.QueryOperatorStatusV0(operator.OperatorStatusQueryV0{
		RequestRef:         "req-status",
		SubjectRef:         "run-1",
		StatusConnectorRef: "ignored-status-ref",
	})
	_, _ = connector.RequestOperatorSupervisedBurstV0(operator.OperatorSupervisedBurstRequestV0{
		RequestRef:        "req-burst",
		RunRef:            "run-1",
		BurstConnectorRef: "ignored-burst-ref",
		SupervisionRef:    "supervision-1",
		MaxSteps:          2,
	})
	_, _ = connector.ListOperatorPendingOutboxV0(operator.OperatorPendingOutboxQueryV0{
		RequestRef:         "req-outbox",
		SubjectRef:         "run-1",
		OutboxConnectorRef: "ignored-outbox-ref",
		Limit:              2,
	})
	_, _ = connector.RaiseOperatorDirectedQueryV0(operator.OperatorDirectedQueryV0{
		QueryRef:          "query-1",
		TargetRef:         "director-1",
		QueryConnectorRef: "ignored-query-ref",
		Question:          "Decision?",
	})

	assertCallV0(t, fake.calls[0], "hermes.status", "hermes-status-ref")
	assertCallV0(t, fake.calls[1], "hermes.burst", "hermes-burst-ref")
	assertCallV0(t, fake.calls[2], "hermes.outbox", "hermes-outbox-ref")
	assertCallV0(t, fake.calls[3], "hermes.query", "hermes-query-ref")
}

func TestOperatorMCPClientConnectorV0PropagaErroresPublicosEstables(t *testing.T) {
	connector := NewOperatorMCPClientConnectorV0(OperatorMCPClientConfigV0{})
	_, err := connector.QueryOperatorStatusV0(operator.OperatorStatusQueryV0{})
	assertPublicCodeV0(t, err, operator.ErrOperatorMCPConnectorUnavailableV0)

	fake := &fakeGenericMCPClientV0{err: errors.New("remote down")}
	connector = NewOperatorMCPClientConnectorV0(OperatorMCPClientConfigV0{Client: fake})
	_, err = connector.QueryOperatorStatusV0(operator.OperatorStatusQueryV0{})
	assertPublicCodeV0(t, err, operator.ErrOperatorMCPPortErrorV0)

	fake = &fakeGenericMCPClientV0{errorCode: operator.ErrOperatorMCPConnectorUnavailableV0}
	connector = NewOperatorMCPClientConnectorV0(OperatorMCPClientConfigV0{Client: fake})
	_, err = connector.ListOperatorPendingOutboxV0(operator.OperatorPendingOutboxQueryV0{})
	assertPublicCodeV0(t, err, operator.ErrOperatorMCPConnectorUnavailableV0)

	fake = &fakeGenericMCPClientV0{errorCode: "remote_private_stacktrace"}
	connector = NewOperatorMCPClientConnectorV0(OperatorMCPClientConfigV0{Client: fake})
	_, err = connector.RaiseOperatorDirectedQueryV0(operator.OperatorDirectedQueryV0{})
	assertPublicCodeV0(t, err, operator.ErrOperatorMCPPortErrorV0)
}

func TestOperatorMCPClientConnectorV0AplicaDeadlineYMapeaCancelacion(t *testing.T) {
	fake := &fakeGenericMCPClientV0{}
	connector := NewOperatorMCPClientConnectorV0(OperatorMCPClientConfigV0{
		Client:  fake,
		Timeout: 50 * time.Millisecond,
	})
	_, err := connector.QueryOperatorStatusV0(operator.OperatorStatusQueryV0{})
	if err != nil {
		t.Fatalf("status con deadline inesperado: %v", err)
	}
	if !fake.sawDeadline {
		t.Fatalf("CallToolV0 no recibio deadline")
	}

	fake = &fakeGenericMCPClientV0{err: context.DeadlineExceeded}
	connector = NewOperatorMCPClientConnectorV0(OperatorMCPClientConfigV0{
		Client:  fake,
		Timeout: time.Second,
	})
	_, err = connector.ListOperatorPendingOutboxV0(operator.OperatorPendingOutboxQueryV0{})
	assertPublicCodeV0(t, err, operator.ErrOperatorMCPTimeoutV0)

	parent, cancel := context.WithCancel(context.Background())
	cancel()
	fake = &fakeGenericMCPClientV0{returnContextErr: true}
	connector = NewOperatorMCPClientConnectorV0(OperatorMCPClientConfigV0{
		Client:         fake,
		Timeout:        time.Second,
		ContextFactory: func() context.Context { return parent },
	})
	_, err = connector.RaiseOperatorDirectedQueryV0(operator.OperatorDirectedQueryV0{})
	assertPublicCodeV0(t, err, operator.ErrOperatorMCPCancelledV0)
}

type fakeGenericMCPClientV0 struct {
	calls            []fakeMCPClientCallV0
	err              error
	errorCode        string
	sawDeadline      bool
	returnContextErr bool
}

type fakeMCPClientCallV0 struct {
	toolName     string
	connectorRef string
}

func (fake *fakeGenericMCPClientV0) CallToolV0(
	ctx context.Context,
	toolName string,
	input any,
	output any,
) error {
	_, fake.sawDeadline = ctx.Deadline()
	fake.calls = append(fake.calls, fakeMCPClientCallV0{
		toolName:     toolName,
		connectorRef: connectorRefFromInputV0(input),
	})
	if fake.returnContextErr {
		return ctx.Err()
	}
	if fake.err != nil {
		return fake.err
	}
	result := output.(*operatorMCPClientToolResultV0)
	if fake.errorCode != "" {
		result.Estado = "error"
		result.ErrorCode = fake.errorCode
		return nil
	}
	result.Estado = "ok"
	switch toolName {
	case operator.OperatorMCPStatusToolNameV0, "hermes.status":
		result.Status = &operator.OperatorMCPStatusResultV0{Status: "healthy"}
	case operator.OperatorMCPBurstToolNameV0, "hermes.burst":
		result.Burst = &operator.OperatorMCPBurstResultV0{BurstRef: "burst-ref-remote", ExecutedSteps: 2}
	case operator.OperatorMCPOutboxToolNameV0, "hermes.outbox":
		result.Outbox = &operator.OperatorMCPOutboxResultV0{
			PendingCount: 1,
			Items: []operator.OperatorMCPOutboxItemV0{{
				MessageRef: "message-ref-1",
				Kind:       "director_question",
			}},
		}
	default:
		result.DirectedQuery = &operator.OperatorMCPDirectedQueryResultV0{Accepted: true}
	}
	return nil
}

func connectorRefFromInputV0(input any) string {
	switch typed := input.(type) {
	case operator.OperatorStatusQueryV0:
		return typed.StatusConnectorRef
	case operator.OperatorSupervisedBurstRequestV0:
		return typed.BurstConnectorRef
	case operator.OperatorPendingOutboxQueryV0:
		return typed.OutboxConnectorRef
	case operator.OperatorDirectedQueryV0:
		return typed.QueryConnectorRef
	default:
		return ""
	}
}

func assertCallV0(t *testing.T, call fakeMCPClientCallV0, toolName string, connectorRef string) {
	t.Helper()
	if call.toolName != toolName || call.connectorRef != connectorRef {
		t.Fatalf("call inesperada: got=%+v want tool=%s ref=%s", call, toolName, connectorRef)
	}
}

func assertPublicCodeV0(t *testing.T, err error, code string) {
	t.Helper()
	got, ok := operator.PublicOperatorMCPErrorCodeV0(err)
	if !ok || got != code {
		t.Fatalf("error publico inesperado: got=%q ok=%v err=%v want=%q", got, ok, err, code)
	}
}
