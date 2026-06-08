package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"
	"time"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestadirectorsupervisor "orquesta/modulos/orquesta-director-supervisor"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

const (
	CodexStackResidentDirectorStatusCompletedV0 = "completed"
	CodexStackResidentDirectorStatusIdleV0      = "idle"
)

type CodexStackResidentDirectorCommandV0 struct {
	QueueRef              string
	OccurredAt            string
	CorrelationID         string
	EvidenceRefs          []string
	MaxRunsPerTick        int
	MaxExecutions         int
	MaxActions            int
	MaxBursts             int
	MaxStepsPerBurst      int
	MaxDispatchesPerWait  int
	MaxCommands           int
	MaxOutboxPerCycle     int
	MaxDecisionCycles     int
	MaxExternalWaits      int
	RunQueueReadLimit     int
	WaitAgentRefs         []string
	WaitScopeApplied      bool
	RequestedBy           string
	QueueRankingPolicyNow time.Time
}

type CodexStackResidentDirectorResultV0 struct {
	Status          string
	RunRef          string
	RunRefs         []string
	ExecutedActions int
	EvidenceRefs    []string
	Loop            orquestacionnucleoapp.ResidentDirectorBriefingLoopResultV0
	Runs            []CodexStackResidentDirectorRunResultV0
}

type CodexStackResidentDirectorRunResultV0 struct {
	Status          string
	RunRef          string
	ExecutedActions int
	EvidenceRefs    []string
	Loop            orquestacionnucleoapp.ResidentDirectorBriefingLoopResultV0
	Closed          bool
}

func (stack StackV0) RunCodexStackResidentDirectorV0(
	ctx context.Context,
	command CodexStackResidentDirectorCommandV0,
) (CodexStackResidentDirectorResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	command = normalizeCodexStackResidentDirectorCommandV0(command, stack)
	if stack.Stores.RunQueue == nil {
		return CodexStackResidentDirectorResultV0{
			Status:       CodexStackResidentDirectorStatusIdleV0,
			EvidenceRefs: compactStringsV0(command.EvidenceRefs),
		}, nil
	}
	accumulator := &codexStackResidentDirectorAccumulatorV0{}
	tick, err := orquestaruncoordinator.CoordinateRunsTickV0(
		ctx,
		orquestaruncoordinator.RunCoordinatorDepsV0{
			QueueReader:   stack.Stores.RunQueue,
			QueueUpdater:  stack.Stores.RunQueue,
			ControlReader: stack.Stores.RunControl,
			Drainer: codexStackResidentRunDrainerV0{
				stack:       stack,
				command:     command,
				accumulator: accumulator,
			},
		},
		orquestaruncoordinator.RunCoordinatorTickCommandV0{
			QueueRef:             command.QueueRef,
			QueueLimit:           command.RunQueueReadLimit,
			MaxRuns:              command.MaxRunsPerTick,
			OccurredAt:           command.QueueRankingPolicyNow,
			CorrelationID:        command.CorrelationID,
			DrainLimits:          codexStackResidentDrainLimitsV0(command),
			RankingPolicy:        orquestarunqueue.DefaultRunQueueRankingPolicyV0(command.QueueRankingPolicyNow),
			ContinueOnDrainError: true,
		},
	)
	result := codexStackResidentResultFromCoordinatorV0(command, tick, accumulator)
	if err != nil {
		return result, err
	}
	return result, nil
}

