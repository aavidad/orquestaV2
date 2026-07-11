package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
	orquestaserver "orquesta/modulos/orquesta-server"
)

type serverStackSupervisorV0 struct {
	stack                   *orquestaappcodexstack.StackV0
	projectWorkDir          string
	runtimeWorkDir          string
	stateDir                string
	selfAuditBacklogEnabled bool
	curatedSkills           []orquestaautoprogramming.AutoprogrammingCuratedSkillV0
	operatorNotifier        operatorTaskTerminalNotifierV0
}

func (supervisor serverStackSupervisorV0) RunGlobalSupervisorV0(
	ctx context.Context,
	command orquestarunsupervisor.RunSupervisorCommandV0,
) (orquestarunsupervisor.RunSupervisorResultV0, error) {
	if supervisor.stack == nil {
		return orquestarunsupervisor.RunSupervisorResultV0{}, fmt.Errorf("stack requerido")
	}
	return supervisor.stack.RunGlobalSupervisorV0(ctx, command)
}

func (supervisor serverStackSupervisorV0) PrepareIdleSelfImprovementV0(
	ctx context.Context,
	request orquestaserver.IdleSelfImprovementRequestV0,
) (orquestaserver.IdleSelfImprovementResultV0, error) {
	if supervisor.stack == nil {
		return orquestaserver.IdleSelfImprovementResultV0{}, fmt.Errorf("stack requerido")
	}
	prepareStack := supervisor.autoprogrammingPrepareStackV0()
	prepare := orquestaappcodexstack.NewCodexStackAutoprogrammingPrepareRunExecutorV0(
		prepareStack,
		request.OccurredAt,
		firstNonEmptyServerStackV0(request.RequestedBy, "orquesta-server"),
		prepareStack.Stores.RunQueue,
		prepareStack.RunQueue,
		prepareStack.Clock,
		firstNonEmptyServerStackV0(supervisor.runtimeWorkDir, prepareStack.CodexRuntimeWorkDir),
	)
	self := orquestamcp.NewMCPAutoprogrammingSelfImprovementToolExecutorV0(prepare)
	out, err := self.Execute(ctx, orquestamcp.MCPAutoprogrammingSelfImprovementToolInputV0{
		RequestID:      request.RequestRef,
		CorrelationID:  request.CorrelationID,
		AutoPrepareRun: true,
		Proposal:       idleSelfImprovementProposalFromServerV0(request),
	})
	if err != nil {
		return orquestaserver.IdleSelfImprovementResultV0{}, err
	}
	return supervisor.ensureIdleSelfImprovementQueueVisibleV0(ctx, idleSelfImprovementResultToServerV0(out)), nil
}

func (supervisor serverStackSupervisorV0) autoprogrammingPrepareStackV0() *orquestaappcodexstack.StackV0 {
	if supervisor.stack == nil {
		return nil
	}
	stack := *supervisor.stack
	if projectWorkDir := strings.TrimSpace(supervisor.projectWorkDir); projectWorkDir != "" {
		stack.Codex.ProjectWorkDir = projectWorkDir
	}
	return &stack
}

func idleSelfImprovementProposalFromServerV0(
	request orquestaserver.IdleSelfImprovementRequestV0,
) orquestaautoprogramming.AutoprogrammingSelfImprovementProposalV0 {
	return orquestaautoprogramming.AutoprogrammingSelfImprovementProposalV0{
		RequestRef:         request.RequestRef,
		ProjectRef:         request.ProjectRef,
		WorktreeRef:        request.WorktreeRef,
		WorktreeIsolated:   true,
		BranchRef:          request.BranchRef,
		ObservedBy:         firstNonEmptyServerStackV0(request.RequestedBy, request.Source),
		FailureKind:        request.FailureKind,
		FailureSummary:     request.FailureSummary,
		SuggestedArea:      request.SuggestedArea,
		SuggestedWriteSet:  append([]string(nil), request.WriteSet...),
		RequiredTests:      append([]string(nil), request.RequiredTests...),
		AcceptanceCriteria: append([]string(nil), request.AcceptanceCriteria...),
		AcceptanceChecks:   append([]orquestaautoprogramming.AutoprogrammingAcceptanceCheckV0(nil), request.AcceptanceChecks...),
		CompactRules:       append([]string(nil), request.CompactRules...),
		ContextRefs:        append([]string(nil), request.ContextRefs...),
		EvidenceRefs:       append([]string(nil), request.EvidenceRefs...),
		BacklogScan:        idleSelfImprovementBacklogScanToAutoprogrammingV0(request),
		PriorityScore:      request.PriorityScore,
	}
}

