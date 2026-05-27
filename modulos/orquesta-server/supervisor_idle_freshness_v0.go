package orquestaserver

import (
	"context"
	"strconv"
	"time"
)

func (runtime *RuntimeV0) idleSelfImprovementRunFreshnessV0(
	ctx context.Context,
	knownRunRefs []string,
	knownRequestRefs []string,
) IdleSelfImprovementRunFreshnessResultV0 {
	port, ok := runtime.supervisor.(IdleSelfImprovementRunFreshnessPortV0)
	if !ok || port == nil {
		return IdleSelfImprovementRunFreshnessResultV0{}
	}
	result, err := port.RetryableIdleSelfImprovementRunRefsV0(ctx, IdleSelfImprovementRunFreshnessRequestV0{
		KnownRunRefs:     append([]string(nil), knownRunRefs...),
		KnownRequestRefs: append([]string(nil), knownRequestRefs...),
	})
	if err != nil {
		runtime.auditEventV0(ctx, "idle_self_improvement_freshness_error", "error", err.Error(), map[string]interface{}{
			"known_run_refs":     append([]string(nil), knownRunRefs...),
			"known_request_refs": append([]string(nil), knownRequestRefs...),
		})
		return IdleSelfImprovementRunFreshnessResultV0{}
	}
	result.RetryableRunRefs = compactConfigStringsV0(result.RetryableRunRefs)
	result.RetryableRequestRefs = compactConfigStringsV0(result.RetryableRequestRefs)
	return result
}

func idleSelfImprovementWithoutRetryableRefsV0(values []string, retryableRunRefs []string, retryableRequestRefs []string) []string {
	retryable := map[string]bool{}
	for _, value := range append(append([]string(nil), retryableRunRefs...), retryableRequestRefs...) {
		if ref := idleSelfImprovementRequestRefFromRunRefV0(value); ref != "" {
			retryable[ref] = true
		}
	}
	out := make([]string, 0, len(values))
	for _, value := range compactConfigStringsV0(values) {
		if retryable[idleSelfImprovementRequestRefFromRunRefV0(value)] {
			continue
		}
		out = append(out, value)
	}
	return compactConfigStringsV0(out)
}

func (decision idleSelfImprovementScheduleDecisionV0) QueueSizeString() string {
	return strconv.Itoa(decision.QueueSize)
}

func (decision idleSelfImprovementScheduleDecisionV0) TargetQueueString() string {
	return strconv.Itoa(decision.TargetQueue)
}

func (decision idleSelfImprovementScheduleDecisionV0) FreeCapacityString() string {
	return strconv.Itoa(decision.FreeCapacity)
}

func (runtime *RuntimeV0) markIdleSelfImprovementCheckedV0(
	ctx context.Context,
	reason string,
	now time.Time,
) {
	runtime.persistStateTransitionV0(ctx, runtime.tracker.MarkIdleSelfImprovementCheckedV0(reason, now), "idle_self_improvement_checked")
}
