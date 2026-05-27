package orquestaappcodexstack

import (
	"context"
	"errors"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func (stack StackV0) recoverQueuedControlledDomainWorkArtifactsV0(
	ctx context.Context,
	command orquestaruncoordinator.RunCoordinatorTickCommandV0,
) error {
	if !stack.domainWorkDeliveryBridgeReadyV0() ||
		stack.Stores.RunQueue == nil ||
		stack.Stores.RunControl == nil {
		return nil
	}
	candidates, err := stack.Stores.RunQueue.ListRunSchedulingCandidatesV0(
		ctx,
		orquestarunqueue.RunQueueReadRequestV0{
			QueueRef:             command.QueueRef,
			AppRefs:              append([]string(nil), command.AppRefs...),
			IncludeNonExecutable: true,
		},
	)
	if err != nil {
		return err
	}
	for _, candidate := range candidates {
		ok, err := stack.queuedCandidateAllowsDomainWorkRecoveryV0(ctx, candidate.RunRef)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		run, err := stack.Stores.RunStore.LoadRunV0(ctx, candidate.RunRef)
		if err != nil {
			return err
		}
		if err := stack.submitRecoverableDomainWorkArtifactsWithoutCoreDeliveryV0(
			ctx,
			DrainRunRequestV0{
				RunRef:        candidate.RunRef,
				OccurredAt:    formatStackCoordinatorTimeV0(command.OccurredAt),
				CorrelationID: command.CorrelationID,
			},
			run,
		); err != nil {
			return err
		}
	}
	return nil
}

func (stack StackV0) queuedCandidateAllowsDomainWorkRecoveryV0(
	ctx context.Context,
	runRef string,
) (bool, error) {
	state, err := stack.Stores.RunControl.ReadRunControlStateV0(
		ctx,
		orquestaruncontrol.RunControlReadRequestV0{RunRef: runRef},
	)
	if err != nil {
		var notFound orquestaruncontrol.RunControlStateNotFoundErrorV0
		if errorAsRunControlNotFoundV0(err, &notFound) {
			return false, nil
		}
		return false, err
	}
	evaluation := orquestaruncontrol.EvaluateRunControlV0(state)
	return evaluation.Terminal || evaluation.StopAgentsAllowed, nil
}

func errorAsRunControlNotFoundV0(
	err error,
	target *orquestaruncontrol.RunControlStateNotFoundErrorV0,
) bool {
	return err != nil && target != nil && errors.As(err, target)
}
