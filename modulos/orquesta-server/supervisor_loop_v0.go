package orquestaserver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

func (runtime *RuntimeV0) runSupervisorLoopV0(ctx context.Context) {
	if runtime.supervisor == nil {
		return
	}
	runtime.runSupervisorTickAsyncV0(ctx)
	ticker := time.NewTicker(runtime.config.TickInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runtime.runSupervisorTickAsyncV0(ctx)
		}
	}
}

func (runtime *RuntimeV0) runSupervisorTickAsyncV0(ctx context.Context) bool {
	if !atomic.CompareAndSwapInt32(&runtime.supervisorTickActive, 0, 1) {
		return false
	}
	go func() {
		defer atomic.StoreInt32(&runtime.supervisorTickActive, 0)
		runtime.runSupervisorTickV0(ctx)
	}()
	return true
}

func (runtime *RuntimeV0) runSupervisorTickV0(ctx context.Context) {
	defer runtime.recoverSupervisorTickPanicV0(ctx)
	if err := ctx.Err(); err != nil {
		return
	}
	command := runtime.config.SupervisorCommand
	if command.MaxTicks <= 0 {
		command.MaxTicks = DefaultSupervisorMaxTicksV0
	}
	runtime.auditEventV0(ctx, "supervisor_tick_start", "running", "", map[string]interface{}{
		"command": command,
	})
	result, err := runtime.supervisor.RunGlobalSupervisorV0(ctx, command)
	now := runtime.clock.Now()
	if err != nil {
		runtime.auditEventV0(ctx, "supervisor_tick_error", "error", err.Error(), map[string]interface{}{
			"command": command,
			"result":  result,
		})
		runtime.persistStateV0(ctx, runtime.tracker.MarkSupervisorErrorV0(
			command,
			result,
			err.Error(),
			now,
		))
		return
	}
	runtime.auditEventV0(ctx, "supervisor_tick_result", "ok", "", map[string]interface{}{
		"command": command,
		"result":  result,
	})
	runtime.persistStateV0(ctx, runtime.tracker.MarkSupervisorV0(command, result, now))
	runtime.maybeScheduleIdleSelfImprovementV0(ctx, result, now)
}

func (runtime *RuntimeV0) maybeScheduleIdleSelfImprovementV0(
	ctx context.Context,
	result orquestarunsupervisor.RunSupervisorResultV0,
	now time.Time,
) {
	if runtime.config.IdleSelfImprovementAfter <= 0 {
		runtime.auditEventV0(ctx, "idle_self_improvement_check", "skipped", "", map[string]interface{}{"reason": "disabled"})
		runtime.markIdleSelfImprovementCheckedV0(ctx, "disabled", now)
		return
	}
	port, ok := runtime.supervisor.(IdleSelfImprovementPortV0)
	if !ok || port == nil {
		runtime.auditEventV0(ctx, "idle_self_improvement_check", "skipped", "", map[string]interface{}{"reason": "port_unavailable"})
		runtime.markIdleSelfImprovementCheckedV0(ctx, "port_unavailable", now)
		return
	}
	decision := runtime.idleSelfImprovementScheduleDecisionV0(ctx, result, now)
	if !decision.Schedule {
		runtime.auditEventV0(ctx, "idle_self_improvement_check", decision.AuditStatus, "", decision.AuditPayload())
		runtime.markIdleSelfImprovementCheckedV0(ctx, decision.Reason, now)
		return
	}
	request := runtime.idleSelfImprovementRequestV0(decision, now)
	requests := runtime.idleSelfImprovementRequestsV0(ctx, request, decision)
	if len(requests) == 0 {
		runtime.auditEventV0(ctx, "idle_self_improvement_check", "skipped", "", map[string]interface{}{"reason": "planner_empty", "request": request})
		runtime.markIdleSelfImprovementCheckedV0(ctx, "planner_empty", now)
		return
	}
	runtime.auditEventV0(ctx, "idle_self_improvement_scheduled", "scheduled", "", map[string]interface{}{"requests": requests})
	runtime.persistStateV0(ctx, runtime.tracker.MarkIdleSelfImprovementScheduledV0(now))
	go runtime.prepareIdleSelfImprovementBatchV0(ctx, port, requests)
}

const (
	idleSelfImprovementTriggerIdleV0         = "idle_self_improvement"
	idleSelfImprovementTriggerCapacityFreeV0 = "capacity_free"
)

