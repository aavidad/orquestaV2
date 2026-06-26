package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaappchangedirectorsource "orquesta/modulos/orquesta-app-change-director-source"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

const (
	codexStackExternalJobStatusIntegrationRequiredV0 = "integration_required"
	codexStackExternalJobStatusParentAckReceivedV0   = "parent_ack_received"

	codexStackExternalJobStatusReasonParentIntegrationPendingV0                   = "parent_integration_pending"
	codexStackExternalJobStatusReasonCohortOpenV0                                 = "cohort_open"
	codexStackExternalJobStatusReasonProductNotConsolidatedDueWriteSetNarrowingV0 = "product_not_consolidated_due_write_set_narrowing"
)

type CodexStackExternalJobStatsSourceV0 struct {
	RunStore       orquestacionnucleoapp.RunStorePortV0
	AppChangeStore orquestaappchange.AppChangeRecordSourcePortV0
	ReceiptStore   CodexReceiptStorePortV0
	TaskStore      orquestacionnucleoapp.WorkflowTaskStorePortV0
}

func (source CodexStackExternalJobStatsSourceV0) ResolveDirectorExternalJobStatsV0(
	ctx context.Context,
	request orquestamcp.MCPDirectorExternalJobStatsRequestV0,
) (orquestamcp.MCPDirectorExternalJobStatsV0, bool, error) {
	jobRef := strings.TrimSpace(request.ExternalJobRef)
	if jobRef == "" || source.AppChangeStore == nil {
		return orquestamcp.MCPDirectorExternalJobStatsV0{}, false, nil
	}
	record, ok, err := source.externalJobRecordV0(ctx, request)
	if err != nil || !ok {
		return orquestamcp.MCPDirectorExternalJobStatsV0{}, ok, err
	}
	return source.externalJobStatsForRecordV0(ctx, record)
}

func (source CodexStackExternalJobStatsSourceV0) externalJobRecordV0(
	ctx context.Context,
	request orquestamcp.MCPDirectorExternalJobStatsRequestV0,
) (orquestaappchange.AppChangeRecordV0, bool, error) {
	records, err := source.AppChangeStore.ListAppChangeRecordsV0(
		ctx,
		orquestaappchange.AppChangeRecordFilterV0{RunRef: strings.TrimSpace(request.RunRef)},
	)
	if err != nil {
		return orquestaappchange.AppChangeRecordV0{}, false, err
	}
	jobRef := strings.TrimSpace(request.ExternalJobRef)
	appRef := strings.TrimSpace(request.AppRef)
	var found []orquestaappchange.AppChangeRecordV0
	for _, record := range records {
		work := record.Request.ExternalWork
		if work == nil ||
			strings.TrimSpace(work.JobRef) != jobRef ||
			(appRef != "" && strings.TrimSpace(record.Request.AppRef) != appRef) {
			continue
		}
		found = append(found, record)
	}
	if len(found) == 0 {
		return orquestaappchange.AppChangeRecordV0{}, false, nil
	}
	if len(found) > 1 && strings.TrimSpace(request.RunRef) == "" {
		return orquestaappchange.AppChangeRecordV0{}, false, fmt.Errorf("external_job_ref_ambiguous")
	}
	return found[0], true, nil
}