func normalizeCodexStackResidentDirectorCommandV0(
	command CodexStackResidentDirectorCommandV0,
	stack StackV0,
) CodexStackResidentDirectorCommandV0 {
	queue := normalizeRunQueueConfigV0(stack.RunQueue)
	command.QueueRef = firstNonEmptyQueuedSourceV0(command.QueueRef, queue.QueueRef)
	command.OccurredAt = strings.TrimSpace(command.OccurredAt)
	if command.OccurredAt == "" {
		command.OccurredAt = stackNowV0(stack.Clock).Format(time.RFC3339)
	}
	command.CorrelationID = strings.TrimSpace(command.CorrelationID)
	command.EvidenceRefs = compactStringsV0(command.EvidenceRefs)
	command.WaitAgentRefs = compactStringsV0(command.WaitAgentRefs)
	command.RequestedBy = firstNonEmptyQueuedSourceV0(command.RequestedBy, "orquesta-codex-stack-resident-director")
	if command.MaxRunsPerTick <= 0 {
		command.MaxRunsPerTick = queue.MaxRunsPerTick
	}
	if command.MaxRunsPerTick <= 0 {
		command.MaxRunsPerTick = 1
	}
	if command.MaxExecutions <= 0 {
		command.MaxExecutions = command.MaxRunsPerTick
	}
	if command.MaxExecutions > 0 && command.MaxRunsPerTick > command.MaxExecutions {
		command.MaxRunsPerTick = command.MaxExecutions
	}
	if command.MaxActions <= 0 {
		command.MaxActions = 1
	}
	limits := codexStackResidentDrainLimitsV0(command)
	command.MaxBursts = limits.MaxBursts
	command.MaxStepsPerBurst = limits.MaxStepsPerBurst
	command.MaxDispatchesPerWait = limits.MaxDispatchesPerWait
	command.MaxCommands = limits.MaxCommands
	command.MaxOutboxPerCycle = limits.MaxOutboxPerCycle
	command.MaxDecisionCycles = limits.MaxDecisionCycles
	command.MaxExternalWaits = limits.MaxExternalWaits
	if command.RunQueueReadLimit <= 0 {
		command.RunQueueReadLimit = queue.QueueLimit
	}
	if command.RunQueueReadLimit <= 0 {
		command.RunQueueReadLimit = command.MaxRunsPerTick
	}
	if command.QueueRankingPolicyNow.IsZero() {
		if parsed, err := time.Parse(time.RFC3339, command.OccurredAt); err == nil {
			command.QueueRankingPolicyNow = parsed
		}
	}
	if command.QueueRankingPolicyNow.IsZero() {
		command.QueueRankingPolicyNow = stackNowV0(stack.Clock)
	}
	return command
}

func codexStackResidentDrainLimitsV0(
	command CodexStackResidentDirectorCommandV0,
) orquestaruncoordinator.RunDrainLimitsV0 {
	return normalizeGlobalDrainLimitsV0(orquestaruncoordinator.RunDrainLimitsV0{
		MaxBursts:            command.MaxBursts,
		MaxStepsPerBurst:     command.MaxStepsPerBurst,
		MaxDispatchesPerWait: command.MaxDispatchesPerWait,
		MaxCommands:          command.MaxCommands,
		MaxOutboxPerCycle:    command.MaxOutboxPerCycle,
		MaxDecisionCycles:    command.MaxDecisionCycles,
		MaxExternalWaits:     command.MaxExternalWaits,
	})
}

type codexStackResidentDirectorAccumulatorV0 struct {
	Runs []CodexStackResidentDirectorRunResultV0
}

type codexStackResidentRunDrainerV0 struct {
	stack       StackV0
	command     CodexStackResidentDirectorCommandV0
	accumulator *codexStackResidentDirectorAccumulatorV0
}

func (drainer codexStackResidentRunDrainerV0) DrainRunV0(
	ctx context.Context,
	request orquestaruncoordinator.RunDrainRequestV0,
) (orquestaruncoordinator.RunDrainResultV0, error) {
	result, err := drainer.stack.runCodexStackResidentDirectorRunV0(ctx, drainer.command, request)
	if drainer.accumulator != nil {
		drainer.accumulator.Runs = append(drainer.accumulator.Runs, result)
	}
	return orquestaruncoordinator.RunDrainResultV0{
		RunRef:       request.RunRef,
		AppRef:       request.AppRef,
		Outcome:      result.Status,
		QueueStatus:  drainer.stack.codexStackResidentQueueStatusV0(ctx, result),
		EvidenceRefs: result.EvidenceRefs,
		Diagnostics: []orquestaruncoordinator.RunDrainDiagnosticV0{{
			Kind:          "resident_director_loop",
			Status:        result.Status,
			RunRef:        request.RunRef,
			ExecutedSteps: result.ExecutedActions,
			EvidenceRefs:  result.EvidenceRefs,
		}},
	}, err
}

