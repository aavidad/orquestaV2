package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaappchangedirectorsource "orquesta/modulos/orquesta-app-change-director-source"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

const (
	codexStackExternalJobStatusIntegrationRequiredV0                = "integration_required"
	codexStackExternalJobStatusGoalCompletePendingClosureV0         = "goal_complete_pending_closure"
	codexStackExternalJobStatusGoalStateMissingV0                   = "goal_state_missing"
	codexStackExternalJobStatusParentAckReceivedV0                  = "parent_ack_received"
	codexStackExternalJobStatusParentRunningWithChildAckCollisionV0 = "parent_running_with_child_ack_collision"

	codexStackExternalJobStatusReasonGoalFirstRunningV0                           = "goal_first_running"
	codexStackExternalJobStatusReasonGoalFirstCompletePendingClosureV0            = "goal_first_complete_pending_closure"
	codexStackExternalJobStatusReasonGoalFirstClosureAcceptedV0                   = "goal_first_closure_accepted"
	codexStackExternalJobStatusReasonGoalFirstClosureBlockedV0                    = "goal_first_closure_blocked"
	codexStackExternalJobStatusReasonGoalFirstClosureMissingDomainReceiptV0       = "goal_first_closure_missing_domain_receipt"
	codexStackExternalJobStatusReasonGoalFirstBlockedV0                           = "goal_first_blocked"
	codexStackExternalJobStatusReasonGoalFirstInvalidV0                           = "goal_first_invalid"
	codexStackExternalJobStatusReasonGoalFirstStateMissingV0                      = "goal_first_state_missing"
	codexStackExternalJobStatusReasonParentIntegrationPendingV0                   = "parent_integration_pending"
	codexStackExternalJobStatusReasonCohortOpenV0                                 = "cohort_open"
	codexStackExternalJobStatusReasonChildAckCollisionV0                          = "child_ack_collision"
	codexStackExternalJobStatusReasonProductNotConsolidatedDueWriteSetNarrowingV0 = "product_not_consolidated_due_write_set_narrowing"
)

