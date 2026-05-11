package orquestacionnucleoapp

import (
	"context"

	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
)

type progressiveDispatchWaitResultV0 struct {
	Once  []OutboxDispatchOnceResultV0
	Batch []OutboxDispatchBatchRunResultV0
}

func (service ServiceV0) dispatchProgressiveWaitV0(
	ctx context.Context,
	request ProgressiveLoopRequestV0,
) (progressiveDispatchWaitResultV0, error) {
	result := progressiveDispatchWaitResultV0{}
	dispatchedCount := 0
	for dispatchedCount < request.MaxDispatchesPerWait {
		dispatched := false
		for _, dispatcher := range request.BatchDispatchers {
			batch, err := RunOutboxDispatchBatchV0(ctx, OutboxDispatchBatchRunRequestV0{
				RunRef:      request.RunRef,
				TargetPort:  dispatcher.TargetPort,
				MessageType: dispatcher.MessageType,
				MaxReady:    dispatcher.MaxReady,
				Reader:      dispatcher.Reader,
				Claimer:     dispatcher.Claimer,
				Executor:    dispatcher.Executor,
				Acker:       dispatcher.Acker,
			})
			result.Batch = append(result.Batch, batch)
			if err != nil {
				return result, err
			}
			if batch.Status == OutboxDispatchBatchRunDispatchedV0 {
				dispatchedCount += progressiveBatchDispatchWeightV0(batch)
				dispatched = true
				break
			}
		}
		if dispatched {
			continue
		}
		for _, dispatcher := range request.Dispatchers {
			once, err := RunOutboxDispatchOnceV0(ctx, OutboxDispatchOnceRequestV0{
				RunRef:      request.RunRef,
				TargetPort:  dispatcher.TargetPort,
				MessageType: dispatcher.MessageType,
				Reader:      dispatcher.Reader,
				Claimer:     dispatcher.Claimer,
				Executor:    dispatcher.Executor,
				Acker:       dispatcher.Acker,
			})
			result.Once = append(result.Once, once)
			if err != nil {
				return result, err
			}
			if once.Status == string(orquestaoutboxdispatch.RunOutboxDispatchOnceDispatchedV0) {
				dispatchedCount++
				dispatched = true
				break
			}
		}
		if !dispatched {
			return result, nil
		}
	}
	return result, nil
}

func hasProgressiveDispatchProgressV0(
	result progressiveDispatchWaitResultV0,
) bool {
	return hasProgressiveOnceDispatchedV0(result.Once) ||
		hasProgressiveBatchDispatchedV0(result.Batch)
}

func hasProgressiveOnceDispatchedV0(
	results []OutboxDispatchOnceResultV0,
) bool {
	for _, result := range results {
		if result.Status == string(orquestaoutboxdispatch.RunOutboxDispatchOnceDispatchedV0) {
			return true
		}
	}
	return false
}

func hasProgressiveBatchDispatchedV0(
	results []OutboxDispatchBatchRunResultV0,
) bool {
	for _, result := range results {
		if result.Status == OutboxDispatchBatchRunDispatchedV0 {
			return true
		}
	}
	return false
}

func progressiveBatchDispatchWeightV0(
	result OutboxDispatchBatchRunResultV0,
) int {
	if result.AckedCount > 0 {
		return result.AckedCount
	}
	return 1
}