func (stack StackV0) runCodexStackResidentDirectorRunV0(
	ctx context.Context,
	command CodexStackResidentDirectorCommandV0,
	request orquestaruncoordinator.RunDrainRequestV0,
) (CodexStackResidentDirectorRunResultV0, error) {
	drainRequest := stackDrainRequestFromCoordinatorV0(request)
	if len(command.WaitAgentRefs) > 0 {
		drainRequest.WaitAgentRefs = append([]string(nil), command.WaitAgentRefs...)
	}
	enriched, err := stack.enrichQueuedOperationalDirectorDrainRequestV0(ctx, drainRequest)
	if err != nil {
		return CodexStackResidentDirectorRunResultV0{
			Status:       "error",
			RunRef:       request.RunRef,
			EvidenceRefs: compactStringsV0(command.EvidenceRefs),
		}, err
	}
	continueRequest := codexStackResidentContinueRequestV0(command, enriched)
	runtime, err := orquestaappdirectorservice.BuildContinueAppDirectorLoopRuntimeV0(ctx, continueRequest, stack.Ports)
	if err != nil {
		return CodexStackResidentDirectorRunResultV0{
			Status:       "error",
			RunRef:       continueRequest.RunRef,
			EvidenceRefs: compactStringsV0(command.EvidenceRefs),
		}, err
	}
	if runtime.Closed {
		return CodexStackResidentDirectorRunResultV0{
			Status:       CodexStackResidentDirectorStatusCompletedV0,
			RunRef:       continueRequest.RunRef,
			Closed:       true,
			EvidenceRefs: compactStringsV0(append(command.EvidenceRefs, runtime.ClosedResult.EvidenceRefs...)),
		}, nil
	}
	loop, err := runtime.Service.RunResidentDirectorBriefingLoopV0(ctx, orquestacionnucleoapp.ResidentDirectorBriefingLoopRequestV0{
		RunRef:               continueRequest.RunRef,
		ObjectiveRef:         "resident-director:" + continueRequest.RunRef,
		ContextRefs:          []string{"context-ref-codex-stack-resident-director"},
		OccurredAt:           runtime.LoopRequest.OccurredAt,
		CorrelationID:        runtime.LoopRequest.CorrelationID,
		EvidenceRefs:         compactStringsV0(append(command.EvidenceRefs, runtime.LoopRequest.EvidenceRefs...)),
		MaxActions:           command.MaxActions,
		MaxDispatchesPerWait: runtime.LoopRequest.MaxDispatchesPerWait,
		WaitAgentRefs:        runtime.LoopRequest.WaitAgentRefs,
		WaitScopeApplied:     runtime.LoopRequest.WaitScopeApplied,
		BriefingSource: codexStackResidentBriefingSourceV0{
			OutboxLedger: stack.Stores.OutboxLedger,
			ObjectiveRef: "resident-director:" + continueRequest.RunRef,
			ContextRefs:  []string{"context-ref-codex-stack-resident-director"},
		},
		Dispatchers:           runtime.LoopRequest.Dispatchers,
		BatchDispatchers:      runtime.LoopRequest.BatchDispatchers,
		ExternalActionHandler: codexStackResidentCloseHandlerV0{Request: runtime.Request, Ports: stack.Ports},
	})
	result := CodexStackResidentDirectorRunResultV0{
		Status:          firstNonEmptyQueuedSourceV0(loop.Status, CodexStackResidentDirectorStatusCompletedV0),
		RunRef:          continueRequest.RunRef,
		ExecutedActions: loop.ExecutedActions,
		EvidenceRefs:    compactStringsV0(loop.EvidenceRefs),
		Loop:            loop,
	}
	if err != nil {
		return result, err
	}
	return result, nil
}

