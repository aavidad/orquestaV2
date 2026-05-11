package orquestadirector

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestProgressiveOrquestaV0GobiernaCapacidadArrancaYParaAgentesLogicos(t *testing.T) {
	h := newProgressiveHarnessV0(t)

	h.openPhase(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0)
	h.handle(h.requestBrainstorm("brainstorm-ref-app-simple-001"))
	h.openPhase(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0)
	h.handle(h.requestVote("vote-ref-app-simple-001", "brainstorm-ref-app-simple-001"))
	h.handle(h.acceptDecision("decision-ref-app-simple-001", "vote-ref-app-simple-001"))
	h.openPhase(orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0)
	h.handle(h.publishFunctionContract("contract-ref-lista-simple-001", "decision-ref-app-simple-001"))
	h.handle(h.createMicrotask("task-ref-lista-simple-001", "contract-ref-lista-simple-001"))
	h.openPhase(orquestacoreworkflow.OrchestrationPhaseProgramacionV0)

	low := progressiveLowCapacityDecisionV0(t, "task-ref-lista-simple-001")
	xhigh := progressiveXHighCapacityDecisionV0(t)
	h.handle(h.requestCapacity("capacity-request-ref-low-001", "task-ref-lista-simple-001", "low"))
	h.handle(h.registerCapacityDecision("capacity-request-ref-low-001", "low"))
	h.handle(h.requestCapacity("capacity-request-ref-xhigh-001", "task-ref-lista-simple-001", "xhigh"))
	h.handle(h.registerCapacityDecision("capacity-request-ref-xhigh-001", "xhigh"))

	agentOne := h.handle(h.requestAgent("agent-request-ref-001", "task-ref-lista-simple-001", "capacity-request-ref-xhigh-001", "implementador"))
	agentTwo := h.handle(h.requestAgent("agent-request-ref-002", "task-ref-lista-simple-001", "capacity-request-ref-xhigh-001", "revisor"))

	runtimeFake := newProgressiveRuntimeFakeOutboxDispatcherV0(t)
	launchedOne := runtimeFake.mustDispatchOnly(agentOne.Outbox)
	runtimeFake.mustDispatchOnly(agentTwo.Outbox)

	progress := progressiveLoopProgressReportV0(t, h.run.RunID, "agent-request-ref-001")
	looped := runtimeFake.mustReportProgress(progress)
	assessmentStop := h.handle(h.assessAgentLoop("assessment-ref-loop-001", "agent-request-ref-001", "task-ref-lista-simple-001", progress.ReportID))
	stopped := runtimeFake.mustDispatchOnly(assessmentStop.Outbox)

	assertProgressiveRuntimeLifecycleV0(t, "agent-request-ref-001", launchedOne, looped, stopped)
	assertProgressiveRunRefsV0(t, h.run, "task-ref-lista-simple-001", "capacity-request-ref-low-001", "capacity-request-ref-xhigh-001")
	assertProgressiveStoppedAgentV0(t, h.run, "agent-request-ref-001")
	assertProgressiveAssessmentV0(t, h.run, "assessment-ref-loop-001")

	if low.Response.NivelCapacidad != "low" || xhigh.Response.NivelCapacidad != "xhigh" {
		t.Fatalf("escalado inesperado: low=%s xhigh=%s", low.Response.NivelCapacidad, xhigh.Response.NivelCapacidad)
	}
	t.Logf(
		"progresiva: eventos=%d agentes=%d evaluaciones=%d parados=%d runtime=%s->%s->%s progress=%s capacity=%v low=%s xhigh=%s",
		len(h.events),
		len(h.run.Agents),
		len(h.run.AgentAssessments),
		len(h.run.StoppedAgents),
		launchedOne.Status,
		looped.Status,
		stopped.Status,
		progress.Status,
		h.run.CapacityRequests,
		low.Response.ReasoningEffort,
		xhigh.Response.ReasoningEffort,
	)
}

type progressiveHarnessV0 struct {
	t      *testing.T
	run    orquestacoreworkflow.OrchestrationRunV0
	events []orquestacoreworkflow.OrchestrationEventV0
}

func newProgressiveHarnessV0(t *testing.T) *progressiveHarnessV0 {
	t.Helper()
	result, err := BootstrapProyectoDesdeAppSpecV0(smokeAppSimpleCommandV0(t))
	if err != nil {
		t.Fatalf("BootstrapProyectoDesdeAppSpecV0: %v", err)
	}
	events := append([]orquestacoreworkflow.OrchestrationEventV0{}, result.WorkflowResult.Events...)
	return &progressiveHarnessV0{
		t:      t,
		run:    replaySmokeRunV0(t, events),
		events: events,
	}
}

