package orquestaautoprogramming

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

const (
	autoprogrammingWorkflowTaskPayloadSoftLimitV0 = 3800
	autoprogrammingWorkflowTaskContextRefsMaxV0   = 18
)

func EnsureAutoprogrammingWorkflowTaskAcceptedByCoreV0(
	task orquestacoreworkflow.WorkflowTaskV0,
) (orquestacoreworkflow.WorkflowTaskV0, AutoprogrammingRequestIssueV0) {
	repaired := compactAutoprogrammingWorkflowTaskForPayloadV0(task)
	if err := orquestacoreworkflow.ValidateWorkflowTaskV0(repaired); err == nil {
		return repaired, AutoprogrammingRequestIssueV0{}
	} else {
		repaired = repairAutoprogrammingWorkflowTaskForCoreErrorV0(repaired, err)
	}
	repaired = compactAutoprogrammingWorkflowTaskForPayloadV0(repaired)
	if err := orquestacoreworkflow.ValidateWorkflowTaskV0(repaired); err != nil {
		return task, autoprogrammingWorkflowTaskIssueV0(err)
	}
	return repaired, AutoprogrammingRequestIssueV0{}
}

func compactAutoprogrammingWorkflowTaskForPayloadV0(
	task orquestacoreworkflow.WorkflowTaskV0,
) orquestacoreworkflow.WorkflowTaskV0 {
	task.Summary = compactAutoprogrammingWorkflowTextV0(
		task.Summary,
		autoprogrammingWorkflowSummaryLimitV0,
	)
	task.AcceptanceCriteria = compactAutoprogrammingWorkflowTextsLimitedV0(
		task.AcceptanceCriteria,
		autoprogrammingWorkflowCriteriaLimitV0,
		autoprogrammingWorkflowCriteriaMaxV0,
	)
	task.ContextRefs = compactAutoprogrammingWorkflowContextRefsForPayloadV0(task.ContextRefs)
	if autoprogrammingWorkflowTaskPayloadLenV0(task) <= autoprogrammingWorkflowTaskPayloadSoftLimitV0 {
		return task
	}
	task.AcceptanceCriteria = compactAutoprogrammingWorkflowTextsLimitedV0(
		task.AcceptanceCriteria,
		80,
		8,
	)
	task.ContextRefs = compactAutoprogrammingWorkflowContextRefsLimitedV0(task.ContextRefs, 12)
	if autoprogrammingWorkflowTaskPayloadLenV0(task) <= autoprogrammingWorkflowTaskPayloadSoftLimitV0 {
		return task
	}
	task.AcceptanceCriteria = compactAutoprogrammingWorkflowTextsLimitedV0(
		task.AcceptanceCriteria,
		70,
		5,
	)
	task.ContextRefs = compactAutoprogrammingWorkflowContextRefsLimitedV0(task.ContextRefs, 8)
	return task
}

func repairAutoprogrammingWorkflowTaskForCoreErrorV0(
	task orquestacoreworkflow.WorkflowTaskV0,
	err error,
) orquestacoreworkflow.WorkflowTaskV0 {
	var taskErr orquestacoreworkflow.WorkflowTaskErrorV0
	if !errors.As(err, &taskErr) {
		return task
	}
	switch taskErr.Code {
	case orquestacoreworkflow.ErrWorkflowTaskPayloadInvalidoV0:
		return compactAutoprogrammingWorkflowTaskAggressivelyV0(task)
	case orquestacoreworkflow.ErrDetalleProhibidoV0:
		return repairAutoprogrammingWorkflowTaskForbiddenDetailV0(task, taskErr.Field)
	default:
		return task
	}
}

func compactAutoprogrammingWorkflowTaskAggressivelyV0(
	task orquestacoreworkflow.WorkflowTaskV0,
) orquestacoreworkflow.WorkflowTaskV0 {
	task.Summary = compactAutoprogrammingWorkflowTextV0(task.Summary, 100)
	task.AcceptanceCriteria = compactAutoprogrammingWorkflowTextsLimitedV0(
		task.AcceptanceCriteria,
		70,
		5,
	)
	task.ContextRefs = compactAutoprogrammingWorkflowContextRefsLimitedV0(task.ContextRefs, 8)
	return task
}

