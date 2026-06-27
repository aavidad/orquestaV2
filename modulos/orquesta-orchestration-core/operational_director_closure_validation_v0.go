package orquestacionnucleoapp

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

const (
	maxOperationalDirectorClosureEvidenceRefsV0  = 20
	maxOperationalDirectorClosureEvidenceBytesV0 = 900
)

func operationalDirectorClosedRunRequestIssuesV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	request OperationalDirectorClosureRequestV0,
) []ErrorV0 {
	issues := make([]ErrorV0, 0)
	for _, check := range []struct {
		field  string
		values []string
		want   string
	}{
		{field: "task_id", values: run.ClosedTasks, want: request.TaskID},
		{field: "delivery_ref", values: run.Deliveries, want: request.DeliveryRef},
		{field: "accepted_review_ref", values: run.AcceptedReviews, want: request.AcceptedReviewRef},
		{field: "validation_ref", values: run.Validations, want: request.ValidationRef},
		{field: "closure_ref", values: run.Closures, want: request.ClosureRef},
	} {
		if !operationalDirectorClosureReflectedV0(check.values, check.want) {
			issues = append(issues, errorV0(
				ErrNucleoOrquestacionInvalidoV0,
				check.field,
				"request de cierre no corresponde al run cerrado",
			))
		}
	}
	return issues
}

func normalizeOperationalDirectorClosureRequestV0(
	request OperationalDirectorClosureRequestV0,
) OperationalDirectorClosureRequestV0 {
	request.RunRef = strings.TrimSpace(request.RunRef)
	request.TaskID = strings.TrimSpace(request.TaskID)
	request.DeliveryRef = strings.TrimSpace(request.DeliveryRef)
	request.AcceptedReviewRef = strings.TrimSpace(request.AcceptedReviewRef)
	request.ValidationRef = strings.TrimSpace(request.ValidationRef)
	request.ClosureRef = strings.TrimSpace(request.ClosureRef)
	request.OccurredAt = strings.TrimSpace(request.OccurredAt)
	request.CorrelationID = strings.TrimSpace(request.CorrelationID)
	request.RequestedBy = strings.TrimSpace(request.RequestedBy)
	request.Summary = strings.TrimSpace(request.Summary)
	request.RequiredTestEvidenceRefs = compactStringsV0(request.RequiredTestEvidenceRefs)
	request.EvidenceRefs = compactStringsV0(request.EvidenceRefs)
	return request
}

func (closer OperationalDirectorClosureV0) validateOperationalDirectorClosureV0(
	request OperationalDirectorClosureRequestV0,
) []ErrorV0 {
	issues := make([]ErrorV0, 0)
	if closer.RunStore == nil {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "run_store", "run_store requerido"))
	}
	if closer.EventSink == nil {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "event_sink", "event_sink requerido"))
	}
	if strings.TrimSpace(request.RunRef) == "" {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "run_ref", "run_ref requerido"))
	}
	if strings.TrimSpace(request.TaskID) == "" {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "task_id", "task_id requerido"))
	}
	if strings.TrimSpace(request.DeliveryRef) == "" {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "delivery_ref", "delivery_ref requerido"))
	}
	if strings.TrimSpace(request.AcceptedReviewRef) == "" {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "accepted_review_ref", "accepted_review_ref requerido"))
	}
	if strings.TrimSpace(request.ValidationRef) == "" {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "validation_ref", "validation_ref requerido"))
	}
	if strings.TrimSpace(request.ClosureRef) == "" {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "closure_ref", "closure_ref requerido"))
	}
	if strings.TrimSpace(request.OccurredAt) == "" {
		issues = append(issues, errorV0(ErrNucleoOrquestacionInvalidoV0, "occurred_at", "occurred_at requerido"))
	}
	return issues
}

func operationalDirectorClosureTaskLoadedV0(
	tasks []orquestacoreworkflow.WorkflowTaskV0,
	taskID string,
) bool {
	taskID = strings.TrimSpace(taskID)
	for _, task := range tasks {
		if strings.TrimSpace(task.TaskID) == taskID {
			return true
		}
	}
	return false
}

func operationalDirectorClosureOpenTasksV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) []string {
	open := make([]string, 0, len(run.Tasks))
	closed := compactStringsV0(run.ClosedTasks)
	for _, taskRef := range compactStringsV0(run.Tasks) {
		if !operationalDirectorClosureReflectedV0(closed, taskRef) {
			open = append(open, taskRef)
		}
	}
	return open
}

func operationalDirectorClosureReflectedV0(values []string, want string) bool {
	want = strings.TrimSpace(want)
	if want == "" {
		return false
	}
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}

func operationalDirectorClosureEvidenceRefsV0(
	request OperationalDirectorClosureRequestV0,
) []string {
	refs := compactStringsV0(append(append([]string(nil), request.RequiredTestEvidenceRefs...), request.EvidenceRefs...))
	if operationalDirectorClosureEvidenceRefsFitV0(refs) {
		return refs
	}
	overflow := operationalDirectorClosureEvidenceOverflowRefV0(refs)
	keep := make([]string, 0, maxOperationalDirectorClosureEvidenceRefsV0)
	used := len(overflow)
	for _, ref := range refs {
		if len(keep) >= maxOperationalDirectorClosureEvidenceRefsV0-1 {
			break
		}
		nextUsed := used + len(ref)
		if nextUsed > maxOperationalDirectorClosureEvidenceBytesV0 && len(keep) > 0 {
			break
		}
		keep = append(keep, ref)
		used = nextUsed
	}
	return compactStringsV0(append(keep, overflow))
}

func operationalDirectorClosureEvidenceRefsFitV0(refs []string) bool {
	if len(refs) > maxOperationalDirectorClosureEvidenceRefsV0 {
		return false
	}
	total := 0
	for _, ref := range refs {
		total += len(ref)
	}
	return total <= maxOperationalDirectorClosureEvidenceBytesV0
}

func operationalDirectorClosureEvidenceOverflowRefV0(refs []string) string {
	return "evidence-ref-operational-director-closure-overflow-" +
		deterministicRefDigestPrefixV0("operational_director_closure_overflow", 32, refs...)
}

func operationalDirectorClosureSummaryV0(
	request OperationalDirectorClosureRequestV0,
	fallback string,
) string {
	if request.Summary != "" {
		return request.Summary
	}
	return fallback
}

func operationalDirectorClosureRequestedByV0(
	request OperationalDirectorClosureRequestV0,
) string {
	requestedBy := strings.TrimSpace(request.RequestedBy)
	if requestedBy != "" {
		return requestedBy
	}
	return "orquesta-operational-director-closure"
}

func operationalDirectorClosurePhaseV0(
	phase orquestacoreworkflow.OrchestrationPhaseIDV0,
) orquestacoreworkflow.OrchestrationPhaseIDV0 {
	return orquestacoreworkflow.OrchestrationPhaseIDV0(strings.TrimSpace(string(phase)))
}
