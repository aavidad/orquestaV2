package orquestaappdirectorservice

import (
	"context"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestadirectorrunner "orquesta/modulos/orquesta-director-runner"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

const (
	ContinueAppDirectorResultSchemaVersionV0 = "continue_app_director_result.v0"
	ContinueAppDirectorStatusContinuedV0     = "continued"
	ContinueAppDirectorStatusPendingV0       = "pending"
)

type ContinueAppDirectorRequestV0 struct {
	RunRef                                  string                                               `json:"run_ref"`
	OccurredAt                              string                                               `json:"occurred_at,omitempty"`
	CorrelationID                           string                                               `json:"correlation_id,omitempty"`
	RequestedBy                             string                                               `json:"requested_by,omitempty"`
	MaxBursts                               int                                                  `json:"max_bursts,omitempty"`
	MaxStepsPerBurst                        int                                                  `json:"max_steps_per_burst,omitempty"`
	MaxDispatchesPerWait                    int                                                  `json:"max_dispatches_per_wait,omitempty"`
	WaitAgentRefs                           []string                                             `json:"wait_agent_refs,omitempty"`
	WaitCohortRef                           string                                               `json:"wait_cohort_ref,omitempty"`
	WaitWaveRef                             string                                               `json:"wait_wave_ref,omitempty"`
	WaitParentTaskRef                       string                                               `json:"wait_parent_task_ref,omitempty"`
	MaxCommands                             int                                                  `json:"max_commands,omitempty"`
	MaxOutboxPerCycle                       int                                                  `json:"max_outbox_per_cycle,omitempty"`
	MaxDecisionCycles                       int                                                  `json:"max_decision_cycles,omitempty"`
	MaxExternalWaits                        int                                                  `json:"max_external_waits,omitempty"`
	OperationalDirectorPlanRef              string                                               `json:"operational_director_plan_ref,omitempty"`
	OperationalDirectorPlan                 orquestadirectoroperativo.OperationalDirectorPlanV0  `json:"operational_director_plan,omitempty"`
	OperationalDirectorFunctionContractRefs []orquestacoreworkflow.WorkflowFunctionContractRefV0 `json:"operational_director_function_contract_refs,omitempty"`
	OperationalDirectorTargetPhaseID        orquestacoreworkflow.OrchestrationPhaseIDV0          `json:"operational_director_target_phase_id,omitempty"`
	OperationalDirectorMaxItems             int                                                  `json:"operational_director_max_items,omitempty"`
}

type ContinueAppDirectorResultV0 struct {
	SchemaVersion            string                                        `json:"schema_version"`
	Status                   string                                        `json:"status"`
	CorrelationID            string                                        `json:"correlation_id,omitempty"`
	Run                      orquestacoreworkflow.OrchestrationRunV0       `json:"run,omitempty"`
	LoopStatus               orquestacionnucleoapp.ProgressiveLoopStatusV0 `json:"loop_status,omitempty"`
	StartedAgents            []string                                      `json:"started_agents,omitempty"`
	OperationalClosureIssues []orquestacionnucleoapp.ErrorV0               `json:"operational_closure_issues,omitempty"`
	EvidenceRefs             []string                                      `json:"evidence_refs,omitempty"`
}

func ContinueAppDirectorV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
) (ContinueAppDirectorResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	request = normalizeContinueAppDirectorRequestV0(request)
	now, err := startAppDirectorNowV0(request.OccurredAt)
	if err != nil {
		return ContinueAppDirectorResultV0{}, err
	}
	if request.OccurredAt == "" {
		request.OccurredAt = now.Format("2006-01-02T15:04:05Z07:00")
	}
	if err := validateContinueAppDirectorRequestV0(request, ports); err != nil {
		return ContinueAppDirectorResultV0{}, err
	}
	materialized, err := materializeContinueOperationalDirectorPlanV0(ctx, request, ports)
	if err != nil {
		return ContinueAppDirectorResultV0{}, err
	}
	request = continueRequestWithOperationalDirectorWaitV0(request, materialized)
	request, err = continueRequestWithOperationalDirectorPlanStateV0(ctx, request, ports)
	if err != nil {
		return ContinueAppDirectorResultV0{}, err
	}
	if _, err := ensureOperationalDirectorReviewPhaseV0(ctx, request, ports); err != nil {
		return ContinueAppDirectorResultV0{}, err
	}
	autonomy, err := runExistingDirectorAutonomyLoopV0(ctx, request, ports)
	if err != nil {
		return ContinueAppDirectorResultV0{}, err
	}
	request = autonomy.Request
	operationalLoop := operationalDirectorScopedQuiescentLoopV0(request, autonomy.Loop, autonomy.LoopRequest)
	postLoop, err := continueOperationalDirectorPlanStatePostLoopV0(
		ctx,
		request,
		ports,
		operationalLoop,
		autonomy.LoopRequest,
		autonomy.ManagedLoop,
	)
	if err != nil {
		return ContinueAppDirectorResultV0{}, err
	}
	request = postLoop.Request
	operationalLoop = postLoop.Loop
	loopRequest := postLoop.LoopRequest
	loop, closureIssues, err := maybeCloseOperationalDirectorV0(ctx, request, ports, operationalLoop, loopRequest)
	if err != nil {
		return ContinueAppDirectorResultV0{}, err
	}
	return continueAppDirectorResultV0(request, loop, closureIssues), nil
}

