package orquestamcp

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func TestMCPRunQueuePriorityExecutorV0RankDelegaEnReaderYRanking(t *testing.T) {
	now := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	reader := &fakeMCPRunQueueReaderV0{
		candidates: []orquestarunqueue.RunSchedulingCandidateV0{
			runQueueCandidateMCPTestV0("same-priority-new", "app-1", "ready", 10, now.Add(-10*time.Minute)),
			runQueueCandidateMCPTestV0("high-priority-new", "app-2", "ready", 20, now.Add(-10*time.Minute)),
			runQueueCandidateMCPTestV0("same-priority-aged", "app-3", "ready", 10, now.Add(-2*time.Hour)),
			runQueueCandidateMCPTestV0("paused-run", "app-4", "paused", 100, now.Add(-3*time.Hour)),
		},
	}
	executor := MCPRunQueuePriorityToolExecutorV0{Reader: reader}

	result, err := executor.Execute(context.Background(), MCPRunQueuePriorityToolInputV0{
		RequestID:  "req-1",
		Action:     " rank ",
		QueueRef:   " global ",
		AppRefs:    []string{" app-1 ", "app-2", "app-1"},
		Limit:      7,
		OccurredAt: now.Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if result.Estado != MCPRunQueuePriorityEstadoOKV0 || result.Count != 3 {
		t.Fatalf("resultado inesperado: %+v", result)
	}
	got := []string{result.Ranked[0].RunRef, result.Ranked[1].RunRef, result.Ranked[2].RunRef}
	want := []string{"high-priority-new", "same-priority-aged", "same-priority-new"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ranking inesperado: got=%#v want=%#v", got, want)
	}
	if result.Ranked[1].AgingBoost == 0 || result.Ranked[0].AgingBoost >= result.Ranked[1].AgingBoost {
		t.Fatalf("aging no aplicado como desempate secundario: %+v", result.Ranked)
	}
	if reader.request.QueueRef != "global" || reader.request.Limit != 7 {
		t.Fatalf("request no normalizada: %+v", reader.request)
	}
	if !reflect.DeepEqual(reader.request.AppRefs, []string{"app-1", "app-2"}) {
		t.Fatalf("app_refs no compactas: %#v", reader.request.AppRefs)
	}
}

func TestMCPRunQueuePriorityExecutorV0ValidaActionYPuerto(t *testing.T) {
	result, err := MCPRunQueuePriorityToolExecutorV0{}.Execute(
		context.Background(),
		MCPRunQueuePriorityToolInputV0{Action: "restart", QueueRef: "global"},
	)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Estado != MCPRunQueuePriorityEstadoErrorV0 ||
		result.Errores[0].Code != "action_no_soportada" {
		t.Fatalf("error inesperado: %+v", result)
	}

	result, err = MCPRunQueuePriorityToolExecutorV0{}.Execute(
		context.Background(),
		MCPRunQueuePriorityToolInputV0{Action: "rank", QueueRef: "global"},
	)
	if err != nil {
		t.Fatalf("execute sin reader: %v", err)
	}
	if result.Errores[0].Code != "run_queue_reader_no_disponible" {
		t.Fatalf("reader nil no validado: %+v", result)
	}

	result, err = MCPRunQueuePriorityToolExecutorV0{}.Execute(
		context.Background(),
		MCPRunQueuePriorityToolInputV0{
			RequestID:     "request-ref-run-queue-priority-writer-001",
			Action:        "set_priority",
			RunRef:        "run-1",
			PriorityScore: 30,
		},
	)
	if err != nil {
		t.Fatalf("execute set_priority sin writer: %v", err)
	}
	if result.Errores[0].Code != "run_queue_writer_no_disponible" {
		t.Fatalf("writer nil no validado: %+v", result)
	}
}

func TestMCPRunQueuePriorityExecutorV0RankExponeIntentoActivoV0(t *testing.T) {
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	group := orquestarunqueue.RunQueueAttemptGroupV0{
		ConsumerRef:  "consumer",
		ObjectiveRef: "objective",
		WorkItemRef:  "topic-001",
		WriteSetRefs: []string{"topic/001"},
	}
	reader := &fakeMCPRunQueueReaderV0{
		candidates: []orquestarunqueue.RunSchedulingCandidateV0{
			{
				RunRef:        "run-original",
				AppRef:        "app",
				Status:        "ready",
				PriorityScore: 1,
				UpdatedAt:     now.Add(-20 * time.Minute),
				AttemptGroup:  group,
			},
			{
				RunRef:           "run-rescue",
				AppRef:           "app",
				Status:           "ready",
				PriorityScore:    9,
				UpdatedAt:        now,
				AttemptGroup:     group,
				ParentRunRef:     "run-original",
				SupersedesRunRef: "run-original",
				RescueReason:     "estado_incierto",
			},
		},
	}

	result, err := (MCPRunQueuePriorityToolExecutorV0{Reader: reader}).Execute(
		context.Background(),
		MCPRunQueuePriorityToolInputV0{Action: "rank", OccurredAt: now.Format(time.RFC3339)},
	)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if len(result.Ranked) != 2 ||
		result.Ranked[0].RunRef != "run-rescue" ||
		result.Ranked[0].ActiveAttemptRef != "run-rescue" ||
		result.Ranked[0].ParentRunRef != "run-original" ||
		result.Ranked[0].SupersedesRunRef != "run-original" ||
		result.Ranked[0].RescueReason != "estado_incierto" ||
		result.Ranked[0].WorkItemRef != "topic-001" {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPRunQueuePriorityExecutorV0RankExponeTerminalesSoloSiSePidenV0(t *testing.T) {
	now := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)
	reader := &fakeMCPRunQueueReaderV0{
		candidates: []orquestarunqueue.RunSchedulingCandidateV0{
			runQueueCandidateMCPTestV0("run-ready", "app", "ready", 10, now),
			{
				RunRef:        "request-ref-autoprogramming-backlog-scanner-15eeecb9",
				AppRef:        "app-ref-autoprogramming",
				Status:        "stopped",
				PriorityScore: 99,
				RescueReason:  "idle_self_improvement_suppressed_by_domain_session",
				EvidenceRefs:  []string{"evidence-ref-idle-self-improvement-domain-session"},
			},
		},
	}

	result, err := (MCPRunQueuePriorityToolExecutorV0{Reader: reader}).Execute(
		context.Background(),
		MCPRunQueuePriorityToolInputV0{
			Action:               "rank",
			IncludeNonExecutable: true,
			OccurredAt:           now.Format(time.RFC3339),
		},
	)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !reader.request.IncludeNonExecutable ||
		result.Count != 1 ||
		len(result.Ranked) != 1 ||
		result.Ranked[0].RunRef != "run-ready" ||
		len(result.Terminal) != 1 ||
		result.Terminal[0].RescueReason != "idle_self_improvement_suppressed_by_domain_session" {
		t.Fatalf("result=%+v request=%+v", result, reader.request)
	}
}

func TestMCPRunQueuePriorityExecutorV0SetPriorityDelegaEnWriter(t *testing.T) {
	writer := &fakeMCPRunQueueWriterV0{}
	executor := MCPRunQueuePriorityToolExecutorV0{Writer: writer}

	result, err := executor.Execute(context.Background(), MCPRunQueuePriorityToolInputV0{
		RequestID:        "req-set-priority-001",
		Action:           " set_priority ",
		QueueRef:         "global",
		RunRef:           " run-priority-001 ",
		AppRef:           " app-priority-001 ",
		Status:           " canceled ",
		PriorityScore:    77,
		ConsumerRef:      " consumer ",
		ObjectiveRef:     " objective ",
		WorkItemRef:      " topic-001 ",
		WriteSetRefs:     []string{" topic/001 ", "topic/001"},
		ParentRunRef:     " run-original ",
		SupersedesRunRef: " run-original ",
		RescueReason:     " estado_incierto ",
		RequestedBy:      " director ",
		EvidenceRefs:     []string{" evidence-1 ", "evidence-1"},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Estado != MCPRunQueuePriorityEstadoOKV0 ||
		result.Action != MCPRunQueuePriorityActionSetV0 ||
		result.Updated == nil ||
		result.Updated.RunRef != "run-priority-001" ||
		result.Updated.Status != "canceled" ||
		result.Updated.PriorityScore != 77 ||
		result.Updated.ParentRunRef != "run-original" ||
		result.Updated.ActiveAttemptRef != "run-priority-001" {
		t.Fatalf("result=%+v", result)
	}
	if writer.command.RunRef != "run-priority-001" ||
		writer.command.AppRef != "app-priority-001" ||
		writer.command.Status != "canceled" ||
		writer.command.PriorityScore != 77 ||
		writer.command.AttemptGroup.WorkItemRef != "topic-001" ||
		writer.command.ParentRunRef != "run-original" ||
		writer.command.SupersedesRunRef != "run-original" ||
		writer.command.RescueReason != "estado_incierto" ||
		len(writer.command.EvidenceRefs) != 1 {
		t.Fatalf("command=%+v", writer.command)
	}
}

func TestMCPRunQueuePriorityTransportV0InvocaExecutor(t *testing.T) {
	now := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	transport := newFakeMCPTransportV0()
	err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{
		RunQueuePriority: MCPRunQueuePriorityToolExecutorV0{
			Reader: &fakeMCPRunQueueReaderV0{
				candidates: []orquestarunqueue.RunSchedulingCandidateV0{
					runQueueCandidateMCPTestV0("run-a", "app-a", "ready", 5, now),
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("register transport: %v", err)
	}

	output, err := transport.CallToolV0(context.Background(), MCPRunQueuePriorityToolNameV0, MCPRunQueuePriorityToolInputV0{
		Action:     "rank",
		OccurredAt: now.Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	var result MCPRunQueuePriorityToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPRunQueuePriorityEstadoOKV0 ||
		len(result.Ranked) != 1 ||
		result.Ranked[0].RunRef != "run-a" {
		t.Fatalf("resultado transporte inesperado: %+v", result)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, output, 800)
}

func TestMCPRunQueuePriorityTransportV0QuedaOptInSinPuerto(t *testing.T) {
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{}); err != nil {
		t.Fatalf("register transport: %v", err)
	}
	output, err := transport.CallToolV0(context.Background(), MCPRunQueuePriorityToolNameV0, MCPRunQueuePriorityToolInputV0{})
	if err != nil {
		t.Fatalf("call unbound: %v", err)
	}
	var result MCPTransportToolErrorV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode unbound: %v", err)
	}
	if result.ErrorCode != MCPTransportToolUnboundV0 {
		t.Fatalf("debe quedar opt-in: %+v", result)
	}
}

type fakeMCPRunQueueReaderV0 struct {
	request    orquestarunqueue.RunQueueReadRequestV0
	candidates []orquestarunqueue.RunSchedulingCandidateV0
}

type fakeMCPRunQueueWriterV0 struct {
	command orquestarunqueue.RunQueuePriorityCommandV0
}

func (fake *fakeMCPRunQueueReaderV0) ListRunSchedulingCandidatesV0(
	_ context.Context,
	request orquestarunqueue.RunQueueReadRequestV0,
) ([]orquestarunqueue.RunSchedulingCandidateV0, error) {
	fake.request = request
	return append([]orquestarunqueue.RunSchedulingCandidateV0(nil), fake.candidates...), nil
}

func (fake *fakeMCPRunQueueWriterV0) SetRunPriorityV0(
	_ context.Context,
	command orquestarunqueue.RunQueuePriorityCommandV0,
) (orquestarunqueue.RunSchedulingCandidateV0, error) {
	fake.command = command
	status := command.Status
	if status == "" {
		status = "ready"
	}
	return orquestarunqueue.RunSchedulingCandidateV0{
		RunRef:           command.RunRef,
		AppRef:           command.AppRef,
		Status:           status,
		PriorityScore:    command.PriorityScore,
		AttemptGroup:     command.AttemptGroup,
		ParentRunRef:     command.ParentRunRef,
		SupersedesRunRef: command.SupersedesRunRef,
		RescueReason:     command.RescueReason,
		EvidenceRefs:     command.EvidenceRefs,
	}, nil
}

func runQueueCandidateMCPTestV0(
	runRef string,
	appRef string,
	status string,
	priority int,
	updatedAt time.Time,
) orquestarunqueue.RunSchedulingCandidateV0 {
	return orquestarunqueue.RunSchedulingCandidateV0{
		RunRef:        runRef,
		AppRef:        appRef,
		Status:        status,
		PriorityScore: priority,
		UpdatedAt:     updatedAt,
		EvidenceRefs:  []string{"evidence-1", " evidence-1 ", "evidence-2"},
	}
}