func (source CodexStackExternalJobStatsSourceV0) externalJobStatsForRecordV0(
	ctx context.Context,
	record orquestaappchange.AppChangeRecordV0,
) (orquestamcp.MCPDirectorExternalJobStatsV0, bool, error) {
	runRef := strings.TrimSpace(record.Request.RunRef)
	taskRef := orquestaappchangedirectorsource.AppChangeTaskRefV0(record.Request.ChangeRef)
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	stats := orquestamcp.MCPDirectorExternalJobStatsV0{
		AppRef:    strings.TrimSpace(record.Request.AppRef),
		JobRef:    strings.TrimSpace(record.Request.ExternalWork.JobRef),
		WorkKind:  strings.TrimSpace(record.Request.ExternalWork.WorkKind),
		ChangeRef: strings.TrimSpace(record.Request.ChangeRef),
		RunRef:    runRef,
		TaskRef:   taskRef,
		AgentRef:  agentRef,
		Status:    "registered",
		EvidenceRefs: compactCodexStackStringsV0(append(
			[]string{runRef, taskRef, agentRef, strings.TrimSpace(record.Request.ExternalWork.JobRef)},
			record.Request.CurrentStateRefs...,
		)),
	}
	if source.RunStore == nil {
		return stats, true, nil
	}
	run, err := source.RunStore.LoadRunV0(ctx, runRef)
	if err != nil {
		return stats, true, nil
	}
	stats.Status = externalJobStatusFromRunV0(run, taskRef, agentRef)
	stats.DeliveryRefs = source.externalJobDeliveryRefsV0(ctx, run.RunID, taskRef, run.Deliveries)
	if stats.Status != "completed" && len(stats.DeliveryRefs) > 0 {
		stats.Status = "delivered"
	}
	source.enrichExternalJobTaskDiagnosticsV0(ctx, run, &stats)
	stats.EvidenceRefs = compactCodexStackStringsV0(append(stats.EvidenceRefs, stats.DeliveryRefs...))
	return stats, true, nil
}

func (source CodexStackExternalJobStatsSourceV0) enrichExternalJobTaskDiagnosticsV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
	stats *orquestamcp.MCPDirectorExternalJobStatsV0,
) {
	if stats == nil || source.TaskStore == nil || strings.TrimSpace(stats.TaskRef) == "" {
		return
	}
	tasks, err := source.TaskStore.LoadWorkflowTasksV0(ctx, run.RunID, []string{stats.TaskRef})
	if err != nil || len(tasks) == 0 {
		return
	}
	parent := tasks[0]
	childRefs := compactCodexStackStringsV0(parent.ChildTaskRefs)
	if len(childRefs) == 0 {
		return
	}
	parentResolved := externalJobTaskResolvedV0(run, parent.TaskID)
	childrenResolved := externalJobAllTasksResolvedV0(run, childRefs)
	if parentResolved && !childrenResolved {
		source.markExternalJobParentAckWithOpenCohortV0(run, stats, parent, childRefs)
		return
	}
	if parentResolved || !childrenResolved {
		return
	}
	reason := codexStackExternalJobStatusReasonParentIntegrationPendingV0
	code := "external_job_parent_integration_pending"
	if externalJobParentWriteSetOnlyCoordinationV0(parent.WriteSet) {
		reason = codexStackExternalJobStatusReasonProductNotConsolidatedDueWriteSetNarrowingV0
		code = reason
	}
	stats.Status = codexStackExternalJobStatusIntegrationRequiredV0
	stats.StatusReason = reason
	stats.IssueRefs = compactCodexStackStringsV0(append(
		stats.IssueRefs,
		"issue-ref-"+code+"-"+codexStackOperationalClosureSafeRefV0(parent.TaskID),
		parent.TaskID,
	))
	stats.EvidenceRefs = compactCodexStackStringsV0(append(
		stats.EvidenceRefs,
		parent.TaskID,
	))
	stats.Diagnostics = append(stats.Diagnostics, orquestamcp.MCPDirectorExternalJobDiagnosticV0{
		Code:         code,
		Scope:        strings.TrimSpace(stats.JobRef),
		Message:      "integracion del padre pendiente tras resolver las tareas hijas",
		EvidenceRefs: compactCodexStackStringsV0(append([]string{parent.TaskID}, childRefs...)),
	})
}

