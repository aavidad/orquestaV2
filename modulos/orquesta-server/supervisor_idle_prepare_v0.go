package orquestaserver

import (
	"context"
	"strings"
)

func (runtime *RuntimeV0) idleSelfImprovementRequestsV0(
	ctx context.Context,
	base IdleSelfImprovementRequestV0,
	decision idleSelfImprovementScheduleDecisionV0,
) []IdleSelfImprovementRequestV0 {
	requests := []IdleSelfImprovementRequestV0{base}
	if planner, ok := runtime.supervisor.(IdleSelfImprovementPlannerPortV0); ok && planner != nil {
		planned, err := planner.PlanIdleSelfImprovementV0(ctx, IdleSelfImprovementPlanRequestV0{
			BaseRequest:      base,
			MaxRequests:      decision.MaxRequests,
			Trigger:          decision.Trigger,
			QueueSize:        decision.QueueSize,
			FreeCapacity:     decision.FreeCapacity,
			Skips:            decision.Skips,
			KnownRunRefs:     append([]string(nil), decision.KnownRunRefs...),
			KnownRequestRefs: append([]string(nil), decision.KnownRequestRefs...),
			RetryableRunRefs: append([]string(nil), decision.RetryableRunRefs...),
			RetryableRequestRefs: append(
				[]string(nil),
				decision.RetryableRequestRefs...,
			),
			RetryableEvidenceRefs: append(
				[]string(nil),
				decision.RetryableEvidenceRefs...,
			),
		})
		if err != nil {
			runtime.auditEventV0(ctx, "idle_self_improvement_plan_error", "error", err.Error(), nil)
			return nil
		}
		requests = planned.Requests
	}
	requests = runtime.filterIdleSelfImprovementRequestsV0(ctx, base, requests, decision)
	requests = normalizeIdleSelfImprovementCausalRequestsV0(requests)
	requests = filterIdleSelfImprovementKnownQueueRequestsV0(requests, decision)
	if decision.MaxRequests > 0 && len(requests) > decision.MaxRequests {
		requests = requests[:decision.MaxRequests]
	}
	return requests
}

func (runtime *RuntimeV0) filterIdleSelfImprovementRequestsV0(
	ctx context.Context,
	base IdleSelfImprovementRequestV0,
	requests []IdleSelfImprovementRequestV0,
	decision idleSelfImprovementScheduleDecisionV0,
) []IdleSelfImprovementRequestV0 {
	filter, ok := runtime.supervisor.(IdleSelfImprovementRequestFilterPortV0)
	if !ok || filter == nil {
		return requests
	}
	result, err := filter.FilterIdleSelfImprovementRequestsV0(ctx, IdleSelfImprovementRequestFilterRequestV0{
		BaseRequest: base,
		Requests:    append([]IdleSelfImprovementRequestV0(nil), requests...),
		Trigger:     decision.Trigger,
	})
	if err != nil {
		runtime.auditEventV0(ctx, "idle_self_improvement_filter_error", "error", err.Error(), nil)
		return nil
	}
	return result.Requests
}

func filterIdleSelfImprovementKnownQueueRequestsV0(
	requests []IdleSelfImprovementRequestV0,
	decision idleSelfImprovementScheduleDecisionV0,
) []IdleSelfImprovementRequestV0 {
	if len(requests) == 0 {
		return []IdleSelfImprovementRequestV0{}
	}
	known := map[string]bool{}
	for _, ref := range append(append([]string(nil), decision.KnownRequestRefs...), decision.KnownRunRefs...) {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		known[ref] = true
		known[idleSelfImprovementRequestRefFromRunRefV0(ref)] = true
		known[idleSelfImprovementCausalBaseRefV0(ref)] = true
	}
	out := make([]IdleSelfImprovementRequestV0, 0, len(requests))
	for _, request := range requests {
		baseRef := idleSelfImprovementCausalBaseRefV0(request.RequestRef)
		target := idleSelfImprovementDedupeTargetV0(request)
		if known[request.RequestRef] || known[baseRef] || known[target] {
			continue
		}
		out = append(out, request)
		known[request.RequestRef] = true
		known[baseRef] = true
		known[target] = true
	}
	if out == nil {
		return []IdleSelfImprovementRequestV0{}
	}
	return out
}