func idleSelfImprovementBacklogScanToAutoprogrammingV0(
	request orquestaserver.IdleSelfImprovementRequestV0,
) orquestaautoprogramming.AutoprogrammingBacklogScanV0 {
	docs := make([]orquestaautoprogramming.AutoprogrammingBacklogDocumentV0, 0, len(request.BacklogScanDocs))
	for _, doc := range request.BacklogScanDocs {
		docs = append(docs, orquestaautoprogramming.AutoprogrammingBacklogDocumentV0{
			Path:       doc.Path,
			StartLine:  doc.StartLine,
			SHA256:     doc.SHA256,
			Missing:    doc.Missing,
			SectionRef: doc.SectionRef,
		})
	}
	return orquestaautoprogramming.AutoprogrammingBacklogScanV0{
		Epoch:           request.BacklogScanEpoch,
		Documents:       docs,
		ReservationRefs: append([]string(nil), request.ReservationRefs...),
	}
}

func idleSelfImprovementResultToServerV0(
	out orquestamcp.MCPAutoprogrammingSelfImprovementToolResultV0,
) orquestaserver.IdleSelfImprovementResultV0 {
	result := orquestaserver.IdleSelfImprovementResultV0{
		Accepted:    out.Accepted,
		RequestRef:  out.RequestID,
		Status:      out.Estado,
		NextActions: append([]string(nil), out.NextActions...),
	}
	if out.PreparedRun != nil {
		result.Accepted = result.Accepted && out.PreparedRun.Accepted
		result.RunRef = out.PreparedRun.RunRef
		result.Status = out.PreparedRun.Estado
		if out.PreparedRun.Accepted && strings.TrimSpace(out.PreparedRun.RunRef) == "" {
			result.Accepted = false
			result.Message = "idle_self_improvement_prepare_run_missing_run_ref"
		}
		if !out.PreparedRun.Accepted {
			result.Message = firstNonEmptyServerStackV0(
				result.Message,
				firstPrepareRunIssueMessageServerStackV0(out.PreparedRun.Errores),
				out.PreparedRun.Estado,
			)
		}
	} else if result.Accepted {
		result.Accepted = false
		result.Status = "error"
		result.Message = "idle_self_improvement_prepare_run_missing_result"
	}
	if !result.Accepted && len(out.Errores) > 0 {
		result.Message = firstNonEmptyServerStackV0(result.Message, out.Errores[0].Message)
	}
	return result
}

