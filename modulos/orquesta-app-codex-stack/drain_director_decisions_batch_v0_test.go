package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestadirectoragentfilesource "orquesta/modulos/orquesta-director-agent-file-source"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
	orquestadirectorsupervisedburst "orquesta/modulos/orquesta-director-supervised-burst"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestDrainRunV0ConsumeDecisionFileConLoteProgramacionGrande(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)

	director := postDirectorAPIV0(t, stack)
	codexStackWriteDelayedBatchDirectorDecisionsForTestV0(t, stack, director.RunRef)

	drain, err := stack.DrainRunV0(context.Background(), DrainRunRequestV0{
		RunRef:               director.RunRef,
		CorrelationID:        "corr-stack-batch-decisions-001",
		MaxBursts:            16,
		MaxStepsPerBurst:     12,
		MaxDispatchesPerWait: 8,
		MaxCommands:          20,
		MaxOutboxPerCycle:    8,
		MaxExternalWaits:     4,
	})
	if err != nil {
		t.Fatalf("DrainRunV0 batch: %v issues=%+v drain=%+v", err, codexStackBurstIssuesForErrorV0(err), drain)
	}
	run, err := stack.Stores.RunStore.LoadRunV0(context.Background(), director.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	for _, taskRef := range codexStackBatchDecisionTaskRefsForTestV0() {
		agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
		if !codexStackStringInSetForTestV0(run.Tasks, taskRef) {
			t.Fatalf("tasks=%v missing=%s", run.Tasks, taskRef)
		}
		if !codexStackStringInSetForTestV0(run.StartedAgents, agentRef) {
			t.Fatalf("started_agents=%v missing=%s", run.StartedAgents, agentRef)
		}
	}
	if runtime.launchCountV0() < 9 {
		t.Fatalf("launches=%d want>=9", runtime.launchCountV0())
	}
}

func TestDrainRunV0NormalizaCreateMicrotaskPhaseObjetivoDelDirectorReal(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)

	director := postDirectorAPIV0(t, stack)
	codexStackWriteDelayedBatchDirectorDecisionsWithMicrotaskPhaseForTestV0(
		t,
		stack,
		director.RunRef,
		string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
	)

	drain, err := stack.DrainRunV0(context.Background(), DrainRunRequestV0{
		RunRef:               director.RunRef,
		CorrelationID:        "corr-stack-batch-decisions-real-phase-001",
		MaxBursts:            16,
		MaxStepsPerBurst:     12,
		MaxDispatchesPerWait: 8,
		MaxCommands:          20,
		MaxOutboxPerCycle:    8,
		MaxExternalWaits:     4,
	})
	if err != nil {
		t.Fatalf("DrainRunV0 batch real phase: %v issues=%+v drain=%+v",
			err,
			codexStackBurstIssuesForErrorV0(err),
			drain,
		)
	}
	run, err := stack.Stores.RunStore.LoadRunV0(context.Background(), director.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	for _, taskRef := range codexStackBatchDecisionTaskRefsForTestV0() {
		if !codexStackStringInSetForTestV0(run.Tasks, taskRef) {
			t.Fatalf("tasks=%v missing=%s", run.Tasks, taskRef)
		}
	}
}

