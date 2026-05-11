package orquestaruntime

import (
	"context"
	"sync"
)

const externalAgentProcessBatchMaxLaunchAttemptsV0 = 2

func LaunchExternalAgentProcessBatchV0(
	ctx context.Context,
	items []ExternalAgentProcessBatchItemV0,
	maxConcurrency int,
) []ExternalAgentProcessBatchResultV0 {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(items) == 0 {
		return nil
	}
	limit := externalAgentProcessBatchLimitV0(maxConcurrency, len(items))
	results := make([]ExternalAgentProcessBatchResultV0, len(items))
	sem := make(chan struct{}, limit)
	var wg sync.WaitGroup

	for index := range items {
		item := items[index]
		results[index] = externalAgentProcessBatchResultV0(
			index,
			item,
			ExternalAgentProcessLaunchResultV0{},
		)
		sem <- struct{}{}
		wg.Add(1)
		go func(index int, item ExternalAgentProcessBatchItemV0) {
			defer wg.Done()
			defer func() { <-sem }()

			result := launchExternalAgentProcessBatchItemV0(ctx, item)
			results[index] = externalAgentProcessBatchResultV0(index, item, result)
		}(index, item)
	}

	wg.Wait()
	return results
}

func launchExternalAgentProcessBatchItemV0(
	ctx context.Context,
	item ExternalAgentProcessBatchItemV0,
) ExternalAgentProcessLaunchResultV0 {
	var result ExternalAgentProcessLaunchResultV0
	for attempt := 1; attempt <= externalAgentProcessBatchMaxLaunchAttemptsV0; attempt++ {
		result = LaunchExternalAgentProcessV0(
			ctx,
			item.Spec,
			item.Resolver,
			item.Runtime,
		)
		if !externalAgentProcessLaunchRetryableV0(result) {
			return result
		}
	}
	return result
}

func externalAgentProcessLaunchRetryableV0(
	result ExternalAgentProcessLaunchResultV0,
) bool {
	if result.Status != ExternalAgentProcessLaunchBlockedV0 {
		return false
	}
	for _, issue := range result.Issues {
		if issue.Retryable {
			return true
		}
	}
	return false
}

func externalAgentProcessBatchLimitV0(maxConcurrency int, itemCount int) int {
	if maxConcurrency < 1 {
		maxConcurrency = 1
	}
	if maxConcurrency > itemCount {
		return itemCount
	}
	return maxConcurrency
}

func externalAgentProcessBatchResultV0(
	index int,
	item ExternalAgentProcessBatchItemV0,
	result ExternalAgentProcessLaunchResultV0,
) ExternalAgentProcessBatchResultV0 {
	spec := item.Spec
	return ExternalAgentProcessBatchResultV0{
		Index:         index,
		ItemRef:       item.ItemRef,
		RequestID:     spec.RequestID,
		CorrelationID: spec.CorrelationID,
		ProfileRef:    spec.ProfileRef,
		ConnectorRef:  spec.ConnectorRef,
		RuntimeKind:   spec.RuntimeKind,
		Result:        result,
	}
}
