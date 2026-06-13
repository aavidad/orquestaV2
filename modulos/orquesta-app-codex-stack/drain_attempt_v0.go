package orquestaappcodexstack

import (
	"context"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

type drainRunAttemptControlV0 struct {
	Loop orquestacionnucleoapp.ProgressiveLoopResultV0
}

func (stack StackV0) drainRunAttemptV0(
	ctx context.Context,
	request DrainRunRequestV0,
) (orquestacionnucleoapp.ProgressiveLoopResultV0, error) {
	control, err := stack.drainRunAttemptControlV0(ctx, request)
	return control.Loop, err
}

func (stack StackV0) drainRunAttemptControlV0(
	ctx context.Context,
	request DrainRunRequestV0,
) (drainRunAttemptControlV0, error) {
	run, err := stack.Stores.RunStore.LoadRunV0(ctx, request.RunRef)
	if err != nil {
		return drainRunAttemptControlV0{}, err
	}
	if !stackRunIsActiveV0(run) {
		recovered, ok, err := stack.recoverBlockedDirectorDecisionSourceRunForDrainV0(ctx, request, run)
		if err != nil {
			return drainRunAttemptControlV0{}, err
		}
		if ok {
			run = recovered
		}
		if !stackRunIsActiveV0(run) {
			recovered, ok, err = stack.recoverBlockedOpenPhaseProjectionRunForDrainV0(ctx, request, run)
			if err != nil {
				return drainRunAttemptControlV0{}, err
			}
			if ok {
				run = recovered
			}
		}
	}
	if err := stack.submitPendingDomainWorkArtifactsV0(ctx, request, run); err != nil {
		return drainRunAttemptControlV0{}, err
	}
	if recovered, err := stack.recoverMissingLaunchOutboxForRequestedAgentsV0(ctx, run); err != nil || recovered {
		if err != nil {
			return drainRunAttemptControlV0{}, err
		}
		return stack.continueDrainRunControlAfterExternalV0(ctx, request)
	}
	if err := stack.submitRecoverableDomainWorkArtifactsWithoutCoreDeliveryV0(ctx, request, run); err != nil {
		return drainRunAttemptControlV0{}, err
	}
	observations, err := stack.drainAvailableObservationsV0(ctx, request, run)
	if err != nil {
		return drainRunAttemptControlV0{}, err
	}
	recovered, err := stack.recoverableDomainWorkDeliveryObservationsV0(ctx, request, run)
	if err != nil {
		return drainRunAttemptControlV0{}, err
	}
	observations = append(observations, recovered...)
	observations = drainObservationsForWaitAgentRefsV0(observations, request.WaitAgentRefs)
	if len(observations) > 0 {
		run, err = stack.applyDrainObservationsV0(ctx, request, observations)
		if err != nil {
			return drainRunAttemptControlV0{}, err
		}
		if err := stack.submitDomainWorkArtifactsAfterDrainObservationsV0(ctx, request, run, observations); err != nil {
			return drainRunAttemptControlV0{}, err
		}
		if drainRunHasPendingExternalAgentsV0(run, request.WaitAgentRefs) {
			return drainRunAttemptControlV0{
				Loop: drainProgressiveResultV0(
					orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
					run,
				),
			}, nil
		}
		return stack.continueDrainRunControlAfterExternalV0(ctx, request)
	}
	if drainRunHasPendingExternalAgentsV0(run, request.WaitAgentRefs) {
		if len(compactStringsV0(request.WaitAgentRefs)) > 0 {
			if control, progressed, err := stack.continuePendingExternalProgressV0(ctx, request, run); err != nil {
				return control, err
			} else if progressed {
				if !drainRunHasPendingExternalAgentsV0(control.Loop.Run, request.WaitAgentRefs) {
					return control, nil
				}
				if drainRunHasScopedPartialClosureV0(control.Loop.Run, request.WaitAgentRefs) {
					return control, nil
				}
				return control, nil
			}
			return drainRunAttemptControlV0{
				Loop: drainProgressiveResultV0(
					orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
					run,
				),
			}, nil
		}
		if control, progressed, err := stack.continuePendingExternalProgressV0(ctx, request, run); err != nil || progressed {
			return control, err
		}
		if stackDrainRunAwaitsLateDirectorDecisionsV0(run) {
			if control, progressed, err := stack.continuePendingExternalDirectorV0(ctx, request, run); err != nil || progressed {
				return control, err
			}
		}
		if hold, err := stack.holdDirectorProgressWhileExternalAgentsPendingV0(ctx, run); err != nil || hold {
			return drainRunAttemptControlV0{
				Loop: drainProgressiveResultV0(
					orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
					run,
				),
			}, err
		}
		if control, progressed, err := stack.continuePendingExternalDirectorV0(ctx, request, run); err != nil || progressed {
			return control, err
		}
		return drainRunAttemptControlV0{
			Loop: drainProgressiveResultV0(
				orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
				run,
			),
		}, nil
	}
	if control, progressed, err := stack.continueTerminalExternalProgressV0(ctx, request, run); err != nil || progressed {
		return control, err
	}
	return stack.continueDrainRunControlAfterExternalV0(ctx, request)
}

func (stack StackV0) holdDirectorProgressWhileExternalAgentsPendingV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
) (bool, error) {
	return RunHasOpenAutoprogrammingTasksV0(ctx, stack.Stores.TaskStore, run)
}