type idleSelfImprovementScheduleDecisionV0 struct {
	Schedule         bool
	AuditStatus      string
	Reason           string
	Trigger          string
	IdleSince        time.Time
	LastAttempt      time.Time
	InFlight         bool
	Accepted         bool
	QueueSize        int
	TargetQueue      int
	FreeCapacity     int
	MaxRequests      int
	Skips            int
	KnownRunRefs     []string
	KnownRequestRefs []string
}

func (runtime *RuntimeV0) idleSelfImprovementScheduleDecisionV0(
	_ context.Context,
	result orquestarunsupervisor.RunSupervisorResultV0,
	now time.Time,
) idleSelfImprovementScheduleDecisionV0 {
	idleSince, lastAttempt, inFlight, accepted := runtime.tracker.IdleSelfImprovementWindowV0()
	metrics := collectSupervisorResultMetricsV0(result)
	knownRunRefs := idleSelfImprovementKnownRunRefsV0(result)
	queueSize := len(knownRunRefs)
	if metrics.QueueSize > queueSize {
		queueSize = metrics.QueueSize
	}
	decision := idleSelfImprovementScheduleDecisionV0{
		AuditStatus:      "skipped",
		Reason:           "supervisor_not_idle",
		Trigger:          idleSelfImprovementTriggerIdleV0,
		IdleSince:        idleSince,
		LastAttempt:      lastAttempt,
		InFlight:         inFlight,
		Accepted:         accepted,
		QueueSize:        queueSize,
		TargetQueue:      runtime.config.IdleSelfImprovementTargetQueue,
		MaxRequests:      runtime.config.IdleSelfImprovementMaxRequests,
		Skips:            metrics.Skips,
		KnownRunRefs:     knownRunRefs,
		KnownRequestRefs: idleSelfImprovementKnownRequestRefsV0(knownRunRefs),
	}
	if supervisorResultIsIdleForSelfImprovementV0(result) {
		return runtime.idleSelfImprovementIdleDecisionV0(decision, now)
	}
	return runtime.idleSelfImprovementCapacityDecisionV0(decision, now)
}

func (runtime *RuntimeV0) idleSelfImprovementIdleDecisionV0(
	decision idleSelfImprovementScheduleDecisionV0,
	now time.Time,
) idleSelfImprovementScheduleDecisionV0 {
	decision.Trigger = idleSelfImprovementTriggerIdleV0
	if decision.IdleSince.IsZero() || now.Sub(decision.IdleSince) < runtime.config.IdleSelfImprovementAfter {
		decision.AuditStatus = "waiting"
		decision.Reason = "idle_window_waiting"
		return decision
	}
	if decision.InFlight || idleSelfImprovementAttemptBlocksV0(
		decision.IdleSince,
		decision.LastAttempt,
		now,
		runtime.config.IdleSelfImprovementAfter,
		decision.Accepted,
	) {
		decision.AuditStatus = "blocked"
		decision.Reason = "attempt_blocked"
		return decision
	}
	decision.Schedule = true
	decision.AuditStatus = "scheduled"
	decision.Reason = "scheduled"
	return decision
}

func (runtime *RuntimeV0) idleSelfImprovementCapacityDecisionV0(
	decision idleSelfImprovementScheduleDecisionV0,
	now time.Time,
) idleSelfImprovementScheduleDecisionV0 {
	decision.Trigger = idleSelfImprovementTriggerCapacityFreeV0
	if _, ok := runtime.supervisor.(IdleSelfImprovementPlannerPortV0); !ok {
		decision.AuditStatus = "skipped"
		decision.Reason = "capacity_planner_unavailable"
		return decision
	}
	if decision.InFlight {
		decision.AuditStatus = "blocked"
		decision.Reason = "capacity_attempt_in_flight"
		return decision
	}
	if idleSelfImprovementCooldownBlocksV0(decision.LastAttempt, now, runtime.config.IdleSelfImprovementAfter) {
		decision.AuditStatus = "blocked"
		decision.Reason = "capacity_cooldown"
		return decision
	}
	if decision.QueueSize <= 0 {
		decision.AuditStatus = "skipped"
		decision.Reason = "capacity_queue_unknown"
		return decision
	}
	if decision.Skips > 0 {
		decision.AuditStatus = "skipped"
		decision.Reason = "capacity_pending_skips"
		return decision
	}
	target := runtime.config.IdleSelfImprovementTargetQueue
	if target <= 0 {
		target = runtime.config.IdleSelfImprovementMaxRequests
	}
	decision.TargetQueue = target
	decision.FreeCapacity = target - decision.QueueSize
	if decision.FreeCapacity <= 0 {
		decision.AuditStatus = "skipped"
		decision.Reason = "capacity_full"
		return decision
	}
	decision.MaxRequests = minPositiveIdleSelfImprovementV0(runtime.config.IdleSelfImprovementMaxRequests, decision.FreeCapacity)
	if decision.MaxRequests <= 0 {
		decision.AuditStatus = "skipped"
		decision.Reason = "capacity_no_request_budget"
		return decision
	}
	decision.Schedule = true
	decision.AuditStatus = "scheduled"
	decision.Reason = "capacity_free"
	return decision
}