func codexStackResidentContinueRequestV0(
	command CodexStackResidentDirectorCommandV0,
	request DrainRunRequestV0,
) orquestaappdirectorservice.ContinueAppDirectorRequestV0 {
	return orquestaappdirectorservice.ContinueAppDirectorRequestV0{
		RunRef:                     request.RunRef,
		OccurredAt:                 request.OccurredAt,
		CorrelationID:              request.CorrelationID,
		RequestedBy:                firstNonEmptyQueuedSourceV0(command.RequestedBy, "orquesta-codex-stack-resident-director"),
		MaxBursts:                  command.MaxBursts,
		MaxStepsPerBurst:           command.MaxStepsPerBurst,
		MaxDispatchesPerWait:       command.MaxDispatchesPerWait,
		WaitAgentRefs:              request.WaitAgentRefs,
		MaxCommands:                command.MaxCommands,
		MaxOutboxPerCycle:          command.MaxOutboxPerCycle,
		MaxDecisionCycles:          command.MaxDecisionCycles,
		MaxExternalWaits:           command.MaxExternalWaits,
		OperationalDirectorPlanRef: request.OperationalDirectorPlanRef,
	}
}

func (stack StackV0) codexStackResidentQueueStatusV0(
	ctx context.Context,
	result CodexStackResidentDirectorRunResultV0,
) string {
	if result.Closed {
		return orquestarunqueue.RunStatusClosedV0
	}
	if stack.Ports.RunStore == nil || strings.TrimSpace(result.RunRef) == "" {
		return ""
	}
	run, err := stack.Ports.RunStore.LoadRunV0(ctx, result.RunRef)
	if err != nil {
		return ""
	}
	if run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		return orquestarunqueue.RunStatusClosedV0
	}
	return ""
}

func codexStackResidentResultFromCoordinatorV0(
	command CodexStackResidentDirectorCommandV0,
	tick orquestaruncoordinator.RunCoordinatorTickResultV0,
	accumulator *codexStackResidentDirectorAccumulatorV0,
) CodexStackResidentDirectorResultV0 {
	result := CodexStackResidentDirectorResultV0{
		Status:       CodexStackResidentDirectorStatusIdleV0,
		EvidenceRefs: compactStringsV0(command.EvidenceRefs),
	}
	if accumulator == nil || len(accumulator.Runs) == 0 || len(tick.Executions) == 0 {
		return result
	}
	result.Status = CodexStackResidentDirectorStatusCompletedV0
	result.Runs = append([]CodexStackResidentDirectorRunResultV0(nil), accumulator.Runs...)
	for _, run := range accumulator.Runs {
		result.RunRef = run.RunRef
		result.RunRefs = append(result.RunRefs, run.RunRef)
		result.ExecutedActions += run.ExecutedActions
		result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, run.EvidenceRefs...))
		result.Loop = run.Loop
		if strings.TrimSpace(run.Status) != "" {
			result.Status = strings.TrimSpace(run.Status)
		}
	}
	result.RunRefs = compactStringsV0(result.RunRefs)
	return result
}

type codexStackResidentBriefingSourceV0 struct {
	OutboxLedger orquestadirectorcycleoutbox.DirectorCycleOutboxLedgerPortV0
	ObjectiveRef string
	ContextRefs  []string
}