func (stack StackV0) continuePendingExternalDirectorV0(
	ctx context.Context,
	request DrainRunRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) (drainRunAttemptControlV0, bool, error) {
	if stack.Ports.DirectorDecisionSource == nil {
		return drainRunAttemptControlV0{}, false, nil
	}
	directorRequest := request
	directorRequest.WaitAgentRefs = nil
	ports := stack.directorPortsWithClosureSourceV0(stack.Ports)
	ports.DeliverySource = nil
	ports.ProgressSource = nil
	ports.LeaseSource = nil
	ports.AssessmentReplanSource = nil
	control, err := stack.continueDrainRunControlAfterExternalWithPortsV0(ctx, directorRequest, ports)
	if err != nil {
		return drainRunAttemptControlV0{}, false, err
	}
	if !drainRunProgressedV0(run, control.Loop.Run) {
		return drainRunAttemptControlV0{}, false, nil
	}
	return control, true, nil
}

func (stack StackV0) continuePendingExternalProgressV0(
	ctx context.Context,
	request DrainRunRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) (drainRunAttemptControlV0, bool, error) {
	if stack.Ports.ProgressSource == nil &&
		stack.Ports.LeaseSource == nil &&
		stack.Ports.AssessmentReplanSource == nil {
		return drainRunAttemptControlV0{}, false, nil
	}
	ports := stack.directorPortsWithClosureSourceV0(stack.Ports)
	ports.DirectorDecisionSource = nil
	reconciled, applied, err := stack.reconcileStoppedPendingAgentsV0(ctx, request, run)
	if err != nil || applied {
		if err == nil && applied && !drainRunHasPendingExternalAgentsV0(reconciled, request.WaitAgentRefs) {
			control, continueErr := stack.continueDrainRunControlAfterExternalWithPortsV0(ctx, request, ports)
			return control, true, continueErr
		}
		return drainRunAttemptControlV0{
			Loop: drainProgressiveResultV0(orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0, reconciled),
		}, applied, err
	}
	control, err := stack.continueDrainRunControlAfterExternalWithPortsV0(ctx, request, ports)
	if err != nil {
		return drainRunAttemptControlV0{}, false, err
	}
	if !drainRunProgressedV0(run, control.Loop.Run) {
		return drainRunAttemptControlV0{}, false, nil
	}
	return control, true, nil
}

func (stack StackV0) continueTerminalExternalProgressV0(
	ctx context.Context,
	request DrainRunRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) (drainRunAttemptControlV0, bool, error) {
	if stack.Ports.AssessmentReplanSource == nil {
		return drainRunAttemptControlV0{}, false, nil
	}
	ports := stack.directorPortsWithClosureSourceV0(stack.Ports)
	ports.DirectorDecisionSource = nil
	reconciled, applied, err := stack.reconcileTerminalOpenTaskAgentsV0(ctx, request, run)
	if err != nil {
		return drainRunAttemptControlV0{}, false, err
	}
	if applied {
		run = reconciled
		control, continueErr := stack.continueDrainRunControlAfterExternalWithPortsV0(ctx, request, ports)
		return control, true, continueErr
	}
	if !stackRunHasRecoverableTerminalAssessmentV0(run) {
		return drainRunAttemptControlV0{}, false, nil
	}
	control, err := stack.continueDrainRunControlAfterExternalWithPortsV0(ctx, request, ports)
	if err != nil {
		return drainRunAttemptControlV0{}, false, err
	}
	return control, drainRunProgressedV0(run, control.Loop.Run), nil
}

