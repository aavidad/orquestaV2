package orquestacionnucleoapp

import (
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorsupervisor "orquesta/modulos/orquesta-director-supervisor"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
)

type ProgressiveLoopStatusV0 string

const (
	ProgressiveLoopStatusWaitUnhandledOutboxV0 ProgressiveLoopStatusV0 = "wait_unhandled_outbox"
	ProgressiveLoopStatusQuiescentV0           ProgressiveLoopStatusV0 = "quiescent"
	ProgressiveLoopStatusWaitExternalV0        ProgressiveLoopStatusV0 = "wait_external"
	ProgressiveLoopStatusNeedsDirectorV0       ProgressiveLoopStatusV0 = "needs_director"
	ProgressiveLoopStatusBlockedV0             ProgressiveLoopStatusV0 = "blocked"
	ProgressiveLoopStatusStopMaxStepsV0        ProgressiveLoopStatusV0 = "stop_max_steps"
	ProgressiveLoopStatusMaxBurstsV0           ProgressiveLoopStatusV0 = "max_bursts"
	ProgressiveLoopStatusStopErrorV0           ProgressiveLoopStatusV0 = "stop_error"
	ProgressiveLoopStatusRunPausedV0           ProgressiveLoopStatusV0 = "run_paused"
	ProgressiveLoopStatusRunStopRequestedV0    ProgressiveLoopStatusV0 = "run_stop_requested"
	ProgressiveLoopStatusRunCanceledV0         ProgressiveLoopStatusV0 = "run_canceled"
	ProgressiveLoopStatusRunTerminalV0         ProgressiveLoopStatusV0 = "run_terminal"
)

type OutboxDispatcherBindingV0 struct {
	TargetPort  string
	MessageType string
	Reader      orquestaoutboxdispatch.PendingOutboxReaderPortV0
	Claimer     orquestaoutboxdispatch.OutboxDispatchClaimerPortV0
	Executor    orquestaoutboxdispatch.OutboxDispatchExecutorPortV0
	Acker       orquestaoutboxdispatch.OutboxDispatchAckPortV0
}

type OutboxBatchDispatcherBindingV0 struct {
	TargetPort  string
	MessageType string
	MaxReady    int
	Reader      orquestaoutboxdispatch.PendingOutboxReaderPortV0
	Claimer     orquestaoutboxdispatch.OutboxDispatchClaimerPortV0
	Executor    OutboxDispatchBatchExecutorPortV0
	Acker       orquestaoutboxdispatch.OutboxDispatchAckPortV0
}

type ProgressiveLoopRequestV0 struct {
	RunRef               string
	OccurredAt           string
	MaxBursts            int
	MaxStepsPerBurst     int
	MaxDispatchesPerWait int
	CorrelationID        string
	EvidenceRefs         []string
	WaitAgentRefs        []string
	WaitScopeApplied     bool
	Dispatchers          []OutboxDispatcherBindingV0
	BatchDispatchers     []OutboxBatchDispatcherBindingV0
}

type ProgressiveLoopBurstV0 struct {
	BurstNumber  int
	Executed     int
	FinalAction  orquestadirectorsupervisor.DirectorSupervisorActionV0
	EvidenceRefs []string
}

type ProgressiveLoopResultV0 struct {
	Status             ProgressiveLoopStatusV0
	Run                orquestacoreworkflow.OrchestrationRunV0
	Bursts             []ProgressiveLoopBurstV0
	Dispatches         []OutboxDispatchOnceResultV0
	BatchDispatches    []OutboxDispatchBatchRunResultV0
	TotalExecutedSteps int
	FinalAction        orquestadirectorsupervisor.DirectorSupervisorActionV0
	FirstPendingRefs   []string
	FirstPendingCount  int
	PendingOutboxRefs  []string
	PendingOutboxCount int
}
