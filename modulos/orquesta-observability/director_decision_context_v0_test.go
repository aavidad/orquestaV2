package orquestaobservability

import "testing"

func TestValidateDirectorDecisionContextV0Completo(t *testing.T) {
	context := validDirectorDecisionContextV0()

	if err := ValidateDirectorDecisionContextV0(context); err != nil {
		t.Fatalf("context should validate: %v", err)
	}
	if context.Privacy.ContainsSecret ||
		context.Privacy.ContainsTranscript ||
		context.Privacy.ContainsPrompt ||
		context.Privacy.ContainsCompletion ||
		context.Privacy.ContainsConnectionDetail {
		t.Fatalf("privacy flags should be false: %+v", context.Privacy)
	}
}

func TestValidateDirectorDecisionContextV0CubreDecisionDirector(t *testing.T) {
	context := validDirectorDecisionContextV0()

	if context.Progress.TasksOpen != 1 ||
		context.Lifecycle.AgentsRunning != 1 ||
		context.Lifecycle.AgentsFailed != 1 ||
		context.Closure.BlockedBy[0] != "validacion_final" ||
		context.ReworkReplan.ReworkRequests != 1 ||
		context.ReworkReplan.ReplanDecisions != 1 ||
		context.Quietness.MaxNoProgressTicks != 3 ||
		len(context.Phases) != 2 ||
		len(context.Tasks) != 2 ||
		len(context.Agents) != 2 ||
		len(context.Blockers) != 1 ||
		len(context.Activity) < 3 {
		t.Fatalf("context incompleto para decision: %+v", context)
	}
}

func TestValidateDirectorDecisionContextV0RechazaRefsYContenidoProhibido(t *testing.T) {
	tests := []struct {
		name string
		edit func(*DirectorDecisionContextV0)
		code string
	}{
		{
			name: "run ref no opaca",
			edit: func(context *DirectorDecisionContextV0) {
				context.RunRef = "/home/alberto/run"
			},
			code: ErrReferenciaNoOpacaV0,
		},
		{
			name: "process ref con secreto",
			edit: func(context *DirectorDecisionContextV0) {
				context.Agents[0].ProcessRef = "process-token-ref-001"
			},
			code: ErrSecretoDetectadoV0,
		},
		{
			name: "summary key invalida",
			edit: func(context *DirectorDecisionContextV0) {
				context.Activity[0].SummaryKey = "Texto visible"
			},
			code: ErrOperationalStatusQueryInvalidaV0,
		},
		{
			name: "privacy transcript",
			edit: func(context *DirectorDecisionContextV0) {
				context.Privacy.ContainsTranscript = true
			},
			code: ErrTranscriptNoPermitidoV0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			context := validDirectorDecisionContextV0()
			test.edit(&context)
			assertOperationalStatusIssueV0(t, ValidateDirectorDecisionContextV0(context), test.code)
		})
	}
}

func TestValidateDirectorDecisionContextV0ListasCompactas(t *testing.T) {
	context := validDirectorDecisionContextV0()
	context.Activity = make([]DirectorDecisionActivityV0, maxOperationalListItemsV0+1)
	for index := range context.Activity {
		context.Activity[index] = DirectorDecisionActivityV0{
			ActivityRef: "activity-ref-director-" + leftPadOperationalStatusTestV0(index),
			Kind:        DirectorDecisionActivityTaskProgressV0,
			SourceRef:   "task-ref-director-" + leftPadOperationalStatusTestV0(index),
			SummaryKey:  "director.decision_context.activity.task_progress",
		}
	}

	assertOperationalStatusIssueV0(t, ValidateDirectorDecisionContextV0(context), ErrConsultaDemasiadoAmpliaV0)
}

func validDirectorDecisionContextV0() DirectorDecisionContextV0 {
	return DirectorDecisionContextV0{
		SchemaVersion: DirectorDecisionContextSchemaVersionV0,
		RunRef:        "run-ref-director-context-001",
		ObservedAt:    "2026-05-10T10:00:00Z",
		CurrentPhase:  "programacion",
		Progress: DirectorDecisionProgressV0{
			PercentComplete:   50,
			TasksTotal:        2,
			TasksClosed:       1,
			TasksOpen:         1,
			TasksObserved:     2,
			ObservedAgents:    2,
			ProgressingAgents: 1,
			StalledAgents:     1,
			NoSignalAgentRefs: []string{"agent-ref-director-context-002"},
		},
		Lifecycle: DirectorDecisionLifecycleV0{
			AgentsRequested:     2,
			AgentsStarted:       2,
			AgentsRunning:       1,
			AgentsFailed:        1,
			AgentsInFlight:      1,
			AgentsNeedAttention: 1,
		},
		Closure: DirectorDecisionClosureV0{
			Status:      "blocked",
			Blocked:     true,
			BlockedBy:   []string{"validacion_final"},
			BlockerRefs: []string{"blocker-ref-director-context-001"},
		},
		Quietness: DirectorDecisionQuietnessV0{
			NoSignalAgentRefs:  []string{"agent-ref-director-context-002"},
			StalledAgents:      1,
			MaxNoProgressTicks: 3,
		},
		ReworkReplan: DirectorDecisionReworkReplanV0{
			ReworkRequests:     1,
			ReworkRequestRefs:  []string{"rework-ref-director-context-001"},
			ReplanDecisions:    1,
			ReplanDecisionRefs: []string{"replan-ref-director-context-001"},
		},
		Phases: []DirectorDecisionPhaseV0{
			{PhaseID: "brainstorming_arquitectura", Status: "cerrada", ClosedAt: "2026-05-10T09:00:00Z"},
			{PhaseID: "programacion", Status: "activa", Current: true, OpenedAt: "2026-05-10T09:00:00Z", DurationSeconds: 3600},
		},
		Tasks: []DirectorDecisionTaskV0{
			{TaskRef: "task-ref-director-context-001", Status: "closed"},
			{TaskRef: "task-ref-director-context-002", Status: "stalled", AgentRequestID: "agent-ref-director-context-002", NoProgressTicks: 3},
		},
		Agents: []DirectorDecisionAgentV0{
			{AgentRequestID: "agent-ref-director-context-001", Status: "running", Running: true, InFlight: true, ProcessRef: "process-ref-director-context-001", SessionRef: "session-ref-director-context-001"},
			{AgentRequestID: "agent-ref-director-context-002", Status: "failed", Failed: true, NeedsAttention: true},
		},
		Blockers: []DirectorDecisionBlockerV0{
			{BlockerRef: "blocker-ref-director-context-001", Cause: "validacion_final", Source: "closure", SummaryKey: "director.decision_context.blocker.closure_blocked"},
		},
		Activity: []DirectorDecisionActivityV0{
			{ActivityRef: "activity-ref-director-context-001", Kind: DirectorDecisionActivityPhaseCurrentV0, SourceRef: "run-ref-director-context-001", SummaryKey: "director.decision_context.activity.phase_current"},
			{ActivityRef: "activity-ref-director-context-002", Kind: DirectorDecisionActivityTaskProgressV0, SourceRef: "task-ref-director-context-002", TaskRef: "task-ref-director-context-002", SummaryKey: "director.decision_context.activity.task_progress"},
			{ActivityRef: "activity-ref-director-context-003", Kind: DirectorDecisionActivityClosureBlockedV0, SourceRef: "blocker-ref-director-context-001", SummaryKey: "director.decision_context.activity.closure_blocked"},
		},
		Privacy: DiagnosticoPrivacyV0{},
	}
}