func (source CodexStackExternalJobStatsSourceV0) markExternalJobParentAckWithOpenCohortV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	stats *orquestamcp.MCPDirectorExternalJobStatsV0,
	parent orquestacoreworkflow.WorkflowTaskV0,
	childRefs []string,
) {
	stats.Status = codexStackExternalJobStatusParentAckReceivedV0
	stats.StatusReason = codexStackExternalJobStatusReasonCohortOpenV0
	stats.IssueRefs = compactCodexStackStringsV0(append(
		stats.IssueRefs,
		"issue-ref-external-job-parent-ack-received-cohort-open-"+codexStackOperationalClosureSafeRefV0(parent.TaskID),
		parent.TaskID,
	))
	stats.EvidenceRefs = compactCodexStackStringsV0(append(
		stats.EvidenceRefs,
		parent.TaskID,
	))
	stats.Diagnostics = append(stats.Diagnostics, orquestamcp.MCPDirectorExternalJobDiagnosticV0{
		Code:         "external_job_parent_ack_received_cohort_open",
		Scope:        strings.TrimSpace(stats.JobRef),
		Message:      "ack del padre recibido con cohorte de tareas hijas aun abierta",
		EvidenceRefs: compactCodexStackStringsV0(append([]string{run.RunID, parent.TaskID}, childRefs...)),
	})
}

func externalJobAllTasksResolvedV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	taskRefs []string,
) bool {
	if len(taskRefs) == 0 {
		return false
	}
	for _, taskRef := range taskRefs {
		if !externalJobTaskResolvedV0(run, taskRef) {
			return false
		}
	}
	return true
}

func externalJobTaskResolvedV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	taskRef string,
) bool {
	return codexStackStringInSetV0(run.DeliveredTasks, taskRef) ||
		codexStackStringInSetV0(run.ClosedTasks, taskRef)
}

func externalJobParentWriteSetOnlyCoordinationV0(writeSet []string) bool {
	entries := compactCodexStackStringsV0(writeSet)
	if len(entries) == 0 {
		return false
	}
	for _, entry := range entries {
		if !strings.HasSuffix(strings.Trim(strings.TrimSpace(entry), "/"), "/coordinacion") {
			return false
		}
	}
	return true
}

func (source CodexStackExternalJobStatsSourceV0) externalJobDeliveryRefsV0(
	ctx context.Context,
	runRef string,
	taskRef string,
	runDeliveryRefs []string,
) []string {
	if source.ReceiptStore == nil {
		return []string{}
	}
	descriptors, err := source.ReceiptStore.ListCodexReceiptDescriptorsV0(
		ctx,
		orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0{RunID: runRef},
	)
	if err != nil {
		return []string{}
	}
	delivered := map[string]struct{}{}
	for _, ref := range compactCodexStackStringsV0(runDeliveryRefs) {
		delivered[ref] = struct{}{}
	}
	out := make([]string, 0, len(descriptors))
	for _, descriptor := range descriptors {
		ackRef := strings.TrimSpace(descriptor.Spec.AgentPacket.DeliveryRefs.AckRef)
		if strings.TrimSpace(descriptor.Spec.AgentPacket.Task.TaskRef) != strings.TrimSpace(taskRef) ||
			ackRef == "" {
			continue
		}
		if _, ok := delivered[ackRef]; ok {
			out = append(out, ackRef)
		}
	}
	return compactCodexStackStringsV0(out)
}

func externalJobStatusFromRunV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	taskRef string,
	agentRef string,
) string {
	switch {
	case codexStackStringInSetV0(run.ClosedTasks, taskRef):
		return "completed"
	case codexStackStringInSetV0(run.DeliveredTasks, taskRef):
		return "delivered"
	case codexStackStringInSetV0(run.FailedAgents, agentRef):
		return "failed"
	case codexStackStringInSetV0(run.ConfirmedStoppedAgents, agentRef):
		return "stopped"
	case codexStackStringInSetV0(run.StoppedAgents, agentRef):
		return "stop_requested"
	case codexStackStringInSetV0(run.StartedAgents, agentRef):
		return "running"
	case codexStackStringInSetV0(run.Agents, agentRef):
		return "requested"
	case codexStackStringInSetV0(run.Tasks, taskRef):
		return "pending"
	default:
		return "registered"
	}
}

func codexStackStringInSetV0(values []string, want string) bool {
	want = strings.TrimSpace(want)
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}