func (source codexStackResidentBriefingSourceV0) BuildResidentDirectorBriefingV0(
	ctx context.Context,
	request orquestacionnucleoapp.ResidentDirectorBriefingBuildRequestV0,
) (orquestadirectorsupervisor.DirectorSupervisorBriefingV0, error) {
	if previous := request.PreviousStep; previous != nil && previous.Execution != nil {
		if previous.Execution.Status == orquestacionnucleoapp.DirectorBriefingExecutionStatusExternalAppliedV0 &&
			previous.Execution.Action.Kind == orquestadirectorsupervisor.DirectorSupervisorActionKindCloseOrIdleV0 {
			return source.completedBriefingV0(request), nil
		}
		if previous.Execution.Burst != nil && previous.Execution.Burst.FinalBriefing != nil {
			briefing := *previous.Execution.Burst.FinalBriefing
			if residentDirectorReusableFinalBriefingV0(briefing) {
				return briefing, nil
			}
		}
	}
	if source.OutboxLedger != nil {
		pending, issues := source.OutboxLedger.ListPending(ctx, orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0{
			RunRef: request.RunRef,
		})
		if len(issues) > 0 {
			return orquestadirectorsupervisor.DirectorSupervisorBriefingV0{},
				fmt.Errorf("resident_director_outbox_issues:%s", issues[0].Code)
		}
		if len(pending) > 0 {
			return source.briefingFromDecisionV0(
				request,
				orquestadirectorsupervisor.DirectorSupervisorActionWaitOutboxV0,
				false,
				pendingOutboxRefsForResidentDirectorV0(pending),
			)
		}
	}
	return source.briefingFromDecisionV0(
		request,
		orquestadirectorsupervisor.DirectorSupervisorActionContinueV0,
		true,
		nil,
	)
}

func (source codexStackResidentBriefingSourceV0) completedBriefingV0(
	request orquestacionnucleoapp.ResidentDirectorBriefingBuildRequestV0,
) orquestadirectorsupervisor.DirectorSupervisorBriefingV0 {
	return orquestadirectorsupervisor.DirectorSupervisorBriefingV0{
		SchemaVersion:            orquestadirectorsupervisor.DirectorSupervisorBriefingSchemaV0,
		RunRef:                   request.RunRef,
		ObjectiveRef:             firstNonEmptyQueuedSourceV0(source.ObjectiveRef, request.ObjectiveRef),
		AutonomousRecommendation: orquestadirectorsupervisor.DirectorSupervisorAutonomousStopV0,
		ReasonCode:               "resident_director_completed",
		ContextRefs:              compactStringsV0(append(source.ContextRefs, request.ContextRefs...)),
		EvidenceRefs:             compactStringsV0(request.EvidenceRefs),
	}
}

func residentDirectorReusableFinalBriefingV0(
	briefing orquestadirectorsupervisor.DirectorSupervisorBriefingV0,
) bool {
	if briefing.DecisionAction == orquestadirectorsupervisor.DirectorSupervisorActionStopMaxStepsV0 {
		return false
	}
	action := briefing.NextAction
	if action == nil && len(briefing.ActionQueue) > 0 {
		action = &briefing.ActionQueue[0]
	}
	if action == nil {
		return true
	}
	return action.SourceAction != orquestadirectorsupervisor.DirectorSupervisorActionStopMaxStepsV0 &&
		action.Kind != orquestadirectorsupervisor.DirectorSupervisorActionKindStopBudgetV0
}

func (source codexStackResidentBriefingSourceV0) briefingFromDecisionV0(
	request orquestacionnucleoapp.ResidentDirectorBriefingBuildRequestV0,
	action orquestadirectorsupervisor.DirectorSupervisorActionV0,
	shouldContinue bool,
	pendingOutboxRefs []string,
) (orquestadirectorsupervisor.DirectorSupervisorBriefingV0, error) {
	decision := orquestadirectorsupervisor.DirectorSupervisorDecisionV0{
		RunRef:                   request.RunRef,
		Action:                   action,
		ShouldContinue:           shouldContinue,
		AutonomousRecommendation: residentDirectorSupervisorRecommendationV0(action),
		ReasonCode:               residentDirectorSupervisorReasonV0(action),
		StepNumber:               request.StepNumber,
		MaxSteps:                 request.StepNumber + 1,
		PendingOutboxRefs:        compactStringsV0(pendingOutboxRefs),
		EvidenceRefs:             compactStringsV0(request.EvidenceRefs),
	}
	return orquestadirectorsupervisor.BuildDirectorSupervisorBriefingV0(
		orquestadirectorsupervisor.DirectorSupervisorBriefingInputV0{
			Decision:     decision,
			ObjectiveRef: firstNonEmptyQueuedSourceV0(source.ObjectiveRef, request.ObjectiveRef),
			ContextRefs:  compactStringsV0(append(source.ContextRefs, request.ContextRefs...)),
		},
	)
}

