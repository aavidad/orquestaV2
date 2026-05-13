package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	orquestacorereplanner "orquesta/modulos/orquesta-core-replanner"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestCodexStackV0AssessmentLoopStoppedWorkflowTaskReplansReplacementAgent(t *testing.T) {
	ctx := context.Background()
	runtime := newProgrammingPendingCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	director := postDirectorAPIV0(t, stack)
	codexStackWriteDelayedDirectorDecisionsForTestV0(t, stack, director.RunRef)

	taskRef := "task-ref-stack-agenda-001"
	oldAgentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	replacementAgentRef := "agent-ref-stack-assessment-replan-replacement-001"

	if _, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
		RunRef:               director.RunRef,
		CorrelationID:        "corr-stack-assessment-replan-start-001",
		MaxBursts:            16,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 8,
		MaxExternalWaits:     2,
	}); err != nil {
		t.Fatalf("DrainRunV0 start programming: %v", err)
	}
	run, err := stack.Stores.RunStore.LoadRunV0(ctx, director.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 start: %v", err)
	}
	if !codexStackHasRefV0(run.StartedAgents, oldAgentRef) {
		t.Fatalf("workflow task agent no arrancado: task=%s agent=%s run=%+v", taskRef, oldAgentRef, run)
	}

	stack.Ports.ProgressSource = loopOnceProgressSourceForTestV0{
		AgentRef: oldAgentRef,
		TaskRef:  taskRef,
	}
	stack.Ports.AssessmentReplanSource = assessmentLoopReplacementSourceForTestV0{
		OldAgentRef:         oldAgentRef,
		TaskRef:             taskRef,
		ReplacementAgentRef: replacementAgentRef,
	}

	drain, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
		RunRef:               director.RunRef,
		OccurredAt:           "2026-05-10T12:40:00Z",
		CorrelationID:        "corr-stack-assessment-replan-001",
		MaxBursts:            20,
		MaxStepsPerBurst:     6,
		MaxDispatchesPerWait: 8,
		MaxCommands:          20,
		MaxOutboxPerCycle:    8,
		MaxExternalWaits:     3,
	})
	if err != nil {
		t.Fatalf("DrainRunV0 replan assessment: %#v status=%s final=%+v", err, drain.Status, drain.Final)
	}
	run, err = stack.Stores.RunStore.LoadRunV0(ctx, director.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 final: %v", err)
	}

	if !codexStackRefsContainPartV0(run.AgentAssessments, stackAssessmentLoopRefForTestV0(oldAgentRef)) ||
		!codexStackHasRefV0(run.StoppedAgents, oldAgentRef) ||
		!codexStackHasRefV0(run.ConfirmedStoppedAgents, oldAgentRef) ||
		!codexStackHasRefV0(run.ReplanDecisions, stackAssessmentReplanProjectionForTestV0(oldAgentRef, taskRef, replacementAgentRef)) ||
		!codexStackHasRefV0(run.CapacityRequests, stackAssessmentCapacityRefForTestV0(taskRef)) ||
		!codexStackRefsContainPartV0(run.CapacityDecisions, stackAssessmentCapacityRefForTestV0(taskRef)) ||
		!codexStackHasRefV0(run.Agents, replacementAgentRef) ||
		!codexStackHasRefV0(run.StartedAgents, replacementAgentRef) {
		t.Fatalf("run final sin replan completo para task=%s old=%s replacement=%s: %+v",
			taskRef,
			oldAgentRef,
			replacementAgentRef,
			run,
		)
	}
	assertCodexStackReplacementDescriptorTaskForTestV0(t, stack, director.RunRef, replacementAgentRef, taskRef)

	sink := stack.Stores.EventSink.(*orquestacionnucleoapp.InMemoryEventSinkV0)
	for _, eventType := range []string{
		orquestacoreworkflow.OrchestrationEventAgentWorkAssessedV0,
		orquestacoreworkflow.OrchestrationEventAgentStopConfirmedV0,
		orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0,
		orquestacoreworkflow.OrchestrationEventCapacityRequestedV0,
		orquestacoreworkflow.OrchestrationEventAgentRequestedV0,
		orquestacoreworkflow.OrchestrationEventAgentStartedV0,
	} {
		if !codexStackRealSmokeHasEventV0(sink.EventsV0(), eventType) {
			t.Fatalf("falta evento %s: %+v", eventType, sink.EventsV0())
		}
	}
}