func repairAutoprogrammingWorkflowTaskForbiddenDetailV0(
	task orquestacoreworkflow.WorkflowTaskV0,
	field string,
) orquestacoreworkflow.WorkflowTaskV0 {
	switch strings.TrimSpace(field) {
	case "title":
		task.Title = "Autoprogramacion validada"
	case "summary":
		task.Summary = "summary_ref:" + autoprogrammingWorkflowListHashV0([]string{task.Summary})
	case "acceptance_criteria":
		task.AcceptanceCriteria = compactAutoprogrammingWorkflowRefsV0("criterion_ref", task.AcceptanceCriteria)
	case "required_tests":
		task.RequiredTests = compactAutoprogrammingWorkflowRefsV0("required_test_ref", task.RequiredTests)
	case "context_refs":
		task.ContextRefs = compactAutoprogrammingWorkflowRefsV0("context_ref", task.ContextRefs)
	case "write_set":
		task.WriteSet = repairAutoprogrammingWorkflowWriteSetRefsV0(task.WriteSet)
	default:
		task.Title = "Autoprogramacion validada"
		task.Summary = "workflow_task_repaired_ref:" + autoprogrammingWorkflowListHashV0(
			append([]string{task.Summary}, task.AcceptanceCriteria...),
		)
		task.AcceptanceCriteria = compactAutoprogrammingWorkflowRefsV0("criterion_ref", task.AcceptanceCriteria)
		task.ContextRefs = compactAutoprogrammingWorkflowRefsV0("context_ref", task.ContextRefs)
	}
	return task
}

func compactAutoprogrammingWorkflowRefsV0(prefix string, values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range compactStringsV0(values) {
		out = append(out, prefix+":"+autoprogrammingWorkflowListHashV0([]string{value}))
	}
	return compactStringsV0(out)
}

func repairAutoprogrammingWorkflowWriteSetRefsV0(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range compactStringsV0(values) {
		if parent := autoprogrammingWorkflowWriteSetParentV0(value); parent != "" {
			out = append(out, parent)
			continue
		}
		out = append(out, "autoprogramming-redacted-"+autoprogrammingWorkflowListHashV0([]string{value}))
	}
	return compactStringsV0(out)
}

func autoprogrammingWorkflowWriteSetParentV0(value string) string {
	value = strings.Trim(strings.TrimSpace(value), "/")
	if value == "" || strings.Contains(value, "\\") || strings.HasPrefix(value, ".") {
		return ""
	}
	if before, _, ok := strings.Cut(value, "/"); ok {
		return strings.TrimSpace(before)
	}
	return ""
}

func compactAutoprogrammingWorkflowContextRefsForPayloadV0(values []string) []string {
	return compactAutoprogrammingWorkflowContextRefsLimitedV0(
		autoprogrammingSafeWorkflowContextRefsV0(values),
		autoprogrammingWorkflowTaskContextRefsMaxV0,
	)
}

func compactAutoprogrammingWorkflowContextRefsLimitedV0(values []string, maxItems int) []string {
	values = compactStringsV0(values)
	if maxItems <= 0 || len(values) <= maxItems {
		return values
	}
	keep := make([]string, 0, maxItems+1)
	keepSet := map[string]bool{}
	for _, value := range values {
		if len(keep) >= maxItems {
			break
		}
		if !autoprogrammingWorkflowContextRefPriorityV0(value) {
			continue
		}
		keep = append(keep, value)
		keepSet[value] = true
	}
	for _, value := range values {
		if len(keep) >= maxItems {
			break
		}
		if keepSet[value] {
			continue
		}
		keep = append(keep, value)
	}
	var hidden []string
	for _, value := range values {
		if !keepSet[value] {
			hidden = append(hidden, value)
		}
	}
	keep = append(keep, "context_refs_extra_ref:"+autoprogrammingWorkflowListHashV0(hidden))
	return compactStringsV0(keep)
}

func autoprogrammingWorkflowContextRefPriorityV0(value string) bool {
	for _, prefix := range []string{
		"request_ref:",
		"project_ref:",
		"worktree_ref:",
		"branch_ref:",
		"source_task_ref:",
		"backlog_scan_epoch:",
		"backlog_scan_reservation_ref:",
		"backlog_scan_doc:",
	} {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func autoprogrammingWorkflowTaskPayloadLenV0(task orquestacoreworkflow.WorkflowTaskV0) int {
	data, err := json.Marshal(task)
	if err != nil {
		return autoprogrammingWorkflowTaskPayloadSoftLimitV0 + 1
	}
	return len(data)
}

func autoprogrammingWorkflowListHashV0(values []string) string {
	sum := sha256.Sum256([]byte(strings.Join(compactStringsV0(values), "\n")))
	return fmt.Sprintf("%s-%d", hex.EncodeToString(sum[:])[:12], len(values))
}
