package orquestaappcodexstack

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	orquestacorereplanner "orquesta/modulos/orquesta-core-replanner"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestCodexStackGeneratedRecursiveTreeFakeRuntimeV0(t *testing.T) {
	ctx := context.Background()
	cfg := codexStackRequiredTestLocalConfigV0(t)
	cfg.MaxBatchReady = 2
	// La runtime fake no marca procesos como parados tras escribir ACK; esta prueba
	// valida recursion completa, no el gate global de procesos vivos.
	cfg.MaxConcurrency = 8
	writeCodexStackRequiredTestTinyGoModuleV0(t, cfg.ProjectWorkDir)
	goCommand := codexStackRequiredTestGoCommandV0(t)
	outputDir := filepath.Join(t.TempDir(), "required-test-output")

	runtime := &codexStackRequiredTestPendingRuntimeV0{}
	evidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0()
	stack := codexStackRealRequiredTestRunnerStackV0(t, cfg, runtime, evidenceStore, goCommand, outputDir)
	stack.Ports.ProgressSource = nil
	stack.Ports.ReviewGateSource = nil

	fixture := newGeneratedRecursiveFlowFixtureV0()
	stack.Ports.ReviewReworkReplanSource = generatedRecursiveFlowSplitSourceV0{
		plansByReworkRef: fixture.SplitPlansByReworkRef,
	}
	seedGeneratedRecursiveFlowRootV0(t, ctx, stack, fixture)

	parentAgent := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(fixture.Parent.TaskID)
	childAgents := generatedRecursiveFlowAgentRefsV0(fixture.Children)
	grandAAgents := generatedRecursiveFlowAgentRefsV0(fixture.GrandchildrenA)
	grandBAgents := generatedRecursiveFlowAgentRefsV0(fixture.GrandchildrenB)

	codexStackRecursiveTreeContinueLaunchV0(t, ctx, stack, runtime, fixture.RunRef, appDirectorWaitFilterForRecursiveTreeV0{
		WaveRef:   fixture.WaveRef,
		CohortRef: fixture.CohortRef,
	}, []string{parentAgent})
	parentDelivery := codexStackRecursiveTreeDrainAgentV0(t, ctx, stack, fixture.RunRef, parentAgent, cfg, true)
	assertGeneratedRecursiveFlowTasksMissingV0(t, ctx, stack, fixture.RunRef, generatedRecursiveFlowTaskRefsV0(fixture.Children))

	ensureGeneratedRecursiveFlowPhaseV0(t, stack, fixture.RunRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0, "Revisar padre recursivo generado.")
	seedGeneratedRecursiveFlowChangesRequestedV0(t, ctx, stack, fixture.RunRef, fixture.Parent.TaskID, parentDelivery, "parent", 10)
	continueGeneratedRecursiveFlowReplanV0(t, ctx, stack, fixture.RunRef, "parent")
	assertGeneratedRecursiveFlowTasksStoredV0(t, ctx, stack, fixture.RunRef, fixture.Children)
	assertGeneratedRecursiveFlowRunRefsV0(t, stack, fixture.RunRef, generatedRecursiveFlowTaskRefsV0(fixture.Children), childAgents)
	codexStackRecursiveTreeAssertWaitSnapshotV0(t, ctx, stack, fixture.RunRef, fixture.Parent.TaskID, fixture.WaveRef, childAgents, childAgents)

	childADelivery := codexStackRecursiveTreeDrainAgentV0(t, ctx, stack, fixture.RunRef, childAgents[0], cfg, true)
	ensureGeneratedRecursiveFlowPhaseV0(t, stack, fixture.RunRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0, "Revisar hijo A generado.")
	seedGeneratedRecursiveFlowChangesRequestedV0(t, ctx, stack, fixture.RunRef, fixture.Children[0].TaskID, childADelivery, "child-a", 20)
	continueGeneratedRecursiveFlowReplanV0(t, ctx, stack, fixture.RunRef, "child-a")
	assertGeneratedRecursiveFlowTasksStoredV0(t, ctx, stack, fixture.RunRef, fixture.GrandchildrenA)
	assertGeneratedRecursiveFlowRunRefsV0(t, stack, fixture.RunRef, generatedRecursiveFlowTaskRefsV0(fixture.GrandchildrenA), grandAAgents)
	codexStackRecursiveTreeAssertWaitSnapshotV0(t, ctx, stack, fixture.RunRef, fixture.Children[0].TaskID, fixture.WaveRef, grandAAgents, grandAAgents)

	childBDelivery := codexStackRecursiveTreeDrainAgentV0(t, ctx, stack, fixture.RunRef, childAgents[1], cfg, true)
	ensureGeneratedRecursiveFlowPhaseV0(t, stack, fixture.RunRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0, "Revisar hijo B generado.")
	seedGeneratedRecursiveFlowChangesRequestedV0(t, ctx, stack, fixture.RunRef, fixture.Children[1].TaskID, childBDelivery, "child-b", 30)
	continueGeneratedRecursiveFlowReplanV0(t, ctx, stack, fixture.RunRef, "child-b")
	assertGeneratedRecursiveFlowTasksStoredV0(t, ctx, stack, fixture.RunRef, fixture.GrandchildrenB)
	assertGeneratedRecursiveFlowRunRefsV0(t, stack, fixture.RunRef, generatedRecursiveFlowTaskRefsV0(fixture.GrandchildrenB), grandBAgents)
	codexStackRecursiveTreeAssertWaitSnapshotV0(t, ctx, stack, fixture.RunRef, fixture.Children[1].TaskID, fixture.WaveRef, grandBAgents, grandBAgents)

	allTasks := append([]orquestacoreworkflow.WorkflowTaskV0{fixture.Parent}, fixture.Children...)
	allTasks = append(allTasks, fixture.GrandchildrenA...)
	allTasks = append(allTasks, fixture.GrandchildrenB...)
	stackOperationalClosureAssertRecursiveLimitsForTestV0(t, allTasks, 2)
	if runtime.launchCountV0() != len(allTasks) {
		t.Fatalf("launches=%d want=%d", runtime.launchCountV0(), len(allTasks))
	}
}

