package orquestaserver

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

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
		CompactRules: compactConfigStringsV0(append(append([]string(nil), runtime.config.IdleSelfImprovementCompactRules...),
			"un agente padre por tarea; subagentes maximo 6 si ayudan",
		)),
		ContextRefs:   contextRefs,
		EvidenceRefs:  evidenceRefs,
		OccurredAt:    formatTimeV0(now),
		PriorityScore: runtime.config.IdleSelfImprovementPriorityScore,
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

func idleSelfImprovementEvidenceRefsV0(config ConfigV0) []string {
	return compactConfigStringsV0(append(
		append([]string(nil), config.IdleSelfImprovementEvidenceRefs...),
		"evidence-ref-orquesta-server-idle-no-execution",
		"evidence-ref-orquesta-server-idle-window",
	))
}