func continueOperationalDirectorPlanStatePostLoopV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	loop orquestacionnucleoapp.ProgressiveLoopResultV0,
	loopRequest orquestacionnucleoapp.ProgressiveLoopRequestV0,
	managedLoop orquestacionnucleoapp.ManagedProgressiveLoopResultV0,
) (existingDirectorAutonomyLoopResultV0, error) {
	expired, err := applyOperationalDirectorPlanStateAfterExternalWaitExhaustedV0(ctx, request, ports, managedLoop)
	if err != nil || expired {
		return existingDirectorAutonomyLoopResultV0{Request: request, Loop: loop, LoopRequest: loopRequest, ManagedLoop: managedLoop}, err
	}
	changed, err := applyOperationalDirectorPlanStateAfterLoopV0(ctx, request, ports, loop)
	if err != nil || !changed {
		return existingDirectorAutonomyLoopResultV0{Request: request, Loop: loop, LoopRequest: loopRequest, ManagedLoop: managedLoop}, err
	}
	_, _, activeReview, err := operationalDirectorActiveReviewDeliveriesStepV0(ctx, request, ports)
	if err != nil || !activeReview {
		return existingDirectorAutonomyLoopResultV0{Request: request, Loop: loop, LoopRequest: loopRequest, ManagedLoop: managedLoop}, err
	}
	nextRequest, err := continueRequestWithOperationalDirectorPlanStateV0(ctx, request, ports)
	if err != nil {
		return existingDirectorAutonomyLoopResultV0{Request: request, Loop: loop, LoopRequest: loopRequest, ManagedLoop: managedLoop}, err
	}
	if _, err := ensureOperationalDirectorReviewPhaseV0(ctx, nextRequest, ports); err != nil {
		return existingDirectorAutonomyLoopResultV0{Request: request, Loop: loop, LoopRequest: loopRequest, ManagedLoop: managedLoop}, err
	}
	followup, err := runExistingDirectorAutonomyLoopV0(ctx, nextRequest, ports)
	if err != nil {
		return existingDirectorAutonomyLoopResultV0{Request: nextRequest, Loop: followup.Loop, LoopRequest: followup.LoopRequest}, err
	}
	followupLoop := operationalDirectorScopedQuiescentLoopV0(followup.Request, followup.Loop, followup.LoopRequest)
	if _, err := applyOperationalDirectorPlanStateAfterLoopV0(ctx, followup.Request, ports, followupLoop); err != nil {
		return existingDirectorAutonomyLoopResultV0{Request: followup.Request, Loop: followupLoop, LoopRequest: followup.LoopRequest}, err
	}
	return existingDirectorAutonomyLoopResultV0{Request: followup.Request, Loop: followupLoop, LoopRequest: followup.LoopRequest}, nil
}

func operationalDirectorScopedQuiescentLoopV0(
	request ContinueAppDirectorRequestV0,
	loop orquestacionnucleoapp.ProgressiveLoopResultV0,
	loopRequest orquestacionnucleoapp.ProgressiveLoopRequestV0,
) orquestacionnucleoapp.ProgressiveLoopResultV0 {
	if continueOperationalDirectorPlanRefV0(request) == "" ||
		loop.Status != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 ||
		loop.PendingOutboxCount > 0 ||
		len(loopRequest.WaitAgentRefs) == 0 ||
		appDirectorRunHasPendingAgentRefsV0(loop.Run, loopRequest.WaitAgentRefs) {
		return loop
	}
	loop.Status = orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0
	return loop
}

func appDirectorRunHasPendingAgentRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	agentRefs []string,
) bool {
	for _, agentRef := range compactServiceRefsV0(agentRefs) {
		if !startAppDirectorStringInSetV0(run.StartedAgents, agentRef) {
			continue
		}
		if startAppDirectorStringInSetV0(run.DeliveredAgents, agentRef) ||
			startAppDirectorStringInSetV0(run.FailedAgents, agentRef) ||
			startAppDirectorStringInSetV0(run.LostAgents, agentRef) ||
			startAppDirectorStringInSetV0(run.ConfirmedStoppedAgents, agentRef) ||
			startAppDirectorStringInSetV0(run.StoppedAgents, agentRef) {
			continue
		}
		return true
	}
	return false
}

type existingDirectorAutonomyLoopResultV0 struct {
	Request     ContinueAppDirectorRequestV0
	Loop        orquestacionnucleoapp.ProgressiveLoopResultV0
	LoopRequest orquestacionnucleoapp.ProgressiveLoopRequestV0
	ManagedLoop orquestacionnucleoapp.ManagedProgressiveLoopResultV0
}

func runExistingDirectorAutonomyLoopV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
) (existingDirectorAutonomyLoopResultV0, error) {
	loop, loopRequest, managedLoop, err := runExistingDirectorLoopV0(ctx, request, ports)
	if err != nil || ports.DirectorDecisionSource == nil {
		return existingDirectorAutonomyLoopResultV0{Request: request, Loop: loop, LoopRequest: loopRequest, ManagedLoop: managedLoop}, err
	}
	startRequest := continueAsStartRequestV0(request)
	for cycle := 0; cycle < request.MaxDecisionCycles; cycle++ {
		next, progressed, err := consumeStartAppDirectorDecisionsV0(ctx, startRequest, ports, loop)
		if err != nil || !progressed {
			return existingDirectorAutonomyLoopResultV0{Request: request, Loop: next, LoopRequest: loopRequest, ManagedLoop: managedLoop}, err
		}
		request, err = continueRequestWithOperationalDirectorPlanStateV0(ctx, request, ports)
		if err != nil {
			return existingDirectorAutonomyLoopResultV0{Request: request, Loop: next, LoopRequest: loopRequest, ManagedLoop: managedLoop}, err
		}
		if _, err := ensureOperationalDirectorReviewPhaseV0(ctx, request, ports); err != nil {
			return existingDirectorAutonomyLoopResultV0{Request: request, Loop: next, LoopRequest: loopRequest, ManagedLoop: managedLoop}, err
		}
		startRequest = continueAsStartRequestV0(request)
		loop, loopRequest, managedLoop, err = runExistingDirectorLoopV0(ctx, request, ports)
		if err != nil {
			return existingDirectorAutonomyLoopResultV0{Request: request, Loop: loop, LoopRequest: loopRequest, ManagedLoop: managedLoop}, err
		}
	}
	return existingDirectorAutonomyLoopResultV0{Request: request, Loop: loop, LoopRequest: loopRequest, ManagedLoop: managedLoop}, nil
}

func runExistingDirectorLoopV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
) (orquestacionnucleoapp.ProgressiveLoopResultV0, orquestacionnucleoapp.ProgressiveLoopRequestV0, orquestacionnucleoapp.ManagedProgressiveLoopResultV0, error) {
	service := existingDirectorLoopServiceV0(request, ports)
	loopRequest, err := existingDirectorLoopRequestV0(ctx, request, ports)
	if err != nil {
		return orquestacionnucleoapp.ProgressiveLoopResultV0{}, loopRequest, orquestacionnucleoapp.ManagedProgressiveLoopResultV0{}, err
	}
	if ports.ExternalWaiter == nil {
		loop, err := service.RunProgressiveLoopV0(ctx, loopRequest)
		return loop, loopRequest, orquestacionnucleoapp.ManagedProgressiveLoopResultV0{}, err
	}
	managed, err := service.RunManagedProgressiveLoopV0(
		ctx,
		orquestacionnucleoapp.ManagedProgressiveLoopRequestV0{
			Loop:             loopRequest,
			ExternalWaiter:   ports.ExternalWaiter,
			MaxExternalWaits: request.MaxExternalWaits,
		},
	)
	return managed.Final, loopRequest, managed, err
}