func drainRunProgressedV0(
	before orquestacoreworkflow.OrchestrationRunV0,
	after orquestacoreworkflow.OrchestrationRunV0,
) bool {
	return after.LastSequence != before.LastSequence
}

func (stack StackV0) drainAvailableObservationsV0(
	ctx context.Context,
	request DrainRunRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) ([]orquestacionnucleoapp.AgentDeliveryObservationV0, error) {
	if stack.Ports.DeliverySource == nil {
		return nil, nil
	}
	observations, err := stack.Ports.DeliverySource.BuildAgentDeliveryObservationsV0(
		ctx,
		orquestacionnucleoapp.AgentDeliveryObservationRequestV0{
			Run:           drainRunWithStartedAgentsKnownV0(run),
			OccurredAt:    request.OccurredAt,
			CorrelationID: request.CorrelationID,
			EvidenceRefs:  []string{"evidence-ref-app-stack-drain-observations"},
			WaitAgentRefs: request.WaitAgentRefs,
		},
	)
	if err != nil {
		return nil, err
	}
	return drainObservationsForWaitAgentRefsV0(observations, request.WaitAgentRefs), nil
}

func (stack StackV0) continueDrainRunAfterExternalV0(
	ctx context.Context,
	request DrainRunRequestV0,
) (orquestacionnucleoapp.ProgressiveLoopResultV0, error) {
	control, err := stack.continueDrainRunControlAfterExternalV0(ctx, request)
	return control.Loop, err
}

func (stack StackV0) continueDrainRunControlAfterExternalV0(
	ctx context.Context,
	request DrainRunRequestV0,
) (drainRunAttemptControlV0, error) {
	return stack.continueDrainRunControlAfterExternalWithPortsV0(
		ctx,
		request,
		stack.directorPortsWithClosureSourceV0(stack.Ports),
	)
}

func (stack StackV0) continueDrainRunControlAfterExternalWithPortsV0(
	ctx context.Context,
	request DrainRunRequestV0,
	ports orquestaappdirectorservice.StartAppDirectorPortsV0,
) (drainRunAttemptControlV0, error) {
	ports.DeliverySource = drainDeliverySourceForWaitAgentRefsV0(ports.DeliverySource, request.WaitAgentRefs)
	ports.ProgressSource = drainProgressSourceForWaitAgentRefsV0(ports.ProgressSource, request.WaitAgentRefs)
	continued, err := orquestaappdirectorservice.ContinueAppDirectorV0(
		ctx,
		orquestaappdirectorservice.ContinueAppDirectorRequestV0{
			RunRef:                     request.RunRef,
			OccurredAt:                 request.OccurredAt,
			CorrelationID:              request.CorrelationID,
			RequestedBy:                "orquesta-app-codex-stack-drain",
			MaxBursts:                  request.MaxBursts,
			MaxStepsPerBurst:           request.MaxStepsPerBurst,
			MaxDispatchesPerWait:       request.MaxDispatchesPerWait,
			MaxCommands:                request.MaxCommands,
			MaxOutboxPerCycle:          request.MaxOutboxPerCycle,
			MaxDecisionCycles:          request.MaxDecisionCycles,
			MaxExternalWaits:           0,
			WaitAgentRefs:              request.WaitAgentRefs,
			OperationalDirectorPlanRef: request.OperationalDirectorPlanRef,
		},
		ports,
	)
	if err != nil {
		return drainRunAttemptControlV0{}, err
	}
	loop := drainProgressiveResultV0(continued.LoopStatus, continued.Run)
	if err := stack.submitPendingDomainWorkArtifactsV0(ctx, request, loop.Run); err != nil {
		return drainRunAttemptControlV0{}, err
	}
	return drainRunAttemptControlV0{
		Loop: loop,
	}, nil
}