func (decision idleSelfImprovementScheduleDecisionV0) AuditPayload() map[string]interface{} {
	return map[string]interface{}{
		"reason":             decision.Reason,
		"trigger":            decision.Trigger,
		"idle_since":         formatTimeV0(decision.IdleSince),
		"last_attempt":       formatTimeV0(decision.LastAttempt),
		"in_flight":          decision.InFlight,
		"accepted":           decision.Accepted,
		"queue_size":         decision.QueueSize,
		"target_queue":       decision.TargetQueue,
		"free_capacity":      decision.FreeCapacity,
		"max_requests":       decision.MaxRequests,
		"skips":              decision.Skips,
		"known_run_refs":     append([]string(nil), decision.KnownRunRefs...),
		"known_request_refs": append([]string(nil), decision.KnownRequestRefs...),
	}
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
	runtime.persistStateV0(ctx, runtime.tracker.MarkIdleSelfImprovementCheckedV0(reason, now))
}

func (runtime *RuntimeV0) prepareIdleSelfImprovementBatchV0(
	ctx context.Context,
	port IdleSelfImprovementPortV0,
	requests []IdleSelfImprovementRequestV0,
) {
	aggregate := IdleSelfImprovementResultV0{}
	failed := ""
	for _, request := range requests {
		runtime.auditEventV0(ctx, "idle_self_improvement_prepare_start", "running", "", map[string]interface{}{"request": request})
		prepared, err := port.PrepareIdleSelfImprovementV0(ctx, request)
		now := runtime.clock.Now()
		if err != nil {
			runtime.auditEventV0(ctx, "idle_self_improvement_prepare_error", "error", err.Error(), map[string]interface{}{"request": request})
			runtime.persistStateV0(ctx, runtime.tracker.MarkIdleSelfImprovementErrorV0(err.Error(), now))
			return
		}
		if prepared.Accepted && strings.TrimSpace(prepared.RunRef) == "" {
			prepared.Accepted = false
			prepared.Message = firstNonEmptyIdleSelfImprovementV0(prepared.Message, "idle_self_improvement_prepare_run_missing_run_ref")
		}
		if !prepared.Accepted {
			failed = firstNonEmptyIdleSelfImprovementV0(failed, prepared.Message, prepared.Status, "idle_self_improvement_prepare_run_not_accepted")
		}
		if aggregate.Status == "" {
			aggregate.Status = strings.TrimSpace(prepared.Status)
		}
		if prepared.Accepted {
			aggregate.Accepted = true
			aggregate.RunRef = firstNonEmptyIdleSelfImprovementV0(aggregate.RunRef, prepared.RunRef)
			aggregate.RequestRef = firstNonEmptyIdleSelfImprovementV0(aggregate.RequestRef, prepared.RequestRef)
			aggregate.EvidenceRefs = compactConfigStringsV0(append(aggregate.EvidenceRefs, prepared.EvidenceRefs...))
		}
		aggregate.NextActions = compactConfigStringsV0(append(aggregate.NextActions, prepared.NextActions...))
		if prepared.Message != "" {
			aggregate.Message = prepared.Message
		}
		runtime.auditEventV0(ctx, "idle_self_improvement_prepare_result", "ok", "", map[string]interface{}{"request": request, "result": prepared})
	}
	now := runtime.clock.Now()
	if failed != "" || !aggregate.Accepted {
		message := firstNonEmptyIdleSelfImprovementV0(failed, aggregate.Message, aggregate.Status, "idle_self_improvement_prepare_run_not_accepted")
		runtime.auditEventV0(ctx, "idle_self_improvement_prepare_batch", "not_accepted", message, map[string]interface{}{"result": aggregate})
		runtime.persistStateV0(ctx, runtime.tracker.MarkIdleSelfImprovementErrorV0(message, now))
		return
	}
	runtime.auditEventV0(ctx, "idle_self_improvement_prepare_batch", "accepted", "", map[string]interface{}{"result": aggregate})
	runtime.persistStateV0(ctx, runtime.tracker.MarkIdleSelfImprovementPreparedV0(aggregate, now))
}

