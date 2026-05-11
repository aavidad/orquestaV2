package orquestaruntimecodexdelivery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestadirectorsupervisedburst "orquesta/modulos/orquesta-director-supervised-burst"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
	orquestacionnucleoapp "orquesta/orquestacionnucleoapp"
)

func programmingTeamSmokeStateDiagnosticsV0(
	ctx context.Context,
	failure error,
	store *orquestacionnucleoapp.InMemoryRunStoreV0,
	taskStore *orquestacionnucleoapp.InMemoryWorkflowTaskStoreV0,
	receiptStore *InMemoryCodexReceiptDescriptorStoreV0,
	sink *orquestacionnucleoapp.InMemoryEventSinkV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
	processRegistry orquestacionnucleoapp.AgentProcessRegistryPortV0,
	snapshotSource CodexProcessSnapshotSourcePortV0,
	worktreeStore orquestaruntimeworktree.WorktreeSnapshotStorePortV0,
	runRef string,
	tasks []programmingTeamTaskV0,
) string {
	if ctx == nil || ctx.Err() != nil {
		ctx = context.Background()
	}
	var out strings.Builder
	if failure != nil {
		fmt.Fprintf(&out, "failure=%v kind=%s\n", failure, programmingTeamFailureKindV0(failure))
	}
	run, err := store.LoadRunV0(ctx, runRef)
	if err != nil {
		return "run diagnostics unavailable: " + err.Error()
	}
	fmt.Fprintf(&out, "run phase=%s tasks=%v agents=%v started=%v deliveries=%v phase_artifacts=%v questions=%v assessments=%v\n",
		run.CurrentPhase, run.Tasks, run.Agents, run.StartedAgents, run.Deliveries,
		run.PhaseArtifacts, run.DirectorQuestions, run.AgentAssessments)
	fmt.Fprintf(&out, "expected_programming_agents=%v missing_started=%v missing_deliveries=%v\n",
		programmingTeamProgrammingAgentsV0(tasks),
		programmingTeamMissingRefsV0(programmingTeamProgrammingAgentsV0(tasks), run.StartedAgents),
		programmingTeamMissingDeliveryRefsV0(tasks, run.Deliveries),
	)
	if issues := orquestacoreworkflow.ValidateOrchestrationRunV0(run); len(issues) > 0 {
		fmt.Fprintf(&out, "run_validation=%v\n", issues)
	} else {
		fmt.Fprintf(&out, "run_validation=ok\n")
	}
	storedTasks, taskErr := taskStore.LoadWorkflowTasksV0(ctx, runRef, programmingTeamTaskRefsV0(tasks))
	if taskErr != nil {
		fmt.Fprintf(&out, "task_store_error=%v\n", taskErr)
	} else {
		fmt.Fprintf(&out, "task_store_loaded=%d tasks=%v\n",
			len(storedTasks), programmingTeamStoredTaskSummariesV0(storedTasks))
	}
	programmingTeamAppendOutboxDiagnosticsV0(ctx, &out, ledger, runRef)
	descriptors, descErr := receiptStore.ListCodexReceiptDescriptorsV0(ctx, CodexReceiptDescriptorRequestV0{
		RunID: run.RunID,
	})
	if descErr != nil {
		fmt.Fprintf(&out, "descriptor_error=%v\n", descErr)
	} else {
		fmt.Fprintf(&out, "descriptors=%v\n", programmingTeamDescriptorSummariesV0(descriptors))
	}
	programmingTeamAppendProcessDiagnosticsV0(ctx, &out, processRegistry, run)
	programmingTeamAppendSourceProbeDiagnosticsV0(
		ctx, &out, receiptStore, processRegistry, snapshotSource, worktreeStore, run)
	fmt.Fprintf(&out, "events_by_type=%v events=%v",
		programmingTeamEventCountsV0(sink.EventsV0()), programmingTeamEventTypesV0(sink.EventsV0()))
	return out.String()
}

func programmingTeamTaskRefsV0(tasks []programmingTeamTaskV0) []string {
	refs := make([]string, 0, len(tasks))
	for _, task := range tasks {
		refs = append(refs, task.TaskRef)
	}
	return refs
}

func programmingTeamDescriptorRefsV0(descriptors []CodexReceiptDescriptorV0) []string {
	refs := make([]string, 0, len(descriptors))
	for _, descriptor := range descriptors {
		refs = append(refs, descriptor.AgentRef)
	}
	return refs
}

func programmingTeamDescriptorSummariesV0(descriptors []CodexReceiptDescriptorV0) []string {
	summaries := make([]string, 0, len(descriptors))
	for _, descriptor := range descriptors {
		packet := descriptor.Spec.AgentPacket
		summary := fmt.Sprintf("%s phase=%s task=%s ack_ref=%s ack=%s",
			descriptor.AgentRef,
			packet.Phase,
			packet.Task.TaskRef,
			packet.DeliveryRefs.AckRef,
			programmingTeamAckStatusV0(descriptor),
		)
		if packet.Phase != string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0) {
			summary += " director_decisions=" + programmingTeamDirectorDecisionFileStatusV0(descriptor)
		}
		summaries = append(summaries, summary)
	}
	return summaries
}

func programmingTeamAckStatusV0(descriptor CodexReceiptDescriptorV0) string {
	if strings.TrimSpace(descriptor.AckPath) == "" {
		return "missing_path"
	}
	observation, issues := orquestaruntimecodex.ReadCodexDeliveryObservationFileV0(
		descriptor.AckPath,
		descriptor.Spec,
	)
	if len(issues) == 0 {
		return "ready delivery=" + observation.DeliveryRef
	}
	issue := issues[0]
	status := "invalid"
	if codexReceiptIssueMeansAckNotReadyV0(issue) {
		status = "not_ready"
	}
	return fmt.Sprintf("%s code=%s field=%s evidence=%s",
		status, issue.Code, issue.Field, strings.Join(issue.Evidence, ","))
}