func residentDirectorSupervisorRecommendationV0(
	action orquestadirectorsupervisor.DirectorSupervisorActionV0,
) orquestadirectorsupervisor.DirectorSupervisorAutonomousRecommendationV0 {
	switch action {
	case orquestadirectorsupervisor.DirectorSupervisorActionContinueV0:
		return orquestadirectorsupervisor.DirectorSupervisorAutonomousContinueV0
	case orquestadirectorsupervisor.DirectorSupervisorActionWaitOutboxV0:
		return orquestadirectorsupervisor.DirectorSupervisorAutonomousWaitV0
	default:
		return orquestadirectorsupervisor.DirectorSupervisorAutonomousNeedsDirectorV0
	}
}

func residentDirectorSupervisorReasonV0(
	action orquestadirectorsupervisor.DirectorSupervisorActionV0,
) string {
	switch action {
	case orquestadirectorsupervisor.DirectorSupervisorActionWaitOutboxV0:
		return orquestadirectorsupervisor.DirectorSupervisorReasonOutboxPendingV0
	default:
		return orquestadirectorsupervisor.DirectorSupervisorReasonContinueV0
	}
}

func pendingOutboxRefsForResidentDirectorV0(
	messages []orquestacoreworkflow.OutboxMessageV0,
) []string {
	out := make([]string, 0, len(messages))
	for _, message := range messages {
		out = append(out, strings.TrimSpace(message.MessageID))
	}
	return compactStringsV0(out)
}

type codexStackResidentCloseHandlerV0 struct {
	Request orquestaappdirectorservice.ContinueAppDirectorRequestV0
	Ports   orquestaappdirectorservice.StartAppDirectorPortsV0
}

func (handler codexStackResidentCloseHandlerV0) ExecuteDirectorBriefingExternalActionV0(
	ctx context.Context,
	request orquestacionnucleoapp.DirectorBriefingExternalActionRequestV0,
) (orquestacionnucleoapp.DirectorBriefingExternalActionResultV0, error) {
	if request.Action.Kind != orquestadirectorsupervisor.DirectorSupervisorActionKindCloseOrIdleV0 {
		return orquestacionnucleoapp.DirectorBriefingExternalActionResultV0{
			Status:       orquestacionnucleoapp.DirectorBriefingExecutionStatusExternalPendingV0,
			RunRef:       request.Briefing.RunRef,
			ActionRef:    request.Action.ActionRef,
			EvidenceRefs: compactStringsV0(request.EvidenceRefs),
		}, nil
	}
	closeRequest := handler.Request
	closeRequest.RunRef = firstNonEmptyQueuedSourceV0(request.Briefing.RunRef, closeRequest.RunRef)
	closeRequest.OccurredAt = firstNonEmptyQueuedSourceV0(request.OccurredAt, closeRequest.OccurredAt)
	closeRequest.CorrelationID = firstNonEmptyQueuedSourceV0(request.CorrelationID, closeRequest.CorrelationID)
	continued, err := orquestaappdirectorservice.ContinueAppDirectorV0(ctx, closeRequest, handler.Ports)
	return orquestacionnucleoapp.DirectorBriefingExternalActionResultV0{
		Status:    orquestacionnucleoapp.DirectorBriefingExecutionStatusExternalAppliedV0,
		RunRef:    closeRequest.RunRef,
		ActionRef: request.Action.ActionRef,
		EvidenceRefs: compactStringsV0(append(
			append([]string{"evidence-ref-codex-stack-resident-director-close"}, request.EvidenceRefs...),
			continued.EvidenceRefs...,
		)),
	}, err
}