func (runtime *RuntimeV0) idleSelfImprovementRequestsV0(
	ctx context.Context,
	base IdleSelfImprovementRequestV0,
	decision idleSelfImprovementScheduleDecisionV0,
) []IdleSelfImprovementRequestV0 {
	planner, ok := runtime.supervisor.(IdleSelfImprovementPlannerPortV0)
	if !ok || planner == nil {
		return []IdleSelfImprovementRequestV0{base}
	}
	plan, err := planner.PlanIdleSelfImprovementV0(ctx, IdleSelfImprovementPlanRequestV0{
		BaseRequest:      base,
		MaxRequests:      decision.MaxRequests,
		Trigger:          decision.Trigger,
		QueueSize:        decision.QueueSize,
		FreeCapacity:     decision.FreeCapacity,
		KnownRunRefs:     append([]string(nil), decision.KnownRunRefs...),
		KnownRequestRefs: append([]string(nil), decision.KnownRequestRefs...),
	})
	if err != nil {
		now := runtime.clock.Now()
		runtime.auditEventV0(ctx, "idle_self_improvement_plan_error", "error", err.Error(), map[string]interface{}{"request": base})
		runtime.persistStateV0(ctx, runtime.tracker.MarkIdleSelfImprovementErrorV0(err.Error(), now))
		return []IdleSelfImprovementRequestV0{base}
	}
	requests := normalizeIdleSelfImprovementRequestsV0(plan.Requests, decision.MaxRequests)
	runtime.auditEventV0(ctx, "idle_self_improvement_plan_result", "ok", "", map[string]interface{}{"base_request": base, "plan": plan, "selected": requests})
	if len(requests) == 0 {
		return []IdleSelfImprovementRequestV0{}
	}
	return requests
}

func (runtime *RuntimeV0) idleSelfImprovementRequestV0(
	decision idleSelfImprovementScheduleDecisionV0,
	now time.Time,
) IdleSelfImprovementRequestV0 {
	seedTime := decision.IdleSince
	if seedTime.IsZero() {
		seedTime = now.UTC().Truncate(runtime.config.IdleSelfImprovementAfter)
	}
	requestRef := "request-ref-idle-self-improvement-" + idleSelfImprovementHashV0(
		runtime.config.ProjectWorkDir+"|"+decision.Trigger+"|"+formatTimeV0(seedTime),
	)
	contextRefs := append([]string(nil), runtime.config.IdleSelfImprovementContextRefs...)
	contextRefs = append(contextRefs,
		"idle_since:"+formatTimeV0(decision.IdleSince),
		"trigger:orquesta-server-"+decision.Trigger,
		"queue_size:"+strings.TrimSpace(firstNonEmptyIdleSelfImprovementV0(decision.QueueSizeString(), "0")),
		"free_capacity:"+strings.TrimSpace(firstNonEmptyIdleSelfImprovementV0(decision.FreeCapacityString(), "0")),
	)
	evidenceRefs := idleSelfImprovementEvidenceRefsV0(runtime.config)
	if decision.Trigger == idleSelfImprovementTriggerCapacityFreeV0 {
		evidenceRefs = compactConfigStringsV0(append(evidenceRefs,
			"evidence-ref-orquesta-server-capacity-free",
		))
	}
	failureKind := "idle_capacity"
	failureSummary := "cola sin ejecuciones durante " + runtime.config.IdleSelfImprovementAfter.String()
	if decision.Trigger == idleSelfImprovementTriggerCapacityFreeV0 {
		failureKind = "capacity_free"
		failureSummary = "capacidad libre para automejora: cola=" + decision.QueueSizeString() +
			" objetivo=" + decision.TargetQueueString() +
			" libres=" + decision.FreeCapacityString()
	}
	return IdleSelfImprovementRequestV0{
		RequestRef:         requestRef,
		CorrelationID:      "corr-" + requestRef,
		ProjectRef:         runtime.config.IdleSelfImprovementProjectRef,
		WorktreeRef:        runtime.config.IdleSelfImprovementWorktreeRef,
		BranchRef:          runtime.config.IdleSelfImprovementBranchRef,
		RequestedBy:        "orquesta-server",
		Source:             "orquesta-server-idle-supervisor",
		FailureKind:        failureKind,
		FailureSummary:     failureSummary,
		SuggestedArea:      runtime.config.IdleSelfImprovementSuggestedArea,
		WriteSet:           append([]string(nil), runtime.config.IdleSelfImprovementWriteSet...),
		RequiredTests:      append([]string(nil), runtime.config.IdleSelfImprovementRequiredTests...),
		AcceptanceCriteria: append([]string(nil), runtime.config.IdleSelfImprovementAcceptance...),
		CompactRules:       append([]string(nil), runtime.config.IdleSelfImprovementCompactRules...),
		ContextRefs:        contextRefs,
		EvidenceRefs:       evidenceRefs,
		OccurredAt:         formatTimeV0(now),
		PriorityScore:      runtime.config.IdleSelfImprovementPriorityScore,
	}
}