type programmingTeamDecisionFileEnvelopeV0 struct {
	Decisions []programmingTeamDecisionFileItemV0 `json:"decisions"`
}

type programmingTeamDecisionFileItemV0 struct {
	DecisionRef string `json:"decision_ref"`
	CommandType string `json:"command_type"`
	PhaseID     string `json:"phase_id"`
}

func programmingTeamDirectorDecisionFileStatusV0(descriptor CodexReceiptDescriptorV0) string {
	if strings.TrimSpace(descriptor.AckPath) == "" {
		return "missing_ack_path"
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(descriptor.AckPath), DefaultDirectorAgentDecisionFileNameV0))
	if err != nil {
		if os.IsNotExist(err) {
			return "absent"
		}
		return "read_error"
	}
	var envelope programmingTeamDecisionFileEnvelopeV0
	if err := json.Unmarshal(data, &envelope); err != nil {
		return "invalid_json"
	}
	return fmt.Sprintf("count=%d refs=%v", len(envelope.Decisions), programmingTeamDecisionFileRefsV0(envelope.Decisions))
}

func programmingTeamDecisionFileRefsV0(decisions []programmingTeamDecisionFileItemV0) []string {
	refs := make([]string, 0, len(decisions))
	for _, decision := range decisions {
		refs = append(refs, decision.DecisionRef+":"+decision.CommandType+":"+decision.PhaseID)
	}
	return refs
}

func programmingTeamAppendOutboxDiagnosticsV0(
	ctx context.Context,
	out *strings.Builder,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
	runRef string,
) {
	pending, issues := ledger.ListPending(ctx, orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0{
		RunRef: runRef,
	})
	if len(issues) > 0 {
		fmt.Fprintf(out, "pending_outbox_error=%v\n", issues)
		return
	}
	fmt.Fprintf(out, "pending_outbox=%v\n", programmingTeamOutboxSummariesV0(pending))
}

func programmingTeamOutboxSummariesV0(messages []orquestacoreworkflow.OutboxMessageV0) []string {
	summaries := make([]string, 0, len(messages))
	for _, message := range messages {
		summaries = append(summaries, message.TargetPort+":"+message.MessageType+":"+message.MessageID)
	}
	return summaries
}

func programmingTeamAppendProcessDiagnosticsV0(
	ctx context.Context,
	out *strings.Builder,
	registry orquestacionnucleoapp.AgentProcessRegistryPortV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) {
	agents := programmingTeamUniqueRefsV0(append(append([]string{}, run.Agents...), run.StartedAgents...))
	statuses := make([]string, 0, len(agents))
	for _, agentRef := range agents {
		record, err := registry.ResolveAgentProcessV0(ctx, run.RunID, agentRef)
		if err != nil {
			statuses = append(statuses, agentRef+"=missing_process")
			continue
		}
		statuses = append(statuses, agentRef+"=process:"+record.ProcessRef+":session:"+record.SessionRef)
	}
	fmt.Fprintf(out, "agent_processes=%v\n", statuses)
}

func programmingTeamStoredTaskSummariesV0(tasks []orquestacoreworkflow.WorkflowTaskV0) []string {
	summaries := make([]string, 0, len(tasks))
	for _, task := range tasks {
		summaries = append(summaries, fmt.Sprintf("%s:%s:writes=%d", task.TaskID, task.PhaseID, len(task.WriteSet)))
	}
	return summaries
}

func programmingTeamMissingDeliveryRefsV0(tasks []programmingTeamTaskV0, deliveries []string) []string {
	expected := make([]string, 0, len(tasks))
	for _, task := range tasks {
		expected = append(expected, "ack-ref-programming-"+task.Area)
	}
	return programmingTeamMissingRefsV0(expected, deliveries)
}

func programmingTeamMissingRefsV0(expected []string, actual []string) []string {
	missing := []string{}
	for _, ref := range expected {
		if !codexDeliveryLoopContainsRefV0(actual, ref) {
			missing = append(missing, ref)
		}
	}
	return missing
}

func programmingTeamUniqueRefsV0(values []string) []string {
	seen := map[string]bool{}
	refs := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		refs = append(refs, value)
	}
	return refs
}

func programmingTeamFailureKindV0(err error) string {
	var burstErr orquestadirectorsupervisedburst.DirectorSupervisedBurstErrorV0
	if errors.As(err, &burstErr) {
		if burstErr.Code == orquestadirectorsupervisedburst.ErrDirectorSupervisedBurstStepInputV0 {
			return "tick_input step_input_builder"
		}
		return burstErr.Code
	}
	if strings.Contains(err.Error(), orquestadirectorsupervisedburst.ErrDirectorSupervisedBurstStepInputV0) {
		return "tick_input step_input_builder"
	}
	return "other"
}

func programmingTeamEventTypesV0(
	events []orquestacoreworkflow.OrchestrationEventV0,
) []string {
	types := make([]string, 0, len(events))
	for _, event := range events {
		types = append(types, event.EventType)
	}
	return types
}

func programmingTeamEventCountsV0(
	events []orquestacoreworkflow.OrchestrationEventV0,
) map[string]int {
	counts := map[string]int{}
	for _, event := range events {
		counts[event.EventType]++
	}
	return counts
}
