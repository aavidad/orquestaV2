package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"strings"
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
	// StreamingSubwaveEnabled activa el avance incremental por sub-ola: en cuanto
	// un subconjunto de agentes entrega, avanza a review sin esperar al resto de
	// la ola. Opt-in: por defecto se mantiene la barrera de ola completa.
	StreamingSubwaveEnabled bool `json:"streaming_subwave_enabled,omitempty"`
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
	if result, handled, err := continueAppDirectorGoalFirstContainerResultV0(ctx, request, ports); err != nil || handled {
		return result, err
	}
	if err := validateContinueAppDirectorRequestV0(request, ports); err != nil {
		return ContinueAppDirectorResultV0{}, err
	}
	if result, closed, err := closedOperationalDirectorPlanStateContinueResultV0(ctx, request, ports); err != nil || closed {
		return result, err
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
	if err := ensureOperationalDirectorPassedRequiredTestsQualityGateV0(ctx, request, ports); err != nil {
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

func closedOperationalDirectorPlanStateContinueResultV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
) (ContinueAppDirectorResultV0, bool, error) {
	planRef := strings.TrimSpace(request.OperationalDirectorPlanRef)
	if planRef == "" {
		planRef = defaultOperationalDirectorDecisionPlanRefV0(request.RunRef)
	}
	if planRef == "" || ports.OperationalPlanStateStore == nil {
		return ContinueAppDirectorResultV0{}, false, nil
	}
	state, err := ports.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, request.RunRef, planRef)
	if err != nil {
		if operationalDirectorPlanStateMissingV0(err) {
			return ContinueAppDirectorResultV0{}, false, nil
		}
		return ContinueAppDirectorResultV0{}, true, err
	}
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 {
		return ContinueAppDirectorResultV0{}, false, nil
	}
	run, err := ports.RunStore.LoadRunV0(ctx, request.RunRef)
	if err != nil {
		return ContinueAppDirectorResultV0{}, true, err
	}
	return continueAppDirectorResultV0(
		request,
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run:    run,
		},
		nil,
	), true, nil
}