func idleSelfImprovementHashV0(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(sum[:])[:12]
}

func normalizeIdleSelfImprovementRequestsV0(
	requests []IdleSelfImprovementRequestV0,
	maxRequests int,
) []IdleSelfImprovementRequestV0 {
	if maxRequests <= 0 {
		maxRequests = DefaultIdleSelfImprovementMaxRequestsV0
	}
	out := make([]IdleSelfImprovementRequestV0, 0, len(requests))
	seen := map[string]bool{}
	for _, request := range requests {
		request.RequestRef = strings.TrimSpace(request.RequestRef)
		if request.RequestRef == "" || seen[request.RequestRef] {
			continue
		}
		request.CorrelationID = strings.TrimSpace(request.CorrelationID)
		if request.CorrelationID == "" {
			request.CorrelationID = "corr-" + request.RequestRef
		}
		seen[request.RequestRef] = true
		out = append(out, request)
		if len(out) >= maxRequests {
			break
		}
	}
	return out
}
func firstNonEmptyIdleSelfImprovementV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
func idleSelfImprovementAttemptBlocksV0(
	idleSince time.Time,
	lastAttempt time.Time,
	now time.Time,
	retryAfter time.Duration,
	accepted bool,
) bool {
	if lastAttempt.IsZero() || lastAttempt.Before(idleSince) {
		return false
	}
	_ = accepted
	return idleSelfImprovementCooldownBlocksV0(lastAttempt, now, retryAfter)
}
func idleSelfImprovementCooldownBlocksV0(
	lastAttempt time.Time,
	now time.Time,
	retryAfter time.Duration,
) bool {
	if lastAttempt.IsZero() {
		return false
	}
	if retryAfter <= 0 {
		retryAfter = DefaultIdleSelfImprovementAfterV0
	}
	return now.Sub(lastAttempt) < retryAfter
}
func minPositiveIdleSelfImprovementV0(left int, right int) int {
	if left <= 0 {
		return right
	}
	if right <= 0 || left < right {
		return left
	}
	return right
}
func idleSelfImprovementEvidenceRefsV0(config ConfigV0) []string {
	return compactConfigStringsV0(append(
		append([]string(nil), config.IdleSelfImprovementEvidenceRefs...),
		"evidence-ref-orquesta-server-idle-no-execution",
		"evidence-ref-orquesta-server-idle-window",
	))
}
func supervisorResultIsIdleForSelfImprovementV0(
	result orquestarunsupervisor.RunSupervisorResultV0,
) bool {
	if strings.TrimSpace(result.StopReason) != orquestarunsupervisor.RunSupervisorStopNoExecutionV0 ||
		result.TotalExecutions > 0 ||
		result.TotalSkips > 0 {
		return false
	}
	for _, tick := range result.Ticks {
		if len(tick.Result.Executions) > 0 ||
			len(tick.Result.Skips) > 0 {
			return false
		}
	}
	return true
}
func idleSelfImprovementKnownRunRefsV0(
	result orquestarunsupervisor.RunSupervisorResultV0,
) []string {
	var refs []string
	for _, tick := range result.Ticks {
		for _, ranked := range tick.Result.Ranked {
			refs = append(refs, ranked.RunRef)
		}
		for _, execution := range tick.Result.Executions {
			refs = append(refs, execution.RunRef)
		}
		for _, skip := range tick.Result.Skips {
			refs = append(refs, skip.RunRef)
		}
	}
	return compactConfigStringsV0(refs)
}
func idleSelfImprovementKnownRequestRefsV0(runRefs []string) []string {
	out := make([]string, 0, len(runRefs))
	for _, runRef := range runRefs {
		out = append(out, idleSelfImprovementRequestRefFromRunRefV0(runRef))
	}
	return compactConfigStringsV0(out)
}
func idleSelfImprovementRequestRefFromRunRefV0(runRef string) string {
	runRef = strings.TrimSpace(runRef)
	if retryIndex := strings.Index(runRef, "-retry-"); retryIndex > 0 {
		runRef = runRef[:retryIndex]
	}
	return runRef
}