type programmingPendingCodexStackRuntimeV0 struct {
	*fakeCodexStackRuntimeV0
}

func newProgrammingPendingCodexStackRuntimeV0() *programmingPendingCodexStackRuntimeV0 {
	return &programmingPendingCodexStackRuntimeV0{
		fakeCodexStackRuntimeV0: newFakeCodexStackRuntimeV0(),
	}
}

func (runtime *programmingPendingCodexStackRuntimeV0) LaunchV0(
	_ context.Context,
	req orquestaruntime.ProcessRuntimeLaunchRequestV0,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	packet, err := codexStackLaunchPacketForTestV0(req)
	if err != nil {
		return orquestaruntime.ProcessRuntimeSnapshotV0{}, err
	}
	if packet.Phase != string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0) {
		if err := runtime.writeAckV0(req); err != nil {
			return orquestaruntime.ProcessRuntimeSnapshotV0{}, err
		}
	}
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	runtime.next++
	ref := strconv.Itoa(runtime.next)
	snapshot := orquestaruntime.ProcessRuntimeSnapshotV0{
		SchemaVersion: orquestaruntime.ProcessRuntimeConnectorVersionV0,
		ProcessRef:    "process-ref-app-stack-assessment-replan-" + ref,
		SessionRef:    "session-ref-app-stack-assessment-replan-" + ref,
		LaunchRef:     "launch-ref-app-stack-assessment-replan-" + ref,
		Status:        orquestaruntime.ProcessRuntimeRunningV0,
	}
	runtime.snapshots[snapshot.ProcessRef] = snapshot
	return snapshot, nil
}

func codexStackLaunchPacketForTestV0(
	req orquestaruntime.ProcessRuntimeLaunchRequestV0,
) (orquestaruntime.AgentStartPacketV0, error) {
	runtimeDir := filepath.Dir(req.CommandPath)
	packetPath := filepath.Join(runtimeDir, orquestaruntimecodex.CodexAgentPacketFileNameV0)
	data, err := os.ReadFile(packetPath)
	if err != nil {
		return orquestaruntime.AgentStartPacketV0{}, err
	}
	var packet orquestaruntime.AgentStartPacketV0
	if err := json.Unmarshal(data, &packet); err != nil {
		return orquestaruntime.AgentStartPacketV0{}, err
	}
	return packet, nil
}

type loopOnceProgressSourceForTestV0 struct {
	AgentRef string
	TaskRef  string
}

func (source loopOnceProgressSourceForTestV0) BuildAgentProgressObservationsV0(
	_ context.Context,
	request orquestacionnucleoapp.AgentProgressObservationRequestV0,
) ([]orquestacionnucleoapp.AgentProgressObservationV0, error) {
	if request.Run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 ||
		!codexStackHasRefV0(request.Run.StartedAgents, source.AgentRef) ||
		codexStackHasRefV0(request.Run.StoppedAgents, source.AgentRef) ||
		codexStackRefsContainPartV0(request.Run.AgentAssessments, stackAssessmentLoopRefForTestV0(source.AgentRef)) {
		return nil, nil
	}
	reportRef := "agent-progress-report-ref-" + source.AgentRef + "-loop"
	return []orquestacionnucleoapp.AgentProgressObservationV0{{
		CandidateRef:  "progress-candidate-ref-" + reportRef,
		AssessmentRef: stackAssessmentLoopRefForTestV0(source.AgentRef),
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskRef:       source.TaskRef,
		Report: orquestaruntime.AgentProgressReportV0{
			ReportID:            reportRef,
			RunID:               request.Run.RunID,
			AgentRequestID:      source.AgentRef,
			Status:              orquestaruntime.AgentLoopDetectedV0,
			NoProgressTicks:     2,
			RepeatedActionCount: 2,
			Summary:             "Bucle compacto observado.",
			EvidenceRefs:        []string{"evidence-ref-stack-assessment-loop"},
		},
		EvidenceRefs: []string{"evidence-ref-stack-assessment-progress"},
	}}, nil
}

type assessmentLoopReplacementSourceForTestV0 struct {
	OldAgentRef         string
	TaskRef             string
	ReplacementAgentRef string
}

