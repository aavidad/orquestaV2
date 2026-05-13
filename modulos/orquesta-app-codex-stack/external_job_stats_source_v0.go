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

type CodexStackExternalJobStatsSourceV0 struct {
	RunStore       orquestacionnucleoapp.RunStorePortV0
	AppChangeStore orquestaappchange.AppChangeRecordSourcePortV0
	ReceiptStore   CodexReceiptStorePortV0
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
	stats.EvidenceRefs = compactCodexStackStringsV0(append(stats.EvidenceRefs, stats.DeliveryRefs...))
	return stats, true, nil
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