func (runtime *RuntimeV0) prepareIdleSelfImprovementBatchV0(
	ctx context.Context,
	port IdleSelfImprovementPortV0,
	requests []IdleSelfImprovementRequestV0,
) {
	failed := false
	failures := make([]IdleSelfImprovementResultV0, 0, len(requests))
	for _, request := range requests {
		result, err := port.PrepareIdleSelfImprovementV0(ctx, request)
		if err != nil {
			failed = true
			failures = append(failures, idleSelfImprovementPrepareFailureFromErrorV0(request, err))
			continue
		}
		if !result.Accepted {
			failure := idleSelfImprovementPrepareFailureFromResultV0(request, result)
			if idleSelfImprovementPreparePendingV0(failure) {
				runtime.persistStateTransitionV0(
					ctx,
					runtime.tracker.MarkIdleSelfImprovementPreparePendingV0(failure, runtime.clock.Now()),
					"idle_self_improvement_prepare_pending",
				)
				continue
			}
			failed = true
			failures = append(failures, failure)
			continue
		}
		if failed {
			result.NextActions = compactConfigStringsV0(append(
				result.NextActions,
				"repair_failed_prepare_requests_and_retry",
			))
		}
		runtime.persistStateTransitionV0(
			ctx,
			runtime.tracker.MarkIdleSelfImprovementPreparedV0(result, runtime.clock.Now()),
			"idle_self_improvement_prepared",
		)
	}
	if failed {
		runtime.markIdleSelfImprovementPrepareFailedV0(ctx, failures)
	}
}

func idleSelfImprovementPrepareFailureFromErrorV0(
	request IdleSelfImprovementRequestV0,
	err error,
) IdleSelfImprovementResultV0 {
	message := "prepare_failed"
	if err != nil {
		message = err.Error()
	}
	return idleSelfImprovementPrepareFailureFromResultV0(request, IdleSelfImprovementResultV0{
		Accepted: false,
		Status:   "error",
		Message:  message,
	})
}

func idleSelfImprovementPrepareFailureFromResultV0(
	request IdleSelfImprovementRequestV0,
	result IdleSelfImprovementResultV0,
) IdleSelfImprovementResultV0 {
	result.Accepted = false
	if strings.TrimSpace(result.RequestRef) == "" {
		result.RequestRef = request.RequestRef
	}
	if strings.TrimSpace(result.Status) == "" {
		result.Status = "error"
	}
	if strings.TrimSpace(result.Message) == "" {
		result.Message = "prepare_failed"
	}
	if len(result.EvidenceRefs) == 0 {
		result.EvidenceRefs = append([]string(nil), request.EvidenceRefs...)
	}
	return result
}

func (runtime *RuntimeV0) markIdleSelfImprovementPrepareFailedV0(
	ctx context.Context,
	failures []IdleSelfImprovementResultV0,
) {
	state := runtime.tracker.SnapshotV0()
	if strings.HasPrefix(state.IdleSelfImprovementReason, "prepared") {
		return
	}
	failure := firstIdleSelfImprovementPrepareFailureV0(failures)
	runtime.persistStateTransitionV0(
		ctx,
		runtime.tracker.MarkIdleSelfImprovementPrepareFailedV0(failure, runtime.clock.Now()),
		"idle_self_improvement_prepare_failed",
	)
}

func firstIdleSelfImprovementPrepareFailureV0(
	failures []IdleSelfImprovementResultV0,
) IdleSelfImprovementResultV0 {
	for _, failure := range failures {
		if strings.TrimSpace(failure.Message) != "" ||
			strings.TrimSpace(failure.RequestRef) != "" ||
			strings.TrimSpace(failure.RunRef) != "" ||
			len(failure.NextActions) > 0 ||
			len(failure.EvidenceRefs) > 0 {
			return failure
		}
	}
	return IdleSelfImprovementResultV0{
		Accepted: false,
		Status:   "error",
		Message:  "prepare_failed",
	}
}