type CodexStackExternalJobStatsSourceV0 struct {
	RunStore                orquestacionnucleoapp.RunStorePortV0
	AppChangeStore          orquestaappchange.AppChangeRecordSourcePortV0
	ReceiptStore            CodexReceiptStorePortV0
	TaskStore               orquestacionnucleoapp.WorkflowTaskStorePortV0
	GoalStateStore          orquestagoal.GoalWorkStateStorePortV0
	GoalFirstRunMarkerStore orquestagoal.GoalWorkRunMarkerStorePortV0
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
		if source.applyExternalJobGoalFirstStatsV0(ctx, &stats) {
			return stats, true, nil
		}
		if source.applyExternalJobGoalFirstMarkerMissingStateV0(ctx, &stats) {
			return stats, true, nil
		}
		return stats, true, nil
	}
	run, err := source.RunStore.LoadRunV0(ctx, runRef)
	if err != nil {
		if source.applyExternalJobGoalFirstStatsV0(ctx, &stats) {
			return stats, true, nil
		}
		if source.applyExternalJobGoalFirstMarkerMissingStateV0(ctx, &stats) {
			return stats, true, nil
		}
		return stats, true, nil
	}
	if source.applyExternalJobGoalFirstStatsV0(ctx, &stats) {
		return stats, true, nil
	}
	if source.applyExternalJobGoalFirstMarkerMissingStateV0(ctx, &stats) {
		return stats, true, nil
	}
	if codexStackRunSupervisorGoalFirstContainerWithoutStateV0(run) {
		source.markExternalJobGoalFirstStateMissingV0(
			run.RunID,
			[]string{"evidence-ref-external-job-goal-first-container", run.AppSpecRef},
			&stats,
		)
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

func (source CodexStackExternalJobStatsSourceV0) applyExternalJobGoalFirstStatsV0(
	ctx context.Context,
	stats *orquestamcp.MCPDirectorExternalJobStatsV0,
) bool {
	if stats == nil || source.GoalStateStore == nil || strings.TrimSpace(stats.RunRef) == "" {
		return false
	}
	state, err := source.GoalStateStore.LoadGoalWorkStateV0(ctx, strings.TrimSpace(stats.RunRef))
	if err != nil {
		return false
	}
	state, err = orquestagoal.NewGoalWorkStateV0(state)
	if err != nil {
		return false
	}
	source.markExternalJobGoalFirstV0(stats, state)
	return true
}

func (source CodexStackExternalJobStatsSourceV0) applyExternalJobGoalFirstMarkerMissingStateV0(
	ctx context.Context,
	stats *orquestamcp.MCPDirectorExternalJobStatsV0,
) bool {
	if stats == nil || source.GoalFirstRunMarkerStore == nil || strings.TrimSpace(stats.RunRef) == "" {
		return false
	}
	runRef := strings.TrimSpace(stats.RunRef)
	marker, err := source.GoalFirstRunMarkerStore.LoadGoalWorkRunMarkerV0(ctx, runRef)
	if err != nil {
		return false
	}
	marker, err = orquestagoal.NewGoalWorkRunMarkerV0(marker)
	if err != nil || marker.RunRef != runRef {
		return false
	}
	source.markExternalJobGoalFirstStateMissingV0(
		marker.RunRef,
		compactCodexStackStringsV0(append(
			[]string{"evidence-ref-external-job-goal-first-marker"},
			marker.EvidenceRefs...,
		)),
		stats,
	)
	return true
}

func (source CodexStackExternalJobStatsSourceV0) markExternalJobGoalFirstV0(
	stats *orquestamcp.MCPDirectorExternalJobStatsV0,
	state orquestagoal.GoalWorkStateV0,
) {
	stats.TaskRef = ""
	stats.AgentRef = ""
	stats.DirectorExecutionMode = "goal_first"
	stats.GoalRef = strings.TrimSpace(state.GoalRef)
	stats.ExternalGoalRef = strings.TrimSpace(state.ExternalGoalRef)
	stats.GoalStatus = strings.TrimSpace(state.Status)
	stats.Status, stats.StatusReason = externalJobGoalFirstStatusV0(state)
	metadata := orquestamcp.GoalWorkStateDomainOperationalMetadataV0(state)
	stats.CurrentPhase = strings.TrimSpace(metadata.CurrentPhase)
	stats.RetryFromPhase = strings.TrimSpace(metadata.RetryFromPhase)
	stats.OperationalReason = strings.TrimSpace(metadata.OperationalReason)
	stats.DomainCounters = copyCodexStackStringIntMapV0(metadata.DomainCounters)
	stats.EvidenceRefs = compactCodexStackStringsV0(append(
		stats.EvidenceRefs,
		externalJobGoalFirstEvidenceRefsV0(state)...,
	))
	if state.LastResult != nil {
		stats.DeliveryRefs = compactCodexStackStringsV0(append(
			stats.DeliveryRefs,
			state.LastResult.DomainReceiptRefs...,
		))
	}
	if state.LastClosure != nil {
		stats.ClosureStatus = strings.TrimSpace(state.LastClosure.Status)
		stats.ClosureAccepted = state.LastClosure.Accepted
		stats.ClosureNeedsRework = state.LastClosure.NeedsRework
	}
	if stats.Status == "blocked" || stats.Status == codexStackExternalJobStatusGoalCompletePendingClosureV0 {
		stats.IssueRefs = compactCodexStackStringsV0(append(
			stats.IssueRefs,
			"issue-ref-external-job-goal-first-"+codexStackOperationalClosureSafeRefV0(state.RunRef),
		))
	}
	stats.Diagnostics = append(stats.Diagnostics, orquestamcp.MCPDirectorExternalJobDiagnosticV0{
		Code:         stats.StatusReason,
		Scope:        strings.TrimSpace(stats.JobRef),
		Message:      externalJobGoalFirstDiagnosticMessageV0(stats.Status, stats.StatusReason),
		EvidenceRefs: compactCodexStackStringsV0([]string{state.RunRef, state.GoalRef}),
	})
}

func copyCodexStackStringIntMapV0(values map[string]int) map[string]int {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]int, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" || value < 0 {
			continue
		}
		out[key] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (source CodexStackExternalJobStatsSourceV0) markExternalJobGoalFirstStateMissingV0(
	runRef string,
	evidenceRefs []string,
	stats *orquestamcp.MCPDirectorExternalJobStatsV0,
) {
	if stats == nil {
		return
	}
	stats.TaskRef = ""
	stats.AgentRef = ""
	stats.DirectorExecutionMode = "goal_first"
	stats.Status = codexStackExternalJobStatusGoalStateMissingV0
	stats.StatusReason = codexStackExternalJobStatusReasonGoalFirstStateMissingV0
	stats.IssueRefs = compactCodexStackStringsV0(append(
		stats.IssueRefs,
		"issue-ref-external-job-goal-state-missing-"+codexStackOperationalClosureSafeRefV0(runRef),
	))
	stats.EvidenceRefs = compactCodexStackStringsV0(append(
		stats.EvidenceRefs,
		compactCodexStackStringsV0(append(
			[]string{
				runRef,
				"evidence-ref-external-job-goal-state-missing",
			},
			evidenceRefs...,
		))...,
	))
	stats.Diagnostics = append(stats.Diagnostics, orquestamcp.MCPDirectorExternalJobDiagnosticV0{
		Code:         codexStackExternalJobStatusReasonGoalFirstStateMissingV0,
		Scope:        strings.TrimSpace(stats.JobRef),
		Message:      "run external-work goal-first sin GoalWorkStateV0 observable; no se proyecta como tarea legacy registrada",
		EvidenceRefs: compactCodexStackStringsV0(append([]string{runRef}, evidenceRefs...)),
	})
}

