package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestRunProgressiveLoopV0StopRequestedCompletaSinAgentesVivos(t *testing.T) {
	runRef := "run-nucleo-progressive-control-complete-stop-001"
	terminal := &recordingRunControlTerminalWriterV0{}
	result, err := ServiceV0{
		RunStore:           NewInMemoryRunStoreV0(mustActiveProgrammingRunV0(t, runRef)),
		EventSink:          NewInMemoryEventSinkV0(),
		CandidateProvider:  StaticCandidateProviderV0{},
		RunControl:         sequenceRunControlForTestV0(stopRequestedRunControlStateForTestV0(runRef)),
		RunControlTerminal: terminal,
		OutboxLedger:       NewInMemoryOutboxLedgerV0(),
		MaxCommands:        4,
	}.RunProgressiveLoopV0(context.Background(), ProgressiveLoopRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-05-11T09:30:00Z",
		MaxBursts:            1,
		MaxStepsPerBurst:     1,
		MaxDispatchesPerWait: 1,
		CorrelationID:        "corr-run-control-complete-stop-001",
	})
	if err != nil {
		t.Fatalf("progressive loop: %v", err)
	}
	if result.Status != ProgressiveLoopStatusRunTerminalV0 ||
		terminal.lastTargetV0() != orquestaruncontrol.RunControlStatusStoppedV0 {
		t.Fatalf("result=%+v terminal=%+v", result, terminal.commands)
	}
}

func TestRunProgressiveLoopV0CancelRequestedCompletaSinAgentesVivos(t *testing.T) {
	runRef := "run-nucleo-progressive-control-complete-cancel-001"
	terminal := &recordingRunControlTerminalWriterV0{}
	result, err := ServiceV0{
		RunStore:           NewInMemoryRunStoreV0(mustActiveProgrammingRunV0(t, runRef)),
		EventSink:          NewInMemoryEventSinkV0(),
		CandidateProvider:  StaticCandidateProviderV0{},
		RunControl:         sequenceRunControlForTestV0(cancelRequestedRunControlStateForTestV0(runRef)),
		RunControlTerminal: terminal,
		OutboxLedger:       NewInMemoryOutboxLedgerV0(),
		MaxCommands:        4,
	}.RunProgressiveLoopV0(context.Background(), ProgressiveLoopRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-05-11T09:31:00Z",
		MaxBursts:            1,
		MaxStepsPerBurst:     1,
		MaxDispatchesPerWait: 1,
		CorrelationID:        "corr-run-control-complete-cancel-001",
	})
	if err != nil {
		t.Fatalf("progressive loop: %v", err)
	}
	if result.Status != ProgressiveLoopStatusRunTerminalV0 ||
		terminal.lastTargetV0() != orquestaruncontrol.RunControlStatusCanceledV0 {
		t.Fatalf("result=%+v terminal=%+v", result, terminal.commands)
	}
}

func TestRunProgressiveLoopV0StopRequestedCompletaSiAgenteYaEntrego(t *testing.T) {
	runRef := "run-nucleo-progressive-control-complete-delivered-001"
	agentRef := "agent-ref-delivered-001"
	run := mustActiveProgrammingRunV0(t, runRef)
	run.Agents = []string{agentRef}
	run.StartedAgents = []string{agentRef}
	run.DeliveredAgents = []string{agentRef}
	ledger := NewInMemoryOutboxLedgerV0()
	terminal := &recordingRunControlTerminalWriterV0{}

	result, err := ServiceV0{
		RunStore:           NewInMemoryRunStoreV0(run),
		EventSink:          NewInMemoryEventSinkV0(),
		CandidateProvider:  StaticCandidateProviderV0{},
		RunControl:         sequenceRunControlForTestV0(stopRequestedRunControlStateForTestV0(runRef)),
		RunControlTerminal: terminal,
		OutboxLedger:       ledger,
		MaxCommands:        4,
	}.RunProgressiveLoopV0(context.Background(), ProgressiveLoopRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-05-11T09:31:30Z",
		MaxBursts:            1,
		MaxStepsPerBurst:     1,
		MaxDispatchesPerWait: 1,
		CorrelationID:        "corr-run-control-complete-delivered-001",
	})
	if err != nil {
		t.Fatalf("progressive loop: %v", err)
	}
	if result.Status != ProgressiveLoopStatusRunTerminalV0 ||
		terminal.lastTargetV0() != orquestaruncontrol.RunControlStatusStoppedV0 {
		t.Fatalf("result=%+v terminal=%+v", result, terminal.commands)
	}
	if pending := pendingOutboxRefsForTargetV0(t, ledger, runRef, orquestacoreworkflow.OutboxTargetAgentLauncherV0); len(pending) != 0 {
		t.Fatalf("pending agent_launcher=%v", pending)
	}
}