func (h *progressiveHarnessV0) handle(command orquestacoreworkflow.OrchestrationCommandV0) orquestacoreworkflow.OrchestrationCommandResultV0 {
	h.t.Helper()
	result, err := orquestacoreworkflow.HandleCommandV0(h.run, command)
	if err != nil {
		h.t.Fatalf("HandleCommandV0 %s: %v", command.CommandType, err)
	}
	h.events = append(h.events, result.Events...)
	h.run = replaySmokeRunV0(h.t, h.events)
	return result
}

func (h *progressiveHarnessV0) openPhase(phase orquestacoreworkflow.OrchestrationPhaseIDV0) {
	h.t.Helper()
	cmd, err := orquestacoreworkflow.NewOpenPhaseCommandV0(h.meta("open-"+string(phase)), orquestacoreworkflow.OpenPhaseCommandPayloadV0{
		PhaseID: string(phase),
		Reason:  "Avance progresivo de prueba.",
	})
	if err != nil {
		h.t.Fatalf("NewOpenPhaseCommandV0: %v", err)
	}
	h.handle(cmd)
}

func (h *progressiveHarnessV0) meta(suffix string) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      "cmd-progress-" + suffix,
		RunID:          h.run.RunID,
		IdempotencyKey: "idem-progress-" + suffix,
		CorrelationID:  "corr-progressive-001",
		RequestedBy:    "director",
		OccurredAt:     "2026-05-04T13:10:00Z",
	}
}

func (h *progressiveHarnessV0) requestBrainstorm(ref string) orquestacoreworkflow.OrchestrationCommandV0 {
	cmd, err := orquestacoreworkflow.NewRequestBrainstormCommandV0(h.meta("brainstorm"), orquestacoreworkflow.RequestBrainstormCommandPayloadV0{
		BrainstormRequestID:        ref,
		PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0),
		TopicRef:                   "topic-ref-app-simple-001",
		Summary:                    "Evaluar opciones tecnicas para app simple.",
		MinimumRecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
		EvidenceRefs:               []string{"evidence-ref-appspec-001"},
	})
	if err != nil {
		h.t.Fatalf("NewRequestBrainstormCommandV0: %v", err)
	}
	return cmd
}

func (h *progressiveHarnessV0) requestVote(ref string, brainstormRef string) orquestacoreworkflow.OrchestrationCommandV0 {
	cmd, err := orquestacoreworkflow.NewRequestVoteCommandV0(h.meta("vote"), orquestacoreworkflow.RequestVoteCommandPayloadV0{
		VoteRequestID:              ref,
		PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0),
		DecisionTopicRef:           "decision-topic-ref-app-simple-001",
		BrainstormRef:              brainstormRef,
		Summary:                    "Elegir estructura hexagonal inicial.",
		MinimumRecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityXHighV0,
		EvidenceRefs:               []string{"evidence-ref-brainstorm-001"},
	})
	if err != nil {
		h.t.Fatalf("NewRequestVoteCommandV0: %v", err)
	}
	return cmd
}

func (h *progressiveHarnessV0) acceptDecision(ref string, voteRef string) orquestacoreworkflow.OrchestrationCommandV0 {
	cmd, err := orquestacoreworkflow.NewAcceptDecisionCommandV0(h.meta("decision"), orquestacoreworkflow.AcceptDecisionCommandPayloadV0{
		DecisionRef:       ref,
		PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0),
		VoteRef:           voteRef,
		AcceptedOptionRef: "option-ref-hexagonal-001",
		Summary:           "Aceptar estructura hexagonal inicial.",
		EvidenceRefs:      []string{"evidence-ref-vote-001"},
	})
	if err != nil {
		h.t.Fatalf("NewAcceptDecisionCommandV0: %v", err)
	}
	return cmd
}

func (h *progressiveHarnessV0) publishFunctionContract(ref string, decisionRef string) orquestacoreworkflow.OrchestrationCommandV0 {
	cmd, err := orquestacoreworkflow.NewPublishFunctionContractCommandV0(h.meta("function-contract"), orquestacoreworkflow.PublishFunctionContractCommandPayloadV0{
		ContractRef:   ref,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0),
		DecisionRef:   decisionRef,
		Summary:       "Contrato compacto para primera tarea.",
		FunctionNames: []string{"CrearListaSimple"},
		EvidenceRefs:  []string{"evidence-ref-decision-001"},
	})
	if err != nil {
		h.t.Fatalf("NewPublishFunctionContractCommandV0: %v", err)
	}
	return cmd
}