func (supervisor serverStackSupervisorV0) ensureIdleSelfImprovementQueueVisibleV0(
	ctx context.Context,
	result orquestaserver.IdleSelfImprovementResultV0,
) orquestaserver.IdleSelfImprovementResultV0 {
	if !result.Accepted {
		return result
	}
	runRef := strings.TrimSpace(result.RunRef)
	queueRef := ""
	if supervisor.stack != nil {
		queueRef = strings.TrimSpace(supervisor.stack.RunQueue.QueueRef)
		if queueRef == "" {
			queueRef = orquestaappcodexstack.DefaultRunQueueRefV0
		}
	}
	if runRef == "" || supervisor.stack == nil || supervisor.stack.Stores.RunQueue == nil {
		result.Accepted = false
		result.Message = "idle_self_improvement_queue_visibility_unavailable"
		return result
	}
	candidates, err := supervisor.stack.Stores.RunQueue.ListRunSchedulingCandidatesV0(
		ctx,
		orquestarunqueue.RunQueueReadRequestV0{QueueRef: queueRef, IncludeNonExecutable: true},
	)
	if err != nil {
		result.Accepted = false
		result.Message = "idle_self_improvement_queue_read_failed"
		return result
	}
	status, ok := idleSelfImprovementQueueCandidateStatusV0(candidates, runRef)
	if !ok {
		result.Message = firstNonEmptyServerStackV0(
			result.Message,
			"idle_self_improvement_queue_candidate_not_visible_advisory",
		)
		result.NextActions = compactServerStackStringsV0(append(result.NextActions,
			"queue_candidate_not_visible_after_prepare",
			"supervisor_will_recheck_run_store",
		))
		result.EvidenceRefs = compactServerStackStringsV0(append(result.EvidenceRefs,
			"evidence-ref-idle-self-improvement-queue-not-visible-advisory",
		))
		if supervisor.idleSelfImprovementPreparedRunPersistedV0(ctx, runRef) {
			result.NextActions = compactServerStackStringsV0(append(result.NextActions,
				"prepared_run_persisted_not_scheduling_visible",
			))
			result.EvidenceRefs = compactServerStackStringsV0(append(result.EvidenceRefs,
				"evidence-ref-idle-self-improvement-run-persisted",
			))
		}
		return result
	}
	runStatus, runOK := supervisor.idleSelfImprovementPreparedRunStatusV0(ctx, runRef)
	if runOK && runStatus == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		if err := supervisor.markIdleSelfImprovementQueueCandidateClosedV0(ctx, queueRef, candidates, runRef); err != nil {
			result.Accepted = false
			result.Message = "idle_self_improvement_queue_candidate_close_sync_failed"
			return result
		}
		result.Message = firstNonEmptyServerStackV0(
			result.Message,
			"idle_self_improvement_queue_candidate_already_closed_advisory",
		)
		result.NextActions = compactServerStackStringsV0(append(result.NextActions,
			"queue_candidate_closed_after_prepare",
			"planner_should_skip_closed_backlog_ref",
		))
		result.EvidenceRefs = compactServerStackStringsV0(append(result.EvidenceRefs,
			"evidence-ref-idle-self-improvement-queue-candidate-already-closed",
			"evidence-ref-idle-self-improvement-queue-candidate-closed-synced",
		))
		return result
	}
	if !orquestarunqueue.IsExecutableRunStatusV0(status) {
		if controlStatus, controlBlocked := supervisor.idleSelfImprovementPreparedRunControlBlocksSchedulingV0(ctx, runRef); controlBlocked {
			result.Accepted = false
			result.Message = "idle_self_improvement_queue_candidate_control_blocked"
			result.NextActions = compactServerStackStringsV0(append(result.NextActions,
				"queue_candidate_previous_status="+status,
				"run_control_status="+string(controlStatus),
				"planner_should_skip_control_blocked_backlog_ref",
			))
			result.EvidenceRefs = compactServerStackStringsV0(append(result.EvidenceRefs,
				"evidence-ref-idle-self-improvement-control-blocked",
			))
			return result
		}
		if disposition, goalFirst := supervisor.stack.GoalFirstQueueDispositionV0(ctx, runRef); goalFirst {
			if queueStatus := strings.TrimSpace(disposition.QueueStatus); queueStatus != "" && queueStatus != status {
				if _, err := supervisor.stack.Stores.RunQueue.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
					RunRef:         runRef,
					QueueRef:       queueRef,
					Status:         queueStatus,
					PriorityScore:  idleSelfImprovementCandidatePriorityScoreV0(candidates, runRef),
					UpdatedAt:      time.Now().UTC(),
					RequestedBy:    "orquesta-server-idle-self-improvement",
					Reason:         "prepared_goal_first_non_executable_synced",
					IdempotencyKey: "idem-idle-self-improvement-goal-first-" + runRef,
					EvidenceRefs: compactServerStackStringsV0(append(
						append([]string(nil), disposition.EvidenceRefs...),
						"evidence-ref-idle-self-improvement-goal-first-not-requeued",
					)),
				}); err != nil {
					result.Accepted = false
					result.Message = "idle_self_improvement_goal_first_queue_sync_failed"
					return result
				}
			}
			result.Message = firstNonEmptyServerStackV0(
				result.Message,
				"idle_self_improvement_goal_first_observe_required",
			)
			nextGoalAction := "observe_autoprogramming_goal"
			if strings.TrimSpace(disposition.Outcome) == "goal_first_state_missing" {
				nextGoalAction = "repair_goal_state_from_launcher_receipt_or_mark_blocked"
				result.Message = "idle_self_improvement_goal_first_state_missing"
			}
			result.NextActions = compactServerStackStringsV0(append(result.NextActions,
				nextGoalAction,
				"do_not_requeue_goal_first_to_legacy",
				"goal_first_outcome="+strings.TrimSpace(disposition.Outcome),
			))
			result.EvidenceRefs = compactServerStackStringsV0(append(
				append(result.EvidenceRefs, disposition.EvidenceRefs...),
				"evidence-ref-idle-self-improvement-goal-first-not-requeued",
			))
			return result
		}
		if runOK && runStatus == orquestacoreworkflow.OrchestrationRunStatusActiveV0 &&
			idleSelfImprovementCanRequeueNonExecutableStatusV0(status) {
			if _, err := supervisor.stack.Stores.RunQueue.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
				RunRef:         runRef,
				QueueRef:       queueRef,
				Status:         orquestarunqueue.RunStatusReadyV0,
				PriorityScore:  idleSelfImprovementCandidatePriorityScoreV0(candidates, runRef),
				UpdatedAt:      time.Now().UTC(),
				RequestedBy:    "orquesta-server-idle-self-improvement",
				Reason:         "prepared_active_non_executable_run_requeued",
				IdempotencyKey: "idem-idle-self-improvement-requeue-" + runRef,
				EvidenceRefs: []string{
					"evidence-ref-idle-self-improvement-active-non-executable-requeued",
				},
			}); err != nil {
				result.Accepted = false
				result.Message = "idle_self_improvement_queue_candidate_requeue_failed"
				return result
			}
			result.NextActions = compactServerStackStringsV0(append(result.NextActions,
				"queue_candidate_requeued_ready_after_prepare",
				"queue_candidate_previous_status="+status,
			))
			result.EvidenceRefs = compactServerStackStringsV0(append(result.EvidenceRefs,
				"evidence-ref-idle-self-improvement-active-non-executable-requeued",
			))
			return result
		}
		result.Accepted = false
		result.Message = "idle_self_improvement_queue_candidate_not_executable"
		return result
	}
	result.NextActions = compactServerStackStringsV0(append(result.NextActions,
		"queue_ref="+queueRef,
		"queue_candidate_run_ref="+runRef,
		"queue_candidate_status="+status,
	))
	result.EvidenceRefs = compactServerStackStringsV0(append(result.EvidenceRefs,
		"evidence-ref-idle-self-improvement-queue-visible",
	))
	return result
}

func (supervisor serverStackSupervisorV0) idleSelfImprovementPreparedRunControlBlocksSchedulingV0(
	ctx context.Context,
	runRef string,
) (orquestaruncontrol.RunControlStatusV0, bool) {
	state, ok := supervisor.idleSelfImprovementPreparedRunControlStateV0(ctx, runRef)
	if !ok {
		return "", false
	}
	evaluation := orquestaruncontrol.EvaluateRunControlV0(state)
	if evaluation.SchedulingAllowed {
		return state.Status, false
	}
	switch orquestaruncontrol.NormalizeRunControlStatusV0(state.Status) {
	case orquestaruncontrol.RunControlStatusPausedV0,
		orquestaruncontrol.RunControlStatusStopRequestedV0,
		orquestaruncontrol.RunControlStatusStoppedV0,
		orquestaruncontrol.RunControlStatusCancelRequestedV0,
		orquestaruncontrol.RunControlStatusCanceledV0:
		return orquestaruncontrol.NormalizeRunControlStatusV0(state.Status), true
	default:
		return state.Status, false
	}
}