func existingDirectorLoopServiceV0(
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
) orquestacionnucleoapp.ServiceV0 {
	return orquestacionnucleoapp.ServiceV0{
		RunStore:           ports.RunStore,
		EventSink:          ports.EventSink,
		CandidateProvider:  composeStartAppDirectorProviderV0(nil, ports, request.RequestedBy),
		RunControl:         ports.RunControl,
		RunControlTerminal: ports.RunControlTerminal,
		OutboxLedger:       ports.OutboxLedger,
		MaxCommands:        request.MaxCommands,
		MaxOutboxPerCycle:  request.MaxOutboxPerCycle,
	}
}

func existingDirectorLoopRequestV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
) (orquestacionnucleoapp.ProgressiveLoopRequestV0, error) {
	waitResolution, err := appDirectorResolvedWaitV0(
		ctx,
		request.RunRef,
		request.WaitAgentRefs,
		appDirectorWaitFilterV0{
			CohortRef:     request.WaitCohortRef,
			WaveRef:       request.WaitWaveRef,
			ParentTaskRef: request.WaitParentTaskRef,
		},
		ports,
		appDirectorWaitStateMetaV0{
			OccurredAt:       request.OccurredAt,
			CorrelationID:    request.CorrelationID,
			EvidenceRefs:     []string{"evidence-ref-app-director-continue-v0"},
			MaxExternalWaits: request.MaxExternalWaits,
		},
	)
	if err != nil {
		return orquestacionnucleoapp.ProgressiveLoopRequestV0{}, err
	}
	return orquestacionnucleoapp.ProgressiveLoopRequestV0{
		RunRef:               request.RunRef,
		OccurredAt:           request.OccurredAt,
		MaxBursts:            request.MaxBursts,
		MaxStepsPerBurst:     request.MaxStepsPerBurst,
		MaxDispatchesPerWait: request.MaxDispatchesPerWait,
		WaitAgentRefs:        waitResolution.AgentRefs,
		WaitScopeApplied:     waitResolution.ScopeApplied,
		CorrelationID:        request.CorrelationID,
		EvidenceRefs:         []string{"evidence-ref-app-director-continue-v0"},
		Dispatchers:          ports.Dispatchers,
		BatchDispatchers:     ports.BatchDispatchers,
	}, nil
}

func continueAsStartRequestV0(request ContinueAppDirectorRequestV0) StartAppDirectorRequestV0 {
	return StartAppDirectorRequestV0{
		RunRef:                                  request.RunRef,
		OccurredAt:                              request.OccurredAt,
		CorrelationID:                           request.CorrelationID,
		RequestedBy:                             request.RequestedBy,
		MaxBursts:                               request.MaxBursts,
		MaxStepsPerBurst:                        request.MaxStepsPerBurst,
		MaxDispatchesPerWait:                    request.MaxDispatchesPerWait,
		WaitAgentRefs:                           request.WaitAgentRefs,
		WaitCohortRef:                           request.WaitCohortRef,
		WaitWaveRef:                             request.WaitWaveRef,
		WaitParentTaskRef:                       request.WaitParentTaskRef,
		MaxCommands:                             request.MaxCommands,
		MaxOutboxPerCycle:                       request.MaxOutboxPerCycle,
		MaxDecisionCycles:                       request.MaxDecisionCycles,
		MaxExternalWaits:                        request.MaxExternalWaits,
		OperationalDirectorPlanRef:              request.OperationalDirectorPlanRef,
		OperationalDirectorPlan:                 request.OperationalDirectorPlan,
		OperationalDirectorFunctionContractRefs: append([]orquestacoreworkflow.WorkflowFunctionContractRefV0(nil), request.OperationalDirectorFunctionContractRefs...),
		OperationalDirectorTargetPhaseID:        request.OperationalDirectorTargetPhaseID,
		OperationalDirectorMaxItems:             request.OperationalDirectorMaxItems,
	}
}