func (h *progressiveHarnessV0) createMicrotask(taskID string, contractRef string) orquestacoreworkflow.OrchestrationCommandV0 {
	cmd, err := orquestacoreworkflow.NewCreateMicrotaskCommandV0(h.meta("microtask-"+taskID), orquestacoreworkflow.CreateMicrotaskCommandPayloadV0{
		Task: orquestacoreworkflow.WorkflowTaskV0{
			SchemaVersion: orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
			TaskID:        taskID,
			RunID:         h.run.RunID,
			PhaseID:       orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0,
			Title:         "Crear primera tarea de app simple",
			Summary:       "Preparar primer corte de lista de tareas.",
			WriteSet:      []string{"core/lista_tareas"},
			AcceptanceCriteria: []string{
				"Contrato publicado antes de programar.",
				"Prueba de tarea en verde.",
			},
			FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
				ContractRef:  contractRef,
				FunctionName: "CrearListaSimple",
			}},
		},
	})
	if err != nil {
		h.t.Fatalf("NewCreateMicrotaskCommandV0: %v", err)
	}
	return cmd
}

func (h *progressiveHarnessV0) requestCapacity(ref string, taskRef string, level string) orquestacoreworkflow.OrchestrationCommandV0 {
	cmd, err := orquestacoreworkflow.NewRequestCapacityCommandV0(h.meta("capacity-"+level), orquestacoreworkflow.RequestCapacityCommandPayloadV0{
		CapacityRequestID:          ref,
		PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskRef:                    taskRef,
		ReasonCode:                 "ajuste-capacidad",
		Summary:                    "Ajustar esfuerzo por evidencia de calidad.",
		MinimumRecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityRecommendationV0(level),
		EvidenceRefs:               []string{"evidence-ref-quality-001"},
	})
	if err != nil {
		h.t.Fatalf("NewRequestCapacityCommandV0: %v", err)
	}
	return cmd
}

func (h *progressiveHarnessV0) registerCapacityDecision(ref string, level string) orquestacoreworkflow.OrchestrationCommandV0 {
	cmd, err := orquestacoreworkflow.NewRegisterCapacityDecisionCommandV0(h.meta("capacity-decision-"+ref), orquestacoreworkflow.RegisterCapacityDecisionCommandPayloadV0{
		CapacityRequestID: ref,
		DecisionRef:       "capacity-decision-" + ref,
		Tier:              orquestacoreworkflow.OrchestrationCapacityRecommendationV0(level),
		ReasoningEffort:   orquestacoreworkflow.OrchestrationCapacityRecommendationV0(level),
		Summary:           "Registrar decision compacta de capacidad antes de lanzar agente.",
		EvidenceRefs:      []string{"evidence-ref-capacity-decision-001"},
	})
	if err != nil {
		h.t.Fatalf("NewRegisterCapacityDecisionCommandV0: %v", err)
	}
	return cmd
}

func (h *progressiveHarnessV0) requestAgent(agentRef string, taskRef string, capacityRef string, role string) orquestacoreworkflow.OrchestrationCommandV0 {
	cmd, err := orquestacoreworkflow.NewRequestAgentCommandV0(h.meta(agentRef), orquestacoreworkflow.RequestAgentCommandPayloadV0{
		AgentRequestID:     agentRef,
		PhaseID:            string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskRef:            taskRef,
		CapacityRequestRef: capacityRef,
		Role:               role,
		Summary:            "Ejecutar tarea compacta con evidencia.",
		EvidenceRefs:       []string{"evidence-ref-task-001"},
	})
	if err != nil {
		h.t.Fatalf("NewRequestAgentCommandV0: %v", err)
	}
	return cmd
}

func (h *progressiveHarnessV0) assessAgentLoop(assessmentRef string, agentRef string, taskRef string, progressRef string) orquestacoreworkflow.OrchestrationCommandV0 {
	cmd, err := orquestacoreworkflow.NewAssessAgentWorkCommandV0(h.meta("assess-"+assessmentRef), orquestacoreworkflow.AssessAgentWorkCommandPayloadV0{
		AssessmentRef:  assessmentRef,
		PhaseID:        string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		AgentRequestID: agentRef,
		TaskRef:        taskRef,
		Verdict:        orquestacoreworkflow.AgentAssessmentVerdictLoopDetectedV0,
		Action:         orquestacoreworkflow.AgentAssessmentActionStopAgentV0,
		Severity:       orquestacoreworkflow.AgentAssessmentSeverityCriticalV0,
		Summary:        "Bucle detectado por supervision compacta; detener agente logico.",
		EvidenceRefs:   []string{progressRef},
	})
	if err != nil {
		h.t.Fatalf("NewAssessAgentWorkCommandV0: %v", err)
	}
	return cmd
}