func externalJobGoalFirstStatusV0(
	state orquestagoal.GoalWorkStateV0,
) (string, string) {
	if state.LastClosure != nil {
		if state.LastClosure.NeedsRework ||
			len(state.LastClosure.Issues) > 0 ||
			strings.TrimSpace(state.LastClosure.Status) == orquestagoal.GoalStatusBlockedV0 {
			return "blocked", codexStackExternalJobStatusReasonGoalFirstClosureBlockedV0
		}
		if state.LastClosure.Accepted {
			if state.LastResult == nil || len(compactCodexStackStringsV0(state.LastResult.DomainReceiptRefs)) == 0 {
				return "blocked", codexStackExternalJobStatusReasonGoalFirstClosureMissingDomainReceiptV0
			}
			return "completed", codexStackExternalJobStatusReasonGoalFirstClosureAcceptedV0
		}
	}
	switch strings.TrimSpace(state.Status) {
	case orquestagoal.GoalStatusRunningV0, orquestagoal.GoalStatusAcceptedV0:
		return "running", codexStackExternalJobStatusReasonGoalFirstRunningV0
	case orquestagoal.GoalStatusCompleteV0:
		return codexStackExternalJobStatusGoalCompletePendingClosureV0, codexStackExternalJobStatusReasonGoalFirstCompletePendingClosureV0
	case orquestagoal.GoalStatusBlockedV0:
		return "blocked", codexStackExternalJobStatusReasonGoalFirstBlockedV0
	case orquestagoal.GoalStatusInvalidV0:
		return "blocked", codexStackExternalJobStatusReasonGoalFirstInvalidV0
	default:
		return "registered", "goal_first_state_loaded"
	}
}

func externalJobGoalFirstEvidenceRefsV0(
	state orquestagoal.GoalWorkStateV0,
) []string {
	out := []string{state.RunRef, state.GoalRef, state.ExternalGoalRef}
	out = append(out, state.EvidenceRefs...)
	out = append(out, state.LaunchReceipt.EvidenceRefs...)
	if state.LastResult != nil {
		out = append(out, state.LastResult.EvidenceRefs...)
		out = append(out, state.LastResult.ArtifactRefs...)
		out = append(out, state.LastResult.DomainReceiptRefs...)
	}
	if state.LastClosure != nil {
		out = append(out, state.LastClosure.EvidenceRefs...)
	}
	return compactCodexStackStringsV0(out)
}

func externalJobGoalFirstDiagnosticMessageV0(
	status string,
	reason string,
) string {
	switch reason {
	case codexStackExternalJobStatusReasonGoalFirstRunningV0:
		return "external job gobernado por GoalWorkStateV0; el loop legacy no aplica"
	case codexStackExternalJobStatusReasonGoalFirstClosureAcceptedV0:
		return "external job cerrado por goal-first con cierre aceptado"
	case codexStackExternalJobStatusReasonGoalFirstClosureMissingDomainReceiptV0:
		return "external job goal-first aceptado sin recibo de dominio; reparar receipt terminal antes de settled/completed"
	case codexStackExternalJobStatusReasonGoalFirstClosureBlockedV0,
		codexStackExternalJobStatusReasonGoalFirstBlockedV0,
		codexStackExternalJobStatusReasonGoalFirstInvalidV0:
		return "external job goal-first bloqueado; observar goal/rework en vez de supervisor legacy"
	case codexStackExternalJobStatusReasonGoalFirstCompletePendingClosureV0:
		return "goal completo pendiente de cierre validado; observar goal antes de completar el job externo"
	default:
		return "external job goal-first proyectado desde GoalWorkStateV0: " + strings.TrimSpace(status)
	}
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
	if collisionRefs := externalJobActiveParentAckChildCollisionRefsV0(run, stats.DeliveryRefs); len(collisionRefs) > 0 &&
		(!parentResolved || !childrenResolved) {
		source.markExternalJobParentAckChildCollisionV0(run, stats, parent, childRefs, collisionRefs)
		return
	}
	if parentResolved && !childrenResolved {
		source.markExternalJobParentAckWithOpenCohortV0(run, stats, parent, childRefs)
		return
	}
	if childrenResolved && externalJobParentWriteSetOnlyCoordinationV0(parent.WriteSet) {
		source.markExternalJobIntegrationRequiredV0(
			stats,
			parent,
			childRefs,
			codexStackExternalJobStatusReasonProductNotConsolidatedDueWriteSetNarrowingV0,
			codexStackExternalJobStatusReasonProductNotConsolidatedDueWriteSetNarrowingV0,
			"producto canonico pendiente: el padre solo tenia write-set de coordinacion",
		)
		return
	}
	if parentResolved || !childrenResolved {
		return
	}
	source.markExternalJobIntegrationRequiredV0(
		stats,
		parent,
		childRefs,
		codexStackExternalJobStatusReasonParentIntegrationPendingV0,
		"external_job_parent_integration_pending",
		"integracion del padre pendiente tras resolver las tareas hijas",
	)
}