func continueAppDirectorResultV0(
	request ContinueAppDirectorRequestV0,
	loop orquestacionnucleoapp.ProgressiveLoopResultV0,
	closureIssues []orquestacionnucleoapp.ErrorV0,
) ContinueAppDirectorResultV0 {
	status := ContinueAppDirectorStatusPendingV0
	if loop.Status == orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0 ||
		loop.Status == orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 {
		status = ContinueAppDirectorStatusContinuedV0
	}
	return ContinueAppDirectorResultV0{
		SchemaVersion:            ContinueAppDirectorResultSchemaVersionV0,
		Status:                   status,
		CorrelationID:            request.CorrelationID,
		Run:                      loop.Run,
		LoopStatus:               loop.Status,
		StartedAgents:            append([]string(nil), loop.Run.StartedAgents...),
		OperationalClosureIssues: append([]orquestacionnucleoapp.ErrorV0(nil), closureIssues...),
		EvidenceRefs:             []string{"evidence-ref-app-director-continue-v0"},
	}
}

func normalizeContinueAppDirectorRequestV0(
	request ContinueAppDirectorRequestV0,
) ContinueAppDirectorRequestV0 {
	request.RunRef = strings.TrimSpace(request.RunRef)
	request.OccurredAt = strings.TrimSpace(request.OccurredAt)
	request.CorrelationID = strings.TrimSpace(request.CorrelationID)
	request.RequestedBy = strings.TrimSpace(request.RequestedBy)
	request.WaitAgentRefs = compactServiceRefsV0(request.WaitAgentRefs)
	request.WaitCohortRef = strings.TrimSpace(request.WaitCohortRef)
	request.WaitWaveRef = strings.TrimSpace(request.WaitWaveRef)
	request.WaitParentTaskRef = strings.TrimSpace(request.WaitParentTaskRef)
	request.OperationalDirectorPlanRef = strings.TrimSpace(request.OperationalDirectorPlanRef)
	request.OperationalDirectorFunctionContractRefs = normalizeServiceWorkflowFunctionContractRefsV0(request.OperationalDirectorFunctionContractRefs)
	request.OperationalDirectorTargetPhaseID = orquestacoreworkflow.OrchestrationPhaseIDV0(strings.TrimSpace(string(request.OperationalDirectorTargetPhaseID)))
	if request.OperationalDirectorMaxItems < 0 {
		request.OperationalDirectorMaxItems = 0
	}
	if request.CorrelationID == "" {
		request.CorrelationID = "corr-" + request.RunRef
	}
	if request.RequestedBy == "" {
		request.RequestedBy = "orquesta-app-director-continue"
	}
	if request.MaxBursts <= 0 {
		request.MaxBursts = defaultStartAppDirectorMaxBurstsV0
	}
	if request.MaxStepsPerBurst <= 0 {
		request.MaxStepsPerBurst = defaultStartAppDirectorMaxStepsPerBurstV0
	}
	if request.MaxDispatchesPerWait <= 0 {
		request.MaxDispatchesPerWait = defaultStartAppDirectorMaxDispatchesPerWaitV0
	}
	if request.MaxCommands <= 0 {
		request.MaxCommands = defaultStartAppDirectorMaxCommandsV0
	}
	request.MaxCommands = boundedStartAppDirectorLimitV0(
		request.MaxCommands,
		orquestadirectorrunner.DirectorCycleMaxCommandsV0,
	)
	if request.MaxOutboxPerCycle <= 0 {
		request.MaxOutboxPerCycle = defaultStartAppDirectorMaxOutboxPerCycleV0
	}
	request.MaxOutboxPerCycle = boundedStartAppDirectorLimitV0(
		request.MaxOutboxPerCycle,
		orquestadirectorrunner.DirectorCycleMaxOutboxV0,
	)
	if request.MaxDecisionCycles <= 0 {
		request.MaxDecisionCycles = defaultStartAppDirectorMaxDecisionCyclesV0
	}
	if request.MaxExternalWaits < 0 {
		request.MaxExternalWaits = 0
	}
	return request
}

func validateContinueAppDirectorRequestV0(
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
) error {
	if strings.TrimSpace(request.RunRef) == "" {
		return AppDirectorServiceIssueV0{Field: "run_ref"}
	}
	if _, err := startAppDirectorNowV0(request.OccurredAt); err != nil {
		return err
	}
	return validateStartAppDirectorPortsV0(ports)
}