type generatedRecursiveFlowFixtureV0 struct {
	RunRef                string
	PlanRef               string
	WaveRef               string
	CohortRef             string
	Parent                orquestacoreworkflow.WorkflowTaskV0
	Children              []orquestacoreworkflow.WorkflowTaskV0
	GrandchildrenA        []orquestacoreworkflow.WorkflowTaskV0
	GrandchildrenB        []orquestacoreworkflow.WorkflowTaskV0
	SplitPlansByReworkRef map[string]generatedRecursiveFlowSplitPlanV0
}

type generatedRecursiveFlowSplitPlanV0 struct {
	TaskRef string
	Tasks   []orquestacoreworkflow.WorkflowTaskV0
}

type generatedRecursiveFlowSplitSourceV0 struct {
	plansByReworkRef map[string]generatedRecursiveFlowSplitPlanV0
}

func (source generatedRecursiveFlowSplitSourceV0) BuildReviewReworkReplanPlansV0(
	_ context.Context,
	request orquestacionnucleoapp.ReviewReworkReplanPlanRequestV0,
) ([]orquestacionnucleoapp.ReviewReworkReplanPlanV0, error) {
	plans := []orquestacionnucleoapp.ReviewReworkReplanPlanV0{}
	for _, raw := range request.Run.ReworkRequests {
		reworkRef, tail, ok := strings.Cut(strings.TrimSpace(raw), "#review_result:")
		if !ok {
			continue
		}
		split, ok := source.plansByReworkRef[reworkRef]
		if !ok {
			continue
		}
		reviewResultRef, tail, ok := strings.Cut(tail, "#review_request:")
		if !ok {
			continue
		}
		reviewRequestRef, deliveryRef, ok := strings.Cut(tail, "#delivery:")
		if !ok {
			continue
		}
		plans = append(plans, generatedRecursiveFlowReviewReworkPlanV0(
			reworkRef,
			strings.TrimSpace(reviewResultRef),
			strings.TrimSpace(reviewRequestRef),
			strings.TrimSpace(deliveryRef),
			split,
		))
	}
	return plans, nil
}