func (source CodexStackExternalJobStatsSourceV0) markExternalJobIntegrationRequiredV0(
	stats *orquestamcp.MCPDirectorExternalJobStatsV0,
	parent orquestacoreworkflow.WorkflowTaskV0,
	childRefs []string,
	reason string,
	code string,
	message string,
) {
	if stats == nil {
		return
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
		Message:      message,
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

func (source CodexStackExternalJobStatsSourceV0) markExternalJobParentAckChildCollisionV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	stats *orquestamcp.MCPDirectorExternalJobStatsV0,
	parent orquestacoreworkflow.WorkflowTaskV0,
	childRefs []string,
	collisionRefs []string,
) {
	if stats == nil {
		return
	}
	stats.Status = codexStackExternalJobStatusParentRunningWithChildAckCollisionV0
	stats.StatusReason = codexStackExternalJobStatusReasonChildAckCollisionV0
	stats.IssueRefs = compactCodexStackStringsV0(append(
		stats.IssueRefs,
		"issue-ref-external-job-parent-child-ack-collision-"+codexStackOperationalClosureSafeRefV0(parent.TaskID),
		parent.TaskID,
	))
	stats.EvidenceRefs = compactCodexStackStringsV0(append(
		append(stats.EvidenceRefs, parent.TaskID),
		collisionRefs...,
	))
	stats.Diagnostics = append(stats.Diagnostics, orquestamcp.MCPDirectorExternalJobDiagnosticV0{
		Code:         codexStackExternalJobStatusParentRunningWithChildAckCollisionV0,
		Scope:        strings.TrimSpace(stats.JobRef),
		Message:      "ack reservado al padre contiene task_ref de hijo o subrol; requiere ack final valido del padre o rework",
		EvidenceRefs: compactCodexStackStringsV0(append(append([]string{run.RunID, parent.TaskID}, childRefs...), collisionRefs...)),
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

func externalJobActiveParentAckChildCollisionRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	deliveryRefs []string,
) []string {
	collisionRefs := externalJobParentAckChildCollisionRefsV0(run)
	if len(collisionRefs) == 0 ||
		externalJobAcceptedReviewForDeliveryRefsV0(run, deliveryRefs) {
		return []string{}
	}
	return collisionRefs
}

func externalJobParentAckChildCollisionRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) []string {
	out := []string{}
	groups := [][]string{
		run.AgentAssessments,
		run.QualityGates,
		run.PhaseArtifacts,
		run.Deliveries,
		run.Reviews,
		run.ReviewResults,
		run.ReworkRequests,
		run.ReplanDecisions,
		run.Blockers,
		run.CommandEffects,
	}
	for _, group := range groups {
		for _, ref := range compactCodexStackStringsV0(group) {
			if externalJobRefContainsAnyParentAckChildCollisionV0(ref) {
				out = append(out, ref)
			}
		}
	}
	return compactCodexStackStringsV0(out)
}

func externalJobAcceptedReviewForDeliveryRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	deliveryRefs []string,
) bool {
	for _, deliveryRef := range compactCodexStackStringsV0(deliveryRefs) {
		if codexStackStringInSetV0(run.AcceptedReviews, "accepted-review-ref-"+deliveryRef) {
			return true
		}
		for _, resultRef := range compactCodexStackStringsV0(run.ReviewResults) {
			if strings.Contains(resultRef, "#review_result:accepted") &&
				strings.Contains(resultRef, "#delivery:"+deliveryRef) {
				return true
			}
		}
	}
	return false
}

func externalJobRefContainsAnyParentAckChildCollisionV0(ref string) bool {
	ref = strings.TrimSpace(ref)
	return strings.Contains(ref, orquestaruntimecodex.CodexAgentAckInvalidParentSubroleCollisionEvidenceV0) ||
		strings.Contains(ref, orquestaruntimecodex.CodexAgentAckInvalidParentChildTaskCollisionEvidenceV0)
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
	case codexStackStringInSetV0(run.LostAgents, agentRef):
		return "lost"
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