func TestDrainRunV0ConsumeDecisionFileVerticalGoConGlobsDeDirectorReal(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)

	director := postDirectorAPIV0(t, stack)
	codexStackWriteDelayedVerticalGoDirectorDecisionsForTestV0(t, stack, director.RunRef)

	for cycle := 1; cycle <= 4; cycle++ {
		drain, err := stack.DrainRunV0(context.Background(), DrainRunRequestV0{
			RunRef:               director.RunRef,
			CorrelationID:        "corr-stack-vertical-go-decisions-00" + string(rune('0'+cycle)),
			MaxBursts:            16,
			MaxStepsPerBurst:     12,
			MaxDispatchesPerWait: 8,
			MaxCommands:          20,
			MaxOutboxPerCycle:    8,
			MaxDecisionCycles:    16,
			MaxExternalWaits:     4,
		})
		if err != nil {
			t.Fatalf("DrainRunV0 vertical go ciclo=%d: %v issues=%+v drain=%+v", cycle, err, codexStackBurstIssuesForErrorV0(err), drain)
		}
	}
	run, err := stack.Stores.RunStore.LoadRunV0(context.Background(), director.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	taskRef := "workflow-task-agenda-programacion-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	if !codexStackStringInSetForTestV0(run.Tasks, taskRef) {
		decisions, decisionErr := stack.Ports.DirectorDecisionSource.ListDirectorAgentDecisionsV0(
			context.Background(),
			orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{
				Run:           run,
				CorrelationID: "corr-stack-vertical-go-diagnostic",
				RequestedBy:   "orquesta-app-codex-stack-test",
			},
		)
		appSpec := codexStackAppSpecRequestV0()
		decisionsWithHints, decisionWithHintsErr := stack.Ports.DirectorDecisionSource.ListDirectorAgentDecisionsV0(
			context.Background(),
			orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{
				Run:           run,
				RequestKind:   appSpec.RequestKind,
				ExecutionMode: appSpec.ExecutionMode,
				ObjectiveHints: []string{
					appSpec.Nombre,
					appSpec.Objetivo,
					appSpec.PreferenciasTecnicas.Lenguaje,
					appSpec.PreferenciasTecnicas.Arquitectura,
					appSpec.Datos.NecesidadFuncional,
				},
				CorrelationID: "corr-stack-vertical-go-diagnostic-hints",
				RequestedBy:   "orquesta-app-codex-stack-test",
			},
		)
		decisionCommands := make([]string, 0, len(decisions))
		applyErr := error(nil)
		applyIssues := []orquestadirectoragentworkflow.DirectorAgentWorkflowIssueV0{}
		for _, decision := range decisions {
			decisionCommands = append(decisionCommands, decision.CommandType+":"+decision.DecisionRef)
			if decision.CreateMicrotask == nil {
				continue
			}
			applied, err := orquestadirectoragentworkflow.ApplyDirectorAgentDecisionV0(
				context.Background(),
				orquestadirectoragentworkflow.ApplyDirectorAgentDecisionRequestV0{
					Decision:      decision,
					OccurredAt:    "2026-05-22T12:00:00Z",
					CorrelationID: "corr-stack-vertical-go-diagnostic-apply",
					RequestedBy:   "orquesta-app-codex-stack-test",
				},
				orquestadirectoragentworkflow.ApplyDirectorAgentDecisionPortsV0{
					RunStore:  stack.Ports.RunStore,
					EventSink: stack.Ports.EventSink,
					TaskStore: stack.Ports.DirectorTaskStore,
				},
			)
			applyErr = err
			applyIssues = applied.Issues
			break
		}
		t.Fatalf(
			"tasks=%v missing=%s phase=%s votes=%v decisions=%v contracts=%v artifacts=%v source_len=%d source_hints_len=%d commands=%v source_err=%v source_hints_err=%v apply_err=%v apply_issues=%+v",
			run.Tasks,
			taskRef,
			run.CurrentPhase,
			run.Votes,
			run.Decisions,
			run.FunctionContracts,
			run.PhaseArtifacts,
			len(decisions),
			len(decisionsWithHints),
			decisionCommands,
			decisionErr,
			decisionWithHintsErr,
			applyErr,
			applyIssues,
		)
	}
	if !codexStackStringInSetForTestV0(run.StartedAgents, agentRef) {
		t.Fatalf("started_agents=%v missing=%s phase=%s", run.StartedAgents, agentRef, run.CurrentPhase)
	}
}

func codexStackBurstIssuesForErrorV0(err error) []orquestadirectorsupervisedburst.DirectorSupervisedBurstIssueV0 {
	var burstErr orquestadirectorsupervisedburst.DirectorSupervisedBurstErrorV0
	if errors.As(err, &burstErr) {
		return burstErr.Issues
	}
	return nil
}

func codexStackWriteDelayedBatchDirectorDecisionsForTestV0(
	t *testing.T,
	stack StackV0,
	runRef string,
) {
	t.Helper()
	store, ok := stack.Stores.ReceiptStore.(*orquestaruntimecodexdelivery.InMemoryCodexReceiptDescriptorStoreV0)
	if !ok {
		t.Fatalf("receipt store inesperado: %T", stack.Stores.ReceiptStore)
	}
	for _, descriptor := range codexStackRealSmokeDescriptorsV0(t, store) {
		if descriptor.Spec.AgentPacket.TargetModule != "orquesta-app-stack-director" {
			continue
		}
		writeDelayedBatchDirectorDecisionFileForTestV0(t, descriptor, runRef)
		return
	}
	t.Fatalf("descriptor director no encontrado")
}

func codexStackWriteDelayedVerticalGoDirectorDecisionsForTestV0(
	t *testing.T,
	stack StackV0,
	runRef string,
) {
	t.Helper()
	store, ok := stack.Stores.ReceiptStore.(*orquestaruntimecodexdelivery.InMemoryCodexReceiptDescriptorStoreV0)
	if !ok {
		t.Fatalf("receipt store inesperado: %T", stack.Stores.ReceiptStore)
	}
	for _, descriptor := range codexStackRealSmokeDescriptorsV0(t, store) {
		if descriptor.Spec.AgentPacket.TargetModule != "orquesta-app-stack-director" {
			continue
		}
		writeDelayedVerticalGoDirectorDecisionFileForTestV0(t, descriptor, runRef)
		return
	}
	t.Fatalf("descriptor director no encontrado")
}

func codexStackWriteDelayedBatchDirectorDecisionsWithMicrotaskPhaseForTestV0(
	t *testing.T,
	stack StackV0,
	runRef string,
	phaseID string,
) {
	t.Helper()
	store, ok := stack.Stores.ReceiptStore.(*orquestaruntimecodexdelivery.InMemoryCodexReceiptDescriptorStoreV0)
	if !ok {
		t.Fatalf("receipt store inesperado: %T", stack.Stores.ReceiptStore)
	}
	for _, descriptor := range codexStackRealSmokeDescriptorsV0(t, store) {
		if descriptor.Spec.AgentPacket.TargetModule != "orquesta-app-stack-director" {
			continue
		}
		writeDelayedBatchDirectorDecisionFileWithMicrotaskPhaseForTestV0(t, descriptor, runRef, phaseID)
		return
	}
	t.Fatalf("descriptor director no encontrado")
}