func TestRunProgressiveLoopV0NoCompletaConParadaPendiente(t *testing.T) {
	runRef := "run-nucleo-progressive-control-complete-pending-001"
	ledger := NewInMemoryOutboxLedgerV0()
	if _, issues := ledger.SavePending(context.Background(), []orquestacoreworkflow.OutboxMessageV0{
		{
			MessageID:   "outbox-run-control-stop-pending-001",
			RunID:       runRef,
			TargetPort:  orquestacoreworkflow.OutboxTargetAgentLauncherV0,
			MessageType: orquestacoreworkflow.OutboxMessageStopRuntimeAgentV0,
		},
	}); len(issues) > 0 {
		t.Fatalf("seed outbox issues=%+v", issues)
	}
	terminal := &recordingRunControlTerminalWriterV0{}

	result, err := ServiceV0{
		RunStore:           NewInMemoryRunStoreV0(mustActiveProgrammingRunV0(t, runRef)),
		EventSink:          NewInMemoryEventSinkV0(),
		CandidateProvider:  StaticCandidateProviderV0{},
		RunControl:         sequenceRunControlForTestV0(stopRequestedRunControlStateForTestV0(runRef)),
		RunControlTerminal: terminal,
		OutboxLedger:       ledger,
		MaxCommands:        4,
	}.RunProgressiveLoopV0(context.Background(), ProgressiveLoopRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-05-11T09:32:00Z",
		MaxBursts:            1,
		MaxStepsPerBurst:     1,
		MaxDispatchesPerWait: 1,
		CorrelationID:        "corr-run-control-complete-pending-001",
	})
	if err != nil {
		t.Fatalf("progressive loop: %v", err)
	}
	if result.Status != ProgressiveLoopStatusRunStopRequestedV0 ||
		len(terminal.commands) != 0 ||
		result.PendingOutboxCount != 1 {
		t.Fatalf("result=%+v terminal=%+v", result, terminal.commands)
	}
}

func TestRunProgressiveLoopV0StopRequestedCompletaTrasDrenarAgente(t *testing.T) {
	runRef := "run-nucleo-progressive-control-complete-drain-001"
	agentRef := "agent-ref-001"
	store := NewInMemoryRunStoreV0(mustActiveProgrammingRunV0(t, runRef))
	ledger := NewInMemoryOutboxLedgerV0()
	sink := NewInMemoryEventSinkV0()
	runtime := orquestaruntime.NewRuntimeFakeLifecycleV0()
	terminal := &recordingRunControlTerminalWriterV0{}

	launched := runProgressiveLoopWithFakeAgentRuntimeV0(t,
		ServiceV0{
			RunStore:  store,
			EventSink: sink,
			CandidateProvider: StaticCandidateProviderV0{Candidates: SchedulerCandidateSetV0{
				WorkCandidates: []orquestadirectorscheduler.SchedulableWorkCandidateV0{
					workCandidateWithAgentV0(runRef),
				},
			}},
			OutboxLedger: ledger,
			MaxCommands:  4,
		},
		store,
		sink,
		ledger,
		runtime,
		runRef,
	)
	if !containsNucleoRefV0(launched.Run.StartedAgents, agentRef) {
		t.Fatalf("started agents=%v", launched.Run.StartedAgents)
	}

	result, err := ServiceV0{
		RunStore:           store,
		EventSink:          sink,
		CandidateProvider:  StaticCandidateProviderV0{},
		RunControl:         sequenceRunControlForTestV0(stopRequestedRunControlStateForTestV0(runRef)),
		RunControlTerminal: terminal,
		OutboxLedger:       ledger,
		MaxCommands:        4,
	}.RunProgressiveLoopV0(context.Background(), ProgressiveLoopRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-05-11T09:32:00Z",
		MaxBursts:            1,
		MaxStepsPerBurst:     1,
		MaxDispatchesPerWait: 2,
		CorrelationID:        "corr-run-control-complete-drain-001",
		Dispatchers: []OutboxDispatcherBindingV0{
			runControlStopDispatcherForTestV0(store, sink, ledger, runtime),
		},
	})
	if err != nil {
		t.Fatalf("progressive loop: %v", err)
	}
	if result.Status != ProgressiveLoopStatusRunTerminalV0 ||
		terminal.lastTargetV0() != orquestaruncontrol.RunControlStatusStoppedV0 ||
		!containsNucleoRefV0(result.Run.ConfirmedStoppedAgents, agentRef) {
		t.Fatalf("result=%+v terminal=%+v", result, terminal.commands)
	}
	if pending := pendingOutboxRefsForTargetV0(t, ledger, runRef, orquestacoreworkflow.OutboxTargetAgentLauncherV0); len(pending) != 0 {
		t.Fatalf("pending agent_launcher=%v", pending)
	}
}

type recordingRunControlTerminalWriterV0 struct {
	commands []orquestaruncontrol.CompleteRunControlCommandV0
}

func (writer *recordingRunControlTerminalWriterV0) CompleteRunControlV0(
	_ context.Context,
	command orquestaruncontrol.CompleteRunControlCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	command = orquestaruncontrol.NormalizeCompleteRunControlCommandV0(command)
	writer.commands = append(writer.commands, command)
	return orquestaruncontrol.RunControlStateV0{
		RunRef:       command.RunRef,
		Status:       command.TargetStatus,
		EvidenceRefs: append([]string(nil), command.EvidenceRefs...),
	}, nil
}

func (writer *recordingRunControlTerminalWriterV0) lastTargetV0() orquestaruncontrol.RunControlStatusV0 {
	if len(writer.commands) == 0 {
		return ""
	}
	return writer.commands[len(writer.commands)-1].TargetStatus
}

func cancelRequestedRunControlStateForTestV0(runRef string) orquestaruncontrol.RunControlStateV0 {
	return orquestaruncontrol.RunControlStateV0{
		RunRef:             runRef,
		Status:             orquestaruncontrol.RunControlStatusCancelRequestedV0,
		CheckpointRecorded: true,
	}
}