func newGeneratedRecursiveFlowFixtureV0() generatedRecursiveFlowFixtureV0 {
	runRef := "run-stack-generated-recursive-tree-001"
	waveRef := "wave-generated-recursive-tree"
	cohortRef := "cohort-generated-recursive-tree"
	parent := stackOperationalClosureRecursiveTaskForTestV0(runRef, "task-generated-parent", "", waveRef, cohortRef, 0, 2)
	childA := stackOperationalClosureRecursiveTaskForTestV0(runRef, "task-generated-child-a", parent.TaskID, waveRef, cohortRef, 1, 2)
	childB := stackOperationalClosureRecursiveTaskForTestV0(runRef, "task-generated-child-b", parent.TaskID, waveRef, cohortRef, 1, 2)
	grandA1 := stackOperationalClosureRecursiveTaskForTestV0(runRef, "task-generated-grand-a1", childA.TaskID, waveRef, cohortRef, 2, 0)
	grandA2 := stackOperationalClosureRecursiveTaskForTestV0(runRef, "task-generated-grand-a2", childA.TaskID, waveRef, cohortRef, 2, 0)
	grandB1 := stackOperationalClosureRecursiveTaskForTestV0(runRef, "task-generated-grand-b1", childB.TaskID, waveRef, cohortRef, 2, 0)
	grandB2 := stackOperationalClosureRecursiveTaskForTestV0(runRef, "task-generated-grand-b2", childB.TaskID, waveRef, cohortRef, 2, 0)
	childA.DependsOn = []string{parent.TaskID}
	childB.DependsOn = []string{parent.TaskID}
	grandA1.DependsOn = []string{childA.TaskID}
	grandA2.DependsOn = []string{childA.TaskID}
	grandB1.DependsOn = []string{childB.TaskID}
	grandB2.DependsOn = []string{childB.TaskID}
	return generatedRecursiveFlowFixtureV0{
		RunRef:         runRef,
		PlanRef:        "plan-stack-generated-recursive-tree-001",
		WaveRef:        waveRef,
		CohortRef:      cohortRef,
		Parent:         parent,
		Children:       []orquestacoreworkflow.WorkflowTaskV0{childA, childB},
		GrandchildrenA: []orquestacoreworkflow.WorkflowTaskV0{grandA1, grandA2},
		GrandchildrenB: []orquestacoreworkflow.WorkflowTaskV0{grandB1, grandB2},
		SplitPlansByReworkRef: map[string]generatedRecursiveFlowSplitPlanV0{
			"rework-generated-parent":  {TaskRef: parent.TaskID, Tasks: []orquestacoreworkflow.WorkflowTaskV0{childA, childB}},
			"rework-generated-child-a": {TaskRef: childA.TaskID, Tasks: []orquestacoreworkflow.WorkflowTaskV0{grandA1, grandA2}},
			"rework-generated-child-b": {TaskRef: childB.TaskID, Tasks: []orquestacoreworkflow.WorkflowTaskV0{grandB1, grandB2}},
		},
	}
}

func seedGeneratedRecursiveFlowRootV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	fixture generatedRecursiveFlowFixtureV0,
) {
	t.Helper()
	run := (codexStackRequiredTestRefsV0{
		RunRef:  fixture.RunRef,
		PlanRef: fixture.PlanRef,
		TaskRef: fixture.Parent.TaskID,
	}).withDefaultsV0().pendingRunV0()
	run.Tasks = []string{fixture.Parent.TaskID}
	run.FunctionContracts = []string{"contract:function:operational-director:v0"}
	if err := stack.Stores.RunStore.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0 root: %v", err)
	}
	taskWriter, ok := stack.Stores.TaskStore.(orquestacionnucleoapp.WorkflowTaskWriterPortV0)
	if !ok {
		t.Fatalf("TaskStore no escribe WorkflowTaskV0: %T", stack.Stores.TaskStore)
	}
	if err := taskWriter.SaveWorkflowTaskV0(ctx, fixture.Parent); err != nil {
		t.Fatalf("SaveWorkflowTaskV0 parent: %v", err)
	}
}

func generatedRecursiveFlowReviewReworkPlanV0(
	reworkRef string,
	reviewResultRef string,
	reviewRequestRef string,
	deliveryRef string,
	split generatedRecursiveFlowSplitPlanV0,
) orquestacionnucleoapp.ReviewReworkReplanPlanV0 {
	suffix := strings.TrimPrefix(reworkRef, "rework-generated-")
	return orquestacionnucleoapp.ReviewReworkReplanPlanV0{
		CandidateRef:     "candidate-generated-" + suffix,
		ReplanRef:        "replan-generated-" + suffix,
		SignalRef:        "signal-generated-" + suffix,
		ReworkRequestRef: reworkRef,
		TaskRef:          split.TaskRef,
		ReasonRef:        "reason-generated-" + suffix,
		RequestedAction:  orquestacorereplanner.ReplanActionSplitTaskV0,
		Summary:          "Dividir subarbol recursivo generado tras review.",
		EvidenceRefs:     []string{"evidence-generated-" + suffix},
		ReviewResult: orquestacoreworkflow.ReviewResultV0{
			ReviewResultRef: reviewResultRef,
			ReviewRequestID: reviewRequestRef,
			DeliveryRef:     deliveryRef,
			Status:          orquestacoreworkflow.ReviewResultStatusChangesRequestedV0,
			Summary:         "La entrega requiere delegacion recursiva.",
			EvidenceRefs:    []string{"evidence-review-generated-" + suffix},
			QualityGateRef:  "quality-gate-generated-" + suffix,
		},
		SplitTasks: split.Tasks,
	}
}

