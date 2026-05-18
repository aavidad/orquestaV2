package orquestacionnucleoapp

import (
	"context"
	"strings"
)

func (service ServiceV0) RunManagedProgressiveLoopV0(
	ctx context.Context,
	request ManagedProgressiveLoopRequestV0,
) (ManagedProgressiveLoopResultV0, error) {
	request = normalizeManagedProgressiveLoopRequestV0(request)
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateManagedProgressiveLoopRequestV0(request); err != nil {
		return ManagedProgressiveLoopResultV0{}, err
	}
	result := ManagedProgressiveLoopResultV0{}
	for attempt := 1; attempt <= request.MaxExternalWaits+1; attempt++ {
		loop, err := service.RunProgressiveLoopV0(ctx, request.Loop)
		result = appendManagedProgressiveAttemptV0(result, attempt, loop)
		if err != nil {
			result.Status = loop.Status
			return result, err
		}
		if loop.Status != ProgressiveLoopStatusWaitExternalV0 {
			result.Status = loop.Status
			return result, nil
		}
		if attempt > request.MaxExternalWaits {
			result.Status = loop.Status
			return result, nil
		}
		wait, err := request.ExternalWaiter.WaitExternalProgressV0(
			ctx,
			externalProgressWaitRequestV0(request, attempt, loop),
		)
		if err != nil {
			result.Status = loop.Status
			return result, err
		}
		result = appendManagedProgressiveWaitV0(result, attempt, wait)
		if !wait.Continue {
			result.Status = loop.Status
			return result, nil
		}
	}
	return result, nil
}

func normalizeManagedProgressiveLoopRequestV0(
	request ManagedProgressiveLoopRequestV0,
) ManagedProgressiveLoopRequestV0 {
	request.Loop = normalizeProgressiveLoopRequestV0(request.Loop)
	if request.MaxExternalWaits < 0 {
		request.MaxExternalWaits = 0
	}
	return request
}

func validateManagedProgressiveLoopRequestV0(
	request ManagedProgressiveLoopRequestV0,
) error {
	if strings.TrimSpace(request.Loop.RunRef) == "" {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "run_ref", "run_ref requerido")
	}
	if request.MaxExternalWaits > 0 && request.ExternalWaiter == nil {
		return errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"external_waiter",
			"external_waiter requerido",
		)
	}
	return nil
}

func externalProgressWaitRequestV0(
	request ManagedProgressiveLoopRequestV0,
	waitNumber int,
	last ProgressiveLoopResultV0,
) ExternalProgressWaitRequestV0 {
	return ExternalProgressWaitRequestV0{
		RunRef:        request.Loop.RunRef,
		WaitNumber:    waitNumber,
		LastResult:    last,
		CorrelationID: request.Loop.CorrelationID,
		EvidenceRefs:  request.Loop.EvidenceRefs,
		WaitAgentRefs: request.Loop.WaitAgentRefs,
	}
}

func appendManagedProgressiveAttemptV0(
	result ManagedProgressiveLoopResultV0,
	attemptNumber int,
	loop ProgressiveLoopResultV0,
) ManagedProgressiveLoopResultV0 {
	result.Final = loop
	result.Attempts = append(result.Attempts, ManagedProgressiveLoopAttemptV0{
		AttemptNumber: attemptNumber,
		Result:        loop,
	})
	return result
}

func appendManagedProgressiveWaitV0(
	result ManagedProgressiveLoopResultV0,
	waitNumber int,
	wait ExternalProgressWaitResultV0,
) ManagedProgressiveLoopResultV0 {
	result.ExternalWaits = append(result.ExternalWaits, ManagedProgressiveLoopExternalWaitV0{
		WaitNumber:   waitNumber,
		Continue:     wait.Continue,
		EvidenceRefs: compactStringsV0(wait.EvidenceRefs),
	})
	return result
}
