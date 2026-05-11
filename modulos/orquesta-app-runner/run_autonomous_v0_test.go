package orquestaapprunner

import (
	"context"
	"testing"

	orquestaappplanner "orquesta/modulos/orquesta-app-planner"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestRunPreparedAppOrchestrationV0UsaDirectorAutonomoOptInYPropagaStats(t *testing.T) {
	prepared := validPreparedLargeAppForRunTestV0(t)
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	policy := &runnerRecordingAutonomousPolicyForTestV0{}
	ports := validRunPreparedPortsForTestV0(store, sink, ledger)
	ports.AutonomousDirectorPolicy = policy
	request := validRunPreparedRequestForTestV0(prepared)
	request.UseAutonomousDirectorLoop = true

	result, err := RunPreparedAppOrchestrationV0(context.Background(), request, ports)
	if err != nil {
		t.Fatalf("RunPreparedAppOrchestrationV0 autonomous: %v", err)
	}
	if !policy.Called {
		t.Fatalf("autonomous policy no invocada")
	}
	if result.DirectorLoopStats == nil {
		t.Fatalf("director_loop_stats no propagadas: %+v", result)
	}
	if result.DirectorLoopStats.Decision.Summary != runnerAutonomousSummaryForTestV0 {
		t.Fatalf("stats=%+v", result.DirectorLoopStats)
	}
	if result.DirectorLoopStats.Run.RunRef != prepared.Run.RunID ||
		result.DirectorLoopStats.Loop.Status != string(result.LoopStatus) {
		t.Fatalf("stats no corresponden al loop final: %+v", result.DirectorLoopStats)
	}
	if policy.Input.Limits.MaxCommandsPerCycle != request.MaxCommands ||
		policy.Input.Limits.MaxOutboxPerCycle != request.MaxOutboxPerCycle {
		t.Fatalf("limits=%+v request=%+v", policy.Input.Limits, request)
	}
}

func TestRunPreparedAppOrchestrationV0DirectorAutonomoMantieneEsperaExterna(t *testing.T) {
	prepared := validPreparedLargeAppForRunTestV0(t)
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	source := newRunnerStepwiseDeliverySourceForTestV0(prepared.Plan)
	ports := validRunPreparedPortsForTestV0(store, sink, ledger)
	ports.DeliverySource = source
	ports.ExternalWaiter = source
	request := validRunPreparedRequestForTestV0(prepared)
	request.UseAutonomousDirectorLoop = true
	request.MaxExternalWaits = 0

	result, err := RunPreparedAppOrchestrationV0(context.Background(), request, ports)
	if err != nil {
		t.Fatalf("RunPreparedAppOrchestrationV0 autonomous wait: %v", err)
	}
	if result.Status != AppOrchestrationRunStatusCompleteV0 || result.ExternalWaits == 0 {
		t.Fatalf("result=%+v", result)
	}
	if result.DirectorLoopStats == nil ||
		result.DirectorLoopStats.Loop.Status != string(result.LoopStatus) {
		t.Fatalf("stats=%+v result=%+v", result.DirectorLoopStats, result)
	}
}

func TestRunPreparedAppOrchestrationV0DirectorAutonomoEnriqueceStatsConProgressSource(t *testing.T) {
	prepared := validPreparedLargeAppForRunTestV0(t)
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	progressSource := runnerMixedProgressSourceForTestV0{Plan: prepared.Plan}
	ports := validRunPreparedPortsForTestV0(store, sink, ledger)
	ports.DeliverySource = runnerStartedAgentDeliverySourceForTestV0{Plan: prepared.Plan}
	ports.ProgressSource = progressSource
	request := validRunPreparedRequestForTestV0(prepared)
	request.UseAutonomousDirectorLoop = true
	request.MaxBursts = 80
	request.MaxStepsPerBurst = 8
	request.MaxDispatchesPerWait = 8
	request.MaxCommands = 24
	request.MaxOutboxPerCycle = 24

	result, err := RunPreparedAppOrchestrationV0(context.Background(), request, ports)
	if err != nil {
		t.Fatalf("RunPreparedAppOrchestrationV0 autonomous progress: %v", err)
	}
	if result.DirectorLoopStats == nil {
		t.Fatalf("director_loop_stats no propagadas: %+v", result)
	}
	progress := result.DirectorLoopStats.Run.Progress
	if progress.SourceStatus != orquestacionnucleoapp.DirectorProgressSourceLoadedV0 ||
		progress.ProgressingAgents != 1 ||
		progress.StalledAgents != 1 ||
		progress.TasksObserved != 2 {
		t.Fatalf("progress=%+v", progress)
	}
	if !runnerStatsContainTaskStatusV0(
		progress.Tasks,
		orquestacionnucleoapp.DirectorTaskProgressStalledV0,
	) {
		t.Fatalf("tasks sin estancamiento util: %+v", progress.Tasks)
	}
	if !runnerStatsContainAgentProgressV0(
		result.DirectorLoopStats.Run.Agents,
		string(orquestaruntime.AgentStalledV0),
	) {
		t.Fatalf("agents sin last_progress stalled: %+v", result.DirectorLoopStats.Run.Agents)
	}
}

const runnerAutonomousSummaryForTestV0 = "runner autonomous test policy"

type runnerRecordingAutonomousPolicyForTestV0 struct {
	Called bool
	Input  orquestacionnucleoapp.AutonomousDirectorDecisionInputV0
}

func (policy *runnerRecordingAutonomousPolicyForTestV0) DecideAutonomousDirectorV0(
	_ context.Context,
	input orquestacionnucleoapp.AutonomousDirectorDecisionInputV0,
) (orquestacionnucleoapp.AutonomousDirectorDecisionV0, error) {
	policy.Called = true
	policy.Input = input
	return orquestacionnucleoapp.AutonomousDirectorDecisionV0{
		TeamSize:             1,
		MaxParallelAgents:    1,
		MaxBursts:            input.Limits.MaxBursts,
		MaxStepsPerBurst:     input.Limits.MaxStepsPerBurst,
		MaxDispatchesPerWait: input.Limits.MaxDispatchesPerWait,
		MaxCommandsPerCycle:  input.Limits.MaxCommandsPerCycle,
		MaxOutboxPerCycle:    input.Limits.MaxOutboxPerCycle,
		Summary:              runnerAutonomousSummaryForTestV0,
		EvidenceRefs: compactAppRunnerRefsV0(append(
			input.Requests.EvidenceRefs,
			"evidence-ref-runner-autonomous-policy-test",
		)),
	}, nil
}

type runnerMixedProgressSourceForTestV0 struct {
	Plan orquestaappplanner.AppMicrotaskPlanV0
}

func (source runnerMixedProgressSourceForTestV0) BuildAgentProgressObservationsV0(
	_ context.Context,
	request orquestacionnucleoapp.AgentProgressObservationRequestV0,
) ([]orquestacionnucleoapp.AgentProgressObservationV0, error) {
	if len(request.Run.StartedAgents) < 2 {
		return nil, nil
	}
	statuses := []orquestaruntime.AgentProgressStatusV0{
		orquestaruntime.AgentProgressingV0,
		orquestaruntime.AgentStalledV0,
	}
	observations := make([]orquestacionnucleoapp.AgentProgressObservationV0, 0, len(statuses))
	for index, status := range statuses {
		agentRef := request.Run.StartedAgents[index]
		unit, ok := runnerUnitForAgentV0(source.Plan, agentRef)
		if !ok {
			continue
		}
		observations = append(observations, runnerProgressObservationForTestV0(
			request.Run.RunID,
			unit,
			index+1,
			status,
		))
	}
	return observations, nil
}

func runnerProgressObservationForTestV0(
	runRef string,
	unit orquestaappplanner.AppWorkUnitV0,
	index int,
	status orquestaruntime.AgentProgressStatusV0,
) orquestacionnucleoapp.AgentProgressObservationV0 {
	report := orquestaruntime.AgentProgressReportV0{
		ReportID:       runnerProgressReportRefV0(index),
		RunID:          runRef,
		AgentRequestID: unit.AgentRequestID,
		Status:         status,
		Summary:        "Avance compacto observado.",
		EvidenceRefs:   []string{"evidence-ref-app-runner-progress-report"},
	}
	if status == orquestaruntime.AgentStalledV0 {
		report.NoProgressTicks = 3
		report.RepeatedActionCount = 1
		report.Summary = "Sin avance observable; requiere direccion."
	}
	return orquestacionnucleoapp.AgentProgressObservationV0{
		CandidateRef:  "progress-candidate-ref-app-runner-" + runnerProgressSuffixV0(index),
		Report:        report,
		TaskRef:       unit.TaskRef,
		DeliveryRef:   unit.DeliveryRef,
		AssessmentRef: "assessment-ref-app-runner-progress-" + runnerProgressSuffixV0(index),
		QuestionID:    "question-ref-app-runner-progress-" + runnerProgressSuffixV0(index),
		EvidenceRefs:  []string{"evidence-ref-app-runner-progress-source"},
	}
}

func runnerUnitForAgentV0(
	plan orquestaappplanner.AppMicrotaskPlanV0,
	agentRef string,
) (orquestaappplanner.AppWorkUnitV0, bool) {
	for _, unit := range plan.Units {
		if unit.AgentRequestID == agentRef {
			return unit, true
		}
	}
	return orquestaappplanner.AppWorkUnitV0{}, false
}

func runnerProgressReportRefV0(index int) string {
	return "agent-progress-report-ref-app-runner-" + runnerProgressSuffixV0(index)
}

func runnerProgressSuffixV0(index int) string {
	if index == 1 {
		return "001"
	}
	return "002"
}

func runnerStatsContainTaskStatusV0(
	tasks []orquestacionnucleoapp.DirectorTaskProgressV0,
	status string,
) bool {
	for _, task := range tasks {
		if task.Status == status {
			return true
		}
	}
	return false
}

func runnerStatsContainAgentProgressV0(
	agents []orquestacionnucleoapp.DirectorAgentStatsV0,
	status string,
) bool {
	for _, agent := range agents {
		if agent.LastProgress != nil && agent.LastProgress.Status == status {
			return true
		}
	}
	return false
}