func writeDelayedBatchDirectorDecisionFileForTestV0(
	t *testing.T,
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
	runRef string,
) {
	t.Helper()
	brainstormRef := codexStackObjectiveValueForTestV0(
		descriptor.Spec.AgentPacket.Task.Objective,
		"BrainstormRef inicial:",
	)
	if brainstormRef == "" {
		t.Fatalf("brainstorm ref vacio en objetivo director")
	}
	data, err := json.Marshal(orquestadirectoragentfilesource.DirectorAgentDecisionFileEnvelopeV0{
		SchemaVersion: orquestadirectoragentfilesource.DirectorAgentDecisionFileSchemaVersionV0,
		Decisions:     codexStackBatchDirectorDecisionsForTestV0(runRef, brainstormRef),
	})
	if err != nil {
		t.Fatalf("marshal batch decisions: %v", err)
	}
	path := filepath.Join(filepath.Dir(descriptor.AckPath), orquestaruntimecodex.CodexDirectorDecisionsFileNameV0)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write batch decisions: %v", err)
	}
}

func writeDelayedVerticalGoDirectorDecisionFileForTestV0(
	t *testing.T,
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
	runRef string,
) {
	t.Helper()
	brainstormRef := codexStackObjectiveValueForTestV0(
		descriptor.Spec.AgentPacket.Task.Objective,
		"BrainstormRef inicial:",
	)
	if brainstormRef == "" {
		t.Fatalf("brainstorm ref vacio en objetivo director")
	}
	data, err := json.Marshal(orquestadirectoragentfilesource.DirectorAgentDecisionFileEnvelopeV0{
		SchemaVersion: orquestadirectoragentfilesource.DirectorAgentDecisionFileSchemaVersionV0,
		Decisions:     codexStackVerticalGoDirectorDecisionsForTestV0(runRef, brainstormRef),
	})
	if err != nil {
		t.Fatalf("marshal vertical decisions: %v", err)
	}
	path := filepath.Join(filepath.Dir(descriptor.AckPath), orquestaruntimecodex.CodexDirectorDecisionsFileNameV0)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write vertical decisions: %v", err)
	}
}

func writeDelayedBatchDirectorDecisionFileWithMicrotaskPhaseForTestV0(
	t *testing.T,
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
	runRef string,
	phaseID string,
) {
	t.Helper()
	brainstormRef := codexStackObjectiveValueForTestV0(
		descriptor.Spec.AgentPacket.Task.Objective,
		"BrainstormRef inicial:",
	)
	if brainstormRef == "" {
		t.Fatalf("brainstorm ref vacio en objetivo director")
	}
	decisions := codexStackBatchDirectorDecisionsForTestV0(runRef, brainstormRef)
	for i := range decisions {
		if decisions[i].CreateMicrotask != nil {
			decisions[i].PhaseID = phaseID
		}
	}
	data, err := json.Marshal(orquestadirectoragentfilesource.DirectorAgentDecisionFileEnvelopeV0{
		SchemaVersion: orquestadirectoragentfilesource.DirectorAgentDecisionFileSchemaVersionV0,
		Decisions:     decisions,
	})
	if err != nil {
		t.Fatalf("marshal batch decisions real phase: %v", err)
	}
	path := filepath.Join(filepath.Dir(descriptor.AckPath), orquestaruntimecodex.CodexDirectorDecisionsFileNameV0)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write batch decisions real phase: %v", err)
	}
}

func codexStackBatchDirectorDecisionsForTestV0(
	runRef string,
	brainstormRef string,
) []orquestadirectoragent.DirectorAgentDecisionV0 {
	decisions := []orquestadirectoragent.DirectorAgentDecisionV0{
		codexStackOpenPhaseDecisionForTestV0(
			runRef,
			"director-decision-stack-open-vote-batch",
			"command-ref-stack-open-vote-batch",
			orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0,
			orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0,
		),
		codexStackVoteDecisionForTestV0(runRef, brainstormRef),
		codexStackAcceptDecisionForTestV0(runRef),
		codexStackOpenPhaseDecisionForTestV0(
			runRef,
			"director-decision-stack-open-plan-batch",
			"command-ref-stack-open-plan-batch",
			orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0,
			orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0,
		),
		codexStackContractDecisionForTestV0(runRef),
	}
	for _, spec := range codexStackBatchDecisionTaskSpecsForTestV0() {
		decisions = append(decisions, codexStackMicrotaskDecisionForBatchTestV0(runRef, spec))
	}
	return append(decisions, codexStackOpenPhaseDecisionForTestV0(
		runRef,
		"director-decision-stack-open-program-batch",
		"command-ref-stack-open-program-batch",
		orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0,
		orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
	))
}
