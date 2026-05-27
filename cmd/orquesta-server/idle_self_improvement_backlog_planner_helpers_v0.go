package main

import (
	"strconv"
	"strings"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func idleSelfImprovementExcludedRequestRefsV0(
	request orquestaserver.IdleSelfImprovementPlanRequestV0,
) map[string]bool {
	excluded := map[string]bool{}
	for _, value := range request.KnownRequestRefs {
		if ref := idleSelfImprovementNormalizeQueuedRequestRefV0(value); ref != "" {
			excluded[ref] = true
		}
	}
	for _, value := range request.KnownRunRefs {
		if ref := idleSelfImprovementNormalizeQueuedRequestRefV0(value); ref != "" {
			excluded[ref] = true
		}
	}
	return excluded
}

func idleSelfImprovementKnownAttemptsByBaseRefV0(
	request orquestaserver.IdleSelfImprovementPlanRequestV0,
) map[string]int {
	out := map[string]int{}
	for _, value := range append(append([]string(nil), request.KnownRequestRefs...), request.KnownRunRefs...) {
		if ref := idleSelfImprovementNormalizeQueuedRequestRefV0(value); ref != "" {
			out[ref]++
		}
	}
	return out
}

func idleSelfImprovementNormalizeQueuedRequestRefV0(value string) string {
	value = strings.TrimSpace(value)
	for _, marker := range []string{"-retry-", "-reconcile-"} {
		if index := strings.Index(value, marker); index > 0 {
			value = value[:index]
		}
	}
	return value
}

func idleSelfImprovementHasKnownBacklogWorkV0(excluded map[string]bool) bool {
	for ref := range excluded {
		if strings.HasPrefix(ref, "request-ref-autoprogramming-backlog-") {
			return true
		}
	}
	return false
}

func idleSelfImprovementPendingKnownRequestV0(
	request orquestaserver.IdleSelfImprovementRequestV0,
	section idleSelfImprovementBacklogSectionV0,
	knownAttempts map[string]int,
) orquestaserver.IdleSelfImprovementRequestV0 {
	if !section.PendingExplicit {
		return request
	}
	baseRef := idleSelfImprovementNormalizeQueuedRequestRefV0(request.RequestRef)
	attempt := knownAttempts[baseRef]
	if baseRef == "" || attempt <= 0 {
		return request
	}
	if attempt > 1 {
		return idleSelfImprovementReconcileExplicitPendingKnownRefV0(request, section, baseRef, attempt)
	}
	hash := idleSelfImprovementBacklogHashV0(strings.Join([]string{
		baseRef,
		section.TaskInstanceRef,
		strconv.Itoa(attempt),
		strings.Join(section.StateEvidenceRefs, ","),
	}, "|"))
	request.RequestRef = baseRef + "-retry-" + hash
	request.CorrelationID = "corr-" + request.RequestRef
	request.ContextRefs = compactServerStackStringsV0(append(request.ContextRefs,
		"previous_run_ref:"+baseRef,
		"pending_canonical_overrides_closed_attempt:true",
		"previous_attempt_count:"+strconv.Itoa(attempt),
	))
	request.AcceptanceCriteria = compactServerStackStringsV0(append(request.AcceptanceCriteria,
		"no cerrar la tarea si el backlog canonico sigue en Estado: pendiente; actualizar backlog o dejar bloqueo verificable",
	))
	request.EvidenceRefs = compactServerStackStringsV0(append(request.EvidenceRefs,
		"evidence-ref-autoprogramming-backlog-pending-overrides-closed-run",
	))
	return request
}

func idleSelfImprovementReconcileExplicitPendingKnownRefV0(
	request orquestaserver.IdleSelfImprovementRequestV0,
	section idleSelfImprovementBacklogSectionV0,
	baseRef string,
	attempt int,
) orquestaserver.IdleSelfImprovementRequestV0 {
	hash := idleSelfImprovementBacklogHashV0(strings.Join([]string{
		baseRef,
		section.TaskInstanceRef,
		strconv.Itoa(attempt),
		"reconcile",
		strings.Join(section.StateEvidenceRefs, ","),
	}, "|"))
	request.RequestRef = baseRef + "-reconcile-" + hash
	request.CorrelationID = "corr-" + request.RequestRef
	request.FailureKind = "backlog_documentation_review"
	request.FailureSummary = "reconciliar backlog pendiente tras intentos cerrados de " + section.Heading
	request.SuggestedArea = firstNonEmptyServerStackV0(section.Ref+"-reconciliacion", request.SuggestedArea)
	request.WriteSet = idleSelfImprovementBacklogDocumentReviewWriteSetV0(section)
	request.ContextRefs = compactServerStackStringsV0(append(request.ContextRefs,
		"previous_run_ref:"+baseRef,
		"pending_canonical_after_multiple_attempts:true",
		"previous_attempt_count:"+strconv.Itoa(attempt),
	))
	request.AcceptanceCriteria = compactServerStackStringsV0(append([]string(nil),
		"revisar los ACKs/receipts ya entregados de la tarea base",
		"si la evidencia cumple, actualizar el backlog canonico a cerrado con fecha, archivos y tests",
		"si falta evidencia real, dejar Estado: bloqueado o pendiente verificable con causa exacta y no tocar codigo",
		"no relanzar otra implementacion padre para la misma tarea desde esta reconciliacion",
	))
	request.CompactRules = compactServerStackStringsV0(append(request.CompactRules,
		"reconciliacion documental compacta; no reimplementar la tarea",
	))
	request.EvidenceRefs = compactServerStackStringsV0(append(request.EvidenceRefs,
		"evidence-ref-autoprogramming-backlog-pending-reconciliation",
	))
	return request
}

func idleSelfImprovementShouldAddScannerRequestV0(
	request orquestaserver.IdleSelfImprovementPlanRequestV0,
	requests []orquestaserver.IdleSelfImprovementRequestV0,
) bool {
	trigger := strings.TrimSpace(request.Trigger)
	if trigger == "backlog_scan" || trigger == "manual_backlog_scan" || trigger == "scanner" {
		return true
	}
	if idleSelfImprovementHasExecutableBacklogRequestsV0(requests) {
		return false
	}
	return len(requests) == 0
}

func idleSelfImprovementHasExecutableBacklogRequestsV0(
	requests []orquestaserver.IdleSelfImprovementRequestV0,
) bool {
	for _, request := range requests {
		switch strings.TrimSpace(request.FailureKind) {
		case "backlog_autoprogramming", "backlog_documentation_review":
			return true
		}
	}
	return false
}