func seedGeneratedRecursiveFlowChangesRequestedV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	runRef string,
	taskRef string,
	actualDeliveryRef string,
	suffix string,
	sequenceBase int64,
) {
	t.Helper()
	reworkRef := "rework-generated-" + suffix
	deliveryRef := strings.TrimSpace(actualDeliveryRef)
	reviewRequestRef := "review-request-generated-" + suffix
	reviewResultRef := "review-result-generated-" + suffix
	if deliveryRef == "" {
		t.Fatalf("delivery real vacia para %s", taskRef)
	}
	run := mustLoadCodexStackRunForTestV0(t, stack, runRef)
	events := []orquestacoreworkflow.OrchestrationEventV0{}
	requested, err := orquestacoreworkflow.NewReviewRequestedEventV0(
		generatedRecursiveFlowEventMetaV0(runRef, sequenceBase+1, "review-request-"+suffix),
		orquestacoreworkflow.ReviewRequestedPayloadV0{
			ReviewRequestID: reviewRequestRef,
			PhaseID:         string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			DeliveryRef:     deliveryRef,
			Summary:         "Review causal para split_task generado.",
			EvidenceRefs:    []string{"evidence-request-generated-" + suffix, actualDeliveryRef},
		},
	)
	if err != nil {
		t.Fatalf("NewReviewRequestedEventV0 %s: %v", suffix, err)
	}
	recorded, err := orquestacoreworkflow.NewReviewResultRecordedEventV0(
		generatedRecursiveFlowEventMetaV0(runRef, sequenceBase+2, "review-result-"+suffix),
		orquestacoreworkflow.ReviewResultV0{
			ReviewResultRef: reviewResultRef,
			ReviewRequestID: reviewRequestRef,
			DeliveryRef:     deliveryRef,
			Status:          orquestacoreworkflow.ReviewResultStatusChangesRequestedV0,
			Summary:         "Review pide dividir la tarea en subagentes.",
			EvidenceRefs:    []string{"evidence-result-generated-" + suffix, taskRef},
		},
	)
	if err != nil {
		t.Fatalf("NewReviewResultRecordedEventV0 %s: %v", suffix, err)
	}
	rework, err := orquestacoreworkflow.NewReworkRequestedEventV0(
		generatedRecursiveFlowEventMetaV0(runRef, sequenceBase+3, "rework-"+suffix),
		orquestacoreworkflow.ReworkRequestedPayloadV0{
			ReworkRequestRef: reworkRef,
			PhaseID:          string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			ReviewResultRef:  reviewResultRef,
			ReviewRequestID:  reviewRequestRef,
			DeliveryRef:      deliveryRef,
			Summary:          "Rework estructurado como split_task.",
			EvidenceRefs:     []string{"evidence-rework-generated-" + suffix},
		},
	)
	if err != nil {
		t.Fatalf("NewReworkRequestedEventV0 %s: %v", suffix, err)
	}
	events = append(events, requested, recorded, rework)
	if err := stack.Stores.EventSink.AppendRunEventsV0(ctx, runRef, events); err != nil {
		t.Fatalf("AppendRunEventsV0 %s: %v", suffix, err)
	}
	run.Reviews = append(run.Reviews, reviewRequestRef)
	run.ReviewResults = append(run.ReviewResults, reviewResultRef+"#review_result:changes_requested#review_request:"+reviewRequestRef+"#delivery:"+deliveryRef)
	run.ReworkRequests = append(run.ReworkRequests, reworkRef+"#review_result:"+reviewResultRef+"#review_request:"+reviewRequestRef+"#delivery:"+deliveryRef)
	run.LastSequence = sequenceBase + int64(len(events))
	run.LastEventID = events[len(events)-1].EventID
	if err := stack.Stores.RunStore.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0 review/rework %s: %v", suffix, err)
	}
}