func (source assessmentLoopReplacementSourceForTestV0) BuildAgentAssessmentReplanPlansV0(
	_ context.Context,
	request orquestacionnucleoapp.AgentAssessmentReplanPlanRequestV0,
) ([]orquestacionnucleoapp.AgentAssessmentReplanPlanV0, error) {
	if request.Run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 ||
		!codexStackHasRefV0(request.Run.ConfirmedStoppedAgents, source.OldAgentRef) ||
		!codexStackRefsContainPartV0(request.Run.AgentAssessments, stackAssessmentLoopRefForTestV0(source.OldAgentRef)) ||
		codexStackHasRefV0(request.Run.Agents, source.ReplacementAgentRef) {
		return nil, nil
	}
	return []orquestacionnucleoapp.AgentAssessmentReplanPlanV0{{
		CandidateRef:               "assessment-replan-candidate-ref-stack-loop-001",
		ReplanRef:                  stackAssessmentReplanRefForTestV0(source.OldAgentRef),
		SignalRef:                  "signal-ref-stack-assessment-loop-001",
		TaskRef:                    source.TaskRef,
		ReasonRef:                  "reason-ref-stack-assessment-loop",
		RequestedAction:            orquestacorereplanner.ReplanActionReplaceAgentV0,
		ReplacementRole:            "implementacion",
		CapacityRequestRef:         stackAssessmentCapacityRefForTestV0(source.TaskRef),
		AgentRequestID:             source.ReplacementAgentRef,
		MinimumRecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
		Summary:                    "Reemplazar agente detenido tras evaluacion de bucle.",
		EvidenceRefs:               []string{"evidence-ref-stack-assessment-replan"},
		Assessment: orquestacoreworkflow.AgentWorkAssessedPayloadV0{
			AssessmentRef:  stackAssessmentLoopRefForTestV0(source.OldAgentRef),
			PhaseID:        string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			AgentRequestID: source.OldAgentRef,
			TaskRef:        source.TaskRef,
			Verdict:        orquestacoreworkflow.AgentAssessmentVerdictLoopDetectedV0,
			Action:         orquestacoreworkflow.AgentAssessmentActionStopAgentV0,
			Severity:       orquestacoreworkflow.AgentAssessmentSeverityCriticalV0,
			Summary:        "Bucle compacto observado; agente detenido.",
			EvidenceRefs:   []string{"evidence-ref-stack-assessment-loop"},
		},
	}}, nil
}

func stackAssessmentLoopRefForTestV0(agentRef string) string {
	return "assessment-ref-stack-loop-" + strings.TrimSpace(agentRef)
}

func stackAssessmentReplanRefForTestV0(agentRef string) string {
	return "replan-ref-stack-loop-" + strings.TrimSpace(agentRef)
}

func stackAssessmentCapacityRefForTestV0(taskRef string) string {
	return "capacity-ref-stack-loop-" + strings.TrimSpace(taskRef)
}

func stackAssessmentReplanProjectionForTestV0(
	agentRef string,
	taskRef string,
	replacementAgentRef string,
) string {
	return stackAssessmentReplanRefForTestV0(agentRef) +
		"#source:" + stackAssessmentLoopRefForTestV0(agentRef) +
		"#task:" + strings.TrimSpace(taskRef) +
		"#action:" + string(orquestacoreworkflow.ReplanDecisionActionReplaceAgentV0) +
		"#followups:" + strings.TrimSpace(replacementAgentRef) +
		"+" + stackAssessmentCapacityRefForTestV0(taskRef)
}

func assertCodexStackReplacementDescriptorTaskForTestV0(
	t *testing.T,
	stack StackV0,
	runRef string,
	agentRef string,
	taskRef string,
) {
	t.Helper()
	descriptors, err := stack.Stores.ReceiptStore.ListCodexReceiptDescriptorsV0(
		context.Background(),
		orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0{
			RunID:         runRef,
			StartedAgents: []string{agentRef},
		},
	)
	if err != nil {
		t.Fatalf("ListCodexReceiptDescriptorsV0: %v", err)
	}
	for _, descriptor := range descriptors {
		if descriptor.AgentRef == agentRef &&
			descriptor.Spec.AgentPacket.Task.TaskRef == taskRef {
			return
		}
	}
	t.Fatalf("descriptor replacement no encontrado agent=%s task=%s descriptors=%+v", agentRef, taskRef, descriptors)
}
