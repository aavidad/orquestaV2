package orquestadirector

import (
	"os"
	"path/filepath"
	"testing"

	orquestacapacity "orquesta/modulos/orquesta-capacity"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func progressiveLowCapacityDecisionV0(t *testing.T, taskRef string) orquestacapacity.CapacityDecisionV0 {
	t.Helper()
	decision := progressiveCapacityDecisionFixtureV0(t, "decision_minima_valida.json")
	decision.Request.TaskRef = taskRef
	decision.Response.DecisionID = "capdec-progress-low-001"
	decision.Response.NivelCapacidad = "low"
	decision.Response.ReasoningEffort = "low"
	requireProgressiveCapacityDecisionValidV0(t, decision)
	return decision
}

func progressiveXHighCapacityDecisionV0(t *testing.T) orquestacapacity.CapacityDecisionV0 {
	t.Helper()
	decision := progressiveCapacityDecisionFixtureV0(t, "decision_xhigh_evidencia_valida.json")
	requireProgressiveCapacityDecisionValidV0(t, decision)
	return decision
}

func progressiveCapacityDecisionFixtureV0(t *testing.T, name string) orquestacapacity.CapacityDecisionV0 {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "orquesta-capacity", "docs", "fixtures", "capacity_decision_v0", name))
	if err != nil {
		t.Fatalf("read capacity fixture %s: %v", name, err)
	}
	decision, err := orquestacapacity.DecodeCapacityDecisionV0(data)
	if err != nil {
		t.Fatalf("DecodeCapacityDecisionV0 %s: %v", name, err)
	}
	return decision
}

func requireProgressiveCapacityDecisionValidV0(t *testing.T, decision orquestacapacity.CapacityDecisionV0) {
	t.Helper()
	if issues := orquestacapacity.ValidateCapacityDecisionV0(decision); len(issues) != 0 {
		t.Fatalf("capacity decision invalida: %#v", issues)
	}
}

func progressiveLoopProgressReportV0(t *testing.T, runID string, agentRef string) orquestaruntime.AgentProgressReportV0 {
	t.Helper()
	report := orquestaruntime.AgentProgressReportV0{
		ReportID:            "agent-progress-report-ref-001",
		RunID:               runID,
		AgentRequestID:      agentRef,
		Status:              orquestaruntime.AgentLoopDetectedV0,
		NoProgressTicks:     4,
		RepeatedActionCount: 3,
		Summary:             "Sin avance y acciones repetidas durante supervision compacta.",
		EvidenceRefs:        []string{"supervision-evidence-ref-001"},
	}
	if issues := orquestaruntime.ValidateAgentProgressReportV0(report); len(issues) != 0 {
		t.Fatalf("AgentProgressReport invalido: %#v", issues)
	}
	return report
}

func assertProgressiveRuntimeLifecycleV0(
	t *testing.T,
	expectedAgentRef string,
	launched orquestaruntime.RuntimeFakeLifecycleSnapshotV0,
	looped orquestaruntime.RuntimeFakeLifecycleSnapshotV0,
	stopped orquestaruntime.RuntimeFakeLifecycleSnapshotV0,
) {
	t.Helper()
	if launched.Status != orquestaruntime.RuntimeFakeLifecycleLaunchedV0 ||
		looped.Status != orquestaruntime.RuntimeFakeLifecycleLoopDetectedV0 ||
		stopped.Status != orquestaruntime.RuntimeFakeLifecycleStoppedV0 {
		t.Fatalf("runtime lifecycle inesperado: launch=%+v loop=%+v stop=%+v", launched, looped, stopped)
	}
	if launched.AgentRequestID != looped.AgentRequestID || looped.AgentRequestID != stopped.AgentRequestID {
		t.Fatalf("runtime lifecycle cambio de agente: launch=%+v loop=%+v stop=%+v", launched, looped, stopped)
	}
	if stopped.AgentRequestID != expectedAgentRef {
		t.Fatalf("runtime lifecycle agent=%q, want %q", stopped.AgentRequestID, expectedAgentRef)
	}
}

func assertProgressiveRunRefsV0(
	t *testing.T,
	run orquestacoreworkflow.OrchestrationRunV0,
	taskRef string,
	lowCapacityRef string,
	xhighCapacityRef string,
) {
	t.Helper()
	if !containsProgressiveRefV0(run.Tasks, taskRef) {
		t.Fatalf("task %q no proyectada: %+v", taskRef, run.Tasks)
	}
	if !containsProgressiveRefV0(run.CapacityRequests, lowCapacityRef) ||
		!containsProgressiveRefV0(run.CapacityRequests, xhighCapacityRef) {
		t.Fatalf("capacity refs inesperadas: %+v", run.CapacityRequests)
	}
	if len(run.Agents) != 2 ||
		!containsProgressiveRefV0(run.Agents, "agent-request-ref-001") ||
		!containsProgressiveRefV0(run.Agents, "agent-request-ref-002") {
		t.Fatalf("agentes inesperados: %+v", run.Agents)
	}
}

func assertProgressiveStoppedAgentV0(t *testing.T, run orquestacoreworkflow.OrchestrationRunV0, agentRef string) {
	t.Helper()
	if !containsProgressiveRefV0(run.StoppedAgents, agentRef) {
		t.Fatalf("agente parado %q no proyectado: %+v", agentRef, run.StoppedAgents)
	}
}

func assertProgressiveAssessmentV0(t *testing.T, run orquestacoreworkflow.OrchestrationRunV0, assessmentRef string) {
	t.Helper()
	if !containsProgressiveAssessmentRefV0(run.AgentAssessments, assessmentRef) {
		t.Fatalf("evaluacion %q no proyectada: %+v", assessmentRef, run.AgentAssessments)
	}
}

func containsProgressiveAssessmentRefV0(values []string, ref string) bool {
	for _, value := range values {
		if orquestacoreworkflow.AgentAssessmentProjectionIDV0(value) == ref {
			return true
		}
	}
	return false
}

func containsProgressiveRefV0(values []string, ref string) bool {
	for _, value := range values {
		if value == ref {
			return true
		}
	}
	return false
}