func continueGeneratedRecursiveFlowReplanV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	runRef string,
	suffix string,
) {
	t.Helper()
	result, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-05-22T21:00:00Z",
		CorrelationID:        "corr-generated-recursive-replan-" + suffix,
		MaxBursts:            8,
		MaxStepsPerBurst:     12,
		MaxDispatchesPerWait: 4,
		MaxCommands:          40,
		MaxOutboxPerCycle:    24,
		MaxDecisionCycles:    4,
		MaxExternalWaits:     1,
	})
	if err != nil {
		t.Fatalf("DrainRunV0 replan %s: %v issues=%+v status=%s final=%+v", suffix, err, codexStackBurstIssuesForErrorV0(err), result.Status, result.Final)
	}
}

func ensureGeneratedRecursiveFlowPhaseV0(
	t *testing.T,
	stack StackV0,
	runRef string,
	phase orquestacoreworkflow.OrchestrationPhaseIDV0,
	reason string,
) {
	t.Helper()
	run := mustLoadCodexStackRunForTestV0(t, stack, runRef)
	run.CurrentPhase = phase
	for index := range run.Phases {
		if run.Phases[index].ID == phase {
			run.Phases[index].Status = orquestacoreworkflow.OrchestrationPhaseStatusActiveV0
			continue
		}
		if run.Phases[index].Status == orquestacoreworkflow.OrchestrationPhaseStatusActiveV0 {
			run.Phases[index].Status = orquestacoreworkflow.OrchestrationPhaseStatusPendingV0
		}
	}
	_ = reason
	if err := stack.Stores.RunStore.SaveRunV0(context.Background(), run); err != nil {
		t.Fatalf("SaveRunV0 ensure phase %s: %v", phase, err)
	}
}

func generatedRecursiveFlowEventMetaV0(
	runRef string,
	sequence int64,
	suffix string,
) orquestacoreworkflow.OrchestrationEventMetaV0 {
	return orquestacoreworkflow.OrchestrationEventMetaV0{
		EventID:       fmt.Sprintf("evt-generated-recursive-%s-%03d", suffix, sequence),
		RunID:         runRef,
		Sequence:      sequence,
		CorrelationID: "corr-generated-recursive-seed-review",
		OccurredAt:    "2026-05-22T21:00:00Z",
	}
}

func generatedRecursiveFlowTaskRefsV0(tasks []orquestacoreworkflow.WorkflowTaskV0) []string {
	out := make([]string, 0, len(tasks))
	for _, task := range tasks {
		out = append(out, task.TaskID)
	}
	return out
}

func generatedRecursiveFlowAgentRefsV0(tasks []orquestacoreworkflow.WorkflowTaskV0) []string {
	out := make([]string, 0, len(tasks))
	for _, task := range tasks {
		out = append(out, orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(task.TaskID))
	}
	return out
}

func assertGeneratedRecursiveFlowTasksMissingV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	runRef string,
	taskRefs []string,
) {
	t.Helper()
	if _, err := stack.Stores.TaskStore.LoadWorkflowTasksV0(ctx, runRef, taskRefs); err == nil {
		t.Fatalf("tasks no debian estar presembradas: %v", taskRefs)
	}
}

func assertGeneratedRecursiveFlowTasksStoredV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	runRef string,
	want []orquestacoreworkflow.WorkflowTaskV0,
) {
	t.Helper()
	got, err := stack.Stores.TaskStore.LoadWorkflowTasksV0(ctx, runRef, generatedRecursiveFlowTaskRefsV0(want))
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0 generated: %v", err)
	}
	for index, task := range want {
		if got[index].TaskID != task.TaskID ||
			got[index].ParentTaskRef != task.ParentTaskRef ||
			got[index].DelegationDepth != task.DelegationDepth ||
			got[index].WaveRef != task.WaveRef ||
			got[index].CohortRef != task.CohortRef {
			t.Fatalf("task generada sin linaje: got=%+v want=%+v", got[index], task)
		}
	}
}

func assertGeneratedRecursiveFlowRunRefsV0(
	t *testing.T,
	stack StackV0,
	runRef string,
	taskRefs []string,
	agentRefs []string,
) {
	t.Helper()
	run := mustLoadCodexStackRunForTestV0(t, stack, runRef)
	for _, taskRef := range taskRefs {
		if !codexStackStringInSetForTestV0(run.Tasks, taskRef) {
			t.Fatalf("task generada no reflejada en run: task=%s tasks=%v", taskRef, run.Tasks)
		}
	}
	for _, agentRef := range agentRefs {
		if !codexStackStringInSetForTestV0(run.StartedAgents, agentRef) {
			t.Fatalf("agente generado no arrancado: agent=%s started=%v run=%+v", agentRef, run.StartedAgents, run)
		}
	}
}
